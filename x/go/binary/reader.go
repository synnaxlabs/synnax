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
	"errors"
	"io"
	"math"
)

// ErrRecursionDepth is returned when decoding exceeds the maximum recursion depth.
var ErrRecursionDepth = errors.New("binary: recursion depth exceeded")

// Reader reads primitive data types from an io.Reader using a given byte order.
// Unlike encoding/binary.Read, this avoids reflection overhead by using direct
// byte operations. For decoding from a []byte, wrap it with bytes.NewReader.
type Reader struct {
	r         io.Reader
	byteOrder binary.ByteOrder
	buf       [8]byte
	depth     int
}

// NewReader creates a new Reader with the given io.Reader and byte order.
func NewReader(r io.Reader, order binary.ByteOrder) *Reader {
	return &Reader{r: r, byteOrder: order}
}

// Reset resets the reader to use a new io.Reader and clears the recursion depth.
func (r *Reader) Reset(reader io.Reader) {
	r.r = reader
	r.depth = 0
}

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

// Int8 reads a signed 8-bit integer.
func (r *Reader) Int8() (int8, error) {
	v, err := r.Uint8()
	return int8(v), err
}

// Int16 reads a signed 16-bit integer.
func (r *Reader) Int16() (int16, error) {
	v, err := r.Uint16()
	return int16(v), err
}

// Int32 reads a signed 32-bit integer.
func (r *Reader) Int32() (int32, error) {
	v, err := r.Uint32()
	return int32(v), err
}

// Int64 reads a signed 64-bit integer.
func (r *Reader) Int64() (int64, error) {
	v, err := r.Uint64()
	return int64(v), err
}

// Float32 reads a 32-bit float.
func (r *Reader) Float32() (float32, error) {
	v, err := r.Uint32()
	return math.Float32frombits(v), err
}

// Float64 reads a 64-bit float.
func (r *Reader) Float64() (float64, error) {
	v, err := r.Uint64()
	return math.Float64frombits(v), err
}

// Bool reads a single byte and returns true if non-zero.
func (r *Reader) Bool() (bool, error) {
	v, err := r.Uint8()
	return v != 0, err
}

// String reads a length-prefixed string (4-byte length + raw bytes).
func (r *Reader) String() (string, error) {
	n, err := r.Uint32()
	if err != nil {
		return "", err
	}
	buf := make([]byte, n)
	if _, err = io.ReadFull(r.r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// Read reads exactly len(data) bytes. Returns the number of bytes read.
func (r *Reader) Read(data []byte) (int, error) {
	return io.ReadFull(r.r, data)
}

// PushDepth increments the recursion depth counter and returns an error if
// the limit is exceeded. Use this when decoding recursive types.
func (r *Reader) PushDepth(limit int) error {
	r.depth++
	if r.depth > limit {
		r.depth--
		return ErrRecursionDepth
	}
	return nil
}

// PopDepth decrements the recursion depth counter.
func (r *Reader) PopDepth() { r.depth-- }
