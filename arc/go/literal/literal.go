// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package literal

import (
	"math"
	"strconv"
	"strings"

	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// ParsedValue represents a parsed literal value with its type.
type ParsedValue struct {
	// Value is the parsed Go value (e.g., int64, float64, string)
	Value any
	// Type is the Arc type of the value
	Type types.Type
}

// Parse parses a literal AST node and returns its value and type. It supports numeric
// literals (integer, float, with optional unit suffix) and validates type
// compatibility. The targetType parameter specifies the expected type for conversion.
// Float-to-int conversions that lose precision will fail (matching Go semantics for
// constant conversions).
func Parse(
	literal parser.ILiteralContext,
	targetType types.Type,
) (ParsedValue, error) {
	if num := literal.NumericLiteral(); num != nil {
		return ParseNumeric(num, targetType)
	}
	if str := parser.StringTerminal(literal); str != nil {
		return ParseString(str.GetText(), targetType)
	}
	if series := literal.SeriesLiteral(); series != nil {
		return ParsedValue{}, errors.New(
			"series literals not supported for default values",
		)
	}
	if boolLit := literal.BooleanLiteral(); boolLit != nil {
		if targetType.IsValid() && targetType.Kind != types.KindBool {
			return ParsedValue{}, errors.Newf("cannot assign bool to %s", targetType)
		}
		return ParsedValue{Value: boolLit.TRUE() != nil, Type: types.Bool()}, nil
	}
	return ParsedValue{}, errors.New("unknown literal type")
}

// ParseString parses any of the eight string literal forms ("..." / `...`
// crossed with optional r/f/rf/fr prefix) and returns the cooked Go string. For
// format-prefixed strings without placeholders, {{ and }} are collapsed to
// literal braces; for format strings with placeholders the body is returned
// with placeholders intact so callers can run FmtStrParse themselves.
func ParseString(text string, targetType types.Type) (ParsedValue, error) {
	if targetType.IsValid() && targetType.Kind != types.KindString {
		return ParsedValue{}, errors.Newf("cannot assign string to %s", targetType)
	}
	body, flags, ok := StripQuotes(text)
	if !ok {
		return ParsedValue{}, errors.Newf("invalid string literal: %s", text)
	}
	value := body
	if !flags.Raw {
		var err error
		value, err = UnescapeString(body, flags.Multi)
		if err != nil {
			return ParsedValue{}, errors.Wrapf(err, "invalid string literal %s", text)
		}
	}
	if flags.Format {
		segments, err := FmtStrSplit(body, flags)
		if err != nil {
			return ParsedValue{}, errors.Wrapf(err, "invalid format string %s", text)
		}
		if !FmtStrHasPlaceholder(segments) {
			var sb strings.Builder
			for _, seg := range segments {
				sb.WriteString(seg.Text)
			}
			value = sb.String()
		}
	}
	return ParsedValue{Value: value, Type: types.String()}, nil
}

// ParseNumeric parses a numeric literal (integer or float, with optional unit suffix)
// and returns its value and type.
// It validates that the value fits within the target type's range and rejects lossy
// conversions.
// If a unit suffix is present (e.g., "300ms"), unit conversion is applied.
func ParseNumeric(
	numLit parser.INumericLiteralContext,
	targetType types.Type,
) (ParsedValue, error) {
	return parseNumericSigned(numLit, numLit.MINUS() != nil, targetType)
}

// parseNumericSigned parses numLit's magnitude and applies negate before the range
// check and unit scaling, so type minimums like -128 for i8 fit and negated units keep
// sign.
func parseNumericSigned(
	numLit parser.INumericLiteralContext,
	negate bool,
	targetType types.Type,
) (ParsedValue, error) {
	if intLit := numLit.INTEGER_LITERAL(); intLit != nil {
		intValue, err := strconv.ParseInt(intLit.GetText(), 10, 64)
		if err != nil {
			return ParsedValue{}, errors.Wrapf(
				err,
				"invalid integer literal: %s",
				intLit.GetText(),
			)
		}
		if negate {
			intValue = -intValue
		}
		if unitID := numLit.IDENTIFIER(); unitID != nil {
			return parseNumericWithUnit(
				float64(intValue),
				true,
				unitID.GetText(),
				targetType,
			)
		}
		return parseIntegerLiteral(intValue, targetType)
	}

	if floatLit := numLit.FLOAT_LITERAL(); floatLit != nil {
		floatValue, err := strconv.ParseFloat(floatLit.GetText(), 64)
		if err != nil {
			return ParsedValue{}, errors.Wrapf(
				err,
				"invalid float literal: %s",
				floatLit.GetText(),
			)
		}
		if negate {
			floatValue = -floatValue
		}
		if unitID := numLit.IDENTIFIER(); unitID != nil {
			return parseNumericWithUnit(floatValue, false, unitID.GetText(), targetType)
		}
		return parseFloatLiteral(floatValue, targetType)
	}

	return ParsedValue{}, errors.New("unknown numeric literal")
}

// parseNumericWithUnit handles numeric literals with unit suffixes (e.g., "300ms",
// "5km"). It looks up the unit, applies scale conversion, and returns the appropriate
// value.
func parseNumericWithUnit(
	numericValue float64,
	isIntLiteral bool,
	unitName string,
	targetType types.Type,
) (ParsedValue, error) {
	unit, ok := units.Resolve(unitName)
	if !ok {
		return ParsedValue{}, errors.Newf("unknown unit: %s", unitName)
	}

	// If target type has a unit, convert to target unit's scale
	// Example: 300ms with target TimeSpan() (ns) → 300 * (1e-3 / 1e-9) = 300,000,000
	if targetType.Unit != nil {
		factor, err := units.ScaleFactor(unit, targetType.Unit)
		if err != nil {
			return ParsedValue{}, err
		}
		targetValue := numericValue * factor
		return convertToTargetKind(targetValue, targetType, unit)
	}

	// If target type has a kind but no unit, convert to SI then to target kind
	if targetType.IsValid() {
		siValue := numericValue * unit.Scale
		return convertToTargetKind(siValue, targetType, unit)
	}

	// Default (no target type): use type inference int64 if scaled result is exact
	// integer AND original literal was integer, else f64
	siValue := numericValue * unit.Scale

	resultType := types.Type{Unit: unit}
	if isIntLiteral && IsExactInteger(siValue) {
		resultType.Kind = types.KindI64
		return ParsedValue{Value: int64(math.Round(siValue)), Type: resultType}, nil
	}
	resultType.Kind = types.KindF64
	return ParsedValue{Value: siValue, Type: resultType}, nil
}

func boundedIntegerRange(kind types.Kind) (lo, hi int64, ok bool) {
	switch kind {
	case types.KindI8:
		return math.MinInt8, math.MaxInt8, true
	case types.KindI16:
		return math.MinInt16, math.MaxInt16, true
	case types.KindI32:
		return math.MinInt32, math.MaxInt32, true
	case types.KindU8:
		return 0, math.MaxUint8, true
	case types.KindU16:
		return 0, math.MaxUint16, true
	case types.KindU32:
		return 0, math.MaxUint32, true
	default:
		return 0, 0, false
	}
}

func castBoundedInteger(value int64, kind types.Kind) any {
	switch kind {
	case types.KindI8:
		return int8(value)
	case types.KindI16:
		return int16(value)
	case types.KindI32:
		return int32(value)
	case types.KindU8:
		return uint8(value)
	case types.KindU16:
		return uint16(value)
	case types.KindU32:
		return uint32(value)
	default:
		panic(errors.Newf("%s is not a bounded integer kind", kind))
	}
}

// parseIntegerLiteral converts an integer value to the target type.
// It takes int64 directly to preserve full precision for large integer literals.
func parseIntegerLiteral(intValue int64, targetType types.Type) (ParsedValue, error) {
	if !targetType.IsValid() {
		return ParsedValue{Value: intValue, Type: types.I64()}, nil
	}
	kind := targetType.Kind
	if lo, hi, ok := boundedIntegerRange(kind); ok {
		t := types.Type{Kind: kind}
		if intValue < lo || intValue > hi {
			return ParsedValue{}, errors.Newf(
				"value %d out of range for %s (must be in [%d, %d])",
				intValue,
				t,
				lo,
				hi,
			)
		}
		return ParsedValue{Value: castBoundedInteger(intValue, kind), Type: t}, nil
	}
	switch kind {
	case types.KindU64:
		if intValue < 0 {
			return ParsedValue{}, errors.Newf(
				"value %d out of range for u64 (must be non-negative)",
				intValue,
			)
		}
		return ParsedValue{Value: uint64(intValue), Type: types.U64()}, nil
	case types.KindF32:
		return ParsedValue{Value: float32(intValue), Type: types.F32()}, nil
	case types.KindF64:
		return ParsedValue{Value: float64(intValue), Type: types.F64()}, nil
	default:
		return ParsedValue{Value: intValue, Type: types.I64()}, nil
	}
}

// parseFloatLiteral converts a float value to the target type.
func parseFloatLiteral(value float64, targetType types.Type) (ParsedValue, error) {
	if !targetType.IsValid() {
		return ParsedValue{Value: value, Type: types.F64()}, nil
	}
	kind := targetType.Kind
	switch kind {
	case types.KindF32:
		if math.Abs(value) > math.MaxFloat32 {
			return ParsedValue{}, errors.Newf("value %f out of range for f32", value)
		}
		return ParsedValue{Value: float32(value), Type: types.F32()}, nil
	case types.KindF64:
		return ParsedValue{Value: value, Type: types.F64()}, nil
	case types.KindI8, types.KindI16, types.KindI32, types.KindI64,
		types.KindU8, types.KindU16, types.KindU32, types.KindU64:
		return parseIntegerFloat(value, kind)
	default:
		return ParsedValue{Value: value, Type: types.F64()}, nil
	}
}

func parseIntegerFloat(value float64, kind types.Kind) (ParsedValue, error) {
	t := types.Type{Kind: kind}
	if value != math.Trunc(value) {
		return ParsedValue{}, errors.Newf(
			"cannot convert non-integer float %f to %s",
			value,
			t,
		)
	}
	switch kind {
	case types.KindI64:
		return ParsedValue{Value: int64(value), Type: t}, nil
	case types.KindU64:
		if value < 0 {
			return ParsedValue{}, errors.Newf(
				"value %f out of range for u64 (must be non-negative)",
				value,
			)
		}
		return ParsedValue{Value: uint64(value), Type: t}, nil
	}
	lo, hi, _ := boundedIntegerRange(kind)
	intVal := int64(value)
	if intVal < lo || intVal > hi {
		return ParsedValue{}, errors.Newf("value %f out of range for %s", value, t)
	}
	return ParsedValue{Value: castBoundedInteger(intVal, kind), Type: t}, nil
}

// convertToTargetKind converts a float64 value to the target type's kind. Uses Go-like
// constant conversion semantics: errors on fractional parts for integer types.
func convertToTargetKind(
	value float64,
	targetType types.Type,
	sourceUnit *types.Unit,
) (ParsedValue, error) {
	resultType := types.Type{Kind: targetType.Kind, Unit: sourceUnit}
	switch targetType.Kind {
	case types.KindI8, types.KindI16, types.KindI32, types.KindI64,
		types.KindU8, types.KindU16, types.KindU32, types.KindU64:
		return convertToIntegerKind(value, resultType)
	case types.KindF32:
		if math.Abs(value) > math.MaxFloat32 && !math.IsInf(value, 0) {
			return ParsedValue{}, errors.Newf("value %g out of range for f32", value)
		}
		return ParsedValue{Value: float32(value), Type: resultType}, nil
	case types.KindF64:
		return ParsedValue{Value: value, Type: resultType}, nil
	default:
		resultType.Kind = types.KindF64
		return ParsedValue{Value: value, Type: resultType}, nil
	}
}

func convertToIntegerKind(value float64, resultType types.Type) (ParsedValue, error) {
	kind := resultType.Kind
	if !IsExactInteger(value) {
		return ParsedValue{}, errors.Newf(
			"cannot convert %g to %s: value has fractional part",
			value, kind,
		)
	}
	value = math.Round(value)
	switch kind {
	case types.KindI64:
		if resultType.Unit.Dimensions.Equal(types.TimeSpan().Unit.Dimensions) {
			return ParsedValue{Value: telem.TimeSpan(value), Type: resultType}, nil
		}
		return ParsedValue{Value: int64(value), Type: resultType}, nil
	case types.KindU64:
		if value < 0 {
			return ParsedValue{}, errors.Newf(
				"value %g out of range for u64 (must be non-negative)",
				value,
			)
		}
		return ParsedValue{Value: uint64(value), Type: resultType}, nil
	}
	lo, hi, _ := boundedIntegerRange(kind)
	intVal := int64(value)
	if intVal < lo || intVal > hi {
		return ParsedValue{}, errors.Newf(
			"value %g out of range for %s",
			value,
			types.Type{Kind: kind},
		)
	}
	return ParsedValue{Value: castBoundedInteger(intVal, kind), Type: resultType}, nil
}

// IsExactInteger checks if a float64 value is very close to an integer.
// Uses a relative epsilon to handle floating-point precision issues at any scale.
func IsExactInteger(value float64) bool {
	rounded := math.Round(value)
	if rounded == 0 {
		return math.Abs(value) < 1e-9
	}
	// Use relative tolerance: difference should be tiny compared to the magnitude
	return math.Abs(value-rounded)/math.Abs(rounded) < 1e-9
}

// ParseConst parses a bare or single-negated literal against targetType, applying the
// sign before the range check so type minimums like -128 for i8 fit. Errors on anything
// else.
func ParseConst(
	expr parser.IExpressionContext,
	targetType types.Type,
) (ParsedValue, error) {
	if expr == nil {
		return ParsedValue{}, errors.New("expression is not a constant")
	}
	if primary := parser.GetPrimaryExpression(expr); primary != nil {
		if lit := primary.Literal(); lit != nil {
			return Parse(lit, targetType)
		}
	}
	if parser.IsNegatedLiteral(expr) {
		if lit := parser.GetLiteral(expr); lit != nil {
			if numLit := lit.NumericLiteral(); numLit != nil {
				return parseNumericSigned(numLit, true, targetType)
			}
		}
	}
	return ParsedValue{}, errors.New("expression is not a compile-time constant")
}
