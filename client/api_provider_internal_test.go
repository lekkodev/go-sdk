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
	context "context"
	"encoding/json"
	"testing"

	v1beta1 "buf.build/gen/go/lekkodev/sdk/protocolbuffers/go/lekko/client/v1beta1"
	"connectrpc.com/connect"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type testBackendClient struct {
	boolVal    bool
	intVal     int64
	floatVal   float64
	stringVal  string
	protoVal   proto.Message
	protoValV2 proto.Message
	jsonVal    []byte
	err        error
}

func (tbc *testBackendClient) GetBoolValue(ctx context.Context, req *connect.Request[v1beta1.GetBoolValueRequest]) (*connect.Response[v1beta1.GetBoolValueResponse], error) {
	return connect.NewResponse(&v1beta1.GetBoolValueResponse{
		Value: tbc.boolVal,
	}), tbc.err
}

func (tbc *testBackendClient) GetIntValue(ctx context.Context, req *connect.Request[v1beta1.GetIntValueRequest]) (*connect.Response[v1beta1.GetIntValueResponse], error) {
	return connect.NewResponse(&v1beta1.GetIntValueResponse{
		Value: tbc.intVal,
	}), tbc.err
}

func (tbc *testBackendClient) GetFloatValue(ctx context.Context, req *connect.Request[v1beta1.GetFloatValueRequest]) (*connect.Response[v1beta1.GetFloatValueResponse], error) {
	return connect.NewResponse(&v1beta1.GetFloatValueResponse{
		Value: tbc.floatVal,
	}), tbc.err
}

func (tbc *testBackendClient) GetStringValue(ctx context.Context, req *connect.Request[v1beta1.GetStringValueRequest]) (*connect.Response[v1beta1.GetStringValueResponse], error) {
	return connect.NewResponse(&v1beta1.GetStringValueResponse{
		Value: tbc.stringVal,
	}), tbc.err
}

func (tbc *testBackendClient) GetProtoValue(context.Context, *connect.Request[v1beta1.GetProtoValueRequest]) (*connect.Response[v1beta1.GetProtoValueResponse], error) {
	if tbc.protoVal != nil {
		anyVal, err := anypb.New(tbc.protoVal)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal anyval")
		}
		return connect.NewResponse(&v1beta1.GetProtoValueResponse{
			Value: anyVal,
		}), tbc.err
	}
	if tbc.protoValV2 != nil {
		anyVal, err := anypb.New(tbc.protoValV2)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal anyval")
		}
		return connect.NewResponse(&v1beta1.GetProtoValueResponse{
			ValueV2: &v1beta1.Any{
				TypeUrl: anyVal.GetTypeUrl(),
				Value:   anyVal.GetValue(),
			},
		}), tbc.err
	}
	return nil, tbc.err
}

func (tbc *testBackendClient) GetJSONValue(context.Context, *connect.Request[v1beta1.GetJSONValueRequest]) (*connect.Response[v1beta1.GetJSONValueResponse], error) {
	return connect.NewResponse(&v1beta1.GetJSONValueResponse{
		Value: tbc.jsonVal,
	}), tbc.err
}

func (tbc *testBackendClient) Register(context.Context, *connect.Request[v1beta1.RegisterRequest]) (*connect.Response[v1beta1.RegisterResponse], error) {
	return connect.NewResponse(&v1beta1.RegisterResponse{}), nil
}

func (tbc *testBackendClient) Deregister(context.Context, *connect.Request[v1beta1.DeregisterRequest]) (*connect.Response[v1beta1.DeregisterResponse], error) {
	return connect.NewResponse(&v1beta1.DeregisterResponse{}), nil
}

func testProvider(backendCli *testBackendClient) Provider {
	return &apiProvider{
		lekkoClient: backendCli,
	}
}

func TestGetBoolConfig(t *testing.T) {
	// success
	ctx := context.Background()
	cli := testProvider(&testBackendClient{boolVal: true})
	result, err := cli.GetBool(ctx, "test_key", "namespace")
	assert.NoError(t, err)
	assert.True(t, result)

	// test passing up backend error
	cli = testProvider(&testBackendClient{err: errors.New("error")})
	_, err = cli.GetBool(ctx, "test_key", "namespace")
	assert.Error(t, err)
}

func TestGetIntConfig(t *testing.T) {
	// success
	ctx := context.Background()
	cli := testProvider(&testBackendClient{intVal: 8})
	result, err := cli.GetInt(ctx, "test_key", "namespace")
	assert.NoError(t, err)
	assert.Equal(t, int64(8), result)

	// test passing up backend error
	cli = testProvider(&testBackendClient{err: errors.New("error")})
	_, err = cli.GetInt(ctx, "test_key", "namespace")
	assert.Error(t, err)
}

