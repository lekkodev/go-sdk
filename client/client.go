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
	"log"
	"net/http"

	"github.com/lekkodev/cli/pkg/gen/proto/go-connect/lekko/backend/v1beta1/backendv1beta1connect"
	backendv1beta1 "github.com/lekkodev/cli/pkg/gen/proto/go/lekko/backend/v1beta1"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/bufbuild/connect-go"
)

const (
	LekkoURL          = "https://grpc.lekko.dev:443"
	LekkoAPIKeyHeader = "apikey"
)

type Client struct {
	apikey, namespace string
	lekkoClient       backendv1beta1connect.ConfigurationServiceClient
}

func NewClient(apikey, namespace string) *Client {
	return &Client{
		apikey:      apikey,
		namespace:   namespace,
		lekkoClient: backendv1beta1connect.NewConfigurationServiceClient(http.DefaultClient, LekkoURL),
	}
}

func (c *Client) GetBool(ctx context.Context, key string, defaultValue bool) bool {
	lc, err := toProto(fromContext(ctx))
	if err != nil {
		log.Printf("error transforming context: %v", err)
		return defaultValue
	}
	req := connect.NewRequest(&backendv1beta1.GetBoolValueRequest{
		Key:       key,
		Namespace: c.namespace,
		Context:   lc,
	})
	req.Header().Set(LekkoAPIKeyHeader, c.apikey)
	resp, err := c.lekkoClient.GetBoolValue(ctx, req)
	if err != nil {
		log.Printf("error hitting lekko backend: resp: %v, err: %v\n", resp, err)
		return defaultValue
	}
	return resp.Msg.GetValue()
}

func (c *Client) GetProto(ctx context.Context, key string, defaultValue *anypb.Any) *anypb.Any {
	lc, err := toProto(fromContext(ctx))
	if err != nil {
		log.Printf("error transforming context: %v", err)
		return defaultValue
	}
	req := connect.NewRequest(&backendv1beta1.GetProtoValueRequest{
		Key:       key,
		Namespace: c.namespace,
		Context:   lc,
	})
	req.Header().Set(LekkoAPIKeyHeader, c.apikey)
	resp, err := c.lekkoClient.GetProtoValue(ctx, req)
	if err != nil {
		log.Printf("error hitting lekko backend: resp: %v, err: %v\n", resp, err)
		return defaultValue
	}
	return resp.Msg.GetValue()
}
