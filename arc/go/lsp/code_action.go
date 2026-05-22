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
	"fmt"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/lsp/protocol"
)

// CodeAction returns quick-fix actions for the diagnostics that intersect
// the requested range. Actions are matched by diagnostic code, so analyzer
// passes that want to expose a fix should tag their diagnostic with a
// stable codes.ErrorCode and add a case in the dispatch below.
func (s *Server) CodeAction(
	ctx context.Context,
	params *protocol.CodeActionParams,
) ([]protocol.CodeAction, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	var actions []protocol.CodeAction
	for _, diag := range params.Context.Diagnostics {
		code, _ := diag.Code.(string)
		switch code {
		case string(codes.UnusedImport):
			if action := s.unusedImportQuickFix(doc, diag); action != nil {
				actions = append(actions, *action)
			}
		case string(codes.DeprecatedSymbol):
			if action := s.deprecatedSymbolQuickFix(ctx, doc, diag); action != nil {
				actions = append(actions, *action)
			}
		}
	}
	return actions, nil
}

// unusedImportQuickFix builds a quick-fix action for an unused-import
// diagnostic. The fix deletes the entire import statement when the
// targeted item is the only one in its statement, and deletes just the
// item (with adjacent whitespace) otherwise.
func (s *Server) unusedImportQuickFix(
	doc *Document,
	diag protocol.Diagnostic,
) *protocol.CodeAction {
	alias := findAliasAtDiagnostic(doc, diag)
	if alias == nil {
		return nil
	}
	item, ok := alias.AST.(parser.IImportItemContext)
	if !ok {
		return nil
	}
	stmt, ok := item.GetParent().(parser.IImportStatementContext)
	if !ok {
		return nil
	}
	edit := buildUnusedImportEdit(doc.Content, stmt, item)
	return &protocol.CodeAction{
		Title:       "Remove unused import",
		Kind:        protocol.QuickFix,
		Diagnostics: []protocol.Diagnostic{diag},
		IsPreferred: true,
		Edit: &protocol.WorkspaceEdit{
			Changes: map[protocol.DocumentURI][]protocol.TextEdit{
				doc.URI: {edit},
			},
		},
	}
}

// findAliasAtDiagnostic returns the KindModuleAlias child of the document
// root scope whose AST starts at the diagnostic range's start, or nil if
// the document was not analyzed or no alias matches. Matching by start
// position keeps the resolution robust even when a parse error has
// truncated the symbol tree.
func findAliasAtDiagnostic(doc *Document, diag protocol.Diagnostic) *symbol.Symbol {
	if doc.IR.Symbols == nil {
		return nil
	}
	for _, child := range doc.IR.Symbols.Children() {
		if child.Kind != symbol.KindModuleAlias || child.AST == nil {
			continue
		}
		start := child.AST.GetStart()
		if start == nil {
			continue
		}
		if uint32(start.GetLine()-1) != diag.Range.Start.Line {
			continue
		}
		if uint32(start.GetColumn()) != diag.Range.Start.Character {
			continue
		}
		return child
	}
	return nil
}

// buildUnusedImportEdit computes the TextEdit that removes the unused
// item. When item is the sole member of its statement the whole statement
// is deleted (extending through the trailing newline so no blank line is
// left behind). Otherwise the item is deleted along with one neighboring
// run of whitespace, preserving the surrounding parenthesized block.
func buildUnusedImportEdit(
	content string,
	stmt parser.IImportStatementContext,
	item parser.IImportItemContext,
) protocol.TextEdit {
	items := stmt.AllImportItem()
	if len(items) <= 1 {
		return deleteStatementEdit(content, stmt)
	}
	return deleteItemEdit(items, item)
}

// deleteStatementEdit deletes the whole import statement plus a trailing
// newline when one is present, so removing the only import in a file
// leaves no orphan blank line.
func deleteStatementEdit(
	content string,
	stmt parser.IImportStatementContext,
) protocol.TextEdit {
	start := stmt.GetStart()
	stop := stmt.GetStop()
	startLine := start.GetLine() - 1
	startChar := start.GetColumn()
	endLine, endChar := tokenEndLineChar(stop)
	if eatsTrailingNewline(content, endLine, endChar) {
		endLine++
		endChar = 0
	}
	return protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(startLine),
				Character: uint32(startChar),
			},
			End: protocol.Position{
				Line:      uint32(endLine),
				Character: uint32(endChar),
			},
		},
	}
}

