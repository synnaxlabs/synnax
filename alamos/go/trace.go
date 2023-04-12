// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos

import (
	"context"
	"fmt"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
	"runtime/pprof"
)

// Span is a span in a trace.
type Span interface {
	// Error records the given error as an error on the span, setting the span's status
	// to Error if the error is non-nil. If exclude is provided, the status will only be
	// set if the error is not one of the excluded errors.
	Error(err error, exclude ...error) error
	// Status sets the span's status.
	Status(status Status)
	// EndWith combines Error and End, and also returns the given err unmodified.
	EndWith(err error, exclude ...error) error
	// End ends the span.
	End()
}

// TracingConfig is the configuration for a Tracer.
type TracingConfig struct {
	// OtelProvider sets the open telemetry tracing provider used to create spans.
	// [REQUIRED]
	OtelProvider oteltrace.TracerProvider
	// OtelPropagator sets the open telemetry propagator used to propagate spans across
	// process boundaries.
	// [REQUIRED]
	OtelPropagator propagation.TextMapPropagator
	// Filter is a function that is called to determine if a span should be created for
	// the given key and environment. If the filter returns false, the span will not be created.
	Filter EnvironmentFilter
}

var _ config.Config[TracingConfig] = (*TracingConfig)(nil)

// Validate implements config.Config.
func (c TracingConfig) Validate() error {
	v := validate.New("alamos.TracingConfig")
	validate.NotNil(v, "OtelProvider", c.OtelProvider)
	validate.NotNil(v, "OtelPropagator", c.OtelPropagator)
	return v.Error()
}

// Override implements config.Config.
func (c TracingConfig) Override(other TracingConfig) TracingConfig {
	c.OtelProvider = override.Nil(c.OtelProvider, other.OtelProvider)
	c.OtelPropagator = override.Nil(c.OtelPropagator, other.OtelPropagator)
	return c
}

// Tracer provides tracing functionality, and is one of the core components of
// Instrumentation. Tracer's should not be used on their own, and instead should
// be used as part of Instrumentation. To creat a Tracer, use NewTracer and pass
// it in a call to alamos.New using the WithTracer option.
type Tracer struct {
	meta   InstrumentationMeta
	config TracingConfig
}

// NewTracer initializes a new tracer using the given configuration. If no configuration
// is provided, NewTracer will return a validation error. If you want a no-op tracer,
// simply use a nil pointer.
func NewTracer(configs ...TracingConfig) (*Tracer, error) {
	cfg, err := config.New(TracingConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	return &Tracer{config: cfg}, nil
}

// Debug starts a span at the debug level with the given key. If the context is already
// wrapped in a span, the span will be a child of the existing span.
func (t *Tracer) Debug(ctx context.Context, key string) (context.Context, Span) {
	return t.Trace(ctx, key, Debug)
}

// Prod starts a span at the production level. If the context is already wrapped in a
// span, the span will be a child of the existing span.
func (t *Tracer) Prod(ctx context.Context, key string) (context.Context, Span) {
	return t.Trace(ctx, key, Prod)
}

// Bench starts a span at the benchmark level. If the context is already wrapped in a
// span, the span will be a child of the existing span.
func (t *Tracer) Bench(ctx context.Context, key string) (context.Context, Span) {
	return t.Trace(ctx, key, Bench)
}

// Trace wraps the given context in a span with the given key and level. If the context
// is already wrapped in a span, the span will be a child of the existing span.
func (t *Tracer) Trace(ctx context.Context, key string, env Environment) (context.Context, Span) {
	if t == nil || !t.config.Filter(env, key) {
		return ctx, nopSpanV
	}
	// Pulled from go implementation of pprof.Do:
	// https://cs.opensource.google/go/go/+/master:src/runtime/pprof/runtime.go;l=40?q=Do%20pprof&sq=&ss=go%2Fgo
	ctx = pprof.WithLabels(ctx, pprof.Labels("routine", key))
	pprof.SetGoroutineLabels(ctx)

	ctx, otel := t.config.OtelProvider.Tracer(t.meta.Key).Start(ctx, fmt.Sprintf("%s.%s", t.meta.Key, key))
	return ctx, span{
		pprofEnd: func() { pprof.SetGoroutineLabels(ctx) },
		otel:     otel,
	}
}

// Transfer transfers the trace from the source context to the target context.
// Transfer should be used sparingly, and is typically only used when preventing
// the propagation of context cancellation but the preservation of tracing:
//
//	def myFunc(ctx context.Context) {
//		// Transfer the trace so the given context cant be used for cancellation,
//		ctx = alamos.Transfer(ctx, context.Background())
//		// Will never return.
//		<-ctx.Done()
//	}
func (t *Tracer) Transfer(source, target context.Context) context.Context {
	return oteltrace.ContextWithSpan(target, oteltrace.SpanFromContext(source))
}

func (t *Tracer) sub(meta InstrumentationMeta) *Tracer {
	if t == nil {
		return nil
	}
	return &Tracer{meta: meta, config: t.config}
}

type span struct {
	pprofEnd func()
	otel     oteltrace.Span
}

var _ Span = span{}

// Error implements Span.
func (s span) Error(err error, exclude ...error) error {
	if err == nil {
		return err
	}
	s.otel.RecordError(err)
	if !errutil.IsAny(err, exclude...) {
		s.Status(Error)
	}
	return err
}

// Status implements Span.
func (s span) Status(status Status) {
	s.otel.SetStatus(status.otel(), "")
}

// End implements Span.
func (s span) End() {
	s.pprofEnd()
	s.otel.End()
}

// EndWith implements Span.
func (s span) EndWith(err error, exclude ...error) error {
	_ = s.Error(err, exclude...)
	s.End()
	return err
}

// nopSpan is a span that does nothing.
type nopSpan struct{}

var nopSpanV Span = nopSpan{}

// Error implements Span.
func (s nopSpan) Error(err error, _ ...error) error { return err }

// Status implements Span.
func (s nopSpan) Status(_ Status) {}

// End implements Span.
func (s nopSpan) End() {}

// EndWith implements Span.
func (s nopSpan) EndWith(err error, _ ...error) error { return err }
