// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import "github.com/antlr4-go/antlr/v4"

// ScopeAt returns the deepest scope under root whose source holds the position at line,
// counted from 1, and col, counted in characters from 0, or root when none does. It
// skips the members of an imported module, whose source is the module's text.
func ScopeAt(root *Symbol, line, col int) *Symbol {
	if root == nil {
		return nil
	}
	var deepest *Symbol
	scopeAt(root, line, col, &deepest)
	if deepest == nil {
		return root
	}
	return deepest
}

func scopeAt(scope *Symbol, line, col int, deepest **Symbol) {
	if scope == nil {
		return
	}
	if scope.AST != nil && !scope.Internal {
		start := scope.AST.GetStart()
		stop := scope.AST.GetStop()
		if start != nil && stop != nil && inTokenRange(line, col, start, stop) {
			*deepest = scope
		}
	}
	// An import's children are the module's members, positioned in the module's text.
	if scope.Kind == KindModuleAlias {
		return
	}
	for _, child := range scope.Children() {
		scopeAt(child, line, col, deepest)
	}
}

func inTokenRange(line, col int, start, stop antlr.Token) bool {
	startLine := start.GetLine()
	startCol := start.GetColumn()
	stopLine := stop.GetLine()
	stopCol := stop.GetColumn() + len(stop.GetText())
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
