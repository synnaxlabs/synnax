// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package query generates gorp Retrieve query wrappers from Oracle schema definitions.
package query

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/plugin"
	plugindomain "github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/matcher"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	goprimitives "github.com/synnaxlabs/oracle/plugin/go/primitives"
	"github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/pluralize"
	"github.com/synnaxlabs/x/set"
)

// Plugin generates gorp Retrieve query wrappers for structs annotated with @retrieve.
type Plugin struct{ Options Options }

// Options configures the go/query plugin.
type Options struct {
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{FileNamePattern: "retrieve.gen.go"}
}

// New creates a new go/query plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "go/query" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"go"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"go/types"} }

// Generate produces retrieve.gen.go files for structs and unions with a @retrieve
// annotation.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	// Group @retrieve structs by output path.
	type entry struct {
		path    string
		structs []resolution.Type
		// roots are the @retrieve references types, which several can share.
		roots []resolution.Type
	}
	entries := make(map[string]*entry)
	searched := make(set.Set[fieldDecl])
	referenced := make(set.Set[fieldDecl])
	rt := newResourceTypes(req.Resolutions)

	for _, typ := range slices.Concat(
		req.Resolutions.StructTypes(),
		req.Resolutions.UnionTypes(),
	) {
		if plugindomain.HasDomainFromType(typ, "search") {
			return nil, errors.Newf(
				"%s: @search goes on the fields to search, not on the type",
				typ.QualifiedName,
			)
		}
		if !plugindomain.HasDomainFromType(typ, "retrieve") {
			continue
		}
		outputPath := output.GetPath(typ, "go")
		if outputPath == "" {
			continue
		}
		e, ok := entries[outputPath]
		if !ok {
			e = &entry{path: outputPath}
			entries[outputPath] = e
		}
		if plugindomain.HasExprFromType(typ, "retrieve", "references") {
			w := newRefWalker(req.Resolutions)
			if _, err := w.walk(typ); err != nil {
				return nil, err
			}
			referenced.Add(w.declared.Slice()...)
			e.roots = append(e.roots, typ)
			continue
		}
		addDeclared(typ, req.Resolutions, "search", searched)
		addDeclared(typ, req.Resolutions, "ref", referenced)
		w := newRefWalker(req.Resolutions)
		if _, err := w.walk(typ); err != nil {
			return nil, err
		}
		referenced.Add(w.declared.Slice()...)
		e.structs = append(e.structs, typ)
	}

	for _, typ := range req.Resolutions.StructTypes() {
		form := typ.Form.(resolution.StructForm)
		inherited := resolution.InheritedFields(form.Extends, req.Resolutions)
		for _, f := range form.Fields {
			// An inherited @search is checked on the parent that declares it.
			if parent, ok := inherited[f.Name]; ok &&
				plugindomain.HasDomainFromField(parent, "search") {
				continue
			}
			if plugindomain.HasDomainFromField(f, "search") &&
				!searched.Contains(fieldDecl{typ: typ.QualifiedName, field: f.Name}) {
				return nil, errors.Newf(
					"%s: @search field %s needs a @retrieve type to read it",
					typ.QualifiedName,
					f.Name,
				)
			}
			if _, ok := inherited[f.Name]; !ok &&
				plugindomain.HasDomainFromField(f, "ref") &&
				!referenced.Contains(fieldDecl{typ: typ.QualifiedName, field: f.Name}) {
				return nil, errors.Newf(
					"%s: @ref field %s needs a @retrieve type to store it",
					typ.QualifiedName,
					f.Name,
				)
			}
		}
	}

	for _, e := range entries {
		// Every entry emits the same Retrieve, Filter, and MatchKeys names.
		if len(e.structs) > 1 {
			return nil, errors.Newf(
				"%s: only one @retrieve type is allowed per output, found %d",
				e.path,
				len(e.structs),
			)
		}
		content, err := generateRetrieveFile(
			e.path,
			e.structs,
			e.roots,
			req.Resolutions,
			rt,
			req.RepoRoot,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate %s", e.path)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", e.path, p.Options.FileNamePattern),
			Content: content,
		})
	}

	return resp, nil
}

// filterInfo holds extracted data about a field with a matcher.
type filterInfo struct {
	FieldName  string // oracle field name
	GoName     string // PascalCase
	Access     string // selector that reads the field from an entry
	Func       string // matcher function name
	GoType     string // resolved Go type (full type, including Array<...> if a slice)
	ElemGoType string // resolved Go type of the slice element (only set when IsSlice)
	IsScalar   bool   // the matcher takes one value
	IsBool     bool   // underlying primitive is bool
	IsSlice    bool   // field is a slice/array; emit a contains-style filter
	// IndexFieldName is the camelCase field name on the per-Retrieve indexes struct if
	// the same field also carries @index. When non-empty, the generated MatchX /
	// MatchXs routes through r.indexes.<IndexFieldName> instead of a bespoke scan
	// closure. Empty means no index on this field.
	IndexFieldName string
	// IndexTextual is true when that index holds compare.TextKey of the value, so the
	// filter narrows by the index and then keeps only exact matches.
	IndexTextual bool
	// Ref is the reference the field holds, if it carries @ref. The generated filter
	// then routes through the reference's index.
	Ref *refInfo
}

// indexInfo holds extracted data about an @index-annotated field. Emitted as a
// constructor function (newXIndex) and a field on the per-Retrieve indexes struct that
// the Service constructs once and threads onto every Retrieve.
type indexInfo struct {
	FieldName   string // oracle field name (e.g. "created_at")
	GoName      string // PascalCase field name (e.g. "CreatedAt")
	Access      string // selector that reads the field from an entry
	GoType      string // Go type of the indexed value (e.g. "telem.TimeStamp")
	ValueGoType string // Go type of the field's value, which GoType keys for Text
	StructField string // camelCase field on the indexes struct (e.g. "createdAt")
	Kind        string // "lookup" or "sorted"
	// Textual is true for a sorted index on a string, which holds compare.TextKey of
	// the value.
	Textual bool
}

