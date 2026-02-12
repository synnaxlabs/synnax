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
)

type documentMetadata struct {
	isFunctionBlock bool
}

func extractMetadataFromURI(uri protocol.DocumentURI) *documentMetadata {
	uriStr := string(uri)
	return &documentMetadata{
		isFunctionBlock: strings.HasPrefix(uriStr, "arc://block/"),
	}
}

type Document struct {
	metadata    *documentMetadata
	Content     string
	URI         protocol.DocumentURI
	IR          ir.IR
	Diagnostics diagnostics.Diagnostics
	Version     int32
	debouncer   *debounce.Debouncer
}

func (d *Document) isBlock() bool {
	return d.metadata != nil && d.metadata.isFunctionBlock
}

func (d *Document) displayContent() string {
	return d.Content
}

func (d *Document) toASTPosition(pos protocol.Position) protocol.Position {
	if !d.isBlock() {
		return pos
	}
	if pos.Line == 0 {
		return protocol.Position{Line: pos.Line, Character: pos.Character + 1}
	}
	return pos
}

func (d *Document) toDocPosition(pos protocol.Position) protocol.Position {
	if !d.isBlock() {
		return pos
	}
	if pos.Line == 0 && pos.Character > 0 {
		return protocol.Position{Line: pos.Line, Character: pos.Character - 1}
	}
	return pos
}

func (d *Document) toDocRange(r protocol.Range) protocol.Range {
	return protocol.Range{
		Start: d.toDocPosition(r.Start),
		End:   d.toDocPosition(r.End),
	}
}

func (d *Document) toDocLocation(loc *protocol.Location) *protocol.Location {
	if loc == nil {
		return nil
	}
	return &protocol.Location{
		URI:   loc.URI,
		Range: d.toDocRange(loc.Range),
	}
}

func (d *Document) toDocLocations(locs []protocol.Location) []protocol.Location {
	if locs == nil {
		return nil
	}
	result := make([]protocol.Location, len(locs))
	for i, loc := range locs {
		result[i] = *d.toDocLocation(&loc)
	}
	return result
}

func (d *Document) getWordAtPosition(pos protocol.Position) string {
	return lsp.GetWordAtPosition(d.displayContent(), pos)
}

func (d *Document) getWordRangeAtPosition(pos protocol.Position) *protocol.Range {
	return lsp.GetWordRangeAtPosition(d.displayContent(), pos)
}

func (d *Document) findScopeAtPosition(pos protocol.Position) *symbol.Scope {
	if d.IR.Symbols == nil {
		return nil
	}
	astPos := d.toASTPosition(pos)
	internalPos := fromProtocol(astPos)
	var deepest *symbol.Scope
	findScopeRecursive(d.IR.Symbols, internalPos.Line, internalPos.Col, &deepest)
	if deepest == nil {
		return d.IR.Symbols
	}
	return deepest
}

func (d *Document) resolveSymbolAtPosition(ctx context.Context, pos protocol.Position) (*symbol.Scope, error) {
	word := d.getWordAtPosition(pos)
	if word == "" {
		return nil, nil
	}
	scope := d.findScopeAtPosition(pos)
	if scope == nil {
		return nil, nil
	}
	return scope.Resolve(ctx, word)
}
