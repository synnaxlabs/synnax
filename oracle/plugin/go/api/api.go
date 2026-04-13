// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package api generates typed filter node structs and interpreters from Oracle schema
// definitions. For each entity annotated with @retrieve and @api output, this plugin
// produces a filter.gen.go file containing a recursive FilterNode struct (with per-field
// typed comparison slots and And/Or/Not combinators) and a ToFilter method that routes
// each leaf to the corresponding service-level MatchX function.
package api

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/key"
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

// Plugin generates typed filter node structs for API packages.
type Plugin struct{ Options Options }

// Options configures the go/api plugin.
type Options struct {
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{FileNamePattern: "filter.gen.go"}
}

// New creates a new go/api plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "go/api" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"go"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"go/query"} }

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

// Generate produces filter.gen.go files for structs with @retrieve and @api output.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	type entry struct {
		apiPath string
		structs []resolution.Type
	}
	entries := make(map[string]*entry)

	for _, typ := range req.Resolutions.StructTypes() {
		if !plugindomain.HasDomainFromType(typ, "retrieve") {
			continue
		}
		apiPath := output.GetPath(typ, "api")
		if apiPath == "" {
			continue
		}
		e, ok := entries[apiPath]
		if !ok {
			e = &entry{apiPath: apiPath}
			entries[apiPath] = e
		}
		e.structs = append(e.structs, typ)
	}

	for _, e := range entries {
		content, err := generateFilterFile(e.apiPath, e.structs, req.Resolutions, req.RepoRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s: %w", e.apiPath, err)
		}
		if content == nil {
			continue
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", e.apiPath, p.Options.FileNamePattern),
			Content: content,
		})
	}

	return resp, nil
}

// filterFieldInfo describes a @filter-annotated field on the entity.
type filterFieldInfo struct {
	FieldName string // oracle field name (e.g. "name")
	GoName    string // PascalCase (e.g. "Name")
	GoType    string // resolved Go type (e.g. "string", "telem.DataType")
	// FilterType is the Go type for the filter struct slot on the FilterNode,
	// e.g. "*filter.StringFilter" or "*filter.BoolFilter".
	FilterType string
	// FilterKind classifies the filter for template branching.
	FilterKind string // "string", "bool", "numeric", "time"
	// NeedsCast is true when the entity field's Go type is a named type distinct
	// from its underlying primitive (e.g. telem.DataType is a named string).
	// The template emits a cast to the primitive before passing to MatchX.
	NeedsCast bool
	// CastType is the primitive type to cast to (e.g. "string") when NeedsCast is true.
	CastType string
	// IsScalar is true for bool fields and @filter scalar fields.
	IsScalar bool
	// HasIndex is true when the field also carries @index and the service-level
	// MatchX can be routed through the index for eq/in operations.
	HasIndex bool
}

// orderFieldInfo describes an @index sorted field for OrderBy generation.
type orderFieldInfo struct {
	FieldName string // oracle field name
	GoName    string // PascalCase
	GoType    string // resolved Go type
	OrderType string // e.g. "*filter.StringOrderBy"
	OrderKind string // "string", "timestamp", "numeric"
}

// filterNodeInfo holds template data for one entity.
type filterNodeInfo struct {
	GoName         string
	KeyType        string
	ServicePkg     string // import alias for service package
	ServicePkgPath string // full import path
	Fields         []filterFieldInfo
	OrderFields    []orderFieldInfo
}

// HasOrderFields reports whether the entity has any sorted index fields.
func (n filterNodeInfo) HasOrderFields() bool { return len(n.OrderFields) > 0 }

type templateData struct {
	imports *imports.Manager
	Package string
	Nodes   []filterNodeInfo
}

func (d *templateData) HasImports() bool          { return d.imports.HasImports() }
func (d *templateData) ExternalImports() []string { return d.imports.ExternalImports() }
func (d *templateData) InternalImports() []imports.InternalImportData {
	return d.imports.InternalImports()
}

