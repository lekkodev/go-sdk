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
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/lekkodev/go-sdk/harness"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 0, "port to serve harness server on")
	flag.Parse()
	ctx := context.Background()
	fmt.Printf("Serving harness server on port %d\n", port)
	server := harness.NewHarnessServer(int32(port))

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGTERM)

		// block until we receive the signal
		sig := <-sigChan
		fmt.Printf("received signal: %v\n", sig)

		if err := server.Close(ctx); err != nil {
			fmt.Printf("failed to close harness server cleanly")
		}
	}()

	wg.Wait()
}
