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
	namespace string
	provider  Provider
}

func NewClient(namespace string, provider Provider) *Client {
	return &Client{namespace, provider}
}

func (c *Client) GetBool(ctx context.Context, key string) (bool, error) {
	return c.provider.GetBoolFeature(ctx, key, c.namespace)
}

func (c *Client) GetProto(ctx context.Context, key string, result proto.Message) error {
	return c.provider.GetProtoFeature(ctx, key, c.namespace, result)
}

func (c *Client) GetJSON(ctx context.Context, key string, result interface{}) error {
	return c.provider.GetJSONFeature(ctx, key, c.namespace, result)
}
