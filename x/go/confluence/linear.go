// Copyright 2026 Synnax Labs, Inc.
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
	v, shouldSend, err := l.Transform(ctx, i)
	if err != nil || !shouldSend {
		return err
	}
	return signal.SendUnderContext(ctx, l.Out.Inlet(), v)
}
