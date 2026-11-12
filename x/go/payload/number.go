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

// IntegerNumber returns v as a Number of the integer kind.
func IntegerNumber(v Integer) Number { return Number{kind: KindInteger, i: int64(v)} }

// UnsignedNumber returns v as a Number of the unsigned kind.
func UnsignedNumber(v Unsigned) Number {
	return Number{kind: KindUnsigned, u: uint64(v)}
}

// Kind returns the kind of n.
func (n Number) Kind() Kind { return n.kind }

// Float returns n, which must be of the float kind.
func (n Number) Float() Float { return Float(n.f) }

// Integer returns n, which must be of the integer kind.
func (n Number) Integer() Integer { return Integer(n.i) }

// Unsigned returns n, which must be of the unsigned kind.
func (n Number) Unsigned() Unsigned { return Unsigned(n.u) }

// String implements fmt.Stringer.
func (n Number) String() string {
	switch n.kind {
	case KindInteger:
		return strconv.FormatInt(n.i, 10)
	case KindUnsigned:
		return strconv.FormatUint(n.u, 10)
	}
	return strconv.FormatFloat(n.f, 'g', -1, 64)
}

// exact returns n as an exact big.Float, or nil when n is NaN or infinite.
func (n Number) exact() *big.Float {
	switch n.kind {
	case KindInteger:
		return new(big.Float).SetInt64(n.i)
	case KindUnsigned:
		return new(big.Float).SetUint64(n.u)
	}
	if math.IsNaN(n.f) || math.IsInf(n.f, 0) {
		return nil
	}
	return new(big.Float).SetFloat64(n.f)
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
	default:
		return Number{}, false
	}
	return out, acc == big.Exact
}

// Fits reports whether a channel of data type dt holds n. An integer data type holds n
// exactly. A float data type holds the nearest value to n, and a NaN or an infinity.
func (n Number) Fits(dt telem.DataType) bool {
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
	}
	return false
}

func (n Number) float64() float64 {
	switch n.kind {
	case KindInteger:
		return float64(n.i)
	case KindUnsigned:
		return float64(n.u)
	}
	return n.f
}

// Append appends n to dst as one sample of the numeric data type dt, which must hold n.
func (n Number) Append(dst []byte, dt telem.DataType) []byte {
	order := telem.ByteOrder
	switch dt {
	case telem.Float64T:
		return order.AppendUint64(dst, math.Float64bits(n.float64()))
	case telem.Float32T:
		return order.AppendUint32(dst, math.Float32bits(float32(n.float64())))
	}
	k, _ := KindOf(dt)
	v, _ := n.To(k)
	bits := v.u
	if k == KindInteger {
		bits = uint64(v.i)
	}
	switch dt.Density() {
	case 1:
		return append(dst, byte(bits))
	case 2:
		return order.AppendUint16(dst, uint16(bits))
	case 4:
		return order.AppendUint32(dst, uint32(bits))
	}
	return order.AppendUint64(dst, bits)
}
