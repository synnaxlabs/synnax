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
func InferFromTypeContext(ctx text.ITypeContext) (types.Type, error) {
	if ctx == nil {
		return nil, nil
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
	return nil, errors.New("unknown type")
}

func inferPrimitiveType(ctx text.IPrimitiveTypeContext) (types.Type, error) {
	if numeric := ctx.NumericType(); numeric != nil {
		return inferNumericType(numeric)
	}
	if ctx.STRING() != nil {
		return types.String{}, nil
	}
	return nil, errors.New("unknown primitive type")
}

func inferNumericType(ctx text.INumericTypeContext) (types.Type, error) {
	if integer := ctx.IntegerType(); integer != nil {
		return inferIntegerType(integer)
	}
	if float := ctx.FloatType(); float != nil {
		return inferFloatType(float)
	}
	if temporal := ctx.TemporalType(); temporal != nil {
		return inferTemporalType(temporal)
	}
	return nil, errors.New("unknown numeric type")
}

func inferTemporalType(ctx text.ITemporalTypeContext) (types.Type, error) {
	text := ctx.GetText()
	switch text {
	case "timestamp":
		return types.TimeStamp{}, nil
	case "timespan":
		return types.TimeSpan{}, nil
	default:
		return nil, errors.New("unknown temporal type")
	}
}

func inferIntegerType(ctx text.IIntegerTypeContext) (types.Type, error) {
	text := ctx.GetText()
	switch text {
	case "i8":
		return types.I8{}, nil
	case "i16":
		return types.I16{}, nil
	case "i32":
		return types.I32{}, nil
	case "i64":
		return types.I64{}, nil
	case "u8":
		return types.U8{}, nil
	case "u16":
		return types.U16{}, nil
	case "u32":
		return types.U32{}, nil
	case "u64":
		return types.U64{}, nil
	default:
		return nil, errors.Newf("unknown integer type: %s", text)
	}
}

func inferFloatType(ctx text.IFloatTypeContext) (types.Type, error) {
	text := ctx.GetText()
	switch text {
	case "f32":
		return types.F32{}, nil
	case "f64":
		return types.F64{}, nil
	default:
		return nil, errors.Newf("unknown float type: %s", text)
	}
}

func inferChannelType(ctx text.IChannelTypeContext) (types.Type, error) {
	var valueType types.Type
	var err error
	if primitive := ctx.PrimitiveType(); primitive != nil {
		valueType, err = inferPrimitiveType(primitive)
	} else if series := ctx.SeriesType(); series != nil {
		valueType, err = inferSeriesType(series)
	}
	if err != nil {
		return nil, err
	}
	return types.Chan{ValueType: valueType}, nil
}

func inferSeriesType(ctx text.ISeriesTypeContext) (types.Type, error) {
	if primitive := ctx.PrimitiveType(); primitive != nil {
		valueType, err := inferPrimitiveType(primitive)
		if err != nil {
			return nil, err
		}
		return types.Series{ValueType: valueType}, nil
	}
	return nil, errors.New("series must have primitive type")
}

// Compatible checks if two types are compatible for operations
func Compatible(
	t1, t2 types.Type,
) bool {
	if t1 == nil || t2 == nil {
		return false
	}
	if t1Chan, ok := t1.(types.Chan); ok {
		t1 = t1Chan.ValueType
	}
	if t2Chan, ok := t2.(types.Chan); ok {
		t2 = t2Chan.ValueType
	}
	if t1.String() == t2.String() {
		return true
	}
	return false
}

func LiteralAssignmentCompatible(
	variableType, literalType types.Type,
) bool {
	if variableType == nil || literalType == nil {
		return false
	}
	if variableType.String() == literalType.String() {
		return true
	}
	if types.IsInteger(variableType) && types.IsSignedInteger(literalType) {
		return true
	}
	if types.IsFloat(variableType) && types.IsNumeric(literalType) {
		return true
	}
	return false
}
