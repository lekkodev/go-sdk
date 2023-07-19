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

package eval

import (
	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
)

// Indicates the lekko-specific types that are allowed in feature flags.
type FeatureType string

const (
	FeatureTypeBool   FeatureType = "bool"
	FeatureTypeInt    FeatureType = "int"
	FeatureTypeFloat  FeatureType = "float"
	FeatureTypeString FeatureType = "string"
	FeatureTypeProto  FeatureType = "proto"
	FeatureTypeJSON   FeatureType = "json"
)

func FeatureTypes() []string {
	var ret []string
	for _, ftype := range []FeatureType{
		FeatureTypeBool,
		FeatureTypeString,
		FeatureTypeInt,
		FeatureTypeFloat,
		FeatureTypeJSON,
		FeatureTypeProto,
	} {
		ret = append(ret, string(ftype))
	}
	return ret
}

func (ft FeatureType) ToProto() featurev1beta1.FeatureType {
	switch ft {
	case FeatureTypeBool:
		return featurev1beta1.FeatureType_FEATURE_TYPE_BOOL
	case FeatureTypeInt:
		return featurev1beta1.FeatureType_FEATURE_TYPE_INT
	case FeatureTypeFloat:
		return featurev1beta1.FeatureType_FEATURE_TYPE_FLOAT
	case FeatureTypeString:
		return featurev1beta1.FeatureType_FEATURE_TYPE_STRING
	case FeatureTypeJSON:
		return featurev1beta1.FeatureType_FEATURE_TYPE_JSON
	case FeatureTypeProto:
		return featurev1beta1.FeatureType_FEATURE_TYPE_PROTO
	default:
		return featurev1beta1.FeatureType_FEATURE_TYPE_UNSPECIFIED
	}
}

func (ft FeatureType) IsPrimitive() bool {
	primitiveTypes := map[FeatureType]struct{}{
		FeatureTypeBool:   {},
		FeatureTypeString: {},
		FeatureTypeInt:    {},
		FeatureTypeFloat:  {},
	}
	_, ok := primitiveTypes[ft]
	return ok
}

func FeatureTypeFromProto(ft featurev1beta1.FeatureType) FeatureType {
	switch ft {
	case featurev1beta1.FeatureType_FEATURE_TYPE_BOOL:
		return FeatureTypeBool
	case featurev1beta1.FeatureType_FEATURE_TYPE_INT:
		return FeatureTypeInt
	case featurev1beta1.FeatureType_FEATURE_TYPE_FLOAT:
		return FeatureTypeFloat
	case featurev1beta1.FeatureType_FEATURE_TYPE_STRING:
		return FeatureTypeString
	case featurev1beta1.FeatureType_FEATURE_TYPE_JSON:
		return FeatureTypeJSON
	case featurev1beta1.FeatureType_FEATURE_TYPE_PROTO:
		return FeatureTypeProto
	default:
		return ""
	}
}