// IsSorted reports whether the index is a sorted index. Used by the template
// to decide whether to emit an OrderBy helper.
func (i indexInfo) IsSorted() bool { return i.Kind == "sorted" }

// searchInfo holds extracted data about a @search-annotated field.
type searchInfo struct {
	Access string // selector that reads the field from an entry
	// IsString is false when the field's type is not the string primitive itself, so
	// the generated code converts it.
	IsString bool
}

// retrieveInfo holds extracted data about a @retrieve-annotated struct.
type retrieveInfo struct {
	// TableName is the type's @resource type, which names its table for references.
	TableName string
	// Name is the Go name of the retrieved type, with no type arguments.
	Name string
	// AnyArgs passes any for each type parameter of an indexes-only generic type, whose
	// table holds records of every type argument.
	AnyArgs string
	// Params declares the type parameters of a generic type, such as [D any], and Args
	// passes them, such as [D]. Both are empty for a type with none.
	Params, Args string
	// GoName is the Go type of a record in the table.
	GoName   string
	KeyType  string
	IsCustom bool // user defines the struct (@retrieve custom)
	// IndexesOnly is true under @retrieve indexes: the service writes its own Retrieve
	// and matchers, so only the indexes are generated.
	IndexesOnly bool
	// Filtered is true when a filter definition targets the table, whose generated
	// Apply skips missing keys.
	Filtered bool
	Filters  []filterInfo
	Indexes  []indexInfo
	Refs     []refInfo
	// NestedRefs are the references the type holds below its top-level fields.
	NestedRefs []nestedRef
	Search     []searchInfo
}

// HasSearch reports whether the retrieve declared any @search fields.
func (r retrieveInfo) HasSearch() bool { return len(r.Search) > 0 }

// HasIndexes reports whether the retrieve declared any @index, @ref, or @search fields.
// Used by the template to gate the indexes struct.
func (r retrieveInfo) HasIndexes() bool {
	return len(r.Indexes) > 0 || len(r.Refs) > 0 || len(r.NestedRefs) > 0 ||
		r.HasSearch()
}

// HasRefs reports whether the type holds any reference.
func (r retrieveInfo) HasRefs() bool { return len(r.Refs) > 0 || len(r.NestedRefs) > 0 }

// IDGoType returns the Go type of a resource ID, as the first @ref field that holds
// resource IDs resolves it.
func (r retrieveInfo) IDGoType() string {
	for _, ref := range r.Refs {
		if ref.IsID {
			return ref.ElemGoType
		}
	}
	return ""
}

// HasIDRefs reports whether any @ref field holds resource IDs.
func (r retrieveInfo) HasIDRefs() bool {
	return lo.SomeBy(r.Refs, func(ref refInfo) bool { return ref.IsID })
}

// needsSlices reports whether the generated code calls the slices package.
func (r retrieveInfo) needsSlices() bool {
	return lo.SomeBy(r.Filters, func(f filterInfo) bool {
		return f.Ref == nil && f.IsSlice
	}) || lo.SomeBy(r.Refs, func(ref refInfo) bool {
		return ref.IsSlice && ref.OnDelete == "Detach"
	}) || lo.SomeBy(r.NestedRefs, func(ref nestedRef) bool { return ref.usesSlices })
}

// needsSet reports whether the generated code calls the set package.
func (r retrieveInfo) needsSet() bool {
	return lo.SomeBy(r.Filters, func(f filterInfo) bool {
		return f.Ref == nil && !f.IsScalar && (f.IndexFieldName == "" || f.IndexTextual)
	})
}

// HasSortedIndex reports whether the retrieve declared any @index sorted fields. Used
// by the template to gate the Order closure type, the Retrieve.OrderBy method, and the
// per-field OrderByX free functions.
func (r retrieveInfo) HasSortedIndex() bool {
	for _, idx := range r.Indexes {
		if idx.IsSorted() {
			return true
		}
	}
	return false
}

