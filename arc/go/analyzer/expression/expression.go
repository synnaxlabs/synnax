// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package expression implements type checking and semantic analysis for Arc
// expressions.
package expression

import (
	"math"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	basetypes "github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// operand is what an operator requires of its operands.
type operand struct {
	// accepts reports whether a type with a known kind fits.
	accepts func(t basetypes.Type) bool
	// bind constrains an unconstrained type variable, such as the element of an empty
	// series literal, to fit, and reports whether it can. Nil accepts any variable.
	bind func(
		cs *constraints.System,
		t basetypes.Type,
		source antlr.ParserRuleContext,
	) bool
}

var (
	anyOperand     = operand{accepts: func(basetypes.Type) bool { return true }}
	boolOperand    = operand{accepts: basetypes.Type.IsBool, bind: types.Bool}
	numericOperand = operand{accepts: basetypes.Type.IsNumeric, bind: types.Numeric}
	// A + of two untyped operands is numeric, which empty literals default to.
	numericOrStringOperand = operand{
		accepts: func(t basetypes.Type) bool {
			return t.IsNumeric() || t.Kind == basetypes.KindString
		},
		bind: types.Numeric,
	}
)

// fits reports whether t fits o. It binds an unconstrained type variable.
func (o operand) fits(
	cs *constraints.System,
	t basetypes.Type,
	source antlr.ParserRuleContext,
) bool {
	if t.Kind == basetypes.KindVariable && t.Constraint == nil {
		return o.bind == nil || o.bind(cs, t, source)
	}
	return o.accepts(resolveConstraint(t))
}

// resolveConstraint replaces an unresolved constant's type variable with its
// constraint so operand checks can classify it. Concrete types pass through.
func resolveConstraint(t basetypes.Type) basetypes.Type {
	if t.Kind == basetypes.KindVariable && t.Constraint != nil {
		return *t.Constraint
	}
	return t
}

// tracksChannelRead reports whether reading resolved reads from a channel: a
// channel symbol, a chan-typed param, or a channel alias. Value vars are not.
func tracksChannelRead(resolved *symbol.Symbol) bool {
	switch {
	case resolved.Kind == symbol.KindChannel:
		return true
	case resolved.Type.Kind == basetypes.KindChan &&
		resolved.Kind == symbol.KindInput:
		return true
	case resolved.Type.Kind == basetypes.KindChan && resolved.SourceID != nil:
		return true
	default:
		return false
	}
}

// getSignedIntegerLiteral extracts a signed integer value from a node.
// Supports both plain integer literals (2) and negated ones (-2).
// Returns (value, true) if successful, (0, false) otherwise.
func getSignedIntegerLiteral(node antlr.ParserRuleContext) (int64, bool) {
	if node == nil {
		return 0, false
	}
	var (
		sign    int64 = 1
		current       = node
	)
	if unary, ok := current.(parser.IUnaryExpressionContext); ok {
		if unary.MINUS() != nil {
			sign = -1
			current = unary.UnaryExpression()
		} else if unary.NOT() != nil {
			// NOT doesn't make sense for integer exponent
			return 0, false
		}
	}
	if power, ok := current.(parser.IPowerExpressionContext); ok {
		if power.CARET() != nil {
			return 0, false
		}
		current = power.PostfixExpression()
	}
	lit := parser.GetLiteralNode(current)
	if lit == nil {
		return 0, false
	}
	numLit := lit.NumericLiteral()
	if numLit == nil {
		return 0, false
	}
	intLit := numLit.INTEGER_LITERAL()
	if intLit == nil {
		return 0, false
	}
	if numLit.IDENTIFIER() != nil {
		return 0, false
	}
	parsed, err := literal.ParseNumeric(numLit, basetypes.I64())
	if err != nil {
		return 0, false
	}
	if parsed.Type.Unit != nil {
		return 0, false
	}
	intVal, ok := parsed.Value.(int64)
	if !ok {
		return 0, false
	}
	return sign * intVal, true
}

// Analyze validates type correctness of an expression and accumulates constraints.
func Analyze(ctx context.Context[parser.IExpressionContext]) {
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
		analyzeLogicalOr(ctx.Child(logicalOr))
	}
}

