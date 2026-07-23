// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package flow

import (
	"github.com/antlr4-go/antlr/v4"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// AnalyzeSingleExpression converts an inline expression into a synthetic function
// node. Format-string placeholders are analyzed here; IR shape is chosen downstream.
func AnalyzeSingleExpression(ctx acontext.Context[parser.IExpressionContext]) {
	// Re-analyzing an already-registered expression would duplicate its synth.
	if _, err := ctx.Scope.Root().GetChildByParserRule(ctx.AST); err == nil {
		return
	}
	// enclosing is the lexical scope the expression appears in.
	enclosing := ctx.Scope
	exprType := atypes.InferFromExpression(ctx).Unwrap()
	t := types.Function(types.FunctionProperties{})
	t.Outputs = append(t.Outputs, types.Param{Name: ir.DefaultOutputParam, Type: exprType})

	// Literals register as KindConstant; format strings with placeholders register
	// as synthetic functions so placeholder channel reads track on the right symbol.
	if parser.IsLiteral(ctx.AST) {
		if lit := parser.GetLiteral(ctx.AST); lit != nil {
			if strTerm := parser.StringTerminal(lit); strTerm != nil {
				body, flags, ok := literal.StripQuotes(strTerm.GetText())
				if !ok {
					ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
						"invalid string literal: %s", strTerm.GetText()))
					return
				}
				if flags.Format {
					segs, err := literal.FmtStrParse(body)
					if err != nil {
						ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
						return
					}
					if literal.FmtStrHasPlaceholder(segs) {
						fnScope, err := ctx.Scope.Root().Add(ctx, symbol.Symbol{Kind: symbol.KindFunction, Type: t, AST: ctx.AST})
						if err != nil {
							ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
							return
						}
						fnScope.AutoName("fmt_str_")
						fnScope.SetLexicalResolver(enclosing)
						expression.AnalyzeFmtStrLiteral(ctx.WithScope(fnScope), strTerm)
						return
					}
				}
			}
		}
		t.Inputs = append(t.Inputs, types.Param{Name: "value", Type: exprType})
		scope, err := ctx.Scope.Root().Add(ctx, symbol.Symbol{
			Kind: symbol.KindConstant,
			Type: t,
			AST:  ctx.AST,
		})
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return
		}
		scope.AutoName("constant_")
		return
	}

	// Complex expressions become synthetic functions that need compilation.
	// The function takes no inputs — its body reads from channel state directly
	// via host calls, and activation is handled by stratum membership rather
	// than an input edge.
	fnScope, err := ctx.Scope.Root().Add(ctx, symbol.Symbol{
		Kind: symbol.KindFunction,
		Type: t,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	fnScope = fnScope.AutoName("expression_")
	// Set on the function scope, not the block: the compiler resolves the
	// synth expression under fnScope, so the hook must live there too.
	fnScope.SetLexicalResolver(enclosing)

	blockScope, err := fnScope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindBlock,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	expression.Analyze(ctx.WithScope(blockScope))
}

// LiftFmtStrVarReads lifts reassigned-variable reads in a format string's
// placeholders into input params. No-op for non-format literals.
func LiftFmtStrVarReads[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	fn *symbol.Symbol,
	expr parser.IExpressionContext,
) {
	lit := parser.GetLiteral(expr)
	if lit == nil {
		return
	}
	strTerm := parser.StringTerminal(lit)
	if strTerm == nil {
		return
	}
	body, flags, ok := literal.StripQuotes(strTerm.GetText())
	if !ok || !flags.Format {
		return
	}
	segments, err := literal.FmtStrParse(body)
	if err != nil {
		return
	}
	for _, seg := range segments {
		if !seg.IsPlaceholder || seg.Text == "" {
			continue
		}
		phExpr, diags := parser.ParseExpression(seg.Text, ctx.Config)
		if diags != nil && !diags.Ok() {
			continue
		}
		LiftVarReads(ctx, fn, phExpr)
	}
}

// LiftVarReads lifts reassigned-variable reads in a synth expression into input
// params fed from the variable's node. Run after Reassigned flags are final.
func LiftVarReads[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	fn *symbol.Symbol,
	expr parser.IExpressionContext,
) {
	for _, name := range parser.CollectIdentifiers(expr) {
		if _, ok := fn.Type.Inputs.Get(name); ok {
			continue
		}
		sym, err := fn.Resolve(ctx, name)
		if err != nil {
			continue
		}
		exprRead := sym.IsReactive() && sym.SourceID == nil
		if !sym.Reassigned && !exprRead {
			continue
		}
		// A channel-coupled var lifts as its value type; a node feeds it data.
		t := sym.Type
		if t.Kind == types.KindChan {
			t = t.Unwrap()
		}
		if _, err = fn.Add(ctx, symbol.Symbol{
			Name:     name,
			Kind:     symbol.KindInput,
			Type:     t,
			Internal: true,
			AST:      expr,
		}); err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, expr))
			return
		}
		fn.Type.Inputs = append(fn.Type.Inputs, types.Param{Name: name, Type: t})
	}
}
