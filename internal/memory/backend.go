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
	"fmt"
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
	eventsBatchSize   = 100
)

type Store interface {
	Evaluate(key string, namespace string, lekkoContext map[string]interface{}, dest proto.Message) error
	Close(ctx context.Context) error
}

// Constructs an in-memory store that fetches configs from lekko's backend.
func NewBackendStore(
	ctx context.Context,
	apiKey, url, ownerName, repoName string,
	updateInterval time.Duration,
	serverPort int32,
) (Store, error) {
	return newBackendStore(
		ctx,
		apiKey, ownerName, repoName,
		updateInterval,
		backendv1beta1connect.NewDistributionServiceClient(http.DefaultClient, url),
		eventsBatchSize,
		serverPort,
	)
}

func newBackendStore(
	ctx context.Context,
	apiKey, ownerName, repoName string,
	updateInterval time.Duration,
	distClient backendv1beta1connect.DistributionServiceClient,
	eventsBatchSize int,
	serverPort int32,
) (*backendStore, error) {
	ctx, cancel := context.WithCancel(ctx)
	b := &backendStore{
		distClient: distClient,
		store:      newStore(ownerName, repoName),
		repoKey: &backendv1beta1.RepositoryKey{
			OwnerName: ownerName,
			RepoName:  repoName,
		},
		apiKey:         apiKey,
		updateInterval: updateInterval,
		cancel:         cancel,
	}
	// register with lekko backend
	sessionKey, err := b.registerWithBackoff(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "error registering client")
	}
	b.sessionKey = sessionKey
	b.eb = newEventBatcher(ctx, distClient, b.sessionKey, b.apiKey, eventsBatchSize)
	// initialize the store once with configs
	if _, err := b.updateStoreWithBackoff(ctx); err != nil {
		return nil, err
	}
	b.server = newSDKServer(serverPort, b.store)
	// kick off an asynchronous goroutine that updates the store periodically
	b.loop(ctx)
	return b, nil
}

type backendStore struct {
	distClient         backendv1beta1connect.DistributionServiceClient
	store              *store
	repoKey            *backendv1beta1.RepositoryKey
	apiKey, sessionKey string
	wg                 sync.WaitGroup
	updateInterval     time.Duration
	cancel             context.CancelFunc
	eb                 *eventBatcher
	server             *sdkServer
}

// Close implements Store.
func (b *backendStore) Close(ctx context.Context) error {
	// cancel any ongoing background loops
	b.cancel()
	b.server.close(ctx)
	b.eb.close(ctx)
	// wait for background work to complete
	b.wg.Wait()
	req := connect.NewRequest(&backendv1beta1.DeregisterClientRequest{
		SessionKey: b.sessionKey,
	})
	req.Header().Set(lekkoAPIKeyHeader, b.apiKey)
	_, err := b.distClient.DeregisterClient(ctx, req)
	return err
}

// Evaluate implements Store.
func (b *backendStore) Evaluate(key string, namespace string, lc map[string]interface{}, dest protoreflect.ProtoMessage) error {
	cfg, rp, err := b.store.evaluateType(key, namespace, lc, dest)
	if err != nil {
		return err
	}
	// track metrics
	b.eb.track(&backendv1beta1.FlagEvaluationEvent{
		RepoKey:       b.repoKey,
		CommitSha:     cfg.CommitSHA,
		FeatureSha:    cfg.ConfigSHA,
		NamespaceName: namespace,
		FeatureName:   cfg.Config.GetKey(),
		ContextKeys:   toContextKeysProto(lc),
		ResultPath:    toResultPathProto(rp),
	})
	return nil
}

func (b *backendStore) registerWithBackoff(ctx context.Context) (string, error) {
	req := connect.NewRequest(&backendv1beta1.RegisterClientRequest{
		RepoKey:       b.repoKey,
		NamespaceList: []string{}, // register all namespaces
	})
	req.Header().Set(lekkoAPIKeyHeader, b.apiKey)
	var resp *connect.Response[backendv1beta1.RegisterClientResponse]
	var err error
	op := func() error {
		resp, err = b.distClient.RegisterClient(ctx, req)
		return err
	}

	exp := backoff.NewExponentialBackOff()
	exp.MaxElapsedTime = 5 * time.Second
	if err := backoff.Retry(op, exp); err != nil {
		return "", err
	}
	var sessionKey string
	if resp != nil {
		sessionKey = resp.Msg.GetSessionKey()
	}
	return sessionKey, nil
}

func (b *backendStore) updateStoreWithBackoff(ctx context.Context) (bool, error) {
	req := connect.NewRequest(&backendv1beta1.GetRepositoryContentsRequest{
		RepoKey:    b.repoKey,
		SessionKey: b.sessionKey,
	})
	req.Header().Set(lekkoAPIKeyHeader, b.apiKey)
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
	return b.store.update(contents)
}

func (b *backendStore) shouldUpdateStore(ctx context.Context) (bool, error) {
	req := connect.NewRequest(&backendv1beta1.GetRepositoryVersionRequest{
		RepoKey:    b.repoKey,
		SessionKey: b.sessionKey,
	})
	req.Header().Set(lekkoAPIKeyHeader, b.apiKey)
	resp, err := b.distClient.GetRepositoryVersion(ctx, req)
	if err != nil {
		return false, errors.Wrap(err, "failed to get repository version")
	}
	return b.store.getCommitSha() != resp.Msg.GetCommitSha(), nil
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

func toContextKeysProto(lc map[string]interface{}) []*backendv1beta1.ContextKey {
	var ret []*backendv1beta1.ContextKey
	for k, v := range lc {
		ret = append(ret, &backendv1beta1.ContextKey{
			Key:  k,
			Type: contextKeyTypeToProto(v),
		})
	}
	return ret
}

func contextKeyTypeToProto(v interface{}) string {
	switch v.(type) {
	case bool:
		return "bool"
	case string:
		return "string"
	case int:
		return "int"
	case int8:
		return "int"
	case int16:
		return "int"
	case int32:
		return "int"
	case int64:
		return "int"
	case uint:
		return "int"
	case uint16:
		return "int"
	case uint32:
		return "int"
	case uint64:
		return "int"
	case uint8:
		return "int"
	case float32:
		return "float"
	case float64:
		return "float"
	default:
		return fmt.Sprintf("%T", v)
	}
}

func toResultPathProto(rp []int) []int32 {
	ret := make([]int32, len(rp))
	for i := 0; i < len(rp); i++ {
		ret[i] = int32(rp[i])
	}
	return ret
}