// deleteItemEdit deletes a single item from a multi-item parenthesized
// import block. When the item has a following sibling the deletion eats
// the whitespace between them; for the last item the deletion eats the
// whitespace after the previous sibling instead.
func deleteItemEdit(
	items []parser.IImportItemContext,
	item parser.IImportItemContext,
) protocol.TextEdit {
	idx := indexOfItem(items, item)
	startLine, startChar := tokenStartLineChar(item.GetStart())
	endLine, endChar := tokenEndLineChar(item.GetStop())
	if idx >= 0 && idx < len(items)-1 {
		nextStartLine, nextStartChar := tokenStartLineChar(items[idx+1].GetStart())
		endLine, endChar = nextStartLine, nextStartChar
	} else if idx > 0 {
		prevEndLine, prevEndChar := tokenEndLineChar(items[idx-1].GetStop())
		startLine, startChar = prevEndLine, prevEndChar
	}
	return protocol.TextEdit{
		Range: protocol.Range{
			Start: protocol.Position{
				Line:      uint32(startLine),
				Character: uint32(startChar),
			},
			End: protocol.Position{
				Line:      uint32(endLine),
				Character: uint32(endChar),
			},
		},
	}
}

func indexOfItem(items []parser.IImportItemContext, target parser.IImportItemContext) int {
	for i, it := range items {
		if it == target {
			return i
		}
	}
	return -1
}

func tokenStartLineChar(t antlr.Token) (int, int) {
	return t.GetLine() - 1, t.GetColumn()
}

func tokenEndLineChar(t antlr.Token) (int, int) {
	return t.GetLine() - 1, t.GetColumn() + len(t.GetText())
}

// eatsTrailingNewline reports whether the character at (line, char) in
// content is a newline. Both \n and \r\n collapse to a single LSP newline
// because the End position sits before the \n in either case.
func eatsTrailingNewline(content string, line, char int) bool {
	offset := offsetAt(content, line, char)
	return offset >= 0 && offset < len(content) && content[offset] == '\n'
}

func offsetAt(content string, line, char int) int {
	curLine, curChar, offset := 0, 0, 0
	for offset < len(content) {
		if curLine == line && curChar == char {
			return offset
		}
		if content[offset] == '\n' {
			if curLine == line {
				return offset
			}
			curLine++
			curChar = 0
		} else {
			curChar++
		}
		offset++
	}
	if curLine == line && curChar == char {
		return offset
	}
	return -1
}

// deprecatedSymbolQuickFix builds a quick-fix action for a deprecated
// reference. It finds the bare identifier inside the diagnostic range,
// resolves it through the document's scope, and emits a text edit that
// replaces the identifier with the canonical replacement. When the
// replacement's module is not already in scope, an additional import edit
// is appended via buildAutoImportEdit. Qualified references like
// `module.member` are skipped in this version; users see the warning but
// not a fix.
func (s *Server) deprecatedSymbolQuickFix(
	ctx context.Context,
	doc *Document,
	diag protocol.Diagnostic,
) *protocol.CodeAction {
	if doc.IR.Symbols == nil {
		return nil
	}
	token, sym := findDeprecatedBareRef(ctx, doc, diag.Range)
	if token == nil || sym == nil || sym.Deprecated == nil {
		return nil
	}
	repl := sym.Deprecated
	replacementName := buildReplacementName(doc, repl)
	tokenStartLine, tokenStartChar := tokenStartLineChar(token)
	tokenEndLine, tokenEndChar := tokenEndLineChar(token)
	edits := []protocol.TextEdit{{
		Range: protocol.Range{
			Start: protocol.Position{Line: uint32(tokenStartLine), Character: uint32(tokenStartChar)},
			End:   protocol.Position{Line: uint32(tokenEndLine), Character: uint32(tokenEndChar)},
		},
		NewText: replacementName,
	}}
	if module := replacementModule(repl); module != "" {
		edits = append(buildAutoImportEdit(doc, module), edits...)
	}
	return &protocol.CodeAction{
		Title:       fmt.Sprintf("Replace '%s' with '%s'", token.GetText(), replacementName),
		Kind:        protocol.QuickFix,
		Diagnostics: []protocol.Diagnostic{diag},
		IsPreferred: true,
		Edit: &protocol.WorkspaceEdit{
			Changes: map[protocol.DocumentURI][]protocol.TextEdit{doc.URI: edits},
		},
	}
}

