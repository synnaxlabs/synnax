package alamos

import (
	"context"
	"fmt"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.opentelemetry.io/otel/propagation"
	oteltrace "go.opentelemetry.io/otel/trace"
	"runtime/pprof"
)

var nopTracer = &Tracer{}

// Trace starts a new span with the given insKey. If the context already contains a span,
// the new span will be a child of the existing span. If the context does not contain
// a span, a no-op span will be returned.
func Trace(ctx context.Context, key string, level Level) (context.Context, Span) {
	return extract(ctx).T.Trace(ctx, key, level)
}

// TransferTrace transfers the trace from the source context to the target context.
// TransferTrace should be used sparingly, and is typically only used when preventing
// the propagation of context cancellation but the preservation of tracing:
//
//	def myFunc(ctx context.Context) {
//		// Transfer the trace so the given context cant be used for cancellation,
//		ctx = alamos.TransferTrace(ctx, context.Background())
//		// Will never return.
//		<-ctx.Done()
//	}
func TransferTrace(source context.Context, target context.Context) context.Context {
	ins := extract(source)
	return lo.Ternary(ins.valid(), attach(target, ins), target)
}

// Span is a span in a trace.
type Span interface {
	// Error records the given error as an error on the span, setting the span's status
	// to Error if the error is non-nil. If exclude is provided, the status will only be
	// set if the error is not one of the excluded errors.
	Error(err error, exclude ...error)
	// Status sets the span's status.
	Status(status Status)
	// EndWith combines Error and End, and also returns the given err unmodified.
	EndWith(err error, exclude ...error) error
	// End ends the span.
	End()
}

type TracingConfig struct {
	Otel       oteltrace.Tracer
	Propagator propagation.TextMapPropagator
}

var _ config.Config[TracingConfig] = (*TracingConfig)(nil)

func (c TracingConfig) Validate() error {
	v := validate.New("alamos.TracingConfig")
	validate.NotNil(v, "Otel", c.Otel)
	validate.NotNil(v, "Propagator", c.Propagator)
	return v.Error()
}

func (c TracingConfig) Override(other TracingConfig) TracingConfig {
	c.Otel = override.Nil(c.Otel, other.Otel)
	c.Propagator = override.Nil(c.Propagator, other.Propagator)
	return c
}

type Tracer struct {
	meta   InstrumentationMeta
	config TracingConfig
}

func NewTracer(configs ...TracingConfig) (*Tracer, error) {
	cfg, err := config.New(TracingConfig{}, configs...)
	if err != nil {
		return nil, err
	}
	return &Tracer{config: cfg}, nil
}

var _ sub[*Tracer] = (*Tracer)(nil)

func (t *Tracer) sub(meta InstrumentationMeta) *Tracer {
	if t == nil {
		return nil
	}
	return &Tracer{meta: meta, config: t.config}
}

func (t *Tracer) Trace(ctx context.Context, key string, level Level) (context.Context, Span) {
	if t == nil || !t.meta.Filter(level, key) {
		return ctx, nopSpan{}
	}
	// Pulled from go implementation of pprof.Do:
	// https://cs.opensource.google/go/go/+/master:src/runtime/pprof/runtime.go;l=40?q=Do%20pprof&sq=&ss=go%2Fgo
	ctx = pprof.WithLabels(ctx, pprof.Labels("routine", key))
	pprof.SetGoroutineLabels(ctx)

	ctx, otel := t.config.Otel.Start(ctx, fmt.Sprintf("%s.%s", t.meta.Key, key))
	return ctx, span{
		pprofEnd: func() { pprof.SetGoroutineLabels(ctx) },
		otel:     otel,
	}
}

type span struct {
	pprofEnd func()
	otel     oteltrace.Span
}

var _ Span = span{}

// Error implements Span.
func (s span) Error(err error, exclude ...error) {
	if err == nil {
		return
	}
	s.otel.RecordError(err)
	if !errutil.IsAny(err, exclude...) {
		s.Status(Error)
	}
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
	s.Error(err, exclude...)
	s.End()
	return err
}

// nopSpan is a span that does nothing.
type nopSpan struct{}

var _ Span = nopSpan{}

// Error implements Span.
func (s nopSpan) Error(_ error, _ ...error) {}

// Status implements Span.
func (s nopSpan) Status(_ Status) {}

// End implements Span.
func (s nopSpan) End() {}

// EndWith implements Span.
func (s nopSpan) EndWith(err error, _ ...error) error {
	return err
}
