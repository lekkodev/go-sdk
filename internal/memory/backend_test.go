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
	"sync"
	"testing"
	"time"

	backendv1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/backend/v1beta1"
	featurev1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/feature/v1beta1"
	connect "github.com/bufbuild/connect-go"
	"github.com/cenkalti/backoff/v4"
	"github.com/lekkodev/go-sdk/testdata"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	testSessionKey = "test-session"
	testVersion    = "test-version"
)

func repositoryContents() *backendv1beta1.GetRepositoryContentsResponse {
	callBytes := make([]byte, 0)
	callBytes = protowire.AppendTag(callBytes, 1, protowire.BytesType)
	callBytes = protowire.AppendString(callBytes, "type.googleapis.com/google.protobuf.StringValue")
	callBytes = protowire.AppendTag(callBytes, 2, protowire.BytesType)
	callBytes = protowire.AppendString(callBytes, "ns-1")
	callBytes = protowire.AppendTag(callBytes, 3, protowire.BytesType)
	callBytes = protowire.AppendString(callBytes, "string")
	protoCallBytes := make([]byte, 0)
	protoCallBytes = protowire.AppendTag(protoCallBytes, 1, protowire.BytesType)
	protoCallBytes = protowire.AppendString(protoCallBytes, "type.googleapis.com/google.protobuf.Int32Value")
	protoCallBytes = protowire.AppendTag(protoCallBytes, 2, protowire.BytesType)
	protoCallBytes = protowire.AppendString(protoCallBytes, "ns-1")
	protoCallBytes = protowire.AppendTag(protoCallBytes, 3, protowire.BytesType)
	protoCallBytes = protowire.AppendString(protoCallBytes, "proto")
	protoCallBytes = protowire.AppendTag(protoCallBytes, 4, protowire.VarintType)
	protoCallBytes = protowire.AppendVarint(protoCallBytes, 1)
	return &backendv1beta1.GetRepositoryContentsResponse{
		CommitSha: "commitsha",
		Namespaces: []*backendv1beta1.Namespace{
			{
				Name: "ns-1",
				Features: []*backendv1beta1.Feature{
					testdata.Config(featurev1beta1.FeatureType_FEATURE_TYPE_BOOL, true),
					testdata.Config(featurev1beta1.FeatureType_FEATURE_TYPE_STRING, "foo"),
					testdata.Config(featurev1beta1.FeatureType_FEATURE_TYPE_FLOAT, float64(1.2)),
					testdata.Config(featurev1beta1.FeatureType_FEATURE_TYPE_INT, int64(42)),
					testdata.Config(featurev1beta1.FeatureType_FEATURE_TYPE_JSON, []any{1, 2.5, "bar"}),
					testdata.Config(featurev1beta1.FeatureType_FEATURE_TYPE_PROTO, wrapperspb.Int32(58)),
					{
						Name: "want-foo",
						Sha:  "want-foo",
						Feature: &featurev1beta1.Feature{
							Key: "want-foo",
							Tree: &featurev1beta1.Tree{
								Default: &anypb.Any{
									TypeUrl: "type.googleapis.com/lekko.protobuf.ConfigCall",
									Value:   callBytes,
								},
							},
							Type: featurev1beta1.FeatureType_FEATURE_TYPE_STRING,
						},
					},
					{
						Name: "want-proto",
						Sha:  "want-proto",
						Feature: &featurev1beta1.Feature{
							Key: "want-proto",
							Tree: &featurev1beta1.Tree{
								Default: &anypb.Any{
									TypeUrl: "type.googleapis.com/lekko.protobuf.ConfigCall",
									Value:   protoCallBytes,
								},
							},
							Type: featurev1beta1.FeatureType_FEATURE_TYPE_INT,
						},
					},
				},
			},
		},
	}
}

func TestBackendStore(t *testing.T) {
	tds := &testDistService{
		sessionKey: testSessionKey,
		contents:   repositoryContents(),
	}
	ctx := context.Background()
	b, err := newBackendStore(ctx, "apikey", "owner", "repo", 5*time.Second, tds, 6, 0, testVersion)
	require.NoError(t, err, "no error during register and init")

	assert.Equal(t, testVersion, tds.registrationVersion)

	bv := &wrapperspb.BoolValue{}
	require.NoError(t, b.Evaluate("bool", "ns-1", nil, bv))
	assert.Equal(t, true, bv.Value)
	sv := &wrapperspb.StringValue{}
	require.NoError(t, b.Evaluate("string", "ns-1", nil, sv))
	assert.Equal(t, "foo", sv.Value)
	require.NoError(t, b.Evaluate("want-foo", "ns-1", nil, sv))
	assert.Equal(t, "foo", sv.Value)
	iv := &wrapperspb.Int64Value{}
	require.NoError(t, b.Evaluate("int", "ns-1", nil, iv))
	assert.Equal(t, int64(42), iv.Value)
	fv := &wrapperspb.DoubleValue{}
	require.NoError(t, b.Evaluate("float", "ns-1", nil, fv))
	assert.Equal(t, float64(1.2), fv.Value)
	vv := &structpb.Value{}
	require.NoError(t, b.Evaluate("json", "ns-1", nil, vv))
	expectedValue, err := structpb.NewValue([]any{1, 2.5, "bar"})
	require.NoError(t, err)
	assert.True(t, proto.Equal(expectedValue, vv))
	pv := &wrapperspb.Int32Value{}
	require.NoError(t, b.Evaluate("proto", "ns-1", nil, pv))
	assert.Equal(t, int32(58), pv.Value)
	// Google WKT are not actually proto messages, so proto isn't really useful for testing this
	//require.NoError(t, b.Evaluate("want-proto", "ns-1", nil, pv))
	//assert.Equal(t, int32(58), pv.Value)

	err = b.Close(ctx)
	require.NoError(t, err, "no error during close")
	events := tds.getEventsReceived()
	require.Equal(t, 7, len(events), "expecting 7 events, got %d: %v", len(events), events)
}

