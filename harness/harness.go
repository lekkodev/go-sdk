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

package harness

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
	"buf.build/gen/go/lekkodev/conformance/bufbuild/connect-go/lekko/conformance/v1beta1/conformancev1beta1connect"
	conformancev1beta1 "buf.build/gen/go/lekkodev/conformance/protocolbuffers/go/lekko/conformance/v1beta1"
	"github.com/bufbuild/connect-go"
	"github.com/lekkodev/go-sdk/client"
	"github.com/lekkodev/go-sdk/pkg/eval"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func NewHarnessServer(port int32) *harnessServer {
	if port <= 0 {
		return nil
	}
	handler := &harnessServerHandler{}

	addr := fmt.Sprintf("0.0.0.0:%d", port)

	mux := http.NewServeMux()
	mux.Handle(conformancev1beta1connect.NewHarnessServiceHandler(handler))
	srv := &http.Server{
		Addr:              addr,
		Handler:           h2c.NewHandler(mux, &http2.Server{}),
		ReadHeaderTimeout: 3 * time.Second,
	}

	ss := &harnessServer{
		Server: srv,
	}
	ss.wg.Add(1)
	go ss.serve()
	return ss
}

type harnessServer struct {
	*http.Server
	wg sync.WaitGroup
}

func (ss *harnessServer) serve() {
	defer ss.wg.Done()
	if err := ss.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("sdk server error: %v", err)
	}
}

func (ss *harnessServer) Close(ctx context.Context) error {
	if ss == nil {
		return nil
	}
	if err := ss.Shutdown(ctx); err != nil {
		return err
	}
	ss.wg.Wait()
	return nil
}

type harnessServerHandler struct {
	conformancev1beta1connect.UnimplementedHarnessServiceHandler
}

func (hsh *harnessServerHandler) Evaluate(
	ctx context.Context,
	req *connect.Request[conformancev1beta1.EvaluateRequest],
) (*connect.Response[conformancev1beta1.EvaluateResponse], error) {
	if req.Msg.Config == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no config provided"))
	}
	if len(req.Msg.Namespace) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no namespace provided"))
	}
	lc, err := client.FromProto(req.Msg.GetContext())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	evaluableConfig := eval.NewV1Beta3(req.Msg.GetConfig(), req.Msg.GetNamespace())
	a, rp, err := evaluableConfig.Evaluate(lc)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&conformancev1beta1.EvaluateResponse{
		Result: &featurev1beta1.Any{
			TypeUrl: a.GetTypeUrl(),
			Value:   a.GetValue(),
		},
		Path: toPathProto(rp),
	}), nil
}

func toPathProto(rp []int) []int32 {
	ret := make([]int32, len(rp))
	for i := 0; i < len(rp); i++ {
		ret[i] = int32(rp[i])
	}
	return ret
}
