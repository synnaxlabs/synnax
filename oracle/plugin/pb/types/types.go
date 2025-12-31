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
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// protoModulePrefix is the base import path for the Synnax monorepo.
const protoModulePrefix = "github.com/synnaxlabs/synnax/"

// Plugin generates Protocol Buffer definitions from Oracle schemas.
type Plugin struct{ Options Options }

// Options configures the protobuf types plugin.
type Options struct {
	FileNamePattern string
	MessagePrefix   string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		FileNamePattern: "types.gen.proto",
		MessagePrefix:   "PB",
	}
}

// New creates a new protobuf types plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "pb/types" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"pb"} }

// Requires returns plugin dependencies (none for this plugin).
func (p *Plugin) Requires() []string { return nil }

// Check verifies generated files are up-to-date.
func (p *Plugin) Check(req *plugin.Request) error { return nil }

// Generate produces protobuf definition files from the analyzed schemas.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	outputStructs := make(map[string][]resolution.Struct)
	var outputOrder []string

	for _, entry := range req.Resolutions.AllStructs() {
		if outputPath := output.GetPath(entry, "pb"); outputPath != "" {
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

// generateFile generates the protobuf file for a set of structs.
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
		Package:    derivePackageName(outputPath, structs),
		GoPackage:  deriveGoPackage(outputPath, structs),
		OutputPath: outputPath,
		Namespace:  namespace,
		Messages:   make([]messageData, 0, len(structs)),
		Enums:      make([]enumData, 0, len(enums)),
		imports:    newImportManager(),
		table:      table,
		prefix:     p.Options.MessagePrefix,
	}

	// Process enums that are in the same namespace
	for _, e := range enums {
		if e.Namespace == namespace && !omit.IsEnum(e, "pb") {
			data.Enums = append(data.Enums, p.processEnum(e))
		}
	}

	// Process structs
	for _, entry := range structs {
		if omit.IsStruct(entry, "pb") {
			continue
		}
		msg := p.processStruct(entry, data)
		if !msg.Skip {
			data.Messages = append(data.Messages, msg)
		}
	}

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// derivePackageName extracts or derives the proto package name.
func derivePackageName(outputPath string, structs []resolution.Struct) string {
	// Check for explicit package in domain
	for _, s := range structs {
		if domain, ok := s.Domains["pb"]; ok {
			if expr, found := domain.Expressions.Find("package"); found {
				if len(expr.Values) > 0 {
					return expr.Values[0].StringValue
				}
			}
		}
	}
	// Default: derive from output path
	parts := strings.Split(outputPath, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + ".v1"
	}
	return filepath.Base(outputPath) + ".v1"
}

// deriveGoPackage extracts or derives the Go package option.
func deriveGoPackage(outputPath string, structs []resolution.Struct) string {
	// Check for explicit go_package in domain
	for _, s := range structs {
		if domain, ok := s.Domains["pb"]; ok {
			if expr, found := domain.Expressions.Find("go_package"); found {
				if len(expr.Values) > 0 {
					return expr.Values[0].StringValue
				}
			}
		}
	}
	// Default: derive from output path
	return protoModulePrefix + outputPath
}

// processEnum converts an Enum to template data.
func (p *Plugin) processEnum(e resolution.Enum) enumData {
	ed := enumData{
		Name:   p.Options.MessagePrefix + e.Name,
		Values: make([]enumValueData, 0, len(e.Values)+1),
	}

	// Add UNSPECIFIED as first value (proto3 best practice)
	enumPrefix := toScreamingSnake(e.Name)
	ed.Values = append(ed.Values, enumValueData{
		Name:   fmt.Sprintf("%s_%s_UNSPECIFIED", p.Options.MessagePrefix, enumPrefix),
		Number: 0,
	})

	// Add actual enum values
	for i, v := range e.Values {
		ed.Values = append(ed.Values, enumValueData{
			Name:   fmt.Sprintf("%s_%s_%s", p.Options.MessagePrefix, enumPrefix, toScreamingSnake(v.Name)),
			Number: i + 1,
		})
	}

	return ed
}

// processStruct converts a Struct to template data.
func (p *Plugin) processStruct(entry resolution.Struct, data *templateData) messageData {
	// Skip type aliases - protobuf doesn't support them
	if entry.IsAlias() {
		return messageData{Skip: true}
	}

	md := messageData{
		Name:   p.Options.MessagePrefix + entry.Name,
		Fields: make([]fieldData, 0),
	}

	// Use UnifiedFields() to get flattened fields (handles inheritance)
	fieldNumber := 1
	for _, field := range entry.UnifiedFields() {
		fd := p.processField(field, fieldNumber, data)
		md.Fields = append(md.Fields, fd)
		fieldNumber++
	}

	return md
}

// processField converts a Field to template data.
func (p *Plugin) processField(field resolution.Field, number int, data *templateData) fieldData {
	protoType := p.typeToProto(field.TypeRef, data)

	return fieldData{
		Name:       toSnakeCase(field.Name),
		Type:       protoType,
		Number:     number,
		IsOptional: field.TypeRef.IsOptional || field.TypeRef.IsHardOptional,
		IsRepeated: field.TypeRef.IsArray,
	}
}

// typeToProto converts an Oracle type reference to a protobuf type string.
func (p *Plugin) typeToProto(typeRef *resolution.TypeRef, data *templateData) string {
	switch typeRef.Kind {
	case resolution.TypeKindPrimitive:
		mapping := primitiveProtoTypes[typeRef.Primitive]
		if mapping.importPath != "" {
			data.imports.add(mapping.importPath)
		}
		return mapping.protoType
	case resolution.TypeKindStruct:
		return p.resolveStructType(typeRef, data)
	case resolution.TypeKindEnum:
		return p.resolveEnumType(typeRef, data)
	case resolution.TypeKindMap:
		return p.resolveMapType(typeRef, data)
	case resolution.TypeKindTypeParam:
		// Generic type parameter -> google.protobuf.Any
		data.imports.add("google/protobuf/any.proto")
		return "google.protobuf.Any"
	default:
		return "bytes"
	}
}

// resolveStructType resolves a struct type reference to a protobuf type string.
func (p *Plugin) resolveStructType(typeRef *resolution.TypeRef, data *templateData) string {
	if typeRef.StructRef == nil {
		return "bytes"
	}

	structRef := typeRef.StructRef

	// If the struct is a type alias, follow the alias to the underlying type
	if structRef.IsAlias() && structRef.AliasOf != nil {
		return p.typeToProto(structRef.AliasOf, data)
	}

	// Same namespace - use prefixed name
	if structRef.Namespace == data.Namespace {
		return p.Options.MessagePrefix + structRef.Name
	}

	// Cross-namespace - need import
	targetOutputPath := output.GetPath(*structRef, "pb")
	if targetOutputPath == "" {
		return "bytes" // No pb output defined
	}

	// Add import for cross-namespace reference
	importPath := targetOutputPath + "/types.gen.proto"
	data.imports.add(importPath)

	// Use fully qualified name with package prefix
	pkg := derivePackageName(targetOutputPath, []resolution.Struct{*structRef})
	return fmt.Sprintf("%s.%s%s", pkg, p.Options.MessagePrefix, structRef.Name)
}

// resolveEnumType resolves an enum type reference to a protobuf type string.
func (p *Plugin) resolveEnumType(typeRef *resolution.TypeRef, data *templateData) string {
	if typeRef.EnumRef == nil {
		return "int32"
	}

	enumRef := typeRef.EnumRef

	// Same namespace - use prefixed name
	if enumRef.Namespace == data.Namespace {
		return p.Options.MessagePrefix + enumRef.Name
	}

	// Cross-namespace - need import
	targetOutputPath := enum.FindOutputPath(*enumRef, data.table, "pb")
	if targetOutputPath == "" {
		return "int32" // No pb output defined
	}

	// Add import for cross-namespace reference
	importPath := targetOutputPath + "/types.gen.proto"
	data.imports.add(importPath)

	// Use fully qualified name with package prefix
	pkg := derivePackageName(targetOutputPath, nil)
	return fmt.Sprintf("%s.%s%s", pkg, p.Options.MessagePrefix, enumRef.Name)
}

// resolveMapType resolves a map type reference to a protobuf type string.
func (p *Plugin) resolveMapType(typeRef *resolution.TypeRef, data *templateData) string {
	keyType := "string"
	valueType := "bytes"

	if typeRef.MapKeyType != nil {
		keyType = p.typeToProto(typeRef.MapKeyType, data)
	}
	if typeRef.MapValueType != nil {
		valueType = p.typeToProto(typeRef.MapValueType, data)
	}

	return fmt.Sprintf("map<%s, %s>", keyType, valueType)
}

// toSnakeCase converts a name to snake_case.
func toSnakeCase(s string) string {
	return lo.SnakeCase(s)
}

// toScreamingSnake converts a name to SCREAMING_SNAKE_CASE.
func toScreamingSnake(s string) string {
	return strings.ToUpper(lo.SnakeCase(s))
}

// primitiveMapping defines how an Oracle primitive maps to protobuf.
type primitiveMapping struct {
	protoType  string
	importPath string
}

// primitiveProtoTypes maps Oracle primitives to protobuf types.
var primitiveProtoTypes = map[string]primitiveMapping{
	"uuid":               {protoType: "string"},
	"string":             {protoType: "string"},
	"bool":               {protoType: "bool"},
	"int8":               {protoType: "int32"},
	"int16":              {protoType: "int32"},
	"int32":              {protoType: "int32"},
	"int64":              {protoType: "int64"},
	"uint8":              {protoType: "uint32"},
	"uint16":             {protoType: "uint32"},
	"uint32":             {protoType: "uint32"},
	"uint64":             {protoType: "uint64"},
	"float32":            {protoType: "float"},
	"float64":            {protoType: "double"},
	"timestamp":          {protoType: "int64"},
	"timespan":           {protoType: "int64"},
	"time_range":         {protoType: "telem.PBTimeRange", importPath: "x/go/telem/telem.proto"},
	"time_range_bounded": {protoType: "telem.PBTimeRange", importPath: "x/go/telem/telem.proto"},
	"json":               {protoType: "google.protobuf.Struct", importPath: "google/protobuf/struct.proto"},
	"bytes":              {protoType: "bytes"},
}

// importManager tracks protobuf imports needed for the generated file.
type importManager struct {
	imports map[string]bool
}

// newImportManager creates a new import manager.
func newImportManager() *importManager {
	return &importManager{imports: make(map[string]bool)}
}

// add adds an import path.
func (m *importManager) add(path string) {
	m.imports[path] = true
}

// sorted returns sorted imports.
func (m *importManager) sorted() []string {
	imports := make([]string, 0, len(m.imports))
	for imp := range m.imports {
		imports = append(imports, imp)
	}
	sort.Strings(imports)
	return imports
}

// templateData holds data for the protobuf file template.
type templateData struct {
	Package    string
	GoPackage  string
	OutputPath string
	Namespace  string
	Messages   []messageData
	Enums      []enumData
	imports    *importManager
	table      *resolution.Table
	prefix     string
}

// Imports returns sorted imports.
func (d *templateData) Imports() []string {
	return d.imports.sorted()
}

// messageData holds data for a single message definition.
type messageData struct {
	Name   string
	Fields []fieldData
	Skip   bool
}

// fieldData holds data for a single field definition.
type fieldData struct {
	Name       string
	Type       string
	Number     int
	IsOptional bool
	IsRepeated bool
}

// enumData holds data for an enum definition.
type enumData struct {
	Name   string
	Values []enumValueData
}

// enumValueData holds data for an enum value.
type enumValueData struct {
	Name   string
	Number int
}

var fileTemplate = template.Must(template.New("proto").Parse(`// Code generated by oracle. DO NOT EDIT.

syntax = "proto3";

package {{.Package}};

option go_package = "{{.GoPackage}}";
{{- if .Imports}}
{{range .Imports}}
import "{{.}}";
{{- end}}
{{- end}}
{{- range .Enums}}

enum {{.Name}} {
{{- range .Values}}
  {{.Name}} = {{.Number}};
{{- end}}
}
{{- end}}
{{- range .Messages}}

message {{.Name}} {
{{- range .Fields}}
  {{if .IsRepeated}}repeated {{else if .IsOptional}}optional {{end}}{{.Type}} {{.Name}} = {{.Number}};
{{- end}}
}
{{- end}}
`))
