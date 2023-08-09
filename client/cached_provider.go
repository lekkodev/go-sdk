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
// caching the configs in-memory. ConnectionOptions is a required argument.
func CachedAPIProvider(ctx context.Context, co *ConnectionOptions, repoKey RepositoryKey, updateInterval time.Duration) (Provider, error) {
	if err := co.validate(true); err != nil {
		return nil, err
	}
	if len(repoKey.OwnerName) == 0 || len(repoKey.RepoName) == 0 {
		return nil, errors.New("missing repo key information")
	}
	if updateInterval.Seconds() < minUpdateInterval.Seconds() {
		return nil, errors.Errorf("update interval too small, minimum %v", minUpdateInterval)
	}
	backend, err := memory.NewBackendStore(ctx, co.getAPIKey(), co.getURL(), repoKey.OwnerName, repoKey.RepoName, updateInterval)
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
// If ConnectionOptions are provided, this provider will send metrics back to lekko.
// Provide the path to the root of the repository. 'path/.git/' should be a valid directory.
func CachedGitProvider(ctx context.Context, path string, co *ConnectionOptions, repoKey RepositoryKey) (Provider, error) {
	if err := co.validate(false); err != nil {
		return nil, err
	}
	if len(repoKey.OwnerName) == 0 || len(repoKey.RepoName) == 0 {
		return nil, errors.New("missing repo key information")
	}
	gitStore, err := memory.NewGitStore(ctx, co.getAPIKey(), co.getURL(), repoKey.OwnerName, repoKey.RepoName, path)
	if err != nil {
		return nil, err
	}
	return &cachedProvider{
		store: gitStore,
	}, nil
}

// Arguments needed to connect to Lekko.
type ConnectionOptions struct {
	// Lekko API key (lekko_*****)
	APIKey string
	// URL to connect to. If empty, connects to Lekko's backend.
	URL string
}

func (co *ConnectionOptions) validate(required bool) error {
	if !required && co == nil {
		return nil
	}
	if len(co.APIKey) == 0 {
		return errors.New("api key is required")
	}
	if len(co.URL) == 0 {
		co.URL = defaultAPIURL
	}
	return nil
}

func (co *ConnectionOptions) getAPIKey() string {
	if co == nil {
		return ""
	}
	return co.APIKey
}

func (co *ConnectionOptions) getURL() string {
	if co == nil {
		return ""
	}
	return co.URL
}

type cachedProvider struct {
	store memory.Store
}

// Close implements Provider.
func (cp *cachedProvider) Close(ctx context.Context) error {
	return cp.store.Close(ctx)
}

// GetBoolFeature implements Provider.
func (cp *cachedProvider) GetBoolFeature(ctx context.Context, key string, namespace string) (bool, error) {
	dest := &wrapperspb.BoolValue{}
	if err := cp.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return false, err
	}
	return dest.GetValue(), nil
}

// GetFloatFeature implements Provider.
func (cp *cachedProvider) GetFloatFeature(ctx context.Context, key string, namespace string) (float64, error) {
	dest := &wrapperspb.DoubleValue{}
	if err := cp.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return 0, err
	}
	return dest.GetValue(), nil
}

// GetIntFeature implements Provider.
func (cp *cachedProvider) GetIntFeature(ctx context.Context, key string, namespace string) (int64, error) {
	dest := &wrapperspb.Int64Value{}
	if err := cp.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return 0, err
	}
	return dest.GetValue(), nil
}

// GetJSONFeature implements Provider.
func (cp *cachedProvider) GetJSONFeature(ctx context.Context, key string, namespace string, result interface{}) error {
	dest := &structpb.Value{}
	if err := cp.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
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
func (cp *cachedProvider) GetProtoFeature(ctx context.Context, key string, namespace string, result protoreflect.ProtoMessage) error {
	return cp.store.Evaluate(key, namespace, fromContext(ctx), result)
}

// GetStringFeature implements Provider.
func (cp *cachedProvider) GetStringFeature(ctx context.Context, key string, namespace string) (string, error) {
	dest := &wrapperspb.StringValue{}
	if err := cp.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return "", err
	}
	return dest.GetValue(), nil
}
