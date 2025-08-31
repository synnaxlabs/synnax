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
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/x/errors"
)

// InferFromTypeContext infers a symbol.Type from a parser type context
func InferFromTypeContext(ctx parser.ITypeContext) (symbol.Type, error) {
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

func inferPrimitiveType(ctx parser.IPrimitiveTypeContext) (symbol.Type, error) {
	if numeric := ctx.NumericType(); numeric != nil {
		return inferNumericType(numeric)
	}
	if ctx.STRING() != nil {
		return String{}, nil
	}
	return nil, errors.New("unknown primitive type")
}

func inferNumericType(ctx parser.INumericTypeContext) (symbol.Type, error) {
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

func inferTemporalType(ctx parser.ITemporalTypeContext) (symbol.Type, error) {
	text := ctx.GetText()
	switch text {
	case "timestamp":
		return TimeStamp{}, nil
	case "timespan":
		return TimeSpan{}, nil
	default:
		return nil, errors.New("unknown temporal type")
	}
}

func inferIntegerType(ctx parser.IIntegerTypeContext) (symbol.Type, error) {
	text := ctx.GetText()
	switch text {
	case "i8":
		return I8{}, nil
	case "i16":
		return I16{}, nil
	case "i32":
		return I32{}, nil
	case "i64":
		return I64{}, nil
	case "u8":
		return U8{}, nil
	case "u16":
		return U16{}, nil
	case "u32":
		return U32{}, nil
	case "u64":
		return U64{}, nil
	default:
		return nil, errors.Newf("unknown integer type: %s", text)
	}
}

func inferFloatType(ctx parser.IFloatTypeContext) (symbol.Type, error) {
	text := ctx.GetText()
	switch text {
	case "f32":
		return F32{}, nil
	case "f64":
		return F64{}, nil
	default:
		return nil, errors.Newf("unknown float type: %s", text)
	}
}

func inferChannelType(ctx parser.IChannelTypeContext) (symbol.Type, error) {
	var valueType symbol.Type
	var err error
	if primitive := ctx.PrimitiveType(); primitive != nil {
		valueType, err = inferPrimitiveType(primitive)
	} else if series := ctx.SeriesType(); series != nil {
		valueType, err = inferSeriesType(series)
	}
	if err != nil {
		return nil, err
	}
	return Chan{ValueType: valueType}, nil
}

func inferSeriesType(ctx parser.ISeriesTypeContext) (symbol.Type, error) {
	if primitive := ctx.PrimitiveType(); primitive != nil {
		valueType, err := inferPrimitiveType(primitive)
		if err != nil {
			return nil, err
		}
		return Series{ValueType: valueType}, nil
	}
	return nil, errors.New("series must have primitive type")
}

// Compatible checks if two types are compatible for operations
func Compatible(
	t1, t2 symbol.Type,
) bool {
	if t1 == nil || t2 == nil {
		return false
	}
	if t1Chan, ok := t1.(Chan); ok {
		t1 = t1Chan.ValueType
	}
	if t2Chan, ok := t2.(Chan); ok {
		t2 = t2Chan.ValueType
	}
	// Same type is always compatible
	if t1.String() == t2.String() {
		return true
	}
	return false
}

func LiteralCompatible(
	t1, t2 symbol.Type,
) bool {
	if t1 == nil || t2 == nil {
		return false
	}
	if t1.String() == t2.String() {
		return true
	}
	if IsInteger(t1) && IsSignedInteger(t2) {
		return true
	}
	if IsFloat(t1) && IsFloat(t2) {
		return true
	}
	return false
}
