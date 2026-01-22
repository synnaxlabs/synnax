// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// AnalyzeBlock validates a block of statements in a new scope.
func AnalyzeBlock(ctx context.Context[parser.IBlockContext]) {
	blockScope, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindBlock,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return
	}
	for _, stmt := range ctx.AST.AllStatement() {
		Analyze(context.Child(ctx, stmt).WithScope(blockScope))
	}
}

// Analyze validates a statement and dispatches to specialized handlers based on statement type.
func Analyze(ctx context.Context[parser.IStatementContext]) {
	switch {
	case ctx.AST.VariableDeclaration() != nil:
		analyzeVariableDeclaration(context.Child(ctx, ctx.AST.VariableDeclaration()))
	case ctx.AST.IfStatement() != nil:
		analyzeIfStatement(context.Child(ctx, ctx.AST.IfStatement()))
	case ctx.AST.ReturnStatement() != nil:
		analyzeReturnStatement(context.Child(ctx, ctx.AST.ReturnStatement()))
	case ctx.AST.Assignment() != nil:
		analyzeAssignment(context.Child(ctx, ctx.AST.Assignment()))
	case ctx.AST.Expression() != nil:
		expression.Analyze(context.Child(ctx, ctx.AST.Expression()))
	}
}

func analyzeVariableDeclaration(ctx context.Context[parser.IVariableDeclarationContext]) {
	if local := ctx.AST.LocalVariable(); local != nil {
		analyzeLocalVariable(context.Child(ctx, local))
		return
	}
	if stateful := ctx.AST.StatefulVariable(); stateful != nil {
		analyzeStatefulVariable(context.Child(ctx, stateful))
	}
}

func analyzeVariableDeclarationType[ASTNode antlr.ParserRuleContext](
	ctx context.Context[ASTNode],
	name string,
	expression parser.IExpressionContext,
	typeCtx parser.ITypeContext,
) types.Type {
	if typeCtx != nil {
		varType, err := atypes.InferFromTypeContext(typeCtx)
		if err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return types.Type{}
		}
		if expression != nil {
			exprType := atypes.InferFromExpression(context.Child(ctx, expression))
			if exprType.IsValid() && varType.IsValid() {
				// Check magnitude safety for unit conversions (warnings only)
				if varType.Unit != nil && exprType.Unit != nil {
					units.CheckAssignmentScaleSafety(ctx, exprType, varType, nil)
				}

				// If either type is a type variable, add a constraint instead of checking directly
				if exprType.Kind == types.KindVariable || varType.Kind == types.KindVariable {
					if err := atypes.Check(ctx.Constraints, varType, exprType, ctx.AST, "assignment type compatibility"); err != nil {
						ctx.Diagnostics.AddError(err, ctx.AST)
						return types.Type{}
					}
				} else {
					isLiteral := isLiteralExpression(context.Child(ctx, expression))
					if (isLiteral && !atypes.LiteralAssignmentCompatible(varType, exprType)) || (!isLiteral && !atypes.Compatible(varType, exprType)) {
						ctx.Diagnostics.AddError(
							errors.Newf("type mismatch: cannot assign %s to '%s' (type %s)", exprType, name, varType),
							ctx.AST,
						)
						return types.Type{}
					}
				}
			}
		}
		return varType
	}
	if expression != nil {
		return atypes.InferFromExpression(context.Child(ctx, expression))
	}
	ctx.Diagnostics.AddError(
		errors.Newf("no type declaration found for %s", ctx.AST), ctx.AST,
	)
	return types.Type{}
}

func isLiteralExpression(ctx context.Context[parser.IExpressionContext]) bool {
	primary := parser.GetPrimaryExpression(ctx.AST)
	return primary != nil && primary.Literal() != nil
}

func analyzeLocalVariable(ctx context.Context[parser.ILocalVariableContext]) {
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
				}
				return
			}
		}
	}

	if expr != nil {
		expression.Analyze(context.Child(ctx, expr))
	}
	varType := analyzeVariableDeclarationType(
		ctx,
		name,
		expr,
		ctx.AST.Type_(),
	)
	if !varType.IsValid() {
		_, _ = ctx.Scope.Add(ctx, symbol.Symbol{
			Name: name,
			Type: types.Type{},
			AST:  ctx.AST,
		})
		return
	}
	_, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name: name,
		Type: varType,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
	}
}

