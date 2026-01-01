// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package pb provides an Oracle plugin that generates protobuf translator functions
// for the pb/ subdirectory pattern. It produces translator.gen.go files that convert
// between Go domain types and protobuf types.
package pb

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	gointernal "github.com/synnaxlabs/oracle/plugin/go/internal"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Plugin generates protobuf translator functions for the pb/ subdirectory pattern.
type Plugin struct{ Options Options }

// Options configures the go/pb plugin.
type Options struct {
	// TranslatorFileNamePattern is the filename pattern for translator files.
	TranslatorFileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		TranslatorFileNamePattern: "translator.gen.go",
	}
}

// New creates a new go/pb plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "go/pb" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"pb"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"go/types", "pb/types"} }

// Check verifies generated files are up-to-date. Currently unimplemented.
func (p *Plugin) Check(*plugin.Request) error { return nil }

// Generate produces translator functions for structs with @pb flag.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	// Group structs by their pb output path (derived from @go output + /pb/)
	outputStructs := make(map[string][]resolution.Struct)
	var outputOrder []string

	for _, entry := range req.Resolutions.AllStructs() {
		// Use GetPBPath which derives from @go output when @pb flag present
		outputPath := output.GetPBPath(entry)
		if outputPath == "" {
			continue
		}

		// Skip if struct is omitted from pb
		if omit.IsStruct(entry, "pb") {
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
		enums := enum.CollectReferenced(structs)
		content, err := p.generateFile(outputPath, structs, enums, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		if len(content) > 0 {
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.TranslatorFileNamePattern),
				Content: content,
			})
		}
	}

	return resp, nil
}

