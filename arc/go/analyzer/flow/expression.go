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
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// AnalyzeSingleExpression converts an inline expression into a synthetic function that
// can be used as a node in a flow graph. Pure literals are registered as KindConstant
// symbols and don't require code compilation.
func AnalyzeSingleExpression(ctx acontext.Context[parser.IExpressionContext]) {
	exprType := atypes.InferFromExpression(ctx).Unwrap()
	t := types.Function(types.FunctionProperties{})
	t.Outputs = append(t.Outputs, types.Param{Name: ir.DefaultOutputParam, Type: exprType})

	// Pure literals become constants - no code to compile
	if parser.IsLiteral(ctx.AST) {
		t.Config = append(t.Config, types.Param{Name: "value", Type: exprType})
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

	// Complex expressions become synthetic functions that need compilation
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
	fnScope.AccumulateReadChannels()

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