func isChannelIdentifier(ctx context.Context[parser.IExpressionContext]) bool {
	primary := parser.GetPrimaryExpression(ctx.AST)
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
	primary := parser.GetPrimaryExpression(ctx.AST)
	if primary == nil || primary.IDENTIFIER() == nil {
		return types.Type{}
	}
	sym, err := ctx.Scope.Resolve(ctx, primary.IDENTIFIER().GetText())
	if err != nil {
		return types.Type{}
	}
	return sym.Type
}

func analyzeStatefulVariable(ctx context.Context[parser.IStatefulVariableContext]) {
	name := ctx.AST.IDENTIFIER().GetText()
	expr := ctx.AST.Expression()
	varType := analyzeVariableDeclarationType(
		ctx,
		name,
		expr,
		ctx.AST.Type_(),
	)
	if !varType.IsValid() {
		_, _ = ctx.Scope.Add(ctx, symbol.Symbol{
			Name: name,
			Kind: symbol.KindStatefulVariable,
			Type: types.Type{},
			AST:  ctx.AST,
		})
		return
	}
	_, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Name: name,
		Kind: symbol.KindStatefulVariable,
		Type: varType,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return
	}
	if expr != nil {
		expression.Analyze(context.Child(ctx, expr))
	}
}

func analyzeIfStatement(ctx context.Context[parser.IIfStatementContext]) {
	if expr := ctx.AST.Expression(); expr != nil {
		expression.Analyze(context.Child(ctx, expr))
	}

	if block := ctx.AST.Block(); block != nil {
		AnalyzeBlock(context.Child(ctx, block))
	}

	for _, elseIfClause := range ctx.AST.AllElseIfClause() {
		if expr := elseIfClause.Expression(); expr != nil {
			expression.Analyze(context.Child(ctx, expr))
		}
		if block := elseIfClause.Block(); block != nil {
			AnalyzeBlock(context.Child(ctx, block))
		}
	}

	if elseClause := ctx.AST.ElseClause(); elseClause != nil {
		if block := elseClause.Block(); block != nil {
			AnalyzeBlock(context.Child(ctx, block))
		}
	}
}

func analyzeReturnStatement(ctx context.Context[parser.IReturnStatementContext]) {
	enclosingScope, err := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
	if err != nil {
		enclosingScope, err = ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
		if err != nil {
			ctx.Diagnostics.AddError(
				errors.New("return statement can only be used inside a function body"),
				ctx.AST,
			)
			return
		}
	}
	funcName := enclosingScope.Name
	var expectedReturnType types.Type
	if enclosingScope.Kind == symbol.KindFunction {
		if param, ok := enclosingScope.Type.Outputs.Get(ir.DefaultOutputParam); ok {
			expectedReturnType = param.Type
		}
	}
	returnExpr := ctx.AST.Expression()
	if returnExpr != nil {
		expression.Analyze(context.Child(ctx, returnExpr))
		actualReturnType := atypes.InferFromExpression(context.Child(ctx, returnExpr).WithTypeHint(expectedReturnType)).UnwrapChan()

		// Check for void function first - this error applies even in type inference mode
		if !expectedReturnType.IsValid() && !ctx.InTypeInferenceMode {
			ctx.Diagnostics.AddError(
				errors.New("cannot return a value from a function with no return type"),
				ctx.AST,
			)
			return
		}

		// Skip type compatibility validation in type inference mode - we're just collecting types
		if ctx.InTypeInferenceMode {
			return
		}
		if actualReturnType.IsValid() && expectedReturnType.IsValid() {
			// If either type is a type variable, add a constraint instead of checking directly
			if actualReturnType.Kind == types.KindVariable || expectedReturnType.Kind == types.KindVariable {
				if err := atypes.Check(ctx.Constraints, expectedReturnType, actualReturnType, ctx.AST, "return type compatibility"); err != nil {
					ctx.Diagnostics.AddError(err, ctx.AST)
					return
				}
			} else {
				isLiteral := isLiteralExpression(context.Child(ctx, returnExpr))
				useLiteralRules := isLiteral || (actualReturnType.IsNumeric() && expectedReturnType.IsNumeric())
				if useLiteralRules {
					if !atypes.LiteralAssignmentCompatible(expectedReturnType, actualReturnType) {
						ctx.Diagnostics.AddError(
							errors.Newf(
								"cannot return %s from '%s': expected %s",
								actualReturnType,
								funcName,
								expectedReturnType,
							),
							ctx.AST,
						)
						return
					}
				} else {
					if !atypes.Compatible(expectedReturnType, actualReturnType) {
						ctx.Diagnostics.AddError(
							errors.Newf(
								"cannot return %s from '%s': expected %s",
								actualReturnType,
								funcName,
								expectedReturnType,
							),
							ctx.AST,
						)
						return
					}
				}
			}
		}
		return
	}
	if expectedReturnType.IsValid() {
		ctx.Diagnostics.AddError(
			errors.Newf(
				"return statement in '%s' missing value of type %s",
				funcName,
				expectedReturnType,
			),
			ctx.AST,
		)
	}
}

