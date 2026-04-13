// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package filter generates TypeScript Zod schemas for per-entity filter nodes.
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

// Plugin generates TypeScript filter Zod schemas.
type Plugin struct{ Options Options }

// Options configures the ts/filter plugin.
type Options struct {
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{FileNamePattern: "filter.gen.ts"}
}

// New creates a new ts/filter plugin.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "ts/filter" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"ts"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"ts/types"} }

// Check verifies generated files are up-to-date.
func (p *Plugin) Check(*plugin.Request) error { return nil }

var tsPostWriter = &exec.PostWriter{
	ConfigFile: "package.json",
	Commands: [][]string{
		{"npx", "prettier", "--write"},
		{"npx", "eslint", "--fix"},
	},
}

// PostWrite runs prettier and eslint on generated files.
func (p *Plugin) PostWrite(files []string) error {
	return tsPostWriter.PostWrite(files)
}

type filterFieldInfo struct {
	FieldName  string // camelCase wire name
	TSName     string // camelCase TS property name
	FilterKind string // "string", "bool", "numeric", "time"
	ZodType    string // e.g. "filter.stringFilterZ"
	TSValType  string // TS value type for builders, e.g. "string", "boolean", "number"
}

type orderFieldInfo struct {
	FieldName string
	TSName    string
	ZodType   string // e.g. "filter.stringOrderByZ"
}

type filterNodeInfo struct {
	EntityName    string // PascalCase entity name
	CamelName     string // camelCase entity name
	Fields        []filterFieldInfo
	OrderFields   []orderFieldInfo
	HasOrderBy    bool
}

type templateData struct {
	Nodes []filterNodeInfo
}

