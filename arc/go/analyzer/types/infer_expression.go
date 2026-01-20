// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"fmt"

	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
)

// InferFromExpression determines the type of an Arc expression through recursive descent.
func InferFromExpression(ctx context.Context[parser.IExpressionContext]) types.Type {
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
		return InferLogicalOr(context.Child(ctx, logicalOr))
	}
	return types.Type{}
}

func InferLogicalOr(ctx context.Context[parser.ILogicalOrExpressionContext]) types.Type {
	ands := ctx.AST.AllLogicalAndExpression()
	if len(ands) > 1 {
		return types.U8()
	}
	if len(ands) == 1 {
		return InferLogicalAnd(context.Child(ctx, ands[0]))
	}
	return types.Type{}
}

func InferLogicalAnd(ctx context.Context[parser.ILogicalAndExpressionContext]) types.Type {
	equalities := ctx.AST.AllEqualityExpression()
	if len(equalities) > 1 {
		return types.U8()
	}
	if len(equalities) == 1 {
		return InferEquality(context.Child(ctx, equalities[0]))
	}
	return types.Type{}
}

func InferEquality(ctx context.Context[parser.IEqualityExpressionContext]) types.Type {
	relExpressions := ctx.AST.AllRelationalExpression()
	if len(relExpressions) > 1 {
		return types.U8()
	}
	if len(relExpressions) == 1 {
		return InferRelational(context.Child(ctx, relExpressions[0]))
	}
	return types.Type{}
}

func InferRelational(ctx context.Context[parser.IRelationalExpressionContext]) types.Type {
	additives := ctx.AST.AllAdditiveExpression()
	if len(additives) > 1 {
		return types.U8()
	}
	if len(additives) == 1 {
		return InferAdditive(context.Child(ctx, additives[0]))
	}
	return types.Type{}
}

func InferAdditive(ctx context.Context[parser.IAdditiveExpressionContext]) types.Type {
	multiplicatives := ctx.AST.AllMultiplicativeExpression()
	if len(multiplicatives) == 0 {
		return types.Type{}
	}
	if len(multiplicatives) > 1 {
		firstType := InferMultiplicative(context.Child(ctx, multiplicatives[0]))
		// Track if any operand is a series - if so, result is a series
		isSeries := firstType.Kind == types.KindSeries
		// Use series element type when available, as it's more concrete than scalar type variables
		elemType := firstType.Unwrap()

		for i := 1; i < len(multiplicatives); i++ {
			nextType := InferMultiplicative(context.Child(ctx, multiplicatives[i]))
			if nextType.Kind == types.KindSeries {
				isSeries = true
				// Prefer series element type over scalar type variable
				nextElem := nextType.Unwrap()
				if nextElem.Kind != types.KindVariable {
					elemType = nextElem
				}
				if !Compatible(elemType, nextElem) {
					if isSeries {
						return types.Series(elemType)
					}
					return elemType
				}
			} else if !Compatible(elemType, nextType.Unwrap()) {
				if isSeries {
					return types.Series(elemType)
				}
				return elemType
			}
		}
		if isSeries {
			return types.Series(elemType)
		}
		return elemType
	}
	return InferMultiplicative(context.Child(ctx, multiplicatives[0]))
}

func InferMultiplicative(ctx context.Context[parser.IMultiplicativeExpressionContext]) types.Type {
	powers := ctx.AST.AllPowerExpression()
	if len(powers) == 0 {
		return types.Type{}
	}
	if len(powers) > 1 {
		firstType := InferPower(context.Child(ctx, powers[0]))
		// Track if any operand is a series - if so, result is a series
		isSeries := firstType.Kind == types.KindSeries
		// Use series element type when available, as it's more concrete than scalar type variables
		elemType := firstType.Unwrap()

		for i := 1; i < len(powers); i++ {
			nextType := InferPower(context.Child(ctx, powers[i]))
			if nextType.Kind == types.KindSeries {
				isSeries = true
				// Prefer series element type over scalar type variable
				nextElem := nextType.Unwrap()
				if nextElem.Kind != types.KindVariable {
					elemType = nextElem
				}
				if !Compatible(elemType, nextElem) {
					resultType := lo.Ternary(isSeries, types.Series(elemType), elemType)
					ctx.TypeMap[ctx.AST] = resultType
					return resultType
				}
			} else if !Compatible(elemType, nextType.Unwrap()) {
				resultType := lo.Ternary(isSeries, types.Series(elemType), elemType)
				ctx.TypeMap[ctx.AST] = resultType
				return resultType
			}
		}
		resultType := lo.Ternary(isSeries, types.Series(elemType), elemType)
		ctx.TypeMap[ctx.AST] = resultType
		return resultType
	}
	resultType := InferPower(context.Child(ctx, powers[0]))
	ctx.TypeMap[ctx.AST] = resultType
	return resultType
}

