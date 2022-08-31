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

	"github.com/bufbuild/connect-go"
	v1beta1 "github.com/lekkodev/cli/pkg/gen/proto/go/lekko/backend/v1beta1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type testBackendClient struct {
	boolVal                    bool
	protoVal                   proto.Message
	jsonVal                    []byte
	boolErr, protoErr, jsonErr error
}

func (tbc *testBackendClient) GetBoolValue(ctx context.Context, req *connect.Request[v1beta1.GetBoolValueRequest]) (*connect.Response[v1beta1.GetBoolValueResponse], error) {
	return connect.NewResponse(&v1beta1.GetBoolValueResponse{
		Value: tbc.boolVal,
	}), tbc.boolErr
}

func (tbc *testBackendClient) GetProtoValue(context.Context, *connect.Request[v1beta1.GetProtoValueRequest]) (*connect.Response[v1beta1.GetProtoValueResponse], error) {
	anyVal, err := anypb.New(tbc.protoVal)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal anyval")
	}
	return connect.NewResponse(&v1beta1.GetProtoValueResponse{
		Value: anyVal,
	}), tbc.protoErr
}

func (tbc *testBackendClient) GetJSONValue(context.Context, *connect.Request[v1beta1.GetJSONValueRequest]) (*connect.Response[v1beta1.GetJSONValueResponse], error) {
	return connect.NewResponse(&v1beta1.GetJSONValueResponse{
		Value: tbc.jsonVal,
	}), tbc.jsonErr
}

func testClient(backendCli *testBackendClient) *Client {
	return &Client{
		lekkoClient: backendCli,
	}
}

func TestGetBoolValue(t *testing.T) {
	// success
	ctx := context.Background()
	cli := testClient(&testBackendClient{boolVal: true})
	result, err := cli.GetBool(ctx, "test_key")
	assert.NoError(t, err)
	assert.True(t, result)

	// test passing up backend error
	cli = testClient(&testBackendClient{boolErr: errors.New("error")})
	_, err = cli.GetBool(ctx, "test_key")
	assert.Error(t, err)
}

func TestGetProtoValue(t *testing.T) {
	// success
	ctx := context.Background()
	cli := testClient(&testBackendClient{protoVal: wrapperspb.Int64(59)})
	result := &wrapperspb.Int64Value{}
	require.NoError(t, cli.GetProto(ctx, "test_key", result))
	assert.EqualValues(t, int64(59), result.Value)

	// test passing up backend error
	cli = testClient(&testBackendClient{protoVal: wrapperspb.Int64(59), protoErr: errors.New("error")})
	assert.Error(t, cli.GetProto(ctx, "test_key", result))

	// type mismatch in proto value
	cli = testClient(&testBackendClient{protoVal: wrapperspb.Int64(59)})
	badResult := &wrapperspb.BoolValue{}
	assert.Error(t, cli.GetProto(ctx, "test_key", badResult))
}

func TestUnsupportedContextType(t *testing.T) {
	ctx := Add(context.Background(), "foo", []string{})
	cli := testClient(&testBackendClient{boolVal: true})
	_, err := cli.GetBool(ctx, "test_key")
	require.Error(t, err)
}

type barType struct {
	Baz int `json:"baz"`
}

type testStruct struct {
	Foo int      `json:"foo"`
	Bar *barType `json:"bar"`
}

func TestGetJSONValue(t *testing.T) {
	// success
	ctx := context.Background()
	ts := &testStruct{Foo: 1, Bar: &barType{Baz: 12}}
	bytes, err := json.Marshal(ts)
	require.NoError(t, err)
	cli := testClient(&testBackendClient{jsonVal: bytes})
	result := &testStruct{}
	require.NoError(t, cli.GetJSON(ctx, "test_key", result))
	assert.EqualValues(t, ts, result)

	// test passing up backend error
	cli = testClient(&testBackendClient{jsonVal: bytes, jsonErr: errors.New("error")})
	assert.Error(t, cli.GetJSON(ctx, "test_key", result))

	// type mismatch in result
	cli = testClient(&testBackendClient{jsonVal: bytes})
	badResult := new(int)
	assert.Error(t, cli.GetJSON(ctx, "test_key", badResult))
}

func TestGetJSONArrValue(t *testing.T) {
	ctx := context.Background()
	ts := []int{1, 2, 3}
	bytes, err := json.Marshal(&ts)
	require.NoError(t, err)
	cli := testClient(&testBackendClient{jsonVal: bytes})
	result := new([]int)
	require.NoError(t, cli.GetJSON(ctx, "test_key", result))
	assert.EqualValues(t, &ts, result)
}
