// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/oracle/format"
	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// MergeLive derives the canonical live-file source for a versioned resource:
// version-owned content (type membership, fields, docs, persistence tags)
// projected from the chain's current surface, live-owned content (outputs,
// language bindings, validation tags, wire-only declarations) carried from the
// existing live source. The result is formatter-canonical and carries the
// repository license header. It returns "" for a history-only chain with no
// live source.
func MergeLive(
	ctx context.Context, r *Resolver, chain Chain, source string,
) (string, error) {
	livePath := chain.LivePath()
	surf, err := r.Surface(ctx, livePath, chain.Current())
	if err != nil {
		return "", err
	}
	if len(surf) == 0 && strings.TrimSpace(source) == "" {
		return "", nil
	}
	cur, err := r.File(ctx, livePath, chain.Current())
	if err != nil {
		return "", err
	}
	lc, err := extractLive(source)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse %s.oracle", livePath)
	}

	nsToPath, err := chainNamespacePaths(ctx, r, chain)
	if err != nil {
		return "", err
	}
	opts := RenderOptions{
		KeepExpression: func(domain string, expr resolution.Expression) bool {
			return VersionOwned(domain, expr.Name) && expr.Name != "pinned"
		},
		Qualifier: func(ns string) string {
			if versionNS.MatchString(ns) || ns == chain.Resource {
				return ""
			}
			if live, _, ok := ParseDepNS(ns); ok {
				return resourceOf(live)
			}
			return ns
		},
		Resolve: func(name string) (resolution.Type, bool) {
			for _, k := range chain.Numbers {
				f, err := r.File(ctx, livePath, k)
				if err != nil {
					continue
				}
				if t, ok := f.Table.Get(name); ok {
					return t, true
				}
			}
			return resolution.Type{}, false
		},
		ExtraTypeLines: func(name string) []string {
			if ex, ok := lc.extras[name]; ok {
				return ex.typeLines
			}
			return nil
		},
		ExtraFieldLines: func(name, field string) []string {
			if ex, ok := lc.extras[name]; ok {
				return ex.fieldLines[field]
			}
			return nil
		},
		ExtraValueLines: func(name, value string) []string {
			if ex, ok := lc.extras[name]; ok {
				return ex.valueLines[value]
			}
			return nil
		},
	}

	imports := set.New(lc.imports...)
	var parts []string
	rendered := make(set.Set[string])
	renderMember := func(name string) {
		def := surf[name]
		t := def.Type
		domains := maps.Clone(t.Domains)
		if domains == nil {
			domains = make(map[string]resolution.Domain)
		}
		if def.Doc != "" {
			domains["doc"] = docDomain(def.Doc)
		} else {
			delete(domains, "doc")
		}
		// The hand marker is version-local: a type hand-written in its
		// definer's frozen package still generates in the current package as
		// a backward alias, so hand projects onto the live surface only from
		// a current-version definer.
		if def.Version != chain.Current() {
			if dom, ok := domains["go"]; ok {
				kept := make(resolution.Expressions, 0, len(dom.Expressions))
				for _, e := range dom.Expressions {
					if e.Name != "hand" {
						kept = append(kept, e)
					}
				}
				dom.Expressions = kept
				domains["go"] = dom
			}
		}
		t.Domains = domains
		for ns := range referencedNamespaces(t, opts.Resolve) {
			if versionNS.MatchString(ns) || ns == chain.Resource {
				continue
			}
			if live, _, ok := ParseDepNS(ns); ok {
				ns = resourceOf(live)
			}
			if p, ok := nsToPath[ns]; ok {
				imports.Add(p)
			}
		}
		parts = append(parts, strings.TrimSuffix(Render([]Decl{{Type: t}}, opts), "\n"))
		rendered.Add(name)
	}

	for _, name := range lc.order {
		if _, member := surf[name]; member {
			renderMember(name)
			continue
		}
		parts = append(parts, lc.raw[name])
	}
	for _, name := range cur.Order {
		if _, member := surf[name]; !member || rendered.Contains(name) {
			continue
		}
		renderMember(name)
	}

	var b strings.Builder
	importList := imports.Slice()
	slices.Sort(importList)
	for _, imp := range importList {
		b.WriteString("import \"" + imp + "\"\n")
	}
	if len(importList) > 0 {
		b.WriteString("\n")
	}
	if len(lc.fileDomains) > 0 {
		b.WriteString(strings.Join(lc.fileDomains, "\n") + "\n\n")
	}
	b.WriteString(strings.Join(parts, "\n\n") + "\n")

	formatted, err := formatter.Format(b.String())
	if err != nil {
		return "", errors.Wrapf(err, "failed to format merged %s.oracle", livePath)
	}
	l, err := format.NewLicense(r.RepoRoot())
	if err != nil {
		return "", err
	}
	out, err := l.Format(ctx, []byte(formatted), "live.oracle")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// chainNamespacePaths maps each dependency resource namespace a chain's files
