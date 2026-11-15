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
	"maps"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer/codes"
	"github.com/synnaxlabs/arc/formatter"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/diagnostics"
	lsp "github.com/synnaxlabs/x/lsp"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
	"go.uber.org/zap"
)

// codeActionSnapshot bundles the document state CodeAction needs after
// releasing s.mu. Snapshotting once at the top of CodeAction avoids racing
// with DidChange and runAnalysis, which write doc.Content and doc.IR under
// s.mu.Lock.
type codeActionSnapshot struct {
	URI         uri.URI
	Content     string
	Symbols     *symbol.Symbol
	Diagnostics diagnostics.Diagnostics
	Tokens      []antlr.Token
	IsBlock     bool
}

// CodeAction returns quick-fix actions for the diagnostics that intersect the requested
// range. A fix that the analyzer attached to its diagnostic needs no code here. A fix
// that needs the state of the server takes a case for its code in the dispatch below.
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
	var analyzed diagnostics.Diagnostics
	// A fix from the analysis of older content has the wrong ranges.
	if doc.analyzed == content {
		analyzed = doc.Diagnostics
	}
	isBlock := doc.isBlock()
	s.mu.RUnlock()
	snap := codeActionSnapshot{
		URI:         doc.URI,
		Content:     content,
		Symbols:     symbols,
		Diagnostics: analyzed,
		Tokens:      tokenizeContent(content),
		IsBlock:     isBlock,
	}
	var (
		actions []protocol.CommandOrCodeAction
		hoist   *protocol.CodeAction
		modules []*symbol.Symbol
		listed  bool
	)
	// Every diagnostic of a request shares one listing of the user modules.
	listModules := func() []*symbol.Symbol {
		if listed {
			return modules
		}
		listed = true
		r := symbols.ModuleResolver()
		if r == nil {
			return nil
		}
		var err error
		if modules, err = r.Search(ctx, ""); err != nil {
			s.cfg.L.Error("failed to search modules", zap.Error(err))
		}
		return modules
	}
	for _, diag := range params.Context.Diagnostics {
		code, _ := diag.Code.(protocol.String)
		switch string(code) {
		case string(codes.DeclarationAfterStep.Code):
			// One hoist covers the document, so every such warning shares it.
			if hoist == nil {
				hoist = hoistDeclarationsQuickFix(snap)
			}
			if hoist != nil {
				hoist.Diagnostics = append(hoist.Diagnostics, diag)
			}
		}
		actions = append(actions, analyzedQuickFixes(snap, diag, string(code))...)
		// Missing-import is offered for any diagnostic whose range contains
		// a leading identifier that names an unimported ambient module.
		// The handler's own filters (local resolution succeeds, no ambient
		// match, already imported) reject non-candidates, so we don't need
		// a separate code to gate it.
		actions = append(
			actions,
			missingImportQuickFixes(ctx, snap, diag, listModules)...,
		)
	}
	if hoist != nil {
		actions = append(actions, hoist)
	}
	return actions, nil
}

// analyzedQuickFixes builds one action for each fix the analyzer attached to diag.
func analyzedQuickFixes(
	snap codeActionSnapshot,
	diag protocol.Diagnostic,
	code string,
) []protocol.CommandOrCodeAction {
	var actions []protocol.CommandOrCodeAction
	for _, analyzed := range snap.Diagnostics {
		if analyzed.Range != diag.Range || string(analyzed.Code) != code {
			continue
		}
		for _, fix := range analyzed.Fixes {
			edits := fix.Edits
			if snap.IsBlock {
				edits = slices.Clone(edits)
				for i := range edits {
					edits[i].Range.Start = unwrapPosition(edits[i].Range.Start)
					edits[i].Range.End = unwrapPosition(edits[i].Range.End)
				}
			}
			actions = append(actions, &protocol.CodeAction{
				Title:       fix.Title,
				Kind:        new(protocol.CodeActionKindQuickFix),
				Diagnostics: []protocol.Diagnostic{diag},
				IsPreferred: new(len(actions) == 0),
				Edit: &protocol.WorkspaceEdit{
					Changes: map[uri.URI][]protocol.TextEdit{snap.URI: edits},
				},
			})
		}
	}
	return actions
}

// hoistDeclarationsQuickFix builds the action that applies the formatter's hoists.
func hoistDeclarationsQuickFix(snap codeActionSnapshot) *protocol.CodeAction {
	moves := formatter.DeclarationHoists(snap.Content)
	if len(moves) == 0 {
		return nil
	}
	var edits []protocol.TextEdit
	inserts := make(map[int]string)
	for _, m := range moves {
		edits = append(edits, protocol.TextEdit{Range: lineRange(m.Start, m.End)})
		inserts[m.Anchor] += linesOf(snap.Content, m.Start, m.End)
	}
	for _, anchor := range slices.Sorted(maps.Keys(inserts)) {
		edits = append(edits, protocol.TextEdit{
			Range:   lineRange(anchor, anchor),
			NewText: inserts[anchor],
		})
	}
	return &protocol.CodeAction{
		Title:       "Move declarations to the top",
		Kind:        new(protocol.CodeActionKindQuickFix),
		IsPreferred: new(true),
		Edit: &protocol.WorkspaceEdit{
			Changes: map[uri.URI][]protocol.TextEdit{snap.URI: edits},
		},
	}
}

// lineRange spans the whole lines in [startLine, endLine).
func lineRange(startLine, endLine int) protocol.Range {
	return protocol.Range{
		Start: protocol.Position{Line: uint32(startLine)},
		End:   protocol.Position{Line: uint32(endLine)},
	}
}

// linesOf returns the text of the whole lines in [startLine, endLine).
func linesOf(content string, startLine, endLine int) string {
	start := lsp.PositionToOffset(content, protocol.Position{Line: uint32(startLine)})
	end := lsp.PositionToOffset(content, protocol.Position{Line: uint32(endLine)})
	return content[start:end]
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

// missingImportQuickFixes offers `Add import 'X'` for the first identifier in the
// diagnostic range when it does not resolve there and names an ambient module or the
// last segment of a user module path that is not imported yet. It returns none for
// any other diagnostic, so it runs for every diagnostic without code-matching.
func missingImportQuickFixes(
	ctx context.Context,
	snap codeActionSnapshot,
	diag protocol.Diagnostic,
	listModules func() []*symbol.Symbol,
) []protocol.CommandOrCodeAction {
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
	if sym, err := scope.Resolve(
		ctx,
		name,
		symbol.WithoutUsageTracking,
	); err == nil &&
		sym != nil {
		return nil
	}
	var paths []string
	if scope.IsAmbientModule(name) {
		paths = []string{name}
	} else {
		for _, m := range listModules() {
			if m.Name == name || strings.HasSuffix(m.Name, "."+name) {
				paths = append(paths, m.Name)
			}
		}
	}
	var actions []protocol.CommandOrCodeAction
	for _, path := range paths {
		edits := buildAutoImportEditFromSnapshot(snap.Content, snap.Symbols, path)
		if len(edits) == 0 {
			continue
		}
		actions = append(actions, &protocol.CodeAction{
			Title:       fmt.Sprintf("Add import '%s'", path),
			Kind:        new(protocol.CodeActionKindQuickFix),
			Diagnostics: []protocol.Diagnostic{diag},
			IsPreferred: new(len(paths) == 1),
			Edit: &protocol.WorkspaceEdit{
				Changes: map[uri.URI][]protocol.TextEdit{snap.URI: edits},
			},
		})
	}
	return actions
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
