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
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	backendv1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/backend/v1beta1"
	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
	"gopkg.in/yaml.v3"
)

const defaultRootConfigRepoMetadataFileName = "lekko.root.yaml"

func newRepository(storer storage.Storer, fs billy.Filesystem) (*repository, error) {
	repo, err := git.Open(storer, fs)
	if err != nil {
		return nil, errors.Wrapf(err, "could not open git repo")
	}
	return &repository{
		Repository: repo,
		fs:         fs,
	}, err
}

type repository struct {
	*git.Repository
	fs billy.Filesystem
}

func (r *repository) getContents() (*backendv1beta1.GetRepositoryContentsResponse, error) {
	ret := &backendv1beta1.GetRepositoryContentsResponse{}
	sha, err := r.getCommitSha()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get commit sha")
	}
	ret.CommitSha = sha
	namespaces, err := r.getNamespaces()
	if err != nil {
		return nil, err
	}
	ret.Namespaces = namespaces
	return ret, nil
}

func (r *repository) getCommitSha() (string, error) {
	ref, err := r.Head()
	if err != nil {
		return "", err
	}
	baseSHA := ref.Hash().String()
	wt, err := r.Worktree()
	if err != nil {
		return "", err
	}
	status, err := wt.Status()
	if err != nil {
		return "", err
	}
	var suffix string
	if !status.IsClean() {
		suffix = "-dirty"
	}
	return fmt.Sprintf("%s%s", baseSHA, suffix), nil
}

type RootConfigRepoMetadata struct {
	// This version refers to the version of the metadata.
	Version          string   `json:"version,omitempty" yaml:"version,omitempty"`
	Namespaces       []string `json:"namespaces,omitempty" yaml:"namespaces,omitempty"`
	ProtoDirectory   string   `json:"protoDir,omitempty" yaml:"protoDir,omitempty"`
	UseExternalTypes bool     `json:"externalTypes,omitempty" yaml:"externalTypes,omitempty"`
}

func (r *repository) getNamespaces() ([]*backendv1beta1.Namespace, error) {
	md, err := r.readMetadata()
	if err != nil {
		return nil, err
	}
	var ret []*backendv1beta1.Namespace
	for _, ns := range md.Namespaces {
		configs, err := r.getConfigs(ns)
		if err != nil {
			return nil, errors.Wrapf(err, "get configs for ns '%s'", ns)
		}
		ret = append(ret, &backendv1beta1.Namespace{
			Name:     ns,
			Features: configs,
		})
	}
	return ret, nil
}

func (r *repository) getConfigs(ns string) ([]*backendv1beta1.Feature, error) {
	nsPath := ns
	dirEntries, err := r.fs.ReadDir(nsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read '%s'", nsPath)
	}
	var configNames []string
	for _, entry := range dirEntries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, ".star") {
			configNames = append(configNames, strings.TrimRight(name, ".star"))
		}
	}
	var configs []*backendv1beta1.Feature
	for _, configName := range configNames {
		genPath := filepath.Join(ns, "gen", "proto", fmt.Sprintf("%s.proto.bin", configName))
		fileContents, err := r.readFile(genPath)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, errors.Wrapf(err, "read file at path '%s'", genPath)
		}
		configProto := featurev1beta1.Feature{}
		if err := proto.Unmarshal(fileContents, &configProto); err != nil {
			return nil, errors.Wrapf(err, "malformed proto file at '%s'", genPath)
		}
		configs = append(configs, &backendv1beta1.Feature{
			Name:    configName,
			Sha:     plumbing.ComputeHash(plumbing.BlobObject, fileContents).String(),
			Feature: &configProto,
		})
	}
	return configs, nil
}

func (r *repository) readMetadata() (*RootConfigRepoMetadata, error) {
	mdPath := defaultRootConfigRepoMetadataFileName
	contents, err := r.readFile(mdPath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not read md file at '%s'", mdPath)
	}
	if len(contents) == 0 {
		return nil, errors.Errorf("md file empty at path '%s'", mdPath)
	}
	var ret RootConfigRepoMetadata
	if err := yaml.NewDecoder(bytes.NewReader(contents)).Decode(&ret); err != nil {
		return nil, err
	}
	return &ret, nil
}

func (r *repository) readFile(path string) ([]byte, error) {
	file, err := r.fs.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return io.ReadAll(file)
}
