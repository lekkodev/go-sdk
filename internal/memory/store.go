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
	"sync"

	backendv1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/backend/v1beta1"
	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
	"github.com/lekkodev/go-sdk/pkg/eval"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

var (
	ErrConfigNotFound error = fmt.Errorf("config not found")
)

func newStore() *store {
	return &store{
		configs: make(map[string]map[string]configData),
	}
}

type configData struct {
	config    *featurev1beta1.Feature
	configSHA string
}

// store implements an in-memory store for configurations. It stores
// the commit sha of the contents. It supports atomically updating
// all the configs with the contents of a new commit via the Update method.
type store struct {
	sync.RWMutex
	configs     map[string]map[string]configData
	commitSHA   string
	contentHash string
}

type storedConfig struct {
	Config               *featurev1beta1.Feature
	CommitSHA, ConfigSHA string
}

// Attempts to atomically update the store. This method will first hold a read-lock
// to check if the store should be updated. If not, we can exit early and avoid holding
// a write lock. To do this, we rely on two things - the git commit hash and a sha256
// hash of the configs.
func (s *store) update(contents *backendv1beta1.GetRepositoryContentsResponse) (bool, error) {
	if contents == nil {
		return false, errors.New("update with empty contents")
	}
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

func (s *store) evaluateType(key string, namespace string, lc map[string]interface{}, dest proto.Message) (*storedConfig, eval.ResultPath, error) {
	cfg, err := s.get(namespace, key)
	if err != nil {
		return nil, nil, err
	}
	evaluableConfig := eval.NewV1Beta3(cfg.Config, namespace)
	a, rp, err := evaluableConfig.Evaluate(lc)
	if err != nil {
		return nil, nil, err
	}
	if err := a.UnmarshalTo(dest); err != nil {
		return nil, nil, errors.Wrapf(err, "invalid type, expecting %T", dest)
	}
	return cfg, rp, nil
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
