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

type InMemoryProviderOptions struct {
	// API key used to communicate with Lekko.
	// lekko_**********
	APIKey string
	// Repository Key of the configuration repository you wish to read
	// configs from.
	RepositoryKey RepositoryKey
	// Interval at which you want to refresh configs from Lekko.
	// Recommend 15 to 30 seconds.
	UpdateInterval time.Duration
}

type inMemoryProviderType string

const (
	inMemoryProviderTypeBackend inMemoryProviderType = "backend"

	minUpdateInterval = time.Second
)

// Constructs a provider that refreshes configs from Lekko backend repeatedly in the background,
// caching the configs in-memory.
func BackendInMemoryProvider(ctx context.Context, opts *InMemoryProviderOptions) (Provider, error) {
	if err := opts.validate(inMemoryProviderTypeBackend); err != nil {
		return nil, err
	}
	backend, err := memory.NewBackendStore(ctx, opts.APIKey, defaultAPIURL, opts.RepositoryKey.OwnerName, opts.RepositoryKey.RepoName, opts.UpdateInterval)
	if err != nil {
		return nil, err
	}
	return &inMemoryProvider{
		store: backend,
	}, nil
}

func (opts *InMemoryProviderOptions) validate(t inMemoryProviderType) error {
	if opts == nil {
		return errors.New("options cannot be nil")
	}
	if t == inMemoryProviderTypeBackend && len(opts.APIKey) == 0 {
		return errors.New("no apikey given for backend provider")
	}
	if len(opts.RepositoryKey.OwnerName) == 0 || len(opts.RepositoryKey.RepoName) == 0 {
		return errors.New("incomplete repository key given")
	}
	if opts.UpdateInterval.Seconds() < minUpdateInterval.Seconds() {
		return errors.Errorf("update interval too small, minimum %s", minUpdateInterval)
	}
	return nil
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
