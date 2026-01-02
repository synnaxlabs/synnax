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
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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

// goModulePrefix is the fallback import path prefix.
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
	outputStructs := make(map[string][]resolution.Type)
	outputEnums := make(map[string][]resolution.Type)
	outputTypeDefs := make(map[string][]resolution.Type)
	var structOrder []string
	var enumOrder []string
	var typeDefOrder []string

	// Collect structs by output path
	for _, entry := range req.Resolutions.StructTypes() {
		if outputPath := output.GetPath(entry, "go"); outputPath != "" {
			if req.RepoRoot != "" {
				if err := req.ValidateOutputPath(outputPath); err != nil {
					return nil, errors.Wrapf(err, "invalid output path for struct %s", entry.Name)
				}
			}
			if _, exists := outputStructs[outputPath]; !exists {
				structOrder = append(structOrder, outputPath)
			}
			outputStructs[outputPath] = append(outputStructs[outputPath], entry)
		}
	}

	// Collect distinct types and aliases by output path
	for _, entry := range req.Resolutions.DistinctTypes() {
		if outputPath := output.GetPath(entry, "go"); outputPath != "" {
			if req.RepoRoot != "" {
				if err := req.ValidateOutputPath(outputPath); err != nil {
					return nil, errors.Wrapf(err, "invalid output path for typedef %s", entry.Name)
				}
			}
			if _, exists := outputTypeDefs[outputPath]; !exists {
				typeDefOrder = append(typeDefOrder, outputPath)
			}
			outputTypeDefs[outputPath] = append(outputTypeDefs[outputPath], entry)
		}
	}
	for _, entry := range req.Resolutions.AliasTypes() {
		if outputPath := output.GetPath(entry, "go"); outputPath != "" {
			if req.RepoRoot != "" {
				if err := req.ValidateOutputPath(outputPath); err != nil {
					return nil, errors.Wrapf(err, "invalid output path for alias %s", entry.Name)
				}
			}
			if _, exists := outputTypeDefs[outputPath]; !exists {
				typeDefOrder = append(typeDefOrder, outputPath)
			}
			outputTypeDefs[outputPath] = append(outputTypeDefs[outputPath], entry)
		}
	}

	// Collect standalone enums with their own @go output
	for _, e := range enum.CollectWithOwnOutput(req.Resolutions.EnumTypes(), "go") {
		enumPath := output.GetPath(e, "go")
		if req.RepoRoot != "" {
			if err := req.ValidateOutputPath(enumPath); err != nil {
				return nil, errors.Wrapf(err, "invalid output path for enum %s", e.Name)
			}
		}
		if _, exists := outputEnums[enumPath]; !exists {
			enumOrder = append(enumOrder, enumPath)
		}
		outputEnums[enumPath] = append(outputEnums[enumPath], e)
	}

	// Generate files for structs (merging in enums and typedefs from same output path)
	for _, outputPath := range structOrder {
		structs := outputStructs[outputPath]
		// Start with enums referenced by struct fields
		enums := enum.CollectReferenced(structs, req.Resolutions)
		// Merge in standalone enums that share the same output path
		if standaloneEnums, ok := outputEnums[outputPath]; ok {
			enums = mergeEnums(enums, standaloneEnums)
			delete(outputEnums, outputPath)
		}
		// Also include enums in the same namespace (inheriting file-level @go output)
		enums = mergeEnums(enums, collectNamespaceEnums(structs, req.Resolutions, outputPath))
		// Merge in typedefs that share the same output path
		var typeDefs []resolution.Type
		if tds, ok := outputTypeDefs[outputPath]; ok {
			typeDefs = tds
			delete(outputTypeDefs, outputPath)
		}
		content, err := p.generateFile(outputPath, structs, enums, typeDefs, req.Resolutions, req.RepoRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
			Content: content,
		})
	}

	// Generate files for standalone typedefs not already handled
	for _, outputPath := range typeDefOrder {
		typeDefs, ok := outputTypeDefs[outputPath]
		if !ok {
			continue
		}
		content, err := p.generateFile(outputPath, nil, nil, typeDefs, req.Resolutions, req.RepoRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
			Content: content,
		})
	}

	// Generate files for standalone enums not already handled
	for _, outputPath := range enumOrder {
		enums, ok := outputEnums[outputPath]
		if !ok {
			continue
		}
		content, err := p.generateFile(outputPath, nil, enums, nil, req.Resolutions, req.RepoRoot)
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

// mergeEnums combines two enum slices, deduplicating by QualifiedName.
func mergeEnums(a, b []resolution.Type) []resolution.Type {
	seen := make(map[string]bool, len(a))
	for _, e := range a {
		seen[e.QualifiedName] = true
	}
	result := append([]resolution.Type{}, a...)
	for _, e := range b {
		if !seen[e.QualifiedName] {
			result = append(result, e)
		}
	}
	return result
}

// collectNamespaceEnums finds enums in the same namespace as the structs
// that inherit their @go output from the file level (matching the output path).
func collectNamespaceEnums(structs []resolution.Type, table *resolution.Table, outputPath string) []resolution.Type {
	if len(structs) == 0 {
		return nil
	}
	namespace := structs[0].Namespace
	var result []resolution.Type
	for _, e := range table.EnumTypes() {
		if e.Namespace != namespace {
			continue
		}
		// Check if this enum's @go output matches (via file-level inheritance)
		if enumPath := enum.FindOutputPath(e, table, "go"); enumPath == outputPath {
			result = append(result, e)
		}
	}
	return result
}

// generateFile generates the Go source file for a set of types.
func (p *Plugin) generateFile(
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	typeDefs []resolution.Type,
	table *resolution.Table,
	repoRoot string,
) ([]byte, error) {
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	} else if len(typeDefs) > 0 {
		namespace = typeDefs[0].Namespace
	} else if len(enums) > 0 {
		namespace = enums[0].Namespace
	}

	data := &templateData{
		Package:    gointernal.DerivePackageName(outputPath),
		OutputPath: outputPath,
		Namespace:  namespace,
		Structs:    make([]structData, 0, len(structs)),
		Enums:      make([]enumData, 0, len(enums)),
		TypeDefs:   make([]typeDefData, 0, len(typeDefs)),
		imports:    gointernal.NewImportManager(),
		table:      table,
		repoRoot:   repoRoot,
	}

	// Process typedefs
	for _, td := range typeDefs {
		if !omit.IsType(td, "go") {
			data.TypeDefs = append(data.TypeDefs, p.processTypeDef(td, data))
		}
	}

	// Process enums that are in the same namespace
	for _, e := range enums {
		if e.Namespace == namespace && !omit.IsType(e, "go") {
			data.Enums = append(data.Enums, p.processEnum(e))
		}
	}

	// Process structs
	for _, entry := range structs {
		// Skip omitted structs
		if omit.IsType(entry, "go") {
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

// resolveGoImportPath resolves a repo-relative output path to a full Go import path.
// It searches for go.mod files to determine the correct module path.
func resolveGoImportPath(outputPath, repoRoot string) string {
	if repoRoot == "" {
		return goModulePrefix + outputPath
	}

	absPath := filepath.Join(repoRoot, outputPath)
	dir := absPath
	for {
		modPath := filepath.Join(dir, "go.mod")
		if info, err := os.Stat(modPath); err == nil && !info.IsDir() {
			moduleName, err := parseModuleName(modPath)
			if err != nil {
				break
			}
			relPath, err := filepath.Rel(dir, absPath)
			if err != nil {
				break
			}
			if relPath == "." {
				return moduleName
			}
			return moduleName + "/" + filepath.ToSlash(relPath)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return goModulePrefix + outputPath
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
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}
	return "", errors.Newf("no module directive found in %s", modPath)
}

// processEnum converts an enum Type to template data.
func (p *Plugin) processEnum(e resolution.Type) enumData {
	form := e.Form.(resolution.EnumForm)
	values := make([]enumValueData, 0, len(form.Values))
	for _, v := range form.Values {
		values = append(values, enumValueData{
			Name:     gointernal.ToPascalCase(v.Name),
			Value:    v.StringValue(),
			IntValue: v.IntValue(),
		})
	}
	return enumData{
		Name:      e.Name,
		Values:    values,
		IsIntEnum: form.IsIntEnum,
	}
}

// processTypeDef converts a TypeDef/Alias Type to template data.
func (p *Plugin) processTypeDef(td resolution.Type, data *templateData) typeDefData {
	// Check for @go name override
	name := getGoName(td)
	if name == "" {
		name = td.Name
	}

	switch form := td.Form.(type) {
	case resolution.DistinctForm:
		return typeDefData{
			Name:     name,
			BaseType: p.typeRefToGo(form.Base, data),
			IsAlias:  false, // DistinctForm → "type X Y" (distinct type)
		}
	case resolution.AliasForm:
		return typeDefData{
			Name:     name,
			BaseType: p.typeRefToGo(form.Target, data),
			IsAlias:  true, // AliasForm → "type X = Y" (transparent alias)
		}
	default:
		return typeDefData{Name: name, BaseType: "any"}
	}
}

// processStruct converts a struct Type to template data.
func (p *Plugin) processStruct(entry resolution.Type, data *templateData) structData {
	form := entry.Form.(resolution.StructForm)
	sd := structData{
		Name:      entry.Name,
		Doc:       doc.Get(entry.Domains),
		Fields:    make([]fieldData, 0, len(form.Fields)),
		IsGeneric: form.IsGeneric(),
		IsAlias:   false,
	}

	// Check for @go name override
	if goDomain, ok := entry.Domains["go"]; ok {
		for _, expr := range goDomain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				sd.Name = expr.Values[0].StringValue
			}
		}
	}

	// Process type parameters
	for _, tp := range form.TypeParams {
		sd.TypeParams = append(sd.TypeParams, p.processTypeParam(tp, data))
	}

	// Handle struct extension
	if form.Extends != nil {
		parent, ok := form.Extends.Resolve(data.table)
		if ok {
			// If omitting fields, fall back to field flattening
			// since Go struct embedding can't exclude individual parent fields
			if len(form.OmittedFields) > 0 {
				// Use UnifiedFields() which respects OmittedFields
				for _, field := range resolution.UnifiedFields(entry, data.table) {
					sd.Fields = append(sd.Fields, p.processField(field, data))
				}
				return sd
			}

			// Use struct embedding (idiomatic Go pattern)
			sd.HasExtends = true
			sd.ExtendsType = p.resolveExtendsType(form.Extends, parent, data)

			// Only include child's own fields (parent fields come via embedding)
			for _, field := range form.Fields {
				sd.Fields = append(sd.Fields, p.processField(field, data))
			}
			return sd
		}
	}

	// Process fields for non-extending structs
	for _, field := range resolution.UnifiedFields(entry, data.table) {
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
		tpd.Constraint = p.constraintToGo(*tp.Constraint, data)
	}

	return tpd
}

// constraintToGo converts a type constraint to a Go constraint string.
func (p *Plugin) constraintToGo(constraint resolution.TypeRef, data *templateData) string {
	if resolution.IsPrimitive(constraint.Name) {
		switch constraint.Name {
		case "json":
			return "any"
		case "string":
			return "~string"
		case "int", "int8", "int16", "int32", "int64":
			return "~int | ~int8 | ~int16 | ~int32 | ~int64"
		case "uint", "uint8", "uint16", "uint32", "uint64":
			return "~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64"
		default:
			return p.typeRefToGo(constraint, data)
		}
	}
	return p.typeRefToGo(constraint, data)
}

// processField converts a Field to template data.
func (p *Plugin) processField(field resolution.Field, data *templateData) fieldData {
	goType := p.typeRefToGo(field.Type, data)
	// Hard optional (??) fields become pointers in Go to distinguish nil from zero value.
	// Arrays and maps are reference types and don't need pointers (nil is their zero value).
	if field.IsHardOptional && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map[") {
		goType = "*" + goType
	}
	return fieldData{
		GoName:     gointernal.ToPascalCase(field.Name),
		GoType:     goType,
		JSONName:   field.Name,
		IsOptional: field.IsOptional || field.IsHardOptional,
		Doc:        doc.Get(field.Domains),
	}
}

// typeRefToGo converts an Oracle type reference to a Go type string.
func (p *Plugin) typeRefToGo(typeRef resolution.TypeRef, data *templateData) string {
	// Handle type parameters
	if typeRef.IsTypeParam() {
		return typeRef.TypeParam.Name
	}

	// Handle Array<T>
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		elemType := p.typeRefToGo(typeRef.TypeArgs[0], data)
		return "[]" + elemType
	}

	// Handle Map<K, V>
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		keyType := p.typeRefToGo(typeRef.TypeArgs[0], data)
		valType := p.typeRefToGo(typeRef.TypeArgs[1], data)
		return fmt.Sprintf("map[%s]%s", keyType, valType)
	}

	// Handle primitives
	if resolution.IsPrimitive(typeRef.Name) {
		mapping := primitiveGoTypes[typeRef.Name]
		if mapping.importPath != "" {
			data.imports.AddExternal(mapping.importPath)
		}
		return mapping.goType
	}

	// Resolve named type
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return "any"
	}

	switch resolved.Form.(type) {
	case resolution.StructForm:
		return p.resolveStructType(resolved, typeRef.TypeArgs, data)
	case resolution.EnumForm:
		return p.resolveEnumType(resolved, data)
	case resolution.DistinctForm:
		return p.resolveDistinctType(resolved, data)
	case resolution.AliasForm:
		// For aliases, use the alias name (like DistinctForm) so struct fields
		// reference the defined alias type, not the expanded target
		return p.resolveAliasType(resolved, data)
	default:
		return "any"
	}
}

