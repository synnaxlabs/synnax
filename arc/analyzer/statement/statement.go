// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package statement implements semantic analysis for Arc statements including variable
// declarations, assignments, conditionals, and channel operations.
package statement

import (
	stdcontext "context"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
)

// AnalyzeBlock validates a block of statements in a new scope.
func AnalyzeBlock(ctx context.Context[parser.IBlockContext]) bool {
	blockScope, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindBlock,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	for _, stmt := range ctx.AST.AllStatement() {
		if !Analyze(context.Child(ctx, stmt).WithScope(blockScope)) {
			return false
		}
	}
	return true
}

// Analyze validates a statement and dispatches to specialized handlers based on statement type.
func Analyze(ctx context.Context[parser.IStatementContext]) bool {
	switch {
	case ctx.AST.VariableDeclaration() != nil:
		return analyzeVariableDeclaration(context.Child(ctx, ctx.AST.VariableDeclaration()))
	case ctx.AST.IfStatement() != nil:
		return analyzeIfStatement(context.Child(ctx, ctx.AST.IfStatement()))
	case ctx.AST.ReturnStatement() != nil:
		return analyzeReturnStatement(context.Child(ctx, ctx.AST.ReturnStatement()))
	case ctx.AST.ChannelOperation() != nil:
		return analyzeChannelOperation(context.Child(ctx, ctx.AST.ChannelOperation()))
	case ctx.AST.Assignment() != nil:
		return analyzeAssignment(context.Child(ctx, ctx.AST.Assignment()))
	case ctx.AST.Expression() != nil:
		return expression.Analyze(context.Child(ctx, ctx.AST.Expression()))
	}
	return true
}

func analyzeVariableDeclaration(ctx context.Context[parser.IVariableDeclarationContext]) bool {
	if local := ctx.AST.LocalVariable(); local != nil {
		return analyzeLocalVariable(context.Child(ctx, local))
	}
	if stateful := ctx.AST.StatefulVariable(); stateful != nil {
		return analyzeStatefulVariable(context.Child(ctx, stateful))
	}
	return true
}

func analyzeVariableDeclarationType[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	expression parser.IExpressionContext,
	typeCtx parser.ITypeContext,
) (types.Type, bool) {
	if typeCtx != nil {
		varType, err := atypes.InferFromTypeContext(typeCtx)
		if err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return types.Type{}, false
		}
		if expression != nil {
			exprType := atypes.InferFromExpression(context.Child(ctx, expression))
			if exprType.IsValid() && varType.IsValid() {
				// If either type is a type variable, add a constraint instead of checking directly
				if exprType.Kind == types.KindVariable || varType.Kind == types.KindVariable {
					if err := atypes.Check(ctx.Constraints, varType, exprType, ctx.AST, "assignment type compatibility"); err != nil {
						ctx.Diagnostics.AddError(err, ctx.AST)
						return types.Type{}, false
					}
				} else {
					isLiteral := isLiteralExpression(context.Child(ctx, expression))
					if (isLiteral && !atypes.LiteralAssignmentCompatible(varType, exprType)) || (!isLiteral && !atypes.Compatible(varType, exprType)) {
						ctx.Diagnostics.AddError(
							errors.Newf("type mismatch: cannot assign %s to %s", exprType, varType),
							ctx.AST,
						)
						return types.Type{}, false
					}
				}
			}
		}
		return varType, true
	}
	if expression != nil {
		return atypes.InferFromExpression(context.Child(ctx, expression)), true
	}
	ctx.Diagnostics.AddError(
		errors.Newf("no type declaration found for %s", ctx.AST), ctx.AST,
	)
	return types.Type{}, false
}