func getEqualityOperator(ctx antlr.ParserRuleContext) string {
	if eqCtx, ok := ctx.(parser.IEqualityExpressionContext); ok {
		if len(eqCtx.AllEQ()) > 0 {
			return "=="
		}
		if len(eqCtx.AllNEQ()) > 0 {
			return "!="
		}
	}
	return "equality"
}

func getAdditiveOperator(ctx antlr.ParserRuleContext) string {
	if addCtx, ok := ctx.(parser.IAdditiveExpressionContext); ok {
		if len(addCtx.AllPLUS()) > 0 {
			return "+"
		}
		if len(addCtx.AllMINUS()) > 0 {
			return "-"
		}
	}
	return "additive"
}

func getMultiplicativeOperator(ctx antlr.ParserRuleContext) string {
	if mulCtx, ok := ctx.(parser.IMultiplicativeExpressionContext); ok {
		if len(mulCtx.AllSTAR()) > 0 {
			return "*"
		}
		if len(mulCtx.AllSLASH()) > 0 {
			return "/"
		}
		if len(mulCtx.AllPERCENT()) > 0 {
			return "%"
		}
	}
	return "multiplicative"
}

func getRelationalOperator(ctx antlr.ParserRuleContext) string {
	if relCtx, ok := ctx.(parser.IRelationalExpressionContext); ok {
		if len(relCtx.AllLT()) > 0 {
			return "<"
		}
		if len(relCtx.AllGT()) > 0 {
			return ">"
		}
		if len(relCtx.AllLEQ()) > 0 {
			return "<="
		}
		if len(relCtx.AllGEQ()) > 0 {
			return ">="
		}
	}
	return "comparison"
}

func validateType[T, N antlr.ParserRuleContext](
	ctx context.Context[N],
	items []T,
	opName string,
	infer func(ctx context.Context[T]) basetypes.Type,
	want operand,
) {
	if len(items) <= 1 {
		return
	}
	firstType := infer(ctx.Child(items[0])).Unwrap()

	// If first operand is Invalid, skip validation - we can't check types we don't know
	if firstType.Kind == basetypes.KindInvalid {
		return
	}

	if !want.fits(ctx.Constraints, firstType, items[0]) {
		ctx.Diagnostics.Add(
			diagnostics.Errorf(
				ctx.AST,
				"cannot use %s in %s operation",
				firstType,
				opName,
			),
		)
		return
	}

	for i := 1; i < len(items); i++ {
		nextType := infer(ctx.Child(items[i]).WithTypeHint(firstType)).Unwrap()

		// Skip if this operand is Invalid - we can't check types we don't know
		if nextType.Kind == basetypes.KindInvalid {
			continue
		}

		// Check dimensional compatibility first if either operand has units This must
		// be checked even for type variables since the unit is known at parse time
		// Note: Power operations (^) are handled separately in analyzePower via
		// ValidatePowerOp.
		if firstType.Unit != nil || nextType.Unit != nil {
			if !units.ValidateBinaryOp(ctx, opName, firstType, nextType) {
				return
			}
		}

		if firstType.Kind == basetypes.KindVariable ||
			nextType.Kind == basetypes.KindVariable {
			if !want.fits(ctx.Constraints, nextType, items[i]) {
				ctx.Diagnostics.Add(diagnostics.Errorf(
					ctx.AST,
					"cannot use %s in %s operation",
					nextType,
					opName,
				))
				return
			}
			if err := ctx.Constraints.AddCompatible(
				firstType,
				nextType,
				items[i],
				opName+" operands must be compatible",
			); err != nil {
				ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
				return
			}
			// When the first operand is a literal (type variable) and we encounter
			// a concrete type, adopt it so subsequent operands are checked against
			// the concrete type rather than the permissive constraint. Without this,
			// `1000.0 - f32_ch + f64_ch` would pass because both f32 and f64
			// individually satisfy FloatConstraint, even though f32 != f64.
			if firstType.Kind == basetypes.KindVariable &&
				nextType.Kind != basetypes.KindVariable {
				firstType = nextType
			}
		} else {
			// Unit compatibility is already validated above by units.ValidateBinaryOp
			if !types.Compatible(firstType, nextType) {
				ctx.Diagnostics.Add(
					diagnostics.Errorf(
						ctx.AST,
						"type mismatch: cannot use %s and %s in %s operation",
						firstType,
						nextType,
						opName,
					),
				)
				return
			}
		}
	}
}