func analyzeChannelAssignment(ctx context.Context[parser.IAssignmentContext], channelSym *symbol.Symbol) {
	// Validate we're in a function context (channel writes only allowed in imperative context)
	fn, fnErr := ctx.Scope.ClosestAncestorOfKind(symbol.KindFunction)
	if errors.Skip(fnErr, query.ErrNotFound) != nil {
		ctx.Diagnostics.AddError(fnErr, ctx.AST)
		return
	}
	if fn != nil {
		fn.Channels.Write.Add(uint32(channelSym.ID))
	}

	// Track this as a channel write in the function

	// Analyze and type-check the expression
	expr := ctx.AST.Expression()
	if expr == nil {
		return
	}
	expression.Analyze(context.Child(ctx, expr))

	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	chanValueType := channelSym.Type.Unwrap()

	if !exprType.IsValid() || !chanValueType.IsValid() {
		return
	}

	// Check magnitude safety for unit conversions (warnings only)
	if chanValueType.Unit != nil && exprType.Unit != nil {
		units.CheckAssignmentScaleSafety(ctx, exprType, chanValueType, nil)
	}

	// If either type is a type variable, add a constraint instead of checking directly
	if exprType.Kind == types.KindVariable || chanValueType.Kind == types.KindVariable {
		if err := atypes.Check(ctx.Constraints, chanValueType, exprType, ctx.AST, "channel write type compatibility"); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
			return
		}
	} else {
		isLiteral := isLiteralExpression(context.Child(ctx, expr))
		if (isLiteral && !atypes.LiteralAssignmentCompatible(chanValueType, exprType)) || (!isLiteral && !atypes.Compatible(chanValueType, exprType)) {
			channelName := ctx.AST.IDENTIFIER().GetText()
			ctx.Diagnostics.AddError(
				errors.Newf("type mismatch: cannot write %s to channel '%s' (type %s)", exprType, channelName, chanValueType),
				ctx.AST,
			)
		}
	}
}

// analyzeIndexedAssignment validates indexed assignment statements (series[i] = value)
func analyzeIndexedAssignment(
	ctx context.Context[parser.IAssignmentContext],
	varScope *symbol.Scope,
	indexOrSlice parser.IIndexOrSliceContext,
) {
	// 1. Verify base is a series type
	if varScope.Type.Kind != types.KindSeries {
		ctx.Diagnostics.AddError(
			errors.New("indexed assignment only supported on series types"),
			ctx.AST,
		)
		return
	}

	// 2. Only support single index (not slices) for now
	if indexOrSlice.COLON() != nil {
		ctx.Diagnostics.AddError(
			errors.New("slice assignment not supported"),
			ctx.AST,
		)
		return
	}

	// 3. Analyze index expression
	indexExprs := indexOrSlice.AllExpression()
	if len(indexExprs) != 1 {
		return
	}
	expression.Analyze(context.Child(ctx, indexExprs[0]))

	// 4. Analyze value expression and check type compatibility
	valueExpr := ctx.AST.Expression()
	expression.Analyze(context.Child(ctx, valueExpr))

	elemType := *varScope.Type.Elem
	exprType := atypes.InferFromExpression(context.Child(ctx, valueExpr))

	if !exprType.IsValid() || !elemType.IsValid() {
		return
	}

	// If either type is a type variable, add a constraint instead of checking directly
	if exprType.Kind == types.KindVariable || elemType.Kind == types.KindVariable {
		if err := atypes.Check(ctx.Constraints, elemType, exprType, ctx.AST, "indexed assignment type compatibility"); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
		}
		return
	}

	isLiteral := isLiteralExpression(context.Child(ctx, valueExpr))
	if (isLiteral && !atypes.LiteralAssignmentCompatible(elemType, exprType)) || (!isLiteral && !atypes.Compatible(elemType, exprType)) {
		ctx.Diagnostics.AddError(
			errors.Newf("type mismatch: cannot assign %s to series element of type %s", exprType, elemType),
			ctx.AST,
		)
	}
}

