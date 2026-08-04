// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package parser

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/x/set"
)

// IsLiteral checks if an expression is a single literal value (optionally negated)
// with no other operators.
func IsLiteral(expr IExpressionContext) bool {
	return isLiteral(expr.LogicalOrExpression())
}

// IsNegatedLiteral returns true if the expression is a negated literal (e.g. -3h).
func IsNegatedLiteral(expr IExpressionContext) bool {
	return isNegatedLiteral(expr.LogicalOrExpression())
}

func isNegatedLiteral(node antlr.ParserRuleContext) bool {
	if node == nil {
		return false
	}
	switch ctx := node.(type) {
	case ILogicalOrExpressionContext:
		ands := ctx.AllLogicalAndExpression()
		return len(ands) == 1 && isNegatedLiteral(ands[0])
	case ILogicalAndExpressionContext:
		eqs := ctx.AllEqualityExpression()
		return len(eqs) == 1 && isNegatedLiteral(eqs[0])
	case IEqualityExpressionContext:
		rels := ctx.AllRelationalExpression()
		return len(rels) == 1 && isNegatedLiteral(rels[0])
	case IRelationalExpressionContext:
		adds := ctx.AllAdditiveExpression()
		return len(adds) == 1 && isNegatedLiteral(adds[0])
	case IAdditiveExpressionContext:
		muls := ctx.AllMultiplicativeExpression()
		return len(muls) == 1 && isNegatedLiteral(muls[0])
	case IMultiplicativeExpressionContext:
		pows := ctx.AllPowerExpression()
		return len(pows) == 1 && isNegatedLiteral(pows[0])
	case IPowerExpressionContext:
		return ctx.CARET() == nil && isNegatedLiteral(ctx.UnaryExpression())
	case IUnaryExpressionContext:
		return ctx.MINUS() != nil && ctx.UnaryExpression() != nil &&
			isLiteral(ctx.UnaryExpression())
	}
	return false
}

func isLiteral(node antlr.ParserRuleContext) bool {
	if node == nil {
		return false
	}
	switch ctx := node.(type) {
	case ILogicalOrExpressionContext:
		ands := ctx.AllLogicalAndExpression()
		return len(ands) == 1 && isLiteral(ands[0])
	case ILogicalAndExpressionContext:
		eqs := ctx.AllEqualityExpression()
		return len(eqs) == 1 && isLiteral(eqs[0])
	case IEqualityExpressionContext:
		rels := ctx.AllRelationalExpression()
		return len(rels) == 1 && isLiteral(rels[0])
	case IRelationalExpressionContext:
		adds := ctx.AllAdditiveExpression()
		return len(adds) == 1 && isLiteral(adds[0])
	case IAdditiveExpressionContext:
		muls := ctx.AllMultiplicativeExpression()
		return len(muls) == 1 && isLiteral(muls[0])
	case IMultiplicativeExpressionContext:
		pows := ctx.AllPowerExpression()
		return len(pows) == 1 && isLiteral(pows[0])
	case IPowerExpressionContext:
		return ctx.CARET() == nil && isLiteral(ctx.UnaryExpression())
	case IUnaryExpressionContext:
		if ctx.MINUS() != nil && ctx.UnaryExpression() != nil {
			return isLiteral(ctx.UnaryExpression())
		}
		return ctx.UnaryExpression() == nil && isLiteral(ctx.PostfixExpression())
	case IPostfixExpressionContext:
		return len(ctx.AllIndexOrSlice()) == 0 &&
			len(ctx.AllFunctionCallSuffix()) == 0 &&
			isLiteral(ctx.PrimaryExpression())
	case IPrimaryExpressionContext:
		return ctx.Literal() != nil
	}
	return false
}

// GetLiteral extracts the literal node from a pure literal expression.
// Callers should first verify IsLiteral returns true.
func GetLiteral(expr IExpressionContext) ILiteralContext {
	return GetLiteralNode(expr.LogicalOrExpression())
}

