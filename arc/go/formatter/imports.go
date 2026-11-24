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
	"fmt"
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
	names, stmts, imported := moduleImports(root, module)
	if imported {
		return nil
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
		edits = append(edits, RemoveStatement(stmt))
	}
	return edits
}

// moduleImports returns the import names and the import statements of the file of
// root. It returns true when the file already imports module.
func moduleImports(
	root *symbol.Symbol,
	module string,
) ([]string, []antlr.ParserRuleContext, bool) {
	if root == nil {
		return nil, nil, false
	}
	var (
		names []string
		stmts []antlr.ParserRuleContext
		seen  = set.New[antlr.ParserRuleContext]()
	)
	for _, child := range root.Children() {
		if child.Kind != symbol.KindModuleAlias || child.AST == nil {
			continue
		}
		if child.Name == module {
			return nil, nil, true
		}
		item, ok := child.AST.(parser.IImportItemContext)
		if !ok {
			continue
		}
		name := item.ImportPath().GetText()
		if name == module {
			return nil, nil, true
		}
		if item.AS() != nil {
			name += " as " + item.IDENTIFIER().GetText()
		}
		names = append(names, name)
		stmt := statementOf(item)
		if !seen.Contains(stmt) {
			seen.Add(stmt)
			stmts = append(stmts, stmt)
		}
	}
	return names, stmts, false
}

// RemoveImport returns the edit that removes item, or its statement when it is alone.
func RemoveImport(item parser.IImportItemContext) protocol.TextEdit {
	stmt := statementOf(item)
	items := stmt.AllImportItem()
	if len(items) == 1 {
		return RemoveStatement(stmt)
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

// RemoveStatement returns the edit that removes stmt and the newline after it. It
// removes the indent too when stmt is alone on its line.
func RemoveStatement(stmt antlr.ParserRuleContext) protocol.TextEdit {
	var (
		first, last = stmt.GetStart(), stmt.GetStop()
		input       = first.GetInputStream()
		r           = protocol.Range{Start: startOf(first), End: endOf(last)}
	)
	if input.GetText(last.GetStop()+1, last.GetStop()+1) != "\n" {
		return protocol.TextEdit{Range: r}
	}
	r.End = protocol.Position{Line: r.End.Line + 1}
	indent := input.GetText(first.GetStart()-first.GetColumn(), first.GetStart()-1)
	if strings.TrimSpace(indent) == "" {
		r.Start.Character = 0
	}
	return protocol.TextEdit{Range: r}
}

// Replace returns the edit that changes the code at node to text.
func Replace(node antlr.ParserRuleContext, text string) protocol.TextEdit {
	return protocol.TextEdit{
		Range: protocol.Range{
			Start: startOf(node.GetStart()),
			End:   endOf(node.GetStop()),
		},
		NewText: text,
	}
}

// RemoveToken returns the edit that removes t and the white space after it.
func RemoveToken(t antlr.Token) protocol.TextEdit {
	end := endOf(t)
	for i := t.GetStop() + 1; ; i++ {
		switch t.GetInputStream().GetText(i, i) {
		case " ", "\t", "\r":
			end.Character++
		case "\n":
			end = protocol.Position{Line: end.Line + 1}
		default:
			return protocol.TextEdit{Range: protocol.Range{Start: startOf(t), End: end}}
		}
	}
}

// ReplaceToken returns the edit that changes t to text.
func ReplaceToken(t antlr.Token, text string) protocol.TextEdit {
	return protocol.TextEdit{
		Range:   protocol.Range{Start: startOf(t), End: endOf(t)},
		NewText: text,
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

// statementOf returns the import statement that holds item.
func statementOf(item parser.IImportItemContext) parser.IImportStatementContext {
	stmt, ok := item.GetParent().(parser.IImportStatementContext)
	if !ok {
		panic(fmt.Sprintf("import item parent is a %T", item.GetParent()))
	}
	return stmt
}
