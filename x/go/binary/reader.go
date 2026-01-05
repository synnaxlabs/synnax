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

// Reader makes it easy to read various primitive data types from binary using a given
// byte order. Unlike encoding/binary.Read, this avoids reflection overhead by using
// direct byte operations.
type Reader struct {
	r         io.Reader
	byteOrder binary.ByteOrder
	buf       [8]byte // reusable buffer for reading primitives
}

// NewReader creates a new reader with the given io.Reader and byte order.
func NewReader(r io.Reader, order binary.ByteOrder) *Reader {
	return &Reader{r: r, byteOrder: order}
}

// Reset resets the reader to use a new io.Reader.
func (r *Reader) Reset(reader io.Reader) {
	r.r = reader
}

// Uint8 reads a Uint8 from the reader.
func (r *Reader) Uint8() (uint8, error) {
	if _, err := io.ReadFull(r.r, r.buf[:1]); err != nil {
		return 0, err
	}
	return r.buf[0], nil
}

// Uint32 reads a Uint32 from the reader.
func (r *Reader) Uint32() (uint32, error) {
	if _, err := io.ReadFull(r.r, r.buf[:4]); err != nil {
		return 0, err
	}
	return r.byteOrder.Uint32(r.buf[:4]), nil
}

// Uint64 reads a Uint64 from the reader.
func (r *Reader) Uint64() (uint64, error) {
	if _, err := io.ReadFull(r.r, r.buf[:8]); err != nil {
		return 0, err
	}
	return r.byteOrder.Uint64(r.buf[:8]), nil
}

// Read reads len(data) bytes into data. Returns the number of bytes read.
func (r *Reader) Read(data []byte) (int, error) {
	return io.ReadFull(r.r, data)
}
