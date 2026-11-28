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
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/arc/analyzer"
	"github.com/synnaxlabs/arc/analyzer/authority"
	"github.com/synnaxlabs/arc/analyzer/codes"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/statement"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/arc/types"
	resource "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/diagnostics"
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
	// after holds the key of each renamed channel by its new name.
	after map[string]Key
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
	return r.symbol(name, b.key), nil
}

// symbol returns the channel symbol of key under name.
func (r *Renames) symbol(name string, key Key) *symbol.Symbol {
	return &symbol.Symbol{
		Name: name,
		Kind: symbol.KindChannel,
		Type: types.Chan(types.FromTelem(r.channels[key].DataType)),
		ID:   int(key),
	}
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
	// a FormBody text. Functions are every function of its graph in order, and Index
	// the position of that function in them. With no Functions, the graph holds that
	// function alone.
	Function        string
	Inputs, Outputs types.Params
	Functions       []GraphFunction
	Index           int
}

// GraphFunction is a function of the graph of a FormBody text.
type GraphFunction struct {
	Key string
	// Bodiless is true when the function has no body yet, which a v0.62 Core declared
	// with no source.
	Bodiless bool
}

// Reference is a bare identifier in stored text that the analyzer reads as the old name
// of a renamed channel. A key, a unit, and the name of a declaration are not
// references.
type Reference struct {
	// Name is the old name, and NewName the new name of the channel.
	Name, NewName string
	// Line counts from 1, and Column counts characters from 0.
	Line, Column int
	// Binding is the kind of the symbol that Name resolves to at the reference. Bound
	// is false when it resolves to none, or the text does not parse or bind.
	Binding symbol.Kind
	Bound   bool
	// Captured is true when the reference reads the channel, but a symbol that the text
	// declares takes the old or the new name there: in the scopes that the analyzer has
	// bound when it reads the reference, or in the finished scopes.
	Captured bool
	// Kept is true when the rewrite left the reference on the old name although it may
	// read the channel: the text does not parse or bind, or a reference of it is
	// captured.
	Kept bool
}

