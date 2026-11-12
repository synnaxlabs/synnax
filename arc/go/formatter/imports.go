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
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/set"
	"go.lsp.dev/protocol"
)

// AddImport returns the edits that import module into the file of root, or nil when
// the file already imports it.
func AddImport(root *symbol.Symbol, module string) []protocol.TextEdit {
	var (
		names []string
		stmts []antlr.ParserRuleContext
		seen  = set.New[antlr.ParserRuleContext]()
	)
	if root != nil {
		for _, child := range root.Children() {
			if child.Kind != symbol.KindModuleAlias || child.AST == nil {
				continue
			}
			if child.Name == module {
				return nil
			}
			item, ok := child.AST.(parser.IImportItemContext)
			if !ok {
				continue
			}
			name := item.ImportPath().GetText()
			if name == module {
				return nil
			}
			if item.AS() != nil {
				name += " as " + item.IDENTIFIER().GetText()
			}
			names = append(names, name)
			stmt := item.GetParent().(antlr.ParserRuleContext)
			if !seen.Contains(stmt) {
				seen.Add(stmt)
				stmts = append(stmts, stmt)
			}
		}
	}
	if len(stmts) == 0 {
		return []protocol.TextEdit{{NewText: "import " + module + "\n\n"}}
	}
	slices.SortFunc(stmts, func(a, b antlr.ParserRuleContext) int {
		return a.GetStart().GetStart() - b.GetStart().GetStart()
	})
	var block strings.Builder
	block.WriteString("import (\n")
	for _, name := range append(names, module) {
		block.WriteString("    " + name + "\n")
	}
	block.WriteString(")\n")
	// Each later statement is deleted on its own, so the comments between them stay.
	edits := []protocol.TextEdit{{
		Range: protocol.Range{
			Start: startOf(stmts[0].GetStart()),
			End:   endOf(stmts[0].GetStop()),
		},
		NewText: block.String(),
	}}
	for _, stmt := range stmts[1:] {
		edits = append(edits, removeStatement(stmt))
	}
	return edits
}

// RemoveImport returns the edit that removes item, or its statement when it is alone.
func RemoveImport(item parser.IImportItemContext) protocol.TextEdit {
	stmt := item.GetParent().(parser.IImportStatementContext)
	items := stmt.AllImportItem()
	if len(items) == 1 {
		return removeStatement(stmt)
	}
	i := slices.Index(items, item)
	r := protocol.Range{Start: startOf(item.GetStart()), End: endOf(item.GetStop())}
	if i < len(items)-1 {
		r.End = startOf(items[i+1].GetStart())
	} else {
		r.Start = endOf(items[i-1].GetStop())
	}
	return protocol.TextEdit{Range: r}
}

// removeStatement returns the edit that removes stmt and the newline after it.
func removeStatement(stmt antlr.ParserRuleContext) protocol.TextEdit {
	stop := stmt.GetStop()
	end := endOf(stop)
	if stop.GetInputStream().GetText(stop.GetStop()+1, stop.GetStop()+1) == "\n" {
		end = protocol.Position{Line: end.Line + 1}
	}
	return protocol.TextEdit{
		Range: protocol.Range{Start: startOf(stmt.GetStart()), End: end},
	}
}

func startOf(t antlr.Token) protocol.Position {
	return protocol.Position{
		Line:      uint32(t.GetLine() - 1),
		Character: uint32(t.GetColumn()),
	}
}

func endOf(t antlr.Token) protocol.Position {
	return protocol.Position{
		Line:      uint32(t.GetLine() - 1),
		Character: uint32(t.GetColumn() + len(t.GetText())),
	}
}
