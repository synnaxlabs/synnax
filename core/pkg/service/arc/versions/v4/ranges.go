// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4

import (
	"context"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/stl"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	"go.lsp.dev/protocol"
)

// CastRanges rewrites each range() call whose typed integer arguments differ. Every
// typed argument not of the widest type is cast to it, signed when any argument is
// signed. Untyped number arguments and the rest of src are kept. A call with an
// argument whose type does not resolve is left unchanged, and so is a src that does not
// parse. cfgs sets how names lex, as in parser.Parse.
func CastRanges(ctx context.Context, src string, cfgs ...parser.Config) string {
	// A cast can widen a loop variable that a later range() reads, so each pass casts
	// one call and the next pass analyzes the result.
	for {
		next := castRanges(ctx, src, cfgs)
		if next == src {
			return src
		}
		src = next
	}
}

func castRanges(ctx context.Context, src string, cfgs []parser.Config) string {
	t, diag := text.Parse(text.Text{Raw: src}, cfgs...)
	if diag != nil {
		return src
	}
	// A stored program can name channels and modules no root here resolves, so only
	// the arguments that resolved are typed.
	program, diag := text.Analyze(
		ctx,
		t,
		symbol.NewRoot(nil, stl.NewSymbols()),
		cfgs...,
	)
	c := rangeCaster{types: program.TypeMap, diag: diag, insert: map[int]string{}}
	antlr.ParseTreeWalkerDefault.Walk(&c, t.AST)
	if len(c.insert) == 0 {
		return src
	}
	var b strings.Builder
	runes := []rune(src)
	for i, r := range runes {
		b.WriteString(c.insert[i])
		b.WriteRune(r)
	}
	b.WriteString(c.insert[len(runes)])
	return b.String()
}

type rangeCaster struct {
	parser.BaseArcParserListener
	types map[antlr.ParserRuleContext]types.Type
	diag  *diagnostics.Diagnostics
	// insert holds the text to write before the rune at each offset.
	insert map[int]string
}

func (c *rangeCaster) EnterForStatement(stmt *parser.ForStatementContext) {
	clause := stmt.ForClause()
	if len(c.insert) > 0 || clause == nil || !c.rejected(stmt) {
		return
	}
	args := rangeArgs(clause.Expression())
	var typed []parser.IExpressionContext
	var argTypes []types.Type
	for _, arg := range args {
		if untyped(arg) {
			continue
		}
		t, ok := c.types[additive(arg)]
		if !ok || !t.IsInteger() {
			return
		}
		typed = append(typed, arg)
		argTypes = append(argTypes, t)
	}
	target := widest(argTypes)
	for i, arg := range typed {
		if argTypes[i].Kind == target.Kind {
			continue
		}
		c.insert[arg.GetStart().GetStart()] = target.String() + "("
		c.insert[arg.GetStop().GetStop()+1] = ")"
	}
}

// rejected reports whether the analyzer rejected the range() of stmt for mixing integer
// types.
func (c *rangeCaster) rejected(stmt *parser.ForStatementContext) bool {
	tok := stmt.GetStart()
	at := protocol.Position{
		Line:      uint32(tok.GetLine() - 1),
		Character: uint32(tok.GetColumn()),
	}
	for _, i := range c.diag.AtLocation(at) {
		if strings.HasSuffix((*c.diag)[i].Message, "in range()") {
			return true
		}
	}
	return false
}

// rangeArgs returns the arguments of expr when it is a range() call.
func rangeArgs(expr parser.IExpressionContext) []parser.IExpressionContext {
	primary := parser.GetPrimaryExpression(expr)
	if primary == nil || primary.IDENTIFIER() == nil ||
		primary.IDENTIFIER().GetText() != "range" {
		return nil
	}
	postfix, ok := primary.GetParent().(parser.IPostfixExpressionContext)
	if !ok {
		return nil
	}
	calls := postfix.AllFunctionCallSuffix()
	if len(calls) != 1 || calls[0].ArgumentList() == nil {
		return nil
	}
	return calls[0].ArgumentList().AllExpression()
}

// additive returns the node the analyzer records the type of arg on, or nil when arg
// is not a lone additive expression.
func additive(arg parser.IExpressionContext) antlr.ParserRuleContext {
	or := arg.LogicalOrExpression()
	if or == nil || len(or.AllLogicalAndExpression()) != 1 {
		return nil
	}
	and := or.AllLogicalAndExpression()[0]
	if len(and.AllEqualityExpression()) != 1 {
		return nil
	}
	eq := and.AllEqualityExpression()[0]
	if len(eq.AllRelationalExpression()) != 1 {
		return nil
	}
	rel := eq.AllRelationalExpression()[0]
	if len(rel.AllAdditiveExpression()) != 1 {
		return nil
	}
	return rel.AllAdditiveExpression()[0]
}

// untyped reports whether arg is built only from number literals and arithmetic, so
// it takes the type of the typed arguments.
func untyped(arg parser.IExpressionContext) bool {
	for _, tok := range terminals(arg) {
		switch tok.GetTokenType() {
		case parser.ArcLexerINTEGER_LITERAL, parser.ArcLexerFLOAT_LITERAL,
			parser.ArcLexerPLUS, parser.ArcLexerMINUS, parser.ArcLexerSTAR,
			parser.ArcLexerSLASH, parser.ArcLexerPERCENT, parser.ArcLexerCARET,
			parser.ArcLexerLPAREN, parser.ArcLexerRPAREN:
		default:
			return false
		}
	}
	return true
}

func terminals(tree antlr.Tree) []antlr.Token {
	if t, ok := tree.(antlr.TerminalNode); ok {
		return []antlr.Token{t.GetSymbol()}
	}
	var tokens []antlr.Token
	for _, child := range tree.GetChildren() {
		tokens = append(tokens, terminals(child)...)
	}
	return tokens
}

// widest returns the widest integer type of ts, signed when any of ts is signed.
func widest(ts []types.Type) types.Type {
	var (
		bytes  int
		signed bool
	)
	for _, t := range ts {
		signed = signed || t.IsSignedInteger()
		bytes = max(bytes, t.Density())
	}
	return integerType(bytes, signed)
}

func integerType(bytes int, signed bool) types.Type {
	switch {
	case bytes <= 1 && signed:
		return types.I8()
	case bytes <= 1:
		return types.U8()
	case bytes <= 2 && signed:
		return types.I16()
	case bytes <= 2:
		return types.U16()
	case bytes <= 4 && signed:
		return types.I32()
	case bytes <= 4:
		return types.U32()
	case signed:
		return types.I64()
	}
	return types.U64()
}
