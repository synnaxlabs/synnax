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
	"strconv"

	"github.com/antlr4-go/antlr/v4"
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

// InferFromExpression determines the type of an Arc expression through recursive
// descent.
func InferFromExpression(ctx context.Context[parser.IExpressionContext]) types.Type {
	if logicalOr := ctx.AST.LogicalOrExpression(); logicalOr != nil {
		t := inferLogicalOr(ctx.Child(logicalOr))
		ctx.TypeMap[ctx.AST] = t
		return t
	}
	return types.Type{}
}

func inferLogicalOr(
	ctx context.Context[parser.ILogicalOrExpressionContext],
) types.Type {
	ands := ctx.AST.AllLogicalAndExpression()
	if len(ands) > 1 {
		for _, a := range ands {
			if InferLogicalAnd(ctx.Child(a)).Kind == types.KindSeries {
				return types.Series(types.Bool())
			}
		}
		return types.Bool()
	}
	if len(ands) == 1 {
		return InferLogicalAnd(ctx.Child(ands[0]))
	}
	return types.Type{}
}

func InferLogicalAnd(
	ctx context.Context[parser.ILogicalAndExpressionContext],
) types.Type {
	eqs := ctx.AST.AllEqualityExpression()
	if len(eqs) > 1 {
		for _, e := range eqs {
			if InferEquality(ctx.Child(e)).Kind == types.KindSeries {
				return types.Series(types.Bool())
			}
		}
		return types.Bool()
	}
	if len(eqs) == 1 {
		return InferEquality(ctx.Child(eqs[0]))
	}
	return types.Type{}
}

func InferEquality(ctx context.Context[parser.IEqualityExpressionContext]) types.Type {
	relExpressions := ctx.AST.AllRelationalExpression()
	if len(relExpressions) > 1 {
		for _, r := range relExpressions {
			if InferRelational(ctx.Child(r)).Kind == types.KindSeries {
				return types.Series(types.Bool())
			}
		}
		return types.Bool()
	}
	if len(relExpressions) == 1 {
		return InferRelational(ctx.Child(relExpressions[0]))
	}
	return types.Type{}
}

// InferRelational types a comparison. A series operand makes it element-wise,
// yielding a bool series.
func InferRelational(
	ctx context.Context[parser.IRelationalExpressionContext],
) types.Type {
	additives := ctx.AST.AllAdditiveExpression()
	t := types.Type{}
	if len(additives) > 1 {
		t = types.Bool()
		for _, a := range additives {
			if InferAdditive(ctx.Child(a)).Kind == types.KindSeries {
				t = types.Series(types.Bool())
				break
			}
		}
	} else if len(additives) == 1 {
		t = InferAdditive(ctx.Child(additives[0]))
	}
	ctx.TypeMap[ctx.AST] = t
	return t
}

func inferBinaryType(elemType, nextElem types.Type) (types.Type, bool) {
	if elemType.Kind == types.KindVariable && nextElem.Kind != types.KindVariable {
		elemType = nextElem
	}
	// An integer literal takes the type of a float literal beside it.
	if elemType.Kind == types.KindVariable && nextElem.Kind == types.KindVariable &&
		elemType.Constraint != nil && nextElem.Constraint != nil &&
		elemType.Constraint.Kind == types.KindIntegerConstant &&
		nextElem.Constraint.Kind == types.KindFloatConstant {
		elemType = nextElem
	}
	// Only early-return on incompatibility when both types are concrete.
	// When either is a type variable (literal), we must keep iterating
	// to discover later concrete types that determine the actual type.
	bothConcrete := elemType.Kind != types.KindVariable &&
		nextElem.Kind != types.KindVariable
	if bothConcrete && !Compatible(elemType, nextElem) {
		return elemType, true
	}
	return elemType, false
}

