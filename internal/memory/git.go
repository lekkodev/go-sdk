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
	"path/filepath"
	"sync"
	"time"

	"buf.build/gen/go/lekkodev/cli/bufbuild/connect-go/lekko/backend/v1beta1/backendv1beta1connect"
	backendv1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/backend/v1beta1"
	"github.com/bufbuild/connect-go"
	"github.com/cenkalti/backoff/v4"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/rjeczalik/notify"
	"google.golang.org/protobuf/proto"
)

// Constructs an in-memory store that fetches configs from a local git repo at the given path.
// If api key is empty, the store runs in local (offline) mode, and does not communicate with Lekko.
func NewGitStore(
	ctx context.Context,
	apiKey, url, ownerName, repoName, path string,
	client *http.Client,
	port int32,
) (Store, error) {
	var distClient backendv1beta1connect.DistributionServiceClient
	if len(apiKey) > 0 {
		distClient = backendv1beta1connect.NewDistributionServiceClient(client, url, connect.WithGRPC())
	}
	fs := osfs.New(path)
	gitfs, err := fs.Chroot(git.GitDirName)
	if err != nil {
		return nil, err
	}
	storer := filesystem.NewStorage(gitfs, cache.NewObjectLRUDefault())
	return newGitStore(
		ctx,
		apiKey, ownerName, repoName,
		storer, fs, distClient,
		eventsBatchSize, true,
		port,
	)
}

func newGitStore(
	ctx context.Context,
	apiKey, ownerName, repoName string,
	storer storage.Storer, fs billy.Filesystem,
	distClient backendv1beta1connect.DistributionServiceClient,
	eventsBatchSize int, watch bool, port int32,
) (*gitStore, error) {
	g := &gitStore{
		distClient: distClient,
		store:      newStore(ownerName, repoName),
		repoKey: &backendv1beta1.RepositoryKey{
			OwnerName: ownerName,
			RepoName:  repoName,
		},
		apiKey: apiKey,
		storer: storer,
		fs:     fs,
	}
	if distClient != nil {
		// register with lekko backend
		sessionKey, err := g.registerWithBackoff(ctx)
		if err != nil {
			// silently fail on registration errors in git mode
			log.Printf("error registering lekko client: %v", err)
		}
		g.sessionKey = sessionKey
		g.eb = newEventBatcher(ctx, distClient, g.sessionKey, g.apiKey, eventsBatchSize)
	}
	if _, err := g.load(ctx); err != nil {
		return nil, err
	}
	g.server = newSDKServer(port, g.store)
	bgCtx, bgCancel := noInheritCancel(ctx)
	g.cancel = bgCancel
	if watch {
		if err := g.startWatcher(bgCtx); err != nil {
			return nil, err
		}
	}
	return g, nil
}

type gitStore struct {
	distClient         backendv1beta1connect.DistributionServiceClient
	store              *store
	repoKey            *backendv1beta1.RepositoryKey
	apiKey, sessionKey string
	wg                 sync.WaitGroup
	cancel             context.CancelFunc
	eb                 *eventBatcher
	fsChan             chan<- notify.EventInfo
	storer             storage.Storer
	fs                 billy.Filesystem
	server             *sdkServer
}

func (g *gitStore) registerWithBackoff(ctx context.Context) (string, error) {
	// registration should not take forever
	ctx, cancel := context.WithTimeout(ctx, defaultRPCDeadline)
	defer cancel()
	req := connect.NewRequest(&backendv1beta1.RegisterClientRequest{
		RepoKey:        g.repoKey,
		NamespaceList:  []string{}, // register all namespaces
		SidecarVersion: sdkVersion,
	})
	setAPIKey(req, g.apiKey)
	var resp *connect.Response[backendv1beta1.RegisterClientResponse]
	var err error
	op := func() error {
		resp, err = g.distClient.RegisterClient(ctx, req)
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

func (g *gitStore) load(ctx context.Context) (bool, error) {
	repo, err := newRepository(g.storer, g.fs)
	if err != nil {
		return false, err
	}
	contents, err := repo.getContents()
	if err != nil {
		return false, err
	}
	return g.store.update(contents)
}

func (g *gitStore) startWatcher(ctx context.Context) error {
	fsChan := make(chan notify.EventInfo, 100)
	if err := notify.Watch(filepath.Join(g.fs.Root(), "..."), fsChan, notify.All); err != nil {
		return err
	}
	g.fsChan = fsChan
	g.wg.Add(1)
	go g.watch(ctx, fsChan)
	return nil
}

func (g *gitStore) watch(ctx context.Context, fsChan <-chan notify.EventInfo) {
	defer g.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case <-fsChan:
			_, err := g.load(ctx)
			if err != nil {
				log.Printf("error loading contents: %v\n", err)
			}
		}
	}
}

func (g *gitStore) Evaluate(key string, namespace string, lekkoContext map[string]interface{}, dest proto.Message) error {
	cfg, rp, err := g.store.evaluateType(key, namespace, lekkoContext, dest)
	if err != nil {
		return err
	}
	if g.eb != nil {
		g.eb.track(&backendv1beta1.FlagEvaluationEvent{
			RepoKey:       g.repoKey,
			CommitSha:     cfg.CommitSHA,
			FeatureSha:    cfg.ConfigSHA,
			NamespaceName: namespace,
			FeatureName:   cfg.Config.GetKey(),
			ContextKeys:   toContextKeysProto(lekkoContext),
			ResultPath:    toResultPathProto(rp),
		})
	}
	return nil
}

func (g *gitStore) Close(ctx context.Context) error {
	// cancel any ongoing background loops
	g.cancel()
	g.server.close(ctx)
	g.eb.close(ctx)
	if g.fsChan != nil {
		notify.Stop(g.fsChan)
		close(g.fsChan)
	}
	// wait for background work to complete
	g.wg.Wait()
	if g.distClient != nil {
		req := connect.NewRequest(&backendv1beta1.DeregisterClientRequest{
			SessionKey: g.sessionKey,
		})
		setAPIKey(req, g.apiKey)
		if _, err := g.distClient.DeregisterClient(ctx, req); err != nil {
			log.Printf("error deregistering lekko client: %v", err)
		}
	}
	return nil
}
