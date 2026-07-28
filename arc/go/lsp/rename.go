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
	"go.lsp.dev/uri"
	"go.uber.org/zap"
)

func (s *Server) logUnexpectedSymbolError(sym *symbol.Symbol, err error) {
	if err != nil && !errors.Is(err, query.ErrNotFound) {
		s.cfg.L.Error("unexpected failure resolving symbol", zap.Stringer("symbol", sym), zap.Error(err))
	}
}

func isRenameable(sym *symbol.Symbol, err error) bool {
	if errors.Is(err, query.ErrNotFound) || sym == nil {
		return false
	}
	// A positional module alias (`import (math)`) carries the canonical
	// module name as its own — renaming the source text just breaks the
	// import, since no module by the new name exists. An `as` clause
	// (`import (math as t)`) introduces a user-chosen local name, which
	// is fair game to rename.
	if sym.Kind == symbol.KindModuleAlias &&
		sym.Target != nil &&
		sym.Name == sym.Target.Name {
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
) (protocol.PrepareRenameResult, error) {
	doc, ok := s.getDocument(params.TextDocument.URI)
	if !ok || doc.IR.Symbols == nil {
		return nil, nil
	}
	sym, err := doc.resolveSymbolAtPosition(ctx, params.Position)
	if !isRenameable(sym, err) {
		return nil, nil
	}
	s.logUnexpectedSymbolError(sym, err)
	if r := doc.getWordRangeAtPosition(params.Position); r != nil {
		return r, nil
	}
	return nil, nil
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
	// the old name unresolvable and renameTextEdits returns nothing.
	edits := s.renameTextEdits(ctx, doc, targetSym, params.NewName)
	if len(edits) == 0 {
		return nil, nil
	}
	if s.cfg.OnRename != nil {
		if err := s.cfg.OnRename(ctx, targetSym, targetSym.Name, params.NewName); err != nil {
			return nil, err
		}
	}
	return &protocol.WorkspaceEdit{
		Changes: map[uri.URI][]protocol.TextEdit{
			params.TextDocument.URI: edits,
		},
	}, nil
}

// renameTextEdits walks every identifier token in doc, keeps the ones that
// resolve to the same symbol as targetSym, and emits a TextEdit per
// occurrence that replaces the token with newName. The block-mode column
// shift is applied inline via toDocRange so the walk produces protocol-
// ready edits in a single pass instead of a Location → TextEdit hop.
func (s *Server) renameTextEdits(
	ctx context.Context,
	doc *Document,
	targetSym *symbol.Symbol,
	newName string,
) []protocol.TextEdit {
	if doc.IR.Symbols == nil || targetSym == nil {
		return nil
	}
	// Source-defined symbols match by AST identity. Global symbols (e.g.,
	// channels) have no AST and instead match by kind + name, since every
	// in-scope reference to the same name resolves to the same global symbol.
	matchByAST := targetSym.AST != nil
	var edits []protocol.TextEdit
	for _, t := range tokenizeContent(doc.Content) {
		if t.GetTokenType() != parser.ArcLexerIDENTIFIER {
			continue
		}
		tokenText := t.GetText()
		if tokenText != targetSym.Name {
			continue
		}
		pos := position{Line: t.GetLine(), Col: t.GetColumn()}
		scope := findScopeAtInternalPosition(doc.IR.Symbols, pos)
		sym, err := scope.Resolve(ctx, tokenText, symbol.WithoutUsageTracking)
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
		edits = append(edits, protocol.TextEdit{
			Range: doc.toDocRange(protocol.Range{
				Start: protocol.Position{
					Line:      uint32(pos.Line - 1),
					Character: uint32(pos.Col),
				},
				End: protocol.Position{
					Line:      uint32(pos.Line - 1),
					Character: uint32(pos.Col + len(tokenText)),
				},
			}),
			NewText: newName,
		})
	}
	return edits
}
