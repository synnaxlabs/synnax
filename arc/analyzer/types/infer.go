// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// InferFromTypeContext infers a types.Type from a parser type context
func InferFromTypeContext(ctx parser.ITypeContext) (types.Type, error) {
	if ctx == nil {
		return types.Type{}, nil
	}
	if primitive := ctx.PrimitiveType(); primitive != nil {
		return inferPrimitiveType(primitive)
	}
	if channel := ctx.ChannelType(); channel != nil {
		return inferChannelType(channel)
	}
	if series := ctx.SeriesType(); series != nil {
		return inferSeriesType(series)
	}
	return types.Type{}, errors.New("unknown type")
}

func inferPrimitiveType(ctx parser.IPrimitiveTypeContext) (types.Type, error) {
	if numeric := ctx.NumericType(); numeric != nil {
		return inferNumericType(numeric)
	}
	if ctx.STRING() != nil {
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
	if temporal := ctx.TemporalType(); temporal != nil {
		return inferTemporalType(temporal)
	}
	return types.Type{}, errors.New("unknown numeric type")
}

func inferTemporalType(ctx parser.ITemporalTypeContext) (types.Type, error) {
	text := ctx.GetText()
	switch text {
	case "timestamp":
		return types.TimeStamp(), nil
	case "timespan":
		return types.TimeSpan(), nil
	default:
		return types.Type{}, errors.New("unknown temporal type")
	}
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
	} else if series := ctx.SeriesType(); series != nil {
		valueType, err = inferSeriesType(series)
	}
	if err != nil {
		return types.Type{}, err
	}
	return types.Chan(valueType), nil
}

func inferSeriesType(ctx parser.ISeriesTypeContext) (types.Type, error) {
	if primitive := ctx.PrimitiveType(); primitive != nil {
		valueType, err := inferPrimitiveType(primitive)
		if err != nil {
			return types.Type{}, err
		}
		return types.Series(valueType), nil
	}
	return types.Type{}, errors.New("series must have primitive type")
}

func Compatible(
	t1, t2 types.Type,
) bool {
	if t1.Kind == types.KindInvalid || t2.Kind == types.KindInvalid {
		return false
	}
	if t1.Kind == types.KindChan && t1.ValueType != nil {
		t1 = *t1.ValueType
	}
	if t2.Kind == types.KindChan && t2.ValueType != nil {
		t2 = *t2.ValueType
	}
	return t1.String() == t2.String()
}

func LiteralAssignmentCompatible(
	variableType, literalType types.Type,
) bool {
	if variableType.Kind == types.KindInvalid || literalType.Kind == types.KindInvalid {
		return false
	}
	if variableType.String() == literalType.String() {
		return true
	}
	if variableType.IsInteger() && literalType.IsSignedInteger() {
		return true
	}
	return variableType.IsFloat() && literalType.IsNumeric()
}
