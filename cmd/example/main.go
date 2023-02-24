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
)

func main() {
	key := flag.String("lekko-apikey", "", "API key for lekko given to your organization")
	path := flag.String("lekko-static-path", "", "Optional bootstrap, if provided, will operate in static mode")
	flag.Parse()

	var provider client.Provider
	if path != nil {
		var err error
		provider, err = client.NewStaticProvider(*path)
		if err != nil {
			log.Fatalf("error when starting in static mode: %v\n", err)
		}
	} else if key == nil {
		log.Fatal("Lekko API key not provided. Exiting...")
	} else {
		var err error
		ctx, cancelF := context.WithTimeout(context.Background(), time.Second)
		defer cancelF()
		provider, err = client.ConnectAPIProvider(ctx, *key, &client.RepositoryKey{
			OwnerName: "lekkodev",
			RepoName:  "template",
		})
		if err != nil {
			log.Fatalf("error when starting in API mode: %v\n", err) // nolint
		}
	}
	cl, closeF := client.NewClient("default", provider)
	defer func() {
		_ = closeF(context.Background())
	}()
	flag, err := cl.GetBool(context.TODO(), "example")
	log.Printf("Retrieving feature flag: %v (err=%v)\n", flag, err)
}
