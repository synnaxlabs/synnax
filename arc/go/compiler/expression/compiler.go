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
	"math"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/wasm"
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

// Compile compiles an expression and returns its type.
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
	return compileArithmetic(ctx, muls, compileMultiplicative)
}

func compileMultiplicative(
	ctx context.Context[parser.IMultiplicativeExpressionContext],
) (types.Type, error) {
	unaries := validateNonZeroArray(ctx.AST.AllUnaryExpression(), "multiplicative")
	if len(unaries) == 1 {
		return compileUnary(ctx.Child(unaries[0]))
	}
	return compileArithmetic(ctx, unaries, compileUnary)
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
	baseType.Unit = nil
	return baseType, nil
}

func compilePostfix(
	ctx context.Context[parser.IPostfixExpressionContext],
) (types.Type, error) {
	primary := ctx.AST.PrimaryExpression()
	ops := ctx.AST.GetChildren()[1:]
	var (
		currentType types.Type
		err         error
		call        parser.IFunctionCallSuffixContext
		isCall      bool
	)
	if len(ops) > 0 {
		call, isCall = ops[0].(parser.IFunctionCallSuffixContext)
	}
	if isCall && parser.PrimaryName(primary).Head != "" {
		n := parser.PrimaryName(primary)
		currentType, err = CompileCall(ctx, n.Head, n.Tail, symbol.CallArguments(call))
		ops = ops[1:]
	} else {
		currentType, err = compilePrimary(ctx.Child(primary))
	}
	if err != nil {
		return types.Type{}, err
	}

	for _, op := range ops {
		indexOrSlice, ok := op.(parser.IIndexOrSliceContext)
		if !ok {
			return types.Type{}, errors.New("only a function name can be called")
		}
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
	ordered, err := orderArguments(funcName, funcType.Inputs, args)
	if err != nil {
		return types.Type{}, err
	}

	concreteInputs, varMap, err := compileArguments(
		ctx,
		funcName,
		funcType.Inputs,
		ordered,
	)
	if err != nil {
		return types.Type{}, err
	}
	concreteOutputs := resolveOutputs(funcType.Outputs, varMap)
	concreteType := types.Function(types.FunctionProperties{
		Inputs:  concreteInputs,
		Outputs: concreteOutputs,
	})
	target := scope
	if target.Deprecated != nil {
		target = target.Deprecated
	}
	if check, ok := floatChecks[target.QualifiedName()]; ok {
		return emitFloatCheck(ctx, check, concreteInputs[0].Type)
	}
	ctx.Resolver.EmitCall(ctx.Writer, ctx.WriterID, target, concreteType)
	defaultOutput, hasDefault := concreteOutputs.Get(ir.DefaultOutputParam)
	if hasDefault {
		return defaultOutput.Type, nil
	}
	return types.Type{}, nil
}

// compileArguments emits each argument, or the default of a parameter with no
// argument. It returns the parameters with their argument types and the type each
// type variable resolved to.
func compileArguments[T antlr.ParserRuleContext](
	ctx context.Context[T],
	funcName string,
	inputs types.Params,
	ordered []parser.IExpressionContext,
) (types.Params, map[string]types.Type, error) {
	concreteInputs := make(types.Params, len(inputs))
	copy(concreteInputs, inputs)

	// Track resolved type variables as arguments are compiled so that
	// later arguments sharing the same variable get a concrete hint
	// (e.g. pow(channel_f32, 2) resolves T=f32 on the first arg,
	// then compiles the literal 2 as f32 instead of defaulting to i64).
	varMap := make(map[string]types.Type)

	for i, param := range inputs {
		arg := ordered[i]
		if arg == nil {
			if err := emitDefaultArgument(ctx, funcName, param); err != nil {
				return nil, nil, err
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
			return nil, nil, errors.Wrapf(err, "argument %d", i)
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
				return nil, nil, err
			}
		}
	}
	return concreteInputs, varMap, nil
}

// orderArguments places each argument at the index of its parameter. A parameter with
// no argument gets nil.
func orderArguments(
	funcName string,
	inputs types.Params,
	args []symbol.Argument,
) ([]parser.IExpressionContext, error) {
	ordered := make([]parser.IExpressionContext, len(inputs))
	for _, arg := range args {
		i := arg.Index
		if arg.Name != "" {
			i = inputs.GetIndex(arg.Name)
			if i < 0 {
				return nil, errors.Newf(
					"function %s has no parameter %s",
					funcName, arg.Name,
				)
			}
		}
		if i >= len(inputs) {
			return nil, errors.Newf(
				"function %s expects at most %d arguments",
				funcName, len(inputs),
			)
		}
		ordered[i] = arg.Expr
	}
	return ordered, nil
}

func emitDefaultArgument[T antlr.ParserRuleContext](
	ctx context.Context[T],
	funcName string,
	param types.Param,
) error {
	if param.Value == nil {
		return errors.Newf(
			"function %s is missing an argument for parameter %s",
			funcName, param.Name,
		)
	}
	if err := emitLiteralValue(ctx, param.Type, param.Value); err != nil {
		return errors.Wrapf(err, "default value for parameter %s", param.Name)
	}
	return nil
}

// resolveOutputs returns a copy of outputs with each type variable replaced by the
// type an argument bound it to.
func resolveOutputs(
	outputs types.Params,
	varMap map[string]types.Type,
) types.Params {
	resolved := make(types.Params, len(outputs))
	copy(resolved, outputs)
	for i, out := range resolved {
		if out.Type.Kind != types.KindVariable {
			continue
		}
		if t, ok := varMap[out.Type.Name]; ok {
			resolved[i].Type = t
		}
	}
	return resolved
}

// CompileIndex compiles a series index or slice bound to the i32 that the series host
// functions take. A value that does not fit in an i32 traps at run time.
func CompileIndex[T antlr.ParserRuleContext](
	ctx context.Context[T],
	expr parser.IExpressionContext,
) error {
	t, err := Compile(ctx.Child(expr).WithHint(types.Type{}))
	if err != nil {
		return err
	}
	if t.Kind == types.KindU32 {
		ctx.Writer.WriteOpcode(wasm.OpI64ExtendI32U)
		t = types.I64()
	}
	if wasm.ConvertType(t) == wasm.I64 {
		ctx.Resolver.EmitIndexFrom(ctx.Writer, ctx.WriterID, t)
	}
	return nil
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
		if err := CompileIndex(ctx, expressions[0]); err != nil {
			return types.Type{}, err
		}
		t := operandType.Unwrap()
		ctx.Resolver.EmitSeriesIndex(ctx.Writer, ctx.WriterID, t)
		return t, nil
	}

	if operandType.Kind != types.KindSeries {
		return types.Type{}, errors.New("slicing is only supported on series types")
	}

	startExpr, endExpr, err := sliceBounds(indexOrSlice, expressions)
	if err != nil {
		return types.Type{}, err
	}

	if startExpr != nil {
		if err := CompileIndex(ctx, startExpr); err != nil {
			return types.Type{}, err
		}
	} else {
		ctx.Writer.WriteI32Const(0)
	}

	if endExpr != nil {
		if err := CompileIndex(ctx, endExpr); err != nil {
			return types.Type{}, err
		}
	} else {
		// The host stops an end past the length at the length.
		ctx.Writer.WriteI32Const(math.MaxInt32)
	}

	ctx.Resolver.EmitSeriesSlice(ctx.Writer, ctx.WriterID)

	return operandType, nil
}