func generateRetrieveFile(
	outputPath string,
	structs []resolution.Type,
	roots []resolution.Type,
	table *resolution.Table,
	rt resourceTypes,
	repoRoot string,
) ([]byte, error) {
	pkg := naming.DerivePackageName(outputPath)
	imps := imports.NewManager()

	namespace := slices.Concat(structs, roots)[0].Namespace

	ctx := &resolver.Context{
		Table:                         table,
		OutputPath:                    outputPath,
		Namespace:                     namespace,
		RepoRoot:                      repoRoot,
		DomainName:                    "go",
		SubstituteDefaultedTypeParams: true,
	}

	r := &resolver.Resolver{
		Formatter: types.GoFormatter(),
		ImportResolver: &types.GoImportResolver{
			RepoRoot:       repoRoot,
			CurrentPackage: pkg,
		},
		ImportAdder:     imps,
		PrimitiveMapper: goprimitives.Mapper(),
	}

	var infos []retrieveInfo
	for _, typ := range structs {
		info, err := extractRetrieveInfo(typ, table, rt, r, ctx)
		if err != nil {
			return nil, err
		}
		if info != nil {
			infos = append(infos, *info)
		}
	}

	var refRoots []referencesInfo
	for _, typ := range roots {
		info, err := extractReferencesInfo(typ, table, rt, r, ctx)
		if err != nil {
			return nil, err
		}
		refRoots = append(refRoots, info)
	}

	if len(infos) == 0 && len(refRoots) == 0 {
		return nil, nil
	}

	if lo.SomeBy(infos, func(i retrieveInfo) bool { return !i.IndexesOnly }) {
		imps.AddExternal("context")
	}
	imps.AddInternal(
		"gorp",
		gomod.ResolveImportPath("gorp", repoRoot, gomod.DefaultModulePrefix),
	)

	if lo.SomeBy(infos, func(i retrieveInfo) bool {
		return lo.SomeBy(i.Indexes, func(idx indexInfo) bool { return idx.Textual })
	}) {
		imps.AddInternal(
			"compare",
			gomod.ResolveImportPath(
				"x/go/compare",
				repoRoot,
				gomod.DefaultModulePrefix,
			),
		)
	}
	if lo.SomeBy(infos, retrieveInfo.needsSlices) ||
		lo.SomeBy(refRoots, referencesInfo.needsSlices) {
		imps.AddExternal("slices")
	}
	if lo.SomeBy(infos, retrieveInfo.needsSet) {
		imps.AddInternal(
			"set",
			gomod.ResolveImportPath("x/go/set", repoRoot, gomod.DefaultModulePrefix),
		)
	}

	data := &templateData{
		Package:    pkg,
		Retrieves:  infos,
		References: refRoots,
		Manager:    imps,
	}

	var buf bytes.Buffer
	if err := retrieveTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func extractRetrieveInfo(
	typ resolution.Type,
	table *resolution.Table,
	rt resourceTypes,
	r *resolver.Resolver,
	ctx *resolver.Context,
) (*retrieveInfo, error) {
	allFields, ok := resolution.RecordFields(typ, table)
	if !ok {
		return nil, nil
	}
	matchers, err := matcher.List(typ, table)
	if err != nil {
		return nil, err
	}
	byField := lo.KeyBy(
		matchers,
		func(m matcher.Matcher) string { return m.Field.Name },
	)

	// Resolve the key type. A struct-level `@retrieve key "TypeName"` override takes
	// precedence; this is useful when the entry type has a computed GorpKey that is not
	// stored as a field on the struct (e.g. channel.Channel). Otherwise, fall back to
	// the first field carrying `@key`.
	var keyType string
	if keyRef, ok := resolution.RetrieveKey(typ); ok {
		keyType = r.ResolveTypeRef(keyRef, ctx)
	} else {
		var keyField *resolution.Field
		for i := range allFields {
			if plugindomain.HasDomainFromField(allFields[i], "key") {
				keyField = &allFields[i]
				break
			}
		}
		if keyField == nil {
			return nil, nil
		}
		keyType = r.ResolveTypeRef(keyField.Type, ctx)
	}

	name := naming.GetGoName(typ)
	var anyArgs, typeArgs, typeParams string

	isCustom := plugindomain.HasExprFromType(typ, "retrieve", "custom")
	indexesOnly := plugindomain.HasExprFromType(typ, "retrieve", "indexes")
	if form, ok := typ.Form.(resolution.StructForm); ok {
		params := resolution.NonDefaultedTypeParams(form.TypeParams)
		if len(params) > 0 {
			if !indexesOnly {
				return nil, errors.Newf(
					"%s: a generic @retrieve type needs @retrieve indexes",
					typ.QualifiedName,
				)
			}
			// The table of a generic type holds records of every type argument.
			anys := make([]string, len(params))
			names := make([]string, len(params))
			decls := make([]string, len(params))
			for i, tp := range params {
				anys[i], names[i], decls[i] = "any", tp.Name, tp.Name+" any"
			}
			anyArgs = "[" + strings.Join(anys, ", ") + "]"
			typeArgs = "[" + strings.Join(names, ", ") + "]"
			typeParams = "[" + strings.Join(decls, ", ") + "]"
		}
	}

	// Collect @filter and @index fields. A field may carry both: in that
	// case, MatchX / MatchXs are generated to route through the index.
	// Resolve the Go type only for fields that actually contribute generated
	// code — otherwise ResolveTypeRef would add imports for types that are
	// never referenced in retrieve.gen.go.
	var (
		filters  []filterInfo
		indexes  []indexInfo
		refs     []refInfo
		search   []searchInfo
		ownTable = plugindomain.GetStringFromType(typ, resourceDomain, "type")
	)
	for _, field := range allFields {
		if indexesOnly && plugindomain.HasDomainFromField(field, "filter") {
			return nil, errors.Newf(
				"%s: @retrieve indexes generates no matchers, so field %s cannot "+
					"take @filter",
				typ.QualifiedName,
				field.Name,
			)
		}
		m, hasFilter := byField[field.Name]
		// The service writes the matchers for the filter's bare lines.
		hasFilter = hasFilter && !indexesOnly
		hasIndex := plugindomain.HasDomainFromField(field, "index")
		hasSearch := plugindomain.HasDomainFromField(field, "search")
		hasRef := plugindomain.HasDomainFromField(field, "ref")
		if !hasFilter && !hasIndex && !hasSearch && !hasRef {
			continue
		}
		access, ok := matcher.Access(typ, field, table)
		if !ok {
			return nil, errors.Newf(
				"%s: a variant drops field %s, so no index can read it",
				typ.QualifiedName,
				field.Name,
			)
		}
		if hasSearch {
			textType := field.Type
			if textType.IsTypeParam() && textType.TypeParam.HasDefault() {
				textType = *textType.TypeParam.Default
			}
			if field.Optional || key.ResolvePrimitive(textType, table) != "string" {
				return nil, errors.Newf(
					"%s: @search field %s must be a string that is not optional",
					typ.QualifiedName,
					field.Name,
				)
			}
			search = append(search, searchInfo{
				Access:   access,
				IsString: textType.Name == "string",
			})
		}
		if hasRef {
			if hasIndex || plugindomain.HasDomainFromField(field, "filter") {
				return nil, errors.Newf(
					"%s: @ref field %s has its own index and filter, so it cannot "+
						"carry @index or @filter",
					typ.QualifiedName,
					field.Name,
				)
			}
			if _, isUnion := typ.Form.(resolution.UnionForm); isUnion {
				return nil, errors.Newf(
					"%s: @ref is not supported on union fields (%s)",
					typ.QualifiedName,
					field.Name,
				)
			}
			info, err := extractRef(typ, field, ownTable, false, rt, table, r, ctx)
			if err != nil {
				return nil, err
			}
			refs = append(refs, info)
			filters = append(filters, filterInfo{
				FieldName:  field.Name,
				GoName:     info.GoName,
				Func:       matcher.RefFunc(info.GoName, info.IsSlice),
				GoType:     info.ElemGoType,
				ElemGoType: info.ElemGoType,
				IsSlice:    info.IsSlice,
				Ref:        &info,
			})
			continue
		}
		if !hasFilter && !hasIndex {
			continue
		}

		isSlice := field.Type.Name == "Array" && len(field.Type.TypeArgs) == 1

		goType := r.ResolveTypeRef(field.Type, ctx)
		var elemGoType string
		if isSlice {
			elemGoType = r.ResolveTypeRef(field.Type.TypeArgs[0], ctx)
		}
		goFieldName := naming.GetFieldName(field)

		// indexField names the index that backs the field's filter. A sub-field index
		// holds part of the value, so it cannot answer a match on the field.
		var (
			indexField   string
			indexTextual bool
		)
		if hasIndex {
			idx := indexInfo{
				FieldName:   field.Name,
				GoName:      goFieldName,
				Access:      access,
				GoType:      goType,
				ValueGoType: goType,
				Kind:        "lookup",
			}
			if plugindomain.HasExprFromField(field, "index", "sorted") {
				sorts, err := resolution.Sorts([]resolution.Field{field}, table)
				if err != nil {
					return nil, errors.Wrapf(err, "%s", typ.QualifiedName)
				}
				sort := sorts[0]
				idx.Kind, idx.Textual = "sorted", sort.Textual
				idx.FieldName = sort.Name
				if len(sort.Path) > 1 {
					leaf := sort.Leaf()
					idx.GoName += naming.GetFieldName(leaf)
					idx.Access += "." + naming.GetFieldName(leaf)
					idx.ValueGoType = r.ResolveTypeRef(leaf.Type, ctx)
				}
				idx.GoType = idx.ValueGoType
				if idx.Textual {
					idx.GoType = "string"
				}
			}
			// camelCase field name on the per-Retrieve indexes struct. Unexported
			// because the indexes struct itself is unexported.
			idx.StructField = naming.LowerFirst(idx.GoName)
			if idx.Access == access {
				indexField, indexTextual = idx.StructField, idx.Textual
			}
			indexes = append(indexes, idx)
		}

		if hasFilter {
			filters = append(filters, filterInfo{
				FieldName:      field.Name,
				GoName:         goFieldName,
				Access:         access,
				Func:           m.Func,
				GoType:         goType,
				ElemGoType:     elemGoType,
				IsScalar:       m.Kind.Scalar(),
				IsBool:         key.ResolvePrimitive(field.Type, table) == "bool",
				IsSlice:        m.Kind == matcher.Element,
				IndexFieldName: indexField,
				IndexTextual:   indexTextual,
			})
		}
	}

	walked, err := newRefWalker(table).walk(typ)
	if err != nil {
		return nil, err
	}
	// The loop above declares the references of a struct's own fields. A union's own
	// fields hold none, so every reference it holds is in a variant.
	if _, ok := typ.Form.(resolution.StructForm); ok {
		walked = lo.Filter(walked, func(ref *walkedRef, _ int) bool {
			return strings.Contains(ref.path, ".")
		})
	}
	nested, err := renderNestedRefs(
		typ, walked, ownTable, keyType, name+anyArgs, rt, table, r, ctx,
	)
	if err != nil {
		return nil, err
	}

	if len(search) > 0 {
		for _, idx := range indexes {
			if idx.StructField == "search" {
				return nil, errors.Newf(
					"%s: @index field %s collides with the search index",
					typ.QualifiedName,
					idx.FieldName,
				)
			}
		}
		// A union's variant references have no dotted path, so one can be named search.
		nestedRefs := lo.Map(nested, func(n nestedRef, _ int) refInfo {
			return n.refInfo
		})
		for _, ref := range slices.Concat(refs, nestedRefs) {
			if ref.StructField == "search" {
				return nil, errors.Newf(
					"%s: @ref field %s collides with the search index",
					typ.QualifiedName,
					ref.FieldName,
				)
			}
		}
	}

	info := &retrieveInfo{
		TableName:   ownTable,
		Name:        name,
		AnyArgs:     anyArgs,
		Params:      typeParams,
		Args:        typeArgs,
		GoName:      name + anyArgs,
		KeyType:     keyType,
		IsCustom:    isCustom,
		IndexesOnly: indexesOnly,
		Filtered:    isFilterTarget(typ, table),
		Filters:     filters,
		Indexes:     indexes,
		Refs:        refs,
		NestedRefs:  nested,
		Search:      search,
	}
	if indexesOnly && !info.HasIndexes() {
		return nil, errors.Newf(
			"%s: @retrieve indexes needs a field with @index, @ref, or @search",
			typ.QualifiedName,
		)
	}
	return info, nil
}

// isFilterTarget reports whether a filter definition targets typ.
func isFilterTarget(typ resolution.Type, table *resolution.Table) bool {
	return lo.SomeBy(table.StructTypes(), func(t resolution.Type) bool {
		if t.Filter == nil {
			return false
		}
		target, _, ok := resolution.Dealias(t.Filter.Target, table)
		return ok && target.QualifiedName == typ.QualifiedName
	})
}

// fieldDecl names a field of a struct.
type fieldDecl struct{ typ, field string }

// addDeclared adds to declared each field annotated with domainName that the table of
// retrieved reads: a field of retrieved, or of a type it extends, that keeps the
// annotation in the fields of retrieved.
func addDeclared(
	retrieved resolution.Type,
	table *resolution.Table,
	domainName string,
	declared set.Set[fieldDecl],
) {
	read := make(set.Set[string])
	fields, _ := resolution.RecordFields(retrieved, table)
	for _, f := range fields {
		if plugindomain.HasDomainFromField(f, domainName) {
			read.Add(f.Name)
		}
	}
	visited := make(set.Set[string])
	var visit func(typ resolution.Type)
	visit = func(typ resolution.Type) {
		if visited.Contains(typ.QualifiedName) {
			return
		}
		visited.Add(typ.QualifiedName)
		var extends []resolution.TypeRef
		switch form := typ.Form.(type) {
		case resolution.StructForm:
			for _, f := range form.Fields {
				if read.Contains(f.Name) {
					declared.Add(fieldDecl{typ: typ.QualifiedName, field: f.Name})
				}
			}
			extends = form.Extends
		case resolution.UnionForm:
			extends = form.Extends
		}
		for _, ext := range extends {
			if parent, _, ok := resolution.Dealias(ext, table); ok {
				visit(parent)
			}
		}
	}
	visit(retrieved)
}

type templateData struct {
	*imports.Manager
	Package    string
	Retrieves  []retrieveInfo
	References []referencesInfo
}

var templateFuncs = template.FuncMap{
	"formatDoc": doc.FormatGo,
	"toPascal":  naming.ToPascalCase,
	"toLower":   strings.ToLower,
	"pluralize": pluralize.String,
}

var retrieveTemplate = template.Must(
	template.New("go-retrieve").
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
{{range $ret := .Retrieves}}
{{- if $ret.TableName}}
// tableName names the {{$ret.Name}} table for references. Pass it as
// gorp.TableConfig.Name.
const tableName = "{{$ret.TableName}}"
{{end}}
{{- if not (or $ret.IsCustom $ret.IndexesOnly)}}
// Retrieve is used to retrieve {{$ret.GoName}} records from the database using a
// builder pattern for constructing queries.
type Retrieve struct {
	gorp gorp.Retrieve[{{$ret.KeyType}}, {{$ret.GoName}}]
{{- if $ret.HasIndexes}}
	indexes    indexes
{{- end}}
}
{{- end}}
{{- if $ret.HasIndexes}}

// indexes bundles the per-Service secondary indexes registered on the
// {{$ret.GoName}} table. Each index is constructed via newIndexes and threaded
// onto the Retrieve so filter functions can resolve them off r.indexes
// instead of relying on package-level state.
type indexes struct {
{{- range $idx := $ret.Indexes}}
	{{$idx.StructField}} *gorp.{{if $idx.IsSorted}}SortedIndex{{else}}LookupIndex{{end}}[{{$ret.KeyType}}, {{$ret.GoName}}, {{$idx.GoType}}]
{{- end}}
{{- range $ref := $ret.Refs}}
	{{$ref.StructField}} *{{$ref.GorpType $ret.KeyType $ret.GoName}}
{{- end}}
{{- range $ref := $ret.NestedRefs}}
	{{$ref.StructField}} *{{$ref.GorpType $ret.KeyType $ret.GoName}}
{{- end}}
{{- if $ret.HasSearch}}
	search *gorp.SearchIndex[{{$ret.KeyType}}, {{$ret.GoName}}]
{{- end}}
}

// newIndexes constructs a fresh indexes value, allocating one instance of each
// index. Call once per Service in OpenService and store the result on the
// Service struct.
func newIndexes() indexes {
	return indexes{
{{- range $idx := $ret.Indexes}}
		{{$idx.StructField}}: gorp.{{if $idx.IsSorted}}NewSortedIndex{{else}}NewLookupIndex{{end}}[{{$ret.KeyType}}, {{$ret.GoName}}, {{$idx.GoType}}](
			"{{$idx.FieldName}}",
{{- if $idx.Textual}}
			func(e *{{$ret.GoName}}) string { return compare.TextKey(string(e.{{$idx.Access}})) },
{{- else}}
			func(e *{{$ret.GoName}}) {{$idx.GoType}} { return e.{{$idx.Access}} },
{{- end}}
		),
{{- end}}
{{- range $ref := $ret.Refs}}
		{{$ref.StructField}}: {{$ref.Constructor $ret.KeyType $ret.GoName}}{
			Name:     "{{$ref.FieldName}}",
			{{$ref.TablesField}}
			OnDelete: gorp.{{$ref.OnDelete}},
			Kind:     gorp.{{$ref.Kind}},
{{- if $ref.Unique}}
			Unique:   true,
{{- end}}
{{- if $ref.Acyclic}}
			Acyclic:  true,
{{- end}}
			Targets: func(e *{{$ret.GoName}}) []{{$ref.TargetType}} {
{{- if and $ref.IsSlice (not $ref.IsID)}}
				return e.{{$ref.GoName}}
{{- else if $ref.IsSlice}}
				targets := make([]gorp.Target, len(e.{{$ref.GoName}}))
				for i, v := range e.{{$ref.GoName}} {
					targets[i] = {{$ref.TargetOf "v"}}
				}
				return targets
{{- else if $ref.IsOptional}}
				if e.{{$ref.GoName}} == nil {
					return nil
				}
				return []{{$ref.TargetType}}{ {{- $ref.TargetOf (printf "*e.%s" $ref.GoName) -}} }
{{- else if $ref.Zero}}
				if e.{{$ref.GoName}} == {{$ref.ZeroOperand}} {
					return nil
				}
				return []{{$ref.TargetType}}{ {{- $ref.TargetOf (printf "e.%s" $ref.GoName) -}} }
{{- else}}
				return []{{$ref.TargetType}}{ {{- $ref.TargetOf (printf "e.%s" $ref.GoName) -}} }
{{- end}}
			},
{{- if eq $ref.OnDelete "Detach"}}
{{- if $ref.IsSlice}}
			Detach: func(e *{{$ret.GoName}}, t {{$ref.TargetType}}) {
				e.{{$ref.GoName}} = slices.DeleteFunc(e.{{$ref.GoName}}, func(v {{$ref.ElemGoType}}) bool {
					return {{$ref.TargetOf "v"}} == t
				})
			},
{{- else if $ref.Zero}}
			Detach: func(e *{{$ret.GoName}}, _ {{$ref.TargetType}}) { e.{{$ref.GoName}} = {{$ref.Zero}} },
{{- else}}
			Detach: func(e *{{$ret.GoName}}, _ {{$ref.TargetType}}) { e.{{$ref.GoName}} = nil },
{{- end}}
{{- end}}
		}),
{{- end}}
{{- range $ref := $ret.NestedRefs}}
		{{$ref.StructField}}: {{template "nestedReference" $ref}},
{{- end}}
{{- if $ret.HasSearch}}
		search: gorp.NewSearchIndex[{{$ret.KeyType}}, {{$ret.GoName}}](
			"search",
			func(e *{{$ret.GoName}}) []string {
				return []string{
{{- range $i, $f := $ret.Search}}{{if $i}}, {{end}}{{if $f.IsString}}e.{{$f.Access}}{{else}}string(e.{{$f.Access}}){{end}}{{end -}}
				}
			},
		),
{{- end}}
	}
}

// all returns the indexes packaged as a heterogeneous slice for registration
// via gorp.TableConfig.Indexes when opening the underlying table.
func (i indexes) all() []gorp.Index[{{$ret.KeyType}}, {{$ret.GoName}}] {
	return []gorp.Index[{{$ret.KeyType}}, {{$ret.GoName}}]{
{{- range $idx := $ret.Indexes}}
		i.{{$idx.StructField}},
{{- end}}
{{- if $ret.HasSearch}}
		i.search,
{{- end}}
	}
}
{{- if $ret.HasRefs}}

// references returns the references packaged for registration via
// gorp.TableConfig.References when opening the underlying table.
func (i indexes) references() []gorp.Reference[{{$ret.KeyType}}, {{$ret.GoName}}] {
	return []gorp.Reference[{{$ret.KeyType}}, {{$ret.GoName}}]{
{{- range $ref := $ret.Refs}}
		i.{{$ref.StructField}},
{{- end}}
{{- range $ref := $ret.NestedRefs}}
		i.{{$ref.StructField}},
{{- end}}
	}
}
{{- end}}
{{- if $ret.HasIDRefs}}

// resourceTarget returns the gorp.Target of the resource that id identifies.
func resourceTarget(id {{$ret.IDGoType}}) gorp.Target {
	return gorp.Target{Table: string(id.Type), Key: id.Key}
}
{{- end}}
{{- if $ret.HasSearch}}

// MatchSearch matches the {{$ret.Name | toLower | pluralize}} whose text matches term.
// The closest match comes first.
func MatchSearch(term string) Filter{{$ret.AnyArgs}} {
	return func(r Retrieve{{$ret.AnyArgs}}) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
		return r.indexes.search.Filter(term)
	}
}
{{- end}}
{{- if $ret.HasSortedIndex}}

