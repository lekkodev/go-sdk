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

type Client interface {
	GetBool(ctx context.Context, key string) (bool, error)
	GetBoolDefault(ctx context.Context, key string, defaultValue bool) bool
	GetInt(ctx context.Context, key string) (int64, error)
	GetIntDefault(ctx context.Context, key string, defaultValue int64) int64
	GetFloat(ctx context.Context, key string) (float64, error)
	GetFloatDefault(ctx context.Context, key string, defaultValue float64) float64
	GetString(ctx context.Context, key string) (string, error)
	GetStringDefault(ctx context.Context, key string, defaultValue string) string
	GetJSON(ctx context.Context, key string, result interface{}) error
	GetJSONDefault(ctx context.Context, key string, result interface{})
	GetProto(ctx context.Context, key string, result proto.Message) error
	GetProtoDefault(ctx context.Context, key string, result proto.Message)
}

type CloseFunc func(context.Context) error

// A function is returned to close the client. It is also strongly recommended
// to call this when the program is exiting or the lekko provider is no longer needed.
func NewClient(namespace string, provider Provider) (Client, CloseFunc) {
	return newClient(namespace, provider, nil, nil)
}

type Logger interface {
	Printf(format string, v ...any)
}

type ClientOptions struct {
	// Lekko namespace to read configurations from
	Namespace string
	// Arbitrary keys and values for rules evaluation that are
	// injected into the context at startup. Context keys passed
	// at runtime will override these values in case of conflict.
	StartupContext map[string]interface{}
	// Logger that will log errors when using GetXDefault methods.
	Logger Logger
}

func (o ClientOptions) NewClient(provider Provider) (Client, CloseFunc) {
	return newClient(o.Namespace, provider, o.StartupContext, o.Logger)
}

type client struct {
	namespace      string
	provider       Provider
	startupContext map[string]interface{}
	logger         Logger
}

func newClient(namespace string, provider Provider, startupCtx map[string]interface{}, logger Logger) (*client, CloseFunc) {
	return &client{namespace: namespace, provider: provider, startupContext: startupCtx, logger: logger}, provider.Close
}

func (c *client) GetBool(ctx context.Context, key string) (bool, error) {
	return c.provider.GetBoolFeature(c.wrap(ctx), key, c.namespace)
}

func (c *client) GetBoolDefault(ctx context.Context, key string, defaultValue bool) bool {
	if ret, err := c.GetBool(ctx, key); err != nil {
		c.log(key, err)
		return defaultValue
	} else {
		return ret
	}
}

func (c *client) GetInt(ctx context.Context, key string) (int64, error) {
	return c.provider.GetIntFeature(c.wrap(ctx), key, c.namespace)
}

func (c *client) GetIntDefault(ctx context.Context, key string, defaultValue int64) int64 {
	if ret, err := c.GetInt(ctx, key); err != nil {
		c.log(key, err)
		return defaultValue
	} else {
		return ret
	}
}

func (c *client) GetFloat(ctx context.Context, key string) (float64, error) {
	return c.provider.GetFloatFeature(c.wrap(ctx), key, c.namespace)
}

func (c *client) GetFloatDefault(ctx context.Context, key string, defaultValue float64) float64 {
	if ret, err := c.GetFloat(ctx, key); err != nil {
		c.log(key, err)
		return defaultValue
	} else {
		return ret
	}
}

func (c *client) GetString(ctx context.Context, key string) (string, error) {
	return c.provider.GetStringFeature(c.wrap(ctx), key, c.namespace)
}

func (c *client) GetStringDefault(ctx context.Context, key string, defaultValue string) string {
	if ret, err := c.GetString(ctx, key); err != nil {
		c.log(key, err)
		return defaultValue
	} else {
		return ret
	}
}

func (c *client) GetProto(ctx context.Context, key string, result proto.Message) error {
	return c.provider.GetProtoFeature(c.wrap(ctx), key, c.namespace, result)
}

func (c *client) GetProtoDefault(ctx context.Context, key string, result proto.Message) {
	if err := c.GetProto(ctx, key, result); err != nil {
		c.log(key, err)
	}
}

func (c *client) GetJSON(ctx context.Context, key string, result interface{}) error {
	return c.provider.GetJSONFeature(c.wrap(ctx), key, c.namespace, result)
}

func (c *client) GetJSONDefault(ctx context.Context, key string, result interface{}) {
	if err := c.GetJSON(ctx, key, result); err != nil {
		c.log(key, err)
	}
}

func (c *client) wrap(ctx context.Context) context.Context {
	if c.startupContext == nil {
		return ctx
	}

	return Merge(ctx, c.startupContext)
}

func (c *client) log(key string, err error) {
	if c.logger == nil {
		return
	}
	c.logger.Printf("%s/%s: %v", c.namespace, key, err)
}
