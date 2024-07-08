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
	"log"
	"sync"
	"time"

	"buf.build/gen/go/lekkodev/cli/bufbuild/connect-go/lekko/backend/v1beta1/backendv1beta1connect"
	backendv1beta1 "buf.build/gen/go/lekkodev/cli/protocolbuffers/go/lekko/backend/v1beta1"
	"github.com/bufbuild/connect-go"
	"github.com/cenkalti/backoff/v4"
)

const (
	eventChannelSize = 10000
)

func newEventBatcher(
	ctx context.Context,
	distClient backendv1beta1connect.DistributionServiceClient,
	sessionKey, apiKey string,
	batchSize int,
) *eventBatcher {
	eb := &eventBatcher{
		distClient: distClient,
		events:     make(chan *backendv1beta1.FlagEvaluationEvent, eventChannelSize),
		sessionKey: sessionKey,
		apiKey:     apiKey,
		batchSize:  batchSize,
	}
	bgCtx, bgCancel := noInheritCancel(ctx)
	eb.cancel = bgCancel
	eb.wg.Add(1)
	go eb.loop(bgCtx, eb.events)
	return eb
}

type eventBatcher struct {
	distClient         backendv1beta1connect.DistributionServiceClient
	events             chan *backendv1beta1.FlagEvaluationEvent
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
	sessionKey, apiKey string
	batchSize          int
}

func (e *eventBatcher) track(event *backendv1beta1.FlagEvaluationEvent) {
	if event != nil {
		e.events <- event
	}
}

func (e *eventBatcher) loop(ctx context.Context, events chan *backendv1beta1.FlagEvaluationEvent) {
	defer e.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	batch := make([]*backendv1beta1.FlagEvaluationEvent, 0)

	flushBatch := func() {
		e.sendBatchWithBackoff(ctx, batch)
		batch = batch[:0] // clear the batch without deallocating memory
	}
	bufferEvent := func(event *backendv1beta1.FlagEvaluationEvent) {
		if event != nil {
			batch = append(batch, event)
		}
	}
	drainChan := func() {
		for event := range events {
			bufferEvent(event)
		}
	}

	for {
		select {
		case <-ctx.Done():
			drainChan()
			if len(batch) > 0 { // send the last batch with a fresh, capped ctx
				ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
				defer cancel()
				e.sendBatchWithBackoff(ctx, batch)
			}
			return
		case <-ticker.C:
			if len(batch) > 0 {
				flushBatch()
			}
		case event := <-events:
			bufferEvent(event)
			if len(batch) >= e.batchSize {
				flushBatch()
			}
		}
	}
}

func (e *eventBatcher) sendBatchWithBackoff(ctx context.Context, batch []*backendv1beta1.FlagEvaluationEvent) {
	req := connect.NewRequest(&backendv1beta1.SendFlagEvaluationMetricsRequest{
		Events:     batch,
		SessionKey: e.sessionKey,
	})
	setAPIKey(req, e.apiKey)
	op := func() error {
		_, err := e.distClient.SendFlagEvaluationMetrics(ctx, req)
		if err != nil {
			return err
		}
		return nil
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = 3 * time.Second
	if err := backoff.Retry(op, b); err != nil {
		log.Printf("error sending metrics batch: %v", err)
	}
}

func (e *eventBatcher) close() error {
	if e == nil {
		return nil
	}
	close(e.events)
	e.cancel()
	e.wg.Wait()
	return nil
}
