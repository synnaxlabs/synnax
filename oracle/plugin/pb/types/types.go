// Copyright 2026 Synnax Labs, Inc.
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
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	pbprimitives "github.com/synnaxlabs/oracle/plugin/pb/primitives"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const defaultModulePrefix = "github.com/synnaxlabs/synnax/"

// primitiveMapper is the Protobuf-specific primitive type mapper.
var primitiveMapper = pbprimitives.Mapper()

type Plugin struct{ Options Options }

type Options struct {
	FileNamePattern string
}

func DefaultOptions() Options {
	return Options{
		FileNamePattern: "types.gen.proto",
	}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string { return "pb/types" }

func (p *Plugin) Domains() []string { return []string{"pb"} }

func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) Check(req *plugin.Request) error { return nil }

var (
	bufFormatCmd   = []string{"buf", "format", "-w"}
	bufGenerateCmd = []string{"buf", "generate"}
)

func (p *Plugin) PostWrite(files []string) error {
	if len(files) == 0 {
		return nil
	}
	firstFile := files[0]
	repoRoot := gomod.FindRepoRoot(firstFile)
	if repoRoot == "" {
		return errors.New("could not determine repo root from file paths")
	}
	// buf format -w and buf generate don't accept file arguments - they operate
	// on the entire directory. Run both without file arguments from the repo root.
	if err := exec.OnFiles(bufFormatCmd, nil, repoRoot); err != nil {
		return err
	}
	return exec.OnFiles(bufGenerateCmd, nil, repoRoot)
}

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
	return prefix + "." + namespace + ".pb"
}

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

func deriveGoPackage(outputPath string, structs []resolution.Type, repoRoot string) string {
	return gomod.ResolveImportPath(outputPath, repoRoot, defaultModulePrefix)
}

func (p *Plugin) processEnum(e resolution.Type) enumData {
	form, ok := e.Form.(resolution.EnumForm)
	if !ok {
		return enumData{Name: e.Name}
	}

	ed := enumData{
		Name:   e.Name,
		Values: make([]enumValueData, 0, len(form.Values)),
	}

	// Generate prefix from enum name (e.g., "OperationType" -> "OPERATION_TYPE_")
	prefix := toScreamingSnake(e.Name) + "_"

	// Use enum values exactly as defined in the oracle file.
	// The oracle schema is the source of truth - no implicit UNSPECIFIED values.
	// This keeps Go, C++, and Proto enum values aligned.
	//
	// For numeric enums (int values), use the explicit values from oracle.
	// For string enums, use sequential numbering starting from 0.
	for i, v := range form.Values {
		number := i // Default to sequential for string enums
		switch n := v.Value.(type) {
		case int64:
			number = int(n)
		case int:
			number = n
		}
		ed.Values = append(ed.Values, enumValueData{
			Name:   prefix + toScreamingSnake(v.Name),
			Number: number,
		})
	}

	return ed
}

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

func (p *Plugin) processField(field resolution.Field, number int, data *templateData) (fieldData, error) {
	// Check for fixed-size uint8 arrays (like Color [4]uint8) - these become bytes
	if p.isFixedSizeUint8Array(field.Type, data.table) {
		return fieldData{
			Name:       toSnakeCase(field.Name),
			Type:       "bytes",
			Number:     number,
			IsOptional: field.IsHardOptional,
			IsRepeated: false, // bytes is not repeated
		}, nil
	}

	// Check if the type is an array (following aliases)
	isArray := p.isArrayType(field.Type, data.table)

	// Check if the type is a nested array (array of arrays)
	if p.isNestedArrayType(field.Type, data.table) {
		// Generate a wrapper message for the inner array
		wrapperName, err := p.generateNestedArrayWrapper(field.Type, data)
		if err != nil {
			return fieldData{}, errors.Wrapf(err, "field %q", field.Name)
		}
		// Use the wrapper type instead
		return fieldData{
			Name:       toSnakeCase(field.Name),
			Type:       wrapperName,
			Number:     number,
			IsOptional: field.IsHardOptional,
			IsRepeated: true, // outer array is repeated
		}, nil
	}

	protoType, err := p.typeToProto(field.Type, data)
	if err != nil {
		return fieldData{}, errors.Wrapf(err, "field %q", field.Name)
	}

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

// isArrayType checks if a type reference resolves to an array type, following aliases and distinct types.
func (p *Plugin) isArrayType(typeRef resolution.TypeRef, table *resolution.Table) bool {
	// Direct Array type
	if typeRef.Name == "Array" {
		return true
	}

	// Resolve the type and check if it's an alias/distinct to an array
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return false
	}

	switch form := resolved.Form.(type) {
	case resolution.BuiltinGenericForm:
		return form.Name == "Array"
	case resolution.AliasForm:
		return p.isArrayType(form.Target, table)
	case resolution.DistinctForm:
		return p.isArrayType(form.Base, table)
	default:
		return false
	}
}

