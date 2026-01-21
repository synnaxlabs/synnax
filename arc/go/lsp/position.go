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
	"context"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/symbol"
	"go.lsp.dev/protocol"
)

type Position struct {
	Line int
	Col  int
}

func FromProtocol(pos protocol.Position) Position {
	return Position{Line: int(pos.Line) + 1, Col: int(pos.Character)}
}

func getLine(content string, line uint32) (string, bool) {
	lines := strings.Split(content, "\n")
	if int(line) >= len(lines) {
		return "", false
	}
	return lines[line], true
}

func getWordAtPosition(content string, pos protocol.Position) string {
	line, ok := getLine(content, pos.Line)
	if !ok || int(pos.Character) >= len(line) {
		return ""
	}
	start := int(pos.Character)
	end := int(pos.Character)
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}
	for end < len(line) && isWordChar(line[end]) {
		end++
	}
	return line[start:end]
}

func isWordChar(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_'
}

func resolveSymbolAtPosition(
	ctx context.Context,
	rootScope *symbol.Scope,
	content string,
	pos protocol.Position,
) (*symbol.Scope, error) {
	word := getWordAtPosition(content, pos)
	if word == "" {
		return nil, nil
	}
	scope := FindScopeAtPosition(rootScope, FromProtocol(pos))
	return scope.Resolve(ctx, word)
}

func getWordRangeAtPosition(content string, pos protocol.Position) *protocol.Range {
	word := getWordAtPosition(content, pos)
	if word == "" {
		return nil
	}
	line, ok := getLine(content, pos.Line)
	if !ok || int(pos.Character) >= len(line) {
		return nil
	}
	start := int(pos.Character)
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}
	return &protocol.Range{
		Start: protocol.Position{Line: pos.Line, Character: uint32(start)},
		End:   protocol.Position{Line: pos.Line, Character: uint32(start + len(word))},
	}
}

func FindScopeAtPosition(rootScope *symbol.Scope, pos Position) *symbol.Scope {
	if rootScope == nil {
		return nil
	}
	var deepest *symbol.Scope
	findScopeRecursive(rootScope, pos.Line, pos.Col, &deepest)
	if deepest == nil {
		return rootScope
	}
	return deepest
}

func findScopeRecursive(scope *symbol.Scope, line, col int, deepest **symbol.Scope) {
	if scope.AST != nil {
		start := scope.AST.GetStart()
		stop := scope.AST.GetStop()
		if start != nil && stop != nil && isPositionInTokenRange(line, col, start, stop) {
			*deepest = scope
		}
	}
	for _, child := range scope.Children {
		findScopeRecursive(child, line, col, deepest)
	}
}

func isPositionInTokenRange(line, col int, start, stop antlr.Token) bool {
	startLine := start.GetLine()
	startCol := start.GetColumn()
	stopLine := stop.GetLine()
	stopCol := stop.GetColumn() + len(stop.GetText())

	if line > startLine && line < stopLine {
		return true
	}
	if line == startLine && line == stopLine {
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
