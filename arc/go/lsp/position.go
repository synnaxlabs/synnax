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
