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

	/*	registry, err := r.BuildDynamicTypeRegistry(ctx, rootMD.ProtoDirectory)
		if err != nil {
			return nil, errors.Wrap(err, "failed to build dynamic type registry")
		}*/

	return &staticProvider{repo: r, rootMD: rootMD, nsMDs: nsMDs}, nil
}

type staticProvider struct {
	repo   *repo.Repo
	rootMD *metadata.RootConfigRepoMetadata
	nsMDs  map[string]*metadata.NamespaceConfigRepoMetadata
}

func (f *staticProvider) eval(ctx context.Context, key string, namespace string) (*anypb.Any, error) {
	return f.repo.Eval(ctx, namespace, key, fromContext(ctx))
}

func (f *staticProvider) GetBoolFeature(ctx context.Context, key string, namespace string) (bool, error) {
	a, err := f.eval(ctx, key, namespace)
	if err != nil {
		return false, err
	}
	boolVal := new(wrapperspb.BoolValue)
	if !a.MessageIs(boolVal) {
		return false, fmt.Errorf("invalid type in config %T", a)
	}
	if err := a.UnmarshalTo(boolVal); err != nil {
		return false, err
	}
	return boolVal.Value, nil
}

func (f *staticProvider) GetStringFeature(ctx context.Context, key string, namespace string) (string, error) {
	return "", fmt.Errorf("unimplemented")
}
func (f *staticProvider) GetProtoFeature(ctx context.Context, key string, namespace string, result proto.Message) error {
	a, err := f.eval(ctx, key, namespace)
	if err != nil {
		return err
	}
	if err := a.UnmarshalTo(result); err != nil {
		return err
	}
	return nil
}
func (f *staticProvider) GetJSONFeature(ctx context.Context, key string, namespace string, result interface{}) error {
	a, err := f.eval(ctx, key, namespace)
	if err != nil {
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