func getPrimaryExpression(expr parser.IExpressionContext) parser.IPrimaryExpressionContext {
	if expr == nil {
		return nil
	}
	logicalOr := expr.LogicalOrExpression()
	if logicalOr == nil || len(logicalOr.AllLogicalAndExpression()) != 1 {
		return nil
	}
	ands := logicalOr.AllLogicalAndExpression()[0]
	if len(ands.AllEqualityExpression()) != 1 {
		return nil
	}
	eq := ands.AllEqualityExpression()[0]
	if len(eq.AllRelationalExpression()) != 1 {
		return nil
	}
	rel := eq.AllRelationalExpression()[0]
	if len(rel.AllAdditiveExpression()) != 1 {
		return nil
	}
	add := rel.AllAdditiveExpression()[0]
	if len(add.AllMultiplicativeExpression()) != 1 {
		return nil
	}
	mult := add.AllMultiplicativeExpression()[0]
	if len(mult.AllPowerExpression()) != 1 {
		return nil
	}
	pow := mult.AllPowerExpression()[0]
	unary := pow.UnaryExpression()
	if unary == nil {
		return nil
	}
	postfix := unary.PostfixExpression()
	if postfix == nil {
		return nil
	}
	return postfix.PrimaryExpression()
}

func isLiteralExpression(ctx context.Context[parser.IExpressionContext]) bool {
	primary := getPrimaryExpression(ctx.AST)
	return primary != nil && primary.Literal() != nil
}

