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
		WithBool("bool-feature", true).
		WithProto("proto-feature", protoBytes).
		WithJSON("json-feature", jsonBytes).
		WithString("repeat-key", "foo").
		WithError("repeat-key", errors.New("error")).
		WithFloat("float-feature", 1.0).
		WithInt("int-feature", 9)

	ctx := context.Background()

	result, err := testClient.GetBool(ctx, "bool-feature")
	assert.NoError(t, err)
	assert.True(t, result)

	protoResult := &durationpb.Duration{}
	assert.NoError(t, testClient.GetProto(ctx, "proto-feature", protoResult))
	assert.True(t, proto.Equal(protoValue, protoResult))

	jsonResult := []int{}
	assert.NoError(t, testClient.GetJSON(ctx, "json-feature", &jsonResult))
	assert.Equal(t, jsonValue, jsonResult)

	_, err = testClient.GetString(ctx, "repeat-key")
	assert.Error(t, err)

	_, err = testClient.GetInt(ctx, "bool-feature")
	assert.Error(t, err, "expecting type mismatch")

	intResult, err := testClient.GetInt(ctx, "int-feature")
	assert.NoError(t, err)
	assert.Equal(t, int64(9), intResult)

	floatResult, err := testClient.GetFloat(ctx, "float-feature")
	assert.NoError(t, err)
	assert.Equal(t, 1.0, floatResult)

	_, err = testClient.GetBool(ctx, "not-found")
	assert.Error(t, err, "expecting not found error")
}
