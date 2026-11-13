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
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/codes"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
)

// Write is a value that goes to a variable, a parameter, or a function result.
type Write struct {
	// AST is where a diagnostic goes.
	AST antlr.ParserRuleContext
	// Expr is the value. It is nil for a value that is not an expression.
	Expr parser.IExpressionContext
	// Source and Target are the types of the value and of the target.
	Source, Target types.Type
	// Subject names the target in a message.
	Subject string
	// Value is a value that is not an expression.
	Value antlr.ParserRuleContext
	// Declaration is where the source types the target. It is nil for no fix.
	Declaration antlr.ParserRuleContext
	// Lowered is set when lowering gives the truncation warning.
	Lowered bool
}

// CheckUnits applies the unit rules to w. ok is false when it reports an error, and
// converts is set when the value converts at run time.
func CheckUnits[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	w Write,
) (converts, ok bool) {
	from, to := w.Source.Unwrap().Unit, w.Target.Unwrap().Unit
	if types.UnitsIncompatible(from, to) {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			w.AST,
			"incompatible dimensions: cannot write %s to %s (%s)",
			from.Name,
			w.Subject,
			to.Name,
		))
		return false, false
	}
	if from != nil && to == nil && w.Target.IsNumeric() {
		d := codes.UnitMissing.At(w.AST, w.Subject, from.Name)
		t := types.Concrete(ctx.Constraints.ApplySubstitutions(w.Target))
		inUnit := t
		inUnit.Unit = from
		if r, ok := retyped(w, inUnit); ok {
			d = d.WithReplacements(r)
		} else {
			d = d.WithNote("hint: declare " + w.Subject + " in " + from.Name)
		}
		// A cast with no unit drops the unit of its value.
		if value := valueOf(w); value != nil {
			d = d.WithFix(value, t.String()+"("+sourceText(value)+")")
		}
		ctx.Diagnostics.Add(d)
		return false, false
	}
	if from == nil && to != nil && w.Expr != nil {
		warnDroppedUnit(ctx, w, to)
	}
	if !types.UnitsConvertible(from, to) {
		return false, true
	}
	isLiteral := w.Expr != nil && parser.IsLiteral(w.Expr)
	if seriesToSeries(w) && !isLiteral {
		ctx.Diagnostics.Add(diagnostics.Errorf(
			w.AST,
			"%s is a series in %s and cannot take a series in %s",
			w.Subject,
			to.Name,
			from.Name,
		))
		return false, false
	}
	if w.Expr != nil {
		TakeUnit(ctx, w.Expr, to)
	}
	if !w.Lowered && truncates(ctx, w, isLiteral) {
		d := codes.UnitTruncation.At(w.AST, from.Name, to.Name)
		t := types.Concrete(ctx.Constraints.ApplySubstitutions(w.Target))
		if decl, ok := redeclarable(w); ok && t.IsInteger() {
			float := diagnostics.Replacement{Node: decl, Text: redeclared(
				decl,
				types.Type{Kind: types.KindF64, Unit: to}.String(),
				sourceText(decl.Expression()),
			)}
			// A value that is not a literal needs a cast that keeps its unit.
			if isLiteral {
				d = d.WithReplacements(float)
			} else if value := valueOf(w); value != nil {
				d = d.WithReplacements(float, diagnostics.Replacement{
					Node: value,
					Text: types.Type{Kind: types.KindF64, Unit: from}.String() +
						"(" + sourceText(value) + ")",
				})
			}
			if init, ok := converted(decl.Expression(), t, to, from); ok {
				typ := ""
				if decl.Type_() != nil {
					t.Unit = from
					typ = t.String()
				}
				d = d.WithFix(decl, redeclared(decl, typ, init))
			}
		}
		ctx.Diagnostics.Add(d)
	}
	return !isLiteral, true
}

// warnDroppedUnit warns when the value of w is a cast with no unit of a value in
// another unit of the dimension of to.
func warnDroppedUnit[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	w Write,
	to *types.Unit,
) {
	primary := parser.GetPrimaryExpression(w.Expr)
	if primary == nil || primary.TypeCast() == nil {
		return
	}
	cast := primary.TypeCast()
	value := cast.Expression()
	castType, err := atypes.InferFromTypeContext(cast.Type_())
	if value == nil || err != nil {
		return
	}
	from := ctx.Constraints.ApplySubstitutions(
		atypes.InferFromExpression(ctx.Child(value)),
	).Unwrap().Unit
	if !types.UnitsConvertible(from, to) {
		return
	}
	d := codes.UnitDropped.At(w.AST, castType, from.Name, w.Subject, to.Name)
	castType.Unit = to
	var number parser.INumericLiteralContext
	if parser.IsLiteral(value) {
		number = parser.GetLiteral(value).NumericLiteral()
	}
	// A float literal does not go to an integer without a cast.
	if number == nil || number.INTEGER_LITERAL() == nil && !castType.IsFloat() {
		d = d.WithFix(w.Expr, castType.String()+"("+sourceText(value)+")")
		ctx.Diagnostics.Add(d)
		return
	}
	unit := number.IDENTIFIER().GetText()
	ctx.Diagnostics.Add(d.
		WithFix(w.Expr, sourceText(value)).
		WithFix(w.Expr, strings.TrimSuffix(sourceText(value), unit)+to.Name))
}

