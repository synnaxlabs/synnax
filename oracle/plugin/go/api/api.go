// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package api provides an Oracle plugin that generates API layer types and gRPC
// translator functions. It produces:
//   - Type aliases for service types that need API visibility
//   - Forward/Backward translation functions between Go API types and Protocol Buffer types
package api

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Plugin generates API layer types and gRPC translators from Oracle schemas.
type Plugin struct{ Options Options }

// Options configures the go/api plugin.
type Options struct {
	// AliasFileNamePattern is the filename pattern for type alias files.
	AliasFileNamePattern string
	// TranslatorFileNamePattern is the filename pattern for translator files.
	TranslatorFileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		AliasFileNamePattern:      "types.gen.go",
		TranslatorFileNamePattern: "translator.gen.go",
	}
}

// New creates a new go/api plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "go/api" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"api"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"go/types", "pb/types"} }

// Check verifies generated files are up-to-date.
func (p *Plugin) Check(req *plugin.Request) error {
	// TODO: Implement freshness check
	return nil
}

// Generate produces API type aliases and translator functions from the analyzed schemas.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	// Collect structs that have @api domain
	apiStructs := collectAPIStructs(req.Resolutions)

	// Generate type aliases for simple cases (structs with @api but no extends)
	aliasFiles, err := p.generateAliasFiles(apiStructs, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate alias files")
	}
	resp.Files = append(resp.Files, aliasFiles...)

	// Generate translators for structs that have both @api and @pb
	translatorFiles, err := p.generateTranslatorFiles(apiStructs, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate translator files")
	}
	resp.Files = append(resp.Files, translatorFiles...)

	return resp, nil
}

// collectAPIStructs returns all structs that have the @api domain.
func collectAPIStructs(table *resolution.Table) []resolution.Struct {
	var result []resolution.Struct
	for _, s := range table.AllStructs() {
		if _, hasAPI := s.Domains["api"]; hasAPI {
			result = append(result, s)
		}
	}
	return result
}

// generateAliasFiles generates type alias files for structs that have @api but don't
// use extends (meaning they're just exposing service types at the API layer).
func (p *Plugin) generateAliasFiles(
	apiStructs []resolution.Struct,
	req *plugin.Request,
) ([]plugin.File, error) {
	// Group structs by their @api output path
	outputGroups := make(map[string][]resolution.Struct)
	var outputOrder []string

	for _, s := range apiStructs {
		// Skip if struct is omitted
		if omit.IsStruct(s, "api") {
			continue
		}

		// Skip structs that use extends (go/types handles embedding)
		if s.HasExtends() {
			continue
		}

		apiOutput := output.GetPath(s, "api")
		if apiOutput == "" {
			continue
		}

		// Get the Go output path for the aliased type
		goOutput := output.GetPath(s, "go")
		if goOutput == "" {
			continue
		}

		// Skip if api output == go output (no alias needed)
		if apiOutput == goOutput {
			continue
		}

		if _, exists := outputGroups[apiOutput]; !exists {
			outputOrder = append(outputOrder, apiOutput)
		}
		outputGroups[apiOutput] = append(outputGroups[apiOutput], s)
	}

	var files []plugin.File
	for _, outputPath := range outputOrder {
		structs := outputGroups[outputPath]
		content, err := p.generateAliasFile(outputPath, structs, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate alias file for %s", outputPath)
		}
		files = append(files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.AliasFileNamePattern),
			Content: content,
		})
	}

	return files, nil
}

