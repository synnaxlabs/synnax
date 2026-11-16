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
		return false, compileForSingleIdent(ctx, idents[0].GetText(), expr)
	case expr != nil:
		return false, compileForCondition(ctx, expr)
	default:
		return false, compileForInfinite(ctx)
	}
}

func compileForSingleIdent(
	ctx context.Context[parser.IForStatementContext],
	name string,
	expr parser.IExpressionContext,
) error {
	if funcCall, ok := isRangeCallExpr(expr); ok {
		return compileForRange(ctx, name, funcCall)
	}
	return compileForSeriesIteration(ctx, name, "", expr)
}

func compileForTwoIdent(
	ctx context.Context[parser.IForStatementContext],
	clause parser.IForClauseContext,
	expr parser.IExpressionContext,
) error {
	idents := clause.AllIDENTIFIER()
	indexName := idents[0].GetText()
	elemName := idents[1].GetText()
	return compileForSeriesIteration(ctx, elemName, indexName, expr)
}

func isRangeCallExpr(
	expr parser.IExpressionContext,
) (parser.IFunctionCallSuffixContext, bool) {
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
		AllUnaryExpression()[0].
		PowerExpression().
		PostfixExpression()
	if postfix == nil {
		return nil, false
	}
	calls := postfix.AllFunctionCallSuffix()
	if len(calls) != 1 || len(postfix.AllIndexOrSlice()) > 0 {
		return nil, false
	}
	return calls[0], true
}

func compileForRange(
	ctx context.Context[parser.IForStatementContext],
	name string,
	funcCall parser.IFunctionCallSuffixContext,
) error {
	loopScope, err := ctx.Scope.GetChildByParserRule(ctx.AST)
	if err != nil {
		return err
	}
	loopCtx := ctx.WithScope(loopScope)
	r, err := initRangeLoop(loopCtx, name, funcCall.ArgumentList().AllExpression())
	if err != nil {
		return err
	}
	loopCtx, breakDepth := openLoop(loopCtx)
	if err = r.emitExitCheck(ctx.Writer); err != nil {
		return err
	}
	if err = compileLoopBody(loopCtx, breakDepth); err != nil {
		return err
	}
	if err = r.emitIncrement(ctx.Writer); err != nil {
		return err
	}
	closeLoop(ctx.Writer)
	return nil
}

// rangeLoop holds the locals of a for loop over range(...).
type rangeLoop struct {
	varType types.Type
	varIdx  int
	// limitIdx and stepIdx hold the end and step, evaluated once before the loop.
	limitIdx int
	stepIdx  int
	stepped  bool
}

// initRangeLoop writes the initial values of the loop variable, limit, and step.
func initRangeLoop(
	loopCtx context.Context[parser.IForStatementContext],
	name string,
	args []parser.IExpressionContext,
) (rangeLoop, error) {
	varSym, err := loopCtx.Scope.Resolve(loopCtx, name)
	if err != nil {
		return rangeLoop{}, err
	}
	r := rangeLoop{varType: varSym.Type, varIdx: varSym.ID}
	var startExpr, endExpr, stepExpr parser.IExpressionContext
	switch len(args) {
	case 1:
		endExpr = args[0]
	case 2:
		startExpr, endExpr = args[0], args[1]
	case 3:
		startExpr, endExpr, stepExpr = args[0], args[1], args[2]
	}
	hintCtx := loopCtx.WithHint(r.varType)
	if startExpr != nil {
		if _, err = expression.Compile(hintCtx.Child(startExpr)); err != nil {
			return rangeLoop{}, err
		}
	} else {
		emitZero(loopCtx.Writer, r.varType)
	}
	loopCtx.Writer.WriteLocalSet(r.varIdx)
	if r.limitIdx, err = initLoopLocal(hintCtx, "__for_limit", endExpr); err != nil {
		return rangeLoop{}, err
	}
	if stepExpr == nil {
		return r, nil
	}
	r.stepped = true
	if r.stepIdx, err = initLoopLocal(hintCtx, "__for_step", stepExpr); err != nil {
		return rangeLoop{}, err
	}
	return r, nil
}

