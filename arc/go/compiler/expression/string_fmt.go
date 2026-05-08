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
	"github.com/synnaxlabs/arc/fmtstring"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// compileStringFmt lowers a `string.fmt(format)` call when format is a string
// literal: literal segments become string.from_literal pushes, placeholder
// segments compile their expression and (for numerics) convert to string via
// EmitNumericToString. Segments are joined left-to-right with string.concat.
//
// When format is not a literal the call falls through to the regular call
// path so the runtime helper is invoked.
func compileStringFmt(
	ctx context.Context[parser.IPostfixExpressionContext],
	scope *symbol.Scope,
	funcCall parser.IFunctionCallSuffixContext,
) (types.Type, error) {
	argList := funcCall.ArgumentList()
	if argList == nil {
		return types.Type{}, errors.New("string.fmt requires a format argument")
	}
	args := argList.AllExpression()
	if len(args) == 0 {
		return types.Type{}, errors.New("string.fmt requires a format argument")
	}
	formatArg := args[0]
	litNode := parser.GetLiteral(formatArg)
	if litNode == nil {
		return compileFunctionCallExpr(ctx, "string.fmt", scope, funcCall)
	}
	parsed, err := literal.Parse(litNode, types.String())
	if err != nil {
		return types.Type{}, err
	}
	body, ok := parsed.Value.(string)
	if !ok {
		return types.Type{}, errors.New("string.fmt format argument must be a string literal")
	}
	segments, err := fmtstring.Parse(body)
	if err != nil {
		return types.Type{}, err
	}
	if len(segments) == 0 {
		emitLiteralSegment(ctx, "")
		return types.String(), nil
	}
	if err := emitFmtSegment(ctx, segments[0]); err != nil {
		return types.Type{}, err
	}
	for _, seg := range segments[1:] {
		if err := emitFmtSegment(ctx, seg); err != nil {
			return types.Type{}, err
		}
		ctx.Resolver.EmitStringConcat(ctx.Writer, ctx.WriterID)
	}
	return types.String(), nil
}

func emitFmtSegment(
	ctx context.Context[parser.IPostfixExpressionContext],
	seg fmtstring.Segment,
) error {
	if !seg.IsPlaceholder {
		emitLiteralSegment(ctx, seg.Text)
		return nil
	}
	expr, diags := parser.ParseExpression(seg.Text)
	if diags != nil && !diags.Ok() {
		return errors.Newf("invalid placeholder %q: %s", seg.Text, diags.String())
	}
	t, err := Compile(context.Child(ctx, expr).WithHint(types.Type{}))
	if err != nil {
		return err
	}
	if t.Kind == types.KindString {
		return nil
	}
	if t.IsNumeric() {
		return ctx.Resolver.EmitNumericToString(ctx.Writer, ctx.WriterID, t)
	}
	return errors.Newf(
		"placeholder %q has type %s; only numeric and string types are supported",
		seg.Text, t,
	)
}

func emitLiteralSegment(
	ctx context.Context[parser.IPostfixExpressionContext],
	text string,
) {
	bytes := []byte(text)
	offset := ctx.Module.AddData(bytes)
	ctx.Writer.WriteI32Const(int32(offset))
	ctx.Writer.WriteI32Const(int32(len(bytes)))
	ctx.Resolver.EmitStringFromLiteral(ctx.Writer, ctx.WriterID)
}
