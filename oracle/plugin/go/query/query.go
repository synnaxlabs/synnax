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
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/domain/ontology"
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	plugindomain "github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	goprimitives "github.com/synnaxlabs/oracle/plugin/go/primitives"
	"github.com/synnaxlabs/oracle/plugin/go/types"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/pluralize"
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

// Check verifies generated files are up-to-date.
func (p *Plugin) Check(*plugin.Request) error { return nil }

var goPostWriter = &exec.PostWriter{
	Extensions: []string{".go"},
	Commands:   [][]string{{"gofmt", "-w"}},
}

// PostWrite runs gofmt on generated files.
func (p *Plugin) PostWrite(files []string) error {
	return goPostWriter.PostWrite(files)
}

// Generate produces retrieve.gen.go files for structs with @retrieve annotation.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	// Group @retrieve structs by output path.
	type entry struct {
		path    string
		structs []resolution.Type
	}
	entries := make(map[string]*entry)

	for _, typ := range req.Resolutions.StructTypes() {
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
		e.structs = append(e.structs, typ)
	}

	for _, e := range entries {
		content, err := generateRetrieveFile(e.path, e.structs, req.Resolutions, req.RepoRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s: %w", e.path, err)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", e.path, p.Options.FileNamePattern),
			Content: content,
		})
	}

	return resp, nil
}

// filterInfo holds extracted data about a @filter-annotated field.
type filterInfo struct {
	FieldName string // oracle field name
	GoName    string // PascalCase
	GoType    string // resolved Go type
	IsScalar  bool   // @filter scalar or bool type
	IsBool    bool   // underlying primitive is bool
	// IndexFieldName is the camelCase field name on the per-Retrieve indexes
	// struct if the same field also carries @index. When non-empty, the
	// generated MatchX / MatchXs routes through r.indexes.<IndexFieldName>
	// instead of a bespoke scan closure. Empty means no index on this field.
	IndexFieldName string
}

// indexInfo holds extracted data about an @index-annotated field. Emitted as
// a constructor function (newXIndex) and a field on the per-Retrieve indexes
// struct that the Service constructs once and threads onto every Retrieve.
type indexInfo struct {
	FieldName   string // oracle field name (e.g. "created_at")
	GoName      string // PascalCase field name (e.g. "CreatedAt")
	GoType      string // resolved Go type (e.g. "telem.TimeStamp")
	StructField string // camelCase field on the indexes struct (e.g. "createdAt")
	Kind        string // "lookup" or "sorted"
}

// IsSorted reports whether the index is a sorted index. Used by the template
// to decide whether to emit an OrderBy helper.
func (i indexInfo) IsSorted() bool { return i.Kind == "sorted" }

// retrieveInfo holds extracted data about a @retrieve-annotated struct.
type retrieveInfo struct {
	TypeName                    string
	GoName                      string
	KeyType                     string
	KeyPrimitive                string
	HasSearch                   bool
	IsCustom                    bool // @retrieve custom - user defines the struct
	OntologyType                string
	KeysFromOntologyIDsHasError bool
	Filters                     []filterInfo
	Indexes                     []indexInfo
}

// HasIndexes reports whether the retrieve declared any @index fields. Used by
// the template to gate the Indexes registration slice.
func (r retrieveInfo) HasIndexes() bool { return len(r.Indexes) > 0 }

