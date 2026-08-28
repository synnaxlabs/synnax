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
	"github.com/antlr4-go/antlr/v4"
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

func isBool(t basetypes.Type) bool    { return t.IsBool() }
func isNumeric(t basetypes.Type) bool { return t.IsNumeric() }

// resolveConstraint replaces an unresolved constant's type variable with its
// constraint so operand checks can classify it. Concrete types pass through.
func resolveConstraint(t basetypes.Type) basetypes.Type {
	if t.Kind == basetypes.KindVariable && t.Constraint != nil {
		return *t.Constraint
	}
	return t
}

func isNumericOrString(
	t basetypes.Type,
) bool {
	return t.IsNumeric() || t.Kind == basetypes.KindString
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
func isAny(basetypes.Type) bool { return true }

// getSignedIntegerLiteral extracts a signed integer value from a node.
// Supports both plain integer literals (2) and negated ones (-2).
// Returns (value, true) if successful, (0, false) otherwise.
func getSignedIntegerLiteral(node antlr.ParserRuleContext) (int, bool) {
	if node == nil {
		return 0, false
	}
	var (
		sign    = 1
		current = node
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
	return sign * int(intVal), true
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
	check func(t basetypes.Type) bool,
) {
	if len(items) <= 1 {
		return
	}
	firstType := infer(ctx.Child(items[0])).Unwrap()

	// If first operand is Invalid, skip validation - we can't check types we don't know
	if firstType.Kind == basetypes.KindInvalid {
		return
	}

	resolvedFirst := resolveConstraint(firstType)
	if resolvedFirst.Kind != basetypes.KindVariable && !check(resolvedFirst) {
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
		func(t basetypes.Type) bool { return t.IsBool() },
	)
}

func analyzeLogicalAnd(ctx context.Context[parser.ILogicalAndExpressionContext]) {
	equalities := ctx.AST.AllEqualityExpression()
	for _, equality := range equalities {
		analyzeEquality(ctx.Child(equality))
	}
	validateType(ctx, equalities, "and", types.InferEquality, isBool)
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
		isAny,
	)
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
		isNumeric,
	)
}

func analyzeAdditive(ctx context.Context[parser.IAdditiveExpressionContext]) {
	mults := ctx.AST.AllMultiplicativeExpression()
	for _, multiplicative := range mults {
		analyzeMultiplicative(ctx.Child(multiplicative))
	}
	// Determine the operator - strings are only allowed for + (concatenation)
	op := getAdditiveOperator(ctx.AST)
	var check func(basetypes.Type) bool
	if op == "+" {
		check = isNumericOrString
	} else {
		check = isNumeric
	}
	validateType[parser.IMultiplicativeExpressionContext](
		ctx,
		mults,
		op,
		types.InferMultiplicative,
		check,
	)
}

