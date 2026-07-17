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
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	lsp "github.com/synnaxlabs/x/lsp"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// codeActionSnapshot bundles the document state CodeAction needs after
// releasing s.mu. Snapshotting once at the top of CodeAction avoids racing
// with DidChange and runAnalysis, which write doc.Content and doc.IR under
// s.mu.Lock.
type codeActionSnapshot struct {
	URI     uri.URI
	Content string
	Symbols *symbol.Symbol
	Tokens  []antlr.Token
	IsBlock bool
}

// CodeAction returns quick-fix actions for the diagnostics that intersect
// the requested range. Actions are matched by diagnostic code, so analyzer
// passes that want to expose a fix should tag their diagnostic with a
// stable codes.ErrorCode and add a case in the dispatch below.
func (s *Server) CodeAction(
	ctx context.Context,
	params *protocol.CodeActionParams,
) ([]protocol.CommandOrCodeAction, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok {
		return nil, nil
	}
	s.mu.RLock()
	content := doc.Content
	symbols := doc.IR.Symbols
	isBlock := doc.isBlock()
	s.mu.RUnlock()
	snap := codeActionSnapshot{
		URI:     doc.URI,
		Content: content,
		Symbols: symbols,
		Tokens:  tokenizeContent(content),
		IsBlock: isBlock,
	}
	var actions []protocol.CommandOrCodeAction
	for _, diag := range params.Context.Diagnostics {
		code, _ := diag.Code.(protocol.String)
		switch string(code) {
		case string(codes.UnusedImport):
			if action := unusedImportQuickFix(snap, diag); action != nil {
				actions = append(actions, action)
			}
		case string(codes.DeprecatedSymbol):
			if action := deprecatedSymbolQuickFix(ctx, snap, diag); action != nil {
				actions = append(actions, action)
			}
		}
		// Missing-import is offered for any diagnostic whose range contains
		// a leading identifier that names an unimported ambient module.
		// The handler's own filters (local resolution succeeds, no ambient
		// match, already imported) reject non-candidates, so we don't need
		// a separate code to gate it.
		if action := missingImportQuickFix(ctx, snap, diag); action != nil {
			actions = append(actions, action)
		}
	}
	return actions, nil
}

// unusedImportQuickFix builds a quick-fix action for an unused-import
// diagnostic. The fix deletes the entire import statement when the
// targeted item is the only one in its statement, and deletes just the
// item (with adjacent whitespace) otherwise.
func unusedImportQuickFix(
	snap codeActionSnapshot,
	diag protocol.Diagnostic,
) *protocol.CodeAction {
	alias := findAliasAtDiagnostic(snap.Symbols, diag)
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
	items := stmt.AllImportItem()
	idx := -1
	for i, it := range items {
		if it == item {
			idx = i
			break
		}
	}
	if len(items) > 1 && idx < 0 {
		return nil
	}
	edit := buildUnusedImportEdit(snap.Content, stmt, items, idx)
	return &protocol.CodeAction{
		Title:       "Remove unused import",
		Kind:        lo.ToPtr(protocol.CodeActionKindQuickFix),
		Diagnostics: []protocol.Diagnostic{diag},
		IsPreferred: new(true),
		Edit: &protocol.WorkspaceEdit{
			Changes: map[uri.URI][]protocol.TextEdit{
				snap.URI: {edit},
			},
		},
	}
}

