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
	"strings"

	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/x/debounce"
	"github.com/synnaxlabs/x/diagnostics"
	lsp "github.com/synnaxlabs/x/lsp"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

type documentMetadata struct {
	isFunctionBlock bool
	isModule        bool
}

func extractMetadataFromURI(docURI uri.URI) *documentMetadata {
	uriStr := string(docURI)
	return &documentMetadata{
		isFunctionBlock: strings.HasPrefix(uriStr, "arc://block/"),
		isModule:        strings.HasPrefix(uriStr, "arc://module/"),
	}
}

type document struct {
	metadata    *documentMetadata
	Content     string
	URI         uri.URI
	IR          ir.IR
	Diagnostics diagnostics.Diagnostics
	// analyzed is the content that IR and Diagnostics come from.
	analyzed  string
	Version   int32
	debouncer *debounce.Debouncer
}

func (d *document) isBlock() bool {
	return d.metadata != nil && d.metadata.isFunctionBlock
}

func (d *document) displayContent() string {
	return d.Content
}

func (d *document) toDocPosition(pos protocol.Position) protocol.Position {
	if !d.isBlock() {
		return pos
	}
	return unwrapPosition(pos)
}

// unwrapPosition moves a position in a parsed block to its place in the document.
func unwrapPosition(pos protocol.Position) protocol.Position {
	if pos.Line == 0 && pos.Character > 0 {
		pos.Character--
	}
	return pos
}

func unwrapPositions(r protocol.Range) protocol.Range {
	return protocol.Range{Start: unwrapPosition(r.Start), End: unwrapPosition(r.End)}
}

// unwrapDiagnostics moves the positions of diags, from analysis of a parsed block, to
// their places in the document. It modifies diags in place.
func unwrapDiagnostics(diags diagnostics.Diagnostics) diagnostics.Diagnostics {
	for i := range diags {
		d := &diags[i]
		d.Range = unwrapPositions(d.Range)
		for j := range d.Notes {
			d.Notes[j].Location.Range = unwrapPositions(d.Notes[j].Location.Range)
		}
		for _, fix := range d.Fixes {
			for k := range fix.Edits {
				fix.Edits[k].Range = unwrapPositions(fix.Edits[k].Range)
			}
		}
	}
	return diags
}

func (d *document) toDocRange(r protocol.Range) protocol.Range {
	return protocol.Range{
		Start: d.toDocPosition(r.Start),
		End:   d.toDocPosition(r.End),
	}
}

func (d *document) toDocLocation(loc *protocol.Location) *protocol.Location {
	if loc == nil {
		return nil
	}
	return &protocol.Location{
		URI:   loc.URI,
		Range: d.toDocRange(loc.Range),
	}
}

func (d *document) toDocLocations(locs []protocol.Location) []protocol.Location {
	if locs == nil {
		return nil
	}
	result := make([]protocol.Location, len(locs))
	for i, loc := range locs {
		result[i] = *d.toDocLocation(&loc)
	}
	return result
}

func (d *document) getWordAtPosition(pos protocol.Position) string {
	return lsp.GetWordAtPosition(d.displayContent(), pos)
}

func (d *document) getWordRangeAtPosition(pos protocol.Position) *protocol.Range {
	return lsp.GetWordRangeAtPosition(d.displayContent(), pos)
}

func (d *document) findScopeAtPosition(pos protocol.Position) *symbol.Symbol {
	return findScopeAt(d.IR.Symbols, d.isBlock(), pos)
}

// findScopeAt is the snapshot-friendly form of (*Document).findScopeAtPosition:
// it operates on a symbol tree and isBlock flag passed in by the caller so
// handlers can snapshot doc fields under the server lock and then resolve a
// scope without touching the live Document.
func findScopeAt(
	symbols *symbol.Symbol,
	isBlock bool,
	pos protocol.Position,
) *symbol.Symbol {
	if symbols == nil {
		return nil
	}
	astPos := pos
	if isBlock && pos.Line == 0 {
		astPos.Character++
	}
	return findScopeAtInternalPosition(symbols, fromProtocol(astPos))
}

// symbolAt returns the symbol at pos, or nil when no symbol resolves there.
func (d *document) symbolAt(ctx context.Context, pos protocol.Position) *symbol.Symbol {
	sym, err := d.resolveSymbolAtPosition(ctx, pos)
	if err != nil {
		return nil
	}
	return sym
}

func (d *document) resolveSymbolAtPosition(
	ctx context.Context,
	pos protocol.Position,
) (*symbol.Symbol, error) {
	word := d.getWordAtPosition(pos)
	if word == "" {
		return nil, nil
	}
	scope := d.findScopeAtPosition(pos)
	if scope == nil {
		return nil, nil
	}
	return scope.Resolve(ctx, word, symbol.WithoutUsageTracking)
}
