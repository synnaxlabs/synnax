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
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func compileForStatement(
	ctx context.Context[parser.IForStatementContext],
) (diverged bool, err error) {
	clause := ctx.AST.ForClause()
	if clause == nil {
		return false, nil
	}

	idents := clause.AllIDENTIFIER()
	hasDeclare := clause.DECLARE() != nil
	hasComma := clause.COMMA() != nil
	expr := clause.Expression()

	switch {
	case hasComma && len(idents) == 2:
		return false, compileForTwoIdent(ctx, clause, expr)
	case hasDeclare && len(idents) == 1:
		return false, compileForSingleIdent(ctx, clause, idents[0].GetText(), expr)
	case expr != nil:
		return false, compileForCondition(ctx, expr)
	default:
		return false, compileForInfinite(ctx)
	}
}

func compileForSingleIdent(
	ctx context.Context[parser.IForStatementContext],
	clause parser.IForClauseContext,
	name string,
	expr parser.IExpressionContext,
) error {
	if funcCall, ok := isRangeCallExpr(expr); ok {
		return compileForRange(ctx, clause, name, funcCall)
	}
	return compileForSeriesIteration(ctx, clause, name, "", expr)
}

func compileForTwoIdent(
	ctx context.Context[parser.IForStatementContext],
	clause parser.IForClauseContext,
	expr parser.IExpressionContext,
) error {
	idents := clause.AllIDENTIFIER()
	indexName := idents[0].GetText()
	elemName := idents[1].GetText()
	return compileForSeriesIteration(ctx, clause, elemName, indexName, expr)
}

func isRangeCallExpr(expr parser.IExpressionContext) (parser.IFunctionCallSuffixContext, bool) {
	primary := parser.GetPrimaryExpression(expr)
	if primary == nil || primary.IDENTIFIER() == nil {
		return nil, false
	}
	if primary.IDENTIFIER().GetText() != "range" {
		return nil, false
	}
	postfix := expr.LogicalOrExpression().
		AllLogicalAndExpression()[0].
		AllEqualityExpression()[0].
		AllRelationalExpression()[0].
		AllAdditiveExpression()[0].
		AllMultiplicativeExpression()[0].
		AllPowerExpression()[0].
		UnaryExpression().
		PostfixExpression()
	if postfix == nil {
		return nil, false
	}
	calls := postfix.AllFunctionCallSuffix()
	if len(calls) != 1 {
		return nil, false
	}
	return calls[0], true
}

