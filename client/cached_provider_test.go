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
	"testing"

	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
	"github.com/lekkodev/go-sdk/testdata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func makeConfigs() map[string]*anypb.Any {
	ret := make(map[string]*anypb.Any)
	bf := testdata.Config(featurev1beta1.FeatureType_FEATURE_TYPE_BOOL, true)
	ret["bool"] = bf.GetFeature().GetTree().GetDefault()
	sf := testdata.Config(featurev1beta1.FeatureType_FEATURE_TYPE_STRING, "foo")
	ret["string"] = sf.GetFeature().GetTree().GetDefault()
	intf := testdata.Config(featurev1beta1.FeatureType_FEATURE_TYPE_INT, int64(42))
	ret["int"] = intf.GetFeature().GetTree().GetDefault()
	ff := testdata.Config(featurev1beta1.FeatureType_FEATURE_TYPE_FLOAT, float64(1.2))
	ret["float"] = ff.GetFeature().GetTree().GetDefault()
	jf := testdata.Config(featurev1beta1.FeatureType_FEATURE_TYPE_JSON, []any{1.0, 2.0})
	ret["json"] = jf.GetFeature().GetTree().GetDefault()
	pf := testdata.Config(featurev1beta1.FeatureType_FEATURE_TYPE_PROTO, wrapperspb.Int32(58))
	ret["proto"] = pf.GetFeature().GetTree().GetDefault()
	return ret
}

func TestInMemoryProviderSuccess(t *testing.T) {
	im := &cachedProvider{
		store: &testStore{
			configs: makeConfigs(),
		},
	}
	ctx := context.Background()
	// happy paths
	br, err := im.GetBool(ctx, "bool", "")
	require.NoError(t, err)
	assert.True(t, br)
	sr, err := im.GetString(ctx, "string", "")
	require.NoError(t, err)
	assert.Equal(t, "foo", sr)
	ir, err := im.GetInt(ctx, "int", "")
	require.NoError(t, err)
	assert.Equal(t, int64(42), ir)
	fr, err := im.GetFloat(ctx, "float", "")
	require.NoError(t, err)
	assert.Equal(t, float64(1.2), fr)
	var result []any
	err = im.GetJSON(ctx, "json", "", &result)
	require.NoError(t, err)
	assert.EqualValues(t, []any{1.0, 2.0}, result)
	protoResult := &wrapperspb.Int32Value{}
	err = im.GetProto(ctx, "proto", "", protoResult)
	require.NoError(t, err)
	assert.EqualValues(t, int32(58), protoResult.Value)
	require.NoError(t, im.Close(ctx), "no error during close")
}

func TestInMemoryProviderTypeMismatch(t *testing.T) {
	im := &cachedProvider{
		store: &testStore{
			configs: makeConfigs(),
		},
	}
	ctx := context.Background()
	_, err := im.GetString(ctx, "bool", "")
	require.Error(t, err)
	_, err = im.GetInt(ctx, "bool", "")
	require.Error(t, err)
	_, err = im.GetFloat(ctx, "bool", "")
	require.Error(t, err)
	var result bool
	err = im.GetJSON(ctx, "bool", "", &result)
	require.Error(t, err)
	protoResult := &wrapperspb.FloatValue{}
	err = im.GetProto(ctx, "bool", "", protoResult)
	require.Error(t, err)
	require.NoError(t, im.Close(ctx), "no error during close")
}

func TestInMemoryProviderMissingConfig(t *testing.T) {
	im := &cachedProvider{
		store: &testStore{
			configs: makeConfigs(),
		},
	}
	ctx := context.Background()
	_, err := im.GetBool(ctx, "missing", "")
	require.Error(t, err)
	require.NoError(t, im.Close(ctx), "no error during close")
}

func TestInMemoryProviderCloseError(t *testing.T) {
	im := &cachedProvider{
		store: &testStore{
			configs:  makeConfigs(),
			closeErr: errors.Errorf("close error"),
		},
	}
	ctx := context.Background()
	require.Error(t, im.Close(ctx), "error during close")
}

type testStore struct {
	configs  map[string]*anypb.Any
	closeErr error
}

func (ts *testStore) Evaluate(key string, namespace string, lc map[string]interface{}, dest proto.Message) error {
	a, ok := ts.configs[key]
	if !ok {
		return errors.Errorf("key %s not found", key)
	}
	return a.UnmarshalTo(dest)
}

func (ts *testStore) EvaluateAny(key string, namespace string, lc map[string]interface{}) (protoreflect.ProtoMessage, error) {
	return nil, nil
}

func (ts *testStore) Close(ctx context.Context) error {
	return ts.closeErr
}