// GetLiteralNode extracts the literal from any AST node type.
func GetLiteralNode(node antlr.ParserRuleContext) ILiteralContext {
	if node == nil {
		return nil
	}
	switch ctx := node.(type) {
	case ILogicalOrExpressionContext:
		ands := ctx.AllLogicalAndExpression()
		if len(ands) == 1 {
			return GetLiteralNode(ands[0])
		}
	case ILogicalAndExpressionContext:
		eqs := ctx.AllEqualityExpression()
		if len(eqs) == 1 {
			return GetLiteralNode(eqs[0])
		}
	case IEqualityExpressionContext:
		rels := ctx.AllRelationalExpression()
		if len(rels) == 1 {
			return GetLiteralNode(rels[0])
		}
	case IRelationalExpressionContext:
		adds := ctx.AllAdditiveExpression()
		if len(adds) == 1 {
			return GetLiteralNode(adds[0])
		}
	case IAdditiveExpressionContext:
		muls := ctx.AllMultiplicativeExpression()
		if len(muls) == 1 {
			return GetLiteralNode(muls[0])
		}
	case IMultiplicativeExpressionContext:
		pows := ctx.AllPowerExpression()
		if len(pows) == 1 {
			return GetLiteralNode(pows[0])
		}
	case IPowerExpressionContext:
		if ctx.CARET() == nil {
			return GetLiteralNode(ctx.UnaryExpression())
		}
	case IUnaryExpressionContext:
		if ctx.MINUS() != nil && ctx.UnaryExpression() != nil {
			return GetLiteralNode(ctx.UnaryExpression())
		}
		if ctx.UnaryExpression() == nil {
			return GetLiteralNode(ctx.PostfixExpression())
		}
	case IPostfixExpressionContext:
		if len(ctx.AllIndexOrSlice()) == 0 && len(ctx.AllFunctionCallSuffix()) == 0 {
			return GetLiteralNode(ctx.PrimaryExpression())
		}
	case IPrimaryExpressionContext:
		return ctx.Literal()
	}
	return nil
}

// StringTerminal returns the string literal terminal node for either
// STR_LITERAL or STR_LITERAL_MULTI form, or
// nil if the literal is not a string.
func StringTerminal(lit ILiteralContext) antlr.TerminalNode {
	if t := lit.STR_LITERAL(); t != nil {
		return t
	}
	if t := lit.STR_LITERAL_MULTI(); t != nil {
		return t
	}
	return nil
}

// IsNumericLiteral checks if an expression is a numeric literal (int or float),
// possibly with a unary minus. This is more permissive than IsLiteral for cases
// like [-1, -2.0] where we want to treat negated numbers as literals.
func IsNumericLiteral(expr IExpressionContext) bool {
	return isNumericLiteral(expr.LogicalOrExpression())
}

func isNumericLiteral(node antlr.ParserRuleContext) bool {
	if node == nil {
		return false
	}
	switch ctx := node.(type) {
	case ILogicalOrExpressionContext:
		ands := ctx.AllLogicalAndExpression()
		return len(ands) == 1 && isNumericLiteral(ands[0])
	case ILogicalAndExpressionContext:
		eqs := ctx.AllEqualityExpression()
		return len(eqs) == 1 && isNumericLiteral(eqs[0])
	case IEqualityExpressionContext:
		rels := ctx.AllRelationalExpression()
		return len(rels) == 1 && isNumericLiteral(rels[0])
	case IRelationalExpressionContext:
		adds := ctx.AllAdditiveExpression()
		return len(adds) == 1 && isNumericLiteral(adds[0])
	case IAdditiveExpressionContext:
		muls := ctx.AllMultiplicativeExpression()
		return len(muls) == 1 && isNumericLiteral(muls[0])
	case IMultiplicativeExpressionContext:
		pows := ctx.AllPowerExpression()
		return len(pows) == 1 && isNumericLiteral(pows[0])
	case IPowerExpressionContext:
		return ctx.CARET() == nil && isNumericLiteral(ctx.UnaryExpression())
	case IUnaryExpressionContext:
		if ctx.MINUS() != nil {
			return isNumericLiteral(ctx.UnaryExpression())
		}
		return ctx.UnaryExpression() == nil && isNumericLiteral(ctx.PostfixExpression())
	case IPostfixExpressionContext:
		return len(ctx.AllIndexOrSlice()) == 0 &&
			len(ctx.AllFunctionCallSuffix()) == 0 &&
			isNumericLiteral(ctx.PrimaryExpression())
	case IPrimaryExpressionContext:
		if lit := ctx.Literal(); lit != nil {
			return lit.NumericLiteral() != nil
		}
		return false
	}
	return false
}

// CollectIdentifiers returns the primary-expression identifier names in expr,
// deduplicated, in source order.
func CollectIdentifiers(expr IExpressionContext) []string {
	var (
		names []string
		seen  = make(set.Set[string])
		walk  func(t antlr.Tree)
	)
	walk = func(t antlr.Tree) {
		if pe, ok := t.(IPrimaryExpressionContext); ok {
			if id := pe.IDENTIFIER(); id != nil {
				name := id.GetText()
				if !seen.Contains(name) {
					seen.Add(name)
					names = append(names, name)
				}
			}
		}
		for i := 0; i < t.GetChildCount(); i++ {
			walk(t.GetChild(i))
		}
	}
	walk(expr)
	return names
}

