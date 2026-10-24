// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression

import (
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

func validateNonZeroArray[T any](exprs []T, opName string) []T {
	if len(exprs) == 0 {
		panic(errors.Newf("cannot compile an empty %s expression", opName))
	}
	return exprs
}

func validateNonZero(expr parser.IPostfixExpressionContext, opName string) {
	if expr == nil {
		panic(errors.Newf("cannot compile an empty %s expression", opName))
	}
}

// Compile compiles an expression and returns its type
func Compile(
	ctx context.Context[parser.IExpressionContext],
) (types.Type, error) {
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
		return compileLogicalOr(ctx.Child(logicalOr))
	}
	return types.Type{}, errors.New("unknown expression type")
}

func compileLogicalOr(
	ctx context.Context[parser.ILogicalOrExpressionContext],
) (types.Type, error) {
	ands := validateNonZeroArray(ctx.AST.AllLogicalAndExpression(), "logical OR")
	if len(ands) == 1 {
		return compileLogicalAnd(ctx.Child(ands[0]))
	}
	return compileLogicalOrImpl(ctx)
}

func compileLogicalAnd(
	ctx context.Context[parser.ILogicalAndExpressionContext],
) (types.Type, error) {
	eqs := validateNonZeroArray(ctx.AST.AllEqualityExpression(), "logical AND")
	if len(eqs) == 1 {
		return compileEquality(ctx.Child(eqs[0]))
	}
	return compileLogicalAndImpl(ctx)
}

func compileEquality(
	ctx context.Context[parser.IEqualityExpressionContext],
) (types.Type, error) {
	rels := validateNonZeroArray(ctx.AST.AllRelationalExpression(), "equality")
	if len(rels) == 1 {
		return compileRelational(ctx.Child(rels[0]))
	}
	return compileBinaryEquality(ctx)
}

func compileRelational(
	ctx context.Context[parser.IRelationalExpressionContext],
) (types.Type, error) {
	adds := validateNonZeroArray(ctx.AST.AllAdditiveExpression(), "relational")
	if len(adds) == 1 {
		return compileAdditive(ctx.Child(adds[0]))
	}
	return compileBinaryRelational(ctx)
}

func compileAdditive(
	ctx context.Context[parser.IAdditiveExpressionContext],
) (types.Type, error) {
	muls := validateNonZeroArray(ctx.AST.AllMultiplicativeExpression(), "additive")
	if len(muls) == 1 {
		return compileMultiplicative(ctx.Child(muls[0]))
	}
	return compileBinaryAdditive(ctx)
}

func compileMultiplicative(
	ctx context.Context[parser.IMultiplicativeExpressionContext],
) (types.Type, error) {
	unaries := validateNonZeroArray(ctx.AST.AllUnaryExpression(), "multiplicative")
	if len(unaries) == 1 {
		return compileUnary(ctx.Child(unaries[0]))
	}
	return compileBinaryMultiplicative(ctx)
}

func compilePower(
	ctx context.Context[parser.IPowerExpressionContext],
) (types.Type, error) {
	postfix := ctx.AST.PostfixExpression()
	validateNonZero(postfix, "power")

	baseType, err := compilePostfix(ctx.Child(postfix))
	if err != nil {
		return types.Type{}, err
	}

	if ctx.AST.CARET() == nil || ctx.AST.UnaryExpression() == nil {
		return baseType, nil
	}

	_, err = compileUnary(
		ctx.Child(ctx.AST.UnaryExpression()).WithHint(baseType.Unwrap()),
	)
	if err != nil {
		return types.Type{}, err
	}

	ctx.Resolver.EmitMathPow(ctx.Writer, ctx.WriterID, baseType)

	return baseType, nil
}

