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
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/net/http2"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// A provider evaluates configuration from a number of sources.
type Provider interface {
	GetBool(ctx context.Context, key string, namespace string) (bool, error)
	GetInt(ctx context.Context, key string, namespace string) (int64, error)
	GetFloat(ctx context.Context, key string, namespace string) (float64, error)
	GetString(ctx context.Context, key string, namespace string) (string, error)
	GetProto(ctx context.Context, key string, namespace string, result proto.Message) error
	GetJSON(ctx context.Context, key string, namespace string, result interface{}) error
	GetAny(ctx context.Context, key string, namespace string) (protoreflect.ProtoMessage, error)
	// Error will get called by the closure returned in Client initialization.
	Close(ctx context.Context) error
}

type ProviderOption interface {
	apply(*providerConfig)
	String() string
}

type URLOption struct {
	URL string
}

// Used to override the default Lekko URL.
// For API providers, the default url is Lekko backend.
// For the Sidecar provider, the default URL is on localhost.
func WithURL(url string) ProviderOption {
	return &URLOption{URL: url}
}

func (o *URLOption) apply(pc *providerConfig) { pc.url = o.URL }

func (o *URLOption) String() string {
	return fmt.Sprintf("%+v", *o)
}

type APIKeyOption struct {
	APIKey string
}

// For providers that communicate directly with Lekko backend,
// api key is required.
func WithAPIKey(apiKey string) ProviderOption {
	return &APIKeyOption{APIKey: apiKey}
}

func (o *APIKeyOption) apply(pc *providerConfig) { pc.apiKey = o.APIKey }

func (o *APIKeyOption) String() string {
	return fmt.Sprintf("%+v", *o)
}

type AllowHTTPOption struct{}

// For connecting to the sidecar without TLS configured.
func WithAllowHTTP() ProviderOption {
	return &AllowHTTPOption{}
}

func (o *AllowHTTPOption) apply(pc *providerConfig) { pc.allowHTTP = true }

func (o *AllowHTTPOption) String() string {
	return fmt.Sprintf("%+v", *o)
}

type UpdateIntervalOption struct {
	UpdateInterval time.Duration
}

// Optionally configure an update interval for the cached provider.
// If none is provided, a default will be picked.
func WithUpdateInterval(interval time.Duration) ProviderOption {
	return &UpdateIntervalOption{UpdateInterval: interval}
}

func (o *UpdateIntervalOption) apply(pc *providerConfig) {
	pc.updateInterval = o.UpdateInterval
}

func (o *UpdateIntervalOption) String() string {
	return fmt.Sprintf("%+v", *o)
}

type ServerOption struct {
	Port int32
}

// If this option is set, the cached provider will expose a
// web server at the provided port for debugging.
func WithServerOption(port int32) ProviderOption {
	return &ServerOption{Port: port}
}

func (o *ServerOption) apply(pc *providerConfig) { pc.serverPort = o.Port }

func (o *ServerOption) String() string {
	return fmt.Sprintf("%+v", *o)
}

type providerConfig struct {
	repoKey        *RepositoryKey
	apiKey, url    string
	updateInterval time.Duration
	serverPort     int32
	allowHTTP      bool
	otelTracing    bool
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
	if cfg.allowHTTP && strings.HasPrefix(cfg.url, "https://") {
		return errors.Errorf("connecting to https endpoint: %s over gRPC/h2c, please unset AllowHTTP option", cfg.url)
	}
	if !cfg.allowHTTP && strings.HasPrefix(cfg.url, "http://") {
		return errors.Errorf("connecting to http endpoint: %s over gRPC/TLS, please set AllowHTTP option", cfg.url)
	}
	return nil
}

func (cfg *providerConfig) getHTTPClient() *http.Client {
	if cfg.allowHTTP {
		return &http.Client{
			Transport: &http2.Transport{
				AllowHTTP: true,
				DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
					return net.Dial(network, addr)
				},
			},
		}
	}
	return http.DefaultClient
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

func (o *fallbackURLOption) String() string {
	return fmt.Sprintf("%+v", *o)
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

func (o *repositoryKeyOption) String() string {
	return fmt.Sprintf("%+v", *o)
}

type otelTracingOption struct{}

func WithOtelTracing() ProviderOption {
	return &otelTracingOption{}
}

func (o *otelTracingOption) apply(cfg *providerConfig) {
	cfg.otelTracing = true
}

func (o *otelTracingOption) String() string {
	return "otelTracing"
}

type otelTracing struct{}

func (o *otelTracing) addTracingEvent(ctx context.Context, key, stringValue, version string) {
	if o == nil {
		return
	}

	span := trace.SpanFromContext(ctx)

	var attributes []attribute.KeyValue
	if len(stringValue) > 0 {
		attributes = append(attributes, attribute.String(fmt.Sprintf("lekko.%s.string_value", key), stringValue))
	}

	span.SetAttributes(attributes...)

	// TODO: add repo and namespace
	if len(version) > 0 {
		attributes = append(attributes, attribute.String(fmt.Sprintf("lekko.%s.version", key), version))
	}
	attributes = append(attributes, attribute.String("lekko.key", key))
	span.AddEvent(
		fmt.Sprintf("lekko.%s", key),
		trace.WithAttributes(attributes...),
	)
}
