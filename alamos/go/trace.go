package alamos

import (
	"context"
	"go.opentelemetry.io/otel/trace"
	"runtime/pprof"
)

func Trace(ctx context.Context, key string) (context.Context, Span) {
	exp := FromContext(ctx)
	return exp.startTrace(ctx, key)
}

type tracer interface {
	startTrace(ctx context.Context, key string) (context.Context, Span)
}

type Span interface {
	Error(err error)
	Status(status Status)
	End()
}

type defaultTracer struct {
	otel trace.Tracer
}

func (t *defaultTracer) startTrace(ctx context.Context, key string) (context.Context, Span) {
	defer pprof.SetGoroutineLabels(ctx)
	ctx = pprof.WithLabels(ctx, pprof.Labels(key))
	pprof.SetGoroutineLabels(ctx)
	_, otel := t.otel.Start(ctx, key)
	return ctx, span{
		pprofEnd: func() { pprof.SetGoroutineLabels(ctx) },
		otel:     otel,
	}
}

type span struct {
	pprofEnd func()
	otel     trace.Span
}

func (s span) Error(err error) {
	s.otel.RecordError(err)
}

func (s span) Status(status Status) {
	s.otel.SetStatus(status.otel(), "")
}

func (s span) End() {
	s.pprofEnd()
	s.otel.End()
}
