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
	"encoding/json"
	"fmt"
	"maps"
	"math"
	"slices"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// IsRead returns true if the direction includes read.
func (d ChanDirection) IsRead() bool { return d&ChanDirectionRead != 0 }

// IsWrite returns true if the direction includes write.
func (d ChanDirection) IsWrite() bool { return d&ChanDirectionWrite != 0 }

// IsSet returns true if any direction has been specified.
func (d ChanDirection) IsSet() bool { return d != ChanDirectionNone }

// CheckCompatibility returns an error if actual is incompatible with the
// required direction d. Returns nil when compatible or when either side
// has no direction set.
func (d ChanDirection) CheckCompatibility(actual ChanDirection) error {
	if !d.IsSet() || !actual.IsSet() {
		return nil
	}
	if d.IsWrite() && !actual.IsWrite() {
		return errors.New("expected a write channel, but got a read channel")
	}
	if d.IsRead() && !actual.IsRead() {
		return errors.New("expected a read channel, but got a write channel")
	}
	return nil
}

var _ json.Marshaler = (Params)(nil)

// MarshalJSON implements the json.Marshal interface.
func (p Params) MarshalJSON() ([]byte, error) {
	if p == nil {
		return json.Marshal([]Param{})
	}
	type params []Param
	return json.Marshal(params(p))
}

// Get retrieves a parameter by name. Returns the parameter and true if found,
// otherwise returns a zero Param and false.
func (p Params) Get(name string) (Param, bool) {
	return lo.Find(p, func(item Param) bool {
		return item.Name == name
	})
}

// GetIndex returns the index of a parameter by name. Returns -1 if not found.
func (p Params) GetIndex(name string) int {
	_, i, ok := lo.FindIndexOf(p, func(item Param) bool {
		return item.Name == name
	})
	if !ok {
		return -1
	}
	return i
}

// Has returns true if a parameter with the given name exists.
func (p Params) Has(name string) bool {
	_, ok := p.Get(name)
	return ok
}

// ValueMap returns a map of parameter names to their values.
func (p Params) ValueMap() map[string]any {
	return lo.SliceToMap(p, func(item Param) (string, any) {
		return item.Name, item.Value
	})
}

// RequiredCount returns the number of required (non-optional) parameters.
// A parameter is optional if its Value field is non-nil (has a default).
func (p Params) RequiredCount() int {
	count := 0
	for _, param := range p {
		if param.Value == nil {
			count++
		}
	}
	return count
}

// IntegerMaxValue returns the maximum value representable by this integer type.
// Panics if the type is not an integer type.
// Note: For U64, returns math.MaxInt64 for comparison safety since MaxUint64
// cannot be represented in int64.
func (t Type) IntegerMaxValue() int64 {
	if !t.IsInteger() {
		panic(fmt.Sprintf("[type.IntegerMaxValue] attempted to call on non-integer %s", t))
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

// IntegerMinValue returns the minimum value representable by this integer type.
// Panics if the type is not an integer type.
// Returns 0 for unsigned integer types.
func (t Type) IntegerMinValue() int64 {
	if !t.IsInteger() {
		panic(fmt.Sprintf("[types.IntegerMinValue] attempted to call on non-integer %s", t))
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

// DebugString returns a detailed string representation of the type for debugging.
// Unlike String(), this includes type variable names for better debugging visibility.
func (t Type) DebugString() string {
	switch t.Kind {
	case KindChan:
		if t.Elem != nil {
			return "chan " + t.Elem.DebugString()
		}
		return "chan <invalid>"
	case KindSeries:
		if t.Elem != nil {
			return "series " + t.Elem.DebugString()
		}
		return "series <invalid>"
	case KindVariable:
		if t.Constraint != nil {
			return t.Name + ":" + t.Constraint.DebugString()
		}
		return t.Name
	default:
		return t.String()
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

// Chan returns a channel type wrapping the given value type.
func Chan(valueType Type) Type {
	return Type{Kind: KindChan, Elem: &valueType}
}

// ReadChan returns a channel type annotated for read access.
func ReadChan(valueType Type) Type {
	return Type{Kind: KindChan, Elem: &valueType, ChanDirection: ChanDirectionRead}
}

// WriteChan returns a channel type annotated for write access.
func WriteChan(valueType Type) Type {
	return Type{Kind: KindChan, Elem: &valueType, ChanDirection: ChanDirectionWrite}
}

// Series returns a series/array type wrapping the given value type.
func Series(valueType Type) Type { return Type{Kind: KindSeries, Elem: &valueType} }

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

// Function creates a function type with the given inputs, outputs, and optional config.
func Function(props FunctionProperties) Type {
	return Type{Kind: KindFunction, FunctionProperties: props}
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

func (t Type) IsSigned() bool {
	return t.IsSignedInteger() || t.IsFloat()
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
	return t.Unwrap().Kind == KindU8
}

// Unwrap returns the value type of chan/series types, or the type itself otherwise.
func (t Type) Unwrap() Type {
	if (t.Kind == KindChan || t.Kind == KindSeries) && t.Elem != nil {
		return *t.Elem
	}
	return t
}

// UnwrapChan returns the effective value type when a type is used as a value.
// Channels are implicitly read: chan<T> -> T
// Series stay as series: series<T> -> series<T>
func (t Type) UnwrapChan() Type {
	if t.Kind == KindChan && t.Elem != nil {
		return *t.Elem
	}
	return t
}

// StructuralMatch returns true if both types have the same wrapper structure.
func StructuralMatch(t1, t2 Type) bool {
	return (t1.Kind == KindSeries) == (t2.Kind == KindSeries) &&
		(t1.Kind == KindChan) == (t2.Kind == KindChan)
}

// IsValid returns true if the type is not invalid or uninitialized.
func (t *Type) IsValid() bool { return t.Kind != KindInvalid }

// Equal compares two types for strict structural equality, including units.
func Equal(t Type, v Type) bool {
	if t.Kind != v.Kind {
		return false
	}
	if t.Kind == KindChan || t.Kind == KindSeries {
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
		if !paramsEqual(t.Outputs, v.Outputs) {
			return false
		}
		return paramsEqual(t.Config, v.Config)
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

var (
	UnsignedIntegers = []Type{U8(), U16(), U32(), U64()}
	SignedIntegers   = []Type{I8(), I16(), I32(), I64()}
	Floats           = []Type{F32(), F64()}
	Numerics         = slices.Concat(UnsignedIntegers, SignedIntegers, Floats)
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

// Copy returns a deep copy of the Channels.
func (c Channels) Copy() Channels {
	if c.Read == nil {
		c.Read = make(map[uint32]string)
	}
	if c.Write == nil {
		c.Write = make(map[uint32]string)
	}
	return Channels{Read: maps.Clone(c.Read), Write: maps.Clone(c.Write)}
}

// NewChannels creates a new Channels with empty read and write sets.
func NewChannels() Channels {
	return Channels{Read: make(map[uint32]string), Write: make(map[uint32]string)}
}
