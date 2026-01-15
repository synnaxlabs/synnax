// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	gointernal "github.com/synnaxlabs/oracle/plugin/go/internal"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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

var goPostWriter = &exec.PostWriter{
	Extensions: []string{".go"},
	Commands:   [][]string{{"gofmt", "-w"}},
}

// PostWrite runs gofmt on all generated Go files.
func (p *Plugin) PostWrite(files []string) error {
	return goPostWriter.PostWrite(files)
}

// Generate produces translator functions for structs with @pb flag.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	// Group structs by their pb output path (derived from @go output + /pb/)
	outputStructs := make(map[string][]resolution.Type)
	// Group DistinctForm types that wrap structs by their pb output path
	outputTypeDefs := make(map[string][]resolution.Type)
	var outputOrder []string

	for _, entry := range req.Resolutions.StructTypes() {
		outputPath := output.GetPBPath(entry)
		if outputPath == "" {
			continue
		}

		// Skip if struct is omitted from pb
		if omit.IsType(entry, "pb") {
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

	// Collect DistinctForm types that wrap structs (for delegation translators)
	for _, entry := range req.Resolutions.DistinctTypes() {
		outputPath := output.GetPBPath(entry)
		if outputPath == "" {
			continue
		}
		if omit.IsType(entry, "pb") {
			continue
		}

		// Check if this DistinctForm wraps a struct type
		form, ok := entry.Form.(resolution.DistinctForm)
		if !ok {
			continue
		}
		if !p.isStructWrappingTypedef(form.Base, req.Resolutions) {
			continue
		}

		if req.RepoRoot != "" {
			if err := req.ValidateOutputPath(outputPath); err != nil {
				return nil, errors.Wrapf(err, "invalid output path for typedef %s", entry.Name)
			}
		}
		if _, exists := outputStructs[outputPath]; !exists {
			if _, exists := outputTypeDefs[outputPath]; !exists {
				outputOrder = append(outputOrder, outputPath)
			}
		}
		outputTypeDefs[outputPath] = append(outputTypeDefs[outputPath], entry)
	}

	for _, outputPath := range outputOrder {
		structs := outputStructs[outputPath]
		typeDefs := outputTypeDefs[outputPath]
		// Get all enums in the namespace, not just referenced ones
		namespace := ""
		if len(structs) > 0 {
			namespace = structs[0].Namespace
		} else if len(typeDefs) > 0 {
			namespace = typeDefs[0].Namespace
		}
		enums := req.Resolutions.EnumsInNamespace(namespace)
		content, err := p.generateFile(outputPath, structs, typeDefs, enums, req)
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

// isStructWrappingTypedef checks if a typeRef ultimately resolves to a struct type.
func (p *Plugin) isStructWrappingTypedef(typeRef resolution.TypeRef, table *resolution.Table) bool {
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return false
	}
	// Direct struct
	if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
		return true
	}
	// Alias to struct
	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		return p.isStructWrappingTypedef(aliasForm.Target, table)
	}
	return false
}

