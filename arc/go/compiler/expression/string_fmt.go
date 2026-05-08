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
	"github.com/synnaxlabs/arc/fmtstring"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// EmitFmtSegments lowers parsed format segments into WASM on ctx.Writer:
// literals emit string.from_literal, placeholders compile their expression.
func EmitFmtSegments[T antlr.ParserRuleContext](
	ctx context.Context[T],
	segments []fmtstring.Segment,
) (types.Type, error) {
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

func emitFmtSegment[T antlr.ParserRuleContext](
	ctx context.Context[T],
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

func emitLiteralSegment[T antlr.ParserRuleContext](
	ctx context.Context[T],
	text string,
) {
	bytes := []byte(text)
	offset := ctx.Module.AddData(bytes)
	ctx.Writer.WriteI32Const(int32(offset))
	ctx.Writer.WriteI32Const(int32(len(bytes)))
	ctx.Resolver.EmitStringFromLiteral(ctx.Writer, ctx.WriterID)
}