func analyzeLogicalOr(ctx context.Context[parser.ILogicalOrExpressionContext]) {
	logicalAnds := ctx.AST.AllLogicalAndExpression()
	for _, logicalAnd := range logicalAnds {
		analyzeLogicalAnd(ctx.Child(logicalAnd))
	}
	validateType(
		ctx,
		logicalAnds,
		"or",
		types.InferLogicalAnd,
		boolOperand,
	)
}

func analyzeLogicalAnd(ctx context.Context[parser.ILogicalAndExpressionContext]) {
	equalities := ctx.AST.AllEqualityExpression()
	for _, equality := range equalities {
		analyzeEquality(ctx.Child(equality))
	}
	validateType(ctx, equalities, "and", types.InferEquality, boolOperand)
}

func analyzeEquality(ctx context.Context[parser.IEqualityExpressionContext]) {
	relExpressions := ctx.AST.AllRelationalExpression()
	for _, relational := range relExpressions {
		analyzeRelational(ctx.Child(relational))
	}
	validateType(
		ctx,
		relExpressions,
		getEqualityOperator(ctx.AST),
		types.InferRelational,
		anyOperand,
	)
	checkNaNComparison(ctx, relExpressions, getEqualityOperator(ctx.AST))
}

// checkNaNComparison rejects a comparison with operator op that has the NaN constant as
// one of the operands the compiler runs, the first two of a chain. Such a comparison
// always has the same result. The error names the constant as the code wrote it, and
// the module's is_nan when the module has one.
func checkNaNComparison[T antlr.ParserRuleContext, O antlr.Tree](
	ctx context.Context[T],
	operands []O,
	op string,
) {
	if len(operands) < 2 {
		return
	}
	for _, operand := range operands[:2] {
		if value, ok := ctx.FloatConstant(operand); !ok || !math.IsNaN(value) {
			continue
		}
		primary, _ := parser.SoleOperand(operand)
		qid := primary.QualifiedIdentifier()
		head, _ := parser.QualifiedNameParts(qid)
		check := "is_nan from the math module"
		if module, err := ctx.Scope.Resolve(ctx, head); err == nil {
			if _, err := module.Resolve(ctx, "is_nan"); err == nil {
				check = head + ".is_nan"
			}
		}
		if op == "!=" {
			check = "not " + check
		}
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"%s with %s is always %t, use %s instead",
			op,
			qid.GetText(),
			op == "!=",
			check,
		).WithCode(codes.NaNComparison))
		return
	}
}

func analyzeRelational(ctx context.Context[parser.IRelationalExpressionContext]) {
	additives := ctx.AST.AllAdditiveExpression()
	for _, additive := range additives {
		analyzeAdditive(ctx.Child(additive))
	}
	validateType(
		ctx,
		additives,
		getRelationalOperator(ctx.AST),
		types.InferAdditive,
		numericOperand,
	)
	checkNaNComparison(ctx, additives, getRelationalOperator(ctx.AST))
}

func analyzeAdditive(ctx context.Context[parser.IAdditiveExpressionContext]) {
	mults := ctx.AST.AllMultiplicativeExpression()
	for _, multiplicative := range mults {
		analyzeMultiplicative(ctx.Child(multiplicative))
	}
	// Determine the operator - strings are only allowed for + (concatenation)
	op := getAdditiveOperator(ctx.AST)
	want := numericOperand
	if op == "+" {
		want = numericOrStringOperand
	}
	validateType(ctx, mults, op, types.InferMultiplicative, want)
}

