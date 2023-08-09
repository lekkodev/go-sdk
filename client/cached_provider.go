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
	"github.com/pkg/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	minUpdateInterval = time.Second
)

// Constructs a provider that refreshes configs from Lekko backend repeatedly in the background,
// caching the configs in-memory. url is optional, it defaults to Lekko backend.
func CachedAPIProvider(ctx context.Context, apiKey, url string, repoKey RepositoryKey, updateInterval time.Duration) (Provider, error) {
	if len(apiKey) == 0 {
		return nil, errors.New("api key is required")
	}
	if len(repoKey.OwnerName) == 0 || len(repoKey.RepoName) == 0 {
		return nil, errors.New("missing repo key information")
	}
	if updateInterval.Seconds() < minUpdateInterval.Seconds() {
		return nil, errors.Errorf("update interval too small, minimum %v", minUpdateInterval)
	}
	if len(url) == 0 {
		url = defaultAPIURL
	}
	backend, err := memory.NewBackendStore(ctx, apiKey, url, repoKey.OwnerName, repoKey.RepoName, updateInterval)
	if err != nil {
		return nil, err
	}
	return &inMemoryProvider{
		store: backend,
	}, nil
}

// Reads configuration from a git repository on-disk. This provider will remain up to date with
// changes made to the git repository on-disk. If on-disk contents change, this provider's internal
// state will be updated without restart.
// This provider requires an api key to communicate with Lekko.
// url is optional, it defaults to Lekko backend.
// Provide the path to the root of the repository. 'path/.git/' should be a valid directory.
func CachedGitProvider(ctx context.Context, path, apiKey, url string, repoKey RepositoryKey) (Provider, error) {
	if len(apiKey) == 0 {
		return nil, errors.New("api key is required")
	}
	if len(repoKey.OwnerName) == 0 || len(repoKey.RepoName) == 0 {
		return nil, errors.New("missing repo key information")
	}
	if len(url) == 0 {
		url = defaultAPIURL
	}
	gitStore, err := memory.NewGitStore(ctx, apiKey, url, repoKey.OwnerName, repoKey.RepoName, path)
	if err != nil {
		return nil, err
	}
	return &inMemoryProvider{
		store: gitStore,
	}, nil
}

// Reads configuration from a git repository on-disk. This provider will remain up to date with
// changes made to the git repository on-disk. If on-disk contents change, this provider's internal
// state will be updated without restart.
// This provider does not require an api key and can be used while developing locally.
// Provide the path to the root of the repository. 'path/.git/' should be a valid directory.
func LocalCachedGitProvider(ctx context.Context, path string, repoKey RepositoryKey) (Provider, error) {
	if len(repoKey.OwnerName) == 0 || len(repoKey.RepoName) == 0 {
		return nil, errors.New("missing repo key information")
	}
	gitStore, err := memory.NewGitStore(ctx, "", "", repoKey.OwnerName, repoKey.RepoName, path)
	if err != nil {
		return nil, err
	}
	return &inMemoryProvider{
		store: gitStore,
	}, nil
}

type inMemoryProvider struct {
	store memory.Store
}

// Close implements Provider.
func (im *inMemoryProvider) Close(ctx context.Context) error {
	return im.store.Close(ctx)
}

// GetBoolFeature implements Provider.
func (im *inMemoryProvider) GetBoolFeature(ctx context.Context, key string, namespace string) (bool, error) {
	dest := &wrapperspb.BoolValue{}
	if err := im.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return false, err
	}
	return dest.GetValue(), nil
}

// GetFloatFeature implements Provider.
func (im *inMemoryProvider) GetFloatFeature(ctx context.Context, key string, namespace string) (float64, error) {
	dest := &wrapperspb.DoubleValue{}
	if err := im.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return 0, err
	}
	return dest.GetValue(), nil
}

// GetIntFeature implements Provider.
func (im *inMemoryProvider) GetIntFeature(ctx context.Context, key string, namespace string) (int64, error) {
	dest := &wrapperspb.Int64Value{}
	if err := im.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return 0, err
	}
	return dest.GetValue(), nil
}

// GetJSONFeature implements Provider.
func (im *inMemoryProvider) GetJSONFeature(ctx context.Context, key string, namespace string, result interface{}) error {
	dest := &structpb.Value{}
	if err := im.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return err
	}
	bytes, err := dest.MarshalJSON()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bytes, result); err != nil {
		return errors.Wrapf(err, "failed to unmarshal json into go type %T", result)
	}
	return nil
}

// GetProtoFeature implements Provider.
func (im *inMemoryProvider) GetProtoFeature(ctx context.Context, key string, namespace string, result protoreflect.ProtoMessage) error {
	return im.store.Evaluate(key, namespace, fromContext(ctx), result)
}

// GetStringFeature implements Provider.
func (im *inMemoryProvider) GetStringFeature(ctx context.Context, key string, namespace string) (string, error) {
	dest := &wrapperspb.StringValue{}
	if err := im.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return "", err
	}
	return dest.GetValue(), nil
}