func TestGetFloatConfig(t *testing.T) {
	// success
	ctx := context.Background()
	cli := testProvider(&testBackendClient{floatVal: 8.89})
	result, err := cli.GetFloat(ctx, "test_key", "namespace")
	assert.NoError(t, err)
	assert.Equal(t, 8.89, result)

	// test passing up backend error
	cli = testProvider(&testBackendClient{err: errors.New("error")})
	_, err = cli.GetFloat(ctx, "test_key", "namespace")
	assert.Error(t, err)
}

func TestGetStringConfig(t *testing.T) {
	// success
	ctx := context.Background()
	cli := testProvider(&testBackendClient{stringVal: "foo"})
	result, err := cli.GetString(ctx, "test_key", "namespace")
	assert.NoError(t, err)
	assert.Equal(t, "foo", result)

	// test passing up backend error
	cli = testProvider(&testBackendClient{err: errors.New("error")})
	_, err = cli.GetString(ctx, "test_key", "namespace")
	assert.Error(t, err)
}

func TestGetProtoConfig(t *testing.T) {
	protoProvider := func(version string, p proto.Message, err error) Provider {
		if version == "v1" {
			return testProvider(&testBackendClient{protoVal: p, err: err})
		}
		return testProvider(&testBackendClient{protoValV2: p, err: err})
	}
	// test that get proto works for both versions of the 'any'
	// that is returned via the api.
	for _, version := range []string{"v1", "v2"} {
		t.Run(version, func(t *testing.T) {
			// success
			ctx := context.Background()
			cli := protoProvider(version, wrapperspb.Int64(59), nil)
			result := &wrapperspb.Int64Value{}
			require.NoError(t, cli.GetProto(ctx, "test_key", "namespace", result))
			assert.EqualValues(t, int64(59), result.Value)

			// test passing up backend error
			cli = protoProvider(version, wrapperspb.Int64(59), errors.New("error"))
			assert.Error(t, cli.GetProto(ctx, "test_key", "namespace", result))

			// type mismatch in proto value
			cli = protoProvider(version, wrapperspb.Int64(59), nil)
			badResult := &wrapperspb.BoolValue{Value: true}
			assert.Error(t, cli.GetProto(ctx, "test_key", "namespace", badResult))
			// Note: the value of badResult is now undefined and api
			// behavior may change, so it should not be depended on
		})
	}
}

func TestUnsupportedContextType(t *testing.T) {
	ctx := Add(context.Background(), "foo", []string{})
	cli := testProvider(&testBackendClient{boolVal: true})
	_, err := cli.GetBool(ctx, "test_key", "namespace")
	require.Error(t, err)
}

type barType struct {
	Baz int `json:"baz"`
}

type testStruct struct {
	Foo int      `json:"foo"`
	Bar *barType `json:"bar"`
}

func TestGetJSONConfig(t *testing.T) {
	// success
	ctx := context.Background()
	ts := &testStruct{Foo: 1, Bar: &barType{Baz: 12}}
	bytes, err := json.Marshal(ts)
	require.NoError(t, err)
	cli := testProvider(&testBackendClient{jsonVal: bytes})
	result := &testStruct{}
	require.NoError(t, cli.GetJSON(ctx, "test_key", "namespace", result))
	assert.EqualValues(t, ts, result)

	// test passing up backend error
	cli = testProvider(&testBackendClient{jsonVal: bytes, err: errors.New("error")})
	assert.Error(t, cli.GetJSON(ctx, "test_key", "namespace", result))

	// type mismatch in result
	cli = testProvider(&testBackendClient{jsonVal: bytes})
	badResult := new(int)
	assert.Error(t, cli.GetJSON(ctx, "test_key", "namespace", badResult))
}

func TestGetJSONConfigArr(t *testing.T) {
	ctx := context.Background()
	ts := []int{1, 2, 3}
	bytes, err := json.Marshal(&ts)
	require.NoError(t, err)
	cli := testProvider(&testBackendClient{jsonVal: bytes})
	result := []int{45}
	require.NoError(t, cli.GetJSON(ctx, "test_key", "namespace", &result))
	assert.EqualValues(t, ts, result)
}

func TestGetJSONConfigError(t *testing.T) {
	ctx := context.Background()
	ts := []int{1, 2, 3}
	bytes, err := json.Marshal(&ts)
	require.NoError(t, err)
	cli := testProvider(&testBackendClient{jsonVal: bytes})
	result := []string{"foo"}
	err = cli.GetJSON(ctx, "test_key", "namespace", &result)
	assert.Error(t, err)
	// Note: the value of &result is now undefined and api
	// behavior may change, so it should not be depended on
}
