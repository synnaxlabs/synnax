// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// TextChannel is a channel as the text passes read it.
type TextChannel struct {
	Name     string
	DataType telem.DataType
}

// Renames is how text that a v0.62 Core stored reads each channel name, and the new
// name of each channel that the upgrade renamed (TextRename).
type Renames struct {
	bound    map[string]binding
	channels map[Key]TextChannel
}

// binding is the channel that a v0.62 Core bound a name to.
type binding struct {
	key     Key
	renamed bool
}

var _ symbol.Resolver = (*Renames)(nil)

// Rename returns the new name of the channel that a v0.62 Core bound name to, and false
// when the upgrade did not rename that channel.
func (r *Renames) Rename(name string) (string, bool) {
	b, ok := r.bound[name]
	if !ok || !b.renamed {
		return "", false
	}
	return r.channels[b.key].Name, true
}

// Resolve implements symbol.Resolver. A name resolves to the channel that a v0.62 Core
// bound it to.
func (r *Renames) Resolve(_ context.Context, name string) (*symbol.Symbol, error) {
	b, ok := r.bound[name]
	if !ok {
		return nil, errors.Wrapf(query.ErrNotFound, "channel %q", name)
	}
	return &symbol.Symbol{
		Name: name,
		Kind: symbol.KindChannel,
		Type: types.Chan(types.FromTelem(r.channels[b.key].DataType)),
		ID:   int(b.key),
	}, nil
}

// Search implements symbol.Resolver. It finds nothing, since no pass completes names.
func (r *Renames) Search(context.Context, string) ([]*symbol.Symbol, error) {
	return nil, nil
}

// Form is how a v0.62 Core parsed a stored text.
type Form uint8

const (
	// FormCalculation is the expression of a calculated channel, parsed as the body of
	// a function.
	FormCalculation Form = iota + 1
	// FormBody is the body of a graph function, with the inputs and outputs of the
	// function bound.
	FormBody
	// FormProgram is the text of an Arc.
	FormProgram
)

// Source is a text that a v0.62 Core stored, and how it parsed it.
type Source struct {
	Text string
	Form Form
	// Function, Inputs, and Outputs are the key and the params of the graph function of
	// a FormBody text.
	Function        string
	Inputs, Outputs types.Params
}

// Reference is a bare identifier in stored text that holds the old name of a renamed
// channel.
type Reference struct {
	// Name is the old name, and NewName the new name of the channel.
	Name, NewName string
	// Line counts from 1, and Column counts characters from 0.
	Line, Column int
	// Binding is the kind of the symbol that Name resolves to in the scope of the
	// reference. Bound is false when it resolves to none, or the text does not parse.
	Binding symbol.Kind
	Bound   bool
	// Captured is true when a symbol in scope at the reference holds the new name.
	Captured bool
	// Kept is true when the rewrite left the reference on the old name although it
	// reads the channel: the text does not parse, or a reference of it is captured.
	Kept bool
}

// Result is what a rewrite made of a stored text.
type Result struct {
	// Text is the rewritten text. It is the stored text when a reference is kept.
	Text string
	// Parsed is false when the text does not parse.
	Parsed bool
	// References are the bare identifiers of the text that hold the old name of a
	// renamed channel, in order.
	References []Reference
}

// Rewriter writes the new name of each renamed channel into text that a v0.62 Core
// stored.
type Rewriter struct {
	// Renames binds each name as a v0.62 Core did.
	Renames *Renames
	// Config lexes and parses text as the Core did.
	Config parser.Config
	// Ambient holds the global symbols beside the channels.
	Ambient []*symbol.Symbol
}

// Rewrite rewrites the text of src: each bare identifier that reads a renamed channel
// in its scope becomes the new name of the channel. A local, input, stage, or module
// member of the name takes the identifier first, as in a v0.62 Core, and the head or a
// member of a qualified name is never a candidate. When src does not parse, or a symbol
// in scope at a reference holds the new name, Rewrite leaves the text unchanged and
// marks each reference that it could not write as kept.
func (w Rewriter) Rewrite(ctx context.Context, src Source) (Result, error) {
	res, spans := w.find(src.Text)
	if len(res.References) == 0 {
		return res, nil
	}
	scope, shift, parsed, err := w.analyze(ctx, src)
	if err != nil {
		return Result{}, err
	}
	if !parsed {
		res.Parsed = false
		for i := range res.References {
			res.References[i].Kept = true
		}
		return res, nil
	}
	if bindAll(ctx, scope, shift, res.References) {
		return res, nil
	}
	var b strings.Builder
	last := 0
	for i, r := range res.References {
		if r.Bound && r.Binding == symbol.KindChannel {
			b.WriteString(src.Text[last:spans[i].start])
			b.WriteString(r.NewName)
			last = spans[i].end
		}
	}
	b.WriteString(src.Text[last:])
	res.Text = b.String()
	return res, nil
}

