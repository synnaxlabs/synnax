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
	}
	entries := make(map[string]*entry)
	searched := make(set.Set[searchDecl])

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
		addSearched(typ, req.Resolutions, searched)
		e, ok := entries[outputPath]
		if !ok {
			e = &entry{path: outputPath}
			entries[outputPath] = e
		}
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
				!searched.Contains(searchDecl{typ: typ.QualifiedName, field: f.Name}) {
				return nil, errors.Newf(
					"%s: @search field %s needs a @retrieve type to read it",
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
			req.Resolutions,
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
}

// indexInfo holds extracted data about an @index-annotated field. Emitted as a
// constructor function (newXIndex) and a field on the per-Retrieve indexes struct that
// the Service constructs once and threads onto every Retrieve.
type indexInfo struct {
	FieldName   string // oracle field name (e.g. "created_at")
	GoName      string // PascalCase field name (e.g. "CreatedAt")
	Access      string // selector that reads the field from an entry
	GoType      string // resolved Go type (e.g. "telem.TimeStamp")
	StructField string // camelCase field on the indexes struct (e.g. "createdAt")
	Kind        string // "lookup" or "sorted"
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
	// Name is the Go name of the retrieved type, with no type arguments.
	Name string
	// AnyArgs passes any for each type parameter of an indexes-only generic type, whose
	// table holds records of every type argument.
	AnyArgs string
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
	Search   []searchInfo
}

// HasSearch reports whether the retrieve declared any @search fields.
func (r retrieveInfo) HasSearch() bool { return len(r.Search) > 0 }

// HasIndexes reports whether the retrieve declared any @index or @search fields. Used
// by the template to gate the indexes struct.
func (r retrieveInfo) HasIndexes() bool { return len(r.Indexes) > 0 || r.HasSearch() }

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
	table *resolution.Table,
	repoRoot string,
) ([]byte, error) {
	pkg := naming.DerivePackageName(outputPath)
	imps := imports.NewManager()

	namespace := structs[0].Namespace

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
		info, err := extractRetrieveInfo(typ, table, r, ctx)
		if err != nil {
			return nil, err
		}
		if info != nil {
			infos = append(infos, *info)
		}
	}

	if len(infos) == 0 {
		return nil, nil
	}

	if lo.SomeBy(infos, func(i retrieveInfo) bool { return !i.IndexesOnly }) {
		imps.AddExternal("context")
	}
	imps.AddInternal(
		"gorp",
		gomod.ResolveImportPath("gorp", repoRoot, gomod.DefaultModulePrefix),
	)

	// Index-routed filters use the index's .Filter(...) method, so only slice and
	// non-indexed variadic filters need slices.Contains.
	needsSlices := lo.SomeBy(infos, func(i retrieveInfo) bool {
		return lo.SomeBy(i.Filters, func(f filterInfo) bool {
			return f.IsSlice || (!f.IsScalar && f.IndexFieldName == "")
		})
	})
	if needsSlices {
		imps.AddExternal("slices")
	}

	data := &templateData{
		Package:   pkg,
		Retrieves: infos,
		Manager:   imps,
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
	var anyArgs string

	isCustom := plugindomain.HasExprFromType(typ, "retrieve", "custom")
	indexesOnly := plugindomain.HasExprFromType(typ, "retrieve", "indexes")
	if form, ok := typ.Form.(resolution.StructForm); ok && indexesOnly {
		// The table of a generic type holds records of every type argument.
		var args []string
		for _, param := range form.TypeParams {
			if !param.HasDefault() {
				args = append(args, "any")
			}
		}
		if len(args) > 0 {
			anyArgs = "[" + strings.Join(args, ", ") + "]"
		}
	}

	// Collect @filter and @index fields. A field may carry both: in that
	// case, MatchX / MatchXs are generated to route through the index.
	// Resolve the Go type only for fields that actually contribute generated
	// code — otherwise ResolveTypeRef would add imports for types that are
	// never referenced in retrieve.gen.go.
	var (
		filters []filterInfo
		indexes []indexInfo
		search  []searchInfo
	)
	for _, field := range allFields {
		if indexesOnly && (plugindomain.HasDomainFromField(field, "filter") ||
			plugindomain.HasExprFromField(field, "index", "sorted")) {
			return nil, errors.Newf(
				"%s: @retrieve indexes generates no matchers or orderings, so field "+
					"%s cannot take @filter or @index sorted",
				typ.QualifiedName,
				field.Name,
			)
		}
		m, hasFilter := byField[field.Name]
		// The service writes the matchers for the filter's bare lines.
		hasFilter = hasFilter && !indexesOnly
		hasIndex := plugindomain.HasDomainFromField(field, "index")
		hasSearch := plugindomain.HasDomainFromField(field, "search")
		if !hasFilter && !hasIndex && !hasSearch {
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

		var structField string
		if hasIndex {
			kind := "lookup"
			if plugindomain.HasExprFromField(field, "index", "sorted") {
				kind = "sorted"
			}
			// camelCase field name on the per-Retrieve indexes struct. Unexported
			// because the indexes struct itself is unexported.
			structField = naming.LowerFirst(goFieldName)
			indexes = append(indexes, indexInfo{
				FieldName:   field.Name,
				GoName:      goFieldName,
				Access:      access,
				GoType:      goType,
				StructField: structField,
				Kind:        kind,
			})
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
				IndexFieldName: structField,
			})
		}
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
	}

	return &retrieveInfo{
		Name:        name,
		AnyArgs:     anyArgs,
		GoName:      name + anyArgs,
		KeyType:     keyType,
		IsCustom:    isCustom,
		IndexesOnly: indexesOnly,
		Filtered:    isFilterTarget(typ, table),
		Filters:     filters,
		Indexes:     indexes,
		Search:      search,
	}, nil
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