func analyzeLocalVariable(ctx context.Context[parser.ILocalVariableContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	expr := ctx.AST.Expression()

	if expr != nil && ctx.AST.Type_() == nil {
		childCtx := context.Child(ctx, expr)
		if isChannelIdentifier(childCtx) {
			chanType := getChannelType(childCtx)
			if chanType.IsValid() {
				_, err := childCtx.Scope.Add(ctx, symbol.Symbol{
					Name: name,
					Kind: symbol.KindChannel,
					Type: chanType.Unwrap(),
					AST:  ctx.AST,
				})
				if err != nil {
					ctx.Diagnostics.AddError(err, ctx.AST)
					return false
				}
				return true
			}
		}
	}

	if expr != nil {
		if !expression.Analyze(context.Child(ctx, expr)) {
			return false
		}
	}
	varType, ok := analyzeVariableDeclarationType(
		ctx,
		expr,
		ctx.AST.Type_(),
	)
	if !ok {
		return false
	}
	_, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name: name,
		Type: varType,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	return true
}

func isChannelIdentifier(ctx context.Context[parser.IExpressionContext]) bool {
	primary := getPrimaryExpression(ctx.AST)
	if primary == nil || primary.IDENTIFIER() == nil {
		return false
	}
	sym, err := ctx.Scope.Resolve(ctx, primary.IDENTIFIER().GetText())
	if err != nil {
		return false
	}
	return sym.Type.Kind == types.KindChan
}

func getChannelType(ctx context.Context[parser.IExpressionContext]) types.Type {
	primary := getPrimaryExpression(ctx.AST)
	if primary == nil || primary.IDENTIFIER() == nil {
		return types.Type{}
	}
	sym, err := ctx.Scope.Resolve(ctx, primary.IDENTIFIER().GetText())
	if err != nil {
		return types.Type{}
	}
	return sym.Type
}

func analyzeStatefulVariable(ctx context.Context[parser.IStatefulVariableContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	expr := ctx.AST.Expression()
	varType, ok := analyzeVariableDeclarationType(
		ctx,
		expr,
		ctx.AST.Type_(),
	)
	if !ok {
		return false
	}
	_, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name: name,
		Kind: symbol.KindStatefulVariable,
		Type: varType,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	if expr != nil {
		return expression.Analyze(context.Child(ctx, expr))
	}
	return true
}

func analyzeIfStatement(ctx context.Context[parser.IIfStatementContext]) bool {
	if expr := ctx.AST.Expression(); expr != nil {
		if !expression.Analyze(context.Child(ctx, expr)) {
			return false
		}
	}

	if block := ctx.AST.Block(); block != nil {
		if !AnalyzeBlock(context.Child(ctx, block)) {
			return false
		}
	}

	for _, elseIfClause := range ctx.AST.AllElseIfClause() {
		if expr := elseIfClause.Expression(); expr != nil {
			if !expression.Analyze(context.Child(ctx, expr)) {
				return false
			}
		}
		if block := elseIfClause.Block(); block != nil {
			if !AnalyzeBlock(context.Child(ctx, block)) {
				return false
			}
		}
	}

	if elseClause := ctx.AST.ElseClause(); elseClause != nil {
		if block := elseClause.Block(); block != nil {
			if !AnalyzeBlock(context.Child(ctx, block)) {
				return false
			}
		}
	}
	return true
}

func analyzeReturnStatement(ctx context.Context[parser.IReturnStatementContext]) bool {
	enclosingScope, err := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
	if err != nil {
		enclosingScope, err = ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
		if err != nil {
			ctx.Diagnostics.AddError(
				errors.New("return statement not in function or fn"),
				ctx.AST,
			)
			return false
		}
	}
	var expectedReturnType types.Type
	if enclosingScope.Kind == symbol.KindFunction {
		expectedReturnType, _ = enclosingScope.Type.Outputs.Get(ir.DefaultOutputParam)
	}
	returnExpr := ctx.AST.Expression()
	if returnExpr != nil {
		if !expression.Analyze(context.Child(ctx, returnExpr)) {
			return false
		}
		actualReturnType := atypes.InferFromExpression(context.Child(ctx, returnExpr).WithTypeHint(expectedReturnType))

		// Check for void function first - this error applies even in type inference mode
		if !expectedReturnType.IsValid() && !ctx.InTypeInferenceMode {
			ctx.Diagnostics.AddError(
				errors.New("unexpected return value in function/func with void return type"),
				ctx.AST,
			)
			return false
		}

		// Skip type compatibility validation in type inference mode - we're just collecting types
		if ctx.InTypeInferenceMode {
			return true
		}
		if actualReturnType.IsValid() && expectedReturnType.IsValid() {
			// If either type is a type variable, add a constraint instead of checking directly
			if actualReturnType.Kind == types.KindVariable || expectedReturnType.Kind == types.KindVariable {
				if err := atypes.Check(ctx.Constraints, expectedReturnType, actualReturnType, ctx.AST, "return type compatibility"); err != nil {
					ctx.Diagnostics.AddError(err, ctx.AST)
					return false
				}
			} else {
				isLiteral := isLiteralExpression(context.Child(ctx, returnExpr))
				useLiteralRules := isLiteral || (actualReturnType.IsNumeric() && expectedReturnType.IsNumeric())
				if useLiteralRules {
					if !atypes.LiteralAssignmentCompatible(expectedReturnType, actualReturnType) {
						ctx.Diagnostics.AddError(
							errors.Newf(
								"cannot return %s, expected %s",
								actualReturnType,
								expectedReturnType,
							),
							ctx.AST,
						)
						return false
					}
				} else {
					if !atypes.Compatible(expectedReturnType, actualReturnType) {
						ctx.Diagnostics.AddError(
							errors.Newf(
								"cannot return %s, expected %s",
								actualReturnType,
								expectedReturnType,
							),
							ctx.AST,
						)
						return false
					}
				}
			}
		}
		return true
	}
	if expectedReturnType.IsValid() {
		ctx.Diagnostics.AddError(
			errors.Newf(
				"return statement missing value of type %s",
				expectedReturnType,
			),
			ctx.AST,
		)
		return false
	}
	return true
}

func analyzeChannelOperation(ctx context.Context[parser.IChannelOperationContext]) bool {
	if write := ctx.AST.ChannelWrite(); write != nil {
		return analyzeChannelWrite(context.Child(ctx, write))
	}
	if read := ctx.AST.ChannelRead(); read != nil {
		return analyzeChannelRead(context.Child(ctx, read))
	}
	return true
}

func analyzeChannelWrite(ctx context.Context[parser.IChannelWriteContext]) bool {
	var channelName string
	if ctx.AST.IDENTIFIER() != nil {
		channelName = ctx.AST.IDENTIFIER().GetText()
	} else {
		return false
	}

	fn, fnErr := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
	var savedHook func(stdcontext.Context, *symbol.Scope) error
	if fnErr == nil && fn != nil {
		savedHook = fn.OnResolve
		fn.OnResolve = nil
	}

	channelSym, err := ctx.Scope.Resolve(ctx, channelName)

	if fnErr == nil && fn != nil {
		fn.OnResolve = savedHook
	}

	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}

	if channelSym.Type.Kind != types.KindChan {
		ctx.Diagnostics.AddError(
			errors.Newf("%s is not a channel", channelName),
			ctx.AST,
		)
		return false
	}

	if fnErr == nil && fn != nil {
		fn.Channels.Write.Add(uint32(channelSym.ID))
	}

	expr := ctx.AST.Expression()
	if expr == nil {
		return true
	}

	if !expression.Analyze(context.Child(ctx, expr)) {
		return false
	}

	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if exprType.IsValid() {
		chanValueType := channelSym.Type.Unwrap()
		// If either type is a type variable, add a constraint instead of checking directly
		if exprType.Kind == types.KindVariable || chanValueType.Kind == types.KindVariable {
			if err := atypes.Check(ctx.Constraints, chanValueType, exprType, ctx.AST, "channel write type compatibility"); err != nil {
				ctx.Diagnostics.AddError(err, ctx.AST)
				return false
			}
		} else if !atypes.Compatible(chanValueType, exprType) {
			ctx.Diagnostics.AddError(
				errors.Newf(
					"type mismatch: cannot write %s to channel of type %s",
					exprType,
					chanValueType,
				),
				ctx.AST,
			)
			return false
		}
	}

	return true
}

func analyzeChannelRead(ctx context.Context[parser.IChannelReadContext]) bool {
	if blocking := ctx.AST.BlockingRead(); blocking != nil {
		return analyzeBlockingRead(context.Child(ctx, blocking))
	}
	if nonBlocking := ctx.AST.NonBlockingRead(); nonBlocking != nil {
		return analyzeNonBlockingRead(context.Child(ctx, nonBlocking))
	}
	return true
}

func analyzeBlockingRead(ctx context.Context[parser.IBlockingReadContext]) bool {
	ids := ctx.AST.AllIDENTIFIER()
	if len(ids) != 2 {
		return false
	}
	return createChannelReadVariable(ctx, ids[0].GetText(), ids[1].GetText())
}

func analyzeNonBlockingRead(ctx context.Context[parser.INonBlockingReadContext]) bool {
	ids := ctx.AST.AllIDENTIFIER()
	if len(ids) != 2 {
		return false
	}
	return createChannelReadVariable(ctx, ids[0].GetText(), ids[1].GetText())
}

func createChannelReadVariable[T antlr.ParserRuleContext](
	ctx context.Context[T],
	varName, channelName string,
) bool {
	channelSym, err := ctx.Scope.Resolve(ctx, channelName)
	if err != nil {
		ctx.Diagnostics.AddError(errors.Wrapf(err, "undefined channel: %s", channelName), ctx.AST)
		return false
	}
	if channelSym.Kind != symbol.KindChannel && channelSym.Kind != symbol.KindConfig && channelSym.Kind != symbol.KindInput && channelSym.Kind != symbol.KindOutput {
		ctx.Diagnostics.AddError(errors.Newf("%s is not a channel", channelName), ctx.AST)
		return false
	}
	if channelSym.Type.Kind != types.KindChan {
		ctx.Diagnostics.AddError(errors.Newf("%s is not a channel", channelName), ctx.AST)
		return false
	}
	_, err = ctx.Scope.Add(ctx, symbol.Symbol{
		Name: varName,
		Kind: symbol.KindVariable,
		Type: channelSym.Type,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	return true
}

func analyzeAssignment(ctx context.Context[parser.IAssignmentContext]) bool {
	name := ctx.AST.IDENTIFIER().GetText()
	varScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return false
	}
	expr := ctx.AST.Expression()
	if expr == nil {
		return true
	}
	if !expression.Analyze(context.Child(ctx, expr)) {
		return false
	}
	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if !exprType.IsValid() || !varScope.Type.IsValid() {
		return true
	}
	varType := varScope.Type
	// If either type is a type variable, add a constraint instead of checking directly
	if exprType.Kind == types.KindVariable || varType.Kind == types.KindVariable {
		if err := atypes.Check(ctx.Constraints, varType, exprType, ctx.AST, "assignment type compatibility"); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return false
		}
		return true
	}
	if atypes.Compatible(varType, exprType) {
		return true
	}
	ctx.Diagnostics.AddError(
		errors.Newf("type mismatch: cannot assign %s to variable of type %s", exprType, varType),
		ctx.AST,
	)
	return false
}

// AnalyzeFunctionBody analyzes a block and infers its return type by examining
// all return statements across control flow paths.
// Returns (ok bool, inferredReturnType types.Type)
func AnalyzeFunctionBody(ctx context.Context[parser.IBlockContext]) (types.Type, bool) {
	ctx.InTypeInferenceMode = true
	funcScope, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{
			Inputs:  &types.Params{},
			Outputs: &types.Params{},
			Config:  &types.Params{},
		}),
		AST: ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return types.Type{}, false
	}
	blockScope, err := funcScope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindBlock,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return types.Type{}, false
	}
	var collectedReturnTypes []types.Type
	for _, stmt := range ctx.AST.AllStatement() {
		if !Analyze(context.Child(ctx, stmt).WithScope(blockScope)) {
			return types.Type{}, false
		}
		returnTypes := collectStatementReturnTypes(
			context.Child(ctx, stmt).WithScope(blockScope),
		)
		for _, rt := range returnTypes {
			if rt.IsValid() {
				collectedReturnTypes = append(collectedReturnTypes, rt)
			}
		}
	}
	inferredType, err := unifyReturnTypes(collectedReturnTypes)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return types.Type{}, false
	}
	return inferredType.Unwrap(), true
}