// findDeprecatedBareRef scans IDENTIFIER tokens that fall inside r and
// returns the first one that resolves to a deprecated symbol when looked
// up in the document's scope at that position. Identifiers preceded or
// followed by a `.` are skipped — they belong to a qualified path, which
// this version does not rewrite.
func findDeprecatedBareRef(
	ctx context.Context,
	doc *Document,
	r protocol.Range,
) (antlr.Token, *symbol.Symbol) {
	tokens := tokenizeContent(doc.Content)
	for i, t := range tokens {
		if t.GetTokenType() != parser.ArcLexerIDENTIFIER {
			continue
		}
		if !tokenInRange(t, r) {
			continue
		}
		if isPartOfQualifiedPath(tokens, i) {
			continue
		}
		pos := protocol.Position{
			Line:      uint32(t.GetLine() - 1),
			Character: uint32(t.GetColumn()),
		}
		scope := doc.findScopeAtPosition(pos)
		if scope == nil {
			continue
		}
		sym, err := scope.Resolve(ctx, t.GetText())
		if err != nil || sym == nil || sym.Deprecated == nil {
			continue
		}
		return t, sym
	}
	return nil, nil
}

// tokenInRange reports whether t's start position falls within the
// half-open LSP range r. We anchor on the start because the diagnostic's
// own range may extend past the identifier (e.g., onto trailing call
// parentheses) and we only care that the token begins inside.
func tokenInRange(t antlr.Token, r protocol.Range) bool {
	line := uint32(t.GetLine() - 1)
	char := uint32(t.GetColumn())
	if line < r.Start.Line || line > r.End.Line {
		return false
	}
	if line == r.Start.Line && char < r.Start.Character {
		return false
	}
	if line == r.End.Line && char > r.End.Character {
		return false
	}
	return true
}

// isPartOfQualifiedPath reports whether the identifier at idx is adjacent
// to a `.` token on either side, ignoring intervening whitespace and
// comments. Such identifiers belong to a `head.tail` reference where one
// segment is a module qualifier; rewriting them requires more context
// than this version provides.
func isPartOfQualifiedPath(tokens []antlr.Token, idx int) bool {
	for j := idx - 1; j >= 0; j-- {
		if isTriviaToken(tokens[j]) {
			continue
		}
		if tokens[j].GetTokenType() == parser.ArcLexerDOT {
			return true
		}
		break
	}
	for j := idx + 1; j < len(tokens); j++ {
		if isTriviaToken(tokens[j]) {
			continue
		}
		if tokens[j].GetTokenType() == parser.ArcLexerDOT {
			return true
		}
		break
	}
	return false
}

func isTriviaToken(t antlr.Token) bool {
	switch t.GetTokenType() {
	case parser.ArcLexerWS,
		parser.ArcLexerSINGLE_LINE_COMMENT,
		parser.ArcLexerMULTI_LINE_COMMENT:
		return true
	}
	return false
}

// buildReplacementName returns the dotted reference the user should write
// for repl, preferring an existing module alias when one is in scope and
// falling back to repl's canonical QualifiedName otherwise.
func buildReplacementName(doc *Document, repl *symbol.Symbol) string {
	if repl.Parent == nil || repl.Parent.Kind != symbol.KindModule {
		return repl.QualifiedName()
	}
	for _, child := range doc.IR.Symbols.Children() {
		if child.Kind != symbol.KindModuleAlias || child.Target == nil {
			continue
		}
		if child.Target == repl.Parent || child.Target.Name == repl.Parent.Name {
			return child.Name + "." + repl.Name
		}
	}
	return repl.QualifiedName()
}

// replacementModule returns the module name that needs to be imported for
// repl to resolve. Empty when repl is not a module member (no import is
// needed), which short-circuits the auto-import edit.
func replacementModule(repl *symbol.Symbol) string {
	if repl.Parent == nil || repl.Parent.Kind != symbol.KindModule {
		return ""
	}
	return repl.Parent.Name
}
