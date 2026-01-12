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

func compileIfStatement(ctx context.Context[parser.IIfStatementContext]) (diverged bool, err error) {
	if _, err = expression.Compile(context.Child(ctx, ctx.AST.Expression())); err != nil {
		return false, errors.Wrap(err, "failed to compile if condition")
	}

	var (
		hasElseClause = ctx.AST.ElseClause() != nil
		hasElseIf     = len(ctx.AST.AllElseIfClause()) > 0
		hasElse       = hasElseClause || hasElseIf
	)
	if hasElse {
		ctx.Writer.WriteIf(wasm.BlockTypeEmpty)

		// Compile the main if block
		ifDiverged, err := CompileBlock(context.Child(ctx, ctx.AST.Block()))
		if err != nil {
			return false, errors.Wrap(err, "failed to compile if block")
		}

		// Track whether all branches diverge (starts with if branch)
		allBranchesDiverge := ifDiverged

		// Compile else-if clauses
		for i, elseIfClause := range ctx.AST.AllElseIfClause() {
			ctx.Writer.WriteElse()
			_, err := expression.Compile(context.Child(ctx, elseIfClause.Expression()))
			if err != nil {
				return false, errors.Wrapf(err, "failed to compile else-if[%d] condition", i)
			}
			ctx.Writer.WriteIf(wasm.BlockTypeEmpty)
			elseIfDiverged, err := CompileBlock(context.Child(ctx, elseIfClause.Block()))
			if err != nil {
				return false, errors.Wrapf(err, "failed to compile else-if[%d] block", i)
			}
			allBranchesDiverge = allBranchesDiverge && elseIfDiverged
		}

		// Compile the final else clause
		if hasElseClause {
			ctx.Writer.WriteElse()
			elseDiverged, err := CompileBlock(context.Child(ctx, ctx.AST.ElseClause().Block()))
			if err != nil {
				return false, errors.Wrap(err, "failed to compile else block")
			}
			allBranchesDiverge = allBranchesDiverge && elseDiverged
		} else if hasElseIf {
			// If we only have else-if without final else, not all paths are covered
			ctx.Writer.WriteElse()
			allBranchesDiverge = false
		}

		// Close all nested if blocks
		for range ctx.AST.AllElseIfClause() {
			ctx.Writer.WriteEnd()
		}
		ctx.Writer.WriteEnd()

		// Only add unreachable if we have a complete else clause and all branches diverged
		if hasElseClause && allBranchesDiverge {
			ctx.Writer.WriteUnreachable()
			return true, nil
		}

		return false, nil
	}

	// Simple if without else
	ctx.Writer.WriteIf(wasm.BlockTypeEmpty)
	if _, err = CompileBlock(context.Child(ctx, ctx.AST.Block())); err != nil {
		return false, errors.Wrap(err, "failed to compile if block")
	}
	ctx.Writer.WriteEnd()
	return false, nil
}

func compileReturnStatement(ctx context.Context[parser.IReturnStatementContext]) error {
	expr := ctx.AST.Expression()
	defer ctx.Writer.WriteReturn()
	if expr == nil {
		return nil
	}
	exprType, err := expression.Compile(context.Child(ctx, expr))
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

func compileFunctionCall(ctx context.Context[parser.IFunctionCallContext]) (types.Type, error) {
	funcName := ctx.AST.IDENTIFIER().GetText()
	scope, err := ctx.Scope.Resolve(ctx, funcName)
	if err != nil {
		return types.Type{}, errors.Wrapf(err, "undefined function: %s", funcName)
	}
	if scope.Kind != symbol.KindFunction {
		return types.Type{}, errors.Newf("%s is not a function", funcName)
	}

	funcType := scope.Type

	funcIdx, ok := ctx.FunctionIndices[funcName]
	if !ok {
		return types.Type{}, errors.Newf("function %s not found in index map", funcName)
	}

	if argList := ctx.AST.ArgumentList(); argList != nil {
		args := argList.AllExpression()
		if len(args) != len(funcType.Inputs) {
			return types.Type{}, errors.Newf(
				"function %s expects %d arguments, got %d",
				funcName, len(funcType.Inputs), len(args),
			)
		}
		for i, arg := range args {
			paramType := funcType.Inputs[i].Type
			argType, err := expression.Compile(context.Child(ctx, arg).WithHint(paramType))
			if err != nil {
				return types.Type{}, errors.Wrapf(err, "argument %d", i)
			}
			if !types.Equal(argType, paramType) {
				if err := expression.EmitCast(ctx, argType, paramType); err != nil {
					return types.Type{}, err
				}
			}
		}
	}

	ctx.Writer.WriteCall(funcIdx)
	// Drop return value for statement-level calls
	defaultOutput, hasDefault := funcType.Outputs.Get(ir.DefaultOutputParam)
	if hasDefault && defaultOutput.Type.IsValid() {
		ctx.Writer.WriteOpcode(wasm.OpDrop)
	}
	return types.Type{}, nil
}
