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
// the chain leaves undocumented, or gives a resource type the chain lacks, since the
// merge would erase it.
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
	m := &liveMerger{
		chain:      chain,
		surf:       surf,
		importsOf:  importsOf,
		opts:       mergeRenderOptions(r, chain, lc),
		referenced: make(set.Set[string]),
		rendered:   make(set.Set[string]),
	}
	liveNames := liveNamesOf(lc)
	for _, name := range lc.order {
		if _, member := surf[name]; member {
			m.renderMember(name)
			continue
		}
		liveNames.Add(analyzer.QualifiedNames(lc.raw[name]).Slice()...)
		m.parts = append(m.parts, lc.raw[name])
	}
	for _, name := range cur.Order {
		if _, member := surf[name]; !member || m.rendered.Contains(name) {
			continue
		}
		m.renderMember(name)
	}
	formatted, err := m.format(r, lc, liveNames)
	if err != nil {
		return "", err
	}
	merged, err := extractLive(formatted)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse merged %s.oracle", livePath)
	}
	if err := checkErased(chain, surf, lc, merged); err != nil {
		return "", err
	}
	return license(ctx, r.RepoRoot(), livePath+".oracle", formatted)
}

// mergeRenderOptions returns the options that render a chain's surface into its live
// file, carrying the live-owned lines of lc.
func mergeRenderOptions(r *Resolver, chain Chain, lc *liveContent) RenderOptions {
	livePath := chain.LivePath()
	return RenderOptions{
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
}

// liveNamesOf returns the qualified names that the live-owned lines of lc use.
func liveNamesOf(lc *liveContent) set.Set[string] {
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
	return liveNames
}

// liveMerger accumulates the declarations of a merged live file.
type liveMerger struct {
	chain      Chain
	surf       map[string]Definition
	importsOf  map[int]map[string]string
	opts       RenderOptions
	referenced set.Set[string]
	rendered   set.Set[string]
	parts      []string
}

func (m *liveMerger) renderMember(name string) {
	def := m.surf[name]
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
	if def.Version != m.chain.Current() {
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
	for name := range referencedNames(t, m.opts.Resolve) {
		m.addReference(def.Version, name)
	}
	m.parts = append(
		m.parts, strings.TrimSuffix(Render([]Decl{{Type: t}}, m.opts), "\n"),
	)
	m.rendered.Add(name)
}

// addReference records the live path that a name referenced from version imports.
func (m *liveMerger) addReference(version int, name string) {
	ns, _, _ := strings.Cut(name, ".")
	if versionNS.MatchString(ns) {
		return
	}
	if live, _, ok := ParseDepNS(ns); ok {
		m.referenced.Add(live)
	} else if p, ok := m.importsOf[version][name]; ok {
		m.referenced.Add(p)
	} else if p, ok := m.importsOf[version][ns]; ok {
		m.referenced.Add(p)
	}
}

// format joins the file domains of lc and the merged declarations under their imports,
// and returns the formatted source.
func (m *liveMerger) format(
	r *Resolver, lc *liveContent, liveNames set.Set[string],
) (string, error) {
	livePath := m.chain.LivePath()
	var body strings.Builder
	if len(lc.fileDomains) > 0 {
		body.WriteString(strings.Join(lc.fileDomains, "\n"))
		body.WriteString("\n\n")
	}
	body.WriteString(strings.Join(m.parts, "\n\n"))
	body.WriteString("\n")
	bodyAST, diag := parser.Parse(body.String())
	if diag != nil && !diag.Ok() {
		return "", errors.Newf(
			"failed to parse merged %s.oracle: %s", livePath, diag.String(),
		)
	}
	importList, err := liveImportList(
		lc.imports,
		m.referenced,
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
	return formatted, nil
}

// checkErased fails when the merge dropped a live @doc or @resource from a declaration
// that it wrote. It does not report one that differs from the chain's, or a doc on a
// restated inherited field: neither can be told apart from a change to the version
// files.
func checkErased(
	chain Chain, surf map[string]Definition, live, merged *liveContent,
) error {
	byFile := make(map[string][]string)
	collect := func(domain string, liveSet, mergedSet set.Set[string]) {
		for path := range liveSet {
			if !merged.declared.Contains(path) || mergedSet.Contains(path) {
				continue
			}
			name, _, _ := strings.Cut(path, ".")
			file := chain.FilePath(surf[name].Version) + ".oracle"
			byFile[file] = append(byFile[file], "@"+domain+" of "+path)
		}
	}
	collect("doc", live.documented, merged.documented)
	collect("resource", live.typed, merged.typed)
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
		"%s.oracle: sync would erase version-owned annotations: %s, "+
			"or delete them from the live file",
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
		if f.Narrows() {
			n.ref(f.Superset)
		}
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
	// declared, documented, and typed hold dotted paths ("Task", "Task.key",
	// "Mode.on"): every declaration in the source, the ones that carry a @doc, and the
	// ones that carry a @resource.
	declared   set.Set[string]
	documented set.Set[string]
	typed      set.Set[string]
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
		typed:      make(set.Set[string]),
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
	x := &liveExtractor{lc: lc, runes: []rune(source)}
	for _, imp := range ast.AllImportStmt() {
		lc.imports = append(
			lc.imports, strings.Trim(imp.STRING_LIT().GetText(), `"`),
		)
	}
	for _, fd := range ast.AllFileDomain() {
		lc.fileDomains = append(lc.fileDomains, x.text(fd))
	}
	for _, def := range ast.AllDefinition() {
		x.definition(def)
	}
	return lc, nil
}

// liveExtractor collects one live source's content into lc.
type liveExtractor struct {
	lc    *liveContent
	runes []rune
}

func (x *liveExtractor) text(ctx antlr.ParserRuleContext) string {
	return string(x.runes[ctx.GetStart().GetStart() : ctx.GetStop().GetStop()+1])
}

// liveOwned passes each live-owned expression line of d to add, and records on path
// whether d is a doc or a resource type.
func (x *liveExtractor) liveOwned(d domainNode, path string, add func(line string)) {
	name := d.IDENT().GetText()
	switch name {
	case "doc":
		x.lc.documented.Add(path)
	case "resource":
		x.lc.typed.Add(path)
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
			add("@" + name + " " + x.text(e))
		}
		return
	}
	if blk := content.DomainBlock(); blk != nil {
		for _, e := range blk.AllExpression() {
			if !versionOwned(name, e.IDENT().GetText()) {
				add("@" + name + " " + x.text(e))
			}
		}
	}
}

func (x *liveExtractor) liveOwnedAll(
	domains []parser.IDomainContext, path string, add func(line string),
) {
	for _, d := range domains {
		x.liveOwned(d, path, add)
	}
}

// field records a field's live-only domain lines under keyPrefix + field name, the same
// key renderField looks them up by.
func (x *liveExtractor) field(
	name string, fd parser.IFieldDefContext, ex *declExtras, keyPrefix string,
) {
	fieldName := keyPrefix + fd.IDENT().GetText()
	path := name + "." + fieldName
	x.lc.declared.Add(path)
	add := func(line string) {
		ex.fieldLines[fieldName] = append(ex.fieldLines[fieldName], line)
	}
	for _, d := range fd.AllInlineDomain() {
		x.liveOwned(d, path, add)
	}
	if body := fd.FieldBody(); body != nil {
		x.liveOwnedAll(body.AllDomain(), path, add)
	}
}

func (x *liveExtractor) definition(def parser.IDefinitionContext) {
	name, ok := definitionName(def)
	if !ok {
		return
	}
	x.lc.order = append(x.lc.order, name)
	x.lc.raw[name] = x.text(def)
	ex := &declExtras{
		fieldLines:   make(map[string][]string),
		valueLines:   make(map[string][]string),
		variantLines: make(map[string][]string),
	}
	x.lc.extras[name] = ex
	x.lc.declared.Add(name)
	addType := func(line string) { ex.typeLines = append(ex.typeLines, line) }
	switch {
	case def.StructDef() != nil:
		x.structDef(name, def.StructDef(), ex, addType)
	case def.EnumDef() != nil:
		x.enumDef(name, def.EnumDef(), ex, addType)
	case def.TypeDefDef() != nil:
		if body := def.TypeDefDef().TypeDefBody(); body != nil {
			x.liveOwnedAll(body.AllDomain(), name, addType)
		}
	case def.UnionDef() != nil:
		if body := def.UnionDef().UnionBody(); body != nil {
			x.unionBody(name, body, ex, addType)
		}
	case def.FilterDef() != nil:
		if body := def.FilterDef().FilterBody(); body != nil {
			for _, fd := range body.AllFieldDef() {
				x.field(name, fd, ex, "")
			}
			x.liveOwnedAll(body.AllDomain(), name, addType)
		}
	}
}

func (x *liveExtractor) structDef(
	name string, def parser.IStructDefContext, ex *declExtras, addType func(string),
) {
	switch s := def.(type) {
	case *parser.StructFullContext:
		body := s.StructBody()
		if body == nil {
			return
		}
		for _, fd := range body.AllFieldDef() {
			x.field(name, fd, ex, "")
		}
		x.liveOwnedAll(body.AllDomain(), name, addType)
		for _, ad := range body.AllActionDef() {
			ex.actionBlocks = append(ex.actionBlocks, x.text(ad))
		}
	case *parser.StructAliasContext:
		if body := s.AliasBody(); body != nil {
			x.liveOwnedAll(body.AllDomain(), name, addType)
		}
	}
}

func (x *liveExtractor) enumDef(
	name string, def parser.IEnumDefContext, ex *declExtras, addType func(string),
) {
	body := def.EnumBody()
	if body == nil {
		return
	}
	x.liveOwnedAll(body.AllDomain(), name, addType)
	for _, v := range body.AllEnumValue() {
		valueName := v.IDENT().GetText()
		path := name + "." + valueName
		x.lc.declared.Add(path)
		add := func(line string) {
			ex.valueLines[valueName] = append(ex.valueLines[valueName], line)
		}
		if vb := v.EnumValueBody(); vb != nil {
			x.liveOwnedAll(vb.AllDomain(), path, add)
		}
	}
}

func (x *liveExtractor) unionBody(
	name string, body parser.IUnionBodyContext, ex *declExtras, addType func(string),
) {
	x.liveOwnedAll(body.AllDomain(), name, addType)
	for _, ad := range body.AllActionDef() {
		ex.actionBlocks = append(ex.actionBlocks, x.text(ad))
	}
	for _, v := range body.AllUnionVariant() {
		x.unionVariant(name, v, ex)
	}
}

func (x *liveExtractor) unionVariant(
	name string, v parser.IUnionVariantContext, ex *declExtras,
) {
	var variant string
	add := func(line string) {
		ex.variantLines[variant] = append(ex.variantLines[variant], line)
	}
	switch uv := v.(type) {
	case *parser.NamedVariantContext:
		variant = uv.VariantName().GetText()
		path := name + "." + variant
		x.lc.declared.Add(path)
		if vb := uv.UnionVariantBody(); vb != nil {
			x.liveOwnedAll(vb.AllDomain(), path, add)
		}
	case *parser.InlineVariantContext:
		variant = uv.VariantName().GetText()
		path := name + "." + variant
		x.lc.declared.Add(path)
		if uv.StructBody() == nil {
			return
		}
		x.liveOwnedAll(uv.StructBody().AllDomain(), path, add)
		for _, fd := range uv.StructBody().AllFieldDef() {
			x.field(name, fd, ex, uv.VariantName().GetText()+".")
		}
	}
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
		return def.EnumDef().IDENT(0).GetText(), true
	case def.TypeDefDef() != nil:
		return def.TypeDefDef().IDENT().GetText(), true
	case def.UnionDef() != nil:
		return def.UnionDef().AllIDENT()[0].GetText(), true
	case def.FilterDef() != nil:
		return def.FilterDef().AllIDENT()[0].GetText(), true
	}
	return "", false
}
