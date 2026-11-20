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
// must have exclusive write access to the file. If the file is empty, the counter
// starts at zero. Otherwise, it starts at the value stored in the file, and it returns
// io.ErrUnexpectedEOF if the file holds fewer than 4 bytes.
func NewInt32Counter(f ReaderAtWriterAtCloser) (*Int32Counter, error) {
	c := &Int32Counter{f: f, buf: make([]byte, 4)}
	n, err := f.ReadAt(c.buf, 0)
	if n < len(c.buf) && errors.Is(err, io.EOF) {
		if n == 0 {
			return c, nil
		}
		return nil, io.ErrUnexpectedEOF
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	c.wrapped = int32(binary.LittleEndian.Uint32(c.buf))
	return c, nil
}

// Add increments the counter by the provided delta. If the write to the file fails,
// the counter keeps its previous value.
func (c *Int32Counter) Add(delta int32) (int32, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	next := c.wrapped + delta
	binary.LittleEndian.PutUint32(c.buf, uint32(next))
	if _, err := c.f.WriteAt(c.buf, 0); err != nil {
		return 0, err
	}
	c.wrapped = next
	return next, nil
}

// Value returns the current counter value.
func (c *Int32Counter) Value() int32 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.wrapped
}