// searchDecl names a field that declares @search on a struct.
type searchDecl struct{ typ, field string }

// addSearched adds to searched each @search declaration that the search index of
// retrieved reads: a field of retrieved, or of a type it extends, that keeps @search in
// the fields of retrieved.
func addSearched(
	retrieved resolution.Type,
	table *resolution.Table,
	searched set.Set[searchDecl],
) {
	read := make(set.Set[string])
	fields, _ := resolution.RecordFields(retrieved, table)
	for _, f := range fields {
		if plugindomain.HasDomainFromField(f, "search") {
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
					searched.Add(searchDecl{typ: typ.QualifiedName, field: f.Name})
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
	Package   string
	Retrieves []retrieveInfo
}

var templateFuncs = template.FuncMap{
	"toPascal":  naming.ToPascalCase,
	"toLower":   strings.ToLower,
	"pluralize": matcher.Plural,
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
{{- if not (or $ret.IsCustom $ret.IndexesOnly)}}
// Retrieve is used to retrieve {{$ret.GoName}} records from the database using a
// builder pattern for constructing queries.
type Retrieve struct {
	baseTX gorp.Tx
	gorp   gorp.Retrieve[{{$ret.KeyType}}, {{$ret.GoName}}]
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
			func(e *{{$ret.GoName}}) {{$idx.GoType}} { return e.{{$idx.Access}} },
		),
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

// Order is a per-service sort handle that is bound to the Retrieve when
// passed to OrderBy. Construct one via an OrderByX free function (e.g.
// OrderByCreatedAt). Order mirrors the Filter pattern: a closure that
// receives the Retrieve at evaluation time and resolves the typed
// SortedQuery against r.indexes, so the cursor stays statically typed
// without needing a method on Retrieve.
type Order func(r Retrieve) gorp.OrderQuery[{{$ret.KeyType}}, {{$ret.GoName}}]

// OrderBy walks the results in the order defined by the given Order handle.
// The Order is invoked with this Retrieve, which lets it reach r.indexes to
// resolve the typed SortedQuery before passing it to gorp.
func (r Retrieve) OrderBy(o Order) Retrieve {
	r.gorp = r.gorp.OrderBy(o(r))
	return r
}
{{- range $idx := $ret.Indexes}}
{{- if $idx.IsSorted}}

// OrderBy{{$idx.GoName}} returns an Order that walks
// {{$ret.GoName | toLower | pluralize}} via the {{$idx.FieldName}} sorted index in
// the given direction. Pass an optional cursor to resume pagination after a
// previously visited value; the cursor is type-checked at the call site.
func OrderBy{{$idx.GoName}}(dir gorp.Direction, cursor ...{{$idx.GoType}}) Order {
	return func(r Retrieve) gorp.OrderQuery[{{$ret.KeyType}}, {{$ret.GoName}}] {
		q := r.indexes.{{$idx.StructField}}.Ordered(dir)
		if len(cursor) > 0 {
			q = q.After(cursor[0])
		}
		return q
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
{{- if .IsBool}}
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
{{- if .IndexFieldName}}
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
{{- if .IndexFieldName}}
		return r.indexes.{{.IndexFieldName}}.Filter(vals...)
{{- else}}
		return gorp.Match(func(_ gorp.Context, e *{{$ret.GoName}}) (bool, error) {
			return slices.Contains(vals, e.{{.Access}}), nil
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

// Offset sets the starting index of the {{$ret.GoName | toLower | pluralize}} to
// return.
func (r Retrieve) Offset(offset int) Retrieve {
	r.gorp = r.gorp.Offset(offset)
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
// Exec executes the query against the provided transaction.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}

// Count returns the number of {{$ret.GoName | toLower | pluralize}} matching the query.
func (r Retrieve) Count(ctx context.Context, tx gorp.Tx) (int, error) {
	return r.gorp.Count(ctx, gorp.OverrideTx(r.baseTX, tx))
}

// Exists checks whether any {{$ret.GoName | toLower | pluralize}} match the query.
func (r Retrieve) Exists(ctx context.Context, tx gorp.Tx) (bool, error) {
	return r.gorp.Exists(ctx, gorp.OverrideTx(r.baseTX, tx))
}
{{- end}}
{{end}}`),
)
