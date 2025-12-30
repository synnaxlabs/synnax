// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
)

type Plugin struct{ Options Options }

type Options struct {
	OutputPath      string
	FileNamePattern string
	GenerateTypes   bool
}

func DefaultOptions() Options {
	return Options{
		OutputPath:      "{{.Namespace}}",
		FileNamePattern: "types.gen.ts",
		GenerateTypes:   true,
	}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string { return "ts/types" }

func (p *Plugin) Domains() []string { return nil }

func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) Check(req *plugin.Request) error { return nil }

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	outputStructs := make(map[string][]*resolution.StructEntry)
	for _, entry := range req.Resolutions.AllStructs() {
		if outputPath := output.GetPath(entry, "ts"); outputPath != "" {
			if req.RepoRoot != "" {
				if err := req.ValidateOutputPath(outputPath); err != nil {
					return nil, fmt.Errorf("invalid output path for struct %s: %w", entry.Name, err)
				}
			}
			outputStructs[outputPath] = append(outputStructs[outputPath], entry)
		}
	}
	for outputPath, structs := range outputStructs {
		enums := enum.CollectReferenced(structs)
		content, err := p.generateFile(structs[0].Namespace, outputPath, structs, enums, req)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s: %w", outputPath, err)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
			Content: content,
		})
	}
	return resp, nil
}

// calculateRelativeImport calculates the relative import path from one output
// directory to another, including the file pattern (e.g., "types.gen").
func calculateRelativeImport(from, to, filePattern string) string {
	importFile := strings.TrimSuffix(filePattern, filepath.Ext(filePattern))
	rel, err := filepath.Rel(from, to)
	if err != nil {
		return "./" + to + "/" + importFile
	}
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel + "/" + importFile
}

