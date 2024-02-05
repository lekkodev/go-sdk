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

package main

import (
	"context"
	"flag"
	"log"
	"time"

	client "github.com/lekkodev/go-sdk/client"
	"github.com/pkg/errors"
)

func main() {
	var key, mode, namespace, config, path, owner, repo, url, cfgType string
	var port int
	var sleep time.Duration
	var allowHTTP bool
	flag.StringVar(&key, "lekko-apikey", "", "API key for lekko given to your organization")
	flag.StringVar(&mode, "mode", "api", "Mode to start the sdk in (api, cached, git, gitlocal)")
	flag.StringVar(&namespace, "namespace", "default", "namespace to request the config from")
	flag.StringVar(&config, "config", "hello", "name of the config to request")
	flag.StringVar(&cfgType, "type", "string", "Type of config to read (string, int, float, bool)")
	flag.StringVar(&path, "path", "", "path to config repo if operating in git mode")
	flag.StringVar(&owner, "owner", "lekkodev", "name of the repository's github owner")
	flag.StringVar(&repo, "repo", "example", "name of the repository on github")
	flag.IntVar(&port, "port", 0, "port to serve web server on")
	flag.DurationVar(&sleep, "sleep", 0, "optional sleep duration to invoke web server")
	flag.StringVar(&url, "url", "", "optional URL to configure the provider")
	flag.BoolVar(&allowHTTP, "allow-http", false, "whether or not to allow http/2 requests")
	flag.Parse()

	var provider client.Provider
	var err error
	ctx := context.Background()
	provider, err = getProvider(ctx, key, mode, path, owner, repo, url, port, allowHTTP)
	if err != nil {
		log.Fatalf("error when starting in %s mode: %v\n", mode, err)
	}

	cl, closeF := client.NewClient(provider)
	defer func() {
		_ = closeF(context.Background())
	}()
	var result any
	switch cfgType {
	case "string":
		result, err = cl.GetString(ctx, namespace, config)
	case "int":
		result, err = cl.GetInt(ctx, namespace, config)
	case "float":
		result, err = cl.GetFloat(ctx, namespace, config)
	case "bool":
		result, err = cl.GetBool(ctx, namespace, config)
	default:
		log.Fatalf("unknown config type %s\n", cfgType)
	}
	if err != nil {
		log.Fatalf("error retrieving config: %v\n", err)
	}
	log.Printf("%s/%s [%T]: %v\n", namespace, config, result, result)
	time.Sleep(sleep)
}

func getProvider(
	ctx context.Context,
	key, mode, path, owner, repo, url string,
	port int, allowHTTP bool,
) (client.Provider, error) {
	rk := &client.RepositoryKey{
		OwnerName: owner,
		RepoName:  repo,
	}
	var opts []client.ProviderOption
	if port > 0 {
		opts = append(opts, client.WithServerOption(int32(port)))
	}
	if len(url) > 0 {
		opts = append(opts, client.WithURL(url))
	}
	if allowHTTP {
		opts = append(opts, client.WithAllowHTTP())
	}
	var provider client.Provider
	var err error
	switch mode {
	case "api":
		provider, err = client.ConnectAPIProvider(ctx, key, rk)
	case "cached":
		opts = append(opts, client.WithAPIKey(key))
		provider, err = client.CachedAPIProvider(ctx, rk, opts...)
	case "git":
		opts = append(opts, client.WithAPIKey(key))
		provider, err = client.CachedGitFsProvider(ctx, rk, path, opts...)
	case "gitlocal":
		provider, err = client.CachedGitFsProvider(ctx, rk, path, opts...)
	default:
		err = errors.Errorf("unknown mode %s", mode)
	}
	if err != nil {
		return nil, err
	}
	return provider, nil
}
