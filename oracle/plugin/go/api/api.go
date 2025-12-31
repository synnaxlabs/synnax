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

		// Skip generic types - they can't be aliased without instantiation
		if s.IsGeneric() {
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

		// Note: We no longer skip generic types - they get generic translator functions
		// that accept converter functions for type parameters (like TypeScript's Zod pattern)

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
		Package:            derivePackageName(outputPath),
		GroupName:          groupName,
		Namespace:          namespace,
		Translators:        make([]translatorData, 0, len(structs)),
		GenericTranslators: make([]genericTranslatorData, 0),
		EnumTranslators:    make([]enumTranslatorData, 0),
		AnyHelpers:         make([]anyHelperData, 0),
		imports:            newImportManager(),
		repoRoot:           req.RepoRoot,
		table:              req.Resolutions,
		usedEnums:          make(map[string]*resolution.Enum),
		generatedAnyHelpers: make(map[string]bool),
	}

	// Always need context import
	data.imports.addExternal("context")

	for _, s := range structs {
		if s.IsGeneric() {
			// Generate generic translator with type parameters
			genericTranslator, err := p.processGenericStructForTranslation(s, data, req)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to process generic struct %s", s.Name)
			}
			if genericTranslator != nil {
				data.GenericTranslators = append(data.GenericTranslators, *genericTranslator)
			}
		} else {
			// Generate regular translator
			translator, err := p.processStructForTranslation(s, data, req)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to process struct %s", s.Name)
			}
			if translator != nil {
				data.Translators = append(data.Translators, *translator)
			}
		}
	}

	if len(data.Translators) == 0 && len(data.GenericTranslators) == 0 {
		return nil, nil
	}

	// Generate enum translators for all used enums
	for _, enumRef := range data.usedEnums {
		enumTranslator, err := p.generateEnumTranslator(enumRef, data, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate enum translator for %s", enumRef.Name)
		}
		if enumTranslator != nil {
			data.EnumTranslators = append(data.EnumTranslators, *enumTranslator)
		}
	}

	var buf bytes.Buffer
	if err := translatorFileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// generateEnumTranslator generates translator data for an enum.
