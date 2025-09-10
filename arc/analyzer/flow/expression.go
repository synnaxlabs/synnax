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
	"context"

	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/errors"
)

// analyzeExpression converts an inline expression into a synthetic stage
func analyzeExpression(ctx acontext.Context[parser.IExpressionContext]) bool {
	exprType := atypes.InferFromExpression(ctx)
	// If the expression type is a channel, the stage returns the channel's value type
	// (because the stage will read from the channel)
	if chanType, ok := exprType.(ir.Chan); ok {
		exprType = chanType.ValueType
	}
	t := ir.Stage{Return: exprType, Channels: ir.NewChannels()}
	stageScope, err := ctx.Scope.Root().Add(ctx, ir.Symbol{
		Name:       "",
		Kind:       ir.KindStage,
		Type:       t,
		ParserRule: ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	stageScope = stageScope.AutoName("__expr_")
	t.Key = stageScope.Name
	blockScope, err := stageScope.Add(ctx, ir.Symbol{
		Name:       "",
		Kind:       ir.KindBlock,
		Type:       t,
		ParserRule: ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	blockScope.OnResolve = func(ctx_ context.Context, s *ir.Scope) error {
		_, ok := s.Type.(ir.Chan)
		if !ok {
			ctx.Diagnostics.AddError(errors.Newf(
				"type mismatch: only channels can be used in flow expressions, encountered symbol %s with type %s",
				s.Name,
				s.Type.String(),
			), s.ParserRule)
		}
		if s.Kind == ir.KindChannel {
			t.Channels.Read.Add(uint32(s.ID))
		}
		return nil
	}
	if !expression.Analyze(ctx.WithScope(blockScope)) {
		return false
	}
	stageScope.Type = t
	return true
}
