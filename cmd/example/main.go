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
	var key, mode, namespace, config, path, owner, repo string
	var port int
	flag.StringVar(&key, "lekko-apikey", "", "API key for lekko given to your organization")
	flag.StringVar(&mode, "mode", "api", "Mode to start the sdk in (api, cached, git, gitlocal)")
	flag.StringVar(&namespace, "namespace", "default", "namespace to request the config from")
	flag.StringVar(&config, "config", "hello", "name of the config to request")
	flag.StringVar(&path, "path", "", "path to config repo if operating in git mode")
	flag.StringVar(&owner, "owner", "lekkodev", "name of the repository's github owner")
	flag.StringVar(&repo, "repo", "example", "name of the repository on github")
	flag.IntVar(&port, "port", 0, "port to serve web server on")
	flag.Parse()

	var provider client.Provider
	if mode != "gitlocal" && key == "" {
		log.Fatal("Lekko API key not provided. Exiting...")
	}
	var err error
	ctx, cancelF := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelF()
	provider, err = getProvider(ctx, key, mode, path, owner, repo, port)
	if err != nil {
		log.Fatalf("error when starting in %s mode: %v\n", mode, err)
	}

	cl, closeF := client.NewClient(namespace, provider)
	defer func() {
		_ = closeF(context.Background())
	}()
	result, err := cl.GetString(ctx, config)
	if err != nil {
		log.Fatalf("error retrieving config: %v\n", err)
	}
	log.Printf("%s/%s [%T]: %v\n", namespace, config, result, result)
}

func getProvider(ctx context.Context, key, mode, path, owner, repo string, port int) (client.Provider, error) {
	rk := &client.RepositoryKey{
		OwnerName: owner,
		RepoName:  repo,
	}
	var so *client.ServerOptions
	if port > 0 {
		so = &client.ServerOptions{
			Port: int32(port),
		}
	}
	var provider client.Provider
	var err error
	switch mode {
	case "api":
		provider, err = client.ConnectAPIProvider(ctx, key, rk)
	case "cached":
		provider, err = client.CachedAPIProvider(ctx, &client.ConnectionOptions{
			APIKey: key,
			URL:    "",
		}, *rk, 10*time.Second, so)
	case "git":
		provider, err = client.CachedGitFsProvider(ctx, path, &client.ConnectionOptions{
			APIKey: key,
			URL:    "",
		}, *rk, so)
	case "gitlocal":
		provider, err = client.CachedGitFsProvider(ctx, path, nil, *rk, so)
	default:
		err = errors.Errorf("unknown mode %s", mode)
	}
	if err != nil {
		return nil, err
	}
	return provider, nil
}