// resolveStructType resolves a struct type to a Go type string.
func (p *Plugin) resolveStructType(resolved resolution.Type, typeArgs []resolution.TypeRef, data *templateData) string {
	// Check for @go name override
	typeName := getGoName(resolved)
	if typeName == "" {
		typeName = resolved.Name
	}
	targetOutputPath := output.GetPath(resolved, "go")
	// Same namespace AND same output path -> use unqualified name
	if resolved.Namespace == data.Namespace && (targetOutputPath == "" || targetOutputPath == data.OutputPath) {
		return p.buildGenericType(typeName, typeArgs, data)
	}
	if targetOutputPath == "" {
		return "any"
	}
	alias := gointernal.DerivePackageAlias(targetOutputPath, data.Package)
	data.imports.AddInternal(alias, resolveGoImportPath(targetOutputPath, data.repoRoot))
	return fmt.Sprintf("%s.%s", alias, p.buildGenericType(typeName, typeArgs, data))
}

// resolveEnumType resolves an enum type to a Go type string.
func (p *Plugin) resolveEnumType(resolved resolution.Type, data *templateData) string {
	targetOutputPath := enum.FindOutputPath(resolved, data.table, "go")
	// Same namespace AND same output path -> use unqualified name
	if resolved.Namespace == data.Namespace && (targetOutputPath == "" || targetOutputPath == data.OutputPath) {
		return resolved.Name
	}
	if targetOutputPath == "" {
		return "any"
	}
	alias := gointernal.DerivePackageAlias(targetOutputPath, data.Package)
	data.imports.AddInternal(alias, resolveGoImportPath(targetOutputPath, data.repoRoot))
	return fmt.Sprintf("%s.%s", alias, resolved.Name)
}

