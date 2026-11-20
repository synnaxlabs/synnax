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
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/synnaxlabs/oracle/analyzer"
	"github.com/synnaxlabs/oracle/formatter"
	"github.com/synnaxlabs/oracle/parser"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// MergeLive derives the canonical live-file source for a versioned resource:
// version-owned content (type membership, fields, docs, persistence tags) projected
// from the chain's current surface, live-owned content (outputs, language bindings,
// validation tags, wire-only declarations) carried from the existing live source. The
// result is formatter-canonical and carries the repository license header. It returns
// "" for an ended or history-only chain, whose live source is hand-owned. It returns an
// error wrapping validate.ErrValidation when the live source documents a declaration
// the chain leaves undocumented, since the merge would erase that doc.
func MergeLive(
	ctx context.Context, r *Resolver, chain Chain, source string,
) (string, error) {
	livePath := chain.LivePath()
	surf, err := r.Surface(livePath, chain.Current())
	if err != nil {
		return "", err
	}
	// An empty current surface means the chain ended (or never had a live schema):
	// nothing projects, and the live file stays hand-owned.
	if len(surf) == 0 {
		return "", nil
	}
	cur, err := r.File(livePath, chain.Current())
	if err != nil {
		return "", err
	}
	lc, err := extractLive(source)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse %s.oracle", livePath)
	}

	importsOf, err := importsByVersion(r, chain)
	if err != nil {
		return "", err
	}
	opts := RenderOptions{
		KeepExpression: func(domain string, expr resolution.Expression) bool {
			return versionOwned(domain, expr.Name)
		},
		Qualifier: func(ns string) string {
			if versionNS.MatchString(ns) {
				return ""
			}
			if live, _, ok := ParseDepNS(ns); ok {
				return resourceOf(live)
			}
			return ns
		},
		Resolve: func(name string) (resolution.Type, bool) {
			for _, k := range chain.Numbers {
				f, err := r.File(livePath, k)
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
		ExtraVariantLines: func(name, variant string) []string {
			if ex, ok := lc.extras[name]; ok {
				return ex.variantLines[variant]
			}
			return nil
		},
		ExtraBodyBlocks: func(name string) []string {
			if ex, ok := lc.extras[name]; ok {
				return ex.actionBlocks
			}
			return nil
		},
	}

	referenced := make(set.Set[string])
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
		// The hand marker is version-local: a type hand-written in its definer's frozen
		// package still generates in the current package as a backward alias, so hand
		// projects onto the live surface only from a current-version definer.
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
		for name := range referencedNames(t, opts.Resolve) {
			ns, _, _ := strings.Cut(name, ".")
			if versionNS.MatchString(ns) {
				continue
			}
			if live, _, ok := ParseDepNS(ns); ok {
				referenced.Add(live)
			} else if p, ok := importsOf[def.Version][name]; ok {
				referenced.Add(p)
			} else if p, ok := importsOf[def.Version][ns]; ok {
				referenced.Add(p)
			}
		}
		parts = append(parts, strings.TrimSuffix(Render([]Decl{{Type: t}}, opts), "\n"))
		rendered.Add(name)
	}

	liveNames := make(set.Set[string])
	useLive := func(src ...string) {
		for _, s := range src {
			liveNames.Add(analyzer.QualifiedNames(s).Slice()...)
		}
	}
	useLive(lc.fileDomains...)
	for _, ex := range lc.extras {
		useLive(ex.typeLines...)
		useLive(ex.actionBlocks...)
		for _, lines := range ex.fieldLines {
			useLive(lines...)
		}
		for _, lines := range ex.valueLines {
			useLive(lines...)
		}
	}
	for _, name := range lc.order {
		if _, member := surf[name]; member {
			renderMember(name)
			continue
		}
		useLive(lc.raw[name])
		parts = append(parts, lc.raw[name])
	}
	for _, name := range cur.Order {
		if _, member := surf[name]; !member || rendered.Contains(name) {
			continue
		}
		renderMember(name)
	}
	var body strings.Builder
	if len(lc.fileDomains) > 0 {
		body.WriteString(strings.Join(lc.fileDomains, "\n"))
		body.WriteString("\n\n")
	}
	body.WriteString(strings.Join(parts, "\n\n"))
	body.WriteString("\n")
	bodyAST, diag := parser.Parse(body.String())
	if diag != nil && !diag.Ok() {
		return "", errors.Newf(
			"failed to parse merged %s.oracle: %s", livePath, diag.String(),
		)
	}
	importList, err := liveImportList(
		lc.imports,
		referenced,
		analyzer.UsedNamespaces(bodyAST),
		liveNames,
		r.loader,
	)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	for _, imp := range importList {
		fmt.Fprintf(&b, "import %q\n", imp)
	}
	if len(importList) > 0 {
		b.WriteString("\n")
	}
	b.WriteString(body.String())

	formatted, err := formatter.Format(b.String())
	if err != nil {
		return "", errors.Wrapf(err, "failed to format merged %s.oracle", livePath)
	}
	merged, err := extractLive(formatted)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse merged %s.oracle", livePath)
	}
	if err := checkErasedDocs(chain, surf, lc, merged); err != nil {
		return "", err
	}
	return license(ctx, r.RepoRoot(), livePath+".oracle", formatted)
}

