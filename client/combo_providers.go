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

package client

import (
	"fmt"
	"os"
)

// TODO: There should be a clear story of how to chain together
// providers, as well as passing out information on the flow of
// the decisions used to fallback, for tracability purposes. Right
// now we log, but we should return reasons instead.

// Use this to chain a file fallback to a production lekko provider.
// If an error is passed in, this will attempt to read the LEKKO_BOOTSTRAP
// environment variable and initialize a file provider from there.
// Example usage:
// client.WithFileFallback(client.KubeProvider("test-namespace"))
func WithFileFallback(p Provider, origErr error) (Provider, error) {
	if origErr == nil {
		return p, nil
	}
	bootstrapPath, ok := os.LookupEnv("LEKKO_BOOTSTRAP")
	if !ok {
		return nil, fmt.Errorf("error encountered when initializing provider: %v, and LEKKO_BOOSTRAP for fallback is unset", origErr)
	}
	fp, err := NewFileProvider(bootstrapPath)
	if err != nil {
		return nil, err
	}
	fmt.Printf("falling back to file provider, received an error when initializing initial provider: %v.\n", origErr)
	return fp, nil
}