func InferPower(ctx context.Context[parser.IPowerExpressionContext]) types.Type {
	if unary := ctx.AST.UnaryExpression(); unary != nil {
		baseType := InferFromUnaryExpression(context.Child(ctx, unary))

		// If no caret operator, return base type
		if ctx.AST.CARET() == nil || ctx.AST.PowerExpression() == nil {
			return baseType
		}

		// Recursively infer exponent type (right-associative)
		_ = InferPower(context.Child(ctx, ctx.AST.PowerExpression()))

		// Power operation returns the unwrapped base type
		// (e.g., chan f32 ^ i32 = f32, f64 ^ f64 = f64)
		return baseType.Unwrap()
	}
	return types.Type{}
}

func InferFromUnaryExpression(ctx context.Context[parser.IUnaryExpressionContext]) types.Type {
	if ctx.AST.UnaryExpression() != nil {
		// Unary operator (- or not) - unwrap channels in the operand
		return InferFromUnaryExpression(context.Child(ctx, ctx.AST.UnaryExpression())).Unwrap()
	}
	if postfix := ctx.AST.PostfixExpression(); postfix != nil {
		return inferPostfixType(context.Child(ctx, postfix))
	}
	return types.Type{}
}

func inferPostfixType(ctx context.Context[parser.IPostfixExpressionContext]) types.Type {
	if primary := ctx.AST.PrimaryExpression(); primary != nil {
		primaryType := inferPrimaryType(context.Child(ctx, primary))

		// Handle function call suffixes - return the function's return type
		funcCalls := ctx.AST.AllFunctionCallSuffix()
		if len(funcCalls) > 0 && primaryType.Kind == types.KindFunction {
			// Get the return type of the function
			if len(primaryType.Outputs) > 0 {
				return primaryType.Outputs[0].Type
			}
			// Function with no return type
			return types.Type{}
		}

		// Handle index/slice operations - return the element type for series
		indexOps := ctx.AST.AllIndexOrSlice()
		if len(indexOps) > 0 && primaryType.Kind == types.KindSeries {
			if primaryType.Elem != nil {
				return *primaryType.Elem
			}
		}

		return primaryType
	}
	return types.Type{}
}

func inferPrimaryType(ctx context.Context[parser.IPrimaryExpressionContext]) types.Type {
	if id := ctx.AST.IDENTIFIER(); id != nil {
		text := id.GetText()
		// Handle boolean literals (parsed as identifiers in the grammar)
		if text == "true" || text == "false" {
			return types.U8()
		}
		if varScope, err := ctx.Scope.Resolve(ctx, text); err == nil {
			if varScope.Type.Kind != types.KindInvalid {
				// When a variable is referenced, resolve literal constraints to concrete types.
				// This ensures `x := 10; x + 3.2` fails (x becomes i64, can't add float to int),
				// while `2 + 3.2` (both fresh literals) can still promote to float.
				return resolveLiteralConstraint(varScope.Type)
			}
		}
		return types.Type{}
	}
	if literal := ctx.AST.Literal(); literal != nil {
		return inferLiteralType(context.Child(ctx, literal))
	}
	if expr := ctx.AST.Expression(); expr != nil {
		return InferFromExpression(context.Child(ctx, expr))
	}
	if typeCast := ctx.AST.TypeCast(); typeCast != nil {
		if typeCtx := typeCast.Type_(); typeCtx != nil {
			t, _ := InferFromTypeContext(typeCtx)
			return t
		}
	}
	return types.Type{}
}

