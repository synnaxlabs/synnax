// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec

import (
	"math"
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/validate"
)

// bitRange reads the raw bits of one binary field in a payload. Two ranges are equal
// when they read the same bits in the same order.
type bitRange struct {
	// segments holds one entry for each byte, as a 64-bit field can span 9.
	segments [9]segment
	count    int
	width    int
	// end is the number of payload bytes that the field needs.
	end int
}

// segment is the part of a field that lies in one payload byte.
type segment struct {
	byte int
	// shift is the position of the lowest bit of the segment in its byte.
	shift uint8
	// mask selects the bits of the segment after the shift.
	mask uint8
	// dest is the position of the lowest bit of the segment in the raw value.
	dest uint8
}

// newBitRange returns the bit range of f, whose start bit is numbered as in a DBC file.
// It returns an error wrapping validate.ErrValidation when the bit length is not from 1
// to 64, or the byte order is unknown.
func newBitRange(f icd.BinaryBaseField) (bitRange, error) {
	width := int(f.BitLength)
	if width < 1 || width > 64 {
		return bitRange{}, errors.Wrapf(
			validate.ErrValidation, "bit length must be from 1 to 64, got %d", width,
		)
	}
	big := f.ByteOrder == icd.ByteOrderBigEndian
	if !big && f.ByteOrder != icd.ByteOrderLittleEndian {
		return bitRange{}, errors.Wrapf(
			validate.ErrValidation, "unknown byte order %s", f.ByteOrder,
		)
	}
	// Number the bits of either order as one run: a little-endian position counts up
	// from the least significant bit of byte 0, and a big-endian position counts up
	// from the most significant bit of byte 0.
	start := int(f.StartBit)
	first := start
	if big {
		first = start/8*8 + 7 - start%8
	}
	last := first + width - 1
	r := bitRange{width: width, end: last/8 + 1}
	for b := first / 8; b <= last/8; b++ {
		lo, hi := max(first, b*8), min(last, b*8+7)
		s := segment{byte: b, mask: uint8(1<<(hi-lo+1) - 1)}
		if big {
			s.shift, s.dest = uint8(7-(hi-b*8)), uint8(last-hi)
		} else {
			s.shift, s.dest = uint8(lo-b*8), uint8(lo-first)
		}
		r.segments[r.count] = s
		r.count++
	}
	return r, nil
}

// read returns the raw bits of the field. p must hold at least r.end bytes.
func (r *bitRange) read(p []byte) uint64 {
	var raw uint64
	for _, s := range r.segments[:r.count] {
		raw |= uint64(p[s.byte]>>s.shift&s.mask) << s.dest
	}
	return raw
}

// readSigned returns the raw bits of the field as a two's complement integer of its
// width. p must hold at least r.end bytes.
func (r *bitRange) readSigned(p []byte) int64 {
	shift := 64 - r.width
	return int64(r.read(p)<<shift) >> shift
}

// write writes the low bits of raw into the field and leaves every other bit of p as it
// was. p must hold at least r.end bytes.
func (r *bitRange) write(p []byte, raw uint64) {
	for _, s := range r.segments[:r.count] {
		kept := p[s.byte] &^ (s.mask << s.shift)
		p[s.byte] = kept | (uint8(raw>>s.dest)&s.mask)<<s.shift
	}
}

// encoding is how the raw bits of a binary field hold its value.
type encoding uint8

const (
	encodingUnsigned encoding = iota
	encodingSigned
	encodingFloat32
	encodingFloat64
)

// encodingOf returns how the raw bits of f hold its value. It returns an error wrapping
// validate.ErrValidation when f is a float field that is not 32 or 64 bits.
func encodingOf(f icd.BinaryField) (encoding, error) {
	switch f.Variant.(type) {
	case icd.UnsignedBinaryField:
		return encodingUnsigned, nil
	case icd.IntegerBinaryField:
		return encodingSigned, nil
	case icd.FloatBinaryField:
		switch n := f.Base().BitLength; n {
		case 32:
			return encodingFloat32, nil
		case 64:
			return encodingFloat64, nil
		default:
			return 0, errors.Wrapf(
				validate.ErrValidation,
				"a float field must be 32 or 64 bits, not %d",
				n,
			)
		}
	default:
		panic(errors.Newf("codec: unknown binary field %T", f.Variant))
	}
}

// kind returns the payload kind of the values of a field of encoding e.
func (e encoding) kind() payload.Kind {
	switch e {
	case encodingSigned:
		return payload.KindInteger
	case encodingUnsigned:
		return payload.KindUnsigned
	case encodingFloat32, encodingFloat64:
		return payload.KindFloat
	}
	panic(errors.Newf("codec: unknown encoding %d", e))
}

// value returns raw, the bits of a field of encoding e, as a number before the field's
// scale and offset.
func (e encoding) value(raw uint64) float64 {
	switch e {
	case encodingUnsigned:
		return float64(raw)
	case encodingSigned:
		return float64(int64(raw))
	case encodingFloat32:
		return float64(math.Float32frombits(uint32(raw)))
	case encodingFloat64:
		return math.Float64frombits(raw)
	}
	panic(errors.Newf("codec: unknown encoding %d", e))
}

// defaultRaw returns the raw bits of the default of f, of encoding e, and false when f
// has none. It returns an error wrapping validate.ErrValidation when the default
// overflows float32 or is out of range for the field's bits.
func defaultRaw(f icd.BinaryField, e encoding) (uint64, bool, error) {
	n := f.Base().BitLength
	outOfRange := func(v string) error {
		return errors.Wrapf(
			validate.ErrValidation, "%s is out of range for %d bits", v, n,
		)
	}
	switch v := f.Variant.(type) {
	case icd.FloatBinaryField:
		if v.Default == nil {
			return 0, false, nil
		}
		d := float64(*v.Default)
		if e == encodingFloat64 {
			return math.Float64bits(d), true, nil
		}
		if math.Abs(d) > math.MaxFloat32 {
			return 0, false, errors.Wrapf(
				validate.ErrValidation,
				"%s overflows float32",
				strconv.FormatFloat(d, 'g', -1, 64),
			)
		}
		return uint64(math.Float32bits(float32(d))), true, nil
	case icd.IntegerBinaryField:
		if v.Default == nil {
			return 0, false, nil
		}
		i, shift := int64(*v.Default), 64-n
		if i<<shift>>shift != i {
			return 0, false, outOfRange(strconv.FormatInt(i, 10))
		}
		return uint64(i), true, nil
	case icd.UnsignedBinaryField:
		if v.Default == nil {
			return 0, false, nil
		}
		u := uint64(*v.Default)
		if n < 64 && u>>n != 0 {
			return 0, false, outOfRange(strconv.FormatUint(u, 10))
		}
		return u, true, nil
	default:
		panic(errors.Newf("codec: unknown binary field %T", f.Variant))
	}
}
