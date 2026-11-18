// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"bytes"
	"cmp"
	"fmt"
	"math"
	"slices"
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/plugin/go/internal/typemap"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	goprimitives "github.com/synnaxlabs/oracle/plugin/go/primitives"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// fieldDocIndent is the display width of the single tab indenting field-level doc
// comments (tabs render at four columns per .editorconfig).
const fieldDocIndent = 4

// primitiveMapper is the Go-specific primitive type mapper.
var primitiveMapper = goprimitives.Mapper()

// Plugin generates Go type definitions from Oracle schema definitions.
type Plugin struct{ Options Options }

// Options configures the go/types plugin.
type Options struct {
	// FileNamePattern is the filename pattern for generated type files.
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		FileNamePattern: "types.gen.go",
	}
}

// New creates a new go/types plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "go/types" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"go"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return nil }

// Generate produces Go type definitions for structs, enums, and typedefs with @go flag.
// Version-laid-out packages (chain-covered output paths) emit their types into the
// current versions/vN sub-package, with a root alias file re-exporting the surface.
// Types whose shape is unchanged from the predecessor version alias it instead of being
// re-defined. Version packages resolve every dependency reference, omitted fields and
// transient declarations included, at the dependency version their version file pins,
// as a frozen package does, so a freeze never moves them. Transient declarations at a
// versioned root path resolve to dependency roots. Unversioned packages reference the
// root re-export surface.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	rewritten, pathMap, members, err := versioning.RewriteCurrent(
		req.Resolutions, req.Versions,
	)
	if err != nil {
		return nil, err
	}
	embedded, err := allEmbeddedTypes(req, rewritten)
	if err != nil {
		return nil, err
	}
	versionedReq := *req
	versionedReq.Resolutions = rewritten
	g := generation{
		req:       req,
		versioned: &versionedReq,
		pattern:   p.Options.FileNamePattern,
		pathMap:   pathMap,
		members:   members,
		closure:   PersistedClosure(rewritten),
		embedded:  embedded,
	}
	resp, err := g.versionPackages()
	if err != nil {
		return nil, err
	}
	unversionedResp, err := g.generator(
		g.pattern,
		&goFileGenerator{embedded: embedded},
		func(outputPath string) bool { return !g.isVersioned(outputPath) },
	).Generate(req)
	if err != nil {
		return nil, err
	}
	resp.Files = append(resp.Files, unversionedResp.Files...)
	// Ended chains have no current-surface entries, so pathMap can be empty while
	// frozen packages still need regeneration.
	if len(pathMap) > 0 {
		if resp.Files, err = g.versionedRoots(resp.Files); err != nil {
			return nil, err
		}
	}
	frozen, err := chainFrozenFiles(req, g.pattern, embedded)
	if err != nil {
		return nil, err
	}
	resp.Files = append(resp.Files, frozen...)
	return resp, nil
}

// generation holds the state shared by the passes of one Generate call.
type generation struct {
	req *plugin.Request
	// versioned is req with its resolutions rewritten to the current version layout.
	versioned *plugin.Request
	pattern   string
	pathMap   map[string]string
	members   set.Set[string]
	closure   set.Set[string]
	embedded  set.Set[string]
}

func (g generation) isVersioned(outputPath string) bool {
	_, ok := g.pathMap[outputPath]
	return ok
}

func (g generation) generator(
	pattern string, fg framework.FileGenerator, filter func(string) bool,
) *framework.Generator {
	return &framework.Generator{
		Domain:          "go",
		FilePattern:     pattern,
		FileGenerator:   fg,
		PathFilter:      filter,
		MergeByName:     false,
		CollectTypeDefs: true,
		CollectEnums:    true,
		CollectUnions:   true,
	}
}

// versionPackages generates the current versions/vN packages.
func (g generation) versionPackages() (*plugin.Response, error) {
	preds, err := chainPredecessors(g.req)
	if err != nil {
		return nil, err
	}
	orders, err := chainOrders(g.req)
	if err != nil {
		return nil, err
	}
	pins, err := versioning.NewPins(g.req.Resolutions, g.req.Versions)
	if err != nil {
		return nil, err
	}
	currentPaths := make(set.Set[string], len(g.pathMap))
	for _, current := range g.pathMap {
		currentPaths.Add(current)
	}
	return g.generator(g.pattern, &goFileGenerator{
		preds:    preds,
		orders:   orders,
		original: g.req.Resolutions,
		pathMap:  g.pathMap,
		closure:  g.closure,
		pins:     pins,
		embedded: g.embedded,
	}, currentPaths.Contains).Generate(g.versioned)
}

// versionedRoots adds the root files of version-laid-out packages to files: transient
// declarations, the root alias files merged over them, and the versions selector
// files.
func (g generation) versionedRoots(files []plugin.File) ([]plugin.File, error) {
	// Transient types — those outside the current version file at a versioned path —
	// stay at the package root, generating real declarations merged into the root alias
	// file. Referenced enums that live in the version layout must not be re-declared
	// there.
	transientGen := g.generator(g.pattern, &goFileGenerator{
		original: g.req.Resolutions,
		pathMap:  g.pathMap,
		closure:  g.closure,
		embedded: g.embedded,
	}, g.isVersioned)
	transientGen.FilterEnums = localEnums
	transientResp, err := transientGen.Generate(g.versioned)
	if err != nil {
		return nil, err
	}
	files = append(files, transientResp.Files...)
	aliasGen := g.generator(
		g.pattern,
		&aliasFileGenerator{pathMap: g.pathMap, members: g.members},
		g.isVersioned,
	)
	aliasGen.IncludeHand = true
	aliasResp, err := aliasGen.Generate(g.req)
	if err != nil {
		return nil, err
	}
	files = mergeByPath(files, aliasResp.Files)
	selectorGen := g.generator(
		"versions/"+g.pattern,
		&aliasFileGenerator{pathMap: g.pathMap, members: g.members, pkg: "versions"},
		g.isVersioned,
	)
	selectorGen.IncludeHand = true
	selectorResp, err := selectorGen.Generate(g.req)
	if err != nil {
		return nil, err
	}
	return append(files, selectorResp.Files...), nil
}