// Order is a sort that OrderBy applies. An OrderBy function builds one.
type Order{{$ret.Params}} func(r Retrieve{{$ret.Args}}) gorp.OrderQuery[{{$ret.KeyType}}, {{$ret.Name}}{{$ret.Args}}]

// OrderBy walks the {{$ret.Name | toLower | pluralize}} in the order o defines.
func (r Retrieve{{$ret.Args}}) OrderBy(o Order{{$ret.Args}}) Retrieve{{$ret.Args}} {
	r.gorp = r.gorp.OrderBy(o(r))
	return r
}
{{- range $idx := $ret.Indexes}}
{{- if $idx.IsSorted}}

{{formatDoc (print "OrderBy" $idx.GoName) (print "returns an Order by " $idx.FieldName (or (and $idx.Textual " in lower case") "") ", then by key.")}}
func OrderBy{{$idx.GoName}}(dir gorp.Direction) Order{{$ret.AnyArgs}} {
	return func(r Retrieve{{$ret.AnyArgs}}) gorp.OrderQuery[{{$ret.KeyType}}, {{$ret.GoName}}] {
		return r.indexes.{{$idx.StructField}}.Ordered(dir)
	}
}

{{formatDoc (print "OrderBy" $idx.GoName "After") (print "returns an Order like OrderBy" $idx.GoName " that starts strictly past the position of value and key.")}}
func OrderBy{{$idx.GoName}}After(
	dir gorp.Direction,
	value {{$idx.ValueGoType}},
	key {{$ret.KeyType}},
) Order{{$ret.AnyArgs}} {
	return func(r Retrieve{{$ret.AnyArgs}}) gorp.OrderQuery[{{$ret.KeyType}}, {{$ret.GoName}}] {
{{- if $idx.Textual}}
		return r.indexes.{{$idx.StructField}}.Ordered(dir).After(
			compare.TextKey(string(value)),
			key,
		)
{{- else}}
		return r.indexes.{{$idx.StructField}}.Ordered(dir).After(value, key)
{{- end}}
	}
}
{{- end}}
{{- end}}
{{- end}}
{{- end}}
{{- if not $ret.IndexesOnly}}

