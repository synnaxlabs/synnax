// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package types provides an Oracle plugin that generates Go struct type definitions
// from Oracle schema files. It handles primitive type mapping, struct embedding for
// extends relationships, generic type parameters, and cross-namespace imports.
package types

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	gointernal "github.com/synnaxlabs/oracle/plugin/go/internal"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// goModulePrefix is the base import path for the Synnax monorepo.
const goModulePrefix = "github.com/synnaxlabs/synnax/"

// Plugin generates Go struct type definitions from Oracle schemas.
type Plugin struct{ Options Options }

// Options configures the Go types plugin.
type Options struct {
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		FileNamePattern: "types.gen.go",
	}
}

// New creates a new Go types plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "go/types" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"go"} }

// Requires returns plugin dependencies (none for this plugin).
func (p *Plugin) Requires() []string { return nil }

// Check verifies generated files are up-to-date. Currently unimplemented.
func (p *Plugin) Check(*plugin.Request) error { return nil }

// Generate produces Go type definition files from the analyzed schemas.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	outputStructs := make(map[string][]resolution.Struct)
	var outputOrder []string

	for _, entry := range req.Resolutions.AllStructs() {
		if outputPath := output.GetPath(entry, "go"); outputPath != "" {
			if req.RepoRoot != "" {
				if err := req.ValidateOutputPath(outputPath); err != nil {
					return nil, errors.Wrapf(err, "invalid output path for struct %s", entry.Name)
				}
			}
			if _, exists := outputStructs[outputPath]; !exists {
				outputOrder = append(outputOrder, outputPath)
			}
			outputStructs[outputPath] = append(outputStructs[outputPath], entry)
		}
	}

	for _, outputPath := range outputOrder {
		structs := outputStructs[outputPath]
		enums := enum.CollectReferenced(structs)
		content, err := p.generateFile(outputPath, structs, enums, req.Resolutions)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
			Content: content,
		})
	}

	return resp, nil
}

// generateFile generates the Go source file for a set of structs.
func (p *Plugin) generateFile(
	outputPath string,
	structs []resolution.Struct,
	enums []resolution.Enum,
	table *resolution.Table,
) ([]byte, error) {
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	}

	data := &templateData{
		Package:    gointernal.DerivePackageName(outputPath),
		OutputPath: outputPath,
		Namespace:  namespace,
		Structs:    make([]structData, 0, len(structs)),
		Enums:      make([]enumData, 0, len(enums)),
		imports:    gointernal.NewImportManager(),
		table:      table,
	}

	// Process enums that are in the same namespace
	for _, e := range enums {
		if e.Namespace == namespace && !omit.IsEnum(e, "go") {
			data.Enums = append(data.Enums, p.processEnum(e))
		}
	}

	// Process structs
	for _, entry := range structs {
		// Skip omitted structs
		if omit.IsStruct(entry, "go") {
			continue
		}
		data.Structs = append(data.Structs, p.processStruct(entry, data))
	}

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// toGoImportPath converts an Oracle output path to a Go import path.
func toGoImportPath(outputPath string) string { return goModulePrefix + outputPath }

// processEnum converts an Enum to template data.
func (p *Plugin) processEnum(e resolution.Enum) enumData {
	values := make([]enumValueData, 0, len(e.Values))
	for _, v := range e.Values {
		values = append(values, enumValueData{
			Name:     gointernal.ToPascalCase(v.Name),
			Value:    v.StringValue,
			IntValue: v.IntValue,
		})
	}
	return enumData{
		Name:      e.Name,
		Values:    values,
		IsIntEnum: e.IsIntEnum,
	}
}

