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

package memory

import (
	"context"
	"fmt"
	"testing"

	"github.com/lekkodev/go-sdk/testdata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type TestStore struct {
	configs  map[string]*anypb.Any
	closeErr error
}

func (ts *TestStore) Evaluate(key string, namespace string, lc map[string]interface{}, dest proto.Message) (*StoredConfig, error) {
	a, ok := ts.configs[key]
	if !ok {
		return nil, errors.Errorf("key %s not found", key)
	}
	return &StoredConfig{}, a.UnmarshalTo(dest)
}

func (ts *TestStore) Close(ctx context.Context) error {
	return ts.closeErr
}

func TestHashUpdateRequest(t *testing.T) {
	req := &updateRequest{
		contents: testdata.RepositoryContents(),
	}
	require.NoError(t, req.calculateContentHash())
	require.NotNil(t, req.contentHash)
	expected := *req.contentHash
	bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(req.contents)
	require.NoError(t, err)
	// test the determinism of the hash
	for i := 0; i < 10; i++ {
		require.Equal(t, expected, hashContentsSHA256(bytes))
	}
}

// Benchmarks

type configKey struct {
	namespace, config string
}

func BenchmarkMapKeyAccess(b *testing.B) {
	type benchmarkData struct {
		namespaceName string
		configName    string
	}
	var data []benchmarkData
	for i := 0; i < 5; i++ {
		for j := 0; j < 100; j++ {
			data = append(data, benchmarkData{namespaceName: fmt.Sprint(i), configName: fmt.Sprint(j)})
		}
	}

	structMap := make(map[configKey]configData)
	for _, d := range data {
		structMap[configKey{
			namespace: d.namespaceName,
			config:    d.configName,
		}] = configData{}
	}

	mapMap := make(map[string]map[string]configData)
	for _, d := range data {
		// see if ns exists
		_, ok := mapMap[d.namespaceName]
		if !ok {
			mapMap[d.namespaceName] = make(map[string]configData)
		}
		mapMap[d.namespaceName][d.configName] = configData{}
	}

	b.Run("struct map access", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, d := range data {
				_ = structMap[configKey{
					namespace: d.namespaceName,
					config:    d.configName,
				}]
			}
		}
	})

	b.Run("map map access", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			for _, d := range data {
				_ = mapMap[d.namespaceName][d.configName]
			}
		}
	})
}
