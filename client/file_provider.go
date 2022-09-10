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
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/lekkodev/cli/pkg/encoding"
	"github.com/lekkodev/cli/pkg/feature"
	"github.com/lekkodev/cli/pkg/fs"
	"github.com/lekkodev/cli/pkg/metadata"
)

// The file provider will load the result of a file into memory.
// This file provider DOES NOT refresh from disk within the lifetime of the process.
// fsnotify or similar technology will be implemented in a different provider.
// This is meant as a backup for local testing when production configuration
// options are not available.
func NewFileProvider(pathToRoot string) (Provider, error) {
	// TODO: this function should be refactored into the cli from the SDK.
	rootMD, nsMDs, err := metadata.ParseFullConfigRepoMetadataStrict(context.Background(), pathToRoot, fs.LocalProvider())
	if err != nil {
		return nil, err
	}
	res := make(map[string]map[string]feature.EvaluableFeature)
	for _, ns := range rootMD.Namespaces {
		nsMap := make(map[string]feature.EvaluableFeature)
		nsMD, ok := nsMDs[ns]
		if !ok {
			return nil, fmt.Errorf("could not find namespace metadata: %s", ns)
		}
		ffs, err := feature.GroupFeatureFiles(context.Background(), filepath.Join(pathToRoot, ns), nsMD, fs.LocalProvider(), false)
		if err != nil {
			return nil, err
		}
		for _, ff := range ffs {
			evalF, err := encoding.ParseFeature(pathToRoot, ff, nsMD, fs.LocalProvider())
			if err != nil {
				return nil, err
			}
			nsMap[ff.Name] = evalF
		}
		res[ns] = nsMap
	}
	return &fileProvider{res}, nil
}

type fileProvider struct {
	namespaceToFeatureMap map[string]map[string]feature.EvaluableFeature
}

func (f *fileProvider) GetEvaluableFeature(ctx context.Context, key string, namespace string) (feature.EvaluableFeature, error) {
	nsMap, ok := f.namespaceToFeatureMap[namespace]
	if !ok {
		return nil, fmt.Errorf("unknown namepsace %s", namespace)
	}

	evalF, ok := nsMap[key]
	if !ok {
		return nil, fmt.Errorf("unknown feature key %s in namespace %s", key, namespace)
	}
	return evalF, nil
}

func (f *fileProvider) GetBoolFeature(ctx context.Context, key string, namespace string) (bool, error) {
	evalF, err := f.GetEvaluableFeature(ctx, key, namespace)
	if err != nil {
		return false, err
	}
	resp, err := evalF.Evaluate(fromContext(ctx))
	if err != nil {
		return false, err
	}
	boolVal := new(wrapperspb.BoolValue)
	if !resp.MessageIs(boolVal) {
		return false, fmt.Errorf("invalid type in config map %T", resp)
	}
	if err := resp.UnmarshalTo(boolVal); err != nil {
		return false, err
	}
	return boolVal.Value, nil
}

func (f *fileProvider) GetStringFeature(ctx context.Context, key string, namespace string) (string, error) {
	return "", fmt.Errorf("unimplemented")
}
func (f *fileProvider) GetProtoFeature(ctx context.Context, key string, namespace string, result proto.Message) error {
	evalF, err := f.GetEvaluableFeature(ctx, key, namespace)
	if err != nil {
		return err
	}
	resp, err := evalF.Evaluate(fromContext(ctx))
	if err != nil {
		return err
	}
	if err := resp.UnmarshalTo(result); err != nil {
		return err
	}
	return nil
}
func (f *fileProvider) GetJSONFeature(ctx context.Context, key string, namespace string, result interface{}) error {
	evalF, err := f.GetEvaluableFeature(ctx, key, namespace)
	if err != nil {
		return err
	}
	resp, err := evalF.Evaluate(fromContext(ctx))
	if err != nil {
		return err
	}
	val := &structpb.Value{}
	if !resp.MessageIs(val) {
		return fmt.Errorf("invalid type %T", resp)
	}
	if err := resp.UnmarshalTo(val); err != nil {
		return fmt.Errorf("failed to unmarshal any to value: %w", err)
	}
	bytes, err := val.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal value into bytes: %w", err)
	}
	if err := json.Unmarshal(bytes, result); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to unmarshal json into go type %T", result))
	}
	return nil
}
