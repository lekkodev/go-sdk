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

type Client struct {
	namespace   string
	provider    Provider
	environment *string
}

func NewClient(namespace string, provider Provider) *Client {
	return &Client{namespace, provider, nil}
}

const lekkoEnvironmentContextKey = "lekko_env"

type ClientOptions struct {
	// Lekko namespace to read configurations from
	Namespace string
	// Runtime environment used for rules evaluation. Can be any
	// freeform string, but most common values are 'dev', 'staging',
	// 'prod', etc.
	Environment string
}

func (o ClientOptions) NewClient(provider Provider) *Client {
	return &Client{o.Namespace, provider, &o.Environment}
}

func (c *Client) GetBool(ctx context.Context, key string) (bool, error) {
	return c.provider.GetBoolFeature(c.wrap(ctx), key, c.namespace)
}

func (c *Client) GetProto(ctx context.Context, key string, result proto.Message) error {
	return c.provider.GetProtoFeature(c.wrap(ctx), key, c.namespace, result)
}

func (c *Client) GetJSON(ctx context.Context, key string, result interface{}) error {
	return c.provider.GetJSONFeature(c.wrap(ctx), key, c.namespace, result)
}

func (c *Client) wrap(ctx context.Context) context.Context {
	if c.environment == nil {
		return ctx
	}
	return Add(ctx, lekkoEnvironmentContextKey, *c.environment)
}
