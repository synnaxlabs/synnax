// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

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

// InputScale returns what to multiply a variable by to get the unit of the input that
// reads it. A variable in s read by an input in ns gives 1e9.
func InputScale(t Type) float64 {
	if t.Kind != KindVarRef || t.Unit == nil || t.Elem == nil || t.Elem.Unit == nil {
		return 1
	}
	return t.Elem.Unit.Scale / t.Unit.Scale
}

// UnitsIncompatible reports whether a and b are units of different dimensions.
func UnitsIncompatible(a, b *Unit) bool {
	return a != nil && b != nil && !a.Dimensions.Equal(b.Dimensions)
}

// UnitsConvertible reports whether a and b are different units of the same dimensions.
func UnitsConvertible(a, b *Unit) bool {
	return a != nil && b != nil && !a.Equal(*b) && a.Dimensions.Equal(b.Dimensions)
}
