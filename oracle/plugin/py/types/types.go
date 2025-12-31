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
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/handwritten"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/domain/ontology"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/output"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	pluginoutput "github.com/synnaxlabs/oracle/plugin/output"
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
		FileNamePattern: "types_gen.py",
	}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string { return "py/types" }

func (p *Plugin) Domains() []string { return nil }

func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) Check(req *plugin.Request) error { return nil }

// PostWrite runs isort and black on the generated Python files using Poetry.
// Files are grouped by their Poetry project directory (containing pyproject.toml).
// isort runs first to sort imports, then black to format.
func (p *Plugin) PostWrite(files []string) error {
	if len(files) == 0 {
		return nil
	}
	byProject := make(map[string][]string)
	for _, file := range files {
		if projDir := findPoetryDir(file); projDir != "" {
			byProject[projDir] = append(byProject[projDir], file)
		}
	}
	for projDir, projFiles := range byProject {
		// Run isort first
		output.PostWriteStep("isort", len(projFiles), "sorting imports")
		isortArgs := append([]string{"run", "isort"}, projFiles...)
		isortCmd := exec.Command("poetry", isortArgs...)
		isortCmd.Dir = projDir
		if err := isortCmd.Run(); err != nil {
			return fmt.Errorf("isort failed in %s: %w", projDir, err)
		}
		// Then run black
		output.PostWriteStep("black", len(projFiles), "formatting")
		blackArgs := append([]string{"run", "black"}, projFiles...)
		blackCmd := exec.Command("poetry", blackArgs...)
		blackCmd.Dir = projDir
		if err := blackCmd.Run(); err != nil {
			return fmt.Errorf("black failed in %s: %w", projDir, err)
		}
	}
	return nil
}

