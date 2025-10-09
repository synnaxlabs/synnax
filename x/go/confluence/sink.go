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

// UnarySink is a basic implementation of Sink that can receive values from a single
// Inlet.
type UnarySink[V Value] struct {
	// Sink is called whenever a value is received from the Outlet.
	Sink func(context.Context, V) error
	AbstractUnarySink[V]
}

// Flow implements the Flow interface.
func (us *UnarySink[V]) Flow(ctx signal.Context, opts ...Option) {
	us.GoRange(ctx, us.Sink, NewOptions(opts).Signal...)
}

func (us *UnarySink[V]) GoRange(
	ctx signal.Context,
	f func(context.Context, V) error,
	opts ...signal.RoutineOption,
) {
	signal.GoRange(ctx, us.In.Outlet(), f, opts...)
}

type AbstractUnarySink[V Value] struct{ In Outlet[V] }

// InFrom implements the Sink interface.
func (as *AbstractUnarySink[V]) InFrom(outlets ...Outlet[V]) {
	if len(outlets) != 1 {
		panic("[confluence.UnarySink] - must have exactly one outlet")
	}
	as.In = outlets[0]
}
