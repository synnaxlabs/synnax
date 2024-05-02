// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/timeout"
	"time"
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
	Delta[I, O]
	Transform TransformFunc[I, O]
}

// Flow implements the Segment interface.
func (d *DeltaTransformMultiplier[I, O]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	o.AttachClosables(InletsToClosables(d.Out)...)
	d.GoRange(ctx, d.transformAndMultiply, o.Signal...)
}

func (d *DeltaTransformMultiplier[I, O]) transformAndMultiply(ctx context.Context, i I) error {
	o, ok, err := d.Transform(ctx, i)
	if !ok || err != nil {
		return err
	}
	return d.SendToEach(ctx, o)
}

type DynamicDeltaMultiplier[V Value] struct {
	UnarySink[V]
	Source         AbstractMultiSource[V]
	connections    chan []Inlet[V]
	disconnections chan []Inlet[V]
	timeout        time.Duration
}

func NewDynamicDeltaMultiplier[V Value](timeout time.Duration, connectionBuffers ...int) *DynamicDeltaMultiplier[V] {
	buf := parseBuffer(connectionBuffers)
	return &DynamicDeltaMultiplier[V]{
		connections:    make(chan []Inlet[V], buf),
		disconnections: make(chan []Inlet[V], buf),
		timeout:        timeout,
	}
}

func (d *DynamicDeltaMultiplier[V]) Connect(inlets ...Inlet[V]) {
	d.connections <- inlets
}

func (d *DynamicDeltaMultiplier[V]) Disconnect(inlets ...Inlet[V]) {
	d.disconnections <- inlets
}

func (d *DynamicDeltaMultiplier[v]) Flow(ctx signal.Context, opts ...Option) {
	o := NewOptions(opts)
	ctx.Go(func(ctx context.Context) error {
		var timer *time.Timer
		if d.timeout > 0 {
			timer = time.NewTimer(d.timeout)
		}
		defer func() {
			if timer != nil && !timer.Stop() {
				<-timer.C
			}
			d.disconnectAll()
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
				var err error
				if timer != nil {

					if !timer.Stop() {
						// If the timer had already fired, drain the channel.
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(d.timeout)
					err = d.Source.SendToEachWithTimeout(ctx, res, timer.C)
				} else {
					err = d.Source.SendToEach(ctx, res)
				}
				if err != nil {
					if !errors.Is(err, timeout.Timeout) {
						return err
					}
					fmt.Println("delta - slow consumer")
				}
			}
		}
	}, o.Signal...)
}

func (d *DynamicDeltaMultiplier[V]) disconnectAll() {
	for _, inlet := range d.Source.Out {
		inlet.Close()
	}
	d.Source.Out = nil
}

func (d *DynamicDeltaMultiplier[V]) disconnect(inlets []Inlet[V]) {
	for _, inlet := range inlets {
		i, ok := d.findInletIndex(inlet)
		if !ok {
			panic(fmt.Sprintf(
				"[confluence] - attempted to disconnect inlet %v, but it was never connected",
				inlet,
			))
		}
		d.Source.Out = append(d.Source.Out[:i], d.Source.Out[i+1:]...)
		inlet.Close()
	}
}

func (d *DynamicDeltaMultiplier[V]) connect(inlets []Inlet[V]) {
	for _, inlet := range inlets {
		_, ok := d.findInletIndex(inlet)
		if ok {
			panic(fmt.Sprintf(
				"[confluence] - attempted to connect inlet that was already connected: %s",
				inlet.InletAddress()))
		}
		inlet.Acquire(1)
		d.Source.Out = append(d.Source.Out, inlet)
	}
}

func (d *DynamicDeltaMultiplier[V]) findInletIndex(inlet Inlet[V]) (int, bool) {
	_, i, ok := lo.FindIndexOf(d.Source.Out, func(i Inlet[V]) bool {
		return i.InletAddress() == inlet.InletAddress()
	})
	return i, ok
}