// findPoetryDir finds the nearest directory containing pyproject.toml.
func findPoetryDir(file string) string {
	dir := filepath.Dir(file)
	for dir != "/" && dir != "." {
		if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return ""
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	outputStructs := make(map[string][]*resolution.StructEntry)
	for _, entry := range req.Resolutions.AllStructs() {
		if outputPath := pluginoutput.GetPath(entry, "py"); outputPath != "" {
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
		KeyFields: make([]keyFieldData, 0),
		Structs:   make([]structData, 0, len(structs)),
		Enums:     make([]enumData, 0, len(enums)),
		imports:   newImportManager(),
	}
	data.imports.addPydantic("BaseModel")
	skip := func(s *resolution.StructEntry) bool { return handwritten.IsStruct(s, "py") }
	rawKeyFields := key.Collect(structs, skip)
	keyFields := p.convertKeyFields(rawKeyFields, data)
	data.KeyFields = keyFields
	data.Ontology = p.extractOntology(structs, rawKeyFields, keyFields, skip)
	if data.Ontology != nil {
		data.imports.addOntology("ID")
	}
	for _, enum := range enums {
		data.Enums = append(data.Enums, p.processEnum(enum, data))
	}
	for _, entry := range structs {
		data.Structs = append(data.Structs, p.processStruct(entry, table, data, keyFields))
	}
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Plugin) convertKeyFields(fields []key.Field, data *templateData) []keyFieldData {
	result := make([]keyFieldData, 0, len(fields))
	for _, f := range fields {
		result = append(result, keyFieldData{
			Name:   f.Name,
			PyType: primitiveToPython(f.Primitive, data),
		})
	}
	return result
}

func (p *Plugin) extractOntology(structs []*resolution.StructEntry, rawFields []key.Field, keyFields []keyFieldData, skip ontology.SkipFunc) *ontologyData {
	data := ontology.Extract(structs, rawFields, skip)
	if data == nil || len(keyFields) == 0 {
		return nil
	}
	return &ontologyData{
		TypeName:   data.TypeName,
		KeyType:    keyFields[0].PyType,
		StructName: data.StructName,
	}
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
	keyFields []keyFieldData,
) structData {
	sd := structData{
		Name:   entry.Name,
		PyName: entry.Name, // Default to original name
	}
	// Check for py domain expressions (name, handwritten)
	if pyDomain, ok := entry.Domains["py"]; ok {
		for _, expr := range pyDomain.Expressions {
			switch expr.Name {
			case "handwritten":
				sd.Skip = true
				return sd
			case "name":
				if len(expr.Values) > 0 {
					sd.PyName = expr.Values[0].StringValue
				}
			}
		}
	}
	// Handle struct aliases: generate type alias expression
	if entry.IsAlias() {
		sd.IsAlias = true
		sd.AliasOf = p.typeRefToPythonAlias(entry.AliasOf, table, data)
		return sd
	}

	// Handle struct extension with Pydantic inheritance
	if entry.HasExtends() && entry.Extends.StructRef != nil {
		sd.HasExtends = true
		sd.ExtendsName = entry.Extends.StructRef.Name
		sd.OmittedFields = entry.OmittedFields

		// Add ConfigDict import if we have omitted fields
		if len(entry.OmittedFields) > 0 {
			data.imports.addPydantic("ConfigDict")
		}

		// For extends, only include child's own fields (not inherited)
		sd.Fields = make([]fieldData, 0, len(entry.Fields))
		for _, field := range entry.Fields {
			sd.Fields = append(sd.Fields, p.processField(field, table, data, keyFields))
		}
		return sd
	}

	// Non-extending struct: use all fields (flattened for compatibility)
	allFields := entry.AllFields()
	sd.Fields = make([]fieldData, 0, len(allFields))
	for _, field := range allFields {
		sd.Fields = append(sd.Fields, p.processField(field, table, data, keyFields))
	}
	return sd
}

func (p *Plugin) processField(
	field *resolution.FieldEntry,
	table *resolution.Table,
	data *templateData,
	keyFields []keyFieldData,
) fieldData {
	fd := fieldData{
		Name:           field.Name,
		IsOptional:     field.TypeRef.IsOptional,
		IsHardOptional: field.TypeRef.IsHardOptional,
		IsArray:        field.TypeRef.IsArray,
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

	// Both soft optional (?) and hard optional (??) become T | None in Python
	if field.TypeRef.IsOptional || field.TypeRef.IsHardOptional {
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

	isAnyOptional := typeRef.IsOptional || typeRef.IsHardOptional
	if isAnyOptional {
		if hasConstraints {
			data.imports.addPydantic("Field")
			return fmt.Sprintf(" = Field(default=None, %s)", strings.Join(constraints, ", "))
		}
		return " = None"
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
	rules := validation.Parse(domain)
	if validation.IsEmpty(rules) {
		return nil
	}
	var constraints []string
	isString := typeRef.Kind == resolution.TypeKindPrimitive && resolution.IsStringPrimitive(typeRef.Primitive)
	isNumber := typeRef.Kind == resolution.TypeKindPrimitive && resolution.IsNumberPrimitive(typeRef.Primitive)
	if isString {
		if rules.Email {
			constraints = append(constraints, `pattern=r"^[\w\.-]+@[\w\.-]+\.\w+$"`)
		}
		if rules.URL {
			constraints = append(constraints, `pattern=r"^https?://"`)
		}
		if rules.MinLength != nil {
			constraints = append(constraints, fmt.Sprintf("min_length=%d", *rules.MinLength))
		}
		if rules.MaxLength != nil {
			constraints = append(constraints, fmt.Sprintf("max_length=%d", *rules.MaxLength))
		}
		if rules.Pattern != nil {
			constraints = append(constraints, fmt.Sprintf("pattern=r%q", *rules.Pattern))
		}
	}
	if isNumber {
		if rules.Min != nil {
			if rules.Min.IsInt {
				constraints = append(constraints, fmt.Sprintf("ge=%d", rules.Min.Int))
			} else {
				constraints = append(constraints, fmt.Sprintf("ge=%f", rules.Min.Float))
			}
		}
		if rules.Max != nil {
			if rules.Max.IsInt {
				constraints = append(constraints, fmt.Sprintf("le=%d", rules.Max.Int))
			} else {
				constraints = append(constraints, fmt.Sprintf("le=%f", rules.Max.Float))
			}
		}
	}
	if rules.Default != nil {
		switch rules.Default.Kind {
		case resolution.ValueKindBool:
			if rules.Default.BoolValue {
				constraints = append(constraints, "default=True")
			} else {
				constraints = append(constraints, "default=False")
			}
		case resolution.ValueKindInt:
			constraints = append(constraints, fmt.Sprintf("default=%d", rules.Default.IntValue))
		case resolution.ValueKindFloat:
			constraints = append(constraints, fmt.Sprintf("default=%f", rules.Default.FloatValue))
		case resolution.ValueKindString:
			constraints = append(constraints, fmt.Sprintf("default=%q", rules.Default.StringValue))
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
			outputPath := pluginoutput.GetPath(typeRef.StructRef, "py")
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

// typeRefToPythonAlias converts a TypeRef to a Python type alias expression.
// For example: status.Status<StatusDetails> -> "status.Status[StatusDetails]"
func (p *Plugin) typeRefToPythonAlias(
	typeRef *resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	if typeRef.Kind != resolution.TypeKindStruct || typeRef.StructRef == nil {
		return p.typeToPython(typeRef, table, data)
	}

	// Get the base type name with proper import handling
	var baseName string
	structRef := typeRef.StructRef
	if structRef.Namespace != data.Namespace {
		outputPath := pluginoutput.GetPath(structRef, "py")
		if outputPath == "" {
			outputPath = structRef.Namespace
		}
		modulePath := toPythonModulePath(outputPath)
		// For cross-namespace struct references, we need to import the parent module
		// and access the struct as module.StructName to avoid naming conflicts.
		// e.g., "synnax.status" -> import "from synnax import status", use "status.Status"
		parts := strings.Split(modulePath, ".")
		if len(parts) >= 2 {
			parentPath := strings.Join(parts[:len(parts)-1], ".")
			moduleName := parts[len(parts)-1]
			data.imports.addModuleImport(parentPath, moduleName)
			baseName = fmt.Sprintf("%s.%s", moduleName, structRef.Name)
		} else {
			// Fallback for single-level module paths
			data.imports.addNamespace(structRef.Namespace, modulePath)
			baseName = fmt.Sprintf("%s.%s", structRef.Namespace, structRef.Name)
		}
	} else {
		baseName = structRef.Name
	}

	// If no type arguments, just return the base name
	if len(typeRef.TypeArgs) == 0 {
		return baseName
	}

	// Build type arguments for generic: Status[Details] or Status[Details, Data]
	var typeArgs []string
	for _, arg := range typeRef.TypeArgs {
		typeArgs = append(typeArgs, p.typeToPython(arg, table, data))
	}
	return fmt.Sprintf("%s[%s]", baseName, strings.Join(typeArgs, ", "))
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
	modules    map[string]string // module name -> parent path (for "from parent import module")
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
		modules:    make(map[string]string),
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
func (m *importManager) addModuleImport(parentPath, moduleName string) {
	m.modules[moduleName] = parentPath
}

type templateData struct {
	Namespace string
	KeyFields []keyFieldData
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

// ModuleImports returns imports of the form "from parent import module"
// e.g., "from synnax import status"
func (d *templateData) ModuleImports() []moduleImportData {
	var result []moduleImportData
	for moduleName, parentPath := range d.imports.modules {
		result = append(result, moduleImportData{Module: moduleName, Parent: parentPath})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Module < result[j].Module })
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

type keyFieldData struct {
	Name   string
	PyType string
}

type structData struct {
	Name    string // Original struct name from schema
	PyName  string // Python name (can be overridden via py domain { name "..." })
	Fields  []fieldData
	Skip    bool   // If true, skip generating this struct (handwritten)
	IsAlias bool   // If true, this struct is a type alias
	AliasOf string // Python expression for the aliased type (e.g., "status.Status[StatusDetails]")

	// Extension support
	HasExtends    bool
	ExtendsName   string   // Parent class name
	OmittedFields []string // Fields to exclude from parent
}

type fieldData struct {
	Name           string
	PyType         string
	Default        string
	IsOptional     bool
	IsHardOptional bool
	IsArray        bool
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

type moduleImportData struct {
	Module string // module name to import (e.g., "status")
	Parent string // parent path to import from (e.g., "synnax")
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
{{- range .ModuleImports }}
from {{ .Parent }} import {{ .Module }}
{{- end }}
{{- range .KeyFields }}

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
{{- if not .Skip }}
{{- if .IsAlias }}


{{ .PyName }} = {{ .AliasOf }}
{{- else if .HasExtends }}


class {{ .PyName }}({{ .ExtendsName }}):
{{- if or .Fields .OmittedFields }}
{{- range .Fields }}
    {{ .Name }}: {{ .PyType }}{{ .Default }}
{{- end }}
{{- if .OmittedFields }}

    model_config = ConfigDict(
        fields={
{{- range .OmittedFields }}
            "{{ . }}": {"exclude": True},
{{- end }}
        }
    )
{{- end }}
{{- else }}
    pass
{{- end }}
{{- else }}


class {{ .PyName }}(BaseModel):
{{- range .Fields }}
    {{ .Name }}: {{ .PyType }}{{ .Default }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- if .Ontology }}


{{ .Ontology.StructName | upper }}_ONTOLOGY_TYPE = ID(type="{{ .Ontology.TypeName }}")


def ontology_id(key: {{ .Ontology.KeyType }}) -> ID:
    return ID(type="{{ .Ontology.TypeName }}", key=str(key))
{{- end }}
`))