func (p *Plugin) generateEnumTranslator(
	enumRef *resolution.Enum,
	data *translatorTemplateData,
	req *plugin.Request,
) (*enumTranslatorData, error) {
	goOutput := output.GetEnumPath(*enumRef, "go")
	pbOutput := output.GetEnumPath(*enumRef, "pb")

	if goOutput == "" || pbOutput == "" {
		return nil, nil
	}

	// Resolve Go import path for enum
	goImportPath, err := resolveGoImportPath(goOutput, req.RepoRoot)
	if err != nil {
		return nil, err
	}

	// Resolve proto import path
	pbImportPath, err := resolveGoImportPath(pbOutput, req.RepoRoot)
	if err != nil {
		return nil, err
	}

	// Get package aliases
	goAlias := derivePackageName(goOutput)
	pbAlias := derivePackageAlias(pbOutput, data.Package) + "v1"

	data.imports.addInternal(goAlias, goImportPath)
	data.imports.addInternal(pbAlias, pbImportPath)

	// Build enum value translations
	values := make([]enumValueTranslatorData, 0, len(enumRef.Values))
	var goDefault string

	for i, v := range enumRef.Values {
		valueName := toPascalCase(v.Name)

		// Go enum value format: depends on whether it's hand-written or generated
		// For hand-written enums like status.Variant: <Value>Variant (e.g., SuccessVariant)
		// For generated enums: <Enum><Value> (e.g., VariantSuccess)
		// Check if Go is omitted to determine pattern
		isGoOmitted := omit.IsEnum(*enumRef, "go")
		var goValue string
		if isGoOmitted {
			// Hand-written Go enum - try common patterns
			// Pattern 1: <Value><Enum> (e.g., SuccessVariant)
			goValue = fmt.Sprintf("%s.%s%s", goAlias, valueName, enumRef.Name)
		} else {
			// Generated Go enum: <Enum><Value>
			goValue = fmt.Sprintf("%s.%s%s", goAlias, enumRef.Name, valueName)
		}

		// Proto enum value format: PB<Enum>_PB_<ENUM>_<VALUE>
		pbEnumName := "PB" + enumRef.Name
		pbValueName := fmt.Sprintf("%s_PB_%s_%s", pbEnumName, toScreamingSnake(enumRef.Name), toScreamingSnake(v.Name))
		pbValue := fmt.Sprintf("%s.%s", pbAlias, pbValueName)

		values = append(values, enumValueTranslatorData{
			GoValue: goValue,
			PBValue: pbValue,
		})

		if i == 0 {
			goDefault = goValue
		}
	}

	// Proto default is always UNSPECIFIED
	pbDefault := fmt.Sprintf("%s.PB%s_PB_%s_UNSPECIFIED", pbAlias, enumRef.Name, toScreamingSnake(enumRef.Name))

	return &enumTranslatorData{
		Name:      enumRef.Name,
		GoType:    fmt.Sprintf("%s.%s", goAlias, enumRef.Name),
		PBType:    fmt.Sprintf("%s.PB%s", pbAlias, enumRef.Name),
		Values:    values,
		PBDefault: pbDefault,
		GoDefault: goDefault,
	}, nil
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

	// Check if Go types are omitted (meaning they're hand-written, not generated)
	isGoOmitted := omit.IsStruct(s, "go")

	// Determine the Go type location:
	// - If @go omit is set, use @go output (the hand-written type location)
	// - If @api output != @go output, the API layer has its own type (use @api output)
	// - Otherwise, use @go output
	var goTypeOutput string
	if isGoOmitted {
		// Hand-written Go types - use @go output
		goTypeOutput = goOutput
	} else if apiOutput != "" && apiOutput != goOutput {
		// Generated API layer type
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

	// Check if any field uses context
	translator.UsesContext = fieldUsesContext(translator.Fields) || fieldUsesContext(translator.OptionalFields)

	return translator, nil
}

// processGenericStructForTranslation processes a generic struct and generates translator data
// with type parameters. This creates translator functions that accept converter functions
// for each type parameter, following the TypeScript Zod pattern:
//
//	func TranslateStatusForward[D any](ctx, s, translateD func(D) *anypb.Any) *PBStatus
func (p *Plugin) processGenericStructForTranslation(
	s resolution.Struct,
	data *translatorTemplateData,
	req *plugin.Request,
) (*genericTranslatorData, error) {
	goOutput := output.GetPath(s, "go")
	pbOutput := output.GetPath(s, "pb")

	if goOutput == "" || pbOutput == "" {
		return nil, nil
	}

	// Check if Go types are omitted (meaning they're hand-written, not generated)
	isGoOmitted := omit.IsStruct(s, "go")

	// Determine the Go type location
	var goTypeOutput string
	if isGoOmitted {
		goTypeOutput = goOutput
	} else {
		goTypeOutput = goOutput
	}

	// Derive the translator output path
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
		goTypePrefix = ""
	} else {
		goAlias = derivePackageAlias(goTypeOutput, data.Package)
		data.imports.addInternal(goAlias, goImportPath)
		goTypePrefix = goAlias + "."
	}

	pbAlias := derivePackageAlias(pbOutput, data.Package) + "v1"
	data.imports.addInternal(pbAlias, pbImportPath)

	// Need anypb for generic type parameters
	data.imports.addExternal("google.golang.org/protobuf/types/known/anypb")

	// Get type names
	goName := getGoName(s)
	if goName == "" {
		goName = s.Name
	}
	pbName := "PB" + s.Name

	// Build type parameters
	typeParams := make([]typeParamData, 0, len(s.TypeParams))
	for _, tp := range s.TypeParams {
		typeParams = append(typeParams, typeParamData{
			Name:       tp.Name,
			Constraint: "any", // Go uses 'any' as the unconstrained type
		})
	}

	// Build Go type with type parameters: "status.Status[D]"
	typeParamNames := make([]string, len(typeParams))
	for i, tp := range typeParams {
		typeParamNames[i] = tp.Name
	}
	goTypeWithParams := fmt.Sprintf("%s%s[%s]", goTypePrefix, goName, strings.Join(typeParamNames, ", "))

	translator := &genericTranslatorData{
		Name:            s.Name,
		GoType:          goTypeWithParams,
		GoTypeBase:      goTypePrefix + goName,
		PBType:          fmt.Sprintf("%s.%s", pbAlias, pbName),
		GoTypeShort:     goName,
		PBTypeShort:     pbName,
		TypeParams:      typeParams,
		Fields:          make([]fieldTranslatorData, 0),
		TypeParamFields: make([]fieldTranslatorData, 0),
		OptionalFields:  make([]fieldTranslatorData, 0),
	}

	// Process fields - need special handling for type parameter references
	for _, field := range s.UnifiedFields() {
		fieldData, isTypeParam := p.processGenericFieldForTranslation(field, data, s, typeParams)
		if isTypeParam {
			// Type param fields need error handling, put them in TypeParamFields
			translator.TypeParamFields = append(translator.TypeParamFields, fieldData)
		} else if fieldData.IsOptional {
			translator.OptionalFields = append(translator.OptionalFields, fieldData)
		} else {
			translator.Fields = append(translator.Fields, fieldData)
		}
	}

	// Check if any field uses context
	translator.UsesContext = fieldUsesContext(translator.Fields) ||
		fieldUsesContext(translator.OptionalFields) ||
		fieldUsesContext(translator.TypeParamFields)

	return translator, nil
}