// resolveDistinctType resolves a distinct type to a Go type string.
func (p *Plugin) resolveDistinctType(resolved resolution.Type, data *templateData) string {
	// Use @go name override if present
	typeName := getGoName(resolved)
	if typeName == "" {
		typeName = resolved.Name
	}
	targetOutputPath := output.GetPath(resolved, "go")
	// Same namespace AND same output path -> use unqualified name
	if resolved.Namespace == data.Namespace && (targetOutputPath == "" || targetOutputPath == data.OutputPath) {
		return typeName
	}
	if targetOutputPath == "" {
		return "any"
	}
	alias := gointernal.DerivePackageAlias(targetOutputPath, data.Package)
	data.imports.AddInternal(alias, resolveGoImportPath(targetOutputPath, data.repoRoot))
	return fmt.Sprintf("%s.%s", alias, typeName)
}

// resolveAliasType resolves a type alias to a Go type string.
// Unlike expanding the target, this uses the alias name directly.
func (p *Plugin) resolveAliasType(resolved resolution.Type, data *templateData) string {
	// Use @go name override if present
	typeName := getGoName(resolved)
	if typeName == "" {
		typeName = resolved.Name
	}
	targetOutputPath := output.GetPath(resolved, "go")
	// Same namespace AND same output path -> use unqualified name
	if resolved.Namespace == data.Namespace && (targetOutputPath == "" || targetOutputPath == data.OutputPath) {
		return typeName
	}
	if targetOutputPath == "" {
		return "any"
	}
	alias := gointernal.DerivePackageAlias(targetOutputPath, data.Package)
	data.imports.AddInternal(alias, resolveGoImportPath(targetOutputPath, data.repoRoot))
	return fmt.Sprintf("%s.%s", alias, typeName)
}

