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
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// analyzeBodyCall checks a call of a sequence or stage with parameters, if ctx is one.
func analyzeBodyCall(ctx context.Context[parser.IFunctionContext]) bool {
	head, tail := parser.FunctionNameParts(ctx.AST)
	// ResolveQualified would warn about a deprecated callee a second time.
	body, err := ctx.Scope.ResolveMember(ctx, head, tail)
	if err != nil || !body.IsParameterized() {
		return false
	}
	var (
		name  = parser.FunctionName(ctx.AST)
		noun  = BodyNoun(body)
		args  = inputArguments(ctx, ctx.AST)
		fixed = true
	)
	for _, arg := range args {
		if param, ok := bodyParam(body, arg); ok {
			fixed = checkBodyArgument(ctx, noun, name, param, arg) && fixed
		}
	}
	if fixed {
		expression.AnalyzeCall(ctx, noun, name, body.Type, args, nil, ctx.AST, "")
	}
	return true
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

// checkBodyArgument reports an argument that is not fixed or is the wrong kind.
func checkBodyArgument(
	ctx context.Context[parser.IFunctionContext],
	noun, name string,
	param types.Param,
	arg symbol.Argument,
) bool {
	channel, ok := fixedArgument(ctx, arg.Expr)
	if !ok {
		ctx.Diagnostics.Add(diagnostics.Errorf(arg.AST,
			"argument for parameter '%s' of %s '%s' is not fixed at compile time",
			param.Name, noun, name,
		).WithNote(
			"use a literal, a channel, a top-level constant or alias, or a parameter",
		))
		return false
	}
	if isChan := param.Type.Kind == types.KindChan; channel != isChan {
		must := "cannot be"
		if isChan {
			must = "must be"
		}
		ctx.Diagnostics.Add(diagnostics.Errorf(arg.AST,
			"argument for parameter '%s' of %s '%s' %s a channel",
			param.Name, noun, name, must,
		))
		return false
	}
	return true
}

// fixedArgument reports whether expr is fixed at compile time and if it is a channel.
func fixedArgument(
	ctx context.Context[parser.IFunctionContext],
	expr parser.IExpressionContext,
) (channel, ok bool) {
	if parser.IsLiteral(expr) || parser.IsNegatedLiteral(expr) {
		return false, true
	}
	if _, isConstant := ctx.FloatConstant(expr); isConstant {
		return false, true
	}
	head, tail := parser.ExpressionNameParts(expr)
	if head == "" {
		return false, false
	}
	sym, err := ctx.Scope.ResolveMember(ctx, head, tail)
	if err != nil {
		return false, false
	}
	channel = sym.Type.Kind == types.KindChan
	switch {
	case sym.Kind == symbol.KindChannel:
		return channel, true
	case sym.IsSequenceInput():
		return channel, sym.Parent == enclosingBody(ctx.Scope)
	case sym.Kind == symbol.KindVariable && sym.Parent == sym.Root():
		// A top-level alias is never rebound, and a top-level literal never changes.
		if channel {
			return true, sym.SourceID != nil
		}
		return false, sym.DefaultValue != nil
	default:
		return false, false
	}
}
