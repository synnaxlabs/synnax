// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package filter generates the Go interpreter of a filter definition. MatchWhere turns
// a Where into the service's Filter, and RetrieveRequest.Apply, generated wherever the
// request is, narrows a Retrieve to the records a request selects. The request also
// rejects unknown JSON members.
package filter

import (
	"bytes"
	"slices"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin"
	plugindomain "github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/matcher"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/typemap"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/internal/arrays"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// Options configures the go/filter plugin.
type Options struct {
	// FileNamePattern is the basename written next to each filter.
	FileNamePattern string
	// RequestFileNamePattern is the basename written next to each derived request.
	RequestFileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		FileNamePattern:        "where.gen.go",
		RequestFileNamePattern: "request.gen.go",
	}
}

// Plugin generates the Go interpreter of each filter definition.
type Plugin struct {
	// Options sets the names of the generated files.
	Options Options
}

// New creates a new go/filter plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "go/filter" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"go"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"go/types", "go/query"} }

// Generate emits a where.gen.go next to each filter definition with a Go output, and a
// request.gen.go next to its derived request. A filter in a version file gets neither,
// since the service reads only the live one.
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
		request, ok := req.Resolutions.Get(typ.Filter.Request)
		if !ok {
			return nil, errors.Newf(
				"filter %s: has no derived retrieve request", typ.QualifiedName,
			)
		}
		requestPath := output.GetPath(request, "go")
		content, err = generateRequestFile(typ, request, outputPath, requestPath, req)
		if err != nil {
			return nil, err
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    requestPath + "/" + p.Options.RequestFileNamePattern,
			Content: content,
		})
	}
	return resp, nil
}

// comparisons maps each ordering member to its Go operator.
var comparisons = map[string]string{"gt": ">", "gte": ">=", "lt": "<", "lte": "<="}

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
}

type lineData struct {
	// Path is the line's name in a validate.PathError.
	Path string
	// GoName is the Where field that holds the line.
	GoName string
	// Func is the matcher of a bare line, or the hand-written function of a typed line.
	Func string
	// Typed is true when the line calls a hand-written matcher.
	Typed bool
	// Members are the members of a bare line's operator.
	Members []string
	// Operators are the Go names of the optional members, the ones that select values.
	Operators []string
	// Scalar is true when Func takes one value, so an `in` ORs one call per value.
	Scalar bool
	// Field is the selector that reads the target field, for members no matcher covers.
	Field string
	// Pointer is true when a typed line's value is a pointer.
	Pointer bool
	// Variadic is true when a typed line's value is a slice passed as variadic values.
	Variadic bool
	// Enum is true when a bare line selects string enum values, which it validates.
	Enum bool
	// Text is true when a bare line's operator is filter.String.
	Text bool
}

// Has reports whether the line's operator has the member.
func (l lineData) Has(member string) bool { return lo.Contains(l.Members, member) }

// NoOperator returns the condition that the line sets no operator, wrapped to the line
// width of the generated file.
func (l lineData) NoOperator() string {
	const width, tab = 88, 4
	var b strings.Builder
	col := 2*tab + len("if ")
	for i, op := range l.Operators {
		term := "l." + op + " == nil"
		end := " &&"
		if i == len(l.Operators)-1 {
			end = " {"
		}
		if i > 0 && col+len(" && ")+len(term)+len(end) > width {
			b.WriteString(" &&\n\t\t\t")
			col = 3 * tab
		} else if i > 0 {
			b.WriteString(" && ")
			col += len(" && ")
		}
		b.WriteString(term)
		col += len(term)
	}
	return b.String()
}

type comparisonData struct{ Member, Op string }

