// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package orc implements the oracle binary encoding format. Oracle is the code
// generation framework used by Synnax to produce high-performance binary codecs for
// Go structs. The format is positional, untagged, and big-endian: fixed-size
// primitives are written directly, while variable-length types (strings, byte slices,
// nested records) are prefixed with a 4-byte big-endian length.
package orc

import (
	"encoding/binary"
	"math"
)

var order = binary.BigEndian

// Writer encodes primitive data types into a growable byte buffer using big-endian
// byte order. Methods append to the internal buffer, which grows as needed. Use
// Reset to reuse the buffer across operations and Resize to pre-allocate capacity.
type Writer struct {
	buf []byte
}

// NewWriter creates a new Writer with the given initial capacity.
func NewWriter(cap int) *Writer {
	return &Writer{buf: make([]byte, 0, cap)}
}

// Reset clears the buffer for reuse without releasing the underlying memory.
func (w *Writer) Reset() { w.buf = w.buf[:0] }

// Resize ensures the buffer has at least the given capacity. Existing data is
// preserved. If the current capacity is sufficient, Resize is a no-op.
func (w *Writer) Resize(size int) {
	if cap(w.buf) >= size {
		return
	}
	nb := make([]byte, len(w.buf), size)
	copy(nb, w.buf)
	w.buf = nb
}

// Uint8 appends a single byte.
func (w *Writer) Uint8(v uint8) { w.buf = append(w.buf, v) }

// Uint16 appends a 16-bit unsigned integer.
func (w *Writer) Uint16(v uint16) { w.buf = order.AppendUint16(w.buf, v) }

// Uint32 appends a 32-bit unsigned integer.
func (w *Writer) Uint32(v uint32) { w.buf = order.AppendUint32(w.buf, v) }

// Uint64 appends a 64-bit unsigned integer.
func (w *Writer) Uint64(v uint64) { w.buf = order.AppendUint64(w.buf, v) }

// Int8 appends a signed 8-bit integer.
func (w *Writer) Int8(v int8) { w.buf = append(w.buf, byte(v)) }

// Int16 appends a signed 16-bit integer.
func (w *Writer) Int16(v int16) { w.buf = order.AppendUint16(w.buf, uint16(v)) }

// Int32 appends a signed 32-bit integer.
func (w *Writer) Int32(v int32) { w.buf = order.AppendUint32(w.buf, uint32(v)) }

// Int64 appends a signed 64-bit integer.
func (w *Writer) Int64(v int64) { w.buf = order.AppendUint64(w.buf, uint64(v)) }

// Float32 appends a 32-bit float.
func (w *Writer) Float32(v float32) {
	w.buf = order.AppendUint32(w.buf, math.Float32bits(v))
}

// Float64 appends a 64-bit float.
func (w *Writer) Float64(v float64) {
	w.buf = order.AppendUint64(w.buf, math.Float64bits(v))
}

// Bool appends a single byte: 1 for true, 0 for false.
func (w *Writer) Bool(v bool) {
	if v {
		w.buf = append(w.buf, 1)
	} else {
		w.buf = append(w.buf, 0)
	}
}

// String appends a length-prefixed string (4-byte length + raw bytes). Panics
// if the string length exceeds math.MaxUint32.
func (w *Writer) String(v string) {
	if len(v) > math.MaxUint32 {
		panic("orc: string length exceeds maximum encodable size")
	}
	w.buf = order.AppendUint32(w.buf, uint32(len(v)))
	w.buf = append(w.buf, v...)
}

// Write appends raw bytes without any length prefix.
func (w *Writer) Write(data []byte) { w.buf = append(w.buf, data...) }

// WriteWithLen appends a length-prefixed byte slice (4-byte length + raw bytes).
// Panics if the slice length exceeds math.MaxUint32.
func (w *Writer) WriteWithLen(data []byte) {
	if len(data) > math.MaxUint32 {
		panic("orc: bytes length exceeds maximum encodable size")
	}
	w.buf = order.AppendUint32(w.buf, uint32(len(data)))
	w.buf = append(w.buf, data...)
}

// Bytes returns the encoded bytes. The returned slice is only valid until the
// next call to Reset or any write method that triggers growth.
func (w *Writer) Bytes() []byte { return w.buf }

// Len returns the number of bytes written so far.
func (w *Writer) Len() int { return len(w.buf) }

// Copy returns an owned copy of the encoded bytes.
func (w *Writer) Copy() []byte {
	out := make([]byte, len(w.buf))
	copy(out, w.buf)
	return out
}