func TestBackendStoreRegisterError(t *testing.T) {
	tds := &testDistService{
		sessionKey:  testSessionKey,
		contents:    repositoryContents(),
		registerErr: backoff.Permanent(errors.New("registration failed")),
	}
	ctx := context.Background()
	_, err := newBackendStore(ctx, "apikey", "owner", "repo", 5*time.Second, tds, eventsBatchSize, 0, testVersion)
	require.Error(t, err)
}

func TestBackendStoreDeregisterError(t *testing.T) {
	tds := &testDistService{
		sessionKey:    testSessionKey,
		contents:      repositoryContents(),
		deregisterErr: errors.New("deregistration failed"),
	}
	ctx := context.Background()
	b, err := newBackendStore(ctx, "apikey", "owner", "repo", 5*time.Second, tds, eventsBatchSize, 0, testVersion)
	require.NoError(t, err)

	require.Error(t, b.Close(ctx))
}

func TestBackendStoreGetContentsError(t *testing.T) {
	tds := &testDistService{
		sessionKey:     testSessionKey,
		contents:       repositoryContents(),
		getContentsErr: backoff.Permanent(errors.New("get contents failed")),
	}
	ctx := context.Background()
	_, err := newBackendStore(ctx, "apikey", "owner", "repo", 5*time.Second, tds, eventsBatchSize, 0, testVersion)
	require.Error(t, err)
}

type testDistService struct {
	sync.RWMutex
	sessionKey                                                string
	registerErr, deregisterErr, getContentsErr, getVersionErr error
	contents                                                  *backendv1beta1.GetRepositoryContentsResponse
	events                                                    []*backendv1beta1.FlagEvaluationEvent
	registrationVersion                                       string
}

func (tds *testDistService) DeregisterClient(context.Context, *connect.Request[backendv1beta1.DeregisterClientRequest]) (*connect.Response[backendv1beta1.DeregisterClientResponse], error) {
	return connect.NewResponse(&backendv1beta1.DeregisterClientResponse{}), tds.deregisterErr
}

func (*testDistService) GetDeveloperAccessToken(context.Context, *connect.Request[backendv1beta1.GetDeveloperAccessTokenRequest]) (*connect.Response[backendv1beta1.GetDeveloperAccessTokenResponse], error) {
	return connect.NewResponse(&backendv1beta1.GetDeveloperAccessTokenResponse{}), nil
}

func (tds *testDistService) GetRepositoryContents(context.Context, *connect.Request[backendv1beta1.GetRepositoryContentsRequest]) (*connect.Response[backendv1beta1.GetRepositoryContentsResponse], error) {
	return connect.NewResponse(tds.contents), tds.getContentsErr
}

func (tds *testDistService) GetRepositoryVersion(context.Context, *connect.Request[backendv1beta1.GetRepositoryVersionRequest]) (*connect.Response[backendv1beta1.GetRepositoryVersionResponse], error) {
	return connect.NewResponse(&backendv1beta1.GetRepositoryVersionResponse{
		CommitSha: tds.contents.GetCommitSha(),
	}), tds.getVersionErr
}

func (tds *testDistService) RegisterClient(ctx context.Context, req *connect.Request[backendv1beta1.RegisterClientRequest]) (*connect.Response[backendv1beta1.RegisterClientResponse], error) {
	tds.registrationVersion = req.Msg.GetSidecarVersion()
	return connect.NewResponse(&backendv1beta1.RegisterClientResponse{
		SessionKey: tds.sessionKey,
	}), tds.registerErr
}

func (tds *testDistService) SendFlagEvaluationMetrics(ctx context.Context, req *connect.Request[backendv1beta1.SendFlagEvaluationMetricsRequest]) (*connect.Response[backendv1beta1.SendFlagEvaluationMetricsResponse], error) {
	tds.Lock()
	defer tds.Unlock()
	tds.events = append(tds.events, req.Msg.GetEvents()...)
	return connect.NewResponse(&backendv1beta1.SendFlagEvaluationMetricsResponse{}), nil
}

func (tds *testDistService) getEventsReceived() []*backendv1beta1.FlagEvaluationEvent {
	tds.RLock()
	defer tds.RUnlock()
	return tds.events
}

func TestToContextKeysProto(t *testing.T) {
	lekkoContext := make(map[string]interface{})
	lekkoContext["bool"] = true
	lekkoContext["int"] = 12
	lekkoContext["int32"] = int32(12)
	lekkoContext["uint32"] = uint32(12)
	lekkoContext["float"] = 32.5
	lekkoContext["float64"] = float64(32.5)
	lekkoContext["string"] = "foo"

	result := toContextKeysProto(lekkoContext)
	assert.Contains(t, result, &backendv1beta1.ContextKey{Key: "bool", Type: "bool"})
	assert.Contains(t, result, &backendv1beta1.ContextKey{Key: "int", Type: "int"})
	assert.Contains(t, result, &backendv1beta1.ContextKey{Key: "int32", Type: "int"})
	assert.Contains(t, result, &backendv1beta1.ContextKey{Key: "uint32", Type: "int"})
	assert.Contains(t, result, &backendv1beta1.ContextKey{Key: "float", Type: "float"})
	assert.Contains(t, result, &backendv1beta1.ContextKey{Key: "float64", Type: "float"})
	assert.Contains(t, result, &backendv1beta1.ContextKey{Key: "string", Type: "string"})
}