// collectStatementReturnTypes extracts all return types from a statement.
// Returns a slice of types (empty if no returns).
func collectStatementReturnTypes(
	ctx context.Context[parser.IStatementContext],
) []types.Type {
	switch {
	case ctx.AST.ReturnStatement() != nil:
		returnStmt := ctx.AST.ReturnStatement()
		returnExpr := returnStmt.Expression()
		if returnExpr == nil {
			// Return statement with no expression (void return)
			return []types.Type{{}}
		}
		returnType := atypes.InferFromExpression(context.Child(ctx, returnExpr))
		if returnType.IsValid() {
			return []types.Type{returnType}
		}
		return []types.Type{}

	case ctx.AST.IfStatement() != nil:
		_, returnTypes := getIfStatementReturnTypes(
			context.Child(ctx, ctx.AST.IfStatement()),
		)
		return returnTypes

	default:
		return []types.Type{}
	}
}

// getIfStatementReturnTypes recursively extracts return types from if/else branches.
// Returns (allPathsReturn bool, returnTypes []types.Type)
func getIfStatementReturnTypes(
	ctx context.Context[parser.IIfStatementContext],
) (bool, []types.Type) {
	var returnTypes []types.Type
	allPathsReturn := true

	// Check main if block
	if block := ctx.AST.Block(); block != nil {
		hasReturn, blockTypes := getBlockReturnTypes(
			context.Child(ctx, block),
		)
		if hasReturn {
			returnTypes = append(returnTypes, blockTypes...)
		} else {
			allPathsReturn = false
		}
	}

	// Check else-if clauses
	for _, elseIfClause := range ctx.AST.AllElseIfClause() {
		if block := elseIfClause.Block(); block != nil {
			hasReturn, blockTypes := getBlockReturnTypes(
				context.Child(ctx, block),
			)
			if hasReturn {
				returnTypes = append(returnTypes, blockTypes...)
			} else {
				allPathsReturn = false
			}
		}
	}

	// Check else clause
	if elseClause := ctx.AST.ElseClause(); elseClause != nil {
		if block := elseClause.Block(); block != nil {
			hasReturn, blockTypes := getBlockReturnTypes(
				context.Child(ctx, block),
			)
			if hasReturn {
				returnTypes = append(returnTypes, blockTypes...)
			} else {
				allPathsReturn = false
			}
		}
	} else {
		// No else clause means not all paths return
		allPathsReturn = false
	}

	return allPathsReturn, returnTypes
}