func analyzeMultiplicative(
	ctx context.Context[parser.IMultiplicativeExpressionContext],
) {
	unaries := ctx.AST.AllUnaryExpression()
	for _, unary := range unaries {
		analyzeUnary(ctx.Child(unary))
	}
	validateType(
		ctx,
		unaries,
		getMultiplicativeOperator(ctx.AST),
		types.InferFromUnaryExpression,
		numericOperand,
	)
}

func analyzePower(ctx context.Context[parser.IPowerExpressionContext]) {
	if postfix := ctx.AST.PostfixExpression(); postfix != nil {
		analyzePostfix(ctx.Child(postfix))
	}
	exponent := ctx.AST.UnaryExpression()
	if exponent != nil {
		analyzeUnary(ctx.Child(exponent))
	}

	if ctx.AST.CARET() != nil && exponent != nil {
		baseType := types.InferPostfix(ctx.Child(ctx.AST.PostfixExpression())).
			Unwrap()
		expType := types.InferFromUnaryExpression(ctx.Child(exponent)).Unwrap()
		for _, t := range []basetypes.Type{baseType, expType} {
			if t.IsValid() && !t.IsNumeric() {
				ctx.Diagnostics.Add(diagnostics.Errorf(
					ctx.AST,
					"cannot use %s in ^ operation",
					t,
				))
				return
			}
		}

		if baseType.Unit != nil || expType.Unit != nil {
			_, isLiteral := getSignedIntegerLiteral(exponent)
			if err := units.ValidatePowerOp(baseType, expType, isLiteral); err != nil {
				ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
				return
			}
		}
	}
}

func analyzeUnary(ctx context.Context[parser.IUnaryExpressionContext]) {
	if innerUnary := ctx.AST.UnaryExpression(); innerUnary != nil {
		childCtx := ctx.Child(innerUnary)
		analyzeUnary(childCtx)
		operandType := types.InferFromUnaryExpression(childCtx)
		if ctx.AST.MINUS() != nil {
			if !types.Numeric(ctx.Constraints, operandType, ctx.AST) {
				ctx.Diagnostics.Add(
					diagnostics.Errorf(
						ctx.AST,
						"operator - not supported for type %s",
						operandType,
					),
				)
				return
			}
		} else if ctx.AST.NOT() != nil {
			if !types.Bool(ctx.Constraints, operandType, ctx.AST) {
				ctx.Diagnostics.Add(
					diagnostics.Errorf(
						ctx.AST,
						"operator 'not' requires boolean operand, received %s",
						operandType,
					),
				)
				return
			}
		}
		return
	}
	if power := ctx.AST.PowerExpression(); power != nil {
		analyzePower(ctx.Child(power))
	}
}

// AnalyzeIndex analyzes a series index or slice bound and reports an error if its type
// is not an integer. A literal index takes the type i32.
func AnalyzeIndex[T antlr.ParserRuleContext](
	ctx context.Context[T],
	expr parser.IExpressionContext,
) {
	exprCtx := ctx.Child(expr)
	Analyze(exprCtx)
	t := types.InferFromExpression(exprCtx).UnwrapChan()
	if !t.IsValid() {
		return
	}
	if !resolveConstraint(t).IsInteger() {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			expr,
			"series index must be an integer type, got %s",
			t,
		))
		return
	}
	if t.Kind != basetypes.KindVariable || !parser.IsLiteral(expr) {
		return
	}
	// The series host functions take an i32, so a literal index needs no conversion.
	if err := ctx.Constraints.AddEquality(
		t,
		basetypes.I32(),
		expr,
		"series index",
	); err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, expr))
	}
}

