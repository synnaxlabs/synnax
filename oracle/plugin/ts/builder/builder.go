// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package builder generates the TypeScript builder of a filter definition: a condition
// function for each line, and, or, not, where, and an option for each field of the
// derived retrieve request. A client's retrieve takes what the builder makes.
package builder

import (
	"bytes"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/internal/arrays"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/ts/internal/imports"
	tspaths "github.com/synnaxlabs/oracle/plugin/ts/internal/paths"
	tstypes "github.com/synnaxlabs/oracle/plugin/ts/types"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// Options configures the ts/builder plugin.
type Options struct {
	// FileNamePattern is the basename written for each output package.
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options { return Options{FileNamePattern: "builder.gen.ts"} }

// Plugin generates the TypeScript builder of each filter definition.
type Plugin struct {
	// Options sets the names of the generated files.
	Options Options
}

// New creates a new ts/builder plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "ts/builder" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"ts"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"ts/types"} }

// Generate emits a builder.gen.ts next to each filter definition with a TypeScript
// output. A filter in a version file gets none.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{}
	for _, typ := range req.Resolutions.StructTypes() {
		if typ.Filter == nil || output.IsOmitted(typ, "ts") {
			continue
		}
		if _, _, ok := paths.VersionFile(typ.FilePath); ok {
			continue
		}
		outputPath := output.GetPath(typ, "ts")
		if outputPath == "" {
			continue
		}
		content, err := generateFile(typ, outputPath, req.Resolutions)
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

// kinds maps each operator type to the x/ts/src/filter builder function of its lines.
var kinds = map[string]string{
	resolution.OperatorSet:      "set",
	resolution.OperatorBool:     "bool",
	resolution.OperatorOrdered:  "ordered",
	resolution.OperatorContains: "contains",
	resolution.OperatorString:   "string",
}

// reserved holds the JavaScript reserved words and the names strict mode forbids as
// bindings, which a line exports under an alias.
var reserved = set.New(
	"arguments", "eval",
	"await", "break", "case", "catch", "class", "const", "continue", "debugger",
	"default", "delete", "do", "else", "enum", "export", "extends", "false",
	"finally", "for", "function", "if", "implements", "import", "in", "instanceof",
	"interface", "let", "new", "null", "package", "private", "protected", "public",
	"return", "static", "super", "switch", "this", "throw", "true", "try", "typeof",
	"var", "void", "while", "with", "yield",
)

type templateData struct {
	*imports.Manager
	// Entity tags the conditions, so that a retrieve rejects another entity's.
	Entity string
	// Where is the TypeScript name of the filter.
	Where   string
	Lines   []lineData
	Options []optionData
}

type lineData struct {
	// Name is the line's field on Where and the name of its export.
	Name string
	// Kind is the x/ts/src/filter builder function that makes the line.
	Kind string
	Doc  string
	// Reserved is true when Name is a JavaScript reserved word.
	Reserved bool
}

type optionData struct {
	Name string
	// Value is the expression that makes the option.
	Value string
	Doc   string
}

func generateFile(
	where resolution.Type,
	outputPath string,
	table *resolution.Table,
) ([]byte, error) {
	fail := func(format string, args ...any) ([]byte, error) {
		return nil, errors.Newf(
			"filter %s: "+format,
			append([]any{where.QualifiedName}, args...)...,
		)
	}
	data := templateData{
		Manager: imports.NewManager(),
		Entity:  where.QualifiedName,
		Where:   domain.GetName(where, "ts"),
	}
	form := where.Form.(resolution.StructForm)
	for _, line := range where.Filter.Lines {
		field := lo.Must(form.Field(line.Name))
		ld := lineData{Name: tstypes.FieldName(line.Name), Doc: doc.Get(field.Domains)}
		ld.Reserved = reserved.Contains(ld.Name)
		switch {
		case !line.HasType():
			op, ok := table.Get(field.Type.Name)
			if !ok {
				return fail(
					"line %s has the unknown operator %s",
					line.Name,
					field.Type.Name,
				)
			}
			if ld.Kind, ok = kinds[op.QualifiedName]; !ok {
				return fail(
					"line %s: ts/builder has no builder for the operator %s",
					line.Name,
					op.QualifiedName,
				)
			}
		case arrays.IsArray(field.Type, table):
			ld.Kind = "list"
		case field.Type.Name == "bool":
			ld.Kind = "flag"
		default:
			ld.Kind = "value"
		}
		data.Lines = append(data.Lines, ld)
	}
	if where.Filter.Request != "" {
		request, ok := table.Get(where.Filter.Request)
		if !ok {
			return fail("unknown retrieve request %s", where.Filter.Request)
		}
		for _, f := range request.Form.(resolution.StructForm).Fields {
			od := optionData{Name: tstypes.FieldName(f.Name), Doc: doc.Get(f.Domains)}
			switch {
			case f.Name == "where":
				continue
			case f.Name == "limit" || f.Name == "offset":
				od.Value = "filter." + od.Name
			case f.Name == "search_term":
				od.Name = "search"
				od.Value = "filter.search"
			case strings.HasPrefix(f.Name, "include_"):
				od.Value = `filter.include("` + od.Name + `")`
			default:
				return fail("the request field %s has no builder option", f.Name)
			}
			data.Options = append(data.Options, od)
		}
	}
	data.AddImport(tspaths.CalculateImport(outputPath, "x/ts/src/filter"), "filter")
	data.AddImport(
		tspaths.CalculateImport(outputPath, outputPath)+"/types.gen",
		data.Where,
	)
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var fileTemplate = template.Must(
	template.New("ts-builder").
		Funcs(template.FuncMap{"formatDoc": doc.FormatTS}).
		Parse(`// Code generated by Oracle. DO NOT EDIT.
{{- range .SynnaxImports }}
import { {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} } from "{{ .Path }}";
{{- end }}
{{- range .InternalNamedImports }}
import { {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} } from "{{ .Path }}";
{{- end }}

const builder = filter.builder<"{{ .Entity }}", {{ .Where }}>();

/** Condition is a condition that a retrieve of this entity takes. */
export type Condition = filter.Condition<"{{ .Entity }}", {{ .Where }}>;
{{- range .Lines }}

{{ formatDoc .Name .Doc }}
{{- if .Reserved }}
const {{ .Name }}_ = builder.{{ .Kind }}("{{ .Name }}");
export { {{ .Name }}_ as {{ .Name }} };
{{- else }}
export const {{ .Name }} = builder.{{ .Kind }}("{{ .Name }}");
{{- end }}
{{- end }}

/** and matches the records that every condition matches. */
export const and = builder.and;

/** or matches the records that any condition matches. */
export const or = builder.or;

/** not matches the records that the condition does not match. */
export const not = builder.not;

/** where turns a stored tree into a condition. */
export const where = builder.where;
{{- range .Options }}

{{ formatDoc .Name .Doc }}
export const {{ .Name }} = {{ .Value }};
{{- end }}

/**
 * one makes a retrieve return one record. It goes after every other part. The retrieve
 * throws if no record or more than one record matches.
 */
export const one = filter.one;
`),
)