// localEnums returns the enums whose Go output path is outputPath.
func localEnums(enums []resolution.Type, outputPath string) []resolution.Type {
	filtered := make([]resolution.Type, 0, len(enums))
	for _, e := range enums {
		if output.GetPath(e, "go") == outputPath {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// predecessor identifies the version package that a current version package's unchanged
// types alias.
type predecessor struct {
	// path is the repo-relative predecessor package directory (…/versions/vN).
	path string
	// aliased holds qualified names of types emitted as aliases.
	aliased set.Set[string]
	// names maps an aliased type's qualified name to the Go name the predecessor
	// package declares. A frozen package declares its version file's name, which a
	// live @go name does not change.
	names map[string]string
}

// AliasedTypes returns the qualified names of all types that alias their predecessor
// version package instead of being defined at the current one. Sibling plugins
// (go/marshal) use it to skip emission for aliased types.
func AliasedTypes(req *plugin.Request) (set.Set[string], error) {
	preds, err := chainPredecessors(req)
	if err != nil {
		return nil, err
	}
	aliased := make(set.Set[string])
	for _, pred := range preds {
		for qualified := range pred.aliased {
			aliased.Add(qualified)
		}
	}
	return aliased, nil
}

// goFileGenerator implements framework.FileGenerator for Go code generation.
type goFileGenerator struct {
	// preds maps current version output paths to their predecessor alias split; types
	// in the split alias the predecessor instead of defining.
	preds map[string]predecessor
	// orders maps current version output paths to their version file's declaration
	// order.
	orders map[string]map[string]int
	// original is the unrewritten table. Transient declarations at a versioned root
	// path resolve against it, so their dependency references hit root re-exports.
	original *resolution.Table
	// pathMap maps each version-laid-out root path to its current versions/vN sub-path.
	pathMap map[string]string
	// closure is the persisted closure of the whole table; declarations outside it at
	// marshal-rooted paths are transient.
	closure set.Set[string]
	// pins resolves a stored reference to the dependency version its version file
	// imports.
	pins versioning.Pins
	// embedded holds the embedKey of every type that a generated struct embeds.
	embedded set.Set[string]
}

// fileLivePath returns the live schema path of the declarations ctx renders.
func fileLivePath(ctx *framework.GenerateContext) string {
	for _, group := range [][]resolution.Type{
		ctx.Structs, ctx.Enums, ctx.TypeDefs, ctx.Unions,
	} {
		if len(group) > 0 {
			return strings.TrimSuffix(group[0].FilePath, ".oracle")
		}
	}
	return ""
}

func (g *goFileGenerator) GenerateFile(ctx *framework.GenerateContext) (string, error) {
	latest := latestTable(g.original, g.pathMap, ctx.OutputPath)
	_, transient := g.pathMap[ctx.OutputPath]
	// Every declaration a current package renders belongs to its schema's current
	// version file, so that file's pins resolve the package's stored references.
	livePath := fileLivePath(ctx)
	pinned := func(t resolution.Type) (string, bool) {
		return g.pins.CurrentPath(livePath, t)
	}
	content, err := generateGoFile(goFile{
		outputPath:    ctx.OutputPath,
		structs:       ctx.Structs,
		enums:         ctx.Enums,
		typeDefs:      ctx.TypeDefs,
		unions:        ctx.Unions,
		table:         ctx.Table,
		repoRoot:      ctx.RepoRoot,
		pred:          g.preds[ctx.OutputPath],
		latest:        latest,
		closure:       g.closure,
		transientPass: transient,
		order:         g.orders[ctx.OutputPath],
		pinned:        pinned,
		embedded:      g.embedded,
	})
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// latestTable returns the latest-resolution table for a versioned root path (the
// transient pass): the original table, where versioned siblings stay local and
// dependencies hit their root re-exports. A version package gets nil, so every
// reference it renders resolves to the dependency version its version file pins, as a
// frozen package does.
func latestTable(
	original *resolution.Table, pathMap map[string]string, outputPath string,
) *resolution.Table {
	if _, root := pathMap[outputPath]; root {
		return original
	}
	return nil
}

// Survey re-exports the chain survey for the persistence gate: entries maps
// every chain-covered @go output path to its current version, and members
// holds every current-surface member's qualified name.
func Survey(
	table *resolution.Table, res *versions.Resolver,
) (map[string]int, set.Set[string], error) {
	return versioning.Survey(table, res)
}

// StructurallyEqual reports whether two declarations share a persisted shape,
// resolving each side's references through its own table. It re-exports the
// internal schemadiff judgment for the versions gate.
func StructurallyEqual(
	old, next resolution.Type, oldTable, newTable *resolution.Table,
) bool {
	return schemadiff.StructurallyEqual(old, next, oldTable, newTable)
}

// PersistedClosure returns the qualified names of every type reachable from a @go
// marshal or @go migrate root through stored references: non-omitted struct fields,
// extends, type arguments, alias targets, distinct bases, and union variants. Types
// outside the closure never reach a table's wire format.
func PersistedClosure(table *resolution.Table) set.Set[string] {
	w := closureWalker{table: table, closure: make(set.Set[string])}
	for _, t := range table.Types {
		if domain.HasExprFromType(t, "go", "marshal") ||
			domain.HasExprFromType(t, "go", "migrate") {
			w.walkType(t)
		}
	}
	return w.closure
}

// closureWalker adds each type it walks, and each type that type stores, to closure.
type closureWalker struct {
	table   *resolution.Table
	closure set.Set[string]
}

func (w closureWalker) walkRef(ref resolution.TypeRef) {
	for _, arg := range ref.TypeArgs {
		w.walkRef(arg)
	}
	if resolved, ok := ref.Resolve(w.table); ok {
		w.walkType(resolved)
	}
}

func (w closureWalker) walkRefs(refs []resolution.TypeRef) {
	for _, ref := range refs {
		w.walkRef(ref)
	}
}

func (w closureWalker) walkType(t resolution.Type) {
	if w.closure.Contains(t.QualifiedName) {
		return
	}
	w.closure.Add(t.QualifiedName)
	switch form := t.Form.(type) {
	case resolution.StructForm:
		w.walkStruct(t, form)
	case resolution.EnumForm:
		w.walkRefs(form.Extends)
	case resolution.AliasForm:
		w.walkRef(form.Target)
	case resolution.DistinctForm:
		w.walkRef(form.Base)
	case resolution.UnionForm:
		w.walkRefs(form.Extends)
		for _, v := range form.Variants {
			w.walkRef(v.Type)
		}
	}
}

func (w closureWalker) walkStruct(t resolution.Type, form resolution.StructForm) {
	w.walkRefs(form.Extends)
	for _, tp := range form.TypeParams {
		if tp.Constraint != nil {
			w.walkRef(*tp.Constraint)
		}
		if tp.Default != nil {
			w.walkRef(*tp.Default)
		}
	}
	persisted := schemadiff.PersistedFields(resolution.UnifiedFields(t, w.table))
	for _, f := range persisted {
		w.walkRef(f.Type)
	}
}

// goFile holds the inputs of one generated types file. Types named in pred.aliased are
// emitted as aliases to the predecessor version package instead of full definitions.
type goFile struct {
	outputPath    string
	structs       []resolution.Type
	enums         []resolution.Type
	typeDefs      []resolution.Type
	unions        []resolution.Type
	table         *resolution.Table
	repoRoot      string
	pred          predecessor
	latest        *resolution.Table
	closure       set.Set[string]
	transientPass bool
	order         map[string]int
	pinned        func(resolution.Type) (string, bool)
	embedded      set.Set[string]
}

// addDecl adds t to the declarations of its form.
func (f *goFile) addDecl(t resolution.Type) {
	switch t.Form.(type) {
	case resolution.StructForm:
		f.structs = append(f.structs, t)
	case resolution.EnumForm:
		f.enums = append(f.enums, t)
	case resolution.UnionForm:
		f.unions = append(f.unions, t)
	case resolution.DistinctForm, resolution.AliasForm:
		f.typeDefs = append(f.typeDefs, t)
	}
}

// generateGoFile generates the Go types file that f describes.
func generateGoFile(f goFile) ([]byte, error) {
	content, conflicts, err := f.render(nil)
	if err != nil || len(conflicts) == 0 {
		return content, err
	}
	// Two versions of one dependency, or two packages with one name, derive the same
	// alias. Rebind them and render again.
	var conflicted []string
	for _, paths := range conflicts {
		conflicted = append(conflicted, paths...)
	}
	overrides := naming.ImportAliases(
		conflicted, resolveGoImportPath(f.outputPath, f.repoRoot),
	)
	content, conflicts, err = f.render(overrides)
	if err != nil {
		return nil, err
	}
	if len(conflicts) > 0 {
		return nil, errors.Newf(
			"unresolvable import alias conflicts in %s: %v", f.outputPath, conflicts,
		)
	}
	return content, nil
}

// render renders the file. Rendered references bake their import alias in, so alias
// collisions cannot be repaired in place: the caller inspects the returned conflicts
// and re-renders with aliasOverrides.
func (f *goFile) render(
	aliasOverrides map[string]string,
) ([]byte, map[string][]string, error) {
	data := f.templateData(aliasOverrides)
	// Transient declarations — outside the persisted closure of a marshal-rooted
	// package — never reach the stored wire format, so their dependency references
	// resolve against the latest table and track the dependencies' current versions
	// instead of pinning one.
	var latestData *templateData
	if f.latest != nil && (f.transientPass || pathHasMarshalRoot(f.structs)) {
		latestData = data.latestView(f.latest)
		data.latest = latestData
	}
	var predAlias string
	for _, d := range orderDecls(
		f.table, f.order, f.typeDefs, f.enums, f.structs, f.unions,
	) {
		if omit.IsSkipped(d.typ, "go") ||
			(d.kind == declEnum && d.typ.Namespace != data.Namespace) {
			continue
		}
		if d.kind != declStruct && domain.HasExprFromType(d.typ, "go", "to_any") {
			return nil, nil, errors.Newf(
				"%s: @go to_any goes on a struct", d.typ.QualifiedName,
			)
		}
		if f.pred.aliased.Contains(d.typ.QualifiedName) {
			if predAlias == "" {
				predAlias = naming.DerivePackageName(f.pred.path)
				data.AddInternal(
					predAlias,
					resolveGoImportPath(f.pred.path, f.repoRoot),
				)
			}
			target := f.pred.names[d.typ.QualifiedName]
			for _, a := range buildAliasDecls(d, predAlias, target, data) {
				data.Decls = append(data.Decls, declData{Alias: &a})
			}
			continue
		}
		procData := data
		if latestData != nil && !f.closure.Contains(d.typ.QualifiedName) {
			procData = latestData
		}
		decl, err := processDecl(d, procData)
		if err != nil {
			return nil, nil, err
		}
		data.Decls = append(data.Decls, decl)
	}
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), data.Conflicts(), nil
}

// templateData returns the template data of the file, with no declarations yet.
func (f *goFile) templateData(aliasOverrides map[string]string) *templateData {
	namespace := fileNamespace(f.structs, f.unions, f.typeDefs, f.enums)
	pkg := naming.DerivePackageName(f.outputPath)
	imports := imports.NewManager()
	importResolver := &GoImportResolver{
		RepoRoot:       f.repoRoot,
		CurrentPackage: pkg,
		AliasOverrides: aliasOverrides,
	}
	return &templateData{
		Package:    pkg,
		OutputPath: f.outputPath,
		Namespace:  namespace,
		Manager:    imports,
		table:      f.table,
		repoRoot:   f.repoRoot,
		resolver: &resolver.Resolver{
			Formatter:       GoFormatter(),
			ImportResolver:  importResolver,
			ImportAdder:     imports,
			PrimitiveMapper: primitiveMapper,
		},
		probeResolver: &resolver.Resolver{
			Formatter:       GoFormatter(),
			ImportResolver:  importResolver,
			ImportAdder:     discardedImports{},
			PrimitiveMapper: primitiveMapper,
		},
		ctx: &resolver.Context{
			Table:                         f.table,
			OutputPath:                    f.outputPath,
			Namespace:                     namespace,
			RepoRoot:                      f.repoRoot,
			DomainName:                    "go",
			SubstituteDefaultedTypeParams: true,
			PinnedPath:                    f.pinned,
		},
		aliasOverrides: aliasOverrides,
		embedded:       f.embedded,
	}
}

// latestView returns a copy of d that resolves references against latest and pins none
// of them.
func (d *templateData) latestView(latest *resolution.Table) *templateData {
	ctx := *d.ctx
	ctx.Table = latest
	ctx.PinnedPath = nil
	view := *d
	view.table = latest
	view.ctx = &ctx
	return &view
}

// fileNamespace returns the namespace of the first declaration a file renders, checking
// structs, unions, typedefs, and enums in that order.
func fileNamespace(structs, unions, typeDefs, enums []resolution.Type) string {
	for _, group := range [][]resolution.Type{structs, unions, typeDefs, enums} {
		if len(group) > 0 {
			return group[0].Namespace
		}
	}
	return ""
}

func processDecl(d orderedDecl, data *templateData) (declData, error) {
	switch d.kind {
	case declTypeDef:
		td := processTypeDef(d.typ, data)
		return declData{TypeDef: &td}, nil
	case declEnum:
		e := processEnum(d.typ)
		return declData{Enum: &e}, nil
	case declStruct:
		s, err := processStruct(d.typ, data)
		if err != nil {
			return declData{}, err
		}
		return declData{Struct: &s}, nil
	case declUnion:
		u, err := processUnion(d.typ, data)
		if err != nil {
			return declData{}, err
		}
		return declData{Union: &u}, nil
	}
	return declData{}, errors.Newf("unknown declaration kind %d", d.kind)
}

func resolveGoImportPath(outputPath, repoRoot string) string {
	return gomod.ResolveImportPath(outputPath, repoRoot, gomod.DefaultModulePrefix)
}

type declKind int

const (
	declTypeDef declKind = iota
	declEnum
	declStruct
	declUnion
)

type orderedDecl struct {
	kind declKind
	typ  resolution.Type
}

// discardedImports drops every import registered through it. Resolution registers the
// imports a rendering needs as a side effect, which is wrong when the rendering is
// thrown away.
type discardedImports struct{}

// AddImport implements resolver.ImportAdder.
func (discardedImports) AddImport(string, string, string) {}

// probe returns a view of d whose resolution registers no imports. Predicates that ask
// whether a field renders anything build the rendering to find out and discard it; the
// imports that rendering would need belong to the file only if it is emitted.
func (d *templateData) probe() *templateData {
	probed := *d
	probed.resolver = d.probeResolver
	return &probed
}

// orderDecls merges the kind-grouped type lists into schema declaration order. An
// explicit order wins when present; otherwise the table's registration order is the
// source position. Frozen surface tables register sorted, so the frozen path must
// pass the version file's declaration order explicitly.
func orderDecls(
	table *resolution.Table,
	order map[string]int,
	typeDefs, enums, structs, unions []resolution.Type,
) []orderedDecl {
	index := order
	if index == nil {
		index = make(map[string]int, len(table.Types))
		for i, t := range table.Types {
			index[t.QualifiedName] = i
		}
	}
	pos := func(t resolution.Type) int {
		if i, ok := index[t.QualifiedName]; ok {
			return i
		}
		return math.MaxInt
	}
	decls := make(
		[]orderedDecl, 0, len(typeDefs)+len(enums)+len(structs)+len(unions),
	)
	for _, t := range typeDefs {
		decls = append(decls, orderedDecl{kind: declTypeDef, typ: t})
	}
	for _, t := range enums {
		decls = append(decls, orderedDecl{kind: declEnum, typ: t})
	}
	for _, t := range structs {
		decls = append(decls, orderedDecl{kind: declStruct, typ: t})
	}
	for _, t := range unions {
		decls = append(decls, orderedDecl{kind: declUnion, typ: t})
	}
	slices.SortStableFunc(decls, func(a, b orderedDecl) int {
		return cmp.Compare(pos(a.typ), pos(b.typ))
	})
	return decls
}

func processEnum(e resolution.Type) enumData {
	name := naming.GetGoName(e)
	form := resolution.MustForm[resolution.EnumForm](e)
	values := make([]enumValueData, 0, len(form.Values))
	for _, v := range form.Values {
		values = append(values, enumValueData{
			Name:     naming.ToPascalCase(v.Name),
			Doc:      doc.Get(v.Domains),
			Value:    v.StringValue(),
			IntValue: v.IntValue(),
		})
	}
	startsAtOne := form.IsIntEnum && len(values) > 0 && values[0].IntValue == 1
	if startsAtOne {
		for i, v := range values {
			if v.IntValue != int64(i+1) {
				startsAtOne = false
				break
			}
		}
	}
	return enumData{
		Name:        name,
		Doc:         doc.Get(e.Domains),
		Values:      values,
		IsIntEnum:   form.IsIntEnum,
		StartsAtOne: startsAtOne,
	}
}

func processTypeDef(td resolution.Type, data *templateData) typeDefData {
	name := naming.GetGoName(td)

	switch form := td.Form.(type) {
	case resolution.DistinctForm:
		result := typeDefData{
			Name:     name,
			Doc:      doc.Get(td.Domains),
			BaseType: data.resolver.ResolveTypeRef(form.Base, data.ctx),
			IsAlias:  false,
		}
		for _, tp := range resolution.NonDefaultedTypeParams(form.TypeParams) {
			result.TypeParams = append(result.TypeParams, processTypeParam(tp, data))
		}
		result.IsGeneric = len(result.TypeParams) > 0
		return result
	case resolution.AliasForm:
		result := typeDefData{
			Name:     name,
			Doc:      doc.Get(td.Domains),
			BaseType: data.resolver.ResolveTypeRef(form.Target, data.ctx),
			IsAlias:  true,
		}
		for _, tp := range resolution.NonDefaultedTypeParams(form.TypeParams) {
			result.TypeParams = append(result.TypeParams, processTypeParam(tp, data))
		}
		result.IsGeneric = len(result.TypeParams) > 0
		return result
	default:
		return typeDefData{Name: name, BaseType: "any"}
	}
}

func processStruct(entry resolution.Type, data *templateData) (structData, error) {
	form := resolution.MustForm[resolution.StructForm](entry)
	sd := structData{
		Name:    entry.Name,
		Doc:     doc.Get(entry.Domains),
		Fields:  make([]fieldData, 0, len(form.Fields)),
		IsAlias: false,
	}

	if name := domain.GetStringFromType(entry, "go", "name"); name != "" {
		sd.Name = name
	}

	for _, tp := range resolution.NonDefaultedTypeParams(form.TypeParams) {
		sd.TypeParams = append(sd.TypeParams, processTypeParam(tp, data))
	}
	sd.IsGeneric = len(sd.TypeParams) > 0

	fields, embeds, defaultOnly := structMembers(&sd, entry, form, data)
	genMethods := !sd.IsGeneric
	if genMethods && len(embeds) > 0 {
		sd.ValidateRecurse = append(
			sd.ValidateRecurse,
			embedRecurseSteps(embeds, nil, data, validateRule)...,
		)
	}
	if err := checkHeld(entry.Name, sd.IsGeneric, embeds, fields, data); err != nil {
		return structData{}, err
	}
	for _, field := range fields {
		if !defaultOnly.Contains(field.Name) {
			sd.Fields = append(sd.Fields, processField(field, data))
		}
		if genMethods {
			addFieldMethods(&sd, field, defaultOnly.Contains(field.Name), data)
		}
	}
	if genMethods {
		sd.DefaultGroups = goDefaultGroups(fields, data)
		sd.SetSteps = setDefaultsSteps(embeds, fields, data)
	}
	sd.DecodeHooks = sd.SetDefaultsView().Sets() &&
		!data.embedded.Contains(embedKey(data.OutputPath, naming.GetGoName(entry)))
	addMethodImports(&sd, data)
	if len(sd.Name) > 0 {
		sd.Receiver = receiverName(sd.Name, hasSliceRecurse(sd.ValidateRecurse))
	}

	if err := applyDirectives(&sd, entry, form, data); err != nil {
		return structData{}, err
	}
	return sd, nil
}

// structMembers returns the fields a struct declares, the parents it embeds, and the
// names of the fields it redeclares only to change an inherited default. A struct that
// cannot embed its parents flattens their fields into its own.
func structMembers(
	sd *structData,
	entry resolution.Type,
	form resolution.StructForm,
	data *templateData,
) ([]resolution.Field, []embeddedType, set.Set[string]) {
	if len(form.Extends) == 0 || !CanEmbed(form, data.table) {
		return resolution.UnifiedFields(entry, data.table), nil, nil
	}
	sd.HasExtends = true
	embeds := embedExtends(sd, form, data)
	// The embedded parent already declares a default-only field, so it contributes a
	// default fill and nothing else. Declaring it again would shadow the parent's field
	// with a duplicate JSON tag.
	defaultOnly := resolver.DefaultOnlyOverrides(form.Extends, form.Fields, data.table)
	return form.Fields, embeds, defaultOnly
}

// checkHeld checks that each field of a struct, and each parent a generic struct
// embeds, holds an embedded type only where it may.
func checkHeld(
	structName string,
	generic bool,
	embeds []embeddedType,
	fields []resolution.Field,
	data *templateData,
) error {
	if generic {
		for _, e := range embeds {
			parent := resolution.Field{
				Name:     embedFieldName(e.rendered),
				Type:     e.ref,
				Optional: true,
			}
			if err := checkHeldEmbedded(structName, parent, data); err != nil {
				return err
			}
		}
	}
	for _, field := range fields {
		held := field
		// A generic struct sets no defaults, so it holds each of its fields.
		held.Optional = field.Optional || generic
		if err := checkHeldEmbedded(structName, held, data); err != nil {
			return err
		}
	}
	return nil
}

// applyDirectives applies the @go fields, imports, and to_any directives of entry.
func applyDirectives(
	sd *structData,
	entry resolution.Type,
	form resolution.StructForm,
	data *templateData,
) error {
	sd.ExtraFields = domain.GetAllStringsFromType(entry, "go", "fields")
	for _, imp := range domain.GetAllStringsFromType(entry, "go", "imports") {
		data.AddExternal(imp)
	}
	sd.ToAny = domain.HasExprFromType(entry, "go", "to_any")
	if sd.ToAny {
		return validateToAny(entry, form, *sd)
	}
	return nil
}

// embedExtends renders each resolved parent of form as an embedded type.
func embedExtends(
	sd *structData,
	form resolution.StructForm,
	data *templateData,
) []embeddedType {
	var embeds []embeddedType
	for _, extendsRef := range form.Extends {
		parent, ok := extendsRef.Resolve(data.table)
		if ok {
			rendered := resolveExtendsType(extendsRef, parent, data)
			sd.ExtendsTypes = append(sd.ExtendsTypes, rendered)
			embeds = append(embeds, embeddedType{ref: extendsRef, rendered: rendered})
		}
	}
	return embeds
}

// addFieldMethods adds the default fills and validation checks of field. defaultOnly
// marks a field that only restates an inherited default.
func addFieldMethods(
	sd *structData,
	field resolution.Field,
	defaultOnly bool,
	data *templateData,
) {
	fd := data.forField(field)
	if defaultGroupName(field) == "" {
		fills := goDefaultFills(field, fd)
		if defaultOnly {
			fills = overrideFills(field, fd)
		}
		sd.SetFills = append(sd.SetFills, fills...)
	}
	if validateSkip(field, fd) {
		return
	}
	if chk, ok := goEnumCheck(field, fd); ok {
		sd.EnumChecks = append(sd.EnumChecks, chk)
	}
	sd.ConstraintChecks = append(
		sd.ConstraintChecks,
		goConstraintChecks(field, defaultOnly, fd)...,
	)
	if step, ok := goRecurseStep(field, fd, validateRule); ok {
		sd.ValidateRecurse = append(sd.ValidateRecurse, step)
	}
}

// addMethodImports adds the imports that the struct's decode hooks and validation need.
func addMethodImports(sd *structData, data *templateData) {
	if sd.DecodeHooks {
		data.AddExternal("encoding/json/v2")
		data.AddExternal("encoding/json/jsontext")
		data.AddExternal("reflect")
		// The alias keeps clear of x's msgpack package and of a v5 version package.
		data.AddInternal("vmsgpack", msgpackImportPath)
	}
	if len(sd.EnumChecks) > 0 || len(sd.ConstraintChecks) > 0 ||
		len(sd.ValidateRecurse) > 0 {
		data.AddExternal(validateImportPath)
	}
	if hasSliceRecurse(sd.ValidateRecurse) {
		data.AddExternal(strconvImportPath)
	}
}

// validateToAny errors when @go to_any marks a struct that its ToAny method cannot
// convert field by field.
func validateToAny(
	entry resolution.Type,
	form resolution.StructForm,
	sd structData,
) error {
	name := entry.QualifiedName
	if !sd.IsGeneric {
		return errors.Newf("%s: @go to_any needs a type parameter", name)
	}
	for _, tp := range sd.TypeParams {
		if tp.Constraint != "any" {
			return errors.Newf(
				"%s: @go to_any needs type parameter %s to be unconstrained, not %s",
				name, tp.Name, tp.Constraint,
			)
		}
	}
	if len(form.Extends) > 0 {
		return errors.Newf("%s: @go to_any does not support extends", name)
	}
	if len(sd.ExtraFields) > 0 {
		return errors.Newf("%s: @go to_any cannot copy hand-written @go fields", name)
	}
	for _, f := range form.Fields {
		bare := f.Type.IsTypeParam() && !f.Optional
		if !bare && holdsTypeParam(f.Type) {
			return errors.Newf(
				"%s: @go to_any cannot convert field %s; a type parameter must be "+
					"the whole type of a required field",
				name, f.Name,
			)
		}
	}
	return nil
}

// holdsTypeParam reports whether ref is or contains a type parameter that Go keeps.
// Go substitutes a defaulted type parameter with its default.
func holdsTypeParam(ref resolution.TypeRef) bool {
	if ref.IsTypeParam() {
		return !ref.TypeParam.HasDefault()
	}
	return slices.ContainsFunc(ref.TypeArgs, holdsTypeParam)
}

// receiverName derives a method receiver from the type name. It widens to two letters
// when the first would be "v", which generated Validate bodies use for their validator,
// or "i" when the methods loop over a slice with that index.
func receiverName(name string, loops bool) string {
	r := strings.ToLower(name[:1])
	if r != "v" && (r != "i" || !loops) {
		return r
	}
	if len(name) > 1 {
		return strings.ToLower(name[:2])
	}
	return r + r
}

func processTypeParam(tp resolution.TypeParam, data *templateData) typeParamData {
	tpd := typeParamData{
		Name:       tp.Name,
		Constraint: "any",
	}

	if tp.Constraint != nil {
		tpd.Constraint = constraintToGo(*tp.Constraint, data)
	}

	return tpd
}

func constraintToGo(constraint resolution.TypeRef, data *templateData) string {
	if resolution.IsConstraint(constraint.Name) {
		return constraint.Name
	}
	if resolution.IsPrimitive(constraint.Name) {
		switch constraint.Name {
		case "record":
			return "any"
		case "string":
			return "~string"
		case "int", "int8", "int16", "int32", "int64":
			return "~int | ~int8 | ~int16 | ~int32 | ~int64"
		case "uint", "uint8", "uint16", "uint32", "uint64":
			return "~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64"
		default:
			return data.resolver.ResolveTypeRef(constraint, data.ctx)
		}
	}
	return data.resolver.ResolveTypeRef(constraint, data.ctx)
}

// forField returns the data that resolves field. An omitted field resolves against the
// latest table, so its type, defaults, and checks track its dependencies' current
// versions together.
func (d *templateData) forField(field resolution.Field) *templateData {
	if d.latest != nil && domain.GetStringFromField(field, "go", "marshal") == "omit" {
		return d.latest
	}
	return d
}

func processField(field resolution.Field, data *templateData) fieldData {
	data = data.forField(field)
	goType := data.resolver.ResolveTypeRef(field.Type, data.ctx)
	if field.Optional && !typemap.Nilable(field.Type, data.table) {
		goType = "*" + goType
	}
	return fieldData{
		GoName:      naming.GetFieldName(field),
		GoType:      goType,
		JSONName:    casing.FieldSnake(field.Name),
		IsOptional:  field.Optional,
		Doc:         doc.Get(field.Domains),
		WireOmitted: domain.GetStringFromField(field, "go", "wire") == "omit",
	}
}

func buildGenericType(
	baseName string,
	typeArgs []resolution.TypeRef,
	targetType *resolution.Type,
	data *templateData,
) string {
	if len(typeArgs) == 0 {
		return baseName
	}

	var typeParams []resolution.TypeParam
	if targetType != nil {
		typeParams = resolution.TypeParams(targetType.Form)
	}
	var args []string
	for i, arg := range typeArgs {
		if i < len(typeParams) && typeParams[i].HasDefault() {
			continue
		}
		args = append(args, data.resolver.ResolveTypeRef(arg, data.ctx))
	}

	if len(args) == 0 {
		return baseName
	}
	return fmt.Sprintf("%s[%s]", baseName, strings.Join(args, ", "))
}

func resolveExtendsType(
	extendsRef resolution.TypeRef,
	parent resolution.Type,
	data *templateData,
) string {
	targetOutputPath := output.GetPath(parent, "go")

	name := naming.GetGoName(parent)

	if parent.Namespace == data.Namespace &&
		(targetOutputPath == "" || targetOutputPath == data.OutputPath) {
		return buildGenericType(name, extendsRef.TypeArgs, &parent, data)
	}

	if targetOutputPath == "" {
		return name
	}
	alias := data.importAlias(targetOutputPath)
	data.AddInternal(
		alias,
		resolveGoImportPath(targetOutputPath, data.repoRoot),
	)
	return fmt.Sprintf(
		"%s.%s",
		alias,
		buildGenericType(name, extendsRef.TypeArgs, &parent, data),
	)
}

// pathHasMarshalRoot reports whether any struct generated at the path is a
// gorp entry (@go marshal); only such packages distinguish transient
// declarations.
func pathHasMarshalRoot(structs []resolution.Type) bool {
	for _, s := range structs {
		if domain.HasExprFromType(s, "go", "marshal") {
			return true
		}
	}
	return false
}

type templateData struct {
	*imports.Manager
	table    *resolution.Table
	resolver *resolver.Resolver
	ctx      *resolver.Context
	// latest, when set, resolves omitted (memory-only) fields against the latest table
	// so they track dependencies' current versions.
	latest *templateData
	// probeResolver resolves exactly like resolver but discards the imports it would
	// register, for predicates that build a rendering only to ask whether one exists.
	probeResolver *resolver.Resolver
	// aliasOverrides remaps specific output paths to explicit import aliases
	// during the conflict re-render pass.
	aliasOverrides map[string]string
	// embedded holds the embedKey of every type that a generated struct embeds.
	embedded   set.Set[string]
	Package    string
	OutputPath string
	Namespace  string
	repoRoot   string
	Decls      []declData
}

// importAlias returns the import alias for a cross-package reference,
// honoring the conflict re-render pass's overrides (keyed by import path).
func (d *templateData) importAlias(outputPath string) string {
	if a, ok := d.aliasOverrides[resolveGoImportPath(outputPath, d.repoRoot)]; ok {
		return a
	}
	return naming.DerivePackageAlias(outputPath, d.Package)
}

// declData wraps one processed declaration in schema order; exactly one field is
// non-nil.
type declData struct {
	TypeDef *typeDefData
	Enum    *enumData
	Struct  *structData
	Union   *unionData
	// Alias re-exports a type from the predecessor version package instead of
	// defining it.
	Alias *aliasDecl
}

type structData struct {
	Name         string
	Doc          string
	AliasOf      string
	Receiver     string
	Fields       []fieldData
	TypeParams   []typeParamData
	ExtendsTypes []string
	ExtraFields  []string
	// SetSteps and SetFills are the nested SetDefaults calls and the assignments of
	// the generated SetDefaults.
	SetSteps []recurseStepData
	SetFills []defaultFillData
	// DefaultGroups are the struct's own `@default group`s, set only when all of
	// their members are zero.
	DefaultGroups []defaultGroupData
	// DecodeHooks marks a struct whose zero value decodes JSON and msgpack over its
	// defaults. A type that some struct embeds takes none: its embedder sets them.
	DecodeHooks      bool
	EnumChecks       []enumCheckData
	ConstraintChecks []constraintCheckData
	ValidateRecurse  []recurseStepData
	IsGeneric        bool
	IsAlias          bool
	HasExtends       bool
	// ToAny is set by @go to_any: the struct gets a ToAny method.
	ToAny bool
}

// setDefaultsView is the template view of a generated SetDefaults method.
type setDefaultsView struct {
	Receiver string
	Name     string
	Steps    []recurseStepData
	Fills    []defaultFillData
	// Groups are the type's own `@default group`s, each set only when all of its
	// members are zero.
	Groups []defaultGroupData
}

// Sets reports whether the type emits SetDefaults.
func (v setDefaultsView) Sets() bool {
	return len(v.Steps) > 0 || len(v.Fills) > 0 || len(v.Groups) > 0
}

// HasGroups reports whether the type or a type it sets the defaults of has a
// `@default group`, so it splits SetDefaults into SetUngroupedDefaults and
// DefaultGroups.
func (v setDefaultsView) HasGroups() bool {
	if len(v.Groups) > 0 {
		return true
	}
	for _, s := range v.Steps {
		if s.Groups {
			return true
		}
	}
	return false
}

// SetDefaultsView returns the view of the struct's SetDefaults method.
func (s structData) SetDefaultsView() setDefaultsView {
	return setDefaultsView{
		Receiver: s.Receiver,
		Name:     s.Name,
		Steps:    s.SetSteps,
		Fills:    s.SetFills,
		Groups:   s.DefaultGroups,
	}
}

// Instance renders the struct instantiated with its own type parameters.
func (s structData) Instance() string {
	names := make([]string, len(s.TypeParams))
	for i, tp := range s.TypeParams {
		names[i] = tp.Name
	}
	return s.Name + "[" + strings.Join(names, ", ") + "]"
}

// AnyInstance renders the struct instantiated with every type parameter set to any.
func (s structData) AnyInstance() string {
	anys := slices.Repeat([]string{"any"}, len(s.TypeParams))
	return s.Name + "[" + strings.Join(anys, ", ") + "]"
}

type typeParamData struct {
	Name       string
	Constraint string
}

type fieldData struct {
	GoName     string
	GoType     string
	JSONName   string
	Doc        string
	IsOptional bool
	// WireOmitted is set by @go wire omit: JSON and msgpack skip the field.
	WireOmitted bool
}

// JSONTag returns the field's JSON tag. An optional field uses `,omitzero` so its Go
// zero — a nil slice, map or pointer — is omitted while any allocated value
// serializes, including an empty collection or a pointer to "". A required field always
// serializes, because the schema says it always applies.
func (f fieldData) JSONTag() string {
	if f.WireOmitted {
		return "-"
	}
	if f.IsOptional {
		return f.JSONName + ",omitzero"
	}
	return f.JSONName
}

// MsgpackTag returns the field's msgpack tag. vmihailenco/msgpack honors only
// `,omitempty`, which also omits an allocated empty collection.
func (f fieldData) MsgpackTag() string {
	if f.WireOmitted {
		return "-"
	}
	if f.IsOptional {
		return f.JSONName + ",omitempty"
	}
	return f.JSONName
}

type enumData struct {
	Name        string
	Doc         string
	Values      []enumValueData
	IsIntEnum   bool
	StartsAtOne bool
}

// Receiver returns the single-letter, lowercased method receiver name for the enum type
// (e.g. "n" for Notation).
func (e enumData) Receiver() string { return strings.ToLower(e.Name[:1]) }

type enumValueData struct {
	Name     string
	Doc      string
	Value    string
	IntValue int64
}

type typeDefData struct {
	Name       string
	Doc        string
	BaseType   string
	TypeParams []typeParamData
	IsAlias    bool
	IsGeneric  bool
}

var templateFuncs = template.FuncMap{
	"join":      strings.Join,
	"formatDoc": doc.FormatGo,
	"formatFieldDoc": func(name, docStr string) string {
		return doc.FormatGo(name, docStr, fieldDocIndent)
	},
}

var fileTemplate = template.Must(
	template.New("go-types").
		Funcs(templateFuncs).
		Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}
{{- if .HasImports}}

import (
{{- range .StdImports}}
	"{{.}}"
{{- end}}
{{- if and .StdImports .NonStdImports}}
{{end}}
{{- range .NonStdImports}}
{{- if .NeedsAlias}}
	{{.Alias}} "{{.Path}}"
{{- else}}
	"{{.Path}}"
{{- end}}
{{- end}}
)
{{- end}}
{{- range .Decls}}
{{- with .Alias}}
{{- if .Doc}}

{{formatDoc .Name .Doc}}
{{- end}}
type {{.LHS}} = {{.RHS}}
{{- if eq (len .Consts) 1}}
{{- with index .Consts 0}}

{{- if .Doc}}

{{formatDoc .Name .Doc}}
{{- end}}
const {{.Name}} {{.Type}} = {{.Target}}
{{- end}}
{{- else if .Consts}}

const (
{{- range .Consts}}
{{- if .Doc}}
	{{formatFieldDoc .Name .Doc | printf "%s"}}
{{- end}}
	{{.Name}} {{.Type}} = {{.Target}}
{{- end}}
)
{{- end}}
{{- end}}
{{- with .TypeDef}}
{{- if .Doc}}

{{formatDoc .Name .Doc}}
{{- end}}
{{- if .IsAlias}}
type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} = {{.BaseType}}
{{- else}}
type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} {{.BaseType}}
{{- end}}
{{- end}}
{{- with .Enum}}
{{- $enum := .}}

{{- if $enum.Doc}}
{{formatDoc $enum.Name $enum.Doc}}
{{- end}}
{{- if $enum.IsIntEnum}}
type {{$enum.Name}} uint8

{{"//"}}go:generate stringer -type={{$enum.Name}}
{{- if eq (len $enum.Values) 1}}
{{- with index $enum.Values 0}}

{{- if .Doc}}

{{formatDoc (printf "%s%s" $enum.Name .Name) .Doc}}
{{- end}}
const {{$enum.Name}}{{.Name}} {{$enum.Name}} = {{if $enum.StartsAtOne}}1{{else}}0{{end}}
{{- end}}
{{- else}}

const (
{{- range $i, $v := $enum.Values}}
{{- if $v.Doc}}
	{{formatFieldDoc (printf "%s%s" $enum.Name $v.Name) $v.Doc | printf "%s"}}
{{- end}}
{{- if eq $i 0}}
	{{$enum.Name}}{{$v.Name}} {{$enum.Name}} = iota{{if $enum.StartsAtOne}} + 1{{end}}
{{- else}}
	{{$enum.Name}}{{$v.Name}}
{{- end}}
{{- end}}
)
{{- end}}
{{- else}}

{{- if not $enum.Doc}}
{{- end}}
type {{$enum.Name}} string
{{- if eq (len $enum.Values) 1}}
{{- with index $enum.Values 0}}

{{- if .Doc}}

{{formatDoc (printf "%s%s" $enum.Name .Name) .Doc}}
{{- end}}
const {{$enum.Name}}{{.Name}} {{$enum.Name}} = "{{.Value}}"
{{- end}}
{{- else}}

const (
{{- range $enum.Values}}
{{- if .Doc}}
	{{formatFieldDoc (printf "%s%s" $enum.Name .Name) .Doc | printf "%s"}}
{{- end}}
	{{$enum.Name}}{{.Name}} {{$enum.Name}} = "{{.Value}}"
{{- end}}
)
{{- end}}
{{- if $enum.Values}}

// IsValid reports whether {{$enum.Receiver}} is one of the defined {{$enum.Name}}
// values.
func ({{$enum.Receiver}} {{$enum.Name}}) IsValid() bool {
	switch {{$enum.Receiver}} {
	case {{range $i, $v := $enum.Values}}{{if $i}}, {{end}}{{$enum.Name}}{{$v.Name}}{{end}}:
		return true
	default:
		return false
	}
}
{{- end}}
{{- end}}
{{- end}}
{{- with .Struct}}
{{if .Doc}}{{formatDoc .Name .Doc}}
{{end -}}
{{if .IsAlias -}}
type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} = {{.AliasOf}}
{{else if .HasExtends -}}
type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} struct {
{{- range .ExtendsTypes}}
	{{.}}
{{- end}}
{{- range .Fields}}
{{- if .Doc}}
	{{formatFieldDoc .GoName .Doc | printf "%s"}}
{{- end}}
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JSONTag}}" msgpack:"{{.MsgpackTag}}"` + "`" + `
{{- end}}
{{- range .ExtraFields}}
	{{.}}
{{- end}}
}
{{else -}}
type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} struct {
{{- range .Fields}}
{{- if .Doc}}
	{{formatFieldDoc .GoName .Doc | printf "%s"}}
{{- end}}
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JSONTag}}" msgpack:"{{.MsgpackTag}}"` + "`" + `
{{- end}}
{{- range .ExtraFields}}
	{{.}}
{{- end}}
}
{{end -}}
{{- $s := .}}
{{- template "setDefaults" .SetDefaultsView}}
{{- if .DecodeHooks}}
{{- if .SetDefaultsView.HasGroups}}

