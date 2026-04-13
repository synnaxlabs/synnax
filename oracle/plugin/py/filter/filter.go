// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package filter generates Python Pydantic models for per-entity filter nodes.
package filter

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	plugindomain "github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
)

// Plugin generates Python filter Pydantic models.
type Plugin struct{ Options Options }

// Options configures the py/filter plugin.
type Options struct {
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{FileNamePattern: "filter_gen.py"}
}

// New creates a new py/filter plugin.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "py/filter" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"py"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"py/types"} }

// Check verifies generated files are up-to-date.
func (p *Plugin) Check(*plugin.Request) error { return nil }

var pyPostWriter = &exec.PostWriter{
	ConfigFile: "pyproject.toml",
	Commands: [][]string{
		{"uv", "run", "ruff", "format"},
		{"uv", "run", "ruff", "check", "--fix"},
	},
}

// PostWrite runs ruff on generated files.
func (p *Plugin) PostWrite(files []string) error {
	return pyPostWriter.PostWrite(files)
}

type filterFieldInfo struct {
	FieldName  string // snake_case wire name
	PyName     string // snake_case Python attribute name
	FilterKind string // "string", "bool", "numeric", "time"
	PyType     string // e.g. "StringFilter"
}

type orderFieldInfo struct {
	FieldName string
	PyName    string
	PyType    string // e.g. "StringOrderBy"
}

type filterNodeInfo struct {
	EntityName  string // PascalCase class name
	Fields      []filterFieldInfo
	OrderFields []orderFieldInfo
	HasOrderBy  bool
}

type templateData struct {
	Nodes []filterNodeInfo
}

// Generate produces filter_gen.py files for entities with @retrieve and @py output.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	type entry struct {
		pyPath  string
		structs []resolution.Type
	}
	entries := make(map[string]*entry)

	for _, typ := range req.Resolutions.StructTypes() {
		if !plugindomain.HasDomainFromType(typ, "retrieve") {
			continue
		}
		if output.IsOmitted(typ, "py") {
			continue
		}
		pyPath := output.GetPath(typ, "py")
		if pyPath == "" {
			continue
		}
		if !hasFilterFields(typ, req.Resolutions) {
			continue
		}
		e, ok := entries[pyPath]
		if !ok {
			e = &entry{pyPath: pyPath}
			entries[pyPath] = e
		}
		e.structs = append(e.structs, typ)
	}

	for _, e := range entries {
		content, err := generatePyFilterFile(e.structs, req.Resolutions)
		if err != nil {
			return nil, fmt.Errorf("failed to generate Py filter for %s: %w", e.pyPath, err)
		}
		if content == nil {
			continue
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", e.pyPath, p.Options.FileNamePattern),
			Content: content,
		})
	}

	return resp, nil
}

func hasFilterFields(typ resolution.Type, table *resolution.Table) bool {
	allFields := resolution.UnifiedFields(typ, table)
	for _, field := range allFields {
		if plugindomain.HasDomainFromField(field, "filter") {
			return true
		}
	}
	return false
}

func generatePyFilterFile(
	structs []resolution.Type,
	table *resolution.Table,
) ([]byte, error) {
	var nodes []filterNodeInfo
	for _, typ := range structs {
		node := extractPyFilterNode(typ, table)
		if node != nil {
			nodes = append(nodes, *node)
		}
	}
	if len(nodes) == 0 {
		return nil, nil
	}

	data := &templateData{Nodes: nodes}
	var buf bytes.Buffer
	if err := pyFilterTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func extractPyFilterNode(typ resolution.Type, table *resolution.Table) *filterNodeInfo {
	if _, ok := typ.Form.(resolution.StructForm); !ok {
		return nil
	}

	entityName := toPascalCase(typ.Name)

	var fields []filterFieldInfo
	var orderFields []orderFieldInfo
	allFields := resolution.UnifiedFields(typ, table)
	for _, field := range allFields {
		hasFilter := plugindomain.HasDomainFromField(field, "filter")
		hasIndex := plugindomain.HasDomainFromField(field, "index")

		if hasFilter {
			primitive := key.ResolvePrimitive(field.Type, table)
			fi := classifyPyField(primitive, field.Name)
			fields = append(fields, fi)
		}

		if hasIndex && plugindomain.HasExprFromField(field, "index", "sorted") {
			primitive := key.ResolvePrimitive(field.Type, table)
			oi := classifyPyOrderField(primitive, field.Name)
			orderFields = append(orderFields, oi)
		}
	}

	if len(fields) == 0 && len(orderFields) == 0 {
		return nil
	}

	return &filterNodeInfo{
		EntityName:  entityName,
		Fields:      fields,
		OrderFields: orderFields,
		HasOrderBy:  len(orderFields) > 0,
	}
}

func classifyPyField(primitive string, fieldName string) filterFieldInfo {
	fi := filterFieldInfo{FieldName: fieldName, PyName: fieldName}
	switch primitive {
	case "bool":
		fi.FilterKind = "bool"
		fi.PyType = "BoolFilter"
	case "string":
		fi.FilterKind = "string"
		fi.PyType = "StringFilter"
	case "int8", "int16", "int32", "int64",
		"uint8", "uint12", "uint16", "uint20", "uint32", "uint64",
		"float32", "float64":
		fi.FilterKind = "numeric"
		fi.PyType = "NumericFilter"
	default:
		fi.FilterKind = "string"
		fi.PyType = "StringFilter"
	}
	return fi
}

func classifyPyOrderField(primitive string, fieldName string) orderFieldInfo {
	oi := orderFieldInfo{FieldName: fieldName, PyName: fieldName}
	switch primitive {
	case "string":
		oi.PyType = "StringOrderBy"
	case "int64":
		oi.PyType = "TimestampOrderBy"
	default:
		oi.PyType = "NumericOrderBy"
	}
	return oi
}

func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

var pyFuncs = template.FuncMap{
	"toCamelCase": func(s string) string {
		parts := strings.Split(s, "_")
		for i := 1; i < len(parts); i++ {
			if len(parts[i]) > 0 {
				parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
			}
		}
		return strings.Join(parts, "")
	},
}

var pyFilterTemplate = template.Must(template.New("py-filter").Funcs(pyFuncs).Parse(`# Code generated by Oracle. DO NOT EDIT.

from __future__ import annotations

from pydantic import BaseModel, Field

from synnax.filter import (
    BoolFilter,
    NumericFilter,
    StringFilter,
    TimeFilter,
    StringOrderBy,
    NumericOrderBy,
    TimestampOrderBy,
)
{{range $node := .Nodes}}

class {{$node.EntityName}}FilterNode(BaseModel):
{{- range $f := $node.Fields}}
    {{$f.PyName}}: {{$f.PyType}} | None = None
{{- end}}
    and_: list[{{$node.EntityName}}FilterNode] | None = Field(default=None, alias="and")
    or_: list[{{$node.EntityName}}FilterNode] | None = Field(default=None, alias="or")
    not_: {{$node.EntityName}}FilterNode | None = Field(default=None, alias="not")

    model_config = {"populate_by_name": True}
{{- if $node.HasOrderBy}}


class {{$node.EntityName}}OrderBy(BaseModel):
{{- range $o := $node.OrderFields}}
    {{$o.PyName}}: {{$o.PyType}} | None = None
{{- end}}
{{- end}}
{{end}}`))
