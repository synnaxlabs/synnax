// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// InferFromTypeContext extracts the concrete type from an Arc type annotation.
func InferFromTypeContext(ctx parser.ITypeContext) (types.Type, error) {
	if ctx == nil {
		return types.Type{}, nil
	}
	if primitive := ctx.PrimitiveType(); primitive != nil {
		t, err := inferPrimitiveType(primitive)
		if err != nil {
			return types.Type{}, err
		}
		if unitSuffix := ctx.UnitSuffix(); unitSuffix != nil {
			t, err = applyUnitSuffix(t, unitSuffix)
			if err != nil {
				return types.Type{}, err
			}
		}
		return t, nil
	}
	if channel := ctx.ChannelType(); channel != nil {
		return inferChannelType(channel)
	}
	if series := ctx.SeriesType(); series != nil {
		return inferSeriesType(series)
	}
	return types.Type{}, errors.New("unknown type")
}

func applyUnitSuffix(t types.Type, ctx parser.IUnitSuffixContext) (types.Type, error) {
	unitName := ctx.IDENTIFIER().GetText()
	unit, ok := units.Resolve(unitName)
	if !ok {
		return types.Type{}, errors.Newf("unknown unit: %s", unitName)
	}
	t.Unit = unit
	return t, nil
}

func inferPrimitiveType(ctx parser.IPrimitiveTypeContext) (types.Type, error) {
	if numeric := ctx.NumericType(); numeric != nil {
		return inferNumericType(numeric)
	}
	if ctx.STR() != nil {
		return types.String(), nil
	}
	return types.Type{}, errors.New("unknown primitive type")
}

func inferNumericType(ctx parser.INumericTypeContext) (types.Type, error) {
	if integer := ctx.IntegerType(); integer != nil {
		return inferIntegerType(integer)
	}
	if float := ctx.FloatType(); float != nil {
		return inferFloatType(float)
	}
	return types.Type{}, errors.New("unknown numeric type")
}

func inferIntegerType(ctx parser.IIntegerTypeContext) (types.Type, error) {
	text := ctx.GetText()
	switch text {
	case "i8":
		return types.I8(), nil
	case "i16":
		return types.I16(), nil
	case "i32":
		return types.I32(), nil
	case "i64":
		return types.I64(), nil
	case "u8":
		return types.U8(), nil
	case "u16":
		return types.U16(), nil
	case "u32":
		return types.U32(), nil
	case "u64":
		return types.U64(), nil
	default:
		return types.Type{}, errors.Newf("unknown integer type: %s", text)
	}
}

func inferFloatType(ctx parser.IFloatTypeContext) (types.Type, error) {
	text := ctx.GetText()
	switch text {
	case "f32":
		return types.F32(), nil
	case "f64":
		return types.F64(), nil
	default:
		return types.Type{}, errors.Newf("unknown float type: %s", text)
	}
}

func inferChannelType(ctx parser.IChannelTypeContext) (types.Type, error) {
	var valueType types.Type
	var err error
	if primitive := ctx.PrimitiveType(); primitive != nil {
		valueType, err = inferPrimitiveType(primitive)
		if err != nil {
			return types.Type{}, err
		}
		if unitSuffix := ctx.UnitSuffix(); unitSuffix != nil {
			valueType, err = applyUnitSuffix(valueType, unitSuffix)
			if err != nil {
				return types.Type{}, err
			}
		}
	} else if series := ctx.SeriesType(); series != nil {
		valueType, err = inferSeriesType(series)
		if err != nil {
			return types.Type{}, err
		}
	}
	return types.Chan(valueType), nil
}

func inferSeriesType(ctx parser.ISeriesTypeContext) (types.Type, error) {
	if primitive := ctx.PrimitiveType(); primitive != nil {
		valueType, err := inferPrimitiveType(primitive)
		if err != nil {
			return types.Type{}, err
		}
		if unitSuffix := ctx.UnitSuffix(); unitSuffix != nil {
			valueType, err = applyUnitSuffix(valueType, unitSuffix)
			if err != nil {
				return types.Type{}, err
			}
		}
		return types.Series(valueType), nil
	}
	return types.Type{}, errors.New("series must have primitive type")
}

// Compatible returns true if t1 and t2 are structurally compatible after unwrapping
// one level of channel or series wrapper. This is used during expression type inference
// where channels are automatically read to access their value type.
//
// The function ensures that wrapper types are preserved - channels only match channels,
// and series only match series. After unwrapping, the underlying types are compared
// using structural equality.
//
// This function should NOT be used for:
//   - Type variables (use the constraint system instead)
//   - Strict type checking (use Check function)
//
// Examples:
//
//	chan<int> ~ chan<int>    -> true
//	chan<int> ~ int          -> true (one is wrapped, one is not)
//	series<f32> ~ series<f32> -> true
//	chan<int> ~ series<int>  -> false (different wrapper types)
//	f32 ~ f64                -> false (incompatible base types)
func Compatible(t1, t2 types.Type) bool {
	if t1.Kind == types.KindInvalid || t2.Kind == types.KindInvalid {
		return false
	}

	// Type variables should use the constraint system, not this function
	if t1.Kind == types.KindVariable || t2.Kind == types.KindVariable {
		return false
	}

	// If both types are wrapped (channel or series), they must use the same wrapper
	if (t1.Kind == types.KindChan || t1.Kind == types.KindSeries) &&
		(t2.Kind == types.KindChan || t2.Kind == types.KindSeries) {
		if t1.Kind != t2.Kind {
			return false
		}
	}

	t1 = t1.Unwrap()
	t2 = t2.Unwrap()
	return types.Equal(t1, t2)
}

// LiteralAssignmentCompatible returns true if a literal of literalType can be assigned
// to variableType with implicit numeric widening.
func LiteralAssignmentCompatible(
	variableType, literalType types.Type,
) bool {
	if variableType.Kind == types.KindInvalid || literalType.Kind == types.KindInvalid {
		return false
	}
	// Unwrap channels to their value type, just like Compatible does
	variableType = variableType.Unwrap()
	literalType = literalType.Unwrap()
	if variableType.String() == literalType.String() {
		return true
	}
	if variableType.IsInteger() && literalType.IsSignedInteger() {
		return true
	}
	return variableType.IsFloat() && literalType.IsNumeric()
}