func compilePostfix(
	ctx context.Context[parser.IPostfixExpressionContext],
) (types.Type, error) {
	primary := ctx.AST.PrimaryExpression()
	funcCalls := ctx.AST.AllFunctionCallSuffix()

	head, tail := parser.PrimaryNameParts(primary)
	if len(funcCalls) > 0 && parser.PrimaryName(primary) != "" {
		return CompileCall(ctx, head, tail, symbol.CallArguments(funcCalls[0]))
	}

	currentType, err := compilePrimary(ctx.Child(primary))
	if err != nil {
		return types.Type{}, err
	}

	for _, indexOrSlice := range ctx.AST.AllIndexOrSlice() {
		currentType, err = compileIndexOrSlice(ctx, indexOrSlice, currentType)
		if err != nil {
			return types.Type{}, err
		}
	}

	return currentType, nil
}

// CompileCall emits a call to the function named head.tail with args bound by name or
// position and defaults for the rest. It returns the default output type.
func CompileCall[T antlr.ParserRuleContext](
	ctx context.Context[T],
	head, tail string,
	args []symbol.Argument,
) (types.Type, error) {
	funcName := head
	if tail != "" {
		funcName = head + "." + tail
	}
	// len() dispatches on its argument type, so it bypasses symbol resolution.
	if funcName == "len" || funcName == "series.len" || funcName == "string.len" {
		return compileBuiltinLen(ctx, args, funcName)
	}
	scope, err := ctx.Scope.Resolve(ctx, head)
	if err == nil && tail != "" {
		scope, err = scope.Resolve(ctx, tail)
	}
	if err != nil {
		return types.Type{}, err
	}
	if scope.Kind != symbol.KindFunction {
		return types.Type{}, errors.Newf("%s is not a function", funcName)
	}
	if scope.Exec == symbol.ExecFlow {
		return types.Type{}, errors.Newf(
			"function '%s' cannot be called inside a func block. Use it as a flow statement instead: %s()",
			funcName,
			funcName,
		)
	}
	return compileFunctionCallExpr(ctx, funcName, scope, args)
}

func compileFunctionCallExpr[T antlr.ParserRuleContext](
	ctx context.Context[T],
	funcName string,
	scope *symbol.Symbol,
	args []symbol.Argument,
) (types.Type, error) {
	funcType := scope.Type
	totalCount := len(funcType.Inputs)
	ordered := make([]parser.IExpressionContext, totalCount)
	for _, arg := range args {
		i := arg.Index
		if arg.Name != "" {
			i = funcType.Inputs.GetIndex(arg.Name)
			if i < 0 {
				return types.Type{}, errors.Newf(
					"function %s has no parameter %s",
					funcName, arg.Name,
				)
			}
		}
		if i >= totalCount {
			return types.Type{}, errors.Newf(
				"function %s expects at most %d arguments",
				funcName, totalCount,
			)
		}
		ordered[i] = arg.Expr
	}

	concreteInputs := make(types.Params, totalCount)
	copy(concreteInputs, funcType.Inputs)

	// Track resolved type variables as arguments are compiled so that
	// later arguments sharing the same variable get a concrete hint
	// (e.g. pow(channel_f32, 2) resolves T=f32 on the first arg,
	// then compiles the literal 2 as f32 instead of defaulting to i64).
	varMap := make(map[string]types.Type)

	for i, param := range funcType.Inputs {
		arg := ordered[i]
		if arg == nil {
			if param.Value == nil {
				return types.Type{}, errors.Newf(
					"function %s is missing an argument for parameter %s",
					funcName, param.Name,
				)
			}
			if err := emitLiteralValue(ctx, param.Type, param.Value); err != nil {
				return types.Type{}, errors.Wrapf(
					err,
					"default value for parameter %s",
					param.Name,
				)
			}
			continue
		}
		paramType := param.Type
		hint := paramType
		if paramType.Kind == types.KindVariable {
			if resolved, ok := varMap[paramType.Name]; ok {
				hint = resolved
			}
		}
		argType, err := Compile(ctx.Child(arg).WithHint(hint))
		if err != nil {
			return types.Type{}, errors.Wrapf(err, "argument %d", i)
		}
		concreteInputs[i].Type = argType
		if paramType.Kind == types.KindVariable &&
			argType.Kind != types.KindNumericConstant &&
			argType.Kind != types.KindIntegerConstant &&
			argType.Kind != types.KindFloatConstant {
			varMap[paramType.Name] = argType
		}
		if !types.Equal(argType, paramType) && paramType.Kind != types.KindVariable {
			if err := EmitCast(ctx, argType, paramType); err != nil {
				return types.Type{}, err
			}
		}
	}

	concreteOutputs := make(types.Params, len(funcType.Outputs))
	copy(concreteOutputs, funcType.Outputs)
	for i, out := range concreteOutputs {
		if out.Type.Kind == types.KindVariable {
			if resolved, ok := varMap[out.Type.Name]; ok {
				concreteOutputs[i].Type = resolved
			}
		}
	}

	concreteType := types.Function(types.FunctionProperties{
		Inputs:  concreteInputs,
		Outputs: concreteOutputs,
	})
	target := scope
	if target.Deprecated != nil {
		target = target.Deprecated
	}
	ctx.Resolver.EmitCall(ctx.Writer, ctx.WriterID, target, concreteType)
	defaultOutput, hasDefault := concreteOutputs.Get(ir.DefaultOutputParam)
	if hasDefault {
		return defaultOutput.Type, nil
	}
	return types.Type{}, nil
}

