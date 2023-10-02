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
	"strconv"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

// Client allows retrieving lekko configs. The appropriate method must
// be called based on the type of the config. For instance, calling
// GetBool on an Int config will result in an error.
type Client interface {
	GetBool(ctx context.Context, namespace, key string) (bool, error)
	GetInt(ctx context.Context, namespace, key string) (int64, error)
	GetFloat(ctx context.Context, namespace, key string) (float64, error)
	GetString(ctx context.Context, namespace, key string) (string, error)
	// result should be an empty Go variable that the JSON config
	// can be unmarshalled into. If an error is returned, the data
	// stored in result is unpredictable and should not be relied upon.
	GetJSON(ctx context.Context, namespace, key string, result interface{}) error
	// result should be an empty proto message that the proto config
	// can be unmarshalled into. If an error is returned, the data
	// stored in result is unpredictable and should not be relied upon.
	GetProto(ctx context.Context, namespace, key string, result proto.Message) error
	Close(ctx context.Context) error
}

type CloseFunc func(context.Context) error

// A function is returned to close the client. It is also strongly recommended
// to call this when the program is exiting or the lekko provider is no longer needed.
func NewClient(provider Provider) (Client, CloseFunc) {
	return newClient(provider, ClientOptions{})
}

type ClientOptions struct {
	// Arbitrary keys and values for rules evaluation that are
	// injected into the context at startup. Context keys passed
	// at runtime will override these values in case of conflict.
	StartupContext    map[string]interface{}
	EnableOTelTracing bool
}

func (o ClientOptions) NewClient(provider Provider) (Client, CloseFunc) {
	return newClient(provider, o)
}

type client struct {
	provider          Provider
	startupContext    map[string]interface{}
	enableOTelTracing bool
}

func newClient(provider Provider, options ClientOptions) (*client, CloseFunc) {
	return &client{
		provider:          provider,
		startupContext:    options.StartupContext,
		enableOTelTracing: options.EnableOTelTracing,
	}, provider.Close
}

func (c *client) addTracingEvent(ctx context.Context, key, stringValue, version string) {
	if !c.enableOTelTracing {
		return
	}
	span := trace.SpanFromContext(ctx)
	attributes := []attribute.KeyValue{
		attribute.String("config.key", key),
	}
	if len(stringValue) > 0 {
		attributes = append(attributes, attribute.String("config.string_value", stringValue))
	}
	if len(version) > 0 {
		attributes = append(attributes, attribute.String("config.version", version))
	}

	span.AddEvent(
		"lekko_config",
		trace.WithAttributes(attributes...),
	)
}

func (c *client) GetBool(ctx context.Context, namespace, key string) (bool, error) {
	res, meta, err := c.provider.GetBool(c.wrap(ctx), key, namespace)
	c.addTracingEvent(ctx, key, strconv.FormatBool(res), meta.LastUpdateCommitSHA)
	return res, err
}

func (c *client) GetInt(ctx context.Context, namespace, key string) (int64, error) {
	res, meta, err := c.provider.GetInt(c.wrap(ctx), key, namespace)
	c.addTracingEvent(ctx, key, strconv.FormatInt(res, 10), meta.LastUpdateCommitSHA)
	return res, err
}

func (c *client) GetFloat(ctx context.Context, namespace, key string) (float64, error) {
	res, meta, err := c.provider.GetFloat(c.wrap(ctx), key, namespace)
	c.addTracingEvent(ctx, key, strconv.FormatFloat(res, 'E', -1, 64), meta.LastUpdateCommitSHA)
	return res, err
}

func (c *client) GetString(ctx context.Context, namespace, key string) (string, error) {
	res, meta, err := c.provider.GetString(c.wrap(ctx), key, namespace)
	c.addTracingEvent(ctx, key, "", meta.LastUpdateCommitSHA)
	return res, err
}

func (c *client) GetProto(ctx context.Context, namespace, key string, result proto.Message) error {
	meta, err := c.provider.GetProto(c.wrap(ctx), key, namespace, result)
	c.addTracingEvent(ctx, key, "", meta.LastUpdateCommitSHA)
	return err
}

func (c *client) GetJSON(ctx context.Context, namespace, key string, result interface{}) error {
	meta, err := c.provider.GetJSON(c.wrap(ctx), key, namespace, result)
	c.addTracingEvent(ctx, key, "", meta.LastUpdateCommitSHA)
	return err
}

func (c *client) wrap(ctx context.Context) context.Context {
	if c.startupContext == nil {
		return ctx
	}

	return Merge(ctx, c.startupContext)
}

func (c *client) Close(ctx context.Context) error {
	return c.provider.Close(ctx)
}