// analyzeIndexedCompoundAssignment validates indexed compound assignment statements (series[i] += value)
func analyzeIndexedCompoundAssignment(
	ctx context.Context[parser.IAssignmentContext],
	varScope *symbol.Scope,
	indexOrSlice parser.IIndexOrSliceContext,
	compoundOp parser.ICompoundOpContext,
) {
	if varScope.Type.Kind != types.KindSeries {
		ctx.Diagnostics.AddError(
			errors.New("indexed compound assignment only supported on series types"),
			ctx.AST,
		)
		return
	}

	if indexOrSlice.COLON() != nil {
		ctx.Diagnostics.AddError(
			errors.New("slice compound assignment not supported"),
			ctx.AST,
		)
		return
	}

	elemType := *varScope.Type.Elem
	if elemType.Kind == types.KindString {
		if compoundOp.PLUS_ASSIGN() == nil {
			ctx.Diagnostics.AddError(
				errors.New("string series elements only support += operator"),
				ctx.AST,
			)
			return
		}
	} else if !elemType.IsNumeric() {
		ctx.Diagnostics.AddError(
			errors.Newf("compound assignment requires numeric element type, got %s", elemType),
			ctx.AST,
		)
		return
	}

	indexExpressions := indexOrSlice.AllExpression()
	if len(indexExpressions) != 1 {
		return
	}
	expression.Analyze(context.Child(ctx, indexExpressions[0]))

	expr := ctx.AST.Expression()
	if expr == nil {
		return
	}
	expression.Analyze(context.Child(ctx, expr))

	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if !exprType.IsValid() || !elemType.IsValid() {
		return
	}

	if exprType.Kind == types.KindVariable || elemType.Kind == types.KindVariable {
		if err := atypes.Check(ctx.Constraints, elemType, exprType, ctx.AST,
			"indexed compound assignment type compatibility"); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
		}
		return
	}

	isLiteral := isLiteralExpression(context.Child(ctx, expr))
	if (isLiteral && !atypes.LiteralAssignmentCompatible(elemType, exprType)) ||
		(!isLiteral && !atypes.Compatible(elemType, exprType)) {
		ctx.Diagnostics.AddError(
			errors.Newf("type mismatch: cannot use %s in compound assignment to series element of type %s",
				exprType, elemType),
			ctx.AST,
		)
	}
}

// analyzeSeriesCompoundAssignment validates whole-series compound assignment (series += value)
// Supports both series += scalar (broadcast) and series += series (element-wise)
func analyzeSeriesCompoundAssignment(
	ctx context.Context[parser.IAssignmentContext],
	varScope *symbol.Scope,
	compoundOp parser.ICompoundOpContext,
) {
	elemType := *varScope.Type.Elem

	if elemType.Kind == types.KindString {
		if compoundOp.PLUS_ASSIGN() == nil {
			ctx.Diagnostics.AddError(
				errors.New("string series only support += operator"),
				ctx.AST,
			)
			return
		}
	} else if !elemType.IsNumeric() {
		ctx.Diagnostics.AddError(
			errors.Newf("compound assignment requires numeric element type, got %s", elemType),
			ctx.AST,
		)
		return
	}

	expr := ctx.AST.Expression()
	if expr == nil {
		return
	}
	expression.Analyze(context.Child(ctx, expr))

	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if !exprType.IsValid() || !elemType.IsValid() {
		return
	}

	if exprType.Kind == types.KindVariable || elemType.Kind == types.KindVariable {
		targetType := elemType
		if exprType.Kind == types.KindSeries {
			targetType = *exprType.Elem
		}
		if err := atypes.Check(ctx.Constraints, elemType, targetType, ctx.AST,
			"series compound assignment type compatibility"); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
		}
		return
	}

	// Type compatibility: RHS must be scalar or series with matching element type
	if exprType.Kind == types.KindSeries {
		rhsElemType := *exprType.Elem
		if !atypes.Compatible(elemType, rhsElemType) {
			ctx.Diagnostics.AddError(
				errors.Newf("type mismatch: cannot use %s in compound assignment to %s",
					exprType, varScope.Type),
				ctx.AST,
			)
		}
	} else {
		isLiteral := isLiteralExpression(context.Child(ctx, expr))
		if (isLiteral && !atypes.LiteralAssignmentCompatible(elemType, exprType)) ||
			(!isLiteral && !atypes.Compatible(elemType, exprType)) {
			ctx.Diagnostics.AddError(
				errors.Newf("type mismatch: cannot use %s in compound assignment to series of %s",
					exprType, elemType),
				ctx.AST,
			)
		}
	}
}

