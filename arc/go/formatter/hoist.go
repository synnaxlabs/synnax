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
	Start, End, Anchor int
}

// DeclarationHoists returns the moves that put each late declaration of a sequence
// or stage body above its first step. Unparsable content yields none.
func DeclarationHoists(content string) []LineMove {
	program, diags := parser.Parse(content)
	if program == nil || (diags != nil && !diags.Ok()) {
		return nil
	}
	var moves []LineMove
	walk(program, func(node antlr.Tree) {
		switch body := node.(type) {
		case parser.ISequenceDeclarationContext:
			moves = append(moves, bodyHoists(bodyItems(body.AllSequenceItem()))...)
		case parser.IStageBodyContext:
			moves = append(moves, bodyHoists(bodyItems(body.AllStageItem()))...)
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
func bodyHoists(items []bodyItem) []LineMove {
	anchor := -1
	var moves []LineMove
	for _, item := range items {
		if !item.decl {
			if anchor < 0 {
				anchor = item.node.GetStart().GetLine() - 1
			}
			continue
		}
		if anchor < 0 {
			continue
		}
		moves = append(moves, LineMove{
			Start:  item.node.GetStart().GetLine() - 1,
			End:    item.node.GetStop().GetLine(),
			Anchor: anchor,
		})
	}
	return moves
}

func walk(node antlr.Tree, visit func(antlr.Tree)) {
	visit(node)
	for _, child := range node.GetChildren() {
		walk(child, visit)
	}
}