// Filter is a per-service filter that is bound to the Retrieve when passed to
// Where. Pure filters ignore the Retrieve argument; service-bound filters read
// from it (e.g. r.indexes, r.label, r.hostProvider) to evaluate. Use Match to
// construct one from a closure.
//
// Filter is a type alias for gorp.BoundFilter[Retrieve, K, E] so the
// composition helpers (Match / And / Or / Not) can be one-line wrappers
// around their gorp.*Bound counterparts instead of re-emitting closure
// plumbing per service.
type Filter = gorp.BoundFilter[Retrieve, {{$ret.KeyType}}, {{$ret.GoName}}]

// Match wraps a closure that needs the Retrieve into a Filter. The Retrieve
// value is supplied by Retrieve.Where at evaluation time.
func Match(f func(ctx gorp.Context, r Retrieve, e *{{$ret.GoName}}) (bool, error)) Filter {
	return gorp.MatchBound[Retrieve, {{$ret.KeyType}}, {{$ret.GoName}}](f)
}

// And returns a filter that matches when all provided filters match.
func And(fs ...Filter) Filter {
	return gorp.AndBound[Retrieve, {{$ret.KeyType}}, {{$ret.GoName}}](fs...)
}

// Or returns a filter that matches when any provided filter matches.
func Or(fs ...Filter) Filter {
	return gorp.OrBound[Retrieve, {{$ret.KeyType}}, {{$ret.GoName}}](fs...)
}

