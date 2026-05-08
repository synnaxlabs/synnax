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
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/fmtstring"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	basetypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// analyzeStringFmtPlaceholders runs the extra analysis needed for a
// `string.fmt(format)` call when format is a string literal: it parses the
// format body, then for each placeholder parses and analyzes the placeholder
// source as an Arc expression in the call-site scope. This causes channel and
// identifier references inside placeholders to be registered exactly as if
// they had appeared inline.
//
// If format is not a literal, placeholder analysis is skipped; the regular
// call validation still applies.
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
	formatArg := args[0]
	litNode := parser.GetLiteral(formatArg)
	if litNode == nil {
		return
	}
	parsed, err := literal.Parse(litNode, basetypes.String())
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, formatArg))
		return
	}
	body, ok := parsed.Value.(string)
	if !ok {
		return
	}
	segments, err := fmtstring.Parse(body)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, formatArg))
		return
	}
	for _, seg := range segments {
		if !seg.IsPlaceholder {
			continue
		}
		expr, diags := parser.ParseExpression(seg.Text)
		if diags != nil && !diags.Ok() {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				formatArg,
				"invalid placeholder expression %q: %s", seg.Text, diags.String(),
			))
			continue
		}
		Analyze(context.Child(ctx, expr))
		t := types.InferFromExpression(context.Child(ctx, expr)).UnwrapChan()
		if !t.IsNumeric() && t.Kind != basetypes.KindString {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				formatArg,
				"placeholder %q has type %s; only numeric and string types are supported",
				seg.Text, t,
			))
		}
	}
}
