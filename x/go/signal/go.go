// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package signal

import (
	"context"
	"time"
)

// Go is the core interface for forking a new goroutine.
type Go interface {
	// Go starts a new goroutine with the provided key under the given Context.
	// When the Context is canceled, the goroutine should abort its work and exit.
	// Additional parameters can be passed to the goroutine to modify its behavior.
	// See the RoutineOption documentation for more.
	Go(f func(ctx context.Context) error, opts ...RoutineOption)
}

// Go implements the Go interface.
func (c *core) Go(f func(ctx context.Context) error, opts ...RoutineOption) {
	newRoutine(c, opts).goRun(f)
}

// GoRange starts a new goroutine controlled by the provided Go that ranges
// over the values in ch. The goroutine will exit when the context is canceled
// or the channel is closed. Additional parameters can be passed to the goroutine
// to modify its behavior. See the RoutineOption documentation for more.
func GoRange[V any](
	g Go,
	ch <-chan V,
	f func(context.Context, V) error,
	opts ...RoutineOption,
) {
	g.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case v, ok := <-ch:
				if !ok {
					return nil
				}
				if err := f(ctx, v); err != nil {
					return err
				}
			}
		}
	}, opts...)
}

// GoTick starts a new goroutine controlled by the provided Go that
// ticks at the provided interval. The goroutine will exit when the context
// is cancelled. Additional parameters can be passed to the goroutine to
// modify its behavior. See the RoutineOption documentation for more.
func GoTick(
	g Go,
	interval time.Duration,
	f func(context.Context, time.Time) error,
	opts ...RoutineOption,
) {
	t := time.NewTicker(interval)
	GoRange(g, t.C, f, append(opts, Defer(t.Stop, WithKey("stopTicker")))...)
}
