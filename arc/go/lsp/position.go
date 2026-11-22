// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp

import (
	"strings"
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/symbol"
	lsp "github.com/synnaxlabs/x/lsp"
	"go.lsp.dev/protocol"
)

type position struct {
	Line int
	Col  int
}

func fromProtocol(pos protocol.Position) position {
	return position{Line: int(pos.Line) + 1, Col: int(pos.Character)}
}

func findScopeAtInternalPosition(
	rootScope *symbol.Symbol,
	pos position,
) *symbol.Symbol {
	if rootScope == nil {
		return nil
	}
	var deepest *symbol.Symbol
	findScopeRecursive(rootScope, pos.Line, pos.Col, &deepest)
	if deepest == nil {
		return rootScope
	}
	return deepest
}

func findScopeRecursive(scope *symbol.Symbol, line, col int, deepest **symbol.Symbol) {
	if scope == nil {
		return
	}
	if scope.AST != nil && !scope.Internal {
		start := scope.AST.GetStart()
		stop := scope.AST.GetStop()
		if start != nil && stop != nil &&
			isPositionInTokenRange(line, col, start, stop) {
			*deepest = scope
		}
	}
	// An import's children are the module's members, positioned in the module's text.
	if scope.Kind == symbol.KindModuleAlias {
		return
	}
	for _, child := range scope.Children() {
		findScopeRecursive(child, line, col, deepest)
	}
}

func isPositionInTokenRange(line, col int, start, stop antlr.Token) bool {
	startLine := start.GetLine()
	startCol := start.GetColumn()
	stopLine := stop.GetLine()
	// Columns count runes, as the token's indexes do.
	stopCol := stop.GetColumn() + stop.GetStop() - stop.GetStart() + 1

	if line > startLine && line < stopLine {
		return true
	}
	if line == startLine && startLine == stopLine {
		return col >= startCol && col <= stopCol
	}
	if line == startLine {
		return col >= startCol
	}
	if line == stopLine {
		return col <= stopCol
	}
	return false
}

// lineText is the lines of a document. Token positions count code points, and protocol
// positions count UTF-16 units, so lineText converts between the two.
type lineText []string

func newLineText(content string) lineText {
	return strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
}

// toUTF16 returns pos, whose column counts code points, with a column in UTF-16 units.
func (l lineText) toUTF16(pos protocol.Position) protocol.Position {
	if int(pos.Line) < len(l) {
		return toUTF16In(l[pos.Line], pos)
	}
	return pos
}

// toUTF16In returns pos, whose column counts code points of line, the text of its
// line, with a column in UTF-16 units.
func toUTF16In(line string, pos protocol.Position) protocol.Position {
	pos.Character = lsp.UTF16Column(line, runeOffset(line, int(pos.Character)))
	return pos
}

// fromUTF16In returns pos, whose column counts UTF-16 units of line, the text of its
// line, with a column in code points.
func fromUTF16In(line string, pos protocol.Position) protocol.Position {
	pos.Character = uint32(
		utf8.RuneCountInString(line[:lsp.ByteOffset(line, pos.Character)]),
	)
	return pos
}

func (l lineText) rangeToUTF16(r protocol.Range) protocol.Range {
	return protocol.Range{Start: l.toUTF16(r.Start), End: l.toUTF16(r.End)}
}

// runeOffset returns the byte offset in line of the code point column col.
func runeOffset(line string, col int) int {
	for i := range line {
		if col == 0 {
			return i
		}
		col--
	}
	return len(line)
}
