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
	"github.com/synnaxlabs/arc/symbol"
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

	varSym, err := loopScope.Resolve(ctx, name)
	if err != nil {
		return err
	}
	r, err := compileRangeInit(
		loopCtx.WithHint(varSym.Type),
		loopScope,
		varSym,
		funcCall.ArgumentList().AllExpression(),
	)
	if err != nil {
		return err
	}

	return compileSteppedLoop(
		loopCtx,
		func() error { return r.emitExitCheck(ctx.Writer) },
		func() error { return r.emitAdvance(ctx.Writer) },
	)
}

// compileSteppedLoop emits a loop that runs header at the top of each pass and advance
// after the body. A break leaves the loop, and a continue jumps to advance.
func compileSteppedLoop(
	loopCtx context.Context[parser.IForStatementContext],
	header, advance func() error,
) error {
	w := loopCtx.Writer
	// block $break
	w.WriteBlock(wasm.BlockTypeEmpty)
	loopCtx = loopCtx.EnterBlock()
	breakDepth := loopCtx.BlockDepth()

	// loop $loop_header
	w.WriteLoop(wasm.BlockTypeEmpty)
	loopCtx = loopCtx.EnterBlock()
	w.WriteLoopBudget(loopCtx.LoopBudget)
	if err := header(); err != nil {
		return err
	}

	// block $continue — continue lands at end of this block, before increment
	w.WriteBlock(wasm.BlockTypeEmpty)
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
	// end block $continue
	w.WriteEnd()

	if err := advance(); err != nil {
		return err
	}
	// br $loop_header
	w.WriteBr(0)
	// end loop
	w.WriteEnd()
	// end block $break
	w.WriteEnd()
	return nil
}

// rangeLoop holds the locals of a range loop.
type rangeLoop struct {
	varType                   types.Type
	varIdx, limitIdx, stepIdx int
	hasStep                   bool
}

// compileRangeInit stores the start, limit, and step of a range loop in their locals.
func compileRangeInit(
	ctx context.Context[parser.IForStatementContext],
	loopScope *symbol.Symbol,
	varSym *symbol.Symbol,
	args []parser.IExpressionContext,
) (rangeLoop, error) {
	r := rangeLoop{varType: varSym.Type, varIdx: varSym.ID}
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
	if startExpr != nil {
		if _, err := expression.Compile(ctx.Child(startExpr)); err != nil {
			return rangeLoop{}, err
		}
	} else {
		emitZero(ctx.Writer, r.varType)
	}
	ctx.Writer.WriteLocalSet(r.varIdx)
	limitSym, err := loopScope.Resolve(ctx, "__for_limit")
	if err != nil {
		return rangeLoop{}, err
	}
	r.limitIdx = limitSym.ID
	if _, err = expression.Compile(ctx.Child(endExpr)); err != nil {
		return rangeLoop{}, err
	}
	ctx.Writer.WriteLocalSet(r.limitIdx)
	if stepExpr == nil {
		return r, nil
	}
	r.hasStep = true
	stepSym, err := loopScope.Resolve(ctx, "__for_step")
	if err != nil {
		return rangeLoop{}, err
	}
	r.stepIdx = stepSym.ID
	if _, err = expression.Compile(ctx.Child(stepExpr)); err != nil {
		return rangeLoop{}, err
	}
	ctx.Writer.WriteLocalSet(r.stepIdx)
	return r, nil
}

