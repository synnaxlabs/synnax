// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir

import (
	"fmt"
	"slices"

	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type Type interface {
	fmt.Stringer
}

type U8 struct{}

func (u U8) String() string { return "u8" }

type U16 struct{}

func (u U16) String() string { return "u16" }

type U32 struct{}

func (u U32) String() string { return "u32" }

type U64 struct{}

func (u U64) String() string { return "u64" }

type I8 struct{}

func (i I8) String() string { return "i8" }

type I16 struct{}

func (i I16) String() string { return "i16" }

type I32 struct{}

func (i I32) String() string { return "i32" }

type I64 struct{}

func (i I64) String() string { return "i64" }

type F32 struct{}

func (f F32) String() string { return "f32" }

type F64 struct{}

func (f F64) String() string { return "f64" }

type String struct{}

func (s String) String() string { return "string" }

type Series struct {
	ValueType Type
}

func (s Series) String() string { return "series " + s.ValueType.String() }

type Body struct {
	Raw string
	AST parser.IBlockContext
}

type Function struct {
	Key    string
	Params NamedTypes
	Return Type
	Body   Body
}

func (f Function) String() string { return "function" }

type Chan struct {
	ValueType Type
}

func (c Chan) String() string { return "chan " + c.ValueType.String() }

type TimeSpan struct{}

func (t TimeSpan) String() string { return "timespan" }

type TimeStamp struct{}

func (t TimeStamp) String() string { return "timestamp" }

// TypeVariable represents a polymorphic type that will be resolved during analysis.
// It allows stages to accept and return generic types with constraints.
type TypeVariable struct {
	// Name identifies the type variable (e.g., "T", "T1", "T2")
	Name string
	// Constraint specifies what types this variable can be unified with.
	// nil means unconstrained, Number{} means numeric types only, etc.
	Constraint Type
}

func (tv TypeVariable) String() string {
	if tv.Constraint != nil {
		return fmt.Sprintf("%s:%s", tv.Name, tv.Constraint.String())
	}
	return tv.Name
}

// NumericConstraint constrains a type variable to numeric types
type NumericConstraint struct{}

func (nc NumericConstraint) String() string { return "numeric" }

func IsNumeric(t Type) bool {
	if ch, isChan := t.(Chan); isChan {
		t = ch.ValueType
	}
	// Type variables with numeric constraints are considered numeric
	if tv, ok := t.(TypeVariable); ok {
		if tv.Constraint == nil {
			return false // Unconstrained type variable is not specifically numeric
		}
		if _, ok := tv.Constraint.(NumericConstraint); ok {
			return true
		}
		// Check if constraint is a concrete numeric type
		return IsNumeric(tv.Constraint)
	}
	switch t {
	case U8{}, U16{}, U32{}, U64{}, I8{}, I16{}, I32{}, I64{}, F32{}, F64{}:
		return true
	default:
		return false
	}
}

func IsInteger(t Type) bool {
	switch t {
	case U8{}, U16{}, U32{}, U64{}, I8{}, I16{}, I32{}, I64{}:
		return true
	default:
		return false
	}
}

func IsSignedInteger(t Type) bool {
	switch t {
	case I8{}, I16{}, I32{}, I64{}:
		return true
	default:
		return false
	}
}

func IsUnsignedInteger(t Type) bool {
	switch t {
	case U8{}, U16{}, U32{}, U64{}:
		return true
	default:
		return false
	}
}

func IsFloat(t Type) bool {
	switch t {
	case F32{}, F64{}:
		return true
	default:
		return false
	}
}

func IsBool(t Type) bool {
	_, ok := t.(U8)
	return ok
}

func Equal(t Type, v Type) bool {
	return t == v
}

func Is64Bit(t Type) bool {
	switch t {
	case I64{}, U64{}, TimeStamp{}, TimeSpan{}, F64{}:
		return true
	default:
		return false
	}
}

func Assert[T Type](in Type) (T, error) {
	casted, ok := in.(T)
	if !ok {
		return casted, errors.Newf("type mismatch: expected %T, got %T", casted.String(), in)
	}
	return casted, nil
}

var (
	UnsignedIntegers = []Type{U8{}, U16{}, U32{}, U64{}}
	SignedIntegers   = []Type{I8{}, I16{}, I32{}, I64{}}
	Floats           = []Type{F32{}, F64{}}
	Numerics         = slices.Concat(UnsignedIntegers, SignedIntegers, Floats)
)

func TypeFromTelem(t telem.DataType) Type {
	switch t {
	case telem.Uint8T:
		return U8{}
	case telem.Uint16T:
		return U16{}
	case telem.Uint32T:
		return U32{}
	case telem.Uint64T:
		return U64{}
	case telem.Int8T:
		return I8{}
	case telem.Int16T:
		return I16{}
	case telem.Int32T:
		return I32{}
	case telem.Int64T:
		return I64{}
	case telem.Float32T:
		return F32{}
	case telem.Float64T:
		return F64{}
	case telem.StringT, telem.JSONT, telem.UUIDT:
		return String{}
	case telem.TimeStampT:
		return TimeStamp{}
	default:
		return nil
	}
}

// IsTypeVariable checks if a type is a type variable
func IsTypeVariable(t Type) bool {
	_, ok := t.(TypeVariable)
	return ok
}

// NewTypeVariable creates a new type variable with the given name and constraint
func NewTypeVariable(name string, constraint Type) TypeVariable {
	return TypeVariable{Name: name, Constraint: constraint}
}
