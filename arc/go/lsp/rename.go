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
	sym, err := resolveSymbolAtPosition(ctx, doc.IR.Symbols, doc.Content, params.Position)
	if err != nil || sym == nil || sym.AST == nil {
		return nil, nil
	}
	return getWordRangeAtPosition(doc.Content, params.Position), nil
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
	targetSym, err := resolveSymbolAtPosition(ctx, doc.IR.Symbols, doc.Content, params.Position)
	if err != nil || targetSym == nil || targetSym.AST == nil {
		return nil, nil
	}

	locations := s.findAllReferences(ctx, doc, targetSym)
	if len(locations) == 0 {
		return nil, nil
	}

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

func (s *Server) findAllReferences(
	ctx context.Context,
	doc *Document,
	targetSym *symbol.Scope,
) []protocol.Location {
	if doc.IR.Symbols == nil || targetSym == nil || targetSym.AST == nil {
		return nil
	}
	allTokens := TokenizeContent(doc.Content)
	var locations []protocol.Location
	for _, t := range allTokens {
		if t.GetTokenType() != parser.ArcLexerIDENTIFIER {
			continue
		}
		tokenText := t.GetText()
		if tokenText != targetSym.Name {
			continue
		}
		pos := Position{Line: t.GetLine(), Col: t.GetColumn()}
		scope := FindScopeAtPosition(doc.IR.Symbols, pos)
		if scope == nil {
			scope = doc.IR.Symbols
		}
		sym, err := scope.Resolve(ctx, tokenText)
		if err != nil || sym == nil || sym.AST == nil {
			continue
		}
		if sym.AST == targetSym.AST {
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
