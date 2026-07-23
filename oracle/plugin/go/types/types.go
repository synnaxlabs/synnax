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
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	goprimitives "github.com/synnaxlabs/oracle/plugin/go/primitives"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

const goModulePrefix = "github.com/synnaxlabs/synnax/"

// fieldDocIndent is the display width of the single tab indenting field-level
// doc comments (tabs render at four columns per .editorconfig).
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

// Check verifies generated files are up-to-date. Currently unimplemented.
func (p *Plugin) Check(*plugin.Request) error { return nil }

// Generate produces Go type definitions for structs, enums, and typedefs with @go flag.
// Version-laid-out packages (@go version + a keyed struct) emit their types into the
// current types/vN sub-package, with a root alias file re-exporting the surface. Types
// whose shape is unchanged from the predecessor version alias it instead of being
// re-defined. Version-laid-out packages pin persisted references to their
// dependencies' current version directories (they must stay importable from frozen
// packages); omitted fields and declarations outside the persisted closure resolve
// to dependency roots and track the latest version, since they never reach the
// stored wire format. Unversioned packages reference the root re-export surface.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	rewritten, pathMap, err := versioning.RewriteCurrent(req.Resolutions)
	if err != nil {
		return nil, err
	}
	preds, err := computeSplit(req, rewritten)
	if err != nil {
		return nil, err
	}
	currentPaths := make(set.Set[string], len(pathMap))
	for _, current := range pathMap {
		currentPaths.Add(current)
	}
	closure := PersistedClosure(rewritten)
	versionedGen := &framework.Generator{
		Domain:      "go",
		FilePattern: p.Options.FileNamePattern,
		FileGenerator: &goFileGenerator{
			preds:    preds,
			original: req.Resolutions,
			pathMap:  pathMap,
			closure:  closure,
		},
		PathFilter:      currentPaths.Contains,
		MergeByName:     false,
		CollectTypeDefs: true,
		CollectEnums:    true,
		CollectUnions:   true,
	}
	versionedReq := *req
	versionedReq.Resolutions = rewritten
	resp, err := versionedGen.Generate(&versionedReq)
	if err != nil {
		return nil, err
	}
	unversionedGen := &framework.Generator{
		Domain:        "go",
		FilePattern:   p.Options.FileNamePattern,
		FileGenerator: &goFileGenerator{},
		PathFilter: func(outputPath string) bool {
			_, versioned := pathMap[outputPath]
			return !versioned
		},
		MergeByName:     false,
		CollectTypeDefs: true,
		CollectEnums:    true,
		CollectUnions:   true,
	}
	unversionedResp, err := unversionedGen.Generate(req)
	if err != nil {
		return nil, err
	}
	resp.Files = append(resp.Files, unversionedResp.Files...)
	if len(pathMap) == 0 {
		return resp, nil
	}
	pathFilter := func(outputPath string) bool { _, ok := pathMap[outputPath]; return ok }
	// Transient types — those without their own @go version at a versioned
	// path — stay at the package root, generating real declarations into a
	// transient.gen.go beside the root alias file. Referenced enums that live
	// in the version layout must not be re-declared there.
	transientGen := &framework.Generator{
		Domain:        "go",
		FilePattern:   "transient.gen.go",
		FileGenerator: &goFileGenerator{},
		PathFilter:    pathFilter,
		FilterEnums: func(enums []resolution.Type, outputPath string) []resolution.Type {
			filtered := make([]resolution.Type, 0, len(enums))
			for _, e := range enums {
				if output.GetPath(e, "go") == outputPath {
					filtered = append(filtered, e)
				}
			}
			return filtered
		},
		MergeByName:     false,
		CollectTypeDefs: true,
		CollectEnums:    true,
		CollectUnions:   true,
	}
	transientResp, err := transientGen.Generate(&versionedReq)
	if err != nil {
		return nil, err
	}
	resp.Files = append(resp.Files, transientResp.Files...)
	aliasGen := &framework.Generator{
		Domain:          "go",
		FilePattern:     p.Options.FileNamePattern,
		FileGenerator:   &aliasFileGenerator{pathMap: pathMap},
		PathFilter:      pathFilter,
		MergeByName:     false,
		CollectTypeDefs: true,
		CollectEnums:    true,
		CollectUnions:   true,
	}
	aliasResp, err := aliasGen.Generate(req)
	if err != nil {
		return nil, err
	}
	resp.Files = append(resp.Files, aliasResp.Files...)
	selectorGen := &framework.Generator{
		Domain:          "go",
		FilePattern:     "types/" + p.Options.FileNamePattern,
		FileGenerator:   &aliasFileGenerator{pathMap: pathMap, pkg: "types"},
		PathFilter:      pathFilter,
		MergeByName:     false,
		CollectTypeDefs: true,
		CollectEnums:    true,
		CollectUnions:   true,
	}
	selectorResp, err := selectorGen.Generate(req)
	if err != nil {
		return nil, err
	}
	resp.Files = append(resp.Files, selectorResp.Files...)
	return resp, nil
}