// generateFile generates the translator file for a set of structs.
func (p *Plugin) generateFile(
	outputPath string,
	structs []resolution.Type,
	typeDefs []resolution.Type,
	enums []resolution.Type,
	req *plugin.Request,
) ([]byte, error) {
	// Get namespace from first struct or typedef
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	} else if len(typeDefs) > 0 {
		namespace = typeDefs[0].Namespace
	}

	// Get parent Go package path (outputPath minus /pb)
	parentGoPath := strings.TrimSuffix(outputPath, "/pb")

	data := &templateData{
		Package:               "pb",
		OutputPath:            outputPath,
		ParentGoPath:          parentGoPath,
		Namespace:             namespace,
		Translators:           make([]translatorData, 0, len(structs)),
		GenericTranslators:    make([]genericTranslatorData, 0),
		EnumTranslators:       make([]enumTranslatorData, 0),
		AnyHelpers:            make([]anyHelperData, 0),
		DelegationTranslators: make([]delegationTranslatorData, 0),
		imports:               gointernal.NewImportManager(),
		repoRoot:              req.RepoRoot,
		table:                 req.Resolutions,
		usedEnums:             make(map[string]*resolution.Type),
		generatedAnyHelpers:   make(map[string]bool),
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
		if omit.IsType(s, "pb") {
			continue
		}
		form, ok := s.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		if form.IsGeneric() {
			// Generate generic translator with type parameters
			genericTranslator, err := p.processGenericStructForTranslation(s, form, data, req)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to process generic struct %s", s.Name)
			}
			if genericTranslator != nil {
				data.GenericTranslators = append(data.GenericTranslators, *genericTranslator)
			}
		} else {
			translator, err := p.processStructForTranslation(s, form, data, req)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to process struct %s", s.Name)
			}
			if translator != nil {
				data.Translators = append(data.Translators, *translator)
			}
		}
	}

	// Add enums that belong to this output path (not all enums in namespace)
	// Enums with a different @go output should be imported, not regenerated
	for i := range enums {
		e := enums[i]
		if omit.IsType(e, "pb") {
			continue
		}
		// Only generate translator for enums whose pb path matches this output path
		enumPBPath := output.GetPBPath(e)
		if enumPBPath != "" && enumPBPath != outputPath {
			continue // This enum belongs to a different pb package
		}
		data.usedEnums[e.QualifiedName] = &e
	}

	// Generate enum translators for enums that belong to this output path
	for _, enumRef := range data.usedEnums {
		enumTranslator := p.generateEnumTranslator(enumRef, data)
		if enumTranslator != nil {
			data.EnumTranslators = append(data.EnumTranslators, *enumTranslator)
		}
	}

	// Process typedefs that wrap structs (delegation translators)
	for _, td := range typeDefs {
		if omit.IsType(td, "pb") {
			continue
		}
		form, ok := td.Form.(resolution.DistinctForm)
		if !ok {
			continue
		}
		delegator, err := p.processDelegationTranslator(td, form, data, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process delegation translator %s", td.Name)
		}
		if delegator != nil {
			data.DelegationTranslators = append(data.DelegationTranslators, *delegator)
		}
	}

	// Collect all distinct primitive types for any converter helper
	if data.needsAnyConverter {
		p.collectDistinctPrimitives(data, req)
	}

	// Skip file generation if no translators of any kind
	if len(data.Translators) == 0 && len(data.GenericTranslators) == 0 && len(data.EnumTranslators) == 0 && len(data.DelegationTranslators) == 0 {
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
	s resolution.Type,
	form resolution.StructForm,
	data *templateData,
	req *plugin.Request,
) (*translatorData, error) {
	// Skip type aliases
	if _, ok := s.Form.(resolution.AliasForm); ok {
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

	for _, field := range resolution.UnifiedFields(s, data.table) {
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
	parentStruct resolution.Type,
) fieldTranslatorData {
	goName := gointernal.ToPascalCase(field.Name)
	// Proto uses snake_case, protoc converts to PascalCase
	// e.g., WASM -> wasm (proto) -> Wasm (Go protobuf)
	pbName := gointernal.ToPascalCase(lo.SnakeCase(field.Name))

	// Only hard optional (??) results in a pointer in Go
	isHardOptional := field.IsHardOptional
	isOptional := isHardOptional
	isOptionalStruct := isOptional && isStructType(field.Type, data.table)

	forwardExpr, backwardExpr, backwardCast, hasError, hasBackwardError := p.generateFieldConversion(field, data, parentStruct)

	return fieldTranslatorData{
		GoName:           goName,
		PBName:           pbName,
		ForwardExpr:      forwardExpr,
		BackwardExpr:     backwardExpr,
		BackwardCast:     backwardCast,
		IsOptional:       isOptional,
		IsOptionalStruct: isOptionalStruct,
		HasError:         hasError,
		HasBackwardError: hasBackwardError,
	}
}

// processGenericStructForTranslation processes a generic struct and generates translator data
// with type parameters. Creates translator functions that accept converter functions for each
// type parameter.
func (p *Plugin) processGenericStructForTranslation(
	s resolution.Type,
	form resolution.StructForm,
	data *templateData,
	req *plugin.Request,
) (*genericTranslatorData, error) {
	// Skip type aliases
	if _, ok := s.Form.(resolution.AliasForm); ok {
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

	// Build type parameters, skipping defaulted ones
	// (Go doesn't support advanced generics with defaults)
	typeParams := make([]typeParamData, 0, len(form.TypeParams))
	typeParamNames := make([]string, 0, len(form.TypeParams))
	for _, tp := range form.TypeParams {
		if tp.HasDefault() {
			continue // Skip defaulted type params
		}
		typeParams = append(typeParams, typeParamData{Name: tp.Name, Constraint: "any"})
		typeParamNames = append(typeParamNames, tp.Name)
	}

	goTypeBase := fmt.Sprintf("%s.%s", data.parentAlias, goName)
	goTypeWithParams := goTypeBase
	if len(typeParamNames) > 0 {
		goTypeWithParams = fmt.Sprintf("%s[%s]", goTypeBase, strings.Join(typeParamNames, ", "))
	}

	translator := &genericTranslatorData{
		Name:            pbName,
		GoType:          goTypeWithParams,
		GoTypeBase:      goTypeBase,
		PBType:          pbName,
		GoTypeShort:     goName,
		PBTypeShort:     pbName,
		TypeParams:      typeParams,
		Fields:          make([]fieldTranslatorData, 0),
		ErrorFields:     make([]fieldTranslatorData, 0),
		TypeParamFields: make([]fieldTranslatorData, 0),
		OptionalFields:  make([]fieldTranslatorData, 0),
	}

	for _, field := range resolution.UnifiedFields(s, data.table) {
		fieldData, isTypeParam := p.processGenericFieldForTranslation(field, data, s, form, typeParams)
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
	parentStruct resolution.Type,
	parentForm resolution.StructForm,
	typeParams []typeParamData,
) (fieldTranslatorData, bool) {
	goName := gointernal.ToPascalCase(field.Name)
	// Proto uses snake_case, protoc converts to PascalCase
	pbName := gointernal.ToPascalCase(lo.SnakeCase(field.Name))
	typeRef := field.Type

	isHardOptional := field.IsHardOptional
	isOptional := isHardOptional

	goFieldName := "r." + goName
	pbFieldName := "pb." + pbName

	// Check if this field's type is a type parameter
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		// For defaulted type params, substitute the default type instead
		if typeRef.TypeParam.HasDefault() {
			// Treat as regular field with the default type
			forwardExpr, backwardExpr, backwardCast, hasError, hasBackwardError := p.generateFieldConversion(
				resolution.Field{
					Name:           field.Name,
					Type:           *typeRef.TypeParam.Default,
					IsOptional:     field.IsOptional,
					IsHardOptional: field.IsHardOptional,
				},
				data, parentStruct,
			)
			return fieldTranslatorData{
				GoName:           goName,
				PBName:           pbName,
				ForwardExpr:      forwardExpr,
				BackwardExpr:     backwardExpr,
				BackwardCast:     backwardCast,
				IsOptional:       isOptional,
				IsOptionalStruct: isOptional && isStructType(*typeRef.TypeParam.Default, data.table),
				HasError:         hasError,
				HasBackwardError: hasBackwardError,
			}, false // Not a type param field anymore
		}

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
			HasError:         true, // Type param conversions return error
			HasBackwardError: true, // Type param conversions return error
		}, true // This is a type param field
	}

	// For non-type-param fields, use the regular field processing
	forwardExpr, backwardExpr, backwardCast, hasError, hasBackwardError := p.generateFieldConversion(field, data, parentStruct)

	return fieldTranslatorData{
		GoName:           goName,
		PBName:           pbName,
		ForwardExpr:      forwardExpr,
		BackwardExpr:     backwardExpr,
		BackwardCast:     backwardCast,
		IsOptional:       isOptional,
		IsOptionalStruct: isOptional && isStructType(typeRef, data.table),
		HasError:         hasError,
		HasBackwardError: hasBackwardError,
	}, false // Not a type param field
}

// processDelegationTranslator creates a delegation translator for a DistinctForm
// that wraps a struct type. Instead of generating independent translators, it
// generates thin wrappers that cast to the underlying type and delegate.
func (p *Plugin) processDelegationTranslator(
	td resolution.Type,
	form resolution.DistinctForm,
	data *templateData,
	req *plugin.Request,
) (*delegationTranslatorData, error) {
	// Get the Go name for this typedef (respecting @go name)
	goName := getGoName(td)
	if goName == "" {
		goName = td.Name
	}

	// Build type parameters, skipping defaulted ones
	typeParams := make([]typeParamData, 0, len(form.TypeParams))
	typeParamNames := make([]string, 0, len(form.TypeParams))
	for _, tp := range form.TypeParams {
		if tp.HasDefault() {
			continue // Skip defaulted type params
		}
		typeParams = append(typeParams, typeParamData{Name: tp.Name, Constraint: "any"})
		typeParamNames = append(typeParamNames, tp.Name)
	}

	// Build the Go type for this typedef (e.g., "status.Status[Details]")
	goType := fmt.Sprintf("%s.%s", data.parentAlias, goName)
	if len(typeParamNames) > 0 {
		goType = fmt.Sprintf("%s.%s[%s]", data.parentAlias, goName, strings.Join(typeParamNames, ", "))
	}

	// Resolve the underlying struct type
	underlyingResolved, ok := form.Base.Resolve(data.table)
	if !ok {
		return nil, nil
	}

	// Follow alias chain to find actual struct
	actualStruct := underlyingResolved
	for {
		if aliasForm, isAlias := actualStruct.Form.(resolution.AliasForm); isAlias {
			if target, ok := aliasForm.Target.Resolve(data.table); ok {
				actualStruct = target
				continue
			}
		}
		break
	}

	// Get the underlying struct's Go name
	underlyingGoName := getGoName(actualStruct)
	if underlyingGoName == "" {
		underlyingGoName = actualStruct.Name
	}

	// Get the underlying struct's pb path
	underlyingPBPath := output.GetPBPath(actualStruct)
	if underlyingPBPath == "" {
		// Try to find pb path via extends chain
		_, underlyingPBPath = findStructWithPB(actualStruct, data.table)
		if underlyingPBPath == "" {
			return nil, nil // No pb defined for underlying type
		}
	}

	// Import the underlying type's Go package
	underlyingGoPath := output.GetPath(actualStruct, "go")
	if underlyingGoPath == "" {
		return nil, nil
	}
	underlyingGoImportPath, err := resolveGoImportPath(underlyingGoPath, data.repoRoot)
	if err != nil {
		return nil, err
	}
	underlyingGoAlias := gointernal.DerivePackageAlias(underlyingGoPath, data.parentAlias)
	data.imports.AddInternal(underlyingGoAlias, underlyingGoImportPath)

	// Import the underlying type's pb package
	underlyingPBImportPath, err := resolveGoImportPath(underlyingPBPath, data.repoRoot)
	if err != nil {
		return nil, err
	}
	underlyingPBAlias := underlyingGoAlias + "_pb"
	data.imports.AddInternal(underlyingPBAlias, underlyingPBImportPath)

	// Import anypb for generic type parameters
	if len(typeParams) > 0 {
		data.imports.AddExternal("google.golang.org/protobuf/types/known/anypb")
	}

	// Build the underlying Go type for casting (e.g., "gostatus.Status[Details]")
	underlyingGoType := fmt.Sprintf("%s.%s", underlyingGoAlias, underlyingGoName)
	if len(typeParamNames) > 0 {
		underlyingGoType = fmt.Sprintf("%s.%s[%s]", underlyingGoAlias, underlyingGoName, strings.Join(typeParamNames, ", "))
	}

	// Get the pb name for the underlying struct
	underlyingPBName := getPBName(actualStruct)
	if underlyingPBName == "" {
		underlyingPBName = actualStruct.Name
	}

	return &delegationTranslatorData{
		Name:                       goName,
		GoType:                     goType,
		TypeParams:                 typeParams,
		UnderlyingName:             underlyingPBName,
		UnderlyingGoType:           underlyingGoType,
		UnderlyingPBType:           fmt.Sprintf("%s.%s", underlyingPBAlias, underlyingPBName),
		UnderlyingTranslatorPrefix: underlyingPBAlias + ".",
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
// This must match the naming in pb/types plugin.
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

// isFixedSizeUint8Array checks if a type is a fixed-size uint8 array (like Color [4]uint8).
// These are encoded as bytes in protobuf and need special conversion.
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

// generateFixedSizeUint8ArrayConversion generates conversion for fixed-size uint8 arrays.
// These types (like Color [4]uint8) are encoded as bytes in proto.
// Forward: uses .Bytes() method to convert [N]uint8 to []byte
// Backward: uses FromBytes() function to convert []byte back to [N]uint8
func (p *Plugin) generateFixedSizeUint8ArrayConversion(
	typeRef resolution.TypeRef,
	data *templateData,
	goField, pbField string,
) (forward, backward string) {
	// Resolve the type to get its package info
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		// Fallback for unresolved types
		return fmt.Sprintf("%s[:]", goField), pbField
	}

	// Get the Go output path for the type
	goOutput := output.GetPath(resolved, "go")
	if goOutput == "" {
		// Fallback
		return fmt.Sprintf("%s[:]", goField), pbField
	}

	// Import the type's package
	importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
	if err != nil {
		return fmt.Sprintf("%s[:]", goField), pbField
	}

	alias := gointernal.DerivePackageName(goOutput)
	data.imports.AddInternal(alias, importPath)

	// Forward: use .Bytes() method (e.g., r.Color.Bytes())
	forward = fmt.Sprintf("%s.Bytes()", goField)

	// Backward: use FromBytes function (e.g., color.FromBytes(pb.Color))
	backward = fmt.Sprintf("%s.FromBytes(%s)", alias, pbField)

	return forward, backward
}

// generateFieldConversion generates the forward and backward conversion expressions.
// Returns forward expr, backward expr, backward cast, and whether forward/backward conversions return errors.
func (p *Plugin) generateFieldConversion(
	field resolution.Field,
	data *templateData,
	parentStruct resolution.Type,
) (forward, backward, backwardCast string, hasError, hasBackwardError bool) {
	typeRef := field.Type
	goFieldName := "r." + gointernal.ToPascalCase(field.Name)
	// Proto uses snake_case, protoc converts to PascalCase
	// e.g., WASM -> wasm (proto) -> Wasm (Go protobuf)
	pbFieldName := "pb." + gointernal.ToPascalCase(lo.SnakeCase(field.Name))

	// Handle fixed-size uint8 arrays (like Color [4]uint8) - these are bytes in proto
	if p.isFixedSizeUint8Array(typeRef, data.table) {
		f, b := p.generateFixedSizeUint8ArrayConversion(typeRef, data, goFieldName, pbFieldName)
		return f, b, "", false, false
	}

	// Handle arrays (including via aliases and distinct types)
	if p.isArrayType(typeRef, data.table) {
		f, b, e := p.generateArrayConversion(field, data, goFieldName, pbFieldName)
		return f, b, "", e, e
	}

	// Handle @key fields
	if hasKeyDomain(field) && resolution.IsPrimitive(typeRef.Name) && isNumericPrimitive(typeRef.Name) {
		protoType := primitiveToProtoType(typeRef.Name)
		// Get the Key type's package - it may be in a different package than the parent
		keyPkgAlias := data.parentAlias
		goOutput := output.GetPath(parentStruct, "go")
		if goOutput != "" && goOutput != data.ParentGoPath {
			importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
			if err == nil {
				keyPkgAlias = gointernal.DerivePackageName(goOutput)
				data.imports.AddInternal(keyPkgAlias, importPath)
			}
		}
		return fmt.Sprintf("%s(%s)", protoType, goFieldName),
			fmt.Sprintf("%s.Key(%s)", keyPkgAlias, pbFieldName),
			"", false, false
	}

	// Handle primitives
	if resolution.IsPrimitive(typeRef.Name) {
		f, b, e, be := p.generatePrimitiveConversion(typeRef.Name, goFieldName, pbFieldName, data)
		return f, b, "", e, be
	}

	// Resolve the type from the table
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		// Default: direct copy
		return goFieldName, pbFieldName, "", false, false
	}

	// Handle struct references
	if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
		f, b, c, hasErr := p.generateStructConversion(typeRef, resolved, field.IsHardOptional, data, goFieldName, pbFieldName)
		return f, b, c, hasErr, hasErr
	}

	// Handle aliases that point to structs (e.g., GoStatus = status.Status<...>)
	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		if target, ok := aliasForm.Target.Resolve(data.table); ok {
			if _, isStruct := target.Form.(resolution.StructForm); isStruct {
				f, b, c, hasErr := p.generateStructConversion(typeRef, resolved, field.IsHardOptional, data, goFieldName, pbFieldName)
				return f, b, c, hasErr, hasErr
			}
		}
	}

	// Handle enums
	if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
		f, b := p.generateEnumConversion(typeRef, resolved, data, goFieldName, pbFieldName)
		return f, b, "", false, false
	}

	// Handle type definitions (typedefs / distinct types)
	if form, isDistinct := resolved.Form.(resolution.DistinctForm); isDistinct {
		// Check if this distinct type wraps a struct - if so, handle it like a struct
		if baseResolved, ok := form.Base.Resolve(data.table); ok {
			if _, isStruct := baseResolved.Form.(resolution.StructForm); isStruct {
				f, b, c, hasErr := p.generateStructConversion(typeRef, resolved, field.IsHardOptional, data, goFieldName, pbFieldName)
				return f, b, c, hasErr, hasErr
			}
			// Also check if base is an alias to a struct
			if aliasForm, isAlias := baseResolved.Form.(resolution.AliasForm); isAlias {
				if target, ok := aliasForm.Target.Resolve(data.table); ok {
					if _, isStruct := target.Form.(resolution.StructForm); isStruct {
						f, b, c, hasErr := p.generateStructConversion(typeRef, resolved, field.IsHardOptional, data, goFieldName, pbFieldName)
						return f, b, c, hasErr, hasErr
					}
				}
			}
		}
		f, b := p.generateTypeDefConversion(typeRef, resolved, form, data, goFieldName, pbFieldName)
		return f, b, "", false, false
	}

	// Handle type aliases to primitives (e.g., Key = uuid)
	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		if resolution.IsPrimitive(aliasForm.Target.Name) {
			f, b := p.generateAliasConversion(typeRef, resolved, aliasForm, data, goFieldName, pbFieldName)
			return f, b, "", false, false
		}
	}

	// Default: direct copy
	return goFieldName, pbFieldName, "", false, false
}

