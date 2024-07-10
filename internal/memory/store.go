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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"

	backendv1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/backend/v1beta1"
	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
	clientv1beta1 "buf.build/gen/go/lekkodev/sdk/protocolbuffers/go/lekko/client/v1beta1"
	serverv1beta1 "buf.build/gen/go/lekkodev/sdk/protocolbuffers/go/lekko/server/v1beta1"
	"github.com/lekkodev/go-sdk/pkg/eval"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var (
	ErrConfigNotFound       error = fmt.Errorf("config not found")
	ErrCycleDependentConfig error = fmt.Errorf("cycle in config dependency")
)

func newStore(owner, repo string) *store {
	return &store{
		configs:   make(map[string]map[string]configData),
		ownerName: owner,
		repoName:  repo,
	}
}

type configData struct {
	config    *featurev1beta1.Feature
	configSHA string
}

type SerializableTypes struct {
	Types             *protoregistry.Types
	FileDescriptorSet *descriptorpb.FileDescriptorSet
}

func (st *SerializableTypes) AddFileDescriptor(fd protoreflect.FileDescriptor, checkNotExists bool) error {
	existingTypes := make(map[string]struct{})
	if checkNotExists {
		st.Types.RangeEnums(func(et protoreflect.EnumType) bool {
			existingTypes[string(et.Descriptor().FullName())] = struct{}{}
			return true
		})
		st.Types.RangeMessages(func(et protoreflect.MessageType) bool {
			existingTypes[string(et.Descriptor().FullName())] = struct{}{}
			return true
		})
		st.Types.RangeExtensions(func(et protoreflect.ExtensionType) bool {
			existingTypes[string(et.TypeDescriptor().FullName())] = struct{}{}
			return true
		})
	}
	var numRegistered int
	for i := 0; i < fd.Enums().Len(); i++ {
		ed := fd.Enums().Get(i)
		if _, ok := existingTypes[string(ed.FullName())]; ok {
			continue
		}
		if err := st.Types.RegisterEnum(dynamicpb.NewEnumType(ed)); err != nil {
			return errors.Wrap(err, "register enum")
		}
		numRegistered++
	}
	for i := 0; i < fd.Messages().Len(); i++ {
		md := fd.Messages().Get(i)
		if _, ok := existingTypes[string(md.FullName())]; ok {
			continue
		}
		if err := st.Types.RegisterMessage(dynamicpb.NewMessageType(md)); err != nil {
			return errors.Wrap(err, "register message")
		}
		numRegistered++
	}
	for i := 0; i < fd.Extensions().Len(); i++ {
		exd := fd.Extensions().Get(i)
		if _, ok := existingTypes[string(exd.FullName())]; ok {
			continue
		}
		if err := st.Types.RegisterExtension(dynamicpb.NewExtensionType(exd)); err != nil {
			return errors.Wrap(err, "register extension")
		}
		numRegistered++
	}
	if numRegistered > 0 {
		st.FileDescriptorSet.File = append(st.FileDescriptorSet.File, protodesc.ToFileDescriptorProto(fd))
	}
	return nil
}

// store implements an in-memory store for configurations. It stores
// the commit sha of the contents. It supports atomically updating
// all the configs with the contents of a new commit via the Update method.
type store struct {
	sync.RWMutex
	configs             map[string]map[string]configData
	commitSHA           string
	contentHash         string
	ownerName, repoName string
	registry            *SerializableTypes
}

type storedConfig struct {
	Config               *featurev1beta1.Feature
	CommitSHA, ConfigSHA string
}

var requiredFileDescriptors = []protoreflect.FileDescriptor{
	// Since we're internally converting starlark primitive types to google.protobuf.*Value,
	// we need to ensure that the wrapperspb types exist in the type registry in order for
	// json marshaling to work. However, we also need to ensure that type registration does
	// not panic in the event that the user also imported wrappers.proto
	wrapperspb.File_google_protobuf_wrappers_proto,
	// In the case of json feature flags, we will internally represent the json object as a structpb
	// proto object.
	structpb.File_google_protobuf_struct_proto,
}

func BuildDynamicTypeRegistryFromBufImage(image []byte) (*SerializableTypes, error) {
	fds := &descriptorpb.FileDescriptorSet{}

	if err := proto.Unmarshal(image, fds); err != nil {
		return nil, errors.Wrap(err, "proto unmarshal")
	}
	files, err := protodesc.NewFiles(fds)

	if err != nil {
		return nil, errors.Wrap(err, "protodesc.NewFiles")
	}
	return RegisterDynamicTypes(files)
}

