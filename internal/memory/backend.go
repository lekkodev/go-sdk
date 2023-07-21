// Copyright 2022 Lekko Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package memory

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"buf.build/gen/go/lekkodev/cli/bufbuild/connect-go/lekko/backend/v1beta1/backendv1beta1connect"
	backendv1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/backend/v1beta1"
	"github.com/bufbuild/connect-go"
	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	lekkoAPIKeyHeader = "apikey"
)

type Store interface {
	Evaluate(key string, namespace string, lekkoContext map[string]interface{}, dest proto.Message) error
	Close(ctx context.Context) error
}

// Constructs an in-memory store that fetches configs from lekko's backend.
func NewBackendStore(ctx context.Context, apikey, url, ownerName, repoName string, updateInterval time.Duration) (Store, error) {
	return newBackendStore(ctx, apikey, ownerName, repoName, updateInterval, backendv1beta1connect.NewDistributionServiceClient(http.DefaultClient, url))
}

func newBackendStore(ctx context.Context, apikey, ownerName, repoName string, updateInterval time.Duration, distClient backendv1beta1connect.DistributionServiceClient) (*backendStore, error) {
	ctx, cancel := context.WithCancel(ctx)
	b := &backendStore{
		distClient: distClient,
		store:      newStore(),
		repoKey: &backendv1beta1.RepositoryKey{
			OwnerName: ownerName,
			RepoName:  repoName,
		},
		apikey:         apikey,
		updateInterval: updateInterval,
		cancel:         cancel,
	}
	// register with lekko backend
	if err := b.registerWithBackoff(ctx); err != nil {
		return nil, errors.Wrap(err, "error registering client")
	}
	// initialize the store once with configs
	if _, err := b.updateStoreWithBackoff(ctx); err != nil {
		return nil, err
	}
	// kick off an asynchronous goroutine that updates the store periodically
	b.loop(ctx)
	return b, nil
}

type backendStore struct {
	distClient         backendv1beta1connect.DistributionServiceClient
	store              *store
	repoKey            *backendv1beta1.RepositoryKey
	apikey, sessionKey string
	wg                 sync.WaitGroup
	updateInterval     time.Duration
	cancel             context.CancelFunc
}

// Close implements Store.
func (b *backendStore) Close(ctx context.Context) error {
	// cancel any ongoing background loops
	b.cancel()
	// wait for background work to complete
	b.wg.Wait()
	_, err := b.distClient.DeregisterClient(ctx, connect.NewRequest(&backendv1beta1.DeregisterClientRequest{
		SessionKey: b.sessionKey,
	}))
	return err
}

// Evaluate implements Store.
func (b *backendStore) Evaluate(key string, namespace string, lc map[string]interface{}, dest protoreflect.ProtoMessage) error {
	return b.store.evaluateType(key, namespace, lc, dest)
}

func (b *backendStore) registerWithBackoff(ctx context.Context) error {
	req := connect.NewRequest(&backendv1beta1.RegisterClientRequest{
		RepoKey:       b.repoKey,
		NamespaceList: []string{}, // register all namespaces
	})
	req.Header().Set(lekkoAPIKeyHeader, b.apikey)
	var resp *connect.Response[backendv1beta1.RegisterClientResponse]
	var err error
	op := func() error {
		resp, err = b.distClient.RegisterClient(ctx, req)
		return err
	}

	exp := backoff.NewExponentialBackOff()
	exp.MaxElapsedTime = 5 * time.Second
	if err := backoff.Retry(op, exp); err != nil {
		return err
	}
	if resp != nil {
		b.sessionKey = resp.Msg.GetSessionKey()
	}
	return nil
}

func (b *backendStore) updateStoreWithBackoff(ctx context.Context) (bool, error) {
	req := connect.NewRequest(&backendv1beta1.GetRepositoryContentsRequest{
		RepoKey:    b.repoKey,
		SessionKey: b.sessionKey,
	})
	req.Header().Set(lekkoAPIKeyHeader, b.apikey)
	var contents *backendv1beta1.GetRepositoryContentsResponse
	op := func() error {
		resp, err := b.distClient.GetRepositoryContents(ctx, req)
		if err != nil {
			return errors.Wrap(err, "get repository contents from backend")
		}
		contents = resp.Msg
		return nil
	}
	exp := backoff.NewExponentialBackOff()
	exp.MaxElapsedTime = 10 * time.Second
	if err := backoff.Retry(op, exp); err != nil {
		return false, err
	}
	updated := b.store.update(contents)
	return updated, nil
}

func (b *backendStore) shouldUpdateStore(ctx context.Context) (bool, error) {
	req := connect.NewRequest(&backendv1beta1.GetRepositoryVersionRequest{
		RepoKey:    b.repoKey,
		SessionKey: b.sessionKey,
	})
	req.Header().Set(lekkoAPIKeyHeader, b.apikey)
	resp, err := b.distClient.GetRepositoryVersion(ctx, req)
	if err != nil {
		return false, errors.Wrap(err, "failed to get repository version")
	}
	should := b.store.shouldUpdate(resp.Msg.GetCommitSha())
	return should, nil
}

func (b *backendStore) loop(ctx context.Context) {
	b.wg.Add(1)
	go func() {
		defer b.wg.Done()
		tick := time.NewTicker(b.updateInterval)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
				should, err := b.shouldUpdateStore(ctx)
				if err != nil {
					log.Printf("failed to compare repo version: %v", err)
					continue
				}
				if !should {
					continue
				}
				_, err = b.updateStoreWithBackoff(ctx)
				if err != nil {
					log.Printf("failed to update store: %v", err)
					continue
				}
			}
		}
	}()
}
