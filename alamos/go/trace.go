package alamos

import (
	"context"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"runtime/pprof"
)

func Trace(ctx context.Context, key string) (context.Context, Span) {
	ins, ok := Extract(ctx)
	if !ok {
		return ctx, nopSpan{}
	}
	return ins.T.startTrace(ctx, key)
}

func TraceI(ctx context.Context, i Instrumentation, key string) (context.Context, Span) {
	return i.T.startTrace(ctx, key)
}

type Span interface {
	Error(err error) error
	EndWith(err error) error
	Status(status Status)
	End()
}

type Tracer struct {
	Otel       trace.Tracer
	Propagator propagation.TextMapPropagator
}

func ExtractTracer(i Instrumentation) *Tracer {
	return i.t()
}

func (t *Tracer) startTrace(ctx context.Context, key string) (context.Context, Span) {
	defer pprof.SetGoroutineLabels(ctx)
	ctx = pprof.WithLabels(ctx, pprof.Labels("routine", key))
	pprof.SetGoroutineLabels(ctx)
	ctx, otel := t.Otel.Start(ctx, key)
	return ctx, span{
		pprofEnd: func() { pprof.SetGoroutineLabels(ctx) },
		otel:     otel,
	}
}

type span struct {
	pprofEnd func()
	otel     trace.Span
}

func (s span) Error(err error) error {
	if err != nil {
		s.otel.RecordError(err)
		s.Status(Error)
	}
	return err
}

func (s span) Status(status Status) {
	s.otel.SetStatus(status.otel(), "")
}

func (s span) End() {
	s.pprofEnd()
	s.otel.End()
}

func (s span) EndWith(err error) error {
	s.Error(err)
	s.End()
	return err
}

type nopSpan struct{}

func (s nopSpan) Error(err error) error { return err }

func (s nopSpan) ErrorStatus(err error) error { return err }

func (s nopSpan) Status(status Status) {}

func (s nopSpan) End() {}

func (s nopSpan) EndWith(err error) error {
	return err
}

func newDevTracer(serviceName string) *Tracer {
	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN("https://Q8rwNMXo9z91P2PhDyUVng@uptrace.dev/1614"),
		uptrace.WithServiceName(serviceName),
		uptrace.WithServiceVersion("1.0.0"),
	)

	// Create a tracer. Usually, tracer is a global variable.
	return &Tracer{
		Otel:       otel.Tracer(serviceName),
		Propagator: otel.GetTextMapPropagator(),
	}
}
