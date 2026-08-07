// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"fmt"
	"math"

	"github.com/synnaxlabs/x/telem"
)

// IntegerMaxValue returns the maximum value representable by this integer type. Panics
// if the type is not an integer type.
//
// Note: For U64, returns math.MaxInt64 for comparison safety since MaxUint64 cannot be
// represented in int64.
func (t Type) IntegerMaxValue() int64 {
	if !t.IsInteger() {
		panic(
			fmt.Sprintf(
				"[type.IntegerMaxValue] attempted to call on non-integer %s",
				t,
			),
		)
	}
	switch t.Kind {
	case KindI8:
		return math.MaxInt8
	case KindI16:
		return math.MaxInt16
	case KindI32:
		return math.MaxInt32
	case KindI64:
		return math.MaxInt64
	case KindU8:
		return math.MaxUint8
	case KindU16:
		return math.MaxUint16
	case KindU32:
		return math.MaxUint32
	case KindU64:
		return math.MaxInt64
	default:
		return math.MaxInt64
	}
}

// IntegerMinValue returns the minimum value representable by this integer type. Panics
// if the type is not an integer type. Returns 0 for unsigned integer types.
func (t Type) IntegerMinValue() int64 {
	if !t.IsInteger() {
		panic(
			fmt.Sprintf(
				"[types.IntegerMinValue] attempted to call on non-integer %s",
				t,
			),
		)
	}
	switch t.Kind {
	case KindI8:
		return math.MinInt8
	case KindI16:
		return math.MinInt16
	case KindI32:
		return math.MinInt32
	case KindI64:
		return math.MinInt64
	case KindU8, KindU16, KindU32, KindU64:
		return 0
	default:
		return math.MinInt64
	}
}

// String returns the string representation of the type
func (t Type) String() string {
	var base string
	switch t.Kind {
	case KindU8:
		base = "u8"
	case KindU16:
		base = "u16"
	case KindU32:
		base = "u32"
	case KindU64:
		base = "u64"
	case KindI8:
		base = "i8"
	case KindI16:
		base = "i16"
	case KindI32:
		base = "i32"
	case KindI64:
		base = "i64"
	case KindF32:
		base = "f32"
	case KindF64:
		base = "f64"
	case KindString:
		return "str"
	case KindChan:
		if t.Elem != nil {
			return "chan " + t.Elem.String()
		}
		return "chan <invalid>"
	case KindSeries:
		if t.Elem != nil {
			return "series " + t.Elem.String()
		}
		return "series <invalid>"
	case KindVarRef:
		if t.Elem != nil {
			return "var " + t.Elem.String()
		}
		return "var <invalid>"
	case KindVariable:
		if t.Constraint != nil {
			return t.Constraint.String()
		}
		return "unknown"
	case KindNumericConstant:
		return "numeric"
	case KindIntegerConstant:
		return "integer"
	case KindFloatConstant:
		return "float"
	case KindExactIntegerFloatConstant:
		return "exact integer float"
	case KindFunction:
		return "function"
	case KindSequence:
		return "sequence"
	case KindStage:
		return "stage"
	default:
		return "invalid"
	}

	if t.Unit != nil && t.Unit.Name != "" {
		return base + " " + t.Unit.Name
	}
	return base
}

// IsNumeric returns true if the type is a numeric type (integer or float).
func (t Type) IsNumeric() bool {
	unwrapped := t.Unwrap()
	if unwrapped.Kind == KindVariable {
		if unwrapped.Constraint == nil {
			return false
		}
		if unwrapped.Constraint.Kind == KindNumericConstant ||
			unwrapped.Constraint.Kind == KindIntegerConstant ||
			unwrapped.Constraint.Kind == KindFloatConstant ||
			unwrapped.Constraint.Kind == KindExactIntegerFloatConstant {
			return true
		}
		return unwrapped.Constraint.IsNumeric()
	}
	switch unwrapped.Kind {
	case KindU8, KindU16, KindU32, KindU64,
		KindI8, KindI16, KindI32, KindI64,
		KindF32, KindF64,
		KindNumericConstant, KindIntegerConstant, KindFloatConstant, KindExactIntegerFloatConstant:
		return true
	default:
		return false
	}
}

// IsInteger returns true if the type is an integer type (signed or unsigned).
func (t Type) IsInteger() bool {
	switch t.Kind {
	case KindU8, KindU16, KindU32, KindU64,
		KindI8, KindI16, KindI32, KindI64:
		return true
	default:
		return false
	}
}

// IsSignedInteger returns true if the type is a signed integer type.
func (t Type) IsSignedInteger() bool {
	switch t.Kind {
	case KindI8, KindI16, KindI32, KindI64:
		return true
	default:
		return false
	}
}

// IsSigned returns true if the type is a signed type (integer or float).
func (t Type) IsSigned() bool { return t.IsSignedInteger() || t.IsFloat() }

// IsUnsignedInteger returns true if the type is an unsigned integer type.
func (t Type) IsUnsignedInteger() bool {
	switch t.Kind {
	case KindU8, KindU16, KindU32, KindU64:
		return true
	default:
		return false
	}
}

// IsFloat returns true if the type is a floating-point type.
func (t Type) IsFloat() bool {
	switch t.Kind {
	case KindF32, KindF64:
		return true
	default:
		return false
	}
}

// IsBool returns true if the type is a boolean type (u8).
func (t Type) IsBool() bool { return t.Unwrap().Kind == KindU8 }

// Unwrap returns the value type of chan/series types, or the type itself otherwise.
func (t Type) Unwrap() Type {
	if (t.Kind == KindChan || t.Kind == KindSeries) && t.Elem != nil {
		return *t.Elem
	}
	return t
}

// UnwrapChan returns the effective value type when a type is used as a value. Channels
// are implicitly read: chan<T> -> T Series stay as series: series<T> -> series<T>
func (t Type) UnwrapChan() Type {
	if t.Kind == KindChan && t.Elem != nil {
		return *t.Elem
	}
	return t
}

// IsValid returns true if the type is not invalid or uninitialized.
func (t *Type) IsValid() bool { return t.Kind != KindInvalid }

// Is64Bit returns true if the type uses 64-bit representation.
func (t Type) Is64Bit() bool {
	switch t.Kind {
	case KindI64, KindU64, KindF64:
		return true
	default:
		return false
	}
}

// Density returns the size in bytes of the primitive type.
func (t Type) Density() int {
	switch t.Kind {
	case KindU8, KindI8:
		return 1
	case KindU16, KindI16:
		return 2
	case KindU32, KindI32, KindF32:
		return 4
	case KindU64, KindI64, KindF64:
		return 8
	default:
		panic("Density: type is not a fixed-size primitive: " + t.String())
	}
}

// ToTelem converts the Arc type to a telemetry data type.
func (t Type) ToTelem() telem.DataType {
	if t.Kind == KindI64 && t.Unit != nil &&
		t.Unit.Dimensions.Equal(Dimensions{Time: 1}) && t.Unit.Name == "ns" {
		return telem.TimeStampT
	}
	switch t.Kind {
	case KindU8:
		return telem.Uint8T
	case KindU16:
		return telem.Uint16T
	case KindU32:
		return telem.Uint32T
	case KindU64:
		return telem.Uint64T
	case KindF32:
		return telem.Float32T
	case KindF64:
		return telem.Float64T
	case KindString:
		return telem.StringT
	case KindI8:
		return telem.Int8T
	case KindI16:
		return telem.Int16T
	case KindI32:
		return telem.Int32T
	case KindI64:
		return telem.Int64T
	default:
		return telem.UnknownT
	}
}