// UnmarshalJSONFrom decodes the input over the value. A zero value first takes its
// defaults outside its default groups. After the decode, each group whose members are
// all zero takes its defaults, so an explicit all-zero group takes them too and a group
// the input sets in part keeps its zero members. A value that held data keeps its
// groups as decoded. An explicit zero outside a group stays.
{{- else}}

// UnmarshalJSONFrom decodes the input over the value. A zero value first takes the
// schema defaults, so an omitted member takes its default and a zero stays.
{{- end}}
func ({{$s.Receiver}} *{{$s.Name}}) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain {{$s.Name}}
{{- if .SetDefaultsView.HasGroups}}
	fresh := reflect.ValueOf(*{{$s.Receiver}}).IsZero()
	if fresh {
		{{$s.Receiver}}.SetUngroupedDefaults()
	}
	if err := json.UnmarshalDecode(dec, (*plain)({{$s.Receiver}})); err != nil {
		return err
	}
	if fresh {
		{{$s.Receiver}}.DefaultGroups()
	}
	return nil
{{- else}}
	if reflect.ValueOf(*{{$s.Receiver}}).IsZero() {
		{{$s.Receiver}}.SetDefaults()
	}
	return json.UnmarshalDecode(dec, (*plain)({{$s.Receiver}}))
{{- end}}
}

