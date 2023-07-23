// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io

import (
	"encoding/binary"
	"github.com/synnaxlabs/x/atomic"
)

// Int32Counter is an atomic, file backed counter.
type Int32Counter struct {
	wrapped *atomic.Int32Counter
	f       ReaderAtWriterAtCloser
	buf     []byte
}

// NewInt32Counter opens a new, atomic counter backed by the given file. The counter
// must have exclusive write access to the file.
func NewInt32Counter(f ReaderAtWriterAtCloser) *Int32Counter {
	i := &Int32Counter{
		wrapped: &atomic.Int32Counter{},
		f:       f,
		buf:     make([]byte, 4),
	}
	return i
}

// Add increments the counter by the provided delta.
func (c *Int32Counter) Add(delta int32) (int32, error) {
	v := c.wrapped.Add(delta)
	binary.LittleEndian.PutUint32(c.buf, uint32(v))
	_, err := c.f.WriteAt(c.buf, 0)
	return v, err
}

// Value returns the current counter value.
func (c *Int32Counter) Value() int32 { return c.wrapped.Value() }

func (c *Int32Counter) load() (int32, error) {
	_, err := c.f.ReadAt(c.buf, 0)
	return int32(binary.LittleEndian.Uint32(c.buf)), err
}
