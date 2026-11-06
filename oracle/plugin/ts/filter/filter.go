// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package filter generates the TypeScript matcher and input of a filter definition. The
// matcher keeps cached lists live: matches checks a record against a Where, and
// serverComputed tells whether only the Core can check one. The hand-written client
// passes in a Lines table with a function, or an explicit null, for each typed line.
// The input reads a partial record, in which a plain value or a list stands for an
// operator, into a Where, beside options that the client gives.
package filter

import (
	"bytes"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/ts/internal/imports"
	tspaths "github.com/synnaxlabs/oracle/plugin/ts/internal/paths"
	tstypes "github.com/synnaxlabs/oracle/plugin/ts/types"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Options configures the ts/filter plugin.
type Options struct {
	// FileNamePattern is the basename written for each output package.
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options { return Options{FileNamePattern: "where.gen.ts"} }

// Plugin generates the TypeScript matcher of each filter definition.
type Plugin struct {
	// Options sets the names of the generated files.
	Options Options
}

// New creates a new ts/filter plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "ts/filter" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"ts"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"ts/types"} }

// Generate emits a where.gen.ts next to each filter definition with a TypeScript
// output. A filter in a version file gets none.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	req = req.OmitFields("ts")
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

// checks maps each operator type to the x/ts/src/filter function that checks it.
var checks = map[string]string{
	resolution.OperatorSet:      "matchesSet",
	resolution.OperatorBool:     "matchesSet",
	resolution.OperatorOrdered:  "matchesOrdered",
	resolution.OperatorContains: "matchesContains",
	resolution.OperatorString:   "matchesString",
}

// inputs maps each operator type to the x/ts/src/filter schema that reads it from a
// plain value, a list, or the operator.
var inputs = map[string]string{
	resolution.OperatorSet:      "setInputZ",
	resolution.OperatorBool:     "boolInputZ",
	resolution.OperatorOrdered:  "orderedInputZ",
	resolution.OperatorContains: "containsInputZ",
	resolution.OperatorString:   "stringInputZ",
}

type templateData struct {
	*imports.Manager
	// Record is the TypeScript type of a cached record.
	Record string
	// Lines are the lines of the filter definition.
	Lines []lineData
	// ShortcutKey is the Zod schema of the key that shortcutZ reads beside a name, and
	// empty when the filter takes no name shortcut.
	ShortcutKey string
}

type lineData struct {
	// Name is the line's field on Where.
	Name string
	// Check is the x/ts/src/filter function of a bare line, and empty for a typed line.
	Check string
	// Field is the record field a bare line reads.
	Field string
	// Regex is true when the line's operator has a regex, which only the Core checks.
	Regex bool
	// Input is the Zod schema that reads the line from a partial record.
	Input string
	// Doc is the line's doc comment.
	Doc string
}

// HasTyped reports whether the filter has a typed line.
func (d templateData) HasTyped() bool {
	return lo.SomeBy(d.Lines, func(l lineData) bool { return l.Check == "" })
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
	// The cache holds the records the response returns, so bare lines read those.
	response, ok := table.Get(where.Filter.Response)
	if !ok {
		return fail("has no derived retrieve response")
	}
	elem := response.Form.(resolution.StructForm).Fields[0].Type.TypeArgs[0]
	record, _, ok := resolution.Dealias(elem, table)
	if !ok {
		return fail("unknown response type %s", elem.Name)
	}
	// The matcher names the record with no type arguments, as the client cache does.
	if tp, ok := lo.Find(
		resolution.TypeParams(record.Form),
		func(tp resolution.TypeParam) bool { return !tp.Optional && !tp.HasDefault() },
	); ok {
		return fail(
			"the type parameter %s of %s needs a default, since the matcher names %s "+
				"with no type arguments",
			tp.Name,
			record.Name,
			record.Name,
		)
	}
	fields, ok := resolution.RecordFields(record, table)
	if !ok {
		return fail("the response type %s must be a struct or a union", record.Name)
	}

	data := templateData{
		Manager: imports.NewManager(),
		Record:  domain.GetName(record, "ts"),
	}
	fp := &tstypes.FieldProcessor{
		Imports:    data.Manager,
		Namespace:  where.Namespace,
		OutputPath: outputPath,
		Request:    req,
	}
	// zodOf returns the Zod schema of a plain, required value of ref.
	zodOf := func(ref resolution.TypeRef) string {
		fp.CollectTypeImports(&ref)
		return fp.ProcessField(resolution.Field{Type: ref}, where).ZodType
	}
	form := where.Form.(resolution.StructForm)
	var key *resolution.TypeRef
	named := false
	for _, line := range where.Filter.Lines {
		field := lo.Must(form.Field(line.Name))
		ld := lineData{Name: tstypes.FieldName(line.Name), Doc: doc.Get(field.Domains)}
		if line.HasType() {
			ld.Input = zodOf(field.Type)
			if field.Type.Name == "Array" && field.Type.ArraySize == nil {
				ld.Input = "filter.listInputZ(" + zodOf(field.Type.TypeArgs[0]) + ")"
			}
		} else {
			recordField, ok := lo.Find(fields, func(f resolution.Field) bool {
				return f.Name == line.Name
			})
			if !ok {
				return fail(
					"the bare line %s names no field of %s, which the cache holds",
					line.Name,
					record.Name,
				)
			}
			opRef := field.Type
			op, ok := table.Get(opRef.Name)
			if !ok {
				return fail(
					"line %s has the unknown operator %s",
					line.Name,
					opRef.Name,
				)
			}
			opForm, ok := op.Form.(resolution.StructForm)
			if !ok {
				return fail(
					"line %s: the operator %s is not a struct",
					line.Name,
					op.Name,
				)
			}
			known, isOperator := resolution.OperatorMembers(op.QualifiedName)
			ld.Check, ok = checks[op.QualifiedName]
			if !ok || !isOperator {
				return fail(
					"line %s: ts/filter has no check for the operator %s",
					line.Name,
					op.QualifiedName,
				)
			}
			ms := lo.Map(opForm.Fields, func(f resolution.Field, _ int) string {
				return f.Name
			})
			for _, m := range ms {
				if !lo.Contains(known, m) {
					return fail(
						"line %s: ts/filter cannot check the member %s of %s",
						line.Name,
						m,
						op.Name,
					)
				}
			}
			ld.Field = tstypes.FieldName(recordField.Name)
			ld.Regex = lo.Contains(ms, "regex")
			ld.Input = "filter." + inputs[op.QualifiedName]
			if len(opRef.TypeArgs) == 1 {
				ld.Input += "(" + zodOf(opRef.TypeArgs[0]) + ")"
			}
			switch {
			case line.Name == "key" && len(opRef.TypeArgs) == 1:
				key = &opRef.TypeArgs[0]
			case line.Name == "name" && op.QualifiedName == resolution.OperatorString:
				named = true
			}
		}
		data.Lines = append(data.Lines, ld)
	}
	// A name could look like a free-text key, so only a key of another type takes one.
	if key != nil && named && resolution.PrimitiveBase(*key, table) != "string" {
		data.ShortcutKey = zodOf(*key)
	}

	data.AddImport("zod", "z")
	data.AddImport(tspaths.CalculateImport(outputPath, "x/ts/src/filter"), "filter")
	data.AddImport(
		tspaths.CalculateImport(outputPath, outputPath)+"/types.gen",
		"Where",
	)
	data.AddImport(
		tspaths.CalculateImport(outputPath, output.GetPath(record, "ts"))+"/types.gen",
		data.Record,
	)
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var fileTemplate = template.Must(
	template.New("ts-filter").
		Funcs(template.FuncMap{"formatDoc": doc.FormatTS}).
		Parse(`// Code generated by Oracle. DO NOT EDIT.
{{- range .SynnaxImports }}
import {{ if .Namespace }}* as {{ .Namespace }}{{ else }}{ {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} }{{ end }} from "{{ .Path }}";
{{- end }}
{{- range .ExternalNamedImports }}
import {{ if .Namespace }}* as {{ .Namespace }}{{ else }}{ {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} }{{ end }} from "{{ .Path }}";
{{- end }}
{{- range .InternalNamedImports }}
import {{ if .Namespace }}* as {{ .Namespace }}{{ else }}{ {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} }{{ end }} from "{{ .Path }}";
{{- end }}

{{- $lines := "" }}
{{- if .HasTyped }}
{{- $lines = "lines, " }}

/**
 * Lines checks the typed lines of a Where against a record. A null entry marks a line
 * that only the Core can check.
 */
export interface Lines {
{{- range .Lines }}
{{- if not .Check }}
  {{ .Name }}: ((record: {{ $.Record }}, value: NonNullable<Where["{{ .Name }}"]>) => boolean) | null;
{{- end }}
{{- end }}
}
{{- end }}

/**
 * @returns whether the record matches where. An absent where matches every record.
{{- if .HasTyped }}
 * @throws {Error} if where is server-computed, since only the Core can check it.
{{- end }}
 */
export const matches = ({{ if .HasTyped }}lines: Lines, {{ end }}record: {{ .Record }}, where?: Where): boolean => {
  if (where == null) return true;
  if (where.and != null && !where.and.every((w) => matches({{ $lines }}record, w))) return false;
  if (where.or != null && !where.or.some((w) => matches({{ $lines }}record, w))) return false;
  if (where.not != null && matches({{ $lines }}record, where.not)) return false;
{{- range .Lines }}
{{- if .Check }}
  if (where.{{ .Name }} != null && !filter.{{ .Check }}(where.{{ .Name }}, record.{{ .Field }})) return false;
{{- else }}
  if (where.{{ .Name }} != null) {
    if (lines.{{ .Name }} == null) throw new Error("only the Core can check {{ .Name }}");
    if (!lines.{{ .Name }}(record, where.{{ .Name }})) return false;
  }
{{- end }}
{{- end }}
  return true;
};

/**
{{- if .HasTyped }}
 * @returns whether only the Core can check where, because it holds a regex or a line
 * whose entry in lines is null. An absent where is not server-computed.
{{- else }}
 * @returns whether only the Core can check where, because it holds a regex. An absent
 * where is not server-computed.
{{- end }}
 */
export const serverComputed = ({{ if .HasTyped }}lines: Lines, {{ end }}where?: Where): boolean =>
  where != null && (
  where.and?.some((w) => serverComputed({{ $lines }}w)) === true ||
  where.or?.some((w) => serverComputed({{ $lines }}w)) === true ||
  (where.not != null && serverComputed({{ $lines }}where.not))
{{- range .Lines }}
{{- if .Regex }} ||
  where.{{ .Name }}?.regex != null
{{- else if not .Check }} ||
  (where.{{ .Name }} != null && lines.{{ .Name }} == null)
{{- end }}
{{- end }});

const inputs = {
{{- range .Lines }}
{{- if .Doc }}
  {{ formatDoc .Name .Doc 2 }}
{{- end }}
  {{ .Name }}: {{ .Input }}.exactOptional(),
{{- end }}
};

/** WhereInput is a Where that takes a value or a list in place of an operator. */
export type WhereInput = filter.Input<typeof inputs> & {
  and?: WhereInput[];
  or?: WhereInput[];
  not?: WhereInput;
};

const whereInputZ: z.ZodType<Where, WhereInput> = z.strictObject({
  get and() {
    return whereInputZ.array().exactOptional();
  },
  get or() {
    return whereInputZ.array().exactOptional();
  },
  get not() {
    return whereInputZ.exactOptional();
  },
  ...inputs,
});

/**
 * retrieveInputZ reads the fields of a WhereInput and the given options, side by side,
 * into a request whose where holds the fields. A field set to undefined throws, since
 * dropping it would select every record.
 */
export const retrieveInputZ = <O extends z.core.$ZodShape>(options: O) =>
  z
    .strictObject({
      and: whereInputZ.array().exactOptional(),
      or: whereInputZ.array().exactOptional(),
      not: whereInputZ.exactOptional(),
      ...inputs,
      ...options,
    })
    .transform((input) =>
      filter.split(input, ["and", "or", "not"{{ range .Lines }}, "{{ .Name }}"{{ end }}]),
    );
{{- with .ShortcutKey }}

/**
 * shortcutZ reads a key, a name, or a list of either into a request whose where holds
 * it. A list reads as keys only when every member is a key.
 */
export const shortcutZ = z.union([
  {{ . }}.transform((key) => ({ where: { key: { eq: key } } })),
  z.string().transform((name) => ({ where: { name: { eq: name } } })),
  {{ . }}.array().transform((keys) => ({ where: { key: { in: keys } } })),
  z.string().array().transform((names) => ({ where: { name: { in: names } } })),
]);
{{- end }}
`),
)
