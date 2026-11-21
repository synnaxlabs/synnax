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
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/x/set"
	"golang.org/x/text/unicode/norm"
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
		unaries := ctx.AllUnaryExpression()
		return len(unaries) == 1 && isNegatedLiteral(unaries[0])
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
		unaries := ctx.AllUnaryExpression()
		return len(unaries) == 1 && isLiteral(unaries[0])
	case IUnaryExpressionContext:
		if ctx.MINUS() != nil && ctx.UnaryExpression() != nil {
			return isLiteral(ctx.UnaryExpression())
		}
		return ctx.UnaryExpression() == nil && isLiteral(ctx.PowerExpression())
	case IPowerExpressionContext:
		return ctx.CARET() == nil && isLiteral(ctx.PostfixExpression())
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
	if primary, ok := node.(IPrimaryExpressionContext); ok {
		return primary.Literal()
	}
	if operand := literalOperand(node); operand != nil {
		return GetLiteralNode(operand)
	}
	return nil
}

// literalOperand returns the operand of node that can hold its literal: the only
// operand of a binary chain, the operand of a negation, or the wrapped expression.
// Returns nil when node applies any other operator.
func literalOperand(node antlr.ParserRuleContext) antlr.ParserRuleContext {
	switch ctx := node.(type) {
	case ILogicalOrExpressionContext:
		return soleOperand(ctx.AllLogicalAndExpression())
	case ILogicalAndExpressionContext:
		return soleOperand(ctx.AllEqualityExpression())
	case IEqualityExpressionContext:
		return soleOperand(ctx.AllRelationalExpression())
	case IRelationalExpressionContext:
		return soleOperand(ctx.AllAdditiveExpression())
	case IAdditiveExpressionContext:
		return soleOperand(ctx.AllMultiplicativeExpression())
	case IMultiplicativeExpressionContext:
		return soleOperand(ctx.AllUnaryExpression())
	case IUnaryExpressionContext:
		if ctx.MINUS() != nil && ctx.UnaryExpression() != nil {
			return ctx.UnaryExpression()
		}
		if ctx.UnaryExpression() == nil {
			return ctx.PowerExpression()
		}
	case IPowerExpressionContext:
		if ctx.CARET() == nil {
			return ctx.PostfixExpression()
		}
	case IPostfixExpressionContext:
		if len(ctx.AllIndexOrSlice()) == 0 && len(ctx.AllFunctionCallSuffix()) == 0 {
			return ctx.PrimaryExpression()
		}
	}
	return nil
}

