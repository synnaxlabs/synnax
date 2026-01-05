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
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/primitives"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// defaultModulePrefix is the fallback import path when go.mod resolution fails.
const defaultModulePrefix = "github.com/synnaxlabs/synnax/"

// Plugin generates Protocol Buffer definitions from Oracle schemas.
type Plugin struct{ Options Options }

// Options configures the protobuf types plugin.
type Options struct {
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		FileNamePattern: "types.gen.proto",
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

var bufGenerateCmd = []string{"buf", "generate"}

func (p *Plugin) PostWrite(files []string) error {
	if len(files) == 0 {
		return nil
	}
	firstFile := files[0]
	repoRoot := gomod.FindRepoRoot(firstFile)
	if repoRoot == "" {
		return errors.New("could not determine repo root from file paths")
	}
	return exec.OnFiles(bufGenerateCmd, nil, repoRoot)
}

// Generate produces protobuf definition files from the analyzed schemas.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	// Collect structs using the framework collector with custom PB path function
	structCollector := framework.NewCollector("pb", req).
		WithPathFunc(output.GetPBPath).
		WithSkipFunc(nil) // Don't skip - handle omit at generation time
	if err := structCollector.AddAll(req.Resolutions.StructTypes()); err != nil {
		return nil, err
	}

	err := structCollector.ForEach(func(outputPath string, structs []resolution.Type) error {
		// Get all enums in the namespace, not just referenced ones
		namespace := ""
		if len(structs) > 0 {
			namespace = structs[0].Namespace
		}
		enums := req.Resolutions.EnumsInNamespace(namespace)
		content, err := p.generateFile(outputPath, structs, enums, req.Resolutions, req.RepoRoot)
		if err != nil {
			return errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		// Use namespace-based filename: {namespace}.proto
		filename := namespace + ".proto"
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, filename),
			Content: content,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// generateFile generates the protobuf file for a set of structs.
func (p *Plugin) generateFile(
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	table *resolution.Table,
	repoRoot string,
) ([]byte, error) {
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	}

	data := &templateData{
		Package:    derivePackageName(outputPath, structs),
		GoPackage:  deriveGoPackage(outputPath, structs, repoRoot),
		OutputPath: outputPath,
		Namespace:  namespace,
		Messages:   make([]messageData, 0, len(structs)),
		Enums:      make([]enumData, 0, len(enums)),
		imports:    newImportManager(),
		table:      table,
		repoRoot:   repoRoot,
	}

	// Process enums that are in the same namespace
	for _, e := range enums {
		if e.Namespace == namespace && !omit.IsType(e, "pb") {
			data.Enums = append(data.Enums, p.processEnum(e))
		}
	}

	// Process structs
	for _, entry := range structs {
		if omit.IsType(entry, "pb") {
			continue
		}
		msg, err := p.processStruct(entry, data)
		if err != nil {
			return nil, err
		}
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

// derivePackageName derives the proto package name using {layer}.{namespace} format.
// For core/pkg/{layer}/... paths, uses the layer (distribution, api, service).
// For other paths like x/go/... or aspen/..., uses the first path component.
func derivePackageName(outputPath string, structs []resolution.Type) string {
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	} else {
		// Fallback: derive namespace from output path
		parts := strings.Split(outputPath, "/")
		if len(parts) >= 1 {
			if parts[len(parts)-1] == "pb" && len(parts) >= 2 {
				namespace = parts[len(parts)-2]
			} else {
				namespace = parts[len(parts)-1]
			}
		} else {
			namespace = filepath.Base(outputPath)
		}
	}
	prefix := deriveLayerPrefix(outputPath)
	return prefix + "." + namespace
}

// deriveLayerPrefix extracts the layer prefix from an output path.
// For core/pkg/{layer}/... returns the layer (distribution, api, service).
// For other paths returns the first component (x, aspen, cesium, etc.).
func deriveLayerPrefix(outputPath string) string {
	parts := strings.Split(outputPath, "/")
	// For core/pkg/{layer}/... paths, use the layer as prefix
	if len(parts) >= 3 && parts[0] == "core" && parts[1] == "pkg" {
		return parts[2]
	}
	// For other paths, use the first component
	if len(parts) >= 1 && parts[0] != "" {
		return parts[0]
	}
	return "synnax"
}

// deriveGoPackage derives the Go package option from the output path.
func deriveGoPackage(outputPath string, structs []resolution.Type, repoRoot string) string {
	return gomod.ResolveImportPath(outputPath, repoRoot, defaultModulePrefix)
}

// processEnum converts an Enum type to template data.
func (p *Plugin) processEnum(e resolution.Type) enumData {
	form, ok := e.Form.(resolution.EnumForm)
	if !ok {
		return enumData{Name: e.Name}
	}

	ed := enumData{
		Name:   e.Name,
		Values: make([]enumValueData, 0, len(form.Values)+1),
	}

	// Add UNSPECIFIED as first value (proto3 best practice)
	// Values don't need enum prefix since proto generates EnumName_VALUE format
	ed.Values = append(ed.Values, enumValueData{
		Name:   "UNSPECIFIED",
		Number: 0,
	})

	// Add actual enum values
	for i, v := range form.Values {
		ed.Values = append(ed.Values, enumValueData{
			Name:   toScreamingSnake(v.Name),
			Number: i + 1,
		})
	}

	return ed
}

// processStruct converts a Struct type to template data.
func (p *Plugin) processStruct(entry resolution.Type, data *templateData) (messageData, error) {
	_, ok := entry.Form.(resolution.StructForm)
	if !ok {
		// Not a struct form - check if it's an alias
		if _, isAlias := entry.Form.(resolution.AliasForm); isAlias {
			// Skip type aliases - protobuf doesn't support them
			return messageData{Skip: true}, nil
		}
		return messageData{Skip: true}, nil
	}

	// Use @pb name if specified, otherwise fall back to struct name
	name := getPBName(entry)
	if name == "" {
		name = entry.Name
	}

	md := messageData{
		Name:   name,
		Fields: make([]fieldData, 0),
	}

	// Use UnifiedFields() to get flattened fields (handles inheritance)
	fieldNumber := 1
	for _, field := range resolution.UnifiedFields(entry, data.table) {
		fd, err := p.processField(field, fieldNumber, data)
		if err != nil {
			return messageData{}, errors.Wrapf(err, "struct %q", entry.Name)
		}
		md.Fields = append(md.Fields, fd)
		fieldNumber++
	}

	return md, nil
}

// processField converts a Field to template data.
func (p *Plugin) processField(field resolution.Field, number int, data *templateData) (fieldData, error) {
	protoType, err := p.typeToProto(field.Type, data)
	if err != nil {
		return fieldData{}, errors.Wrapf(err, "field %q", field.Name)
	}

	// Check if the type is an array
	isArray := field.Type.Name == "Array"

	// Only hard optional (??) types are optional in proto.
	// Soft optional (?) types are regular fields in proto.
	return fieldData{
		Name:       toSnakeCase(field.Name),
		Type:       protoType,
		Number:     number,
		IsOptional: field.IsHardOptional,
		IsRepeated: isArray,
	}, nil
}

// typeToProto converts an Oracle type reference to a protobuf type string.
// Returns an error if the type cannot be mapped to a protobuf type.
func (p *Plugin) typeToProto(typeRef resolution.TypeRef, data *templateData) (string, error) {
	// Check if it's a type parameter
	if typeRef.IsTypeParam() {
		// Generic type parameter -> google.protobuf.Any
		data.imports.add("google/protobuf/any.proto")
		return "google.protobuf.Any", nil
	}

	// Resolve the type from the table
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return "", errors.Newf("failed to resolve type reference %q", typeRef.Name)
	}

	switch form := resolved.Form.(type) {
	case resolution.PrimitiveForm:
		mapping := primitives.GetMapping(form.Name, "pb")
		if mapping.TargetType == "any" {
			return "", errors.Newf("primitive type %q has no protobuf mapping", form.Name)
		}
		for _, imp := range mapping.Imports {
			data.imports.add(imp.Path)
		}
		return mapping.TargetType, nil

	case resolution.BuiltinGenericForm:
		if form.Name == "Array" {
			// For arrays, return the element type (repeated is handled at field level)
			if len(typeRef.TypeArgs) > 0 {
				return p.typeToProto(typeRef.TypeArgs[0], data)
			}
			return "", errors.New("Array type has no type arguments")
		}
		if form.Name == "Map" {
			return p.resolveMapType(typeRef, data)
		}
		return "", errors.Newf("unknown builtin generic type %q", form.Name)

	case resolution.StructForm:
		return p.resolveStructType(typeRef, resolved, data)

	case resolution.EnumForm:
		return p.resolveEnumType(resolved, data), nil

	case resolution.AliasForm:
		// Type aliases resolve to their underlying type in protobuf
		return p.typeToProto(form.Target, data)

	case resolution.DistinctForm:
		// Distinct types resolve to their underlying primitive type in protobuf
		// (protobuf has no type definition support)
		return p.typeToProto(form.Base, data)

	default:
		return "", errors.Newf("unknown type form %T for type %q", resolved.Form, resolved.Name)
	}
}

// resolveStructType resolves a struct type reference to a protobuf type string.
func (p *Plugin) resolveStructType(typeRef resolution.TypeRef, resolved resolution.Type, data *templateData) (string, error) {
	form, ok := resolved.Form.(resolution.StructForm)
	if !ok {
		// Check if it's an alias and follow it
		if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
			return p.typeToProto(aliasForm.Target, data)
		}
		return "", errors.Newf("expected struct form for type %q, got %T", resolved.Name, resolved.Form)
	}
	_ = form // form is available if needed

	// Get the protobuf name for this struct (use @pb name if set, else struct name)
	pbName := getPBName(resolved)
	if pbName == "" {
		pbName = resolved.Name
	}

	// Get target output path to compare with current file's output path
	targetOutputPath := output.GetPBPath(resolved)

	// Same output path - use pb name directly (handles self-references and same-file structs)
	if targetOutputPath == data.OutputPath {
		return pbName, nil
	}

	// Different output path - need import
	if targetOutputPath == "" {
		return "", errors.Newf("struct %q has no @pb output defined", resolved.Name)
	}

	// Add import for cross-namespace reference using {namespace}.proto pattern
	importPath := targetOutputPath + "/" + resolved.Namespace + ".proto"
	data.imports.add(importPath)

	// Use fully qualified name with package prefix
	pkg := derivePackageName(targetOutputPath, []resolution.Type{resolved})
	return fmt.Sprintf("%s.%s", pkg, pbName), nil
}

// resolveEnumType resolves an enum type reference to a protobuf type string.
func (p *Plugin) resolveEnumType(resolved resolution.Type, data *templateData) string {
	// Same namespace - use name directly
	if resolved.Namespace == data.Namespace {
		return resolved.Name
	}

	// Cross-namespace - need import using new pb/ pattern
	// Find the go output path for the enum and derive pb path
	targetOutputPath := enum.FindPBOutputPath(resolved, data.table)
	if targetOutputPath == "" {
		return "int32" // No pb output defined
	}

	// Add import for cross-namespace reference using {namespace}.proto pattern
	importPath := targetOutputPath + "/" + resolved.Namespace + ".proto"
	data.imports.add(importPath)

	// Use fully qualified name with package prefix
	pkg := deriveLayerPrefix(targetOutputPath) + "." + resolved.Namespace
	return fmt.Sprintf("%s.%s", pkg, resolved.Name)
}

// resolveMapType resolves a map type reference to a protobuf type string.
func (p *Plugin) resolveMapType(typeRef resolution.TypeRef, data *templateData) (string, error) {
	if len(typeRef.TypeArgs) < 2 {
		return "", errors.Newf("Map type requires 2 type arguments, got %d", len(typeRef.TypeArgs))
	}

	keyType, err := p.typeToProto(typeRef.TypeArgs[0], data)
	if err != nil {
		return "", errors.Wrap(err, "map key type")
	}

	valueType, err := p.typeToProto(typeRef.TypeArgs[1], data)
	if err != nil {
		return "", errors.Wrap(err, "map value type")
	}

	return fmt.Sprintf("map<%s, %s>", keyType, valueType), nil
}

// getPBName gets the custom protobuf name from @pb name annotation.
func getPBName(typ resolution.Type) string {
	if domain, ok := typ.Domains["pb"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
}

// toSnakeCase converts a name to snake_case.
func toSnakeCase(s string) string {
	return lo.SnakeCase(s)
}

// toScreamingSnake converts a name to SCREAMING_SNAKE_CASE.
func toScreamingSnake(s string) string {
	return strings.ToUpper(lo.SnakeCase(s))
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
	repoRoot   string
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
