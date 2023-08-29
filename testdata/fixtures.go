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

package testdata

import (
	"fmt"
	"strings"

	backendv1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/backend/v1beta1"
	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func ValToAny(configType featurev1beta1.FeatureType, v any) *anypb.Any {
	var err error
	dest := &anypb.Any{}
	switch configType {
	case featurev1beta1.FeatureType_FEATURE_TYPE_BOOL:
		tv, ok := v.(bool)
		if !ok {
			panic("invalid type")
		}
		w := &wrapperspb.BoolValue{Value: tv}
		err = anypb.MarshalFrom(dest, w, proto.MarshalOptions{})
	case featurev1beta1.FeatureType_FEATURE_TYPE_STRING:
		tv, ok := v.(string)
		if !ok {
			panic("invalid type")
		}
		w := &wrapperspb.StringValue{Value: tv}
		err = anypb.MarshalFrom(dest, w, proto.MarshalOptions{})
	case featurev1beta1.FeatureType_FEATURE_TYPE_FLOAT:
		tv, ok := v.(float64)
		if !ok {
			panic("invalid type")
		}
		w := &wrapperspb.DoubleValue{Value: tv}
		err = anypb.MarshalFrom(dest, w, proto.MarshalOptions{})
	case featurev1beta1.FeatureType_FEATURE_TYPE_INT:
		tv, ok := v.(int64)
		if !ok {
			panic("invalid type")
		}
		w := &wrapperspb.Int64Value{Value: tv}
		err = anypb.MarshalFrom(dest, w, proto.MarshalOptions{})
	case featurev1beta1.FeatureType_FEATURE_TYPE_PROTO:
		pv, ok := v.(proto.Message)
		if !ok {
			panic("invalid type")
		}
		err = anypb.MarshalFrom(dest, pv, proto.MarshalOptions{})
	case featurev1beta1.FeatureType_FEATURE_TYPE_JSON:
		sv, structErr := structpb.NewValue(v)
		if structErr != nil {
			panic(err)
		}
		err = anypb.MarshalFrom(dest, sv, proto.MarshalOptions{})
	default:
		err = errors.Errorf("unknown feature type %v", configType)
	}
	if err != nil {
		panic(err)
	}
	return dest
}

func Config(ft featurev1beta1.FeatureType, value any) *backendv1beta1.Feature {
	a := ValToAny(ft, value)
	parts := strings.Split(ft.String(), "_")
	name := strings.ToLower(parts[len(parts)-1])
	return &backendv1beta1.Feature{
		Name: name,
		Sha:  name,
		Feature: &featurev1beta1.Feature{
			Key: name,
			Tree: &featurev1beta1.Tree{
				Default: a,
			},
			Type: ft,
		},
	}
}

func Namespace(name string) *backendv1beta1.Namespace {
	return &backendv1beta1.Namespace{
		Name: name,
		Features: []*backendv1beta1.Feature{
			{
				Name: "example",
				Sha:  "e5a024dd8bf777c3a23ccf1da21fa29aebc5bf28",
				Feature: &featurev1beta1.Feature{
					Key:         "example",
					Description: "an example configuration",
					Tree: &featurev1beta1.Tree{
						Default: ValToAny(featurev1beta1.FeatureType_FEATURE_TYPE_BOOL, true),
					},
					Type: featurev1beta1.FeatureType_FEATURE_TYPE_BOOL,
				},
			},
		},
	}
}

func RepositoryContents() *backendv1beta1.GetRepositoryContentsResponse {
	var namespaces []*backendv1beta1.Namespace
	for i := 0; i < 200; i++ {
		namespaces = append(namespaces, Namespace(fmt.Sprintf("%d", i)))
	}
	return &backendv1beta1.GetRepositoryContentsResponse{
		CommitSha:  "0a06da7c1c4a37bed9ff974009265da5196e81ca",
		Namespaces: namespaces,
	}
}
