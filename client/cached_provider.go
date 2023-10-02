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

package client

import (
	"context"
	"encoding/json"
	"time"

	"github.com/lekkodev/go-sdk/internal/memory"
	"github.com/lekkodev/go-sdk/internal/version"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	minUpdateInterval     = time.Second
	defaultUpdateInterval = 15 * time.Second
)

// Constructs a provider that refreshes configs from Lekko backend repeatedly in the background,
// caching the configs in-memory.
func CachedAPIProvider(
	ctx context.Context,
	rk *RepositoryKey,
	opts ...ProviderOption,
) (Provider, error) {
	cfg := &providerConfig{}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	withFallbackURL(defaultAPIURL).apply(cfg)
	withRepositoryKey(rk).apply(cfg)
	if err := cfg.validate(ctx); err != nil {
		return nil, err
	}

	backend, err := memory.NewBackendStore(
		ctx,
		cfg.apiKey, cfg.url,
		cfg.repoKey.OwnerName, cfg.repoKey.RepoName,
		cfg.getHTTPClient(),
		cfg.updateInterval, cfg.serverPort,
		version.SDKVersion,
	)
	if err != nil {
		return nil, err
	}
	return &cachedProvider{
		store: backend,
	}, nil
}

// Reads configuration from a git repository on-disk. This provider will remain up to date with
// changes made to the git repository on-disk. If on-disk contents change, this provider's internal
// state will be updated without restart.
// If api key and url are provided, this provider will send metrics back to lekko.
// Provide the path to the root of the repository. 'path/.git/' should be a valid directory.
func CachedGitFsProvider(
	ctx context.Context,
	repoKey *RepositoryKey,
	path string,
	opts ...ProviderOption,
) (Provider, error) {
	cfg := &providerConfig{}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	withRepositoryKey(repoKey).apply(cfg)
	if len(cfg.apiKey) > 0 {
		withFallbackURL(defaultAPIURL).apply(cfg)
	}
	if err := cfg.validate(ctx); err != nil {
		return nil, err
	}
	gitStore, err := memory.NewGitStore(
		ctx,
		cfg.apiKey, cfg.url,
		cfg.repoKey.OwnerName, cfg.repoKey.RepoName,
		path, cfg.getHTTPClient(), cfg.serverPort,
		version.SDKVersion,
	)
	if err != nil {
		return nil, err
	}
	return &cachedProvider{
		store: gitStore,
	}, nil
}

type cachedProvider struct {
	store memory.Store
}

func (cp *cachedProvider) Close(ctx context.Context) error {
	return cp.store.Close(ctx)
}

func (cp *cachedProvider) GetBool(ctx context.Context, key string, namespace string) (bool, Metadata, error) {
	dest := &wrapperspb.BoolValue{}
	cfg, err := cp.store.Evaluate(key, namespace, fromContext(ctx), dest)
	if err != nil {
		return false, Metadata{}, err
	}
	return dest.GetValue(), Metadata{LastUpdateCommitSHA: cfg.LastUpdateCommitSHA}, nil
}

func (cp *cachedProvider) GetFloat(ctx context.Context, key string, namespace string) (float64, Metadata, error) {
	dest := &wrapperspb.DoubleValue{}
	cfg, err := cp.store.Evaluate(key, namespace, fromContext(ctx), dest)
	if err != nil {
		return 0, Metadata{}, err
	}
	return dest.GetValue(), Metadata{LastUpdateCommitSHA: cfg.LastUpdateCommitSHA}, nil
}

func (cp *cachedProvider) GetInt(ctx context.Context, key string, namespace string) (int64, Metadata, error) {
	dest := &wrapperspb.Int64Value{}
	cfg, err := cp.store.Evaluate(key, namespace, fromContext(ctx), dest)
	if err != nil {
		return 0, Metadata{}, err
	}
	return dest.GetValue(), Metadata{LastUpdateCommitSHA: cfg.LastUpdateCommitSHA}, nil
}

func (cp *cachedProvider) GetJSON(ctx context.Context, key string, namespace string, result interface{}) (Metadata, error) {
	dest := &structpb.Value{}
	cfg, err := cp.store.Evaluate(key, namespace, fromContext(ctx), dest)
	if err != nil {
		return Metadata{}, err
	}
	bytes, err := dest.MarshalJSON()
	if err != nil {
		return Metadata{}, err
	}
	if err := json.Unmarshal(bytes, result); err != nil {
		return Metadata{}, errors.Wrapf(err, "failed to unmarshal json into go type %T", result)
	}
	return Metadata{LastUpdateCommitSHA: cfg.LastUpdateCommitSHA}, nil
}

func (cp *cachedProvider) GetProto(ctx context.Context, key string, namespace string, result protoreflect.ProtoMessage) (Metadata, error) {
	cfg, err := cp.store.Evaluate(key, namespace, fromContext(ctx), result)
	if err != nil {
		return Metadata{}, err
	}
	return Metadata{LastUpdateCommitSHA: cfg.LastUpdateCommitSHA}, nil
}

func (cp *cachedProvider) GetString(ctx context.Context, key string, namespace string) (string, Metadata, error) {
	dest := &wrapperspb.StringValue{}
	cfg, err := cp.store.Evaluate(key, namespace, fromContext(ctx), dest)
	if err != nil {
		return "", Metadata{}, err
	}
	return dest.GetValue(), Metadata{LastUpdateCommitSHA: cfg.LastUpdateCommitSHA}, nil
}
