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

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
	"github.com/lekkodev/go-sdk/internal/memory"
	"github.com/lekkodev/go-sdk/internal/oteltest"
	"github.com/lekkodev/go-sdk/testdata"
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
		otel: &otelTracing{},
	}

	// happy paths
	ctx, otelHelper := oteltest.InitOtelAndStartSpan()
	br, err := im.GetBool(ctx, "bool", "")
	require.NoError(t, err)
	assert.True(t, br)
	event := otelHelper.EndSpanAndGetConfigEvent(t)
	expected := []attribute.KeyValue{
		attribute.String("lekko.key", "bool"),
		attribute.String("lekko.bool.string_value", "true"),
		attribute.String("lekko.bool.version", testdata.FakeCommitSHA("bool")),
	}
	assert.ElementsMatch(t, expected, event.Attributes)

	ctx, otelHelper = oteltest.InitOtelAndStartSpan()
	sr, err := im.GetString(ctx, "string", "")
	require.NoError(t, err)
	assert.Equal(t, "foo", sr)
	event = otelHelper.EndSpanAndGetConfigEvent(t)
	expected = []attribute.KeyValue{
		attribute.String("lekko.key", "string"),
		attribute.String("lekko.string.string_value", "foo"),
		attribute.String("lekko.string.version", testdata.FakeCommitSHA("string")),
	}
	assert.ElementsMatch(t, expected, event.Attributes)

	ctx, otelHelper = oteltest.InitOtelAndStartSpan()
	ir, err := im.GetInt(ctx, "int", "")
	require.NoError(t, err)
	assert.Equal(t, int64(42), ir)
	event = otelHelper.EndSpanAndGetConfigEvent(t)
	expected = []attribute.KeyValue{
		attribute.String("lekko.key", "int"),
		attribute.String("lekko.int.string_value", "42"),
		attribute.String("lekko.int.version", testdata.FakeCommitSHA("int")),
	}
	assert.ElementsMatch(t, expected, event.Attributes)

	ctx, otelHelper = oteltest.InitOtelAndStartSpan()
	fr, err := im.GetFloat(ctx, "float", "")
	require.NoError(t, err)
	assert.Equal(t, float64(1.2), fr)
	event = otelHelper.EndSpanAndGetConfigEvent(t)
	expected = []attribute.KeyValue{
		attribute.String("lekko.key", "float"),
		attribute.String("lekko.float.string_value", "1.2"),
		attribute.String("lekko.float.version", testdata.FakeCommitSHA("float")),
	}
	assert.ElementsMatch(t, expected, event.Attributes)

	ctx, otelHelper = oteltest.InitOtelAndStartSpan()
	var result []any
	require.NoError(t, im.GetJSON(ctx, "json", "", &result))
	assert.EqualValues(t, []any{1.0, 2.0}, result)
	event = otelHelper.EndSpanAndGetConfigEvent(t)
	expected = []attribute.KeyValue{
		attribute.String("lekko.key", "json"),
		attribute.String("lekko.json.version", testdata.FakeCommitSHA("json")),
	}
	assert.ElementsMatch(t, expected, event.Attributes)

	ctx, otelHelper = oteltest.InitOtelAndStartSpan()
	protoResult := &wrapperspb.Int32Value{}
	err = im.GetProto(ctx, "proto", "", protoResult)
	require.NoError(t, err)
	assert.EqualValues(t, int32(58), protoResult.Value)
	assert.ElementsMatch(t,
		[]attribute.KeyValue{
			attribute.String("lekko.key", "proto"),
			attribute.String("lekko.proto.version", testdata.FakeCommitSHA("proto")),
		},
		otelHelper.EndSpanAndGetConfigEvent(t).Attributes,
	)

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

func (ts *testStore) Evaluate(key string, namespace string, lc map[string]interface{}, dest proto.Message) (*memory.StoredConfig, error) {
	a, ok := ts.configs[key]
	if !ok {
		return nil, errors.Errorf("key %s not found", key)
	}
	return &memory.StoredConfig{CommitSHA: testdata.FakeCommitSHA(key)}, a.UnmarshalTo(dest)
}

func (ts *testStore) EvaluateAny(key string, namespace string, lc map[string]interface{}) (protoreflect.ProtoMessage, *memory.StoredConfig, error) {
	// TODO: not tested
	return nil, nil, nil
}

func (ts *testStore) Close(ctx context.Context) error {
	return ts.closeErr
}
