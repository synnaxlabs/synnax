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

	"github.com/synnaxlabs/x/telem"
)

// Number is the value of a float, integer, or unsigned read field.
type Number interface{ Float | Integer | Unsigned }

// Convert returns v as a To. ok is false when a To cannot hold v exactly: a float with
// a fraction or no finite value as an integer, a value past the range of To, or an
// integer past 2^53 with no equal float.
func Convert[To, From Number](v From) (to To, ok bool) {
	switch v := any(v).(type) {
	case Float:
		return convertFloat[To](float64(v))
	case Integer:
		switch any(to).(type) {
		case Float:
			f := float64(v)
			return To(f), f < -math.MinInt64 && int64(f) == int64(v)
		case Unsigned:
			return To(v), v >= 0
		}
		return To(v), true
	default:
		u := uint64(v.(Unsigned))
		switch any(to).(type) {
		case Float:
			f := float64(u)
			return To(f), f < 2*-math.MinInt64 && uint64(f) == u
		case Integer:
			return To(u), u <= math.MaxInt64
		}
		return To(u), true
	}
}

func convertFloat[To Number](f float64) (to To, ok bool) {
	if _, isFloat := any(to).(Float); isFloat {
		return To(f), true
	}
	if math.Trunc(f) != f {
		return to, false
	}
	if _, signed := any(to).(Integer); signed {
		return To(int64(f)), f >= math.MinInt64 && f < -math.MinInt64
	}
	return To(uint64(f)), f >= 0 && f < 2*-math.MinInt64
}

// Fits returns true when a channel of data type dt holds v exactly. A float fits a
// float32 channel when it is not finite or within its range, and an integer or unsigned
// fits a channel of its kind with the range to hold it. v fits no channel of another
// kind.
func Fits[T Number](v T, dt telem.DataType) bool {
	switch v := any(v).(type) {
	case Float:
		f := float64(v)
		switch dt {
		case telem.Float64T:
			return true
		case telem.Float32T:
			return math.IsNaN(f) || math.IsInf(f, 0) || math.Abs(f) <= math.MaxFloat32
		}
	case Integer:
		switch dt {
		case telem.Int8T:
			return v >= math.MinInt8 && v <= math.MaxInt8
		case telem.Int16T:
			return v >= math.MinInt16 && v <= math.MaxInt16
		case telem.Int32T:
			return v >= math.MinInt32 && v <= math.MaxInt32
		case telem.Int64T:
			return true
		}
	case Unsigned:
		switch dt {
		case telem.Uint8T:
			return v <= math.MaxUint8
		case telem.Uint16T:
			return v <= math.MaxUint16
		case telem.Uint32T:
			return v <= math.MaxUint32
		case telem.Uint64T:
			return true
		}
	}
	return false
}
