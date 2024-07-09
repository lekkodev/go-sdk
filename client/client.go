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
	"fmt"
	"os"

	"github.com/lekkodev/go-sdk/pkg/debug"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Client allows retrieving lekko configs. The appropriate method must
// be called based on the type of the config. For instance, calling
// GetBool on an Int config will result in an error.
type Client interface {
	GetBool(ctx context.Context, namespace, key string) (bool, error)
	GetInt(ctx context.Context, namespace, key string) (int64, error)
	GetFloat(ctx context.Context, namespace, key string) (float64, error)
	GetString(ctx context.Context, namespace, key string) (string, error)
	// result should be an empty Go variable that the JSON config
	// can be unmarshalled into. If an error is returned, the data
	// stored in result is unpredictable and should not be relied upon.
	GetJSON(ctx context.Context, namespace, key string, result interface{}) error
	// result should be an empty proto message that the proto config
	// can be unmarshalled into. If an error is returned, the data
	// stored in result is unpredictable and should not be relied upon.
	GetProto(ctx context.Context, namespace, key string, result proto.Message) error
	GetAny(ctx context.Context, namespace string, key string) (protoreflect.ProtoMessage, error)
	Close(ctx context.Context) error
}

type CloseFunc func(context.Context) error

// A function is returned to close the client. It is also strongly recommended
// to call this when the program is exiting or the lekko provider is no longer needed.
func NewClient(provider Provider) (Client, CloseFunc) {
	return newClient(provider, nil)
}

type ClientOptions struct {
	// Arbitrary keys and values for rules evaluation that are
	// injected into the context at startup. Context keys passed
	// at runtime will override these values in case of conflict.
	StartupContext map[string]interface{}
}

func (o ClientOptions) NewClient(provider Provider) (Client, CloseFunc) {
	return newClient(provider, o.StartupContext)
}

func NewClientFromEnv(ctx context.Context, repoOwner, repoName string, opts ...ProviderOption) (Client, CloseFunc) {
	apiKey := os.Getenv("LEKKO_API_KEY")
	var provider Provider
	if apiKey == "" {
		debug.LogInfo("LEKKO_API_KEY environment variable is not set, in-code fallback will be used")
		provider = &noOpProvider{}
	} else {
		opts = append(opts, WithAPIKey(apiKey))
		var err error
		provider, err = CachedAPIProvider(ctx, &RepositoryKey{
			OwnerName: repoOwner,
			RepoName:  repoName,
		}, opts...)
		if err != nil {
			debug.LogError("Error connecting to Lekko, in-code fallback will be used", "opts", fmt.Sprintf("%v", opts), "err", err)
			provider = &noOpProvider{}
		} else {
			debug.LogInfo("Connected to Lekko", "repository", fmt.Sprintf("%s/%s", repoOwner, repoName), "opts", opts)
		}
	}
	return NewClient(provider)
}

type client struct {
	provider       Provider
	startupContext map[string]interface{}
}

func newClient(provider Provider, startupCtx map[string]interface{}) (*client, CloseFunc) {
	return &client{provider: provider, startupContext: startupCtx}, provider.Close
}

func (c *client) GetBool(ctx context.Context, namespace, key string) (bool, error) {
	return c.provider.GetBool(c.wrap(ctx), key, namespace)
}

func (c *client) GetInt(ctx context.Context, namespace, key string) (int64, error) {
	return c.provider.GetInt(c.wrap(ctx), key, namespace)
}

func (c *client) GetFloat(ctx context.Context, namespace, key string) (float64, error) {
	return c.provider.GetFloat(c.wrap(ctx), key, namespace)
}

func (c *client) GetString(ctx context.Context, namespace, key string) (string, error) {
	return c.provider.GetString(c.wrap(ctx), key, namespace)
}

func (c *client) GetProto(ctx context.Context, namespace, key string, result proto.Message) error {
	return c.provider.GetProto(c.wrap(ctx), key, namespace, result)
}

func (c *client) GetJSON(ctx context.Context, namespace, key string, result interface{}) error {
	return c.provider.GetJSON(c.wrap(ctx), key, namespace, result)
}

func (c *client) GetAny(ctx context.Context, namespace string, key string) (protoreflect.ProtoMessage, error) {
	return c.provider.GetAny(c.wrap(ctx), key, namespace)
}

func (c *client) wrap(ctx context.Context) context.Context {
	if c.startupContext == nil {
		return ctx
	}

	return Merge(ctx, c.startupContext)
}

func (c *client) Close(ctx context.Context) error {
	return c.provider.Close(ctx)
}
