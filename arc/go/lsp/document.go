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
	"sync"
	"time"

	"github.com/synnaxlabs/arc/diagnostics"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/symbol"
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

	// debounce fields - protected by dMu
	dMu            sync.Mutex
	debounceTimer  *time.Timer
	cancelAnalysis context.CancelFunc
	firstChangeAt  time.Time
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
	content := d.displayContent()
	line, ok := getLine(content, pos.Line)
	if !ok || int(pos.Character) >= len(line) {
		return ""
	}
	start := int(pos.Character)
	end := int(pos.Character)
	for start > 0 && isWordChar(line[start-1]) {
		start--
	}
	for end < len(line) && isWordChar(line[end]) {
		end++
	}
	return line[start:end]
}

func (d *Document) getWordRangeAtPosition(pos protocol.Position) *protocol.Range {
	word := d.getWordAtPosition(pos)
	if word == "" {
		return nil
	}
	content := d.displayContent()
	line, ok := getLine(content, pos.Line)
	if !ok || int(pos.Character) >= len(line) {
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

func (d *Document) stopDebounce() {
	d.dMu.Lock()
	defer d.dMu.Unlock()
	if d.debounceTimer != nil {
		d.debounceTimer.Stop()
		d.debounceTimer = nil
	}
	if d.cancelAnalysis != nil {
		d.cancelAnalysis()
		d.cancelAnalysis = nil
	}
}

// PositionToOffset converts an LSP line/character position to a byte offset
// within the given content string.
func PositionToOffset(content string, pos protocol.Position) int {
	line := int(pos.Line)
	char := int(pos.Character)
	offset := 0
	for i := 0; i < line; i++ {
		idx := strings.IndexByte(content[offset:], '\n')
		if idx < 0 {
			return len(content)
		}
		offset += idx + 1
	}
	offset += char
	if offset > len(content) {
		return len(content)
	}
	return offset
}

// IsFullReplacement detects whether a content change event represents a
// full-document replacement (no range specified).
func IsFullReplacement(
	change protocol.TextDocumentContentChangeEvent,
) bool {
	return change.Range == (protocol.Range{}) && change.RangeLength == 0
}

// ApplyIncrementalChange splices a single incremental change into the
// document content and returns the updated string.
func ApplyIncrementalChange(
	content string,
	change protocol.TextDocumentContentChangeEvent,
) string {
	start := PositionToOffset(content, change.Range.Start)
	end := PositionToOffset(content, change.Range.End)
	var b strings.Builder
	b.Grow(start + len(change.Text) + len(content) - end)
	b.WriteString(content[:start])
	b.WriteString(change.Text)
	b.WriteString(content[end:])
	return b.String()
}
