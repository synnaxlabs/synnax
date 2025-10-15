// Copyright 2025 Synnax Labs, Inc.
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

func compileIfStatement(ctx context.Context[parser.IIfStatementContext]) error {
	if _, err := expression.Compile(context.Child(ctx, ctx.AST.Expression())); err != nil {
		return errors.Wrap(err, "failed to compile if condition")
	}
	hasElse := ctx.AST.ElseClause() != nil || len(ctx.AST.AllElseIfClause()) > 0
	if hasElse {
		ctx.Writer.WriteIf(wasm.BlockTypeEmpty)
		if err := CompileBlock(context.Child(ctx, ctx.AST.Block())); err != nil {
			return errors.Wrap(err, "failed to compile if block")
		}

		for i, elseIfClause := range ctx.AST.AllElseIfClause() {
			ctx.Writer.WriteElse()
			_, err := expression.Compile(context.Child(ctx, elseIfClause.Expression()))
			if err != nil {
				return errors.Wrapf(err, "failed to compile else-if[%d] condition", i)
			}
			ctx.Writer.WriteIf(wasm.BlockTypeEmpty)
			if err = CompileBlock(context.Child(ctx, elseIfClause.Block())); err != nil {
				return errors.Wrapf(err, "failed to compile else-if[%d] block", i)
			}
		}
		if elseClause := ctx.AST.ElseClause(); elseClause != nil {
			ctx.Writer.WriteElse()
			if err := CompileBlock(context.Child(ctx, elseClause.Block())); err != nil {
				return errors.Wrap(err, "failed to compile else block")
			}
		} else if len(ctx.AST.AllElseIfClause()) > 0 {
			ctx.Writer.WriteElse()
		}

		for range ctx.AST.AllElseIfClause() {
			ctx.Writer.WriteEnd()
		}
		ctx.Writer.WriteEnd()
		return nil
	}
	ctx.Writer.WriteIf(wasm.BlockTypeEmpty)
	if err := CompileBlock(context.Child(ctx, ctx.AST.Block())); err != nil {
		return errors.Wrap(err, "failed to compile if block")
	}
	ctx.Writer.WriteEnd()
	return nil
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
		returnType, _ = enclosingScope.Type.Outputs.Get(ir.DefaultOutputParam)
	}
	if returnType != exprType {
		return expression.EmitCast(ctx, exprType, returnType)
	}
	return nil
}

// compileChannelOperation handles channel writes and piping
func compileChannelOperation(ctx context.Context[parser.IChannelOperationContext]) error {
	if chanWrite := ctx.AST.ChannelWrite(); chanWrite != nil {
		return compileChannelWrite(context.Child(ctx, chanWrite))
	}

	if chanRead := ctx.AST.ChannelRead(); chanRead != nil {
		return compileChannelRead(context.Child(ctx, chanRead))
	}

	return errors.New("unknown channel operation")
}

func compileChannelWrite(ctx context.Context[parser.IChannelWriteContext]) error {
	var (
		channelName string
		valueExpr   parser.IExpressionContext
	)
	if ctx.AST.Expression() != nil && ctx.AST.IDENTIFIER() != nil {
		valueExpr = ctx.AST.Expression()
		channelName = ctx.AST.IDENTIFIER().GetText()
	}
	valueType, err := expression.Compile(context.Child(ctx, valueExpr))
	if err != nil {
		return errors.Wrap(err, "failed to compile channel write value")
	}
	if _, err = ctx.Scope.Resolve(ctx, channelName); err != nil {
		return errors.Wrapf(err, "channel '%s' not found", channelName)
	}
	sym, err := ctx.Scope.Resolve(ctx, channelName)
	if err != nil {
		return errors.Newf("channel '%s' not in local context", channelName)
	}
	ctx.Writer.WriteLocalGet(sym.ID)
	importIdx, err := ctx.Imports.GetChannelWrite(valueType)
	if err != nil {
		return errors.Wrap(err, "failed to compile channel write import")
	}
	ctx.Writer.WriteCall(importIdx)
	return nil
}

func compileChannelRead(_ context.Context[parser.IChannelReadContext]) error {
	// TODO: Implement this
	return errors.New("standalone channel reads not implemented")
}

// compileFunctionCall handles function calls (may return a value)
func compileFunctionCall(_ context.Context[parser.IFunctionCallContext]) (types.Type, error) {
	// TODO: Implement function calls
	return types.Type{}, errors.New("function calls not yet implemented")
}
