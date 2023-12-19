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
// 		'cached' - periodically read configs from Lekko API, cache and evaluate in-memory
//		'git' - read configs from local git repo and watch for changes, cache and evaluate in-memory
//		'api' - send every evaluation to Lekko API
// LEKKO_REPO_PATH - required in git mode, path where the config repo was clonned
// LEKKO_API_KEY - Lekko API key
func NewClientFromEnv(ctx context.Context, ownerName, repoName string) (client.Client, error) {
	sdkMode := os.Getenv("LEKKO_SDK_MODE")
	var provider client.Provider
	var err error
	repoKey := &client.RepositoryKey{
		OwnerName: ownerName,
		RepoName:  repoName,
	}
	switch sdkMode {
	case "cached":
		apiKey := os.Getenv("LEKKO_API_KEY")
		provider, err = client.CachedAPIProvider(ctx, repoKey, client.WithAPIKey(apiKey))
	case "git":
		repoPath := os.Getenv("LEKKO_REPO_PATH")
		provider, err = client.CachedGitFsProvider(ctx, repoKey, repoPath)
	case "api":
		apiKey := os.Getenv("LEKKO_API_KEY")
		provider, err = client.ConnectAPIProvider(ctx, apiKey, repoKey)
	default:
		return nil, fmt.Errorf("unsupported Lekko SDK mode: %s", sdkMode)
	}
	if err != nil {
		return nil, err
	}
	client, _ := client.NewClient(provider)
	return client, nil
}