// Comparisons returns the ordering members the line compares with a closure.
func (l lineData) Comparisons() []comparisonData {
	var out []comparisonData
	for _, m := range []string{"gt", "gte", "lt", "lte"} {
		if l.Has(m) {
			out = append(out, comparisonData{
				Member: naming.ToPascalCase(m),
				Op:     comparisons[m],
			})
		}
	}
	return out
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
	if !plugindomain.HasDomainFromType(target, "retrieve") {
		return fail(
			"the target %s has no @retrieve, so it has no Retrieve or Filter to build on",
			target.QualifiedName,
		)
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
	funcs := set.New(lo.Map(matchers, func(m matcher.Matcher, _ int) string {
		return m.Func
	})...).Add("MatchKeys", "MatchWhere")
	if hasSearch(target, table) {
		funcs.Add("MatchSearch")
	}

	data := templateData{
		Manager: imports.NewManager(),
		Package: naming.DerivePackageName(outputPath),
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
		if line.HasType() {
			ld.Typed = true
			ld.Func = "Match" + ld.GoName
			if funcs.Contains(ld.Func) {
				return fail(
					"typed line %s calls %s, which the generated code already "+
						"declares; rename the line",
					line.Name,
					ld.Func,
				)
			}
			ld.Variadic = arrays.IsArray(field.Type, table) &&
				arrays.Size(field.Type, table) == nil
			ld.Pointer = !typemap.Nilable(field.Type, table)
			data.Lines = append(data.Lines, ld)
			continue
		}
		op, ok := table.Get(field.Type.Name)
		if !ok {
			return fail(
				"line %s has the unknown operator %s",
				line.Name,
				field.Type.Name,
			)
		}
		opForm, ok := op.Form.(resolution.StructForm)
		if !ok || len(field.Type.TypeArgs) > 1 {
			return fail("line %s: the operator %s is not a struct", line.Name, op.Name)
		}
		known, ok := resolution.OperatorMembers(op.QualifiedName)
		if !ok {
			return fail(
				"line %s: go/filter has no interpreter for the operator %s",
				line.Name,
				op.QualifiedName,
			)
		}
		ld.Members = lo.Map(opForm.Fields, func(f resolution.Field, _ int) string {
			return f.Name
		})
		ld.Operators = lo.FilterMap(
			opForm.Fields,
			func(f resolution.Field, _ int) (string, bool) {
				return naming.ToPascalCase(f.Name), f.Optional
			},
		)
		for _, m := range ld.Members {
			if !lo.Contains(known, m) {
				return fail(
					"line %s: go/filter cannot interpret the member %s of %s",
					line.Name,
					m,
					op.Name,
				)
			}
		}
		ld.Text = op.QualifiedName == resolution.OperatorString
		ld.Enum = len(field.Type.TypeArgs) == 1 &&
			isStringEnum(field.Type.TypeArgs[0], table)
		// The matcher list leaves out the key, which the service's MatchKeys matches.
		m, ok := byField[line.Name]
		if !ok {
			ld.Func = "MatchKeys"
			data.Lines = append(data.Lines, ld)
			continue
		}
		ld.Func = m.Func
		ld.Scalar = m.Kind.Scalar()
		ld.Field = m.Access
		data.Lines = append(data.Lines, ld)
	}

	addImports(data, req.RepoRoot)
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type requestData struct {
	*imports.Manager
	Package string
	// Service qualifies the identifiers of the filter's package, such as "rack.", and
	// is empty when the request is in the same package.
	Service string
	// Args passes any for each type parameter of the target, since a request selects
	// records of every type argument.
	Args string
	// Search is true when the request has a search_term.
	Search bool
}

func generateRequestFile(
	where, request resolution.Type,
	wherePath, requestPath string,
	req *plugin.Request,
) ([]byte, error) {
	target, _, ok := resolution.Dealias(where.Filter.Target, req.Resolutions)
	if !ok {
		return nil, errors.Newf(
			"filter %s: unknown target %s",
			where.QualifiedName,
			where.Filter.Target.Name,
		)
	}
	pkg := naming.DerivePackageName(requestPath)
	data := requestData{Manager: imports.NewManager(), Package: pkg}
	if requestPath != wherePath {
		alias := naming.DerivePackageAlias(wherePath, pkg)
		data.AddInternal(
			alias,
			gomod.ResolveImportPath(wherePath, req.RepoRoot, gomod.DefaultModulePrefix),
		)
		data.Service = alias + "."
	}
	if n := len(resolution.NonDefaultedTypeParams(
		resolution.TypeParams(target.Form),
	)); n > 0 {
		data.Args = "[" + strings.Join(slices.Repeat([]string{"any"}, n), ", ") + "]"
	}
	_, data.Search = request.Form.(resolution.StructForm).Field("search_term")
	data.AddExternal("encoding/json/jsontext")
	data.AddExternal("encoding/json/v2")
	data.AddExternal("github.com/vmihailenco/msgpack/v5")
	data.AddInternal(
		"validate",
		gomod.ResolveImportPath(
			"x/go/validate",
			req.RepoRoot,
			gomod.DefaultModulePrefix,
		),
	)
	var buf bytes.Buffer
	if err := requestTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func hasSearch(target resolution.Type, table *resolution.Table) bool {
	fields, _ := resolution.RecordFields(target, table)
	return lo.SomeBy(fields, func(f resolution.Field) bool {
		return plugindomain.HasDomainFromField(f, "search")
	})
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
	data.AddExternal("strconv")
	internal("validate", "x/go/validate")
	hasText := lo.SomeBy(data.Lines, func(l lineData) bool { return l.Text })
	hasBare := lo.SomeBy(data.Lines, func(l lineData) bool { return !l.Typed })
	needsMatch := hasText || lo.SomeBy(data.Lines, func(l lineData) bool {
		return len(l.Comparisons()) > 0
	})
	if hasBare {
		internal("filter", "x/go/filter")
	}
	if lo.SomeBy(data.Lines, func(l lineData) bool { return l.Enum }) {
		internal("errors", "x/go/errors")
	}
	if needsMatch {
		internal("gorp", "gorp")
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
{{- if not .Typed}}
		if {{.NoOperator}} {
			return nil, validate.PathedError(filter.ErrNoOperator, "{{.Path}}")
		}
{{- end}}
{{- if .Typed}}
		fs = append(fs, {{.Func}}{{$.TypeArgs}}({{if .Pointer}}*{{end}}l{{if .Variadic}}...{{end}}))
{{- else if .Text}}
		text := *l
		if !l.CaseInsensitive {
			if l.Eq != nil {
				fs = append(fs, {{.Func}}{{$.TypeArgs}}(*l.Eq))
			}
{{- if not .Scalar}}
			if l.In != nil {
				fs = append(fs, {{.Func}}{{$.TypeArgs}}(l.In...))
			}
			text.In = nil
{{- end}}
			text.Eq = nil
		}
		if text.CaseInsensitive || text.In != nil || text.Contains != nil ||
			text.StartsWith != nil || text.Regex != nil {
			match, err := filter.CompileString(text)
			if err != nil {
				return nil, validate.PathedError(err, "{{.Path}}")
			}
			fs = append(fs, Match{{$.TypeArgs}}(func(
				_ gorp.Context,
				_ Retrieve{{$.TypeArgs}},
				e *{{$.Target}},
			) (bool, error) {
				return match(e.{{.Field}}), nil
			}))
		}
{{- else}}
{{- if .Has "eq"}}
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
{{- if .Has "in"}}
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
{{- if .Scalar}}
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
{{- if .Has "any"}}
		if l.Any != nil {
			anyOf := make([]Filter{{$.TypeArgs}}, len(l.Any))
			for i, v := range l.Any {
				anyOf[i] = {{.Func}}{{$.TypeArgs}}(v)
			}
			fs = append(fs, Or{{$.TypeArgs}}(anyOf...))
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
`),
)

var requestTemplate = template.Must(
	template.New("go-filter-request").Parse(`// Code generated by oracle. DO NOT EDIT.

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

// Apply narrows q to the records that the request selects. A key with no record is a
// non-match, as for every other line. It returns a validate.PathError for a value that
// no filter can hold.
func (r RetrieveRequest) Apply(q {{$.Service}}Retrieve{{$.Args}}) ({{$.Service}}Retrieve{{$.Args}}, error) {
	q = q.SkipMissingKeys()
	if r.Where != nil {
		f, err := {{$.Service}}MatchWhere{{$.Args}}(*r.Where)
		if err != nil {
			return {{$.Service}}Retrieve{{$.Args}}{}, validate.PathedError(err, "where")
		}
		q = q.Where(f)
	}
{{- if .Search}}
	if r.SearchTerm != "" {
		q = q.Where({{.Service}}MatchSearch(r.SearchTerm))
	}
{{- end}}
	if r.Limit != nil {
		// Gorp reads a zero limit as no limit, and an empty key set matches no record.
		if *r.Limit == 0 {
			return q.Where({{$.Service}}MatchKeys{{$.Args}}()), nil
		}
		q = q.Limit(int(*r.Limit))
	}
	return q.Offset(int(r.Offset)), nil
}

// UnmarshalJSONFrom decodes the request and rejects a member that it does not declare,
// at any depth, so a client that sends a field the Core removed fails.
func (r *RetrieveRequest) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	type plain RetrieveRequest
	return json.UnmarshalDecode(dec, (*plain)(r), json.RejectUnknownMembers(true))
}

// DecodeMsgpack decodes the request and rejects a member that it does not declare, at
// any depth, as UnmarshalJSONFrom does.
func (r *RetrieveRequest) DecodeMsgpack(dec *msgpack.Decoder) error {
	type plain RetrieveRequest
	dec.DisallowUnknownFields(true)
	// The rest of an enclosing value decodes without the flag.
	defer dec.DisallowUnknownFields(false)
	return dec.Decode((*plain)(r))
}
`),
)