func analyzeCompoundAssignment(
	ctx context.Context[parser.IAssignmentContext],
	varScope *symbol.Scope,
	compoundOp parser.ICompoundOpContext,
) {
	if indexOrSlice := ctx.AST.IndexOrSlice(); indexOrSlice != nil {
		analyzeIndexedCompoundAssignment(ctx, varScope, indexOrSlice, compoundOp)
		return
	}

	varType := varScope.Type

	if varType.Kind == types.KindChan {
		ctx.Diagnostics.AddError(
			errors.New("compound assignment not supported on channels"),
			ctx.AST,
		)
		return
	}

	if varType.Kind == types.KindSeries {
		analyzeSeriesCompoundAssignment(ctx, varScope, compoundOp)
		return
	}

	if varType.Kind == types.KindString {
		if compoundOp.PLUS_ASSIGN() == nil {
			ctx.Diagnostics.AddError(
				errors.New("strings only support += operator"),
				ctx.AST,
			)
			return
		}
	} else if !varType.IsNumeric() {
		ctx.Diagnostics.AddError(
			errors.Newf("compound assignment requires numeric type, got %s", varType),
			ctx.AST,
		)
		return
	}

	expr := ctx.AST.Expression()
	if expr == nil {
		return
	}
	expression.Analyze(context.Child(ctx, expr))
	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if !exprType.IsValid() || !varType.IsValid() {
		return
	}
	if exprType.Kind == types.KindVariable || varType.Kind == types.KindVariable {
		if err := atypes.Check(ctx.Constraints, varType, exprType, ctx.AST, "compound assignment type compatibility"); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
		}
		return
	}
	if atypes.Compatible(varType, exprType) {
		return
	}
	ctx.Diagnostics.AddError(
		errors.Newf("type mismatch: cannot use %s in compound assignment to %s", exprType, varType),
		ctx.AST,
	)
}

func analyzeAssignment(ctx context.Context[parser.IAssignmentContext]) {
	name := ctx.AST.IDENTIFIER().GetText()
	varScope, err := ctx.Scope.Resolve(ctx, name)
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return
	}

	if compoundOp := ctx.AST.CompoundOp(); compoundOp != nil {
		analyzeCompoundAssignment(ctx, varScope, compoundOp)
		return
	}

	if indexOrSlice := ctx.AST.IndexOrSlice(); indexOrSlice != nil {
		analyzeIndexedAssignment(ctx, varScope, indexOrSlice)
		return
	}

	if varScope.Type.Kind == types.KindChan {
		analyzeChannelAssignment(ctx, &varScope.Symbol)
		return
	}

	expr := ctx.AST.Expression()
	if expr == nil {
		return
	}
	expression.Analyze(context.Child(ctx, expr))
	exprType := atypes.InferFromExpression(context.Child(ctx, expr))
	if !exprType.IsValid() || !varScope.Type.IsValid() {
		return
	}
	varType := varScope.Type

	// Check magnitude safety for unit conversions (warnings only)
	if varType.Unit != nil && exprType.Unit != nil {
		units.CheckAssignmentScaleSafety(ctx, exprType, varType, nil)
	}

	// Check structural compatibility (series/channel structure must match)
	if !types.StructuralMatch(varType, exprType) {
		ctx.Diagnostics.AddError(
			errors.Newf("type mismatch: cannot assign %s to '%s' (type %s)", exprType, name, varType),
			ctx.AST,
		)
		return
	}

	// If either type is a type variable, add a constraint instead of checking directly
	if exprType.Kind == types.KindVariable || varType.Kind == types.KindVariable {
		if err := atypes.Check(ctx.Constraints, varType, exprType, ctx.AST, "assignment type compatibility"); err != nil {
			ctx.Diagnostics.AddError(err, ctx.AST)
		}
		return
	}
	if atypes.AssignmentCompatible(varType, exprType) {
		return
	}
	ctx.Diagnostics.AddError(
		errors.Newf("type mismatch: cannot assign %s to '%s' (type %s)", exprType, name, varType),
		ctx.AST,
	)
}

