// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/cockroachdb/errors"
	atomicx "github.com/synnaxlabs/x/atomic"
)

// PersistedCounter implements a simple in64 counter that writes its value to a
// key-value store. PersistedCounter is safe for concurrent use. To create a new
// PersistedCounter, call OpenCounter.
type PersistedCounter struct {
	ctx context.Context
	db  Writeable
	atomicx.Int64Counter
	key    []byte
	buffer []byte
}

// OpenCounter opens or creates a persisted counter at the given key. If
// the counter value is found in storage, sets its internal state. If the counter
// value is not found in storage, sets the value to 0.
func OpenCounter(ctx context.Context, db DB, key []byte) (*PersistedCounter, error) {
	c := &PersistedCounter{db: db, key: key, buffer: make([]byte, 8)}
	b, err := db.NewReader(ctx).Get(key)
	if err == nil {
		_, _ = c.Int64Counter.Add(int64(binary.LittleEndian.Uint64(b)))
	} else if errors.Is(err, NotFound) {
		err = nil
	}
	return c, err
}

// Add increments the counter by the sum of the given values. If no values are
// provided, increments the counter by 1.
// as well as any errors encountered while flushing the counter to storage.
func (c *PersistedCounter) Add(delta ...int64) (int64, error) {
	writer := c.db.NewWriter(c.ctx)
	defer func() { _ = writer.Close() }()
	next, _ := c.Int64Counter.Add(delta...)
	binary.LittleEndian.PutUint64(c.buffer, uint64(next))
	if err := writer.Set(c.key, c.buffer); err != nil {
		return next, err
	}
	return next, writer.Commit()
}