func inferLiteralType(ctx context.Context[parser.ILiteralContext]) types.Type {
	if numLit := ctx.AST.NumericLiteral(); numLit != nil {
		return inferNumericLiteralType(ctx, numLit)
	}
	if seriesLit := ctx.AST.SeriesLiteral(); seriesLit != nil {
		return inferSeriesLiteralType(context.Child(ctx, seriesLit))
	}
	text := ctx.AST.GetText()
	if len(text) > 0 && (text[0] == '"' || text[0] == '\'') {
		t := types.String()
		ctx.TypeMap[ctx.AST] = t
		return t
	}
	// Fallback for unknown literals
	t := types.I64()
	ctx.TypeMap[ctx.AST] = t
	return t
}

func inferSeriesLiteralType(ctx context.Context[parser.ISeriesLiteralContext]) types.Type {
	exprList := ctx.AST.ExpressionList()
	if exprList == nil {
		line := ctx.AST.GetStart().GetLine()
		col := ctx.AST.GetStart().GetColumn()
		tvName := fmt.Sprintf("series_elem_%d_%d", line, col)
		constraint := types.NumericConstraint()
		elemType := types.Variable(tvName, &constraint)
		t := types.Series(elemType)
		ctx.TypeMap[ctx.AST] = t
		return t
	}

	exprs := exprList.AllExpression()
	if len(exprs) == 0 {
		line := ctx.AST.GetStart().GetLine()
		col := ctx.AST.GetStart().GetColumn()
		tvName := fmt.Sprintf("series_elem_%d_%d", line, col)
		constraint := types.NumericConstraint()
		elemType := types.Variable(tvName, &constraint)
		t := types.Series(elemType)
		ctx.TypeMap[ctx.AST] = t
		return t
	}

	firstExpr := exprs[0]
	elemType := InferFromExpression(context.Child(ctx, firstExpr))
	allLiterals := parser.IsNumericLiteral(firstExpr)

	for i := 1; i < len(exprs); i++ {
		nextType := InferFromExpression(context.Child(ctx, exprs[i]))
		thisIsLiteral := parser.IsNumericLiteral(exprs[i])
		// Literals can unify across int/float (like Go), but variables cannot
		if allLiterals && thisIsLiteral {
			if !literalsCompatible(elemType, nextType) {
				ctx.Diagnostics.AddError(
					fmt.Errorf("series element %d has incompatible type %s, expected %s", i+1, nextType, elemType),
					exprs[i],
				)
			}
		} else {
			allLiterals = false
			if !seriesElementCompatible(elemType, nextType) {
				ctx.Diagnostics.AddError(
					fmt.Errorf("series element %d has incompatible type %s, expected %s", i+1, nextType, elemType),
					exprs[i],
				)
			}
		}
		if elemType.Kind == types.KindVariable && nextType.Kind != types.KindVariable {
			elemType = nextType
		}
	}

	t := types.Series(elemType)
	ctx.TypeMap[ctx.AST] = t
	return t
}

// seriesElementCompatible checks if two types are compatible for use in the same series literal.
func seriesElementCompatible(t1, t2 types.Type) bool {
	if t1.Kind == types.KindInvalid || t2.Kind == types.KindInvalid {
		return false
	}
	if t1.Kind == types.KindVariable && t2.Kind == types.KindVariable {
		if t1.Constraint == nil || t2.Constraint == nil {
			return true
		}
		// Check if constraints are compatible. IntegerConstraint and FloatConstraint
		// are NOT compatible with each other - mixing inferred int and float variables
		// should be rejected, just like mixing explicit i32 and f64.
		return numericConstraintsCompatible(t1.Constraint.Kind, t2.Constraint.Kind)
	}
	if t1.Kind == types.KindVariable {
		return constraintAccepts(t1.Constraint, t2)
	}
	if t2.Kind == types.KindVariable {
		return constraintAccepts(t2.Constraint, t1)
	}
	return Compatible(t1, t2)
}

