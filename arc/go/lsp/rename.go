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
		s.cfg.L.Error(
			"unexpected failure resolving symbol",
			zap.Stringer("symbol", sym),
			zap.Error(err),
		)
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
	if _, r, quoted := channelNameAt(doc.displayContent(), params.Position); quoted {
		return &r, nil
	}
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
	// Only a channel has a quoted form, so any other symbol takes a bare identifier.
	if targetSym.Kind != symbol.KindChannel && !isIdentifier(params.NewName) {
		return nil, errors.Newf("'%s' is not a valid name", params.NewName)
	}
	// Reference matching for global symbols (no AST) re-resolves each identifier
	// through GlobalResolver, so it must run before OnRename — otherwise an
	// OnRename that renames the backing resource (e.g., a Synnax channel) makes
	// the old name unresolvable and renameTextEdits returns nothing.
	edits := s.renameTextEdits(ctx, doc, targetSym, params.NewName)
	if len(edits) == 0 {
		return nil, nil
	}
	if s.cfg.OnRename != nil {
		if err := s.cfg.OnRename(
			ctx,
			targetSym,
			targetSym.Name,
			params.NewName,
		); err != nil {
			return nil, err
		}
	}
	return &protocol.WorkspaceEdit{
		Changes: map[uri.URI][]protocol.TextEdit{
			params.TextDocument.URI: edits,
		},
	}, nil
}

// renameTextEdits walks every identifier and quoted channel name token in doc,
// including those in format string placeholders, keeps the ones that resolve to the
// same symbol as targetSym, and emits a TextEdit per occurrence that replaces the token
// with newName, quoted for a quoted token. Edit ranges are in UTF-16 units.
func (s *Server) renameTextEdits(
	ctx context.Context,
	doc *document,
	targetSym *symbol.Symbol,
	newName string,
) []protocol.TextEdit {
	if doc.IR.Symbols == nil || targetSym == nil {
		return nil
	}
	lines := newLineText(doc.Content)
	var edits []protocol.TextEdit
	rename := func(n parser.Name, r protocol.Range) {
		if n.Head == "" || n.Head != targetSym.Name {
			return
		}
		scope := findScopeAt(doc.IR.Symbols, doc.isBlock(), r.Start)
		sym, err := scope.ResolveName(ctx, n, symbol.WithoutUsageTracking)
		if !isRenameable(sym, err) {
			return
		}
		s.logUnexpectedSymbolError(sym, err)
		if !sameSymbol(sym, targetSym) {
			return
		}
		text := newName
		if n.Quoted || !bareFits(ctx, scope, newName, targetSym) {
			text = parser.QuoteChannelName(newName)
		}
		edits = append(
			edits,
			protocol.TextEdit{Range: lines.rangeToUTF16(r), NewText: text},
		)
	}
	for _, t := range tokenizeContent(doc.Content) {
		switch t.GetTokenType() {
		case parser.ArcLexerSTR_LITERAL, parser.ArcLexerSTR_LITERAL_MULTI:
			f, ok := parseFormatString(t)
			if !ok {
				continue
			}
			for _, seg := range f.segs {
				if !seg.IsPlaceholder {
					continue
				}
				inner, spans := f.innerTokens(seg)
				for i, it := range inner {
					rename(
						parser.TokenName(it),
						protocol.Range{
							Start: f.at[spans[i][0]],
							End:   f.at[spans[i][1]],
						},
					)
				}
			}
		default:
			rename(parser.TokenName(t), tokenRange(t))
		}
	}
	return edits
}

// sameSymbol reports whether sym is target. Source-defined symbols match by AST
// identity. Global symbols, such as channels, have no AST and match by kind, since
// every in-scope reference to one name resolves to the same global symbol.
func sameSymbol(sym, target *symbol.Symbol) bool {
	if target.AST != nil {
		return sym.AST == target.AST
	}
	return sym.Kind == target.Kind && sym.AST == nil
}

// bareFits reports whether a bare reference to target can take the name newName in
// scope: newName lexes as one identifier, and no other symbol in scope holds it, so the
// reference keeps resolving to target.
func bareFits(
	ctx context.Context,
	scope *symbol.Symbol,
	newName string,
	target *symbol.Symbol,
) bool {
	if !isIdentifier(newName) {
		return false
	}
	sym, err := scope.ResolveName(
		ctx,
		parser.Name{Head: newName},
		symbol.WithoutUsageTracking,
	)
	return errors.Is(err, query.ErrNotFound) || err == nil && sameSymbol(sym, target)
}

// isIdentifier reports whether name lexes as one identifier, and not as a keyword.
func isIdentifier(name string) bool {
	tokens := tokenizeContent(name)
	return len(tokens) == 1 &&
		tokens[0].GetTokenType() == parser.ArcLexerIDENTIFIER &&
		tokens[0].GetText() == name
}
