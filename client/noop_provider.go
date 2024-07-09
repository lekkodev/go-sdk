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

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var ErrNoOpProvider = errors.New("no op provider")

type noOpProvider struct{}

func (p *noOpProvider) GetBool(ctx context.Context, key string, namespace string) (bool, error) {
	return false, ErrNoOpProvider
}
func (p *noOpProvider) GetInt(ctx context.Context, key string, namespace string) (int64, error) {
	return 0, ErrNoOpProvider
}
func (p *noOpProvider) GetFloat(ctx context.Context, key string, namespace string) (float64, error) {
	return 0, ErrNoOpProvider
}
func (p *noOpProvider) GetString(ctx context.Context, key string, namespace string) (string, error) {
	return "", ErrNoOpProvider
}
func (p *noOpProvider) GetProto(ctx context.Context, key string, namespace string, result proto.Message) error {
	return ErrNoOpProvider
}
func (p *noOpProvider) GetJSON(ctx context.Context, key string, namespace string, result interface{}) error {
	return ErrNoOpProvider
}
func (p *noOpProvider) GetAny(ctx context.Context, key string, namespace string) (protoreflect.ProtoMessage, error) {
	return nil, ErrNoOpProvider
}
func (p *noOpProvider) Close(ctx context.Context) error {
	return nil
}
