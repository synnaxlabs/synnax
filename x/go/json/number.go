// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json

import (
	"bytes"
	"math"
	"strconv"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type numberKind uint8

const (
	intNumber numberKind = iota
	uintNumber
	floatNumber
)

// number is a sample or JSON number in the representation that loses no precision.
type number struct {
	kind numberKind
	i    int64
	u    uint64
	f    float64
	// fraction is true when the text of a float had a fraction that f may round away.
	fraction bool
}

// parseNumber converts the text of a JSON number into an int64 or uint64 when it is an
// integer that fits, and a float64 otherwise.
func parseNumber(text []byte) (number, error) {
	if !bytes.ContainsAny(text, ".eE") {
		if i, err := strconv.ParseInt(string(text), 10, 64); err == nil {
			return number{kind: intNumber, i: i}, nil
		}
		if u, err := strconv.ParseUint(string(text), 10, 64); err == nil {
			return number{kind: uintNumber, u: u}, nil
		}
	}
	d, ok := parseDecimal(text, 0)
	if !ok {
		return number{}, invalidNumber(text)
	}
	// A negative zero keeps its sign as a float.
	if !d.fraction && !d.overflow && d.magnitude != 0 {
		switch {
		case !d.negative && d.magnitude <= math.MaxInt64:
			return number{kind: intNumber, i: int64(d.magnitude)}, nil
		case !d.negative:
			return number{kind: uintNumber, u: d.magnitude}, nil
		case d.magnitude <= 1<<63:
			return number{kind: intNumber, i: int64(-d.magnitude)}, nil
		}
	}
	f, err := strconv.ParseFloat(string(text), 64)
	if err != nil {
		return number{}, invalidNumber(text)
	}
	return number{kind: floatNumber, f: f, fraction: d.fraction}, nil
}

func invalidNumber(text []byte) error {
	return errors.Wrapf(validate.ErrValidation, "invalid number %q", text)
}

// decimal is the magnitude of a decimal number times a power of ten, truncated toward
// zero.
type decimal struct {
	magnitude uint64
	negative  bool
	// fraction is true when the truncation dropped a digit other than zero.
	fraction bool
	// overflow is true when the magnitude does not fit a uint64.
	overflow bool
}

// maxExponent bounds the exponent parseDecimal reads. Any larger one overflows or
// truncates to zero.
const maxExponent = 1 << 20

// parseDecimal returns the decimal number text times 10^scale. ok is false when text is
// not a decimal number.
func parseDecimal(text []byte, scale int) (d decimal, ok bool) {
	if len(text) > 0 && (text[0] == '-' || text[0] == '+') {
		d.negative, text = text[0] == '-', text[1:]
	}
	exp := 0
	if i := bytes.IndexAny(text, "eE"); i >= 0 {
		if exp, ok = parseExponent(text[i+1:]); !ok {
			return decimal{}, false
		}
		text = text[:i]
	}
	whole, frac := text, []byte(nil)
	if i := bytes.IndexByte(text, '.'); i >= 0 {
		whole, frac = text[:i], text[i+1:]
	}
	if len(whole)+len(frac) == 0 || !isDigits(whole) || !isDigits(frac) {
		return decimal{}, false
	}
	// The value is the digits of whole and frac times 10^shift.
	shift := exp + scale - len(frac)
	count := len(whole) + len(frac)
	for i := range count {
		c := byte(0)
		if i < len(whole) {
			c = whole[i] - '0'
		} else {
			c = frac[i-len(whole)] - '0'
		}
		if i >= count+shift {
			d.fraction = d.fraction || c != 0
			continue
		}
		if d.magnitude > (math.MaxUint64-uint64(c))/10 {
			d.overflow = true
			return d, true
		}
		d.magnitude = d.magnitude*10 + uint64(c)
	}
	for i := 0; d.magnitude != 0 && i < shift; i++ {
		if d.magnitude > math.MaxUint64/10 {
			d.overflow = true
			return d, true
		}
		d.magnitude *= 10
	}
	return d, true
}

// parseExponent parses the signed decimal exponent of a number, clamped to
// ±maxExponent.
func parseExponent(text []byte) (int, bool) {
	negative := false
	if len(text) > 0 && (text[0] == '-' || text[0] == '+') {
		negative, text = text[0] == '-', text[1:]
	}
	if len(text) == 0 || !isDigits(text) {
		return 0, false
	}
	e := 0
	for _, c := range text {
		e = min(e*10+int(c-'0'), maxExponent)
	}
	if negative {
		return -e, true
	}
	return e, true
}

func isDigits(text []byte) bool {
	for _, c := range text {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func (n number) float() float64 {
	switch n.kind {
	case intNumber:
		return float64(n.i)
	case uintNumber:
		return float64(n.u)
	default:
		return n.f
	}
}

// appendText appends n as text: NaN, Infinity, or -Infinity when it is not finite.
func (n number) appendText(dst []byte) []byte {
	switch {
	case n.kind == intNumber:
		return strconv.AppendInt(dst, n.i, 10)
	case n.kind == uintNumber:
		return strconv.AppendUint(dst, n.u, 10)
	case math.IsInf(n.f, 1):
		return append(dst, "Infinity"...)
	case math.IsInf(n.f, -1):
		return append(dst, "-Infinity"...)
	default:
		return strconv.AppendFloat(dst, n.f, 'g', -1, 64)
	}
}

func (n number) String() string { return string(n.appendText(nil)) }

// bits returns n as a two's complement integer. Its low bytes are the encoding of n in
// any integer type n fits in, so check fits first.
func (n number) bits() uint64 {
	switch n.kind {
	case intNumber:
		return uint64(n.i)
	case uintNumber:
		return n.u
	default:
		if n.f < 0 {
			return uint64(int64(n.f))
		}
		return uint64(n.f)
	}
}

// integerBounds returns the range of the integer data type dt. ok is false when dt is
// not an integer type.
func integerBounds(dt telem.DataType) (lo int64, hi uint64, ok bool) {
	switch dt {
	case telem.Int8T:
		return math.MinInt8, math.MaxInt8, true
	case telem.Int16T:
		return math.MinInt16, math.MaxInt16, true
	case telem.Int32T:
		return math.MinInt32, math.MaxInt32, true
	case telem.Int64T:
		return math.MinInt64, math.MaxInt64, true
	case telem.Uint8T:
		return 0, math.MaxUint8, true
	case telem.Uint16T:
		return 0, math.MaxUint16, true
	case telem.Uint32T:
		return 0, math.MaxUint32, true
	case telem.Uint64T:
		return 0, math.MaxUint64, true
	default:
		return 0, 0, false
	}
}

// checkFits returns an error wrapping validate.ErrValidation when n does not convert
// into dt without wrapping, truncating, or overflowing to infinity.
func checkFits(n number, dt telem.DataType) error {
	if n.kind == floatNumber && dt == telem.Float32T && !math.IsInf(n.f, 0) &&
		math.Abs(n.f) > math.MaxFloat32 {
		return errors.Wrapf(validate.ErrValidation, "%s overflows %s", n, dt)
	}
	lo, hi, ok := integerBounds(dt)
	if !ok {
		return nil
	}
	var fits bool
	switch n.kind {
	case intNumber:
		fits = n.i >= lo && (n.i < 0 || uint64(n.i) <= hi)
	case uintNumber:
		fits = n.u <= hi
	default:
		if n.fraction || n.f != math.Trunc(n.f) {
			return errors.Wrapf(validate.ErrValidation, "%s is not an integer", n)
		}
		// float64(hi)+1 rounds to the first value past hi for every integer type.
		fits = n.f >= float64(lo) && n.f < float64(hi)+1
	}
	if !fits {
		return errors.Wrapf(validate.ErrValidation, "%s is out of range for %s", n, dt)
	}
	return nil
}

// appendNumber appends n as a sample of dt, which must be numeric or boolean. Check
// that n fits dt first.
func appendNumber(dst []byte, dt telem.DataType, n number) []byte {
	order := telem.ByteOrder
	switch dt {
	case telem.Float64T:
		return order.AppendUint64(dst, math.Float64bits(n.float()))
	case telem.Float32T:
		return order.AppendUint32(dst, math.Float32bits(float32(n.float())))
	case telem.BooleanT:
		if n.float() != 0 {
			return append(dst, 1)
		}
		return append(dst, 0)
	default:
		end := len(dst) + int(dt.Density())
		return order.AppendUint64(dst, n.bits())[:end]
	}
}

// sampleNumber reads sample, a sample of the numeric or boolean data type dt.
func sampleNumber(dt telem.DataType, sample []byte) number {
	order := telem.ByteOrder
	switch dt {
	case telem.Float64T:
		return number{kind: floatNumber, f: math.Float64frombits(order.Uint64(sample))}
	case telem.Float32T:
		f := float64(math.Float32frombits(order.Uint32(sample)))
		return number{kind: floatNumber, f: f}
	case telem.Int64T:
		return number{kind: intNumber, i: int64(order.Uint64(sample))}
	case telem.Int32T:
		return number{kind: intNumber, i: int64(int32(order.Uint32(sample)))}
	case telem.Int16T:
		return number{kind: intNumber, i: int64(int16(order.Uint16(sample)))}
	case telem.Int8T:
		return number{kind: intNumber, i: int64(int8(sample[0]))}
	case telem.Uint64T:
		return number{kind: uintNumber, u: order.Uint64(sample)}
	case telem.Uint32T:
		return number{kind: uintNumber, u: uint64(order.Uint32(sample))}
	case telem.Uint16T:
		return number{kind: uintNumber, u: uint64(order.Uint16(sample))}
	case telem.Uint8T:
		return number{kind: uintNumber, u: uint64(sample[0])}
	default:
		n := number{kind: intNumber}
		if sample[0] != 0 {
			n.i = 1
		}
		return n
	}
}

// appendJSONNumber appends n as a JSON number. A float takes the shortest form that
// round trips, and integers stay exact.
func appendJSONNumber(dst []byte, n number) ([]byte, error) {
	if n.kind != floatNumber {
		return n.appendText(dst), nil
	}
	if math.IsNaN(n.f) || math.IsInf(n.f, 0) {
		return nil, errors.Wrapf(validate.ErrValidation, "%s is not a JSON number", n)
	}
	format := byte('f')
	if abs := math.Abs(n.f); abs != 0 && (abs < 1e-6 || abs >= 1e21) {
		format = 'e'
	}
	dst = strconv.AppendFloat(dst, n.f, format, -1, 64)
	// A JSON number writes an exponent without padding: e-9, not e-09.
	if l := len(dst); format == 'e' && l >= 4 && dst[l-4] == 'e' && dst[l-3] == '-' &&
		dst[l-2] == '0' {
		dst[l-2] = dst[l-1]
		dst = dst[:l-1]
	}
	return dst, nil
}

// numeric reports whether dt holds numbers: an integer, float, or boolean type.
func numeric(dt telem.DataType) bool {
	switch dt {
	case telem.Float64T, telem.Float32T, telem.BooleanT,
		telem.Int64T, telem.Int32T, telem.Int16T, telem.Int8T,
		telem.Uint64T, telem.Uint32T, telem.Uint16T, telem.Uint8T:
		return true
	default:
		return false
	}
}
