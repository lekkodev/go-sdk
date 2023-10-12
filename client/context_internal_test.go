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

	clientv1beta1 "buf.build/gen/go/lekkodev/sdk/protocolbuffers/go/lekko/client/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContext(t *testing.T) {
	ctx := context.Background()

	m := map[string]interface{}{
		"a": 42,
	}
	ctx = Merge(ctx, m)

	output := fromContext(ctx)
	assert.EqualValues(t, m, output)
}

func TestContextMerge(t *testing.T) {
	ctx := context.Background()

	m1 := map[string]interface{}{
		"a": 42,
	}
	ctx = Merge(ctx, m1)

	m2 := map[string]interface{}{
		"b": 12,
	}
	ctx = Merge(ctx, m2)

	output := fromContext(ctx)
	assert.Contains(t, output, "a")
	assert.Contains(t, output, "b")
}

func TestContextMergeOverwrite(t *testing.T) {
	ctx := context.Background()

	m1 := map[string]interface{}{
		"a": 42,
	}
	ctx = Merge(ctx, m1)

	m2 := map[string]interface{}{
		"a": 12,
	}
	ctx = Merge(ctx, m2)

	output := fromContext(ctx)
	assert.EqualValues(t, m2, output)
}

func TestContextMergeDoesNotModifyArgs(t *testing.T) {
	ctx := context.Background()

	m1 := map[string]interface{}{
		"a": 42,
	}
	ctx = Merge(ctx, m1)

	m2 := map[string]interface{}{
		"b": 12,
	}
	_ = Merge(ctx, m2)

	assert.Len(t, m1, 1)
	assert.Len(t, m2, 1)
}

func TestAddContextPairs(t *testing.T) {
	ctx := context.Background()

	ctx = Add(ctx, "a", 1)
	ctx = Add(ctx, "b", 2)
	ctx = Add(ctx, "a", 3)

	output := fromContext(ctx)
	assert.Contains(t, output, "a")
	assert.Equal(t, output["a"], 3)
	assert.Contains(t, output, "b")
}

func TestContextToProto(t *testing.T) {
	m := map[string]interface{}{
		"a": int64(1),
		"b": int(2),
		"c": uint8(1),
		"d": 0.1,
		"e": float32(0.1),
		"f": false,
		"g": "foo",
	}
	p, err := toProto(m)
	require.NoError(t, err)
	for k := range m {
		assert.Contains(t, p, k)
	}
}

func TestContextToProtoUnsupportedType(t *testing.T) {
	m := map[string]interface{}{
		"a": struct{}{},
	}
	_, err := toProto(m)
	assert.Error(t, err)
}

func TestContextFromProto(t *testing.T) {
	p := map[string]*clientv1beta1.Value{
		"a": {
			Kind: &clientv1beta1.Value_BoolValue{BoolValue: true},
		},
		"b": {
			Kind: &clientv1beta1.Value_IntValue{IntValue: 42},
		},
		"c": {
			Kind: &clientv1beta1.Value_DoubleValue{DoubleValue: 1.98},
		},
		"d": {
			Kind: &clientv1beta1.Value_StringValue{StringValue: "foo"},
		},
	}
	lc, err := FromProto(p)
	require.NoError(t, err)
	assert.EqualValues(t, lc["a"], true)
	assert.EqualValues(t, lc["b"], int64(42))
	assert.EqualValues(t, lc["c"], float64(1.98))
	assert.EqualValues(t, lc["d"], "foo")
}

func TestContextFromProtoUnknownType(t *testing.T) {
	p := map[string]*clientv1beta1.Value{
		"a": {},
	}
	_, err := FromProto(p)
	require.Error(t, err)
}