// HasSortedIndex reports whether the retrieve declared any @index sorted
// fields. Used by the template to gate the Order closure type, the
// Retrieve.OrderBy method, and the per-field OrderByX free functions.
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
		Formatter:       types.GoFormatter(),
		ImportResolver:  &types.GoImportResolver{RepoRoot: repoRoot, CurrentPackage: pkg},
		ImportAdder:     imps,
		PrimitiveMapper: goprimitives.Mapper(),
	}

	var infos []retrieveInfo
	for _, typ := range structs {
		info := extractRetrieveInfo(typ, table, r, ctx)
		if info != nil {
			infos = append(infos, *info)
		}
	}

	if len(infos) == 0 {
		return nil, nil
	}

	// Always need context and gorp.
	imps.AddExternal("context")
	imps.AddInternal("gorp", gomod.ResolveImportPath("x/go/gorp", repoRoot, gomod.DefaultModulePrefix))

	// Check if any info needs search imports.
	hasSearch := lo.SomeBy(infos, func(i retrieveInfo) bool { return i.HasSearch })
	if hasSearch {
		imps.AddInternal("search", gomod.ResolveImportPath("core/pkg/distribution/search", repoRoot, gomod.DefaultModulePrefix))
		imps.AddInternal("ontology", gomod.ResolveImportPath("core/pkg/distribution/ontology", repoRoot, gomod.DefaultModulePrefix))
	}

	// Check if any info needs lo for slice filters. Index-routed filters use
	// the index's .Filter(...) method instead of lo.Contains, so only filters
	// without an index contribute.
	hasSliceFilter := lo.SomeBy(infos, func(i retrieveInfo) bool {
		return lo.SomeBy(i.Filters, func(f filterInfo) bool {
			return !f.IsScalar && f.IndexFieldName == ""
		})
	})
	if hasSliceFilter {
		imps.AddExternal("github.com/samber/lo")
	}

	data := &templateData{
		Package:   pkg,
		Retrieves: infos,
		imports:   imps,
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
) *retrieveInfo {
	form, ok := typ.Form.(resolution.StructForm)
	if !ok {
		return nil
	}

	// Resolve the key type. A struct-level `@retrieve key "TypeName"` override
	// takes precedence; this is useful when the entry type has a computed
	// GorpKey that is not stored as a field on the struct (e.g. channel.Channel).
	// Otherwise, fall back to the first field carrying `@key`.
	var keyType, keyPrimitive string
	if keyTypeName := plugindomain.GetStringFromType(typ, "retrieve", "key"); keyTypeName != "" {
		qualified := keyTypeName
		if typ.Namespace != "" && !strings.Contains(keyTypeName, ".") {
			qualified = typ.Namespace + "." + keyTypeName
		}
		keyRef := resolution.TypeRef{Name: qualified}
		keyType = r.ResolveTypeRef(keyRef, ctx)
		keyPrimitive = key.ResolvePrimitive(keyRef, table)
	} else {
		var keyField *resolution.Field
		for i := range form.Fields {
			if plugindomain.HasDomainFromField(form.Fields[i], "key") {
				keyField = &form.Fields[i]
				break
			}
		}
		if keyField == nil {
			return nil
		}
		keyType = r.ResolveTypeRef(keyField.Type, ctx)
		keyPrimitive = key.ResolvePrimitive(keyField.Type, table)
	}

	goName := naming.GetGoName(typ)

	isCustom := plugindomain.HasExprFromType(typ, "retrieve", "custom")
	hasSearch := plugindomain.HasDomainFromType(typ, "search")

	// Get ontology type for search. Check @search type "X" first, then fall
	// back to @ontology type "X" on the struct itself.
	var ontologyType string
	if hasSearch {
		if searchType := plugindomain.GetStringFromType(typ, "search", "type"); searchType != "" {
			ontologyType = searchType
		} else {
			keyFields := key.Collect([]resolution.Type{typ}, table, nil)
			ontData := ontology.Extract([]resolution.Type{typ}, keyFields, nil)
			if ontData != nil {
				ontologyType = ontData.TypeName
			}
		}
	}

	// Collect @filter and @index fields. A field may carry both: in that
	// case, MatchX / MatchXs are generated to route through the index.
	// Resolve the Go type only for fields that actually contribute generated
	// code — otherwise ResolveTypeRef would add imports for types that are
	// never referenced in retrieve.gen.go.
	var filters []filterInfo
	var indexes []indexInfo
	allFields := resolution.UnifiedFields(typ, table)
	for _, field := range allFields {
		hasFilter := plugindomain.HasDomainFromField(field, "filter")
		hasIndex := plugindomain.HasDomainFromField(field, "index")
		if !hasFilter && !hasIndex {
			continue
		}
		goType := r.ResolveTypeRef(field.Type, ctx)
		goFieldName := naming.GetFieldName(field)

		var structField string
		if hasIndex {
			kind := "lookup"
			if plugindomain.HasExprFromField(field, "index", "sorted") {
				kind = "sorted"
			}
			// camelCase field name on the per-Retrieve indexes struct.
			// Unexported because the indexes struct itself is unexported.
			structField = naming.LowerFirst(goFieldName)
			indexes = append(indexes, indexInfo{
				FieldName:   field.Name,
				GoName:      goFieldName,
				GoType:      goType,
				StructField: structField,
				Kind:        kind,
			})
		}

		if hasFilter {
			primitive := key.ResolvePrimitive(field.Type, table)
			isBool := primitive == "bool"
			isScalar := isBool || plugindomain.HasExprFromField(field, "filter", "scalar")
			filters = append(filters, filterInfo{
				FieldName:      field.Name,
				GoName:         goFieldName,
				GoType:         goType,
				IsScalar:       isScalar,
				IsBool:         isBool,
				IndexFieldName: structField,
			})
		}
	}

	return &retrieveInfo{
		TypeName:                    typ.Name,
		GoName:                      goName,
		KeyType:                     keyType,
		KeyPrimitive:                keyPrimitive,
		HasSearch:                   hasSearch,
		IsCustom:                    isCustom,
		OntologyType:                ontologyType,
		KeysFromOntologyIDsHasError: keyPrimitive != "string",
		Filters:                     filters,
		Indexes:                     indexes,
	}
}

