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

package lekko

import (
	"context"
	"fmt"
	"os"

	"github.com/lekkodev/go-sdk/client"
)

type Client client.Client

// Initialize Lekko SDK from environment variables:
// LEKKO_SDK_MODE - possible values:
//
//	'cached' - periodically read configs from Lekko API, cache and evaluate in-memory
//	'git' - read configs from local git repo and watch for changes, cache and evaluate in-memory
//	'api' - send every evaluation to Lekko API
//
// LEKKO_REPO_PATH - required in git mode, path where the config repo was clonned
// LEKKO_API_KEY - Lekko API key
//
// If LEKKO_SDK_MODE is not set to any known value
//	and LEKKO_API_KEY is set
//	and LEKKO_REPO_PATH is not set
// then 'cached' mode will be used.
func NewClientFromEnv(ctx context.Context, ownerName, repoName string) (client.Client, error) {
	sdkMode := os.Getenv("LEKKO_SDK_MODE")
	apiKey := os.Getenv("LEKKO_API_KEY")
	repoPath := os.Getenv("LEKKO_REPO_PATH")
	repoKey := &client.RepositoryKey{
		OwnerName: ownerName,
		RepoName:  repoName,
	}
	var provider client.Provider
	var err error
	switch sdkMode {
	case "cached":
		provider, err = client.CachedAPIProvider(ctx, repoKey, client.WithAPIKey(apiKey))
	case "git":
		provider, err = client.CachedGitFsProvider(ctx, repoKey, repoPath)
	case "api":
		provider, err = client.ConnectAPIProvider(ctx, apiKey, repoKey)
	default:
		// api key is set while repo path is not -> assuming 'cached' mode:
		if len(apiKey) > 0 && len(repoPath) == 0 {
			provider, err = client.CachedAPIProvider(ctx, repoKey, client.WithAPIKey(apiKey))
		} else {
			return nil, fmt.Errorf("unsupported Lekko SDK mode: %s", sdkMode)
		}
	}
	if err != nil {
		return nil, err
	}
	client, _ := client.NewClient(provider)
	return client, nil
}