func analyzePostfix(ctx context.Context[parser.IPostfixExpressionContext]) {
	if primary := ctx.AST.PrimaryExpression(); primary != nil {
		analyzePrimary(ctx.Child(primary))
	}
	for _, indexOrSlice := range ctx.AST.AllIndexOrSlice() {
		for _, expr := range indexOrSlice.AllExpression() {
			AnalyzeIndex(ctx, expr)
		}
	}

	funcCalls := ctx.AST.AllFunctionCallSuffix()

	for _, funcCall := range funcCalls {
		if argList := funcCall.ArgumentList(); argList != nil {
			for _, expr := range argList.AllExpression() {
				Analyze(ctx.Child(expr))
			}
		}
	}

	if len(funcCalls) == 0 {
		return
	}
	if call, callee := nonFunctionCall(ctx.AST); call != nil {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			call,
			"cannot call non-function %s",
			callee,
		))
		return
	}
	primary := ctx.AST.PrimaryExpression()
	head, tail := parser.PrimaryNameParts(primary)
	funcName := parser.PrimaryName(primary)
	scope, err := ctx.ResolveQualified(head, tail)
	if err != nil {
		ctx.Diagnostics.Add(diagnostics.Error(err, primary))
		return
	}
	if scope.Kind != symbol.KindFunction {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			funcCalls[0],
			"cannot call non-function %s of type %s",
			funcName,
			scope.Type,
		))
		return
	}
	AnalyzeCallSite(
		ctx,
		scope,
		funcName,
		basetypes.Freshen(scope.Type, ctx.ScopedKey("call", funcCalls[0])),
		symbol.CallArguments(funcCalls[0]),
		funcCalls[0],
	)
}

// nonFunctionCall returns the first call in postfix whose callee is not a name, and the
// callee's source text. Only a name can refer to a function. It returns nil when every
// call is valid.
func nonFunctionCall(
	postfix parser.IPostfixExpressionContext,
) (parser.IFunctionCallSuffixContext, string) {
	named := parser.PrimaryName(postfix.PrimaryExpression()) != ""
	var callee strings.Builder
	for i, child := range postfix.GetChildren() {
		rule, ok := child.(antlr.ParserRuleContext)
		if !ok {
			continue
		}
		if call, isCall := rule.(parser.IFunctionCallSuffixContext); isCall &&
			(i != 1 || !named) {
			return call, callee.String()
		}
		callee.WriteString(rule.GetText())
	}
	return nil, ""
}

// AnalyzeCallSite checks a call to callee and records the call edge. fnType may be a
// freshened copy of callee.Type.
func AnalyzeCallSite[T antlr.ParserRuleContext](
	ctx context.Context[T],
	callee *symbol.Symbol,
	funcName string,
	fnType basetypes.Type,
	args []symbol.Argument,
	site antlr.ParserRuleContext,
) {
	callerFn, err := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
	if err != nil && !errors.Is(err, query.ErrNotFound) {
		ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		return
	}
	if callerFn != nil && callee.Exec == symbol.ExecFlow {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			ctx.AST,
			"function '%s' cannot be called inside a func block. Use it as a flow statement instead: %s()",
			funcName,
			funcName,
		))
		return
	}
	if funcName == "len" || funcName == "series.len" {
		// len dispatches on its argument type at compile time, so only the names
		// and arity are checked here.
		bindArguments(
			ctx,
			funcName,
			fnType,
			args,
			ir.FormatFunctionSignature(funcName, fnType),
			"",
		)
	} else if hasMultipleNamedOutputs(fnType) {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			site,
			"cannot call function %s: functions with multiple named outputs are not callable",
			funcName,
		))
	} else {
		AnalyzeCall(ctx, funcName, fnType, args, callee.AnalyzeArguments, site, "")
	}
	if callerFn == nil {
		return
	}
	argChannels := buildArgChannels(ctx, callee, args)
	propagateChannelsWithArgMap(callerFn, callee, argChannels)
	*ctx.CallEdges = append(*ctx.CallEdges, context.CallEdge{
		Caller:      callerFn,
		Callee:      callee,
		CallSite:    ctx.AST,
		ArgChannels: argChannels,
	})
}

