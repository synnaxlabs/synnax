// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload

import (
	"math"
	"math/big"
	"strconv"

	"github.com/synnaxlabs/x/telem"
)

// Number is a value of a numeric kind: float, integer, or unsigned.
type Number struct {
	kind Kind
	f    float64
	i    int64
	u    uint64
}

// FloatNumber returns v as a Number of the float kind.
func FloatNumber(v Float) Number { return Number{kind: KindFloat, f: float64(v)} }

// integerNumber returns v as a Number of the integer kind.
func integerNumber(v Integer) Number { return Number{kind: KindInteger, i: int64(v)} }

// unsignedNumber returns v as a Number of the unsigned kind.
func unsignedNumber(v Unsigned) Number {
	return Number{kind: KindUnsigned, u: uint64(v)}
}

// float returns n, which must be of the float kind.
func (n Number) float() Float { return Float(n.f) }

// integer returns n, which must be of the integer kind.
func (n Number) integer() Integer { return Integer(n.i) }

// unsigned returns n, which must be of the unsigned kind.
func (n Number) unsigned() Unsigned { return Unsigned(n.u) }

// boolean returns whether n is not 0. ok is false when n is NaN, which has no boolean.
func (n Number) boolean() (v, ok bool) {
	switch n.kind {
	case KindInteger:
		return n.i != 0, true
	case KindUnsigned:
		return n.u != 0, true
	case KindFloat:
		return n.f != 0, !math.IsNaN(n.f)
	case KindString, KindBoolean, KindTimestamp, KindJSON, KindUUID, KindBytes:
	}
	panic("payload: a number has kind " + string(n.kind))
}

// String implements fmt.Stringer.
func (n Number) String() string {
	switch n.kind {
	case KindInteger:
		return strconv.FormatInt(n.i, 10)
	case KindUnsigned:
		return strconv.FormatUint(n.u, 10)
	case KindFloat:
		return strconv.FormatFloat(n.f, 'g', -1, 64)
	case KindString, KindBoolean, KindTimestamp, KindJSON, KindUUID, KindBytes:
	}
	panic("payload: a number has kind " + string(n.kind))
}

// exact returns n as an exact big.Float, or nil when n is NaN or infinite.
func (n Number) exact() *big.Float {
	switch n.kind {
	case KindInteger:
		return new(big.Float).SetInt64(n.i)
	case KindUnsigned:
		return new(big.Float).SetUint64(n.u)
	case KindFloat:
		if math.IsNaN(n.f) || math.IsInf(n.f, 0) {
			return nil
		}
		return new(big.Float).SetFloat64(n.f)
	case KindString, KindBoolean, KindTimestamp, KindJSON, KindUUID, KindBytes:
	}
	panic("payload: a number has kind " + string(n.kind))
}

// To returns n as a value of the numeric kind k. ok is false when k does not hold n
// exactly, such as a fraction as an integer or a NaN as anything but a float.
func (n Number) To(k Kind) (out Number, ok bool) {
	if n.kind == k {
		return n, true
	}
	b := n.exact()
	if b == nil {
		return Number{}, false
	}
	var acc big.Accuracy
	out.kind = k
	switch k {
	case KindFloat:
		out.f, acc = b.Float64()
	case KindInteger:
		if !b.IsInt() {
			return Number{}, false
		}
		out.i, acc = b.Int64()
	case KindUnsigned:
		if !b.IsInt() {
			return Number{}, false
		}
		out.u, acc = b.Uint64()
	case KindString, KindBoolean, KindTimestamp, KindJSON, KindUUID, KindBytes:
		fallthrough
	default:
		return Number{}, false
	}
	return out, acc == big.Exact
}

// fits reports whether a channel of data type dt holds n. An integer data type holds n
// exactly. A float data type holds the nearest value to n, and a NaN or an infinity.
func (n Number) fits(dt telem.DataType) bool {
	k, _ := KindOf(dt)
	switch k {
	case KindFloat:
		if dt == telem.Float64T || n.kind != KindFloat {
			return true
		}
		return math.IsNaN(n.f) || math.IsInf(n.f, 0) ||
			math.Abs(n.f) <= math.MaxFloat32
	case KindInteger:
		v, ok := n.To(k)
		bits := 8 * int(dt.Density())
		return ok && (bits == 64 ||
			(v.i >= -1<<(bits-1) && v.i <= 1<<(bits-1)-1))
	case KindUnsigned:
		v, ok := n.To(k)
		bits := 8 * int(dt.Density())
		return ok && (bits == 64 || v.u < 1<<bits)
	case KindString, KindBoolean, KindTimestamp, KindJSON, KindUUID, KindBytes:
	}
	return false
}

func (n Number) float64() float64 {
	switch n.kind {
	case KindInteger:
		return float64(n.i)
	case KindUnsigned:
		return float64(n.u)
	case KindFloat:
		return n.f
	case KindString, KindBoolean, KindTimestamp, KindJSON, KindUUID, KindBytes:
	}
	panic("payload: a number has kind " + string(n.kind))
}

// appendTo appends n to dst as one sample of the numeric data type dt, which must
// hold n.
func (n Number) appendTo(dst []byte, dt telem.DataType) []byte {
	order := telem.ByteOrder
	var bits uint64
	switch dt {
	case telem.Float64T:
		return order.AppendUint64(dst, math.Float64bits(n.float64()))
	case telem.Float32T:
		return order.AppendUint32(dst, math.Float32bits(float32(n.float64())))
	case telem.Int64T, telem.Int32T, telem.Int16T, telem.Int8T:
		v, _ := n.To(KindInteger)
		bits = uint64(v.i)
	case telem.Uint64T, telem.Uint32T, telem.Uint16T, telem.Uint8T:
		v, _ := n.To(KindUnsigned)
		bits = v.u
	case telem.UnknownT, telem.TimestampT, telem.UUIDT, telem.StringT, telem.BytesT,
		telem.JSONT, telem.BooleanT:
		panic("payload: a number has no samples of " + string(dt))
	}
	switch dt.Density() {
	case telem.Bit8:
		return append(dst, byte(bits))
	case telem.Bit16:
		return order.AppendUint16(dst, uint16(bits))
	case telem.Bit32:
		return order.AppendUint32(dst, uint32(bits))
	case telem.Bit64:
		return order.AppendUint64(dst, bits)
	case telem.UnknownDensity, telem.Bit128:
	}
	panic("payload: a number has no samples of " + string(dt))
}
