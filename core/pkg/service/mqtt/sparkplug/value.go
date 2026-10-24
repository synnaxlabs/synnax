// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sparkplug

import (
	"math"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Value is the value of a metric: an IntValue, a UintValue, a FloatValue, a BoolValue,
// a StringValue, or a DateTimeValue.
type Value interface{ value() }

// IntValue is the value of an Int8, Int16, Int32, or Int64 tag.
type IntValue int64

// UintValue is the value of a UInt8, UInt16, UInt32, or UInt64 tag.
type UintValue uint64

// FloatValue is the value of a Float or Double tag.
type FloatValue float64

// BoolValue is the value of a Boolean tag.
type BoolValue bool

// StringValue is the value of a String, Text, or UUID tag.
type StringValue string

// DateTimeValue is the value of a DateTime tag.
type DateTimeValue telem.TimeStamp

func (IntValue) value()      {}
func (UintValue) value()     {}
func (FloatValue) value()    {}
func (BoolValue) value()     {}
func (StringValue) value()   {}
func (DateTimeValue) value() {}

// String returns the time as telem.TimeStamp formats it.
func (v DateTimeValue) String() string { return telem.TimeStamp(v).String() }

// errValue is returned when a value cannot be sent as a data type.
var errValue = errors.Wrap(validate.ErrValidation, "value does not fit the data type")

func misfit(v Value, dt any) error { return errors.Wrapf(errValue, "%v as %s", v, dt) }

type numberKind uint8

const (
	signed numberKind = iota
	unsigned
	float
)

// number is a numeric Value in the form that holds it exactly.
type number struct {
	kind numberKind
	i    int64
	u    uint64
	f    float64
}

// toNumber is the one type switch over a Value: a target of one family reads a
// StringValue or a DateTimeValue with a type assertion. A BoolValue is 1 or 0. ok is
// false for a value that is not a number or a Boolean.
func toNumber(v Value) (n number, ok bool) {
	switch v := v.(type) {
	case IntValue:
		return number{kind: signed, i: int64(v)}, true
	case UintValue:
		return number{kind: unsigned, u: uint64(v)}, true
	case FloatValue:
		return number{kind: float, f: float64(v)}, true
	case BoolValue:
		if v {
			return number{kind: unsigned, u: 1}, true
		}
		return number{kind: unsigned}, true
	}
	return number{}, false
}

// int64 returns n as an int64. ok is false for a fraction, a NaN, and a number out of
// the int64 range.
func (n number) int64() (int64, bool) {
	switch n.kind {
	case signed:
		return n.i, true
	case unsigned:
		return int64(n.u), n.u <= math.MaxInt64
	}
	return int64(n.f), n.f == math.Trunc(n.f) && n.f >= -(1<<63) && n.f < 1<<63
}

// uint64 returns n as a uint64. ok is false for a fraction, a NaN, and a number out of
// the uint64 range.
func (n number) uint64() (uint64, bool) {
	switch n.kind {
	case signed:
		return uint64(n.i), n.i >= 0
	case unsigned:
		return n.u, true
	}
	return uint64(n.f), n.f == math.Trunc(n.f) && n.f >= 0 && n.f < 1<<64
}

func (n number) float64() float64 {
	switch n.kind {
	case signed:
		return float64(n.i)
	case unsigned:
		return float64(n.u)
	}
	return n.f
}

// float32 returns n as a float32. ok is false for a finite number past the float32
// range, which would become an infinity.
func (n number) float32() (float32, bool) {
	f := n.float64()
	return float32(f), math.Abs(f) <= math.MaxFloat32 || math.IsInf(f, 0) ||
		math.IsNaN(f)
}

func fitsInt(i int64, bits int) bool {
	high := i >> (bits - 1)
	return high == 0 || high == -1
}

func fitsUint(u uint64, bits int) bool { return bits == 64 || u>>bits == 0 }

// Compatible reports whether a channel of type ch can give a tag of type t its samples
// and take the values of the tag. A numeric channel matches a numeric or a Boolean tag,
// a string channel matches a String, Text, or UUID tag, and a timestamp channel matches
// a DateTime tag. No other channel matches a tag.
func Compatible(ch telem.DataType, t DataType) bool {
	switch t {
	case Int8, Int16, Int32, Int64, UInt8, UInt16, UInt32, UInt64, Float, Double,
		Boolean:
		switch ch {
		case telem.Int8T, telem.Int16T, telem.Int32T, telem.Int64T,
			telem.Uint8T, telem.Uint16T, telem.Uint32T, telem.Uint64T,
			telem.Float32T, telem.Float64T:
			return true
		}
	case String, Text, UUID:
		return ch == telem.StringT
	case DateTime:
		return ch == telem.TimestampT
	}
	return false
}

// ValueAt returns sample i of series as a Value. It panics for a data type that
// Compatible matches to no tag.
func ValueAt(series telem.Series, i int) Value {
	switch series.DataType {
	case telem.Int8T:
		return IntValue(series.ValueAt[int8](i))
	case telem.Int16T:
		return IntValue(series.ValueAt[int16](i))
	case telem.Int32T:
		return IntValue(series.ValueAt[int32](i))
	case telem.Int64T:
		return IntValue(series.ValueAt[int64](i))
	case telem.Uint8T:
		return UintValue(series.ValueAt[uint8](i))
	case telem.Uint16T:
		return UintValue(series.ValueAt[uint16](i))
	case telem.Uint32T:
		return UintValue(series.ValueAt[uint32](i))
	case telem.Uint64T:
		return UintValue(series.ValueAt[uint64](i))
	case telem.Float32T:
		return FloatValue(series.ValueAt[float32](i))
	case telem.Float64T:
		return FloatValue(series.ValueAt[float64](i))
	case telem.StringT:
		return StringValue(series.At(i))
	case telem.TimestampT:
		return DateTimeValue(series.ValueAt[telem.TimeStamp](i))
	}
	panic("sparkplug: no tag takes samples of " + series.DataType)
}

// AppendSample appends v to dst as one sample of a channel of type dt, in the encoding
// of telem.Series. It returns an error that wraps validate.ErrValidation for a value
// that the pairs of Compatible do not allow, and for a number that dt cannot hold.
func AppendSample(dst []byte, dt telem.DataType, v Value) ([]byte, error) {
	n, isNumber := toNumber(v)
	bits := int(dt.Density()) * 8
	switch dt {
	case telem.Int8T, telem.Int16T, telem.Int32T, telem.Int64T:
		if i, ok := n.int64(); isNumber && ok && fitsInt(i, bits) {
			return appendUint(dst, bits, uint64(i)), nil
		}
	case telem.Uint8T, telem.Uint16T, telem.Uint32T, telem.Uint64T:
		if u, ok := n.uint64(); isNumber && ok && fitsUint(u, bits) {
			return appendUint(dst, bits, u), nil
		}
	case telem.Float32T:
		if f, ok := n.float32(); isNumber && ok {
			return telem.ByteOrder.AppendUint32(dst, math.Float32bits(f)), nil
		}
	case telem.Float64T:
		if isNumber {
			return telem.ByteOrder.AppendUint64(dst, math.Float64bits(n.float64())), nil
		}
	case telem.StringT:
		if s, ok := v.(StringValue); ok {
			dst = telem.ByteOrder.AppendUint32(dst, uint32(len(s)))
			return append(dst, s...), nil
		}
	case telem.TimestampT:
		if ts, ok := v.(DateTimeValue); ok {
			return telem.ByteOrder.AppendUint64(dst, uint64(ts)), nil
		}
	}
	return dst, misfit(v, dt)
}

// appendUint appends the low bits of v. Two's complement makes this correct for a
// signed value that fits.
func appendUint(dst []byte, bits int, v uint64) []byte {
	switch bits {
	case 64:
		return telem.ByteOrder.AppendUint64(dst, v)
	case 32:
		return telem.ByteOrder.AppendUint32(dst, uint32(v))
	case 16:
		return telem.ByteOrder.AppendUint16(dst, uint16(v))
	}
	return append(dst, byte(v))
}