// Generate produces filter.gen.ts files for entities with @retrieve and @ts output.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	type entry struct {
		tsPath  string
		structs []resolution.Type
	}
	entries := make(map[string]*entry)

	for _, typ := range req.Resolutions.StructTypes() {
		if !plugindomain.HasDomainFromType(typ, "retrieve") {
			continue
		}
		if output.IsOmitted(typ, "ts") {
			continue
		}
		tsPath := output.GetPath(typ, "ts")
		if tsPath == "" {
			continue
		}
		// Only generate for entities that have @filter fields.
		if !hasFilterFields(typ, req.Resolutions) {
			continue
		}
		e, ok := entries[tsPath]
		if !ok {
			e = &entry{tsPath: tsPath}
			entries[tsPath] = e
		}
		e.structs = append(e.structs, typ)
	}

	for _, e := range entries {
		content, err := generateTSFilterFile(e.structs, req.Resolutions)
		if err != nil {
			return nil, fmt.Errorf("failed to generate TS filter for %s: %w", e.tsPath, err)
		}
		if content == nil {
			continue
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", e.tsPath, p.Options.FileNamePattern),
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

func generateTSFilterFile(
	structs []resolution.Type,
	table *resolution.Table,
) ([]byte, error) {
	var nodes []filterNodeInfo
	for _, typ := range structs {
		node := extractTSFilterNode(typ, table)
		if node != nil {
			nodes = append(nodes, *node)
		}
	}
	if len(nodes) == 0 {
		return nil, nil
	}

	data := &templateData{Nodes: nodes}
	var buf bytes.Buffer
	if err := tsFilterTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func extractTSFilterNode(typ resolution.Type, table *resolution.Table) *filterNodeInfo {
	if _, ok := typ.Form.(resolution.StructForm); !ok {
		return nil
	}

	entityName := toPascalCase(typ.Name)
	camelName := strings.ToLower(entityName[:1]) + entityName[1:]

	var fields []filterFieldInfo
	var orderFields []orderFieldInfo
	allFields := resolution.UnifiedFields(typ, table)
	for _, field := range allFields {
		hasFilter := plugindomain.HasDomainFromField(field, "filter")
		hasIndex := plugindomain.HasDomainFromField(field, "index")

		if hasFilter {
			primitive := key.ResolvePrimitive(field.Type, table)
			fi := classifyTSField(primitive, field.Name)
			fields = append(fields, fi)
		}

		if hasIndex && plugindomain.HasExprFromField(field, "index", "sorted") {
			primitive := key.ResolvePrimitive(field.Type, table)
			oi := classifyTSOrderField(primitive, field.Name)
			orderFields = append(orderFields, oi)
		}
	}

	if len(fields) == 0 && len(orderFields) == 0 {
		return nil
	}

	return &filterNodeInfo{
		EntityName:  entityName,
		CamelName:   camelName,
		Fields:      fields,
		OrderFields: orderFields,
		HasOrderBy:  len(orderFields) > 0,
	}
}

func classifyTSField(primitive string, fieldName string) filterFieldInfo {
	tsName := toCamelCase(fieldName)
	fi := filterFieldInfo{FieldName: fieldName, TSName: tsName}
	switch primitive {
	case "bool":
		fi.FilterKind = "bool"
		fi.ZodType = "filter.boolFilterZ"
		fi.TSValType = "boolean"
	case "string":
		fi.FilterKind = "string"
		fi.ZodType = "filter.stringFilterZ"
		fi.TSValType = "string"
	case "int8", "int16", "int32", "int64",
		"uint8", "uint12", "uint16", "uint20", "uint32", "uint64",
		"float32", "float64":
		fi.FilterKind = "numeric"
		fi.ZodType = "filter.numericFilterZ"
		fi.TSValType = "number"
	default:
		fi.FilterKind = "string"
		fi.ZodType = "filter.stringFilterZ"
		fi.TSValType = "string"
	}
	return fi
}

func classifyTSOrderField(primitive string, fieldName string) orderFieldInfo {
	tsName := toCamelCase(fieldName)
	oi := orderFieldInfo{FieldName: fieldName, TSName: tsName}
	switch primitive {
	case "string":
		oi.ZodType = "filter.stringOrderByZ"
	case "int64":
		oi.ZodType = "filter.timestampOrderByZ"
	default:
		oi.ZodType = "filter.numericOrderByZ"
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

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

var tsFilterTemplate = template.Must(template.New("ts-filter").Parse(`// Code generated by Oracle. DO NOT EDIT.

import { type filter } from "@synnaxlabs/x";
import { z } from "zod";
{{range $node := .Nodes}}
export const {{$node.CamelName}}FilterNodeZ: z.ZodType<{{$node.EntityName}}FilterNode> = z.lazy(() =>
  z.object({
{{- range $f := $node.Fields}}
    {{$f.TSName}}: z.union([z.{{$f.TSValType}}(), {{$f.ZodType}}]).optional(),
{{- end}}
    and: z.array({{$node.CamelName}}FilterNodeZ).optional(),
    or: z.array({{$node.CamelName}}FilterNodeZ).optional(),
    not: {{$node.CamelName}}FilterNodeZ.optional(),
  }),
);

export interface {{$node.EntityName}}FilterNode {
{{- range $f := $node.Fields}}
  {{$f.TSName}}?: {{$f.TSValType}} | filter.{{$f.FilterKind | toPascal}}Filter;
{{- end}}
  and?: {{$node.EntityName}}FilterNode[];
  or?: {{$node.EntityName}}FilterNode[];
  not?: {{$node.EntityName}}FilterNode;
}
{{- if $node.HasOrderBy}}

export const {{$node.CamelName}}OrderByZ = z.object({
{{- range $o := $node.OrderFields}}
  {{$o.TSName}}: {{$o.ZodType}}.optional(),
{{- end}}
});
export interface {{$node.EntityName}}OrderBy extends z.infer<typeof {{$node.CamelName}}OrderByZ> {}
{{- end}}
{{range $f := $node.Fields}}
{{- if eq $f.FilterKind "string"}}
const {{$f.TSName}}Builder = {
  eq: (v: string): {{$node.EntityName}}FilterNode => ({ {{$f.TSName}}: { eq: v } }),
  in: (...vs: string[]): {{$node.EntityName}}FilterNode => ({ {{$f.TSName}}: { in: vs } }),
  contains: (v: string): {{$node.EntityName}}FilterNode => ({ {{$f.TSName}}: { contains: v } }),
  regex: (v: string): {{$node.EntityName}}FilterNode => ({ {{$f.TSName}}: { regex: v } }),
};
{{- else if eq $f.FilterKind "bool"}}
const {{$f.TSName}}Builder = {
  eq: (v: boolean): {{$node.EntityName}}FilterNode => ({ {{$f.TSName}}: { eq: v } }),
};
{{- else if eq $f.FilterKind "numeric"}}
const {{$f.TSName}}Builder = {
  eq: (v: number): {{$node.EntityName}}FilterNode => ({ {{$f.TSName}}: { eq: v } }),
  in: (...vs: number[]): {{$node.EntityName}}FilterNode => ({ {{$f.TSName}}: { in: vs } }),
  gt: (v: number): {{$node.EntityName}}FilterNode => ({ {{$f.TSName}}: { gt: v } }),
  lt: (v: number): {{$node.EntityName}}FilterNode => ({ {{$f.TSName}}: { lt: v } }),
  gte: (v: number): {{$node.EntityName}}FilterNode => ({ {{$f.TSName}}: { gte: v } }),
  lte: (v: number): {{$node.EntityName}}FilterNode => ({ {{$f.TSName}}: { lte: v } }),
};
{{- end}}
{{end}}
export const {{$node.CamelName}}Filter = {
{{- range $f := $node.Fields}}
  {{$f.TSName}}: {{$f.TSName}}Builder,
{{- end}}
  and: (...nodes: {{$node.EntityName}}FilterNode[]): {{$node.EntityName}}FilterNode => ({ and: nodes }),
  or: (...nodes: {{$node.EntityName}}FilterNode[]): {{$node.EntityName}}FilterNode => ({ or: nodes }),
  not: (node: {{$node.EntityName}}FilterNode): {{$node.EntityName}}FilterNode => ({ not: node }),
};
{{end}}`))