// hasMultipleNamedOutputs reports whether t has multiple or non-default named outputs.
func hasMultipleNamedOutputs(t basetypes.Type) bool {
	_, hasDefault := t.Outputs.Get(ir.DefaultOutputParam)
	return len(t.Outputs) > 1 || (len(t.Outputs) == 1 && !hasDefault)
}

func analyzePrimary(ctx context.Context[parser.IPrimaryExpressionContext]) {
	if head, tail := parser.PrimaryNameParts(ctx.AST); head != "" {
		resolved, err := ctx.ResolveQualified(head, tail)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
			return
		}
		if tracksChannelRead(resolved) {
			fn, fnErr := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
			if fnErr != nil && !errors.Is(fnErr, query.ErrNotFound) {
				ctx.Diagnostics.Add(diagnostics.Error(fnErr, ctx.AST))
				return
			}
			if fn != nil {
				// Use SourceID if available, otherwise use symbol's own ID
				readID := uint32(resolved.ID)
				if resolved.SourceID != nil {
					readID = uint32(*resolved.SourceID)
				}
				fn.Channels.Read[readID] = resolved.Name
			}
		}
		return
	}
	if lit := ctx.AST.Literal(); lit != nil {
		if strTerm := parser.StringTerminal(lit); strTerm != nil {
			AnalyzeFmtStrLiteral(ctx, strTerm)
		}
		if series := lit.SeriesLiteral(); series != nil &&
			series.ExpressionList() != nil {
			for _, element := range series.ExpressionList().AllExpression() {
				Analyze(ctx.Child(element))
			}
		}
		return
	}
	if expr := ctx.AST.Expression(); expr != nil {
		Analyze(ctx.Child(expr))
		return
	}
	if typeCast := ctx.AST.TypeCast(); typeCast != nil {
		if expr := typeCast.Expression(); expr != nil {
			Analyze(ctx.Child(expr))
			// Validate that the cast is allowed
			sourceType := types.InferFromExpression(ctx.Child(expr)).Unwrap()
			if typeCtx := typeCast.Type_(); typeCtx != nil {
				// bool casting is not allowed. User should explicitly define their
				// numeric comparison. E.g. `x != 0` or `y == 1.0001`
				if prim := typeCtx.PrimitiveType(); prim != nil && prim.BOOL() != nil {
					ctx.Diagnostics.Add(diagnostics.Errorf(ctx.AST,
						"boolean conversions are not supported; define the "+
							"comparison explicitly, e.g. x != 0 or abs(x - 1) < 0.01"))
					return
				}
				targetType, err := types.InferFromTypeContext(typeCtx)
				if err != nil {
					ctx.Diagnostics.Add(diagnostics.Error(err, typeCtx))
					return
				}
				if !isValidCast(sourceType, targetType) {
					ctx.Diagnostics.Add(
						diagnostics.Errorf(
							ctx.AST,
							"cannot cast %s to %s",
							sourceType,
							targetType,
						),
					)
					return
				}
				if _, ok := ctx.FloatConstant(expr); ok && targetType.IsInteger() {
					ctx.Diagnostics.Add(diagnostics.Errorf(
						ctx.AST,
						"cannot cast %s to %s: NaN and infinity have no integer value",
						expr.GetText(),
						targetType,
					))
					return
				}
				if sourceType.Kind == basetypes.KindBool &&
					targetType.IsNumeric() &&
					targetType.Kind != basetypes.KindU8 {
					ctx.Diagnostics.Add(diagnostics.Warningf(ctx.AST,
						"bool to %s is experimental; a future release may only "+
							"allow bool to u8", targetType))
				}
			}
		}
	}
}