func analyzeMultiplicative(
	ctx context.Context[parser.IMultiplicativeExpressionContext],
) {
	unaries := ctx.AST.AllUnaryExpression()
	for _, unary := range unaries {
		analyzeUnary(ctx.Child(unary))
	}
	validateType[parser.IUnaryExpressionContext](
		ctx,
		unaries,
		getMultiplicativeOperator(ctx.AST),
		types.InferFromUnaryExpression,
		isNumeric,
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
			if !operandType.IsNumeric() {
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
			if !operandType.IsBool() {
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

func analyzePostfix(ctx context.Context[parser.IPostfixExpressionContext]) {
	if primary := ctx.AST.PrimaryExpression(); primary != nil {
		analyzePrimary(ctx.Child(primary))
	}
	for _, indexOrSlice := range ctx.AST.AllIndexOrSlice() {
		for _, expr := range indexOrSlice.AllExpression() {
			Analyze(ctx.Child(expr))
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
	primary := ctx.AST.PrimaryExpression()
	head, tail := parser.PrimaryNameParts(primary)
	funcName := parser.PrimaryName(primary)
	if funcName != "" {
		scope, err := ctx.ResolveQualified(head, tail)
		if err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, primary))
			return
		}
		if scope.Kind == symbol.KindFunction {
			callerFn, fnErr := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
			if fnErr != nil && !errors.Is(fnErr, query.ErrNotFound) {
				ctx.Diagnostics.Add(diagnostics.Error(fnErr, ctx.AST))
				return
			}
			if callerFn != nil && scope.Exec == symbol.ExecFlow {
				ctx.Diagnostics.Add(diagnostics.Errorf(
					ctx.AST,
					"function '%s' cannot be called inside a func block. Use it as a flow statement instead: %s{}",
					funcName,
					funcName,
				))
				return
			}
			if funcName != "len" && funcName != "series.len" {
				if hasMultipleNamedOutputs(scope.Type) {
					ctx.Diagnostics.Add(diagnostics.Errorf(
						funcCalls[0],
						"cannot call function %s: functions with multiple named outputs are not callable",
						funcName,
					))
				} else {
					AnalyzeCall(
						ctx,
						funcName,
						scope.Type,
						inputArguments(funcCalls[0]),
						scope.AnalyzeArguments,
						funcCalls[0],
						"",
					)
				}
			}
			if callerFn != nil {
				argChannels := buildArgChannels(ctx, scope, funcCalls[0])
				propagateChannelsWithArgMap(callerFn, scope, argChannels)
				*ctx.CallEdges = append(*ctx.CallEdges, context.CallEdge{
					Caller:      callerFn,
					Callee:      scope,
					CallSite:    ctx.AST,
					ArgChannels: argChannels,
				})
			}
		} else {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				funcCalls[0],
				"cannot call non-function %s of type %s",
				funcName,
				scope.Type,
			))
		}
	}
}

// hasMultipleNamedOutputs reports whether t has multiple or non-default named outputs.
func hasMultipleNamedOutputs(t basetypes.Type) bool {
	_, hasDefault := t.Outputs.Get(ir.DefaultOutputParam)
	return len(t.Outputs) > 1 || (len(t.Outputs) == 1 && !hasDefault)
}

// inputArguments adapts the arguments in a call's parens `(...)` form into the
// unified []symbol.Argument shape.
func inputArguments(funcCall parser.IFunctionCallSuffixContext) []symbol.Argument {
	argList := funcCall.ArgumentList()
	if argList == nil {
		return nil
	}
	exprs := argList.AllExpression()
	args := make([]symbol.Argument, len(exprs))
	for i, expr := range exprs {
		args[i] = symbol.Argument{Index: i, Expr: expr, AST: expr}
	}
	return args
}

func analyzePrimary(ctx context.Context[parser.IPrimaryExpressionContext]) {
	if qid := ctx.AST.QualifiedIdentifier(); qid != nil {
		head, tail := parser.QualifiedNameParts(qid)
		if _, err := ctx.ResolveQualified(head, tail); err != nil {
			ctx.Diagnostics.Add(diagnostics.Error(err, ctx.AST))
		}
		return
	}
	if id := ctx.AST.IDENTIFIER(); id != nil {
		resolved, err := ctx.Resolve(id.GetText())
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
				targetType, _ := types.InferFromTypeContext(typeCtx)
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
func buildArgChannels(
	ctx context.Context[parser.IPostfixExpressionContext],
	callee *symbol.Symbol,
	funcCall parser.IFunctionCallSuffixContext,
) map[int]context.ChannelMapping {
	argList := funcCall.ArgumentList()
	if argList == nil {
		return nil
	}
	args := argList.AllExpression()
	var argChannels map[int]context.ChannelMapping
	for i, param := range callee.Type.Inputs {
		if param.Type.Kind != basetypes.KindChan || i >= len(args) {
			continue
		}
		channelID, channelName, ok := resolveChannelArg(ctx, args[i])
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
func resolveChannelArg(
	ctx context.Context[parser.IPostfixExpressionContext],
	expr parser.IExpressionContext,
) (uint32, string, bool) {
	primary := parser.GetPrimaryExpression(expr)
	if primary == nil || primary.IDENTIFIER() == nil {
		return 0, "", false
	}
	sym, err := ctx.Scope.Resolve(ctx, primary.IDENTIFIER().GetText())
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