// generateAliasFile generates a Go file with type aliases.
func (p *Plugin) generateAliasFile(
	outputPath string,
	structs []resolution.Struct,
	req *plugin.Request,
) ([]byte, error) {
	data := &aliasTemplateData{
		Package: derivePackageName(outputPath),
		Aliases: make([]aliasData, 0, len(structs)),
		imports: newImportManager(),
	}

	for _, s := range structs {
		goOutput := output.GetPath(s, "go")
		importPath, err := resolveGoImportPath(goOutput, req.RepoRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve import for %s", s.Name)
		}

		alias := derivePackageAlias(goOutput, data.Package)
		data.imports.addInternal(alias, importPath)

		// Check if there's a custom name via @go name "Name"
		goName := getGoName(s)
		if goName == "" {
			goName = s.Name
		}

		data.Aliases = append(data.Aliases, aliasData{
			Name:    s.Name,
			AliasOf: fmt.Sprintf("%s.%s", alias, goName),
		})
	}

	var buf bytes.Buffer
	if err := aliasFileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// generateTranslatorFiles generates translator function files for structs that have
// both @api and @pb domains.
func (p *Plugin) generateTranslatorFiles(
	apiStructs []resolution.Struct,
	req *plugin.Request,
) ([]plugin.File, error) {
	// Group structs by their translator output path (derived from @pb output)
	// and then by their pb output basename (e.g., rack, ranger)
	type groupKey struct {
		translatorPath string
		baseName       string
	}
	groups := make(map[groupKey][]resolution.Struct)
	var groupOrder []groupKey

	for _, s := range apiStructs {
		// Skip if struct is omitted
		if omit.IsStruct(s, "api") || omit.IsStruct(s, "pb") {
			continue
		}

		// Need both @api and @pb for translator generation
		pbOutput := output.GetPath(s, "pb")
		if pbOutput == "" {
			continue
		}

		// Derive translator output path from @pb output
		translatorOutput := deriveTranslatorOutputPath(pbOutput)
		if translatorOutput == "" {
			continue
		}

		baseName := filepath.Base(pbOutput)
		key := groupKey{translatorPath: translatorOutput, baseName: baseName}

		if _, exists := groups[key]; !exists {
			groupOrder = append(groupOrder, key)
		}
		groups[key] = append(groups[key], s)
	}

	var files []plugin.File
	for _, key := range groupOrder {
		structs := groups[key]
		content, err := p.generateTranslatorFileForGroup(key.translatorPath, key.baseName, structs, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate translator file for %s/%s", key.translatorPath, key.baseName)
		}
		if len(content) > 0 {
			files = append(files, plugin.File{
				Path:    fmt.Sprintf("%s/%s.gen.go", key.translatorPath, key.baseName),
				Content: content,
			})
		}
	}

	return files, nil
}

// generateTranslatorFileForGroup generates translator content for a group of structs.
func (p *Plugin) generateTranslatorFileForGroup(
	outputPath string,
	groupName string,
	structs []resolution.Struct,
	req *plugin.Request,
) ([]byte, error) {
	// Get namespace from first struct in the group
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	}

	data := &translatorTemplateData{
		Package:     derivePackageName(outputPath),
		GroupName:   groupName,
		Namespace:   namespace,
		Translators: make([]translatorData, 0, len(structs)),
		imports:     newImportManager(),
		repoRoot:    req.RepoRoot,
		table:       req.Resolutions,
	}

	// Always need context import
	data.imports.addExternal("context")

	for _, s := range structs {
		translator, err := p.processStructForTranslation(s, data, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process struct %s", s.Name)
		}
		if translator != nil {
			data.Translators = append(data.Translators, *translator)
		}
	}

	if len(data.Translators) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	if err := translatorFileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// processStructForTranslation processes a struct and generates translator data.
func (p *Plugin) processStructForTranslation(
	s resolution.Struct,
	data *translatorTemplateData,
	req *plugin.Request,
) (*translatorData, error) {
	goOutput := output.GetPath(s, "go")
	pbOutput := output.GetPath(s, "pb")
	apiOutput := output.GetPath(s, "api")

	if goOutput == "" || pbOutput == "" {
		return nil, nil
	}

	// Determine the Go type to use (API type if different from Go type, else Go type)
	var goTypeOutput string
	if apiOutput != "" && apiOutput != goOutput {
		goTypeOutput = apiOutput
	} else {
		goTypeOutput = goOutput
	}

	// Derive the translator output path to check if Go type is in same package
	translatorOutputPath := deriveTranslatorOutputPath(pbOutput)

	// Resolve import paths
	goImportPath, err := resolveGoImportPath(goTypeOutput, req.RepoRoot)
	if err != nil {
		return nil, err
	}

	pbImportPath, err := resolveGoImportPath(pbOutput, req.RepoRoot)
	if err != nil {
		return nil, err
	}

	// Check if Go type is in the same package as the translator
	goTypeInSamePackage := goTypeOutput == translatorOutputPath

	var goAlias, goTypePrefix string
	if goTypeInSamePackage {
		// Don't import - the type is local (available via alias in types.gen.go)
		goTypePrefix = ""
	} else {
		goAlias = derivePackageAlias(goTypeOutput, data.Package)
		data.imports.addInternal(goAlias, goImportPath)
		goTypePrefix = goAlias + "."
	}

	pbAlias := derivePackageAlias(pbOutput, data.Package) + "v1"
	data.imports.addInternal(pbAlias, pbImportPath)

	// Get type names
	goName := getGoName(s)
	if goName == "" {
		goName = s.Name
	}
	pbName := "PB" + s.Name

	translator := &translatorData{
		Name:           s.Name,
		GoType:         goTypePrefix + goName,
		PBType:         fmt.Sprintf("%s.%s", pbAlias, pbName),
		GoTypeShort:    goName,
		PBTypeShort:    pbName,
		Fields:         make([]fieldTranslatorData, 0),
		OptionalFields: make([]fieldTranslatorData, 0),
	}

	// Process fields
	for _, field := range s.UnifiedFields() {
		fieldData := p.processFieldForTranslation(field, data, s)
		if fieldData.IsOptional {
			translator.OptionalFields = append(translator.OptionalFields, fieldData)
		} else {
			translator.Fields = append(translator.Fields, fieldData)
		}
	}

	return translator, nil
}

// processFieldForTranslation processes a field for translation.
func (p *Plugin) processFieldForTranslation(
	field resolution.Field,
	data *translatorTemplateData,
	parentStruct resolution.Struct,
) fieldTranslatorData {
	goName := toPascalCase(field.Name)
	pbName := toPascalCase(field.Name)

	// Only hard optional (??) results in a pointer in Go
	// Soft optional (?) results in a value type in Go but optional in proto
	isHardOptional := field.TypeRef.IsHardOptional

	// For template: fields in OptionalFields get nil checks
	// Only hard optional fields need nil checks (they're pointers in Go)
	isOptional := isHardOptional

	// Check if this is an optional struct (needs error handling) vs optional primitive
	isOptionalStruct := isOptional && field.TypeRef.Kind == resolution.TypeKindStruct

	// If this is a @key field, we need to import the package for type conversion
	if hasKeyDomain(field) {
		goOutput := output.GetPath(parentStruct, "go")
		if goOutput != "" {
			importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
			if err == nil {
				pkgAlias := derivePackageName(goOutput)
				data.imports.addInternal(pkgAlias, importPath)
			}
		}
	}

	forwardExpr, backwardExpr := p.generateFieldConversion(field, data, parentStruct)

	return fieldTranslatorData{
		GoName:           goName,
		PBName:           pbName,
		ForwardExpr:      forwardExpr,
		BackwardExpr:     backwardExpr,
		IsOptional:       isOptional,
		IsOptionalStruct: isOptionalStruct,
	}
}

// generateFieldConversion generates the forward and backward conversion expressions.
func (p *Plugin) generateFieldConversion(
	field resolution.Field,
	data *translatorTemplateData,
	parentStruct resolution.Struct,
) (forward, backward string) {
	typeRef := field.TypeRef
	fieldName := toPascalCase(field.Name)
	goFieldName := "r." + fieldName
	pbFieldName := "pb." + fieldName
	pbGetterName := "pb.Get" + fieldName + "()"

	// Check if this is a @key field - needs type conversion to/from package Key type
	isKeyField := hasKeyDomain(field)

	// Soft optional: Go value, proto pointer
	isSoftOptional := typeRef.IsOptional && !typeRef.IsHardOptional

	// Handle arrays
	if typeRef.IsArray {
		return p.generateArrayConversion(field, data, goFieldName, pbFieldName)
	}

	// Handle @key fields with typed wrappers (e.g., rack.Key -> uint32)
	// Only applies to numeric types that get wrapped, not uuid which has its own conversion
	if isKeyField && typeRef.Kind == resolution.TypeKindPrimitive && isNumericPrimitive(typeRef.Primitive) {
		goOutput := output.GetPath(parentStruct, "go")
		pkgName := derivePackageName(goOutput)
		protoType := primitiveToProtoType(typeRef.Primitive)
		// Forward: uint32(r.Key), Backward: rack.Key(pb.Key)
		return fmt.Sprintf("%s(%s)", protoType, goFieldName),
			fmt.Sprintf("%s.Key(%s)", pkgName, pbFieldName)
	}

	// Handle primitives
	if typeRef.Kind == resolution.TypeKindPrimitive {
		forward, backward := p.generatePrimitiveConversion(typeRef.Primitive, goFieldName, pbFieldName, data)
		// For soft optional primitives, proto field is a pointer
		// Forward: take address, Backward: use getter (handles nil)
		if isSoftOptional {
			// Simple primitives just need address/getter
			if isSimplePrimitive(typeRef.Primitive) {
				return fmt.Sprintf("&%s", goFieldName), pbGetterName
			}
			// Other primitives (uuid, timestamp, etc.) need the conversion with address/getter
			forward, backward = p.generatePrimitiveConversion(typeRef.Primitive, goFieldName, pbGetterName, data)
		}
		return forward, backward
	}

	// Handle struct references
	if typeRef.Kind == resolution.TypeKindStruct && typeRef.StructRef != nil {
		return p.generateStructConversion(typeRef, data, goFieldName, pbFieldName)
	}

	// Handle enums
	if typeRef.Kind == resolution.TypeKindEnum && typeRef.EnumRef != nil {
		return p.generateEnumConversion(typeRef, data, goFieldName, pbFieldName)
	}

	// Default: direct copy
	return goFieldName, pbFieldName
}

// hasKeyDomain checks if a field has the @key annotation.
func hasKeyDomain(field resolution.Field) bool {
	_, hasKey := field.Domains["key"]
	return hasKey
}

// primitiveToProtoType returns the protobuf type name for a primitive.
func primitiveToProtoType(primitive string) string {
	switch primitive {
	case "uint32":
		return "uint32"
	case "uint64":
		return "uint64"
	case "int32":
		return "int32"
	case "int64":
		return "int64"
	default:
		return primitive
	}
}

// isNumericPrimitive returns true if the primitive is a numeric type that can be wrapped.
func isNumericPrimitive(primitive string) bool {
	switch primitive {
	case "uint8", "uint16", "uint32", "uint64",
		"int8", "int16", "int32", "int64":
		return true
	default:
		return false
	}
}

// isSimplePrimitive returns true if the primitive can be directly assigned
// without needing conversion functions (just address/dereference for optionals).
func isSimplePrimitive(primitive string) bool {
	switch primitive {
	case "string", "bool",
		"uint8", "uint16", "uint32", "uint64",
		"int8", "int16", "int32", "int64",
		"float32", "float64":
		return true
	default:
		return false
	}
}

// generatePrimitiveConversion generates conversion for primitive types.
func (p *Plugin) generatePrimitiveConversion(
	primitive, goField, pbField string,
	data *translatorTemplateData,
) (forward, backward string) {
	switch primitive {
	case "uuid":
		data.imports.addExternal("github.com/google/uuid")
		return fmt.Sprintf("%s.String()", goField),
			fmt.Sprintf("uuid.MustParse(%s)", pbField)
	case "timestamp":
		data.imports.addExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("int64(%s)", goField),
			fmt.Sprintf("telem.TimeStamp(%s)", pbField)
	case "timespan":
		data.imports.addExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("int64(%s)", goField),
			fmt.Sprintf("telem.TimeSpan(%s)", pbField)
	case "time_range":
		data.imports.addExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("telem.TranslateTimeRangeForward(%s)", goField),
			fmt.Sprintf("telem.TranslateTimeRangeBackward(%s)", pbField)
	case "json":
		data.imports.addExternal("google.golang.org/protobuf/types/known/structpb")
		return fmt.Sprintf("structpb.NewValue(%s)", goField),
			fmt.Sprintf("%s.AsInterface()", pbField)
	default:
		// Direct copy for string, bool, int*, uint*, float*, bytes
		return goField, pbField
	}
}

// generateStructConversion generates conversion for struct references.
func (p *Plugin) generateStructConversion(
	typeRef *resolution.TypeRef,
	data *translatorTemplateData,
	goField, pbField string,
) (forward, backward string) {
	structRef := typeRef.StructRef

	// Follow alias chain to find the actual underlying struct
	actualStruct := structRef
	for actualStruct.IsAlias() && actualStruct.AliasOf != nil && actualStruct.AliasOf.StructRef != nil {
		actualStruct = actualStruct.AliasOf.StructRef
	}

	structName := actualStruct.Name

	// Check if referenced struct is from a different namespace
	translatorPrefix := ""
	if actualStruct.Namespace != data.Namespace {
		// Find the translator package for the referenced struct
		pbOutput := output.GetPath(*actualStruct, "pb")
		if pbOutput != "" {
			translatorPath := deriveTranslatorOutputPath(pbOutput)
			if translatorPath != "" {
				importPath, err := resolveGoImportPath(translatorPath, data.repoRoot)
				if err == nil {
					// Create a package alias for the translator
					alias := strings.ToLower(actualStruct.Namespace) + "grpc"
					data.imports.addInternal(alias, importPath)
					translatorPrefix = alias + "."
				}
			}
		}
	}

	// Only hard optional (??) fields are pointers in Go
	// Soft optional (?) and non-optional are value types
	if typeRef.IsHardOptional {
		// Hard optional: field is already a pointer, don't add &
		// These will be used inside if-blocks that handle nil checks
		return fmt.Sprintf("%sTranslate%sForward(ctx, %s)", translatorPrefix, structName, goField),
			fmt.Sprintf("%sTranslate%sBackward(ctx, %s)", translatorPrefix, structName, pbField)
	}

	// For non-optional or soft optional struct fields - need to take address
	return fmt.Sprintf("%sTranslate%sForward(ctx, &%s)", translatorPrefix, structName, goField),
		fmt.Sprintf("%sTranslate%sBackward(ctx, %s)", translatorPrefix, structName, pbField)
}

// generateEnumConversion generates conversion for enum types.
func (p *Plugin) generateEnumConversion(
	typeRef *resolution.TypeRef,
	_ *translatorTemplateData,
	goField, pbField string,
) (forward, backward string) {
	enumName := typeRef.EnumRef.Name

	return fmt.Sprintf("translate%sForward(%s)", enumName, goField),
		fmt.Sprintf("translate%sBackward(%s)", enumName, pbField)
}

// generateArrayConversion generates conversion for array types.
func (p *Plugin) generateArrayConversion(
	field resolution.Field,
	data *translatorTemplateData,
	goField, pbField string,
) (forward, backward string) {
	typeRef := field.TypeRef

	// For struct arrays, use slice helper
	if typeRef.Kind == resolution.TypeKindStruct && typeRef.StructRef != nil {
		structName := typeRef.StructRef.Name
		return fmt.Sprintf("Translate%ssForward(ctx, %s)", structName, goField),
			fmt.Sprintf("Translate%ssBackward(ctx, %s)", structName, pbField)
	}

	// For primitive arrays, use lo.Map or direct copy
	if typeRef.Kind == resolution.TypeKindPrimitive {
		switch typeRef.Primitive {
		case "uuid":
			data.imports.addExternal("github.com/google/uuid")
			data.imports.addExternal("github.com/samber/lo")
			return fmt.Sprintf("lo.Map(%s, func(u uuid.UUID, _ int) string { return u.String() })", goField),
				fmt.Sprintf("lo.Map(%s, func(s string, _ int) uuid.UUID { return uuid.MustParse(s) })", pbField)
		default:
			return goField, pbField
		}
	}

	return goField, pbField
}

// deriveTranslatorOutputPath derives the translator output path from @pb output.
// Example: "core/pkg/api/grpc/v1/rack" -> "core/pkg/api/grpc"
func deriveTranslatorOutputPath(pbOutput string) string {
	// Match pattern: .../grpc/v1/...
	re := regexp.MustCompile(`(.*/grpc)/v\d+/.*`)
	matches := re.FindStringSubmatch(pbOutput)
	if len(matches) >= 2 {
		return matches[1]
	}
	// Fallback: strip last two path components
	parts := strings.Split(pbOutput, "/")
	if len(parts) >= 3 {
		return strings.Join(parts[:len(parts)-2], "/")
	}
	return ""
}

// derivePackageName extracts the package name from the output path.
func derivePackageName(outputPath string) string {
	return filepath.Base(outputPath)
}

// derivePackageAlias creates a unique alias for an imported package.
func derivePackageAlias(outputPath, currentPackage string) string {
	base := filepath.Base(outputPath)
	if base == currentPackage {
		parent := filepath.Base(filepath.Dir(outputPath))
		return parent + base
	}
	return base
}

// getGoName gets the custom Go name from @go name annotation.
func getGoName(s resolution.Struct) string {
	if domain, ok := s.Domains["go"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
}

// toPascalCase converts snake_case to PascalCase.
func toPascalCase(s string) string {
	return lo.PascalCase(s)
}

// aliasTemplateData holds data for type alias file generation.
type aliasTemplateData struct {
	Package string
	Aliases []aliasData
	imports *importManager
}

// HasImports returns true if any imports are needed.
func (d *aliasTemplateData) HasImports() bool {
	return len(d.imports.external) > 0 || len(d.imports.internal) > 0
}

// InternalImports returns sorted internal imports.
func (d *aliasTemplateData) InternalImports() []internalImportData {
	imports := make([]internalImportData, 0, len(d.imports.internal))
	for _, imp := range d.imports.internal {
		imports = append(imports, internalImportData{
			Path:  imp.path,
			Alias: imp.alias,
		})
	}
	sort.Slice(imports, func(i, j int) bool { return imports[i].Path < imports[j].Path })
	return imports
}

// aliasData holds data for a single type alias.
type aliasData struct {
	Name    string
	AliasOf string
}

// translatorTemplateData holds data for translator file generation.
type translatorTemplateData struct {
	Package     string
	GroupName   string
	Namespace   string
	Translators []translatorData
	imports     *importManager
	repoRoot    string
	table       *resolution.Table
}

// HasImports returns true if any imports are needed.
func (d *translatorTemplateData) HasImports() bool {
	return len(d.imports.external) > 0 || len(d.imports.internal) > 0
}

// ExternalImports returns sorted external imports.
func (d *translatorTemplateData) ExternalImports() []string {
	imports := make([]string, 0, len(d.imports.external))
	for imp := range d.imports.external {
		imports = append(imports, imp)
	}
	sort.Strings(imports)
	return imports
}

// InternalImports returns sorted internal imports.
func (d *translatorTemplateData) InternalImports() []internalImportData {
	imports := make([]internalImportData, 0, len(d.imports.internal))
	for _, imp := range d.imports.internal {
		imports = append(imports, internalImportData{
			Path:  imp.path,
			Alias: imp.alias,
		})
	}
	sort.Slice(imports, func(i, j int) bool { return imports[i].Path < imports[j].Path })
	return imports
}

// translatorData holds data for a single type's translators.
type translatorData struct {
	Name           string
	GoType         string
	PBType         string
	GoTypeShort    string
	PBTypeShort    string
	Fields         []fieldTranslatorData
	OptionalFields []fieldTranslatorData
}

// fieldTranslatorData holds data for a single field translation.
type fieldTranslatorData struct {
	GoName           string
	PBName           string
	ForwardExpr      string
	BackwardExpr     string
	IsOptional       bool
	IsOptionalStruct bool // True if optional AND struct type (needs error handling)
}

// internalImportData holds data for an internal import.
type internalImportData struct {
	Path  string
	Alias string
}

// NeedsAlias returns true if the import needs an alias.
func (i internalImportData) NeedsAlias() bool {
	return i.Alias != "" && i.Alias != filepath.Base(i.Path)
}

// importManager tracks Go imports.
type importManager struct {
	external map[string]bool
	internal map[string]*internalImport
}

type internalImport struct {
	path  string
	alias string
}

func newImportManager() *importManager {
	return &importManager{
		external: make(map[string]bool),
		internal: make(map[string]*internalImport),
	}
}

func (m *importManager) addExternal(path string) {
	m.external[path] = true
}

func (m *importManager) addInternal(alias, path string) {
	m.internal[alias] = &internalImport{path: path, alias: alias}
}
