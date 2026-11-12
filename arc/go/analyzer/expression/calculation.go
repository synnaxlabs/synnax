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
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/codes"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

// The parts of the hint that gives the new declaration of a variable.
const (
	redeclareHintPrefix = "hint: change `"
	redeclareHintJoin   = "` to `"
)

// RedeclareHint returns the hint that changes the declaration from to the declaration
// to.
func RedeclareHint(from, to string) string {
	return redeclareHintPrefix + from + redeclareHintJoin + to + "`"
}

// ParseRedeclareHint returns the new declaration that a RedeclareHint gives.
func ParseRedeclareHint(hint string) (string, bool) {
	rest, ok := strings.CutPrefix(hint, redeclareHintPrefix)
	if !ok {
		return "", false
	}
	_, to, ok := strings.Cut(rest, redeclareHintJoin)
	if !ok {
		return "", false
	}
	return strings.CutSuffix(to, "`")
}

// Sink is the declaration of a variable that a value goes to.
type Sink struct {
	// Name is the name of the variable.
	Name string
	// Type is the type of the variable.
	Type types.Type
	// AST is the declaration.
	AST antlr.ParserRuleContext
	// Init is the initializer of the declaration.
	Init parser.IExpressionContext
	// Stateful is set for a declaration with $=.
	Stateful bool
}

// SinkOf returns the declaration of sym as a Sink. It is nil for a declaration in
// another module, which a hint cannot change.
func SinkOf[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	sym *symbol.Symbol,
) *Sink {
	if sym.Root() != ctx.Scope.Root() {
		return nil
	}
	return &Sink{
		Name:     sym.Name,
		Type:     sym.Type,
		AST:      sym.AST,
		Init:     sym.Initializer(),
		Stateful: sym.Kind == symbol.KindStatefulVariable,
	}
}

// redeclare is a declaration and the declaration that replaces it.
type redeclare struct {
	at       antlr.ParserRuleContext
	from, to string
}

// WarnTruncation warns at ast when the integer variable source converts to the larger
// unit of target, which truncates it. sink is nil for an input.
func WarnTruncation[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	ast antlr.ParserRuleContext,
	source *symbol.Symbol,
	target types.Type,
	sink *Sink,
) {
	warnTruncation(ctx, ast, source.Type, SinkOf(ctx, source), target, sink)
}

// WarnValueTruncation warns at ast when expr, an integer of type source, converts to
// the larger unit of target. A literal warns only when it leaves a remainder. expr is
// nil for a value that is not an expression, and sink is nil for an input.
func WarnValueTruncation[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	ast antlr.ParserRuleContext,
	expr parser.IExpressionContext,
	source, target types.Type,
	sink *Sink,
) {
	var sourceSink *Sink
	if expr != nil && parser.IsLiteral(expr) {
		if parsed, err := literal.ParseConst(
			expr,
			ctx.Constraints.ApplySubstitutions(target),
		); err != nil || !parsed.Truncated {
			return
		}
	} else if head, tail := parser.ExpressionNameParts(expr); head != "" {
		if sym, err := ctx.Scope.ResolveMember(ctx, head, tail); err == nil {
			sourceSink = SinkOf(ctx, sym)
		}
	}
	warnTruncation(ctx, ast, source, sourceSink, target, sink)
}

func warnTruncation[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	ast antlr.ParserRuleContext,
	source types.Type,
	sourceSink *Sink,
	target types.Type,
	sink *Sink,
) {
	from, to := source.Unwrap().Unit, target.Unwrap().Unit
	if !types.UnitsConvertible(from, to) || from.Scale >= to.Scale {
		return
	}
	for _, t := range []types.Type{source, target} {
		t = ctx.Constraints.ApplySubstitutions(t).Unwrap()
		untyped := t.Kind == types.KindVariable && t.Constraint != nil &&
			t.Constraint.Kind == types.KindIntegerConstant
		if !t.IsInteger() && !untyped {
			return
		}
	}
	var redeclares []redeclare
	if r, ok := redeclaration(ctx, sink, from); ok {
		redeclares = append(redeclares, r)
	}
	if r, ok := redeclaration(ctx, sourceSink, to); ok {
		redeclares = append(redeclares, r)
	}
	d := codes.UnitTruncation.At(ast, from.Name, to.Name)
	d.Notes = hintNotes(redeclares)
	ctx.Diagnostics.Add(d)
}

// ReportMissingUnit reports at ast when a value of type source, which has a unit, goes
// to subject, which is of type target and has none. It returns whether it did. The
// hint is the declaration of sink in that unit, and note when sink gives none.
func ReportMissingUnit[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	ast antlr.ParserRuleContext,
	source, target types.Type,
	subject string,
	sink *Sink,
	note string,
) bool {
	unit := source.Unwrap().Unit
	if unit == nil || target.Unwrap().Unit != nil {
		return false
	}
	d := codes.UnitMissing.At(ast, subject, unit.Name)
	if r, ok := redeclaration(ctx, sink, unit); ok {
		d.Notes = hintNotes([]redeclare{r})
	} else if note != "" {
		d = d.WithNote(note)
	}
	ctx.Diagnostics.Add(d)
	return true
}

// hintNotes returns a hint for each redeclare.
func hintNotes(redeclares []redeclare) []protocol.DiagnosticRelatedInformation {
	notes := make([]protocol.DiagnosticRelatedInformation, len(redeclares))
	for i, r := range redeclares {
		var at diagnostics.Diagnostic
		at.SetRange(r.at)
		notes[i] = protocol.DiagnosticRelatedInformation{
			Message:  RedeclareHint(r.from, r.to),
			Location: protocol.Location{Range: at.Range},
		}
	}
	return notes
}

// redeclaration returns the declaration of sink in unit. ok is false when sink is not
// set from a literal that converts to unit with no remainder.
func redeclaration[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	sink *Sink,
	unit *types.Unit,
) (redeclare, bool) {
	if sink == nil || unit == nil || sink.Init == nil || sink.AST == nil ||
		!parser.IsLiteral(sink.Init) {
		return redeclare{}, false
	}
	t := ctx.Constraints.ApplySubstitutions(sink.Type)
	if t.Kind == types.KindVariable {
		kind := types.KindF64
		if c := t.Constraint; c != nil && c.Kind == types.KindIntegerConstant {
			kind = types.KindI64
		}
		t = types.Type{Kind: kind}
	}
	t.Unit = unit
	if parsed, err := literal.ParseConst(sink.Init, t); err != nil || parsed.Truncated {
		return redeclare{}, false
	}
	declare := ":="
	if sink.Stateful {
		declare = "$="
	}
	start, stop := sink.AST.GetStart(), sink.AST.GetStop()
	return redeclare{
		at:   sink.AST,
		from: start.GetInputStream().GetText(start.GetStart(), stop.GetStop()),
		to: fmt.Sprintf(
			"%s %s %s %s", sink.Name, t, declare, parser.GetExpressionText(sink.Init),
		),
	}, true
}
