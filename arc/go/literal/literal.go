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
	// Truncated is set when the unit conversion dropped a part of an integer.
	Truncated bool
}

// Parse parses a literal AST node and returns its value and type. It supports numeric
// literals (integer, float, with optional unit suffix) and validates type
// compatibility. The targetType parameter specifies the expected type for conversion.
// Float-to-int conversions that lose precision will fail (matching Go semantics for
// constant conversions). A unit literal converts to the unit of targetType, and keeps
// its number as written when targetType has no unit.
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

	targetType = types.Concrete(targetType)
	if !targetType.IsValid() {
		targetType.Kind = types.KindF64
		if isIntLiteral {
			targetType.Kind = types.KindI64
		}
	}
	// A target with no unit takes the number as written.
	if targetType.Unit == nil {
		return convertToTargetKind(numericValue, targetType, unit)
	}
	return Convert(numericValue, unit, targetType)
}

// Convert returns value, a number in the unit from, as a value of targetType in its
// unit. An integer does not hold a part of a unit, so it truncates toward zero.
func Convert(
	value float64,
	from *types.Unit,
	targetType types.Type,
) (ParsedValue, error) {
	factor, err := units.ScaleFactor(from, targetType.Unit)
	if err != nil {
		return ParsedValue{}, err
	}
	value *= factor
	truncated := targetType.IsInteger() && factor != math.Trunc(factor) &&
		!IsExactInteger(value)
	if truncated {
		value = math.Trunc(value)
	}
	parsed, err := convertToTargetKind(value, targetType, targetType.Unit)
	parsed.Truncated = truncated
	return parsed, err
}

// parseIntegerLiteral converts an integer value to the target type.
// It takes int64 directly to preserve full precision for large integer literals.
func parseIntegerLiteral(intValue int64, targetType types.Type) (ParsedValue, error) {
	// If no target type specified, default to i64
	if !targetType.IsValid() {
		return ParsedValue{Value: intValue, Type: types.I64()}, nil
	}

	// Convert to appropriate type with overflow checking
	switch targetType.Kind {
	case types.KindI8:
		if intValue < math.MinInt8 || intValue > math.MaxInt8 {
			return ParsedValue{}, errors.Newf(
				"value %d out of range for i8 (must be in [%d, %d])",
				intValue,
				math.MinInt8,
				math.MaxInt8,
			)
		}
		return ParsedValue{Value: int8(intValue), Type: types.I8()}, nil
	case types.KindI16:
		if intValue < math.MinInt16 || intValue > math.MaxInt16 {
			return ParsedValue{}, errors.Newf(
				"value %d out of range for i16 (must be in [%d, %d])",
				intValue,
				math.MinInt16,
				math.MaxInt16,
			)
		}
		return ParsedValue{Value: int16(intValue), Type: types.I16()}, nil
	case types.KindI32:
		if intValue < math.MinInt32 || intValue > math.MaxInt32 {
			return ParsedValue{}, errors.Newf(
				"value %d out of range for i32 (must be in [%d, %d])",
				intValue,
				math.MinInt32,
				math.MaxInt32,
			)
		}
		return ParsedValue{Value: int32(intValue), Type: types.I32()}, nil
	case types.KindI64:
		if isTime(targetType.Unit) {
			return ParsedValue{Value: telem.TimeSpan(intValue), Type: types.I64()}, nil
		}
		return ParsedValue{Value: intValue, Type: types.I64()}, nil
	case types.KindU8:
		if intValue < 0 || intValue > math.MaxUint8 {
			return ParsedValue{}, errors.Newf(
				"value %d out of range for u8 (must be in [0, %d])",
				intValue,
				math.MaxUint8,
			)
		}
		return ParsedValue{Value: uint8(intValue), Type: types.U8()}, nil
	case types.KindU16:
		if intValue < 0 || intValue > math.MaxUint16 {
			return ParsedValue{}, errors.Newf(
				"value %d out of range for u16 (must be in [0, %d])",
				intValue,
				math.MaxUint16,
			)
		}
		return ParsedValue{Value: uint16(intValue), Type: types.U16()}, nil
	case types.KindU32:
		if intValue < 0 || intValue > math.MaxUint32 {
			return ParsedValue{}, errors.Newf(
				"value %d out of range for u32 (must be in [0, %d])",
				intValue,
				math.MaxUint32,
			)
		}
		return ParsedValue{Value: uint32(intValue), Type: types.U32()}, nil
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
		// Default to i64 if type not recognized
		return ParsedValue{Value: intValue, Type: types.I64()}, nil
	}
}

