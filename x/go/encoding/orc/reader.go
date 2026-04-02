// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package orc

import (
	"io"
	"math"

	"github.com/synnaxlabs/x/errors"
)

// ErrRecursionDepth is returned when decoding exceeds the maximum recursion depth.
var ErrRecursionDepth = errors.New("[orc] recursion depth exceeded")

// ErrExceedStringLen is returned when a string length prefix exceeds MaxStringLen.
var ErrExceedStringLen = errors.New("[orc] string length exceeded")

// ErrExceedCollectionLen is returned when a collection length prefix exceeds MaxCollectionLen.
var ErrExceedCollectionLen = errors.New("[orc] collection length exceeded")

// MaxStringLen is the maximum byte length of a string that can be decoded. In
// direct mode (ResetBytes), the backing slice provides a natural bound. In
// io.Reader mode, this limit prevents a corrupt length prefix from causing a
// massive allocation. Defaults to 128 MB.
var MaxStringLen uint32 = 128 << 20

// MaxCollectionLen is the maximum number of elements that can be decoded for a
// slice, map, or byte array. This prevents a corrupt length prefix from causing
// a massive allocation or an effectively infinite loop. Defaults to 10 million.
var MaxCollectionLen uint32 = 10_000_000

// Reader reads primitive data types using big-endian byte order. It supports two
// modes: direct byte-slice mode (via ResetBytes) for zero-copy decoding from
// in-memory data, and io.Reader mode (via Reset) for streaming. Direct mode
// avoids intermediate buffer allocations and io.ReadFull overhead.
type Reader struct {
	r     io.Reader
	buf   [8]byte
	depth int
	data  []byte
	pos   int
}

// NewReader creates a new Reader in io.Reader mode.
func NewReader(r io.Reader) *Reader {
	return &Reader{r: r}
}

// Reset resets the reader to use an io.Reader and clears the recursion depth.
func (r *Reader) Reset(reader io.Reader) {
	r.r = reader
	r.data = nil
	r.pos = 0
	r.depth = 0
}

// ResetBytes resets the reader to decode directly from a byte slice. This avoids
// the bytes.NewReader allocation and enables zero-copy reads for fixed-size types.
func (r *Reader) ResetBytes(data []byte) {
	r.data = data
	r.pos = 0
	r.r = nil
	r.depth = 0
}

// Uint8 reads a single byte.
func (r *Reader) Uint8() (uint8, error) {
	if r.data != nil {
		if r.pos >= len(r.data) {
			return 0, io.EOF
		}
		v := r.data[r.pos]
		r.pos++
		return v, nil
	}
	if _, err := io.ReadFull(r.r, r.buf[:1]); err != nil {
		return 0, err
	}
	return r.buf[0], nil
}

// Uint16 reads a 16-bit unsigned integer.
func (r *Reader) Uint16() (uint16, error) {
	if r.data != nil {
		if r.pos+2 > len(r.data) {
			return 0, r.shortDataErr()
		}
		v := order.Uint16(r.data[r.pos:])
		r.pos += 2
		return v, nil
	}
	if _, err := io.ReadFull(r.r, r.buf[:2]); err != nil {
		return 0, err
	}
	return order.Uint16(r.buf[:2]), nil
}

// Uint32 reads a 32-bit unsigned integer.
func (r *Reader) Uint32() (uint32, error) {
	if r.data != nil {
		if r.pos+4 > len(r.data) {
			return 0, r.shortDataErr()
		}
		v := order.Uint32(r.data[r.pos:])
		r.pos += 4
		return v, nil
	}
	if _, err := io.ReadFull(r.r, r.buf[:4]); err != nil {
		return 0, err
	}
	return order.Uint32(r.buf[:4]), nil
}

// Uint64 reads a 64-bit unsigned integer.
func (r *Reader) Uint64() (uint64, error) {
	if r.data != nil {
		if r.pos+8 > len(r.data) {
			return 0, r.shortDataErr()
		}
		v := order.Uint64(r.data[r.pos:])
		r.pos += 8
		return v, nil
	}
	if _, err := io.ReadFull(r.r, r.buf[:8]); err != nil {
		return 0, err
	}
	return order.Uint64(r.buf[:8]), nil
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
// In direct mode, slices into the backing data to avoid an intermediate
// buffer allocation. In io.Reader mode, the declared length is validated
// against MaxStringLen to prevent a corrupt prefix from causing a massive
// allocation.
func (r *Reader) String() (string, error) {
	n, err := r.Uint32()
	if err != nil {
		return "", err
	}
	if r.data != nil {
		end := r.pos + int(n)
		if end > len(r.data) {
			return "", io.ErrUnexpectedEOF
		}
		s := string(r.data[r.pos:end])
		r.pos = end
		return s, nil
	}
	if n > MaxStringLen {
		return "", errors.Wrapf(ErrExceedStringLen, "length %d exceeds maximum %d", n, MaxStringLen)
	}
	buf := make([]byte, n)
	if _, err = io.ReadFull(r.r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

// CollectionLen reads a uint32 element count and validates it against
// MaxCollectionLen. Use this instead of Uint32 when the value controls a
// slice, map, or byte array allocation.
func (r *Reader) CollectionLen() (uint32, error) {
	n, err := r.Uint32()
	if err != nil {
		return 0, err
	}
	if n > MaxCollectionLen {
		return 0, errors.Wrapf(ErrExceedCollectionLen, "length %d exceeds maximum %d", n, MaxCollectionLen)
	}
	return n, nil
}

// Read reads exactly len(data) bytes into the provided buffer.
func (r *Reader) Read(data []byte) (int, error) {
	if r.data != nil {
		n := len(data)
		if r.pos+n > len(r.data) {
			copied := copy(data, r.data[r.pos:])
			r.pos = len(r.data)
			if copied == 0 {
				return 0, io.EOF
			}
			return copied, io.ErrUnexpectedEOF
		}
		copy(data, r.data[r.pos:r.pos+n])
		r.pos += n
		return n, nil
	}
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

func (r *Reader) shortDataErr() error {
	if r.pos >= len(r.data) {
		return io.EOF
	}
	return io.ErrUnexpectedEOF
}
