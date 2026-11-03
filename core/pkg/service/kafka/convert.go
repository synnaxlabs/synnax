// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"bytes"
	"math"
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/http"
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
	f, err := strconv.ParseFloat(string(text), 64)
	if err != nil {
		return number{}, errors.Wrapf(validate.ErrValidation, "invalid number %q", text)
	}
	return number{kind: floatNumber, f: f}, nil
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

func (n number) appendText(dst []byte) []byte {
	switch n.kind {
	case intNumber:
		return strconv.AppendInt(dst, n.i, 10)
	case uintNumber:
		return strconv.AppendUint(dst, n.u, 10)
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

// timeUnit returns the span one unit of a numeric time format counts.
func timeUnit(format http.TimeFormat) (telem.TimeSpan, error) {
	switch format {
	case http.TimeFormatUnixSec:
		return telem.Second, nil
	case http.TimeFormatUnixMs:
		return telem.Millisecond, nil
	case http.TimeFormatUnixUs:
		return telem.Microsecond, nil
	case http.TimeFormatUnixNs:
		return telem.Nanosecond, nil
	default:
		return 0, errors.Wrapf(validate.ErrValidation, "unknown time format %q", format)
	}
}

func numericTimestamp(n number, format http.TimeFormat) (telem.TimeStamp, error) {
	unit, err := timeUnit(format)
	if err != nil {
		return 0, err
	}
	switch n.kind {
	case intNumber:
		if n.i > math.MaxInt64/int64(unit) || n.i < math.MinInt64/int64(unit) {
			return 0, timestampOutOfRange(n, format)
		}
		return telem.TimeStamp(n.i) * telem.TimeStamp(unit), nil
	case uintNumber:
		// parseNumber only returns one past the range of an int64.
		return 0, timestampOutOfRange(n, format)
	default:
		// math.MaxInt64 rounds up to 2^63 as a float64, so the bound is exclusive.
		ns := n.f * float64(unit)
		if math.IsNaN(ns) || ns < math.MinInt64 || ns >= math.MaxInt64 {
			return 0, timestampOutOfRange(n, format)
		}
		return telem.TimeStamp(ns), nil
	}
}

func timestampOutOfRange(n number, format http.TimeFormat) error {
	return errors.Wrapf(
		validate.ErrValidation, "%v is out of range for a %s timestamp", n, format,
	)
}

// integerBounds returns the range of the integer data type dt. ok is false when dt
// is not an integer type.
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
	if n.kind == floatNumber && dt == telem.Float32T &&
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
		if n.f != math.Trunc(n.f) {
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
