// Copyright 2026 Synnax Labs, Inc.
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
	"io"
	"sync"

	"github.com/synnaxlabs/x/errors"
)

// Int32Counter is an atomic, file backed counter.
type Int32Counter struct {
	f       ReaderAtWriterAtCloser
	buf     []byte
	mu      sync.RWMutex
	wrapped int32
}

// NewInt32Counter opens a new, atomic counter backed by the given file. The counter
// must have exclusive write access to the file.
// If the file already exists, then it reads and initializes the wrapped value to the
// value stored in the file.
func NewInt32Counter(f ReaderAtWriterAtCloser) (*Int32Counter, error) {
	c := &Int32Counter{
		f:   f,
		buf: make([]byte, 4),
	}

	wrapped, err := c.load()
	if errors.Is(err, io.EOF) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}

	return &Int32Counter{
		wrapped: wrapped,
		f:       f,
		buf:     make([]byte, 4),
	}, nil
}

// Add increments the counter by the provided delta.
func (c *Int32Counter) Add(delta int32) (int32, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.wrapped += delta
	binary.LittleEndian.PutUint32(c.buf, uint32(c.wrapped))
	_, err := c.f.WriteAt(c.buf, 0)
	return c.wrapped, err
}

// Value returns the current counter value.
func (c *Int32Counter) Value() int32 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.wrapped
}

func (c *Int32Counter) load() (int32, error) {
	_, err := c.f.ReadAt(c.buf, 0)
	return int32(binary.LittleEndian.Uint32(c.buf)), err
}
