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
	"time"

	"github.com/synnaxlabs/x/signal"
)

// Emitter is a Source that emits values to an Inlet at a regular interval.
type Emitter[V Value] struct {
	// Emitter is called on each tick. If it returns an error, the Emitter closes and
	// returns a fatal error to the context.
	Emit func(context.Context) (V, error)
	// Interval is the duration between ticks.
	Interval time.Duration
	AbstractUnarySource[V]
}

// Flow implements the Flow interface.
func (e *Emitter[V]) Flow(ctx signal.Context, opts ...Option) {
	fo := NewOptions(opts)
	fo.AttachClosables(e.Out)
	signal.GoTick(ctx, e.Interval, e.emit, fo.Signal...)
}

func (e *Emitter[V]) emit(ctx context.Context, _ time.Time) error {
	v, err := e.Emit(ctx)
	if err != nil {
		return err
	}
	return signal.SendUnderContext(ctx, e.Out.Inlet(), v)
}