func compileForRange(
	ctx context.Context[parser.IForStatementContext],
	clause parser.IForClauseContext,
	name string,
	funcCall parser.IFunctionCallSuffixContext,
) error {
	loopScope, err := ctx.Scope.GetChildByParserRule(ctx.AST)
	if err != nil {
		return err
	}
	loopCtx := ctx.WithScope(loopScope)

	args := funcCall.ArgumentList().AllExpression()
	varSym, err := loopScope.Resolve(ctx, name)
	if err != nil {
		return err
	}
	loopVarIdx := varSym.ID

	var startExpr, endExpr, stepExpr parser.IExpressionContext
	switch len(args) {
	case 1:
		endExpr = args[0]
	case 2:
		startExpr = args[0]
		endExpr = args[1]
	case 3:
		startExpr = args[0]
		endExpr = args[1]
		stepExpr = args[2]
	}

	loopVarType := varSym.Type

	if startExpr != nil {
		if _, err = expression.Compile(context.Child(loopCtx, startExpr)); err != nil {
			return err
		}
		if err = castIfNeeded(loopCtx, startExpr, loopVarType); err != nil {
			return err
		}
	} else {
		emitZero(ctx.Writer, loopVarType)
	}
	ctx.Writer.WriteLocalSet(loopVarIdx)

	limitSym, err := loopScope.Resolve(ctx, "__for_limit")
	if err != nil {
		return err
	}
	limitIdx := limitSym.ID
	if _, err = expression.Compile(context.Child(loopCtx, endExpr)); err != nil {
		return err
	}
	if err = castIfNeeded(loopCtx, endExpr, loopVarType); err != nil {
		return err
	}
	ctx.Writer.WriteLocalSet(limitIdx)

	var stepIdx int
	if stepExpr != nil {
		stepSym, err := loopScope.Resolve(ctx, "__for_step")
		if err != nil {
			return err
		}
		stepIdx = stepSym.ID
		if _, err = expression.Compile(context.Child(loopCtx, stepExpr)); err != nil {
			return err
		}
		if err = castIfNeeded(loopCtx, stepExpr, loopVarType); err != nil {
			return err
		}
		ctx.Writer.WriteLocalSet(stepIdx)
	}

	// block $break
	ctx.Writer.WriteBlock(wasm.BlockTypeEmpty)
	loopCtx.LoopDepth++
	breakDepth := loopCtx.LoopDepth

	// loop $loop_header
	ctx.Writer.WriteLoop(wasm.BlockTypeEmpty)
	loopCtx.LoopDepth++

	// exit condition
	if stepExpr != nil {
		// Direction-aware: check step sign at runtime
		ctx.Writer.WriteLocalGet(stepIdx)
		emitZero(ctx.Writer, loopVarType)
		if err = ctx.Writer.WriteBinaryOpInferred(">", loopVarType); err != nil {
			return err
		}
		ctx.Writer.WriteIf(wasm.BlockTypeI32)
		ctx.Writer.WriteLocalGet(loopVarIdx)
		ctx.Writer.WriteLocalGet(limitIdx)
		if err = ctx.Writer.WriteBinaryOpInferred(">=", loopVarType); err != nil {
			return err
		}
		ctx.Writer.WriteElse()
		ctx.Writer.WriteLocalGet(loopVarIdx)
		ctx.Writer.WriteLocalGet(limitIdx)
		if err = ctx.Writer.WriteBinaryOpInferred("<=", loopVarType); err != nil {
			return err
		}
		ctx.Writer.WriteEnd()
		ctx.Writer.WriteBrIf(1)
	} else {
		ctx.Writer.WriteLocalGet(loopVarIdx)
		ctx.Writer.WriteLocalGet(limitIdx)
		if err = ctx.Writer.WriteBinaryOpInferred(">=", loopVarType); err != nil {
			return err
		}
		ctx.Writer.WriteBrIf(1)
	}

	// block $continue — continue lands at end of this block, before increment
	ctx.Writer.WriteBlock(wasm.BlockTypeEmpty)
	loopCtx.LoopDepth++
	continueDepth := loopCtx.LoopDepth

	loopCtx.LoopStack = append(loopCtx.LoopStack, context.LoopEntry{
		BreakDepth:    breakDepth,
		ContinueDepth: continueDepth,
	})

	// body
	if block := ctx.AST.Block(); block != nil {
		if _, err = CompileBlock(context.Child(loopCtx, block)); err != nil {
			return err
		}
	}

	// end block $continue
	ctx.Writer.WriteEnd()

	// increment: i = i + step
	ctx.Writer.WriteLocalGet(loopVarIdx)
	if stepExpr != nil {
		ctx.Writer.WriteLocalGet(stepIdx)
	} else {
		emitOne(ctx.Writer, loopVarType)
	}
	if err = ctx.Writer.WriteBinaryOpInferred("+", loopVarType); err != nil {
		return err
	}
	ctx.Writer.WriteLocalSet(loopVarIdx)

	// br $loop_header
	ctx.Writer.WriteBr(0)

	// end loop
	ctx.Writer.WriteEnd()

	// end block $break
	ctx.Writer.WriteEnd()

	return nil
}

