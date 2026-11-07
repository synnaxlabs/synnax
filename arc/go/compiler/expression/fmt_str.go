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
)

// EmitFmtSegments lowers the parsed segments of strTerm into WASM on ctx.Writer:
// literals emit string.from_literal, and placeholders compile the expressions the
// analyzer parsed and typed. Assumes the analyzer has already run
// literal.FmtStrValidateSpec on every placeholder spec; the compiler emits spec bytes
// verbatim without revalidation.
func EmitFmtSegments[T antlr.ParserRuleContext](
	ctx context.Context[T],
	strTerm antlr.TerminalNode,
	segments []literal.FmtStrSegment,
) (types.Type, error) {
	if len(segments) == 0 {
		emitLiteralSegment(ctx, "")
		return types.String(), nil
	}
	exprs := ctx.Placeholders[strTerm]
	for i, seg := range segments {
		var expr parser.IExpressionContext
		if seg.IsPlaceholder {
			if i >= len(exprs) || exprs[i] == nil {
				return types.Type{}, errors.Newf(
					"placeholder %q was not analyzed",
					seg.Text,
				)
			}
			expr = exprs[i]
		}
		if err := emitFmtSegment(ctx, seg, expr); err != nil {
			return types.Type{}, err
		}
		if i > 0 {
			ctx.Resolver.EmitStringConcat(ctx.Writer, ctx.WriterID)
		}
	}
	return types.String(), nil
}

func emitFmtSegment[T antlr.ParserRuleContext](
	ctx context.Context[T],
	seg literal.FmtStrSegment,
	expr parser.IExpressionContext,
) error {
	if !seg.IsPlaceholder {
		emitLiteralSegment(ctx, seg.Text)
		return nil
	}
	t, err := Compile(ctx.Child(expr).WithHint(types.Type{}))
	if err != nil {
		return err
	}
	if t.Kind == types.KindString {
		if seg.Spec != "" {
			emitSpecBytes(ctx, seg.Spec)
			return ctx.Resolver.EmitStringFormat(
				ctx.Context,
				ctx.Writer,
				ctx.WriterID,
				ctx.Scope,
			)
		}
		return nil
	}
	if t.Kind == types.KindBool {
		return ctx.Resolver.EmitFixedImportCall(
			ctx.Context,
			ctx.Writer,
			ctx.WriterID,
			ctx.Scope,
			"strings",
			"from_bool",
		)
	}
	if t.IsNumeric() {
		if seg.Spec != "" {
			emitSpecBytes(ctx, seg.Spec)
			return ctx.Resolver.EmitNumericFormat(
				ctx.Context,
				ctx.Writer,
				ctx.WriterID,
				ctx.Scope,
				t,
			)
		}
		return ctx.Resolver.EmitNumericToString(
			ctx.Context,
			ctx.Writer,
			ctx.WriterID,
			ctx.Scope,
			t,
		)
	}
	return errors.Newf(
		"placeholder %q has type %s; only numeric, string, and bool types are supported",
		seg.Text,
		t,
	)
}

func emitSpecBytes[T antlr.ParserRuleContext](ctx context.Context[T], spec string) {
	specBytes := []byte(spec)
	offset := ctx.Module.AddData(specBytes)
	ctx.Writer.WriteI32Const(int32(offset))
	ctx.Writer.WriteI32Const(int32(len(specBytes)))
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