// generatePrimitiveConversion generates conversion for primitive types.
// Returns forward expr, backward expr, and whether forward/backward conversions return errors.
func (p *Plugin) generatePrimitiveConversion(
	primitive, goField, pbField string,
	data *templateData,
) (forward, backward string, hasError, hasBackwardError bool) {
	switch primitive {
	case "uuid":
		data.imports.AddExternal("github.com/google/uuid")
		return fmt.Sprintf("%s.String()", goField),
			fmt.Sprintf("uuid.MustParse(%s)", pbField), false, false
	case "timestamp":
		data.imports.AddExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("int64(%s)", goField),
			fmt.Sprintf("telem.TimeStamp(%s)", pbField), false, false
	case "timespan":
		data.imports.AddExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("int64(%s)", goField),
			fmt.Sprintf("telem.TimeSpan(%s)", pbField), false, false
	case "time_range", "time_range_bounded":
		data.imports.AddExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("telem.TranslateTimeRangeForward(%s)", goField),
			fmt.Sprintf("telem.TranslateTimeRangeBackward(%s)", pbField), false, false
	case "json":
		data.imports.AddExternal("google.golang.org/protobuf/types/known/structpb")
		// structpb.NewStruct returns (*Struct, error) - needs error handling
		// AsMap() does NOT return an error
		return fmt.Sprintf("structpb.NewStruct(%s)", goField),
			fmt.Sprintf("%s.AsMap()", pbField), true, false
	case "uint12":
		data.imports.AddExternal("github.com/synnaxlabs/x/types")
		return fmt.Sprintf("uint32(%s)", goField),
			fmt.Sprintf("types.Uint12(%s)", pbField), false, false
	case "uint20":
		data.imports.AddExternal("github.com/synnaxlabs/x/types")
		return fmt.Sprintf("uint32(%s)", goField),
			fmt.Sprintf("types.Uint20(%s)", pbField), false, false
	case "data_type":
		data.imports.AddExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("string(%s)", goField),
			fmt.Sprintf("telem.DataType(%s)", pbField), false, false
	case "any":
		data.imports.AddExternal("google.golang.org/protobuf/types/known/structpb")
		data.needsAnyConverter = true
		// structpb.NewValue returns (*Value, error) - needs error handling
		// AsInterface() does NOT return an error
		// Use convertAnyForPB to handle custom distinct primitive types
		return fmt.Sprintf("structpb.NewValue(convertAnyForPB(%s))", goField),
			fmt.Sprintf("%s.AsInterface()", pbField), true, false
	case "int8":
		return fmt.Sprintf("int32(%s)", goField),
			fmt.Sprintf("int8(%s)", pbField), false, false
	default:
		return goField, pbField, false, false
	}
}