// DecodeMsgpack decodes the input over the value, as UnmarshalJSONFrom does.
func ({{$s.Receiver}} *{{$s.Name}}) DecodeMsgpack(dec *vmsgpack.Decoder) error {
	type plain {{$s.Name}}
{{- if .SetDefaultsView.HasGroups}}
	fresh := reflect.ValueOf(*{{$s.Receiver}}).IsZero()
	if fresh {
		{{$s.Receiver}}.SetUngroupedDefaults()
	}
	if err := dec.Decode((*plain)({{$s.Receiver}})); err != nil {
		return err
	}
	if fresh {
		{{$s.Receiver}}.DefaultGroups()
	}
	return nil
{{- else}}
	if reflect.ValueOf(*{{$s.Receiver}}).IsZero() {
		{{$s.Receiver}}.SetDefaults()
	}
	return dec.Decode((*plain)({{$s.Receiver}}))
{{- end}}
}
{{- end}}
{{- if or .EnumChecks .ConstraintChecks .ValidateRecurse}}

// Validate returns an error wrapping validate.ErrValidation if any field violates its
// schema constraints.
func ({{$s.Receiver}} {{$s.Name}}) Validate() error {
	v := validate.New("{{$s.Name}}")
{{- range $s.EnumChecks}}
{{- if .MapKey}}
	for key := range {{$s.Receiver}}.{{.GoName}} {
		v.Ternaryf("{{.FieldName}}", !key.IsValid(), "invalid {{.FieldName}} key: %v", key)
	}
{{- else if .Optional}}
	if {{$s.Receiver}}.{{.GoName}} != nil {
		v.Ternaryf("{{.FieldName}}", !{{$s.Receiver}}.{{.GoName}}.IsValid(), "invalid {{.FieldName}}: %v", *{{$s.Receiver}}.{{.GoName}})
	}
{{- else}}
	v.Ternaryf("{{.FieldName}}", !{{$s.Receiver}}.{{.GoName}}.IsValid(), "invalid {{.FieldName}}: %v", {{$s.Receiver}}.{{.GoName}})
{{- end}}
{{- end}}
{{- range $s.ConstraintChecks}}
{{- $val := printf "%s.%s" $s.Receiver .GoName}}
{{- if .Deref}}
{{- $val = printf "*%s" $val}}
{{- end}}
{{- if .Opens}}
	if {{$s.Receiver}}.{{.GoName}} != nil {
{{- end}}
{{- if eq .Kind "non_empty_string"}}
	v.NotEmptyString("{{.FieldName}}", {{$val}})
{{- else if eq .Kind "non_zero"}}
	v.NonZero("{{.FieldName}}", {{$val}})
{{- else if eq .Kind "min_len"}}
	v.MinLength("{{.FieldName}}", {{$val}}, {{.Arg}})
{{- else if eq .Kind "max_len"}}
	v.MaxLength("{{.FieldName}}", {{$val}}, {{.Arg}})
{{- else if eq .Kind "ge"}}
	v.GreaterThanEq("{{.FieldName}}", {{$val}}, {{.Arg}})
{{- else if eq .Kind "gt"}}
	v.GreaterThan("{{.FieldName}}", {{$val}}, {{.Arg}})
{{- else if eq .Kind "le"}}
	v.LessThanEq("{{.FieldName}}", {{$val}}, {{.Arg}})
{{- else if eq .Kind "finite"}}
	v.Finite("{{.FieldName}}", {{$val}})
{{- end}}
{{- if .Closes}}
	}
{{- end}}
{{- end}}
{{- range $s.ValidateRecurse}}
{{- if eq (printf "%s" .Kind) "value"}}
{{- if .JSONName}}
	v.Exec(func() error { return validate.PathedError({{$s.Receiver}}.{{.GoName}}.Validate(), "{{.JSONName}}") })
{{- else}}
	v.Exec({{$s.Receiver}}.{{.GoName}}.Validate)
{{- end}}
{{- else if eq (printf "%s" .Kind) "pointer"}}
	if {{$s.Receiver}}.{{.GoName}} != nil {
		v.Exec(func() error { return validate.PathedError({{$s.Receiver}}.{{.GoName}}.Validate(), "{{.JSONName}}") })
	}
{{- else if eq (printf "%s" .Kind) "slice"}}
	for i := range {{$s.Receiver}}.{{.GoName}} {
		v.Exec(func() error { return validate.PathedError({{$s.Receiver}}.{{.GoName}}[i].Validate(), "{{.JSONName}}", strconv.Itoa(i)) })
	}
{{- else if eq (printf "%s" .Kind) "nested_slice"}}
	for i := range {{$s.Receiver}}.{{.GoName}} {
		for j := range {{$s.Receiver}}.{{.GoName}}[i] {
			v.Exec(func() error { return validate.PathedError({{$s.Receiver}}.{{.GoName}}[i][j].Validate(), "{{.JSONName}}", strconv.Itoa(i), strconv.Itoa(j)) })
		}
	}
{{- else if eq (printf "%s" .Kind) "map"}}
	for key, value := range {{$s.Receiver}}.{{.GoName}} {
		v.Exec(func() error { return validate.PathedError(value.Validate(), "{{.JSONName}}", key) })
	}
{{- end}}
{{- end}}
	return v.Error()
}
{{- end}}
{{- if .ToAny}}