// initLoopLocal evaluates expr into the loop-scoped local with the given name and
// returns the local's index.
func initLoopLocal(
	ctx context.Context[parser.IForStatementContext],
	name string,
	expr parser.IExpressionContext,
) (int, error) {
	sym, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		return 0, err
	}
	if _, err = expression.Compile(ctx.Child(expr)); err != nil {
		return 0, err
	}
	ctx.Writer.WriteLocalSet(sym.ID)
	return sym.ID, nil
}

// emitExitCheck breaks out of the loop once the variable reaches the limit. A
// negative step counts down.
func (r rangeLoop) emitExitCheck(w *wasm.Writer) error {
	if !r.stepped {
		w.WriteLocalGet(r.varIdx)
		w.WriteLocalGet(r.limitIdx)
		if err := w.WriteBinaryOpInferred(">=", r.varType); err != nil {
			return err
		}
		w.WriteBrIf(1)
		return nil
	}
	w.WriteLocalGet(r.stepIdx)
	emitZero(w, r.varType)
	if err := w.WriteBinaryOpInferred(">", r.varType); err != nil {
		return err
	}
	w.WriteIf(wasm.BlockTypeI32)
	w.WriteLocalGet(r.varIdx)
	w.WriteLocalGet(r.limitIdx)
	if err := w.WriteBinaryOpInferred(">=", r.varType); err != nil {
		return err
	}
	w.WriteElse()
	w.WriteLocalGet(r.varIdx)
	w.WriteLocalGet(r.limitIdx)
	if err := w.WriteBinaryOpInferred("<=", r.varType); err != nil {
		return err
	}
	w.WriteEnd()
	w.WriteBrIf(1)
	return nil
}

func (r rangeLoop) emitIncrement(w *wasm.Writer) error {
	if r.stepped {
		if err := emitStepReachesLimit(
			w,
			r.varType,
			r.varIdx,
			r.limitIdx,
			r.stepIdx,
		); err != nil {
			return err
		}
		w.WriteBrIf(1)
	}
	w.WriteLocalGet(r.varIdx)
	if r.stepped {
		w.WriteLocalGet(r.stepIdx)
	} else {
		emitOne(w, r.varType)
	}
	if err := w.WriteBinaryOpInferred("+", r.varType); err != nil {
		return err
	}
	w.WriteLocalSet(r.varIdx)
	return nil
}

// openLoop writes the $break block and the loop header with its budget check. It
// returns the context inside the loop and the block depth that break targets.
func openLoop(
	loopCtx context.Context[parser.IForStatementContext],
) (context.Context[parser.IForStatementContext], int) {
	loopCtx.Writer.WriteBlock(wasm.BlockTypeEmpty)
	loopCtx = loopCtx.EnterBlock()
	breakDepth := loopCtx.BlockDepth()
	loopCtx.Writer.WriteLoop(wasm.BlockTypeEmpty)
	loopCtx = loopCtx.EnterBlock()
	loopCtx.Writer.WriteLoopBudget(loopCtx.LoopBudget)
	return loopCtx, breakDepth
}

// compileLoopBody compiles the loop body inside a $continue block, so continue lands
// before the increment.
func compileLoopBody(
	loopCtx context.Context[parser.IForStatementContext],
	breakDepth int,
) error {
	loopCtx.Writer.WriteBlock(wasm.BlockTypeEmpty)
	loopCtx = loopCtx.EnterBlock()
	bodyCtx := loopCtx.EnterLoop(context.LoopEntry{
		BreakDepth:    breakDepth,
		ContinueDepth: loopCtx.BlockDepth(),
	})
	if block := loopCtx.AST.Block(); block != nil {
		if _, err := CompileBlock(bodyCtx.Child(block)); err != nil {
			return err
		}
	}
	loopCtx.Writer.WriteEnd()
	return nil
}

