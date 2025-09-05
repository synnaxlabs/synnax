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
	"github.com/synnaxlabs/slate/analyzer/expression"
	"github.com/synnaxlabs/slate/analyzer/result"
	atypes "github.com/synnaxlabs/slate/analyzer/types"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	"github.com/synnaxlabs/x/errors"
)

// analyzeExpression converts an inline expression into a synthetic task
func analyzeExpression(
	scope *symbol.Scope,
	res *result.Result,
	expr parser.IExpressionContext,
) bool {
	exprType := atypes.InferFromExpression(scope, expr, nil)
	// If the expression type is a channel, the task returns the channel's value type
	// (because the task will read from the channel)
	if chanType, ok := exprType.(types.Chan); ok {
		exprType = chanType.ValueType
	}
	t := &types.Task{Return: exprType}
	taskScope, err := scope.Root().Add("", symbol.KindTask, t, expr)
	if err != nil {
		res.AddError(err, expr)
		return false
	}
	taskScope = taskScope.AutoName("__expr_")
	blockScope, err := taskScope.Add(
		"",
		symbol.KindBlock,
		nil,
		nil,
	)
	if err != nil {
		res.AddError(err, expr)
		return false
	}
	blockScope.OnResolve = func(s *symbol.Scope) error {
		c, ok := s.Type.(types.Chan)
		if !ok {
			res.AddError(errors.Newf(
				"type mismatch: only channels can be used in flow expressions, encountered symbol %s with type %s",
				s.Name,
				s.Type.String(),
			), s.ParserRule)
		}
		name := "__" + s.Name
		_, _ = taskScope.Add(name, symbol.KindConfigParam, c.ValueType, s.ParserRule)
		t.Config.Put(name, c.ValueType)
		return nil
	}
	return expression.Analyze(blockScope, res, expr)
}
