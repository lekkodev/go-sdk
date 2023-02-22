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

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/lekkodev/cli/pkg/feature"
	"github.com/lekkodev/cli/pkg/metadata"
	"github.com/lekkodev/cli/pkg/repo"
)

type noopAuthProvider struct{}

func (*noopAuthProvider) GetUsername() string { return "" }
func (*noopAuthProvider) GetToken() string    { return "" }

// The static provider will read from a git repo via a relative path.
// This may be useful for local testing when a sidecar is not available.
func NewStaticProvider(pathToRoot string) (Provider, error) {
	ctx := context.TODO()
	r, err := repo.NewLocal(pathToRoot, &noopAuthProvider{})
	if err != nil {
		return nil, errors.Wrap(err, "unable to initialize static bootstrap")
	}

	rootMD, nsMDs, err := r.ParseMetadata(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse config repo metadata")
	}

	return &staticProvider{repo: r, rootMD: rootMD, nsMDs: nsMDs}, nil
}

type staticProvider struct {
	repo   *repo.Repo
	rootMD *metadata.RootConfigRepoMetadata
	nsMDs  map[string]*metadata.NamespaceConfigRepoMetadata
}

func (f *staticProvider) eval(ctx context.Context, key string, namespace string) (*anypb.Any, feature.FeatureType, error) {
	return f.repo.Eval(ctx, namespace, key, fromContext(ctx))
}

func (f *staticProvider) GetBoolFeature(ctx context.Context, key string, namespace string) (bool, error) {
	a, ft, err := f.eval(ctx, key, namespace)
	if err != nil {
		return false, err
	}
	if err := expectFeatureType(feature.FeatureTypeBool, ft); err != nil {
		return false, err
	}
	boolVal := new(wrapperspb.BoolValue)
	if !a.MessageIs(boolVal) {
		return false, fmt.Errorf("invalid type in config %T", a)
	}
	if err := a.UnmarshalTo(boolVal); err != nil {
		return false, err
	}
	return boolVal.GetValue(), nil
}

func (f *staticProvider) GetIntFeature(ctx context.Context, key string, namespace string) (int64, error) {
	a, ft, err := f.eval(ctx, key, namespace)
	if err != nil {
		return 0, err
	}
	if err := expectFeatureType(feature.FeatureTypeInt, ft); err != nil {
		return 0, err
	}
	intVal := new(wrapperspb.Int64Value)
	if !a.MessageIs(intVal) {
		return 0, fmt.Errorf("invalid type in config %T", a)
	}
	if err := a.UnmarshalTo(intVal); err != nil {
		return 0, err
	}
	return intVal.GetValue(), nil
}

func (f *staticProvider) GetFloatFeature(ctx context.Context, key string, namespace string) (float64, error) {
	a, ft, err := f.eval(ctx, key, namespace)
	if err != nil {
		return 0, err
	}
	if err := expectFeatureType(feature.FeatureTypeFloat, ft); err != nil {
		return 0, err
	}
	floatVal := new(wrapperspb.DoubleValue)
	if !a.MessageIs(floatVal) {
		return 0, fmt.Errorf("invalid type in config %T", a)
	}
	if err := a.UnmarshalTo(floatVal); err != nil {
		return 0, err
	}
	return floatVal.GetValue(), nil
}

func (f *staticProvider) GetStringFeature(ctx context.Context, key string, namespace string) (string, error) {
	a, ft, err := f.eval(ctx, key, namespace)
	if err != nil {
		return "", err
	}
	if err := expectFeatureType(feature.FeatureTypeString, ft); err != nil {
		return "", err
	}
	stringVal := new(wrapperspb.StringValue)
	if !a.MessageIs(stringVal) {
		return "", fmt.Errorf("invalid type in config %T", a)
	}
	if err := a.UnmarshalTo(stringVal); err != nil {
		return "", err
	}
	return stringVal.GetValue(), nil
}

func (f *staticProvider) GetProtoFeature(ctx context.Context, key string, namespace string, result proto.Message) error {
	a, ft, err := f.eval(ctx, key, namespace)
	if err != nil {
		return err
	}
	if err := expectFeatureType(feature.FeatureTypeProto, ft); err != nil {
		return err
	}
	if err := a.UnmarshalTo(result); err != nil {
		return err
	}
	return nil
}
func (f *staticProvider) GetJSONFeature(ctx context.Context, key string, namespace string, result interface{}) error {
	a, ft, err := f.eval(ctx, key, namespace)
	if err != nil {
		return err
	}
	if err := expectFeatureType(feature.FeatureTypeJSON, ft); err != nil {
		return err
	}
	val := &structpb.Value{}
	if !a.MessageIs(val) {
		return fmt.Errorf("invalid type %T", a)
	}
	if err := a.UnmarshalTo(val); err != nil {
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

func expectFeatureType(expected, actual feature.FeatureType) error {
	if expected == actual {
		return nil
	}
	return errors.Errorf("requested feature is of type %s, not %s", actual, expected)
}