// generateFile generates the translator file for a set of structs.
func (p *Plugin) generateFile(
	outputPath string,
	structs []resolution.Struct,
	enums []resolution.Enum,
	req *plugin.Request,
) ([]byte, error) {
	// Get namespace from first struct
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	}

	// Get parent Go package path (outputPath minus /pb)
	parentGoPath := strings.TrimSuffix(outputPath, "/pb")

	data := &templateData{
		Package:            "pb",
		OutputPath:         outputPath,
		ParentGoPath:       parentGoPath,
		Namespace:          namespace,
		Translators:        make([]translatorData, 0, len(structs)),
		GenericTranslators: make([]genericTranslatorData, 0),
		EnumTranslators:    make([]enumTranslatorData, 0),
		AnyHelpers:         make([]anyHelperData, 0),
		imports:            gointernal.NewImportManager(),
		repoRoot:           req.RepoRoot,
		table:              req.Resolutions,
		usedEnums:          make(map[string]*resolution.Enum),
		generatedAnyHelpers: make(map[string]bool),
	}

	// Always need context import
	data.imports.AddExternal("context")

	// Import parent package for domain types
	parentImportPath, err := resolveGoImportPath(parentGoPath, req.RepoRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve parent package import")
	}
	parentAlias := gointernal.DerivePackageName(parentGoPath)
	data.imports.AddInternal(parentAlias, parentImportPath)
	data.parentAlias = parentAlias

	for _, s := range structs {
		if omit.IsStruct(s, "pb") {
			continue
		}
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
		enumTranslator := p.generateEnumTranslator(enumRef, data)
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

// processStructForTranslation processes a struct and generates translator data.
func (p *Plugin) processStructForTranslation(
	s resolution.Struct,
	data *templateData,
	req *plugin.Request,
) (*translatorData, error) {
	// Skip type aliases
	if s.IsAlias() {
		return nil, nil
	}

	goName := getGoName(s)
	if goName == "" {
		goName = s.Name
	}

	pbName := getPBName(s)
	if pbName == "" {
		pbName = s.Name
	}

	translator := &translatorData{
		Name:           pbName,
		GoType:         fmt.Sprintf("%s.%s", data.parentAlias, goName),
		PBType:         pbName,
		GoTypeShort:    goName,
		PBTypeShort:    pbName,
		Fields:         make([]fieldTranslatorData, 0),
		ErrorFields:    make([]fieldTranslatorData, 0),
		OptionalFields: make([]fieldTranslatorData, 0),
	}

	for _, field := range s.UnifiedFields() {
		fieldData := p.processFieldForTranslation(field, data, s)
		if fieldData.IsOptional {
			translator.OptionalFields = append(translator.OptionalFields, fieldData)
		} else if fieldData.HasError {
			translator.ErrorFields = append(translator.ErrorFields, fieldData)
		} else {
			translator.Fields = append(translator.Fields, fieldData)
		}
	}

	translator.UsesContext = fieldUsesContext(translator.Fields) ||
		fieldUsesContext(translator.ErrorFields) ||
		fieldUsesContext(translator.OptionalFields)
	return translator, nil
}

// processFieldForTranslation processes a field for translation.
func (p *Plugin) processFieldForTranslation(
	field resolution.Field,
	data *templateData,
	parentStruct resolution.Struct,
) fieldTranslatorData {
	goName := gointernal.ToPascalCase(field.Name)
	pbName := gointernal.ToPascalCase(field.Name)

	// Only hard optional (??) results in a pointer in Go
	isHardOptional := field.TypeRef.IsHardOptional
	isOptional := isHardOptional
	isOptionalStruct := isOptional && field.TypeRef.Kind == resolution.TypeKindStruct

	forwardExpr, backwardExpr, backwardCast, hasError := p.generateFieldConversion(field, data, parentStruct)

	return fieldTranslatorData{
		GoName:           goName,
		PBName:           pbName,
		ForwardExpr:      forwardExpr,
		BackwardExpr:     backwardExpr,
		BackwardCast:     backwardCast,
		IsOptional:       isOptional,
		IsOptionalStruct: isOptionalStruct,
		HasError:         hasError,
	}
}

// processGenericStructForTranslation processes a generic struct and generates translator data
// with type parameters. Creates translator functions that accept converter functions for each
// type parameter.
func (p *Plugin) processGenericStructForTranslation(
	s resolution.Struct,
	data *templateData,
	req *plugin.Request,
) (*genericTranslatorData, error) {
	// Skip type aliases
	if s.IsAlias() {
		return nil, nil
	}

	goName := getGoName(s)
	if goName == "" {
		goName = s.Name
	}

	pbName := getPBName(s)
	if pbName == "" {
		pbName = s.Name
	}

	data.imports.AddExternal("google.golang.org/protobuf/types/known/anypb")

	// Build type parameters
	typeParams := make([]typeParamData, 0, len(s.TypeParams))
	typeParamNames := make([]string, 0, len(s.TypeParams))
	for _, tp := range s.TypeParams {
		typeParams = append(typeParams, typeParamData{Name: tp.Name, Constraint: "any"})
		typeParamNames = append(typeParamNames, tp.Name)
	}

	goTypeWithParams := fmt.Sprintf("%s.%s[%s]", data.parentAlias, goName, strings.Join(typeParamNames, ", "))

	translator := &genericTranslatorData{
		Name:            pbName,
		GoType:          goTypeWithParams,
		GoTypeBase:      fmt.Sprintf("%s.%s", data.parentAlias, goName),
		PBType:          pbName,
		GoTypeShort:     goName,
		PBTypeShort:     pbName,
		TypeParams:      typeParams,
		Fields:          make([]fieldTranslatorData, 0),
		ErrorFields:     make([]fieldTranslatorData, 0),
		TypeParamFields: make([]fieldTranslatorData, 0),
		OptionalFields:  make([]fieldTranslatorData, 0),
	}

	for _, field := range s.UnifiedFields() {
		fieldData, isTypeParam := p.processGenericFieldForTranslation(field, data, s, typeParams)
		if isTypeParam {
			translator.TypeParamFields = append(translator.TypeParamFields, fieldData)
		} else if fieldData.IsOptional {
			translator.OptionalFields = append(translator.OptionalFields, fieldData)
		} else if fieldData.HasError {
			translator.ErrorFields = append(translator.ErrorFields, fieldData)
		} else {
			translator.Fields = append(translator.Fields, fieldData)
		}
	}

	translator.UsesContext = fieldUsesContext(translator.Fields) ||
		fieldUsesContext(translator.ErrorFields) ||
		fieldUsesContext(translator.OptionalFields) ||
		fieldUsesContext(translator.TypeParamFields)

	return translator, nil
}

// processGenericFieldForTranslation processes a field in a generic struct.
// If the field's type is a type parameter, it uses the provided converter function.
// Returns the field data and a boolean indicating if it's a type parameter field.
func (p *Plugin) processGenericFieldForTranslation(
	field resolution.Field,
	data *templateData,
	parentStruct resolution.Struct,
	typeParams []typeParamData,
) (fieldTranslatorData, bool) {
	goName := gointernal.ToPascalCase(field.Name)
	pbName := gointernal.ToPascalCase(field.Name)
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
			HasError:         true, // Type param conversions return error
		}, true // This is a type param field
	}

	// For non-type-param fields, use the regular field processing
	forwardExpr, backwardExpr, backwardCast, hasError := p.generateFieldConversion(field, data, parentStruct)

	return fieldTranslatorData{
		GoName:           goName,
		PBName:           pbName,
		ForwardExpr:      forwardExpr,
		BackwardExpr:     backwardExpr,
		BackwardCast:     backwardCast,
		IsOptional:       isOptional,
		IsOptionalStruct: isOptional && typeRef.Kind == resolution.TypeKindStruct,
		HasError:         hasError,
	}, false // Not a type param field
}

