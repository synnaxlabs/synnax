// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package units

import (
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// CheckBinaryOp performs dimensional analysis for a binary operation.
// It returns the resulting type with appropriate unit metadata, or an error
// if the operation is dimensionally invalid.
//
// Rules:
//   - Addition/Subtraction (+, -): Operands must have matching dimensions.
//     Result has the same dimensions, normalized to SI base units.
//   - Multiplication (*): Dimension exponents are added.
//     Result dimensions = left.dimensions * right.dimensions
//   - Division (/): Dimension exponents are subtracted.
//     Result dimensions = left.dimensions / right.dimensions
//   - Power (^): Only allowed with dimensionless exponent.
//     Result dimensions = left.dimensions * exponent (if integer)
//   - Comparison (==, !=, <, >, <=, >=): Operands must have matching dimensions.
//     Result is always u8 (boolean), no units.
//   - Modulo (%): Same as multiplication/division - dimension arithmetic.
//
// If both operands are dimensionless (no units), the result is also dimensionless.
func CheckBinaryOp(op string, left, right types.Type) (types.Type, error) {
	leftUnit := left.Unit
	rightUnit := right.Unit

	// Handle comparisons first - they always return boolean regardless of units
	switch op {
	case "==", "!=", "<", ">", "<=", ">=":
		return checkComparison(left, right, leftUnit, rightUnit)
	}

	// If neither operand has units, just return the promoted numeric type
	if leftUnit == nil && rightUnit == nil {
		return PromoteNumeric(left, right), nil
	}

	switch op {
	case "+", "-":
		return checkAdditive(left, right, leftUnit, rightUnit)
	case "*":
		return checkMultiplicative(left, right, leftUnit, rightUnit)
	case "/":
		return checkDivision(left, right, leftUnit, rightUnit)
	case "%":
		// Modulo follows same rules as division
		return checkDivision(left, right, leftUnit, rightUnit)
	case "^":
		return checkPower(left, right, leftUnit, rightUnit)
	default:
		return types.Type{}, errors.Newf("unknown binary operator: %s", op)
	}
}

// checkAdditive handles + and - operations.
// Dimensions must match exactly; result is normalized to SI base units.
func checkAdditive(left, right types.Type, leftUnit, rightUnit *types.Unit) (types.Type, error) {
	result := PromoteNumeric(left, right)

	// If one side has units and the other doesn't, it's an error
	// (unless the dimensionless side is just a literal that will be inferred)
	if (leftUnit == nil) != (rightUnit == nil) {
		if leftUnit == nil {
			// Left is dimensionless, right has units - left should adopt right's dimensions
			result.Unit = rightUnit
			return result, nil
		}
		// Right is dimensionless, left has units - right should adopt left's dimensions
		result.Unit = leftUnit
		return result, nil
	}

	// Both have units - dimensions must match
	if !leftUnit.Dimensions.Equal(rightUnit.Dimensions) {
		return types.Type{}, errors.Newf(
			"incompatible dimensions in %s: %s vs %s",
			"+/-", leftUnit.Dimensions.String(), rightUnit.Dimensions.String(),
		)
	}

	// Result is normalized to SI base units (scale = 1)
	result.Unit = &types.Unit{
		Dimensions: leftUnit.Dimensions,
		Scale:      1.0, // Normalized to base unit
		Name:       "",  // Generic SI name would be added here
	}

	return result, nil
}

// checkMultiplicative handles * operations.
// Dimension exponents are added.
func checkMultiplicative(left, right types.Type, leftUnit, rightUnit *types.Unit) (types.Type, error) {
	result := PromoteNumeric(left, right)

	// Get dimensions (defaulting to dimensionless if no unit)
	leftDim := types.DimNone
	rightDim := types.DimNone
	if leftUnit != nil {
		leftDim = leftUnit.Dimensions
	}
	if rightUnit != nil {
		rightDim = rightUnit.Dimensions
	}

	// Multiply dimensions
	resultDim := leftDim.Mul(rightDim)

	// If result is dimensionless, no unit needed
	if resultDim.IsZero() {
		result.Unit = nil
		return result, nil
	}

	result.Unit = &types.Unit{
		Dimensions: resultDim,
		Scale:      1.0,
		Name:       "",
	}

	return result, nil
}

// checkDivision handles / operations.
// Dimension exponents are subtracted.
func checkDivision(left, right types.Type, leftUnit, rightUnit *types.Unit) (types.Type, error) {
	result := PromoteNumeric(left, right)

	// Get dimensions (defaulting to dimensionless if no unit)
	leftDim := types.DimNone
	rightDim := types.DimNone
	if leftUnit != nil {
		leftDim = leftUnit.Dimensions
	}
	if rightUnit != nil {
		rightDim = rightUnit.Dimensions
	}

	// Divide dimensions
	resultDim := leftDim.Div(rightDim)

	// If result is dimensionless, no unit needed
	if resultDim.IsZero() {
		result.Unit = nil
		return result, nil
	}

	result.Unit = &types.Unit{
		Dimensions: resultDim,
		Scale:      1.0,
		Name:       "",
	}

	return result, nil
}

// checkPower handles ^ operations.
// Exponent must be dimensionless. Result dimensions = base dimensions * exponent.
func checkPower(left, right types.Type, leftUnit, rightUnit *types.Unit) (types.Type, error) {
	result := PromoteNumeric(left, right)

	// Exponent must be dimensionless
	if rightUnit != nil && !rightUnit.Dimensions.IsZero() {
		return types.Type{}, errors.Newf(
			"exponent in power operation must be dimensionless, got %s",
			rightUnit.Dimensions.String(),
		)
	}

	// If base is dimensionless, result is dimensionless
	if leftUnit == nil || leftUnit.Dimensions.IsZero() {
		return result, nil
	}

	// For now, we only support integer exponents for dimensional analysis
	// Complex exponents like x^0.5 require special handling
	// Just pass through the base dimensions for now (this is conservative)
	result.Unit = &types.Unit{
		Dimensions: leftUnit.Dimensions,
		Scale:      1.0,
		Name:       "",
	}

	return result, nil
}

// checkComparison handles comparison operations.
// Dimensions must match. Result is always boolean (u8), no units.
func checkComparison(left, right types.Type, leftUnit, rightUnit *types.Unit) (types.Type, error) {
	// If one side has units and the other doesn't, allow it (dimensionless comparison)
	if (leftUnit == nil) != (rightUnit == nil) {
		return types.U8(), nil
	}

	// If both have units, dimensions must match
	if leftUnit != nil && rightUnit != nil {
		if !leftUnit.Dimensions.Equal(rightUnit.Dimensions) {
			return types.Type{}, errors.Newf(
				"incompatible dimensions in comparison: %s vs %s",
				leftUnit.Dimensions.String(), rightUnit.Dimensions.String(),
			)
		}
	}

	// Comparisons always return boolean (u8), no units
	return types.U8(), nil
}

// PromoteNumeric returns the wider numeric type from two types.
// Priority: f64 > f32 > i64 > u64 > i32 > u32 > i16 > u16 > i8 > u8
func PromoteNumeric(left, right types.Type) types.Type {
	// Get the base kinds (ignoring units)
	leftKind := left.Kind
	rightKind := right.Kind

	// Priority ordering
	priority := map[types.TypeKind]int{
		types.KindU8:  1,
		types.KindI8:  2,
		types.KindU16: 3,
		types.KindI16: 4,
		types.KindU32: 5,
		types.KindI32: 6,
		types.KindU64: 7,
		types.KindI64: 8,
		types.KindF32: 9,
		types.KindF64: 10,
	}

	leftPriority := priority[leftKind]
	rightPriority := priority[rightKind]

	if leftPriority >= rightPriority {
		return types.Type{Kind: leftKind}
	}
	return types.Type{Kind: rightKind}
}

// SameDimensions checks if two units have the same dimensions.
// Returns true if both are nil or both have equal dimensions.
func SameDimensions(left, right *types.Unit) bool {
	if left == nil && right == nil {
		return true
	}
	if left == nil || right == nil {
		return false
	}
	return left.Dimensions.Equal(right.Dimensions)
}

// ScaleConversionFactor returns the factor needed to convert from 'from' unit to 'to' unit.
// The dimensions must be compatible. Returns (factor, error).
// To convert: value_in_to_units = value_in_from_units * factor
func ScaleConversionFactor(from, to *types.Unit) (float64, error) {
	// Handle nil cases
	if from == nil && to == nil {
		return 1.0, nil
	}
	if from == nil || to == nil {
		// Can't convert between dimensioned and dimensionless
		return 0, errors.New("cannot convert between dimensioned and dimensionless values")
	}

	// Dimensions must match
	if !from.Dimensions.Equal(to.Dimensions) {
		return 0, errors.Newf(
			"cannot convert between incompatible dimensions: %s vs %s",
			from.Dimensions.String(), to.Dimensions.String(),
		)
	}

	// Conversion factor: from.Scale / to.Scale
	// Example: 1 km = 1000 m, so Scale(km) = 1000, Scale(m) = 1
	// To convert 5 km to m: 5 * (1000 / 1) = 5000 m
	return from.Scale / to.Scale, nil
}