// getBlockReturnTypes extracts all return types from a block's statements.
// Returns (hasReturn bool, returnTypes []types.Type)
func getBlockReturnTypes(
	ctx context.Context[parser.IBlockContext],
) (bool, []types.Type) {
	var returnTypes []types.Type
	for _, stmt := range ctx.AST.AllStatement() {
		stmtTypes := collectStatementReturnTypes(context.Child(ctx, stmt))
		for _, rt := range stmtTypes {
			if rt.IsValid() {
				returnTypes = append(returnTypes, rt)
			}
		}
	}
	return len(returnTypes) > 0, returnTypes
}

// unifyReturnTypes unifies multiple return types to find the smallest reasonable common type.
func unifyReturnTypes(
	returnTypes []types.Type,
) (types.Type, error) {
	if len(returnTypes) == 0 {
		return types.Type{}, nil
	}
	if len(returnTypes) == 1 {
		t := returnTypes[0]
		// If it's a type variable (literal), resolve it to a concrete default type
		if t.Kind == types.KindVariable {
			if t.Constraint != nil && t.Constraint.Kind == types.KindIntegerConstant {
				return types.I64(), nil
			}
			if t.Constraint != nil && t.Constraint.Kind == types.KindFloatConstant {
				return types.F64(), nil
			}
			if t.Constraint != nil && t.Constraint.Kind == types.KindNumericConstant {
				return types.F64(), nil
			}
		}
		return t, nil
	}

	// Separate type variables from concrete types
	var concreteTypes []types.Type
	var typeVariables []types.Type
	for _, t := range returnTypes {
		if t.Kind == types.KindVariable {
			typeVariables = append(typeVariables, t)
		} else {
			concreteTypes = append(concreteTypes, t)
		}
	}

	// If all are type variables (all literals), unify them to a concrete default type
	if len(concreteTypes) == 0 {
		// All literals should unify to a default concrete type
		// For integers, default to i64; for floats, default to f64
		firstVar := typeVariables[0]
		if firstVar.Constraint != nil && firstVar.Constraint.Kind == types.KindIntegerConstant {
			return types.I64(), nil
		}
		if firstVar.Constraint != nil && firstVar.Constraint.Kind == types.KindFloatConstant {
			return types.F64(), nil
		}
		if firstVar.Constraint != nil && firstVar.Constraint.Kind == types.KindNumericConstant {
			return types.F64(), nil
		}
		return typeVariables[0], nil
	}

	// If we have concrete types, use them to guide the unification
	// Replace type variables with types compatible with the concrete types
	resolvedTypes := make([]types.Type, 0, len(returnTypes))
	for _, t := range returnTypes {
		if t.Kind == types.KindVariable {
			// Infer appropriate type based on concrete types present
			resolved := resolveTypeVariableWithContext(t, concreteTypes)
			resolvedTypes = append(resolvedTypes, resolved)
		} else {
			resolvedTypes = append(resolvedTypes, t)
		}
	}

	firstType := resolvedTypes[0]
	allEqual := true
	for _, t := range resolvedTypes[1:] {
		if !types.Equal(firstType, t) {
			allEqual = false
			break
		}
	}
	if allEqual {
		return firstType, nil
	}

	allNumeric := true
	hasFloat := false
	hasSigned := false
	hasUnsigned := false
	maxBits := 0

	for _, t := range resolvedTypes {
		if !t.IsNumeric() {
			allNumeric = false
			break
		}

		if t.IsFloat() {
			hasFloat = true
			if t.Kind == types.KindF32 {
				if maxBits < 32 {
					maxBits = 32
				}
			} else {
				if maxBits < 64 {
					maxBits = 64
				}
			}
		} else if t.IsInteger() {
			if t.IsSignedInteger() {
				hasSigned = true
			} else if t.IsUnsignedInteger() {
				hasUnsigned = true
			}

			bits := getTypeBits(t)
			if bits > maxBits {
				maxBits = bits
			}
		}
	}

	if !allNumeric {
		return types.Type{}, errors.Newf(
			"incompatible return types: cannot unify %s and %s",
			returnTypes[0],
			returnTypes[1],
		)
	}

	if hasFloat {
		hasInteger := false
		for _, t := range returnTypes {
			if t.IsInteger() {
				hasInteger = true
				break
			}
		}
		if hasInteger {
			return types.Type{}, errors.New(
				"mixed integer and floating-point returns are not allowed",
			)
		}
		if maxBits > 32 {
			return types.F64(), nil
		}
		return types.F32(), nil
	}

	if hasSigned && hasUnsigned {
		if maxBits <= 8 {
			return types.I16(), nil
		} else if maxBits <= 16 {
			return types.I32(), nil
		} else if maxBits <= 32 {
			return types.I64(), nil
		}
		return types.I64(), nil
	}

	if hasSigned {
		if maxBits <= 8 {
			return types.I8(), nil
		} else if maxBits <= 16 {
			return types.I16(), nil
		} else if maxBits <= 32 {
			return types.I32(), nil
		}
		return types.I64(), nil
	}
	if maxBits <= 8 {
		return types.U8(), nil
	} else if maxBits <= 16 {
		return types.U16(), nil
	} else if maxBits <= 32 {
		return types.U32(), nil
	}
	return types.U64(), nil
}

