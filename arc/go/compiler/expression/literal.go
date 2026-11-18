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
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

func compileLiteral(
	ctx context.Context[parser.ILiteralContext],
) (types.Type, error) {
	if num := ctx.AST.NumericLiteral(); num != nil {
		return compileNumericLiteral(ctx.Child(num))
	}
	if strTerm := parser.StringTerminal(ctx.AST); strTerm != nil {
		return compileStringLiteral(ctx, strTerm)
	}
	if series := ctx.AST.SeriesLiteral(); series != nil {
		return compileSeriesLiteral(ctx.Child(series))
	}
	if boolLit := ctx.AST.BooleanLiteral(); boolLit != nil {
		if boolLit.TRUE() != nil {
			ctx.Writer.WriteI32Const(1)
		} else {
			ctx.Writer.WriteI32Const(0)
		}
		return types.Bool(), nil
	}
	return types.Type{}, errors.New("unknown literal type")
}

func compileStringLiteral(
	ctx context.Context[parser.ILiteralContext],
	strTerm antlr.TerminalNode,
) (types.Type, error) {
	text := strTerm.GetText()
	body, flags, ok := literal.StripQuotes(text)
	if !ok {
		return types.Type{}, errors.Newf("invalid string literal: %s", text)
	}
	if flags.Format {
		return EmitFmtSegments(ctx, strTerm)
	}
	value := body
	if !flags.Raw {
		var err error
		value, err = literal.UnescapeString(body, flags.Multi)
		if err != nil {
			return types.Type{}, errors.Wrapf(err, "invalid string literal %s", text)
		}
	}
	emitLiteralSegment(ctx, value)
	return types.String(), nil
}

func compileSeriesLiteral(
	ctx context.Context[parser.ISeriesLiteralContext],
) (types.Type, error) {
	seriesType := ctx.Hint
	if !seriesType.IsValid() {
		if t, ok := inferredLiteralType(ctx.AST.GetParent(), ctx.TypeMap); ok {
			seriesType = t
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
	ctx.Resolver.EmitSeriesCreateEmpty(ctx.Writer, ctx.WriterID, *elemType)

	for i, expr := range expressions {
		ctx.Writer.WriteI32Const(int32(i))
		t, err := Compile(ctx.Child(expr).WithHint(*elemType))
		if err != nil {
			return types.Type{}, err
		}
		if t = t.UnwrapChan(); types.WriteConverts(t.Unit, elemType.Unit) {
			if err = EmitCast(ctx, t, *elemType); err != nil {
				return types.Type{}, err
			}
		}
		ctx.Resolver.EmitSeriesSetElement(ctx.Writer, ctx.WriterID, *elemType)
	}

	return seriesType, nil
}

func compileNumericLiteral(
	ctx context.Context[parser.INumericLiteralContext],
) (types.Type, error) {
	targetType := ctx.Hint
	inferred, _ := inferredLiteralType(ctx.AST.GetParent(), ctx.TypeMap)
	if !targetType.IsValid() || !targetType.IsNumeric() ||
		targetType.Kind == types.KindVariable {
		targetType = inferred
	}
	targetType = targetType.Unwrap()
	// A literal beside a value of another dimension keeps its number.
	if types.UnitsIncompatible(inferred.Unit, targetType.Unit) {
		targetType.Unit = nil
	}

	parsed, err := literal.ParseNumeric(ctx.AST, targetType)
	if err != nil {
		return types.Type{}, err
	}

	if err = emitNumericValue(ctx, parsed); err != nil {
		return types.Type{}, err
	}
	return parsed.Type, nil
}

func emitNumericValue(
	ctx context.Context[parser.INumericLiteralContext],
	parsed literal.ParsedValue,
) error {
	unexpected := func() error {
		return errors.Newf(
			"unexpected value type %T for %s literal",
			parsed.Value,
			parsed.Type.Kind,
		)
	}
	switch parsed.Type.Kind {
	case types.KindF32:
		v, ok := parsed.Value.(float32)
		if !ok {
			return unexpected()
		}
		ctx.Writer.WriteF32Const(v)
	case types.KindF64:
		v, ok := parsed.Value.(float64)
		if !ok {
			return unexpected()
		}
		ctx.Writer.WriteF64Const(v)
	case types.KindI8,
		types.KindI16,
		types.KindI32,
		types.KindU8,
		types.KindU16,
		types.KindU32:
		v, ok := int32Value(parsed.Value)
		if !ok {
			return unexpected()
		}
		ctx.Writer.WriteI32Const(v)
	case types.KindI64:
		v, ok := int64Value(parsed.Value)
		if !ok {
			return unexpected()
		}
		ctx.Writer.WriteI64Const(v)
	case types.KindU64:
		v, ok := parsed.Value.(uint64)
		if !ok {
			return unexpected()
		}
		ctx.Writer.WriteI64Const(int64(v))
	case types.KindInvalid, types.KindString, types.KindChan, types.KindSeries,
		types.KindVariable, types.KindNumericConstant, types.KindIntegerConstant,
		types.KindFloatConstant, types.KindExactIntegerFloatConstant,
		types.KindFunction, types.KindSequence, types.KindStage, types.KindVarRef,
		types.KindBool:
		fallthrough
	default:
		return errors.Newf("unsupported numeric type: %s", parsed.Type)
	}
	return nil
}

// inferredLiteralType returns the type the analyzer gave the literal that wraps a
// series or numeric literal.
func inferredLiteralType(
	parent antlr.Tree,
	typeMap map[antlr.ParserRuleContext]types.Type,
) (types.Type, bool) {
	litCtx, ok := parent.(parser.ILiteralContext)
	if !ok {
		return types.Type{}, false
	}
	t, ok := typeMap[litCtx]
	return t, ok
}

// int64Value returns the int64 that WASM stores an i64 literal or a time span in.
func int64Value(v any) (int64, bool) {
	switch v := v.(type) {
	case int64:
		return v, true
	case telem.TimeSpan:
		return int64(v), true
	default:
		return 0, false
	}
}

// int32Value widens a value of a 32-bit or narrower integer type to the int32 that
// WASM stores it in.
func int32Value(v any) (int32, bool) {
	switch v := v.(type) {
	case int8:
		return int32(v), true
	case int16:
		return int32(v), true
	case int32:
		return v, true
	case uint8:
		return int32(v), true
	case uint16:
		return int32(v), true
	case uint32:
		return int32(v), true
	default:
		return 0, false
	}
}