// AnalyzeFunctionBody analyzes a block and infers its return type by examining
// all return statements across control flow paths.
// Returns the inferred return type (invalid if error occurred).
func AnalyzeFunctionBody(ctx context.Context[parser.IBlockContext]) types.Type {
	ctx.InTypeInferenceMode = true
	funcScope, err := ctx.Scope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindFunction,
		Type: types.Function(types.FunctionProperties{}),
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return types.Type{}
	}
	blockScope, err := funcScope.Add(ctx, symbol.Symbol{
		Kind: symbol.KindBlock,
		AST:  ctx.AST,
	})
	if err != nil {
		ctx.Diagnostics.AddError(err, ctx.AST)
		return types.Type{}
	}
	var collectedReturnTypes []types.Type
	for _, stmt := range ctx.AST.AllStatement() {
		Analyze(context.Child(ctx, stmt).WithScope(blockScope))
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
		return types.Type{}
	}
	return inferredType.Unwrap()
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

	// Unwrap all types first (Chan(T) -> T, Series(T) -> T) for consistent handling
	unwrappedTypes := make([]types.Type, len(returnTypes))
	for i, t := range returnTypes {
		unwrappedTypes[i] = t.Unwrap()
	}

	if len(unwrappedTypes) == 1 {
		t := unwrappedTypes[0]
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
			if t.Constraint != nil && t.Constraint.Kind == types.KindExactIntegerFloatConstant {
				return types.F64(), nil
			}
		}
		return t, nil
	}

	// Separate type variables from concrete types (now all unwrapped)
	var concreteTypes []types.Type
	var typeVariables []types.Type
	for _, t := range unwrappedTypes {
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
		if firstVar.Constraint != nil && firstVar.Constraint.Kind == types.KindExactIntegerFloatConstant {
			return types.F64(), nil
		}
		return typeVariables[0], nil
	}

	// If we have concrete types, use them to guide the unification
	// Replace type variables with types compatible with the concrete types
	resolvedTypes := make([]types.Type, 0, len(unwrappedTypes))
	for _, t := range unwrappedTypes {
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

		// Unwrap channel/series types to get the actual value type for classification
		unwrapped := t.Unwrap()

		if unwrapped.IsFloat() {
			hasFloat = true
			if unwrapped.Kind == types.KindF32 {
				if maxBits < 32 {
					maxBits = 32
				}
			} else {
				if maxBits < 64 {
					maxBits = 64
				}
			}
		} else if unwrapped.IsInteger() {
			if unwrapped.IsSignedInteger() {
				hasSigned = true
			} else if unwrapped.IsUnsignedInteger() {
				hasUnsigned = true
			}

			bits := getTypeBits(unwrapped)
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
		// Check if any concrete type is a float - integer literals can be coerced to floats
		for _, t := range concreteTypes {
			if t.IsFloat() {
				// Integer constant can be coerced to match the float type
				return t
			}
		}

		// All concrete types are integers, determine the best matching integer type
		allUnsigned := true
		for _, t := range concreteTypes {
			if !t.IsInteger() {
				// Not a numeric type we can work with, default to I32
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
		for _, t := range concreteTypes {
			if t.IsFloat() {
				return t
			}
		}
		return types.F64()
	}
	if tv.Constraint != nil && tv.Constraint.Kind == types.KindNumericConstant {
		return types.F64()
	}
	if tv.Constraint != nil && tv.Constraint.Kind == types.KindExactIntegerFloatConstant {
		for _, t := range concreteTypes {
			if t.IsNumeric() {
				return t
			}
		}
		return types.F64()
	}
	return tv
}