func RegisterDynamicTypes(files *protoregistry.Files) (*SerializableTypes, error) {
	// Start from an empty type registry.
	ret := &SerializableTypes{
		Types:             &protoregistry.Types{},
		FileDescriptorSet: &descriptorpb.FileDescriptorSet{},
	}
	// First, add required types
	for _, fd := range requiredFileDescriptors {
		if err := ret.AddFileDescriptor(fd, false); err != nil {
			return nil, errors.Wrapf(err, "registering file descriptor: %s", string(fd.FullName()))
		}
	}
	// Import user defined types, ignoring types that already exist. All user-defined types
	// should be explicitly imported in their .proto files, which will end up
	// getting included since we're including imports in our file descriptor set.
	if files != nil {
		var rangeErr error
		files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
			if err := ret.AddFileDescriptor(fd, true); err != nil {
				rangeErr = errors.Wrap(err, "registering user-defined types")
				return false
			}
			return true
		})
		if rangeErr != nil {
			return nil, rangeErr
		}
	}
	return ret, nil
}

// Attempts to atomically update the store. This method will first hold a read-lock
// to check if the store should be updated. If not, we can exit early and avoid holding
// a write lock. To do this, we rely on two things - the git commit hash and a sha256
// hash of the configs.
func (s *store) update(contents *backendv1beta1.GetRepositoryContentsResponse) (bool, error) {
	if contents == nil {
		return false, errors.New("update with empty contents")
	}
	sortContents(contents)
	req := &updateRequest{contents: contents}
	// preemptive check to avoid holding the write lock at the
	// last minute if we can avoid it
	var shouldUpdate bool
	var err error
	s.RLock()
	shouldUpdate, err = s.shouldUpdateUnsafe(req)
	s.RUnlock()
	if err != nil {
		return false, err
	}
	if !shouldUpdate {
		return false, nil
	}
	bytes, err := proto.Marshal(contents.FileDescriptorSet)
	if err != nil {
		return false, err
	}
	registry, err := BuildDynamicTypeRegistryFromBufImage(bytes)
	if err != nil {
		return false, err
	}

	newConfigs := make(map[string]map[string]configData, len(contents.GetNamespaces()))
	for _, ns := range contents.GetNamespaces() {
		newConfigs[ns.GetName()] = make(map[string]configData)
		for _, cfg := range ns.GetFeatures() {
			newConfigs[ns.GetName()][cfg.GetName()] = configData{
				config:    cfg.GetFeature(),
				configSHA: cfg.GetSha(),
			}
		}
	}
	// perform the update
	s.Lock()
	defer s.Unlock()
	ok, err := s.shouldUpdateUnsafe(req)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}
	if err := req.calculateContentHash(); err != nil {
		return false, err
	}
	s.registry = registry
	s.configs = newConfigs
	s.commitSHA = contents.GetCommitSha()
	s.contentHash = *req.contentHash
	return true, nil
}

func (s *store) get(namespace, key string) (*storedConfig, error) {
	var data configData
	var ok bool
	var commitSHA string
	s.RLock()
	ns, nsok := s.configs[namespace]
	if nsok {
		data, ok = ns[key]
	}
	commitSHA = s.commitSHA
	s.RUnlock()
	if !ok {
		return nil, ErrConfigNotFound
	}
	return &storedConfig{
		Config:    data.config,
		CommitSHA: commitSHA,
		ConfigSHA: data.configSHA,
	}, nil
}

// Note: make sure this method call is wrapped in a mutex lock
// Returns the computed content hash (if it exists), allowing
// the caller to prevent rehashing.
func (s *store) shouldUpdateUnsafe(req *updateRequest) (bool, error) {
	if req.contents.GetCommitSha() != s.commitSHA {
		return true, nil
	}
	if err := req.calculateContentHash(); err != nil {
		return false, err
	}
	return *req.contentHash != s.contentHash, nil
}

func (s *store) getCommitSha() string {
	var ret string
	s.RLock()
	ret = s.commitSHA
	s.RUnlock()
	return ret
}

func (s *store) evaluate(key string, namespace string, lc map[string]interface{}) (*anypb.Any, *storedConfig, eval.ResultPath, error) {
	var typeURL string
	var targetFieldNumber uint64
	targetFieldNumber = 0
	for { //TODO - stop loops (especially for server side eval..
		cfg, err := s.get(namespace, key)
		if err != nil {
			return nil, nil, nil, err
		}

		referencedConfigToValueMap, err := s.maybeEvaluateReferencedConfigs(cfg, key, namespace, lc)
		if err != nil {
			return nil, nil, nil, err
		}

		evaluableConfig := eval.NewV1Beta3(cfg.Config, namespace, referencedConfigToValueMap)
		a, rp, err := evaluableConfig.Evaluate(lc)
		if err != nil {
			return nil, nil, nil, err
		}

		if a.TypeUrl == "type.googleapis.com/lekko.protobuf.ConfigCall" {
			b := a.Value
			for len(b) > 0 {
				fid, wireType, n := protowire.ConsumeTag(b)
				if n < 0 {
					return nil, nil, nil, protowire.ParseError(n)
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
					return nil, nil, nil, protowire.ParseError(n)
				}
				b = b[n:]
			}
		} else {
			if targetFieldNumber > 0 {
				b := a.Value
				for len(b) > 0 {
					fid, wireType, n := protowire.ConsumeTag(b)
					if n < 0 {
						return nil, nil, nil, protowire.ParseError(n)
					}
					b = b[n:]
					n = protowire.ConsumeFieldValue(fid, wireType, b)
					if n < 0 {
						return nil, nil, nil, protowire.ParseError(n)
					}
					if targetFieldNumber == uint64(fid) {
						var value []byte
						value = protowire.AppendTag(value, 1, protowire.BytesType)
						value = protowire.AppendBytes(value, b[:n])
						a = &anypb.Any{
							TypeUrl: typeURL,
							Value:   value,
						}
						break // TODO default values and do we want to support deeper nesting?
					} else {
						b = b[n:]
					}
				}
			}
			return a, cfg, rp, nil
		}
	}
}