// ToAny converts {{$s.Receiver}} to a {{$s.AnyInstance}}.
func ({{$s.Receiver}} {{$s.Instance}}) ToAny() {{$s.AnyInstance}} {
	return {{$s.AnyInstance}}{
{{- range $s.Fields}}
		{{.GoName}}: {{$s.Receiver}}.{{.GoName}},
{{- end}}
	}
}
{{- end}}
{{- end}}
{{- with .Union}}
{{- $u := .}}
{{- if not .BorrowsEnum}}

{{formatDoc .DiscType .DiscDoc}}
type {{.DiscType}} string
{{- if eq (len .Variants) 1}}
{{- with index .Variants 0}}

{{formatDoc .ConstName .ConstDoc}}
const {{.ConstName}} {{$u.DiscType}} = "{{.Value}}"
{{- end}}
{{- else}}

const (
{{- range .Variants}}
	{{formatFieldDoc .ConstName .ConstDoc | printf "%s"}}
	{{.ConstName}} {{$u.DiscType}} = "{{.Value}}"
{{- end}}
)
{{- end}}
{{- end}}

{{formatDoc .InterfaceName .InterfaceDoc}}
type {{.InterfaceName}} interface {
	{{.Marker}}()
}
{{range .Variants}}
{{- if .Doc}}
{{formatDoc .TypeName .Doc}}
{{- end}}
type {{.TypeName}} struct {
{{- range .Embeds}}
	{{.}}
{{- end}}
{{- range .Fields}}
{{- if .Doc}}
	{{formatFieldDoc .GoName .Doc | printf "%s"}}
{{- end}}
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JSONTag}}" msgpack:"{{.MsgpackTag}}"` + "`" + `
{{- end}}
}