// generateStructConversion generates conversion for struct references.
// Returns forward expr, backward expr, optional backward cast for type aliases, and whether error handling is needed.
func (p *Plugin) generateStructConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	isHardOptional bool,
	data *templateData,
	goField, pbField string,
) (forward, backward, backwardCast string, hasError bool) {
	// Follow alias chain to find the actual underlying struct and collect type args
	actualStruct := resolved
	var typeArgs []resolution.TypeRef

	// If this is an alias, get the type args from the alias's Target
	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		typeArgs = aliasForm.Target.TypeArgs
		if target, ok := aliasForm.Target.Resolve(data.table); ok {
			actualStruct = target
		}
	}

	// Continue following the chain if needed
	for {
		aliasForm, isAlias := actualStruct.Form.(resolution.AliasForm)
		if !isAlias {
			break
		}
		if len(typeArgs) == 0 && len(aliasForm.Target.TypeArgs) > 0 {
			typeArgs = aliasForm.Target.TypeArgs
		}
		if target, ok := aliasForm.Target.Resolve(data.table); ok {
			actualStruct = target
		} else {
			break
		}
	}

	actualForm, ok := actualStruct.Form.(resolution.StructForm)
	if !ok {
		return goField, pbField, "", false
	}

	// For generic types, synthesize nil type args for optional params without provided args
	if actualForm.IsGeneric() {
		// Count non-defaulted type params
		var nonDefaultedParams []resolution.TypeParam
		for _, tp := range actualForm.TypeParams {
			if !tp.HasDefault() {
				nonDefaultedParams = append(nonDefaultedParams, tp)
			}
		}
		// If there are missing type args for optional params, synthesize nil refs
		providedArgs := len(typeArgs)
		if providedArgs < len(nonDefaultedParams) {
			newTypeArgs := make([]resolution.TypeRef, len(nonDefaultedParams))
			copy(newTypeArgs, typeArgs)
			for i := providedArgs; i < len(nonDefaultedParams); i++ {
				if nonDefaultedParams[i].Optional {
					// Synthesize a nil type reference for optional param
					newTypeArgs[i] = resolution.TypeRef{Name: "nil"}
				}
			}
			typeArgs = newTypeArgs
		}
	}

	// Handle generic types with concrete type arguments
	if actualForm.IsGeneric() && len(typeArgs) > 0 {
		return p.generateGenericStructConversion(typeRef, resolved, actualStruct, actualForm, typeArgs, data, goField, pbField, isHardOptional)
	}

	// For generic types without type args and no optional params to synthesize, fall back to direct assignment
	if actualForm.IsGeneric() {
		return goField, pbField, "", false
	}

	// Use the new helper to resolve translator info (handles extends chain)
	translatorPrefix, translatorStructName := p.resolvePBTranslatorInfo(actualStruct, data)

	if isHardOptional {
		return fmt.Sprintf("%s%sToPB(ctx, *%s)", translatorPrefix, translatorStructName, goField),
			fmt.Sprintf("%s%sFromPB(ctx, %s)", translatorPrefix, translatorStructName, pbField), "", true
	}

	return fmt.Sprintf("%s%sToPB(ctx, %s)", translatorPrefix, translatorStructName, goField),
		fmt.Sprintf("%s%sFromPB(ctx, %s)", translatorPrefix, translatorStructName, pbField), "", true
}

