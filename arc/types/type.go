// Copyright 2025 Synnax Labs, Inc.
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
//   - Temporal types: timestamp, timespan
//   - String type: str
//
// Compound Types:
//   - chan T: Channel type wrapping value type T
//   - series T: Series type wrapping value type T
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
//	seriesType := types.Series(types.TimeStamp())
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

	"github.com/synnaxlabs/x/maps"
	"github.com/synnaxlabs/x/telem"
)

// TypeKind represents the different categories of types in the Arc type system.
// It is used as a discriminator in the Type tagged union.
type TypeKind int

const (
	// KindInvalid represents an invalid or uninitialized type.
	KindInvalid TypeKind = iota

	// KindU8 is an 8-bit unsigned integer type.
	KindU8
	// KindU16 is a 16-bit unsigned integer type.
	KindU16
	// KindU32 is a 32-bit unsigned integer type.
	KindU32
	// KindU64 is a 64-bit unsigned integer type.
	KindU64

	// KindI8 is an 8-bit signed integer type.
	KindI8
	// KindI16 is a 16-bit signed integer type.
	KindI16
	// KindI32 is a 32-bit signed integer type.
	KindI32
	// KindI64 is a 64-bit signed integer type.
	KindI64

	// KindF32 is a 32-bit floating-point type.
	KindF32
	// KindF64 is a 64-bit floating-point type.
	KindF64

	// KindString is a UTF-8 string type.
	KindString

	// KindTimeStamp represents an absolute point in time.
	KindTimeStamp
	// KindTimeSpan represents a duration or time difference.
	KindTimeSpan

	// KindChan is a channel type (requires ValueType).
	KindChan
	// KindSeries is a series/array type (requires ValueType).
	KindSeries

	// KindVariable is a generic type variable (requires Name, optional Constraint).
	KindVariable

	// KindNumericConstant is a constraint for any numeric type (integers or floats).
	KindNumericConstant
	// KindIntegerConstant is a constraint for any integer type (signed or unsigned).
	KindIntegerConstant
	// KindFloatConstant is a constraint for any floating-point type.
	KindFloatConstant

	// KindFunction is a function type (requires Inputs, Outputs, optional Config).
	KindFunction
)

// NewFunctionProperties creates a new FunctionProperties with empty Inputs, Outputs, and Config.
func NewFunctionProperties() FunctionProperties {
	return FunctionProperties{
		Inputs:         &Params{},
		Outputs:        &Params{},
		Config:         &Params{},
		InputDefaults:  make(map[string]any),
		ConfigDefaults: make(map[string]any),
	}
}

// Params are named, ordered parameters for a function.
type Params = maps.Ordered[string, Type]

// FunctionProperties holds the inputs, outputs, and configuration parameters for function
// types.
type FunctionProperties struct {
	// Inputs are the input parameters for the function.
	Inputs *Params `json:"inputs,omitempty" msgpack:"inputs,omitempty"`
	// Outputs are the output/return values for the function.
	Outputs *Params `json:"outputs,omitempty" msgpack:"outputs,omitempty"`
	// Config are the configuration parameters for the function.
	Config *Params `json:"config,omitempty" msgpack:"config,omitempty"`
	// InputDefaults stores default values for optional input parameters.
	// The map key is the parameter name, and the value is the default literal value.
	InputDefaults map[string]any `json:"input_defaults,omitempty" msgpack:"input_defaults,omitempty"`
	// ConfigDefaults stores default values for optional config parameters.
	// The map key is the parameter name, and the value is the default literal value.
	ConfigDefaults map[string]any `json:"config_defaults,omitempty" msgpack:"config_defaults,omitempty"`
}

