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
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"buf.build/gen/go/lekkodev/sdk/connectrpc/go/lekko/server/v1beta1/serverv1beta1connect"
	serverv1beta1 "buf.build/gen/go/lekkodev/sdk/protocolbuffers/go/lekko/server/v1beta1"
	"connectrpc.com/connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func newSDKServer(port int32, store contentStore) *sdkServer {
	if port <= 0 {
		return nil
	}
	handler := &sdkServerHandler{store: store}

	addr := fmt.Sprintf("0.0.0.0:%d", port)

	mux := http.NewServeMux()
	mux.Handle(serverv1beta1connect.NewSDKServiceHandler(handler))
	srv := &http.Server{
		Addr:              addr,
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 3 * time.Second,
	}

	ss := &sdkServer{
		Server: srv,
	}
	ss.wg.Add(1)
	go ss.serve()
	return ss
}

type contentStore interface {
	listContents() (*serverv1beta1.ListContentsResponse, error)
}

type sdkServer struct {
	*http.Server
	wg sync.WaitGroup
}

func (ss *sdkServer) serve() {
	defer ss.wg.Done()
	if err := ss.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("sdk server error: %v", err)
	}
}

func (ss *sdkServer) close(ctx context.Context) error {
	if ss == nil {
		return nil
	}
	if err := ss.Shutdown(ctx); err != nil {
		return err
	}
	ss.wg.Wait()
	return nil
}

type sdkServerHandler struct {
	serverv1beta1connect.UnimplementedSDKServiceHandler
	store contentStore
}

func (ssh *sdkServerHandler) ListContents(
	ctx context.Context,
	req *connect.Request[serverv1beta1.ListContentsRequest],
) (*connect.Response[serverv1beta1.ListContentsResponse], error) {
	resp, err := ssh.store.listContents()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	sortContentsResp(resp)
	return connect.NewResponse(resp), nil
}

func sortContentsResp(contents *serverv1beta1.ListContentsResponse) {
	nss := contents.GetNamespaces()
	sort.Slice(nss, func(i, j int) bool {
		return nss[i].GetName() < nss[j].GetName()
	})
	for _, ns := range nss {
		cfgs := ns.GetConfigs()
		sort.Slice(cfgs, func(i, j int) bool {
			return cfgs[i].GetName() < cfgs[j].GetName()
		})
	}
}
