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
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Plugin generates C++ struct type definitions from Oracle schemas.
type Plugin struct{ Options Options }

// Options configures the C++ types plugin.
type Options struct {
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		FileNamePattern: "types.gen.h",
	}
}

// New creates a new C++ types plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "cpp/types" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"cpp"} }

// Requires returns plugin dependencies (none for this plugin).
func (p *Plugin) Requires() []string { return nil }

// Check verifies generated files are up-to-date.
func (p *Plugin) Check(req *plugin.Request) error {
	return nil
}

// Generate produces C++ type definition files from the analyzed schemas.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	outputStructs := make(map[string][]resolution.Struct)
	var outputOrder []string

	for _, entry := range req.Resolutions.AllStructs() {
		if outputPath := output.GetPath(entry, "cpp"); outputPath != "" {
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

// generateFile generates the C++ header file for a set of structs.
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
		OutputPath: outputPath,
		Namespace:  deriveNamespace(outputPath),
		KeyFields:  make([]keyFieldData, 0),
		Structs:    make([]structData, 0, len(structs)),
		Enums:      make([]enumData, 0, len(enums)),
		includes:   newIncludeManager(),
		table:      table,
		rawNs:      namespace,
	}

	// Collect key fields from structs
	skip := func(s resolution.Struct) bool { return omit.IsStruct(s, "cpp") }
	keyFields := key.Collect(structs, skip)
	for _, kf := range keyFields {
		data.KeyFields = append(data.KeyFields, p.processKeyField(kf, data))
	}

	// Process enums that are in the same namespace
	for _, e := range enums {
		if e.Namespace == namespace && !omit.IsEnum(e, "cpp") {
			data.Enums = append(data.Enums, p.processEnum(e))
		}
	}

	// Process structs
	for _, entry := range structs {
		if omit.IsStruct(entry, "cpp") {
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

// deriveNamespace extracts the C++ namespace from the output path.
// "client/cpp/rack" -> "synnax::rack"
// "x/cpp/status" -> "synnax::status"
func deriveNamespace(outputPath string) string {
	parts := strings.Split(outputPath, "/")
	if len(parts) == 0 {
		return "synnax"
	}
	subNs := parts[len(parts)-1]
	return fmt.Sprintf("synnax::%s", subNs)
}

// processEnum converts an Enum to template data.
func (p *Plugin) processEnum(e resolution.Enum) enumData {
	values := make([]enumValueData, 0, len(e.Values))
	for _, v := range e.Values {
		values = append(values, enumValueData{
			Name:     toPascalCase(v.Name),
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

// processKeyField converts a key field to a C++ type alias.
func (p *Plugin) processKeyField(kf key.Field, data *templateData) keyFieldData {
	cppType := p.primitiveToCpp(kf.Primitive, data)
	return keyFieldData{
		Name:    lo.Capitalize(lo.CamelCase(kf.Name)),
		CppType: cppType,
	}
}

// processStruct converts a Struct to template data.
func (p *Plugin) processStruct(entry resolution.Struct, data *templateData) structData {
	// Check for @cpp name override
	name := entry.Name
	if cppDomain, ok := entry.Domains["cpp"]; ok {
		for _, expr := range cppDomain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				name = expr.Values[0].StringValue
			}
		}
	}

	sd := structData{
		Name:      name,
		Doc:       doc.Get(entry.Domains),
		Fields:    make([]fieldData, 0, len(entry.Fields)),
		IsGeneric: entry.IsGeneric(),
		IsAlias:   entry.IsAlias(),
	}

	// Process type parameters
	for _, tp := range entry.TypeParams {
		sd.TypeParams = append(sd.TypeParams, p.processTypeParam(tp))
	}

	// Handle alias types
	if entry.IsAlias() {
		sd.AliasOf = p.typeToCpp(entry.AliasOf, data)
		return sd
	}

	// For C++, we always flatten fields (no struct embedding like Go)
	// This handles both extending and non-extending structs uniformly
	for _, field := range entry.UnifiedFields() {
		sd.Fields = append(sd.Fields, p.processField(field, data))
	}

	return sd
}

// processTypeParam converts a TypeParam to template data.
func (p *Plugin) processTypeParam(tp resolution.TypeParam) typeParamData {
	return typeParamData{
		Name: tp.Name,
	}
}

// processField converts a Field to template data.
func (p *Plugin) processField(field resolution.Field, data *templateData) fieldData {
	cppType := p.typeToCpp(field.TypeRef, data)

	return fieldData{
		Name:   field.Name,
		CppType: cppType,
		Doc:    doc.Get(field.Domains),
	}
}

// typeToCpp converts an Oracle type reference to a C++ type string.
func (p *Plugin) typeToCpp(typeRef *resolution.TypeRef, data *templateData) string {
	var baseType string

	switch typeRef.Kind {
	case resolution.TypeKindPrimitive:
		baseType = p.primitiveToCpp(typeRef.Primitive, data)
	case resolution.TypeKindStruct:
		baseType = p.resolveStructType(typeRef, data)
	case resolution.TypeKindEnum:
		baseType = p.resolveEnumType(typeRef, data)
	case resolution.TypeKindMap:
		baseType = p.resolveMapType(typeRef, data)
	case resolution.TypeKindTypeParam:
		baseType = p.resolveTypeParam(typeRef)
	default:
		baseType = "void"
	}

	// Handle arrays first
	if typeRef.IsArray {
		data.includes.addSystem("vector")
		baseType = fmt.Sprintf("std::vector<%s>", baseType)
	}

	// Only hard optionals (??) use std::optional in C++
	// Soft optionals (?) are just the bare type
	if typeRef.IsHardOptional {
		data.includes.addSystem("optional")
		baseType = fmt.Sprintf("std::optional<%s>", baseType)
	}

	return baseType
}

// primitiveToCpp converts an Oracle primitive to a C++ type.
func (p *Plugin) primitiveToCpp(primitive string, data *templateData) string {
	mapping, ok := primitiveCppTypes[primitive]
	if !ok {
		return "void"
	}

	for _, inc := range mapping.systemIncludes {
		data.includes.addSystem(inc)
	}
	for _, inc := range mapping.internalIncludes {
		data.includes.addInternal(inc)
	}

	return mapping.cppType
}

// resolveStructType resolves a struct type reference to a C++ type string.
func (p *Plugin) resolveStructType(typeRef *resolution.TypeRef, data *templateData) string {
	if typeRef.StructRef == nil {
		return "void"
	}

	structRef := typeRef.StructRef

	// Check if struct has a @cpp name override
	name := structRef.Name
	var cppInclude string
	var cppNamespace string
	isOmitted := omit.IsStruct(*structRef, "cpp")

	if cppDomain, ok := structRef.Domains["cpp"]; ok {
		for _, expr := range cppDomain.Expressions {
			switch expr.Name {
			case "name":
				if len(expr.Values) > 0 {
					name = expr.Values[0].StringValue
				}
			case "include":
				if len(expr.Values) > 0 {
					cppInclude = expr.Values[0].StringValue
				}
			case "namespace":
				if len(expr.Values) > 0 {
					cppNamespace = expr.Values[0].StringValue
				}
			}
		}
	}

	targetOutputPath := output.GetPath(*structRef, "cpp")

	// Handle cross-namespace references
	if structRef.Namespace != data.rawNs {
		if isOmitted || targetOutputPath == "" {
			// This struct is omitted or has no @cpp output - it's handwritten.
			// Use the @cpp include path if provided, otherwise we can't include it.
			if cppInclude != "" {
				data.includes.addInternal(cppInclude)
			}
			// Use namespace prefix for handwritten types.
			// If @cpp namespace is set, use it; otherwise derive from Oracle namespace.
			ns := cppNamespace
			if ns == "" {
				ns = structRef.Namespace
			}
			if ns != "" {
				name = fmt.Sprintf("%s::%s", ns, name)
			}
		} else {
			// Generated type - include the generated header
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.includes.addInternal(includePath)
		}
	}

	return p.buildGenericType(name, typeRef.TypeArgs, data)
}

// resolveEnumType resolves an enum type reference to a C++ type string.
func (p *Plugin) resolveEnumType(typeRef *resolution.TypeRef, data *templateData) string {
	if typeRef.EnumRef == nil {
		return "int"
	}

	enumRef := typeRef.EnumRef

	// For cross-namespace references, we need to add an include
	if enumRef.Namespace != data.rawNs {
		targetOutputPath := enum.FindOutputPath(*enumRef, data.table, "cpp")
		if targetOutputPath != "" {
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.includes.addInternal(includePath)
		}
	}

	return enumRef.Name
}

// resolveMapType resolves a map type reference to a C++ type string.
func (p *Plugin) resolveMapType(typeRef *resolution.TypeRef, data *templateData) string {
	keyType := "std::string"
	valueType := "void"

	if typeRef.MapKeyType != nil {
		keyType = p.typeToCpp(typeRef.MapKeyType, data)
	}
	if typeRef.MapValueType != nil {
		valueType = p.typeToCpp(typeRef.MapValueType, data)
	}

	data.includes.addSystem("unordered_map")
	return fmt.Sprintf("std::unordered_map<%s, %s>", keyType, valueType)
}

// resolveTypeParam resolves a type parameter reference to a C++ type string.
func (p *Plugin) resolveTypeParam(typeRef *resolution.TypeRef) string {
	if typeRef.TypeParamRef != nil {
		return typeRef.TypeParamRef.Name
	}
	return "void"
}

// buildGenericType builds a generic type string with type arguments.
func (p *Plugin) buildGenericType(baseName string, typeArgs []*resolution.TypeRef, data *templateData) string {
	if len(typeArgs) == 0 {
		return baseName
	}

	args := make([]string, len(typeArgs))
	for i, arg := range typeArgs {
		args[i] = p.typeToCpp(arg, data)
	}
	return fmt.Sprintf("%s<%s>", baseName, strings.Join(args, ", "))
}

// toPascalCase converts snake_case to PascalCase.
func toPascalCase(s string) string {
	return lo.PascalCase(s)
}

// primitiveMapping defines how an Oracle primitive maps to C++.
type primitiveMapping struct {
	cppType          string
	systemIncludes   []string
	internalIncludes []string
}

// primitiveCppTypes maps Oracle primitives to C++ types.
var primitiveCppTypes = map[string]primitiveMapping{
	"uuid":               {cppType: "std::string", systemIncludes: []string{"string"}},
	"string":             {cppType: "std::string", systemIncludes: []string{"string"}},
	"bool":               {cppType: "bool"},
	"int8":               {cppType: "std::int8_t", systemIncludes: []string{"cstdint"}},
	"int16":              {cppType: "std::int16_t", systemIncludes: []string{"cstdint"}},
	"int32":              {cppType: "std::int32_t", systemIncludes: []string{"cstdint"}},
	"int64":              {cppType: "std::int64_t", systemIncludes: []string{"cstdint"}},
	"uint8":              {cppType: "std::uint8_t", systemIncludes: []string{"cstdint"}},
	"uint16":             {cppType: "std::uint16_t", systemIncludes: []string{"cstdint"}},
	"uint32":             {cppType: "std::uint32_t", systemIncludes: []string{"cstdint"}},
	"uint64":             {cppType: "std::uint64_t", systemIncludes: []string{"cstdint"}},
	"float32":            {cppType: "float"},
	"float64":            {cppType: "double"},
	"timestamp":          {cppType: "telem::TimeStamp", internalIncludes: []string{"x/cpp/telem/telem.h"}},
	"timespan":           {cppType: "telem::TimeSpan", internalIncludes: []string{"x/cpp/telem/telem.h"}},
	"time_range":         {cppType: "telem::TimeRange", internalIncludes: []string{"x/cpp/telem/telem.h"}},
	"time_range_bounded": {cppType: "telem::TimeRange", internalIncludes: []string{"x/cpp/telem/telem.h"}},
	"json":               {cppType: "nlohmann::json", internalIncludes: []string{"nlohmann/json.hpp"}},
	"bytes":              {cppType: "std::vector<std::uint8_t>", systemIncludes: []string{"vector", "cstdint"}},
}

// includeManager tracks C++ includes needed for the generated file.
type includeManager struct {
	system   map[string]bool // System includes like <string>, <vector>
	internal map[string]bool // Internal includes like "x/cpp/telem/telem.h"
}

// newIncludeManager creates a new include manager.
func newIncludeManager() *includeManager {
	return &includeManager{
		system:   make(map[string]bool),
		internal: make(map[string]bool),
	}
}

// addSystem adds a system include.
func (m *includeManager) addSystem(name string) {
	m.system[name] = true
}

// addInternal adds an internal include.
func (m *includeManager) addInternal(path string) {
	m.internal[path] = true
}

// templateData holds data for the C++ file template.
type templateData struct {
	OutputPath string
	Namespace  string
	KeyFields  []keyFieldData
	Structs    []structData
	Enums      []enumData
	includes   *includeManager
	table      *resolution.Table
	rawNs      string // Original Oracle namespace for cross-reference detection
}

// keyFieldData holds data for a key type alias.
type keyFieldData struct {
	Name    string // e.g., "Key"
	CppType string // e.g., "std::uint32_t"
}

// HasIncludes returns true if any includes are needed.
func (d *templateData) HasIncludes() bool {
	return len(d.includes.system) > 0 || len(d.includes.internal) > 0
}

// SystemIncludes returns sorted system includes.
func (d *templateData) SystemIncludes() []string {
	includes := make([]string, 0, len(d.includes.system))
	for inc := range d.includes.system {
		includes = append(includes, inc)
	}
	sort.Strings(includes)
	return includes
}

// InternalIncludes returns sorted internal includes.
func (d *templateData) InternalIncludes() []string {
	includes := make([]string, 0, len(d.includes.internal))
	for inc := range d.includes.internal {
		includes = append(includes, inc)
	}
	sort.Strings(includes)
	return includes
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
}

// typeParamData holds data for a type parameter.
type typeParamData struct {
	Name string
}

// fieldData holds data for a single field definition.
type fieldData struct {
	Name    string
	CppType string
	Doc     string
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

var fileTemplate = template.Must(template.New("cpp-types").Funcs(templateFuncs).Parse(`// Code generated by oracle. DO NOT EDIT.

#pragma once
{{- if .HasIncludes}}

{{range .SystemIncludes -}}
#include <{{.}}>
{{end -}}
{{range .InternalIncludes -}}
#include "{{.}}"
{{end}}
{{- end}}
namespace {{.Namespace}} {
{{- range $i, $kf := .KeyFields}}
using {{$kf.Name}} = {{$kf.CppType}};
{{- end}}
{{- range $i, $enum := .Enums}}
{{if $i}}
{{end}}
{{if $enum.IsIntEnum}}
enum class {{$enum.Name}} {
{{- range $j, $v := $enum.Values}}
    {{$v.Name}} = {{$v.IntValue}},
{{- end}}
};
{{- else}}
{{- range $enum.Values}}
constexpr const char* {{$enum.Name}}{{.Name}} = "{{.Value}}";
{{- end}}
{{- end}}
{{- end}}
{{- range $i, $s := .Structs}}
{{if or $i (gt (len $.KeyFields) 0) (gt (len $.Enums) 0)}}
{{end}}
{{- if $s.Doc}}
/// @brief {{$s.Doc}}
{{end}}
{{- if $s.IsAlias}}
{{- if $s.IsGeneric}}template <{{range $j, $p := $s.TypeParams}}{{if $j}}, {{end}}typename {{$p.Name}}{{end}}>
{{end}}using {{$s.Name}} = {{$s.AliasOf}};
{{- else}}
{{- if $s.IsGeneric}}template <{{range $j, $p := $s.TypeParams}}{{if $j}}, {{end}}typename {{$p.Name}}{{end}}>
{{end}}struct {{$s.Name}} {
{{- range $s.Fields}}
{{- if .Doc}}
    /// @brief {{.Doc}}
{{- end}}
    {{.CppType}} {{.Name}};
{{- end}}
};
{{- end}}
{{- end}}
}
`))