func generateFilterFile(
	apiPath string,
	structs []resolution.Type,
	table *resolution.Table,
	repoRoot string,
) ([]byte, error) {
	pkg := naming.DerivePackageName(apiPath)
	imps := imports.NewManager()

	namespace := structs[0].Namespace

	// The resolver context uses the *service* package's @go output path
	// for type resolution (since that's where the entity struct lives),
	// but we're generating into the API package.
	var nodes []filterNodeInfo
	for _, typ := range structs {
		node := extractFilterNodeInfo(typ, table, repoRoot, pkg, imps, namespace)
		if node != nil {
			nodes = append(nodes, *node)
		}
	}

	if len(nodes) == 0 {
		return nil, nil
	}

	// Always need gorp for the Match closure and filter for the comparison structs.
	imps.AddInternal("gorp", gomod.ResolveImportPath("x/go/gorp", repoRoot, gomod.DefaultModulePrefix))
	imps.AddInternal("filter", gomod.ResolveImportPath("x/go/filter", repoRoot, gomod.DefaultModulePrefix))

	// Check if any field needs strings (for Contains) or regexp (for Regex).
	hasStringFilter := lo.SomeBy(nodes, func(n filterNodeInfo) bool {
		return lo.SomeBy(n.Fields, func(f filterFieldInfo) bool {
			return f.FilterKind == "string"
		})
	})
	if hasStringFilter {
		imps.AddExternal("strings")
		imps.AddExternal("regexp")
	}

	data := &templateData{
		Package: pkg,
		Nodes:   nodes,
		imports: imps,
	}

	var buf bytes.Buffer
	if err := filterTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func extractFilterNodeInfo(
	typ resolution.Type,
	table *resolution.Table,
	repoRoot string,
	apiPkg string,
	imps *imports.Manager,
	namespace string,
) *filterNodeInfo {
	if _, ok := typ.Form.(resolution.StructForm); !ok {
		return nil
	}

	// Resolve the service package path from @go output.
	servicePath := output.GetPath(typ, "go")
	if servicePath == "" {
		return nil
	}
	servicePkgPath := gomod.ResolveImportPath(servicePath, repoRoot, gomod.DefaultModulePrefix)
	servicePkgName := naming.DerivePackageName(servicePath)

	// Pick an alias that doesn't collide with the API package name.
	servicePkgAlias := servicePkgName
	if servicePkgAlias == apiPkg {
		// Prefix with parent directory to avoid collision.
		parts := strings.Split(servicePath, "/")
		if len(parts) >= 2 {
			servicePkgAlias = parts[len(parts)-2] + servicePkgName
		}
	}
	imps.AddInternal(servicePkgAlias, servicePkgPath)

	goName := naming.GetGoName(typ)

	// Build resolver for the service package context.
	svcCtx := &resolver.Context{
		Table:                         table,
		OutputPath:                    servicePath,
		Namespace:                     namespace,
		RepoRoot:                      repoRoot,
		DomainName:                    "go",
		SubstituteDefaultedTypeParams: true,
	}
	svcResolver := &resolver.Resolver{
		Formatter:       types.GoFormatter(),
		ImportResolver:  &types.GoImportResolver{RepoRoot: repoRoot, CurrentPackage: servicePkgName},
		PrimitiveMapper: goprimitives.Mapper(),
		// Do not add imports through the resolver since we're in a different
		// package context. We manage imports manually.
		ImportAdder: imports.NewManager(),
	}

	var fields []filterFieldInfo
	var orderFields []orderFieldInfo
	allFields := resolution.UnifiedFields(typ, table)
	for _, field := range allFields {
		hasFilter := plugindomain.HasDomainFromField(field, "filter")
		hasIndex := plugindomain.HasDomainFromField(field, "index")
		if !hasFilter && !hasIndex {
			continue
		}
		goType := svcResolver.ResolveTypeRef(field.Type, svcCtx)
		goFieldName := naming.GetFieldName(field)
		primitive := key.ResolvePrimitive(field.Type, table)

		if hasFilter {
			fi := classifyFilterField(primitive, goType, goFieldName, field, table)
			fi.FieldName = field.Name
			fi.GoName = goFieldName
			fi.GoType = goType
			fi.HasIndex = hasIndex

			isScalar := primitive == "bool" || plugindomain.HasExprFromField(field, "filter", "scalar")
			fi.IsScalar = isScalar

			fields = append(fields, fi)
		}

		if hasIndex && plugindomain.HasExprFromField(field, "index", "sorted") {
			oi := classifyOrderField(primitive, goType, goFieldName)
			oi.FieldName = field.Name
			oi.GoName = goFieldName
			oi.GoType = goType
			orderFields = append(orderFields, oi)
		}
	}

	if len(fields) == 0 && len(orderFields) == 0 {
		return nil
	}

	return &filterNodeInfo{
		GoName:         goName,
		ServicePkg:     servicePkgAlias,
		ServicePkgPath: servicePkgPath,
		Fields:         fields,
		OrderFields:    orderFields,
	}
}

func classifyFilterField(
	primitive string,
	goType string,
	_ string,
	_ resolution.Field,
	_ *resolution.Table,
) filterFieldInfo {
	var fi filterFieldInfo
	switch primitive {
	case "bool":
		fi.FilterType = "*filter.BoolFilter"
		fi.FilterKind = "bool"
	case "string":
		fi.FilterType = "*filter.StringFilter"
		fi.FilterKind = "string"
		if goType != "string" {
			fi.NeedsCast = true
			fi.CastType = "string"
		}
	case "int8", "int16", "int32", "int64",
		"uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		goNumType := primitiveToGoType(primitive)
		fi.FilterType = fmt.Sprintf("*filter.NumericFilter[%s]", goNumType)
		fi.FilterKind = "numeric"
		if goType != goNumType {
			fi.NeedsCast = true
			fi.CastType = goNumType
		}
	case "uint12":
		fi.FilterType = "*filter.NumericFilter[uint16]"
		fi.FilterKind = "numeric"
		fi.NeedsCast = true
		fi.CastType = "uint16"
	case "uint20":
		fi.FilterType = "*filter.NumericFilter[uint32]"
		fi.FilterKind = "numeric"
		fi.NeedsCast = true
		fi.CastType = "uint32"
	default:
		fi.FilterType = "*filter.StringFilter"
		fi.FilterKind = "string"
		if goType != "string" {
			fi.NeedsCast = true
			fi.CastType = "string"
		}
	}
	return fi
}

func classifyOrderField(primitive string, goType string, _ string) orderFieldInfo {
	var oi orderFieldInfo
	switch primitive {
	case "string":
		oi.OrderType = "*filter.StringOrderBy"
		oi.OrderKind = "string"
	case "int64":
		if goType == "telem.TimeStamp" || strings.HasSuffix(goType, ".TimeStamp") {
			oi.OrderType = "*filter.TimestampOrderBy"
			oi.OrderKind = "timestamp"
		} else {
			oi.OrderType = fmt.Sprintf("*filter.NumericOrderBy[%s]", primitiveToGoType(primitive))
			oi.OrderKind = "numeric"
		}
	case "int8", "int16", "int32",
		"uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		goNum := primitiveToGoType(primitive)
		oi.OrderType = fmt.Sprintf("*filter.NumericOrderBy[%s]", goNum)
		oi.OrderKind = "numeric"
	default:
		oi.OrderType = "*filter.StringOrderBy"
		oi.OrderKind = "string"
	}
	return oi
}

func primitiveToGoType(p string) string {
	switch p {
	case "uint12":
		return "uint16"
	case "uint20":
		return "uint32"
	default:
		return p
	}
}

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

var filterTemplate = template.Must(template.New("go-api-filter").Funcs(templateFuncs).Parse(`// Code generated by oracle. DO NOT EDIT.

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

// Ensure imports are used.
var (
	_ = strings.Contains
	_ = regexp.MatchString
)
{{range $node := .Nodes}}
// {{$node.GoName}}FilterNode is a composable, recursive filter tree for
// {{$node.GoName | toLower}} queries. Set any combination of field filters on a
// single node (they are implicitly ANDed). Use And/Or/Not for boolean composition.
type {{$node.GoName}}FilterNode struct {
{{- range $f := $node.Fields}}
	{{$f.GoName}} {{$f.FilterType}} ` + "`" + `json:"{{$f.FieldName}},omitempty" msgpack:"{{$f.FieldName}},omitempty"` + "`" + `
{{- end}}
	And []{{$node.GoName}}FilterNode ` + "`" + `json:"and,omitempty" msgpack:"and,omitempty"` + "`" + `
	Or  []{{$node.GoName}}FilterNode ` + "`" + `json:"or,omitempty" msgpack:"or,omitempty"` + "`" + `
	Not *{{$node.GoName}}FilterNode  ` + "`" + `json:"not,omitempty" msgpack:"not,omitempty"` + "`" + `
}

func (n {{$node.GoName}}FilterNode) toFilter() {{$node.ServicePkg}}.Filter {
	var filters []{{$node.ServicePkg}}.Filter
{{- range $f := $node.Fields}}
	if n.{{$f.GoName}} != nil {
{{- if eq $f.FilterKind "bool"}}
		if n.{{$f.GoName}}.Eq != nil {
{{- if $f.IsScalar}}
			filters = append(filters, {{$node.ServicePkg}}.Match{{$f.GoName}}(*n.{{$f.GoName}}.Eq))
{{- else}}
			filters = append(filters, {{$node.ServicePkg}}.Match{{$f.GoName}}(*n.{{$f.GoName}}.Eq))
{{- end}}
		}
{{- else if eq $f.FilterKind "string"}}
		if n.{{$f.GoName}}.Eq != nil {
{{- if $f.IsScalar}}
			filters = append(filters, {{$node.ServicePkg}}.Match{{$f.GoName}}(*n.{{$f.GoName}}.Eq))
{{- else}}
			filters = append(filters, {{$node.ServicePkg}}.Match{{$f.GoName | pluralize}}(*n.{{$f.GoName}}.Eq))
{{- end}}
		} else if len(n.{{$f.GoName}}.In) > 0 {
{{- if $f.IsScalar}}
			filters = append(filters, {{$node.ServicePkg}}.Match{{$f.GoName}}(n.{{$f.GoName}}.In[0]))
{{- else}}
			filters = append(filters, {{$node.ServicePkg}}.Match{{$f.GoName | pluralize}}(n.{{$f.GoName}}.In...))
{{- end}}
		}
		if n.{{$f.GoName}}.Contains != nil {
			c := *n.{{$f.GoName}}.Contains
			filters = append(filters, {{$node.ServicePkg}}.Match(func(_ gorp.Context, _ {{$node.ServicePkg}}.Retrieve, e *{{$node.ServicePkg}}.{{$node.GoName}}) (bool, error) {
				return strings.Contains({{if $f.NeedsCast}}{{$f.CastType}}(e.{{$f.GoName}}){{else}}e.{{$f.GoName}}{{end}}, c), nil
			}))
		}
		if n.{{$f.GoName}}.Regex != nil {
			rx, err := regexp.Compile(*n.{{$f.GoName}}.Regex)
			if err == nil {
				filters = append(filters, {{$node.ServicePkg}}.Match(func(_ gorp.Context, _ {{$node.ServicePkg}}.Retrieve, e *{{$node.ServicePkg}}.{{$node.GoName}}) (bool, error) {
					return rx.MatchString({{if $f.NeedsCast}}{{$f.CastType}}(e.{{$f.GoName}}){{else}}e.{{$f.GoName}}{{end}}), nil
				}))
			}
		}
{{- else if eq $f.FilterKind "numeric"}}
		if n.{{$f.GoName}}.Eq != nil {
{{- if $f.IsScalar}}
			filters = append(filters, {{$node.ServicePkg}}.Match{{$f.GoName}}(*n.{{$f.GoName}}.Eq))
{{- else}}
			filters = append(filters, {{$node.ServicePkg}}.Match{{$f.GoName | pluralize}}(*n.{{$f.GoName}}.Eq))
{{- end}}
		} else if len(n.{{$f.GoName}}.In) > 0 {
{{- if $f.IsScalar}}
			filters = append(filters, {{$node.ServicePkg}}.Match{{$f.GoName}}(n.{{$f.GoName}}.In[0]))
{{- else}}
			filters = append(filters, {{$node.ServicePkg}}.Match{{$f.GoName | pluralize}}(n.{{$f.GoName}}.In...))
{{- end}}
		}
		if n.{{$f.GoName}}.Gt != nil || n.{{$f.GoName}}.Lt != nil || n.{{$f.GoName}}.Gte != nil || n.{{$f.GoName}}.Lte != nil {
			f := n.{{$f.GoName}}
			filters = append(filters, {{$node.ServicePkg}}.Match(func(_ gorp.Context, _ {{$node.ServicePkg}}.Retrieve, e *{{$node.ServicePkg}}.{{$node.GoName}}) (bool, error) {
				v := {{if $f.NeedsCast}}{{$f.CastType}}(e.{{$f.GoName}}){{else}}e.{{$f.GoName}}{{end}}
				if f.Gt != nil && !(v > *f.Gt) { return false, nil }
				if f.Lt != nil && !(v < *f.Lt) { return false, nil }
				if f.Gte != nil && !(v >= *f.Gte) { return false, nil }
				if f.Lte != nil && !(v <= *f.Lte) { return false, nil }
				return true, nil
			}))
		}
{{- end}}
	}
{{- end}}
	for _, child := range n.And {
		filters = append(filters, child.toFilter())
	}
	if len(n.Or) > 0 {
		orFilters := make([]{{$node.ServicePkg}}.Filter, len(n.Or))
		for i, child := range n.Or {
			orFilters[i] = child.toFilter()
		}
		filters = append(filters, {{$node.ServicePkg}}.Or(orFilters...))
	}
	if n.Not != nil {
		filters = append(filters, {{$node.ServicePkg}}.Not(n.Not.toFilter()))
	}
	if len(filters) == 0 {
		return nil
	}
	if len(filters) == 1 {
		return filters[0]
	}
	return {{$node.ServicePkg}}.And(filters...)
}

// ToFilter converts this {{$node.GoName}}FilterNode to a service-level Filter.
// Returns nil when the node has no constraints.
func (n *{{$node.GoName}}FilterNode) ToFilter() {{$node.ServicePkg}}.Filter {
	if n == nil {
		return nil
	}
	return n.toFilter()
}
{{- if $node.HasOrderFields}}

// {{$node.GoName}}OrderBy specifies sort order for {{$node.GoName | toLower}} queries.
// Only one field may be set per request.
type {{$node.GoName}}OrderBy struct {
{{- range $o := $node.OrderFields}}
	{{$o.GoName}} {{$o.OrderType}} ` + "`" + `json:"{{$o.FieldName}},omitempty" msgpack:"{{$o.FieldName}},omitempty"` + "`" + `
{{- end}}
}

// ToOrderBy converts this {{$node.GoName}}OrderBy to a service-level Order.
// Returns nil when no field is set.
func (o *{{$node.GoName}}OrderBy) ToOrderBy() {{$node.ServicePkg}}.Order {
	if o == nil {
		return nil
	}
{{- range $o := $node.OrderFields}}
	if o.{{$o.GoName}} != nil {
		dir := filter.ParseDirection(o.{{$o.GoName}}.Direction)
{{- if eq $o.OrderKind "string"}}
		if o.{{$o.GoName}}.After != nil {
			return {{$node.ServicePkg}}.OrderBy{{$o.GoName}}(dir, *o.{{$o.GoName}}.After)
		}
		return {{$node.ServicePkg}}.OrderBy{{$o.GoName}}(dir)
{{- else if eq $o.OrderKind "timestamp"}}
		if o.{{$o.GoName}}.After != nil {
			return {{$node.ServicePkg}}.OrderBy{{$o.GoName}}(dir, *o.{{$o.GoName}}.After)
		}
		return {{$node.ServicePkg}}.OrderBy{{$o.GoName}}(dir)
{{- else}}
		if o.{{$o.GoName}}.After != nil {
			return {{$node.ServicePkg}}.OrderBy{{$o.GoName}}(dir, *o.{{$o.GoName}}.After)
		}
		return {{$node.ServicePkg}}.OrderBy{{$o.GoName}}(dir)
{{- end}}
	}
{{- end}}
	return nil
}
{{- end}}
{{end}}`))
