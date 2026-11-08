// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package formatter

import (
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/x/set"
)

// LineMove moves the lines in [Start, End) to just before line Anchor, zero-based.
type LineMove struct {
	// Start is the first line to move.
	Start int
	// End is the line after the last line to move.
	End int
	// Anchor is the line the moved lines go in front of.
	Anchor int
}

// DeclarationHoists returns the moves that put each late declaration of a sequence or
// stage body above its first step. A comment block directly above an item moves with
// it. Unparsable content yields none, and so does a body whose items share a line with
// another item or a brace.
func DeclarationHoists(content string) []LineMove {
	program, diags := parser.Parse(content)
	if program == nil || (diags != nil && !diags.Ok()) {
		return nil
	}
	comments := ownLineComments(content)
	var moves []LineMove
	walk(program, func(node antlr.Tree) {
		switch body := node.(type) {
		case parser.ISequenceDeclarationContext:
			moves = append(
				moves,
				bodyHoists(body, bodyItems(body.AllSequenceItem()), comments)...,
			)
		case parser.IStageBodyContext:
			moves = append(
				moves,
				bodyHoists(body, bodyItems(body.AllStageItem()), comments)...,
			)
		}
	})
	return moves
}

// HoistDeclarations applies DeclarationHoists to content.
func HoistDeclarations(content string) string {
	moves := DeclarationHoists(content)
	if len(moves) == 0 {
		return content
	}
	lines := SplitLines(content)
	moved := make(set.Set[int])
	byAnchor := make(map[int][]string)
	for _, m := range moves {
		for i := m.Start; i < m.End; i++ {
			moved.Add(i)
			byAnchor[m.Anchor] = append(byAnchor[m.Anchor], lines[i])
		}
	}
	out := make([]string, 0, len(lines))
	for i, line := range lines {
		out = append(out, byAnchor[i]...)
		if !moved.Contains(i) {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

type bodyItem struct {
	node antlr.ParserRuleContext
	decl bool
}

func bodyItems[I interface {
	antlr.ParserRuleContext
	VariableDeclaration() parser.IVariableDeclarationContext
}](items []I) []bodyItem {
	out := make([]bodyItem, len(items))
	for i, item := range items {
		out[i] = bodyItem{node: item, decl: item.VariableDeclaration() != nil}
	}
	return out
}

// bodyHoists returns one move per declaration that follows the body's first step.
func bodyHoists(
	body antlr.ParserRuleContext,
	items []bodyItem,
	comments map[int]int,
) []LineMove {
	if !onSeparateLines(body, items) {
		return nil
	}
	anchor := -1
	var moves []LineMove
	for _, item := range items {
		start := leadingLine(item.node.GetStart().GetLine()-1, comments)
		if !item.decl {
			if anchor < 0 {
				anchor = start
			}
			continue
		}
		if anchor < 0 {
			continue
		}
		moves = append(moves, LineMove{
			Start:  start,
			End:    item.node.GetStop().GetLine(),
			Anchor: anchor,
		})
	}
	return moves
}

// ownLineComments maps the last line of each comment with no code on its lines to the
// comment's first line, zero-based.
func ownLineComments(content string) map[int]int {
	lexer := parser.NewArcLexer(antlr.NewInputStream(content))
	starts := make(map[int]int)
	codeLine := -1
	for _, tok := range lexer.GetAllTokens() {
		line := tok.GetLine() - 1
		end := line + strings.Count(tok.GetText(), "\n")
		switch tok.GetTokenType() {
		case parser.ArcLexerWS:
		case parser.ArcLexerSINGLE_LINE_COMMENT, parser.ArcLexerMULTI_LINE_COMMENT:
			if line != codeLine {
				starts[end] = line
			}
		default:
			delete(starts, line)
			codeLine = end
		}
	}
	return starts
}

// leadingLine returns the first line of the comment block directly above line, or line
// when there is none.
func leadingLine(line int, comments map[int]int) int {
	for {
		start, ok := comments[line-1]
		if !ok {
			return line
		}
		line = start
	}
}

// onSeparateLines reports whether no item shares a line with another item or with
// the body's braces, so whole lines can move.
func onSeparateLines(body antlr.ParserRuleContext, items []bodyItem) bool {
	last := body.GetStart().GetLine()
	for _, item := range items {
		if item.node.GetStart().GetLine() <= last {
			return false
		}
		last = item.node.GetStop().GetLine()
	}
	return len(items) == 0 || last < body.GetStop().GetLine()
}

func walk(node antlr.Tree, visit func(antlr.Tree)) {
	visit(node)
	for _, child := range node.GetChildren() {
		walk(child, visit)
	}
}