// Not returns a filter that inverts the provided filter.
func Not(f Filter) Filter {
	return gorp.NotBound[Retrieve, {{$ret.KeyType}}, {{$ret.GoName}}](f)
}
{{if $ret.HasSearch}}
// Search filters the {{$ret.GoName | toLower | pluralize}} to those whose text matches
// term, closest match first. An empty term adds no filter.
func (r Retrieve) Search(term string) Retrieve {
	if term == "" {
		return r
	}
	return r.Where(MatchSearch(term))
}
{{end}}
// MatchKeys returns a filter that restricts results to
// {{$ret.GoName | toLower | pluralize}} whose key matches any of the provided
// values. Composing MatchKeys at the top level of a Where clause (i.e.
// r.Where(MatchKeys(...))) dispatches Exec to the multi-get fast path; composing
// inside Or / Not falls back to a full scan.
func MatchKeys(keys ...{{$ret.KeyType}}) Filter {
	return func(Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
		return gorp.MatchKeys[{{$ret.KeyType}}, {{$ret.GoName}}](keys...)
	}
}
{{range .Filters}}
{{- if .Ref}}
// {{.Func}} returns a filter for {{$ret.GoName | toLower | pluralize}}
// whose {{.GoName}} {{if .IsSlice}}contains{{else}}matches{{end}} any of the provided
// values.
func {{.Func}}(vals ...{{.ElemGoType}}) Filter {
	return func(r Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
{{- if .Ref.IsID}}
		targets := make([]gorp.Target, len(vals))
		for i, v := range vals {
			targets[i] = {{.Ref.TargetOf "v"}}
		}
		return r.indexes.{{.Ref.StructField}}.Filter(targets...)
{{- else}}
		return r.indexes.{{.Ref.StructField}}.Filter(vals...)
{{- end}}
	}
}
{{- if .Ref.HoldsNone}}

