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
	"sort"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
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
	// names is shared by every snapshot of the document.
	names *nameTokens
}

// nameTokens holds the tokens of one content that can hold a quoted channel name, its
// dotted names, and the quote facts of each of its lines. It lexes the tokens and the
// dotted names in one pass on first use and keeps them until the content changes. It
// scans the facts again only from the lines that changed, so a lookup by position costs
// the same at any document length.
type nameTokens struct {
	mu      sync.Mutex
	content string
	lexed   bool
	tokens  []antlr.Token
	scanned string
	scan    *quoteScan
	paths   []pathToken
	scanner *lineScanner
}

// of returns the name tokens of content in order: the quoted names and the string
// literals, whose placeholders can hold them. Callers must not change them.
func (n *nameTokens) of(content string) []antlr.Token {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lex(content)
	return n.tokens
}

// pathsOf returns the dotted names of content in order. Callers must not change them.
func (n *nameTokens) pathsOf(content string) []pathToken {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lex(content)
	return n.paths
}

func (n *nameTokens) lex(content string) {
	if n.lexed && n.content == content {
		return
	}
	tokens := tokenizeContent(content)
	n.content, n.lexed = content, true
	n.tokens, n.paths = keepNameTokens(tokens), linePaths(tokens)
}

// scanOf returns the quote facts of content. Callers must not change them.
func (n *nameTokens) scanOf(content string) *quoteScan {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.scanner == nil {
		n.scanner = newLineScanner()
	}
	switch {
	case n.scan == nil:
		n.scan = scanQuotes(content, n.scanner)
	case n.scanned != content:
		n.scan = n.scan.update(n.scanned, content, n.scanner)
	}
	n.scanned = content
	return n.scan
}

func keepNameTokens(tokens []antlr.Token) []antlr.Token {
	var kept []antlr.Token
	for _, t := range tokens {
		switch t.GetTokenType() {
		case parser.ArcLexerCHANNEL_NAME,
			parser.ArcLexerSTR_LITERAL,
			parser.ArcLexerSTR_LITERAL_MULTI:
			kept = append(kept, t)
		}
	}
	return kept
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

// findScopeAtPosition returns the scope at pos, a protocol position in UTF-16 units.
func (d *document) findScopeAtPosition(pos protocol.Position) *symbol.Symbol {
	if text, ok := lsp.GetLine(d.Content, pos.Line); ok {
		pos = fromUTF16In(text, pos)
	}
	return findScopeAt(d.IR.Symbols, d.isBlock(), pos)
}

// findScopeAt returns the scope of symbols at pos, whose column counts code points as
// token columns do. isBlock reports that symbols come from the body of a function.
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
	scope := d.findScopeAtPosition(pos)
	if scope == nil {
		return nil, nil
	}
	if n, _, ok := d.channelNameAt(pos); ok {
		return scope.ResolveName(ctx, n, symbol.WithoutUsageTracking)
	}
	// A dotted name that names a channel is one symbol, whichever segment holds pos.
	if n, ok := d.pathAt(pos); ok {
		sym, err := scope.ResolveName(ctx, n, symbol.WithoutUsageTracking)
		if err == nil && sym.Kind == symbol.KindChannel {
			return sym, nil
		}
	}
	word := d.getWordAtPosition(pos)
	if word == "" {
		return nil, nil
	}
	return scope.ResolveName(ctx, parser.Name{Head: word}, symbol.WithoutUsageTracking)
}

