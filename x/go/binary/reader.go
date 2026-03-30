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
	"io"
)

// Reader reads primitive data types from an io.Reader using a given byte order.
type Reader struct {
	r         io.Reader
	byteOrder binary.ByteOrder
	buf       [8]byte
}

// NewReader creates a new Reader with the given byte order.
func NewReader(r io.Reader, order binary.ByteOrder) *Reader {
	return &Reader{r: r, byteOrder: order}
}

// Reset resets the reader to use a new io.Reader.
func (r *Reader) Reset(reader io.Reader) { r.r = reader }

// Uint8 reads a single byte.
func (r *Reader) Uint8() (uint8, error) {
	if _, err := io.ReadFull(r.r, r.buf[:1]); err != nil {
		return 0, err
	}
	return r.buf[0], nil
}

// Uint16 reads a 16-bit unsigned integer.
func (r *Reader) Uint16() (uint16, error) {
	if _, err := io.ReadFull(r.r, r.buf[:2]); err != nil {
		return 0, err
	}
	return r.byteOrder.Uint16(r.buf[:2]), nil
}

// Uint32 reads a 32-bit unsigned integer.
func (r *Reader) Uint32() (uint32, error) {
	if _, err := io.ReadFull(r.r, r.buf[:4]); err != nil {
		return 0, err
	}
	return r.byteOrder.Uint32(r.buf[:4]), nil
}

// Uint64 reads a 64-bit unsigned integer.
func (r *Reader) Uint64() (uint64, error) {
	if _, err := io.ReadFull(r.r, r.buf[:8]); err != nil {
		return 0, err
	}
	return r.byteOrder.Uint64(r.buf[:8]), nil
}

// Read reads exactly len(data) bytes into the provided buffer.
func (r *Reader) Read(data []byte) (int, error) {
	return io.ReadFull(r.r, data)
}