func compileIndexOrSlice(
	ctx context.Context[parser.IPostfixExpressionContext],
	indexOrSlice parser.IIndexOrSliceContext,
	operandType types.Type,
) (types.Type, error) {
	expressions := indexOrSlice.AllExpression()
	isSliceOp := indexOrSlice.COLON() != nil

	if !isSliceOp {
		if operandType.Kind != types.KindSeries {
			return types.Type{}, errors.New(
				"indexing is only supported on series types",
			)
		}
		if _, err := Compile(
			ctx.Child(expressions[0]).WithHint(types.I32()),
		); err != nil {
			return types.Type{}, err
		}
		t := operandType.Unwrap()
		ctx.Resolver.EmitSeriesIndex(ctx.Writer, ctx.WriterID, t)
		return t, nil
	}

	if operandType.Kind != types.KindSeries {
		return types.Type{}, errors.New("slicing is only supported on series types")
	}

	var startExpr, endExpr parser.IExpressionContext
	if len(expressions) == 1 {
		if indexOrSlice.GetChild(1) == expressions[0] {
			startExpr = expressions[0]
		} else {
			endExpr = expressions[0]
		}
	} else if len(expressions) == 2 {
		startExpr = expressions[0]
		endExpr = expressions[1]
	} else {
		return types.Type{}, errors.Newf(
			"expected 1 or 2 items in slice expression, received %v",
			len(expressions),
		)
	}

	if startExpr != nil {
		if _, err := Compile(
			ctx.Child(startExpr).WithHint(types.I32()),
		); err != nil {
			return types.Type{}, err
		}
	} else {
		ctx.Writer.WriteI32Const(0)
	}

	if endExpr != nil {
		if _, err := Compile(
			ctx.Child(endExpr).WithHint(types.I32()),
		); err != nil {
			return types.Type{}, err
		}
	} else {
		ctx.Writer.WriteI32Const(-1)
	}

	ctx.Resolver.EmitSeriesSlice(ctx.Writer, ctx.WriterID)

	return operandType, nil
}

func compilePrimary(
	ctx context.Context[parser.IPrimaryExpressionContext],
) (types.Type, error) {
	if lit := ctx.AST.Literal(); lit != nil {
		return compileLiteral(ctx.Child(lit))
	}
	if qid := ctx.AST.QualifiedIdentifier(); qid != nil {
		head, tail := parser.QualifiedNameParts(qid)
		return compileIdentifier(ctx, head, tail)
	}
	if id := ctx.AST.IDENTIFIER(); id != nil {
		return compileIdentifier(ctx, id.GetText(), "")
	}
	if ctx.AST.LPAREN() != nil && ctx.AST.Expression() != nil {
		return Compile(ctx.Child(ctx.AST.Expression()))
	}
	if cast := ctx.AST.TypeCast(); cast != nil {
		return compileTypeCast(ctx.Child(cast))
	}
	return types.Type{}, errors.New("unknown primary expression")
}