// emitExitCheck breaks out of the loop once the loop variable reaches the limit. With
// a step, the direction of the step picks the comparison.
func (r rangeLoop) emitExitCheck(w *wasm.Writer) error {
	if !r.hasStep {
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

// emitAdvance adds the step to the loop variable. With a step, it first breaks out
// when the step would reach the limit.
func (r rangeLoop) emitAdvance(w *wasm.Writer) error {
	if r.hasStep {
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
	if r.hasStep {
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

	locals, err := resolveSeriesLoopLocals(ctx, loopScope, elemName)
	if err != nil {
		return err
	}
	if err = emitSeriesLoopInit(loopCtx, expr, locals); err != nil {
		return err
	}

	return compileSteppedLoop(
		loopCtx,
		func() error {
			emitSeriesElement(ctx, locals)
			if indexName == "" {
				return nil
			}
			indexSym, err := loopScope.Resolve(ctx, indexName)
			if err != nil {
				return err
			}
			ctx.Writer.WriteLocalGet(locals.index)
			ctx.Writer.WriteLocalSet(indexSym.ID)
			return nil
		},
		func() error {
			ctx.Writer.WriteLocalGet(locals.index)
			ctx.Writer.WriteI32Const(1)
			ctx.Writer.WriteOpcode(wasm.OpI32Add)
			ctx.Writer.WriteLocalSet(locals.index)
			return nil
		},
	)
}

type seriesLoopLocals struct {
	handle, length, index, elem int
	elemType                    types.Type
}

func resolveSeriesLoopLocals(
	ctx context.Context[parser.IForStatementContext],
	loopScope *symbol.Symbol,
	elemName string,
) (seriesLoopLocals, error) {
	var l seriesLoopLocals
	for _, local := range []struct {
		name string
		idx  *int
	}{
		{"__for_handle", &l.handle},
		{"__for_len", &l.length},
		{"__for_idx", &l.index},
	} {
		sym, err := loopScope.Resolve(ctx, local.name)
		if err != nil {
			return seriesLoopLocals{}, err
		}
		*local.idx = sym.ID
	}
	elemSym, err := loopScope.Resolve(ctx, elemName)
	if err != nil {
		return seriesLoopLocals{}, err
	}
	l.elem, l.elemType = elemSym.ID, elemSym.Type
	return l, nil
}

// emitSeriesLoopInit stores the iterated series, its length, and a zero index in
// their locals.
func emitSeriesLoopInit(
	ctx context.Context[parser.IForStatementContext],
	expr parser.IExpressionContext,
	l seriesLoopLocals,
) error {
	if _, err := expression.Compile(ctx.Child(expr)); err != nil {
		return err
	}
	ctx.Writer.WriteLocalSet(l.handle)
	ctx.Writer.WriteLocalGet(l.handle)
	ctx.Resolver.EmitSeriesLen(ctx.Writer, ctx.WriterID)
	ctx.Writer.WriteOpcode(wasm.OpI32WrapI64)
	ctx.Writer.WriteLocalSet(l.length)
	ctx.Writer.WriteI32Const(0)
	ctx.Writer.WriteLocalSet(l.index)
	return nil
}

// emitSeriesElement breaks out of the loop past the last element, else stores the
// current element in its local.
func emitSeriesElement(
	ctx context.Context[parser.IForStatementContext],
	l seriesLoopLocals,
) {
	ctx.Writer.WriteLocalGet(l.index)
	ctx.Writer.WriteLocalGet(l.length)
	ctx.Writer.WriteOpcode(wasm.OpI32GeS)
	ctx.Writer.WriteBrIf(1)
	ctx.Writer.WriteLocalGet(l.handle)
	ctx.Writer.WriteLocalGet(l.index)
	ctx.Resolver.EmitSeriesIndex(ctx.Writer, ctx.WriterID, l.elemType)
	ctx.Writer.WriteLocalSet(l.elem)
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
	case wasm.F32, wasm.F64:
		panic("loop counter must be an integer, got " + t.String())
	}
}

func emitOne(w *wasm.Writer, t types.Type) {
	switch wasm.ConvertType(t) {
	case wasm.I32:
		w.WriteI32Const(1)
	case wasm.I64:
		w.WriteI64Const(1)
	case wasm.F32, wasm.F64:
		panic("loop counter must be an integer, got " + t.String())
	}
}
