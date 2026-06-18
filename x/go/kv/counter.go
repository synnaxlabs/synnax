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
	"sync/atomic"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// AtomicInt64Counter implements a simple int64 counter that writes its value to a
// key-value store. AtomicInt64Counter is safe for concurrent use. To create a new
// AtomicInt64Counter, call NewCounter.
type AtomicInt64Counter struct {
	db    Writer
	key   []byte
	value atomic.Int64
}

// NewCounter opens or creates a persisted counter at the given key. If the counter
// value is found in storage, sets its internal state. If the counter value is not found
// in storage, sets the value to 0.
func NewCounter(
	ctx context.Context, db ReadWriter, key []byte,
) (*AtomicInt64Counter, error) {
	c := &AtomicInt64Counter{db: db, key: key}
	b, closer, err := db.Get(ctx, key)
	if err != nil {
		return nil, errors.Skip(err, query.ErrNotFound)
	}
	c.value.Store(int64(binary.LittleEndian.Uint64(b)))
	if err := closer.Close(); err != nil {
		return nil, err
	}
	return c, nil
}

// Value returns the current counter value.
func (c *AtomicInt64Counter) Value() int64 { return c.value.Load() }

// Add increments the counter by the given delta. Returns the new counter value as well
// as any errors encountered while flushing the counter to storage.
func (c *AtomicInt64Counter) Add(ctx context.Context, delta int64) (int64, error) {
	next := c.value.Add(delta)
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(next))
	return next, c.db.Set(ctx, c.key, buf[:])
}

// Set sets the counter to the given value.
func (c *AtomicInt64Counter) Set(ctx context.Context, value int64) error {
	c.value.Store(value)
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(value))
	return c.db.Set(ctx, c.key, buf[:])
}