// Result is what a rewrite made of a stored text.
type Result struct {
	// Text is the rewritten text. It is the stored text when a reference is kept.
	Text string
	// Parsed is false when the text does not parse, or a v0.62 Core could not bind it.
	Parsed bool
	// References are the references of the text, in order. When the text does not
	// parse or bind, they are each bare identifier that holds the old name of a renamed
	// channel.
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
// becomes the new name of the channel. An identifier reads the channel when the
// analyzer reads it as a name, and the channel takes it both in the scopes that the
// analyzer has bound at that point and in the finished scopes that the compiler reads.
// A key, a unit, and the head or a member of a qualified name are never references.
// When src does not parse or bind, or a symbol that the text declares takes the old or
// the new name at a reference, Rewrite leaves the text unchanged and marks each
// reference that it could not write as kept.
func (w Rewriter) Rewrite(ctx context.Context, src Source) (Result, error) {
	candidates, spans := w.find(src.Text)
	base, parsed, err := w.parse(ctx, src, src.Text, names{renames: w.Renames})
	if err != nil {
		return Result{}, err
	}
	res := Result{Text: src.Text, Parsed: parsed}
	if len(candidates) == 0 {
		return res, nil
	}
	if !parsed {
		for i := range candidates {
			candidates[i].Kept = true
		}
		res.References = candidates
		return res, nil
	}
	base.analyze()
	var (
		written  []span
		captured bool
	)
	reads, err := w.readsOf(ctx, src, candidates, spans)
	if err != nil {
		return Result{}, err
	}
	for i, r := range candidates {
		if len(reads[i]) == 0 {
			continue
		}
		if r, err = bind(ctx, base, r, reads[i]); err != nil {
			return Result{}, err
		}
		res.References = append(res.References, r)
		written = append(written, spans[i])
		captured = captured || r.Captured
	}
	if captured {
		for i, r := range res.References {
			res.References[i].Kept = r.Bound && r.Binding == symbol.KindChannel
		}
		return res, nil
	}
	res.Text = write(src.Text, res.References, written)
	return res, nil
}

// write returns text with the new name in place of each of refs that reads the channel
// and is not kept, at its span in spans.
func write(text string, refs []Reference, spans []span) string {
	var b strings.Builder
	last := 0
	for i, r := range refs {
		if r.Bound && r.Binding == symbol.KindChannel && !r.Kept {
			b.WriteString(text[last:spans[i].start])
			b.WriteString(r.NewName)
			last = spans[i].end
		}
	}
	b.WriteString(text[last:])
	return b.String()
}

// span is the byte range [start, end) of a reference in a text.
type span struct{ start, end int }

// find returns each bare identifier of text that holds the old name of a renamed
// channel, and the span of each.
func (w Rewriter) find(text string) ([]Reference, []span) {
	var (
		refs  []Reference
		spans []span
	)
	identity := func(i int) int { return i }
	identifiers(text, w.Config, identity, func(start, end int, name string) {
		to, ok := w.Renames.Rename(name)
		if !ok {
			return
		}
		line, column := position(text, start)
		refs = append(refs, Reference{
			Name:    name,
			NewName: to,
			Line:    line,
			Column:  column,
		})
		spans = append(spans, span{start: start, end: end})
	})
	return refs, spans
}

// read is how the analyzer read a reference once: the symbols that the old and the new
// name of its channel took in the scopes bound at that point, or nil for none.
type read struct{ old, new *symbol.Symbol }

// reads returns each read of r, the reference at sp in the text of src, by the
// analyzer. It analyzes the text with a probe, a name that no symbol of the text holds,
// in place of r, and looks up both names each time that the analyzer resolves the
// probe. It returns no reads when the analyzer does not read r as a name, such as a key
// or a unit, or when r is the name of a declaration.
func (w Rewriter) reads(
	ctx context.Context,
	src Source,
	sp span,
	r Reference,
) ([]read, error) {
	var (
		p       parsed
		reads   []read
		readErr error
	)
	pr := &probe{name: probeName(src.Text), old: r.Name}
	pr.read = func(ctx context.Context) {
		at := symbol.ScopeAt(p.scope, r.Line, p.column(r))
		old, err := lookup(ctx, at, r.Name)
		if err != nil {
			readErr = err
			return
		}
		renamed, err := lookup(ctx, at, r.NewName)
		if err != nil {
			readErr = err
			return
		}
		reads = append(reads, read{old: old, new: renamed})
	}
	probed := src.Text[:sp.start] + pr.name + src.Text[sp.end:]
	p, parsed, err := w.parse(
		ctx,
		src,
		probed,
		names{renames: w.Renames, probes: map[string]*probe{pr.name: pr}},
	)
	if err != nil || !parsed {
		return nil, err
	}
	p.analyze()
	if readErr != nil {
		return nil, readErr
	}
	if declares(p.scope, pr.name) {
		return nil, nil
	}
	return reads, nil
}

// readsOf returns the reads of each of candidates, at spans in the text of src, as
// reads does for one: none for a declaration. For several candidates, one analysis with
// each probed finds the declarations. Each further analysis probes the other candidates
// of one group, which holds at most one candidate of each line, so that each probe
// keeps the column of its reference. A lone candidate, or a probed text that does not
// parse, falls back to reads.
func (w Rewriter) readsOf(
	ctx context.Context,
	src Source,
	candidates []Reference,
	spans []span,
) ([][]read, error) {
	out := make([][]read, len(candidates))
	var (
		declared []bool
		ok       bool
		err      error
	)
	if len(candidates) > 1 {
		if declared, ok, err = w.declarations(ctx, src, candidates, spans); err != nil {
			return nil, err
		}
	}
	if !ok {
		for i, r := range candidates {
			if out[i], err = w.reads(ctx, src, spans[i], r); err != nil {
				return nil, err
			}
		}
		return out, nil
	}
	for _, group := range lineGroups(candidates, declared) {
		got, ok, err := w.groupReads(ctx, src, candidates, spans, group)
		if err != nil {
			return nil, err
		}
		for j, i := range group {
			if ok {
				out[i] = got[j]
			} else if out[i], err = w.reads(ctx, src, spans[i], candidates[i]); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// declarations reports which of candidates, at spans in the text of src, the text
// declares, from one analysis with a probe in place of each. It reports false when the
// probed text does not parse.
func (w Rewriter) declarations(
	ctx context.Context,
	src Source,
	candidates []Reference,
	spans []span,
) ([]bool, bool, error) {
	all := make([]int, len(candidates))
	for i := range all {
		all[i] = i
	}
	probed, probes := probeAll(src.Text, candidates, spans, all, nil)
	p, parsed, err := w.parse(
		ctx,
		src,
		probed,
		names{renames: w.Renames, probes: probes},
	)
	if err != nil || !parsed {
		return nil, false, err
	}
	p.analyze()
	declared := make([]bool, len(candidates))
	for name, pr := range probes {
		declared[pr.index] = declares(p.scope, name)
	}
	return declared, true, nil
}

// groupReads returns the reads of each candidate of group, from one analysis with a
// probe in place of each. It reports false when the probed text does not parse.
func (w Rewriter) groupReads(
	ctx context.Context,
	src Source,
	candidates []Reference,
	spans []span,
	group []int,
) ([][]read, bool, error) {
	var (
		p       parsed
		readErr error
		got     = make([][]read, len(group))
	)
	onRead := func(j int) func(context.Context) {
		r := candidates[group[j]]
		return func(ctx context.Context) {
			at := symbol.ScopeAt(p.scope, r.Line, p.column(r))
			old, err := lookup(ctx, at, r.Name)
			if err != nil {
				readErr = err
				return
			}
			renamed, err := lookup(ctx, at, r.NewName)
			if err != nil {
				readErr = err
				return
			}
			got[j] = append(got[j], read{old: old, new: renamed})
		}
	}
	probed, probes := probeAll(src.Text, candidates, spans, group, onRead)
	p, parsed, err := w.parse(
		ctx,
		src,
		probed,
		names{renames: w.Renames, probes: probes},
	)
	if err != nil || !parsed {
		return nil, false, err
	}
	p.analyze()
	if readErr != nil {
		return nil, false, readErr
	}
	return got, true, nil
}

// probeAll returns text with a probe in place of each candidate at the positions in
// group, and the probes by name. onRead, when not nil, returns what the probe of the
// jth candidate of group calls each time that the analysis resolves it.
func probeAll(
	text string,
	candidates []Reference,
	spans []span,
	group []int,
	onRead func(j int) func(context.Context),
) (string, map[string]*probe) {
	base := probeName(text)
	probes := make(map[string]*probe, len(group))
	var b strings.Builder
	last := 0
	for j, i := range group {
		pr := &probe{name: base + strconv.Itoa(j), old: candidates[i].Name, index: i}
		pr.read = func(context.Context) {}
		if onRead != nil {
			pr.read = onRead(j)
		}
		probes[pr.name] = pr
		b.WriteString(text[last:spans[i].start])
		b.WriteString(pr.name)
		last = spans[i].end
	}
	b.WriteString(text[last:])
	return b.String(), probes
}

// lineGroups returns the positions in candidates of each one that is not declared, in
// groups that each hold at most one candidate of a line, in order.
func lineGroups(candidates []Reference, declared []bool) [][]int {
	var (
		groups [][]int
		onLine = make(map[int]int)
	)
	for i, c := range candidates {
		if declared[i] {
			continue
		}
		k := onLine[c.Line]
		onLine[c.Line] = k + 1
		if k == len(groups) {
			groups = append(groups, nil)
		}
		groups[k] = append(groups[k], i)
	}
	return groups
}

// declares reports whether a symbol in the scopes under scope holds name.
func declares(scope *symbol.Symbol, name string) bool {
	return slices.ContainsFunc(scope.Children(), func(child *symbol.Symbol) bool {
		return child.Name == name || declares(child, name)
	})
}

// bind returns r with how the channel reads it, given reads, each read of r by the
// analyzer, and base, the analyzed text. r reads the channel when the channel takes the
// old name in each read and in the finished scope at r. It is captured when either
// takes the old name, or when a symbol that the text declares takes the new name in
// either.
func bind(
	ctx context.Context,
	base parsed,
	r Reference,
	reads []read,
) (Reference, error) {
	at := symbol.ScopeAt(base.scope, r.Line, base.column(r))
	old, err := lookup(ctx, at, r.Name)
	if err != nil {
		return Reference{}, err
	}
	renamed, err := lookup(ctx, at, r.NewName)
	if err != nil {
		return Reference{}, err
	}
	all, some := isChannel(old), isChannel(old)
	declared := renamed != nil && !isChannel(renamed)
	for _, rd := range reads {
		all = all && isChannel(rd.old)
		some = some || isChannel(rd.old)
		declared = declared || rd.new != nil && !isChannel(rd.new)
	}
	switch {
	case !some:
		if old != nil {
			r.Binding, r.Bound = old.Kind, true
		}
	case all && !declared:
		r.Binding, r.Bound = symbol.KindChannel, true
	default:
		r.Binding, r.Bound, r.Captured = symbol.KindChannel, true, true
	}
	return r, nil
}

func isChannel(sym *symbol.Symbol) bool {
	return sym != nil && sym.Kind == symbol.KindChannel
}

// lookup returns the symbol that name takes in the scopes of at, or nil when none
// holds it.
func lookup(
	ctx context.Context,
	at *symbol.Symbol,
	name string,
) (*symbol.Symbol, error) {
	sym, err := at.Resolve(ctx, name, symbol.WithoutUsageTracking)
	if _, undefined := errors.AsType[*symbol.UndefinedSymbolError](err); undefined {
		return nil, nil
	}
	return sym, err
}

// probeName returns a name that text does not hold.
func probeName(text string) string {
	name := "sy_probe"
	for strings.Contains(text, name) {
		name += "_"
	}
	return name
}

// names resolves the names of a text as the text passes read them: each v0.62 name to
// the channel that a v0.62 Core bound it to, and the new name of each renamed channel
// to that channel.
type names struct {
	renames *Renames
	// probes stand in for references, by name.
	probes map[string]*probe
}

var _ symbol.Resolver = names{}

// probe is a name that stands in for one reference of the old name of a channel, the
// candidate at index, and read is called each time that the analysis resolves it.
type probe struct {
	name, old string
	index     int
	read      func(context.Context)
}

// Resolve implements symbol.Resolver.
func (n names) Resolve(ctx context.Context, name string) (*symbol.Symbol, error) {
	if pr, ok := n.probes[name]; ok {
		pr.read(ctx)
		name = pr.old
	}
	sym, err := n.renames.Resolve(ctx, name)
	if !errors.Is(err, query.ErrNotFound) {
		return sym, err
	}
	if key, ok := n.renames.after[name]; ok {
		return n.renames.symbol(name, key), nil
	}
	return nil, err
}

// Search implements symbol.Resolver. It finds nothing, since no pass completes names.
func (names) Search(context.Context, string) ([]*symbol.Symbol, error) {
	return nil, nil
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

// parsed is a text that parses, with the scope that holds it, ready to analyze.
type parsed struct {
	scope *symbol.Symbol
	// shift is how far the columns of the first line move in scope.
	shift   int
	analyze func()
}

// column returns the column of r in the scope of p.
func (p parsed) column(r Reference) int {
	if r.Line == 1 {
		return r.Column + p.shift
	}
	return r.Column
}

// parse parses raw, the text of src or a probe of it, as a v0.62 Core parsed src, and
// binds the scope that holds it under a root that resolves channels with names. It
// reports false when raw does not parse, or when a v0.62 Core could not bind the graph
// function of src.
func (w Rewriter) parse(
	ctx context.Context,
	src Source,
	raw string,
	names names,
) (parsed, bool, error) {
	root := symbol.NewRoot(names, w.Ambient)
	switch src.Form {
	case FormCalculation:
		if block, diag := parser.ParseBlock("{"+raw+"}", w.Config); diag == nil {
			aCtx := acontext.NewRoot(ctx, block, root).WithConfig(w.Config)
			return parsed{
				scope:   root,
				shift:   1,
				analyze: func() { statement.AnalyzeFunctionBody(aCtx) },
			}, true, nil
		}
	case FormBody:
		if block, diag := parser.ParseBlock(raw, w.Config); diag == nil {
			return w.parseBody(ctx, root, block, src)
		}
	case FormProgram:
		if t, diag := text.Parse(text.Text{Raw: raw}, w.Config); diag == nil {
			aCtx := acontext.NewRoot(ctx, t.AST, root).WithConfig(w.Config)
			return parsed{scope: root, analyze: func() {
				analyzer.AnalyzeProgram(aCtx)
				authority.Analyze(aCtx)
			}}, true, nil
		}
	default:
		return parsed{}, false, errors.Newf("unknown text form %d", src.Form)
	}
	return parsed{}, false, nil
}

// parseBody binds the graph function of src, whose body is block, under root as a v0.62
// Core bound a graph: with each module imported, and each function of the graph
// declared in order with its inputs and outputs. It reports false when a v0.62 Core
// could not bind the graph either, such as an input named like an earlier function.
func (w Rewriter) parseBody(
	ctx context.Context,
	root *symbol.Symbol,
	block parser.IBlockContext,
	src Source,
) (parsed, bool, error) {
	symbol.AutoImportModules(root)
	// The pass reads one body, so each other function declares an empty one.
	other, diag := parser.ParseBlock("{}", w.Config)
	if diag != nil {
		return parsed{}, false, diag
	}
	fns, index := src.Functions, src.Index
	if len(fns) == 0 {
		fns, index = []GraphFunction{{Key: src.Function}}, 0
	}
	var fn *symbol.Symbol
	for i, f := range fns {
		decl := symbol.Symbol{Name: f.Key, Kind: symbol.KindFunction}
		own := i == index
		switch {
		case own:
			decl.AST = block
		case !f.Bodiless:
			decl.AST = other
		}
		added, err := root.Add(ctx, decl)
		if err == nil && own {
			fn = added
			if err = bindParams(ctx, fn, src.Inputs, symbol.KindInput); err == nil {
				err = bindParams(ctx, fn, src.Outputs, symbol.KindOutput)
			}
		}
		if err != nil {
			return parsed{}, false, unlessConflict(err, block)
		}
	}
	aCtx := acontext.NewRoot(ctx, block, root).WithConfig(w.Config).WithScope(fn)
	return parsed{scope: fn, analyze: func() { analyzer.AnalyzeBlock(aCtx) }}, true, nil
}

// unlessConflict returns err, or nil when err is the error of a declaration that reuses
// a name.
func unlessConflict(err error, node antlr.ParserRuleContext) error {
	if diagnostics.Error(err, node).Code == codes.SymbolRedefined.Code {
		return nil
	}
	return err
}

// bindParams adds each of params to scope as a symbol of kind, as the graph analyzer
// does.
func bindParams(
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

// StageKept stages each kept reference of res, in the graph function named function or
// in no function when it is empty, as a text site of holder (resource.StageTextSite),
// so the report of the upgrade names it.
func StageKept(
	ctx context.Context,
	tx kv.ReadWriter,
	holder resource.ID,
	function string,
	res Result,
) error {
	for _, r := range res.References {
		if !r.Kept {
			continue
		}
		if err := resource.StageTextSite(ctx, tx, resource.TextSite{
			Holder:   holder,
			Function: function,
			Name:     r.Name,
			NewName:  r.NewName,
			Line:     r.Line,
			Column:   r.Column,
			Captured: r.Captured,
			// A bound reference that is not captured is kept because another one is.
			BesideCapture: r.Bound && !r.Captured,
		}); err != nil {
			return err
		}
	}
	return nil
}
