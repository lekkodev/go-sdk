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
	"fmt"
	"net/http"

	"github.com/lekkodev/cli/pkg/gen/proto/go-connect/lekko/backend/v1beta1/backendv1beta1connect"
	backendv1beta1 "github.com/lekkodev/cli/pkg/gen/proto/go/lekko/backend/v1beta1"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"

	"github.com/bufbuild/connect-go"
)

const (
	LekkoURL          = "https://grpc.lekko.dev"
	LekkoAPIKeyHeader = "apikey"
)

func NewAPIProvider(apiKey string) Provider {
	return &apiProvider{
		apikey:      apiKey,
		lekkoClient: backendv1beta1connect.NewConfigurationServiceClient(http.DefaultClient, LekkoURL),
	}
}

type apiProvider struct {
	apikey      string
	lekkoClient backendv1beta1connect.ConfigurationServiceClient
}

func (a *apiProvider) GetBoolFeature(ctx context.Context, key string, namespace string) (bool, error) {
	lc, err := toProto(fromContext(ctx))
	if err != nil {
		return false, errors.Wrap(err, "error transforming context")
	}
	req := connect.NewRequest(&backendv1beta1.GetBoolValueRequest{
		Key:       key,
		Namespace: namespace,
		Context:   lc,
	})
	req.Header().Set(LekkoAPIKeyHeader, a.apikey)
	resp, err := a.lekkoClient.GetBoolValue(ctx, req)
	if err != nil {
		return false, errors.Wrap(err, "error hitting lekko backend")
	}
	return resp.Msg.GetValue(), nil
}

func (a *apiProvider) GetStringFeature(ctx context.Context, key string, namespace string) (string, error) {
	return "", fmt.Errorf("unimplemented")
}

func (a *apiProvider) GetProtoFeature(ctx context.Context, key string, namespace string, result proto.Message) error {
	lc, err := toProto(fromContext(ctx))
	if err != nil {
		return errors.Wrap(err, "error transforming context")
	}
	req := connect.NewRequest(&backendv1beta1.GetProtoValueRequest{
		Key:       key,
		Namespace: namespace,
		Context:   lc,
	})
	req.Header().Set(LekkoAPIKeyHeader, a.apikey)
	resp, err := a.lekkoClient.GetProtoValue(ctx, req)
	if err != nil {
		return errors.Wrap(err, "error hitting lekko backend")
	}
	return resp.Msg.GetValue().UnmarshalTo(result)
}

func (a *apiProvider) GetJSONFeature(ctx context.Context, key string, namespace string, result interface{}) error {
	lc, err := toProto(fromContext(ctx))
	if err != nil {
		return errors.Wrap(err, "error transforming context")
	}
	req := connect.NewRequest(&backendv1beta1.GetJSONValueRequest{
		Key:       key,
		Namespace: namespace,
		Context:   lc,
	})
	req.Header().Set(LekkoAPIKeyHeader, a.apikey)
	resp, err := a.lekkoClient.GetJSONValue(ctx, req)
	if err != nil {
		return errors.Wrap(err, "error hitting lekko backend")
	}
	if err := json.Unmarshal(resp.Msg.GetValue(), result); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to unmarshal json into go type %T", result))
	}
	return nil
}
