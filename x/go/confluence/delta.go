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
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/signal"
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
	o.AttachClosables(InletsToClosables(d.Out)...)
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
}

func NewDynamicDeltaMultiplier[V Value]() *DynamicDeltaMultiplier[V] {
	return &DynamicDeltaMultiplier[V]{
		connections:    make(chan []Inlet[V]),
		disconnections: make(chan []Inlet[V]),
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
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case inlets := <-d.connections:
				d.Source.Out = append(d.Source.Out, inlets...)
			case inlets := <-d.disconnections:
				d.performDisconnect(inlets)
			case v := <-d.In.Outlet():
				if err := d.Source.SendToEach(ctx, v); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}

func (d *DynamicDeltaMultiplier[V]) performDisconnect(inlets []Inlet[V]) {
	for _, inlet := range inlets {
		i, ok := d.getInletIndex(inlet)
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

func (d *DynamicDeltaMultiplier[V]) _a(inlets []Inlet[V]) {
	for _, inlet := range inlets {
		_, ok := d.getInletIndex(inlet)
		if ok {
			panic(fmt.Sprintf(
				"[confluence] - attempted to connect inlet that was already connected: %s",
				inlet.InletAddress()))
		}
		d.Source.Out = append(d.Source.Out, inlet)
	}
}

func (d *DynamicDeltaMultiplier[V]) getInletIndex(inlet Inlet[V]) (int, bool) {
	_, i, ok := lo.FindIndexOf(d.Source.Out, func(i Inlet[V]) bool {
		return i.InletAddress() == inlet.InletAddress()
	})
	return i, ok
}