type templateData struct {
	imports   *imports.Manager
	Package   string
	Retrieves []retrieveInfo
}

func (d *templateData) HasImports() bool          { return d.imports.HasImports() }
func (d *templateData) ExternalImports() []string { return d.imports.ExternalImports() }
func (d *templateData) InternalImports() []imports.InternalImportData {
	return d.imports.InternalImports()
}

// pluralizeDistinct returns the plural form of a name. If the plural is the
// same as the singular (already-plural words like "Properties"), it appends
// "List" to avoid name collisions.
func pluralizeDistinct(name string) string {
	plural := pluralize.String(name)
	if plural == name {
		return name + "List"
	}
	return plural
}

var templateFuncs = template.FuncMap{
	"toPascal":  naming.ToPascalCase,
	"toLower":   strings.ToLower,
	"pluralize": pluralizeDistinct,
}

var retrieveTemplate = template.Must(template.New("go-retrieve").Funcs(templateFuncs).Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}
{{- if .HasImports}}

import (
{{- range .ExternalImports}}
	"{{.}}"
{{- end}}
{{- range .InternalImports}}
{{- if .NeedsAlias}}
	{{.Alias}} "{{.Path}}"
{{- else}}
	"{{.Path}}"
{{- end}}
{{- end}}
)
{{- end}}
{{range $ret := .Retrieves}}
{{- if not $ret.IsCustom}}
// Retrieve is used to retrieve {{$ret.GoName}} records from the database using a
// builder pattern for constructing queries.
type Retrieve struct {
	baseTX gorp.Tx
	gorp   gorp.Retrieve[{{$ret.KeyType}}, {{$ret.GoName}}]
{{- if $ret.HasSearch}}
	search     *search.Index
	searchTerm string
{{- end}}
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
	{{$idx.StructField}} *gorp.{{if $idx.IsSorted}}Sorted{{else}}Lookup{{end}}[{{$ret.KeyType}}, {{$ret.GoName}}, {{$idx.GoType}}]
{{- end}}
}

