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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/durationpb"
)

var ns = "test-ns"

func TestTestClient(t *testing.T) {
	// proto
	protoValue := durationpb.New(time.Hour)
	protoBytes, err := proto.Marshal(protoValue)
	require.NoError(t, err)
	// json
	jsonValue := []int{1, 2, 3}
	jsonBytes, err := json.Marshal(jsonValue)
	require.NoError(t, err)

	testClient := NewTestClient().
		WithBool(ns, "bool-config", true).
		WithProto(ns, "proto-config", protoBytes).
		WithJSON(ns, "json-config", jsonBytes).
		WithString(ns, "repeat-key", "foo").
		WithError(ns, "repeat-key", errors.New("error")).
		WithFloat(ns, "float-config", 1.0).
		WithInt(ns, "int-config", 9)

	ctx := context.Background()

	result, err := testClient.GetBool(ctx, ns, "bool-config")
	assert.NoError(t, err)
	assert.True(t, result)

	protoResult := &durationpb.Duration{}
	assert.NoError(t, testClient.GetProto(ctx, ns, "proto-config", protoResult))
	assert.True(t, proto.Equal(protoValue, protoResult))

	jsonResult := []int{}
	assert.NoError(t, testClient.GetJSON(ctx, ns, "json-config", &jsonResult))
	assert.Equal(t, jsonValue, jsonResult)

	_, err = testClient.GetString(ctx, ns, "repeat-key")
	assert.Error(t, err)

	_, err = testClient.GetInt(ctx, ns, "bool-config")
	assert.Error(t, err, "expecting type mismatch")

	intResult, err := testClient.GetInt(ctx, ns, "int-config")
	assert.NoError(t, err)
	assert.Equal(t, int64(9), intResult)

	floatResult, err := testClient.GetFloat(ctx, ns, "float-config")
	assert.NoError(t, err)
	assert.Equal(t, 1.0, floatResult)

	_, err = testClient.GetBool(ctx, ns, "not-found")
	assert.Error(t, err, "expecting not found error")
}