// generateFieldConversion generates the forward and backward conversion expressions.
// Returns forward expr, backward expr, backward cast, and whether the conversion returns an error.
func (p *Plugin) generateFieldConversion(
	field resolution.Field,
	data *templateData,
	parentStruct resolution.Struct,
) (forward, backward, backwardCast string, hasError bool) {
	typeRef := field.TypeRef
	fieldName := gointernal.ToPascalCase(field.Name)
	goFieldName := "r." + fieldName
	pbFieldName := "pb." + fieldName

	// Handle arrays
	if typeRef.IsArray {
		f, b, e := p.generateArrayConversion(field, data, goFieldName, pbFieldName)
		return f, b, "", e
	}

	// Handle @key fields
	if hasKeyDomain(field) && typeRef.Kind == resolution.TypeKindPrimitive && isNumericPrimitive(typeRef.Primitive) {
		protoType := primitiveToProtoType(typeRef.Primitive)
		return fmt.Sprintf("%s(%s)", protoType, goFieldName),
			fmt.Sprintf("%s.Key(%s)", data.parentAlias, pbFieldName),
			"", false
	}

	// Handle primitives
	if typeRef.Kind == resolution.TypeKindPrimitive {
		f, b := p.generatePrimitiveConversion(typeRef.Primitive, goFieldName, pbFieldName, data)
		return f, b, "", false
	}

	// Handle struct references
	if typeRef.Kind == resolution.TypeKindStruct && typeRef.StructRef != nil {
		f, b, c := p.generateStructConversion(typeRef, data, goFieldName, pbFieldName)
		return f, b, c, true // Struct conversions return error
	}

	// Handle enums
	if typeRef.Kind == resolution.TypeKindEnum && typeRef.EnumRef != nil {
		f, b := p.generateEnumConversion(typeRef, data, goFieldName, pbFieldName)
		return f, b, "", false
	}

	// Default: direct copy
	return goFieldName, pbFieldName, "", false
}

// generatePrimitiveConversion generates conversion for primitive types.
func (p *Plugin) generatePrimitiveConversion(
	primitive, goField, pbField string,
	data *templateData,
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
		return goField, pbField
	}
}

// generateStructConversion generates conversion for struct references.
// Returns forward expr, backward expr, and optional backward cast for type aliases.
func (p *Plugin) generateStructConversion(
	typeRef *resolution.TypeRef,
	data *templateData,
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
		// Import the translator package for the referenced struct
		pbOutput := output.GetPBPath(*actualStruct)
		if pbOutput != "" {
			importPath, err := resolveGoImportPath(pbOutput, data.repoRoot)
			if err == nil {
				alias := strings.ToLower(actualStruct.Namespace) + "pb"
				data.imports.AddInternal(alias, importPath)
				translatorPrefix = alias + "."
			}
		}
	}

	if typeRef.IsHardOptional {
		return fmt.Sprintf("%s%sToPB(ctx, *%s)", translatorPrefix, structName, goField),
			fmt.Sprintf("%s%sFromPB(ctx, %s)", translatorPrefix, structName, pbField), ""
	}

	return fmt.Sprintf("%s%sToPB(ctx, %s)", translatorPrefix, structName, goField),
		fmt.Sprintf("%s%sFromPB(ctx, %s)", translatorPrefix, structName, pbField), ""
}

