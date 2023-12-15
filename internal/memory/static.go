// Copyright 2023 Lekko Technologies, Inc.
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

package memory

import (
	"context"
    "errors"

    featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
    "github.com/lekkodev/go-sdk/pkg/eval"
	"google.golang.org/protobuf/proto"
)

func NewStaticStore(featureBytes map[string]map[string][]byte) (Store, error) {
    features := make(map[string]map[string]*featurev1beta1.Feature)
    for ns, fm := range featureBytes {
        features[ns] = make(map[string]*featurev1beta1.Feature)
        for key, fb := range fm {
            f := &featurev1beta1.Feature{}
            err := proto.Unmarshal(fb, f)
            if err != nil {
                return nil, err
            }
            features[ns][key] = f
        }
    }
    return &staticStore{features}, nil
}


type staticStore struct {
    features map[string]map[string]*featurev1beta1.Feature
}


func (s *staticStore) Evaluate(key string, namespace string, lekkoContext map[string]interface{}, dest proto.Message) error {
    ns, ok := s.features[namespace]
    if !ok {
        return errors.New("Unknown Namespace")
    }
    cfg, ok := ns[key]
    if !ok {
        return errors.New("Unknown Key")
    }
    evaluableConfig := eval.NewV1Beta3(cfg, namespace)
    a, _, err := evaluableConfig.Evaluate(lekkoContext)
    if err != nil {
        return err
    }
    return a.UnmarshalTo(dest)
}

func (s *staticStore) Close(ctx context.Context) error {
	return nil
}
