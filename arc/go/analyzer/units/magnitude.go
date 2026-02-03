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
	"math"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/types"
)

// Scale ratio thresholds for magnitude safety warnings.
// These represent the maximum scale difference before precision loss becomes likely.
const (
	// ScaleRatioThresholdF64 is the maximum scale ratio for f64 additive operations
	// before warning about precision loss. Set conservatively at 1e12 (~12 orders
	// of magnitude) to leave headroom for the value magnitudes themselves.
	// f64 has ~15-16 significant decimal digits.
	ScaleRatioThresholdF64 = 1e12

	// ScaleRatioThresholdF32 is the maximum scale ratio for f32 additive operations.
	// f32 has ~7 significant decimal digits.
	ScaleRatioThresholdF32 = 1e5

	// TruncationThreshold is the minimum scale factor below which integer assignment
	// will warn about truncation to zero.
	TruncationThreshold = 1e-9
)

// checkAdditiveScaleSafety checks if combining two units in an additive operation
// (+, -) may cause precision loss. Adds a warning to the context if there's a
// potential problem.
//
// For additive operations, values must be converted to a common unit before adding.
// If the scale ratio is too large, the smaller value may lose precision in the
// floating-point representation.
//
// Example: 5000K + 6pK
//   - K scale: 1.0, pK scale: 1e-12
//   - Scale ratio: 1e12
//   - Warning: adding values with 12 orders of magnitude difference may lose precision
func checkAdditiveScaleSafety[AST antlr.ParserRuleContext](
	ctx context.Context[AST],
	left, right types.Type,
) {
	// Skip if either operand lacks units
	if left.Unit == nil || right.Unit == nil {
		return
	}

	// Skip if dimensions don't match (will be caught by dimensional validation)
	if !left.Unit.Dimensions.Equal(right.Unit.Dimensions) {
		return
	}

	// Calculate scale ratio
	leftScale := left.Unit.Scale
	rightScale := right.Unit.Scale

	var scaleRatio float64
	if leftScale > rightScale {
		scaleRatio = leftScale / rightScale
	} else {
		scaleRatio = rightScale / leftScale
	}

	// Determine threshold based on numeric type (use stricter threshold for f32)
	threshold := ScaleRatioThresholdF64
	if left.Kind == types.KindF32 || right.Kind == types.KindF32 {
		threshold = ScaleRatioThresholdF32
	}

	if scaleRatio >= threshold {
		ordersMagnitude := int(math.Log10(scaleRatio))
		ctx.Diagnostics.Add(diagnostics.Warningf(
			ctx.AST,
			"additive operation combines values with ~%d orders of magnitude difference (%s vs %s); smaller value may lose precision",
			ordersMagnitude,
			left.Unit.Name,
			right.Unit.Name,
		))
	}
}

// CheckAssignmentScaleSafety checks if assigning a value with one unit to a variable
// with a different unit scale may cause truncation or overflow for integer types.
// Adds a warning to the context if there's a potential problem.
//
// Example 1: 6pK assigned to i32 K
//   - 6 * (1e-12 / 1.0) = 6e-12 K -> truncates to 0
//
// Example 2: 5000K assigned to i32 pK
//   - 5000 * (1.0 / 1e-12) = 5e15 pK -> overflows i32
//
// The optional sourceValue parameter allows for more precise overflow checking
// when the literal value is known at compile time.
func CheckAssignmentScaleSafety[AST antlr.ParserRuleContext](
	ctx context.Context[AST],
	sourceType, targetType types.Type,
	sourceValue *float64,
) {
	// Skip if either lacks units
	if sourceType.Unit == nil || targetType.Unit == nil {
		return
	}

	// Skip if dimensions don't match (will be caught by dimensional validation)
	if !sourceType.Unit.Dimensions.Equal(targetType.Unit.Dimensions) {
		return
	}

	// Skip if not assigning to an integer type (floats handle scale differences)
	if !targetType.IsInteger() {
		return
	}

	// Calculate scale conversion factor: value_in_target = value_in_source * factor
	scaleFactor := sourceType.Unit.Scale / targetType.Unit.Scale

	// Check for potential truncation (scaling down to very small values)
	if scaleFactor <= TruncationThreshold {
		ctx.Diagnostics.Add(diagnostics.Warningf(
			ctx.AST,
			"unit conversion from %s to %s may truncate to zero for integer type %s (scale factor: %.2e)",
			sourceType.Unit.Name,
			targetType.Unit.Name,
			targetType.Kind.String(),
			scaleFactor,
		))
		return
	}

	// Check for potential overflow (scaling up to very large values)
	if scaleFactor > 1.0 {
		maxVal := targetType.IntegerMaxValue()

		// If even a value of 1 would overflow, definitely warn
		if scaleFactor > float64(maxVal) {
			ctx.Diagnostics.Add(diagnostics.Warningf(
				ctx.AST,
				"unit conversion from %s to %s may overflow %s (scale factor: %.2e, max: %d)",
				sourceType.Unit.Name,
				targetType.Unit.Name,
				targetType.Kind.String(),
				scaleFactor,
				maxVal,
			))
			return
		}

		// If source value is known, check if it specifically overflows
		if sourceValue != nil {
			convertedValue := *sourceValue * scaleFactor
			minVal := targetType.IntegerMinValue()
			if convertedValue > float64(maxVal) || convertedValue < float64(minVal) {
				ctx.Diagnostics.Add(diagnostics.Warningf(
					ctx.AST,
					"value %.2g %s converts to %.2e %s, which overflows %s (range: %d to %d)",
					*sourceValue,
					sourceType.Unit.Name,
					convertedValue,
					targetType.Unit.Name,
					targetType.Kind.String(),
					minVal,
					maxVal,
				))
			}
		}
	}
}