// sliceBounds returns the start and end expressions of a slice. A nil bound was left
// out of the source.
func sliceBounds(
	indexOrSlice parser.IIndexOrSliceContext,
	expressions []parser.IExpressionContext,
) (start, end parser.IExpressionContext, err error) {
	switch {
	case len(expressions) == 1:
		if indexOrSlice.GetChild(1) == expressions[0] {
			return expressions[0], nil, nil
		}
		return nil, expressions[0], nil
	case len(expressions) == 2:
		return expressions[0], expressions[1], nil
	default:
		return nil, nil, errors.Newf(
			"expected 1 or 2 items in slice expression, received %v",
			len(expressions),
		)
	}
}

func compilePrimary(
	ctx context.Context[parser.IPrimaryExpressionContext],
) (types.Type, error) {
	if lit := ctx.AST.Literal(); lit != nil {
		return compileLiteral(ctx.Child(lit))
	}
	if n := parser.PrimaryName(ctx.AST); n.Head != "" {
		return compileIdentifier(ctx, n)
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
		v, ok := int32Value(defaultVal)
		if !ok {
			return errors.Newf(
				"unexpected default value type %T for %s",
				defaultVal,
				paramType,
			)
		}
		ctx.Writer.WriteI32Const(v)
	case types.KindI64:
		v, ok := defaultVal.(int64)
		if !ok {
			return errors.Newf("unexpected default value type %T for i64", defaultVal)
		}
		ctx.Writer.WriteI64Const(v)
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
		return emitBoolValue(ctx, defaultVal)
	case types.KindString:
		return emitStringValue(ctx, defaultVal)
	case types.KindInvalid, types.KindChan, types.KindSeries, types.KindVariable,
		types.KindNumericConstant, types.KindIntegerConstant, types.KindFloatConstant,
		types.KindExactIntegerFloatConstant, types.KindFunction, types.KindSequence,
		types.KindStage, types.KindVarRef:
		fallthrough
	default:
		return errors.Newf("unsupported default value type: %s", paramType)
	}
	return nil
}

func emitBoolValue[T antlr.ParserRuleContext](
	ctx context.Context[T],
	defaultVal any,
) error {
	v, ok := defaultVal.(bool)
	if !ok {
		return errors.Newf("unexpected default value type %T for bool", defaultVal)
	}
	if v {
		ctx.Writer.WriteI32Const(1)
	} else {
		ctx.Writer.WriteI32Const(0)
	}
	return nil
}

func emitStringValue[T antlr.ParserRuleContext](
	ctx context.Context[T],
	defaultVal any,
) error {
	str, ok := defaultVal.(string)
	if !ok {
		return errors.Newf("unexpected default value type %T for str", defaultVal)
	}
	strBytes := []byte(str)
	offset := ctx.Module.AddData(strBytes)
	ctx.Writer.WriteI32Const(int32(offset))
	ctx.Writer.WriteI32Const(int32(len(strBytes)))
	ctx.Resolver.EmitStringFromLiteral(ctx.Writer, ctx.WriterID)
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
	case types.KindInvalid, types.KindU8, types.KindU16, types.KindU32, types.KindU64,
		types.KindI8, types.KindI16, types.KindI32, types.KindI64, types.KindF32,
		types.KindF64, types.KindChan, types.KindVariable, types.KindNumericConstant,
		types.KindIntegerConstant, types.KindFloatConstant,
		types.KindExactIntegerFloatConstant, types.KindFunction, types.KindSequence,
		types.KindStage, types.KindVarRef, types.KindBool:
	}
	return types.Type{}, errors.Newf(
		"argument 1 of %s: expected series or str, got %s", funcName, argType,
	)
}
