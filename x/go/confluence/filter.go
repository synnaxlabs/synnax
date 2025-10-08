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

// Filter is a segment that reads values from an input Stream, filters them through a
// function, and optionally discards them to an output Stream.
type Filter[V Value] struct {
	AbstractLinear[V, V]
	Filter FilterFunc[V]
	// Rejects is the Inlet that receives values that were discarded by Apply.
	Rejects Inlet[V]
}

// FilterFunc is called on each value passing through the Filter. If it returns false,
// the value is discarded or sent to the Rejects Inlet. If it returns true,
// the value is sent through the standard Inlet. If an error is returned,
// the Filter is closed and a fatal error is returned to the context.
type FilterFunc[V Value] func(ctx context.Context, v V) (ok bool, err error)

// OutTo implements the Segment interface. It accepts either one or two Inlet(sink).
// The first Inlet is where accepted values are sent, and the second Inlet (if provided)
// is where Rejected values are sent.
func (f *Filter[V]) OutTo(inlets ...Inlet[V]) {
	if len(inlets) > 2 || len(inlets) == 0 {
		panic("[confluence.Filter] - provide at most two and at least one inlet")
	}
	if len(inlets) == 1 && f.Out != nil {
		f.Rejects = inlets[0]
		return
	}
	f.AbstractLinear.OutTo(inlets[0])
	if len(inlets) == 2 {
		f.Rejects = inlets[1]
	}
}

// Flow implements the Segment interface.
func (f *Filter[V]) Flow(ctx signal.Context, opts ...Option) {
	fo := NewOptions(opts)
	fo.AttachClosables(f.Out, f.Rejects)
	f.GoRange(ctx, f.filter, fo.Signal...)
}

func (f *Filter[V]) filter(ctx context.Context, v V) error {
	ok, err := f.Filter(ctx, v)
	if err != nil {
		return err
	}
	if ok {
		return signal.SendUnderContext(ctx, f.Out.Inlet(), v)
	} else if f.Rejects != nil {
		return signal.SendUnderContext(ctx, f.Rejects.Inlet(), v)
	}
	return nil
}