// processStruct converts a Struct to template data.
func (p *Plugin) processStruct(entry resolution.Struct, data *templateData) structData {
	sd := structData{
		Name:      entry.Name,
		Doc:       doc.Get(entry.Domains),
		Fields:    make([]fieldData, 0, len(entry.Fields)),
		IsGeneric: entry.IsGeneric(),
		IsAlias:   entry.IsAlias(),
	}

	// Process type parameters
	for _, tp := range entry.TypeParams {
		sd.TypeParams = append(sd.TypeParams, p.processTypeParam(tp, data))
	}

	// Handle alias types
	if entry.IsAlias() {
		sd.AliasOf = p.typeToGo(entry.AliasOf, data)
		return sd
	}

	// Handle struct extension
	if entry.HasExtends() && entry.Extends.StructRef != nil {
		// If omitting fields, fall back to field flattening
		// since Go struct embedding can't exclude individual parent fields
		if len(entry.OmittedFields) > 0 {
			// Use UnifiedFields() which respects OmittedFields
			for _, field := range entry.UnifiedFields() {
				sd.Fields = append(sd.Fields, p.processField(field, data))
			}
			return sd
		}

		// Use struct embedding (idiomatic Go pattern)
		sd.HasExtends = true
		sd.ExtendsType = p.resolveExtendsType(entry.Extends, data)

		// Only include child's own fields (parent fields come via embedding)
		for _, field := range entry.Fields {
			sd.Fields = append(sd.Fields, p.processField(field, data))
		}
		return sd
	}

	// Process fields for non-extending structs
	for _, field := range entry.UnifiedFields() {
		sd.Fields = append(sd.Fields, p.processField(field, data))
	}
	return sd
}

// processTypeParam converts a TypeParam to template data.
func (p *Plugin) processTypeParam(tp resolution.TypeParam, data *templateData) typeParamData {
	tpd := typeParamData{
		Name:       tp.Name,
		Constraint: "any", // Default constraint
	}

	// Map constraint to Go type
	if tp.Constraint != nil {
		tpd.Constraint = p.constraintToGo(tp.Constraint, data)
	}

	return tpd
}

// constraintToGo converts a type constraint to a Go constraint string.
func (p *Plugin) constraintToGo(constraint *resolution.TypeRef, data *templateData) string {
	switch constraint.Primitive {
	case "json":
		return "any"
	case "string":
		return "~string"
	case "int", "int8", "int16", "int32", "int64":
		return "~int | ~int8 | ~int16 | ~int32 | ~int64"
	case "uint", "uint8", "uint16", "uint32", "uint64":
		return "~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64"
	default:
		return p.typeToGo(constraint, data)
	}
}

// processField converts a Field to template data.
func (p *Plugin) processField(field resolution.Field, data *templateData) fieldData {
	return fieldData{
		GoName:     gointernal.ToPascalCase(field.Name),
		GoType:     p.typeToGo(field.TypeRef, data),
		JSONName:   field.Name,
		IsOptional: field.TypeRef.IsOptional || field.TypeRef.IsHardOptional,
		Doc:        doc.Get(field.Domains),
	}
}

// typeToGo converts an Oracle type reference to a Go type string.
func (p *Plugin) typeToGo(typeRef *resolution.TypeRef, data *templateData) string {
	var baseType string
	switch typeRef.Kind {
	case resolution.TypeKindPrimitive:
		mapping := primitiveGoTypes[typeRef.Primitive]
		baseType = mapping.goType
		if mapping.importPath != "" {
			data.imports.AddExternal(mapping.importPath)
		}
	case resolution.TypeKindStruct:
		baseType = p.resolveStructType(typeRef, data)
	case resolution.TypeKindEnum:
		baseType = p.resolveEnumType(typeRef, data)
	case resolution.TypeKindMap:
		baseType = p.resolveMapType(typeRef, data)
	case resolution.TypeKindTypeParam:
		baseType = p.resolveTypeParam(typeRef)
	default:
		baseType = "any"
	}
	if typeRef.IsArray {
		baseType = "[]" + baseType
	}
	if typeRef.IsHardOptional && !typeRef.IsArray {
		baseType = "*" + baseType
	}
	return baseType
}

// resolveStructType resolves a struct type reference to a Go type string.
func (p *Plugin) resolveStructType(typeRef *resolution.TypeRef, data *templateData) string {
	if typeRef.StructRef == nil {
		return "any"
	}
	structRef := typeRef.StructRef
	if structRef.Namespace == data.Namespace {
		return p.buildGenericType(structRef.Name, typeRef.TypeArgs, data)
	}
	targetOutputPath := output.GetPath(*structRef, "go")
	if targetOutputPath == "" {
		return "any"
	}
	alias := gointernal.DerivePackageAlias(targetOutputPath, data.Package)
	data.imports.AddInternal(alias, toGoImportPath(targetOutputPath))
	return fmt.Sprintf("%s.%s", alias, p.buildGenericType(structRef.Name, typeRef.TypeArgs, data))
}

