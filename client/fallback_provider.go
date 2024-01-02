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
	"errors"

	"google.golang.org/protobuf/reflect/protoreflect"
)

func FallbackProvider(primary Provider, backup Provider) Provider {
	return &fallbackProvider{primary, backup}
}

type fallbackProvider struct {
	primary Provider
	backup  Provider
}

func (p *fallbackProvider) Close(ctx context.Context) error {
	primaryErr := p.primary.Close(ctx)
	backupErr := p.backup.Close(ctx)
	return errors.Join(primaryErr, backupErr)
}

func (p *fallbackProvider) GetBool(ctx context.Context, key string, namespace string) (bool, error) {
	ret, err := p.primary.GetBool(ctx, key, namespace)
	if err != nil {
		return p.backup.GetBool(ctx, key, namespace)
	}
	return ret, err
}

func (p *fallbackProvider) GetFloat(ctx context.Context, key string, namespace string) (float64, error) {
	ret, err := p.primary.GetFloat(ctx, key, namespace)
	if err != nil {
		return p.backup.GetFloat(ctx, key, namespace)
	}
	return ret, err
}

func (p *fallbackProvider) GetInt(ctx context.Context, key string, namespace string) (int64, error) {
	ret, err := p.primary.GetInt(ctx, key, namespace)
	if err != nil {
		return p.backup.GetInt(ctx, key, namespace)
	}
	return ret, err
}

func (p *fallbackProvider) GetJSON(ctx context.Context, key string, namespace string, result interface{}) error {
	err := p.primary.GetJSON(ctx, key, namespace, result)
	if err != nil {
		return p.backup.GetJSON(ctx, key, namespace, result)
	}
	return err
}

func (p *fallbackProvider) GetProto(ctx context.Context, key string, namespace string, result protoreflect.ProtoMessage) error {
	err := p.primary.GetProto(ctx, key, namespace, result)
	if err != nil {
		return p.backup.GetProto(ctx, key, namespace, result)
	}
	return err
}

func (p *fallbackProvider) GetString(ctx context.Context, key string, namespace string) (string, error) {
	ret, err := p.primary.GetString(ctx, key, namespace)
	if err != nil {
		return p.backup.GetString(ctx, key, namespace)
	}
	return ret, err
}
