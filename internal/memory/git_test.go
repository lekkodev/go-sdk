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
	"io"
	"testing"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
	gitmemory "github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestGitStore(t *testing.T) {
	ctx := context.Background()
	commitHash, storer, fs := setupFS(t)

	gs, err := newGitStore(ctx, "", "lekkodev", "testrepo", storer, fs, nil, 10, false, 0, testVersion)
	require.NoError(t, err)
	assert.NotNil(t, gs)

	bv := wrapperspb.BoolValue{}
	meta, err := gs.Evaluate("example", "default", nil, &bv)
	require.NoError(t, err)
	assert.True(t, bv.Value)
	assert.Equal(t, meta.LastUpdateCommitSHA, commitHash.String())

	sv := wrapperspb.StringValue{}
	meta, err = gs.Evaluate("tiers", "test-namespace", nil, &sv)
	require.NoError(t, err)
	assert.Equal(t, sv.Value, "foo")
	assert.Equal(t, meta.LastUpdateCommitSHA, commitHash.String())

	_, err = gs.Evaluate("not-a-key", "not-a-ns", nil, &bv)
	require.Error(t, err)

	require.NoError(t, gs.Close(ctx))
}

func setupFS(t *testing.T) (plumbing.Hash, storage.Storer, billy.Filesystem) {
	storer := gitmemory.NewStorage()
	fs := memfs.New()
	repo, err := git.Init(storer, fs)
	require.NoError(t, err)

	testdataFS := osfs.New("../../testdata/testrepo")
	require.NoError(t, copyDir(testdataFS, fs))

	wt, err := repo.Worktree()
	require.NoError(t, err)
	require.NoError(t, wt.AddGlob("."))
	commitHash, err := wt.Commit("initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "test",
			Email: "ci@lekko.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	return commitHash, storer, fs
}

func copyDir(src, dst billy.Filesystem) error {
	fis, err := src.ReadDir(".")
	if err != nil {
		return err
	}
	for _, fi := range fis {
		if fi.IsDir() {
			if err := dst.MkdirAll(fi.Name(), 0644); err != nil {
				return err
			}
			nextSrc, err := src.Chroot(fi.Name())
			if err != nil {
				return err
			}
			nextDst, err := dst.Chroot(fi.Name())
			if err != nil {
				return err
			}
			if err := copyDir(nextSrc, nextDst); err != nil {
				return err
			}
		} else {
			if err := copyFile(src, dst, fi.Name()); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyFile(src, dst billy.Filesystem, filename string) error {
	srcFile, err := src.Open(filename)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	contents, err := io.ReadAll(srcFile)
	if err != nil {
		return err
	}
	newFile, err := dst.Create(filename)
	if err != nil {
		return err
	}
	defer newFile.Close()
	_, err = newFile.Write(contents)
	if err != nil {
		return err
	}
	return nil
}