// generateGenericStructConversion generates conversion for generic struct types
// with concrete type arguments. It calls the generic translator with appropriate
// converter functions for each type parameter.
// Returns forward expr, backward expr, and backward cast for type alias assignment.
func (p *Plugin) generateGenericStructConversion(
	typeRef *resolution.TypeRef,
	actualStruct *resolution.Struct,
	typeArgs []*resolution.TypeRef,
	data *templateData,
	goField, pbField string,
) (forward, backward, backwardCast string) {
	structName := actualStruct.Name

	// Find the translator package for the generic struct
	translatorPrefix := ""
	if actualStruct.Namespace != data.Namespace {
		pbOutput := output.GetPBPath(*actualStruct)
		if pbOutput != "" {
			importPath, err := resolveGoImportPath(pbOutput, data.repoRoot)
			if err == nil {
				alias := strings.ToLower(actualStruct.Namespace) + "pb"
				data.imports.AddInternal(alias, importPath)
				translatorPrefix = alias + "."
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

			// Build explicit type arg - use parent alias since we're in pb package
			explicitTypeArgs = append(explicitTypeArgs, fmt.Sprintf("%s.%s", data.parentAlias, argName))
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

	// Build the generic Go type for casting (e.g., status.Status[rack.StatusDetails])
	// Need to import the generic struct's package for the cast
	var genericGoType string
	goOutput := output.GetPath(*actualStruct, "go")
	if goOutput != "" {
		importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
		if err == nil {
			alias := gointernal.DerivePackageName(goOutput)
			data.imports.AddInternal(alias, importPath)
			genericGoType = fmt.Sprintf("%s.%s[%s]", alias, structName, strings.Join(explicitTypeArgs, ", "))
		}
	}

	// Build the call with explicit type args, converters, and casts
	// Type aliases require explicit casts because Go doesn't infer through aliases
	if typeRef.IsHardOptional {
		if genericGoType != "" {
			// Cast dereferenced value to concrete generic type
			forward = fmt.Sprintf("%s%sToPB%s(ctx, (%s)(*%s), %s)", translatorPrefix, structName, typeArgsStr, genericGoType, goField, forwardArgs)
		} else {
			forward = fmt.Sprintf("%s%sToPB%s(ctx, *%s, %s)", translatorPrefix, structName, typeArgsStr, goField, forwardArgs)
		}
		// The backward expression is the FromPB call
		backward = fmt.Sprintf("%s%sFromPB%s(ctx, %s, %s)", translatorPrefix, structName, typeArgsStr, pbField, backwardArgs)
		// Cast result pointer back to alias type (e.g., (*rack.Status))
		backwardCast = fmt.Sprintf("(*%s.%s)", data.parentAlias, typeRef.StructRef.Name)
	} else {
		if genericGoType != "" {
			forward = fmt.Sprintf("%s%sToPB%s(ctx, (%s)(%s), %s)", translatorPrefix, structName, typeArgsStr, genericGoType, goField, forwardArgs)
		} else {
			forward = fmt.Sprintf("%s%sToPB%s(ctx, %s, %s)", translatorPrefix, structName, typeArgsStr, goField, forwardArgs)
		}
		backward = fmt.Sprintf("%s%sFromPB%s(ctx, %s, %s)", translatorPrefix, structName, typeArgsStr, pbField, backwardArgs)
	}

	return forward, backward, backwardCast
}

// ensureAnyHelper tracks that we need to generate ToPBAny/FromPBAny helpers for a type.
func (p *Plugin) ensureAnyHelper(s *resolution.Struct, data *templateData) {
	if s == nil {
		return
	}

	key := s.QualifiedName
	if data.generatedAnyHelpers[key] {
		return
	}
	data.generatedAnyHelpers[key] = true

	// Add anypb import
	data.imports.AddExternal("google.golang.org/protobuf/types/known/anypb")

	goName := getGoName(*s)
	if goName == "" {
		goName = s.Name
	}

	data.AnyHelpers = append(data.AnyHelpers, anyHelperData{
		TypeName: s.Name,
		GoType:   fmt.Sprintf("%s.%s", data.parentAlias, goName),
		PBType:   s.Name,
	})
}

// generateEnumConversion generates conversion for enum types.
func (p *Plugin) generateEnumConversion(
	typeRef *resolution.TypeRef,
	data *templateData,
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
// Returns forward expr, backward expr, and whether the conversion returns an error.
func (p *Plugin) generateArrayConversion(
	field resolution.Field,
	data *templateData,
	goField, pbField string,
) (forward, backward string, hasError bool) {
	typeRef := field.TypeRef

	// For struct arrays, use slice helper
	if typeRef.Kind == resolution.TypeKindStruct && typeRef.StructRef != nil {
		structRef := typeRef.StructRef
		structName := structRef.Name

		// Check if referenced struct is from a different namespace
		translatorPrefix := ""
		if structRef.Namespace != data.Namespace {
			// Import the translator package for the referenced struct
			pbOutput := output.GetPBPath(*structRef)
			if pbOutput != "" {
				importPath, err := resolveGoImportPath(pbOutput, data.repoRoot)
				if err == nil {
					alias := strings.ToLower(structRef.Namespace) + "pb"
					data.imports.AddInternal(alias, importPath)
					translatorPrefix = alias + "."
				}
			}
		}

		// All slice translators return (result, error)
		return fmt.Sprintf("%s%ssToPB(ctx, %s)", translatorPrefix, structName, goField),
			fmt.Sprintf("%s%ssFromPB(ctx, %s)", translatorPrefix, structName, pbField),
			true
	}

	// For primitive arrays
	if typeRef.Kind == resolution.TypeKindPrimitive {
		switch typeRef.Primitive {
		case "uuid":
			data.imports.AddExternal("github.com/google/uuid")
			data.imports.AddExternal("github.com/samber/lo")
			return fmt.Sprintf("lo.Map(%s, func(u uuid.UUID, _ int) string { return u.String() })", goField),
				fmt.Sprintf("lo.Map(%s, func(s string, _ int) uuid.UUID { return uuid.MustParse(s) })", pbField),
				false
		}
	}

	return goField, pbField, false
}

// generateEnumTranslator generates translator data for an enum.
func (p *Plugin) generateEnumTranslator(
	enumRef *resolution.Enum,
	data *templateData,
) *enumTranslatorData {
	// Build enum value translations
	values := make([]enumValueTranslatorData, 0, len(enumRef.Values))
	var goDefault string

	goAlias := data.parentAlias
	isGoOmitted := omit.IsEnum(*enumRef, "go")

	for i, v := range enumRef.Values {
		valueName := gointernal.ToPascalCase(v.Name)

		var goValue string
		if isGoOmitted {
			// Hand-written Go enum: <Value><Enum>
			goValue = fmt.Sprintf("%s.%s%s", goAlias, valueName, enumRef.Name)
		} else {
			// Generated Go enum: <Enum><Value>
			goValue = fmt.Sprintf("%s.%s%s", goAlias, enumRef.Name, valueName)
		}

		// Proto enum value format: <EnumName>_<VALUE>
		// Go protoc generates: Variant_SUCCESS, not VARIANT_SUCCESS
		pbValueName := fmt.Sprintf("%s_%s", enumRef.Name, toScreamingSnake(v.Name))

		values = append(values, enumValueTranslatorData{
			GoValue: goValue,
			PBValue: pbValueName,
		})

		if i == 0 {
			goDefault = goValue
		}
	}

	// Proto default is always UNSPECIFIED
	pbDefault := fmt.Sprintf("%s_UNSPECIFIED", enumRef.Name)

	return &enumTranslatorData{
		Name:      enumRef.Name,
		GoType:    fmt.Sprintf("%s.%s", goAlias, enumRef.Name),
		PBType:    enumRef.Name,
		Values:    values,
		PBDefault: pbDefault,
		GoDefault: goDefault,
	}
}

// resolveGoImportPath resolves a repo-relative output path to a full Go import path.
func resolveGoImportPath(outputPath, repoRoot string) (string, error) {
	if repoRoot == "" {
		return "github.com/synnaxlabs/synnax/" + outputPath, nil
	}

	absPath := filepath.Join(repoRoot, outputPath)
	dir := absPath
	for {
		modPath := filepath.Join(dir, "go.mod")
		if fileExists(modPath) {
			moduleName, err := parseModuleName(modPath)
			if err != nil {
				return "", errors.Wrapf(err, "failed to parse go.mod at %s", modPath)
			}
			relPath, err := filepath.Rel(dir, absPath)
			if err != nil {
				return "", errors.Wrapf(err, "failed to compute relative path")
			}
			if relPath == "." {
				return moduleName, nil
			}
			return moduleName + "/" + filepath.ToSlash(relPath), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "github.com/synnaxlabs/synnax/" + outputPath, nil
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

// getPBName gets the custom protobuf name from @pb name annotation.
func getPBName(s resolution.Struct) string {
	if domain, ok := s.Domains["pb"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
}

// hasKeyDomain checks if a field has the @key annotation.
func hasKeyDomain(field resolution.Field) bool {
	_, hasKey := field.Domains["key"]
	return hasKey
}

// isNumericPrimitive returns true if the primitive is a numeric type.
func isNumericPrimitive(primitive string) bool {
	switch primitive {
	case "uint8", "uint16", "uint32", "uint64",
		"int8", "int16", "int32", "int64":
		return true
	default:
		return false
	}
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

// templateData holds data for translator file generation.
type templateData struct {
	Package            string
	OutputPath         string
	ParentGoPath       string
	Namespace          string
	Translators        []translatorData
	GenericTranslators []genericTranslatorData
	EnumTranslators    []enumTranslatorData
	AnyHelpers         []anyHelperData
	imports            *gointernal.ImportManager
	repoRoot           string
	table              *resolution.Table
	usedEnums          map[string]*resolution.Enum
	parentAlias        string
	generatedAnyHelpers map[string]bool
}

// HasImports returns true if any imports are needed.
func (d *templateData) HasImports() bool { return d.imports.HasImports() }

// ExternalImports returns sorted external imports.
func (d *templateData) ExternalImports() []string { return d.imports.ExternalImports() }

// InternalImports returns sorted internal imports.
func (d *templateData) InternalImports() []gointernal.InternalImportData {
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
	ErrorFields    []fieldTranslatorData // Fields with error-returning conversions
	OptionalFields []fieldTranslatorData
	UsesContext    bool
}

// fieldTranslatorData holds data for a single field translation.
type fieldTranslatorData struct {
	GoName           string
	PBName           string
	ForwardExpr      string
	BackwardExpr     string
	BackwardCast     string // Optional cast for the backward assignment (e.g., "(*rack.Status)")
	IsOptional       bool
	IsOptionalStruct bool
	HasError         bool // True if conversion returns (result, error)
}

// enumTranslatorData holds data for enum translator functions.
type enumTranslatorData struct {
	Name      string
	GoType    string
	PBType    string
	Values    []enumValueTranslatorData
	PBDefault string
	GoDefault string
}

// enumValueTranslatorData holds data for a single enum value translation.
type enumValueTranslatorData struct {
	GoValue string
	PBValue string
}

// genericTranslatorData holds data for a generic type's translators.
// These are translator functions with type parameters that accept converter
// functions for each type parameter.
type genericTranslatorData struct {
	Name            string
	GoType          string // e.g., "status.Status[D]"
	GoTypeBase      string // e.g., "status.Status" (without type params)
	PBType          string
	GoTypeShort     string
	PBTypeShort     string
	TypeParams      []typeParamData
	Fields          []fieldTranslatorData // Regular fields (non-type-param)
	ErrorFields     []fieldTranslatorData // Fields with error-returning conversions
	TypeParamFields []fieldTranslatorData // Fields that use type parameters (need error handling)
	OptionalFields  []fieldTranslatorData
	UsesContext     bool
}

// typeParamData holds data for a type parameter in a generic translator.
type typeParamData struct {
	Name       string // e.g., "D"
	Constraint string // e.g., "any" (Go constraint)
}

// anyHelperData holds data for ToPBAny/FromPBAny helper functions.
// These are generated for concrete types that are used as type arguments
// to generic structs.
type anyHelperData struct {
	TypeName string // e.g., "StatusDetails"
	GoType   string // e.g., "rack.StatusDetails"
	PBType   string // e.g., "PBStatusDetails"
}