// resolveEnumType resolves an enum type reference to a Go type string.
func (p *Plugin) resolveEnumType(typeRef *resolution.TypeRef, data *templateData) string {
	if typeRef.EnumRef == nil {
		return "any"
	}
	enumRef := typeRef.EnumRef
	if enumRef.Namespace == data.Namespace {
		return enumRef.Name
	}
	targetOutputPath := enum.FindOutputPath(*enumRef, data.table, "go")
	if targetOutputPath == "" {
		return "any"
	}
	alias := gointernal.DerivePackageAlias(targetOutputPath, data.Package)
	data.imports.AddInternal(alias, toGoImportPath(targetOutputPath))
	return fmt.Sprintf("%s.%s", alias, enumRef.Name)
}

// resolveMapType resolves a map type reference to a Go type string.
func (p *Plugin) resolveMapType(typeRef *resolution.TypeRef, data *templateData) string {
	keyType := "string"
	valueType := "any"

	if typeRef.MapKeyType != nil {
		keyType = p.typeToGo(typeRef.MapKeyType, data)
	}
	if typeRef.MapValueType != nil {
		valueType = p.typeToGo(typeRef.MapValueType, data)
	}

	return fmt.Sprintf("map[%s]%s", keyType, valueType)
}

// resolveTypeParam resolves a type parameter reference to a Go type string.
func (p *Plugin) resolveTypeParam(typeRef *resolution.TypeRef) string {
	if typeRef.TypeParamRef != nil {
		return typeRef.TypeParamRef.Name
	}
	return "any"
}

// buildGenericType builds a generic type string with type arguments.
func (p *Plugin) buildGenericType(baseName string, typeArgs []*resolution.TypeRef, data *templateData) string {
	if len(typeArgs) == 0 {
		return baseName
	}

	args := make([]string, len(typeArgs))
	for i, arg := range typeArgs {
		args[i] = p.typeToGo(arg, data)
	}
	return fmt.Sprintf("%s[%s]", baseName, strings.Join(args, ", "))
}

// resolveExtendsType resolves the parent type for struct embedding.
func (p *Plugin) resolveExtendsType(extendsRef *resolution.TypeRef, data *templateData) string {
	if extendsRef == nil || extendsRef.StructRef == nil {
		return ""
	}
	parent := extendsRef.StructRef
	if parent.Namespace == data.Namespace {
		return p.buildGenericType(parent.Name, extendsRef.TypeArgs, data)
	}
	targetOutputPath := output.GetPath(*parent, "go")
	if targetOutputPath == "" {
		return parent.Name
	}
	alias := gointernal.DerivePackageAlias(targetOutputPath, data.Package)
	data.imports.AddInternal(alias, toGoImportPath(targetOutputPath))
	return fmt.Sprintf("%s.%s", alias, p.buildGenericType(parent.Name, extendsRef.TypeArgs, data))
}

// primitiveMapping defines how an Oracle primitive maps to Go.
type primitiveMapping struct {
	goType     string
	importPath string
}

// primitiveGoTypes maps Oracle primitives to Go types.
var primitiveGoTypes = map[string]primitiveMapping{
	"uuid":       {goType: "uuid.UUID", importPath: "github.com/google/uuid"},
	"string":     {goType: "string"},
	"bool":       {goType: "bool"},
	"int8":       {goType: "int8"},
	"int16":      {goType: "int16"},
	"int32":      {goType: "int32"},
	"int64":      {goType: "int64"},
	"uint8":      {goType: "uint8"},
	"uint16":     {goType: "uint16"},
	"uint32":     {goType: "uint32"},
	"uint64":     {goType: "uint64"},
	"float32":    {goType: "float32"},
	"float64":    {goType: "float64"},
	"timestamp":  {goType: "telem.TimeStamp", importPath: "github.com/synnaxlabs/x/telem"},
	"timespan":   {goType: "telem.TimeSpan", importPath: "github.com/synnaxlabs/x/telem"},
	"time_range": {goType: "telem.TimeRange", importPath: "github.com/synnaxlabs/x/telem"},
	"json":       {goType: "map[string]any"},
	"bytes":      {goType: "[]byte"},
}