// findAliasAtDiagnostic returns the KindModuleAlias child of symbols whose
// AST starts at the diagnostic range's start, or nil if symbols is nil or
// no alias matches. Matching by start position keeps the resolution
// robust even when a parse error has truncated the symbol tree.
func findAliasAtDiagnostic(
	symbols *symbol.Symbol,
	diag protocol.Diagnostic,
) *symbol.Symbol {
	if symbols == nil {
		return nil
	}
	for _, child := range symbols.Children() {
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
// item at items[idx]. When items has a single member the whole statement
// is deleted (extending through the trailing newline so no blank line is
// left behind). Otherwise the item at idx is deleted along with one
// neighboring run of whitespace, preserving the surrounding parenthesized
// block. The caller is responsible for finding idx in items; idx is
// ignored when the statement is deleted whole.
func buildUnusedImportEdit(
	content string,
	stmt parser.IImportStatementContext,
	items []parser.IImportItemContext,
	idx int,
) protocol.TextEdit {
	if len(items) <= 1 {
		return deleteStatementEdit(content, stmt)
	}
	return deleteItemEdit(items, idx)
}

// deleteStatementEdit deletes the whole import statement plus a trailing
// newline when one is present, so removing the statement leaves no orphan
// blank line. Accepts the broader ParserRuleContext so the helper can be
// reused for any statement-shaped node (e.g., the loose `import name`
// form used by the auto-import consolidator) rather than just
// IImportStatementContext.
func deleteStatementEdit(
	content string,
	stmt antlr.ParserRuleContext,
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

// deleteItemEdit deletes the item at items[idx] from a multi-item
// parenthesized import block. When idx has a following sibling the
// deletion eats the whitespace between them; for the last item the
// deletion eats the whitespace after the previous sibling instead.
func deleteItemEdit(
	items []parser.IImportItemContext,
	idx int,
) protocol.TextEdit {
	item := items[idx]
	startLine, startChar := tokenStartLineChar(item.GetStart())
	endLine, endChar := tokenEndLineChar(item.GetStop())
	if idx < len(items)-1 {
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
	offset := lsp.PositionToOffset(content, protocol.Position{
		Line:      uint32(line),
		Character: uint32(char),
	})
	return offset < len(content) && content[offset] == '\n'
}

// deprecatedSymbolQuickFix builds a quick-fix action for a deprecated
// reference. It finds the bare identifier inside the diagnostic range,
// resolves it through the document's scope, and emits a text edit that
// replaces the identifier with the canonical replacement. When the
// replacement's module is not already in scope, an additional import edit
// is appended via buildAutoImportEditFromSnapshot. Qualified references
// like `module.member` are skipped in this version; users see the warning
// but not a fix.
func deprecatedSymbolQuickFix(
	ctx context.Context,
	snap codeActionSnapshot,
	diag protocol.Diagnostic,
) *protocol.CodeAction {
	if snap.Symbols == nil {
		return nil
	}
	token, sym := findDeprecatedBareRef(ctx, snap, diag.Range)
	if token == nil || sym == nil || sym.Deprecated == nil {
		return nil
	}
	repl := sym.Deprecated
	replacementName := buildReplacementName(snap.Symbols, repl)
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
		edits = append(buildAutoImportEditFromSnapshot(snap.Content, snap.Symbols, module), edits...)
	}
	return &protocol.CodeAction{
		Title:       fmt.Sprintf("Replace '%s' with '%s'", token.GetText(), replacementName),
		Kind:        lo.ToPtr(protocol.CodeActionKindQuickFix),
		Diagnostics: []protocol.Diagnostic{diag},
		IsPreferred: new(true),
		Edit: &protocol.WorkspaceEdit{
			Changes: map[uri.URI][]protocol.TextEdit{snap.URI: edits},
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
	snap codeActionSnapshot,
	r protocol.Range,
) (antlr.Token, *symbol.Symbol) {
	for i, t := range snap.Tokens {
		if t.GetTokenType() != parser.ArcLexerIDENTIFIER {
			continue
		}
		if !tokenInRange(t, r) {
			continue
		}
		if isPartOfQualifiedPath(snap.Tokens, i) {
			continue
		}
		pos := protocol.Position{
			Line:      uint32(t.GetLine() - 1),
			Character: uint32(t.GetColumn()),
		}
		scope := findScopeAt(snap.Symbols, snap.IsBlock, pos)
		if scope == nil {
			continue
		}
		sym, err := scope.Resolve(ctx, t.GetText(), symbol.WithoutUsageTracking)
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
func buildReplacementName(symbols *symbol.Symbol, repl *symbol.Symbol) string {
	if repl.Parent == nil || repl.Parent.Kind != symbol.KindModule {
		return repl.QualifiedName()
	}
	if symbols == nil {
		return repl.QualifiedName()
	}
	for _, child := range symbols.Children() {
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

// missingImportQuickFix offers `Add import 'X'` when the diagnostic range
// references an ambient module that is not yet in scope. The candidate
// identifier is the first IDENTIFIER token inside the range; it must (a)
// fail to resolve at that position, (b) match a non-Internal KindModule
// reachable through the ambient prelude, and (c) not already be imported
// in the document. Failing any of the three returns nil, which makes the
// handler safe to run for every diagnostic without code-matching.
func missingImportQuickFix(
	ctx context.Context,
	snap codeActionSnapshot,
	diag protocol.Diagnostic,
) *protocol.CodeAction {
	if snap.Symbols == nil {
		return nil
	}
	token := firstIdentifierInRange(snap.Tokens, diag.Range)
	if token == nil {
		return nil
	}
	name := token.GetText()
	pos := protocol.Position{
		Line:      uint32(token.GetLine() - 1),
		Character: uint32(token.GetColumn()),
	}
	scope := findScopeAt(snap.Symbols, snap.IsBlock, pos)
	if scope == nil {
		return nil
	}
	if sym, err := scope.Resolve(ctx, name, symbol.WithoutUsageTracking); err == nil && sym != nil {
		return nil
	}
	if !scope.IsAmbientModule(name) {
		return nil
	}
	edits := buildAutoImportEditFromSnapshot(snap.Content, snap.Symbols, name)
	if len(edits) == 0 {
		return nil
	}
	return &protocol.CodeAction{
		Title:       fmt.Sprintf("Add import '%s'", name),
		Kind:        lo.ToPtr(protocol.CodeActionKindQuickFix),
		Diagnostics: []protocol.Diagnostic{diag},
		IsPreferred: new(true),
		Edit: &protocol.WorkspaceEdit{
			Changes: map[uri.URI][]protocol.TextEdit{snap.URI: edits},
		},
	}
}

// firstIdentifierInRange returns the first IDENTIFIER token whose start
// position falls inside r, or nil when none does.
func firstIdentifierInRange(tokens []antlr.Token, r protocol.Range) antlr.Token {
	for _, t := range tokens {
		if t.GetTokenType() != parser.ArcLexerIDENTIFIER {
			continue
		}
		if tokenInRange(t, r) {
			return t
		}
	}
	return nil
}