// checkMinusSpacing flags a binary '-' not spaced on both sides (dashed mode).
func checkMinusSpacing(ctx context.Context[parser.IAdditiveExpressionContext]) {
	if !ctx.Config.AllowDashedNames {
		return
	}
	operands := ctx.AST.AllMultiplicativeExpression()
	opIdx := 0
	for _, child := range ctx.AST.GetChildren() {
		tn, ok := child.(antlr.TerminalNode)
		if !ok {
			continue
		}
		if opIdx+1 >= len(operands) {
			break
		}
		op := tn.GetSymbol()
		left, right := operands[opIdx], operands[opIdx+1]
		opIdx++
		if op.GetTokenType() != parser.ArcParserMINUS {
			continue
		}
		if left.GetStop().GetStop()+1 == op.GetStart() ||
			op.GetStop()+1 == right.GetStart().GetStart() {
			col := op.GetColumn()
			ctx.Diagnostics.Add(diagnostics.Diagnostic{
				Severity: protocol.DiagnosticSeverityError,
				Message:  "subtraction requires whitespace on both sides of '-'",
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      uint32(op.GetLine() - 1),
						Character: uint32(col),
					},
					End: protocol.Position{
						Line:      uint32(op.GetLine() - 1),
						Character: uint32(col + len(op.GetText())),
					},
				},
			})
		}
	}
}

func InferAdditive(ctx context.Context[parser.IAdditiveExpressionContext]) types.Type {
	multiplicatives := ctx.AST.AllMultiplicativeExpression()
	if len(multiplicatives) == 0 {
		return types.Type{}
	}
	if len(multiplicatives) > 1 {
		checkMinusSpacing(ctx)
		firstType := InferMultiplicative(ctx.Child(multiplicatives[0]))
		isSeries := firstType.Kind == types.KindSeries
		elemType := firstType.Unwrap()
		unit := elemType.Unit

		for i := 1; i < len(multiplicatives); i++ {
			nextType := InferMultiplicative(ctx.Child(multiplicatives[i]))
			unit = types.CommonUnit(unit, nextType.Unwrap().Unit)
			if nextType.Kind == types.KindSeries {
				isSeries = true
				nextElem := nextType.Unwrap()
				if nextElem.Kind != types.KindVariable {
					elemType = nextElem
				}
				bothConcrete := elemType.Kind != types.KindVariable &&
					nextElem.Kind != types.KindVariable
				if bothConcrete && !Compatible(elemType, nextElem) {
					break
				}
			} else {
				var earlyReturn bool
				elemType, earlyReturn = inferBinaryType(elemType, nextType.Unwrap())
				if earlyReturn {
					break
				}
			}
		}
		elemType.Unit = unit
		resultType := lo.Ternary(isSeries, types.Series(elemType), elemType)
		ctx.TypeMap[ctx.AST] = resultType
		return resultType
	}
	resultType := InferMultiplicative(ctx.Child(multiplicatives[0]))
	ctx.TypeMap[ctx.AST] = resultType
	return resultType
}

func InferMultiplicative(
	ctx context.Context[parser.IMultiplicativeExpressionContext],
) types.Type {
	unaries := ctx.AST.AllUnaryExpression()
	if len(unaries) == 0 {
		return types.Type{}
	}
	if len(unaries) > 1 {
		firstType := InferFromUnaryExpression(ctx.Child(unaries[0]))
		isSeries := firstType.Kind == types.KindSeries
		elemType := firstType.Unwrap()
		unit := elemType.Unit
		operators := multiplicativeOperators(ctx.AST)

		for i := 1; i < len(unaries); i++ {
			nextType := InferFromUnaryExpression(ctx.Child(unaries[i]))
			switch nextUnit := nextType.Unwrap().Unit; operators[i-1] {
			case parser.ArcLexerPERCENT:
				unit = types.CommonUnit(unit, nextUnit)
			case parser.ArcLexerSLASH:
				unit = types.QuotientUnit(unit, nextUnit)
			default:
				unit = types.ProductUnit(unit, nextUnit)
			}
			if nextType.Kind == types.KindSeries {
				isSeries = true
				nextElem := nextType.Unwrap()
				if nextElem.Kind != types.KindVariable {
					elemType = nextElem
				}
				bothConcrete := elemType.Kind != types.KindVariable &&
					nextElem.Kind != types.KindVariable
				if bothConcrete && !Compatible(elemType, nextElem) {
					break
				}
			} else {
				var earlyReturn bool
				elemType, earlyReturn = inferBinaryType(elemType, nextType.Unwrap())
				if earlyReturn {
					break
				}
			}
		}
		elemType.Unit = unit
		resultType := lo.Ternary(isSeries, types.Series(elemType), elemType)
		ctx.TypeMap[ctx.AST] = resultType
		return resultType
	}
	resultType := InferFromUnaryExpression(ctx.Child(unaries[0]))
	ctx.TypeMap[ctx.AST] = resultType
	return resultType
}

