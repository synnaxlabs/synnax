package confluence

import (
	"context"
	"github.com/arya-analytics/x/signal"
)

// AbstractLinear is an abstract Segment that reads values from a single Inlet and
// pipes them to a single Outlet. AbstractLinear does not implement the Flow method,
// and is therefore not usable directly. It should be embedded in a concrete segment.
type AbstractLinear[I, O Value] struct {
	UnarySink[I]
	AbstractUnarySource[O]
}

// LinearTransform is a Segment that reads values from a single Inlet, performs a
// transformation, and writes the result to a single Outlet.
type LinearTransform[I, O Value] struct {
	AbstractLinear[I, O]
	TransformFunc[I, O]
}

// Flow implements the Segment interface.
func (l *LinearTransform[I, O]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachInletCloser(l)
	l.GoRange(ctx, l.transform, o.Signal...)
}

func (l *LinearTransform[I, O]) transform(ctx context.Context, i I) error {
	v, ok, err := l.ApplyTransform(ctx, i)
	if err != nil || !ok {
		return err
	}
	return signal.SendUnderContext(ctx, l.Out.Inlet(), v)
}
