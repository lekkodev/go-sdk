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
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/lekkodev/cli/pkg/gen/proto/go-connect/lekko/backend/v1beta1/backendv1beta1connect"
	backendv1beta1 "github.com/lekkodev/cli/pkg/gen/proto/go/lekko/backend/v1beta1"

	"github.com/bufbuild/connect-go"
	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/proto"
)

const (
	LekkoURL          = "https://prod.api.lekko.dev"
	LekkoAPIKeyHeader = "apikey"
	defaultSidecarURL = "https://localhost:50051"
)

// Fetches configuration directly from Lekko Backend APIs.
// This also make repeated RPCs to register the client, so providing a context with a timeout
// is strongly recommended.
func ConnectAPIProvider(ctx context.Context, apiKey string, rk *RepositoryKey) (Provider, error) {
	if rk == nil {
		return nil, fmt.Errorf("no repository key provided")
	}
	provider := &apiProvider{
		apikey:      apiKey,
		lekkoClient: backendv1beta1connect.NewConfigurationServiceClient(http.DefaultClient, LekkoURL),
		rk:          rk,
	}
	if err := provider.register(ctx); err != nil {
		return nil, err
	}
	return provider, nil
}

// Fetches configuration from a lekko sidecar, likely running on the local network.
// Will make repeated RPCs to register the client, so providing context with a timeout is
// strongly preferred. A function is returned to close the client. It is also strongly recommended
// to call this when the program is exiting or the lekko provider is no longer needed.
func ConnectSidecarProvider(ctx context.Context, url, apiKey string, rk *RepositoryKey) (Provider, error) {
	if rk == nil {
		return nil, fmt.Errorf("no repository key provided")
	}
	if url == "" {
		url = defaultSidecarURL
	}
	// The sidecar exposes the same interface as the backend API, so we can transparently
	// use the same underlying apiProvider implementation, just with different client config
	// e.g. grpc instead of raw http.
	// TODO: make use of tls config once the sidecar supports tls.
	provider := &apiProvider{
		apikey: apiKey,
		lekkoClient: backendv1beta1connect.NewConfigurationServiceClient(&http.Client{
			Transport: &http2.Transport{
				AllowHTTP: true,
				DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
					return net.Dial(network, addr)
				},
			},
		}, url, connect.WithGRPC()),
		rk: rk,
	}
	if err := provider.register(ctx); err != nil {
		return nil, err
	}
	return provider, nil
}

// Identifies a configuration repository on github.com
type RepositoryKey struct {
	// The name of the github owner. Can be a github
	// organization name or a personal account name.
	OwnerName string
	// The name of the repository on github.
	RepoName string
}

func (rk *RepositoryKey) toProto() *backendv1beta1.RepositoryKey {
	if rk == nil {
		return nil
	}
	return &backendv1beta1.RepositoryKey{
		OwnerName: rk.OwnerName,
		RepoName:  rk.RepoName,
	}
}

type apiProvider struct {
	apikey      string
	lekkoClient backendv1beta1connect.ConfigurationServiceClient
	rk          *RepositoryKey
}

// Should only be called on initialization.
// This performs an exponential backoff until the context is cancelled.
func (a *apiProvider) register(ctx context.Context) error {
	op := func() error {
		req := connect.NewRequest(&backendv1beta1.RegisterRequest{RepoKey: a.rk.toProto()})
		req.Header().Set(LekkoAPIKeyHeader, a.apikey)
		_, err := a.lekkoClient.Register(ctx, req)
		return err
	}
	return backoff.Retry(op, backoff.WithContext(backoff.NewExponentialBackOff(), ctx))
}

func (a *apiProvider) Close(ctx context.Context) error {
	req := connect.NewRequest(&backendv1beta1.DeregisterRequest{})
	req.Header().Set(LekkoAPIKeyHeader, a.apikey)
	_, err := a.lekkoClient.Deregister(ctx, req)
	return err
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
		RepoKey:   a.rk.toProto(),
	})
	req.Header().Set(LekkoAPIKeyHeader, a.apikey)
	resp, err := a.lekkoClient.GetBoolValue(ctx, req)
	if err != nil {
		return false, errors.Wrap(err, "error hitting lekko backend")
	}
	return resp.Msg.GetValue(), nil
}

func (a *apiProvider) GetIntFeature(ctx context.Context, key string, namespace string) (int64, error) {
	lc, err := toProto(fromContext(ctx))
	if err != nil {
		return 0, errors.Wrap(err, "error transforming context")
	}
	req := connect.NewRequest(&backendv1beta1.GetIntValueRequest{
		Key:       key,
		Namespace: namespace,
		Context:   lc,
		RepoKey:   a.rk.toProto(),
	})
	req.Header().Set(LekkoAPIKeyHeader, a.apikey)
	resp, err := a.lekkoClient.GetIntValue(ctx, req)
	if err != nil {
		return 0, errors.Wrap(err, "error hitting lekko backend")
	}
	return resp.Msg.GetValue(), nil
}

func (a *apiProvider) GetFloatFeature(ctx context.Context, key string, namespace string) (float64, error) {
	lc, err := toProto(fromContext(ctx))
	if err != nil {
		return 0, errors.Wrap(err, "error transforming context")
	}
	req := connect.NewRequest(&backendv1beta1.GetFloatValueRequest{
		Key:       key,
		Namespace: namespace,
		Context:   lc,
		RepoKey:   a.rk.toProto(),
	})
	req.Header().Set(LekkoAPIKeyHeader, a.apikey)
	resp, err := a.lekkoClient.GetFloatValue(ctx, req)
	if err != nil {
		return 0, errors.Wrap(err, "error hitting lekko backend")
	}
	return resp.Msg.GetValue(), nil
}

func (a *apiProvider) GetStringFeature(ctx context.Context, key string, namespace string) (string, error) {
	lc, err := toProto(fromContext(ctx))
	if err != nil {
		return "", errors.Wrap(err, "error transforming context")
	}
	req := connect.NewRequest(&backendv1beta1.GetStringValueRequest{
		Key:       key,
		Namespace: namespace,
		Context:   lc,
		RepoKey:   a.rk.toProto(),
	})
	req.Header().Set(LekkoAPIKeyHeader, a.apikey)
	resp, err := a.lekkoClient.GetStringValue(ctx, req)
	if err != nil {
		return "", errors.Wrap(err, "error hitting lekko backend")
	}
	return resp.Msg.GetValue(), nil
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
		RepoKey:   a.rk.toProto(),
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
		RepoKey:   a.rk.toProto(),
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