// generateGenericStructConversion generates conversion for generic struct types
// with concrete type arguments. It calls the generic translator with appropriate
// converter functions for each type parameter.
// Returns forward expr, backward expr, backward cast for type alias assignment, and whether error handling is needed.
func (p *Plugin) generateGenericStructConversion(
	typeRef resolution.TypeRef,
	originalResolved resolution.Type,
	actualStruct resolution.Type,
	actualForm resolution.StructForm,
	typeArgs []resolution.TypeRef,
	data *templateData,
	goField, pbField string,
	isHardOptional bool,
) (forward, backward, backwardCast string, hasError bool) {
	// For generics, use resolvePBTranslatorInfo to get both the import prefix and the
	// correct struct name (respecting @pb name annotation)
	translatorPrefix, structName := p.resolvePBTranslatorInfo(actualStruct, data)

	// Build converter function arguments and explicit type args for each type arg
	// Skip type args that correspond to defaulted params
	var forwardConverters, backwardConverters []string
	var explicitTypeArgs []string
	for i, typeArg := range typeArgs {
		// Skip type args for defaulted params
		if i < len(actualForm.TypeParams) && actualForm.TypeParams[i].HasDefault() {
			continue
		}

		argResolved, ok := typeArg.Resolve(data.table)
		if ok {
			if _, isStruct := argResolved.Form.(resolution.StructForm); isStruct {
				// Get Go name (respecting @go name directive)
				argGoName := getGoName(argResolved)
				if argGoName == "" {
					argGoName = argResolved.Name
				}

				// Track and generate the Any helper for this type
				p.ensureAnyHelper(argResolved, data)

				// Add converter function calls
				forwardConverters = append(forwardConverters, fmt.Sprintf("%sToPBAny", argGoName))
				backwardConverters = append(backwardConverters, fmt.Sprintf("%sFromPBAny", argGoName))

				// Build explicit type arg - use parent alias since we're in pb package
				explicitTypeArgs = append(explicitTypeArgs, fmt.Sprintf("%s.%s", data.parentAlias, argGoName))
				continue
			}
		}
		// For non-struct type args (primitives, etc.), we'd need different handling
		forwardConverters = append(forwardConverters, "nil")
		backwardConverters = append(backwardConverters, "nil")
		// Special case: nil primitive maps to gotypes.Nil
		if typeArg.Name == "nil" {
			data.imports.AddInternal("gotypes", "go/types")
			explicitTypeArgs = append(explicitTypeArgs, "gotypes.Nil")
		} else {
			explicitTypeArgs = append(explicitTypeArgs, "any")
		}
	}

	forwardArgs := strings.Join(forwardConverters, ", ")
	backwardArgs := strings.Join(backwardConverters, ", ")
	typeArgsStr := "[" + strings.Join(explicitTypeArgs, ", ") + "]"

	// Build the generic Go type for casting (e.g., status.Status[rack.StatusDetails])
	// Need to import the generic struct's package for the cast
	var genericGoType string
	goOutput := output.GetPath(actualStruct, "go")
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
	// Get Go name for the alias struct (respecting @go name directive)
	aliasGoName := getGoName(originalResolved)
	if aliasGoName == "" {
		aliasGoName = originalResolved.Name
	}
	if isHardOptional {
		if genericGoType != "" {
			// Cast dereferenced value to concrete generic type
			forward = fmt.Sprintf("%s%sToPB%s(ctx, (%s)(*%s), %s)", translatorPrefix, structName, typeArgsStr, genericGoType, goField, forwardArgs)
		} else {
			forward = fmt.Sprintf("%s%sToPB%s(ctx, *%s, %s)", translatorPrefix, structName, typeArgsStr, goField, forwardArgs)
		}
		// The backward expression is the FromPB call
		backward = fmt.Sprintf("%s%sFromPB%s(ctx, %s, %s)", translatorPrefix, structName, typeArgsStr, pbField, backwardArgs)
		// Cast result pointer back to alias type (e.g., (*rack.Status))
		backwardCast = fmt.Sprintf("(*%s.%s)", data.parentAlias, aliasGoName)
	} else {
		if genericGoType != "" {
			forward = fmt.Sprintf("%s%sToPB%s(ctx, (%s)(%s), %s)", translatorPrefix, structName, typeArgsStr, genericGoType, goField, forwardArgs)
		} else {
			forward = fmt.Sprintf("%s%sToPB%s(ctx, %s, %s)", translatorPrefix, structName, typeArgsStr, goField, forwardArgs)
		}
		backward = fmt.Sprintf("%s%sFromPB%s(ctx, %s, %s)", translatorPrefix, structName, typeArgsStr, pbField, backwardArgs)
	}

	return forward, backward, backwardCast, true
}