// can reference to its live import path, unioned across every version file.
func chainNamespacePaths(
	ctx context.Context, r *Resolver, chain Chain,
) (map[string]string, error) {
	m := make(map[string]string)
	add := func(livePath string) error {
		ns := resourceOf(livePath)
		if existing, ok := m[ns]; ok && existing != livePath {
			return errors.Newf(
				"namespace %s is ambiguous between %s and %s", ns, existing, livePath,
			)
		}
		m[ns] = livePath
		return nil
	}
	for _, k := range chain.Numbers {
		f, err := r.File(ctx, chain.LivePath(), k)
		if err != nil {
			return nil, err
		}
		for depLive := range f.Pins {
			if err := add(depLive); err != nil {
				return nil, err
			}
		}
		for _, p := range f.LivePaths {
			if err := add(p); err != nil {
				return nil, err
			}
		}
	}
	return m, nil
}

// referencedNamespaces collects the namespace qualifiers a declaration's
// references name: field types, defaults, extends, union variants (inline
// payloads included), distinct bases, and alias targets.
func referencedNamespaces(
	t resolution.Type, resolve func(string) (resolution.Type, bool),
) set.Set[string] {
	out := make(set.Set[string])
	var walkRef func(ref resolution.TypeRef)
	var walkFields func(fields []resolution.Field)
	walkRef = func(ref resolution.TypeRef) {
		if ref.TypeParam != nil {
			return
		}
		if ns, _, found := strings.Cut(ref.Name, "."); found {
			out.Add(ns)
		}
		for _, a := range ref.TypeArgs {
			walkRef(a)
		}
	}
	var walkValue func(v resolution.ExpressionValue)
	walkValue = func(v resolution.ExpressionValue) {
		if v.Kind == resolution.ValueKindIdent {
			if ns, _, found := strings.Cut(v.IdentValue, "."); found {
				out.Add(ns)
			}
		}
		for _, e := range v.Elements {
			walkValue(e)
		}
		for _, f := range v.Fields {
			walkValue(f.Value)
		}
	}
	walkFields = func(fields []resolution.Field) {
		for _, f := range fields {
			walkRef(f.Type)
			if f.Default != nil {
				walkValue(*f.Default)
			}
		}
	}
	switch f := t.Form.(type) {
	case resolution.StructForm:
		walkFields(f.Fields)
		for _, a := range f.Actions {
			walkFields(a.Fields)
			for _, e := range a.Extends {
				walkRef(e)
			}
		}
		for _, e := range f.Extends {
			walkRef(e)
		}
		for _, p := range f.TypeParams {
			if p.Constraint != nil {
				walkRef(*p.Constraint)
			}
			if p.Default != nil {
				walkRef(*p.Default)
			}
		}
	case resolution.EnumForm:
		for _, e := range f.Extends {
			walkRef(e)
		}
	case resolution.UnionForm:
		for _, v := range f.Variants {
			if v.Inline && resolve != nil {
				if pt, ok := resolve(v.Type.Name); ok {
					if sf, isStruct := pt.Form.(resolution.StructForm); isStruct {
						walkFields(sf.Fields)
						continue
					}
				}
			}
			walkRef(v.Type)
		}
		for _, e := range f.Extends {
			walkRef(e)
		}
	case resolution.DistinctForm:
		walkRef(f.Base)
	case resolution.AliasForm:
		walkRef(f.Target)
	}
	return out
}

// docDomain synthesizes a @doc value domain.
func docDomain(doc string) resolution.Domain {
	return resolution.Domain{
		Name: "doc",
		Expressions: resolution.Expressions{{
			Name: "value",
			Values: []resolution.ExpressionValue{{
				Kind:        resolution.ValueKindString,
				StringValue: doc,
			}},
		}},
	}
}

// declExtras is one live declaration's live-owned domain-expression lines.
type declExtras struct {
	typeLines  []string
	fieldLines map[string][]string
	valueLines map[string][]string
}

// liveContent is the live-owned content extracted from an existing live file.
type liveContent struct {
	imports     []string
	fileDomains []string
	order       []string
	raw         map[string]string
	extras      map[string]*declExtras
}

// domainNode is the shared shape of the parser's domain annotation contexts.
type domainNode interface {
	IDENT() antlr.TerminalNode
	DomainContent() parser.IDomainContentContext
}

