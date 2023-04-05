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
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/address"
	atomicx "github.com/synnaxlabs/x/atomic"
	"sync"
)

// NewStream opens a new Stream with the given buffer capacity.
func NewStream[V Value](buffer ...int) Stream[V] {
	return &streamImpl[V]{
		values: make(chan V, parseBuffer(buffer)),
		c:      &atomicx.Int32Counter{},
	}
}

// NewInlet returns an Inlet that wraps the provided channel.
func NewInlet[V Value](ch chan<- V) Inlet[V] {
	return &inletImpl[V]{
		values: ch,
		c:      &atomicx.Int32Counter{},
	}
}

// NewOutlet returns an Outlet that wraps the provided channel.
func NewOutlet[V Value](ch <-chan V) Outlet[V] { return &outletImpl[V]{values: ch} }

type streamImpl[V Value] struct {
	inletAddr, outletAddr address.Address
	values                chan V
	once                  sync.Once
	c                     *atomicx.Int32Counter
}

// Inlet implements Stream.
func (s *streamImpl[V]) Inlet() chan<- V { return s.values }

// Outlet represents Stream.
func (s *streamImpl[V]) Outlet() <-chan V { return s.values }

// InletAddress implements Stream.
func (s *streamImpl[V]) InletAddress() address.Address { return s.inletAddr }

func (s *streamImpl[V]) Acquire(n int32) {
	_, _ = s.c.Add(n)
}

func (s *streamImpl[V]) Close() {
	_, _ = s.c.Add(-1)
	if s.c.Value() <= 0 {
		s.once.Do(func() { close(s.values) })
	}
}

// SetInletAddress implements Stream.
func (s *streamImpl[V]) SetInletAddress(addr address.Address) { s.inletAddr = addr }

// OutletAddress implements Stream.
func (s *streamImpl[V]) OutletAddress() address.Address { return s.outletAddr }

// SetOutletAddress implements Stream.
func (s *streamImpl[V]) SetOutletAddress(addr address.Address) { s.outletAddr = addr }

type inletImpl[V Value] struct {
	addr   address.Address
	values chan<- V
	once   sync.Once
	c      *atomicx.Int32Counter
}

// Inlet implements Inlet.
func (i *inletImpl[V]) Inlet() chan<- V { return i.values }

// InletAddress implements Inlet.
func (i *inletImpl[V]) InletAddress() address.Address { return i.addr }

// SetInletAddress implements Inlet.
func (i *inletImpl[V]) SetInletAddress(addr address.Address) { i.addr = addr }

// Acquire implements Inlet.
func (i *inletImpl[V]) Acquire(n int32) { _, _ = i.c.Add(n) }

// Close implements inlet.
func (i *inletImpl[V]) Close() {
	_, _ = i.c.Add(-1)
	if i.c.Value() <= 0 {
		i.once.Do(func() { close(i.values) })
	}
}

type outletImpl[V Value] struct {
	addr   address.Address
	values <-chan V
}

// Outlet implements Outlet.
func (o *outletImpl[V]) Outlet() <-chan V { return o.values }

// OutletAddress implements Outlet.
func (o *outletImpl[V]) OutletAddress() address.Address { return o.addr }

// SetOutletAddress implements Outlet.
func (o *outletImpl[V]) SetOutletAddress(addr address.Address) { o.addr = addr }

func InletsToClosables[V Value](inlets []Inlet[V]) []Closable {
	return lo.Map(inlets, func(inlet Inlet[V], _ int) Closable { return inlet })
}

func InletMapToClosables[V Value](inlets map[address.Address]Inlet[V]) []Closable {
	return lo.MapToSlice(inlets, func(k address.Address, in Inlet[V]) Closable { return in })
}

const defaultBuffer = 0

func parseBuffer(buffer []int) int {
	if len(buffer) > 0 {
		return buffer[0]
	}
	return defaultBuffer
}