func soleOperand[T antlr.ParserRuleContext](operands []T) antlr.ParserRuleContext {
	if len(operands) == 1 {
		return operands[0]
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
		unaries := ctx.AllUnaryExpression()
		return len(unaries) == 1 && isNumericLiteral(unaries[0])
	case IUnaryExpressionContext:
		if ctx.MINUS() != nil {
			return isNumericLiteral(ctx.UnaryExpression())
		}
		return ctx.UnaryExpression() == nil && isNumericLiteral(ctx.PowerExpression())
	case IPowerExpressionContext:
		return ctx.CARET() == nil && isNumericLiteral(ctx.PostfixExpression())
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

// ExpressionName returns the name expr reads when expr is a lone name, and the
// zero Name otherwise. A call or an index is not a lone name.
func ExpressionName(expr IExpressionContext) Name {
	postfix := getPostfixExpression(expr)
	if postfix == nil || len(postfix.AllFunctionCallSuffix()) > 0 ||
		len(postfix.AllIndexOrSlice()) > 0 || postfix.PrimaryExpression() == nil {
		return Name{}
	}
	return PrimaryName(postfix.PrimaryExpression())
}

// CollectNamedPrimaries returns the primary expressions in expr that read a name,
// bare or qualified, one per name, in source order. A format string in fmtStrings
// gives the names that its placeholders read.
func CollectNamedPrimaries(
	expr IExpressionContext,
	fmtStrings FmtStrings,
) []IPrimaryExpressionContext {
	var (
		primaries []IPrimaryExpressionContext
		seen      = make(set.Set[Name])
		walk      func(t antlr.Tree)
	)
	walk = func(t antlr.Tree) {
		if term, ok := t.(antlr.TerminalNode); ok {
			for _, seg := range fmtStrings[term] {
				if seg.Expr != nil {
					walk(seg.Expr)
				}
			}
			return
		}
		if pe, ok := t.(IPrimaryExpressionContext); ok {
			if name := PrimaryName(pe); name.Head != "" && !seen.Contains(name) {
				seen.Add(name)
				primaries = append(primaries, pe)
			}
		}
		for i := range t.GetChildCount() {
			walk(t.GetChild(i))
		}
	}
	walk(expr)
	return primaries
}

// GetPrimaryExpression extracts the primary expression from an expression that has no
// operators. Returns nil if the expression contains any binary or unary operators.
func GetPrimaryExpression(expr IExpressionContext) IPrimaryExpressionContext {
	postfix := getPostfixExpression(expr)
	if postfix == nil {
		return nil
	}
	return postfix.PrimaryExpression()
}

// getPostfixExpression descends an operator-free expr to its postfix expression.
func getPostfixExpression(expr IExpressionContext) IPostfixExpressionContext {
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
	if len(mult.AllUnaryExpression()) != 1 {
		return nil
	}
	unary := mult.AllUnaryExpression()[0]
	pow := unary.PowerExpression()
	if pow == nil || pow.CARET() != nil {
		return nil
	}
	return pow.PostfixExpression()
}

// SoleOperand returns the primary expression node reduces to after removing
// parentheses, f32 and f64 casts, and unary minus, and whether the minus signs negate
// it. Returns nil if node contains any other operator.
func SoleOperand(node antlr.Tree) (IPrimaryExpressionContext, bool) {
	if primary, ok := node.(IPrimaryExpressionContext); ok {
		if primary.LPAREN() != nil && primary.Expression() != nil {
			return SoleOperand(primary.Expression())
		}
		if cast := primary.TypeCast(); cast != nil && isFloatType(cast.Type_()) {
			return SoleOperand(cast.Expression())
		}
		return primary, false
	}
	unary, isUnary := node.(IUnaryExpressionContext)
	if isUnary && unary.NOT() != nil {
		return nil, false
	}
	var sole antlr.Tree
	for _, child := range node.GetChildren() {
		if _, ok := child.(antlr.TerminalNode); ok {
			continue
		}
		if sole != nil {
			return nil, false
		}
		sole = child
	}
	if sole == nil {
		return nil, false
	}
	primary, negated := SoleOperand(sole)
	if primary != nil && isUnary && unary.MINUS() != nil {
		negated = !negated
	}
	return primary, negated
}

func isFloatType(t ITypeContext) bool {
	primitive := t.PrimitiveType()
	if primitive == nil || primitive.NumericType() == nil {
		return false
	}
	return primitive.NumericType().FloatType() != nil
}

// QualifiedNameParts returns the head and tail of a qualified identifier
// (e.g., "math" and "avg" for `math.avg`). Reads the terminal children
// directly so it works when either side is a lexer keyword (AUTHORITY on
// the left, FOR on the right) and not just an IDENTIFIER.
func QualifiedNameParts(qid IQualifiedIdentifierContext) (head, tail string) {
	return terminalText(qid, 0), terminalText(qid, 2)
}

func terminalText(qid IQualifiedIdentifierContext, i int) string {
	node, ok := qid.GetChild(i).(antlr.TerminalNode)
	if !ok {
		panic(fmt.Sprintf("qualified identifier child %d is a %T", i, qid.GetChild(i)))
	}
	return node.GetText()
}

// FunctionNameParts returns the head and tail of a function context's
// name. A bare function (set_authority()) returns (name, ""); a qualified
// one (math.avg()) returns (head, tail).
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

// Name is the name that a flow identifier, a primary expression, or an assignment
// holds.
type Name struct {
	// Head is the first segment, or the whole path of a channel name.
	Head string
	// Tail is the second segment of a qualified name, and empty otherwise.
	Tail string
	// Quoted is true for a quoted channel name, which always means a channel.
	Quoted bool
}

// String returns the joined name: head and tail with a dot between, or the path of a
// channel name. It returns "" for the zero Name.
func (n Name) String() string {
	if n.Tail == "" {
		return n.Head
	}
	return n.Head + "." + n.Tail
}

// Text returns the name as source text: a quoted channel name in quotes, and any other
// name as String does.
func (n Name) Text() string {
	if n.Quoted {
		return QuoteChannelName(n.Head)
	}
	return n.String()
}

// decodeChannelName returns the path that the text of a quoted channel name token
// holds, with its escapes decoded, in NFC.
func decodeChannelName(text string) string {
	text = text[1 : len(text)-1]
	var b strings.Builder
	b.Grow(len(text))
	for i := 0; i < len(text); i++ {
		// The lexer accepts a backslash only before a quote or a backslash.
		if text[i] == '\\' {
			i++
		}
		b.WriteByte(text[i])
	}
	return norm.NFC.String(b.String())
}

func channelName(token antlr.TerminalNode) Name {
	return Name{Head: decodeChannelName(token.GetText()), Quoted: true}
}

// TokenName returns the name that an IDENTIFIER or a CHANNEL_NAME token holds, and the
// zero Name for any other token.
func TokenName(token antlr.Token) Name {
	switch token.GetTokenType() {
	case ArcLexerIDENTIFIER:
		return Name{Head: token.GetText()}
	case ArcLexerCHANNEL_NAME:
		return Name{Head: decodeChannelName(token.GetText()), Quoted: true}
	}
	return Name{}
}

// QuoteChannelName returns the quoted channel name token that holds path. The path
// must not hold a line break, which no token can hold.
func QuoteChannelName(path string) string {
	var b strings.Builder
	b.Grow(len(path) + 2)
	b.WriteByte('\'')
	for i := range len(path) {
		if path[i] == '\'' || path[i] == '\\' {
			b.WriteByte('\\')
		}
		b.WriteByte(path[i])
	}
	b.WriteByte('\'')
	return b.String()
}

// IdentifierName returns the name of a flow identifier. A bare identifier returns
// its name as the head; a qualified one (m.f) returns the head and the tail.
func IdentifierName(id IIdentifierContext) Name {
	if qid := id.QualifiedIdentifier(); qid != nil {
		head, tail := QualifiedNameParts(qid)
		return Name{Head: head, Tail: tail}
	}
	if token := id.CHANNEL_NAME(); token != nil {
		return channelName(token)
	}
	return Name{Head: id.IDENTIFIER().GetText()}
}

// PrimaryName returns the name of a primary expression. Bare (x) returns the name
// as the head; qualified (math.avg) returns the head and the tail. It returns the zero
// Name when the primary is not a name.
func PrimaryName(primary IPrimaryExpressionContext) Name {
	if qid := primary.QualifiedIdentifier(); qid != nil {
		head, tail := QualifiedNameParts(qid)
		return Name{Head: head, Tail: tail}
	}
	if token := primary.CHANNEL_NAME(); token != nil {
		return channelName(token)
	}
	if id := primary.IDENTIFIER(); id != nil {
		return Name{Head: id.GetText()}
	}
	return Name{}
}

// AssignmentName returns the name of an assignment's target.
func AssignmentName(a IAssignmentContext) Name {
	if token := a.CHANNEL_NAME(); token != nil {
		return channelName(token)
	}
	return Name{Head: a.IDENTIFIER().GetText()}
}

// AuthorityEntryName returns the channel that an authority entry names, and the zero
// Name for the default entry.
func AuthorityEntryName(e IAuthorityEntryContext) Name {
	if token := e.CHANNEL_NAME(); token != nil {
		return channelName(token)
	}
	if id := e.IDENTIFIER(); id != nil {
		return Name{Head: id.GetText()}
	}
	return Name{}
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
	if _, name, ok := strings.CutLast(path, "."); ok {
		return name
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
