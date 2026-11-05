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
	"fmt"
	"slices"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/timeout"
)

// Delta is an abstract Segment that reads values from an input Stream
// and pipes them to multiple output streams. Delta does not implement the
// Flow method, and is therefore not usable directly. It should be embedded in a
// concrete segment.
type Delta[I, O Value] struct {
	UnarySink[I]
	AbstractMultiSource[O]
}

// DeltaMultiplier reads a value from an of input stream and copies the value to
// every output stream.
type DeltaMultiplier[V Value] struct{ Delta[V, V] }

// Flow implements the Segment interface.
func (d *DeltaMultiplier[V]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachClosables(InletsToClosables(d.Out)...)
	d.GoRange(ctx, d.SendToEach, o.Signal...)
}

// DeltaTransformMultiplier reads a value from an input stream, performs a
// transformation on it, and writes the transformed value to every output stream.
type DeltaTransformMultiplier[I, O Value] struct {
	Transform TransformFunc[I, O]
	Delta[I, O]
}

// Flow implements the Segment interface.
func (d *DeltaTransformMultiplier[I, O]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachClosables(InletsToClosables(d.Out)...)
	d.GoRange(ctx, d.transformAndMultiply, o.Signal...)
}

func (d *DeltaTransformMultiplier[I, O]) transformAndMultiply(
	ctx context.Context,
	i I,
) error {
	o, ok, err := d.Transform(ctx, i)
	if !ok || err != nil {
		return err
	}
	return d.SendToEach(ctx, o)
}

// DynamicDeltaMultiplier is a segment that reads values from an input stream and
// dynamically distributes them to multiple output streams. It supports runtime
// connection and disconnection of output streams, with optional timeout handling for
// sending values to downstream consumers. Once Flow exits, the multiplier has stopped:
// every connected inlet is closed, and it closes any inlet connected later.
type DynamicDeltaMultiplier[V Value] struct {
	alamos.Instrumentation
	UnarySink[V]
	connections    chan []Inlet[V]
	disconnections chan []Inlet[V]
	stopped        chan struct{}
	// sending is the inlet of an unfinished send. Only the Flow goroutine accesses it.
	sending Inlet[V]
	Source  AbstractMultiSource[V]
	timeout time.Duration
}

// NewDynamicDeltaMultiplier creates a new DynamicDeltaMultiplier with the specified
// timeout duration and instrumentation.
func NewDynamicDeltaMultiplier[V Value](
	timeout time.Duration,
	instrumentation alamos.Instrumentation,
) *DynamicDeltaMultiplier[V] {
	return &DynamicDeltaMultiplier[V]{
		Instrumentation: instrumentation,
		connections:     make(chan []Inlet[V]),
		disconnections:  make(chan []Inlet[V]),
		stopped:         make(chan struct{}),
		timeout:         timeout,
	}
}

// Connect adds one or more inlets to the DynamicDeltaMultiplier's output streams. The
// inlets will receive values from the input stream until they are disconnected. If the
// multiplier has stopped, Connect closes the inlets.
func (d *DynamicDeltaMultiplier[V]) Connect(inlets ...Inlet[V]) {
	select {
	case d.connections <- inlets:
	case <-d.stopped:
		for _, inlet := range inlets {
			inlet.Acquire(1)
			inlet.Close()
		}
	}
}

// Disconnect removes one or more inlets from the DynamicDeltaMultiplier's output
// streams. The inlets will no longer receive values from the input stream. It is a
// no-op if the multiplier has stopped, since stopping closes every inlet.
func (d *DynamicDeltaMultiplier[V]) Disconnect(inlets ...Inlet[V]) {
	select {
	case d.disconnections <- inlets:
	case <-d.stopped:
	}
}

// Stopped returns a channel that is closed once Flow has exited and will not retry.
func (d *DynamicDeltaMultiplier[V]) Stopped() <-chan struct{} { return d.stopped }