func getTypeBits(t types.Type) int {
	switch t.Kind {
	case types.KindI8, types.KindU8:
		return 8
	case types.KindI16, types.KindU16:
		return 16
	case types.KindI32, types.KindU32:
		return 32
	case types.KindI64, types.KindU64:
		return 64
	case types.KindF32:
		return 32
	case types.KindF64:
		return 64
	default:
		return 0
	}
}

func resolveTypeVariableWithContext(tv types.Type, concreteTypes []types.Type) types.Type {
	if tv.Kind != types.KindVariable {
		return tv
	}
	if tv.Constraint != nil && tv.Constraint.Kind == types.KindIntegerConstant {
		allUnsigned := true
		for _, t := range concreteTypes {
			if !t.IsInteger() {
				return types.I32()
			}
			if t.IsSignedInteger() {
				allUnsigned = false
			}
		}
		if allUnsigned {
			maxBits := 0
			for _, t := range concreteTypes {
				bits := getTypeBits(t)
				if bits > maxBits {
					maxBits = bits
				}
			}
			if maxBits <= 8 {
				return types.U8()
			} else if maxBits <= 16 {
				return types.U16()
			} else if maxBits <= 32 {
				return types.U32()
			}
			return types.U64()
		}
		return types.I32()
	}
	if tv.Constraint != nil && tv.Constraint.Kind == types.KindFloatConstant {
		return types.F64()
	}
	if tv.Constraint != nil && tv.Constraint.Kind == types.KindNumericConstant {
		return types.F64()
	}
	return tv
}