func ({{.TypeName}}) {{$u.Marker}}() {}
{{- template "setDefaults" .SetDefaultsView}}
{{- $vt := .}}
{{- if .NeedsValidate}}

// Validate returns an error wrapping validate.ErrValidation if any field violates its
// schema constraints.
func ({{$vt.Receiver}} {{$vt.TypeName}}) Validate() error {
	v := validate.New("{{$vt.TypeName}}")
{{- range $vt.EnumChecks}}
{{- if .MapKey}}
	for key := range {{$vt.Receiver}}.{{.GoName}} {
		v.Ternaryf("{{.FieldName}}", !key.IsValid(), "invalid {{.FieldName}} key: %v", key)
	}
{{- else if .Optional}}
	if {{$vt.Receiver}}.{{.GoName}} != nil {
		v.Ternaryf("{{.FieldName}}", !{{$vt.Receiver}}.{{.GoName}}.IsValid(), "invalid {{.FieldName}}: %v", *{{$vt.Receiver}}.{{.GoName}})
	}
{{- else}}
	v.Ternaryf("{{.FieldName}}", !{{$vt.Receiver}}.{{.GoName}}.IsValid(), "invalid {{.FieldName}}: %v", {{$vt.Receiver}}.{{.GoName}})
{{- end}}
{{- end}}
{{- range $vt.ConstraintChecks}}
{{- $val := printf "%s.%s" $vt.Receiver .GoName}}
{{- if .Deref}}
{{- $val = printf "*%s" $val}}
{{- end}}
{{- if .Opens}}
	if {{$vt.Receiver}}.{{.GoName}} != nil {
{{- end}}
{{- if eq .Kind "non_empty_string"}}
	v.NotEmptyString("{{.FieldName}}", {{$val}})
{{- else if eq .Kind "non_zero"}}
	v.NonZero("{{.FieldName}}", {{$val}})
{{- else if eq .Kind "min_len"}}
	v.MinLength("{{.FieldName}}", {{$val}}, {{.Arg}})
{{- else if eq .Kind "max_len"}}
	v.MaxLength("{{.FieldName}}", {{$val}}, {{.Arg}})
{{- else if eq .Kind "ge"}}
	v.GreaterThanEq("{{.FieldName}}", {{$val}}, {{.Arg}})
{{- else if eq .Kind "gt"}}
	v.GreaterThan("{{.FieldName}}", {{$val}}, {{.Arg}})
{{- else if eq .Kind "le"}}
	v.LessThanEq("{{.FieldName}}", {{$val}}, {{.Arg}})
{{- else if eq .Kind "finite"}}
	v.Finite("{{.FieldName}}", {{$val}})
{{- end}}
{{- if .Closes}}
	}
{{- end}}
{{- end}}
{{- range $vt.ValidateRecurse}}
{{- if eq (printf "%s" .Kind) "value"}}
{{- if .JSONName}}
	v.Exec(func() error { return validate.PathedError({{$vt.Receiver}}.{{.GoName}}.Validate(), "{{.JSONName}}") })
{{- else}}
	v.Exec({{$vt.Receiver}}.{{.GoName}}.Validate)
{{- end}}
{{- else if eq (printf "%s" .Kind) "pointer"}}
	if {{$vt.Receiver}}.{{.GoName}} != nil {
		v.Exec(func() error { return validate.PathedError({{$vt.Receiver}}.{{.GoName}}.Validate(), "{{.JSONName}}") })
	}
{{- else if eq (printf "%s" .Kind) "slice"}}
	for i := range {{$vt.Receiver}}.{{.GoName}} {
		v.Exec(func() error { return validate.PathedError({{$vt.Receiver}}.{{.GoName}}[i].Validate(), "{{.JSONName}}", strconv.Itoa(i)) })
	}
{{- else if eq (printf "%s" .Kind) "nested_slice"}}
	for i := range {{$vt.Receiver}}.{{.GoName}} {
		for j := range {{$vt.Receiver}}.{{.GoName}}[i] {
			v.Exec(func() error { return validate.PathedError({{$vt.Receiver}}.{{.GoName}}[i][j].Validate(), "{{.JSONName}}", strconv.Itoa(i), strconv.Itoa(j)) })
		}
	}
{{- else if eq (printf "%s" .Kind) "map"}}
	for key, value := range {{$vt.Receiver}}.{{.GoName}} {
		v.Exec(func() error { return validate.PathedError(value.Validate(), "{{.JSONName}}", key) })
	}
{{- end}}
{{- end}}
	return v.Error()
}
{{- end}}
{{end -}}
{{if .Doc}}{{formatDoc .Name .Doc}}
{{end -}}
type {{.Name}} struct {
	Variant {{.InterfaceName}}
}
{{$u := .}}
// MarshalJSONTo encodes the active variant with its "{{.DiscJSONName}}" tag injected.
func (u {{.Name}}) MarshalJSONTo(enc *jsontext.Encoder) error {
	switch v := u.Variant.(type) {
	case nil:
		return enc.WriteToken(jsontext.Null)
{{- range .Variants}}
	case {{.TypeName}}:
		return json.MarshalEncode(enc, struct {
			Type {{$u.DiscType}} ` + "`" + `json:"{{$u.DiscJSONName}}"` + "`" + `
			{{.TypeName}}
		}{Type: {{.ConstName}}, {{.TypeName}}: v})
{{- end}}
	default:
		return errors.Newf("{{.Name}}: unknown variant %T", v)
	}
}

