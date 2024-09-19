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

package oteltest

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

type otelHelper struct {
	traceExporter *tracetest.InMemoryExporter
	tracer        trace.Tracer
	span          trace.Span
}

func InitOtelAndStartSpan() (context.Context, *otelHelper) {
	me := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(me))
	tracer := tp.Tracer("")
	ctx, span := tracer.Start(context.Background(), "")
	return ctx, &otelHelper{
		traceExporter: me,
		tracer:        tracer,
		span:          span,
	}
}

func (o *otelHelper) EndSpanAndGetConfigEvent(t *testing.T) sdktrace.Event {
	o.span.End()
	spans := o.traceExporter.GetSpans().Snapshots()
	var event sdktrace.Event
	for _, s := range spans {
		if s.SpanContext().Equal(o.span.SpanContext()) {
			assert.Equal(t, 1, len(spans[0].Events()))
			event = spans[0].Events()[0]
		}
	}
	assert.NotNil(t, event)
	assert.True(t, strings.HasPrefix(event.Name, "lekko."))
	return event
}
