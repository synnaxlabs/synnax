// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"strconv"

	"github.com/synnaxlabs/arc/compiler/context"
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
	if temp := ctx.AST.TemporalLiteral(); temp != nil {
		return types.TimeSpan(), nil
	}
	if str := ctx.AST.STRING_LITERAL(); str != nil {
		return types.Type{}, errors.New("string literals are not yet supported")
	}
	if series := ctx.AST.SeriesLiteral(); series != nil {
		return types.Type{}, errors.New("series literals not yet implemented")
	}
	return types.Type{}, errors.New("unknown literal type")
}

func compileNumericLiteral(
	ctx context.Context[parser.INumericLiteralContext],
) (types.Type, error) {
	if intLit := ctx.AST.INTEGER_LITERAL(); intLit != nil {
		text := intLit.GetText()
		value, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return types.Type{}, errors.Newf("invalid integer literal: %s", text)
		}
		switch ctx.Hint.Kind {
		case types.KindF32:
			ctx.Writer.WriteF32Const(float32(value))
		case types.KindF64:
			ctx.Writer.WriteF64Const(float64(value))
		case types.KindI32:
			ctx.Writer.WriteI32Const(int32(value))
		default:
			ctx.Writer.WriteI64Const(value)
		}
		return ctx.Hint, nil
	}
	if floatLit := ctx.AST.FLOAT_LITERAL(); floatLit != nil {
		text := floatLit.GetText()
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return types.Type{}, errors.Newf("invalid float literal: %s", text)
		}
		switch ctx.Hint.Kind {
		case types.KindF64:
			ctx.Writer.WriteF32Const(float32(value))
		default:
			ctx.Writer.WriteF64Const(value)
		}
		return ctx.Hint, nil
	}
	return types.Type{}, errors.New("unknown numeric literal")
}