// multiplicativeOperators returns the token type of each operator of ast in order.
func multiplicativeOperators(ast parser.IMultiplicativeExpressionContext) []int {
	var operators []int
	for _, child := range ast.GetChildren() {
		if term, ok := child.(antlr.TerminalNode); ok {
			operators = append(operators, term.GetSymbol().GetTokenType())
		}
	}
	return operators
}

func inferPower(ctx context.Context[parser.IPowerExpressionContext]) types.Type {
	if postfix := ctx.AST.PostfixExpression(); postfix != nil {
		baseType := InferPostfix(ctx.Child(postfix))

		// If no power operator, return base type
		if ctx.AST.CARET() == nil || ctx.AST.UnaryExpression() == nil {
			return baseType
		}

		// Recursively infer exponent type (right-associative)
		_ = InferFromUnaryExpression(ctx.Child(ctx.AST.UnaryExpression()))

		// Power operation returns the unwrapped base type
		// (e.g., chan f32 ^ i32 = f32, f64 ^ f64 = f64). A power of a unit value has
		// no unit.
		t := baseType.Unwrap()
		t.Unit = nil
		return t
	}
	return types.Type{}
}

func InferFromUnaryExpression(
	ctx context.Context[parser.IUnaryExpressionContext],
) types.Type {
	var t types.Type
	if ctx.AST.UnaryExpression() != nil {
		// Unwrap channels but keep series: not/- on a series stays a series.
		t = InferFromUnaryExpression(ctx.Child(ctx.AST.UnaryExpression())).UnwrapChan()
	} else if power := ctx.AST.PowerExpression(); power != nil {
		t = inferPower(ctx.Child(power))
	}
	ctx.TypeMap[ctx.AST] = t
	return t
}

func InferPostfix(
	ctx context.Context[parser.IPostfixExpressionContext],
) types.Type {
	primary := ctx.AST.PrimaryExpression()
	if primary == nil {
		return types.Type{}
	}
	t := inferPrimaryType(ctx.Child(primary))
	for _, op := range ctx.AST.GetChildren()[1:] {
		switch op := op.(type) {
		case parser.IFunctionCallSuffixContext:
			if t.Kind != types.KindFunction {
				continue
			}
			if len(t.Outputs) == 0 {
				return types.Type{}
			}
			t = t.Outputs[0].Type
		case parser.IIndexOrSliceContext:
			if op.COLON() == nil && t.Kind == types.KindSeries && t.Elem != nil {
				t = *t.Elem
			}
		}
	}
	return t
}

func inferPrimaryType(
	ctx context.Context[parser.IPrimaryExpressionContext],
) types.Type {
	if qid := ctx.AST.QualifiedIdentifier(); qid != nil {
		head, tail := parser.QualifiedNameParts(qid)
		headSym, err := ctx.Scope.Resolve(ctx, head)
		if err != nil {
			return types.Type{}
		}
		resolved, err := headSym.Resolve(ctx, tail)
		if err != nil {
			return types.Type{}
		}
		if resolved.Kind == symbol.KindConstant {
			return inferConstantType(ctx, resolved)
		}
		return resolved.Type
	}
	if id := ctx.AST.IDENTIFIER(); id != nil {
		text := id.GetText()
		if varScope, err := ctx.Scope.Resolve(ctx, text); err == nil {
			if varScope.Type.Kind != types.KindInvalid {
				// A read gives an untyped variable one type, so `x := 10; x + 3.2`
				// fails while `2 + 3.2` (both fresh literals) promotes to float.
				return bindOnRead(ctx.Constraints, varScope.Type, ctx.AST)
			}
		}
		return types.Type{}
	}
	if literal := ctx.AST.Literal(); literal != nil {
		return inferLiteralType(ctx.Child(literal))
	}
	if expr := ctx.AST.Expression(); expr != nil {
		return InferFromExpression(ctx.Child(expr))
	}
	if typeCast := ctx.AST.TypeCast(); typeCast != nil {
		if typeCtx := typeCast.Type_(); typeCtx != nil {
			// bool conversions are rejected, so the cast has no type.
			if prim := typeCtx.PrimitiveType(); prim != nil && prim.BOOL() != nil {
				return types.Type{}
			}
			// Analysis reports a type that does not infer.
			t, err := InferFromTypeContext(typeCtx)
			if err != nil {
				return types.Type{}
			}
			return t
		}
	}
	return types.Type{}
}

