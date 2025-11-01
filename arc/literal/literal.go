// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// ParsedValue represents a parsed literal value with its type.
type ParsedValue struct {
	// Value is the parsed Go value (e.g., int64, float64, string)
	Value any
	// Type is the Arc type of the value
	Type types.Type
}

// Parse parses a literal AST node and returns its value and type.
// It supports numeric literals (integer, float) and validates type compatibility.
// The targetType parameter specifies the expected type for conversion.
// Float-to-int conversions that lose precision will fail (matching Go semantics for constant conversions).
func Parse(
	literal parser.ILiteralContext,
	targetType types.Type,
) (ParsedValue, error) {
	if num := literal.NumericLiteral(); num != nil {
		return ParseNumeric(num, targetType)
	}
	if temp := literal.TemporalLiteral(); temp != nil {
		// TODO: Parse temporal literals when needed
		return ParsedValue{}, errors.New("temporal literals not yet supported")
	}
	if str := literal.STR_LITERAL(); str != nil {
		// TODO: Parse string literals when needed
		return ParsedValue{}, errors.New("string literals not yet supported")
	}
	if series := literal.SeriesLiteral(); series != nil {
		return ParsedValue{}, errors.New("series literals not supported for default values")
	}
	return ParsedValue{}, errors.New("unknown literal type")
}

// ParseNumeric parses a numeric literal (integer or float) and returns its value and type.
// It validates that the value fits within the target type's range and rejects lossy conversions.
func ParseNumeric(
	numLit parser.INumericLiteralContext,
	targetType types.Type,
) (ParsedValue, error) {
	if intLit := numLit.INTEGER_LITERAL(); intLit != nil {
		return parseIntegerLiteral(intLit.GetText(), targetType)
	}
	if floatLit := numLit.FLOAT_LITERAL(); floatLit != nil {
		return parseFloatLiteral(floatLit.GetText(), targetType)
	}
	return ParsedValue{}, errors.New("unknown numeric literal")
}

func parseIntegerLiteral(text string, targetType types.Type) (ParsedValue, error) {
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return ParsedValue{}, errors.Wrapf(err, "invalid integer literal: %s", text)
	}

	// If no target type specified, default to i64
	if !targetType.IsValid() {
		return ParsedValue{Value: value, Type: types.I64()}, nil
	}

	// Convert to appropriate type with overflow checking
	switch targetType.Kind {
	case types.KindI8:
		if value < math.MinInt8 || value > math.MaxInt8 {
			return ParsedValue{}, errors.Newf("value %d out of range for i8 (must be in [%d, %d])", value, math.MinInt8, math.MaxInt8)
		}
		return ParsedValue{Value: int8(value), Type: types.I8()}, nil
	case types.KindI16:
		if value < math.MinInt16 || value > math.MaxInt16 {
			return ParsedValue{}, errors.Newf("value %d out of range for i16 (must be in [%d, %d])", value, math.MinInt16, math.MaxInt16)
		}
		return ParsedValue{Value: int16(value), Type: types.I16()}, nil
	case types.KindI32:
		if value < math.MinInt32 || value > math.MaxInt32 {
			return ParsedValue{}, errors.Newf("value %d out of range for i32 (must be in [%d, %d])", value, math.MinInt32, math.MaxInt32)
		}
		return ParsedValue{Value: int32(value), Type: types.I32()}, nil
	case types.KindI64:
		return ParsedValue{Value: value, Type: types.I64()}, nil
	case types.KindU8:
		if value < 0 || value > math.MaxUint8 {
			return ParsedValue{}, errors.Newf("value %d out of range for u8 (must be in [0, %d])", value, math.MaxUint8)
		}
		return ParsedValue{Value: uint8(value), Type: types.U8()}, nil
	case types.KindU16:
		if value < 0 || value > math.MaxUint16 {
			return ParsedValue{}, errors.Newf("value %d out of range for u16 (must be in [0, %d])", value, math.MaxUint16)
		}
		return ParsedValue{Value: uint16(value), Type: types.U16()}, nil
	case types.KindU32:
		if value < 0 || value > math.MaxUint32 {
			return ParsedValue{}, errors.Newf("value %d out of range for u32 (must be in [0, %d])", value, math.MaxUint32)
		}
		return ParsedValue{Value: uint32(value), Type: types.U32()}, nil
	case types.KindU64:
		if value < 0 {
			return ParsedValue{}, errors.Newf("value %d out of range for u64 (must be non-negative)", value)
		}
		return ParsedValue{Value: uint64(value), Type: types.U64()}, nil
	case types.KindF32:
		return ParsedValue{Value: float32(value), Type: types.F32()}, nil
	case types.KindF64:
		return ParsedValue{Value: float64(value), Type: types.F64()}, nil
	default:
		// Default to i64 if type not recognized
		return ParsedValue{Value: value, Type: types.I64()}, nil
	}
}

