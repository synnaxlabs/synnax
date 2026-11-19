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
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// analyzeBodyCall checks a call of a sequence or stage, if ctx is one.
func analyzeBodyCall(ctx context.Context[parser.IFunctionContext]) bool {
	head, tail := parser.FunctionNameParts(ctx.AST)
	// ResolveQualified would warn about a deprecated callee a second time.
	body, err := ctx.Scope.ResolveMember(ctx, head, tail)
	if err != nil ||
		(body.Kind != symbol.KindSequence && body.Kind != symbol.KindStage) {
		return false
	}
	name := parser.FunctionName(ctx.AST)
	if !body.IsParameterized() {
		ctx.Diagnostics.Add(
			codes.TransitionTarget.Atf(ctx.AST, "%s has no parameters", name).
				WithNote("use " + name + " with no parentheses"),
		)
		return true
	}
	var (
		noun   = BodyNoun(body)
		args   = inputArguments(ctx, ctx.AST)
		values []symbol.Argument
		valid  = true
	)
	for _, arg := range args {
		param, ok := bodyParam(body, arg)
		if !ok {
			continue
		}
		valid = checkBodyArgument(ctx, noun, name, param, arg) && valid
		// A negative literal is a constant, so it gets no node.
		if param.Type.Kind != types.KindChan && !parser.IsNegatedLiteral(arg.Expr) {
			values = append(values, arg)
		}
	}
	if valid {
		AnalyzeComputedArgs(ctx, values)
		expression.AnalyzeCall(ctx, noun, name, body.Type, args, nil, ctx.AST, "")
	}
	return true
}

// UsedBody returns the named stage or sequence that node uses, or nil. The node is a
// name or a call, in a flow or as an item of a stage or sequence.
func UsedBody[T antlr.ParserRuleContext](
	ctx context.Context[T],
	node antlr.ParserRuleContext,
) *symbol.Symbol {
	var head, tail string
	switch n := node.(type) {
	case parser.IIdentifierContext:
		head, tail = parser.IdentifierParts(n)
	case parser.IFunctionContext:
		head, tail = parser.FunctionNameParts(n)
	case parser.IExpressionContext:
		head, tail = parser.ExpressionNameParts(n)
	}
	if head == "" {
		return nil
	}
	// ResolveQualified would warn about a deprecated name a second time.
	body, err := ctx.Scope.ResolveMember(ctx, head, tail)
	if err != nil || !body.IsTemplate() {
		return nil
	}
	return body
}

// AnalyzeBodyName checks a name that is an item of a stage or sequence, if it names a
// stage or sequence.
func AnalyzeBodyName(ctx context.Context[parser.IExpressionContext]) bool {
	body := UsedBody(ctx, ctx.AST)
	if body == nil {
		return false
	}
	if body.IsParameterized() {
		ctx.Diagnostics.Add(needsCall(ctx.AST, parser.GetExpressionText(ctx.AST)))
	}
	return true
}

// needsCall returns the diagnostic for the bare name of a body with parameters.
func needsCall(node antlr.ParserRuleContext, name string) diagnostics.Diagnostic {
	return codes.TransitionTarget.Atf(node, "%s has parameters", name).
		WithNote("use " + name + "()")
}

// BodyNoun returns the word that names body in a diagnostic: "stage" or "sequence".
func BodyNoun(body *symbol.Symbol) string {
	if _, ok := body.AST.(parser.IStageDeclarationContext); ok {
		return "stage"
	}
	return "sequence"
}

// enclosingBody returns the top-level sequence or stage that holds scope, or nil.
func enclosingBody(scope *symbol.Symbol) *symbol.Symbol {
	root := scope.Root()
	for s := scope; s != nil && s != root; s = s.Parent {
		if s.Parent == root && s.Kind == symbol.KindSequence {
			return s
		}
	}
	return nil
}

// bodyParam returns the parameter of body that arg binds, by name or by position.
func bodyParam(body *symbol.Symbol, arg symbol.Argument) (types.Param, bool) {
	if arg.Name != "" {
		return body.Type.Inputs.Get(arg.Name)
	}
	if arg.Index >= len(body.Type.Inputs) {
		return types.Param{}, false
	}
	return body.Type.Inputs[arg.Index], true
}

// checkBodyArgument reports a channel that is not fixed or a wrong kind of argument.
func checkBodyArgument(
	ctx context.Context[parser.IFunctionContext],
	noun, name string,
	param types.Param,
	arg symbol.Argument,
) bool {
	channel, fixed := channelArgument(ctx, arg.Expr)
	isChan := param.Type.Kind == types.KindChan
	if channel && isChan && !fixed {
		ctx.Diagnostics.Add(codes.NotConstant.Atf(arg.AST,
			"argument for parameter '%s' of %s '%s' is not fixed at compile time",
			param.Name, noun, name,
		).WithNote("use a channel, a top-level alias, or a channel parameter"))
		return false
	}
	if channel != isChan {
		kind, must := codes.FuncArgValue, "cannot be"
		if isChan {
			kind, must = codes.ChannelExpected, "must be"
		}
		ctx.Diagnostics.Add(kind.Atf(arg.AST,
			"argument for parameter '%s' of %s '%s' %s a channel",
			param.Name, noun, name, must,
		))
		return false
	}
	return true
}

// channelArgument reports whether expr names a channel and if a call can bind it.
func channelArgument(
	ctx context.Context[parser.IFunctionContext],
	expr parser.IExpressionContext,
) (channel, fixed bool) {
	head, tail := parser.ExpressionNameParts(expr)
	if head == "" {
		return false, false
	}
	sym, err := ctx.Scope.ResolveMember(ctx, head, tail)
	if err != nil {
		return false, false
	}
	switch {
	case sym.Kind == symbol.KindChannel:
		return true, true
	case sym.IsSequenceInput():
		return sym.Type.Kind == types.KindChan, sym.Parent == enclosingBody(ctx.Scope)
	case sym.IsValueVariable() && sym.Type.Kind == types.KindChan && sym.SourceID != nil:
		// A top-level alias is never rebound.
		return true, sym.Parent == sym.Root()
	default:
		return false, false
	}
}
