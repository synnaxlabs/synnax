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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
)

// QueuePolicy decides which values a Queue holds. A Queue calls it from one goroutine.
type QueuePolicy[V Value] interface {
	// Push returns held with v added, oldest first. It may drop or change held values
	// and v, and it may reuse the memory of held.
	Push(held []V, v V) []V
	// Pop tells the policy that the Queue delivered v, its oldest held value.
	Pop(v V)
}

// Queue is a Segment that never blocks on its input. It holds the values that its
// output has not taken yet, and its Policy decides which values to drop. When its
// input closes, it delivers the values it holds and exits.
type Queue[V Value] struct {
	alamos.Instrumentation
	AbstractLinear[V, V]
	Policy QueuePolicy[V]
}

// Flow implements the Segment interface.
func (q *Queue[V]) Flow(sCtx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachClosables(q.Out)
	sCtx.Go(func(ctx context.Context) error {
		var (
			held []V
			in   = q.In.Outlet()
		)
		for {
			var (
				out  chan<- V
				next V
			)
			if len(held) > 0 {
				out = q.Out.Inlet()
				next = held[0]
			} else if in == nil {
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case v, ok := <-in:
				if !ok {
					in = nil
					continue
				}
				count := len(held) + 1
				held = q.Policy.Push(held, v)
				if dropped := count - len(held); dropped > 0 {
					q.L.Warn(
						"queue dropped values for a slow consumer",
						zap.Int("count", dropped),
					)
				}
			case out <- next:
				q.Policy.Pop(next)
				var zero V
				held[0] = zero
				held = held[1:]
			}
		}
	}, o.Signal...)
}