func compileForSeriesIteration(
	ctx context.Context[parser.IForStatementContext],
	clause parser.IForClauseContext,
	elemName string,
	indexName string,
	expr parser.IExpressionContext,
) error {
	loopScope, err := ctx.Scope.GetChildByParserRule(ctx.AST)
	if err != nil {
		return err
	}
	loopCtx := ctx.WithScope(loopScope)

	handleSym, err := loopScope.Resolve(ctx, "__for_handle")
	if err != nil {
		return err
	}
	lenSym, err := loopScope.Resolve(ctx, "__for_len")
	if err != nil {
		return err
	}
	idxSym, err := loopScope.Resolve(ctx, "__for_idx")
	if err != nil {
		return err
	}
	elemSym, err := loopScope.Resolve(ctx, elemName)
	if err != nil {
		return err
	}

	handleIdx := handleSym.ID
	lenIdx := lenSym.ID
	idxIdx := idxSym.ID
	elemIdx := elemSym.ID
	elemType := elemSym.Type

	// Compile the iterable expression (series handle)
	if _, err = expression.Compile(context.Child(loopCtx, expr)); err != nil {
		return err
	}
	ctx.Writer.WriteLocalSet(handleIdx)

	// len = series.len(handle)
	ctx.Writer.WriteLocalGet(handleIdx)
	ctx.Resolver.EmitSeriesLen(ctx.Writer, ctx.WriterID)
	// series.len returns i64, wrap to i32
	ctx.Writer.WriteOpcode(wasm.OpI32WrapI64)
	ctx.Writer.WriteLocalSet(lenIdx)

	// idx = 0
	ctx.Writer.WriteI32Const(0)
	ctx.Writer.WriteLocalSet(idxIdx)

	// block $break
	ctx.Writer.WriteBlock(wasm.BlockTypeEmpty)
	loopCtx.LoopDepth++
	breakDepth := loopCtx.LoopDepth

	// loop $loop_header
	ctx.Writer.WriteLoop(wasm.BlockTypeEmpty)
	loopCtx.LoopDepth++

	// exit condition: idx >= len => br $break
	ctx.Writer.WriteLocalGet(idxIdx)
	ctx.Writer.WriteLocalGet(lenIdx)
	ctx.Writer.WriteOpcode(wasm.OpI32GeS)
	ctx.Writer.WriteBrIf(1)

	// elem = series.index(handle, idx)
	ctx.Writer.WriteLocalGet(handleIdx)
	ctx.Writer.WriteLocalGet(idxIdx)
	ctx.Resolver.EmitSeriesIndex(ctx.Writer, ctx.WriterID, elemType)
	ctx.Writer.WriteLocalSet(elemIdx)

	// If two-ident form, set index variable
	if indexName != "" {
		indexSym, err := loopScope.Resolve(ctx, indexName)
		if err != nil {
			return err
		}
		ctx.Writer.WriteLocalGet(idxIdx)
		ctx.Writer.WriteLocalSet(indexSym.ID)
	}

	// block $continue — continue lands at end of this block, before increment
	ctx.Writer.WriteBlock(wasm.BlockTypeEmpty)
	loopCtx.LoopDepth++
	continueDepth := loopCtx.LoopDepth

	loopCtx.LoopStack = append(loopCtx.LoopStack, context.LoopEntry{
		BreakDepth:    breakDepth,
		ContinueDepth: continueDepth,
	})

	// body
	if block := ctx.AST.Block(); block != nil {
		if _, err = CompileBlock(context.Child(loopCtx, block)); err != nil {
			return err
		}
	}

	// end block $continue
	ctx.Writer.WriteEnd()

	// idx++
	ctx.Writer.WriteLocalGet(idxIdx)
	ctx.Writer.WriteI32Const(1)
	ctx.Writer.WriteOpcode(wasm.OpI32Add)
	ctx.Writer.WriteLocalSet(idxIdx)

	// br $loop_header
	ctx.Writer.WriteBr(0)

	// end loop
	ctx.Writer.WriteEnd()

	// end block $break
	ctx.Writer.WriteEnd()

	return nil
}