func (p *Plugin) generateFile(
	namespace string,
	outputPath string,
	structs []*resolution.StructEntry,
	enums []*resolution.EnumEntry,
	req *plugin.Request,
) ([]byte, error) {
	data := &templateData{
		Namespace:     namespace,
		OutputPath:    outputPath,
		Request:       req,
		IDFields:      make([]idFieldData, 0),
		Structs:       make([]structData, 0, len(structs)),
		Enums:         make([]enumData, 0, len(enums)),
		GenerateTypes: p.Options.GenerateTypes,
		Imports:       make(map[string]*importSpec),
	}
	idFields := collectIDFields(structs, data)
	data.IDFields = idFields
	data.Ontology = extractOntology(structs, idFields)
	if data.Ontology != nil {
		data.addNamedImport("@/ontology", "ontology")
	}

	for _, enum := range enums {
		data.Enums = append(data.Enums, p.processEnum(enum))
	}
	for _, entry := range structs {
		data.Structs = append(data.Structs, p.processStruct(entry, req.Resolutions, data, idFields))
	}
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func collectIDFields(structs []*resolution.StructEntry, data *templateData) []idFieldData {
	seen := make(map[string]bool)
	var result []idFieldData
	for _, s := range structs {
		for _, f := range s.Fields {
			if _, hasID := f.Domains["id"]; hasID {
				if !seen[f.Name] {
					seen[f.Name] = true
					result = append(result, idFieldData{
						Name:    f.Name,
						TSName:  lo.CamelCase(f.Name),
						ZodType: primitiveToZod(f.TypeRef.Primitive, data),
					})
				}
			}
		}
	}
	return result
}

// extractOntology extracts ontology data from structs if any has the ontology domain.
// Returns nil if no struct has an ontology domain or if no ID field is found.
func extractOntology(structs []*resolution.StructEntry, idFields []idFieldData) *ontologyData {
	for _, s := range structs {
		domain, ok := s.Domains["ontology"]
		if !ok {
			continue
		}

		// Find the type expression
		var typeName string
		for _, expr := range domain.Expressions {
			if expr.Name == "type" && len(expr.Values) > 0 {
				typeName = expr.Values[0].StringValue
				break
			}
		}
		if typeName == "" {
			continue
		}

		// Find the ID field's TypeScript type name (capitalized)
		if len(idFields) == 0 {
			continue
		}
		keyType := lo.Capitalize(lo.CamelCase(idFields[0].Name))

		return &ontologyData{
			TypeName: typeName,
			KeyType:  keyType,
		}
	}
	return nil
}

func (p *Plugin) processEnum(enum *resolution.EnumEntry) enumData {
	values := make([]enumValueData, 0, len(enum.Values))
	for _, v := range enum.Values {
		values = append(values, enumValueData{
			Name:      v.Name,
			Value:     v.StringValue,
			IntValue:  v.IntValue,
			IsIntEnum: enum.IsIntEnum,
		})
	}
	return enumData{Name: enum.Name, Values: values, IsIntEnum: enum.IsIntEnum}
}

func (p *Plugin) processStruct(entry *resolution.StructEntry, table *resolution.Table, data *templateData, idFields []idFieldData) structData {
	sd := structData{Name: entry.Name, Fields: make([]fieldData, 0, len(entry.Fields))}
	for _, field := range entry.Fields {
		sd.Fields = append(sd.Fields, p.processField(field, table, data, idFields))
	}
	return sd
}

func (p *Plugin) processField(field *resolution.FieldEntry, table *resolution.Table, data *templateData, idFields []idFieldData) fieldData {
	fd := fieldData{
		Name:       field.Name,
		TSName:     lo.CamelCase(field.Name),
		IsOptional: field.TypeRef.IsOptional,
		IsNullable: field.TypeRef.IsNullable,
		IsArray:    field.TypeRef.IsArray,
	}
	if idField := findIDField(field.Name, idFields); idField != nil {
		fd.ZodType = idField.TSName + "Z"
	} else {
		fd.ZodType = p.typeToZod(field.TypeRef, table, data)
		if validateDomain := plugin.GetFieldDomain(field, "validate"); validateDomain != nil {
			fd.ZodType = p.applyValidation(fd.ZodType, validateDomain, field.TypeRef)
		}
	}

	if field.TypeRef.IsArray {
		if field.TypeRef.IsNullable {
			// array.nullableZ transforms null/undefined to empty array
			data.addNamedImport(xImportPath, "array")
			fd.ZodType = fmt.Sprintf("array.nullableZ(%s)", fd.ZodType)
			// If also optional, add .optional() after
			if field.TypeRef.IsOptional {
				fd.ZodType = fd.ZodType + ".optional()"
			}
		} else {
			fd.ZodType = fmt.Sprintf("z.array(%s)", fd.ZodType)
			if field.TypeRef.IsOptional {
				fd.ZodType = fd.ZodType + ".optional()"
			}
		}
	} else {
		if field.TypeRef.IsOptional && field.TypeRef.IsNullable {
			fd.ZodType = fd.ZodType + ".nullish()"
		} else if field.TypeRef.IsNullable {
			fd.ZodType = fd.ZodType + ".nullable()"
		} else if field.TypeRef.IsOptional {
			fd.ZodType = fd.ZodType + ".optional()"
		}
	}
	return fd
}

func findIDField(name string, idFields []idFieldData) *idFieldData {
	for i := range idFields {
		if idFields[i].Name == name {
			return &idFields[i]
		}
	}
	return nil
}

func (p *Plugin) typeToZod(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData) string {
	switch typeRef.Kind {
	case resolution.TypeKindPrimitive:
		return primitiveToZod(typeRef.Primitive, data)
	case resolution.TypeKindStruct:
		if typeRef.StructRef == nil {
			return "z.unknown()"
		}
		schemaName := lo.CamelCase(typeRef.StructRef.Name) + "Z"
		if typeRef.StructRef.Namespace != data.Namespace {
			ns := typeRef.StructRef.Namespace
			// Get target output path from the referenced struct's domain
			targetOutputPath := output.GetPath(typeRef.StructRef, "ts")
			if targetOutputPath == "" {
				// Fallback if no output path defined
				targetOutputPath = ns
			}
			// Calculate relative import path from current output to target output
			relPath := calculateRelativeImport(data.OutputPath, targetOutputPath, p.Options.FileNamePattern)
			data.addNamespaceImport(relPath, ns)
			return fmt.Sprintf("%s.%s", ns, schemaName)
		}
		return schemaName
	case resolution.TypeKindEnum:
		if typeRef.EnumRef == nil {
			return "z.unknown()"
		}
		enumName := lo.CamelCase(typeRef.EnumRef.Name) + "Z"
		if typeRef.EnumRef.Namespace != data.Namespace {
			ns := typeRef.EnumRef.Namespace
			// For enums, look up the struct that references this enum to find its output path
			// Or use namespace as fallback
			targetOutputPath := enum.FindOutputPath(typeRef.EnumRef, table, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			relPath := calculateRelativeImport(data.OutputPath, targetOutputPath, p.Options.FileNamePattern)
			data.addNamespaceImport(relPath, ns)
			return fmt.Sprintf("%s.%s", ns, enumName)
		}
		return enumName
	default:
		return "z.unknown()"
	}
}

type primitiveMapping struct {
	schema   string
	xImports []string // imports needed from @synnaxlabs/x (e.g., "zod", "TimeStamp")
}

var primitiveZodTypes = map[string]primitiveMapping{
	"uuid":       {schema: "z.uuid()"},
	"string":     {schema: "z.string()"},
	"bool":       {schema: "z.boolean()"},
	"int8":       {schema: "zod.int8Z", xImports: []string{"zod"}},
	"int16":      {schema: "zod.int16Z", xImports: []string{"zod"}},
	"int32":      {schema: "zod.int32Z", xImports: []string{"zod"}},
	"int64":      {schema: "zod.int64Z", xImports: []string{"zod"}},
	"uint8":      {schema: "zod.uint8Z", xImports: []string{"zod"}},
	"uint16":     {schema: "zod.uint16Z", xImports: []string{"zod"}},
	"uint32":     {schema: "zod.uint32Z", xImports: []string{"zod"}},
	"uint64":     {schema: "zod.uint64Z", xImports: []string{"zod"}},
	"float32":    {schema: "zod.float32Z", xImports: []string{"zod"}},
	"float64":    {schema: "zod.float64Z", xImports: []string{"zod"}},
	"timestamp":  {schema: "TimeStamp.z", xImports: []string{"TimeStamp"}},
	"timespan":   {schema: "TimeSpan.z", xImports: []string{"TimeSpan"}},
	"time_range": {schema: "TimeRange.z", xImports: []string{"TimeRange"}},
	"json":       {schema: "z.record(z.string(), z.unknown())"},
	"bytes":      {schema: "z.instanceof(Uint8Array)"},
}

const xImportPath = "@synnaxlabs/x"

func primitiveToZod(primitive string, data *templateData) string {
	if mapping, ok := primitiveZodTypes[primitive]; ok {
		for _, name := range mapping.xImports {
			data.addNamedImport(xImportPath, name)
		}
		return mapping.schema
	}
	return "z.unknown()"
}

func (p *Plugin) applyValidation(zodType string, domain *resolution.DomainEntry, typeRef *resolution.TypeRef) string {
	isString := typeRef.Kind == resolution.TypeKindPrimitive && resolution.IsStringPrimitive(typeRef.Primitive)
	isNumber := typeRef.Kind == resolution.TypeKindPrimitive && resolution.IsNumberPrimitive(typeRef.Primitive)
	for _, expr := range domain.Expressions {
		if len(expr.Values) == 0 {
			switch expr.Name {
			case "email":
				if isString {
					zodType += ".email()"
				}
			case "url":
				if isString {
					zodType += ".url()"
				}
			}
			continue
		}
		v := expr.Values[0]
		switch expr.Name {
		case "min_length":
			if isString {
				zodType = fmt.Sprintf("%s.min(%d)", zodType, v.IntValue)
			}
		case "max_length":
			if isString {
				zodType = fmt.Sprintf("%s.max(%d)", zodType, v.IntValue)
			}
		case "pattern":
			if isString {
				zodType = fmt.Sprintf("%s.regex(/%s/)", zodType, v.StringValue)
			}
		case "min":
			if isNumber {
				if v.Kind == resolution.ValueKindInt {
					zodType = fmt.Sprintf("%s.min(%d)", zodType, v.IntValue)
				} else {
					zodType = fmt.Sprintf("%s.min(%f)", zodType, v.FloatValue)
				}
			}
		case "max":
			if isNumber {
				if v.Kind == resolution.ValueKindInt {
					zodType = fmt.Sprintf("%s.max(%d)", zodType, v.IntValue)
				} else {
					zodType = fmt.Sprintf("%s.max(%f)", zodType, v.FloatValue)
				}
			}
		}
	}
	return zodType
}

type templateData struct {
	Namespace     string
	OutputPath    string          // Current output path for calculating relative imports
	Request       *plugin.Request // Request for access to helper methods
	IDFields      []idFieldData
	Structs       []structData
	Enums         []enumData
	GenerateTypes bool
	Imports       map[string]*importSpec
	Ontology      *ontologyData // Ontology data if domain ontology is present
}

// ontologyData holds data for generating ontology ID factory and constant.
type ontologyData struct {
	TypeName string // e.g., "user" - from domain ontology { type "user" }
	KeyType  string // e.g., "Key" - derived from the ID field
}

type idFieldData struct {
	Name    string // e.g., "key"
	TSName  string // e.g., "key"
	ZodType string // e.g., "z.uuid()"
}

type importSpec struct {
	Names map[string]bool // named imports (e.g., "zod", "TimeStamp"), nil for namespace import
	Alias string          // for namespace imports (e.g., "user")
}

func (d *templateData) addNamedImport(path, name string) {
	if d.Imports[path] == nil {
		d.Imports[path] = &importSpec{Names: make(map[string]bool)}
	}
	d.Imports[path].Names[name] = true
}

func (d *templateData) addNamespaceImport(path, alias string) {
	if d.Imports[path] == nil {
		d.Imports[path] = &importSpec{Alias: alias}
	}
}

func (d *templateData) NamedImports() []namedImportData {
	var result []namedImportData
	for path, spec := range d.Imports {
		if spec.Names != nil && len(spec.Names) > 0 {
			names := make([]string, 0, len(spec.Names))
			for name := range spec.Names {
				names = append(names, name)
			}
			sort.Strings(names)
			result = append(result, namedImportData{Path: path, Names: names})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

// ExternalNamedImports returns named imports from external packages (not starting with @/)
func (d *templateData) ExternalNamedImports() []namedImportData {
	var result []namedImportData
	for _, imp := range d.NamedImports() {
		if !strings.HasPrefix(imp.Path, "@/") {
			result = append(result, imp)
		}
	}
	return result
}

// InternalNamedImports returns named imports from internal packages (starting with @/)
func (d *templateData) InternalNamedImports() []namedImportData {
	var result []namedImportData
	for _, imp := range d.NamedImports() {
		if strings.HasPrefix(imp.Path, "@/") {
			result = append(result, imp)
		}
	}
	return result
}

func (d *templateData) NamespaceImports() []namespaceImportData {
	var result []namespaceImportData
	for path, spec := range d.Imports {
		if spec.Alias != "" {
			result = append(result, namespaceImportData{Path: path, Alias: spec.Alias})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Alias < result[j].Alias })
	return result
}

type namedImportData struct {
	Path  string
	Names []string
}

type namespaceImportData struct {
	Path  string
	Alias string
}

type structData struct {
	Name   string
	Fields []fieldData
}

type fieldData struct {
	Name       string
	TSName     string
	ZodType    string
	IsOptional bool
	IsNullable bool
	IsArray    bool
}

type enumData struct {
	Name      string
	Values    []enumValueData
	IsIntEnum bool
}

type enumValueData struct {
	Name      string
	Value     string
	IntValue  int64
	IsIntEnum bool
}

var templateFuncs = template.FuncMap{
	"camelCase": lo.CamelCase,
	"title":     lo.Capitalize,
}

var fileTemplate = template.Must(template.New("zod").Funcs(templateFuncs).Parse(`// Code generated by Oracle. DO NOT EDIT.

import { z } from "zod";
{{- range .ExternalNamedImports }}
import { {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} } from "{{ .Path }}";
{{- end }}
{{- range .NamespaceImports }}
import * as {{ .Alias }} from "{{ .Path }}";
{{- end }}
{{ if .InternalNamedImports }}
{{- range .InternalNamedImports }}
import { {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} } from "{{ .Path }}";
{{- end }}
{{- end }}
{{- range .IDFields }}

export const {{ .TSName }}Z = {{ .ZodType }};
{{- if $.GenerateTypes }}
export type {{ .Name | camelCase | title }} = z.infer<typeof {{ .TSName }}Z>;
{{- end }}
{{- end }}
{{- range .Enums }}

{{- if .IsIntEnum }}
export const {{ camelCase .Name }}Values = [{{ range $i, $v := .Values }}{{ if $i }}, {{ end }}{{ $v.IntValue }}{{ end }}] as const;
export const {{ camelCase .Name }}Z = z.enum([{{ range $i, $v := .Values }}{{ if $i }}, {{ end }}"{{ $v.Name }}"{{ end }}]);
{{- else }}
export const {{ camelCase .Name }}Z = z.enum([{{ range $i, $v := .Values }}{{ if $i }}, {{ end }}"{{ $v.Value }}"{{ end }}]);
{{- end }}
{{- if $.GenerateTypes }}
export type {{ .Name }} = z.infer<typeof {{ camelCase .Name }}Z>;
{{- end }}
{{- end }}
{{- range .Structs }}

export const {{ camelCase .Name }}Z = z.object({
{{- range .Fields }}
  {{ .TSName }}: {{ .ZodType }},
{{- end }}
});
{{- if $.GenerateTypes }}
export type {{ .Name }} = z.infer<typeof {{ camelCase .Name }}Z>;
{{- end }}
{{- end }}
{{- if .Ontology }}

export const ontologyID = ontology.createIDFactory<{{ .Ontology.KeyType }}>("{{ .Ontology.TypeName }}");
export const TYPE_ONTOLOGY_ID = ontologyID("");
{{- end }}
`))