func emitLiteralValue[T antlr.ParserRuleContext](
	ctx context.Context[T],
	paramType types.Type,
	defaultVal any,
) error {
	switch paramType.Kind {
	case types.KindI8,
		types.KindI16,
		types.KindI32,
		types.KindU8,
		types.KindU16,
		types.KindU32:
		var i32Val int32
		switch v := defaultVal.(type) {
		case int8:
			i32Val = int32(v)
		case int16:
			i32Val = int32(v)
		case int32:
			i32Val = v
		case uint8:
			i32Val = int32(v)
		case uint16:
			i32Val = int32(v)
		case uint32:
			i32Val = int32(v)
		default:
			return errors.Newf(
				"unexpected default value type %T for %s",
				defaultVal,
				paramType,
			)
		}
		ctx.Writer.WriteI32Const(i32Val)
	case types.KindI64:
		switch v := defaultVal.(type) {
		case int64:
			ctx.Writer.WriteI64Const(v)
		default:
			return errors.Newf("unexpected default value type %T for i64", defaultVal)
		}
	case types.KindU64:
		v, ok := defaultVal.(uint64)
		if !ok {
			return errors.Newf("unexpected default value type %T for u64", defaultVal)
		}
		ctx.Writer.WriteI64Const(int64(v))
	case types.KindF32:
		v, ok := defaultVal.(float32)
		if !ok {
			return errors.Newf("unexpected default value type %T for f32", defaultVal)
		}
		ctx.Writer.WriteF32Const(v)
	case types.KindF64:
		v, ok := defaultVal.(float64)
		if !ok {
			return errors.Newf("unexpected default value type %T for f64", defaultVal)
		}
		ctx.Writer.WriteF64Const(v)
	case types.KindBool:
		v, ok := defaultVal.(bool)
		if !ok {
			return errors.Newf("unexpected default value type %T for bool", defaultVal)
		}
		if v {
			ctx.Writer.WriteI32Const(1)
		} else {
			ctx.Writer.WriteI32Const(0)
		}
	case types.KindString:
		str, ok := defaultVal.(string)
		if !ok {
			return errors.Newf("unexpected default value type %T for str", defaultVal)
		}
		strBytes := []byte(str)
		offset := ctx.Module.AddData(strBytes)
		ctx.Writer.WriteI32Const(int32(offset))
		ctx.Writer.WriteI32Const(int32(len(strBytes)))
		ctx.Resolver.EmitStringFromLiteral(ctx.Writer, ctx.WriterID)
	default:
		return errors.Newf("unsupported default value type: %s", paramType)
	}
	return nil
}

func compileBuiltinLen[T antlr.ParserRuleContext](
	ctx context.Context[T],
	args []symbol.Argument,
	funcName string,
) (types.Type, error) {
	if len(args) != 1 {
		return types.Type{}, errors.Newf(
			"%s() requires exactly 1 argument, got %d",
			funcName,
			len(args),
		)
	}

	argType, err := Compile(ctx.Child(args[0].Expr))
	if err != nil {
		return types.Type{}, errors.Wrap(err, "argument 1 of "+funcName)
	}

	switch argType.Kind {
	case types.KindSeries:
		if funcName == "string.len" {
			return types.Type{}, errors.Newf(
				"string.len() requires a string argument, got series",
			)
		}
		ctx.Resolver.EmitSeriesLen(ctx.Writer, ctx.WriterID)
		return types.I64(), nil
	case types.KindString:
		if funcName == "series.len" {
			return types.Type{}, errors.Newf(
				"series.len() requires a series argument, got string",
			)
		}
		ctx.Resolver.EmitStringLen(ctx.Writer, ctx.WriterID)
		return types.I64(), nil
	default:
		return types.Type{}, errors.Newf(
			"argument 1 of %s: expected series or str, got %s", funcName, argType,
		)
	}
}
