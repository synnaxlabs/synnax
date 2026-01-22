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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"go.lsp.dev/protocol"
	"go.uber.org/zap"
)

func (s *Server) logUnexpectedSymbolError(sym *symbol.Scope, err error) {
	if err != nil && !errors.Is(err, query.ErrNotFound) {
		s.cfg.L.Error("unexpected failure resolving symbol", zap.Stringer("symbol", sym), zap.Error(err))
	}
}

func isValidSymbol(sym *symbol.Scope, err error) bool {
	return !errors.Is(err, query.ErrNotFound) && sym != nil && sym.AST != nil
}

func (s *Server) PrepareRename(
	ctx context.Context,
	params *protocol.PrepareRenameParams,
) (*protocol.Range, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok || doc.IR.Symbols == nil {
		return nil, nil
	}
	sym, err := doc.resolveSymbolAtPosition(ctx, params.Position)
	if !isValidSymbol(sym, err) {
		return nil, nil
	}
	s.logUnexpectedSymbolError(sym, err)
	return doc.getWordRangeAtPosition(params.Position), nil
}

func (s *Server) Rename(
	ctx context.Context,
	params *protocol.RenameParams,
) (*protocol.WorkspaceEdit, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok || doc.IR.Symbols == nil {
		return nil, nil
	}
	targetSym, err := doc.resolveSymbolAtPosition(ctx, params.Position)
	if !isValidSymbol(targetSym, err) {
		return nil, nil
	}
	s.logUnexpectedSymbolError(targetSym, err)
	locations := s.findAllReferences(ctx, doc, targetSym)
	if len(locations) == 0 {
		return nil, nil
	}
	docLocations := doc.toDocLocations(locations)
	edits := make([]protocol.TextEdit, 0, len(docLocations))
	for _, loc := range docLocations {
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
	allTokens := tokenizeContent(doc.Content)
	var locations []protocol.Location
	for _, t := range allTokens {
		if t.GetTokenType() != parser.ArcLexerIDENTIFIER {
			continue
		}
		tokenText := t.GetText()
		if tokenText != targetSym.Name {
			continue
		}
		pos := position{Line: t.GetLine(), Col: t.GetColumn()}
		scope := findScopeAtInternalPosition(doc.IR.Symbols, pos)
		sym, err := scope.Resolve(ctx, tokenText)
		if !isValidSymbol(sym, err) {
			continue
		}
		s.logUnexpectedSymbolError(sym, err)
		if sym.AST != targetSym.AST {
			continue
		}
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
	return locations
}
