// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"encoding/base64"
	"encoding/hex"
	"math"
	"strconv"

	"github.com/synnaxlabs/x/errors"
)

// AppendSampleText appends the text form of sample to dst and returns the extended
// buffer. sample must hold exactly one sample of data type dt. A float takes the
// shortest text that parses back to the same value, in exponent notation below 1e-6 or
// from 1e21. Infinities are Infinity and -Infinity. Integers are decimal, booleans are
// 0 or 1, a UUID takes its canonical form, and BytesT is base64. StringT and JSONT are
// appended as is. It returns an error when dt has no text form.
func AppendSampleText(dst []byte, dt DataType, sample []byte) ([]byte, error) {
	switch dt {
	case Float64T:
		v := math.Float64frombits(ByteOrder.Uint64(sample))
		return appendFloatText(dst, v, 64), nil
	case Float32T:
		v := math.Float32frombits(ByteOrder.Uint32(sample))
		return appendFloatText(dst, float64(v), 32), nil
	case Int64T, TimestampT:
		return strconv.AppendInt(dst, int64(ByteOrder.Uint64(sample)), 10), nil
	case Int32T:
		return strconv.AppendInt(dst, int64(int32(ByteOrder.Uint32(sample))), 10), nil
	case Int16T:
		return strconv.AppendInt(dst, int64(int16(ByteOrder.Uint16(sample))), 10), nil
	case Int8T:
		return strconv.AppendInt(dst, int64(int8(sample[0])), 10), nil
	case Uint64T:
		return strconv.AppendUint(dst, ByteOrder.Uint64(sample), 10), nil
	case Uint32T:
		return strconv.AppendUint(dst, uint64(ByteOrder.Uint32(sample)), 10), nil
	case Uint16T:
		return strconv.AppendUint(dst, uint64(ByteOrder.Uint16(sample)), 10), nil
	case Uint8T, BooleanT:
		return strconv.AppendUint(dst, uint64(sample[0]), 10), nil
	case UUIDT:
		dst = hex.AppendEncode(dst, sample[:4])
		dst = hex.AppendEncode(append(dst, '-'), sample[4:6])
		dst = hex.AppendEncode(append(dst, '-'), sample[6:8])
		dst = hex.AppendEncode(append(dst, '-'), sample[8:10])
		return hex.AppendEncode(append(dst, '-'), sample[10:16]), nil
	case StringT, JSONT:
		return append(dst, sample...), nil
	case BytesT:
		return base64.StdEncoding.AppendEncode(dst, sample), nil
	default:
		return nil, errors.Newf("data type %s has no text form", dt)
	}
}

func appendFloatText(dst []byte, v float64, bitSize int) []byte {
	if math.IsInf(v, 1) {
		return append(dst, "Infinity"...)
	}
	if math.IsInf(v, -1) {
		return append(dst, "-Infinity"...)
	}
	if a := math.Abs(v); a != 0 && (a < 1e-6 || a >= 1e21) {
		dst = strconv.AppendFloat(dst, v, 'e', -1, bitSize)
		// strconv pads a one-digit exponent with a zero.
		if n := len(dst); dst[n-2] == '0' && (dst[n-3] == '+' || dst[n-3] == '-') {
			dst[n-2] = dst[n-1]
			dst = dst[:n-1]
		}
		return dst
	}
	return strconv.AppendFloat(dst, v, 'f', -1, bitSize)
}
