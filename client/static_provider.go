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

	"github.com/lekkodev/go-sdk/internal/memory"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func StaticProvider(featureBytes map[string]map[string][]byte) (Provider, error) {
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
}

func (p *staticProvider) Close(ctx context.Context) error {
	return p.store.Close(ctx)
}

func (p *staticProvider) GetBool(ctx context.Context, key string, namespace string) (bool, error) {
	dest := &wrapperspb.BoolValue{}
	if err := p.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return false, err
	}
	return dest.GetValue(), nil
}

func (p *staticProvider) GetFloat(ctx context.Context, key string, namespace string) (float64, error) {
	dest := &wrapperspb.DoubleValue{}
	if err := p.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return 0, err
	}
	return dest.GetValue(), nil
}

func (p *staticProvider) GetInt(ctx context.Context, key string, namespace string) (int64, error) {
	dest := &wrapperspb.Int64Value{}
	if err := p.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return 0, err
	}
	return dest.GetValue(), nil
}

func (p *staticProvider) GetJSON(ctx context.Context, key string, namespace string, result interface{}) error {
	dest := &structpb.Value{}
	if err := p.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return err
	}
	bytes, err := dest.MarshalJSON()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bytes, result); err != nil {
		return errors.Wrapf(err, "failed to unmarshal json into go type %T", result)
	}
	return nil
}

func (p *staticProvider) GetProto(ctx context.Context, key string, namespace string, result protoreflect.ProtoMessage) error {
	return p.store.Evaluate(key, namespace, fromContext(ctx), result)
}

func (p *staticProvider) GetString(ctx context.Context, key string, namespace string) (string, error) {
	dest := &wrapperspb.StringValue{}
	if err := p.store.Evaluate(key, namespace, fromContext(ctx), dest); err != nil {
		return "", err
	}
	return dest.GetValue(), nil
}