// UnmarshalJSONFrom decodes the variant selected by the "{{.DiscJSONName}}" field.
func (u *{{.Name}}) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	data, err := dec.ReadValue()
	if err != nil {
		return err
	}
	if data.Kind() == 'n' {
		u.Variant = nil
		return nil
	}
	opts := dec.Options()
	var disc struct {
		Type {{.DiscType}} ` + "`" + `json:"{{.DiscJSONName}}"` + "`" + `
	}
	if err := json.Unmarshal(
		data, &disc, opts, json.RejectUnknownMembers(false),
	); err != nil {
		return err
	}
	switch disc.Type {
{{- range .Variants}}
	case {{.ConstName}}:
		var v struct {
			Type {{$u.DiscType}} ` + "`" + `json:"{{$u.DiscJSONName}}"` + "`" + `
			{{.TypeName}}
		}
{{- if .SetDefaultsView.HasGroups}}
		v.{{.TypeName}}.SetUngroupedDefaults()
{{- else if .SetDefaultsView.Sets}}
		v.{{.TypeName}}.SetDefaults()
{{- end}}
		if err := json.Unmarshal(data, &v, opts); err != nil {
			return err
		}
{{- if .SetDefaultsView.HasGroups}}
		v.{{.TypeName}}.DefaultGroups()
{{- end}}
		u.Variant = v.{{.TypeName}}
{{- end}}
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"{{.Name}}: unknown {{.DiscJSONName}} %q",
			disc.Type,
		)
	}
	return nil
}
{{- if .TagMethod}}

