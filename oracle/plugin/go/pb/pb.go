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

	outputStructs := make(map[string][]resolution.Type)
	outputTypeDefs := make(map[string][]resolution.Type)
	var outputOrder []string

	for _, entry := range req.Resolutions.StructTypes() {
		outputPath := output.GetPBPath(entry)
		if outputPath == "" {
			continue
		}

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

	for _, entry := range req.Resolutions.DistinctTypes() {
		outputPath := output.GetPBPath(entry)
		if outputPath == "" {
			continue
		}
		if omit.IsType(entry, "pb") {
			continue
		}

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

func (p *Plugin) generateFile(
	outputPath string,
	structs []resolution.Type,
	typeDefs []resolution.Type,
	enums []resolution.Type,
	req *plugin.Request,
) ([]byte, error) {
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	} else if len(typeDefs) > 0 {
		namespace = typeDefs[0].Namespace
	}

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

	data.imports.AddExternal("context")

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

	for i := range enums {
		e := enums[i]
		if omit.IsType(e, "pb") {
			continue
		}
		enumPBPath := output.GetPBPath(e)
		if enumPBPath != "" && enumPBPath != outputPath {
			continue
		}
		data.usedEnums[e.QualifiedName] = &e
	}

	for _, enumRef := range data.usedEnums {
		enumTranslator := p.generateEnumTranslator(enumRef, data)
		if enumTranslator != nil {
			data.EnumTranslators = append(data.EnumTranslators, *enumTranslator)
		}
	}

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

	if data.needsAnyConverter {
		p.collectDistinctPrimitives(data, req)
	}

	if len(data.Translators) == 0 && len(data.GenericTranslators) == 0 && len(data.EnumTranslators) == 0 && len(data.DelegationTranslators) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	if err := translatorFileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Plugin) processStructForTranslation(
	s resolution.Type,
	form resolution.StructForm,
	data *templateData,
	req *plugin.Request,
) (*translatorData, error) {
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

func (p *Plugin) processFieldForTranslation(
	field resolution.Field,
	data *templateData,
	parentStruct resolution.Type,
) fieldTranslatorData {
	goName := gointernal.ToPascalCase(field.Name)
	pbName := gointernal.ToPascalCase(lo.SnakeCase(field.Name))

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

func (p *Plugin) processGenericStructForTranslation(
	s resolution.Type,
	form resolution.StructForm,
	data *templateData,
	req *plugin.Request,
) (*genericTranslatorData, error) {
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

	typeParams := make([]typeParamData, 0, len(form.TypeParams))
	typeParamNames := make([]string, 0, len(form.TypeParams))
	for _, tp := range form.TypeParams {
		if tp.HasDefault() {
			continue
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

func (p *Plugin) processGenericFieldForTranslation(
	field resolution.Field,
	data *templateData,
	parentStruct resolution.Type,
	parentForm resolution.StructForm,
	typeParams []typeParamData,
) (fieldTranslatorData, bool) {
	goName := gointernal.ToPascalCase(field.Name)
	pbName := gointernal.ToPascalCase(lo.SnakeCase(field.Name))
	typeRef := field.Type

	isHardOptional := field.IsHardOptional
	isOptional := isHardOptional

	goFieldName := "r." + goName
	pbFieldName := "pb." + pbName

	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		if typeRef.TypeParam.HasDefault() {
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
			}, false
		}

		paramName := typeRef.TypeParam.Name
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
			HasError:         true,
			HasBackwardError: true,
		}, true
	}

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
	}, false
}

func (p *Plugin) processDelegationTranslator(
	td resolution.Type,
	form resolution.DistinctForm,
	data *templateData,
	req *plugin.Request,
) (*delegationTranslatorData, error) {
	goName := getGoName(td)
	if goName == "" {
		goName = td.Name
	}

	typeParams := make([]typeParamData, 0, len(form.TypeParams))
	typeParamNames := make([]string, 0, len(form.TypeParams))
	for _, tp := range form.TypeParams {
		if tp.HasDefault() {
			continue
		}
		typeParams = append(typeParams, typeParamData{Name: tp.Name, Constraint: "any"})
		typeParamNames = append(typeParamNames, tp.Name)
	}

	goType := fmt.Sprintf("%s.%s", data.parentAlias, goName)
	if len(typeParamNames) > 0 {
		goType = fmt.Sprintf("%s.%s[%s]", data.parentAlias, goName, strings.Join(typeParamNames, ", "))
	}

	underlyingResolved, ok := form.Base.Resolve(data.table)
	if !ok {
		return nil, nil
	}

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

	underlyingGoName := getGoName(actualStruct)
	if underlyingGoName == "" {
		underlyingGoName = actualStruct.Name
	}

	underlyingPBPath := output.GetPBPath(actualStruct)
	if underlyingPBPath == "" {
		_, underlyingPBPath = findStructWithPB(actualStruct, data.table)
		if underlyingPBPath == "" {
			return nil, nil
		}
	}

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

	underlyingPBImportPath, err := resolveGoImportPath(underlyingPBPath, data.repoRoot)
	if err != nil {
		return nil, err
	}
	underlyingPBAlias := underlyingGoAlias + "_pb"
	data.imports.AddInternal(underlyingPBAlias, underlyingPBImportPath)

	if len(typeParams) > 0 {
		data.imports.AddExternal("google.golang.org/protobuf/types/known/anypb")
	}

	underlyingGoType := fmt.Sprintf("%s.%s", underlyingGoAlias, underlyingGoName)
	if len(typeParamNames) > 0 {
		underlyingGoType = fmt.Sprintf("%s.%s[%s]", underlyingGoAlias, underlyingGoName, strings.Join(typeParamNames, ", "))
	}

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

func (p *Plugin) isArrayType(typeRef resolution.TypeRef, table *resolution.Table) bool {
	if typeRef.Name == "Array" {
		return true
	}

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

func (p *Plugin) getArrayElementType(typeRef resolution.TypeRef, table *resolution.Table) (resolution.TypeRef, bool) {
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		return typeRef.TypeArgs[0], true
	}

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

func (p *Plugin) getNestedArrayWrapperName(typeRef resolution.TypeRef, table *resolution.Table) string {
	elemType, ok := p.getArrayElementType(typeRef, table)
	if !ok {
		return "ArrayWrapper"
	}

	resolved, ok := elemType.Resolve(table)
	if ok {
		return resolved.Name + "Wrapper"
	}

	if resolution.IsPrimitive(elemType.Name) {
		return cases.Title(language.English).String(elemType.Name) + "Array"
	}

	return "ArrayWrapper"
}

func (p *Plugin) isFixedSizeUint8Array(typeRef resolution.TypeRef, table *resolution.Table) bool {
	arraySize := p.getArraySize(typeRef, table)
	if arraySize == nil {
		return false
	}

	elemType, ok := p.getArrayElementType(typeRef, table)
	if !ok {
		return false
	}

	resolved, ok := elemType.Resolve(table)
	if !ok {
		return elemType.Name == "uint8"
	}

	if prim, ok := resolved.Form.(resolution.PrimitiveForm); ok {
		return prim.Name == "uint8"
	}
	return false
}

func (p *Plugin) getArraySize(typeRef resolution.TypeRef, table *resolution.Table) *int64 {
	if typeRef.Name == "Array" && typeRef.ArraySize != nil {
		return typeRef.ArraySize
	}

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

func (p *Plugin) generateFixedSizeUint8ArrayConversion(
	typeRef resolution.TypeRef,
	data *templateData,
	goField, pbField string,
) (forward, backward string) {
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return fmt.Sprintf("%s[:]", goField), pbField
	}

	goOutput := output.GetPath(resolved, "go")
	if goOutput == "" {
		return fmt.Sprintf("%s[:]", goField), pbField
	}

	importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
	if err != nil {
		return fmt.Sprintf("%s[:]", goField), pbField
	}

	alias := gointernal.DerivePackageName(goOutput)
	data.imports.AddInternal(alias, importPath)

	forward = fmt.Sprintf("%s.Bytes()", goField)
	backward = fmt.Sprintf("%s.FromBytes(%s)", alias, pbField)

	return forward, backward
}

func (p *Plugin) generateFieldConversion(
	field resolution.Field,
	data *templateData,
	parentStruct resolution.Type,
) (forward, backward, backwardCast string, hasError, hasBackwardError bool) {
	typeRef := field.Type
	goFieldName := "r." + gointernal.ToPascalCase(field.Name)
	pbFieldName := "pb." + gointernal.ToPascalCase(lo.SnakeCase(field.Name))

	if p.isFixedSizeUint8Array(typeRef, data.table) {
		f, b := p.generateFixedSizeUint8ArrayConversion(typeRef, data, goFieldName, pbFieldName)
		return f, b, "", false, false
	}

	if p.isArrayType(typeRef, data.table) {
		f, b, e := p.generateArrayConversion(field, data, goFieldName, pbFieldName)
		return f, b, "", e, e
	}

	if hasKeyDomain(field) && resolution.IsPrimitive(typeRef.Name) && isNumericPrimitive(typeRef.Name) {
		protoType := primitiveToProtoType(typeRef.Name)
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

	if resolution.IsPrimitive(typeRef.Name) {
		f, b, e, be := p.generatePrimitiveConversion(typeRef.Name, goFieldName, pbFieldName, data)
		return f, b, "", e, be
	}

	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return goFieldName, pbFieldName, "", false, false
	}

	if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
		f, b, c, hasErr := p.generateStructConversion(typeRef, resolved, field.IsHardOptional, data, goFieldName, pbFieldName)
		return f, b, c, hasErr, hasErr
	}

	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		if target, ok := aliasForm.Target.Resolve(data.table); ok {
			if _, isStruct := target.Form.(resolution.StructForm); isStruct {
				f, b, c, hasErr := p.generateStructConversion(typeRef, resolved, field.IsHardOptional, data, goFieldName, pbFieldName)
				return f, b, c, hasErr, hasErr
			}
		}
	}

	if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
		f, b := p.generateEnumConversion(typeRef, resolved, data, goFieldName, pbFieldName)
		return f, b, "", false, false
	}

	if form, isDistinct := resolved.Form.(resolution.DistinctForm); isDistinct {
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

	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		if resolution.IsPrimitive(aliasForm.Target.Name) {
			f, b := p.generateAliasConversion(typeRef, resolved, aliasForm, data, goFieldName, pbFieldName)
			return f, b, "", false, false
		}
	}

	return goFieldName, pbFieldName, "", false, false
}

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
		return fmt.Sprintf("structpb.NewValue(convertAnyForPB(%s))", goField),
			fmt.Sprintf("%s.AsInterface()", pbField), true, false
	case "int8":
		return fmt.Sprintf("int32(%s)", goField),
			fmt.Sprintf("int8(%s)", pbField), false, false
	case "uint8":
		return fmt.Sprintf("uint32(%s)", goField),
			fmt.Sprintf("uint8(%s)", pbField), false, false
	default:
		return goField, pbField, false, false
	}
}

