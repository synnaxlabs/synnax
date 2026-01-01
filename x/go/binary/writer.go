// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package binary

import (
	"encoding/binary"
)

// Writer makes it easy to writer various primitive data types to binary using a given
// byte order.
type Writer struct {
	offset    int
	byteOrder binary.ByteOrder
	buf       []byte
}

// NewWriter creates a new writer with the given size, starting offset, and byte order.
func NewWriter(size int, order binary.ByteOrder) *Writer {
	return &Writer{buf: make([]byte, size), byteOrder: order}
}

// Reset resets the writer to reuse the underlying buffer.
func (w *Writer) Reset() { w.offset = 0 }

// Resize resizes the writer's buffer to the given capacity.
func (w *Writer) Resize(size int) {
	if size < len(w.buf) {
		w.buf = w.buf[0:size]
		if size < w.offset {
			w.offset = size
		}
	} else if size > len(w.buf) {
		w.buf = append(w.buf, make([]byte, size-len(w.buf))...)
	}
}

// Uint8 writes a new Uint8 to the buffer. If the buffer is already at capacity, Uint8
// returns 0. Otherwise, Uint8 returns 1.
func (w *Writer) Uint8(value uint8) int {
	if w.offset+1 > len(w.buf) {
		return 0
	}
	w.buf[w.offset] = value
	w.offset += 1
	return 1
}

// Uint32 writes a new Uint32 to the buffer. If the buffer is at capacity, Uint32 returns 0.
// Otherwise, Uint32 returns 4.
func (w *Writer) Uint32(value uint32) int {
	if w.offset+4 > len(w.buf) {
		return 0
	}
	w.byteOrder.PutUint32(w.buf[w.offset:w.offset+4], value)
	w.offset += 4
	return 4
}

// Uint64 writes a new Uint64 to the buffer. If the buffer is at capacity, Uint64 returns 0.
// Otherwise, Uint64 returns 8.
func (w *Writer) Uint64(value uint64) int {
	if w.offset+8 > len(w.buf) {
		return 0
	}
	w.byteOrder.PutUint64(w.buf[w.offset:w.offset+8], value)
	w.offset += 8
	return 8
}

// Write writes the given data to the buffer. Returns the number of bytes written,
// which is the min of len(data) and the remaining available buffer capacity.
func (w *Writer) Write(data []byte) int {
	count := copy(w.buf[w.offset:], data)
	w.offset += count
	return count
}

// Bytes returns the underlying binary buffer.
func (w *Writer) Bytes() []byte { return w.buf[:w.offset] }
