// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"
	"encoding/binary"

	atomicx "github.com/synnaxlabs/x/atomic"
	"github.com/synnaxlabs/x/errors"
)

// AtomicInt64Counter implements a simple int64 counter that writes its value to a
// key-value store. AtomicInt64Counter is safe for concurrent use. To create a new
// AtomicInt64Counter, call OpenCounter.
type AtomicInt64Counter struct {
	ctx context.Context
	db  Writer
	atomicx.Int64Counter
	key    []byte
	buffer []byte
}

// OpenCounter opens or creates a persisted counter at the given key. If
// the counter value is found in storage, sets its internal state. If the counter
// value is not found in storage, sets the value to 0.
func OpenCounter(ctx context.Context, db ReadWriter, key []byte) (*AtomicInt64Counter, error) {
	c := &AtomicInt64Counter{ctx: ctx, db: db, key: key, buffer: make([]byte, 8)}
	b, closer, err := db.Get(ctx, key)
	if err == nil {
		c.Int64Counter.Add(int64(binary.LittleEndian.Uint64(b)))
		err = closer.Close()
	} else if errors.Is(err, ErrNotFound) {
		err = nil
	}
	return c, err
}

// Add increments the counter by the sum of the given values. If no values are
// provided, increments the counter by 1. Returns the new counter value
// as well as any errors encountered while flushing the counter to storage.
func (c *AtomicInt64Counter) Add(delta int64) (int64, error) {
	next := c.Int64Counter.Add(delta)
	binary.LittleEndian.PutUint64(c.buffer, uint64(next))
	return next, c.db.Set(c.ctx, c.key, c.buffer)
}

// Set sets the counter to the given value.
func (c *AtomicInt64Counter) Set(value int64) error {
	c.Int64Counter.Set(value)
	binary.LittleEndian.PutUint64(c.buffer, uint64(value))
	return c.db.Set(c.ctx, c.key, c.buffer)
}