func (p *Plugin) generateStructConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	isHardOptional bool,
	data *templateData,
	goField, pbField string,
) (forward, backward, backwardCast string, hasError bool) {
	actualStruct := resolved
	var typeArgs []resolution.TypeRef

	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		typeArgs = aliasForm.Target.TypeArgs
		if target, ok := aliasForm.Target.Resolve(data.table); ok {
			actualStruct = target
		}
	}

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

	if actualForm.IsGeneric() {
		var nonDefaultedParams []resolution.TypeParam
		for _, tp := range actualForm.TypeParams {
			if !tp.HasDefault() {
				nonDefaultedParams = append(nonDefaultedParams, tp)
			}
		}
		providedArgs := len(typeArgs)
		if providedArgs < len(nonDefaultedParams) {
			newTypeArgs := make([]resolution.TypeRef, len(nonDefaultedParams))
			copy(newTypeArgs, typeArgs)
			for i := providedArgs; i < len(nonDefaultedParams); i++ {
				if nonDefaultedParams[i].Optional {
					newTypeArgs[i] = resolution.TypeRef{Name: "nil"}
				}
			}
			typeArgs = newTypeArgs
		}
	}

	if actualForm.IsGeneric() && len(typeArgs) > 0 {
		return p.generateGenericStructConversion(typeRef, resolved, actualStruct, actualForm, typeArgs, data, goField, pbField, isHardOptional)
	}

	if actualForm.IsGeneric() {
		return goField, pbField, "", false
	}

	translatorPrefix, translatorStructName := p.resolvePBTranslatorInfo(actualStruct, data)

	if isHardOptional {
		return fmt.Sprintf("%s%sToPB(ctx, *%s)", translatorPrefix, translatorStructName, goField),
			fmt.Sprintf("%s%sFromPB(ctx, %s)", translatorPrefix, translatorStructName, pbField), "", true
	}

	return fmt.Sprintf("%s%sToPB(ctx, %s)", translatorPrefix, translatorStructName, goField),
		fmt.Sprintf("%s%sFromPB(ctx, %s)", translatorPrefix, translatorStructName, pbField), "", true
}

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
	translatorPrefix, structName := p.resolvePBTranslatorInfo(actualStruct, data)

	var forwardConverters, backwardConverters []string
	var explicitTypeArgs []string
	for i, typeArg := range typeArgs {
		if i < len(actualForm.TypeParams) && actualForm.TypeParams[i].HasDefault() {
			continue
		}

		argResolved, ok := typeArg.Resolve(data.table)
		if ok {
			if _, isStruct := argResolved.Form.(resolution.StructForm); isStruct {
				argGoName := getGoName(argResolved)
				if argGoName == "" {
					argGoName = argResolved.Name
				}

				p.ensureAnyHelper(argResolved, data)

				forwardConverters = append(forwardConverters, fmt.Sprintf("%sToPBAny", argGoName))
				backwardConverters = append(backwardConverters, fmt.Sprintf("%sFromPBAny", argGoName))

				explicitTypeArgs = append(explicitTypeArgs, fmt.Sprintf("%s.%s", data.parentAlias, argGoName))
				continue
			}
		}
		forwardConverters = append(forwardConverters, "nil")
		backwardConverters = append(backwardConverters, "nil")
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

	aliasGoName := getGoName(originalResolved)
	if aliasGoName == "" {
		aliasGoName = originalResolved.Name
	}
	if isHardOptional {
		if genericGoType != "" {
			forward = fmt.Sprintf("%s%sToPB%s(ctx, (%s)(*%s), %s)", translatorPrefix, structName, typeArgsStr, genericGoType, goField, forwardArgs)
		} else {
			forward = fmt.Sprintf("%s%sToPB%s(ctx, *%s, %s)", translatorPrefix, structName, typeArgsStr, goField, forwardArgs)
		}
		backward = fmt.Sprintf("%s%sFromPB%s(ctx, %s, %s)", translatorPrefix, structName, typeArgsStr, pbField, backwardArgs)
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

func (p *Plugin) ensureAnyHelper(s resolution.Type, data *templateData) {
	key := s.QualifiedName
	if data.generatedAnyHelpers[key] {
		return
	}
	data.generatedAnyHelpers[key] = true

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

func (p *Plugin) generateEnumConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	data *templateData,
	goField, pbField string,
) (forward, backward string) {
	enumName := resolved.Name

	if resolved.Namespace != data.Namespace {
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

	if _, exists := data.usedEnums[resolved.QualifiedName]; !exists {
		data.usedEnums[resolved.QualifiedName] = &resolved
	}

	return fmt.Sprintf("%sToPB(%s)", enumName, goField),
		fmt.Sprintf("%sFromPB(%s)", enumName, pbField)
}

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

func (p *Plugin) generateTypeDefConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	form resolution.DistinctForm,
	data *templateData,
	goField, pbField string,
) (forward, backward string) {
	baseType := form.Base
	if !resolution.IsPrimitive(baseType.Name) {
		return goField, pbField
	}

	typedefPrefix := ""
	goOutput := output.GetPath(resolved, "go")
	if resolved.Namespace != data.Namespace || (goOutput != "" && goOutput != data.ParentGoPath) {
		if goOutput != "" {
			importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
			if err == nil {
				alias := gointernal.DerivePackageAlias(goOutput, data.parentAlias)
				data.imports.AddInternal(alias, importPath)
				typedefPrefix = alias + "."
			}
		}
	} else {
		typedefPrefix = data.parentAlias + "."
	}

	if baseType.Name == "uuid" {
		data.imports.AddExternal("github.com/google/uuid")
		forward = fmt.Sprintf("%s.String()", goField)
		backward = fmt.Sprintf("%s%s(uuid.MustParse(%s))", typedefPrefix, resolved.Name, pbField)
		return forward, backward
	}

	protoType := primitiveToProtoType(baseType.Name)

	forward = fmt.Sprintf("%s(%s)", protoType, goField)
	backward = fmt.Sprintf("%s%s(%s)", typedefPrefix, resolved.Name, pbField)

	return forward, backward
}

func (p *Plugin) generateAliasConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	form resolution.AliasForm,
	data *templateData,
	goField, pbField string,
) (forward, backward string) {
	primitiveName := form.Target.Name

	aliasPrefix := ""
	goOutput := output.GetPath(resolved, "go")
	if resolved.Namespace != data.Namespace || (goOutput != "" && goOutput != data.ParentGoPath) {
		if goOutput != "" {
			importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
			if err == nil {
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

	protoType := primitiveToProtoType(primitiveName)
	forward = fmt.Sprintf("%s(%s)", protoType, goField)
	backward = fmt.Sprintf("%s%s(%s)", aliasPrefix, resolved.Name, pbField)
	return forward, backward
}

func (p *Plugin) generateArrayConversion(
	field resolution.Field,
	data *templateData,
	goField, pbField string,
) (forward, backward string, hasError bool) {
	typeRef := field.Type

	if p.isNestedArrayType(typeRef, data.table) {
		return p.generateNestedArrayConversion(typeRef, data, goField, pbField)
	}

	elemType, ok := p.getArrayElementType(typeRef, data.table)
	if !ok {
		return goField, pbField, false
	}

	elemResolved, ok := elemType.Resolve(data.table)
	if ok {
		if _, isStruct := elemResolved.Form.(resolution.StructForm); isStruct {
			translatorPrefix, translatorStructName := p.resolvePBTranslatorInfo(elemResolved, data)

			return fmt.Sprintf("%s%ssToPB(ctx, %s)", translatorPrefix, translatorStructName, goField),
				fmt.Sprintf("%s%ssFromPB(ctx, %s)", translatorPrefix, translatorStructName, pbField),
				true
		}
	}

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

func (p *Plugin) generateNestedArrayConversion(
	typeRef resolution.TypeRef,
	data *templateData,
	goField, pbField string,
) (forward, backward string, hasError bool) {
	wrapperName := p.getNestedArrayWrapperName(typeRef, data.table)

	data.imports.AddExternal("github.com/samber/lo")

	forward = fmt.Sprintf("lo.Map(%s, func(inner []string, _ int) *%s { return &%s{Values: inner} })", goField, wrapperName, wrapperName)
	backward = fmt.Sprintf("lo.Map(%s, func(w *%s, _ int) []string { return w.Values })", pbField, wrapperName)

	return forward, backward, false
}

func (p *Plugin) generateEnumTranslator(
	enumRef *resolution.Type,
	data *templateData,
) *enumTranslatorData {
	form, ok := enumRef.Form.(resolution.EnumForm)
	if !ok {
		return nil
	}

	values := make([]enumValueTranslatorData, 0, len(form.Values))
	var goDefault string

	goAlias := data.parentAlias
	isGoOmitted := omit.IsType(*enumRef, "go")

	for i, v := range form.Values {
		valueName := gointernal.ToPascalCase(v.Name)

		var goValue string
		if isGoOmitted {
			goValue = fmt.Sprintf("%s.%s%s", goAlias, valueName, enumRef.Name)
		} else {
			goValue = fmt.Sprintf("%s.%s%s", goAlias, enumRef.Name, valueName)
		}

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

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

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

func findStructWithPB(s resolution.Type, table *resolution.Table) (*resolution.Type, string) {
	current := &s
	for current != nil {
		pbPath := output.GetPBPath(*current)
		if pbPath != "" {
			return current, pbPath
		}
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

func (p *Plugin) resolvePBTranslatorInfo(
	structRef resolution.Type,
	data *templateData,
) (translatorPrefix, translatorStructName string) {
	pbStruct, pbPath := findStructWithPB(structRef, data.table)
	if pbStruct == nil {
		return "", structRef.Name
	}

	translatorStructName = getPBName(*pbStruct)
	if translatorStructName == "" {
		translatorStructName = pbStruct.Name
	}

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

func (p *Plugin) collectDistinctPrimitives(data *templateData, req *plugin.Request) {
	seen := make(map[string]bool)

	for _, typ := range req.Resolutions.DistinctTypes() {
		form, ok := typ.Form.(resolution.DistinctForm)
		if !ok {
			continue
		}

		if !resolution.IsPrimitive(form.Base.Name) {
			continue
		}

		goOutput := output.GetPath(typ, "go")
		if goOutput == "" {
			continue
		}

		importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
		if err != nil {
			continue
		}

		alias := gointernal.DerivePackageName(goOutput)
		data.imports.AddInternal(alias, importPath)

		goTypeName := fmt.Sprintf("%s.%s", alias, typ.Name)

		if seen[goTypeName] {
			continue
		}
		seen[goTypeName] = true

		protoType := primitiveToProtoType(form.Base.Name)

		data.DistinctPrimitives = append(data.DistinctPrimitives, distinctPrimitiveData{
			GoType:        goTypeName,
			PrimitiveType: protoType,
		})
	}
}

func hasKeyDomain(field resolution.Field) bool {
	_, hasKey := field.Domains["key"]
	return hasKey
}

func isNumericPrimitive(primitive string) bool {
	switch primitive {
	case "uint8", "uint16", "uint32", "uint64",
		"int8", "int16", "int32", "int64":
		return true
	default:
		return false
	}
}

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

func toScreamingSnake(s string) string {
	return strings.ToUpper(lo.SnakeCase(s))
}

func fieldUsesContext(fields []fieldTranslatorData) bool {
	for _, f := range fields {
		if strings.Contains(f.ForwardExpr, "ctx") || strings.Contains(f.BackwardExpr, "ctx") {
			return true
		}
	}
	return false
}

func isStructType(typeRef resolution.TypeRef, table *resolution.Table) bool {
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return false
	}
	if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
		return true
	}
	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		if target, ok := aliasForm.Target.Resolve(table); ok {
			_, isStruct := target.Form.(resolution.StructForm)
			return isStruct
		}
	}
	return false
}

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
	Name        string
	GoType      string
	PBType      string
	GoTypeShort string
	PBTypeShort string
	Fields      []fieldTranslatorData
	// ErrorFields holds fields with error-returning conversions.
	ErrorFields    []fieldTranslatorData
	OptionalFields []fieldTranslatorData
	UsesContext    bool
}

// fieldTranslatorData holds data for a single field translation.
type fieldTranslatorData struct {
	GoName       string
	PBName       string
	ForwardExpr  string
	BackwardExpr string
	// BackwardCast is an optional cast for the backward assignment (e.g., "(*rack.Status)").
	BackwardCast     string
	IsOptional       bool
	IsOptionalStruct bool
	// HasError is true if forward conversion returns (result, error).
	HasError bool
	// HasBackwardError is true if backward conversion returns (result, error).
	HasBackwardError bool
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
	Name string
	// GoType is the full generic type with parameters (e.g., "status.Status[D]").
	GoType string
	// GoTypeBase is the type without parameters (e.g., "status.Status").
	GoTypeBase  string
	PBType      string
	GoTypeShort string
	PBTypeShort string
	TypeParams  []typeParamData
	// Fields holds regular fields that don't use type parameters.
	Fields []fieldTranslatorData
	// ErrorFields holds fields with error-returning conversions.
	ErrorFields []fieldTranslatorData
	// TypeParamFields holds fields that use type parameters and need error handling.
	TypeParamFields []fieldTranslatorData
	OptionalFields  []fieldTranslatorData
	UsesContext     bool
}

// typeParamData holds data for a type parameter in a generic translator.
type typeParamData struct {
	// Name is the type parameter name (e.g., "D").
	Name string
	// Constraint is the Go type constraint (e.g., "any").
	Constraint string
}

// anyHelperData holds data for ToPBAny/FromPBAny helper functions.
// These are generated for concrete types that are used as type arguments
// to generic structs.
type anyHelperData struct {
	// TypeName is the unqualified type name (e.g., "StatusDetails").
	TypeName string
	// GoType is the fully qualified Go type (e.g., "rack.StatusDetails").
	GoType string
	// PBType is the protobuf message type name (e.g., "PBStatusDetails").
	PBType string
}

// delegationTranslatorData holds data for translators that delegate to an underlying type.
// Used for DistinctForm types that wrap struct types - instead of generating independent
// translators, we generate thin wrappers that cast and delegate.
type delegationTranslatorData struct {
	// Name is the typedef name (e.g., "Status").
	Name string
	// GoType is the local typedef with type parameters (e.g., "status.Status[Details]").
	GoType string
	// TypeParams holds the type parameters from the typedef.
	TypeParams []typeParamData
	// UnderlyingName is the underlying struct name (e.g., "Status").
	UnderlyingName string
	// UnderlyingGoType is the underlying Go type for casting (e.g., "gostatus.Status[Details]").
	UnderlyingGoType string
	// UnderlyingPBType is the protobuf type (e.g., "gostatus_pb.Status").
	UnderlyingPBType string
	// UnderlyingTranslatorPrefix is the import prefix for the translator (e.g., "gostatus_pb.").
	UnderlyingTranslatorPrefix string
}

// distinctPrimitiveData holds data for distinct types that wrap primitives.
// Used to generate conversion cases in the any converter helper function.
type distinctPrimitiveData struct {
	// GoType is the fully qualified Go type (e.g., "telem.TimeSpan").
	GoType string
	// PrimitiveType is the proto primitive type (e.g., "int64").
	PrimitiveType string
}
