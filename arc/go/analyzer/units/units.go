// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package units

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

var (
	ErrDimensions             = errors.New("dimensions")
	ErrIncompatibleDimensions = errors.Wrap(ErrDimensions, "incompatible")
)

// ValidateBinaryOp validates dimensional compatibility for a binary operation.
// Returns false and adds diagnostics if the operation is dimensionally invalid.
// For additive operations (+, -), also checks for magnitude safety issues and
// adds warnings if precision loss may occur.
//
// Rules:
//   - Multiplication/Division (*, /): Always valid.
//   - Addition/Subtraction/Modulo/Comparison (+, -, %, ==, !=, <, >, <=, >=):
//     Dimensions must match (or one operand must be dimensionless).
//
// Note: Power operations (^) are handled separately by ValidatePowerOp.
func ValidateBinaryOp[AST antlr.ParserRuleContext](
	ctx context.Context[AST],
	op string,
	left, right types.Type,
) bool {
	// Multiplication and division are always dimensionally valid
	if op == "*" || op == "/" {
		return true
	}
	// If either operand lacks units, it's always valid
	// (dimensionless values are compatible with anything)
	if left.Unit == nil || right.Unit == nil {
		return true
	}
	if !left.Unit.Dimensions.Equal(right.Unit.Dimensions) {
		ctx.Diagnostics.AddError(
			errors.Wrapf(ErrIncompatibleDimensions, "%s vs %s", left.Unit.Dimensions, right.Unit.Dimensions),
			ctx.AST,
		)
		return false
	}
	// Check magnitude safety for additive operations (precision loss warning)
	if op == "+" || op == "-" {
		checkAdditiveScaleSafety(ctx, left, right)
	}
	return true
}

// ValidatePowerOp validates dimensional compatibility for power operations (^).
// Exponent must be dimensionless. If the base has dimensions, the exponent
// must be a literal integer (isLiteral=true).
func ValidatePowerOp(base, exp types.Type, isLiteral bool) error {
	if exp.Unit != nil && !exp.Unit.Dimensions.IsZero() {
		return errors.Wrapf(
			ErrDimensions,
			"exponent in power operation must be dimensionless, got %s",
			exp.Unit.Dimensions.String(),
		)
	}
	if base.Unit == nil || base.Unit.Dimensions.IsZero() {
		return nil
	}
	if !isLiteral {
		return errors.Wrap(
			ErrDimensions,
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
			ErrDimensions,
			"cannot convert between dimensioned and dimensionless values",
		)
	}
	if !from.Dimensions.Equal(to.Dimensions) {
		return 0, errors.Wrapf(
			ErrIncompatibleDimensions,
			"cannot convert %s to %s",
			from.Dimensions.String(), to.Dimensions.String(),
		)
	}
	return from.Scale / to.Scale, nil
}