// extractLive parses live source and splits its content along the ownership
// boundary: imports, file-level domains, declaration order, verbatim
// declaration text, and per-declaration live-owned expression lines.
func extractLive(source string) (*liveContent, error) {
	lc := &liveContent{
		raw:    make(map[string]string),
		extras: make(map[string]*declExtras),
	}
	if strings.TrimSpace(source) == "" {
		return lc, nil
	}
	ast, diag := parser.Parse(source)
	if diag != nil && !diag.Ok() {
		return nil, errors.Newf("%s", diag.String())
	}
	// ANTLR input streams index by rune, so spans slice a rune view of the
	// source.
	runes := []rune(source)
	text := func(ctx antlr.ParserRuleContext) string {
		return string(runes[ctx.GetStart().GetStart() : ctx.GetStop().GetStop()+1])
	}
	for _, imp := range ast.AllImportStmt() {
		lc.imports = append(
			lc.imports, strings.Trim(imp.STRING_LIT().GetText(), `"`),
		)
	}
	for _, fd := range ast.AllFileDomain() {
		lc.fileDomains = append(lc.fileDomains, text(fd))
	}

	liveOwned := func(d domainNode, add func(line string)) {
		name := d.IDENT().GetText()
		content := d.DomainContent()
		if content == nil {
			if !VersionOwned(name, "") {
				add("@" + name)
			}
			return
		}
		if e := content.Expression(); e != nil {
			if !VersionOwned(name, e.IDENT().GetText()) {
				add("@" + name + " " + text(e))
			}
			return
		}
		if blk := content.DomainBlock(); blk != nil {
			for _, e := range blk.AllExpression() {
				if !VersionOwned(name, e.IDENT().GetText()) {
					add("@" + name + " " + text(e))
				}
			}
		}
	}

	for _, def := range ast.AllDefinition() {
		name, ok := definitionName(def)
		if !ok {
			continue
		}
		lc.order = append(lc.order, name)
		lc.raw[name] = text(def)
		ex := &declExtras{
			fieldLines: make(map[string][]string),
			valueLines: make(map[string][]string),
		}
		lc.extras[name] = ex
		addType := func(line string) { ex.typeLines = append(ex.typeLines, line) }

		collectField := func(fd parser.IFieldDefContext, into *declExtras) {
			fieldName := fd.IDENT().GetText()
			add := func(line string) {
				into.fieldLines[fieldName] = append(
					into.fieldLines[fieldName], line,
				)
			}
			for _, d := range fd.AllInlineDomain() {
				liveOwned(d, add)
			}
			if body := fd.FieldBody(); body != nil {
				for _, d := range body.AllDomain() {
					liveOwned(d, add)
				}
			}
		}

		switch {
		case def.StructDef() != nil:
			switch s := def.StructDef().(type) {
			case *parser.StructFullContext:
				body := s.StructBody()
				if body == nil {
					break
				}
				for _, fd := range body.AllFieldDef() {
					collectField(fd, ex)
				}
				for _, d := range body.AllDomain() {
					liveOwned(d, addType)
				}
				// Action extras register under the composite "Type.action"
				// key the renderer scopes them by.
				for _, ad := range body.AllActionDef() {
					aex := &declExtras{
						fieldLines: make(map[string][]string),
						valueLines: make(map[string][]string),
					}
					lc.extras[name+"."+ad.IDENT().GetText()] = aex
					ab := ad.ActionBody()
					if ab == nil {
						continue
					}
					for _, fd := range ab.AllFieldDef() {
						collectField(fd, aex)
					}
					for _, d := range ab.AllDomain() {
						liveOwned(d, func(line string) {
							aex.typeLines = append(aex.typeLines, line)
						})
					}
				}
			case *parser.StructAliasContext:
				if body := s.AliasBody(); body != nil {
					for _, d := range body.AllDomain() {
						liveOwned(d, addType)
					}
				}
			}
		case def.EnumDef() != nil:
			body := def.EnumDef().EnumBody()
			if body == nil {
				break
			}
			for _, d := range body.AllDomain() {
				liveOwned(d, addType)
			}
			for _, v := range body.AllEnumValue() {
				valueName := v.IDENT().GetText()
				add := func(line string) {
					ex.valueLines[valueName] = append(
						ex.valueLines[valueName], line,
					)
				}
				if vb := v.EnumValueBody(); vb != nil {
					for _, d := range vb.AllDomain() {
						liveOwned(d, add)
					}
				}
			}
		case def.TypeDefDef() != nil:
			if body := def.TypeDefDef().TypeDefBody(); body != nil {
				for _, d := range body.AllDomain() {
					liveOwned(d, addType)
				}
			}
		case def.UnionDef() != nil:
			if body := def.UnionDef().UnionBody(); body != nil {
				for _, d := range body.AllDomain() {
					liveOwned(d, addType)
				}
			}
		}
	}
	return lc, nil
}

// definitionName returns a top-level definition's declared name.
func definitionName(def parser.IDefinitionContext) (string, bool) {
	switch {
	case def.StructDef() != nil:
		switch s := def.StructDef().(type) {
		case *parser.StructFullContext:
			return s.IDENT().GetText(), true
		case *parser.StructAliasContext:
			return s.IDENT().GetText(), true
		}
	case def.EnumDef() != nil:
		return def.EnumDef().IDENT().GetText(), true
	case def.TypeDefDef() != nil:
		return def.TypeDefDef().IDENT().GetText(), true
	case def.UnionDef() != nil:
		return def.UnionDef().AllIDENT()[0].GetText(), true
	}
	return "", false
}