func parseFloatLiteral(text string, targetType types.Type) (ParsedValue, error) {
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return ParsedValue{}, errors.Wrapf(err, "invalid float literal: %s", text)
	}

	// If no target type specified, default to f64
	if !targetType.IsValid() {
		return ParsedValue{Value: value, Type: types.F64()}, nil
	}

	// Convert to appropriate type
	switch targetType.Kind {
	case types.KindF32:
		// Check if value is representable as f32
		if math.Abs(value) > math.MaxFloat32 {
			return ParsedValue{}, errors.Newf("value %f out of range for f32", value)
		}
		return ParsedValue{Value: float32(value), Type: types.F32()}, nil
	case types.KindF64:
		return ParsedValue{Value: value, Type: types.F64()}, nil
	case types.KindI8:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf("cannot convert non-integer float %f to i8", value)
		}
		intVal := int64(value)
		if intVal < math.MinInt8 || intVal > math.MaxInt8 {
			return ParsedValue{}, errors.Newf("value %f out of range for i8", value)
		}
		return ParsedValue{Value: int8(intVal), Type: types.I8()}, nil
	case types.KindI16:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf("cannot convert non-integer float %f to i16", value)
		}
		intVal := int64(value)
		if intVal < math.MinInt16 || intVal > math.MaxInt16 {
			return ParsedValue{}, errors.Newf("value %f out of range for i16", value)
		}
		return ParsedValue{Value: int16(intVal), Type: types.I16()}, nil
	case types.KindI32:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf("cannot convert non-integer float %f to i32", value)
		}
		intVal := int64(value)
		if intVal < math.MinInt32 || intVal > math.MaxInt32 {
			return ParsedValue{}, errors.Newf("value %f out of range for i32", value)
		}
		return ParsedValue{Value: int32(intVal), Type: types.I32()}, nil
	case types.KindI64:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf("cannot convert non-integer float %f to i64", value)
		}
		return ParsedValue{Value: int64(value), Type: types.I64()}, nil
	case types.KindU8:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf("cannot convert non-integer float %f to u8", value)
		}
		intVal := int64(value)
		if intVal < 0 || intVal > math.MaxUint8 {
			return ParsedValue{}, errors.Newf("value %f out of range for u8", value)
		}
		return ParsedValue{Value: uint8(intVal), Type: types.U8()}, nil
	case types.KindU16:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf("cannot convert non-integer float %f to u16", value)
		}
		intVal := int64(value)
		if intVal < 0 || intVal > math.MaxUint16 {
			return ParsedValue{}, errors.Newf("value %f out of range for u16", value)
		}
		return ParsedValue{Value: uint16(intVal), Type: types.U16()}, nil
	case types.KindU32:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf("cannot convert non-integer float %f to u32", value)
		}
		intVal := int64(value)
		if intVal < 0 || intVal > math.MaxUint32 {
			return ParsedValue{}, errors.Newf("value %f out of range for u32", value)
		}
		return ParsedValue{Value: uint32(intVal), Type: types.U32()}, nil
	case types.KindU64:
		if value != math.Trunc(value) {
			return ParsedValue{}, errors.Newf("cannot convert non-integer float %f to u64", value)
		}
		if value < 0 {
			return ParsedValue{}, errors.Newf("value %f out of range for u64 (must be non-negative)", value)
		}
		return ParsedValue{Value: uint64(value), Type: types.U64()}, nil
	default:
		// Default to f64 if type not recognized
		return ParsedValue{Value: value, Type: types.F64()}, nil
	}
}