{{formatDoc .TagMethod (printf "returns the %s of the active variant, or an empty %s when there is none." .DiscJSONName .DiscJSONName)}}
func (u {{.Name}}) {{.TagMethod}}() {{.DiscType}} {
	switch u.Variant.(type) {
{{- range .Variants}}
	case {{.TypeName}}:
		return {{.ConstName}}
{{- end}}
	}
	return ""
}
{{- end}}

// Validate returns an error wrapping validate.ErrValidation if no variant is active or
// the active variant violates its schema constraints.
func (u {{.Name}}) Validate() error {
	v := validate.New("{{.Name}}")
	v.Ternary("{{.DiscJSONName}}", u.Variant == nil, "required")
{{- if .ValidatesVariants}}
	switch variant := u.Variant.(type) {
{{- range .Variants}}
{{- if .NeedsValidate}}
	case {{.TypeName}}:
		v.Exec(variant.Validate)
{{- end}}
{{- end}}
	}
{{- end}}
	return v.Error()
}
{{- if .Base}}

// Base returns the fields every variant shares, or the zero value if Variant is nil.
func (u {{.Name}}) Base() {{.Base}} {
	switch variant := u.Variant.(type) {
{{- range .Variants}}
	case {{.TypeName}}:
		return variant.{{$u.BaseField}}
{{- end}}
	}
	return {{.Base}}{}
}

// SetBase replaces the fields every variant shares. It panics if Variant is nil.
func (u *{{.Name}}) SetBase(base {{.Base}}) {
	switch variant := u.Variant.(type) {
{{- range .Variants}}
	case {{.TypeName}}:
		variant.{{$u.BaseField}} = base
		u.Variant = variant
{{- end}}
	default:
		panic(errors.Newf("{{.Name}}: cannot set the base of variant %T", variant))
	}
}
{{- end}}
{{- end}}
{{- end}}
{{- define "setFills"}}
{{- $r := .Receiver}}
{{- range .Fills}}
{{- if .Setup}}
	{
{{- range .Setup}}
		{{.}}
{{- end}}
		{{$r}}.{{.GoName}} = {{.Expr}}
	}
{{- else}}
	{{$r}}.{{.GoName}} = {{.Expr}}
{{- end}}
{{- end}}
{{- end}}
{{- define "setDefaults"}}
{{- $r := .Receiver}}
{{- if .HasGroups}}

// SetDefaults sets every field outside a default group to its schema default. Each
// default group takes its defaults only when all of its members are zero.
func ({{$r}} *{{.Name}}) SetDefaults() {
	{{$r}}.SetUngroupedDefaults()
	{{$r}}.DefaultGroups()
}

// SetUngroupedDefaults sets every field that has a schema default outside a default
// group to that default.
func ({{$r}} *{{.Name}}) SetUngroupedDefaults() {
{{- range .Steps}}
{{- if .Groups}}
	{{$r}}.{{.GoName}}.SetUngroupedDefaults()
{{- else}}
	{{$r}}.{{.GoName}}.SetDefaults()
{{- end}}
{{- end}}
{{- template "setFills" .}}
}

// DefaultGroups sets the defaults of each default group whose members are all zero, in
// this type and in the types it embeds or holds. A group with a non-zero member keeps
// its values.
func ({{$r}} *{{.Name}}) DefaultGroups() {
{{- range .Groups}}
	if {{range $i, $m := .Members}}{{if $i}} && {{end}}{{$r}}.{{$m.GoName}} == {{$m.ZeroLit}}{{end}} {
{{- range .Fills}}
		{{$r}}.{{.GoName}} = {{.Expr}}
{{- end}}
	}
{{- end}}
{{- range .Steps}}
{{- if .Groups}}
	{{$r}}.{{.GoName}}.DefaultGroups()
{{- end}}
{{- end}}
}
{{- else if .Sets}}

// SetDefaults sets every field that has a schema default to that default.
func ({{$r}} *{{.Name}}) SetDefaults() {
{{- range .Steps}}
	{{$r}}.{{.GoName}}.SetDefaults()
{{- end}}
{{- template "setFills" .}}
}
{{- end}}
{{- end}}
`),
)