// MatchHas{{.GoName}} returns a filter for {{$ret.GoName | toLower | pluralize}} whose
// {{.GoName}} holds a target when has is true, and holds none when has is false.
func MatchHas{{.GoName}}(has bool) Filter {
	return func(r Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
		return r.indexes.{{.Ref.StructField}}.FilterHolding(has)
	}
}
{{- end}}
{{- if and (eq .Ref.OnDelete "Detach") (not .IsSlice)}}

// set{{.GoName}} returns an update that sets the {{.GoName}} of the
// {{$ret.GoName | toLower | pluralize}} with keys to v.
func set{{.GoName}}(
	table *gorp.Table[{{$ret.KeyType}}, {{$ret.GoName}}],
	v {{if .Ref.IsOptional}}*{{end}}{{.ElemGoType}},
	keys ...{{$ret.KeyType}},
) gorp.Update[{{$ret.KeyType}}, {{$ret.GoName}}] {
	return table.NewUpdate().
		Where(gorp.MatchKeys[{{$ret.KeyType}}, {{$ret.GoName}}](keys...)).
		Change(func(_ gorp.Context, e {{$ret.GoName}}) {{$ret.GoName}} {
			e.{{.GoName}} = v
			return e
		})
}
{{- end}}
{{else if .IsBool}}
// {{.Func}} returns a filter for {{$ret.GoName | toLower | pluralize}} by their
// {{.GoName}} field.
func {{.Func}}(v {{.GoType}}) Filter {
	return func({{if .IndexFieldName}}r {{end}}Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
{{- if .IndexFieldName}}
		return r.indexes.{{.IndexFieldName}}.Filter(v)
{{- else}}
		return gorp.Match(func(_ gorp.Context, e *{{$ret.GoName}}) (bool, error) {
			return e.{{.Access}} == v, nil
		})
{{- end}}
	}
}
{{else if .IsSlice}}
// {{.Func}} returns a filter for {{$ret.GoName | toLower | pluralize}} whose
// {{.GoName}} contains the provided value.
func {{.Func}}(v {{.ElemGoType}}) Filter {
	return func(Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
		return gorp.Match(func(_ gorp.Context, e *{{$ret.GoName}}) (bool, error) {
			return slices.Contains(e.{{.Access}}, v), nil
		})
	}
}
{{else if .IsScalar}}
// {{.Func}} returns a filter for {{$ret.GoName | toLower | pluralize}} whose
// {{.GoName}} matches the provided value.
func {{.Func}}(v {{.GoType}}) Filter {
	return func({{if .IndexFieldName}}r {{end}}Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
{{- if .IndexTextual}}
		return gorp.And(
			r.indexes.{{.IndexFieldName}}.Filter(compare.TextKey(string(v))),
			gorp.Match(func(_ gorp.Context, e *{{$ret.GoName}}) (bool, error) {
				return e.{{.Access}} == v, nil
			}),
		)
{{- else if .IndexFieldName}}
		return r.indexes.{{.IndexFieldName}}.Filter(v)
{{- else}}
		return gorp.Match(func(_ gorp.Context, e *{{$ret.GoName}}) (bool, error) {
			return e.{{.Access}} == v, nil
		})
{{- end}}
	}
}
{{else}}
// {{.Func}} returns a filter for
// {{$ret.GoName | toLower | pluralize}} whose {{.GoName}} matches any of the
// provided values.
func {{.Func}}(vals ...{{.GoType}}) Filter {
	return func({{if .IndexFieldName}}r {{end}}Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
{{- if .IndexTextual}}
		keys := make([]string, len(vals))
		for i, v := range vals {
			keys[i] = compare.TextKey(string(v))
		}
		want := set.New(vals...)
		return gorp.And(
			r.indexes.{{.IndexFieldName}}.Filter(keys...),
			gorp.Match(func(_ gorp.Context, e *{{$ret.GoName}}) (bool, error) {
				return want.Contains(e.{{.Access}}), nil
			}),
		)
{{- else if .IndexFieldName}}
		return r.indexes.{{.IndexFieldName}}.Filter(vals...)
{{- else}}
		want := set.New(vals...)
		return gorp.Match(func(_ gorp.Context, e *{{$ret.GoName}}) (bool, error) {
			return want.Contains(e.{{.Access}}), nil
		})
{{- end}}
	}
}
{{end}}
{{- end}}
// Where applies the provided filter to the query, binding it to the Retrieve
// so service-bound filters can read from r.indexes, r.label, r.hostProvider,
// etc. To compose multiple filters, chain Where calls or pass a combined
// filter via And / Or.
func (r Retrieve) Where(filter Filter) Retrieve {
	r.gorp = r.gorp.Where(filter(r))
	return r
}