// newIndexes constructs a fresh indexes value, allocating one index instance
// per registered field. Call once per Service in OpenService and store the
// result on the Service struct.
func newIndexes() indexes {
	return indexes{
{{- range $idx := $ret.Indexes}}
		{{$idx.StructField}}: gorp.{{if $idx.IsSorted}}NewSorted{{else}}NewLookup{{end}}[{{$ret.KeyType}}, {{$ret.GoName}}, {{$idx.GoType}}](
			"{{$idx.FieldName}}",
			func(e *{{$ret.GoName}}) {{$idx.GoType}} { return e.{{$idx.GoName}} },
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
	}
}
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

// OrderBy{{$idx.GoName}} returns an Order that walks {{$ret.GoName | toLower | pluralize}} via the
// {{$idx.FieldName}} sorted index in the given direction. Pass an optional
// cursor to resume pagination after a previously visited value; the cursor
// is type-checked at the call site.
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

// Filter is a per-service filter that is bound to the Retrieve when passed to
// Where. Pure filters ignore the Retrieve argument; service-bound filters read
// from it (e.g. r.indexes, r.label, r.hostProvider) to evaluate. Use Match to
// construct one from a closure.
type Filter func(r Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}]

// Match wraps a closure that needs the Retrieve into a Filter. The Retrieve
// value is supplied by Retrieve.Where at evaluation time.
func Match(
	f func(ctx gorp.Context, r Retrieve, e *{{$ret.GoName}}) (bool, error),
) Filter {
	return func(r Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
		return gorp.Match(func(ctx gorp.Context, e *{{$ret.GoName}}) (bool, error) {
			return f(ctx, r, e)
		})
	}
}

// And returns a filter that matches when all provided filters match.
func And(fs ...Filter) Filter {
	return func(r Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
		inner := make([]gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}], len(fs))
		for i, f := range fs {
			inner[i] = f(r)
		}
		return gorp.And(inner...)
	}
}

// Or returns a filter that matches when any provided filter matches.
func Or(fs ...Filter) Filter {
	return func(r Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
		inner := make([]gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}], len(fs))
		for i, f := range fs {
			inner[i] = f(r)
		}
		return gorp.Or(inner...)
	}
}

// Not returns a filter that inverts the provided filter.
func Not(f Filter) Filter {
	return func(r Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
		return gorp.Not(f(r))
	}
}
{{if $ret.HasSearch}}
// Search sets a fuzzy search term that Retrieve will use to filter results.
func (r Retrieve) Search(term string) Retrieve { r.searchTerm = term; return r }
{{end}}
// WhereKeys filters for {{$ret.GoName | toLower | pluralize}} whose key matches any of the provided keys.
func (r Retrieve) WhereKeys(keys ...{{$ret.KeyType}}) Retrieve {
	r.gorp = r.gorp.WhereKeys(keys...)
	return r
}
{{range .Filters}}
{{- if .IsBool}}
// Match{{.GoName}} returns a filter for {{$ret.GoName | toLower | pluralize}} by their {{.GoName}} field.
func Match{{.GoName}}(v bool) Filter {
	return func(r Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
{{- if .IndexFieldName}}
		return r.indexes.{{.IndexFieldName}}.Filter(v)
{{- else}}
		return gorp.Match(func(_ gorp.Context, e *{{$ret.GoName}}) (bool, error) {
			return e.{{.GoName}} == v, nil
		})
{{- end}}
	}
}
{{else if .IsScalar}}
// Match{{.GoName}} returns a filter for {{$ret.GoName | toLower | pluralize}} whose {{.GoName}} matches the provided value.
func Match{{.GoName}}(v {{.GoType}}) Filter {
	return func(r Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
{{- if .IndexFieldName}}
		return r.indexes.{{.IndexFieldName}}.Filter(v)
{{- else}}
		return gorp.Match(func(_ gorp.Context, e *{{$ret.GoName}}) (bool, error) {
			return e.{{.GoName}} == v, nil
		})
{{- end}}
	}
}
{{else}}
// Match{{.GoName | pluralize}} returns a filter for {{$ret.GoName | toLower | pluralize}} whose {{.GoName}} matches any of the provided values.
func Match{{.GoName | pluralize}}(vals ...{{.GoType}}) Filter {
	return func(r Retrieve) gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}] {
{{- if .IndexFieldName}}
		return r.indexes.{{.IndexFieldName}}.Filter(vals...)
{{- else}}
		return gorp.Match(func(_ gorp.Context, e *{{$ret.GoName}}) (bool, error) {
			return lo.Contains(vals, e.{{.GoName}}), nil
		})
{{- end}}
	}
}
{{end}}
{{- end}}
// Where applies the provided filters to the query, binding each filter to the
// Retrieve so service-bound filters can read from r.indexes, r.label,
// r.hostProvider, etc.
func (r Retrieve) Where(filters ...Filter) Retrieve {
	bound := make([]gorp.Filter[{{$ret.KeyType}}, {{$ret.GoName}}], len(filters))
	for i, f := range filters {
		bound[i] = f(r)
	}
	r.gorp = r.gorp.Where(bound...)
	return r
}

