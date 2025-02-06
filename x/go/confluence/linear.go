// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package confluence

import (
	"context"

	"github.com/synnaxlabs/x/signal"
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
	Transform TransformFunc[I, O]
}

// Flow implements the Segment interface.
func (l *LinearTransform[I, O]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachClosables(l.Out)
	l.GoRange(ctx, l.transform, o.Signal...)
}

func (l *LinearTransform[I, O]) transform(ctx context.Context, i I) error {
	v, ok, err := l.Transform(ctx, i)
	if err != nil {
		return err
	}
	if err != nil || !ok {
		return err
	}
	return signal.SendUnderContext(ctx, l.Out.Inlet(), v)
}

type TranslateFunc[I, O Value] func(I) (O, error)

type translator[I, IT, O, OT Value] struct {
	AbstractLinear[I, O]
	inlet     Inlet[IT]
	outlet    Outlet[OT]
	requestT  TranslateFunc[I, IT]
	responseT TranslateFunc[OT, O]
	wrapped   Flow
}

func NewTranslator[I, IT, O, OT Value](
	wrap Segment[IT, OT],
	requests TranslateFunc[I, IT],
	responses TranslateFunc[OT, O],
	buffers ...int,
) Segment[I, O] {
	var (
		buf = parseBuffer(buffers)
		in  = NewStream[IT](buf)
		out = NewStream[OT](buf)
		t   = &translator[I, IT, O, OT]{
			requestT:  requests,
			responseT: responses,
			inlet:     in,
			outlet:    out,
			wrapped:   wrap,
		}
	)
	wrap.InFrom(in)
	wrap.OutTo(out)
	return t
}

func (t *translator[I, IT, O, OT]) Flow(ctx signal.Context, opts ...Option) {
	t.wrapped.Flow(ctx, opts...)
	o := NewOptions(opts)
	o.AttachClosables(t.inlet)
	signal.GoRange(ctx, t.In.Outlet(), func(ctx context.Context, v I) error {
		o, err := t.requestT(v)
		if err != nil {
			return err
		}
		return signal.SendUnderContext(ctx, t.inlet.Inlet(), o)
	}, append(o.Signal, signal.WithKey("request-translator"))...)
	signal.GoRange(ctx, t.outlet.Outlet(), func(ctx context.Context, v OT) error {
		o, err := t.responseT(v)
		if err != nil {
			return err
		}
		return signal.SendUnderContext(ctx, t.Out.Inlet(), o)
	}, append(o.Signal, signal.WithKey("response-translator"))...)
}