// ensureAnyHelper tracks that we need to generate ToPBAny/FromPBAny helpers for a type.
func (p *Plugin) ensureAnyHelper(s resolution.Type, data *templateData) {
	key := s.QualifiedName
	if data.generatedAnyHelpers[key] {
		return
	}
	data.generatedAnyHelpers[key] = true

	// Add imports needed for AnyHelpers
	data.imports.AddExternal("google.golang.org/protobuf/types/known/anypb")
	data.imports.AddExternal("google.golang.org/protobuf/types/known/structpb")
	data.imports.AddExternal("google.golang.org/protobuf/encoding/protojson")
	data.imports.AddExternal("encoding/json")

	goName := getGoName(s)
	if goName == "" {
		goName = s.Name
	}

	pbName := getPBName(s)
	if pbName == "" {
		pbName = s.Name
	}

	data.AnyHelpers = append(data.AnyHelpers, anyHelperData{
		TypeName: goName,
		GoType:   fmt.Sprintf("%s.%s", data.parentAlias, goName),
		PBType:   pbName,
	})
}

// generateEnumConversion generates conversion for enum types.
// For same-namespace enums, generates local translator functions.
// For cross-namespace enums, imports and uses the external pb package's translators.
func (p *Plugin) generateEnumConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	data *templateData,
	goField, pbField string,
) (forward, backward string) {
	enumName := resolved.Name

	// Check if this is a cross-namespace enum reference
	if resolved.Namespace != data.Namespace {
		// Cross-namespace: import the pb package and use its translator
		pbPath := findEnumPBPath(resolved, data.table)
		if pbPath != "" {
			importPath, err := resolveGoImportPath(pbPath, data.repoRoot)
			if err == nil {
				alias := strings.ToLower(resolved.Namespace) + "pb"
				data.imports.AddInternal(alias, importPath)
				return fmt.Sprintf("%s.%sToPB(%s)", alias, enumName, goField),
					fmt.Sprintf("%s.%sFromPB(%s)", alias, enumName, pbField)
			}
		}
	}

	// Same namespace: track for local translator generation
	if _, exists := data.usedEnums[resolved.QualifiedName]; !exists {
		data.usedEnums[resolved.QualifiedName] = &resolved
	}

	return fmt.Sprintf("%sToPB(%s)", enumName, goField),
		fmt.Sprintf("%sFromPB(%s)", enumName, pbField)
}

// findEnumPBPath finds the pb output path for an enum by looking for a struct
// in the same namespace that has @pb defined.
func findEnumPBPath(e resolution.Type, table *resolution.Table) string {
	for _, s := range table.StructTypes() {
		if s.Namespace == e.Namespace {
			if path := output.GetPBPath(s); path != "" {
				return path
			}
		}
	}
	return ""
}

// generateTypeDefConversion generates conversion for typedef references.
// Typedefs are Go type definitions (type Key uint32) that need explicit casting.
// Forward: cast Go typedef to proto primitive (e.g., uint32(r.Rack))
// Backward: cast proto primitive to Go typedef (e.g., rack.Key(pb.Rack))
func (p *Plugin) generateTypeDefConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	form resolution.DistinctForm,
	data *templateData,
	goField, pbField string,
) (forward, backward string) {
	// Get the underlying primitive type for the proto cast
	baseType := form.Base
	if !resolution.IsPrimitive(baseType.Name) {
		// If the base type is not a primitive, fall back to direct copy
		return goField, pbField
	}

	// Get the Go typedef name with package prefix
	typedefPrefix := ""
	goOutput := output.GetPath(resolved, "go")
	// Check if typedef is in a different package (different namespace OR different output path)
	if resolved.Namespace != data.Namespace || (goOutput != "" && goOutput != data.ParentGoPath) {
		// Import the typedef's package
		if goOutput != "" {
			importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
			if err == nil {
				// Use parent alias for conflict detection since we're in pb/ subdirectory
				alias := gointernal.DerivePackageAlias(goOutput, data.parentAlias)
				data.imports.AddInternal(alias, importPath)
				typedefPrefix = alias + "."
			}
		}
	} else {
		// Same namespace and output path, use parent alias
		typedefPrefix = data.parentAlias + "."
	}

	// Handle uuid specially - needs String() for forward, MustParse for backward
	if baseType.Name == "uuid" {
		data.imports.AddExternal("github.com/google/uuid")
		forward = fmt.Sprintf("%s.String()", goField)
		backward = fmt.Sprintf("%s%s(uuid.MustParse(%s))", typedefPrefix, resolved.Name, pbField)
		return forward, backward
	}

	// Determine the proto type (same as primitive for numeric types)
	protoType := primitiveToProtoType(baseType.Name)

	// Forward: cast Go typedef to proto primitive type
	forward = fmt.Sprintf("%s(%s)", protoType, goField)
	// Backward: cast proto primitive to Go typedef
	backward = fmt.Sprintf("%s%s(%s)", typedefPrefix, resolved.Name, pbField)

	return forward, backward
}