// predecessor identifies the version package that a current version package's
// unchanged types alias.
type predecessor struct {
	// path is the repo-relative predecessor package directory (…/types/vN-1).
	path string
	// aliased holds qualified names of types emitted as aliases.
	aliased set.Set[string]
}

// predecessors maps each version-laid-out current output path (…/types/vN) to
// its predecessor alias split. Empty when no snapshots resolve a baseline.
func predecessors(req *plugin.Request) (map[string]predecessor, error) {
	split, err := versioning.AliasSplit(
		req.Resolutions, req.SnapshotVersion, req.LoadSnapshot,
	)
	if err != nil || len(split) == 0 {
		return nil, err
	}
	entries, err := versioning.EntryPaths(req.Resolutions)
	if err != nil {
		return nil, err
	}
	preds := make(map[string]predecessor, len(split))
	for origPath, pa := range split {
		current := versioning.VersionedPath(origPath, entries[origPath])
		preds[current] = predecessor{
			path:    versioning.VersionedPath(origPath, pa.PredecessorVersion),
			aliased: pa.Aliased,
		}
	}
	return preds, nil
}

// captureGenerator records each output path's collected declaration lists
// without emitting files.
type captureGenerator struct {
	contexts map[string]*framework.GenerateContext
}

func (c *captureGenerator) GenerateFile(ctx *framework.GenerateContext) (string, error) {
	c.contexts[ctx.OutputPath] = ctx
	return "", nil
}

// computeSplit resolves the alias split for every version-laid-out current
// package. Snapshot-declared baselines decide first; for paths the snapshots
// cannot anchor (history predating per-resource versioning), the frozen
// predecessor package on disk is the baseline: a type aliases only when its
// declarations are identical to the frozen ones.
func computeSplit(
	req *plugin.Request, rewritten *resolution.Table,
) (map[string]predecessor, error) {
	preds, err := predecessors(req)
	if err != nil {
		return nil, err
	}
	if preds == nil {
		preds = make(map[string]predecessor)
	}
	entries, err := versioning.EntryPaths(req.Resolutions)
	if err != nil {
		return nil, err
	}
	pathMap, err := versioning.CurrentPathMap(req.Resolutions)
	if err != nil {
		return nil, err
	}
	closure := PersistedClosure(rewritten)
	capture := &captureGenerator{contexts: make(map[string]*framework.GenerateContext)}
	capReq := *req
	capReq.Resolutions = rewritten
	capGen := &framework.Generator{
		Domain:          "go",
		FilePattern:     captureFilePattern,
		FileGenerator:   capture,
		MergeByName:     false,
		CollectTypeDefs: true,
		CollectEnums:    true,
		CollectUnions:   true,
	}
	if _, err := capGen.Generate(&capReq); err != nil {
		return nil, err
	}
	for origPath, version := range entries {
		if version == 0 {
			continue
		}
		current := versioning.VersionedPath(origPath, version)
		if _, ok := preds[current]; ok {
			continue
		}
		ctx, ok := capture.contexts[current]
		if !ok {
			continue
		}
		predDir := filepath.Join(
			req.RepoRoot, versioning.VersionedPath(origPath, version-1),
		)
		if _, err := os.Stat(predDir); err != nil {
			continue
		}
		candidate, err := generateGoFile(
			current, ctx.Structs, ctx.Enums, ctx.TypeDefs, ctx.Unions,
			rewritten, req.RepoRoot, predecessor{},
			latestTable(req.Resolutions, pathMap, current), closure,
		)
		if err != nil {
			return nil, err
		}
		aliased, err := frozenAliasSplit(candidate, ctx, predDir)
		if err != nil {
			return nil, errors.Wrapf(err, "frozen baseline for %s", origPath)
		}
		if len(aliased) > 0 {
			preds[current] = predecessor{
				path:    versioning.VersionedPath(origPath, version-1),
				aliased: aliased,
			}
		}
	}
	return preds, nil
}

// captureFilePattern is never written; the capture generator emits nothing.
const captureFilePattern = "capture"

