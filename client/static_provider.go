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
	"strconv"

	"github.com/lekkodev/go-sdk/internal/memory"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func NewStaticProvider(featureBytes map[string]map[string][]byte) (Provider, error) {
	backend, err := memory.NewStaticStore(featureBytes)
	if err != nil {
		return nil, err
	}
	return &staticProvider{
		store: backend,
	}, nil
}

type staticProvider struct {
	store memory.Store
	otel  *otelTracing
}

func (p *staticProvider) Close(ctx context.Context) error {
	return p.store.Close(ctx)
}

func (p *staticProvider) GetBool(ctx context.Context, key string, namespace string) (bool, error) {
	dest := &wrapperspb.BoolValue{}
	cfg, err := p.store.Evaluate(ctx, key, namespace, fromContext(ctx), dest)
	if err != nil {
		return false, err
	}
	p.otel.addTracingEvent(ctx, key, strconv.FormatBool(dest.GetValue()), cfg.LastUpdateCommitSHA)
	return dest.GetValue(), nil
}

func (p *staticProvider) GetFloat(ctx context.Context, key string, namespace string) (float64, error) {
	dest := &wrapperspb.DoubleValue{}
	cfg, err := p.store.Evaluate(ctx, key, namespace, fromContext(ctx), dest)
	if err != nil {
		return 0, err
	}
	p.otel.addTracingEvent(ctx, key, strconv.FormatFloat(dest.GetValue(), 'G', -1, 64), cfg.LastUpdateCommitSHA)
	return dest.GetValue(), nil
}

func (p *staticProvider) GetInt(ctx context.Context, key string, namespace string) (int64, error) {
	dest := &wrapperspb.Int64Value{}
	cfg, err := p.store.Evaluate(ctx, key, namespace, fromContext(ctx), dest)
	if err != nil {
		return 0, err
	}
	p.otel.addTracingEvent(ctx, key, strconv.FormatInt(dest.GetValue(), 10), cfg.LastUpdateCommitSHA)
	return dest.GetValue(), nil
}

func (p *staticProvider) GetJSON(ctx context.Context, key string, namespace string, result interface{}) error {
	dest := &structpb.Value{}
	cfg, err := p.store.Evaluate(ctx, key, namespace, fromContext(ctx), dest)
	if err != nil {
		return err
	}
	bytes, err := dest.MarshalJSON()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bytes, result); err != nil {
		return errors.Wrapf(err, "failed to unmarshal json into go type %T", result)
	}
	p.otel.addTracingEvent(ctx, key, "", cfg.LastUpdateCommitSHA)
	return nil
}

func (p *staticProvider) GetProto(ctx context.Context, key string, namespace string, result protoreflect.ProtoMessage) error {
	cfg, err := p.store.Evaluate(ctx, key, namespace, fromContext(ctx), result)
	if err != nil {
		return err
	}
	p.otel.addTracingEvent(ctx, key, "", cfg.LastUpdateCommitSHA)
	return nil
}

func (p *staticProvider) GetString(ctx context.Context, key string, namespace string) (string, error) {
	dest := &wrapperspb.StringValue{}
	cfg, err := p.store.Evaluate(ctx, key, namespace, fromContext(ctx), dest)
	if err != nil {
		return "", err
	}
	p.otel.addTracingEvent(ctx, key, "", cfg.LastUpdateCommitSHA)
	return dest.GetValue(), nil
}