// getArrayElementType gets the element type of an array, following aliases and distinct types.
func (p *Plugin) getArrayElementType(typeRef resolution.TypeRef, table *resolution.Table) (resolution.TypeRef, bool) {
	// Direct Array type
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		return typeRef.TypeArgs[0], true
	}

	// Resolve the type and follow aliases/distinct types
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return resolution.TypeRef{}, false
	}

	switch form := resolved.Form.(type) {
	case resolution.BuiltinGenericForm:
		if form.Name == "Array" && len(typeRef.TypeArgs) > 0 {
			return typeRef.TypeArgs[0], true
		}
		return resolution.TypeRef{}, false
	case resolution.AliasForm:
		return p.getArrayElementType(form.Target, table)
	case resolution.DistinctForm:
		return p.getArrayElementType(form.Base, table)
	default:
		return resolution.TypeRef{}, false
	}
}

// isFixedSizeUint8Array checks if a type is a fixed-size uint8 array (like Color [4]uint8).
// These are encoded as bytes in protobuf for compact representation.
func (p *Plugin) isFixedSizeUint8Array(typeRef resolution.TypeRef, table *resolution.Table) bool {
	// Get the array size for this type (following aliases/distinct types)
	arraySize := p.getArraySize(typeRef, table)
	if arraySize == nil {
		return false
	}

	// Get the element type
	elemType, ok := p.getArrayElementType(typeRef, table)
	if !ok {
		return false
	}

	// Check if element type is uint8
	resolved, ok := elemType.Resolve(table)
	if !ok {
		// Direct primitive reference
		return elemType.Name == "uint8"
	}

	if prim, ok := resolved.Form.(resolution.PrimitiveForm); ok {
		return prim.Name == "uint8"
	}
	return false
}

// getArraySize returns the array size for a type, following aliases and distinct types.
func (p *Plugin) getArraySize(typeRef resolution.TypeRef, table *resolution.Table) *int64 {
	// Direct Array type with size
	if typeRef.Name == "Array" && typeRef.ArraySize != nil {
		return typeRef.ArraySize
	}

	// Resolve the type and follow aliases/distinct types
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return nil
	}

	switch form := resolved.Form.(type) {
	case resolution.AliasForm:
		return p.getArraySize(form.Target, table)
	case resolution.DistinctForm:
		return p.getArraySize(form.Base, table)
	default:
		return nil
	}
}

// isNestedArrayType checks if a type is an array of arrays (nested array).
func (p *Plugin) isNestedArrayType(typeRef resolution.TypeRef, table *resolution.Table) bool {
	if !p.isArrayType(typeRef, table) {
		return false
	}
	elemType, ok := p.getArrayElementType(typeRef, table)
	if !ok {
		return false
	}
	return p.isArrayType(elemType, table)
}

// getNestedArrayWrapperName returns the wrapper message name for a nested array element type.
func (p *Plugin) getNestedArrayWrapperName(typeRef resolution.TypeRef, table *resolution.Table) string {
	// Get the element type name to generate a wrapper name
	elemType, ok := p.getArrayElementType(typeRef, table)
	if !ok {
		return "ArrayWrapper"
	}

	// Try to get a meaningful name from the element type
	resolved, ok := elemType.Resolve(table)
	if ok {
		return resolved.Name + "Wrapper"
	}

	// For primitives, capitalize the type name
	if resolution.IsPrimitive(elemType.Name) {
		return cases.Title(language.English).String(elemType.Name) + "Array"
	}

	return "ArrayWrapper"
}

