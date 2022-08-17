package confluence

import (
	"context"
	"github.com/arya-analytics/x/signal"
)

// Delta is an abstract Segment that reads values from an input Stream
// and pipes them to multiple output streams. Delta does not implement the
// Flow method, and is therefore not usable directly. It should be embedded in a
// concrete segment.
type Delta[I, O Value] struct {
	UnarySink[I]
	AbstractMultiSource[O]
}

// DeltaMultiplier reads a value from a set of input streams and copies the value to
// every output stream.
type DeltaMultiplier[V Value] struct{ Delta[V, V] }

// Flow implements the Segment interface.
func (d *DeltaMultiplier[V]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachInletCloser(d)
	d.GoRange(ctx, d.SendToEach, o.Signal...)
}

// DeltaTransformMultiplier reads a value from an input stream, performs a
// transformation on it, and writes the transformed value to every output stream.
type DeltaTransformMultiplier[I, O Value] struct {
	Delta[I, O]
	TransformFunc[I, O]
}

// Flow implements the Segment interface.
func (d *DeltaTransformMultiplier[I, O]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachInletCloser(d)
	d.GoRange(ctx, d.transformAndMultiply, o.Signal...)
}

func (d *DeltaTransformMultiplier[I, O]) transformAndMultiply(ctx context.Context, i I) error {
	o, ok, err := d.ApplyTransform(ctx, i)
	if !ok || err != nil {
		return err
	}
	return d.SendToEach(ctx, o)
}
