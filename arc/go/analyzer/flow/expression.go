// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package flow

import (
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
)

// IsPureLiteral checks if an expression is a single literal value with no operators.
func IsPureLiteral(expr parser.IExpressionContext) bool {
	logicalOr := expr.LogicalOrExpression()
	if logicalOr == nil {
		return false
	}
	ands := logicalOr.AllLogicalAndExpression()
	if len(ands) != 1 {
		return false
	}
	equalities := ands[0].AllEqualityExpression()
	if len(equalities) != 1 {
		return false
	}
	relationals := equalities[0].AllRelationalExpression()
	if len(relationals) != 1 {
		return false
	}
	additives := relationals[0].AllAdditiveExpression()
	if len(additives) != 1 {
		return false
	}
	multiplicatives := additives[0].AllMultiplicativeExpression()
	if len(multiplicatives) != 1 {
		return false
	}
	powers := multiplicatives[0].AllPowerExpression()
	if len(powers) != 1 {
		return false
	}
	if powers[0].CARET() != nil {
		return false // has power operator
	}
	unary := powers[0].UnaryExpression()
	if unary == nil || unary.UnaryExpression() != nil {
		return false // has unary operator (like -1)
	}
	postfix := unary.PostfixExpression()
	if postfix == nil {
		return false
	}
	// Check no function calls or indexing
	if len(postfix.AllIndexOrSlice()) > 0 || len(postfix.AllFunctionCallSuffix()) > 0 {
		return false
	}
	primary := postfix.PrimaryExpression()
	if primary == nil {
		return false
	}
	// Must be a literal, not an identifier or parenthesized expression
	return primary.Literal() != nil && primary.IDENTIFIER() == nil && primary.Expression() == nil
}

// GetLiteralFromExpression extracts the literal node from a pure literal expression.
func GetLiteralFromExpression(expr parser.IExpressionContext) parser.ILiteralContext {
	return expr.LogicalOrExpression().
		AllLogicalAndExpression()[0].
		AllEqualityExpression()[0].
		AllRelationalExpression()[0].
		AllAdditiveExpression()[0].
		AllMultiplicativeExpression()[0].
		AllPowerExpression()[0].
		UnaryExpression().
		PostfixExpression().
		PrimaryExpression().
		Literal()
}

// analyzeExpression converts an inline expression into a synthetic function that
// can be used as a node in a flow graph. Pure literals are registered as KindConstant
// symbols and don't require code compilation.
func analyzeExpression(ctx acontext.Context[parser.IExpressionContext]) bool {
	exprType := atypes.InferFromExpression(ctx).Unwrap()
	t := types.Function(types.FunctionProperties{})
	t.Outputs = append(t.Outputs, types.Param{Name: ir.DefaultOutputParam, Type: exprType})

	// Pure literals become constants - no code to compile
	if IsPureLiteral(ctx.AST) {
		t.Config = append(t.Config, types.Param{Name: "value", Type: exprType})
		scope, err := ctx.Scope.Root().Add(ctx, symbol.Symbol{
			Name: "",
			Kind: symbol.KindConstant,
			Type: t,
			AST:  ctx.AST,
		})
		if err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return false
		}
		scope.AutoName("constant_")
		return true
	}

	// Complex expressions become synthetic functions that need compilation
	fnScope, err := ctx.Scope.Root().Add(ctx, symbol.Symbol{
		Name: "",
		Kind: symbol.KindFunction,
		Type: t,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	fnScope = fnScope.AutoName("expression_")

	blockScope, err := fnScope.Add(ctx, symbol.Symbol{
		Name: "",
		Kind: symbol.KindBlock,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	return expression.Analyze(ctx.WithScope(blockScope))
}