// AliasedTypes returns the qualified names of all types that alias their
// predecessor version package instead of being defined at the current one.
// Sibling plugins (go/marshal) use it to skip emission for aliased types.
func AliasedTypes(req *plugin.Request) (set.Set[string], error) {
	rewritten, _, err := versioning.RewriteCurrent(req.Resolutions)
	if err != nil {
		return nil, err
	}
	preds, err := computeSplit(req, rewritten)
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
	// preds maps current version output paths to their predecessor alias
	// split; types in the split alias the predecessor instead of defining.
	preds map[string]predecessor
	// original is the unrewritten table; set only for the versioned pass,
	// where transient declarations resolve against it (self path rewritten)
	// so their dependency references track the latest version.
	original *resolution.Table
	// pathMap maps each version-laid-out root path to its current types/vN
	// sub-path.
	pathMap map[string]string
	// closure is the persisted closure of the whole table; declarations
	// outside it at marshal-rooted paths are transient.
	closure set.Set[string]
}

func (g *goFileGenerator) GenerateFile(ctx *framework.GenerateContext) (string, error) {
	latest := latestTable(g.original, g.pathMap, ctx.OutputPath)
	content, err := generateGoFile(
		ctx.OutputPath, ctx.Structs, ctx.Enums, ctx.TypeDefs, ctx.Unions,
		ctx.Table, ctx.RepoRoot, g.preds[ctx.OutputPath], latest, g.closure,
	)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// latestTable returns the latest-resolution table for a current version path:
// the original table with only this path rewritten to its version directory,
// so local references stay in-package while dependency references resolve to
// the root re-export. Nil when original is nil or the path is not versioned.
func latestTable(
	original *resolution.Table, pathMap map[string]string, outputPath string,
) *resolution.Table {
	if original == nil {
		return nil
	}
	for orig, current := range pathMap {
		if current == outputPath {
			return versioning.RewriteOutputPaths(
				original, map[string]string{orig: outputPath},
			)
		}
	}
	return nil
}

// PersistedClosure returns the qualified names of every type reachable from a
// @go marshal root through stored references: non-omitted struct fields,
// extends, type arguments, alias targets, distinct bases, and union variants.
// Types outside the closure never reach a table's wire format.
func PersistedClosure(table *resolution.Table) set.Set[string] {
	closure := make(set.Set[string])
	var walkType func(t resolution.Type)
	var walkRef func(ref resolution.TypeRef)
	walkRef = func(ref resolution.TypeRef) {
		for _, arg := range ref.TypeArgs {
			walkRef(arg)
		}
		if resolved, ok := ref.Resolve(table); ok {
			walkType(resolved)
		}
	}
	walkType = func(t resolution.Type) {
		if closure.Contains(t.QualifiedName) {
			return
		}
		closure.Add(t.QualifiedName)
		switch form := t.Form.(type) {
		case resolution.StructForm:
			for _, ext := range form.Extends {
				walkRef(ext)
			}
			for _, tp := range form.TypeParams {
				if tp.Constraint != nil {
					walkRef(*tp.Constraint)
				}
				if tp.Default != nil {
					walkRef(*tp.Default)
				}
			}
			for _, f := range schemadiff.PersistedFields(resolution.UnifiedFields(t, table)) {
				walkRef(f.Type)
			}
		case resolution.EnumForm:
			for _, ext := range form.Extends {
				walkRef(ext)
			}
		case resolution.AliasForm:
			walkRef(form.Target)
		case resolution.DistinctForm:
			walkRef(form.Base)
		case resolution.UnionForm:
			for _, ext := range form.Extends {
				walkRef(ext)
			}
			for _, v := range form.Variants {
				walkRef(v.Type)
			}
		}
	}
	for _, t := range table.Types {
		if domain.HasExprFromType(t, "go", "marshal") ||
			domain.HasExprFromType(t, "go", "migrate") {
			walkType(t)
		}
	}
	return closure
}

// generateGoFile generates a Go types file for the given structs, enums, and
// type definitions at the specified output path. Types named in pred.aliased
// are emitted as aliases to the predecessor version package instead of full
// definitions.
func generateGoFile(
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	typeDefs []resolution.Type,
	unions []resolution.Type,
	table *resolution.Table,
	repoRoot string,
	pred predecessor,
	latest *resolution.Table,
	closure set.Set[string],
) ([]byte, error) {
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	} else if len(unions) > 0 {
		namespace = unions[0].Namespace
	} else if len(typeDefs) > 0 {
		namespace = typeDefs[0].Namespace
	} else if len(enums) > 0 {
		namespace = enums[0].Namespace
	}

	pkg := naming.DerivePackageName(outputPath)
	imports := imports.NewManager()

	ctx := &resolver.Context{
		Table:                         table,
		OutputPath:                    outputPath,
		Namespace:                     namespace,
		RepoRoot:                      repoRoot,
		DomainName:                    "go",
		SubstituteDefaultedTypeParams: true,
	}

	r := &resolver.Resolver{
		Formatter:       GoFormatter(),
		ImportResolver:  &GoImportResolver{RepoRoot: repoRoot, CurrentPackage: pkg},
		ImportAdder:     imports,
		PrimitiveMapper: primitiveMapper,
	}

	data := &templateData{
		Package:    pkg,
		OutputPath: outputPath,
		Namespace:  namespace,
		imports:    imports,
		table:      table,
		repoRoot:   repoRoot,
		resolver:   r,
		ctx:        ctx,
	}

	// Transient declarations — outside the persisted closure of a
	// marshal-rooted package — never reach the stored wire format, so their
	// dependency references resolve against the latest table and track the
	// dependencies' current versions instead of pinning one.
	var latestData *templateData
	if latest != nil && pathHasMarshalRoot(structs) {
		lctx := *ctx
		lctx.Table = latest
		ld := *data
		ld.table = latest
		ld.ctx = &lctx
		latestData = &ld
		data.latest = latestData
	}

	var predAlias string
	for _, d := range orderDecls(table, typeDefs, enums, structs, unions) {
		if omit.IsSkipped(d.typ, "go") {
			continue
		}
		if d.kind == declEnum && d.typ.Namespace != namespace {
			continue
		}
		if pred.aliased.Contains(d.typ.QualifiedName) {
			if predAlias == "" {
				predAlias = naming.DerivePackageName(pred.path)
				imports.AddInternal(predAlias, resolveGoImportPath(pred.path, repoRoot))
			}
			for _, a := range buildAliasDecls(d, predAlias, data) {
				data.Decls = append(data.Decls, declData{Alias: &a})
			}
			continue
		}
		procData := data
		if latestData != nil && !closure.Contains(d.typ.QualifiedName) {
			procData = latestData
		}
		switch d.kind {
		case declTypeDef:
			td := processTypeDef(d.typ, procData)
			data.Decls = append(data.Decls, declData{TypeDef: &td})
		case declEnum:
			e := processEnum(d.typ)
			data.Decls = append(data.Decls, declData{Enum: &e})
		case declStruct:
			s := processStruct(d.typ, procData)
			data.Decls = append(data.Decls, declData{Struct: &s})
		case declUnion:
			u := processUnion(d.typ, procData)
			data.Decls = append(data.Decls, declData{Union: &u})
		}
	}

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func resolveGoImportPath(outputPath, repoRoot string) string {
	return gomod.ResolveImportPath(outputPath, repoRoot, goModulePrefix)
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

// orderDecls merges the kind-grouped type lists into schema declaration
// order, using the table's registration order as the source position.
func orderDecls(
	table *resolution.Table,
	typeDefs, enums, structs, unions []resolution.Type,
) []orderedDecl {
	index := make(map[string]int, len(table.Types))
	for i, t := range table.Types {
		index[t.QualifiedName] = i
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
	form := e.Form.(resolution.EnumForm)
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
		targetRef := form.Target
		if targetResolved, ok := targetRef.Resolve(data.table); ok {
			if targetForm, ok := targetResolved.Form.(resolution.StructForm); ok {
				nonDefaultedParams := resolution.NonDefaultedTypeParams(targetForm.TypeParams)
				providedArgs := len(targetRef.TypeArgs)
				if providedArgs < len(nonDefaultedParams) {
					newTypeArgs := make([]resolution.TypeRef, len(nonDefaultedParams))
					copy(newTypeArgs, targetRef.TypeArgs)
					for i := providedArgs; i < len(nonDefaultedParams); i++ {
						if nonDefaultedParams[i].Optional {
							newTypeArgs[i] = resolution.TypeRef{Name: "nil"}
						}
					}
					targetRef = resolution.TypeRef{
						Name:     targetRef.Name,
						TypeArgs: newTypeArgs,
					}
				}
			}
		}
		baseType := data.resolver.ResolveTypeRef(targetRef, data.ctx)
		result := typeDefData{
			Name:     name,
			Doc:      doc.Get(td.Domains),
			BaseType: baseType,
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

func processStruct(entry resolution.Type, data *templateData) structData {
	form := entry.Form.(resolution.StructForm)
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

	if len(form.Extends) > 0 {
		// Flatten (rather than embed the parent) when fields are omitted, parents
		// conflict, or a field removes an inherited domain — none can be expressed
		// through Go struct embedding.
		if len(form.OmittedFields) > 0 ||
			resolver.HasFieldConflicts(form.Extends, data.table) ||
			resolver.HasDomainOmissions(form) {
			for _, field := range resolution.UnifiedFields(entry, data.table) {
				sd.Fields = append(sd.Fields, processField(field, data))
			}
			sd.ExtraFields = domain.GetAllStringsFromType(entry, "go", "fields")
			for _, imp := range domain.GetAllStringsFromType(entry, "go", "imports") {
				data.imports.AddExternal(imp)
			}
			return sd
		}

		sd.HasExtends = true
		for _, extendsRef := range form.Extends {
			parent, ok := extendsRef.Resolve(data.table)
			if ok {
				sd.ExtendsTypes = append(sd.ExtendsTypes, resolveExtendsType(extendsRef, parent, data))
			}
		}

		for _, field := range form.Fields {
			sd.Fields = append(sd.Fields, processField(field, data))
		}
		sd.ExtraFields = domain.GetAllStringsFromType(entry, "go", "fields")
		for _, imp := range domain.GetAllStringsFromType(entry, "go", "imports") {
			data.imports.AddExternal(imp)
		}
		return sd
	}

	genMethods := !sd.IsGeneric
	for _, field := range resolution.UnifiedFields(entry, data.table) {
		sd.Fields = append(sd.Fields, processField(field, data))
		if !genMethods {
			continue
		}
		sd.DefaultFills = append(sd.DefaultFills, goDefaultFills(field, data)...)
		if step, ok := goRecurseStep(field, data, defaultsHasOwn, neverSkip); ok {
			sd.DefaultRecurse = append(sd.DefaultRecurse, step)
		}
		if validateSkip(field, data) {
			continue
		}
		if chk, ok := goEnumCheck(field, data); ok {
			sd.EnumChecks = append(sd.EnumChecks, chk)
		}
		sd.ConstraintChecks = append(sd.ConstraintChecks, goConstraintChecks(field, data)...)
		if step, ok := goRecurseStep(field, data, validateHasOwn, validateSkip); ok {
			sd.ValidateRecurse = append(sd.ValidateRecurse, step)
		}
	}
	if len(sd.EnumChecks) > 0 || len(sd.ConstraintChecks) > 0 || len(sd.ValidateRecurse) > 0 {
		data.imports.AddExternal(validateImportPath)
	}
	if hasSliceRecurse(sd.ValidateRecurse) {
		data.imports.AddExternal(strconvImportPath)
	}
	if len(sd.Name) > 0 {
		sd.Receiver = strings.ToLower(sd.Name[:1])
	}

	sd.ExtraFields = domain.GetAllStringsFromType(entry, "go", "fields")

	for _, imp := range domain.GetAllStringsFromType(entry, "go", "imports") {
		data.imports.AddExternal(imp)
	}

	return sd
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

func processField(field resolution.Field, data *templateData) fieldData {
	if data.latest != nil &&
		domain.GetStringFromField(field, "go", "marshal") == "omit" {
		data = data.latest
	}
	goType := data.resolver.ResolveTypeRef(field.Type, data.ctx)
	if field.Optional && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map[") && !strings.HasPrefix(goType, "msgpack.EncodedJSON") {
		goType = "*" + goType
	}
	// Collection fields (arrays, maps, records) carry `,omitzero` so a nil ("not
	// loaded") collection is omitted from the wire while an allocated empty
	// collection still serializes as [] / {}. Receivers default an absent
	// collection to its empty form, preserving the distinction between "not
	// loaded" and "present but empty". CollectionKind sees through aliases and
	// type-parameter constraints so a field typed as an array/map alias or a
	// collection-constrained type parameter is tagged too.
	_, isContainer := resolution.CollectionKind(field.Type, data.table)
	return fieldData{
		GoName:      naming.GetFieldName(field),
		GoType:      goType,
		JSONName:    casing.FieldSnake(field.Name),
		IsOptional:  field.Optional,
		IsContainer: isContainer,
		Doc:         doc.Get(field.Domains),
	}
}

func buildGenericType(baseName string, typeArgs []resolution.TypeRef, targetType *resolution.Type, data *templateData) string {
	if len(typeArgs) == 0 {
		return baseName
	}

	var args []string
	if targetType != nil {
		if form, ok := targetType.Form.(resolution.StructForm); ok {
			for i, arg := range typeArgs {
				if i < len(form.TypeParams) && form.TypeParams[i].HasDefault() {
					continue
				}
				args = append(args, data.resolver.ResolveTypeRef(arg, data.ctx))
			}
		} else {
			for _, arg := range typeArgs {
				args = append(args, data.resolver.ResolveTypeRef(arg, data.ctx))
			}
		}
	} else {
		for _, arg := range typeArgs {
			args = append(args, data.resolver.ResolveTypeRef(arg, data.ctx))
		}
	}

	if len(args) == 0 {
		return baseName
	}
	return fmt.Sprintf("%s[%s]", baseName, strings.Join(args, ", "))
}

func resolveExtendsType(extendsRef resolution.TypeRef, parent resolution.Type, data *templateData) string {
	targetOutputPath := output.GetPath(parent, "go")

	name := naming.GetGoName(parent)

	if parent.Namespace == data.Namespace && (targetOutputPath == "" || targetOutputPath == data.OutputPath) {
		return buildGenericType(name, extendsRef.TypeArgs, &parent, data)
	}

	if targetOutputPath == "" {
		return name
	}
	alias := naming.DerivePackageAlias(targetOutputPath, data.Package)
	data.imports.AddInternal(alias, resolveGoImportPath(targetOutputPath, data.repoRoot))
	return fmt.Sprintf("%s.%s", alias, buildGenericType(name, extendsRef.TypeArgs, &parent, data))
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
	imports  *imports.Manager
	table    *resolution.Table
	resolver *resolver.Resolver
	ctx      *resolver.Context
	// latest, when set, resolves omitted (memory-only) fields against the
	// latest table so they track dependencies' current versions.
	latest     *templateData
	Package    string
	OutputPath string
	Namespace  string
	repoRoot   string
	Decls      []declData
}

// declData wraps one processed declaration in schema order; exactly one field
// is non-nil.
type declData struct {
	TypeDef *typeDefData
	Enum    *enumData
	Struct  *structData
	Union   *unionData
	// Alias re-exports a type from the predecessor version package instead of
	// defining it.
	Alias *aliasDecl
}

// HasImports returns true if any imports are needed.
func (d *templateData) HasImports() bool { return d.imports.HasImports() }

// ExternalImports returns sorted external imports.
func (d *templateData) ExternalImports() []string { return d.imports.ExternalImports() }

// StdImports returns sorted standard-library imports.
func (d *templateData) StdImports() []string { return d.imports.StdImports() }

// NonStdImports returns every non-standard-library import sorted by path.
func (d *templateData) NonStdImports() []imports.InternalImportData {
	return d.imports.NonStdImports()
}

// InternalImports returns sorted internal imports.
func (d *templateData) InternalImports() []imports.InternalImportData {
	return d.imports.InternalImports()
}

type structData struct {
	Name             string
	Doc              string
	AliasOf          string
	Receiver         string
	Fields           []fieldData
	TypeParams       []typeParamData
	ExtendsTypes     []string
	ExtraFields      []string
	DefaultFills     []defaultFillData
	DefaultRecurse   []recurseStepData
	EnumChecks       []enumCheckData
	ConstraintChecks []constraintCheckData
	ValidateRecurse  []recurseStepData
	IsGeneric        bool
	IsAlias          bool
	HasExtends       bool
}

type typeParamData struct {
	Name       string
	Constraint string
}

type fieldData struct {
	GoName      string
	GoType      string
	JSONName    string
	Doc         string
	IsOptional  bool
	IsContainer bool
}

// TagSuffix returns the JSON/msgpack tag suffix for the field. Collection fields
// use `,omitzero` so a nil collection is omitted while an allocated empty one
// serializes as [] / {}. Other optional fields use `,omitempty`.
func (f fieldData) TagSuffix() string {
	if f.IsContainer {
		return ",omitzero"
	}
	if f.IsOptional {
		return ",omitempty"
	}
	return ""
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

var fileTemplate = template.Must(template.New("go-types").Funcs(templateFuncs).Parse(`// Code generated by oracle. DO NOT EDIT.

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

const {{$enum.Name}}{{.Name}} {{$enum.Name}} = {{if $enum.StartsAtOne}}1{{else}}0{{end}}
{{- end}}
{{- else}}

const (
{{- range $i, $v := $enum.Values}}
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

// IsValid reports whether {{$enum.Receiver}} is one of the defined {{$enum.Name}} values.
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
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JSONName}}{{.TagSuffix}}" msgpack:"{{.JSONName}}{{.TagSuffix}}"` + "`" + `
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
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JSONName}}{{.TagSuffix}}" msgpack:"{{.JSONName}}{{.TagSuffix}}"` + "`" + `
{{- end}}
{{- range .ExtraFields}}
	{{.}}
{{- end}}
}
{{end -}}
{{- $s := .}}
{{- if or .DefaultFills .DefaultRecurse}}

// ApplyDefaults fills zero-valued fields with their schema-declared defaults.
func ({{$s.Receiver}} *{{$s.Name}}) ApplyDefaults() {
{{- range $s.DefaultFills}}
	if {{$s.Receiver}}.{{.GoName}} == {{.ZeroLit}} {
		{{$s.Receiver}}.{{.GoName}} = {{.Expr}}
	}
{{- end}}
{{- range $s.DefaultRecurse}}
{{- if eq (printf "%s" .Kind) "value"}}
	{{$s.Receiver}}.{{.GoName}}.ApplyDefaults()
{{- else if eq (printf "%s" .Kind) "pointer"}}
	if {{$s.Receiver}}.{{.GoName}} != nil {
		{{$s.Receiver}}.{{.GoName}}.ApplyDefaults()
	}
{{- else if eq (printf "%s" .Kind) "slice"}}
	for i := range {{$s.Receiver}}.{{.GoName}} {
		{{$s.Receiver}}.{{.GoName}}[i].ApplyDefaults()
	}
{{- else if eq (printf "%s" .Kind) "map"}}
	for key, value := range {{$s.Receiver}}.{{.GoName}} {
		value.ApplyDefaults()
		{{$s.Receiver}}.{{.GoName}}[key] = value
	}
{{- end}}
{{- end}}
}
{{- end}}
{{- if or .EnumChecks .ConstraintChecks .ValidateRecurse}}

// Validate returns an error wrapping validate.ErrValidation if any field violates its
// schema constraints.
func ({{$s.Receiver}} {{$s.Name}}) Validate() error {
	v := validate.New("{{$s.Name}}")
{{- range $s.EnumChecks}}
	v.Ternaryf("{{.FieldName}}", !{{$s.Receiver}}.{{.GoName}}.IsValid(), "invalid {{.FieldName}}: %v", {{$s.Receiver}}.{{.GoName}})
{{- end}}
{{- range $s.ConstraintChecks}}
{{- if eq .Kind "non_empty_string"}}
	validate.NotEmptyString(v, "{{.FieldName}}", {{$s.Receiver}}.{{.GoName}})
{{- else if eq .Kind "non_zero"}}
	validate.NonZero(v, "{{.FieldName}}", {{$s.Receiver}}.{{.GoName}})
{{- else if eq .Kind "min_len"}}
	v.Ternaryf("{{.FieldName}}", len({{$s.Receiver}}.{{.GoName}}) < {{.Arg}}, "must be at least {{.Arg}} characters long")
{{- else if eq .Kind "max_len"}}
	v.Ternaryf("{{.FieldName}}", len({{$s.Receiver}}.{{.GoName}}) > {{.Arg}}, "must be at most {{.Arg}} characters long")
{{- else if eq .Kind "ge"}}
	validate.GreaterThanEq(v, "{{.FieldName}}", {{$s.Receiver}}.{{.GoName}}, {{.Arg}})
{{- else if eq .Kind "le"}}
	validate.LessThanEq(v, "{{.FieldName}}", {{$s.Receiver}}.{{.GoName}}, {{.Arg}})
{{- end}}
{{- end}}
{{- range $s.ValidateRecurse}}
{{- if eq (printf "%s" .Kind) "value"}}
	v.Exec(func() error { return validate.PathedError({{$s.Receiver}}.{{.GoName}}.Validate(), "{{.JSONName}}") })
{{- else if eq (printf "%s" .Kind) "pointer"}}
	if {{$s.Receiver}}.{{.GoName}} != nil {
		v.Exec(func() error { return validate.PathedError({{$s.Receiver}}.{{.GoName}}.Validate(), "{{.JSONName}}") })
	}
{{- else if eq (printf "%s" .Kind) "slice"}}
	for i := range {{$s.Receiver}}.{{.GoName}} {
		v.Exec(func() error { return validate.PathedError({{$s.Receiver}}.{{.GoName}}[i].Validate(), "{{.JSONName}}", strconv.Itoa(i)) })
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
{{- end}}
{{- with .Union}}
{{- $u := .}}

type {{.DiscType}} string

const (
{{- range .Variants}}
	{{.ConstName}} {{$u.DiscType}} = "{{.Value}}"
{{- end}}
)

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
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JSONName}}{{.TagSuffix}}" msgpack:"{{.JSONName}}{{.TagSuffix}}"` + "`" + `
{{- end}}
}

func ({{.TypeName}}) {{$u.Marker}}() {}
{{- $vt := .}}
{{- if .NeedsApplyDefaults}}

// ApplyDefaults fills zero-valued fields with their schema-declared defaults.
func ({{$vt.Receiver}} *{{$vt.TypeName}}) ApplyDefaults() {
{{- range $vt.DefaultRecurse}}
{{- if eq (printf "%s" .Kind) "value"}}
	{{$vt.Receiver}}.{{.GoName}}.ApplyDefaults()
{{- else if eq (printf "%s" .Kind) "pointer"}}
	if {{$vt.Receiver}}.{{.GoName}} != nil {
		{{$vt.Receiver}}.{{.GoName}}.ApplyDefaults()
	}
{{- else if eq (printf "%s" .Kind) "slice"}}
	for i := range {{$vt.Receiver}}.{{.GoName}} {
		{{$vt.Receiver}}.{{.GoName}}[i].ApplyDefaults()
	}
{{- else if eq (printf "%s" .Kind) "map"}}
	for key, value := range {{$vt.Receiver}}.{{.GoName}} {
		value.ApplyDefaults()
		{{$vt.Receiver}}.{{.GoName}}[key] = value
	}
{{- end}}
{{- end}}
}
{{- end}}
{{- if .NeedsValidate}}

// Validate returns an error wrapping validate.ErrValidation if any field violates its
// schema constraints.
func ({{$vt.Receiver}} {{$vt.TypeName}}) Validate() error {
	v := validate.New("{{$vt.TypeName}}")
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

// MarshalJSON encodes the active variant with its "{{.DiscJSONName}}" tag injected.
func (u {{.Name}}) MarshalJSON() ([]byte, error) {
	if u.Variant == nil {
		return []byte("null"), nil
	}
	var t {{.DiscType}}
	switch u.Variant.(type) {
{{- range .Variants}}
	case {{.TypeName}}:
		t = {{.ConstName}}
{{- end}}
	default:
		return nil, errors.Newf("{{.Name}}: nil or unknown variant %T", u.Variant)
	}
	raw, err := json.Marshal(u.Variant)
	if err != nil {
		return nil, err
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	tag, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	fields["{{.DiscJSONName}}"] = tag
	return json.Marshal(fields)
}

// UnmarshalJSON decodes the variant selected by the "{{.DiscJSONName}}" field.
func (u *{{.Name}}) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		u.Variant = nil
		return nil
	}
	var disc struct {
		Type {{.DiscType}} ` + "`" + `json:"{{.DiscJSONName}}"` + "`" + `
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	switch disc.Type {
{{- range .Variants}}
	case {{.ConstName}}:
		var v {{.TypeName}}
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Variant = v
{{- end}}
	default:
		return errors.Newf("{{.Name}}: unknown {{.DiscJSONName}} %q", disc.Type)
	}
	return nil
}
{{- if .NeedsApplyDefaults}}

// ApplyDefaults fills the active variant's zero-valued fields with their
// schema-declared defaults.
func (u *{{.Name}}) ApplyDefaults() {
	switch variant := u.Variant.(type) {
{{- range .Variants}}
{{- if .NeedsApplyDefaults}}
	case {{.TypeName}}:
		variant.ApplyDefaults()
		u.Variant = variant
{{- end}}
{{- end}}
	}
}
{{- end}}
{{- if .NeedsValidate}}

// Validate returns an error wrapping validate.ErrValidation if the active variant
// violates its schema constraints.
func (u {{.Name}}) Validate() error {
	switch variant := u.Variant.(type) {
{{- range .Variants}}
{{- if .NeedsValidate}}
	case {{.TypeName}}:
		return variant.Validate()
{{- end}}
{{- end}}
	}
	return nil
}
{{- end}}
{{- end}}
{{- end}}
`))

// WalkTypeRefs visits every type t directly references: all struct fields
// (omitted included), extends, type parameters, alias targets, distinct
// bases, and union variants.
func WalkTypeRefs(t resolution.Type, table *resolution.Table, visit func(resolution.Type)) {
	var walkRef func(ref resolution.TypeRef)
	walkRef = func(ref resolution.TypeRef) {
		for _, arg := range ref.TypeArgs {
			walkRef(arg)
		}
		if resolved, ok := ref.Resolve(table); ok {
			visit(resolved)
		}
	}
	switch form := t.Form.(type) {
	case resolution.StructForm:
		for _, ext := range form.Extends {
			walkRef(ext)
		}
		for _, tp := range form.TypeParams {
			if tp.Constraint != nil {
				walkRef(*tp.Constraint)
			}
			if tp.Default != nil {
				walkRef(*tp.Default)
			}
		}
		for _, f := range resolution.UnifiedFields(t, table) {
			walkRef(f.Type)
		}
	case resolution.EnumForm:
		for _, ext := range form.Extends {
			walkRef(ext)
		}
	case resolution.AliasForm:
		walkRef(form.Target)
	case resolution.DistinctForm:
		walkRef(form.Base)
	case resolution.UnionForm:
		for _, ext := range form.Extends {
			walkRef(ext)
		}
		for _, v := range form.Variants {
			walkRef(v.Type)
		}
	}
}
