// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package filter generates TypeScript composable filter APIs for each
// @retrieve entity. Output covers the wire schema, an opinionated typed
// surface for client retrieve calls, and a per-entity descriptor consumed
// by the shared filter runtime in @synnaxlabs/x.
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
	"github.com/synnaxlabs/oracle/plugin/ts/internal/paths"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/pluralize"
)

type Plugin struct{ Options Options }

type Options struct {
	FileNamePattern string
}

func DefaultOptions() Options {
	return Options{FileNamePattern: "filter.gen.ts"}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string { return "ts/filter" }

func (p *Plugin) Domains() []string { return []string{"ts"} }

func (p *Plugin) Requires() []string { return []string{"ts/types"} }

func (p *Plugin) Check(*plugin.Request) error { return nil }

var tsPostWriter = &exec.PostWriter{
	ConfigFile: "package.json",
	Commands: [][]string{
		{"npx", "prettier", "--write"},
		{"npx", "eslint", "--fix"},
	},
}

func (p *Plugin) PostWrite(files []string) error {
	return tsPostWriter.PostWrite(files)
}

// filterFieldInfo captures everything the template needs to emit a single
// filterable field across its wire schema, user-facing typed slot, and
// descriptor entry.
type filterFieldInfo struct {
	WireName   string // snake_case name on the wire
	TSName     string // camelCase TS property name
	FilterKind string // "string" | "bool" | "numeric"
	WireZod    string // e.g. "filter.stringFilterZ"
	WireType   string // e.g. "filter.StringFilter"
	ValueType  string // user-facing value type, e.g. "string" or "number"
	APIType    string // full user-facing union, e.g. "string | string[] | RegExp | filter.OpNode<string>"
}

type orderFieldInfo struct {
	WireName  string
	TSName    string
	WireZod   string // e.g. "filter.stringOrderByZ"
	WireType  string // e.g. "filter.StringOrderBy"
	CursorType string // user-facing cursor type, e.g. "string"
}

// identifyingFieldInfo describes one field that uniquely identifies an entity
// for single-fetch retrieve overloads. Each one produces a dedicated
// RetrieveBy<PascalName> interface in the generated output.
type identifyingFieldInfo struct {
	WireName   string // camelCase property name on the wire/request, e.g. "key"
	PascalName string // PascalCase suffix for the interface name, e.g. "Key"
	TSType     string // TypeScript value type, e.g. "Key" (when IsKey) or "string"
	IsKey      bool   // marks the @key field; controls whether to import the Key alias
}

type filterNodeInfo struct {
	EntityName        string // PascalCase entity, e.g. "Rack"
	CamelName         string // camelCase entity, e.g. "rack"
	WireTag           string // string literal used as the Node brand, e.g. "rack"
	UpperSnake        string // SCREAMING_SNAKE entity, e.g. "RACK"
	PayloadZodName    string // exported payload Zod schema name, e.g. "payloadZ" or "userZ"
	ItemsKey          string // plural lowercase items key on the response, e.g. "racks"
	RetrievePath      string // wire endpoint path, e.g. "/rack/retrieve"
	IdentifyingFields []identifyingFieldInfo
	HasSearch         bool // entity has @search directive → searchTerm option
	HasStatus         bool // entity has a status field → includeStatus option
	EmitRetrieve      bool // emit retrieveReqZ/retrieveResZ (non-generic entities only)
	HasKeyIdentifier  bool // any identifying field is the @key field; gates importing Key
	Fields            []filterFieldInfo
	OrderFields       []orderFieldInfo
	HasOrderBy        bool
}

type templateData struct {
	TypesImport  string
	TypesImports []string // import items pulled from TypesImport (e.g. "payloadZ", "type Key")
	Nodes        []filterNodeInfo
}

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
		filterPath := fmt.Sprintf("%s/%s", e.tsPath, p.Options.FileNamePattern)
		typesImport := paths.CalculateImport(filterPath, e.tsPath+"/types.gen")
		content, err := generateTSFilterFile(e.structs, req.Resolutions, typesImport)
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
	for _, field := range resolution.UnifiedFields(typ, table) {
		if plugindomain.HasDomainFromField(field, "filter") {
			return true
		}
	}
	if d, ok := typ.Domains["filter"]; ok && len(d.Expressions) > 0 {
		return true
	}
	return false
}

