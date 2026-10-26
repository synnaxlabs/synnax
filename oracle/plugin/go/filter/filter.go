// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package filter generates the Go interpreter of a filter definition. MatchWhere turns
// a
// Where into the service's Filter, OrderingOf turns an Order into the service's
// Ordering, and the retrieve request rejects unknown JSON members.
package filter

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/matcher"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/typemap"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/internal/arrays"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Options configures the go/filter plugin.
type Options struct {
	// FileNamePattern is the basename written for each output package.
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options { return Options{FileNamePattern: "where.gen.go"} }

// Plugin generates the Go interpreter of each filter definition.
type Plugin struct{ Options Options }

// New creates a new go/filter plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "go/filter" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"go"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"go/types"} }

// Generate emits a where.gen.go next to each filter definition with a Go output. A
// filter in a version file gets none, since the service reads only the live one.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{}
	for _, typ := range req.Resolutions.StructTypes() {
		if typ.Filter == nil || output.IsOmitted(typ, "go") {
			continue
		}
		if _, _, ok := paths.VersionFile(typ.FilePath); ok {
			continue
		}
		outputPath := output.GetPath(typ, "go")
		if outputPath == "" {
			continue
		}
		content, err := generateFile(typ, outputPath, req)
		if err != nil {
			return nil, err
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    outputPath + "/" + p.Options.FileNamePattern,
			Content: content,
		})
	}
	return resp, nil
}

// operators lists the members of each operator type in schemas/x/filter.oracle.
var operators = map[string][]string{
	"String":   {"Eq", "In", "Regex"},
	"Bool":     {"Eq"},
	"Set":      {"Eq", "In"},
	"Ordered":  {"Eq", "In", "Gt", "Gte", "Lt", "Lte"},
	"Contains": {"Any"},
}

// comparisons maps each Ordered member that no matcher covers to its Go operator.
var comparisons = map[string]string{"Gt": ">", "Gte": ">=", "Lt": "<", "Lte": "<="}

type templateData struct {
	*imports.Manager
	Package string
	// Target is the Go type of a target record, such as Status[Details].
	Target string
	// TypeParams declares the target's type parameters, such as [Details any].
	TypeParams string
	// TypeArgs passes the target's type parameters, such as [Details].
	TypeArgs string
	Lines    []lineData
	// OrderFields lists the sorted fields, or nil when the request has no order_by.
	OrderFields []orderFieldData
}

type lineData struct {
	// Path is the line's name in a validate.PathError.
	Path string
	// GoName is the Where field that holds the line.
	GoName string
	// Func is the matcher of a bare line, or the hand-written function of a typed line.
	Func string
	// Operator is the operator type of a bare line, and empty for a typed line.
	Operator string
	// One is true when Func takes one value, so an `in` ORs one call per value.
	One bool
	// Field is the selector that reads the target field, for members no matcher covers.
	Field string
	// Deref is true when a typed line's value is a pointer.
	Deref bool
	// Spread is true when a typed line's value is a slice passed as variadic values.
	Spread bool
	// Enum is true when a bare line selects string enum values, which it validates.
	Enum bool
}

// Has reports whether the line's operator has the member.
func (l lineData) Has(member string) bool {
	return lo.Contains(operators[l.Operator], member)
}

type comparisonData struct{ Member, Op string }

// Comparisons returns the Ordered members the line compares with a closure.
func (l lineData) Comparisons() []comparisonData {
	if l.Operator != "Ordered" {
		return nil
	}
	return lo.Map(
		[]string{"Gt", "Gte", "Lt", "Lte"},
		func(m string, _ int) comparisonData {
			return comparisonData{Member: m, Op: comparisons[m]}
		},
	)
}

type orderFieldData struct {
	// Const is the OrderField constant.
	Const string
	// Func is the go/query OrderBy function.
	Func string
}