// processGenericFieldForTranslation processes a field in a generic struct.
// If the field's type is a type parameter, it uses the provided converter function.
// Returns the field data and a boolean indicating if it's a type parameter field.
func (p *Plugin) processGenericFieldForTranslation(
	field resolution.Field,
	data *translatorTemplateData,
	parentStruct resolution.Struct,
	typeParams []typeParamData,
) (fieldTranslatorData, bool) {
	goName := toPascalCase(field.Name)
	pbName := toPascalCase(field.Name)
	typeRef := field.TypeRef

	isHardOptional := field.TypeRef.IsHardOptional
	isOptional := isHardOptional

	goFieldName := "r." + goName
	pbFieldName := "pb." + pbName

	// Check if this field's type is a type parameter
	if typeRef.Kind == resolution.TypeKindTypeParam && typeRef.TypeParamRef != nil {
		paramName := typeRef.TypeParamRef.Name
		// Use the converter function: translateD(ctx, r.Details)
		converterFunc := fmt.Sprintf("translate%s", paramName)

		forwardExpr := fmt.Sprintf("%s(ctx, %s)", converterFunc, goFieldName)
		backwardExpr := fmt.Sprintf("%s(ctx, %s)", converterFunc, pbFieldName)

		return fieldTranslatorData{
			GoName:           goName,
			PBName:           pbName,
			ForwardExpr:      forwardExpr,
			BackwardExpr:     backwardExpr,
			IsOptional:       isOptional,
			IsOptionalStruct: false,
		}, true // This is a type param field
	}

	// For non-type-param fields, use the regular field processing
	forwardExpr, backwardExpr, backwardCast := p.generateFieldConversion(field, data, parentStruct)

	return fieldTranslatorData{
		GoName:           goName,
		PBName:           pbName,
		ForwardExpr:      forwardExpr,
		BackwardExpr:     backwardExpr,
		BackwardCast:     backwardCast,
		IsOptional:       isOptional,
		IsOptionalStruct: isOptional && typeRef.Kind == resolution.TypeKindStruct,
	}, false // Not a type param field
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

	forwardExpr, backwardExpr, backwardCast := p.generateFieldConversion(field, data, parentStruct)

	return fieldTranslatorData{
		GoName:           goName,
		PBName:           pbName,
		ForwardExpr:      forwardExpr,
		BackwardExpr:     backwardExpr,
		BackwardCast:     backwardCast,
		IsOptional:       isOptional,
		IsOptionalStruct: isOptionalStruct,
	}
}

