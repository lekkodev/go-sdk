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
	"strings"

	backendv1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/backend/v1beta1"
	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func ValToAny(featureType featurev1beta1.FeatureType, v any) *anypb.Any {
	var err error
	dest := &anypb.Any{}
	switch featureType {
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
		err = errors.Errorf("unknown feature type %v", featureType)
	}
	if err != nil {
		panic(err)
	}
	return dest
}

func Feature(ft featurev1beta1.FeatureType, value any) *backendv1beta1.Feature {
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
