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
		configs: make(map[configKey]configData),
	}
}

type configKey struct {
	namespace, config string
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
	configs   map[configKey]configData
	commitSHA string
}

type storedConfig struct {
	Config               *featurev1beta1.Feature
	CommitSHA, ConfigSHA string
}

func (s *store) update(contents *backendv1beta1.GetRepositoryContentsResponse) bool {
	newConfigs := make(map[configKey]configData)
	for _, ns := range contents.GetNamespaces() {
		for _, cfg := range ns.GetFeatures() {
			newConfigs[configKey{ns.GetName(), cfg.GetName()}] = configData{
				config:    cfg.GetFeature(),
				configSHA: cfg.GetSha(),
			}
		}
	}
	commitSHA := contents.GetCommitSha()
	// preemptive check to avoid holding the write lock at the
	// last minute if we can avoid it
	if !s.shouldUpdate(commitSHA) {
		return false
	}
	var updated bool
	s.Lock()
	if commitSHA != s.commitSHA {
		s.configs = newConfigs
		s.commitSHA = commitSHA
		updated = true
	}
	s.Unlock()
	return updated
}

func (s *store) get(namespace, key string) (*storedConfig, error) {
	ck := configKey{namespace, key}
	var data configData
	var ok bool
	var commitSHA string
	s.RLock()
	data, ok = s.configs[ck]
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

func (s *store) shouldUpdate(newCommitSHA string) bool {
	var shouldUpdate bool
	s.RLock()
	shouldUpdate = newCommitSHA != s.commitSHA
	s.RUnlock()
	return shouldUpdate
}

func (s *store) evaluateType(key string, namespace string, lc map[string]interface{}, dest proto.Message) error {
	cfg, err := s.get(namespace, key)
	if err != nil {
		return err
	}
	evaluableConfig := eval.NewV1Beta3(cfg.Config, namespace)
	a, _, err := evaluableConfig.Evaluate(lc)
	if err != nil {
		return err
	}
	if err := a.UnmarshalTo(dest); err != nil {
		return errors.Wrapf(err, "invalid type, expecting %T", dest)
	}
	return nil
}