func (s *store) evaluateType(
	key string, namespace string, lc map[string]interface{}, dest proto.Message) (*storedConfig, eval.ResultPath, error) {
	a, cfg, rp, err := s.evaluate(key, namespace, lc)
	if err != nil {
		return nil, nil, err
	}
	if err := a.UnmarshalTo(dest); err != nil {
		return nil, nil, errors.Wrapf(err, "invalid type, expecting %T", dest)
	}
	return cfg, rp, nil
}

func (s *store) maybeEvaluateReferencedConfigs(
	cfg *storedConfig, key string, namespace string, lc map[string]interface{}) (map[string]*structpb.Value, error) {
	referencedConfigToValueMap := make(map[string]*structpb.Value)
	referencedConfigKeys := s.getReferencedConfigKeys(cfg)
	if len(referencedConfigKeys) > 0 {
		for configKey := range referencedConfigKeys {
			// TODO: implement more sophisticated cycle detection in config dependency DAG
			if configKey == key {
				return nil, ErrCycleDependentConfig
			}

			// TODO: support non string type
			referencedDest := &wrapperspb.StringValue{}
			_, _, err := s.evaluateType(configKey, namespace, lc, referencedDest)
			if err != nil {
				return nil, err
			}
			referencedConfigToValueMap[configKey] = structpb.NewStringValue(referencedDest.Value)
		}
	}
	return referencedConfigToValueMap, nil
}

func (s *store) getReferencedConfigKeys(cfg *storedConfig) map[string]bool {
	referencedConfigKeys := make(map[string]bool)
	for _, constraint := range cfg.Config.GetTree().GetConstraints() {
		evaluateTo := constraint.GetRuleAstNew().GetCallExpression().GetEvaluateTo()
		if evaluateTo != nil {
			referencedConfigKeys[evaluateTo.ConfigName] = true
		}
	}
	return referencedConfigKeys
}

func (s *store) listContents() (*serverv1beta1.ListContentsResponse, error) {
	s.RLock()
	defer s.RUnlock()
	ret := &serverv1beta1.ListContentsResponse{
		RepoKey: &clientv1beta1.RepositoryKey{
			OwnerName: s.ownerName,
			RepoName:  s.repoName,
		},
		CommitSha:   s.commitSHA,
		ContentHash: s.contentHash,
	}
	for ns, configMap := range s.configs {
		namespace := &serverv1beta1.Namespace{Name: ns}
		for cfgName, cfg := range configMap {
			namespace.Configs = append(namespace.Configs, &serverv1beta1.Config{
				Name: cfgName,
				Sha:  cfg.configSHA,
			})
		}
		ret.Namespaces = append(ret.Namespaces, namespace)
	}
	return ret, nil
}

// wraps the contents in an object that caches its sha256 hash
// for easy comparison.
type updateRequest struct {
	contents    *backendv1beta1.GetRepositoryContentsResponse
	contentHash *string
}

func (ur *updateRequest) calculateContentHash() error {
	if ur.contentHash != nil {
		return nil
	}
	bytes, err := proto.MarshalOptions{Deterministic: true}.Marshal(ur.contents)
	if err != nil {
		return err
	}
	sha := hashContentsSHA256(bytes)
	ur.contentHash = &sha
	return nil
}

func hashContentsSHA256(bytes []byte) string {
	shaBytes := sha256.Sum256(bytes)
	return hex.EncodeToString(shaBytes[:])
}

func sortContents(contents *backendv1beta1.GetRepositoryContentsResponse) {
	nss := contents.GetNamespaces()
	sort.Slice(nss, func(i, j int) bool {
		return nss[i].GetName() < nss[j].GetName()
	})
	for _, ns := range nss {
		cfgs := ns.GetFeatures()
		sort.Slice(cfgs, func(i, j int) bool {
			return cfgs[i].GetName() < cfgs[j].GetName()
		})
	}
}