// buildGenericType builds a generic type string with type arguments.
func (p *Plugin) buildGenericType(baseName string, typeArgs []resolution.TypeRef, data *templateData) string {
	if len(typeArgs) == 0 {
		return baseName
	}

	args := make([]string, len(typeArgs))
	for i, arg := range typeArgs {
		args[i] = p.typeRefToGo(arg, data)
	}
	return fmt.Sprintf("%s[%s]", baseName, strings.Join(args, ", "))
}

// resolveExtendsType resolves the parent type for struct embedding.
func (p *Plugin) resolveExtendsType(extendsRef *resolution.TypeRef, parent resolution.Type, data *templateData) string {
	if extendsRef == nil {
		return ""
	}
	targetOutputPath := output.GetPath(parent, "go")

	// Same namespace AND same output path (or no output path) -> use unqualified name
	if parent.Namespace == data.Namespace && (targetOutputPath == "" || targetOutputPath == data.OutputPath) {
		return p.buildGenericType(parent.Name, extendsRef.TypeArgs, data)
	}

	// Different output path -> need qualified name with import
	if targetOutputPath == "" {
		// No output path but different namespace - can't resolve package, use unqualified
		return parent.Name
	}
	alias := gointernal.DerivePackageAlias(targetOutputPath, data.Package)
	data.imports.AddInternal(alias, resolveGoImportPath(targetOutputPath, data.repoRoot))
	return fmt.Sprintf("%s.%s", alias, p.buildGenericType(parent.Name, extendsRef.TypeArgs, data))
}