// Flow implements the Segment interface. It continuously reads values from the input
// stream and distributes them to all connected output streams. If a timeout is
// configured, it will attempt to send values to downstream consumers within the timeout
// period. The Flow method handles dynamic connection and disconnection of output
// streams through the connections and disconnections channels. If a send panics and
// WithRetryOnPanic retries Flow, the retry disconnects the inlet that panicked and
// keeps every other connection.
func (d *DynamicDeltaMultiplier[v]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.Signal = append(o.Signal, signal.Defer(d.stop, signal.WithKey("stop")))
	ctx.Go(func(ctx context.Context) error {
		if d.sending != nil {
			d.disconnect([]Inlet[v]{d.sending})
			d.sending = nil
		}
		var timer *time.Timer
		if d.timeout > 0 {
			timer = time.NewTimer(d.timeout)
		}
		defer func() {
			if timer != nil && !timer.Stop() {
				<-timer.C
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case inlets := <-d.connections:
				d.connect(inlets)
			case inlets := <-d.disconnections:
				d.disconnect(inlets)
			case res, ok := <-d.In.Outlet():
				if !ok {
					return nil
				}
				if err := d.send(ctx, res, timer); err != nil {
					if !errors.Is(err, timeout.ErrTimeout) {
						return err
					}
					d.L.Warn(fmt.Sprintf("delta: %s", err))
				}
			}
		}
	}, o.Signal...)
}

// send sends v to each inlet. With a timer, an inlet that does not receive v before the
// timer fires is skipped, the remaining inlets still receive v, and send returns
// timeout.ErrTimeout.
func (d *DynamicDeltaMultiplier[V]) send(
	ctx context.Context,
	v V,
	timer *time.Timer,
) error {
	var expired <-chan time.Time
	if timer != nil {
		timer.Reset(d.timeout)
		expired = timer.C
	}
	var timedOut Inlet[V]
	for _, inlet := range d.Source.Out {
		d.sending = inlet
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-expired:
			timer.Reset(d.timeout)
			timedOut = inlet
		case inlet.Inlet() <- v:
		}
	}
	d.sending = nil
	if timedOut != nil {
		return errors.Wrapf(
			timeout.ErrTimeout,
			"timed out sending to inlet %s",
			timedOut.InletAddress(),
		)
	}
	return nil
}

func (d *DynamicDeltaMultiplier[V]) stop() {
	for _, inlet := range d.Source.Out {
		inlet.Close()
	}
	d.Source.Out = nil
	close(d.stopped)
}

// disconnect removes the specified inlets from the output streams and closes them.
// If an inlet is not found in the current connections, it logs a warning.
func (d *DynamicDeltaMultiplier[V]) disconnect(inlets []Inlet[V]) {
	for _, inlet := range inlets {
		i, ok := d.findInletIndex(inlet)
		if !ok {
			d.L.DPanic(fmt.Sprintf(
				"[confluence] - attempted to disconnect inlet %v, but it was never connected",
				inlet,
			))
			return
		}
		d.Source.Out = slices.Delete(d.Source.Out, i, i+1)
		inlet.Close()
	}
}

// connect adds the specified inlets to the output streams. If an inlet is already
// connected, it logs a warning. Each new inlet is acquired with a count of 1.
func (d *DynamicDeltaMultiplier[V]) connect(inlets []Inlet[V]) {
	for _, inlet := range inlets {
		if _, ok := d.findInletIndex(inlet); ok {
			d.L.DPanic(fmt.Sprintf(
				"[confluence] - attempted to connect inlet that was already connected: %s",
				inlet.InletAddress(),
			))
		}
		inlet.Acquire(1)
		d.Source.Out = append(d.Source.Out, inlet)
	}
}

// findInletIndex returns the index of the specified inlet in the output streams
// and a boolean indicating whether the inlet was found.
func (d *DynamicDeltaMultiplier[V]) findInletIndex(inlet Inlet[V]) (int, bool) {
	_, i, ok := lo.FindIndexOf(d.Source.Out, func(i Inlet[V]) bool {
		return i.InletAddress() == inlet.InletAddress()
	})
	return i, ok
}