// closeLoop branches back to the loop header, then ends the loop and $break block.
func closeLoop(w *wasm.Writer) {
	w.WriteBr(0)
	w.WriteEnd()
	w.WriteEnd()
}

func compileForSeriesIteration(
	ctx context.Context[parser.IForStatementContext],
	elemName, indexName string,
	expr parser.IExpressionContext,
) error {
	loopScope, err := ctx.Scope.GetChildByParserRule(ctx.AST)
	if err != nil {
		return err
	}
	loopCtx := ctx.WithScope(loopScope)
	l, err := resolveSeriesLoop(loopCtx, elemName)
	if err != nil {
		return err
	}
	if _, err = expression.Compile(loopCtx.Child(expr)); err != nil {
		return err
	}
	w := ctx.Writer
	w.WriteLocalSet(l.handleIdx)
	w.WriteLocalGet(l.handleIdx)
	ctx.Resolver.EmitSeriesLen(w, ctx.WriterID)
	w.WriteOpcode(wasm.OpI32WrapI64)
	w.WriteLocalSet(l.lenIdx)
	w.WriteI32Const(0)
	w.WriteLocalSet(l.idxIdx)

	loopCtx, breakDepth := openLoop(loopCtx)
	w.WriteLocalGet(l.idxIdx)
	w.WriteLocalGet(l.lenIdx)
	w.WriteOpcode(wasm.OpI32GeS)
	w.WriteBrIf(1)

	w.WriteLocalGet(l.handleIdx)
	w.WriteLocalGet(l.idxIdx)
	ctx.Resolver.EmitSeriesIndex(w, ctx.WriterID, l.elemType)
	w.WriteLocalSet(l.elemIdx)
	if indexName != "" {
		indexSym, err := loopScope.Resolve(ctx, indexName)
		if err != nil {
			return err
		}
		w.WriteLocalGet(l.idxIdx)
		w.WriteLocalSet(indexSym.ID)
	}
	if err = compileLoopBody(loopCtx, breakDepth); err != nil {
		return err
	}
	w.WriteLocalGet(l.idxIdx)
	w.WriteI32Const(1)
	w.WriteOpcode(wasm.OpI32Add)
	w.WriteLocalSet(l.idxIdx)
	closeLoop(w)
	return nil
}

// seriesLoop holds the locals of a for loop over the elements of a series.
type seriesLoop struct {
	elemType  types.Type
	handleIdx int
	lenIdx    int
	idxIdx    int
	elemIdx   int
}

