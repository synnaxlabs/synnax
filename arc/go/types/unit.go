// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import "math"

// wholeFactorTolerance is the relative error under which a factor is a whole number.
// It is 4 times the float64 epsilon.
const wholeFactorTolerance = 4 * 0x1p-52

// UnitsAssignable reports whether a value with unit a may be assigned to a target with
// unit b. A nil unit on either side is treated as a wildcard, matching the WASM and
// Cesium representation where i64 and i64 ns share the same wire type (timestamp <->
// int64). Two non-nil units must match exactly, so f32 psi vs f32 bar still fails.
func UnitsAssignable(a, b *Unit) bool {
	if a == nil || b == nil {
		return true
	}
	return a.Equal(*b)
}

// WithUnit returns t with unit on its value type.
func WithUnit(t Type, unit *Unit) Type {
	if (t.Kind == KindChan || t.Kind == KindSeries) && t.Elem != nil {
		elem := WithUnit(*t.Elem, unit)
		t.Elem = &elem
		return t
	}
	t.Unit = unit
	return t
}

// InputScale returns what to multiply a variable by to get the unit of the input that
// reads it, and what to divide it by. A variable in s read by an input in ns gives 1e9
// and 1e-9.
func InputScale(t Type) (up, down float64) {
	if t.Kind != KindVarRef || t.Unit == nil || t.Elem == nil || t.Elem.Unit == nil {
		return 1, 1
	}
	return UnitFactor(t.Elem.Unit, t.Unit), UnitFactor(t.Unit, t.Elem.Unit)
}

// UnitFactor returns what to multiply a value in from by to get its value in to. A
// factor that floating point puts next to a whole number is that whole number.
func UnitFactor(from, to *Unit) float64 {
	factor := from.Scale / to.Scale
	if whole := math.Round(factor); whole != 0 &&
		math.Abs(factor-whole) <= wholeFactorTolerance*math.Abs(whole) {
		return whole
	}
	return factor
}

// UnitScale returns the factor that converts a value of type t to the unit to, and
// whether to divide by it. exact is false for an integer with no whole factor.
func UnitScale(t Type, to *Unit) (factor float64, divide, exact bool) {
	whole := func(f float64) bool {
		return f == math.Trunc(f) && (t.Is64Bit() || f <= math.MaxInt32)
	}
	factor = UnitFactor(t.Unit, to)
	if t.IsFloat() || whole(factor) {
		return factor, false, true
	}
	if down := UnitFactor(to, t.Unit); whole(down) {
		return down, true, true
	}
	return factor, false, false
}

// CommonUnit returns the smaller of a and b, or the one that is not nil.
func CommonUnit(a, b *Unit) *Unit {
	if a == nil {
		return b
	}
	if b == nil || a.Scale <= b.Scale {
		return a
	}
	return b
}

// ProductUnit returns the unit of a product, which is nil for two units.
func ProductUnit(a, b *Unit) *Unit {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return nil
}

// QuotientUnit returns the unit of a divided by b, which is nil when b has a unit.
func QuotientUnit(a, b *Unit) *Unit {
	if b == nil {
		return a
	}
	return nil
}

// UnitsIncompatible reports whether a and b are units of different dimensions.
func UnitsIncompatible(a, b *Unit) bool {
	return a != nil && b != nil && !a.Dimensions.Equal(b.Dimensions)
}

// UnitsConvertible reports whether a and b are different units of the same dimensions.
func UnitsConvertible(a, b *Unit) bool {
	return a != nil && b != nil && !a.Equal(*b) && a.Dimensions.Equal(b.Dimensions)
}