// inferConstantType gives each reference to a module constant its own type variable,
// so one program can use the same constant as both f32 and f64.
func inferConstantType(
	ctx context.Context[parser.IPrimaryExpressionContext],
	sym *symbol.Symbol,
) types.Type {
	tv := types.Freshen(sym.Type, ctx.ScopedKey("const", ctx.AST))
	ctx.Constraints.AddVariable(tv)
	ctx.TypeMap[ctx.AST] = tv
	return tv
}

func inferLiteralType(ctx context.Context[parser.ILiteralContext]) types.Type {
	if numLit := ctx.AST.NumericLiteral(); numLit != nil {
		return inferNumericLiteralType(ctx, numLit)
	}
	if seriesLit := ctx.AST.SeriesLiteral(); seriesLit != nil {
		return inferSeriesLiteralType(ctx.Child(seriesLit))
	}
	if boolLit := ctx.AST.BooleanLiteral(); boolLit != nil {
		t := types.Bool()
		ctx.TypeMap[ctx.AST] = t
		return t
	}
	if parser.StringTerminal(ctx.AST) != nil {
		t := types.String()
		ctx.TypeMap[ctx.AST] = t
		return t
	}
	// Fallback for unknown literals
	t := types.I64()
	ctx.TypeMap[ctx.AST] = t
	return t
}

func inferSeriesLiteralType(
	ctx context.Context[parser.ISeriesLiteralContext],
) types.Type {
	exprList := ctx.AST.ExpressionList()
	if exprList == nil || len(exprList.AllExpression()) == 0 {
		// An empty literal has no element to constrain its type, so the declared type
		// can be any element type, numeric or not. Undeclared, it is f64.
		elemType := types.Variable(ctx.ScopedKey("series_elem", ctx.AST), nil)
		ctx.Constraints.AddDefault(elemType, types.F64())
		t := types.Series(elemType)
		ctx.TypeMap[ctx.AST] = t
		return t
	}

	exprs := exprList.AllExpression()
	firstExpr := exprs[0]
	elemType := InferFromExpression(ctx.Child(firstExpr)).UnwrapChan()
	allLiterals := parser.IsNumericLiteral(firstExpr) || isUntypedNumber(elemType)
	unit := elemType.Unit

	for i := 1; i < len(exprs); i++ {
		nextType := InferFromExpression(ctx.Child(exprs[i])).UnwrapChan()
		thisIsLiteral := parser.IsNumericLiteral(exprs[i]) || isUntypedNumber(nextType)
		// Literals can unify across int/float (like Go), but variables cannot
		compatible := seriesElementCompatible(elemType, nextType)
		if allLiterals && thisIsLiteral {
			compatible = literalsCompatible(elemType, nextType)
		} else {
			allLiterals = false
		}
		if types.UnitsIncompatible(unit, nextType.Unit) {
			compatible = false
		} else {
			unit = types.CommonUnit(unit, nextType.Unit)
		}
		if !compatible {
			ctx.Diagnostics.Add(diagnostics.Errorf(
				exprs[i],
				"series element %d has incompatible type %s, expected %s",
				i+1,
				nextType,
				elemType,
			))
		} else if elemType.Kind == types.KindVariable ||
			nextType.Kind == types.KindVariable {
			// Linking each element to the first keeps a float literal at the end of
			// the chain, so an integer literal takes the float type.
			if err := ctx.Constraints.AddCompatible(
				nextType,
				elemType,
				exprs[i],
				"series element type",
			); err != nil {
				ctx.Diagnostics.Add(diagnostics.Error(err, exprs[i]))
			}
		}
		if elemType.Kind == types.KindVariable && nextType.Kind != types.KindVariable {
			elemType = nextType
		}
		// A float element makes a series of untyped numbers a float series.
		if allLiterals && constrainedTo(nextType, types.KindFloatConstant) {
			elemType = nextType
		}
	}
	elemType.Unit = unit

	t := types.Series(elemType)
	ctx.TypeMap[ctx.AST] = t
	return t
}

// constrainedTo reports whether t is a type variable with the constraint kind k.
func constrainedTo(t types.Type, k types.Kind) bool {
	return t.Kind == types.KindVariable && t.Constraint != nil && t.Constraint.Kind == k
}

// isUntypedNumber reports whether t is the type of an untyped numeric value, such as
// a literal or a module constant, which mixes with other literals like a literal.
func isUntypedNumber(t types.Type) bool {
	return t.Kind == types.KindVariable && t.Constraint != nil &&
		isNumericConstraint(t.Constraint.Kind)
}

// seriesElementCompatible checks if two types are compatible for use in the same series
// literal.
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