// channelNameAt returns the quoted channel name whose token holds pos, or the dotted
// name in a placeholder that holds pos, and its range. pos and the range are protocol
// positions in UTF-16 units.
func (d *document) channelNameAt(
	pos protocol.Position,
) (parser.Name, protocol.Range, bool) {
	content := d.displayContent()
	text, ok := lsp.GetLine(content, pos.Line)
	if !ok {
		return parser.Name{}, protocol.Range{}, false
	}
	at := fromUTF16In(text, pos)
	ts := d.names.of(content)
	line := int(pos.Line) + 1
	// Tokens do not overlap, so of the tokens that start before line, only the last can
	// reach it.
	i := sort.Search(len(ts), func(i int) bool { return ts[i].GetLine() > line })
	for j := i - 1; j >= 0; j-- {
		// A name token sits on the line of pos.
		if n, r, ok := channelNameIn(ts[j], at); ok {
			return n, protocol.Range{
				Start: toUTF16In(text, r.Start),
				End:   toUTF16In(text, r.End),
			}, true
		}
		if ts[j].GetLine() < line {
			break
		}
	}
	return parser.Name{}, protocol.Range{}, false
}

// importsModule reports whether name is an import alias in the scope at pos.
func (d *document) importsModule(
	ctx context.Context,
	name string,
	pos protocol.Position,
) bool {
	scope := d.findScopeAtPosition(pos)
	if scope == nil {
		return false
	}
	sym, err := scope.Resolve(ctx, name, symbol.WithoutUsageTracking)
	return err == nil && sym.Kind == symbol.KindModuleAlias
}

// pathAt returns the dotted name whose text holds pos.
func (d *document) pathAt(pos protocol.Position) (parser.Name, bool) {
	ps := d.names.pathsOf(d.displayContent())
	// Paths do not overlap, so only the first that ends after pos can hold it.
	i := sort.Search(len(ps), func(i int) bool {
		end := ps[i].rng.End
		return end.Line > pos.Line ||
			(end.Line == pos.Line && end.Character > pos.Character)
	})
	if i < len(ps) && holds(ps[i].rng, pos) {
		return ps[i].name, true
	}
	return parser.Name{}, false
}

// channelNameIn returns the quoted channel name at pos in the name token t, and its
// range.
func channelNameIn(
	t antlr.Token,
	pos protocol.Position,
) (parser.Name, protocol.Range, bool) {
	if t.GetTokenType() != parser.ArcLexerCHANNEL_NAME {
		return placeholderChannelNameAt(t, pos)
	}
	if r := tokenRange(t); holds(r, pos) {
		return parser.TokenName(t), r, true
	}
	return parser.Name{}, protocol.Range{}, false
}

// placeholderChannelNameAt returns the quoted channel name or the dotted name at pos
// inside a placeholder of the format string t, and its range.
func placeholderChannelNameAt(
	t antlr.Token,
	pos protocol.Position,
) (parser.Name, protocol.Range, bool) {
	f, ok := parseFormatString(t)
	if !ok {
		return parser.Name{}, protocol.Range{}, false
	}
	for _, seg := range f.segs {
		if !seg.IsPlaceholder {
			continue
		}
		inner, spans := f.innerTokens(seg)
		for i, it := range inner {
			if it.GetTokenType() != parser.ArcLexerCHANNEL_NAME {
				continue
			}
			r := protocol.Range{Start: f.at[spans[i][0]], End: f.at[spans[i][1]]}
			if holds(r, pos) {
				return parser.TokenName(it), r, true
			}
		}
		for _, p := range barePaths(inner, nil) {
			first, last := p.idents[0], p.idents[len(p.idents)-1]
			r := protocol.Range{Start: f.at[spans[first][0]], End: f.at[spans[last][1]]}
			if holds(r, pos) {
				return p.name, r, true
			}
		}
	}
	return parser.Name{}, protocol.Range{}, false
}

// holds reports whether the one-line range r holds pos, its end included, as a word
// lookup does.
func holds(r protocol.Range, pos protocol.Position) bool {
	return pos.Line == r.Start.Line && pos.Character >= r.Start.Character &&
		pos.Character <= r.End.Character
}

// tokenRange returns the range of a token that spans one line, with columns in code
// points.
func tokenRange(t antlr.Token) protocol.Range {
	start := protocol.Position{
		Line:      uint32(t.GetLine() - 1),
		Character: uint32(t.GetColumn()),
	}
	end := start
	end.Character += uint32(utf8.RuneCountInString(t.GetText()))
	return protocol.Range{Start: start, End: end}
}