// Copy creates a deep copy of the function properties.
func (f FunctionProperties) Copy() FunctionProperties {
	// Deep copy default value maps
	inputDefaults := make(map[string]any, len(f.InputDefaults))
	for k, v := range f.InputDefaults {
		inputDefaults[k] = v
	}
	configDefaults := make(map[string]any, len(f.ConfigDefaults))
	for k, v := range f.ConfigDefaults {
		configDefaults[k] = v
	}

	return FunctionProperties{
		Inputs:         f.Inputs.Copy(),
		Outputs:        f.Outputs.Copy(),
		Config:         f.Config.Copy(),
		InputDefaults:  inputDefaults,
		ConfigDefaults: configDefaults,
	}
}

// Type represents a type in the Arc type system using a tagged union.
type Type struct {
	// Kind is the discriminator that determines which type this represents.
	Kind TypeKind `json:"kind" msgpack:"kind"`
	// ValueType is the element type for compound types (chan, series).
	ValueType *Type `json:"value_type,omitempty" msgpack:"value_type,omitempty"`
	// Name is the identifier for type variables.
	Name string `json:"name,omitempty" msgpack:"name,omitempty"`
	// Constraint is the optional constraint for type variables.
	Constraint *Type `json:"constraint,omitempty" msgpack:"constraint,omitempty"`
	// FunctionProperties contains inputs, outputs, and config for function types.
	FunctionProperties
}

// String returns the string representation of the type
func (t Type) String() string {
	switch t.Kind {
	case KindU8:
		return "u8"
	case KindU16:
		return "u16"
	case KindU32:
		return "u32"
	case KindU64:
		return "u64"
	case KindI8:
		return "i8"
	case KindI16:
		return "i16"
	case KindI32:
		return "i32"
	case KindI64:
		return "i64"
	case KindF32:
		return "f32"
	case KindF64:
		return "f64"
	case KindString:
		return "str"
	case KindTimeStamp:
		return "timestamp"
	case KindTimeSpan:
		return "timespan"
	case KindChan:
		if t.ValueType != nil {
			return "chan " + t.ValueType.String()
		}
		return "chan <invalid>"
	case KindSeries:
		if t.ValueType != nil {
			return "series " + t.ValueType.String()
		}
		return "series <invalid>"
	case KindVariable:
		if t.Constraint != nil {
			return t.Name + ":" + t.Constraint.String()
		}
		return t.Name
	case KindNumericConstant:
		return "numeric"
	case KindIntegerConstant:
		return "integer"
	case KindFloatConstant:
		return "float"
	case KindFunction:
		return "function"
	default:
		return "invalid"
	}
}

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

// TimeStamp returns an absolute point in time type.
func TimeStamp() Type { return Type{Kind: KindTimeStamp} }

// TimeSpan returns a duration or time difference type.
func TimeSpan() Type { return Type{Kind: KindTimeSpan} }

// Chan returns a channel type wrapping the given value type.
func Chan(valueType Type) Type {
	return Type{Kind: KindChan, ValueType: &valueType}
}

// Series returns a series/array type wrapping the given value type.
func Series(valueType Type) Type { return Type{Kind: KindSeries, ValueType: &valueType} }

// Variable returns a generic type parameter with optional constraint.
func Variable(name string, constraint *Type) Type {
	return Type{Kind: KindVariable, Name: name, Constraint: constraint}
}

// NumericConstraint returns a constraint accepting any numeric type (integers or floats).
func NumericConstraint() Type { return Type{Kind: KindNumericConstant} }

// IntegerConstraint returns a constraint accepting any integer type (signed or unsigned).
func IntegerConstraint() Type { return Type{Kind: KindIntegerConstant} }

// FloatConstraint returns a constraint accepting any floating-point type.
func FloatConstraint() Type { return Type{Kind: KindFloatConstant} }

// Function creates a function type with the given inputs, outputs, and optional config
func Function(props FunctionProperties) Type {
	if props.Inputs == nil {
		props.Inputs = &Params{}
	}
	if props.Outputs == nil {
		props.Outputs = &Params{}
	}
	if props.Config == nil {
		props.Config = &Params{}
	}
	return Type{Kind: KindFunction, FunctionProperties: props}
}

