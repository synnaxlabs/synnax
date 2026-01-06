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
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/atomic"
)

// NewStream opens a new Stream with the given buffer capacity.
func NewStream[V Value](buffer ...int) *Stream[V] {
	return &Stream[V]{
		values: make(chan V, parseBuffer(buffer)),
		c:      &atomic.Int32Counter{},
	}
}

func Attach[I, O Value](seg Segment[I, O], buffer ...int) (Inlet[I], Outlet[O]) {
	buf := parseBuffer(buffer)
	req := NewStream[I](buf)
	res := NewStream[O](buf)
	seg.InFrom(req)
	seg.OutTo(res)
	return req, res
}

// Stream represents a stream of values. Each stream has an addressable Outlet and an
// addressable Inlet. These addresses are best represented as unique locations where
// values are received from (Inlet) and sent to (Outlet). It is also generally OK to
// share a Stream across multiple Frame, as long as those segments perform are
// replicates of one another.
type Stream[V Value] struct {
	inletAddr, outletAddr address.Address
	values                chan V
	once                  sync.Once
	c                     *atomic.Int32Counter
}

// Inlet implements Stream.
func (s *Stream[V]) Inlet() chan<- V { return s.values }

// Outlet represents Stream.
func (s *Stream[V]) Outlet() <-chan V { return s.values }

// InletAddress implements Stream.
func (s *Stream[V]) InletAddress() address.Address { return s.inletAddr }

func (s *Stream[V]) Acquire(n int32) { s.c.Add(n) }

func (s *Stream[V]) Close() {
	s.c.Add(-1)
	if s.c.Value() <= 0 {
		s.once.Do(func() { close(s.values) })
	}
}

// SetInletAddress implements Stream.
func (s *Stream[V]) SetInletAddress(addr address.Address) { s.inletAddr = addr }

// OutletAddress implements Stream.
func (s *Stream[V]) OutletAddress() address.Address { return s.outletAddr }

// SetOutletAddress implements Stream.
func (s *Stream[V]) SetOutletAddress(addr address.Address) { s.outletAddr = addr }

func InletsToClosables[V Value](inlets []Inlet[V]) []Closable {
	return lo.Map(inlets, func(inlet Inlet[V], _ int) Closable { return inlet })
}

func InletMapToClosables[V Value](inlets map[address.Address]Inlet[V]) []Closable {
	return lo.MapToSlice(inlets, func(_ address.Address, in Inlet[V]) Closable {
		return in
	})
}

const defaultBuffer = 0

func parseBuffer(buffer []int) int {
	if len(buffer) > 0 {
		return buffer[0]
	}
	return defaultBuffer
}