// generateNestedArrayWrapper generates a wrapper message for a nested array element type.
func (p *Plugin) generateNestedArrayWrapper(typeRef resolution.TypeRef, data *templateData) (string, error) {
	// Get the inner element type (the element of the inner array)
	elemType, ok := p.getArrayElementType(typeRef, data.table)
	if !ok {
		return "", errors.New("could not get element type for nested array")
	}

	// Get the innermost element type (element of the inner array)
	innerElemType, ok := p.getArrayElementType(elemType, data.table)
	if !ok {
		return "", errors.New("could not get inner element type for nested array")
	}

	// Convert the innermost element type to proto
	innerProtoType, err := p.typeToProto(innerElemType, data)
	if err != nil {
		return "", errors.Wrap(err, "could not convert inner element type to proto")
	}

	// Generate wrapper name based on the distinct/alias type name
	wrapperName := p.getNestedArrayWrapperName(typeRef, data.table)

	// Check if we've already generated this wrapper
	if data.wrapperMessages == nil {
		data.wrapperMessages = make(map[string]bool)
	}
	if !data.wrapperMessages[wrapperName] {
		data.wrapperMessages[wrapperName] = true
		// Add the wrapper message
		data.Messages = append(data.Messages, messageData{
			Name: wrapperName,
			Fields: []fieldData{
				{Name: "values", Type: innerProtoType, Number: 1, IsRepeated: true},
			},
		})
	}

	return wrapperName, nil
}

func (p *Plugin) typeToProto(typeRef resolution.TypeRef, data *templateData) (string, error) {
	// Check if it's a type parameter
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		// For defaulted type params, use the default type instead of Any
		if typeRef.TypeParam.HasDefault() {
			return p.typeToProto(*typeRef.TypeParam.Default, data)
		}
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
		mapping := primitiveMapper.Map(form.Name)
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

	// Use fully qualified name with package prefix and leading dot for absolute reference
	// The leading dot prevents protobuf from resolving types relative to the current package
	pkg := derivePackageName(targetOutputPath, []resolution.Type{resolved})
	return fmt.Sprintf(".%s.%s", pkg, pbName), nil
}

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

	// Use fully qualified name with package prefix and leading dot for absolute reference
	pkg := deriveLayerPrefix(targetOutputPath) + "." + resolved.Namespace + ".pb"
	return fmt.Sprintf(".%s.%s", pkg, resolved.Name)
}

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

func getPBName(typ resolution.Type) string {
	return domain.GetStringFromType(typ, "pb", "name")
}

func toSnakeCase(s string) string {
	return lo.SnakeCase(s)
}

func toScreamingSnake(s string) string {
	return strings.ToUpper(lo.SnakeCase(s))
}

type importManager struct {
	imports []string
}

func newImportManager() *importManager {
	return &importManager{}
}

func (m *importManager) add(path string) {
	if !lo.Contains(m.imports, path) {
		m.imports = append(m.imports, path)
	}
}

type templateData struct {
	Package         string
	GoPackage       string
	OutputPath      string
	Namespace       string
	Messages        []messageData
	Enums           []enumData
	imports         *importManager
	table           *resolution.Table
	repoRoot        string
	wrapperMessages map[string]bool
}

func (d *templateData) Imports() []string {
	return d.imports.imports
}

type messageData struct {
	Name   string
	Fields []fieldData
	Skip   bool
}

type fieldData struct {
	Name       string
	Type       string
	Number     int
	IsOptional bool
	IsRepeated bool
}

type enumData struct {
	Name   string
	Values []enumValueData
}

type enumValueData struct {
	Name   string
	Number int
}

var fileTemplate = template.Must(template.New("proto").Parse(`// Code generated by oracle. DO NOT EDIT.

syntax = "proto3";

package {{.Package}};
{{- if .Imports}}
{{range .Imports}}
import "{{.}}";
{{- end}}
{{- end}}

option go_package = "{{.GoPackage}}";
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