// templateData holds data for the Go file template.
type templateData struct {
	Package    string
	OutputPath string
	Namespace  string
	Structs    []structData
	Enums      []enumData
	imports    *gointernal.ImportManager
	table      *resolution.Table
}

// HasImports returns true if any imports are needed.
func (d *templateData) HasImports() bool { return d.imports.HasImports() }

// ExternalImports returns sorted external imports.
func (d *templateData) ExternalImports() []string { return d.imports.ExternalImports() }

// InternalImports returns sorted internal imports.
func (d *templateData) InternalImports() []gointernal.InternalImportData {
	return d.imports.InternalImports()
}

// structData holds data for a single struct definition.
type structData struct {
	Name       string
	Doc        string
	Fields     []fieldData
	TypeParams []typeParamData
	IsGeneric  bool
	IsAlias    bool
	AliasOf    string
	// Extension support
	HasExtends  bool
	ExtendsType string // Parent type (may be qualified: "parent.Parent")
}

// typeParamData holds data for a type parameter.
type typeParamData struct {
	Name       string
	Constraint string
}

// fieldData holds data for a single field definition.
type fieldData struct {
	GoName     string
	GoType     string
	JSONName   string
	IsOptional bool
	Doc        string
}

// TagSuffix returns the omitempty suffix if the field is optional.
func (f fieldData) TagSuffix() string {
	if f.IsOptional {
		return ",omitempty"
	}
	return ""
}

// enumData holds data for an enum definition.
type enumData struct {
	Name      string
	Values    []enumValueData
	IsIntEnum bool
}

// enumValueData holds data for an enum value.
type enumValueData struct {
	Name     string
	Value    string
	IntValue int64
}

var templateFuncs = template.FuncMap{
	"join": strings.Join,
}

var fileTemplate = template.Must(template.New("go-types").Funcs(templateFuncs).Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}
{{- if .HasImports}}

import (
{{- range .ExternalImports}}
	"{{.}}"
{{- end}}
{{- range .InternalImports}}
{{- if .NeedsAlias}}
	{{.Alias}} "{{.Path}}"
{{- else}}
	"{{.Path}}"
{{- end}}
{{- end}}
)
{{- end}}
{{- range $enum := .Enums}}

{{- if $enum.IsIntEnum}}

type {{$enum.Name}} int

const (
{{- range $i, $v := $enum.Values}}
{{- if eq $i 0}}
	{{$enum.Name}}{{$v.Name}} {{$enum.Name}} = iota
{{- else}}
	{{$enum.Name}}{{$v.Name}}
{{- end}}
{{- end}}
)
{{- else}}

type {{$enum.Name}} string

const (
{{- range $enum.Values}}
	{{$enum.Name}}{{.Name}} {{$enum.Name}} = "{{.Value}}"
{{- end}}
)
{{- end}}
{{- end}}
{{range .Structs}}
{{if .Doc -}}
// {{.Doc}}
{{end -}}
{{if .IsAlias -}}
type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} = {{.AliasOf}}
{{else if .HasExtends -}}
type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} struct {
	{{.ExtendsType}}
{{- range .Fields}}
{{- if .Doc}}
	// {{.Doc}}
{{- end}}
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JSONName}}{{.TagSuffix}}" msgpack:"{{.JSONName}}{{.TagSuffix}}"` + "`" + `
{{- end}}
}
{{else -}}
type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} struct {
{{- range .Fields}}
{{- if .Doc}}
	// {{.Doc}}
{{- end}}
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JSONName}}{{.TagSuffix}}" msgpack:"{{.JSONName}}{{.TagSuffix}}"` + "`" + `
{{- end}}
}
{{end -}}
{{end -}}
`))