// span is the byte range [start, end) of a reference in a text.
type span struct{ start, end int }

// find returns the result of text before any rewrite, with each bare identifier that
// holds the old name of a renamed channel, and the span of each.
func (w Rewriter) find(text string) (Result, []span) {
	var (
		res   = Result{Text: text, Parsed: true}
		spans []span
	)
	identity := func(i int) int { return i }
	identifiers(text, w.Config, identity, func(start, end int, name string) {
		to, ok := w.Renames.Rename(name)
		if !ok {
			return
		}
		line, column := position(text, start)
		res.References = append(res.References, Reference{
			Name:    name,
			NewName: to,
			Line:    line,
			Column:  column,
		})
		spans = append(spans, span{start: start, end: end})
	})
	return res, spans
}

// bindAll resolves each of refs in the scopes of scope, where the columns of the first
// line move by shift. It reports whether a reference that reads the channel is
// captured, and marks each captured one as kept.
func bindAll(
	ctx context.Context,
	scope *symbol.Symbol,
	shift int,
	refs []Reference,
) bool {
	captured := false
	for i, r := range refs {
		col := r.Column
		if r.Line == 1 {
			col += shift
		}
		at := symbol.ScopeAt(scope, r.Line, col)
		sym, err := at.Resolve(ctx, r.Name, symbol.WithoutUsageTracking)
		if err != nil {
			continue
		}
		refs[i].Binding, refs[i].Bound = sym.Kind, true
		holder, err := at.Resolve(ctx, r.NewName, symbol.WithoutUsageTracking)
		if sym.Kind == symbol.KindChannel && err == nil &&
			holder.Kind != symbol.KindChannel {
			refs[i].Captured, refs[i].Kept = true, true
			captured = true
		}
	}
	return captured
}

// identifiers calls found with the byte range and the name of each bare identifier that
// the live lexer finds in src with cfg, in code and in each placeholder of a format
// string, in order. at maps a byte offset of src to the text that found counts in. The
// head and the members of a qualified name are not bare.
func identifiers(
	src string,
	cfg parser.Config,
	at func(int) int,
	found func(start, end int, name string),
) {
	lexer := parser.NewLexer(src, cfg)
	lexer.RemoveErrorListeners()
	var code []antlr.Token
	for _, t := range lexer.GetAllTokens() {
		if t.GetChannel() == antlr.TokenDefaultChannel {
			code = append(code, t)
		}
	}
	offset := byteOffsets(src)
	dot := func(i int) bool {
		return i >= 0 && i < len(code) && code[i].GetTokenType() == parser.ArcLexerDOT
	}
	for i, t := range code {
		start, end := offset[t.GetStart()], offset[t.GetStop()+1]
		switch t.GetTokenType() {
		case parser.ArcLexerIDENTIFIER:
			if !dot(i-1) && !dot(i+1) {
				found(at(start), at(end), src[start:end])
			}
		case parser.ArcLexerSTR_LITERAL, parser.ArcLexerSTR_LITERAL_MULTI:
			placeholders(
				src[start:end],
				cfg,
				func(j int) int { return at(start + j) },
				found,
			)
		}
	}
}

// placeholders calls identifiers on the source of each placeholder of the format string
// lit. at maps a byte offset of lit to the text that found counts in.
func placeholders(
	lit string,
	cfg parser.Config,
	at func(int) int,
	found func(start, end int, name string),
) {
	body, flags, ok := literal.StripQuotes(lit)
	if !ok || !flags.Format {
		return
	}
	segments, err := literal.FmtStrSplit(body, flags)
	if err != nil {
		return
	}
	// The body follows the r and f prefixes and the opening quote.
	open := strings.IndexAny(lit, "\"'`") + 1
	for _, seg := range segments {
		if !seg.IsPlaceholder {
			continue
		}
		identifiers(seg.Text, cfg, func(k int) int {
			return at(open + seg.Offsets[k])
		}, found)
	}
}

