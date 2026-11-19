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
// node, or into a constant when its value is known at compile time.
func AnalyzeSingleExpression(ctx acontext.Context[parser.IExpressionContext]) {
	// Re-analyzing an already-registered expression would duplicate its synth.
	if _, err := ctx.Scope.Root().GetChildByParserRule(ctx.AST); err == nil {
		return
	}
	if literal.IsConstant(ctx.AST) {
		analyzeConstantExpression(ctx)
		return
	}
	analyzeComputedExpression(ctx)
}

// analyzeConstantExpression registers a constant literal as a KindConstant.
func analyzeConstantExpression(ctx acontext.Context[parser.IExpressionContext]) {
	t := types.Function(types.FunctionProperties{})
	exprType := atypes.InferFromExpression(ctx).UnwrapChan()
	t.Outputs = types.Params{{Name: ir.DefaultOutputParam, Type: exprType}}
	t.Inputs = append(t.Inputs, types.Param{Name: "value", Type: exprType})
	scope, err := ctx.Scope.Root().Add(ctx, symbol.Symbol{
		Kind: symbol.KindConstant,
		Type: t,
		AST:  ctx.AST,
	})
	if err != nil {
		panic(err)
	}
	scope.AutoName("constant_")
}

// analyzeComputedExpression registers a complex expression as a synthetic function.
// The function takes no inputs: its body reads channel state through host calls, and
// stratum membership, not an input edge, activates it.
func analyzeComputedExpression(ctx acontext.Context[parser.IExpressionContext]) {
	fnScope, err := ctx.Scope.Root().Add(ctx, symbol.Symbol{
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{}),
		AST:  ctx.AST,
	})
	if err != nil {
		panic(err)
	}
	fnScope = fnScope.AutoName("expression_")
	// Set on the function scope, not the block: the compiler resolves the
	// synth expression under fnScope, so the hook must live there too.
	fnScope.SetLexicalResolver(ctx.Scope)

	blockScope, err := fnScope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindBlock,
		AST:  ctx.AST,
	})
	if err != nil {
		panic(err)
	}
	// Inferring in the body's scope gives the output and the body one type variable.
	body := ctx.WithScope(blockScope)
	fnScope.Type.Outputs = types.Params{{
		Name: ir.DefaultOutputParam,
		Type: atypes.InferFromExpression(body).UnwrapChan(),
	}}
	expression.Analyze(body)
}

// AnalyzeComputedArgs registers a synthetic function for each instance argument
// that is not a constant literal, a float constant, or a bare name, so lowering can
// sample it per fire.
func AnalyzeComputedArgs[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	args []symbol.Argument,
) {
	for _, arg := range args {
		head, _ := parser.ExpressionNameParts(arg.Expr)
		if head != "" || literal.IsConstant(arg.Expr) {
			continue
		}
		if _, ok := ctx.FloatConstant(arg.Expr); ok {
			continue
		}
		AnalyzeSingleExpression(ctx.Child(arg.Expr))
	}
}

// analyzeWASMCall registers a WASM-only call in a flow as a synthetic expression node.
func analyzeWASMCall(
	ctx acontext.Context[parser.IFunctionContext],
	callee *symbol.Symbol,
	name string,
) {
	enclosing := ctx.Scope
	freshType := types.Freshen(callee.Type, freshenKey(ctx.AST, name))
	t := types.Function(types.FunctionProperties{})
	if out, ok := freshType.Outputs.Get(ir.DefaultOutputParam); ok {
		t.Outputs = append(t.Outputs, out)
	}
	fnScope, err := ctx.Scope.Root().Add(ctx, symbol.Symbol{
		Kind: symbol.KindFunction,
		Type: t,
		AST:  ctx.AST,
	})
	if err != nil {
		panic(err)
	}
	fnScope = fnScope.AutoName("expression_")
	fnScope.SetLexicalResolver(enclosing)
	blockScope, err := fnScope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindBlock,
		AST:  ctx.AST,
	})
	if err != nil {
		panic(err)
	}
	blockCtx := ctx.WithScope(blockScope)
	args := inputArguments(blockCtx, ctx.AST)
	expression.AnalyzeCallSite(blockCtx, callee, name, freshType, args, ctx.AST)
}

// LiftVarReads lifts reassigned-variable reads in a synth expression into input
// params fed from the variable's node. Run after Reassigned flags are final.
func LiftVarReads[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	fn *symbol.Symbol,
	expr parser.IExpressionContext,
) {
	for _, pe := range parser.CollectNamedPrimaries(expr, ctx.FmtStrings) {
		name := parser.PrimaryName(pe)
		if _, ok := fn.Type.Inputs.Get(name); ok {
			continue
		}
		head, tail := parser.PrimaryNameParts(pe)
		sym, err := fn.ResolveMember(ctx, head, tail)
		if err != nil {
			continue
		}
		exprRead := sym.IsReactive() && sym.SourceID == nil
		if !sym.Reassigned && !exprRead && !sym.IsComputed() && !sym.IsSequenceInput() {
			continue
		}
		// A channel-coupled var lifts as its value type. A channel parameter stays one.
		t := sym.Type
		if t.Kind == types.KindChan && !sym.IsSequenceInput() {
			t = t.Unwrap()
		}
		if _, err = fn.Add(ctx, symbol.Symbol{
			Name:     name,
			Kind:     symbol.KindInput,
			Type:     t,
			Internal: true,
			AST:      expr,
		}); err != nil {
			panic(err)
		}
		fn.Type.Inputs = append(fn.Type.Inputs, types.Param{Name: name, Type: t})
	}
}

// BindOutput binds the output of the node registered for expr to t, the type of the
// variable it writes.
func BindOutput[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	expr parser.IExpressionContext,
	t types.Type,
) {
	expression.TakeUnit(ctx, expr, t.Unit)
	synth, err := ctx.Scope.Root().GetChildByParserRule(expr)
	if err != nil {
		return
	}
	out, ok := synth.Type.Outputs.Get(ir.DefaultOutputParam)
	if !ok || out.Type.Unwrap().Kind != types.KindVariable {
		return
	}
	if err := atypes.Check(
		ctx.Constraints,
		t,
		out.Type,
		expr,
		"assignment",
	); err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, expr))
	}
}
