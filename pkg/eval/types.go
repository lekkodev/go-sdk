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
type ConfigType string

const (
	ConfigTypeBool   ConfigType = "bool"
	ConfigTypeInt    ConfigType = "int"
	ConfigTypeFloat  ConfigType = "float"
	ConfigTypeString ConfigType = "string"
	ConfigTypeProto  ConfigType = "proto"
	ConfigTypeJSON   ConfigType = "json"
)

func ConfigTypes() []string {
	var ret []string
	for _, ctype := range []ConfigType{
		ConfigTypeBool,
		ConfigTypeString,
		ConfigTypeInt,
		ConfigTypeFloat,
		ConfigTypeJSON,
		ConfigTypeProto,
	} {
		ret = append(ret, string(ctype))
	}
	return ret
}

func (ft ConfigType) ToProto() featurev1beta1.FeatureType {
	switch ft {
	case ConfigTypeBool:
		return featurev1beta1.FeatureType_FEATURE_TYPE_BOOL
	case ConfigTypeInt:
		return featurev1beta1.FeatureType_FEATURE_TYPE_INT
	case ConfigTypeFloat:
		return featurev1beta1.FeatureType_FEATURE_TYPE_FLOAT
	case ConfigTypeString:
		return featurev1beta1.FeatureType_FEATURE_TYPE_STRING
	case ConfigTypeJSON:
		return featurev1beta1.FeatureType_FEATURE_TYPE_JSON
	case ConfigTypeProto:
		return featurev1beta1.FeatureType_FEATURE_TYPE_PROTO
	default:
		return featurev1beta1.FeatureType_FEATURE_TYPE_UNSPECIFIED
	}
}

func (ft ConfigType) IsPrimitive() bool {
	primitiveTypes := map[ConfigType]struct{}{
		ConfigTypeBool:   {},
		ConfigTypeString: {},
		ConfigTypeInt:    {},
		ConfigTypeFloat:  {},
	}
	_, ok := primitiveTypes[ft]
	return ok
}

func ConfigTypeFromProto(ft featurev1beta1.FeatureType) ConfigType {
	switch ft {
	case featurev1beta1.FeatureType_FEATURE_TYPE_BOOL:
		return ConfigTypeBool
	case featurev1beta1.FeatureType_FEATURE_TYPE_INT:
		return ConfigTypeInt
	case featurev1beta1.FeatureType_FEATURE_TYPE_FLOAT:
		return ConfigTypeFloat
	case featurev1beta1.FeatureType_FEATURE_TYPE_STRING:
		return ConfigTypeString
	case featurev1beta1.FeatureType_FEATURE_TYPE_JSON:
		return ConfigTypeJSON
	case featurev1beta1.FeatureType_FEATURE_TYPE_PROTO:
		return ConfigTypeProto
	default:
		return ""
	}
}
