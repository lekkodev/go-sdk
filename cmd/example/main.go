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
	var key, mode, namespace, config string
	flag.StringVar(&key, "lekko-apikey", "", "API key for lekko given to your organization")
	flag.StringVar(&mode, "mode", "api", "Mode to start the sdk in (api, in-memory, local)")
	flag.StringVar(&namespace, "namespace", "default", "namespace to request the config from")
	flag.StringVar(&config, "config", "hello", "name of the config to request")
	flag.Parse()

	var provider client.Provider
	if key == "" {
		log.Fatal("Lekko API key not provided. Exiting...")
	}
	var err error
	ctx, cancelF := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelF()
	provider, err = getProvider(ctx, key, mode)
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

func getProvider(ctx context.Context, key, mode string) (client.Provider, error) {
	rk := &client.RepositoryKey{
		OwnerName: "lekkodev", // update me
		RepoName:  "example",  // update me
	}
	var provider client.Provider
	var err error
	switch mode {
	case "api":
		provider, err = client.ConnectAPIProvider(ctx, key, rk)
	case "in-memory":
		provider, err = client.BackendInMemoryProvider(ctx, &client.InMemoryProviderOptions{
			APIKey:         key,
			RepositoryKey:  *rk,
			UpdateInterval: 10 * time.Second,
		})
	default:
		err = errors.Errorf("unknown mode %s", mode)
	}
	if err != nil {
		return nil, err
	}
	return provider, nil
}
