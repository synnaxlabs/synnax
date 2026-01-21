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

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"go.lsp.dev/protocol"
)

// PrepareRename checks if the symbol at the given position can be renamed.
// Returns the range of the symbol if it can be renamed, nil otherwise.
func (s *Server) PrepareRename(
	ctx context.Context,
	params *protocol.PrepareRenameParams,
) (*protocol.Range, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok || doc.IR.Symbols == nil {
		return nil, nil
	}

	word := s.getWordAtPosition(doc.Content, params.Position)
	if word == "" {
		return nil, nil
	}

	scopeAtCursor := FindScopeAtPosition(doc.IR.Symbols, FromProtocol(params.Position))
	sym, err := scopeAtCursor.Resolve(ctx, word)
	if err != nil || sym == nil {
		return nil, nil
	}

	// Cannot rename symbols without AST (globals/builtins)
	if sym.AST == nil {
		return nil, nil
	}

	// Return the range of the word at cursor
	wordRange := s.getWordRangeAtPosition(doc.Content, params.Position)
	return wordRange, nil
}

// Rename renames all occurrences of the symbol at the given position.
func (s *Server) Rename(
	ctx context.Context,
	params *protocol.RenameParams,
) (*protocol.WorkspaceEdit, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok || doc.IR.Symbols == nil {
		return nil, nil
	}

	word := s.getWordAtPosition(doc.Content, params.Position)
	if word == "" {
		return nil, nil
	}

	scopeAtCursor := FindScopeAtPosition(doc.IR.Symbols, FromProtocol(params.Position))
	targetSym, err := scopeAtCursor.Resolve(ctx, word)
	if err != nil || targetSym == nil {
		return nil, nil
	}

	// Cannot rename symbols without AST (globals/builtins)
	if targetSym.AST == nil {
		return nil, nil
	}

	// Find all references to the target symbol
	locations := s.findAllReferences(ctx, doc, targetSym)
	if len(locations) == 0 {
		return nil, nil
	}

	// Build text edits for each location
	edits := make([]protocol.TextEdit, 0, len(locations))
	for _, loc := range locations {
		edits = append(edits, protocol.TextEdit{
			Range:   loc.Range,
			NewText: params.NewName,
		})
	}

	return &protocol.WorkspaceEdit{
		Changes: map[protocol.DocumentURI][]protocol.TextEdit{
			params.TextDocument.URI: edits,
		},
	}, nil
}

// findAllReferences finds all references to the given symbol in the document.
func (s *Server) findAllReferences(
	ctx context.Context,
	doc *Document,
	targetSym *symbol.Scope,
) []protocol.Location {
	if doc.IR.Symbols == nil || targetSym == nil || targetSym.AST == nil {
		return nil
	}

	// Use lexer to walk all tokens (same pattern as semantic.go)
	input := antlr.NewInputStream(doc.Content)
	lexer := parser.NewArcLexer(input)
	allTokens := lexer.GetAllTokens()

	var locations []protocol.Location
	for _, t := range allTokens {
		if t.GetTokenType() != parser.ArcLexerIDENTIFIER {
			continue
		}

		tokenText := t.GetText()
		// Quick filter: only check tokens with the same name
		if tokenText != targetSym.Name {
			continue
		}

		// Find scope at this token's position
		pos := Position{Line: t.GetLine(), Col: t.GetColumn()}
		scope := FindScopeAtPosition(doc.IR.Symbols, pos)
		if scope == nil {
			scope = doc.IR.Symbols
		}

		// Resolve the symbol at this position
		sym, err := scope.Resolve(ctx, tokenText)
		if err != nil || sym == nil || sym.AST == nil {
			continue
		}

		// Check if this is the same symbol (pointer equality on AST)
		if sym.AST == targetSym.AST {
			// Convert token position to LSP location
			locations = append(locations, protocol.Location{
				URI: doc.URI,
				Range: protocol.Range{
					Start: protocol.Position{
						Line:      uint32(pos.Line - 1),
						Character: uint32(pos.Col),
					},
					End: protocol.Position{
						Line:      uint32(pos.Line - 1),
						Character: uint32(pos.Col + len(tokenText)),
					},
				},
			})
		}
	}

	return locations
}

// getWordRangeAtPosition returns the range of the word at the given position.
func (s *Server) getWordRangeAtPosition(
	content string,
	pos protocol.Position,
) *protocol.Range {
	word := s.getWordAtPosition(content, pos)
	if word == "" {
		return nil
	}

	// Find the start of the word
	lines := SplitLines(content)
	if int(pos.Line) >= len(lines) {
		return nil
	}
	line := lines[pos.Line]
	if int(pos.Character) >= len(line) {
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
