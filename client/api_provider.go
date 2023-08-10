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
	"net"
	"net/http"
	"strings"
	"time"

	clientv1beta1connect "buf.build/gen/go/lekkodev/sdk/bufbuild/connect-go/lekko/client/v1beta1/clientv1beta1connect"
	clientv1beta1 "buf.build/gen/go/lekkodev/sdk/protocolbuffers/go/lekko/client/v1beta1"

	"github.com/bufbuild/connect-go"
	"github.com/cenkalti/backoff/v4"
	"github.com/pkg/errors"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

const (
	defaultAPIURL     = "https://prod.api.lekko.dev:443"
	defaultSidecarURL = "https://localhost:50051"
	lekkoAPIKeyHeader = "apikey"
)

// Fetches configuration directly from Lekko Backend APIs.
func ConnectAPIProvider(ctx context.Context, apiKey string, rk *RepositoryKey, opts ...ProviderOption) (Provider, error) {
	cfg := &providerConfig{}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	withFallbackURL(defaultAPIURL).apply(cfg)
	WithAPIKey(apiKey).apply(cfg)
	withRepositoryKey(rk).apply(cfg)
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	provider := &apiProvider{
		apikey:      cfg.apiKey,
		lekkoClient: clientv1beta1connect.NewConfigurationServiceClient(http.DefaultClient, cfg.url),
		rk:          cfg.repoKey,
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
func ConnectSidecarProvider(ctx context.Context, url string, rk *RepositoryKey, opts ...ProviderOption) (Provider, error) {
	cfg := &providerConfig{url: url}
	for _, opt := range opts {
		opt.apply(cfg)
	}
	WithURL(url).apply(cfg)
	withFallbackURL(defaultSidecarURL).apply(cfg)
	withRepositoryKey(rk).apply(cfg)
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	// The sidecar exposes the same interface as the backend API, so we can transparently
	// use the same underlying apiProvider implementation, just with different client config
	// e.g. grpc instead of raw http.
	// TODO: make use of tls config once the sidecar supports tls.
	provider := &apiProvider{
		lekkoClient: clientv1beta1connect.NewConfigurationServiceClient(&http.Client{
			Transport: &http2.Transport{
				AllowHTTP: true,
				DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
					return net.Dial(network, addr)
				},
			},
		}, cfg.url, connect.WithGRPC()),
		rk: cfg.repoKey,
	}
	if err := provider.registerWithBackoff(ctx); err != nil {
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

func (rk *RepositoryKey) toProto() *clientv1beta1.RepositoryKey {
	if rk == nil {
		return nil
	}
	return &clientv1beta1.RepositoryKey{
		OwnerName: rk.OwnerName,
		RepoName:  rk.RepoName,
	}
}

type apiProvider struct {
	apikey      string
	lekkoClient clientv1beta1connect.ConfigurationServiceClient
	rk          *RepositoryKey
}

// This performs an exponential backoff until the context is cancelled.
// We do this to try to alleviate some issues with sidecars starting up.
func (a *apiProvider) registerWithBackoff(ctx context.Context) error {
	req := connect.NewRequest(&clientv1beta1.RegisterRequest{RepoKey: a.rk.toProto()})
	req.Header().Set(lekkoAPIKeyHeader, a.apikey)
	ticker := backoff.NewTicker(backoff.NewExponentialBackOff())
	op := func() error {
		_, err := a.lekkoClient.Register(ctx, req)
		return err
	}
	var err error
	for {
		newErr := op()
		if newErr == nil {
			return nil
		}
		// keep track of errors to return if we time out
		if err != nil {
			err = errors.Wrap(newErr, err.Error())
		} else {
			err = newErr
		}
		select {
		case <-ctx.Done():
			return errors.Wrap(err, "timed out retrying, returning last error")
		case <-ticker.C:
		}
	}
}

// Should only be called on initialization.
func (a *apiProvider) register(ctx context.Context) error {
	req := connect.NewRequest(&clientv1beta1.RegisterRequest{RepoKey: a.rk.toProto()})
	req.Header().Set(lekkoAPIKeyHeader, a.apikey)
	_, err := a.lekkoClient.Register(ctx, req)
	return err
}

func (a *apiProvider) Close(ctx context.Context) error {
	req := connect.NewRequest(&clientv1beta1.DeregisterRequest{})
	req.Header().Set(lekkoAPIKeyHeader, a.apikey)
	_, err := a.lekkoClient.Deregister(ctx, req)
	return err
}

func (a *apiProvider) GetBoolFeature(ctx context.Context, key string, namespace string) (bool, error) {
	lc, err := toProto(fromContext(ctx))
	if err != nil {
		return false, errors.Wrap(err, "error transforming context")
	}
	req := connect.NewRequest(&clientv1beta1.GetBoolValueRequest{
		Key:       key,
		Namespace: namespace,
		Context:   lc,
		RepoKey:   a.rk.toProto(),
	})
	req.Header().Set(lekkoAPIKeyHeader, a.apikey)
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
	req := connect.NewRequest(&clientv1beta1.GetIntValueRequest{
		Key:       key,
		Namespace: namespace,
		Context:   lc,
		RepoKey:   a.rk.toProto(),
	})
	req.Header().Set(lekkoAPIKeyHeader, a.apikey)
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
	req := connect.NewRequest(&clientv1beta1.GetFloatValueRequest{
		Key:       key,
		Namespace: namespace,
		Context:   lc,
		RepoKey:   a.rk.toProto(),
	})
	req.Header().Set(lekkoAPIKeyHeader, a.apikey)
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
	req := connect.NewRequest(&clientv1beta1.GetStringValueRequest{
		Key:       key,
		Namespace: namespace,
		Context:   lc,
		RepoKey:   a.rk.toProto(),
	})
	req.Header().Set(lekkoAPIKeyHeader, a.apikey)
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
	req := connect.NewRequest(&clientv1beta1.GetProtoValueRequest{
		Key:       key,
		Namespace: namespace,
		Context:   lc,
		RepoKey:   a.rk.toProto(),
	})
	req.Header().Set(lekkoAPIKeyHeader, a.apikey)
	resp, err := a.lekkoClient.GetProtoValue(ctx, req)
	if err != nil {
		return errors.Wrap(err, "error hitting lekko backend")
	}
	if resp.Msg.GetValueV2() != nil {
		a := &anypb.Any{
			TypeUrl: resp.Msg.ValueV2.GetTypeUrl(),
			Value:   resp.Msg.ValueV2.GetValue(),
		}
		return a.UnmarshalTo(result)
	}
	return resp.Msg.GetValue().UnmarshalTo(result)
}

func (a *apiProvider) GetJSONFeature(ctx context.Context, key string, namespace string, result interface{}) error {
	lc, err := toProto(fromContext(ctx))
	if err != nil {
		return errors.Wrap(err, "error transforming context")
	}
	req := connect.NewRequest(&clientv1beta1.GetJSONValueRequest{
		Key:       key,
		Namespace: namespace,
		Context:   lc,
		RepoKey:   a.rk.toProto(),
	})
	req.Header().Set(lekkoAPIKeyHeader, a.apikey)
	resp, err := a.lekkoClient.GetJSONValue(ctx, req)
	if err != nil {
		return errors.Wrap(err, "error hitting lekko backend")
	}
	if err := json.Unmarshal(resp.Msg.GetValue(), result); err != nil {
		return errors.Wrapf(err, "failed to unmarshal json into go type %T", result)
	}
	return nil
}

type ProviderOption interface {
	apply(*providerConfig)
}

// Used to override the default Lekko backend API URL.
// Only applicable to API provider.
type URLOption struct {
	URL string
}

func WithURL(url string) ProviderOption {
	return &URLOption{URL: url}
}

func (o *URLOption) apply(pc *providerConfig) { pc.url = o.URL }

type APIKeyOption struct {
	APIKey string
}

func WithAPIKey(apiKey string) ProviderOption {
	return &APIKeyOption{APIKey: apiKey}
}

func (o *APIKeyOption) apply(pc *providerConfig) { pc.apiKey = o.APIKey }

type UpdateIntervalOption struct {
	UpdateInterval time.Duration
}

func WithUpdateInterval(interval time.Duration) ProviderOption {
	return &UpdateIntervalOption{UpdateInterval: interval}
}

func (o *UpdateIntervalOption) apply(pc *providerConfig) {
	pc.updateInterval = o.UpdateInterval
}

type ServerOption struct {
	Port int32
}

func WithServerOption(port int32) ProviderOption {
	return &ServerOption{Port: port}
}

func (o *ServerOption) apply(pc *providerConfig) { pc.serverPort = o.Port }

type providerConfig struct {
	repoKey        *RepositoryKey
	apiKey, url    string
	updateInterval time.Duration
	serverPort     int32
}

func (cfg *providerConfig) validate() error {
	if cfg.repoKey == nil || len(cfg.repoKey.OwnerName) == 0 || len(cfg.repoKey.RepoName) == 0 {
		return errors.New("missing repository key")
	}
	if strings.Contains(cfg.url, "lekko.dev") || strings.Contains(cfg.url, "lekko.com") {
		if len(cfg.apiKey) == 0 {
			return errors.New("api key required when communicating with lekko backend")
		}
	}
	if cfg.updateInterval == 0 {
		cfg.updateInterval = defaultUpdateInterval
	} else if cfg.updateInterval.Seconds() < minUpdateInterval.Seconds() {
		return errors.Errorf("update interval too small, minimum %v", minUpdateInterval)
	}
	return nil
}

type fallbackURLOption struct {
	url string
}

func withFallbackURL(url string) ProviderOption {
	return &fallbackURLOption{url: url}
}

func (o *fallbackURLOption) apply(pc *providerConfig) {
	if len(pc.url) == 0 {
		pc.url = o.url
	}
}

type repositoryKeyOption struct {
	repoKey *RepositoryKey
}

func withRepositoryKey(repoKey *RepositoryKey) ProviderOption {
	return &repositoryKeyOption{repoKey: repoKey}
}

func (o *repositoryKeyOption) apply(cfg *providerConfig) {
	cfg.repoKey = o.repoKey
}