// literalsCompatible checks if two literal types can coexist in a series (int/float can
// mix).
func literalsCompatible(t1, t2 types.Type) bool {
	if t1.Kind == types.KindInvalid || t2.Kind == types.KindInvalid {
		return false
	}
	if t1.Kind == types.KindVariable && t2.Kind == types.KindVariable {
		if t1.Constraint == nil || t2.Constraint == nil {
			return true
		}
		return isNumericConstraint(t1.Constraint.Kind) &&
			isNumericConstraint(t2.Constraint.Kind)
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
		kind == types.KindFloatConstant ||
		kind == types.KindExactIntegerFloatConstant
}

// numericConstraintsCompatible checks if two constraint kinds are compatible for series
// elements. IntegerConstraint and FloatConstraint are NOT compatible - you cannot mix
// inferred int and float variables in a series literal, consistent with how explicit
// i32 and f64 are rejected. NumericConstant and ExactIntegerFloatConstant are
// compatible with both since they can adapt.
func numericConstraintsCompatible(k1, k2 types.Kind) bool {
	// Same constraint kind is always compatible
	if k1 == k2 {
		return true
	}
	// NumericConstant is compatible with any numeric constraint
	if k1 == types.KindNumericConstant || k2 == types.KindNumericConstant {
		return isNumericConstraint(k1) && isNumericConstraint(k2)
	}
	if k1 == types.KindExactIntegerFloatConstant ||
		k2 == types.KindExactIntegerFloatConstant {
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
	case types.KindNumericConstant,
		types.KindIntegerConstant,
		types.KindExactIntegerFloatConstant:
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
		floatLit = numLit.FLOAT_LITERAL()
		isFloat  = floatLit != nil
	)

	var constraint types.Type
	if isFloat {
		floatText := floatLit.GetText()
		floatValue, err := strconv.ParseFloat(floatText, 64)
		if err == nil && literal.IsExactInteger(floatValue) {
			constraint = types.ExactIntegerFloatConstraint()
		} else {
			constraint = types.FloatConstraint()
		}
	} else {
		constraint = types.IntegerConstraint()
	}
	tv := types.Variable(ctx.ScopedKey("lit", ctx.AST), &constraint)

	// Check for unit suffix (e.g., 5psi, 3s, 100Hz)
	if unitID := numLit.IDENTIFIER(); unitID != nil {
		unitName := unitID.GetText()
		if unit, ok := units.Resolve(unitName); ok {
			tv.Unit = unit
		}
	}

	if err := ctx.Constraints.AddEquality(
		tv,
		tv,
		ctx.AST,
		"literal type variable",
	); err != nil {
		zap.S().
			DPanicf("unexpected error registering type variable with itself: %v", err)
	}
	ctx.TypeMap[ctx.AST] = tv
	return tv
}

// bindOnRead returns the type a read of a variable of type t sees. An untyped literal
// type, or a series of one, takes the type an earlier use bound it to. With none, the
// read binds it to its default, so every later use sees the same type. The element of
// an empty series literal stays open for a later use to bind.
func bindOnRead(
	cs *constraints.System,
	t types.Type,
	source antlr.ParserRuleContext,
) types.Type {
	resolved := cs.ApplySubstitutions(t)
	if resolved.Kind == types.KindSeries && resolved.Elem != nil {
		return types.Series(bindOnRead(cs, *resolved.Elem, source))
	}
	if constrainedTo(resolved, types.KindNumericConstant) {
		return resolved
	}
	if resolved.Kind != types.KindVariable {
		return resolved
	}
	def := resolveLiteralConstraint(resolved)
	if def.Kind == types.KindVariable {
		return resolved
	}
	// An unbound variable always unifies with its own default.
	lo.Must0(cs.AddEquality(resolved, def, source, "variable type"))
	return def
}

// resolveLiteralConstraint returns the default type of a type variable with a literal
// constraint, or t unchanged.
func resolveLiteralConstraint(t types.Type) types.Type {
	if t.Kind != types.KindVariable || t.Constraint == nil {
		return t
	}
	switch t.Constraint.Kind {
	case types.KindIntegerConstant:
		result := types.I64()
		result.Unit = t.Unit
		return result
	case types.KindFloatConstant,
		types.KindNumericConstant,
		types.KindExactIntegerFloatConstant:
		result := types.F64()
		result.Unit = t.Unit
		return result
	default:
		return t
	}
}