// primitiveMapping defines how an Oracle primitive maps to Go.
type primitiveMapping struct {
	goType     string
	importPath string
}

// primitiveGoTypes maps Oracle primitives to Go types.
var primitiveGoTypes = map[string]primitiveMapping{
	"uuid":               {goType: "uuid.UUID", importPath: "github.com/google/uuid"},
	"string":             {goType: "string"},
	"bool":               {goType: "bool"},
	"int8":               {goType: "int8"},
	"int16":              {goType: "int16"},
	"int32":              {goType: "int32"},
	"int64":              {goType: "int64"},
	"uint8":              {goType: "uint8"},
	"uint12":             {goType: "types.Uint12", importPath: "github.com/synnaxlabs/x/types"},
	"uint16":             {goType: "uint16"},
	"uint20":             {goType: "types.Uint20", importPath: "github.com/synnaxlabs/x/types"},
	"uint32":             {goType: "uint32"},
	"uint64":             {goType: "uint64"},
	"float32":            {goType: "float32"},
	"float64":            {goType: "float64"},
	"timestamp":          {goType: "telem.TimeStamp", importPath: "github.com/synnaxlabs/x/telem"},
	"timespan":           {goType: "telem.TimeSpan", importPath: "github.com/synnaxlabs/x/telem"},
	"time_range":         {goType: "telem.TimeRange", importPath: "github.com/synnaxlabs/x/telem"},
	"time_range_bounded": {goType: "telem.TimeRange", importPath: "github.com/synnaxlabs/x/telem"},
	"data_type":          {goType: "telem.DataType", importPath: "github.com/synnaxlabs/x/telem"},
	"json":               {goType: "map[string]any"},
	"bytes":              {goType: "[]byte"},
}

// templateData holds data for the Go file template.
type templateData struct {
	Package    string
	OutputPath string
	Namespace  string
	Structs    []structData
	Enums      []enumData
	TypeDefs   []typeDefData
	imports    *gointernal.ImportManager
	table      *resolution.Table
	repoRoot   string
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

// typeDefData holds data for a type definition.
type typeDefData struct {
	Name     string
	BaseType string
	IsAlias  bool // If true, use "type X = Y", otherwise "type X Y"
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
{{- range .TypeDefs}}
{{- if .IsAlias}}

type {{.Name}} = {{.BaseType}}
{{- else}}

type {{.Name}} {{.BaseType}}
{{- end}}
{{- end}}
{{- range $enum := .Enums}}

{{- if $enum.IsIntEnum}}

type {{$enum.Name}} uint8

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

// getGoName gets the custom Go name from @go name annotation.
func getGoName(t resolution.Type) string {
	if domain, ok := t.Domains["go"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
}
