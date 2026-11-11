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
	"github.com/synnaxlabs/x/set"
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

// SinkOf returns the declaration of sym as a Sink.
func SinkOf(sym *symbol.Symbol) *Sink {
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

// ReportCalculation reports expr when it calculates with variables in another unit
// than target, and returns whether it did. sink is nil for an input.
func ReportCalculation[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	expr parser.IExpressionContext,
	target types.Type,
	subject string,
	sink *Sink,
) bool {
	if head, _ := parser.ExpressionNameParts(
		expr,
	); head != "" ||
		parser.IsLiteral(expr) {
		return false
	}
	variables := unitVariables(ctx, ctx.Scope, expr, target.Unit)
	if len(variables) == 0 {
		return false
	}
	names := make([]string, len(variables))
	for i, sym := range variables {
		names[i] = fmt.Sprintf("'%s', which is in %s", sym.Name, sym.Type.Unit.Name)
	}
	var (
		redeclares []redeclare
		hinted     = set.New[*symbol.Symbol]()
		unit       = variables[0].Type.Unit
	)
	for _, sym := range variables {
		redeclares = append(
			redeclares, sourceRedeclares(ctx, sym, target.Unit, hinted)...,
		)
		if !sym.Type.Unit.Equal(*unit) {
			unit = nil
		}
	}
	if r, ok := redeclaration(ctx, sink, unit, expr); ok {
		redeclares = append(redeclares, r)
	}
	d := codes.UnitCalculation.At(expr, subject, strings.Join(names, ", and "))
	d.Notes = hintNotes(redeclares)
	ctx.Diagnostics.Add(d)
	return true
}

// WarnTruncation warns at ast when an integer of source converts to a larger unit of
// sink, which truncates it.
func WarnTruncation[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	ast antlr.ParserRuleContext,
	source, sink *symbol.Symbol,
) {
	from, to := source.Type.Unit, sink.Type.Unit
	if from == nil || to == nil || from.Scale >= to.Scale {
		return
	}
	t := ctx.Constraints.ApplySubstitutions(sink.Type)
	untyped := t.Kind == types.KindVariable && t.Constraint != nil &&
		t.Constraint.Kind == types.KindIntegerConstant
	if !t.IsInteger() && !untyped {
		return
	}
	var redeclares []redeclare
	if r, ok := redeclaration(ctx, SinkOf(sink), from, nil); ok {
		redeclares = append(redeclares, r)
	}
	if r, ok := redeclaration(ctx, SinkOf(source), to, nil); ok {
		redeclares = append(redeclares, r)
	}
	d := codes.UnitTruncation.At(ast, from.Name, to.Name)
	d.Notes = hintNotes(redeclares)
	ctx.Diagnostics.Add(d)
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

// sourceRedeclares returns the redeclare of sym in unit when it is set from a literal,
// and of the variables that set it otherwise. hinted holds the variables it has seen.
func sourceRedeclares[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	sym *symbol.Symbol,
	unit *types.Unit,
	hinted set.Set[*symbol.Symbol],
) []redeclare {
	if hinted.Contains(sym) {
		return nil
	}
	hinted.Add(sym)
	if r, ok := redeclaration(ctx, SinkOf(sym), unit, nil); ok {
		return []redeclare{r}
	}
	init := sym.Initializer()
	if init == nil || sym.Parent == nil {
		return nil
	}
	var out []redeclare
	for _, source := range unitVariables(ctx, sym.Parent, init, unit) {
		out = append(out, sourceRedeclares(ctx, source, unit, hinted)...)
	}
	return out
}

// unitVariables returns the variables that expr reads in scope whose unit converts to
// target.
func unitVariables[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	scope *symbol.Symbol,
	expr antlr.Tree,
	target *types.Unit,
) []*symbol.Symbol {
	var (
		variables []*symbol.Symbol
		seen      = set.New[*symbol.Symbol]()
		walk      func(antlr.Tree)
	)
	walk = func(node antlr.Tree) {
		terminal, ok := node.(antlr.TerminalNode)
		if !ok {
			for _, child := range node.GetChildren() {
				walk(child)
			}
			return
		}
		if terminal.GetSymbol().GetTokenType() != parser.ArcParserIDENTIFIER {
			return
		}
		// The unit suffix of a literal is an identifier too.
		if _, ok = terminal.GetParent().(parser.INumericLiteralContext); ok {
			return
		}
		sym, err := scope.Resolve(ctx, terminal.GetText())
		if err != nil || !sym.IsValueVariable() || seen.Contains(sym) ||
			!types.UnitsConvertible(sym.Type.Unit, target) {
			return
		}
		seen.Add(sym)
		variables = append(variables, sym)
	}
	walk(expr)
	return variables
}

// redeclaration returns the declaration of sink in unit. ok is false when sink is not
// set from a literal that converts to unit, or from the calculation value.
func redeclaration[T antlr.ParserRuleContext](
	ctx acontext.Context[T],
	sink *Sink,
	unit *types.Unit,
	value parser.IExpressionContext,
) (redeclare, bool) {
	if sink == nil || unit == nil || sink.Init == nil || sink.AST == nil {
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
	if parser.IsLiteral(sink.Init) {
		if _, err := literal.ParseConst(sink.Init, t); err != nil {
			return redeclare{}, false
		}
	} else if sink.Init != value {
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