func resolveSeriesLoop(
	loopCtx context.Context[parser.IForStatementContext],
	elemName string,
) (seriesLoop, error) {
	scope := loopCtx.Scope
	handleSym, err := scope.Resolve(loopCtx, "__for_handle")
	if err != nil {
		return seriesLoop{}, err
	}
	lenSym, err := scope.Resolve(loopCtx, "__for_len")
	if err != nil {
		return seriesLoop{}, err
	}
	idxSym, err := scope.Resolve(loopCtx, "__for_idx")
	if err != nil {
		return seriesLoop{}, err
	}
	elemSym, err := scope.Resolve(loopCtx, elemName)
	if err != nil {
		return seriesLoop{}, err
	}
	return seriesLoop{
		elemType:  elemSym.Type,
		handleIdx: handleSym.ID,
		lenIdx:    lenSym.ID,
		idxIdx:    idxSym.ID,
		elemIdx:   elemSym.ID,
	}, nil
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
	loopCtx = loopCtx.EnterBlock()
	breakDepth := loopCtx.BlockDepth()

	// loop $continue
	ctx.Writer.WriteLoop(wasm.BlockTypeEmpty)
	loopCtx = loopCtx.EnterBlock()
	ctx.Writer.WriteLoopBudget(ctx.LoopBudget)
	continueDepth := loopCtx.BlockDepth()

	bodyCtx := loopCtx.EnterLoop(context.LoopEntry{
		BreakDepth:    breakDepth,
		ContinueDepth: continueDepth,
	})

	if _, err = expression.Compile(bodyCtx.Child(expr)); err != nil {
		return err
	}
	ctx.Writer.WriteI32Eqz()
	ctx.Writer.WriteBrIf(1) // br to $break if condition is false

	if block := ctx.AST.Block(); block != nil {
		if _, err = CompileBlock(bodyCtx.Child(block)); err != nil {
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
	loopCtx = loopCtx.EnterBlock()
	breakDepth := loopCtx.BlockDepth()

	// loop $continue
	ctx.Writer.WriteLoop(wasm.BlockTypeEmpty)
	loopCtx = loopCtx.EnterBlock()
	ctx.Writer.WriteLoopBudget(ctx.LoopBudget)
	continueDepth := loopCtx.BlockDepth()

	bodyCtx := loopCtx.EnterLoop(context.LoopEntry{
		BreakDepth:    breakDepth,
		ContinueDepth: continueDepth,
	})

	if block := ctx.AST.Block(); block != nil {
		if _, err = CompileBlock(bodyCtx.Child(block)); err != nil {
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
	entry, ok := ctx.CurrentLoop()
	if !ok {
		return errors.New("break outside loop")
	}
	label := uint32(ctx.BlockDepth() - entry.BreakDepth)
	ctx.Writer.WriteBr(label)
	return nil
}

func compileContinueStatement(
	ctx context.Context[parser.IContinueStatementContext],
) error {
	entry, ok := ctx.CurrentLoop()
	if !ok {
		return errors.New("continue outside loop")
	}
	label := uint32(ctx.BlockDepth() - entry.ContinueDepth)
	ctx.Writer.WriteBr(label)
	return nil
}

// emitStepReachesLimit emits a bool that is true when one more step takes the loop
// variable to the limit or past it. The loop ends there, so the variable cannot wrap
// past the bound of t. The distance to the limit is compared unsigned, so it holds
// across the full range of a signed type.
func emitStepReachesLimit(
	w *wasm.Writer,
	t types.Type,
	loopVarIdx, limitIdx, stepIdx int,
) error {
	lessOrEqual := wasm.OpI32LeU
	if wasm.ConvertType(t) == wasm.I64 {
		lessOrEqual = wasm.OpI64LeU
	}
	w.WriteLocalGet(stepIdx)
	emitZero(w, t)
	if err := w.WriteBinaryOpInferred(">", t); err != nil {
		return err
	}
	w.WriteIf(wasm.BlockTypeI32)
	w.WriteLocalGet(limitIdx)
	w.WriteLocalGet(loopVarIdx)
	if err := w.WriteBinaryOpInferred("-", t); err != nil {
		return err
	}
	w.WriteLocalGet(stepIdx)
	w.WriteOpcode(lessOrEqual)
	w.WriteElse()
	w.WriteLocalGet(loopVarIdx)
	w.WriteLocalGet(limitIdx)
	if err := w.WriteBinaryOpInferred("-", t); err != nil {
		return err
	}
	emitZero(w, t)
	w.WriteLocalGet(stepIdx)
	if err := w.WriteBinaryOpInferred("-", t); err != nil {
		return err
	}
	w.WriteOpcode(lessOrEqual)
	w.WriteEnd()
	return nil
}

func emitZero(w *wasm.Writer, t types.Type) {
	switch wasm.ConvertType(t) {
	case wasm.I32:
		w.WriteI32Const(0)
	case wasm.I64:
		w.WriteI64Const(0)
	}
}

func emitOne(w *wasm.Writer, t types.Type) {
	switch wasm.ConvertType(t) {
	case wasm.I32:
		w.WriteI32Const(1)
	case wasm.I64:
		w.WriteI64Const(1)
	}
}
