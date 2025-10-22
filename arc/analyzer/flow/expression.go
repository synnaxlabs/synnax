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

// analyzeExpression converts an inline expression into a synthetic fn
func analyzeExpression(ctx acontext.Context[parser.IExpressionContext]) bool {
	exprType := atypes.InferFromExpression(ctx).Unwrap()
	t := types.Function(types.FunctionProperties{})
	t.Outputs.Put(ir.DefaultOutputParam, exprType)
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
	fnScope = fnScope.AutoName("__expr_")
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
