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
	"github.com/synnaxlabs/arc/analyzer/codes"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
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
	if from != nil && to == nil {
		ctx.Diagnostics.Add(codes.UnitMissing.At(w.AST, w.Subject, from.Name).
			WithNote("hint: declare " + w.Subject + " in " + from.Name))
		return false, false
	}
	if !types.UnitsConvertible(from, to) {
		return false, true
	}
	isLiteral := w.Expr != nil && parser.IsLiteral(w.Expr)
	if w.Expr != nil {
		TakeUnit(ctx, w.Expr, to)
	}
	if !w.Lowered && truncates(ctx, w, isLiteral) {
		ctx.Diagnostics.Add(codes.UnitTruncation.At(w.AST, from.Name, to.Name))
	}
	return !isLiteral, true
}

// truncates reports whether w drops a part of an integer in a larger unit.
func truncates[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	w Write,
	isLiteral bool,
) bool {
	if w.Source.Unwrap().Unit.Scale >= w.Target.Unwrap().Unit.Scale {
		return false
	}
	for _, t := range []types.Type{w.Source, w.Target} {
		t = ctx.Constraints.ApplySubstitutions(t).Unwrap()
		untyped := t.Kind == types.KindVariable && t.Constraint != nil &&
			t.Constraint.Kind == types.KindIntegerConstant
		if !t.IsInteger() && !untyped {
			return false
		}
	}
	if !isLiteral {
		return true
	}
	parsed, err := literal.ParseConst(
		w.Expr,
		ctx.Constraints.ApplySubstitutions(w.Target),
	)
	return err == nil && parsed.Truncated
}