func virtualFilters(typ resolution.Type) []filterFieldInfo {
	d, ok := typ.Domains["filter"]
	if !ok {
		return nil
	}
	var out []filterFieldInfo
	for _, expr := range d.Expressions {
		if len(expr.Values) == 0 {
			continue
		}
		primitive := expr.Values[0].IdentValue
		out = append(out, classifyFilterField(primitive, expr.Name))
	}
	return out
}

func generateTSFilterFile(
	structs []resolution.Type,
	table *resolution.Table,
	typesImport string,
) ([]byte, error) {
	var nodes []filterNodeInfo
	for _, typ := range structs {
		node := extractFilterNode(typ, table)
		if node != nil {
			nodes = append(nodes, *node)
		}
	}
	if len(nodes) == 0 {
		return nil, nil
	}

	seen := make(map[string]struct{})
	var imports []string
	add := func(item string) {
		if _, ok := seen[item]; ok {
			return
		}
		seen[item] = struct{}{}
		imports = append(imports, item)
	}
	for _, n := range nodes {
		if !n.EmitRetrieve {
			continue
		}
		add(n.PayloadZodName)
		if n.HasKeyIdentifier {
			add("type Key")
		}
	}

	data := &templateData{
		TypesImport:  typesImport,
		TypesImports: imports,
		Nodes:        nodes,
	}
	var buf bytes.Buffer
	if err := tsFilterTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func extractFilterNode(typ resolution.Type, table *resolution.Table) *filterNodeInfo {
	form, ok := typ.Form.(resolution.StructForm)
	if !ok {
		return nil
	}

	entityName := toPascalCase(typ.Name)
	camelName := strings.ToLower(entityName[:1]) + entityName[1:]

	tsName := typ.Name
	if d, ok := typ.Domains["ts"]; ok {
		for _, expr := range d.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				tsName = expr.Values[0].StringValue
			}
		}
	}
	payloadZod := strings.ToLower(tsName[:1]) + tsName[1:] + "Z"
	emitRetrieve := !form.IsGeneric()

	var fields []filterFieldInfo
	var orderFields []orderFieldInfo
	var identifying []identifyingFieldInfo
	hasStatus := false
	hasKeyIdentifier := false
	for _, field := range resolution.UnifiedFields(typ, table) {
		hasFilter := plugindomain.HasDomainFromField(field, "filter")
		hasIndex := plugindomain.HasDomainFromField(field, "index")
		isKey := plugindomain.HasDomainFromField(field, "key")
		isUnique := plugindomain.HasExprFromField(field, "filter", "unique")
		if hasFilter {
			primitive := key.ResolvePrimitive(field.Type, table)
			fields = append(fields, classifyFilterField(primitive, field.Name))
		}
		if (isKey || isUnique) && hasFilter {
			primitive := key.ResolvePrimitive(field.Type, table)
			tsType := classifyFilterField(primitive, field.Name).ValueType
			if isKey {
				tsType = "Key"
				hasKeyIdentifier = true
			}
			identifying = append(identifying, identifyingFieldInfo{
				WireName:   toCamelCase(field.Name),
				PascalName: toPascalCase(field.Name),
				TSType:     tsType,
				IsKey:      isKey,
			})
		}
		if hasIndex && plugindomain.HasExprFromField(field, "index", "sorted") {
			primitive := key.ResolvePrimitive(field.Type, table)
			orderFields = append(orderFields, classifyOrderField(primitive, field.Name))
		}
		if field.Name == "status" {
			hasStatus = true
		}
	}

	fields = append(fields, virtualFilters(typ)...)

	if len(fields) == 0 && len(orderFields) == 0 {
		return nil
	}

	return &filterNodeInfo{
		EntityName:        entityName,
		CamelName:         camelName,
		WireTag:           camelName,
		UpperSnake:        strings.ToUpper(snakeCase(typ.Name)),
		PayloadZodName:    payloadZod,
		ItemsKey:          pluralize.String(camelName),
		RetrievePath:      "/" + camelName + "/retrieve",
		IdentifyingFields: identifying,
		HasSearch:         plugindomain.HasDomainFromType(typ, "search"),
		HasStatus:         hasStatus,
		EmitRetrieve:      emitRetrieve,
		HasKeyIdentifier:  hasKeyIdentifier,
		Fields:            fields,
		OrderFields:       orderFields,
		HasOrderBy:        len(orderFields) > 0,
	}
}