// Entry binds the provided {{$ret.GoName | toLower}} as the result container for the query. If
// multiple {{$ret.GoName | toLower | pluralize}} match, the first one is used.
func (r Retrieve) Entry(e *{{$ret.GoName}}) Retrieve {
	r.gorp = r.gorp.Entry(e)
	return r
}

// Entries binds the provided slice of {{$ret.GoName | toLower | pluralize}} as the result container for the query.
func (r Retrieve) Entries(es *[]{{$ret.GoName}}) Retrieve {
	r.gorp = r.gorp.Entries(es)
	return r
}

// Limit sets the maximum number of {{$ret.GoName | toLower | pluralize}} to return.
func (r Retrieve) Limit(limit int) Retrieve { r.gorp = r.gorp.Limit(limit); return r }

// Offset sets the starting index of the {{$ret.GoName | toLower | pluralize}} to return.
func (r Retrieve) Offset(offset int) Retrieve {
	r.gorp = r.gorp.Offset(offset)
	return r
}
{{if $ret.HasSearch}}
func (r Retrieve) execSearch(ctx context.Context) (Retrieve, error) {
	if r.searchTerm == "" {
		return r, nil
	}
	ids, err := r.search.Search(ctx, search.Request{
		Type: ontology.ResourceType{{$ret.OntologyType | toPascal}},
		Term: r.searchTerm,
	})
	if err != nil {
		return Retrieve{}, err
	}
{{- if $ret.KeysFromOntologyIDsHasError}}
	keys, err := KeysFromOntologyIDs(ids)
	if err != nil {
		return Retrieve{}, err
	}
{{- else}}
	keys := KeysFromOntologyIDs(ids)
{{- end}}
	return r.WhereKeys(keys...), nil
}
{{end}}
// Exec executes the query against the provided transaction.
func (r Retrieve) Exec(ctx context.Context, tx gorp.Tx) error {
{{- if $ret.HasSearch}}
	var err error
	if r, err = r.execSearch(ctx); err != nil {
		return err
	}
{{- end}}
	return r.gorp.Exec(ctx, gorp.OverrideTx(r.baseTX, tx))
}

// Count returns the number of {{$ret.GoName | toLower | pluralize}} matching the query.
func (r Retrieve) Count(ctx context.Context, tx gorp.Tx) (int, error) {
{{- if $ret.HasSearch}}
	var err error
	if r, err = r.execSearch(ctx); err != nil {
		return 0, err
	}
{{- end}}
	return r.gorp.Count(ctx, gorp.OverrideTx(r.baseTX, tx))
}

// Exists checks whether any {{$ret.GoName | toLower | pluralize}} match the query.
func (r Retrieve) Exists(ctx context.Context, tx gorp.Tx) (bool, error) {
{{- if $ret.HasSearch}}
	var err error
	if r, err = r.execSearch(ctx); err != nil {
		return false, err
	}
{{- end}}
	return r.gorp.Exists(ctx, gorp.OverrideTx(r.baseTX, tx))
}
{{end}}`))
