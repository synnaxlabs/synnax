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
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/fmtstring"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	basetypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// AnalyzeStringFmtSegments parses formatExpr as a string literal and analyzes
// each placeholder expression in ctx's scope, registering channel and
// identifier references as if inlined at the caller. Returns nil (without
// diagnostics) if formatExpr is not a literal.
func AnalyzeStringFmtSegments[T antlr.ParserRuleContext](
	ctx context.Context[T],
	formatExpr parser.IExpressionContext,
) []fmtstring.Segment {
	litNode := parser.GetLiteral(formatExpr)
	if litNode == nil {
		return nil
	}
	parsed, err := literal.Parse(litNode, basetypes.String())
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, formatExpr))
		return nil
	}
	body, ok := parsed.Value.(string)
	if !ok {
		return nil
	}
	segments, err := fmtstring.Parse(body)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, formatExpr))
		return nil
	}
	for _, seg := range segments {
		if !seg.IsPlaceholder {
			continue
		}
		expr, diags := parser.ParseExpression(seg.Text)
		if diags != nil && !diags.Ok() {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				formatExpr,
				"invalid placeholder expression %q: %s", seg.Text, diags.String(),
			))
			continue
		}
		Analyze(context.Child(ctx, expr))
		t := types.InferFromExpression(context.Child(ctx, expr)).UnwrapChan()
		if !t.IsNumeric() && t.Kind != basetypes.KindString {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				formatExpr,
				"placeholder %q has type %s; only numeric and string types are supported",
				seg.Text, t,
			))
		}
	}
	return segments
}

// analyzeStringFmtPlaceholders is the call-form entry point: it pulls args[0]
// off the call suffix and delegates to AnalyzeStringFmtSegments.
func analyzeStringFmtPlaceholders(
	ctx context.Context[parser.IPostfixExpressionContext],
	funcCall parser.IFunctionCallSuffixContext,
) {
	argList := funcCall.ArgumentList()
	if argList == nil {
		return
	}
	args := argList.AllExpression()
	if len(args) == 0 {
		return
	}
	AnalyzeStringFmtSegments(ctx, args[0])
}