func snakeCase(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteRune('_')
		}
		if r >= 'A' && r <= 'Z' {
			b.WriteRune(r + ('a' - 'A'))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func classifyFilterField(primitive string, fieldName string) filterFieldInfo {
	tsName := toCamelCase(fieldName)
	fi := filterFieldInfo{WireName: fieldName, TSName: tsName}
	switch primitive {
	case "bool":
		fi.FilterKind = "bool"
		fi.WireZod = "filter.boolFilterZ"
		fi.WireType = "filter.BoolFilter"
		fi.ValueType = "boolean"
		fi.APIType = "boolean | filter.OpNode<boolean>"
	case "string":
		fi.FilterKind = "string"
		fi.WireZod = "filter.stringFilterZ"
		fi.WireType = "filter.StringFilter"
		fi.ValueType = "string"
		fi.APIType = "string | string[] | RegExp | filter.OpNode<string>"
	case "int8", "int16", "int32", "int64",
		"uint8", "uint12", "uint16", "uint20", "uint32", "uint64",
		"float32", "float64":
		fi.FilterKind = "numeric"
		fi.WireZod = "filter.numericFilterZ"
		fi.WireType = "filter.NumericFilter"
		fi.ValueType = "number"
		fi.APIType = "number | number[] | filter.OpNode<number>"
	default:
		fi.FilterKind = "string"
		fi.WireZod = "filter.stringFilterZ"
		fi.WireType = "filter.StringFilter"
		fi.ValueType = "string"
		fi.APIType = "string | string[] | RegExp | filter.OpNode<string>"
	}
	return fi
}

func classifyOrderField(primitive string, fieldName string) orderFieldInfo {
	tsName := toCamelCase(fieldName)
	oi := orderFieldInfo{WireName: fieldName, TSName: tsName}
	switch primitive {
	case "string":
		oi.WireZod = "filter.stringOrderByZ"
		oi.WireType = "filter.StringOrderBy"
		oi.CursorType = "string"
	case "int64":
		oi.WireZod = "filter.timestampOrderByZ"
		oi.WireType = "filter.TimestampOrderBy"
		oi.CursorType = "bigint"
	default:
		oi.WireZod = "filter.numericOrderByZ"
		oi.WireType = "filter.NumericOrderBy"
		oi.CursorType = "number"
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

var tsFilterTemplate = template.Must(template.New("ts-filter").Funcs(template.FuncMap{
	"toPascal": toPascalCase,
}).Parse(`// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Code generated by Oracle. DO NOT EDIT.

import { {{if .TypesImports}}array, {{end}}filter } from "@synnaxlabs/x";
import { z } from "zod";
{{if .TypesImports}}
import { {{range $i, $n := .TypesImports}}{{if $i}}, {{end}}{{$n}}{{end}} } from "{{.TypesImport}}";
{{end}}{{range $node := .Nodes}}
export interface {{$node.EntityName}}FilterNode {
{{- range $f := $node.Fields}}
  {{$f.TSName}}?: {{$f.WireType}};
{{- end}}
  and?: {{$node.EntityName}}FilterNode[];
  or?: {{$node.EntityName}}FilterNode[];
  not?: {{$node.EntityName}}FilterNode;
}

export const {{$node.CamelName}}FilterNodeZ: z.ZodType<{{$node.EntityName}}FilterNode> = z.lazy(() =>
  z.object({
{{- range $f := $node.Fields}}
    {{$f.TSName}}: {{$f.WireZod}}.optional(),
{{- end}}
    and: z.array({{$node.CamelName}}FilterNodeZ).optional(),
    or: z.array({{$node.CamelName}}FilterNodeZ).optional(),
    not: {{$node.CamelName}}FilterNodeZ.optional(),
  }),
);
{{if $node.HasOrderBy}}
export interface {{$node.EntityName}}OrderBy {
{{- range $o := $node.OrderFields}}
  {{$o.TSName}}?: {{$o.WireType}};
{{- end}}
}

export const {{$node.CamelName}}OrderByZ: z.ZodType<{{$node.EntityName}}OrderBy> = z.object({
{{- range $o := $node.OrderFields}}
  {{$o.TSName}}: {{$o.WireZod}}.optional(),
{{- end}}
});
{{end}}
export interface {{$node.EntityName}}Filter {
{{- range $f := $node.Fields}}
  {{$f.TSName}}?: {{$f.APIType}};
{{- end}}
}

export type {{$node.EntityName}}FilterArg =
  | {{$node.EntityName}}Filter
  | filter.Node<"{{$node.WireTag}}">;

export const {{$node.UpperSnake}}_FILTER_DESCRIPTOR: filter.Descriptor = {
  entity: "{{$node.WireTag}}",
  fields: {
{{- range $f := $node.Fields}}
    {{$f.TSName}}: "{{$f.FilterKind}}",
{{- end}}
  },
  orderFields: {
{{- range $o := $node.OrderFields}}
    {{$o.TSName}}: "string",
{{- end}}
  },
};

export const IDENTIFYING_FIELDS: ReadonlySet<string> = new Set([
{{- range $f := $node.IdentifyingFields}}
  "{{$f.WireName}}",
{{- end}}
]);
{{if $node.EmitRetrieve}}
export interface {{$node.EntityName}}RetrieveOptions {
  limit?: number;
  offset?: number;
{{- if $node.HasSearch}}
  searchTerm?: string;
{{- end}}
{{- if $node.HasStatus}}
  includeStatus?: boolean;
{{- end}}
}
{{range $f := $node.IdentifyingFields}}
export interface {{$node.EntityName}}RetrieveBy{{$f.PascalName}} {
  {{$f.WireName}}: {{$f.TSType}};
{{- if $node.HasStatus}}
  includeStatus?: boolean;
{{- end}}
}
{{end}}
export type {{$node.EntityName}}RetrieveArg =
  | ({{$node.EntityName}}Filter & {{$node.EntityName}}RetrieveOptions)
  | filter.Node<"{{$node.WireTag}}">;

export const retrieveReqZ = z.object({
  where: {{$node.CamelName}}FilterNodeZ.optional(),
{{- if $node.HasSearch}}
  searchTerm: z.string().optional(),
{{- end}}
  limit: z.int().optional(),
  offset: z.int().optional(),
{{- if $node.HasStatus}}
  includeStatus: z.boolean().optional(),
{{- end}}
});

export const retrieveResZ = z.object({
  {{$node.ItemsKey}}: array.nullishToEmpty({{$node.PayloadZodName}}),
});
{{end}}

type TwoOrMore = [
  {{$node.EntityName}}FilterArg,
  {{$node.EntityName}}FilterArg,
  ...{{$node.EntityName}}FilterArg[],
];

export const or = (...args: TwoOrMore): filter.Node<"{{$node.WireTag}}"> => ({
  kind: "or",
  children: args,
  [filter.NODE_TAG]: "{{$node.WireTag}}",
});

export const and = (...args: TwoOrMore): filter.Node<"{{$node.WireTag}}"> => ({
  kind: "and",
  children: args,
  [filter.NODE_TAG]: "{{$node.WireTag}}",
});

export const not = (
  arg: {{$node.EntityName}}FilterArg,
): filter.Node<"{{$node.WireTag}}"> => ({
  kind: "not",
  children: [arg],
  [filter.NODE_TAG]: "{{$node.WireTag}}",
});

export const eq = <T>(value: T): filter.OpNode<T, "eq"> => ({
  value,
  [filter.OP_TAG]: "eq",
});

export const gt = <T>(value: T): filter.OpNode<T, "gt"> => ({
  value,
  [filter.OP_TAG]: "gt",
});

export const lt = <T>(value: T): filter.OpNode<T, "lt"> => ({
  value,
  [filter.OP_TAG]: "lt",
});

export const gte = <T>(value: T): filter.OpNode<T, "gte"> => ({
  value,
  [filter.OP_TAG]: "gte",
});

export const lte = <T>(value: T): filter.OpNode<T, "lte"> => ({
  value,
  [filter.OP_TAG]: "lte",
});

export const between = <T>(
  lo: T,
  hi: T,
): filter.OpNode<T, "between"> => ({
  value: [lo, hi] as const,
  [filter.OP_TAG]: "between",
});
{{range $o := $node.OrderFields}}
export const orderBy{{$o.TSName | toPascal}} = (
  direction: "asc" | "desc",
  after?: {{$o.CursorType}},
): {{$node.EntityName}}OrderBy => ({ {{$o.TSName}}: { direction, after } });
{{end}}
export const toWire = (
  args: readonly filter.FilterArg[],
): filter.WireNode | undefined => filter.toWire(args, {{$node.UpperSnake}}_FILTER_DESCRIPTOR);
{{end}}`))
