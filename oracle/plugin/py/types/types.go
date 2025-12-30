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
}

func DefaultOptions() Options {
	return Options{
		OutputPath:      "{{.Namespace}}",
		FileNamePattern: "types.gen.py",
	}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string { return "py/types" }

func (p *Plugin) Domains() []string { return nil }

func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) Check(req *plugin.Request) error { return nil }

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	outputStructs := make(map[string][]*resolution.StructEntry)
	for _, entry := range req.Resolutions.AllStructs() {
		if outputPath := output.GetPath(entry, "py"); outputPath != "" {
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
		content, err := p.generateFile(structs[0].Namespace, structs, enums, req.Resolutions)
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

func (p *Plugin) generateFile(
	namespace string,
	structs []*resolution.StructEntry,
	enums []*resolution.EnumEntry,
	table *resolution.Table,
) ([]byte, error) {
	data := &templateData{
		Namespace: namespace,
		IDFields:  make([]idFieldData, 0),
		Structs:   make([]structData, 0, len(structs)),
		Enums:     make([]enumData, 0, len(enums)),
		imports:   newImportManager(),
	}
	data.imports.addPydantic("BaseModel")
	idFields := collectIDFields(structs, data)
	data.IDFields = idFields
	data.Ontology = extractOntology(structs, idFields)
	if data.Ontology != nil {
		data.imports.addOntology("ID")
	}

	for _, enum := range enums {
		data.Enums = append(data.Enums, p.processEnum(enum, data))
	}
	for _, entry := range structs {
		data.Structs = append(data.Structs, p.processStruct(entry, table, data, idFields))
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
						Name:   f.Name,
						PyType: primitiveToPython(f.TypeRef.Primitive, data),
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

		if len(idFields) == 0 {
			continue
		}
		keyType := idFields[0].PyType

		return &ontologyData{
			TypeName:   typeName,
			KeyType:    keyType,
			StructName: s.Name,
		}
	}
	return nil
}

func (p *Plugin) processEnum(enum *resolution.EnumEntry, data *templateData) enumData {
	values := make([]enumValueData, 0, len(enum.Values))
	var literalValues []string
	for _, v := range enum.Values {
		values = append(values, enumValueData{
			Name:      v.Name,
			Value:     v.StringValue,
			IntValue:  v.IntValue,
			IsIntEnum: enum.IsIntEnum,
		})
		if !enum.IsIntEnum {
			literalValues = append(literalValues, fmt.Sprintf("%q", v.StringValue))
		}
	}
	if enum.IsIntEnum {
		data.imports.addEnum("IntEnum")
	} else {
		data.imports.addTyping("Literal")
	}
	return enumData{
		Name:          enum.Name,
		Values:        values,
		IsIntEnum:     enum.IsIntEnum,
		LiteralValues: strings.Join(literalValues, ", "),
	}
}

func (p *Plugin) processStruct(
	entry *resolution.StructEntry,
	table *resolution.Table,
	data *templateData,
	idFields []idFieldData,
) structData {
	sd := structData{Name: entry.Name, Fields: make([]fieldData, 0, len(entry.Fields))}
	for _, field := range entry.Fields {
		sd.Fields = append(sd.Fields, p.processField(field, table, data, idFields))
	}
	return sd
}

func (p *Plugin) processField(
	field *resolution.FieldEntry,
	table *resolution.Table,
	data *templateData,
	idFields []idFieldData,
) fieldData {
	fd := fieldData{
		Name:       field.Name,
		IsOptional: field.TypeRef.IsOptional,
		IsNullable: field.TypeRef.IsNullable,
		IsArray:    field.TypeRef.IsArray,
	}

	baseType := p.typeToPython(field.TypeRef, table, data)
	var fieldConstraints []string
	if validateDomain := plugin.GetFieldDomain(field, "validate"); validateDomain != nil {
		fieldConstraints = p.collectValidation(validateDomain, field.TypeRef, data)
	}

	if field.TypeRef.IsArray {
		fd.PyType = fmt.Sprintf("list[%s]", baseType)
	} else {
		fd.PyType = baseType
	}

	if field.TypeRef.IsNullable || field.TypeRef.IsOptional {
		fd.PyType = fd.PyType + " | None"
	}

	fd.Default = p.buildDefault(field.TypeRef, fieldConstraints, data)

	return fd
}

func (p *Plugin) buildDefault(
	typeRef *resolution.TypeRef,
	constraints []string,
	data *templateData,
) string {
	hasConstraints := len(constraints) > 0

	if typeRef.IsOptional {
		if hasConstraints {
			data.imports.addPydantic("Field")
			return fmt.Sprintf(" = Field(default=None, %s)", strings.Join(constraints, ", "))
		}
		return " = None"
	}

	if typeRef.IsNullable && typeRef.IsArray {
		data.imports.addPydantic("Field")
		if hasConstraints {
			return fmt.Sprintf(" = Field(default_factory=list, %s)", strings.Join(constraints, ", "))
		}
		return " = Field(default_factory=list)"
	}

	if hasConstraints {
		data.imports.addPydantic("Field")
		return fmt.Sprintf(" = Field(%s)", strings.Join(constraints, ", "))
	}

	return ""
}

func (p *Plugin) collectValidation(
	domain *resolution.DomainEntry,
	typeRef *resolution.TypeRef,
	data *templateData,
) []string {
	var constraints []string
	isString := typeRef.Kind == resolution.TypeKindPrimitive && resolution.IsStringPrimitive(typeRef.Primitive)
	isNumber := typeRef.Kind == resolution.TypeKindPrimitive && resolution.IsNumberPrimitive(typeRef.Primitive)

	for _, expr := range domain.Expressions {
		if len(expr.Values) == 0 {
			switch expr.Name {
			case "email":
				if isString {
					constraints = append(constraints, `pattern=r"^[\w\.-]+@[\w\.-]+\.\w+$"`)
				}
			case "url":
				if isString {
					constraints = append(constraints, `pattern=r"^https?://"`)
				}
			}
			continue
		}
		v := expr.Values[0]
		switch expr.Name {
		case "min_length":
			if isString {
				constraints = append(constraints, fmt.Sprintf("min_length=%d", v.IntValue))
			}
		case "max_length":
			if isString {
				constraints = append(constraints, fmt.Sprintf("max_length=%d", v.IntValue))
			}
		case "pattern":
			if isString {
				constraints = append(constraints, fmt.Sprintf("pattern=r%q", v.StringValue))
			}
		case "min":
			if isNumber {
				if v.Kind == resolution.ValueKindInt {
					constraints = append(constraints, fmt.Sprintf("ge=%d", v.IntValue))
				} else {
					constraints = append(constraints, fmt.Sprintf("ge=%f", v.FloatValue))
				}
			}
		case "max":
			if isNumber {
				if v.Kind == resolution.ValueKindInt {
					constraints = append(constraints, fmt.Sprintf("le=%d", v.IntValue))
				} else {
					constraints = append(constraints, fmt.Sprintf("le=%f", v.FloatValue))
				}
			}
		case "default":
			switch v.Kind {
			case resolution.ValueKindBool:
				if v.BoolValue {
					constraints = append(constraints, "default=True")
				} else {
					constraints = append(constraints, "default=False")
				}
			case resolution.ValueKindInt:
				constraints = append(constraints, fmt.Sprintf("default=%d", v.IntValue))
			case resolution.ValueKindFloat:
				constraints = append(constraints, fmt.Sprintf("default=%f", v.FloatValue))
			case resolution.ValueKindString:
				constraints = append(constraints, fmt.Sprintf("default=%q", v.StringValue))
			}
		}
	}
	return constraints
}

func (p *Plugin) typeToPython(
	typeRef *resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	switch typeRef.Kind {
	case resolution.TypeKindPrimitive:
		return primitiveToPython(typeRef.Primitive, data)
	case resolution.TypeKindStruct:
		if typeRef.StructRef == nil {
			data.imports.addTyping("Any")
			return "Any"
		}
		structName := typeRef.StructRef.Name
		if typeRef.StructRef.Namespace != data.Namespace {
			ns := typeRef.StructRef.Namespace
			outputPath := output.GetPath(typeRef.StructRef, "py")
			if outputPath == "" {
				outputPath = ns
			}
			modulePath := toPythonModulePath(outputPath)
			data.imports.addNamespace(ns, modulePath)
			return fmt.Sprintf("%s.%s", ns, structName)
		}
		return structName
	case resolution.TypeKindEnum:
		if typeRef.EnumRef == nil {
			data.imports.addTyping("Any")
			return "Any"
		}
		enumName := typeRef.EnumRef.Name
		if typeRef.EnumRef.Namespace != data.Namespace {
			ns := typeRef.EnumRef.Namespace
			outputPath := enum.FindOutputPath(typeRef.EnumRef, table, "py")
			if outputPath == "" {
				outputPath = ns
			}
			modulePath := toPythonModulePath(outputPath)
			data.imports.addNamespace(ns, modulePath)
			return fmt.Sprintf("%s.%s", ns, enumName)
		}
		return enumName
	default:
		data.imports.addTyping("Any")
		return "Any"
	}
}

// toPythonModulePath converts a repo-relative path to a Python module path.
// For example: "client/py/synnax/user" -> "synnax.user"
func toPythonModulePath(repoPath string) string {
	prefixes := []string{
		"client/py/",
		"alamos/py/",
		"freighter/py/",
	}

	path := repoPath
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}

	return strings.ReplaceAll(path, "/", ".")
}

type primitiveMapping struct {
	pyType  string
	imports []importEntry
}

type importEntry struct {
	category string // "uuid", "typing", "synnax"
	name     string
}

var primitivePythonTypes = map[string]primitiveMapping{
	"uuid":       {pyType: "UUID", imports: []importEntry{{"uuid", "UUID"}}},
	"string":     {pyType: "str"},
	"bool":       {pyType: "bool"},
	"int8":       {pyType: "int"},
	"int16":      {pyType: "int"},
	"int32":      {pyType: "int"},
	"int64":      {pyType: "int"},
	"uint8":      {pyType: "int"},
	"uint16":     {pyType: "int"},
	"uint32":     {pyType: "int"},
	"uint64":     {pyType: "int"},
	"float32":    {pyType: "float"},
	"float64":    {pyType: "float"},
	"timestamp":  {pyType: "TimeStamp", imports: []importEntry{{"synnax", "TimeStamp"}}},
	"timespan":   {pyType: "TimeSpan", imports: []importEntry{{"synnax", "TimeSpan"}}},
	"time_range": {pyType: "TimeRange", imports: []importEntry{{"synnax", "TimeRange"}}},
	"json":       {pyType: "dict[str, Any]", imports: []importEntry{{"typing", "Any"}}},
	"bytes":      {pyType: "bytes"},
}

func primitiveToPython(primitive string, data *templateData) string {
	if mapping, ok := primitivePythonTypes[primitive]; ok {
		for _, imp := range mapping.imports {
			switch imp.category {
			case "uuid":
				data.imports.addUUID(imp.name)
			case "typing":
				data.imports.addTyping(imp.name)
			case "synnax":
				data.imports.addSynnax(imp.name)
			}
		}
		return mapping.pyType
	}
	data.imports.addTyping("Any")
	return "Any"
}

type importManager struct {
	uuid       map[string]bool
	typing     map[string]bool
	enum       map[string]bool
	pydantic   map[string]bool
	synnax     map[string]bool
	ontology   map[string]bool   // imports from synnax.ontology.payload
	namespaces map[string]string // alias -> path
}

func newImportManager() *importManager {
	return &importManager{
		uuid:       make(map[string]bool),
		typing:     make(map[string]bool),
		enum:       make(map[string]bool),
		pydantic:   make(map[string]bool),
		synnax:     make(map[string]bool),
		ontology:   make(map[string]bool),
		namespaces: make(map[string]string),
	}
}

func (m *importManager) addUUID(name string)      { m.uuid[name] = true }
func (m *importManager) addTyping(name string)    { m.typing[name] = true }
func (m *importManager) addEnum(name string)      { m.enum[name] = true }
func (m *importManager) addPydantic(name string)  { m.pydantic[name] = true }
func (m *importManager) addSynnax(name string)    { m.synnax[name] = true }
func (m *importManager) addOntology(name string)  { m.ontology[name] = true }
func (m *importManager) addNamespace(alias, path string) {
	m.namespaces[alias] = path
}

type templateData struct {
	Namespace string
	IDFields  []idFieldData
	Structs   []structData
	Enums     []enumData
	imports   *importManager
	Ontology  *ontologyData // Ontology data if domain ontology is present
}

// ontologyData holds data for generating ontology ID function and constant.
type ontologyData struct {
	TypeName   string // e.g., "user" - from domain ontology { type "user" }
	KeyType    string // e.g., "UUID" - derived from the ID field
	StructName string // e.g., "User" - the struct name for naming the constant
}

func (d *templateData) UUIDImports() []string {
	return sortedKeys(d.imports.uuid)
}

func (d *templateData) TypingImports() []string {
	return sortedKeys(d.imports.typing)
}

func (d *templateData) EnumImports() []string {
	return sortedKeys(d.imports.enum)
}

func (d *templateData) PydanticImports() []string {
	return sortedKeys(d.imports.pydantic)
}

func (d *templateData) SynnaxImports() []string {
	return sortedKeys(d.imports.synnax)
}

func (d *templateData) OntologyImports() []string {
	return sortedKeys(d.imports.ontology)
}

func (d *templateData) NamespaceImports() []namespaceImportData {
	var result []namespaceImportData
	for alias, path := range d.imports.namespaces {
		result = append(result, namespaceImportData{Alias: alias, Path: path})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Alias < result[j].Alias })
	return result
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

type idFieldData struct {
	Name   string
	PyType string
}

type structData struct {
	Name   string
	Fields []fieldData
}

type fieldData struct {
	Name       string
	PyType     string
	Default    string
	IsOptional bool
	IsNullable bool
	IsArray    bool
}

type enumData struct {
	Name          string
	Values        []enumValueData
	IsIntEnum     bool
	LiteralValues string
}

type enumValueData struct {
	Name      string
	Value     string
	IntValue  int64
	IsIntEnum bool
}

type namespaceImportData struct {
	Alias string
	Path  string
}

var templateFuncs = template.FuncMap{
	"title": lo.Capitalize,
	"join":  strings.Join,
	"upper": strings.ToUpper,
}

var fileTemplate = template.Must(template.New("python").Funcs(templateFuncs).Parse(`# Code generated by Oracle. DO NOT EDIT.

from __future__ import annotations
{{- if .UUIDImports }}
from uuid import {{ join .UUIDImports ", " }}
{{- end }}
{{- if .TypingImports }}
from typing import {{ join .TypingImports ", " }}
{{- end }}
{{- if .EnumImports }}
from enum import {{ join .EnumImports ", " }}
{{- end }}
{{- if .PydanticImports }}
from pydantic import {{ join .PydanticImports ", " }}
{{- end }}
{{- if .SynnaxImports }}
from synnax.telem import {{ join .SynnaxImports ", " }}
{{- end }}
{{- if .OntologyImports }}
from synnax.ontology.payload import {{ join .OntologyImports ", " }}
{{- end }}
{{- range .NamespaceImports }}
from {{ .Path }} import {{ .Alias }}
{{- end }}
{{- range .IDFields }}

{{ .Name | title }} = {{ .PyType }}
{{- end }}
{{- range .Enums }}
{{- if .IsIntEnum }}


class {{ .Name }}(IntEnum):
{{- range .Values }}
    {{ .Name }} = {{ .IntValue }}
{{- end }}
{{- else }}


{{ .Name }} = Literal[{{ .LiteralValues }}]
{{- end }}
{{- end }}
{{- range .Structs }}


class {{ .Name }}(BaseModel):
{{- range .Fields }}
    {{ .Name }}: {{ .PyType }}{{ .Default }}
{{- end }}
{{- end }}
{{- if .Ontology }}


{{ .Ontology.StructName | upper }}_ONTOLOGY_TYPE = ID(type="{{ .Ontology.TypeName }}")


def ontology_id(key: {{ .Ontology.KeyType }}) -> ID:
    return ID(type="{{ .Ontology.TypeName }}", key=str(key))
{{- end }}
`))
