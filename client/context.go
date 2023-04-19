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
	"fmt"

	backendv1beta1 "buf.build/gen/go/lekkodev/sdk/protocolbuffers/go/lekko/client/v1beta1"
)

// TODO: we need a better name that's not 'context'. Conditions? Features? Values?
// 'context' is overloaded in go.

// lekkoContext is the type of the value stored in the context
type lekkoContext map[string]interface{}

// lekkoKey is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type lekkoKey int

// lekkoKeyV1 is the key for the v1 representation of arbitrary realtime
// context that users of the golang sdk can pass in to lekko via context.Context.
var lekkoKeyV1 lekkoKey

// Merge allows you to pass arbitrary context variables in order to perform
// rules evaluation on your feature flags in real time.
// Priority is given to newly added keys in lekkoCtx.
// TODO: allow users to run in safe mode, which will throw errors on ctx conflicts.
// TODO: this is not thread-safe, make it thread-safe.
func Merge(ctx context.Context, lekkoCtx map[string]interface{}) context.Context {
	existing := fromContext(ctx)
	mergedCtx := make(lekkoContext, len(existing))
	for k, v := range existing {
		mergedCtx[k] = v
	}
	for k, v := range lekkoCtx {
		mergedCtx[k] = v
	}
	return context.WithValue(ctx, lekkoKeyV1, mergedCtx)
}

func Add(ctx context.Context, key string, value interface{}) context.Context {
	return Merge(ctx, map[string]interface{}{
		key: value,
	})
}

func fromContext(ctx context.Context) map[string]interface{} {
	lekkoCtx, ok := ctx.Value(lekkoKeyV1).(lekkoContext)
	if !ok {
		return make(map[string]interface{})
	}
	return lekkoCtx
}

func toProto(lc lekkoContext) (map[string]*backendv1beta1.Value, error) {
	ret := make(map[string]*backendv1beta1.Value)
	for k, v := range lc {
		protoValue := &backendv1beta1.Value{}
		switch tv := v.(type) {
		case bool:
			protoValue.Kind = &backendv1beta1.Value_BoolValue{BoolValue: tv}
		case string:
			protoValue.Kind = &backendv1beta1.Value_StringValue{StringValue: tv}
		case int:
			protoValue.Kind = &backendv1beta1.Value_IntValue{IntValue: int64(tv)}
		case int8:
			protoValue.Kind = &backendv1beta1.Value_IntValue{IntValue: int64(tv)}
		case int16:
			protoValue.Kind = &backendv1beta1.Value_IntValue{IntValue: int64(tv)}
		case int32:
			protoValue.Kind = &backendv1beta1.Value_IntValue{IntValue: int64(tv)}
		case int64:
			protoValue.Kind = &backendv1beta1.Value_IntValue{IntValue: tv}
		case uint:
			protoValue.Kind = &backendv1beta1.Value_IntValue{IntValue: int64(tv)}
		case uint16:
			protoValue.Kind = &backendv1beta1.Value_IntValue{IntValue: int64(tv)}
		case uint32:
			protoValue.Kind = &backendv1beta1.Value_IntValue{IntValue: int64(tv)}
		case uint64:
			protoValue.Kind = &backendv1beta1.Value_IntValue{IntValue: int64(tv)}
		case uint8:
			protoValue.Kind = &backendv1beta1.Value_IntValue{IntValue: int64(tv)}
		case float32:
			protoValue.Kind = &backendv1beta1.Value_DoubleValue{DoubleValue: float64(tv)}
		case float64:
			protoValue.Kind = &backendv1beta1.Value_DoubleValue{DoubleValue: tv}
		default:
			return nil, fmt.Errorf("context value of type %T not supported", tv)
		}
		ret[k] = protoValue
	}
	return ret, nil
}
