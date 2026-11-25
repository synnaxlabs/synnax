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

	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	arctext "github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
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
	if n, r, quoted := doc.channelNameAt(params.Position); quoted {
		return &protocol.PrepareRenamePlaceholder{Range: r, Placeholder: n.Head}, nil
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
	if targetSym.Kind != symbol.KindChannel && !parser.IsIdentifier(params.NewName) {
		return nil, validate.Errorf("'%s' is not a valid name", params.NewName)
	}
	// Reference matching for global symbols (no AST) re-resolves each identifier
	// through GlobalResolver, so it must run before OnRename — otherwise an
	// OnRename that renames the backing resource (e.g., a Synnax channel) makes
	// the old name unresolvable and renameTextEdits returns nothing.
	edits, err := s.renameTextEdits(ctx, doc, targetSym, params.NewName)
	if err != nil {
		return nil, err
	}
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

// renameTextEdits returns an edit for each token in doc, placeholders included, that
// names targetSym, writing newName in the token's form: a quoted token stays quoted,
// and a bare one stays bare when parser.ChannelText allows it. For a channel, only the
// tokens where a channel can be named count. Ranges are in UTF-16 units. It returns a
// validation error when newName cannot stand at some token.
func (s *Server) renameTextEdits(
	ctx context.Context,
	doc *document,
	targetSym *symbol.Symbol,
	newName string,
) ([]protocol.TextEdit, error) {
	if doc.IR.Symbols == nil || targetSym == nil {
		return nil, nil
	}
	var (
		channel = targetSym.Kind == symbol.KindChannel
		cfg     = s.parserConfig()
		lines   = newLineText(doc.Content)
		edits   []protocol.TextEdit
	)
	// text returns what replaces the token that holds n at r, and false when the token
	// does not name targetSym.
	text := func(n parser.Name, r protocol.Range) (string, bool, error) {
		if n.Head == "" || n.Head != targetSym.Name {
			return "", false, nil
		}
		scope := findScopeAt(doc.IR.Symbols, doc.isBlock(), r.Start)
		sym, err := scope.ResolveName(ctx, n, symbol.WithoutUsageTracking)
		if !isRenameable(sym, err) {
			return "", false, nil
		}
		s.logUnexpectedSymbolError(sym, err)
		if !sameSymbol(sym, targetSym) {
			return "", false, nil
		}
		taken := s.taken(ctx, scope, targetSym)
		switch {
		case !channel && taken(newName):
			return "", false, validate.Errorf(
				"'%s' already names another symbol where '%s' is used",
				newName,
				targetSym.Name,
			)
		case !channel:
			return newName, true, nil
		case n.Quoted:
			return parser.QuoteChannelName(newName), true, nil
		}
		return parser.ChannelText(newName, taken), true, nil
	}
	sites := arctext.ChannelSites(doc.Content, doc.isBlock(), cfg)
	for _, t := range tokenizeContent(doc.Content, cfg) {
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
				inner, spans := f.innerTokens(seg, cfg)
				innerSites := arctext.ExpressionSites(seg.Text, cfg)
				for i, it := range inner {
					if channel && !innerSites.Contains(it.GetStart()) {
						continue
					}
					r := protocol.Range{
						Start: f.at[spans[i][0]],
						End:   f.at[spans[i][1]],
					}
					newText, ok, err := text(parser.TokenName(it), r)
					if err != nil {
						return nil, err
					}
					if !ok {
						continue
					}
					escaped, fits := literal.Escape(newText, f.flags)
					if !fits {
						return nil, validate.Errorf(
							"a raw string cannot hold %s",
							newText,
						)
					}
					edits = append(
						edits,
						protocol.TextEdit{
							Range:   lines.rangeToUTF16(r),
							NewText: escaped,
						},
					)
				}
			}
		default:
			if channel && !sites.Contains(t.GetStart()) {
				continue
			}
			r := tokenRange(t)
			newText, ok, err := text(parser.TokenName(t), r)
			if err != nil {
				return nil, err
			}
			if ok {
				edits = append(
					edits,
					protocol.TextEdit{Range: lines.rangeToUTF16(r), NewText: newText},
				)
			}
		}
	}
	return edits, nil
}

// sameSymbol reports whether sym is target. Source-defined symbols match by AST
// identity, and global symbols, such as channels, by kind and ID.
func sameSymbol(sym, target *symbol.Symbol) bool {
	if target.AST != nil {
		return sym.AST == target.AST
	}
	return sym.AST == nil && sym.Kind == target.Kind && sym.ID == target.ID
}

// taken returns whether a symbol other than target holds a name in scope. A failed
// lookup counts as taken, so the name is quoted or refused.
func (s *Server) taken(
	ctx context.Context,
	scope *symbol.Symbol,
	target *symbol.Symbol,
) func(string) bool {
	return func(name string) bool {
		sym, err := scope.ResolveName(
			ctx,
			parser.Name{Head: name},
			symbol.WithoutUsageTracking,
		)
		if symbol.IsUndefined(err) {
			return false
		}
		if err != nil {
			s.cfg.L.Warn(
				"failed to resolve a name for a rename",
				zap.String("name", name),
				zap.Error(err),
			)
			return true
		}
		return !sameSymbol(sym, target)
	}
}
