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
	"fmt"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
	rulesv1beta3 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/rules/v1beta3"
)

func NewBasicFeatureOnBeta2() *featurev1beta1.Feature {
	return &featurev1beta1.Feature{
		Key: "basic_feature_on",
		Tree: &featurev1beta1.Tree{
			Default:     NewAnyTrue(),
			Constraints: nil,
		},
	}
}

func NewBasicFeatureOffBeta2() *featurev1beta1.Feature {
	return &featurev1beta1.Feature{
		Key: "basic_feature_off",
		Tree: &featurev1beta1.Tree{
			Default:     NewAnyFalse(),
			Constraints: nil,
		},
	}
}

func NewFeatureOnForUserIDBeta2() *featurev1beta1.Feature {
	return &featurev1beta1.Feature{
		Key: "feature_on_for_user_id",
		Tree: &featurev1beta1.Tree{
			Default:     NewAnyFalse(),
			Constraints: []*featurev1beta1.Constraint{genConstraint("user_id == 1", NewAnyTrue())},
		},
	}
}

func NewFeatureOnForUserIDsBeta2() *featurev1beta1.Feature {
	return &featurev1beta1.Feature{
		Key: "feature_on_for_user_ids",
		Tree: &featurev1beta1.Tree{
			Default:     NewAnyFalse(),
			Constraints: []*featurev1beta1.Constraint{genConstraint("user_id IN [1, 2]", NewAnyTrue())},
		},
	}
}

func NewConstraintOnForUserIDBeta2() *featurev1beta1.Constraint {
	return &featurev1beta1.Constraint{
		Rule:  NewRuleLangEqualUserID(),
		Value: NewAnyTrue(),
	}
}

func NewConstraintOnForUserIDsBeta2() *featurev1beta1.Constraint {
	return &featurev1beta1.Constraint{
		Rule:  NewRuleLangContainsUserID(),
		Value: NewAnyTrue(),
	}
}

func NewAnyFalse() *anypb.Any {
	a, err := anypb.New(&wrapperspb.BoolValue{Value: false})
	if err != nil {
		panic(err)
	}
	return a
}

func NewAnyTrue() *anypb.Any {
	a, err := anypb.New(&wrapperspb.BoolValue{Value: true})
	if err != nil {
		panic(err)
	}
	return a
}

func NewAnyInt(i int64) *anypb.Any {
	a, err := anypb.New(&wrapperspb.Int64Value{Value: i})
	if err != nil {
		panic(err)
	}
	return a
}

func NewRuleLangEqualUserID() string {
	return "user_id == 1"
}

func NewRuleLangContainsUserID() string {
	return "user_id IN [1, 2]"
}

func NewComplexTreeFeature() *featurev1beta1.Feature {
	return &featurev1beta1.Feature{
		Key: "complex-tree",
		Tree: &featurev1beta1.Tree{
			Default: NewAnyInt(12),
			Constraints: []*featurev1beta1.Constraint{
				genConstraint("a == 1", NewAnyInt(38), genConstraint("x IN [\"a\", \"b\"]", NewAnyInt(108))),
				genConstraint("a > 10", nil, genConstraint("x == \"c\"", NewAnyInt(21))),
				genConstraint("a > 5", NewAnyInt(23)),
			},
		},
	}
}

func NewDependencyTreeFeature() *featurev1beta1.Feature {
	return &featurev1beta1.Feature{
		Key: "dependency-tree",
		Tree: &featurev1beta1.Tree{
			Default: NewAnyInt(50),
			Constraints: []*featurev1beta1.Constraint{
				genConstraint("evaluate_to(\"segments\", \"alpha\")", NewAnyInt(10)),
				genConstraint("a > 5", NewAnyInt(20)),
				genConstraint("evaluate_to(\"segments\", \"beta\")", NewAnyInt(30)),
			},
		},
	}
}

func genConstraint(ruleStr string, value *anypb.Any, constraints ...*featurev1beta1.Constraint) *featurev1beta1.Constraint {
	rulesMap := rulesMap()
	ast, ok := rulesMap[ruleStr]
	if !ok {
		panic(fmt.Sprintf("ast mapping not found for rule '%s'", ruleStr))
	}
	return &featurev1beta1.Constraint{
		Rule:        ruleStr,
		RuleAstNew:  ast,
		Value:       value,
		Constraints: constraints,
	}
}

func rulesMap() map[string]*rulesv1beta3.Rule {
	ret := make(map[string]*rulesv1beta3.Rule)
	ret["evaluate_to(\"segments\", \"alpha\")"] = &rulesv1beta3.Rule{
		Rule: &rulesv1beta3.Rule_CallExpression{
			CallExpression: &rulesv1beta3.CallExpression{
				Function: &rulesv1beta3.CallExpression_EvaluateTo_{
					EvaluateTo: &rulesv1beta3.CallExpression_EvaluateTo{
						ConfigName:  "segments",
						ConfigValue: newValue("alpha"),
					},
				},
			},
		},
	}
	ret["evaluate_to(\"segments\", \"beta\")"] = &rulesv1beta3.Rule{
		Rule: &rulesv1beta3.Rule_CallExpression{
			CallExpression: &rulesv1beta3.CallExpression{
				Function: &rulesv1beta3.CallExpression_EvaluateTo_{
					EvaluateTo: &rulesv1beta3.CallExpression_EvaluateTo{
						ConfigName:  "segments",
						ConfigValue: newValue("beta"),
					},
				},
			},
		},
	}
	ret["a == 1"] = atom("a", "==", structpb.NewNumberValue(1))
	ret["a > 10"] = atom("a", ">", structpb.NewNumberValue(10))
	ret["a > 5"] = atom("a", ">", structpb.NewNumberValue(5))
	ret["user_id == 1"] = atom("user_id", "==", structpb.NewNumberValue(1))
	ret["user_id IN [1, 2]"] = atom("user_id", "IN", newValue([]interface{}{1, 2}))
	ret["x IN [\"a\", \"b\"]"] = atom("x", "IN", newValue([]interface{}{"a", "b"}))
	ret["x == \"c\""] = atom("x", "==", newValue("c"))
	return ret
}

func atom(ctxKey, operator string, cmpValue *structpb.Value) *rulesv1beta3.Rule {
	return &rulesv1beta3.Rule{
		Rule: &rulesv1beta3.Rule_Atom{
			Atom: &rulesv1beta3.Atom{
				ContextKey:         ctxKey,
				ComparisonValue:    cmpValue,
				ComparisonOperator: operatorFromString(operator),
			},
		},
	}
}

func operatorFromString(op string) rulesv1beta3.ComparisonOperator {
	switch op {
	case "==":
		return rulesv1beta3.ComparisonOperator_COMPARISON_OPERATOR_EQUALS
	case ">":
		return rulesv1beta3.ComparisonOperator_COMPARISON_OPERATOR_GREATER_THAN
	case "<":
		return rulesv1beta3.ComparisonOperator_COMPARISON_OPERATOR_LESS_THAN
	case "IN":
		return rulesv1beta3.ComparisonOperator_COMPARISON_OPERATOR_CONTAINED_WITHIN
	default:
		panic(fmt.Sprintf("operator not handled '%s'", op))
	}
}

func newValue(from interface{}) *structpb.Value {
	val, err := structpb.NewValue(from)
	if err != nil {
		panic(err)
	}
	return val
}