func generateFile(
	where resolution.Type,
	outputPath string,
	req *plugin.Request,
) ([]byte, error) {
	table := req.Resolutions
	fail := func(format string, args ...any) ([]byte, error) {
		return nil, errors.Newf(
			"filter %s: "+format,
			append([]any{where.QualifiedName}, args...)...,
		)
	}
	target, _, ok := resolution.Dealias(where.Filter.Target, table)
	if !ok {
		return fail("unknown target %s", where.Filter.Target.Name)
	}
	if got := output.GetPath(target, "go"); got != outputPath {
		return fail(
			"the target %s generates into %q, but the filter generates into %q",
			target.QualifiedName,
			got,
			outputPath,
		)
	}
	matchers, err := matcher.List(target, table)
	if err != nil {
		return nil, err
	}
	byField := lo.KeyBy(
		matchers,
		func(m matcher.Matcher) string { return m.Field.Name },
	)
	funcs := lo.SliceToMap(matchers, func(m matcher.Matcher) (string, struct{}) {
		return m.Func, struct{}{}
	})
	funcs["MatchKeys"] = struct{}{}
	funcs["MatchWhere"] = struct{}{}

	pkg := naming.DerivePackageName(outputPath)
	data := templateData{
		Manager: imports.NewManager(),
		Package: pkg,
	}
	var params []string
	for _, tp := range resolution.NonDefaultedTypeParams(
		resolution.TypeParams(target.Form),
	) {
		if tp.Constraint != nil {
			return fail(
				"the type parameter %s of %s has a constraint, which the interpreter "+
					"cannot declare",
				tp.Name,
				target.Name,
			)
		}
		params = append(params, tp.Name)
	}
	data.Target = naming.GetGoName(target)
	if len(params) > 0 {
		data.TypeParams = "[" + strings.Join(params, ", ") + " any]"
		data.TypeArgs = "[" + strings.Join(params, ", ") + "]"
		data.Target += data.TypeArgs
	}

	form := where.Form.(resolution.StructForm)
	for _, line := range where.Filter.Lines {
		field := lo.Must(form.Field(line.Name))
		ld := lineData{Path: line.Name, GoName: naming.GetFieldName(field)}
		if !line.Bare() {
			ld.Func = "Match" + ld.GoName
			if _, taken := funcs[ld.Func]; taken {
				return fail(
					"typed line %s calls %s, which the generated code already "+
						"declares; rename the line",
					line.Name,
					ld.Func,
				)
			}
			ld.Spread = arrays.IsArray(field.Type, table) &&
				arrays.Size(field.Type, table) == nil
			ld.Deref = !typemap.Nilable(field.Type, table)
			data.Lines = append(data.Lines, ld)
			continue
		}
		ld.Operator = strings.TrimPrefix(field.Type.Name, "filter.")
		ld.Enum = ld.Operator == "Set" && isStringEnum(field.Type.TypeArgs[0], table)
		// The matcher list leaves out the key, which MatchKeys matches.
		if m, ok := byField[line.Name]; ok {
			ld.Func = m.Func
			ld.One = m.Kind != matcher.Values
			ld.Field = m.Access
		} else {
			ld.Func = "MatchKeys"
		}
		data.Lines = append(data.Lines, ld)
	}

	request := table.MustGet(where.Namespace + ".RetrieveRequest")
	if _, ok := request.Form.(resolution.StructForm).Field("order_by"); ok {
		orderField := table.MustGet(where.Namespace + ".OrderField")
		for _, v := range orderField.Form.(resolution.EnumForm).Values {
			goName := naming.ToPascalCase(v.Name)
			data.OrderFields = append(data.OrderFields, orderFieldData{
				Const: "OrderField" + goName,
				Func:  "OrderBy" + goName,
			})
		}
	}

	addImports(data, req.RepoRoot)
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func isStringEnum(ref resolution.TypeRef, table *resolution.Table) bool {
	typ, _, ok := resolution.Dealias(ref, table)
	if !ok {
		return false
	}
	form, ok := typ.Form.(resolution.EnumForm)
	return ok && !form.IsIntEnum
}

func addImports(data templateData, repoRoot string) {
	internal := func(name, path string) {
		data.AddInternal(
			name,
			gomod.ResolveImportPath(path, repoRoot, gomod.DefaultModulePrefix),
		)
	}
	data.AddExternal("encoding/json/jsontext")
	data.AddExternal("encoding/json/v2")
	data.AddExternal("strconv")
	internal("validate", "x/go/validate")
	needsRegex := lo.SomeBy(data.Lines, func(l lineData) bool { return l.Has("Regex") })
	needsMatch := needsRegex || lo.SomeBy(data.Lines, func(l lineData) bool {
		return len(l.Comparisons()) > 0
	})
	if needsRegex {
		data.AddExternal("regexp")
	}
	needsEnum := lo.SomeBy(data.Lines, func(l lineData) bool { return l.Enum })
	if needsRegex || needsEnum || data.OrderFields != nil {
		internal("errors", "x/go/errors")
	}
	if needsMatch || data.OrderFields != nil {
		internal("gorp", "gorp")
	}
	if data.OrderFields != nil {
		internal("filter", "x/go/filter")
	}
}

var fileTemplate = template.Must(
	template.New("go-filter").Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

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

// MatchWhere returns the filter that w describes. It returns a validate.PathError for a
// value that no filter can hold, such as an invalid regex.
func MatchWhere{{.TypeParams}}(w Where) (Filter{{.TypeArgs}}, error) {
	var fs []Filter{{.TypeArgs}}
	if w.And != nil {
		and, err := matchWheres{{.TypeArgs}}(w.And, "and")
		if err != nil {
			return nil, err
		}
		fs = append(fs, And{{.TypeArgs}}(and...))
	}
	if w.Or != nil {
		or, err := matchWheres{{.TypeArgs}}(w.Or, "or")
		if err != nil {
			return nil, err
		}
		fs = append(fs, Or{{.TypeArgs}}(or...))
	}
	if w.Not != nil {
		not, err := MatchWhere{{.TypeArgs}}(*w.Not)
		if err != nil {
			return nil, validate.PathedError(err, "not")
		}
		fs = append(fs, Not{{.TypeArgs}}(not))
	}
{{- range .Lines}}
{{- $line := .}}
	if l := w.{{.GoName}}; l != nil {
{{- if not .Operator}}
		fs = append(fs, {{.Func}}{{$.TypeArgs}}({{if .Deref}}*{{end}}l{{if .Spread}}...{{end}}))
{{- else}}
{{- if .Has "Eq"}}
		if l.Eq != nil {
{{- if .Enum}}
			if !l.Eq.IsValid() {
				return nil, validate.PathedError(
					errors.Wrapf(validate.ErrValidation, "invalid {{.Path}} %q", *l.Eq),
					"{{.Path}}",
					"eq",
				)
			}
{{- end}}
			fs = append(fs, {{.Func}}{{$.TypeArgs}}(*l.Eq))
		}
{{- end}}
{{- if .Has "In"}}
		if l.In != nil {
{{- if .Enum}}
			for i, v := range l.In {
				if !v.IsValid() {
					return nil, validate.PathedError(
						errors.Wrapf(validate.ErrValidation, "invalid {{.Path}} %q", v),
						"{{.Path}}",
						"in",
						strconv.Itoa(i),
					)
				}
			}
{{- end}}
{{- if .One}}
			in := make([]Filter{{$.TypeArgs}}, len(l.In))
			for i, v := range l.In {
				in[i] = {{.Func}}{{$.TypeArgs}}(v)
			}
			fs = append(fs, Or{{$.TypeArgs}}(in...))
{{- else}}
			fs = append(fs, {{.Func}}{{$.TypeArgs}}(l.In...))
{{- end}}
		}
{{- end}}
{{- if .Has "Any"}}
		if l.Any != nil {
			anyOf := make([]Filter{{$.TypeArgs}}, len(l.Any))
			for i, v := range l.Any {
				anyOf[i] = {{.Func}}{{$.TypeArgs}}(v)
			}
			fs = append(fs, Or{{$.TypeArgs}}(anyOf...))
		}
{{- end}}
{{- if .Has "Regex"}}
		if l.Regex != nil {
			re, err := regexp.Compile(*l.Regex)
			if err != nil {
				return nil, validate.PathedError(
					errors.Wrap(validate.ErrValidation, err.Error()),
					"{{.Path}}",
					"regex",
				)
			}
			fs = append(fs, Match{{$.TypeArgs}}(func(
				_ gorp.Context,
				_ Retrieve{{$.TypeArgs}},
				e *{{$.Target}},
			) (bool, error) {
				return re.MatchString(string(e.{{.Field}})), nil
			}))
		}
{{- end}}
{{- range .Comparisons}}
		if l.{{.Member}} != nil {
			v := *l.{{.Member}}
			fs = append(fs, Match{{$.TypeArgs}}(func(
				_ gorp.Context,
				_ Retrieve{{$.TypeArgs}},
				e *{{$.Target}},
			) (bool, error) {
				return e.{{$line.Field}} {{.Op}} v, nil
			}))
		}
{{- end}}
{{- end}}
	}
{{- end}}
	return And{{.TypeArgs}}(fs...), nil
}

func matchWheres{{.TypeParams}}(ws []Where, path string) ([]Filter{{.TypeArgs}}, error) {
	fs := make([]Filter{{.TypeArgs}}, len(ws))
	for i, w := range ws {
		f, err := MatchWhere{{.TypeArgs}}(w)
		if err != nil {
			return nil, validate.PathedError(err, path, strconv.Itoa(i))
		}
		fs[i] = f
	}
	return fs, nil
}
{{- if .OrderFields}}

// OrderingOf returns the Ordering that o describes. It returns a validate.PathError for
// a field or direction that Order does not declare.
func OrderingOf{{.TypeParams}}(o Order) (Ordering{{.TypeArgs}}, error) {
	var dir gorp.Direction
	switch o.Direction {
	case filter.DirectionAscending:
		dir = gorp.DirectionAsc
	case filter.DirectionDescending:
		dir = gorp.DirectionDesc
	default:
		return nil, validate.PathedError(
			errors.Wrapf(validate.ErrValidation, "unknown direction %q", o.Direction),
			"direction",
		)
	}
	switch o.Field {
{{- range .OrderFields}}
	case {{.Const}}:
		return {{.Func}}{{$.TypeArgs}}(dir), nil
{{- end}}
	}
	return nil, validate.PathedError(
		errors.Wrapf(validate.ErrValidation, "unknown field %q", o.Field),
		"field",
	)
}
{{- end}}

// UnmarshalJSONFrom decodes the request and rejects a member that it does not declare,
// at any depth, so a client that sends a field the Core removed fails.
func (r *RetrieveRequest) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain RetrieveRequest
	return json.UnmarshalDecode(dec, (*plain)(r), json.RejectUnknownMembers(true))
}
`),
)
