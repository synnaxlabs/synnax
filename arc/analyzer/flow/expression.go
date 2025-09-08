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
	"github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/analyzer/result"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// analyzeExpression converts an inline expression into a synthetic task
func analyzeExpression(
	scope *symbol.Scope,
	res *result.Result,
	expr text.IExpressionContext,
) bool {
	exprType := atypes.InferFromExpression(scope, expr, nil)
	// If the expression type is a channel, the task returns the channel's value type
	// (because the task will read from the channel)
	if chanType, ok := exprType.(types.Chan); ok {
		exprType = chanType.ValueType
	}
	t := types.NewTask()
	t.Return = exprType
	taskScope, err := scope.Root().Add(symbol.Symbol{
		Name:       "",
		Kind:       symbol.KindTask,
		Type:       t,
		ParserRule: expr,
	})
	if err != nil {
		res.AddError(err, expr)
		return false
	}
	taskScope = taskScope.AutoName("__expr_")
	blockScope, err := taskScope.Add(symbol.Symbol{
		Name:       "",
		Kind:       symbol.KindBlock,
		Type:       t,
		ParserRule: expr,
	})
	if err != nil {
		res.AddError(err, expr)
		return false
	}
	taskScope.OnResolve = func(s *symbol.Scope) error {
		_, ok := s.Type.(types.Chan)
		if !ok {
			res.AddError(errors.Newf(
				"type mismatch: only channels can be used in flow expressions, encountered symbol %s with type %s",
				s.Name,
				s.Type.String(),
			), s.ParserRule)
		}
		return nil
	}
	if !expression.Analyze(blockScope, res, expr) {
		return false
	}
	taskScope.Type = t
	return true
}
