// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package types implements the Arc type system, providing type representations,
// type checking utilities, and conversions between Arc types and telemetry types.
//
// # Type Categories
//
// Primitive Types:
//   - Integer types: u8, u16, u32, u64, i8, i16, i32, i64
//   - Floating-point types: f32, f64
//   - String type: str
//
// Compound Types:
//   - chan T: Channel type wrapping value type T
//   - series T: Series type wrapping value type T
//
// Unit Types:
//   - Any numeric type can have unit metadata attached (e.g., f64 psi, i64 ns)
//   - Units include dimensions (length, mass, time, etc.) and scale factors
//   - Dimensional analysis is performed at compile time
//
// Generic Types:
//   - Type variables with optional constraints (numeric, integer, float)
//
// Function Types:
//   - Functions with named inputs, outputs, and configuration parameters
//
// # Usage
//
// Creating types:
//
//	i32Type := types.I32()
//	chanType := types.Chan(types.F64())
//	seriesType := types.Series(types.I32())
//
// Creating unit types:
//
//	pressureType := types.Type{Kind: types.KindF64, Unit: &types.Unit{
//		Dimensions: types.DimPressure,
//		Scale:      6894.76,
//		Name:       "psi",
//	}}
//
// Type checking:
//
//	if t.IsNumeric() { ... }
//	if t.IsInteger() { ... }
//	if types.Equal(t1, t2) { ... }
//
// Unwrapping compound types:
//
//	chanType := types.Chan(types.I32())
//	innerType := chanType.Unwrap() // returns I32()
//
// Converting to/from telemetry types:
//
//	arcType := types.FromTelem(telem.Float64T)
//	telemType := types.ToTelem(types.F64())
package types

import (
	"slices"

	"github.com/synnaxlabs/x/telem"
)

// U8 returns an 8-bit unsigned integer type.
func U8() Type { return Type{Kind: KindU8} }

// U16 returns a 16-bit unsigned integer type.
func U16() Type { return Type{Kind: KindU16} }

// U32 returns a 32-bit unsigned integer type.
func U32() Type { return Type{Kind: KindU32} }

// U64 returns a 64-bit unsigned integer type.
func U64() Type { return Type{Kind: KindU64} }

// I8 returns an 8-bit signed integer type.
func I8() Type { return Type{Kind: KindI8} }

// I16 returns a 16-bit signed integer type.
func I16() Type { return Type{Kind: KindI16} }

// I32 returns a 32-bit signed integer type.
func I32() Type { return Type{Kind: KindI32} }

// I64 returns a 64-bit signed integer type.
func I64() Type { return Type{Kind: KindI64} }

// F32 returns a 32-bit floating-point type.
func F32() Type { return Type{Kind: KindF32} }

// F64 returns a 64-bit floating-point type.
func F64() Type { return Type{Kind: KindF64} }

// String returns a UTF-8 string type.
func String() Type { return Type{Kind: KindString} }

// TimeStamp returns an i64 type with nanosecond time units.
func TimeStamp() Type {
	return Type{
		Kind: KindI64,
		Unit: &Unit{
			Dimensions: DimTime,
			Scale:      1,
			Name:       "ns",
		},
	}
}

// TimeSpan returns an i64 type with nanosecond time units.
func TimeSpan() Type {
	return Type{
		Kind: KindI64,
		Unit: &Unit{
			Dimensions: DimTime,
			Scale:      1,
			Name:       "ns",
		},
	}
}

// Series returns a series/array type wrapping the given value type.
func Series(valueType Type) Type { return Type{Kind: KindSeries, Elem: &valueType} }

// VarRef returns a variable-reference type. Name holds the key of the
// variable's node; the variable's values have valueType.
func VarRef(valueType Type, nodeKey string) Type {
	return Type{Kind: KindVarRef, Elem: &valueType, Name: nodeKey}
}

// Variable returns a generic type parameter with optional constraint.
func Variable(name string, constraint *Type) Type {
	return Type{Kind: KindVariable, Name: name, Constraint: constraint}
}

// NumericConstraint returns a constraint accepting any numeric type.
func NumericConstraint() Type { return Type{Kind: KindNumericConstant} }

