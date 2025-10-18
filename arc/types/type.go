// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"slices"

	"github.com/synnaxlabs/x/maps"
	"github.com/synnaxlabs/x/telem"
)

type TypeKind int

const (
	KindInvalid TypeKind = iota
	KindU8
	KindU16
	KindU32
	KindU64
	KindI8
	KindI16
	KindI32
	KindI64
	KindF32
	KindF64
	KindString
	KindTimeStamp
	KindTimeSpan
	KindChan
	KindSeries
	KindTypeVariable
	KindNumericConstant
	KindFunction
)

func NewFunctionProperties() FunctionProperties {
	return FunctionProperties{Inputs: &Params{}, Outputs: &Params{}, Config: &Params{}}
}

type FunctionProperties struct {
	Inputs  *maps.Ordered[string, Type] `json:"inputs,omitempty" msgpack:"inputs,omitempty"`
	Outputs *maps.Ordered[string, Type] `json:"outputs,omitempty" msgpack:"outputs,omitempty"`
	Config  *maps.Ordered[string, Type] `json:"config,omitempty" msgpack:"config,omitempty"`
}

func (f FunctionProperties) Copy() FunctionProperties {
	return FunctionProperties{
		Inputs:  f.Inputs.Copy(),
		Outputs: f.Outputs.Copy(),
		Config:  f.Config.Copy(),
	}
}

// Type represents a type in the Arc type system using a tagged union approach.
type Type struct {
	Kind       TypeKind `json:"kind" msgpack:"kind"`
	ValueType  *Type    `json:"value_type,omitempty" msgpack:"value_type,omitempty"`
	Name       string   `json:"name,omitempty" msgpack:"name,omitempty"`
	Constraint *Type    `json:"constraint,omitempty" msgpack:"constraint,omitempty"`
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
		return "string"
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
	case KindTypeVariable:
		if t.Constraint != nil {
			return t.Name + ":" + t.Constraint.String()
		}
		return t.Name
	case KindNumericConstant:
		return "numeric"
	case KindFunction:
		return "function"
	default:
		return "invalid"
	}
}

func U8() Type        { return Type{Kind: KindU8} }
func U16() Type       { return Type{Kind: KindU16} }
func U32() Type       { return Type{Kind: KindU32} }
func U64() Type       { return Type{Kind: KindU64} }
func I8() Type        { return Type{Kind: KindI8} }
func I16() Type       { return Type{Kind: KindI16} }
func I32() Type       { return Type{Kind: KindI32} }
func I64() Type       { return Type{Kind: KindI64} }
func F32() Type       { return Type{Kind: KindF32} }
func F64() Type       { return Type{Kind: KindF64} }
func String() Type    { return Type{Kind: KindString} }
func TimeStamp() Type { return Type{Kind: KindTimeStamp} }
func TimeSpan() Type  { return Type{Kind: KindTimeSpan} }

func Chan(valueType Type) Type {
	return Type{Kind: KindChan, ValueType: &valueType}
}

func Series(valueType Type) Type {
	return Type{Kind: KindSeries, ValueType: &valueType}
}

func TypeVariable(name string, constraint *Type) Type {
	return Type{Kind: KindTypeVariable, Name: name, Constraint: constraint}
}

func NumericConstraint() Type {
	return Type{Kind: KindNumericConstant}
}

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

// Params is a type alias for ordered maps of types
type Params = maps.Ordered[string, Type]

// Type utility functions

func (t Type) IsNumeric() bool {
	if t.Kind == KindChan && t.ValueType != nil {
		return t.ValueType.IsNumeric()
	}
	if t.Kind == KindTypeVariable {
		if t.Constraint == nil {
			return false // Unconstrained type variable is not specifically numeric
		}
		if t.Constraint.Kind == KindNumericConstant {
			return true
		}
		return t.Constraint.IsNumeric()
	}
	switch t.Kind {
	case KindU8, KindU16, KindU32, KindU64,
		KindI8, KindI16, KindI32, KindI64,
		KindF32, KindF64:
		return true
	default:
		return false
	}
}

func (t Type) IsInteger() bool {
	switch t.Kind {
	case KindU8, KindU16, KindU32, KindU64,
		KindI8, KindI16, KindI32, KindI64:
		return true
	default:
		return false
	}
}

func (t Type) IsSignedInteger() bool {
	switch t.Kind {
	case KindI8, KindI16, KindI32, KindI64:
		return true
	default:
		return false
	}
}

func (t Type) IsUnsignedInteger() bool {
	switch t.Kind {
	case KindU8, KindU16, KindU32, KindU64:
		return true
	default:
		return false
	}
}

func (t Type) IsFloat() bool {
	switch t.Kind {
	case KindF32, KindF64:
		return true
	default:
		return false
	}
}

func (t Type) IsBool() bool {
	return t.Kind == KindU8
}

func (t *Type) IsValid() bool { return t.Kind != KindInvalid }

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
	if t.Kind == KindTypeVariable {
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
		if !equalNamedTypes(t.Inputs, v.Inputs) {
			return false
		}
		if !equalNamedTypes(t.Outputs, v.Outputs) {
			return false
		}
		return equalNamedTypes(t.Config, v.Config)
	}

	return true
}

// equalNamedTypes checks if two Params (maps.Ordered) are equal
func equalNamedTypes(a, b *Params) bool {
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

func (t Type) Is64Bit() bool {
	switch t.Kind {
	case KindI64, KindU64, KindTimeStamp, KindTimeSpan, KindF64:
		return true
	default:
		return false
	}
}

var (
	UnsignedIntegers = []Type{U8(), U16(), U32(), U64()}
	SignedIntegers   = []Type{I8(), I16(), I32(), I64()}
	Floats           = []Type{F32(), F64()}
	Numerics         = slices.Concat(UnsignedIntegers, SignedIntegers, Floats)
)

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

// NewTypeVariable creates a new type variable with the given name and constraint
func NewTypeVariable(name string, constraint *Type) Type {
	return TypeVariable(name, constraint)
}
