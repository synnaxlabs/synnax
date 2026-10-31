// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package builder generates the Python builder of a filter definition: a condition
// class, a line for each line of the filter, and_, or_, not_, and where. A client's
// retrieve takes the conditions that the lines make.
package builder

import (
	"bytes"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/paths"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/internal/arrays"
	"github.com/synnaxlabs/oracle/plugin/output"
	pytypes "github.com/synnaxlabs/oracle/plugin/py/types"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Options configures the py/builder plugin.
type Options struct {
	// FileNamePattern is the basename written for each output package.
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options { return Options{FileNamePattern: "builder_gen.py"} }

// Plugin generates the Python builder of each filter definition.
type Plugin struct {
	// Options sets the names of the generated files.
	Options Options
}

// New creates a new py/builder plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "py/builder" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"py"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"py/types"} }

// Generate emits a builder_gen.py next to each filter definition with a Python output.
// A filter in a version file gets none.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{}
	for _, typ := range req.Resolutions.StructTypes() {
		if typ.Filter == nil || output.IsOmitted(typ, "py") {
			continue
		}
		if _, _, ok := paths.VersionFile(typ.FilePath); ok {
			continue
		}
		outputPath := output.GetPath(typ, "py")
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

// classes maps each operator type to the x/py/x/filter class of its lines.
var classes = map[string]string{
	resolution.OperatorSet:      "SetLine",
	resolution.OperatorBool:     "BoolLine",
	resolution.OperatorOrdered:  "OrderedLine",
	resolution.OperatorContains: "ContainsLine",
	resolution.OperatorString:   "StringLine",
}

type templateData struct {
	Imports string
	// Where is the Python name of the filter.
	Where string
	Lines []lineData
}

type lineData struct {
	// Name is the line's attribute on Where and the name of its variable.
	Name string
	// Class is the x/py/x/filter class of the line.
	Class string
	// Value is the Python type of the line's values, or empty for a line of fixed type.
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
	form := where.Form.(resolution.StructForm)
	names := []string{"Condition", "and_", "or_", "not_", "where"}
	for _, line := range where.Filter.Lines {
		names = append(names, pytypes.FieldName(line.Name))
	}
	module := pytypes.NewModule(where.Namespace, outputPath, names...)
	data := templateData{
		Where: module.Type(resolution.TypeRef{Name: where.QualifiedName}, table),
	}
	for _, line := range where.Filter.Lines {
		field := lo.Must(form.Field(line.Name))
		ld := lineData{Name: pytypes.FieldName(line.Name), Doc: doc.Get(field.Domains)}
		elem, isArray := arrays.ElementType(field.Type, table)
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
			if ld.Class, ok = classes[op.QualifiedName]; !ok {
				return fail(
					"line %s: py/builder has no builder for the operator %s",
					line.Name,
					op.QualifiedName,
				)
			}
			if len(field.Type.TypeArgs) > 0 {
				ld.Value = module.Type(field.Type.TypeArgs[0], table)
			}
		case isArray:
			ld.Class = "ListLine"
			ld.Value = module.Type(elem, table)
		case field.Type.Name == "bool":
			ld.Class = "FlagLine"
		default:
			ld.Class = "ValueLine"
			ld.Value = module.Type(field.Type, table)
		}
		data.Lines = append(data.Lines, ld)
	}
	var err error
	if data.Imports, err = module.Imports(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var fileTemplate = template.Must(
	template.New("py-builder").
		Funcs(template.FuncMap{"formatDoc": doc.FormatPy}).
		Parse(`# Code generated by Oracle. DO NOT EDIT.

from __future__ import annotations

from x import filter
{{- .Imports }}


class Condition(filter.Condition):
    """A condition that a retrieve of this entity takes."""

{{ range .Lines }}
{{ .Name }}: filter.{{ .Class }}[Condition{{ if .Value }}, {{ .Value }}{{ end }}] = filter.{{ .Class }}(Condition, "{{ .Name }}")
{{- with formatDoc .Doc }}
{{ . }}
{{- end }}
{{- end }}

and_ = Condition.and_
"""Matches the records that every condition matches."""

or_ = Condition.or_
"""Matches the records that any condition matches."""

not_ = Condition.not_
"""Matches the records that the condition does not match."""


def where(tree: {{ .Where }}) -> Condition:
    """Turns a stored tree into a condition."""
    return Condition(tree)
`),
)
