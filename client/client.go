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

	"github.com/lekkodev/core/pkg/gen/proto/go-connect/lekko/backend/v1beta1/backendv1beta1connect"
	backendv1beta1 "github.com/lekkodev/core/pkg/gen/proto/go/lekko/backend/v1beta1"

	"github.com/bufbuild/connect-go"
)

type Client struct {
	orgName, namespace string
	lekkoClient        backendv1beta1connect.ConfigurationServiceClient
}

func NewClient(orgName, namespace string) *Client {
	return &Client{
		orgName:     orgName,
		namespace:   namespace,
		lekkoClient: backendv1beta1connect.NewConfigurationServiceClient(http.DefaultClient, "http://localhost:8080/"),
	}
}

func (c *Client) GetBool(ctx context.Context, key string, defaultValue bool) bool {
	resp, err := c.lekkoClient.GetBoolValue(ctx, connect.NewRequest(&backendv1beta1.GetBoolValueRequest{
		Key:       key,
		Namespace: c.namespace,
	}))
	if err != nil {
		log.Printf("error hitting lekko backend: %v\n", err)
		return defaultValue
	}
	return resp.Msg.GetValue()
}