// byteOffsets returns the byte offset in s of each character of s, and of its end.
func byteOffsets(s string) []int {
	offsets := make([]int, 0, len(s)+1)
	for i := range s {
		offsets = append(offsets, i)
	}
	return append(offsets, len(s))
}

// analyze returns the scopes of src, how far the columns of its first line move in
// them, and false when src does not parse.
func (w Rewriter) analyze(
	ctx context.Context,
	src Source,
) (*symbol.Symbol, int, bool, error) {
	root := symbol.NewRoot(w.Renames, w.Ambient)
	switch src.Form {
	case FormCalculation:
		block, diag := parser.ParseBlock("{"+src.Text+"}", w.Config)
		if diag == nil {
			aCtx := acontext.NewRoot(ctx, block, root).WithConfig(w.Config)
			statement.AnalyzeFunctionBody(aCtx)
			return aCtx.Scope, 1, true, nil
		}
	case FormBody:
		block, diag := parser.ParseBlock(src.Text, w.Config)
		if diag == nil {
			scope, err := w.analyzeBody(ctx, root, block, src)
			return scope, 0, true, err
		}
	case FormProgram:
		t, diag := text.Parse(text.Text{Raw: src.Text}, w.Config)
		if diag == nil {
			aCtx := acontext.NewRoot(ctx, t.AST, root).WithConfig(w.Config)
			analyzer.AnalyzeProgram(aCtx)
			return aCtx.Scope, 0, true, nil
		}
	default:
		return nil, 0, false, errors.Newf("unknown text form %d", src.Form)
	}
	return nil, 0, false, nil
}

// analyzeBody analyzes block, the body of the graph function of src, in a function
// scope under root that binds the inputs and outputs of the function.
func (w Rewriter) analyzeBody(
	ctx context.Context,
	root *symbol.Symbol,
	block parser.IBlockContext,
	src Source,
) (*symbol.Symbol, error) {
	aCtx := acontext.NewRoot(ctx, block, root).WithConfig(w.Config)
	fn, err := aCtx.Scope.Add(ctx, symbol.Symbol{
		Name: src.Function,
		Kind: symbol.KindFunction,
		AST:  block,
	})
	if err != nil {
		return nil, err
	}
	if err = bind(ctx, fn, src.Inputs, symbol.KindInput); err != nil {
		return nil, err
	}
	if err = bind(ctx, fn, src.Outputs, symbol.KindOutput); err != nil {
		return nil, err
	}
	analyzer.AnalyzeBlock(aCtx.WithScope(fn))
	return aCtx.Scope, nil
}

// bind adds each of params to scope as a symbol of kind, as the graph analyzer does.
func bind(
	ctx context.Context,
	scope *symbol.Symbol,
	params types.Params,
	kind symbol.Kind,
) error {
	for _, p := range params {
		if _, err := scope.Add(ctx, symbol.Symbol{
			Name:         p.Name,
			Kind:         kind,
			Type:         p.Type,
			DefaultValue: p.Value,
		}); err != nil {
			return err
		}
	}
	return nil
}

// position returns the line, from 1, and the column in characters, from 0, of the byte
// at offset in text.
func position(text string, offset int) (int, int) {
	line, lineStart := 1, 0
	for i := range offset {
		if text[i] == '\n' {
			line, lineStart = line+1, i+1
		}
	}
	return line, utf8.RuneCountInString(text[lineStart:offset])
}

// StageKept stages each kept reference of res as a text site of holder
// (resource.StageTextSite), so the report of the upgrade names it.
func StageKept(
	ctx context.Context,
	tx kv.ReadWriter,
	holder resource.ID,
	res Result,
) error {
	for _, r := range res.References {
		if !r.Kept {
			continue
		}
		if err := resource.StageTextSite(ctx, tx, resource.TextSite{
			Holder:   holder,
			Name:     r.Name,
			NewName:  r.NewName,
			Line:     r.Line,
			Column:   r.Column,
			Captured: r.Captured,
		}); err != nil {
			return err
		}
	}
	return nil
}