// IsNumeric returns true if the type is a numeric type (integer or float).
// For channel types, it checks the value type. For type variables, it checks
// if the constraint is a numeric constraint or if the constraint itself is numeric.
func (t Type) IsNumeric() bool {
	if t.Kind == KindChan && t.ValueType != nil {
		return t.ValueType.IsNumeric()
	}
	if t.Kind == KindVariable {
		if t.Constraint == nil {
			return false // Unconstrained type variable is not specifically numeric
		}
		if t.Constraint.Kind == KindNumericConstant ||
			t.Constraint.Kind == KindIntegerConstant ||
			t.Constraint.Kind == KindFloatConstant {
			return true
		}
		return t.Constraint.IsNumeric()
	}
	switch t.Kind {
	case KindU8, KindU16, KindU32, KindU64,
		KindI8, KindI16, KindI32, KindI64,
		KindF32, KindF64,
		KindNumericConstant, KindIntegerConstant, KindFloatConstant:
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
func (t Type) IsBool() bool {
	return t.Kind == KindU8
}

// Unwrap returns the value type of chan/series types, or the type itself otherwise.
// This eliminates the need for repeated unwrapping logic throughout the codebase.
func (t Type) Unwrap() Type {
	if (t.Kind == KindChan || t.Kind == KindSeries) && t.ValueType != nil {
		return *t.ValueType
	}
	return t
}

// IsValid returns true if the type is not invalid or uninitialized.
func (t *Type) IsValid() bool { return t.Kind != KindInvalid }

// Equal compares two types for structural equality.
// For compound types (chan, series), it recursively compares value types.
// For type variables, it compares names and constraints.
// For function types, it compares inputs, outputs, and config parameters.
func Equal(t Type, v Type) bool {
	if t.Kind != v.Kind {
		return false
	}

	// For compound types, recursively check value types
	if t.Kind == KindChan || t.Kind == KindSeries {
		if t.ValueType == nil && v.ValueType == nil {
			return true
		}
		if t.ValueType == nil || v.ValueType == nil {
			return false
		}
		return Equal(*t.ValueType, *v.ValueType)
	}

	// For type variables, check name and constraint
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

	// For function types, check inputs, outputs, and config
	if t.Kind == KindFunction {
		if !paramsEqual(t.Inputs, v.Inputs) {
			return false
		}
		if !paramsEqual(t.Outputs, v.Outputs) {
			return false
		}
		return paramsEqual(t.Config, v.Config)
	}

	return true
}

// paramsEqual checks if two Params (maps.Ordered) are equal
func paramsEqual(a, b *Params) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Count() != b.Count() {
		return false
	}
	for k, vA := range a.Iter() {
		vB, ok := b.Get(k)
		if !ok {
			return false
		}
		if !Equal(vA, vB) {
			return false
		}
	}
	return true
}

// Is64Bit returns true if the type uses 64-bit representation.
func (t Type) Is64Bit() bool {
	switch t.Kind {
	case KindI64, KindU64, KindTimeStamp, KindTimeSpan, KindF64:
		return true
	default:
		return false
	}
}

// Density returns the size in bytes of the primitive type.
// Panics if the type is not a fixed-size primitive (e.g., compound types, type variables, string).
func (t Type) Density() int {
	switch t.Kind {
	case KindU8, KindI8:
		return 1
	case KindU16, KindI16:
		return 2
	case KindU32, KindI32, KindF32:
		return 4
	case KindU64, KindI64, KindF64, KindTimeStamp, KindTimeSpan:
		return 8
	default:
		panic("Density: type is not a fixed-size primitive: " + t.String())
	}
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
// Returns an invalid type for unknown telemetry types.
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
// Returns telem.UnknownT for Arc types that don't have a telemetry equivalent.
func ToTelem(t Type) telem.DataType {
	switch t.Kind {
	case KindU8:
		return telem.Uint8T
	case KindU16:
		return telem.Uint16T
	case KindU32:
		return telem.Uint32T
	case KindU64:
		return telem.Uint64T
	case KindTimeStamp:
		return telem.TimeStampT
	case KindTimeSpan:
		return telem.TimeStampT
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