func compileForCondition(
	ctx context.Context[parser.IForStatementContext],
	expr parser.IExpressionContext,
) error {
	loopScope, err := ctx.Scope.GetChildByParserRule(ctx.AST)
	if err != nil {
		return err
	}
	loopCtx := ctx.WithScope(loopScope)

	// block $break
	ctx.Writer.WriteBlock(wasm.BlockTypeEmpty)
	loopCtx.LoopDepth++
	breakDepth := loopCtx.LoopDepth

	// loop $continue
	ctx.Writer.WriteLoop(wasm.BlockTypeEmpty)
	loopCtx.LoopDepth++
	continueDepth := loopCtx.LoopDepth

	loopCtx.LoopStack = append(loopCtx.LoopStack, context.LoopEntry{
		BreakDepth:    breakDepth,
		ContinueDepth: continueDepth,
	})

	// condition
	if _, err = expression.Compile(context.Child(loopCtx, expr)); err != nil {
		return err
	}
	ctx.Writer.WriteI32Eqz()
	ctx.Writer.WriteBrIf(1) // br to $break if condition is false

	// body
	if block := ctx.AST.Block(); block != nil {
		if _, err = CompileBlock(context.Child(loopCtx, block)); err != nil {
			return err
		}
	}

	// br $continue
	ctx.Writer.WriteBr(0)

	// end loop
	ctx.Writer.WriteEnd()

	// end block
	ctx.Writer.WriteEnd()

	return nil
}

func compileForInfinite(
	ctx context.Context[parser.IForStatementContext],
) error {
	loopScope, err := ctx.Scope.GetChildByParserRule(ctx.AST)
	if err != nil {
		return err
	}
	loopCtx := ctx.WithScope(loopScope)

	// block $break
	ctx.Writer.WriteBlock(wasm.BlockTypeEmpty)
	loopCtx.LoopDepth++
	breakDepth := loopCtx.LoopDepth

	// loop $continue
	ctx.Writer.WriteLoop(wasm.BlockTypeEmpty)
	loopCtx.LoopDepth++
	continueDepth := loopCtx.LoopDepth

	loopCtx.LoopStack = append(loopCtx.LoopStack, context.LoopEntry{
		BreakDepth:    breakDepth,
		ContinueDepth: continueDepth,
	})

	// body
	if block := ctx.AST.Block(); block != nil {
		if _, err = CompileBlock(context.Child(loopCtx, block)); err != nil {
			return err
		}
	}

	// br $continue
	ctx.Writer.WriteBr(0)

	// end loop
	ctx.Writer.WriteEnd()

	// end block
	ctx.Writer.WriteEnd()

	return nil
}

func compileBreakStatement(
	ctx context.Context[parser.IBreakStatementContext],
) error {
	if len(ctx.LoopStack) == 0 {
		return errors.New("break outside loop")
	}
	entry := ctx.LoopStack[len(ctx.LoopStack)-1]
	label := uint32(ctx.LoopDepth - entry.BreakDepth)
	ctx.Writer.WriteBr(label)
	return nil
}

func compileContinueStatement(
	ctx context.Context[parser.IContinueStatementContext],
) error {
	if len(ctx.LoopStack) == 0 {
		return errors.New("continue outside loop")
	}
	entry := ctx.LoopStack[len(ctx.LoopStack)-1]
	label := uint32(ctx.LoopDepth - entry.ContinueDepth)
	ctx.Writer.WriteBr(label)
	return nil
}

func castIfNeeded[ASTNode parser.IExpressionContext](
	ctx context.Context[parser.IForStatementContext],
	expr ASTNode,
	target types.Type,
) error {
	exprType, ok := ctx.TypeMap[expr]
	if !ok {
		return nil
	}
	if !types.Equal(exprType, target) {
		return expression.EmitCast(ctx, exprType, target)
	}
	return nil
}

func emitZero(w *wasm.Writer, t types.Type) {
	switch wasm.ConvertType(t) {
	case wasm.I32:
		w.WriteI32Const(0)
	case wasm.I64:
		w.WriteI64Const(0)
	case wasm.F32:
		w.WriteF32Const(0)
	case wasm.F64:
		w.WriteF64Const(0)
	}
}

func emitOne(w *wasm.Writer, t types.Type) {
	switch wasm.ConvertType(t) {
	case wasm.I32:
		w.WriteI32Const(1)
	case wasm.I64:
		w.WriteI64Const(1)
	case wasm.F32:
		w.WriteF32Const(1)
	case wasm.F64:
		w.WriteF64Const(1)
	}
}
