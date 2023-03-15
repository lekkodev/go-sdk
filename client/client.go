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

	"google.golang.org/protobuf/proto"
)

// Client allows retrieving lekko features. The appropriate method must
// be called based on the type of the feature. For instance, calling
// GetBool on an Int feature will result in an error.
type Client interface {
	GetBool(ctx context.Context, key string) (bool, error)
	GetInt(ctx context.Context, key string) (int64, error)
	GetFloat(ctx context.Context, key string) (float64, error)
	GetString(ctx context.Context, key string) (string, error)
	// result should be an empty Go variable that the JSON feature
	// can be unmarshalled into. If an error is returned, the data
	// stored in result is unpredictable and should not be relied upon.
	GetJSON(ctx context.Context, key string, result interface{}) error
	// result should be an empty proto message that the proto feature
	// can be unmarshalled into. If an error is returned, the data
	// stored in result is unpredictable and should not be relied upon.
	GetProto(ctx context.Context, key string, result proto.Message) error
}

type CloseFunc func(context.Context) error

// A function is returned to close the client. It is also strongly recommended
// to call this when the program is exiting or the lekko provider is no longer needed.
func NewClient(namespace string, provider Provider) (Client, CloseFunc) {
	return newClient(namespace, provider, nil)
}

type ClientOptions struct {
	// Lekko namespace to read configurations from
	Namespace string
	// Arbitrary keys and values for rules evaluation that are
	// injected into the context at startup. Context keys passed
	// at runtime will override these values in case of conflict.
	StartupContext map[string]interface{}
}

func (o ClientOptions) NewClient(provider Provider) (Client, CloseFunc) {
	return newClient(o.Namespace, provider, o.StartupContext)
}

type client struct {
	namespace      string
	provider       Provider
	startupContext map[string]interface{}
}

func newClient(namespace string, provider Provider, startupCtx map[string]interface{}) (*client, CloseFunc) {
	return &client{namespace: namespace, provider: provider, startupContext: startupCtx}, provider.Close
}

func (c *client) GetBool(ctx context.Context, key string) (bool, error) {
	return c.provider.GetBoolFeature(c.wrap(ctx), key, c.namespace)
}

func (c *client) GetInt(ctx context.Context, key string) (int64, error) {
	return c.provider.GetIntFeature(c.wrap(ctx), key, c.namespace)
}

func (c *client) GetFloat(ctx context.Context, key string) (float64, error) {
	return c.provider.GetFloatFeature(c.wrap(ctx), key, c.namespace)
}

func (c *client) GetString(ctx context.Context, key string) (string, error) {
	return c.provider.GetStringFeature(c.wrap(ctx), key, c.namespace)
}

func (c *client) GetProto(ctx context.Context, key string, result proto.Message) error {
	return c.provider.GetProtoFeature(c.wrap(ctx), key, c.namespace, result)
}

func (c *client) GetJSON(ctx context.Context, key string, result interface{}) error {
	return c.provider.GetJSONFeature(c.wrap(ctx), key, c.namespace, result)
}

func (c *client) wrap(ctx context.Context) context.Context {
	if c.startupContext == nil {
		return ctx
	}

	return Merge(ctx, c.startupContext)
}