type declaration interface {
	antlr.ParserRuleContext
	IDENTIFIER() antlr.TerminalNode
	Type_() parser.ITypeContext
	Expression() parser.IExpressionContext
}

// redeclarable returns the declaration of the target of w when a literal sets it.
func redeclarable(w Write) (declaration, bool) {
	decl, ok := w.Declaration.(declaration)
	return decl, ok && parser.IsLiteral(decl.Expression())
}

// retyped returns the replacement that gives the target of w the type t.
func retyped(w Write, t types.Type) (r diagnostics.Replacement, ok bool) {
	if typ, ok := w.Declaration.(parser.ITypeContext); ok {
		return diagnostics.Replacement{Node: typ, Text: t.String()}, true
	}
	decl, ok := redeclarable(w)
	if !ok {
		return r, false
	}
	text := redeclared(decl, t.String(), sourceText(decl.Expression()))
	return diagnostics.Replacement{Node: decl, Text: text}, true
}

// valueOf returns the node of the value of w, or nil for a value with no node.
func valueOf(w Write) antlr.ParserRuleContext {
	if w.Expr != nil {
		return w.Expr
	}
	return w.Value
}

// redeclared returns the source of decl with typ and init in place of its own.
func redeclared(decl declaration, typ, init string) string {
	var (
		start = decl.GetStart()
		input = start.GetInputStream()
		name  = decl.IDENTIFIER().GetSymbol()
		rest  = name.GetStop() + 1
	)
	if declared := decl.Type_(); declared != nil {
		rest = declared.GetStop().GetStop() + 1
	}
	if typ != "" {
		typ = " " + typ
	}
	return input.GetText(start.GetStart(), name.GetStop()) + typ +
		input.GetText(rest, decl.Expression().GetStart().GetStart()-1) + init
}

// converted returns the literal init in the unit to, when the number is whole.
func converted(
	init parser.IExpressionContext,
	t types.Type,
	from, to *types.Unit,
) (string, bool) {
	lit := parser.GetLiteral(init)
	if lit == nil || lit.NumericLiteral() == nil {
		return "", false
	}
	if lit.NumericLiteral().IDENTIFIER() != nil {
		t.Unit = to
		parsed, err := literal.ParseConst(init, t)
		if err != nil || parsed.Truncated {
			return "", false
		}
		return fmt.Sprintf("%d%s", parsed.Value, to.Name), true
	}
	number, err := strconv.ParseFloat(sourceText(init), 64)
	if err != nil {
		return "", false
	}
	number *= types.UnitFactor(from, to)
	if !literal.IsExactInteger(number) {
		return "", false
	}
	return strconv.FormatInt(int64(math.Round(number)), 10), true
}

// truncates reports whether w drops a part of a unit from an integer.
func truncates[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	w Write,
	isLiteral bool,
) bool {
	if !integersTruncate(ctx, w.Source, w.Target) {
		return false
	}
	if !isLiteral || seriesToSeries(w) {
		return true
	}
	parsed, err := literal.ParseConst(
		w.Expr,
		ctx.Constraints.ApplySubstitutions(w.Target),
	)
	return err == nil && parsed.Truncated
}

func integersTruncate[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	from, to types.Type,
) bool {
	fromUnit, toUnit := from.Unwrap().Unit, to.Unwrap().Unit
	if !types.UnitsConvertible(fromUnit, toUnit) {
		return false
	}
	if factor := types.UnitFactor(fromUnit, toUnit); factor == math.Trunc(factor) {
		return false
	}
	for _, t := range []types.Type{from, to} {
		t = types.Concrete(ctx.Constraints.ApplySubstitutions(t).Unwrap())
		if !t.IsInteger() {
			return false
		}
	}
	return true
}

func warnOperandTruncation[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	left, right types.Type,
) {
	if left.Unit != nil && right.Unit != nil && left.Unit.Scale < right.Unit.Scale {
		left, right = right, left
	}
	if integersTruncate(ctx, left, right) {
		ctx.Diagnostics.Add(
			codes.UnitTruncation.At(ctx.AST, left.Unit.Name, right.Unit.Name),
		)
	}
}

func seriesToSeries(w Write) bool {
	return w.Source.Kind == types.KindSeries && w.Target.Kind == types.KindSeries
}
