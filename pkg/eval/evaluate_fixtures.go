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
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
	"github.com/lekkodev/rules/pkg/parser"
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

func NewRuleLangInvalid() string {
	return "user_id IN (1, 2)"
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

func genConstraint(ruleStr string, value *anypb.Any, constraints ...*featurev1beta1.Constraint) *featurev1beta1.Constraint {
	ruleASTV3, err := parser.BuildASTV3(ruleStr)
	if err != nil {
		panic(err)
	}
	return &featurev1beta1.Constraint{
		Rule:        ruleStr,
		RuleAstNew:  ruleASTV3,
		Value:       value,
		Constraints: constraints,
	}
}
