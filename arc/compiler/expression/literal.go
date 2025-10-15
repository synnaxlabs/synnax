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

// compileLiteral compiles a literal value
func compileLiteral(
	ctx context.Context[parser.ILiteralContext],
) (types.Type, error) {
	if num := ctx.AST.NumericLiteral(); num != nil {
		return compileNumericLiteral(context.Child(ctx, num))
	}
	if temp := ctx.AST.TemporalLiteral(); temp != nil {
		return types.TimeSpan{}, nil
	}
	if str := ctx.AST.STRING_LITERAL(); str != nil {
		return nil, errors.New("string literals are not yet supported")
	}
	if series := ctx.AST.SeriesLiteral(); series != nil {
		return nil, errors.New("series literals not yet implemented")
	}
	return nil, errors.New("unknown literal type")
}

func compileNumericLiteral(
	ctx context.Context[parser.INumericLiteralContext],
) (types.Type, error) {
	if intLit := ctx.AST.INTEGER_LITERAL(); intLit != nil {
		text := intLit.GetText()
		value, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return nil, errors.Newf("invalid integer literal: %s", text)
		}

		// Check if we have a hint that suggests a float type
		switch ctx.Hint.(type) {
		case types.F32:
			// Coerce integer literal to f32
			ctx.Writer.WriteF32Const(float32(value))
			return types.F32{}, nil
		case types.F64:
			// Coerce integer literal to f64
			ctx.Writer.WriteF64Const(float64(value))
			return types.F64(), nil
		case types.I32:
			// Coerce to i32
			ctx.Writer.WriteI32Const(int32(value))
			return types.I32{}, nil
		default:
			// Default to i64
			ctx.Writer.WriteI64Const(value)
			return types.I64{}, nil
		}
	}
	if floatLit := ctx.AST.FLOAT_LITERAL(); floatLit != nil {
		text := floatLit.GetText()
		value, err := strconv.ParseFloat(text, 64)
		if err != nil {
			return nil, errors.Newf("invalid float literal: %s", text)
		}

		// Check if we have a hint that suggests f32
		switch ctx.Hint.(type) {
		case types.F32:
			// Coerce to f32
			ctx.Writer.WriteF32Const(float32(value))
			return types.F32{}, nil
		default:
			// Default to f64
			ctx.Writer.WriteF64Const(value)
			return types.F64(), nil
		}
	}
	return nil, errors.New("unknown numeric literal")
}
