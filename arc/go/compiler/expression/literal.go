// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func compileLiteral(
	ctx context.Context[parser.ILiteralContext],
) (types.Type, error) {
	if num := ctx.AST.NumericLiteral(); num != nil {
		return compileNumericLiteral(context.Child(ctx, num))
	}
	if str := ctx.AST.STR_LITERAL(); str != nil {
		return compileStringLiteral(ctx, str.GetText())
	}
	if series := ctx.AST.SeriesLiteral(); series != nil {
		return compileSeriesLiteral(context.Child(ctx, series))
	}
	return types.Type{}, errors.New("unknown literal type")
}

func compileStringLiteral(
	ctx context.Context[parser.ILiteralContext],
	text string,
) (types.Type, error) {
	parsed, err := literal.ParseString(text, types.String())
	if err != nil {
		return types.Type{}, err
	}
	strBytes := []byte(parsed.Value.(string))
	offset := ctx.Module.AddData(strBytes)
	ctx.Writer.WriteI32Const(int32(offset))
	ctx.Writer.WriteI32Const(int32(len(strBytes)))
	if err := ctx.Resolver.EmitStringFromLiteral(ctx.Writer, ctx.WriterID); err != nil {
		return types.Type{}, err
	}
	return types.String(), nil
}

func compileSeriesLiteral(
	ctx context.Context[parser.ISeriesLiteralContext],
) (types.Type, error) {
	seriesType := ctx.Hint
	if !seriesType.IsValid() {
		if parent := ctx.AST.GetParent(); parent != nil {
			if litCtx, ok := parent.(parser.ILiteralContext); ok {
				if inferredType, ok := ctx.TypeMap[litCtx]; ok {
					seriesType = inferredType
				}
			}
		}
	}

	if seriesType.Kind != types.KindSeries {
		return types.Type{}, errors.New("series literal requires series type hint")
	}

	elemType := seriesType.Elem
	if elemType == nil {
		return types.Type{}, errors.New("series type missing element type")
	}

	var expressions []parser.IExpressionContext
	if exprList := ctx.AST.ExpressionList(); exprList != nil {
		expressions = exprList.AllExpression()
	}
	length := len(expressions)

	ctx.Writer.WriteI32Const(int32(length))
	if err := ctx.Resolver.EmitSeriesCreateEmpty(ctx.Writer, ctx.WriterID, *elemType); err != nil {
		return types.Type{}, err
	}

	for i, expr := range expressions {
		ctx.Writer.WriteI32Const(int32(i))
		if _, err := Compile(context.Child(ctx, expr).WithHint(*elemType)); err != nil {
			return types.Type{}, err
		}
		if err := ctx.Resolver.EmitSeriesSetElement(ctx.Writer, ctx.WriterID, *elemType); err != nil {
			return types.Type{}, err
		}
	}

	return seriesType, nil
}

func compileNumericLiteral(
	ctx context.Context[parser.INumericLiteralContext],
) (types.Type, error) {
	targetType := ctx.Hint

	if !targetType.IsValid() {
		if parent := ctx.AST.GetParent(); parent != nil {
			if litCtx, ok := parent.(parser.ILiteralContext); ok {
				if inferredType, ok := ctx.TypeMap[litCtx]; ok {
					targetType = inferredType
				}
			}
		}
	}

	targetType = targetType.Unwrap()
	targetType.Unit = nil

	parsed, err := literal.ParseNumeric(ctx.AST, targetType)
	if err != nil {
		return types.Type{}, err
	}

	switch parsed.Type.Kind {
	case types.KindF32:
		ctx.Writer.WriteF32Const(parsed.Value.(float32))
	case types.KindF64:
		ctx.Writer.WriteF64Const(parsed.Value.(float64))
	case types.KindI8, types.KindI16, types.KindI32, types.KindU8, types.KindU16, types.KindU32:
		var i32Val int32
		switch v := parsed.Value.(type) {
		case int8:
			i32Val = int32(v)
		case int16:
			i32Val = int32(v)
		case int32:
			i32Val = v
		case uint8:
			i32Val = int32(v)
		case uint16:
			i32Val = int32(v)
		case uint32:
			i32Val = int32(v)
		}
		ctx.Writer.WriteI32Const(i32Val)
	case types.KindI64:
		ctx.Writer.WriteI64Const(parsed.Value.(int64))
	case types.KindU64:
		ctx.Writer.WriteI64Const(int64(parsed.Value.(uint64)))
	default:
		return types.Type{}, errors.Newf("unsupported numeric type: %s", parsed.Type)
	}

	return parsed.Type, nil
}