// buildArgChannels extracts channel argument mappings for chan-typed parameters at a
// function call site. Maps are keyed by parameter index (position in the callee's
// input list) so the mapping is valid even when the callee hasn't been fully analyzed
// yet (forward references). The param index is resolved to a symbol ID during
// propagation, when all scopes are guaranteed to exist.
func buildArgChannels[T antlr.ParserRuleContext](
	ctx context.Context[T],
	callee *symbol.Symbol,
	args []symbol.Argument,
) map[int]context.ChannelMapping {
	var argChannels map[int]context.ChannelMapping
	for _, arg := range args {
		i := arg.Index
		if arg.Name != "" {
			i = callee.Type.Inputs.GetIndex(arg.Name)
		}
		if i < 0 || i >= len(callee.Type.Inputs) ||
			callee.Type.Inputs[i].Type.Kind != basetypes.KindChan {
			continue
		}
		channelID, channelName, ok := resolveChannelArg(ctx, arg.Expr)
		if !ok {
			continue
		}
		if argChannels == nil {
			argChannels = make(map[int]context.ChannelMapping)
		}
		argChannels[i] = context.ChannelMapping{
			ChannelID:   channelID,
			ChannelName: channelName,
		}
	}
	return argChannels
}

// resolveChannelArg extracts the channel ID and name from an argument expression.
func resolveChannelArg[T antlr.ParserRuleContext](
	ctx context.Context[T],
	expr parser.IExpressionContext,
) (uint32, string, bool) {
	head, tail := parser.ExpressionNameParts(expr)
	if head == "" {
		return 0, "", false
	}
	sym, err := ctx.Scope.ResolveMember(ctx, head, tail)
	if err != nil || sym.Type.Kind != basetypes.KindChan {
		return 0, "", false
	}
	sourceID := uint32(sym.ID)
	if sym.SourceID != nil {
		sourceID = uint32(*sym.SourceID)
	}
	return sourceID, sym.Name, true
}

// propagateChannelsWithArgMap copies callee channel accesses to the caller, remapping
// any accesses through chan-typed parameters to the actual channel IDs from the call
// site arguments. This is used during inline propagation at the call site.
// If the callee hasn't been fully analyzed yet (forward reference), inputScopes will
// be empty and no remapping occurs. The post-pass handles that case.
func propagateChannelsWithArgMap(
	caller, callee *symbol.Symbol,
	argChannels map[int]context.ChannelMapping,
) {
	paramMap := ResolveArgChannels(callee, argChannels)
	for id, name := range callee.Channels.Read {
		if mapping, ok := paramMap[int(id)]; ok {
			caller.Channels.Read[mapping.ChannelID] = mapping.ChannelName
		} else {
			caller.Channels.Read[id] = name
		}
	}
	for id, name := range callee.Channels.Write {
		if mapping, ok := paramMap[int(id)]; ok {
			caller.Channels.Write[mapping.ChannelID] = mapping.ChannelName
		} else {
			caller.Channels.Write[id] = name
		}
	}
}

func ResolveArgChannels(
	callee *symbol.Symbol,
	argChannels map[int]context.ChannelMapping,
) map[int]context.ChannelMapping {
	if len(argChannels) == 0 {
		return nil
	}
	inputScopes := callee.FilterChildrenByKind(symbol.KindInput)
	resolved := make(map[int]context.ChannelMapping, len(argChannels))
	for paramIdx, mapping := range argChannels {
		if paramIdx < len(inputScopes) {
			resolved[inputScopes[paramIdx].ID] = mapping
		}
	}
	return resolved
}

func isValidCast(source, target basetypes.Type) bool {
	// Constraint unification will handle type variables
	if source.Kind == basetypes.KindVariable || target.Kind == basetypes.KindVariable {
		return true
	}
	if source.Kind == target.Kind {
		return true
	}
	if target.Kind == basetypes.KindString &&
		(source.IsNumeric() || source.Kind == basetypes.KindBool) {
		return true
	}
	if source.Kind == basetypes.KindString || target.Kind == basetypes.KindString {
		return false
	}
	return (source.IsNumeric() || source.Kind == basetypes.KindBool) &&
		target.IsNumeric()
}