// checkErasedDocs fails when the merge dropped a live doc from a declaration that it
// wrote. It does not report a doc that differs from the chain's, or one on a restated
// inherited field: neither can be told apart from a change to the version files.
func checkErasedDocs(
	chain Chain, surf map[string]Definition, live, merged *liveContent,
) error {
	byFile := make(map[string][]string)
	for path := range live.documented {
		if !merged.declared.Contains(path) || merged.documented.Contains(path) {
			continue
		}
		name, _, _ := strings.Cut(path, ".")
		file := chain.FilePath(surf[name].Version) + ".oracle"
		byFile[file] = append(byFile[file], path)
	}
	if len(byFile) == 0 {
		return nil
	}
	lines := make([]string, 0, len(byFile))
	for _, file := range slices.Sorted(maps.Keys(byFile)) {
		slices.Sort(byFile[file])
		lines = append(lines, fmt.Sprintf(
			"move %s to %s", strings.Join(byFile[file], ", "), file,
		))
	}
	return errors.Wrapf(
		validate.ErrValidation,
		"%s.oracle: sync would erase @doc: the version files own docs, so %s, "+
			"or delete the doc from the live file",
		chain.LivePath(),
		strings.Join(lines, "; "),
	)
}

// liveImportList returns the sorted imports of a merged live file: each referenced
// path, plus each live import that the file still uses. A live import in a namespace
// that referenced paths take stays only for a name in the live text that it declares
// and they do not.
func liveImportList(
	own []string,
	referenced, used, liveNames set.Set[string],
	loader analyzer.FileLoader,
) ([]string, error) {
	takers := make(map[string][]string)
	for p := range referenced {
		ns := resourceOf(p)
		takers[ns] = append(takers[ns], p)
	}
	imports := referenced.Copy()
	for _, p := range own {
		ns := resourceOf(p)
		taken := takers[ns]
		if len(taken) == 0 {
			if used.Contains(ns) {
				imports.Add(p)
			}
			continue
		}
		var members []string
		for name := range liveNames {
			if member, ok := strings.CutPrefix(name, ns+"."); ok {
				members = append(members, member)
			}
		}
		if referenced.Contains(p) || len(members) == 0 {
			continue
		}
		mine, err := declaredNames(loader, p)
		if err != nil {
			return nil, err
		}
		theirs := make(set.Set[string])
		for _, t := range taken {
			names, err := declaredNames(loader, t)
			if err != nil {
				return nil, err
			}
			theirs.Add(names.Slice()...)
		}
		if slices.ContainsFunc(members, func(m string) bool {
			return mine.Contains(m) && !theirs.Contains(m)
		}) {
			imports.Add(p)
		}
	}
	return slices.Sorted(maps.Keys(imports)), nil
}

// declaredNames returns the names of the top-level declarations in the schema at path.
func declaredNames(loader analyzer.FileLoader, path string) (set.Set[string], error) {
	src, _, err := loader.Load(path)
	if err != nil {
		return nil, err
	}
	ast, diag := parser.Parse(src)
	if diag != nil && !diag.Ok() {
		return nil, errors.Newf("failed to parse %s.oracle: %s", path, diag.String())
	}
	names := make(set.Set[string])
	for _, def := range ast.AllDefinition() {
		if name, ok := definitionName(def); ok {
			names.Add(name)
		}
	}
	return names, nil
}

// importsByVersion maps each version of a chain to the live path of each dependency its
// file imports, keyed by resource namespace. When two imports share a namespace, it
// keys each qualified name that they declare instead. A declaration resolves the names
// it references through the imports of the file that declares it, inherited fields
// included.
func importsByVersion(r *Resolver, chain Chain) (map[int]map[string]string, error) {
	m := make(map[int]map[string]string, len(chain.Numbers))
	for _, k := range chain.Numbers {
		f, err := r.File(chain.LivePath(), k)
		if err != nil {
			return nil, err
		}
		byNS := make(map[string][]string, len(f.Pins)+len(f.LivePaths))
		for _, p := range append(slices.Collect(maps.Keys(f.Pins)), f.LivePaths...) {
			ns := resourceOf(p)
			byNS[ns] = append(byNS[ns], p)
		}
		paths := make(map[string]string, len(byNS))
		for ns, ps := range byNS {
			if len(ps) == 1 {
				paths[ns] = ps[0]
				continue
			}
			for _, p := range ps {
				names, err := importedNames(r, f, p)
				if err != nil {
					return nil, err
				}
				for name := range names {
					paths[ns+"."+name] = p
				}
			}
		}
		m[k] = paths
	}
	return m, nil
}

