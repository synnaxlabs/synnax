// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package core

import (
	"math"
	"sync"
)

const leadingAlignmentEdge = math.MaxUint32 - 50e6

// EntityCount tracks the number of open iterators and writers on a particular entity.
type EntityCount struct {
	mu                 sync.RWMutex
	openIterators      int
	openWriters        int
	writeAlignmentEdge uint32
}

// AddWriter increments the writer count, and returns the current write alignment edge
// i.e. the virtual domain that uniquely identifies/orders the samples in the writer. Also
// returns a function that decrements the writer count.
func (c *EntityCount) AddWriter() (uint32, func()) {
	c.mu.Lock()
	c.openWriters += 1
	c.writeAlignmentEdge += 1
	c.mu.Unlock()
	return c.writeAlignmentEdge, func() {
		c.mu.Lock()
		c.openWriters -= 1
		c.mu.Unlock()
	}
}

// AddIterator increments the iterator count, and returns a function that decrements
// the iterator count.
func (c *EntityCount) AddIterator() func() {
	c.mu.Lock()
	c.openIterators += 1
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		c.openIterators -= 1
		c.mu.Unlock()
	}
}

// LockAndCountOpen locks the count and returns the sum of open iterators and writers. Also returns
// a function that unlocks the count.
func (c *EntityCount) LockAndCountOpen() (int, func()) {
	c.mu.Lock()
	return c.openIterators + c.openWriters, c.mu.Lock
}