// generateAliasConversion generates conversion for type aliases to primitives.
// For example: Key = uuid needs String() for forward and uuid.MustParse for backward.
func (p *Plugin) generateAliasConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	form resolution.AliasForm,
	data *templateData,
	goField, pbField string,
) (forward, backward string) {
	primitiveName := form.Target.Name

	// Get the Go alias name with package prefix
	aliasPrefix := ""
	goOutput := output.GetPath(resolved, "go")
	// Check if alias is in a different package (different namespace OR different output path)
	if resolved.Namespace != data.Namespace || (goOutput != "" && goOutput != data.ParentGoPath) {
		if goOutput != "" {
			importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
			if err == nil {
				// Use parent alias for conflict detection since we're in pb/ subdirectory
				alias := gointernal.DerivePackageAlias(goOutput, data.parentAlias)
				data.imports.AddInternal(alias, importPath)
				aliasPrefix = alias + "."
			}
		}
	} else {
		aliasPrefix = data.parentAlias + "."
	}

	// Handle uuid specially
	if primitiveName == "uuid" {
		data.imports.AddExternal("github.com/google/uuid")
		forward = fmt.Sprintf("%s.String()", goField)
		backward = fmt.Sprintf("%s%s(uuid.MustParse(%s))", aliasPrefix, resolved.Name, pbField)
		return forward, backward
	}

	// For other primitives, use standard primitive conversion and wrap
	protoType := primitiveToProtoType(primitiveName)
	forward = fmt.Sprintf("%s(%s)", protoType, goField)
	backward = fmt.Sprintf("%s%s(%s)", aliasPrefix, resolved.Name, pbField)
	return forward, backward
}

