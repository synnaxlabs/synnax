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

var (
	DimensionsError             = errors.New("dimensions")
	IncompatibleDimensionsError = errors.Wrap(DimensionsError, "incompatible")
)

// ValidateBinaryOp validates dimensional compatibility for a binary operation.
// Returns an error if the operation is dimensionally invalid.
//
// Rules:
//   - Multiplication/Division (*, /): Always valid.
//   - Addition/Subtraction/Modulo/Comparison (+, -, %, ==, !=, <, >, <=, >=):
//     Dimensions must match (or one operand must be dimensionless).
//
// Note: Power operations (^) are handled separately by ValidatePowerOp.
func ValidateBinaryOp(op string, left, right types.Type) error {
	// Multiplication and division are always dimensionally valid
	if op == "*" || op == "/" {
		return nil
	}
	// If either operand lacks units, it's always valid
	// (dimensionless values are compatible with anything)
	if left.Unit == nil || right.Unit == nil || left.Unit.Dimensions.Equal(right.Unit.Dimensions) {
		return nil
	}
	return errors.Wrapf(
		IncompatibleDimensionsError,
		"%s vs %s",
		left.Unit.Dimensions,
		right.Unit.Dimensions,
	)
}

// ValidatePowerOp validates dimensional compatibility for power operations (^).
// Exponent must be dimensionless. If the base has dimensions, the exponent
// must be a literal integer (isLiteral=true).
func ValidatePowerOp(base, exp types.Type, isLiteral bool) error {
	if exp.Unit != nil && !exp.Unit.Dimensions.IsZero() {
		return errors.Wrapf(
			DimensionsError,
			"exponent in power operation must be dimensionless, got %s",
			exp.Unit.Dimensions.String(),
		)
	}
	if base.Unit == nil || base.Unit.Dimensions.IsZero() {
		return nil
	}
	if !isLiteral {
		return errors.Wrap(
			DimensionsError,
			"power operation with dimensioned base requires a literal integer exponent",
		)
	}
	return nil
}

// ScaleFactor returns the factor needed to convert from 'from' unit to 'to' unit.
// The dimensions must be compatible. Returns (factor, error).
// To convert: value_in_to_units = value_in_from_units * factor
func ScaleFactor(from, to *types.Unit) (float64, error) {
	if from == nil && to == nil {
		return 1.0, nil
	}
	if from == nil || to == nil {
		return 0, errors.Wrap(
			DimensionsError,
			"cannot convert between dimensioned and dimensionless values",
		)
	}
	if !from.Dimensions.Equal(to.Dimensions) {
		return 0, errors.Wrapf(
			IncompatibleDimensionsError,
			"cannot convert %s to %s",
			from.Dimensions.String(), to.Dimensions.String(),
		)
	}
	return from.Scale / to.Scale, nil
}
