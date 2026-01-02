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
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin"
	gointernal "github.com/synnaxlabs/oracle/plugin/go/internal"
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

// Check verifies generated files are up-to-date. Currently unimplemented.
func (p *Plugin) Check(*plugin.Request) error { return nil }

// Generate produces API type aliases and translator functions from the analyzed schemas.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	// Collect struct types that have @api domain
	apiStructTypes := collectAPIStructTypes(req.Resolutions)

	// Generate type aliases for simple cases (structs with @api but no extends)
	aliasFiles, err := p.generateAliasFiles(apiStructTypes, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate alias files")
	}
	resp.Files = append(resp.Files, aliasFiles...)

	// Generate translators for structs that have both @api and @pb
	translatorFiles, err := p.generateTranslatorFiles(apiStructTypes, req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to generate translator files")
	}
	resp.Files = append(resp.Files, translatorFiles...)

	return resp, nil
}

// collectAPIStructTypes returns all struct types that have the @api domain.
func collectAPIStructTypes(table *resolution.Table) []resolution.Type {
	var result []resolution.Type
	for _, typ := range table.StructTypes() {
		if _, hasAPI := typ.Domains["api"]; hasAPI {
			result = append(result, typ)
		}
	}
	return result
}

// generateAliasFiles generates type alias files for structs that have @api but don't
// use extends (meaning they're just exposing service types at the API layer).
func (p *Plugin) generateAliasFiles(
	apiStructTypes []resolution.Type,
	req *plugin.Request,
) ([]plugin.File, error) {
	// Group types by their @api output path
	outputGroups := make(map[string][]resolution.Type)
	var outputOrder []string

	for _, typ := range apiStructTypes {
		// Skip if type is omitted
		if omit.IsType(typ, "api") {
			continue
		}

		form, ok := typ.Form.(resolution.StructForm)
		if !ok {
			continue
		}

		// Skip structs that use extends (go/types handles embedding)
		if form.Extends != nil {
			continue
		}

		// Skip generic types - they can't be aliased without instantiation
		if form.IsGeneric() {
			continue
		}

		apiOutput := output.GetPath(typ, "api")
		if apiOutput == "" {
			continue
		}

		// Get the Go output path for the aliased type
		goOutput := output.GetPath(typ, "go")
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
		outputGroups[apiOutput] = append(outputGroups[apiOutput], typ)
	}

	var files []plugin.File
	for _, outputPath := range outputOrder {
		types := outputGroups[outputPath]
		content, err := p.generateAliasFile(outputPath, types, req)
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
	types []resolution.Type,
	req *plugin.Request,
) ([]byte, error) {
	data := &aliasTemplateData{
		Package: gointernal.DerivePackageName(outputPath),
		Aliases: make([]aliasData, 0, len(types)),
		imports: gointernal.NewImportManager(),
	}

	for _, typ := range types {
		goOutput := output.GetPath(typ, "go")
		importPath, err := resolveGoImportPath(goOutput, req.RepoRoot)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to resolve import for %s", typ.Name)
		}

		alias := gointernal.DerivePackageAlias(goOutput, data.Package)
		data.imports.AddInternal(alias, importPath)

		// Check if there's a custom name via @go name "Name"
		goName := getGoName(typ)
		if goName == "" {
			goName = typ.Name
		}

		data.Aliases = append(data.Aliases, aliasData{
			Name:    typ.Name,
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
	apiStructTypes []resolution.Type,
	req *plugin.Request,
) ([]plugin.File, error) {
	// Group types by their translator output path (derived from @pb output)
	// and then by their pb output basename (e.g., rack, ranger)
	type groupKey struct {
		translatorPath string
		baseName       string
	}
	groups := make(map[groupKey][]resolution.Type)
	var groupOrder []groupKey

	for _, typ := range apiStructTypes {
		// Skip if type is omitted
		if omit.IsType(typ, "api") || omit.IsType(typ, "pb") {
			continue
		}

		// Note: We no longer skip generic types - they get generic translator functions
		// that accept converter functions for type parameters (like TypeScript's Zod pattern)

		// Need both @api and @pb for translator generation
		pbOutput := output.GetPath(typ, "pb")
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
		groups[key] = append(groups[key], typ)
	}

	var files []plugin.File
	for _, key := range groupOrder {
		types := groups[key]
		content, err := p.generateTranslatorFileForGroup(key.translatorPath, key.baseName, types, req)
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

// generateTranslatorFileForGroup generates translator content for a group of types.
func (p *Plugin) generateTranslatorFileForGroup(
	outputPath string,
	groupName string,
	types []resolution.Type,
	req *plugin.Request,
) ([]byte, error) {
	// Get namespace from first type in the group
	namespace := ""
	if len(types) > 0 {
		namespace = types[0].Namespace
	}

	data := &translatorTemplateData{
		Package:             gointernal.DerivePackageName(outputPath),
		GroupName:           groupName,
		Namespace:           namespace,
		Translators:         make([]translatorData, 0, len(types)),
		GenericTranslators:  make([]genericTranslatorData, 0),
		EnumTranslators:     make([]enumTranslatorData, 0),
		AnyHelpers:          make([]anyHelperData, 0),
		imports:             gointernal.NewImportManager(),
		repoRoot:            req.RepoRoot,
		table:               req.Resolutions,
		usedEnums:           make(map[string]*resolution.Type),
		generatedAnyHelpers: make(map[string]bool),
	}

	// Always need context import
	data.imports.AddExternal("context")

	for _, typ := range types {
		form, ok := typ.Form.(resolution.StructForm)
		if !ok {
			continue
		}

		if form.IsGeneric() {
			// Generate generic translator with type parameters
			genericTranslator, err := p.processGenericTypeForTranslation(typ, form, data, req)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to process generic struct %s", typ.Name)
			}
			if genericTranslator != nil {
				data.GenericTranslators = append(data.GenericTranslators, *genericTranslator)
			}
		} else {
			// Generate regular translator
			translator, err := p.processTypeForTranslation(typ, form, data, req)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to process struct %s", typ.Name)
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
	for _, enumType := range data.usedEnums {
		enumTranslator, err := p.generateEnumTranslator(enumType, data, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate enum translator for %s", enumType.Name)
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

// generateEnumTranslator generates translator data for an enum type.
func (p *Plugin) generateEnumTranslator(
	enumType *resolution.Type,
	data *translatorTemplateData,
	req *plugin.Request,
) (*enumTranslatorData, error) {
	form, ok := enumType.Form.(resolution.EnumForm)
	if !ok {
		return nil, nil
	}

	goOutput := output.GetPath(*enumType, "go")
	pbOutput := output.GetPath(*enumType, "pb")

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
	goAlias := gointernal.DerivePackageName(goOutput)
	pbAlias := gointernal.DerivePackageAlias(pbOutput, data.Package) + "v1"

	data.imports.AddInternal(goAlias, goImportPath)
	data.imports.AddInternal(pbAlias, pbImportPath)

	// Build enum value translations
	values := make([]enumValueTranslatorData, 0, len(form.Values))
	var goDefault string

	for i, v := range form.Values {
		valueName := gointernal.ToPascalCase(v.Name)

		// Go enum value format: depends on whether it's hand-written or generated
		// For hand-written enums like status.Variant: <Value>Variant (e.g., SuccessVariant)
		// For generated enums: <Enum><Value> (e.g., VariantSuccess)
		// Check if Go is omitted to determine pattern
		isGoOmitted := omit.IsType(*enumType, "go")
		var goValue string
		if isGoOmitted {
			// Hand-written Go enum - try common patterns
			// Pattern 1: <Value><Enum> (e.g., SuccessVariant)
			goValue = fmt.Sprintf("%s.%s%s", goAlias, valueName, enumType.Name)
		} else {
			// Generated Go enum: <Enum><Value>
			goValue = fmt.Sprintf("%s.%s%s", goAlias, enumType.Name, valueName)
		}

		// Proto enum value format: PB<Enum>_PB_<ENUM>_<VALUE>
		pbEnumName := "PB" + enumType.Name
		pbValueName := fmt.Sprintf("%s_PB_%s_%s", pbEnumName, toScreamingSnake(enumType.Name), toScreamingSnake(v.Name))
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
	pbDefault := fmt.Sprintf("%s.PB%s_PB_%s_UNSPECIFIED", pbAlias, enumType.Name, toScreamingSnake(enumType.Name))

	return &enumTranslatorData{
		Name:      enumType.Name,
		GoType:    fmt.Sprintf("%s.%s", goAlias, enumType.Name),
		PBType:    fmt.Sprintf("%s.PB%s", pbAlias, enumType.Name),
		Values:    values,
		PBDefault: pbDefault,
		GoDefault: goDefault,
	}, nil
}

// typeInfo holds resolved type information for a type being translated.
type typeInfo struct {
	goTypePrefix string // Package prefix for Go type (e.g., "rack." or "")
	pbAlias      string // Package alias for protobuf type (e.g., "rackv1")
	goName       string // Go type name
	pbName       string // Protobuf type name
}

// resolveTypeInfo resolves import paths and type prefixes for a type.
// Returns nil if the type cannot be translated (missing outputs).
func (p *Plugin) resolveTypeInfo(
	typ resolution.Type,
	data *translatorTemplateData,
	req *plugin.Request,
) (*typeInfo, error) {
	goOutput := output.GetPath(typ, "go")
	pbOutput := output.GetPath(typ, "pb")
	apiOutput := output.GetPath(typ, "api")
	if goOutput == "" || pbOutput == "" {
		return nil, nil
	}

	// Determine Go type location based on omit status and API output
	goTypeOutput := goOutput
	if !omit.IsType(typ, "go") && apiOutput != "" && apiOutput != goOutput {
		goTypeOutput = apiOutput
	}

	translatorOutputPath := deriveTranslatorOutputPath(pbOutput)
	goImportPath, err := resolveGoImportPath(goTypeOutput, req.RepoRoot)
	if err != nil {
		return nil, err
	}
	pbImportPath, err := resolveGoImportPath(pbOutput, req.RepoRoot)
	if err != nil {
		return nil, err
	}

	var goTypePrefix string
	if goTypeOutput != translatorOutputPath {
		alias := gointernal.DerivePackageAlias(goTypeOutput, data.Package)
		data.imports.AddInternal(alias, goImportPath)
		goTypePrefix = alias + "."
	}

	pbAlias := gointernal.DerivePackageAlias(pbOutput, data.Package) + "v1"
	data.imports.AddInternal(pbAlias, pbImportPath)

	goName := getGoName(typ)
	if goName == "" {
		goName = typ.Name
	}

	return &typeInfo{
		goTypePrefix: goTypePrefix,
		pbAlias:      pbAlias,
		goName:       goName,
		pbName:       "PB" + typ.Name,
	}, nil
}

// processTypeForTranslation processes a type and generates translator data.
func (p *Plugin) processTypeForTranslation(
	typ resolution.Type,
	form resolution.StructForm,
	data *translatorTemplateData,
	req *plugin.Request,
) (*translatorData, error) {
	info, err := p.resolveTypeInfo(typ, data, req)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}

	translator := &translatorData{
		Name:           typ.Name,
		GoType:         info.goTypePrefix + info.goName,
		PBType:         fmt.Sprintf("%s.%s", info.pbAlias, info.pbName),
		GoTypeShort:    info.goName,
		PBTypeShort:    info.pbName,
		Fields:         make([]fieldTranslatorData, 0),
		OptionalFields: make([]fieldTranslatorData, 0),
	}

	for _, field := range resolution.UnifiedFields(typ, data.table) {
		fieldData := p.processFieldForTranslation(field, data, typ)
		if fieldData.IsOptional {
			translator.OptionalFields = append(translator.OptionalFields, fieldData)
		} else {
			translator.Fields = append(translator.Fields, fieldData)
		}
	}

	translator.UsesContext = fieldUsesContext(translator.Fields) || fieldUsesContext(translator.OptionalFields)
	return translator, nil
}

// processGenericTypeForTranslation processes a generic type and generates translator data
// with type parameters. Creates translator functions that accept converter functions for each
// type parameter, following the TypeScript Zod pattern.
func (p *Plugin) processGenericTypeForTranslation(
	typ resolution.Type,
	form resolution.StructForm,
	data *translatorTemplateData,
	req *plugin.Request,
) (*genericTranslatorData, error) {
	info, err := p.resolveTypeInfo(typ, data, req)
	if err != nil {
		return nil, err
	}
	if info == nil {
		return nil, nil
	}

	data.imports.AddExternal("google.golang.org/protobuf/types/known/anypb")

	// Build type parameters
	typeParams := make([]typeParamData, 0, len(form.TypeParams))
	typeParamNames := make([]string, 0, len(form.TypeParams))
	for _, tp := range form.TypeParams {
		typeParams = append(typeParams, typeParamData{Name: tp.Name, Constraint: "any"})
		typeParamNames = append(typeParamNames, tp.Name)
	}

	goTypeWithParams := fmt.Sprintf("%s%s[%s]", info.goTypePrefix, info.goName, strings.Join(typeParamNames, ", "))

	translator := &genericTranslatorData{
		Name:            typ.Name,
		GoType:          goTypeWithParams,
		GoTypeBase:      info.goTypePrefix + info.goName,
		PBType:          fmt.Sprintf("%s.%s", info.pbAlias, info.pbName),
		GoTypeShort:     info.goName,
		PBTypeShort:     info.pbName,
		TypeParams:      typeParams,
		Fields:          make([]fieldTranslatorData, 0),
		TypeParamFields: make([]fieldTranslatorData, 0),
		OptionalFields:  make([]fieldTranslatorData, 0),
	}

	for _, field := range resolution.UnifiedFields(typ, data.table) {
		fieldData, isTypeParam := p.processGenericFieldForTranslation(field, data, typ, form, typeParams)
		if isTypeParam {
			translator.TypeParamFields = append(translator.TypeParamFields, fieldData)
		} else if fieldData.IsOptional {
			translator.OptionalFields = append(translator.OptionalFields, fieldData)
		} else {
			translator.Fields = append(translator.Fields, fieldData)
		}
	}

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
	parentType resolution.Type,
	parentForm resolution.StructForm,
	typeParams []typeParamData,
) (fieldTranslatorData, bool) {
	goName := gointernal.ToPascalCase(field.Name)
	pbName := gointernal.ToPascalCase(field.Name)
	typeRef := field.Type

	isHardOptional := field.IsHardOptional
	isOptional := isHardOptional

	goFieldName := "r." + goName
	pbFieldName := "pb." + pbName

	// Check if this field's type is a type parameter
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		paramName := typeRef.TypeParam.Name
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
	forwardExpr, backwardExpr, backwardCast := p.generateFieldConversion(field, data, parentType)

	// Determine if this is a struct type for optional handling
	isStructType := false
	if !typeRef.IsTypeParam() && typeRef.Name != "" {
		if resolved, ok := data.table.Get(typeRef.Name); ok {
			_, isStructType = resolved.Form.(resolution.StructForm)
		}
	}

	return fieldTranslatorData{
		GoName:           goName,
		PBName:           pbName,
		ForwardExpr:      forwardExpr,
		BackwardExpr:     backwardExpr,
		BackwardCast:     backwardCast,
		IsOptional:       isOptional,
		IsOptionalStruct: isOptional && isStructType,
	}, false // Not a type param field
}

// processFieldForTranslation processes a field for translation.
func (p *Plugin) processFieldForTranslation(
	field resolution.Field,
	data *translatorTemplateData,
	parentType resolution.Type,
) fieldTranslatorData {
	goName := gointernal.ToPascalCase(field.Name)
	pbName := gointernal.ToPascalCase(field.Name)

	// Only hard optional (??) results in a pointer in Go
	// Soft optional (?) results in a value type in Go but optional in proto
	isHardOptional := field.IsHardOptional

	// For template: fields in OptionalFields get nil checks
	// Only hard optional fields need nil checks (they're pointers in Go)
	isOptional := isHardOptional

	// Determine if this is a struct type for optional handling
	typeRef := field.Type
	isStructType := false
	if !typeRef.IsTypeParam() && typeRef.Name != "" {
		if resolved, ok := data.table.Get(typeRef.Name); ok {
			_, isStructType = resolved.Form.(resolution.StructForm)
		}
	}

	// If this is a @key field, we need to import the package for type conversion
	if hasKeyDomain(field) {
		goOutput := output.GetPath(parentType, "go")
		if goOutput != "" {
			importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
			if err == nil {
				pkgAlias := gointernal.DerivePackageName(goOutput)
				data.imports.AddInternal(pkgAlias, importPath)
			}
		}
	}

	forwardExpr, backwardExpr, backwardCast := p.generateFieldConversion(field, data, parentType)

	return fieldTranslatorData{
		GoName:           goName,
		PBName:           pbName,
		ForwardExpr:      forwardExpr,
		BackwardExpr:     backwardExpr,
		BackwardCast:     backwardCast,
		IsOptional:       isOptional,
		IsOptionalStruct: isOptional && isStructType,
	}
}

// generateFieldConversion generates the forward and backward conversion expressions.
// Returns forward expr, backward expr, and optional backward cast for generic struct aliases.
func (p *Plugin) generateFieldConversion(
	field resolution.Field,
	data *translatorTemplateData,
	parentType resolution.Type,
) (forward, backward, backwardCast string) {
	typeRef := field.Type
	fieldName := gointernal.ToPascalCase(field.Name)
	goFieldName := "r." + fieldName
	pbFieldName := "pb." + fieldName

	// Check if this is a @key field - needs type conversion to/from package Key type
	isKeyField := hasKeyDomain(field)

	// Handle arrays - check if type is Array
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		f, b := p.generateArrayConversion(field, typeRef.TypeArgs[0], data, goFieldName, pbFieldName)
		return f, b, ""
	}

	// Handle @key fields with typed wrappers (e.g., rack.Key -> uint32)
	// Only applies to numeric types that get wrapped, not uuid which has its own conversion
	if isKeyField && resolution.IsPrimitive(typeRef.Name) && isNumericPrimitive(typeRef.Name) {
		goOutput := output.GetPath(parentType, "go")
		pkgName := gointernal.DerivePackageName(goOutput)
		protoType := primitiveToProtoType(typeRef.Name)
		// Forward: uint32(r.Key), Backward: rack.Key(pb.Key)
		return fmt.Sprintf("%s(%s)", protoType, goFieldName),
			fmt.Sprintf("%s.Key(%s)", pkgName, pbFieldName), ""
	}

	// Handle primitives
	// Note: Soft optional (?) types are regular fields in proto, only hard optional (??)
	// types are optional in proto. So we don't need special handling for soft optional.
	if resolution.IsPrimitive(typeRef.Name) {
		forward, backward := p.generatePrimitiveConversion(typeRef.Name, goFieldName, pbFieldName, data)
		return forward, backward, ""
	}

	// Resolve the type to check its form
	resolved, ok := data.table.Get(typeRef.Name)
	if !ok {
		// Default: direct copy
		return goFieldName, pbFieldName, ""
	}

	// Handle struct references
	if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
		return p.generateStructConversion(typeRef, &resolved, field, data, goFieldName, pbFieldName)
	}

	// Handle alias forms that point to structs
	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		// Follow alias to target and check if it's a struct
		if targetType, targetOk := data.table.Get(aliasForm.Target.Name); targetOk {
			if _, isStruct := targetType.Form.(resolution.StructForm); isStruct {
				return p.generateStructConversion(typeRef, &resolved, field, data, goFieldName, pbFieldName)
			}
		}
	}

	// Handle enums
	if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
		f, b := p.generateEnumConversion(&resolved, data, goFieldName, pbFieldName)
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
		data.imports.AddExternal("github.com/google/uuid")
		return fmt.Sprintf("%s.String()", goField),
			fmt.Sprintf("uuid.MustParse(%s)", pbField)
	case "timestamp":
		data.imports.AddExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("int64(%s)", goField),
			fmt.Sprintf("telem.TimeStamp(%s)", pbField)
	case "timespan":
		data.imports.AddExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("int64(%s)", goField),
			fmt.Sprintf("telem.TimeSpan(%s)", pbField)
	case "time_range":
		data.imports.AddExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("telem.TranslateTimeRangeForward(%s)", goField),
			fmt.Sprintf("telem.TranslateTimeRangeBackward(%s)", pbField)
	case "json":
		data.imports.AddExternal("google.golang.org/protobuf/types/known/structpb")
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
	typeRef resolution.TypeRef,
	resolved *resolution.Type,
	field resolution.Field,
	data *translatorTemplateData,
	goField, pbField string,
) (forward, backward, backwardCast string) {
	// Follow alias chain to find the actual underlying struct and collect type args
	actualType := resolved
	var typeArgs []resolution.TypeRef

	// If this is an alias, get the type args from the alias's Target
	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		typeArgs = aliasForm.Target.TypeArgs
		if targetType, ok := data.table.Get(aliasForm.Target.Name); ok {
			actualType = &targetType
		}
	}

	// Continue following the chain if needed
	for {
		if aliasForm, isAlias := actualType.Form.(resolution.AliasForm); isAlias {
			if len(typeArgs) == 0 && len(aliasForm.Target.TypeArgs) > 0 {
				typeArgs = aliasForm.Target.TypeArgs
			}
			if targetType, ok := data.table.Get(aliasForm.Target.Name); ok {
				actualType = &targetType
			} else {
				break
			}
		} else {
			break
		}
	}

	// Check if actual type is a struct
	actualForm, isStruct := actualType.Form.(resolution.StructForm)
	if !isStruct {
		return goField, pbField, ""
	}

	// Handle generic types with concrete type arguments
	if actualForm.IsGeneric() && len(typeArgs) > 0 {
		return p.generateGenericStructConversion(typeRef, resolved, actualType, &actualForm, typeArgs, field, data, goField, pbField)
	}

	// For generic types without type args, fall back to direct assignment
	if actualForm.IsGeneric() {
		return goField, pbField, ""
	}

	structName := actualType.Name

	// Check if referenced struct is from a different namespace
	translatorPrefix := ""
	if actualType.Namespace != data.Namespace {
		// Find the translator package for the referenced struct
		pbOutput := output.GetPath(*actualType, "pb")
		if pbOutput != "" {
			translatorPath := deriveTranslatorOutputPath(pbOutput)
			if translatorPath != "" {
				importPath, err := resolveGoImportPath(translatorPath, data.repoRoot)
				if err == nil {
					// Create a package alias for the translator
					alias := strings.ToLower(actualType.Namespace) + "grpc"
					data.imports.AddInternal(alias, importPath)
					translatorPrefix = alias + "."
				}
			}
		}
	}

	// Only hard optional (??) fields are pointers in Go
	// Soft optional (?) and non-optional are value types
	if field.IsHardOptional {
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
	typeRef resolution.TypeRef,
	resolved *resolution.Type,
	actualType *resolution.Type,
	actualForm *resolution.StructForm,
	typeArgs []resolution.TypeRef,
	field resolution.Field,
	data *translatorTemplateData,
	goField, pbField string,
) (forward, backward, backwardCast string) {
	structName := actualType.Name

	// Find the translator package for the generic struct
	translatorPrefix := ""
	if actualType.Namespace != data.Namespace {
		pbOutput := output.GetPath(*actualType, "pb")
		if pbOutput != "" {
			translatorPath := deriveTranslatorOutputPath(pbOutput)
			if translatorPath != "" {
				importPath, err := resolveGoImportPath(translatorPath, data.repoRoot)
				if err == nil {
					alias := strings.ToLower(actualType.Namespace) + "grpc"
					data.imports.AddInternal(alias, importPath)
					translatorPrefix = alias + "."
				}
			}
		}
	}

	// Build converter function arguments and explicit type args for each type arg
	var forwardConverters, backwardConverters []string
	var explicitTypeArgs []string
	for _, typeArg := range typeArgs {
		argResolved, ok := data.table.Get(typeArg.Name)
		if !ok {
			forwardConverters = append(forwardConverters, "nil")
			backwardConverters = append(backwardConverters, "nil")
			explicitTypeArgs = append(explicitTypeArgs, "any")
			continue
		}

		if _, isStruct := argResolved.Form.(resolution.StructForm); isStruct {
			argName := argResolved.Name

			// Track and generate the Any helper for this type
			p.ensureAnyHelper(&argResolved, data)

			// Add converter function calls
			forwardConverters = append(forwardConverters, fmt.Sprintf("%sToPBAny", argName))
			backwardConverters = append(backwardConverters, fmt.Sprintf("%sFromPBAny", argName))

			// Build explicit type arg - always fully qualify since we're generating
			// in a different package (grpc) than where the types are defined
			goOutput := output.GetPath(argResolved, "go")
			if goOutput != "" {
				importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
				if err == nil {
					alias := gointernal.DerivePackageAlias(goOutput, data.Package)
					data.imports.AddInternal(alias, importPath)
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
	goOutput := output.GetPath(*actualType, "go")
	if goOutput != "" {
		importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
		if err == nil {
			alias := gointernal.DerivePackageAlias(goOutput, data.Package)
			data.imports.AddInternal(alias, importPath)
			genericGoType = fmt.Sprintf("%s.%s[%s]", alias, structName, strings.Join(explicitTypeArgs, ", "))
		}
	}

	// Build the backward cast - we need to cast the result back to the alias type
	// e.g., (*rack.Status) to cast *status.Status[rack.StatusDetails] back to *rack.Status
	if resolved != nil {
		if _, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
			// Get the alias type's package and name
			// For type aliases (Status = status.Status<D>), they may not have @go output
			// In that case, look up a sibling struct in the same namespace to get the package
			aliasGoOutput := output.GetPath(*resolved, "go")
			if aliasGoOutput == "" {
				// Try to find a sibling struct in the same namespace with @go output
				for _, sibling := range data.table.StructTypes() {
					if sibling.Namespace == resolved.Namespace {
						if _, isAliasForm := sibling.Form.(resolution.AliasForm); !isAliasForm {
							aliasGoOutput = output.GetPath(sibling, "go")
							if aliasGoOutput != "" {
								break
							}
						}
					}
				}
			}
			if aliasGoOutput != "" {
				importPath, err := resolveGoImportPath(aliasGoOutput, data.repoRoot)
				if err == nil {
					alias := gointernal.DerivePackageAlias(aliasGoOutput, data.Package)
					data.imports.AddInternal(alias, importPath)
					aliasName := getGoName(*resolved)
					if aliasName == "" {
						aliasName = resolved.Name
					}
					backwardCast = fmt.Sprintf("(*%s.%s)", alias, aliasName)
				}
			}
		}
	}

	// Build the call with explicit type args, converters, and casts
	// We need to cast because type aliases don't work with generic type inference
	if field.IsHardOptional {
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
func (p *Plugin) ensureAnyHelper(typ *resolution.Type, data *translatorTemplateData) {
	if typ == nil {
		return
	}

	key := typ.QualifiedName
	if data.generatedAnyHelpers[key] {
		return
	}
	data.generatedAnyHelpers[key] = true

	// Get Go and PB type info
	goOutput := output.GetPath(*typ, "go")
	pbOutput := output.GetPath(*typ, "pb")

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
	goAlias := gointernal.DerivePackageAlias(goOutput, data.Package)
	pbAlias := gointernal.DerivePackageAlias(pbOutput, data.Package) + "v1"

	data.imports.AddInternal(goAlias, goImportPath)
	data.imports.AddInternal(pbAlias, pbImportPath)
	data.imports.AddExternal("google.golang.org/protobuf/types/known/anypb")

	goName := getGoName(*typ)
	if goName == "" {
		goName = typ.Name
	}

	data.AnyHelpers = append(data.AnyHelpers, anyHelperData{
		TypeName: typ.Name,
		GoType:   fmt.Sprintf("%s.%s", goAlias, goName),
		PBType:   fmt.Sprintf("%s.PB%s", pbAlias, typ.Name),
	})
}

// generateEnumConversion generates conversion for enum types.
func (p *Plugin) generateEnumConversion(
	enumType *resolution.Type,
	data *translatorTemplateData,
	goField, pbField string,
) (forward, backward string) {
	enumName := enumType.Name

	// Track this enum for translator generation
	if _, exists := data.usedEnums[enumType.QualifiedName]; !exists {
		data.usedEnums[enumType.QualifiedName] = enumType
	}

	return fmt.Sprintf("translate%sForward(%s)", enumName, goField),
		fmt.Sprintf("translate%sBackward(%s)", enumName, pbField)
}

// generateArrayConversion generates conversion for array types.
func (p *Plugin) generateArrayConversion(
	field resolution.Field,
	elementTypeRef resolution.TypeRef,
	data *translatorTemplateData,
	goField, pbField string,
) (forward, backward string) {
	// For struct arrays, use slice helper
	if resolved, ok := data.table.Get(elementTypeRef.Name); ok {
		if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
			structName := resolved.Name
			return fmt.Sprintf("%ssToPB(ctx, %s)", structName, goField),
				fmt.Sprintf("%ssFromPB(ctx, %s)", structName, pbField)
		}
	}

	// For primitive arrays, use lo.Map or direct copy
	if resolution.IsPrimitive(elementTypeRef.Name) {
		switch elementTypeRef.Name {
		case "uuid":
			data.imports.AddExternal("github.com/google/uuid")
			data.imports.AddExternal("github.com/samber/lo")
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

// getGoName gets the custom Go name from @go name annotation.
func getGoName(typ resolution.Type) string {
	if domain, ok := typ.Domains["go"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
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
	imports *gointernal.ImportManager
}

// HasImports returns true if any imports are needed.
func (d *aliasTemplateData) HasImports() bool { return d.imports.HasImports() }

// InternalImports returns sorted internal imports.
func (d *aliasTemplateData) InternalImports() []gointernal.InternalImportData {
	return d.imports.InternalImports()
}

// aliasData holds data for a single type alias.
type aliasData struct {
	Name    string
	AliasOf string
}

// translatorTemplateData holds data for translator file generation.
type translatorTemplateData struct {
	Package             string
	GroupName           string
	Namespace           string
	Translators         []translatorData
	GenericTranslators  []genericTranslatorData
	EnumTranslators     []enumTranslatorData
	AnyHelpers          []anyHelperData
	imports             *gointernal.ImportManager
	repoRoot            string
	table               *resolution.Table
	usedEnums           map[string]*resolution.Type
	generatedAnyHelpers map[string]bool
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
	Name      string
	GoType    string
	PBType    string
	Values    []enumValueTranslatorData
	PBDefault string // The default proto enum value (UNSPECIFIED)
	GoDefault string // The default Go enum value
}

// enumValueTranslatorData holds data for a single enum value translation.
type enumValueTranslatorData struct {
	GoValue string
	PBValue string
}

// HasImports returns true if any imports are needed.
func (d *translatorTemplateData) HasImports() bool { return d.imports.HasImports() }

// ExternalImports returns sorted external imports.
func (d *translatorTemplateData) ExternalImports() []string { return d.imports.ExternalImports() }

// InternalImports returns sorted internal imports.
func (d *translatorTemplateData) InternalImports() []gointernal.InternalImportData {
	return d.imports.InternalImports()
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