// Entry binds the provided {{$ret.GoName | toLower}} as the result container for the
// query. If multiple {{$ret.GoName | toLower | pluralize}} match, the first one is
// used.
func (r Retrieve) Entry(e *{{$ret.GoName}}) Retrieve {
	r.gorp = r.gorp.Entry(e)
	return r
}

// Entries binds the provided slice of {{$ret.GoName | toLower | pluralize}} as the
// result container for the query.
func (r Retrieve) Entries(es *[]{{$ret.GoName}}) Retrieve {
	r.gorp = r.gorp.Entries(es)
	return r
}

// Limit sets the maximum number of {{$ret.GoName | toLower | pluralize}} to return.
func (r Retrieve) Limit(limit int) Retrieve { r.gorp = r.gorp.Limit(limit); return r }

// After starts the query strictly past key in key order. It panics on a query with an
// OrderBy, since an Order carries its own cursor.
func (r Retrieve) After(key {{$ret.KeyType}}) Retrieve {
	r.gorp = r.gorp.After(key)
	return r
}
{{- if $ret.Filtered}}

// SkipMissingKeys makes Exec treat a key with no {{$ret.GoName | toLower}} as a
// non-match instead of returning query.ErrNotFound.
func (r Retrieve) SkipMissingKeys() Retrieve {
	r.gorp = r.gorp.SkipMissingKeys()
	return r
}
{{- end}}
// Exec executes the query against snap. It panics when snap is nil.
func (r Retrieve) Exec(ctx context.Context, snap gorp.Snapshot) error {
	return r.gorp.Exec(ctx, snap)
}

// Count returns the number of {{$ret.GoName | toLower | pluralize}} in snap that match
// the query. It panics when snap is nil.
func (r Retrieve) Count(ctx context.Context, snap gorp.Snapshot) (int, error) {
	return r.gorp.Count(ctx, snap)
}

// Exists checks whether any {{$ret.GoName | toLower | pluralize}} in snap match the
// query. It panics when snap is nil.
func (r Retrieve) Exists(ctx context.Context, snap gorp.Snapshot) (bool, error) {
	return r.gorp.Exists(ctx, snap)
}
{{- end}}
{{end}}
{{- range $root := .References}}
// {{$root.FuncName}} returns the references of {{$root.GoName}}.
func {{$root.FuncName}}() []gorp.Reference[{{$root.KeyType}}, {{$root.GoName}}] {
	return []gorp.Reference[{{$root.KeyType}}, {{$root.GoName}}]{
{{- range $ref := $root.Refs}}
		{{template "nestedReference" $ref}},
{{- end}}
	}
}
{{end}}
{{- define "nestedReference"}}{{.Constructor .KeyType .EntryType}}{
			Name:     "{{.FieldName}}",
			{{.TablesField}}
			OnDelete: gorp.{{.OnDelete}},
			Kind:     gorp.{{.Kind}},
{{- if .Unique}}
			Unique:   true,
{{- end}}
{{- if .Acyclic}}
			Acyclic:  true,
{{- end}}
			Targets: func(e *{{.EntryType}}) []{{.TargetType}} {
				{{.TargetsBody}}
			},
{{- if .DetachBody}}
			Detach: func(e *{{.EntryType}}, t {{.TargetType}}) {
				{{.DetachBody -}}
			},
{{- end}}
		}){{end}}`),
)
