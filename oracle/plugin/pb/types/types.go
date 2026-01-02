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
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
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

// PostWrite runs buf generate after proto files are written.
// This generates the Go bindings from the proto files.
func (p *Plugin) PostWrite(files []string) error {
	if len(files) == 0 {
		return nil
	}

	// Determine the repo root from the first file's path
	// Files are absolute paths, we need to find the repo root
	firstFile := files[0]
	repoRoot := findRepoRoot(firstFile)
	if repoRoot == "" {
		return errors.New("could not determine repo root from file paths")
	}

	// Run buf generate from the repo root
	cmd := exec.Command("buf", "generate")
	cmd.Dir = repoRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "buf generate failed")
	}

	return nil
}

// findRepoRoot walks up from the given path to find the git repository root.
func findRepoRoot(path string) string {
	dir := filepath.Dir(path)
	for {
		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// Generate produces protobuf definition files from the analyzed schemas.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	outputStructs := make(map[string][]resolution.Type)
	var outputOrder []string

	for _, entry := range req.Resolutions.StructTypes() {
		// Use GetPBPath which derives from @go output when @pb flag present
		outputPath := output.GetPBPath(entry)
		if outputPath == "" {
			continue
		}
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

	for _, outputPath := range outputOrder {
		structs := outputStructs[outputPath]
		// Get all enums in the namespace, not just referenced ones
		namespace := ""
		if len(structs) > 0 {
			namespace = structs[0].Namespace
		}
		enums := req.Resolutions.EnumsInNamespace(namespace)
		content, err := p.generateFile(outputPath, structs, enums, req.Resolutions, req.RepoRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		// Use namespace-based filename: {namespace}.proto
		filename := namespace + ".proto"
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, filename),
			Content: content,
		})
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
	// Resolve from go.mod if repoRoot is available
	if repoRoot != "" {
		if goPackage, err := resolveGoImportPath(outputPath, repoRoot); err == nil {
			return goPackage
		}
	}
	// Fallback: derive from output path with default prefix
	return defaultModulePrefix + outputPath
}

// resolveGoImportPath resolves a repo-relative output path to a full Go import path
// by walking up the directory tree to find the nearest go.mod file.
func resolveGoImportPath(outputPath, repoRoot string) (string, error) {
	absPath := filepath.Join(repoRoot, outputPath)

	// Walk up from the output path to find go.mod
	dir := absPath
	for {
		modPath := filepath.Join(dir, "go.mod")
		if fileExists(modPath) {
			moduleName, err := parseModuleName(modPath)
			if err != nil {
				return "", errors.Wrapf(err, "failed to parse go.mod at %s", modPath)
			}

			// Compute relative path from module root to output
			relPath, err := filepath.Rel(dir, absPath)
			if err != nil {
				return "", errors.Wrapf(err, "failed to compute relative path")
			}

			// Combine module name with relative path
			if relPath == "." {
				return moduleName, nil
			}
			return moduleName + "/" + filepath.ToSlash(relPath), nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root without finding go.mod
			break
		}
		dir = parent
	}

	return "", errors.Newf("no go.mod found for path %s", outputPath)
}

// parseModuleName extracts the module name from a go.mod file.
func parseModuleName(modPath string) (string, error) {
	file, err := os.Open(modPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			// Extract module name (handles both "module foo" and "module foo // comment")
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", errors.Newf("no module directive found in %s", modPath)
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
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
func (p *Plugin) processStruct(entry resolution.Type, data *templateData) messageData {
	_, ok := entry.Form.(resolution.StructForm)
	if !ok {
		// Not a struct form - check if it's an alias
		if _, isAlias := entry.Form.(resolution.AliasForm); isAlias {
			// Skip type aliases - protobuf doesn't support them
			return messageData{Skip: true}
		}
		return messageData{Skip: true}
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
		fd := p.processField(field, fieldNumber, data)
		md.Fields = append(md.Fields, fd)
		fieldNumber++
	}

	return md
}

// processField converts a Field to template data.
func (p *Plugin) processField(field resolution.Field, number int, data *templateData) fieldData {
	protoType := p.typeToProto(field.Type, data)

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
	}
}

// typeToProto converts an Oracle type reference to a protobuf type string.
func (p *Plugin) typeToProto(typeRef resolution.TypeRef, data *templateData) string {
	// Check if it's a type parameter
	if typeRef.IsTypeParam() {
		// Generic type parameter -> google.protobuf.Any
		data.imports.add("google/protobuf/any.proto")
		return "google.protobuf.Any"
	}

	// Resolve the type from the table
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return "bytes"
	}

	switch form := resolved.Form.(type) {
	case resolution.PrimitiveForm:
		mapping := primitiveProtoTypes[form.Name]
		if mapping.importPath != "" {
			data.imports.add(mapping.importPath)
		}
		return mapping.protoType

	case resolution.BuiltinGenericForm:
		if form.Name == "Array" {
			// For arrays, return the element type (repeated is handled at field level)
			if len(typeRef.TypeArgs) > 0 {
				return p.typeToProto(typeRef.TypeArgs[0], data)
			}
			return "bytes"
		}
		if form.Name == "Map" {
			return p.resolveMapType(typeRef, data)
		}
		return "bytes"

	case resolution.StructForm:
		return p.resolveStructType(typeRef, resolved, data)

	case resolution.EnumForm:
		return p.resolveEnumType(resolved, data)

	case resolution.AliasForm:
		// Type aliases resolve to their underlying type in protobuf
		return p.typeToProto(form.Target, data)

	case resolution.DistinctForm:
		// Distinct types resolve to their underlying primitive type in protobuf
		// (protobuf has no type definition support)
		return p.typeToProto(form.Base, data)

	default:
		return "bytes"
	}
}

// resolveStructType resolves a struct type reference to a protobuf type string.
func (p *Plugin) resolveStructType(typeRef resolution.TypeRef, resolved resolution.Type, data *templateData) string {
	form, ok := resolved.Form.(resolution.StructForm)
	if !ok {
		// Check if it's an alias and follow it
		if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
			return p.typeToProto(aliasForm.Target, data)
		}
		return "bytes"
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
		return pbName
	}

	// Different output path - need import
	if targetOutputPath == "" {
		return "bytes" // No pb output defined
	}

	// Add import for cross-namespace reference using {namespace}.proto pattern
	importPath := targetOutputPath + "/" + resolved.Namespace + ".proto"
	data.imports.add(importPath)

	// Use fully qualified name with package prefix
	pkg := derivePackageName(targetOutputPath, []resolution.Type{resolved})
	return fmt.Sprintf("%s.%s", pkg, pbName)
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
func (p *Plugin) resolveMapType(typeRef resolution.TypeRef, data *templateData) string {
	keyType := "string"
	valueType := "bytes"

	if len(typeRef.TypeArgs) >= 1 {
		keyType = p.typeToProto(typeRef.TypeArgs[0], data)
	}
	if len(typeRef.TypeArgs) >= 2 {
		valueType = p.typeToProto(typeRef.TypeArgs[1], data)
	}

	return fmt.Sprintf("map<%s, %s>", keyType, valueType)
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
	"uint12":             {protoType: "uint32"},
	"uint16":             {protoType: "uint32"},
	"uint20":             {protoType: "uint32"},
	"uint32":             {protoType: "uint32"},
	"uint64":             {protoType: "uint64"},
	"float32":            {protoType: "float"},
	"float64":            {protoType: "double"},
	"timestamp":          {protoType: "int64"},
	"timespan":           {protoType: "int64"},
	"time_range":         {protoType: "telem.PBTimeRange", importPath: "x/go/telem/telem.proto"},
	"time_range_bounded": {protoType: "telem.PBTimeRange", importPath: "x/go/telem/telem.proto"},
	"data_type":          {protoType: "string"},
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
