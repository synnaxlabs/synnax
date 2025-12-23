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
		return types.Type{}, errors.New("str literals are not yet supported")
	}
	if series := ctx.AST.SeriesLiteral(); series != nil {
		return compileSeriesLiteral(context.Child(ctx, series))
	}
	return types.Type{}, errors.New("unknown literal type")
}

func compileSeriesLiteral(
	ctx context.Context[parser.ISeriesLiteralContext],
) (types.Type, error) {
	// Get the series type from the hint or TypeMap
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

	// Get expressions from the list
	var exprs []parser.IExpressionContext
	if exprList := ctx.AST.ExpressionList(); exprList != nil {
		exprs = exprList.AllExpression()
	}
	length := len(exprs)

	// Create the series: push length and call series_create_empty_<type>
	ctx.Writer.WriteI32Const(int32(length))
	createIdx, err := ctx.Imports.GetSeriesCreateEmpty(*elemType)
	if err != nil {
		return types.Type{}, err
	}
	ctx.Writer.WriteCall(createIdx)

	// For non-empty series, we need temporary storage for the handle.
	// Use the target local (set by variable declaration) for this.
	if length > 0 {
		tempLocal := ctx.TargetLocal
		if tempLocal < 0 {
			return types.Type{}, errors.New("series literal requires target local for temporary storage")
		}

		// For each element, use local.tee/local.get pattern to preserve handle
		// across SetElement calls (which return void)
		for i, expr := range exprs {
			if i == 0 {
				// Store handle in target local, keep on stack for first SetElement
				// LocalTee keeps the value on stack, so we don't need LocalGet here
				ctx.Writer.WriteLocalTee(tempLocal)
			} else {
				// For subsequent elements, load handle from local
				ctx.Writer.WriteLocalGet(tempLocal)
			}

			// Push index, compile value, call SetElement
			ctx.Writer.WriteI32Const(int32(i))
			_, err := Compile(context.Child(ctx, expr).WithHint(*elemType))
			if err != nil {
				return types.Type{}, err
			}

			setIdx, err := ctx.Imports.GetSeriesSetElement(*elemType)
			if err != nil {
				return types.Type{}, err
			}
			ctx.Writer.WriteCall(setIdx)
		}

		// Put handle back on stack as the expression result
		ctx.Writer.WriteLocalGet(tempLocal)
	}
	// If length == 0, handle is still on stack from CreateEmpty

	return seriesType, nil
}

func compileNumericLiteral(
	ctx context.Context[parser.INumericLiteralContext],
) (types.Type, error) {
	// Determine target type: prefer hint over TypeMap
	// The hint is set when we have an explicit type declaration (e.g., x i32 := 42)
	// The TypeMap contains inferred types from the analyzer
	targetType := ctx.Hint

	// Only use TypeMap if no hint was provided
	if !targetType.IsValid() {
		// TypeMap is keyed by the parent Literal context, so we look up parent
		if parent := ctx.AST.GetParent(); parent != nil {
			if litCtx, ok := parent.(parser.ILiteralContext); ok {
				if inferredType, ok := ctx.TypeMap[litCtx]; ok {
					targetType = inferredType
				}
			}
		}
	}

	// Clear the unit from target type - unit literals should always convert to SI base units.
	// The unit is preserved in the TypeMap for dimensional analysis but should not affect
	// the numeric value emitted (which is always in SI base units).
	targetType.Unit = nil

	// Use shared literal parser to parse and validate the value
	// This enforces Go-like semantics: no truncation for literal constants
	parsed, err := literal.ParseNumeric(ctx.AST, targetType)
	if err != nil {
		return types.Type{}, err
	}

	// Emit WASM bytecode based on the parsed type and value
	switch parsed.Type.Kind {
	case types.KindF32:
		ctx.Writer.WriteF32Const(parsed.Value.(float32))
	case types.KindF64:
		ctx.Writer.WriteF64Const(parsed.Value.(float64))
	case types.KindI8, types.KindI16, types.KindI32, types.KindU8, types.KindU16, types.KindU32:
		// WASM uses i32 for all 32-bit and smaller integers
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