// generateArrayConversion generates conversion for array types.
// Returns forward expr, backward expr, and whether the conversion returns an error.
func (p *Plugin) generateArrayConversion(
	field resolution.Field,
	data *templateData,
	goField, pbField string,
) (forward, backward string, hasError bool) {
	typeRef := field.Type

	// Check if this is a nested array (array of arrays)
	if p.isNestedArrayType(typeRef, data.table) {
		return p.generateNestedArrayConversion(typeRef, data, goField, pbField)
	}

	// Get the element type, following aliases and distinct types
	elemType, ok := p.getArrayElementType(typeRef, data.table)
	if !ok {
		return goField, pbField, false
	}

	// For struct arrays, use slice helper
	elemResolved, ok := elemType.Resolve(data.table)
	if ok {
		if _, isStruct := elemResolved.Form.(resolution.StructForm); isStruct {
			// Use the helper to resolve translator info (handles extends chain)
			translatorPrefix, translatorStructName := p.resolvePBTranslatorInfo(elemResolved, data)

			// All slice translators return (result, error)
			return fmt.Sprintf("%s%ssToPB(ctx, %s)", translatorPrefix, translatorStructName, goField),
				fmt.Sprintf("%s%ssFromPB(ctx, %s)", translatorPrefix, translatorStructName, pbField),
				true
		}
	}

	// For primitive arrays
	if resolution.IsPrimitive(elemType.Name) {
		switch elemType.Name {
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

// generateNestedArrayConversion generates conversion for nested array types (array of arrays).
// This handles types like Strata which is Stratum[] where Stratum = string[].
// Proto uses wrapper messages for these (e.g., StratumWrapper with repeated string values).
func (p *Plugin) generateNestedArrayConversion(
	typeRef resolution.TypeRef,
	data *templateData,
	goField, pbField string,
) (forward, backward string, hasError bool) {
	wrapperName := p.getNestedArrayWrapperName(typeRef, data.table)

	data.imports.AddExternal("github.com/samber/lo")

	// Forward: wrap each inner array in a wrapper message
	// lo.Map(r.Strata, func(inner []string, _ int) *StratumWrapper { return &StratumWrapper{Values: inner} })
	forward = fmt.Sprintf("lo.Map(%s, func(inner []string, _ int) *%s { return &%s{Values: inner} })", goField, wrapperName, wrapperName)

	// Backward: unwrap each wrapper to get the inner array
	// lo.Map(pb.Strata, func(w *StratumWrapper, _ int) []string { return w.Values })
	backward = fmt.Sprintf("lo.Map(%s, func(w *%s, _ int) []string { return w.Values })", pbField, wrapperName)

	return forward, backward, false
}

// generateEnumTranslator generates translator data for an enum.
func (p *Plugin) generateEnumTranslator(
	enumRef *resolution.Type,
	data *templateData,
) *enumTranslatorData {
	form, ok := enumRef.Form.(resolution.EnumForm)
	if !ok {
		return nil
	}

	// Build enum value translations
	values := make([]enumValueTranslatorData, 0, len(form.Values))
	var goDefault string

	goAlias := data.parentAlias
	isGoOmitted := omit.IsType(*enumRef, "go")

	for i, v := range form.Values {
		valueName := gointernal.ToPascalCase(v.Name)

		var goValue string
		if isGoOmitted {
			// Hand-written Go enum: <Value><Enum>
			goValue = fmt.Sprintf("%s.%s%s", goAlias, valueName, enumRef.Name)
		} else {
			// Generated Go enum: <Enum><Value>
			goValue = fmt.Sprintf("%s.%s%s", goAlias, enumRef.Name, valueName)
		}

		// Proto enum value format in .proto file: ENUM_NAME_VALUE (e.g., VARIANT_SUCCESS)
		// Go protoc generates constants: EnumName_PROTO_VALUE (e.g., Variant_VARIANT_SUCCESS)
		// The pb/types plugin adds the enum name prefix to values, so we must include it here too
		enumPrefix := toScreamingSnake(enumRef.Name) + "_"
		pbValueName := fmt.Sprintf("%s_%s%s", enumRef.Name, enumPrefix, toScreamingSnake(v.Name))

		values = append(values, enumValueTranslatorData{
			GoValue: goValue,
			PBValue: pbValueName,
		})

		if i == 0 {
			goDefault = goValue
		}
	}

	// Proto default is the first enum value (oracle schema is source of truth)
	pbDefault := values[0].PBValue

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
	defer func() { _ = file.Close() }()

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
func getGoName(s resolution.Type) string {
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
func getPBName(s resolution.Type) string {
	if domain, ok := s.Domains["pb"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
}

// findStructWithPB walks up the extends chain to find the struct that has @pb.
// Returns the struct with @pb and its pb output path, or nil/"" if not found.
// This is needed because a child struct (e.g., SVCLabel) may not have @pb,
// but its parent (e.g., Label) does, and we need to use the parent's translators.
func findStructWithPB(s resolution.Type, table *resolution.Table) (*resolution.Type, string) {
	current := &s
	for current != nil {
		pbPath := output.GetPBPath(*current)
		if pbPath != "" {
			return current, pbPath
		}
		// Walk up the extends chain (use first parent for multiple inheritance)
		form, ok := current.Form.(resolution.StructForm)
		if !ok || len(form.Extends) == 0 {
			break
		}
		parent, ok := form.Extends[0].Resolve(table)
		if !ok {
			break
		}
		current = &parent
	}
	return nil, ""
}

// resolvePBTranslatorInfo resolves the translator package and function name for a struct.
// It handles cross-package references and inheritance chains.
// Returns: translatorPrefix (e.g., "labelpb."), translatorStructName (e.g., "Label")
func (p *Plugin) resolvePBTranslatorInfo(
	structRef resolution.Type,
	data *templateData,
) (translatorPrefix, translatorStructName string) {
	// Find the struct that actually has @pb (may be a parent)
	pbStruct, pbPath := findStructWithPB(structRef, data.table)
	if pbStruct == nil {
		// No @pb found in chain - use the struct's own name, no prefix
		return "", structRef.Name
	}

	// Use @pb name if specified, otherwise use the struct name
	translatorStructName = getPBName(*pbStruct)
	if translatorStructName == "" {
		translatorStructName = pbStruct.Name
	}

	// Check if we need a package prefix (different namespace OR different output path)
	if pbStruct.Namespace != data.Namespace || (pbPath != "" && pbPath != data.OutputPath) {
		importPath, err := resolveGoImportPath(pbPath, data.repoRoot)
		if err == nil {
			alias := strings.ToLower(pbStruct.Namespace) + "pb"
			data.imports.AddInternal(alias, importPath)
			translatorPrefix = alias + "."
		}
	}

	return translatorPrefix, translatorStructName
}

// collectDistinctPrimitives collects all distinct types that wrap primitives
// from the entire resolution table. These are used to generate conversion cases
// in the any converter helper function.
func (p *Plugin) collectDistinctPrimitives(data *templateData, req *plugin.Request) {
	seen := make(map[string]bool)

	for _, typ := range req.Resolutions.DistinctTypes() {
		form, ok := typ.Form.(resolution.DistinctForm)
		if !ok {
			continue
		}

		// Check if this distinct type wraps a primitive
		if !resolution.IsPrimitive(form.Base.Name) {
			continue
		}

		// Get the Go name for this distinct type
		goOutput := output.GetPath(typ, "go")
		if goOutput == "" {
			continue
		}

		// Import the package that defines this distinct type
		importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
		if err != nil {
			continue
		}

		alias := gointernal.DerivePackageName(goOutput)
		data.imports.AddInternal(alias, importPath)

		// Build the fully qualified Go type name
		goTypeName := fmt.Sprintf("%s.%s", alias, typ.Name)

		// Avoid duplicates
		if seen[goTypeName] {
			continue
		}
		seen[goTypeName] = true

		// Get the protobuf primitive type
		protoType := primitiveToProtoType(form.Base.Name)

		data.DistinctPrimitives = append(data.DistinctPrimitives, distinctPrimitiveData{
			GoType:        goTypeName,
			PrimitiveType: protoType,
		})
	}
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
	case "uint8", "uint12", "uint16", "uint20", "uint32":
		return "uint32"
	case "uint64":
		return "uint64"
	case "int8", "int16", "int32":
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

// isStructType checks if a type reference refers to a struct type.
// This follows aliases to check if the ultimate target is a struct.
func isStructType(typeRef resolution.TypeRef, table *resolution.Table) bool {
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return false
	}
	// Direct struct
	if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
		return true
	}
	// Alias to struct
	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		if target, ok := aliasForm.Target.Resolve(table); ok {
			_, isStruct := target.Form.(resolution.StructForm)
			return isStruct
		}
	}
	return false
}

// templateData holds data for translator file generation.
type templateData struct {
	Package               string
	OutputPath            string
	ParentGoPath          string
	Namespace             string
	Translators           []translatorData
	GenericTranslators    []genericTranslatorData
	EnumTranslators       []enumTranslatorData
	AnyHelpers            []anyHelperData
	DelegationTranslators []delegationTranslatorData
	DistinctPrimitives    []distinctPrimitiveData
	imports               *gointernal.ImportManager
	repoRoot              string
	table                 *resolution.Table
	usedEnums             map[string]*resolution.Type
	parentAlias           string
	generatedAnyHelpers   map[string]bool
	needsAnyConverter     bool
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
	HasError         bool // True if forward conversion returns (result, error)
	HasBackwardError bool // True if backward conversion returns (result, error)
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

// delegationTranslatorData holds data for translators that delegate to an underlying type.
// Used for DistinctForm types that wrap struct types - instead of generating independent
// translators, we generate thin wrappers that cast and delegate.
type delegationTranslatorData struct {
	Name                       string          // e.g., "Status" (the typedef name)
	GoType                     string          // e.g., "status.Status[Details]" (local typedef)
	TypeParams                 []typeParamData // Type parameters from the typedef
	UnderlyingName             string          // e.g., "Status" (underlying struct name)
	UnderlyingGoType           string          // e.g., "gostatus.Status[Details]" (for casting)
	UnderlyingPBType           string          // e.g., "gostatus_pb.Status" (proto type)
	UnderlyingTranslatorPrefix string          // e.g., "gostatus_pb." (import prefix for translator)
}

// distinctPrimitiveData holds data for distinct types that wrap primitives.
// Used to generate conversion cases in the any converter helper function.
type distinctPrimitiveData struct {
	GoType        string // e.g., "telem.TimeSpan"
	PrimitiveType string // e.g., "int64"
}