// literalsCompatible checks if two literal types can coexist in a series (int/float can mix).
func literalsCompatible(t1, t2 types.Type) bool {
	if t1.Kind == types.KindInvalid || t2.Kind == types.KindInvalid {
		return false
	}
	if t1.Kind == types.KindVariable && t2.Kind == types.KindVariable {
		if t1.Constraint == nil || t2.Constraint == nil {
			return true
		}
		return isNumericConstraint(t1.Constraint.Kind) && isNumericConstraint(t2.Constraint.Kind)
	}
	if t1.Kind == types.KindVariable {
		return constraintAccepts(t1.Constraint, t2)
	}
	if t2.Kind == types.KindVariable {
		return constraintAccepts(t2.Constraint, t1)
	}
	return Compatible(t1, t2)
}

// isNumericConstraint returns true if the kind is a numeric constraint kind.
func isNumericConstraint(kind types.Kind) bool {
	return kind == types.KindNumericConstant ||
		kind == types.KindIntegerConstant ||
		kind == types.KindFloatConstant
}

// numericConstraintsCompatible checks if two constraint kinds are compatible for series elements.
// IntegerConstraint and FloatConstraint are NOT compatible - you cannot mix inferred int and
// float variables in a series literal, consistent with how explicit i32 and f64 are rejected.
// NumericConstant is compatible with both since it represents an unconstrained numeric literal.
func numericConstraintsCompatible(k1, k2 types.Kind) bool {
	// Same constraint kind is always compatible
	if k1 == k2 {
		return true
	}
	// NumericConstant is compatible with any numeric constraint
	if k1 == types.KindNumericConstant || k2 == types.KindNumericConstant {
		return isNumericConstraint(k1) && isNumericConstraint(k2)
	}
	// IntegerConstraint and FloatConstraint are NOT compatible with each other
	if isNumericConstraint(k1) && isNumericConstraint(k2) {
		return false
	}
	// Non-numeric constraints must match exactly
	return k1 == k2
}

func constraintAccepts(constraint *types.Type, concreteType types.Type) bool {
	if constraint == nil {
		return true
	}
	switch constraint.Kind {
	case types.KindNumericConstant, types.KindIntegerConstant:
		return concreteType.IsNumeric()
	case types.KindFloatConstant:
		return concreteType.IsFloat()
	default:
		return types.Equal(*constraint, concreteType)
	}
}

func inferNumericLiteralType(
	ctx context.Context[parser.ILiteralContext],
	numLit parser.INumericLiteralContext,
) types.Type {
	// Determine constraint based on literal form (integer vs float)
	// This applies to both plain numeric literals AND unit literals
	var (
		isFloat    = numLit.FLOAT_LITERAL() != nil
		line       = ctx.AST.GetStart().GetLine()
		col        = ctx.AST.GetStart().GetColumn()
		tvName     = fmt.Sprintf("lit_%d_%d", line, col)
		constraint = lo.Ternary(isFloat, types.FloatConstraint(), types.IntegerConstraint())
		tv         = types.Variable(tvName, &constraint)
	)

	// Check for unit suffix (e.g., 5psi, 3s, 100Hz)
	if unitID := numLit.IDENTIFIER(); unitID != nil {
		unitName := unitID.GetText()
		if unit, ok := units.Resolve(unitName); ok {
			tv.Unit = unit
		}
	}

	// Registering a type variable with itself - cannot fail
	_ = ctx.Constraints.AddEquality(tv, tv, ctx.AST, "literal type variable")
	ctx.TypeMap[ctx.AST] = tv
	return tv
}

// resolveLiteralConstraint converts a type variable with a literal constraint to a concrete type.
// This is used when a variable is referenced in an expression to distinguish between:
// - Direct literals (2 + 3.2) which can be promoted
// - Variable references (x := 10; x + 3.2) which should fail
func resolveLiteralConstraint(t types.Type) types.Type {
	if t.Kind != types.KindVariable || t.Constraint == nil {
		return t
	}
	switch t.Constraint.Kind {
	case types.KindIntegerConstant:
		result := types.I64()
		result.Unit = t.Unit
		return result
	case types.KindFloatConstant:
		result := types.F64()
		result.Unit = t.Unit
		return result
	case types.KindNumericConstant:
		result := types.F64()
		result.Unit = t.Unit
		return result
	default:
		return t
	}
}
