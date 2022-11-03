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

	client "github.com/lekkodev/go-sdk/client"
)

func main() {
	key := flag.String("lekko-apikey", "", "API key for lekko given to your organization")
	flag.Parse()

	if key == nil {
		log.Fatal("Lekko API key not provided. Exiting...") // nolint
	}

	cl := client.NewClient("default", client.NewAPIProvider(client.LekkoURL, *key, &client.RepositoryKey{
		OwnerName: "lekkodev",
		RepoName:  "template",
	}))
	flag, err := cl.GetBool(context.TODO(), "example")
	log.Printf("Retrieving feature flag: %v (err=%v)\n", flag, err) // nolint
}
