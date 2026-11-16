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
	"github.com/synnaxlabs/arc/compiler/wasm"
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
		if inferred, ok := parentLiteralType(ctx.TypeMap, ctx.AST); ok {
			seriesType = inferred
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
		if _, err := Compile(ctx.Child(expr).WithHint(*elemType)); err != nil {
			return types.Type{}, err
		}
		ctx.Resolver.EmitSeriesSetElement(ctx.Writer, ctx.WriterID, *elemType)
	}

	return seriesType, nil
}

func compileNumericLiteral(
	ctx context.Context[parser.INumericLiteralContext],
) (types.Type, error) {
	targetType := ctx.Hint
	if !targetType.IsValid() || !targetType.IsNumeric() ||
		targetType.Kind == types.KindVariable {
		targetType, _ = parentLiteralType(ctx.TypeMap, ctx.AST)
	}
	targetType = targetType.Unwrap()
	targetType.Unit = nil
	parsed, err := literal.ParseNumeric(ctx.AST, targetType)
	if err != nil {
		return types.Type{}, err
	}
	if err = emitNumericConst(ctx.Writer, parsed); err != nil {
		return types.Type{}, err
	}
	return parsed.Type, nil
}

// parentLiteralType returns the type the analyzer inferred for the literal that holds
// node.
func parentLiteralType(
	typeMap map[antlr.ParserRuleContext]types.Type,
	node antlr.ParserRuleContext,
) (types.Type, bool) {
	litCtx, ok := node.GetParent().(parser.ILiteralContext)
	if !ok {
		return types.Type{}, false
	}
	t, ok := typeMap[litCtx]
	return t, ok
}

func emitNumericConst(w *wasm.Writer, parsed literal.ParsedValue) error {
	switch parsed.Type.Kind {
	case types.KindF32:
		w.WriteF32Const(parsed.Value.(float32))
	case types.KindF64:
		w.WriteF64Const(parsed.Value.(float64))
	case types.KindI8,
		types.KindI16,
		types.KindI32,
		types.KindU8,
		types.KindU16,
		types.KindU32:
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
		default:
			return unexpectedLiteralValue(parsed)
		}
		w.WriteI32Const(i32Val)
	case types.KindI64:
		switch v := parsed.Value.(type) {
		case int64:
			w.WriteI64Const(v)
		case telem.TimeSpan:
			w.WriteI64Const(int64(v))
		default:
			return unexpectedLiteralValue(parsed)
		}
	case types.KindU64:
		w.WriteI64Const(int64(parsed.Value.(uint64)))
	default:
		return errors.Newf("unsupported numeric type: %s", parsed.Type)
	}
	return nil
}

func unexpectedLiteralValue(parsed literal.ParsedValue) error {
	return errors.Newf(
		"unexpected value type %T for %s literal",
		parsed.Value,
		parsed.Type.Kind,
	)
}
