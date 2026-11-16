// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package statement

import (
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func compileIfStatement(
	ctx context.Context[parser.IIfStatementContext],
) (diverged bool, err error) {
	if _, err = expression.Compile(ctx.Child(ctx.AST.Expression())); err != nil {
		return false, errors.Wrap(err, "failed to compile if condition")
	}
	ctx.Writer.WriteIf(wasm.BlockTypeEmpty)
	innerCtx := ctx.EnterBlock()
	ifDiverged, err := CompileBlock(innerCtx.Child(ctx.AST.Block()))
	if err != nil {
		return false, errors.Wrap(err, "failed to compile if block")
	}
	hasElseClause := ctx.AST.ElseClause() != nil
	if !hasElseClause && len(ctx.AST.AllElseIfClause()) == 0 {
		ctx.Writer.WriteEnd()
		return false, nil
	}
	allBranchesDiverge, err := compileElseBranches(ctx, innerCtx, ifDiverged)
	if err != nil {
		return false, err
	}
	for range ctx.AST.AllElseIfClause() {
		ctx.Writer.WriteEnd()
	}
	ctx.Writer.WriteEnd()
	// Only a complete else clause covers every path.
	if hasElseClause && allBranchesDiverge {
		ctx.Writer.WriteUnreachable()
		return true, nil
	}
	return false, nil
}

// compileElseBranches writes the else-if and else clauses of an if statement, each
// nested in the else of the one before. It returns whether every branch diverges.
func compileElseBranches(
	ctx context.Context[parser.IIfStatementContext],
	innerCtx context.Context[parser.IIfStatementContext],
	ifDiverged bool,
) (bool, error) {
	allBranchesDiverge := ifDiverged
	elseIfCtx := innerCtx
	for i, elseIfClause := range ctx.AST.AllElseIfClause() {
		ctx.Writer.WriteElse()
		_, err := expression.Compile(elseIfCtx.Child(elseIfClause.Expression()))
		if err != nil {
			return false, errors.Wrapf(
				err,
				"failed to compile else-if[%d] condition",
				i,
			)
		}
		ctx.Writer.WriteIf(wasm.BlockTypeEmpty)
		elseIfCtx = elseIfCtx.EnterBlock()
		elseIfDiverged, err := CompileBlock(elseIfCtx.Child(elseIfClause.Block()))
		if err != nil {
			return false, errors.Wrapf(err, "failed to compile else-if[%d] block", i)
		}
		allBranchesDiverge = allBranchesDiverge && elseIfDiverged
	}
	ctx.Writer.WriteElse()
	elseClause := ctx.AST.ElseClause()
	if elseClause == nil {
		return false, nil
	}
	elseDiverged, err := CompileBlock(elseIfCtx.Child(elseClause.Block()))
	if err != nil {
		return false, errors.Wrap(err, "failed to compile else block")
	}
	return allBranchesDiverge && elseDiverged, nil
}

func compileReturnStatement(ctx context.Context[parser.IReturnStatementContext]) error {
	expr := ctx.AST.Expression()
	defer ctx.Writer.WriteReturn()
	if expr == nil {
		return nil
	}
	exprType, err := expression.Compile(ctx.Child(expr))
	if err != nil {
		return errors.Wrap(err, "failed to compile return expression")
	}
	enclosingScope, err := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
	if err != nil {
		return errors.New("return statement not in function")
	}
	var returnType types.Type
	if enclosingScope.Kind == symbol.KindFunction {
		returnParam, _ := enclosingScope.Type.Outputs.Get(ir.DefaultOutputParam)
		returnType = returnParam.Type
	}
	if !types.Equal(returnType, exprType) {
		return expression.EmitCast(ctx, exprType, returnType)
	}
	return nil
}

func compileExpressionStatement(ctx context.Context[parser.IExpressionContext]) error {
	exprType, err := expression.Compile(ctx)
	if err != nil {
		return err
	}
	// Drop the result if the expression produced a value (statement-level expressions
	// discard results)
	if exprType.IsValid() {
		ctx.Writer.WriteOpcode(wasm.OpDrop)
	}
	return nil
}
