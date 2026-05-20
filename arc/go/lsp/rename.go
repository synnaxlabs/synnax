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
	"github.com/synnaxlabs/x/lsp/protocol"
	"github.com/synnaxlabs/x/query"
	"go.uber.org/zap"
)

func (s *Server) logUnexpectedSymbolError(sym *symbol.Scope, err error) {
	if err != nil && !errors.Is(err, query.ErrNotFound) {
		s.cfg.L.Error("unexpected failure resolving symbol", zap.Stringer("symbol", sym), zap.Error(err))
	}
}

func isRenameable(sym *symbol.Scope, err error) bool {
	if errors.Is(err, query.ErrNotFound) || sym == nil {
		return false
	}
	// Source-defined symbols rename via text edits alone. Resolver-supplied
	// symbols opt in via Symbol.Renameable; the host's OnRename callback is
	// responsible for propagating the rename to the underlying resource.
	return sym.AST != nil || sym.Renameable
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
	if !isRenameable(sym, err) {
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
	if !isRenameable(targetSym, err) {
		return nil, nil
	}
	s.logUnexpectedSymbolError(targetSym, err)
	// Reference matching for global symbols (no AST) re-resolves each identifier
	// through GlobalResolver, so it must run before OnRename — otherwise an
	// OnRename that renames the backing resource (e.g., a Synnax channel) makes
	// the old name unresolvable and findAllReferences returns nothing.
	locations := s.findAllReferences(ctx, doc, targetSym)
	if len(locations) == 0 {
		return nil, nil
	}
	if s.cfg.OnRename != nil {
		if err := s.cfg.OnRename(ctx, targetSym, targetSym.Name, params.NewName); err != nil {
			return nil, err
		}
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
	if doc.IR.Symbols == nil || targetSym == nil {
		return nil
	}
	// Source-defined symbols match by AST identity. Global symbols (e.g.,
	// channels) have no AST and instead match by kind + name, since every
	// in-scope reference to the same name resolves to the same global symbol.
	matchByAST := targetSym.AST != nil
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
		if !isRenameable(sym, err) {
			continue
		}
		s.logUnexpectedSymbolError(sym, err)
		if matchByAST {
			if sym.AST != targetSym.AST {
				continue
			}
		} else if sym.Kind != targetSym.Kind || sym.AST != nil {
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
