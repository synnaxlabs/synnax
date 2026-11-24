// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/set"
)

// RenameChannel returns source with each token that names the channel from written as
// a token of the channel to, and false when a raw string cannot hold that token. block
// reads source as the body of a function, as a calculation's expression is. A token
// counts only where a channel can be named (ChannelSites), in code or in a placeholder
// of a format string. A quoted token stays quoted, and a bare one stays bare when
// parser.ChannelText allows it. A bare token does not count when source declares the
// name from, which the token then names. Source that does not parse names no channel.
func RenameChannel(
	source string,
	block bool,
	from, to string,
	cfg parser.Config,
) (string, bool) {
	sites := ChannelSites(source, block, cfg)
	if len(sites) == 0 {
		return source, true
	}
	declared := declaredNames(source, block, cfg)
	token := func(n parser.Name) (string, bool) {
		switch {
		case n.Head != from || (!n.Quoted && declared.Contains(from)):
			return "", false
		case n.Quoted:
			return parser.QuoteChannelName(to), true
		}
		return parser.ChannelText(to, declared.Contains), true
	}
	at := byteOffsets(source)
	var edits []edit
	for _, t := range tokenize(source, cfg) {
		switch t.GetTokenType() {
		case parser.ArcLexerSTR_LITERAL, parser.ArcLexerSTR_LITERAL_MULTI:
			inner, ok := placeholderEdits(t, at[t.GetStart()], cfg, token)
			if !ok {
				return source, false
			}
			edits = append(edits, inner...)
		default:
			if !sites.Contains(t.GetStart()) {
				continue
			}
			if text, ok := token(parser.TokenName(t)); ok {
				edits = append(edits, edit{
					start: at[t.GetStart()],
					end:   at[t.GetStop()+1],
					text:  text,
				})
			}
		}
	}
	if len(edits) == 0 {
		return source, true
	}
	var b strings.Builder
	last := 0
	for _, e := range edits {
		b.WriteString(source[last:e.start])
		b.WriteString(e.text)
		last = e.end
	}
	b.WriteString(source[last:])
	return b.String(), true
}

// edit replaces the bytes [start, end) of a source with text.
type edit struct {
	start, end int
	text       string
}

// placeholderEdits returns the edits of the tokens that token renames in the
// placeholders of t, a string literal at byte start, and false when the literal cannot
// hold one. A literal with no placeholder has none.
func placeholderEdits(
	t antlr.Token,
	start int,
	cfg parser.Config,
	token func(parser.Name) (string, bool),
) ([]edit, bool) {
	source := t.GetText()
	prefix := 0
	for prefix < 2 && prefix < len(source) &&
		(source[prefix] == 'r' || source[prefix] == 'f') {
		prefix++
	}
	body, flags, ok := literal.StripQuotes(source)
	if !ok || !flags.Format {
		return nil, true
	}
	segs, err := literal.FmtStrSplit(body, flags)
	if err != nil {
		return nil, true
	}
	bodyStart := start + prefix + 1
	var edits []edit
	for _, seg := range segs {
		if !seg.IsPlaceholder {
			continue
		}
		sites := ExpressionSites(seg.Text, cfg)
		at := byteOffsets(seg.Text)
		for _, it := range tokenize(seg.Text, cfg) {
			if !sites.Contains(it.GetStart()) {
				continue
			}
			text, ok := token(parser.TokenName(it))
			if !ok {
				continue
			}
			escaped, fits := literal.Escape(text, flags)
			if !fits {
				return nil, false
			}
			edits = append(edits, edit{
				start: bodyStart + seg.Offsets[at[it.GetStart()]],
				end:   bodyStart + seg.Offsets[at[it.GetStop()+1]],
				text:  escaped,
			})
		}
	}
	return edits, true
}

// tokenize returns the tokens of source, comments left out.
func tokenize(source string, cfg parser.Config) []antlr.Token {
	lexer := parser.NewLexer(source, cfg)
	lexer.RemoveErrorListeners()
	return lexer.GetAllTokens()
}

// byteOffsets returns the byte offset of each code point of s, and of its end, since
// tokens count code points.
func byteOffsets(s string) []int {
	at := make([]int, 0, len(s)+1)
	for i := range s {
		at = append(at, i)
	}
	return append(at, len(s))
}

// declaredNames returns each name that source declares: a function, stage, sequence,
// input, output, variable, loop variable, or import alias. block reads source as the
// body of a function. Source that does not parse declares nothing.
func declaredNames(source string, block bool, cfg parser.Config) set.Set[string] {
	var tree antlr.ParseTree
	if block {
		parsed, diags := parser.ParseBlock("{"+source+"}", cfg)
		if diags != nil {
			return nil
		}
		tree = parsed
	} else {
		parsed, diags := parser.Parse(source, cfg)
		if diags != nil {
			return nil
		}
		tree = parsed
	}
	l := &declarationListener{names: make(set.Set[string])}
	antlr.ParseTreeWalkerDefault.Walk(l, tree)
	return l.names
}

type declarationListener struct {
	parser.BaseArcParserListener
	names set.Set[string]
}

func (l *declarationListener) add(nodes ...antlr.TerminalNode) {
	for _, n := range nodes {
		if n != nil {
			l.names.Add(n.GetText())
		}
	}
}

func (l *declarationListener) EnterFunctionDeclaration(
	c *parser.FunctionDeclarationContext,
) {
	l.add(c.IDENTIFIER())
}

func (l *declarationListener) EnterSequenceDeclaration(
	c *parser.SequenceDeclarationContext,
) {
	l.add(c.IDENTIFIER())
}

func (l *declarationListener) EnterStageDeclaration(c *parser.StageDeclarationContext) {
	l.add(c.IDENTIFIER())
}

func (l *declarationListener) EnterInput(c *parser.InputContext) {
	l.add(c.IDENTIFIER())
}

func (l *declarationListener) EnterNamedOutput(c *parser.NamedOutputContext) {
	l.add(c.IDENTIFIER())
}

func (l *declarationListener) EnterLocalVariable(c *parser.LocalVariableContext) {
	l.add(c.IDENTIFIER())
}

func (l *declarationListener) EnterStatefulVariable(c *parser.StatefulVariableContext) {
	l.add(c.IDENTIFIER())
}

func (l *declarationListener) EnterForClause(c *parser.ForClauseContext) {
	l.add(c.AllIDENTIFIER()...)
}

func (l *declarationListener) EnterImportItem(c *parser.ImportItemContext) {
	l.add(c.IDENTIFIER())
}