// generateFieldConversion generates the forward and backward conversion expressions.
// Returns forward expr, backward expr, and optional backward cast for generic struct aliases.
func (p *Plugin) generateFieldConversion(
	field resolution.Field,
	data *translatorTemplateData,
	parentStruct resolution.Struct,
) (forward, backward, backwardCast string) {
	typeRef := field.TypeRef
	fieldName := toPascalCase(field.Name)
	goFieldName := "r." + fieldName
	pbFieldName := "pb." + fieldName

	// Check if this is a @key field - needs type conversion to/from package Key type
	isKeyField := hasKeyDomain(field)

	// Handle arrays
	if typeRef.IsArray {
		f, b := p.generateArrayConversion(field, data, goFieldName, pbFieldName)
		return f, b, ""
	}

	// Handle @key fields with typed wrappers (e.g., rack.Key -> uint32)
	// Only applies to numeric types that get wrapped, not uuid which has its own conversion
	if isKeyField && typeRef.Kind == resolution.TypeKindPrimitive && isNumericPrimitive(typeRef.Primitive) {
		goOutput := output.GetPath(parentStruct, "go")
		pkgName := derivePackageName(goOutput)
		protoType := primitiveToProtoType(typeRef.Primitive)
		// Forward: uint32(r.Key), Backward: rack.Key(pb.Key)
		return fmt.Sprintf("%s(%s)", protoType, goFieldName),
			fmt.Sprintf("%s.Key(%s)", pkgName, pbFieldName), ""
	}

	// Handle primitives
	// Note: Soft optional (?) types are regular fields in proto, only hard optional (??)
	// types are optional in proto. So we don't need special handling for soft optional.
	if typeRef.Kind == resolution.TypeKindPrimitive {
		forward, backward := p.generatePrimitiveConversion(typeRef.Primitive, goFieldName, pbFieldName, data)
		return forward, backward, ""
	}

	// Handle struct references
	if typeRef.Kind == resolution.TypeKindStruct && typeRef.StructRef != nil {
		return p.generateStructConversion(typeRef, data, goFieldName, pbFieldName)
	}

	// Handle enums
	if typeRef.Kind == resolution.TypeKindEnum && typeRef.EnumRef != nil {
		f, b := p.generateEnumConversion(typeRef, data, goFieldName, pbFieldName)
		return f, b, ""
	}

	// Default: direct copy
	return goFieldName, pbFieldName, ""
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
// Returns forward expression, backward expression, and optional backward cast.
func (p *Plugin) generateStructConversion(
	typeRef *resolution.TypeRef,
	data *translatorTemplateData,
	goField, pbField string,
) (forward, backward, backwardCast string) {
	structRef := typeRef.StructRef

	// Follow alias chain to find the actual underlying struct and collect type args
	actualStruct := structRef
	var typeArgs []*resolution.TypeRef

	// If this is an alias, get the type args from the alias's AliasOf
	if structRef.IsAlias() && structRef.AliasOf != nil {
		typeArgs = structRef.AliasOf.TypeArgs
		if structRef.AliasOf.StructRef != nil {
			actualStruct = structRef.AliasOf.StructRef
		}
	}

	// Continue following the chain if needed
	for actualStruct.IsAlias() && actualStruct.AliasOf != nil && actualStruct.AliasOf.StructRef != nil {
		if len(typeArgs) == 0 && len(actualStruct.AliasOf.TypeArgs) > 0 {
			typeArgs = actualStruct.AliasOf.TypeArgs
		}
		actualStruct = actualStruct.AliasOf.StructRef
	}

	// Handle generic types with concrete type arguments
	if actualStruct.IsGeneric() && len(typeArgs) > 0 {
		return p.generateGenericStructConversion(typeRef, actualStruct, typeArgs, data, goField, pbField)
	}

	// For generic types without type args, fall back to direct assignment
	if actualStruct.IsGeneric() {
		return goField, pbField, ""
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
		// Hard optional: field is a pointer, dereference to pass value
		// These will be used inside if-blocks that handle nil checks
		return fmt.Sprintf("%s%sToPB(ctx, *%s)", translatorPrefix, structName, goField),
			fmt.Sprintf("%s%sFromPB(ctx, %s)", translatorPrefix, structName, pbField), ""
	}

	// For non-optional or soft optional struct fields - pass value directly
	return fmt.Sprintf("%s%sToPB(ctx, %s)", translatorPrefix, structName, goField),
		fmt.Sprintf("%s%sFromPB(ctx, %s)", translatorPrefix, structName, pbField), ""
}

// generateGenericStructConversion generates conversion for generic struct types
// with concrete type arguments. It calls the generic translator with appropriate
// converter functions for each type parameter.
// Returns forward expression, backward expression, and backward cast (for type aliases).
func (p *Plugin) generateGenericStructConversion(
	typeRef *resolution.TypeRef,
	actualStruct *resolution.Struct,
	typeArgs []*resolution.TypeRef,
	data *translatorTemplateData,
	goField, pbField string,
) (forward, backward, backwardCast string) {
	structName := actualStruct.Name

	// Find the translator package for the generic struct
	translatorPrefix := ""
	if actualStruct.Namespace != data.Namespace {
		pbOutput := output.GetPath(*actualStruct, "pb")
		if pbOutput != "" {
			translatorPath := deriveTranslatorOutputPath(pbOutput)
			if translatorPath != "" {
				importPath, err := resolveGoImportPath(translatorPath, data.repoRoot)
				if err == nil {
					alias := strings.ToLower(actualStruct.Namespace) + "grpc"
					data.imports.addInternal(alias, importPath)
					translatorPrefix = alias + "."
				}
			}
		}
	}

	// Build converter function arguments and explicit type args for each type arg
	var forwardConverters, backwardConverters []string
	var explicitTypeArgs []string
	for _, typeArg := range typeArgs {
		if typeArg.Kind == resolution.TypeKindStruct && typeArg.StructRef != nil {
			argStruct := typeArg.StructRef
			argName := argStruct.Name

			// Track and generate the Any helper for this type
			p.ensureAnyHelper(argStruct, data)

			// Add converter function calls
			forwardConverters = append(forwardConverters, fmt.Sprintf("%sToPBAny", argName))
			backwardConverters = append(backwardConverters, fmt.Sprintf("%sFromPBAny", argName))

			// Build explicit type arg - always fully qualify since we're generating
			// in a different package (grpc) than where the types are defined
			goOutput := output.GetPath(*argStruct, "go")
			if goOutput != "" {
				importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
				if err == nil {
					alias := derivePackageAlias(goOutput, data.Package)
					data.imports.addInternal(alias, importPath)
					explicitTypeArgs = append(explicitTypeArgs, fmt.Sprintf("%s.%s", alias, argName))
				} else {
					explicitTypeArgs = append(explicitTypeArgs, argName)
				}
			} else {
				explicitTypeArgs = append(explicitTypeArgs, argName)
			}
		} else {
			// For non-struct type args (primitives, etc.), we'd need different handling
			forwardConverters = append(forwardConverters, "nil")
			backwardConverters = append(backwardConverters, "nil")
			explicitTypeArgs = append(explicitTypeArgs, "any")
		}
	}

	forwardArgs := strings.Join(forwardConverters, ", ")
	backwardArgs := strings.Join(backwardConverters, ", ")
	typeArgsStr := "[" + strings.Join(explicitTypeArgs, ", ") + "]"

	// Build the Go type for casting (e.g., status.Status[rack.StatusDetails])
	// Need to import the generic struct's package
	var genericGoType string
	goOutput := output.GetPath(*actualStruct, "go")
	if goOutput != "" {
		importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
		if err == nil {
			alias := derivePackageAlias(goOutput, data.Package)
			data.imports.addInternal(alias, importPath)
			genericGoType = fmt.Sprintf("%s.%s[%s]", alias, structName, strings.Join(explicitTypeArgs, ", "))
		}
	}

	// Build the backward cast - we need to cast the result back to the alias type
	// e.g., (*rack.Status) to cast *status.Status[rack.StatusDetails] back to *rack.Status
	if typeRef.StructRef != nil && typeRef.StructRef.IsAlias() {
		// Get the alias type's package and name
		// For type aliases (Status = status.Status<D>), they may not have @go output
		// In that case, look up a sibling struct in the same namespace to get the package
		aliasGoOutput := output.GetPath(*typeRef.StructRef, "go")
		if aliasGoOutput == "" {
			// Try to find a sibling struct in the same namespace with @go output
			for _, sibling := range data.table.AllStructs() {
				if sibling.Namespace == typeRef.StructRef.Namespace && !sibling.IsAlias() {
					aliasGoOutput = output.GetPath(sibling, "go")
					if aliasGoOutput != "" {
						break
					}
				}
			}
		}
		if aliasGoOutput != "" {
			importPath, err := resolveGoImportPath(aliasGoOutput, data.repoRoot)
			if err == nil {
				alias := derivePackageAlias(aliasGoOutput, data.Package)
				data.imports.addInternal(alias, importPath)
				aliasName := getGoName(*typeRef.StructRef)
				if aliasName == "" {
					aliasName = typeRef.StructRef.Name
				}
				backwardCast = fmt.Sprintf("(*%s.%s)", alias, aliasName)
			}
		}
	}

	// Build the call with explicit type args, converters, and casts
	// We need to cast because type aliases don't work with generic type inference
	if typeRef.IsHardOptional {
		// Hard optional: field is a pointer, dereference to pass value
		if genericGoType != "" {
			forward = fmt.Sprintf("%s%sToPB%s(ctx, (%s)(*%s), %s)", translatorPrefix, structName, typeArgsStr, genericGoType, goField, forwardArgs)
		} else {
			forward = fmt.Sprintf("%s%sToPB%s(ctx, *%s, %s)", translatorPrefix, structName, typeArgsStr, goField, forwardArgs)
		}
		backward = fmt.Sprintf("%s%sFromPB%s(ctx, %s, %s)", translatorPrefix, structName, typeArgsStr, pbField, backwardArgs)
	} else {
		// Non-optional: pass value directly
		if genericGoType != "" {
			forward = fmt.Sprintf("%s%sToPB%s(ctx, (%s)(%s), %s)", translatorPrefix, structName, typeArgsStr, genericGoType, goField, forwardArgs)
		} else {
			forward = fmt.Sprintf("%s%sToPB%s(ctx, %s, %s)", translatorPrefix, structName, typeArgsStr, goField, forwardArgs)
		}
		backward = fmt.Sprintf("%s%sFromPB%s(ctx, %s, %s)", translatorPrefix, structName, typeArgsStr, pbField, backwardArgs)
	}

	return forward, backward, backwardCast
}

// ensureAnyHelper tracks that we need to generate toAny/fromAny helpers for a type.
func (p *Plugin) ensureAnyHelper(s *resolution.Struct, data *translatorTemplateData) {
	if s == nil {
		return
	}

	key := s.QualifiedName
	if data.generatedAnyHelpers[key] {
		return
	}
	data.generatedAnyHelpers[key] = true

	// Get Go and PB type info
	goOutput := output.GetPath(*s, "go")
	pbOutput := output.GetPath(*s, "pb")

	if goOutput == "" || pbOutput == "" {
		return
	}

	// Resolve import paths
	goImportPath, err := resolveGoImportPath(goOutput, data.repoRoot)
	if err != nil {
		return
	}
	pbImportPath, err := resolveGoImportPath(pbOutput, data.repoRoot)
	if err != nil {
		return
	}

	// Add imports
	goAlias := derivePackageAlias(goOutput, data.Package)
	pbAlias := derivePackageAlias(pbOutput, data.Package) + "v1"

	data.imports.addInternal(goAlias, goImportPath)
	data.imports.addInternal(pbAlias, pbImportPath)
	data.imports.addExternal("google.golang.org/protobuf/types/known/anypb")

	goName := getGoName(*s)
	if goName == "" {
		goName = s.Name
	}

	data.AnyHelpers = append(data.AnyHelpers, anyHelperData{
		TypeName: s.Name,
		GoType:   fmt.Sprintf("%s.%s", goAlias, goName),
		PBType:   fmt.Sprintf("%s.PB%s", pbAlias, s.Name),
	})
}

// generateEnumConversion generates conversion for enum types.
func (p *Plugin) generateEnumConversion(
	typeRef *resolution.TypeRef,
	data *translatorTemplateData,
	goField, pbField string,
) (forward, backward string) {
	enumRef := typeRef.EnumRef
	enumName := enumRef.Name

	// Track this enum for translator generation
	if _, exists := data.usedEnums[enumRef.QualifiedName]; !exists {
		data.usedEnums[enumRef.QualifiedName] = enumRef
	}

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
		return fmt.Sprintf("%ssToPB(ctx, %s)", structName, goField),
			fmt.Sprintf("%ssFromPB(ctx, %s)", structName, pbField)
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

// toScreamingSnake converts a name to SCREAMING_SNAKE_CASE.
func toScreamingSnake(s string) string {
	return strings.ToUpper(lo.SnakeCase(s))
}

// fieldUsesContext checks if any field expression contains a ctx reference.
func fieldUsesContext(fields []fieldTranslatorData) bool {
	for _, f := range fields {
		if strings.Contains(f.ForwardExpr, "ctx") || strings.Contains(f.BackwardExpr, "ctx") {
			return true
		}
	}
	return false
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
	Package            string
	GroupName          string
	Namespace          string
	Translators        []translatorData
	GenericTranslators []genericTranslatorData
	EnumTranslators    []enumTranslatorData
	AnyHelpers         []anyHelperData
	imports            *importManager
	repoRoot           string
	table              *resolution.Table
	usedEnums          map[string]*resolution.Enum // tracks which enums are used
	generatedAnyHelpers map[string]bool            // tracks which toAny/fromAny helpers we've generated
}

// genericTranslatorData holds data for a generic type's translators.
// These are translator functions with type parameters that accept converter
// functions for each type parameter, matching the TypeScript Zod pattern:
//
//	func TranslateStatusForward[D any](ctx, s, translateD func(D) (*anypb.Any, error)) (*PBStatus, error)
type genericTranslatorData struct {
	Name            string
	GoType          string // e.g., "status.Status[D]"
	GoTypeBase      string // e.g., "status.Status" (without type params)
	PBType          string
	GoTypeShort     string
	PBTypeShort     string
	TypeParams      []typeParamData
	Fields          []fieldTranslatorData // Regular fields (non-type-param)
	TypeParamFields []fieldTranslatorData // Fields that use type parameters (need error handling)
	OptionalFields  []fieldTranslatorData
	UsesContext     bool // True if any field conversion uses ctx
}

// typeParamData holds data for a type parameter in a generic translator.
type typeParamData struct {
	Name       string // e.g., "D"
	Constraint string // e.g., "any" (Go constraint)
}

// anyHelperData holds data for ToPBAny/FromPBAny helper functions.
// These are generated for concrete types that are used as type arguments
// to generic structs, enabling the generic translator to marshal/unmarshal
// the concrete type through protobuf Any.
type anyHelperData struct {
	TypeName string // e.g., "StatusDetails"
	GoType   string // e.g., "rack.StatusDetails"
	PBType   string // e.g., "pbv1.PBStatusDetails"
}

// enumTranslatorData holds data for enum translator functions.
type enumTranslatorData struct {
	Name       string
	GoType     string
	PBType     string
	Values     []enumValueTranslatorData
	PBDefault  string // The default proto enum value (UNSPECIFIED)
	GoDefault  string // The default Go enum value
}

// enumValueTranslatorData holds data for a single enum value translation.
type enumValueTranslatorData struct {
	GoValue string
	PBValue string
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
	UsesContext    bool // True if any field conversion uses ctx
}

// fieldTranslatorData holds data for a single field translation.
type fieldTranslatorData struct {
	GoName           string
	PBName           string
	ForwardExpr      string
	BackwardExpr     string
	BackwardCast     string // Optional cast for backward assignment (e.g., "(*rack.Status)")
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
