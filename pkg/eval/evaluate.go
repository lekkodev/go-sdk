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

// This package governs the specifics of a config, like what individual
// files make up a config.
package eval

import (
	"google.golang.org/protobuf/types/known/anypb"

	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
	rulesv1beta3 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/rules/v1beta3"
	"github.com/lekkodev/go-sdk/pkg/rules"
	"github.com/pkg/errors"
)

type EvaluableConfig interface {
	// Evaluate returns a protobuf.Any.
	// For user defined protos, we shouldn't attempt to Unmarshal
	// this unless we know the type. For primitive types, we can
	// safely unmarshal into BoolValue, StringValue, etc.
	Evaluate(featureCtx map[string]interface{}) (*anypb.Any, ResultPath, error)
	// Returns the config type (bool, string, json, proto, etc)
	// or "" if the type is not supported
	Type() ConfigType
}

// Stores the path of the tree node that returned the final value
// after successful evaluation. See the readme for an illustration.
type ResultPath []int

type v1beta3 struct {
	*featurev1beta1.Feature
	namespace                  string
	referencedConfigToValueMap map[string]interface{}
}

// evaluate constrcutor needs a getter to Config getConfig(key string)
func NewV1Beta3(f *featurev1beta1.Feature, namespace string, referencedConfigToValueMap map[string]interface{}) EvaluableConfig {
	return &v1beta3{f, namespace, referencedConfigToValueMap}
}

func (v1b3 *v1beta3) Type() ConfigType {
	return ConfigTypeFromProto(v1b3.GetType())
}

func (v1b3 *v1beta3) Evaluate(lekkoCtx map[string]interface{}) (*anypb.Any, ResultPath, error) {
	return v1b3.evaluate(lekkoCtx)
}

func (v1b3 *v1beta3) evaluate(context map[string]interface{}) (*anypb.Any, []int, error) {
	for i, constraint := range v1b3.GetTree().GetConstraints() {
		childVal, childPasses, childPath, err := v1b3.traverse(constraint, context)
		if err != nil {
			return nil, []int{}, err
		}
		if childPasses {
			if childVal != nil {
				return childVal, append([]int{i}, childPath...), nil
			}
			break
		}
	}
	return v1b3.GetTree().Default, []int{}, nil
}

func (v1b3 *v1beta3) traverse(constraint *featurev1beta1.Constraint, lekkoCtx map[string]interface{}) (*anypb.Any, bool, []int, error) {
	passes, err := v1b3.evaluateRule(constraint.GetRuleAstNew(), lekkoCtx)
	if err != nil {
		return nil, false, []int{}, errors.Wrap(err, "processing")
	}
	if !passes {
		// If the rule fails, we avoid further traversal
		return nil, passes, []int{}, nil
	}
	// rule passed
	retVal := constraint.Value // may be null
	for i, child := range constraint.GetConstraints() {
		childVal, childPasses, childPath, err := v1b3.traverse(child, lekkoCtx)
		if err != nil {
			return nil, false, []int{}, errors.Wrapf(err, "traverse %d", i)
		}
		if childPasses {
			// We may stop iterating. But first, remember the traversed
			// value if it exists
			if childVal != nil {
				return childVal, passes, append([]int{i}, childPath...), nil
			}
			break
		}
		// Child evaluation did not pass, continue iterating
	}
	return retVal, passes, []int{}, nil
}

func (v1b3 *v1beta3) evaluateRule(ruleV3 *rulesv1beta3.Rule, lekkoCtx map[string]interface{}) (bool, error) {
	passes, err := rules.NewV1Beta3(
		ruleV3,
		rules.EvalContext{Namespace: v1b3.namespace, FeatureName: v1b3.Key, ReferencedConfigToValueMap: v1b3.referencedConfigToValueMap}).EvaluateRule(lekkoCtx)
	if err != nil {
		return false, errors.Wrap(err, "evaluating rule v3")
	}
	return passes, nil
}