// parseFloatLiteral converts a float value to the target type.
func parseFloatLiteral(value float64, targetType types.Type) (ParsedValue, error) {
	if !targetType.IsValid() {
		return ParsedValue{Value: value, Type: types.F64()}, nil
	}

	switch targetType.Kind {
	case types.KindF32:
		if math.Abs(value) > math.MaxFloat32 {
			return ParsedValue{}, errors.Newf("value %f out of range for f32", value)
		}
		return ParsedValue{Value: float32(value), Type: types.F32()}, nil
	case types.KindF64:
		return ParsedValue{Value: value, Type: types.F64()}, nil
	case types.KindI8:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf(
				"cannot convert non-integer float %f to i8",
				value,
			)
		}
		intVal := int64(value)
		if intVal < math.MinInt8 || intVal > math.MaxInt8 {
			return ParsedValue{}, errors.Newf("value %f out of range for i8", value)
		}
		return ParsedValue{Value: int8(intVal), Type: types.I8()}, nil
	case types.KindI16:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf(
				"cannot convert non-integer float %f to i16",
				value,
			)
		}
		intVal := int64(value)
		if intVal < math.MinInt16 || intVal > math.MaxInt16 {
			return ParsedValue{}, errors.Newf("value %f out of range for i16", value)
		}
		return ParsedValue{Value: int16(intVal), Type: types.I16()}, nil
	case types.KindI32:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf(
				"cannot convert non-integer float %f to i32",
				value,
			)
		}
		intVal := int64(value)
		if intVal < math.MinInt32 || intVal > math.MaxInt32 {
			return ParsedValue{}, errors.Newf("value %f out of range for i32", value)
		}
		return ParsedValue{Value: int32(intVal), Type: types.I32()}, nil
	case types.KindI64:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf(
				"cannot convert non-integer float %f to i64",
				value,
			)
		}
		return ParsedValue{Value: int64(value), Type: types.I64()}, nil
	case types.KindU8:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf(
				"cannot convert non-integer float %f to u8",
				value,
			)
		}
		intVal := int64(value)
		if intVal < 0 || intVal > math.MaxUint8 {
			return ParsedValue{}, errors.Newf("value %f out of range for u8", value)
		}
		return ParsedValue{Value: uint8(intVal), Type: types.U8()}, nil
	case types.KindU16:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf(
				"cannot convert non-integer float %f to u16",
				value,
			)
		}
		intVal := int64(value)
		if intVal < 0 || intVal > math.MaxUint16 {
			return ParsedValue{}, errors.Newf("value %f out of range for u16", value)
		}
		return ParsedValue{Value: uint16(intVal), Type: types.U16()}, nil
	case types.KindU32:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf(
				"cannot convert non-integer float %f to u32",
				value,
			)
		}
		intVal := int64(value)
		if intVal < 0 || intVal > math.MaxUint32 {
			return ParsedValue{}, errors.Newf("value %f out of range for u32", value)
		}
		return ParsedValue{Value: uint32(intVal), Type: types.U32()}, nil
	case types.KindU64:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf(
				"cannot convert non-integer float %f to u64",
				value,
			)
		}
		if value < 0 {
			return ParsedValue{}, errors.Newf(
				"value %f out of range for u64 (must be non-negative)",
				value,
			)
		}
		return ParsedValue{Value: uint64(value), Type: types.U64()}, nil
	default:
		return ParsedValue{Value: value, Type: types.F64()}, nil
	}
}

// convertToTargetKind converts a float64 value to the target type's kind. Uses Go-like
// constant conversion semantics: errors on fractional parts for integer types.
func convertToTargetKind(
	value float64,
	targetType types.Type,
	sourceUnit *types.Unit,
) (ParsedValue, error) {
	resultType := types.Type{Kind: targetType.Kind, Unit: sourceUnit}

	// For integer target types, check for fractional part (Go-like constant semantics)
	// Use a small epsilon for floating-point comparison to handle precision issues
	switch targetType.Kind {
	case types.KindI8, types.KindI16, types.KindI32, types.KindI64,
		types.KindU8, types.KindU16, types.KindU32, types.KindU64:
		if !IsExactInteger(value) {
			return ParsedValue{}, errors.Newf(
				"cannot convert %g to %s: value has fractional part",
				value, targetType.Kind,
			)
		}
		// Round to nearest integer to clean up floating-point precision issues
		value = math.Round(value)
	}

	switch targetType.Kind {
	case types.KindI8:
		intVal := int64(value)
		if intVal < math.MinInt8 || intVal > math.MaxInt8 {
			return ParsedValue{}, errors.Newf("value %g out of range for i8", value)
		}
		return ParsedValue{Value: int8(intVal), Type: resultType}, nil
	case types.KindI16:
		intVal := int64(value)
		if intVal < math.MinInt16 || intVal > math.MaxInt16 {
			return ParsedValue{}, errors.Newf("value %g out of range for i16", value)
		}
		return ParsedValue{Value: int16(intVal), Type: resultType}, nil
	case types.KindI32:
		intVal := int64(value)
		if intVal < math.MinInt32 || intVal > math.MaxInt32 {
			return ParsedValue{}, errors.Newf("value %g out of range for i32", value)
		}
		return ParsedValue{Value: int32(intVal), Type: resultType}, nil
	case types.KindI64:
		if isTime(resultType.Unit) {
			return ParsedValue{Value: telem.TimeSpan(value), Type: resultType}, nil
		}
		return ParsedValue{Value: int64(value), Type: resultType}, nil
	case types.KindU8:
		intVal := int64(value)
		if intVal < 0 || intVal > math.MaxUint8 {
			return ParsedValue{}, errors.Newf("value %g out of range for u8", value)
		}
		return ParsedValue{Value: uint8(intVal), Type: resultType}, nil
	case types.KindU16:
		intVal := int64(value)
		if intVal < 0 || intVal > math.MaxUint16 {
			return ParsedValue{}, errors.Newf("value %g out of range for u16", value)
		}
		return ParsedValue{Value: uint16(intVal), Type: resultType}, nil
	case types.KindU32:
		intVal := int64(value)
		if intVal < 0 || intVal > math.MaxUint32 {
			return ParsedValue{}, errors.Newf("value %g out of range for u32", value)
		}
		return ParsedValue{Value: uint32(intVal), Type: resultType}, nil
	case types.KindU64:
		if value < 0 {
			return ParsedValue{}, errors.Newf(
				"value %g out of range for u64 (must be non-negative)",
				value,
			)
		}
		return ParsedValue{Value: uint64(value), Type: resultType}, nil
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

func isTime(unit *types.Unit) bool {
	return unit != nil && unit.Dimensions.Equal(types.TimeSpan().Unit.Dimensions)
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