// importedNames returns the names that the import p of f declares: the pinned surface
// for a versioned dependency, or the live schema's declarations.
func importedNames(r *Resolver, f *File, p string) (set.Set[string], error) {
	v, pinned := f.Pins[p]
	if !pinned {
		return declaredNames(r.loader, p)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.surfaceNames(p, v)
}

// referencedNames collects the qualified names a declaration's references name: field
// types, defaults, extends, union variants (inline payloads included), distinct bases,
// and alias targets.
func referencedNames(
	t resolution.Type, resolve func(string) (resolution.Type, bool),
) set.Set[string] {
	n := qualifiedNames{out: make(set.Set[string])}
	switch f := t.Form.(type) {
	case resolution.StructForm:
		n.fields(f.Fields)
		n.refs(f.Extends)
		for _, p := range f.TypeParams {
			if p.Constraint != nil {
				n.ref(*p.Constraint)
			}
			if p.Default != nil {
				n.ref(*p.Default)
			}
		}
	case resolution.EnumForm:
		n.refs(f.Extends)
	case resolution.UnionForm:
		n.union(f, resolve)
	case resolution.DistinctForm:
		n.ref(f.Base)
	case resolution.AliasForm:
		n.ref(f.Target)
	}
	return n.out
}

// qualifiedNames collects the qualified names that references and values name.
type qualifiedNames struct{ out set.Set[string] }

func (n qualifiedNames) ref(ref resolution.TypeRef) {
	if ref.TypeParam != nil {
		return
	}
	if strings.Contains(ref.Name, ".") {
		n.out.Add(ref.Name)
	}
	n.refs(ref.TypeArgs)
}

func (n qualifiedNames) refs(refs []resolution.TypeRef) {
	for _, r := range refs {
		n.ref(r)
	}
}

func (n qualifiedNames) value(v resolution.ExpressionValue) {
	if v.Kind == resolution.ValueKindIdent && strings.Contains(v.IdentValue, ".") {
		n.out.Add(v.IdentValue)
	}
	for _, e := range v.Elements {
		n.value(e)
	}
	for _, f := range v.Fields {
		n.value(f.Value)
	}
}

func (n qualifiedNames) fields(fields []resolution.Field) {
	for _, f := range fields {
		n.ref(f.Type)
		if f.Default != nil {
			n.value(*f.Default)
		}
	}
}

// union collects the names of the union's variants, walking an inline variant's payload
// fields and extends in place of the variant.
func (n qualifiedNames) union(
	f resolution.UnionForm, resolve func(string) (resolution.Type, bool),
) {
	for _, v := range f.Variants {
		if sf, ok := inlinePayload(v, resolve); ok {
			n.fields(sf.Fields)
			n.refs(sf.Extends)
			continue
		}
		n.ref(v.Type)
	}
	n.refs(f.Extends)
	n.refs(f.Included)
	if f.BorrowsEnum() {
		n.ref(f.Enum)
	}
}

func inlinePayload(
	v resolution.UnionVariant, resolve func(string) (resolution.Type, bool),
) (resolution.StructForm, bool) {
	if !v.Inline || resolve == nil {
		return resolution.StructForm{}, false
	}
	pt, ok := resolve(v.Type.Name)
	if !ok {
		return resolution.StructForm{}, false
	}
	sf, ok := pt.Form.(resolution.StructForm)
	return sf, ok
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
	// variantLines are keyed by union variant name.
	variantLines map[string][]string
	// actionBlocks are the declaration's action definitions, verbatim. Actions are
	// live-owned: they are wire mutations, not persisted content.
	actionBlocks []string
}

// liveContent is the live-owned content extracted from an existing live file.
type liveContent struct {
	imports     []string
	fileDomains []string
	order       []string
	raw         map[string]string
	extras      map[string]*declExtras
	// declared and documented hold dotted paths ("Task", "Task.key", "Mode.on"): every
	// declaration in the source, and the ones that carry a @doc.
	declared   set.Set[string]
	documented set.Set[string]
}

// domainNode is the shared shape of the parser's domain annotation contexts.
type domainNode interface {
	IDENT() antlr.TerminalNode
	DomainContent() parser.IDomainContentContext
}

// extractLive parses live source and splits its content along the ownership boundary:
// imports, file-level domains, declaration order, verbatim declaration text, and
// per-declaration live-owned expression lines.
func extractLive(source string) (*liveContent, error) {
	lc := &liveContent{
		raw:        make(map[string]string),
		extras:     make(map[string]*declExtras),
		declared:   make(set.Set[string]),
		documented: make(set.Set[string]),
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

	liveOwned := func(d domainNode, path string, add func(line string)) {
		name := d.IDENT().GetText()
		if name == "doc" {
			lc.documented.Add(path)
		}
		content := d.DomainContent()
		if content == nil {
			if !versionOwned(name, "") {
				add("@" + name)
			}
			return
		}
		if e := content.Expression(); e != nil {
			if !versionOwned(name, e.IDENT().GetText()) {
				add("@" + name + " " + text(e))
			}
			return
		}
		if blk := content.DomainBlock(); blk != nil {
			for _, e := range blk.AllExpression() {
				if !versionOwned(name, e.IDENT().GetText()) {
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
			fieldLines:   make(map[string][]string),
			valueLines:   make(map[string][]string),
			variantLines: make(map[string][]string),
		}
		lc.extras[name] = ex
		lc.declared.Add(name)
		addType := func(line string) { ex.typeLines = append(ex.typeLines, line) }

		// collectField records a field's live-only domain lines under keyPrefix +
		// field name, the same key renderField looks them up by.
		collectField := func(
			fd parser.IFieldDefContext,
			into *declExtras,
			keyPrefix string,
		) {
			fieldName := keyPrefix + fd.IDENT().GetText()
			path := name + "." + fieldName
			lc.declared.Add(path)
			add := func(line string) {
				into.fieldLines[fieldName] = append(
					into.fieldLines[fieldName], line,
				)
			}
			for _, d := range fd.AllInlineDomain() {
				liveOwned(d, path, add)
			}
			if body := fd.FieldBody(); body != nil {
				for _, d := range body.AllDomain() {
					liveOwned(d, path, add)
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
					collectField(fd, ex, "")
				}
				for _, d := range body.AllDomain() {
					liveOwned(d, name, addType)
				}
				for _, ad := range body.AllActionDef() {
					ex.actionBlocks = append(ex.actionBlocks, text(ad))
				}
			case *parser.StructAliasContext:
				if body := s.AliasBody(); body != nil {
					for _, d := range body.AllDomain() {
						liveOwned(d, name, addType)
					}
				}
			}
		case def.EnumDef() != nil:
			body := def.EnumDef().EnumBody()
			if body == nil {
				break
			}
			for _, d := range body.AllDomain() {
				liveOwned(d, name, addType)
			}
			for _, v := range body.AllEnumValue() {
				valueName := v.IDENT().GetText()
				path := name + "." + valueName
				lc.declared.Add(path)
				add := func(line string) {
					ex.valueLines[valueName] = append(
						ex.valueLines[valueName], line,
					)
				}
				if vb := v.EnumValueBody(); vb != nil {
					for _, d := range vb.AllDomain() {
						liveOwned(d, path, add)
					}
				}
			}
		case def.TypeDefDef() != nil:
			if body := def.TypeDefDef().TypeDefBody(); body != nil {
				for _, d := range body.AllDomain() {
					liveOwned(d, name, addType)
				}
			}
		case def.UnionDef() != nil:
			if body := def.UnionDef().UnionBody(); body != nil {
				for _, d := range body.AllDomain() {
					liveOwned(d, name, addType)
				}
				for _, ad := range body.AllActionDef() {
					ex.actionBlocks = append(ex.actionBlocks, text(ad))
				}
				for _, v := range body.AllUnionVariant() {
					var variant string
					add := func(line string) {
						ex.variantLines[variant] = append(
							ex.variantLines[variant], line,
						)
					}
					switch uv := v.(type) {
					case *parser.NamedVariantContext:
						variant = uv.VariantName().GetText()
						path := name + "." + variant
						lc.declared.Add(path)
						if vb := uv.UnionVariantBody(); vb != nil {
							for _, d := range vb.AllDomain() {
								liveOwned(d, path, add)
							}
						}
					case *parser.InlineVariantContext:
						variant = uv.VariantName().GetText()
						path := name + "." + variant
						lc.declared.Add(path)
						if uv.StructBody() == nil {
							continue
						}
						for _, d := range uv.StructBody().AllDomain() {
							liveOwned(d, path, add)
						}
						for _, fd := range uv.StructBody().AllFieldDef() {
							collectField(fd, ex, uv.VariantName().GetText()+".")
						}
					}
				}
			}
		case def.FilterDef() != nil:
			if body := def.FilterDef().FilterBody(); body != nil {
				for _, fd := range body.AllFieldDef() {
					collectField(fd, ex, "")
				}
				for _, d := range body.AllDomain() {
					liveOwned(d, name, addType)
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
	case def.FilterDef() != nil:
		return def.FilterDef().AllIDENT()[0].GetText(), true
	}
	return "", false
}
