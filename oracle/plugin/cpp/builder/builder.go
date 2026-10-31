// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package builder generates the C++ builder of a filter definition: a line for each
// line of the filter, and_, or_, not_, where, and an option for each field of the
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
	"github.com/synnaxlabs/oracle/plugin/cpp/naming"
	cpptypes "github.com/synnaxlabs/oracle/plugin/cpp/types"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/internal/arrays"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Options configures the cpp/builder plugin.
type Options struct {
	// FileNamePattern is the basename written for each output package.
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options { return Options{FileNamePattern: "builder.gen.h"} }

// Plugin generates the C++ builder of each filter definition.
type Plugin struct {
	// Options sets the names of the generated files.
	Options Options
}

// New creates a new cpp/builder plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "cpp/builder" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"cpp"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"cpp/types", "cpp/json"} }

// Generate emits a builder.gen.h next to each filter definition with a C++ output. A
// filter in a version file gets none.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{}
	for _, typ := range req.Resolutions.StructTypes() {
		if typ.Filter == nil || output.IsOmitted(typ, "cpp") {
			continue
		}
		if _, _, ok := paths.VersionFile(typ.FilePath); ok {
			continue
		}
		outputPath := output.GetPath(typ, "cpp")
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

// lines maps each operator type to the x/cpp/filter line of its lines.
var lines = map[string]string{
	resolution.OperatorSet:      "SetLine",
	resolution.OperatorBool:     "BoolLine",
	resolution.OperatorOrdered:  "OrderedLine",
	resolution.OperatorContains: "ContainsLine",
	resolution.OperatorString:   "StringLine",
}

type templateData struct {
	// Include is the path of the JSON codec of the filter, which conditions use to
	// describe themselves in errors.
	Include   string
	Namespace string
	// Where is the C++ name of the filter.
	Where string
	// Request is the C++ name of the derived retrieve request, or empty when the filter
	// has none.
	Request  string
	Lines    []lineData
	Options  []string
	Includes []includeData
}

type lineData struct {
	// Name is the line's member of Where and the name of its variable.
	Name string
	// Line is the x/cpp/filter line type.
	Line string
	Doc  string
}

type includeData struct {
	// Name is the request flag that the option sets and the name of its function.
	Name string
	Doc  string
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
		Include:   outputPath + "/json.gen.h",
		Namespace: naming.Namespace(outputPath),
		Where:     domain.GetName(where, "cpp"),
	}
	form := where.Form.(resolution.StructForm)
	for _, line := range where.Filter.Lines {
		field := lo.Must(form.Field(line.Name))
		ld := lineData{Name: cpptypes.FieldName(field), Doc: doc.Get(field.Domains)}
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
			if ld.Line, ok = lines[op.QualifiedName]; !ok {
				return fail(
					"line %s: cpp/builder has no builder for the operator %s",
					line.Name,
					op.QualifiedName,
				)
			}
		case arrays.IsArray(field.Type, table):
			ld.Line = "ListLine"
		case field.Type.Name == "bool":
			ld.Line = "FlagLine"
		default:
			ld.Line = "ValueLine"
		}
		data.Lines = append(data.Lines, ld)
	}
	if where.Filter.Request != "" {
		request, ok := table.Get(where.Filter.Request)
		if !ok {
			return fail("unknown retrieve request %s", where.Filter.Request)
		}
		if output.GetPath(request, "cpp") != outputPath {
			return fail(
				"the retrieve request %s has another C++ output",
				request.QualifiedName,
			)
		}
		data.Request = domain.GetName(request, "cpp")
		for _, f := range request.Form.(resolution.StructForm).Fields {
			name := cpptypes.FieldName(f)
			switch {
			case f.Name == "where":
			case f.Name == "limit" || f.Name == "offset":
				data.Options = append(data.Options, name)
			case f.Name == "search_term":
				data.Options = append(data.Options, "search")
			case strings.HasPrefix(f.Name, "include_"):
				data.Includes = append(
					data.Includes,
					includeData{Name: name, Doc: doc.Get(f.Domains)},
				)
			default:
				return fail("the request field %s has no builder option", f.Name)
			}
		}
	}
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

var fileTemplate = template.Must(
	template.New("cpp-builder").
		Funcs(template.FuncMap{"formatDoc": doc.FormatCpp}).
		Parse(`// Code generated by oracle. DO NOT EDIT.

#pragma once

#include <utility>

#include "{{ .Include }}"
#include "x/cpp/filter/builder.h"

namespace {{ .Namespace }} {
/// @brief Condition is a condition that a retrieve of this entity takes.
using Condition = x::filter::Condition<{{ .Where }}>;
{{- range .Lines }}

{{ with formatDoc .Name .Doc }}{{ . }}
{{ end -}}
inline constexpr x::filter::{{ .Line }}<&{{ $.Where }}::{{ .Name }}> {{ .Name }}{};
{{- end }}

using x::filter::and_;
using x::filter::not_;
using x::filter::or_;

/// @brief where turns a stored filter into a condition.
inline Condition where({{ .Where }} tree) {
    return {std::move(tree)};
}
{{- if .Request }}
{{ range .Options }}
using x::filter::{{ . }};
{{- end }}
using x::filter::one;
{{- range .Includes }}

{{ with formatDoc .Name .Doc }}{{ . }}
{{ end -}}
inline x::filter::Include<&{{ $.Request }}::{{ .Name }}> {{ .Name }}() {
    return {};
}
{{- end }}
{{- end }}
}
`),
)