// IntegerConstraint returns a constraint accepting any integer type.
func IntegerConstraint() Type { return Type{Kind: KindIntegerConstant} }

// FloatConstraint returns a constraint accepting any floating-point type.
func FloatConstraint() Type { return Type{Kind: KindFloatConstant} }

// ExactIntegerFloatConstraint returns a constraint for float literals that represent
// exact integers (like 5.0, 0.0).
func ExactIntegerFloatConstraint() Type { return Type{Kind: KindExactIntegerFloatConstant} }

// Sequence returns a sequence (state machine) type.
func Sequence() Type { return Type{Kind: KindSequence} }

// Stage returns a stage (within a sequence) type.
func Stage() Type { return Type{Kind: KindStage} }

// Function creates a function type with the given inputs and outputs.
func Function(props FunctionProperties) Type {
	return Type{Kind: KindFunction, FunctionProperties: props}
}

// StructuralMatch returns true if both types have the same wrapper structure.
func StructuralMatch(t1, t2 Type) bool {
	return (t1.Kind == KindSeries) == (t2.Kind == KindSeries) &&
		(t1.Kind == KindChan) == (t2.Kind == KindChan)
}

// Equal compares two types for strict structural equality, including units.
func Equal(t Type, v Type) bool {
	if t.Kind != v.Kind {
		return false
	}
	if t.Kind == KindChan || t.Kind == KindSeries || t.Kind == KindVarRef {
		if t.Elem == nil && v.Elem == nil {
			return true
		}
		if t.Elem == nil || v.Elem == nil {
			return false
		}
		return Equal(*t.Elem, *v.Elem)
	}
	if t.Kind == KindVariable {
		if t.Name != v.Name {
			return false
		}
		if t.Constraint == nil && v.Constraint == nil {
			return true
		}
		if t.Constraint == nil || v.Constraint == nil {
			return false
		}
		return Equal(*t.Constraint, *v.Constraint)
	}
	if t.Kind == KindFunction {
		if !paramsEqual(t.Inputs, v.Inputs) {
			return false
		}
		return paramsEqual(t.Outputs, v.Outputs)
	}
	if t.Unit == nil && v.Unit == nil {
		return true
	}
	if t.Unit == nil || v.Unit == nil {
		return false
	}
	return t.Unit.Equal(*v.Unit)
}

func paramsEqual(a, b Params) bool {
	if len(a) != len(b) {
		return false
	}
	for i, pA := range a {
		pB := b[i]
		if pB.Name != pA.Name {
			return false
		}
		if pB.Value != pA.Value {
			return false
		}
		if !Equal(pA.Type, pB.Type) {
			return false
		}
	}
	return true
}

var (
	// UnsignedIntegers contains all unsigned integer types.
	UnsignedIntegers = []Type{U8(), U16(), U32(), U64()}
	// SignedIntegers contains all signed integer types.
	SignedIntegers = []Type{I8(), I16(), I32(), I64()}
	// Floats contains all floating-point types.
	Floats = []Type{F32(), F64()}
	// Numerics contains all numeric types (unsigned, signed, and floating-point).
	Numerics = slices.Concat(UnsignedIntegers, SignedIntegers, Floats)
)

// FromTelem converts a telemetry data type to an Arc type.
func FromTelem(t telem.DataType) Type {
	switch t {
	case telem.Uint8T:
		return U8()
	case telem.Uint16T:
		return U16()
	case telem.Uint32T:
		return U32()
	case telem.Uint64T:
		return U64()
	case telem.Int8T:
		return I8()
	case telem.Int16T:
		return I16()
	case telem.Int32T:
		return I32()
	case telem.Int64T:
		return I64()
	case telem.Float32T:
		return F32()
	case telem.Float64T:
		return F64()
	case telem.StringT, telem.JSONT, telem.UUIDT:
		return String()
	case telem.TimeStampT:
		return TimeStamp()
	default:
		return Type{Kind: KindInvalid}
	}
}

// ToTelem converts an Arc type to a telemetry data type.
func ToTelem(t Type) telem.DataType {
	if t.Kind == KindI64 && t.Unit != nil &&
		t.Unit.Dimensions.Equal(DimTime) && t.Unit.Name == "ns" {
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
