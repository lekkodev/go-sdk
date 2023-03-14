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

package clienttest

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/lekkodev/go-sdk/client"
	"google.golang.org/protobuf/proto"
)

var (
	ErrNotFound     = errors.New("feature not found")
	ErrTypeMismatch = errors.New("type mismatch")
)

// TestClient is an in-memory configuration store intended for
// use in unit tests. It conforms to the  See client_test.go for
// examples on how to use it.
type TestClient struct {
	values map[string]interface{}
	errors map[string]error
}

// Constructs a TestClient for use in unit tests.
func NewTestClient() *TestClient {
	return &TestClient{
		values: make(map[string]interface{}),
		errors: map[string]error{},
	}
}

func (tc *TestClient) WithBool(key string, value bool) *TestClient {
	return tc.withValue(key, value)
}

func (tc *TestClient) WithString(key string, value string) *TestClient {
	return tc.withValue(key, value)
}

func (tc *TestClient) WithInt(key string, value int64) *TestClient {
	return tc.withValue(key, value)
}

func (tc *TestClient) WithFloat(key string, value float64) *TestClient {
	return tc.withValue(key, value)
}

// Accepts a proto-serialized byte array
func (tc *TestClient) WithProto(key string, value []byte) *TestClient {
	return tc.withValue(key, value)
}

// Accepts a JSON-serialized byte array
func (tc *TestClient) WithJSON(key string, value []byte) *TestClient {
	return tc.withValue(key, value)
}

func (tc *TestClient) WithError(key string, err error) *TestClient {
	tc.errors[key] = err
	return tc
}

// Ensure we conform to the client interface
func (tc *TestClient) Client() client.Client {
	return tc
}

func (tc *TestClient) withValue(key string, value interface{}) *TestClient {
	tc.values[key] = value
	return tc
}

func (tc *TestClient) getValue(key string) (interface{}, error) {
	err, ok := tc.errors[key]
	if ok {
		return nil, err
	}
	val, ok := tc.values[key]
	if !ok {
		return nil, ErrNotFound
	}
	return val, nil
}

func (tc *TestClient) GetBool(_ context.Context, key string) (bool, error) {
	val, err := tc.getValue(key)
	if err != nil {
		return false, err
	}
	typedVal, ok := val.(bool)
	if !ok {
		return false, ErrTypeMismatch
	}
	return typedVal, nil
}

func (tc *TestClient) GetInt(_ context.Context, key string) (int64, error) {
	val, err := tc.getValue(key)
	if err != nil {
		return 0, err
	}
	typedVal, ok := val.(int64)
	if !ok {
		return 0, ErrTypeMismatch
	}
	return typedVal, nil
}

func (tc *TestClient) GetFloat(_ context.Context, key string) (float64, error) {
	val, err := tc.getValue(key)
	if err != nil {
		return 0, err
	}
	typedVal, ok := val.(float64)
	if !ok {
		return 0, ErrTypeMismatch
	}
	return typedVal, nil
}

func (tc *TestClient) GetString(_ context.Context, key string) (string, error) {
	val, err := tc.getValue(key)
	if err != nil {
		return "", err
	}
	typedVal, ok := val.(string)
	if !ok {
		return "", ErrTypeMismatch
	}
	return typedVal, nil
}

func (tc *TestClient) GetProto(_ context.Context, key string, result proto.Message) error {
	val, err := tc.getValue(key)
	if err != nil {
		return err
	}
	bytes, ok := val.([]byte)
	if !ok {
		return ErrTypeMismatch
	}
	return proto.Unmarshal(bytes, result)
}

func (tc *TestClient) GetJSON(_ context.Context, key string, result interface{}) error {
	val, err := tc.getValue(key)
	if err != nil {
		return err
	}
	bytes, ok := val.([]byte)
	if !ok {
		return ErrTypeMismatch
	}
	return json.Unmarshal(bytes, result)
}