// GetPrimaryExpression extracts the primary expression from an expression that has no
// operators. Returns nil if the expression contains any binary or unary operators.
func GetPrimaryExpression(expr IExpressionContext) IPrimaryExpressionContext {
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
	if pow.CARET() != nil {
		return nil
	}
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

// QualifiedNameParts returns the head and tail of a qualified identifier
// (e.g., "math" and "avg" for `math.avg`). Reads the terminal children
// directly so it works when either side is a lexer keyword (AUTHORITY on
// the left, FOR on the right) and not just an IDENTIFIER.
func QualifiedNameParts(qid IQualifiedIdentifierContext) (head, tail string) {
	head = qid.GetChild(0).(antlr.TerminalNode).GetText()
	tail = qid.GetChild(2).(antlr.TerminalNode).GetText()
	return head, tail
}

// QualifiedName returns the dot-joined name from a qualified identifier
// (e.g., "math.avg"). For symbol resolution, prefer QualifiedNameParts so
// the head and tail are resolved as separate symbols. This helper is for
// display purposes (diagnostics, completion items, debug strings).
func QualifiedName(qid IQualifiedIdentifierContext) string {
	head, tail := QualifiedNameParts(qid)
	return head + "." + tail
}

// FunctionNameParts returns the head and tail of a function context's
// name. A bare function (set_authority{}) returns (name, ""); a qualified
// one (math.avg{}) returns (head, tail).
func FunctionNameParts(fn IFunctionContext) (head, tail string) {
	if qid := fn.QualifiedIdentifier(); qid != nil {
		return QualifiedNameParts(qid)
	}
	return fn.IDENTIFIER().GetText(), ""
}

// FunctionName extracts the joined name from a function context. For
// resolution use FunctionNameParts.
func FunctionName(fn IFunctionContext) string {
	head, tail := FunctionNameParts(fn)
	if tail == "" {
		return head
	}
	return head + "." + tail
}

// PrimaryNameParts returns the head and tail of a primary expression's
// identifier. Bare (x) returns (name, ""); qualified (math.avg) returns
// (head, tail). Returns ("", "") when the primary is not an identifier.
func PrimaryNameParts(primary IPrimaryExpressionContext) (head, tail string) {
	if qid := primary.QualifiedIdentifier(); qid != nil {
		return QualifiedNameParts(qid)
	}
	if id := primary.IDENTIFIER(); id != nil {
		return id.GetText(), ""
	}
	return "", ""
}

// PrimaryName extracts the joined name from a primary expression. For
// resolution use PrimaryNameParts. Returns "" if the primary is not an
// identifier.
func PrimaryName(primary IPrimaryExpressionContext) string {
	head, tail := PrimaryNameParts(primary)
	if head == "" {
		return ""
	}
	if tail == "" {
		return head
	}
	return head + "." + tail
}

// ImportEntry is one collected import. Path is the dotted source path
// ("math", "math.trig"); Alias is the user-visible qualifier.
type ImportEntry struct {
	Path  string
	Alias string
	AST   IImportItemContext
}

// importPathText returns the dotted source text of an importPath.
// AUTHORITY is a lexer keyword, so the head is read separately.
func importPathText(p IImportPathContext) string {
	var b strings.Builder
	if head := p.ImportPathHead(); head != nil {
		if id := head.IDENTIFIER(); id != nil {
			b.WriteString(id.GetText())
		} else if auth := head.AUTHORITY(); auth != nil {
			b.WriteString(auth.GetText())
		}
	}
	for _, id := range p.AllIDENTIFIER() {
		b.WriteByte('.')
		b.WriteString(id.GetText())
	}
	return b.String()
}

// importAlias returns the user-visible alias: the AS clause identifier
// when present, otherwise the last path segment.
func importAlias(item IImportItemContext) string {
	if item.AS() != nil {
		if id := item.IDENTIFIER(); id != nil {
			return id.GetText()
		}
	}
	path := importPathText(item.ImportPath())
	if idx := strings.LastIndexByte(path, '.'); idx >= 0 {
		return path[idx+1:]
	}
	return path
}

// Imports collects every importStatement entry from a program in source order.
func Imports(prog IProgramContext) []ImportEntry {
	if prog == nil {
		return nil
	}
	var entries []ImportEntry
	for _, item := range prog.AllTopLevelItem() {
		stmt := item.ImportStatement()
		if stmt == nil {
			continue
		}
		for _, imp := range stmt.AllImportItem() {
			entries = append(entries, ImportEntry{
				Path:  importPathText(imp.ImportPath()),
				Alias: importAlias(imp),
				AST:   imp,
			})
		}
	}
	return entries
}

// GetExpressionText extracts the source text of an expression from the token stream.
func GetExpressionText(expr IExpressionContext) string {
	if expr == nil {
		return ""
	}
	start := expr.GetStart()
	stop := expr.GetStop()
	if start != nil && stop != nil {
		stream := start.GetTokenSource().GetInputStream()
		if stream != nil {
			return stream.GetText(start.GetStart(), stop.GetStop())
		}
	}
	return expr.GetText()
}
