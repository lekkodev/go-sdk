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

package memory

import (
	"context"
	"errors"

	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
	"github.com/lekkodev/go-sdk/pkg/eval"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/anypb"
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

func (s *staticStore) EvaluateAny(key string, namespace string, lc map[string]interface{}) (protoreflect.ProtoMessage, *StoredConfig, error) {
	return nil, nil, nil
}

func (s *staticStore) Evaluate(key string, namespace string, lekkoContext map[string]interface{}, dest proto.Message) (*StoredConfig, error) {
	var typeURL string
	var targetFieldNumber uint64
	targetFieldNumber = 0
	for { // TODO - stop loops (especially for server side eval..
		ns, ok := s.features[namespace]
		if !ok {
			return nil, errors.New("unknown namespace")
		}
		cfg, ok := ns[key]
		if !ok {
			return nil, errors.New("unknown key")
		}
		evaluableConfig := eval.NewV1Beta3(cfg, namespace, nil)
		a, _, err := evaluableConfig.Evaluate(lekkoContext)
		if err != nil {
			return nil, err
		}
		if a.TypeUrl == "type.googleapis.com/lekko.protobuf.ConfigCall" {
			b := a.Value
			for len(b) > 0 {
				fid, wireType, n := protowire.ConsumeTag(b)
				if n < 0 {
					return nil, protowire.ParseError(n)
				}
				b = b[n:]
				switch fid {
				case 1:
					typeURL, n = protowire.ConsumeString(b)
				case 2:
					namespace, n = protowire.ConsumeString(b)
				case 3:
					key, n = protowire.ConsumeString(b)
				case 4:
					targetFieldNumber, n = protowire.ConsumeVarint(b)
				default:
					n = protowire.ConsumeFieldValue(fid, wireType, b)
				}
				if n < 0 {
					return nil, protowire.ParseError(n)
				}
				b = b[n:]
			}
		} else {
			if targetFieldNumber > 0 {
				b := a.Value
				for len(b) > 0 {
					fid, wireType, n := protowire.ConsumeTag(b)
					if n < 0 {
						return nil, protowire.ParseError(n)
					}
					b = b[n:]
					n = protowire.ConsumeFieldValue(fid, wireType, b)
					if n < 0 {
						return nil, protowire.ParseError(n)
					}
					if targetFieldNumber == uint64(fid) {
						var value []byte
						value = protowire.AppendTag(value, 1, protowire.BytesType)
						value = protowire.AppendBytes(value, b[:n])
						a = &anypb.Any{
							TypeUrl: typeURL,
							Value:   value,
						}
						// TODO default values and do we want to support deeper nesting?
						err = a.UnmarshalTo(dest)
						if err != nil {
							return nil, err
						}
						return &StoredConfig{
							Config: cfg,
						}, nil
					} else {
						b = b[n:]
					}
				}
			}
			err = a.UnmarshalTo(dest)
			if err != nil {
				return nil, err
			}
			return &StoredConfig{
				Config: cfg,
			}, nil
		}
	}
}

func (s *staticStore) Close(ctx context.Context) error {
	return nil
}
