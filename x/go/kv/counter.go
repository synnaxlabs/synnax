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
	atomicx.Int64Counter
	ctx    context.Context
	writer Writer
	key    []byte
	buffer []byte
}

// OpenCounter opens or creates a persisted counter at the given key. If
// the counter value is found in storage, sets its internal state. If the counter
// value is not found in storage, sets the value to 0.
func OpenCounter(writer Writer, key []byte) (*PersistedCounter, error) {
	c := &PersistedCounter{writer: writer, key: key, buffer: make([]byte, 8)}
	b, err := writer.Get(key)
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
	next, _ := c.Int64Counter.Add(delta...)
	binary.LittleEndian.PutUint64(c.buffer, uint64(next))
	return next, c.writer.Set(c.key, c.buffer)
}
