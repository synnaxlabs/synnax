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
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/errors"
)

// InferFromTypeContext infers a types.Type from a parser type context
func InferFromTypeContext(ctx parser.ITypeContext) (ir.Type, error) {
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

func inferPrimitiveType(ctx parser.IPrimitiveTypeContext) (ir.Type, error) {
	if numeric := ctx.NumericType(); numeric != nil {
		return inferNumericType(numeric)
	}
	if ctx.STRING() != nil {
		return ir.String{}, nil
	}
	return nil, errors.New("unknown primitive type")
}

func inferNumericType(ctx parser.INumericTypeContext) (ir.Type, error) {
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

func inferTemporalType(ctx parser.ITemporalTypeContext) (ir.Type, error) {
	text := ctx.GetText()
	switch text {
	case "timestamp":
		return ir.TimeStamp{}, nil
	case "timespan":
		return ir.TimeSpan{}, nil
	default:
		return nil, errors.New("unknown temporal type")
	}
}

func inferIntegerType(ctx parser.IIntegerTypeContext) (ir.Type, error) {
	text := ctx.GetText()
	switch text {
	case "i8":
		return ir.I8{}, nil
	case "i16":
		return ir.I16{}, nil
	case "i32":
		return ir.I32{}, nil
	case "i64":
		return ir.I64{}, nil
	case "u8":
		return ir.U8{}, nil
	case "u16":
		return ir.U16{}, nil
	case "u32":
		return ir.U32{}, nil
	case "u64":
		return ir.U64{}, nil
	default:
		return nil, errors.Newf("unknown integer type: %s", text)
	}
}

func inferFloatType(ctx parser.IFloatTypeContext) (ir.Type, error) {
	text := ctx.GetText()
	switch text {
	case "f32":
		return ir.F32{}, nil
	case "f64":
		return ir.F64{}, nil
	default:
		return nil, errors.Newf("unknown float type: %s", text)
	}
}

func inferChannelType(ctx parser.IChannelTypeContext) (ir.Type, error) {
	var valueType ir.Type
	var err error
	if primitive := ctx.PrimitiveType(); primitive != nil {
		valueType, err = inferPrimitiveType(primitive)
	} else if series := ctx.SeriesType(); series != nil {
		valueType, err = inferSeriesType(series)
	}
	if err != nil {
		return nil, err
	}
	return ir.Chan{ValueType: valueType}, nil
}

func inferSeriesType(ctx parser.ISeriesTypeContext) (ir.Type, error) {
	if primitive := ctx.PrimitiveType(); primitive != nil {
		valueType, err := inferPrimitiveType(primitive)
		if err != nil {
			return nil, err
		}
		return ir.Series{ValueType: valueType}, nil
	}
	return nil, errors.New("series must have primitive type")
}

// Compatible checks if two types are compatible for operations
func Compatible(
	t1, t2 ir.Type,
) bool {
	if t1 == nil || t2 == nil {
		return false
	}
	if t1Chan, ok := t1.(ir.Chan); ok {
		t1 = t1Chan.ValueType
	}
	if t2Chan, ok := t2.(ir.Chan); ok {
		t2 = t2Chan.ValueType
	}
	if t1.String() == t2.String() {
		return true
	}
	return false
}

func LiteralAssignmentCompatible(
	variableType, literalType ir.Type,
) bool {
	if variableType == nil || literalType == nil {
		return false
	}
	if variableType.String() == literalType.String() {
		return true
	}
	if ir.IsInteger(variableType) && ir.IsSignedInteger(literalType) {
		return true
	}
	if ir.IsFloat(variableType) && ir.IsNumeric(literalType) {
		return true
	}
	return false
}
