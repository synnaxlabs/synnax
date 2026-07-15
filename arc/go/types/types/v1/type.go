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
	"encoding/json"
	"fmt"
	"maps"
	"math"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
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

// Positional returns the params bindable by position: all params except the
// trigger, which the upstream feeds rather than the call site.
func (p Params) Positional(trigger string) Params {
	if trigger == "" {
		return p
	}
	positional := make(Params, 0, len(p))
	for _, param := range p {
		if param.Name == trigger {
			continue
		}
		positional = append(positional, param)
	}
	return positional
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
