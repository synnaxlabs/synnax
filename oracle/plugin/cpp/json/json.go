// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/cpp/keywords"
	cppprimitives "github.com/synnaxlabs/oracle/plugin/cpp/primitives"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

var primitiveMapper = cppprimitives.Mapper()

type Plugin struct{ options Options }

type Options struct {
	FileNamePattern string
}

func DefaultOptions() Options {
	return Options{
		FileNamePattern: "json.gen.h",
	}
}

func New(opts Options) *Plugin { return &Plugin{options: opts} }

func (*Plugin) Name() string { return "cpp/json" }

func (*Plugin) Domains() []string { return []string{"cpp"} }

func (*Plugin) Requires() []string { return []string{"cpp/types"} }

func (*Plugin) Check(*plugin.Request) error { return nil }

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	c, err := collect(req)
	if err != nil {
		return nil, err
	}

	for outputPath := range c.jsonPaths {
		structs := c.structs.Get(outputPath)
		distinctTypes := c.distinct.Get(outputPath)
		unions := c.unions.Get(outputPath)

		var namespace string
		if len(structs) > 0 {
			namespace = structs[0].Namespace
		} else if len(distinctTypes) > 0 {
			namespace = distinctTypes[0].Namespace
		} else {
			namespace = unions[0].Namespace
		}

		content, err := p.generateFile(
			outputPath, structs, distinctTypes, unions, namespace, c.jsonPaths, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate json for %s", outputPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.options.FileNamePattern),
			Content: content,
		})
	}

	// Request deletion for every other C++ output path so stale files from schemas that
	// no longer have anything to serialize get cleaned up.
	deletions := make(set.Set[string])
	for _, t := range req.Resolutions.TypesWithDomain("cpp") {
		outputPath := output.GetPath(t, "cpp")
		if outputPath == "" || c.jsonPaths.Contains(outputPath) {
			continue
		}
		if req.RepoRoot != "" && req.ValidateOutputPath(outputPath) != nil {
			continue
		}
		deletions.Add(fmt.Sprintf("%s/%s", outputPath, p.options.FileNamePattern))
	}
	for d := range deletions {
		resp.Deletions = append(resp.Deletions, d)
	}

	return resp, nil
}

// ContentPaths returns the output paths for which the plugin emits a json.gen.h file.
// Other plugins use it to decide whether a cross-package reference can include
// json.gen.h or must fall back to types.gen.h.
func ContentPaths(req *plugin.Request) (set.Set[string], error) {
	c, err := collect(req)
	if err != nil {
		return nil, err
	}
	return c.jsonPaths, nil
}

// IncludeFor returns the header to include for a reference to types generated at
// outputPath: json.gen.h when that path emits one, types.gen.h otherwise (scalar-only
// packages use default JSON serialization and get no json.gen.h).
func IncludeFor(jsonPaths set.Set[string], outputPath string) string {
	if jsonPaths.Contains(outputPath) {
		return outputPath + "/json.gen.h"
	}
	return outputPath + "/types.gen.h"
}

// collection holds the per-path type collectors and the set of paths with JSON content.
// Collectors exclude cpp-omitted types.
type collection struct {
	structs, distinct, unions *framework.Collector
	jsonPaths                 set.Set[string]
}

func collect(req *plugin.Request) (collection, error) {
	var c collection
	var err error
	if c.structs, err = framework.CollectStructs("cpp", req); err != nil {
		return c, err
	}
	if c.distinct, err = framework.CollectDistinct("cpp", req); err != nil {
		return c, err
	}
	c.unions = framework.NewCollector("cpp", req)
	if err = c.unions.AddAll(req.Resolutions.UnionTypes()); err != nil {
		return c, err
	}

	c.jsonPaths = make(set.Set[string])
	for _, path := range c.structs.Paths() {
		c.jsonPaths.Add(path)
	}
	for _, path := range c.unions.Paths() {
		c.jsonPaths.Add(path)
	}
	for _, path := range c.distinct.Paths() {
		if lo.SomeBy(c.distinct.Get(path), isVariableLengthArray) {
			c.jsonPaths.Add(path)
		}
	}
	return c, nil
}

// isVariableLengthArray reports whether t is a distinct type wrapping a variable-length
// array, the only distinct form that needs JSON serialization.
func isVariableLengthArray(t resolution.Type) bool {
	form, ok := t.Form.(resolution.DistinctForm)
	return ok && form.Base.Name == "Array" && len(form.Base.TypeArgs) > 0 &&
		form.Base.ArraySize == nil
}

func (p *Plugin) generateFile(
	outputPath string,
	structs []resolution.Type,
	distinctTypes []resolution.Type,
	unions []resolution.Type,
	namespace string,
	jsonPaths set.Set[string],
	req *plugin.Request,
) ([]byte, error) {
	data := &templateData{
		OutputPath:    outputPath,
		Namespace:     deriveNamespace(outputPath),
		Serializers:   make([]serializerData, 0, len(structs)),
		ArrayWrappers: make([]arrayWrapperData, 0),
		includes:      newIncludeManager(),
		table:         req.Resolutions,
		jsonPaths:     jsonPaths,
		rawNs:         namespace,
	}

	data.includes.addInternal(fmt.Sprintf("%s/types.gen.h", outputPath))
	data.includes.addInternal("x/cpp/json/json.h")

	for _, s := range structs {
		serializer := p.processStruct(s, data)
		if serializer != nil {
			data.Serializers = append(data.Serializers, *serializer)
		}
	}

	for _, dt := range distinctTypes {
		wrapper := p.processArrayWrapper(dt, data)
		if wrapper != nil {
			data.ArrayWrappers = append(data.ArrayWrappers, *wrapper)
		}
	}

	for _, u := range unions {
		variantSerializers, dispatch := p.processUnion(u, data)
		data.Serializers = append(data.Serializers, variantSerializers...)
		data.Unions = append(data.Unions, dispatch)
	}

	var buf bytes.Buffer
	if err := jsonTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Plugin) processArrayWrapper(
	dt resolution.Type,
	data *templateData,
) *arrayWrapperData {
	if !isVariableLengthArray(dt) {
		return nil
	}
	form := dt.Form.(resolution.DistinctForm)

	name := domain.GetName(dt, "cpp")
	elemType := form.Base.TypeArgs[0]
	elemCppType := p.typeRefToCpp(elemType, data)

	elemNeedsConversion := false
	if elemResolved, ok := elemType.Resolve(data.table); ok {
		if _, isStruct := elemResolved.Form.(resolution.StructForm); isStruct {
			elemNeedsConversion = true
		}
	}

	return &arrayWrapperData{
		Name:                name,
		ElementType:         elemCppType,
		ElementNeedsConvert: elemNeedsConversion,
	}
}

func (p *Plugin) resolveExtendsType(
	extendsRef resolution.TypeRef,
	parent resolution.Type,
	data *templateData,
) string {
	name := domain.GetName(parent, "cpp")

	if parent.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(parent, "cpp")
		if targetOutputPath != "" {
			data.includes.addInternal(data.jsonInclude(targetOutputPath))
			ns := deriveNamespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	}

	if len(extendsRef.TypeArgs) > 0 {
		args := make([]string, 0, len(extendsRef.TypeArgs))
		for _, arg := range extendsRef.TypeArgs {
			args = append(args, p.typeRefToCpp(arg, data))
		}
		name = fmt.Sprintf("%s<%s>", name, strings.Join(args, ", "))
	}

	return name
}

func (p *Plugin) processStruct(
	s resolution.Type,
	data *templateData,
) *serializerData {
	form, ok := s.Form.(resolution.StructForm)
	if !ok {
		return nil
	}

	cppName := domain.GetName(s, "cpp")

	typeParams := make([]typeParamData, 0, len(form.TypeParams))
	typeParamNames := make([]string, 0, len(form.TypeParams))
	for _, tp := range form.TypeParams {
		if tp.HasDefault() {
			continue
		}
		typeParams = append(typeParams, typeParamData{Name: tp.Name})
		typeParamNames = append(typeParamNames, tp.Name)
	}

	serializer := &serializerData{
		Name:           cppName,
		IsGeneric:      len(typeParams) > 0,
		TypeParams:     typeParams,
		TypeParamNames: strings.Join(typeParamNames, ", "),
		Fields:         make([]fieldData, 0),
	}

	if len(typeParams) > 0 {
		data.includes.addSystem("type_traits")
	}

	if resolver.CanUseInheritance(form, data.table) {
		serializer.HasExtends = true
		for _, extendsRef := range form.Extends {
			parent, ok := extendsRef.Resolve(data.table)
			if !ok {
				continue
			}
			qualifiedName := p.resolveExtendsType(extendsRef, parent, data)
			serializer.ParentTypes = append(serializer.ParentTypes, parentTypeData{
				QualifiedName: qualifiedName,
			})
		}
		for _, field := range form.Fields {
			fieldData := p.processField(field, s, data)
			serializer.Fields = append(serializer.Fields, fieldData)
		}
	} else {
		for _, field := range resolution.UnifiedFields(s, data.table) {
			fieldData := p.processField(field, s, data)
			serializer.Fields = append(serializer.Fields, fieldData)
		}
	}

	return serializer
}

// isSelfReference reports whether t directly or transitively references parent.
// Stays consistent with the cpp types plugin's decision on optional
// fields by sharing resolution.RefersTo.
func isSelfReference(
	t resolution.TypeRef,
	parent resolution.Type,
	table *resolution.Table,
) bool {
	return resolution.RefersTo(t, parent.QualifiedName, table)
}

func (*Plugin) resolveToArrayElement(
	typeRef resolution.TypeRef,
	data *templateData,
) (resolution.TypeRef, bool) {
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		return typeRef.TypeArgs[0], true
	}

	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return resolution.TypeRef{}, false
	}

	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		if aliasForm.Target.Name == "Array" && len(aliasForm.Target.TypeArgs) > 0 {
			return aliasForm.Target.TypeArgs[0], true
		}
	}

	if distinctForm, isDistinct := resolved.Form.(resolution.DistinctForm); isDistinct {
		if distinctForm.Base.Name == "Array" && len(distinctForm.Base.TypeArgs) > 0 {
			return distinctForm.Base.TypeArgs[0], true
		}
	}

	return resolution.TypeRef{}, false
}

func (p *Plugin) processField(
	field resolution.Field,
	parent resolution.Type,
	data *templateData,
) fieldData {
	cppType := p.typeRefToCpp(field.Type, data)
	jsonName := toSnakeCase(field.Name)

	cppFieldName := domain.GetFieldName(field, "cpp")
	if cppFieldName == field.Name {
		cppFieldName = toSnakeCase(field.Name)
	}
	cppFieldName = keywords.Escape(cppFieldName)

	isGenericField := field.Type.IsTypeParam() && field.Type.TypeParam != nil &&
		!field.Type.TypeParam.HasDefault()
	typeParamName := ""
	if isGenericField {
		typeParamName = field.Type.TypeParam.Name
	}

	isSelfRef := field.Optional && isSelfReference(field.Type, parent, data.table)

	parseExpr := p.parseExprForField(field, cppType, data, isSelfRef)
	toJSONExpr := p.toJSONExprForField(field, data, isSelfRef)

	var jsonParseExpr, structParseExpr string
	if isGenericField {
		jsonParseExpr, structParseExpr = p.genericParseExprsForField(field)
	}

	return fieldData{
		Name:            cppFieldName,
		CppType:         cppType,
		JSONName:        jsonName,
		ParseExpr:       parseExpr,
		ToJSONExpr:      toJSONExpr,
		IsGenericField:  isGenericField,
		TypeParamName:   typeParamName,
		IsOptional:      field.Optional,
		JSONParseExpr:   jsonParseExpr,
		StructParseExpr: structParseExpr,
	}
}

func (p *Plugin) typeRefToCpp(typeRef resolution.TypeRef, data *templateData) string {
	if typeRef.TypeParam != nil {
		if typeRef.TypeParam.HasDefault() {
			return p.typeRefToCpp(*typeRef.TypeParam.Default, data)
		}
		return typeRef.TypeParam.Name
	}

	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		innerType := p.typeRefToCpp(typeRef.TypeArgs[0], data)
		data.includes.addSystem("vector")
		return fmt.Sprintf("std::vector<%s>", innerType)
	}

	if typeRef.Name == "Map" {
		data.includes.addSystem("unordered_map")
		keyType := "std::string"
		valueType := "void"
		if len(typeRef.TypeArgs) > 0 {
			keyType = p.typeRefToCpp(typeRef.TypeArgs[0], data)
		}
		if len(typeRef.TypeArgs) > 1 {
			valueType = p.typeRefToCpp(typeRef.TypeArgs[1], data)
		}
		return fmt.Sprintf("std::unordered_map<%s, %s>", keyType, valueType)
	}

	if mapping := primitiveMapper.Map(
		typeRef.Name,
	); mapping.TargetType != "" &&
		mapping.TargetType != "void" {
		for _, imp := range mapping.Imports {
			if imp.Category == "system" {
				data.includes.addSystem(imp.Path)
			} else {
				data.includes.addInternal(imp.Path)
			}
		}
		return mapping.TargetType
	}

	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return typeRef.Name
	}

	if enumForm, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
		if !enumForm.IsIntEnum {
			data.includes.addSystem("string")
			return "std::string"
		}
		return resolved.Name
	}

	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		if targetResolved, targetOk := aliasForm.Target.Resolve(data.table); targetOk {
			if _, isStruct := targetResolved.Form.(resolution.StructForm); isStruct {
				if targetResolved.Namespace != data.rawNs {
					targetOutputPath := output.GetPath(targetResolved, "cpp")
					if targetOutputPath != "" {
						data.includes.addInternal(data.jsonInclude(targetOutputPath))
					}
				}
			}
		}
	}

	name := domain.GetName(resolved, "cpp")

	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			var includePath string
			if p.isFixedSizeUint8ArrayType(resolved) {
				headerName := lo.SnakeCase(resolved.Name)
				includePath = fmt.Sprintf("%s/%s.h", targetOutputPath, headerName)
			} else {
				includePath = data.jsonInclude(targetOutputPath)
			}
			data.includes.addInternal(includePath)
			ns := deriveNamespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	}

	if len(typeRef.TypeArgs) > 0 {
		args := make([]string, 0, len(typeRef.TypeArgs))
		for _, arg := range typeRef.TypeArgs {
			args = append(args, p.typeRefToCpp(arg, data))
		}
		name = fmt.Sprintf("%s<%s>", name, strings.Join(args, ", "))
	}

	return name
}

func (p *Plugin) parseExprForField(
	field resolution.Field,
	cppType string,
	data *templateData,
	isSelfRef bool,
) string {
	typeRef := field.Type
	jsonName := toSnakeCase(field.Name)
	hasDefault := field.Optional || hasRenderableDefault(field, data.table)

	if typeRef.TypeParam != nil && !typeRef.TypeParam.HasDefault() {
		if field.Optional {
			return fmt.Sprintf(
				`parser.field<std::optional<%s>>("%s")`,
				typeRef.TypeParam.Name,
				jsonName,
			)
		}
		return fmt.Sprintf(`parser.field<%s>("%s")`, typeRef.TypeParam.Name, jsonName)
	}

	if resolved, ok := typeRef.Resolve(data.table); ok {
		if distinctForm, isDistinct := resolved.Form.(resolution.DistinctForm); isDistinct {
			if distinctForm.Base.Name == "Array" &&
				len(distinctForm.Base.TypeArgs) > 0 {
				wrapperType := domain.GetName(resolved, "cpp")
				if resolved.Namespace != data.rawNs {
					targetOutputPath := output.GetPath(resolved, "cpp")
					if targetOutputPath != "" {
						ns := deriveNamespace(targetOutputPath)
						wrapperType = fmt.Sprintf("::%s::%s", ns, wrapperType)
					}
				}
				return fmt.Sprintf(`parser.field<%s>("%s")`, wrapperType, jsonName)
			}
		}
	}

	if elemType, isArray := p.resolveToArrayElement(typeRef, data); isArray {
		innerType := p.typeRefToCpp(elemType, data)

		if elemType.TypeParam != nil {
			if field.Optional {
				return fmt.Sprintf(
					`parser.field<std::optional<std::vector<%s>>>("%s")`,
					elemType.TypeParam.Name,
					jsonName,
				)
			}
			return fmt.Sprintf(
				`parser.field<std::vector<%s>>("%s")`,
				elemType.TypeParam.Name,
				jsonName,
			)
		}

		if elemResolved, ok := elemType.Resolve(data.table); ok {
			if structForm, isStruct := elemResolved.Form.(resolution.StructForm); isStruct {
				structType := domain.GetName(elemResolved, "cpp")
				if elemResolved.Namespace != data.rawNs {
					targetOutputPath := output.GetPath(elemResolved, "cpp")
					if targetOutputPath != "" {
						ns := deriveNamespace(targetOutputPath)
						structType = fmt.Sprintf("::%s::%s", ns, structType)
					}
				}
				if len(elemType.TypeArgs) > 0 {
					var args []string
					for i, arg := range elemType.TypeArgs {
						if i < len(structForm.TypeParams) &&
							structForm.TypeParams[i].HasDefault() {
							continue
						}
						args = append(args, p.typeRefToCpp(arg, data))
					}
					if len(args) > 0 {
						structType = fmt.Sprintf(
							"%s<%s>",
							structType,
							strings.Join(args, ", "),
						)
					}
				}
				if field.Optional {
					return fmt.Sprintf(
						`parser.field<std::optional<std::vector<%s>>>("%s")`,
						structType,
						jsonName,
					)
				}
				return fmt.Sprintf(
					`parser.field<std::vector<%s>>("%s")`,
					structType,
					jsonName,
				)
			}
			if _, isUnion := elemResolved.Form.(resolution.UnionForm); isUnion {
				parseFn := "parse_" + lo.SnakeCase(domain.GetName(elemResolved, "cpp"))
				if elemResolved.Namespace != data.rawNs {
					targetOutputPath := output.GetPath(elemResolved, "cpp")
					if targetOutputPath != "" {
						ns := deriveNamespace(targetOutputPath)
						parseFn = fmt.Sprintf("::%s::%s", ns, parseFn)
						data.includes.addInternal(data.jsonInclude(targetOutputPath))
					}
				}
				return fmt.Sprintf(
					`[&] {
        std::vector<%s> result;
        parser.iter("%s", [&result](x::json::Parser& p) { result.push_back(%s(p)); });
        return result;
    }()`,
					innerType, jsonName, parseFn,
				)
			}
		}

		if field.Optional {
			return fmt.Sprintf(
				`parser.field<std::optional<std::vector<%s>>>("%s")`,
				innerType,
				jsonName,
			)
		}
		return fmt.Sprintf(`parser.field<std::vector<%s>>("%s")`, innerType, jsonName)
	}

	resolved, resolvedOk := typeRef.Resolve(data.table)
	if resolvedOk {
		if enumForm, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
			if !enumForm.IsIntEnum {
				if field.Optional {
					return fmt.Sprintf(
						`parser.field<std::optional<std::string>>("%s")`,
						jsonName,
					)
				}
				if hasDefault {
					defaultVal := jsonDefaultLiteral(field, data.table)
					if defaultVal == "" {
						defaultVal = `""`
					}
					return fmt.Sprintf(
						`parser.field<std::string>("%s", %s)`,
						jsonName,
						defaultVal,
					)
				}
				return fmt.Sprintf(`parser.field<std::string>("%s")`, jsonName)
			}
			enumType := domain.GetName(resolved, "cpp")
			if resolved.Namespace != data.rawNs {
				targetOutputPath := output.GetPath(resolved, "cpp")
				if targetOutputPath != "" {
					ns := deriveNamespace(targetOutputPath)
					enumType = fmt.Sprintf("::%s::%s", ns, enumType)
				}
			}
			if field.Optional {
				return fmt.Sprintf(
					`parser.field<std::optional<%s>>("%s")`,
					enumType,
					jsonName,
				)
			}
			return fmt.Sprintf(`parser.field<%s>("%s")`, enumType, jsonName)
		}
		if structForm, isStruct := resolved.Form.(resolution.StructForm); isStruct {
			structType := domain.GetName(resolved, "cpp")
			if resolved.Namespace != data.rawNs {
				targetOutputPath := output.GetPath(resolved, "cpp")
				if targetOutputPath != "" {
					ns := deriveNamespace(targetOutputPath)
					structType = fmt.Sprintf("::%s::%s", ns, structType)
				}
			}
			if len(typeRef.TypeArgs) > 0 {
				var args []string
				for i, arg := range typeRef.TypeArgs {
					if i < len(structForm.TypeParams) &&
						structForm.TypeParams[i].HasDefault() {
						continue
					}
					args = append(args, p.typeRefToCpp(arg, data))
				}
				if len(args) > 0 {
					structType = fmt.Sprintf(
						"%s<%s>",
						structType,
						strings.Join(args, ", "),
					)
				}
			}
			if field.Optional {
				if isSelfRef {
					return fmt.Sprintf(
						`parser.field<x::mem::indirect<%s>>("%s")`,
						structType,
						jsonName,
					)
				}
				return fmt.Sprintf(
					`parser.field<std::optional<%s>>("%s")`,
					structType,
					jsonName,
				)
			}
			return fmt.Sprintf(`parser.field<%s>("%s")`, structType, jsonName)
		}
		if _, isUnion := resolved.Form.(resolution.UnionForm); isUnion {
			unionType := domain.GetName(resolved, "cpp")
			parseFn := "parse_" + lo.SnakeCase(unionType)
			if resolved.Namespace != data.rawNs {
				targetOutputPath := output.GetPath(resolved, "cpp")
				if targetOutputPath != "" {
					ns := deriveNamespace(targetOutputPath)
					parseFn = fmt.Sprintf("::%s::%s", ns, parseFn)
					data.includes.addInternal(data.jsonInclude(targetOutputPath))
				}
			}
			if field.Optional {
				return fmt.Sprintf(
					`parser.has("%s") ? std::optional<%s>(%s(parser.child("%s"))) : std::nullopt`,
					jsonName,
					cppType,
					parseFn,
					jsonName,
				)
			}
			if hasDefault {
				return fmt.Sprintf(
					`parser.has("%s") ? %s(parser.child("%s")) : %s{}`,
					jsonName, parseFn, jsonName, cppType,
				)
			}
			return fmt.Sprintf(`%s(parser.child("%s"))`, parseFn, jsonName)
		}
		if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
			if targetResolved, targetOk := aliasForm.Target.Resolve(
				data.table,
			); targetOk {
				if _, isStruct := targetResolved.Form.(resolution.StructForm); isStruct {
					aliasType := domain.GetName(resolved, "cpp")
					if resolved.Namespace != data.rawNs {
						targetOutputPath := output.GetPath(resolved, "cpp")
						if targetOutputPath != "" {
							ns := deriveNamespace(targetOutputPath)
							aliasType = fmt.Sprintf("::%s::%s", ns, aliasType)
						}
					}
					if field.Optional {
						return fmt.Sprintf(
							`parser.field<std::optional<%s>>("%s")`,
							aliasType,
							jsonName,
						)
					}
					return fmt.Sprintf(`parser.field<%s>("%s")`, aliasType, jsonName)
				}
			}
		}
	}

	if mapping := primitiveMapper.Map(
		typeRef.Name,
	); mapping.TargetType != "" &&
		mapping.TargetType != "void" {
		if field.Optional {
			return fmt.Sprintf(
				`parser.field<std::optional<%s>>("%s")`,
				cppType,
				jsonName,
			)
		}
		if hasDefault {
			defaultVal := jsonDefaultLiteral(field, data.table)
			if defaultVal == "" && field.Optional {
				defaultVal = defaultValueForPrimitive(typeRef.Name)
			}
			if defaultVal != "" {
				return fmt.Sprintf(
					`parser.field<%s>("%s", %s)`,
					cppType,
					jsonName,
					defaultVal,
				)
			}
		}
		return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
	}

	if field.Optional {
		if isSelfRef {
			return fmt.Sprintf(
				`parser.field<x::mem::indirect<%s>>("%s")`,
				cppType,
				jsonName,
			)
		}
		return fmt.Sprintf(`parser.field<std::optional<%s>>("%s")`, cppType, jsonName)
	}
	if hasDefault {
		if defaultVal := jsonDefaultLiteral(field, data.table); defaultVal != "" {
			// Telem time types have explicit integer constructors, so bare
			// numeric defaults must be wrapped to bind as the fallback value.
			if field.Default != nil &&
				(field.Default.Kind == resolution.ValueKindInt ||
					field.Default.Kind == resolution.ValueKindFloat) {
				if strings.Contains(cppType, "::telem::TimeStamp") {
					defaultVal = fmt.Sprintf("x::telem::TimeStamp(%s)", defaultVal)
				} else if strings.Contains(cppType, "::telem::TimeSpan") {
					defaultVal = fmt.Sprintf("x::telem::TimeSpan(%s)", defaultVal)
				} else if strings.Contains(cppType, "::telem::Rate") {
					defaultVal = fmt.Sprintf("x::telem::Rate(%s)", defaultVal)
				}
			}
			return fmt.Sprintf(
				`parser.field<%s>("%s", %s)`,
				cppType,
				jsonName,
				defaultVal,
			)
		}
	}
	return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
}

func (*Plugin) genericParseExprsForField(
	field resolution.Field,
) (jsonParseExpr, structParseExpr string) {
	jsonName := toSnakeCase(field.Name)
	typeParamName := field.Type.TypeParam.Name

	if field.Optional {
		jsonParseExpr = fmt.Sprintf(
			`parser.field<std::optional<x::json::json::object_t>>("%s")`,
			jsonName,
		)
		structParseExpr = fmt.Sprintf(
			`parser.field<std::optional<%s>>("%s")`,
			typeParamName,
			jsonName,
		)
	} else {
		jsonParseExpr = fmt.Sprintf(
			`parser.field<x::json::json::object_t>("%s")`,
			jsonName,
		)
		structParseExpr = fmt.Sprintf(`parser.field<%s>("%s")`, typeParamName, jsonName)
	}

	return jsonParseExpr, structParseExpr
}

func (p *Plugin) toJSONExprForField(
	field resolution.Field,
	data *templateData,
	isSelfRef bool,
) string {
	typeRef := field.Type
	jsonName := toSnakeCase(field.Name)

	fieldName := domain.GetFieldName(field, "cpp")
	if fieldName == field.Name {
		fieldName = toSnakeCase(field.Name)
	}
	fieldName = keywords.Escape(fieldName)

	if typeRef.TypeParam != nil && !typeRef.TypeParam.HasDefault() {
		typeName := typeRef.TypeParam.Name
		return fmt.Sprintf(`if constexpr (std::is_same_v<%s, x::json::json::object_t>)
        j["%s"] = this->%s;
    else if constexpr (std::is_same_v<%s, std::monostate>)
        j["%s"] = nullptr;
    else
        j["%s"] = this->%s.to_json();`, typeName, jsonName, fieldName, typeName, jsonName, jsonName, fieldName)
	}

	// Self-referential optional fields are wrapped as x::mem::indirect<T>
	// by the types plugin. indirect<T> has the same has_value() + -> interface
	// as std::optional<T>, and the underlying T (struct, or a distinct/alias
	// resolving to one) always has to_json(). Emit the unwrap pattern here so
	// the cycle-through-distinct case doesn't fall through to the default
	// assignment below, which would be ill-typed against indirect<T>.
	if isSelfRef {
		return fmt.Sprintf(
			`if (this->%s.has_value()) j["%s"] = this->%s->to_json();`,
			fieldName,
			jsonName,
			fieldName,
		)
	}

	if resolved, ok := typeRef.Resolve(data.table); ok {
		if distinctForm, isDistinct := resolved.Form.(resolution.DistinctForm); isDistinct {
			if distinctForm.Base.Name == "Array" &&
				len(distinctForm.Base.TypeArgs) > 0 {
				return fmt.Sprintf(`j["%s"] = this->%s.to_json();`, jsonName, fieldName)
			}
		}
	}

	if elemType, isArray := p.resolveToArrayElement(typeRef, data); isArray {
		if elemType.TypeParam != nil {
			typeName := elemType.TypeParam.Name
			return fmt.Sprintf(`{
        auto arr = x::json::json::array();
        for (const auto& item : this->%s)
            if constexpr (std::is_same_v<%s, x::json::json::object_t>)
                arr.push_back(item);
            else if constexpr (std::is_same_v<%s, std::monostate>)
                arr.push_back(nullptr);
            else
                arr.push_back(item.to_json());
        j["%s"] = arr;
    }`, fieldName, typeName, typeName, jsonName)
		}

		if elemResolved, ok := elemType.Resolve(data.table); ok {
			if _, isStruct := elemResolved.Form.(resolution.StructForm); isStruct {
				if field.Optional {
					return fmt.Sprintf(
						`if (this->%s.has_value()) j["%s"] = x::json::to_array(*this->%s);`,
						fieldName,
						jsonName,
						fieldName,
					)
				}
				return fmt.Sprintf(
					`j["%s"] = x::json::to_array(this->%s);`,
					jsonName,
					fieldName,
				)
			}
			if _, isUnion := elemResolved.Form.(resolution.UnionForm); isUnion {
				// Qualify the free to_json: unqualified lookup inside a member
				// to_json() finds the member and never reaches the overload.
				qualifier := "::" + data.Namespace
				if elemResolved.Namespace != data.rawNs {
					targetOutputPath := output.GetPath(elemResolved, "cpp")
					if targetOutputPath != "" {
						data.includes.addInternal(data.jsonInclude(targetOutputPath))
						qualifier = "::" + deriveNamespace(targetOutputPath)
					}
				}
				return fmt.Sprintf(`{
        auto arr = x::json::json::array();
        for (const auto& item : this->%s) arr.push_back(%s::to_json(item));
        j["%s"] = arr;
    }`, fieldName, qualifier, jsonName)
			}
			// Nested-array-of-struct case: outer element resolves to another
			// array (e.g., Members = []Member) whose inner element is a
			// struct. nlohmann_json can't serialize vector<vector<Struct>>
			// directly — serialize each inner array via to_array and bundle
			// them in a JSON array.
			if innerElem, ok := p.resolveToArrayElement(elemType, data); ok {
				if innerResolved, ok := innerElem.Resolve(data.table); ok {
					if _, isStruct := innerResolved.Form.(resolution.StructForm); isStruct {
						return fmt.Sprintf(`{
        auto arr = x::json::json::array();
        for (const auto& inner : this->%s) arr.push_back(x::json::to_array(inner));
        j["%s"] = arr;
    }`, fieldName, jsonName)
					}
				}
			}
		}

		if field.Optional {
			return fmt.Sprintf(
				`if (this->%s.has_value()) j["%s"] = *this->%s;`,
				fieldName,
				jsonName,
				fieldName,
			)
		}
		return fmt.Sprintf(`j["%s"] = this->%s;`, jsonName, fieldName)
	}

	resolved, resolvedOk := typeRef.Resolve(data.table)
	if resolvedOk {
		if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
			if isSelfRef {
				return fmt.Sprintf(
					`if (this->%s.has_value()) j["%s"] = this->%s->to_json();`,
					fieldName,
					jsonName,
					fieldName,
				)
			}
			if field.Optional {
				return fmt.Sprintf(
					`if (this->%s.has_value()) j["%s"] = this->%s->to_json();`,
					fieldName,
					jsonName,
					fieldName,
				)
			}
			return fmt.Sprintf(`j["%s"] = this->%s.to_json();`, jsonName, fieldName)
		}
		if _, isUnion := resolved.Form.(resolution.UnionForm); isUnion {
			// Qualify the free to_json: unqualified lookup inside a member
			// to_json() finds the member and never reaches the overload.
			qualifier := "::" + data.Namespace
			if resolved.Namespace != data.rawNs {
				targetOutputPath := output.GetPath(resolved, "cpp")
				if targetOutputPath != "" {
					data.includes.addInternal(data.jsonInclude(targetOutputPath))
					qualifier = "::" + deriveNamespace(targetOutputPath)
				}
			}
			if field.Optional {
				return fmt.Sprintf(
					`if (this->%s.has_value()) j["%s"] = %s::to_json(*this->%s);`,
					fieldName,
					jsonName,
					qualifier,
					fieldName,
				)
			}
			return fmt.Sprintf(
				`j["%s"] = %s::to_json(this->%s);`,
				jsonName,
				qualifier,
				fieldName,
			)
		}
		if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
			if targetResolved, targetOk := aliasForm.Target.Resolve(
				data.table,
			); targetOk {
				if _, isStruct := targetResolved.Form.(resolution.StructForm); isStruct {
					if field.Optional {
						return fmt.Sprintf(
							`if (this->%s.has_value()) j["%s"] = this->%s->to_json();`,
							fieldName,
							jsonName,
							fieldName,
						)
					}
					return fmt.Sprintf(
						`j["%s"] = this->%s.to_json();`,
						jsonName,
						fieldName,
					)
				}
			}
		}
	}

	lowerName := strings.ToLower(typeRef.Name)
	if strings.HasSuffix(lowerName, "timestamp") ||
		strings.HasSuffix(lowerName, "timespan") {
		return fmt.Sprintf(`j["%s"] = this->%s.nanoseconds();`, jsonName, fieldName)
	}

	if typeRef.Name == "uuid" || p.resolvesToUUID(typeRef, data) {
		if field.Optional {
			return fmt.Sprintf(
				`if (this->%s.has_value()) j["%s"] = this->%s->to_json();`,
				fieldName,
				jsonName,
				fieldName,
			)
		}
		return fmt.Sprintf(`j["%s"] = this->%s.to_json();`, jsonName, fieldName)
	}

	return fmt.Sprintf(`j["%s"] = this->%s;`, jsonName, fieldName)
}

// hasRenderableDefault reports whether the field declares a default the parse
// expression can honor. Struct and array defaults count (their branches render
// them); identifier defaults count only when they resolve to an enum variant or
// boolean literal, so sentinels like create do not relax a required field.
func hasRenderableDefault(field resolution.Field, table *resolution.Table) bool {
	if field.Default == nil {
		return false
	}
	if field.Default.Kind != resolution.ValueKindIdent {
		return true
	}
	return jsonDefaultLiteral(field, table) != ""
}

// jsonDefaultLiteral renders a field's schema default as a C++ literal usable as
// parser.field's fallback argument. Returns "" when the default has no inline
// scalar rendering (arrays, non-empty structs), leaving the caller's behavior
// unchanged.
func jsonDefaultLiteral(field resolution.Field, table *resolution.Table) string {
	if field.Default == nil {
		return ""
	}
	v := *field.Default
	switch v.Kind {
	case resolution.ValueKindString:
		return fmt.Sprintf("%q", v.StringValue)
	case resolution.ValueKindInt:
		return fmt.Sprintf("%d", v.IntValue)
	case resolution.ValueKindFloat:
		return fmt.Sprintf("%f", v.FloatValue)
	case resolution.ValueKindBool:
		return fmt.Sprintf("%t", v.BoolValue)
	case resolution.ValueKindStruct:
		if len(v.Fields) == 0 && field.Type.Name == "record" {
			return "x::json::json::object_t{}"
		}
		return ""
	case resolution.ValueKindIdent:
		if ev, ok := validation.ResolveEnumVariant(
			v.IdentValue,
			field.Type,
			table,
		); ok {
			return fmt.Sprintf("%q", ev.Variant.StringValue())
		}
		if v.IdentValue == "true" || v.IdentValue == "false" {
			return v.IdentValue
		}
		// Unresolvable idents (magic defaults like create/now) have no C++
		// rendering; the field stays required.
		return ""
	}
	return ""
}

func defaultValueForPrimitive(primitive string) string {
	switch primitive {
	case "string":
		return `""`
	case "uuid":
		return "x::uuid::UUID{}"
	case "bool":
		return "false"
	case "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64":
		return "0"
	case "float32", "float64":
		return "0.0"
	case "timestamp":
		return "x::telem::TimeStamp(0)"
	case "timespan":
		return "x::telem::TimeSpan(0)"
	case "time_range", "time_range_bounded":
		return "x::telem::TimeRange{}"
	case "record":
		return "x::json::json::object_t{}"
	case "bytes":
		return "{}"
	default:
		return "{}"
	}
}

func toSnakeCase(s string) string {
	return lo.SnakeCase(s)
}

func deriveNamespace(outputPath string) string {
	parts := strings.Split(outputPath, "/")
	if len(parts) == 0 {
		return "synnax"
	}

	var topLevel string
	switch {
	case len(parts) >= 2 && parts[0] == "x" && parts[1] == "cpp":
		topLevel = "x"
	case len(parts) >= 2 && parts[0] == "client" && parts[1] == "cpp":
		topLevel = "synnax"
	case len(parts) >= 2 && parts[0] == "arc" && parts[1] == "cpp":
		topLevel = "arc"
	case len(parts) >= 1 && parts[0] == "driver":
		topLevel = "driver"
	default:
		topLevel = "synnax"
	}

	subNs := parts[len(parts)-1]
	return fmt.Sprintf("%s::%s", topLevel, subNs)
}

type includeManager struct {
	system   []string
	internal []string
}

func newIncludeManager() *includeManager {
	return &includeManager{
		system:   make([]string, 0),
		internal: make([]string, 0),
	}
}

func (m *includeManager) addSystem(name string) {
	if !lo.Contains(m.system, name) {
		m.system = append(m.system, name)
	}
}

func (m *includeManager) addInternal(path string) {
	if !lo.Contains(m.internal, path) {
		m.internal = append(m.internal, path)
	}
}

func (p *Plugin) resolvesToUUID(typeRef resolution.TypeRef, data *templateData) bool {
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return false
	}

	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		if aliasForm.Target.Name == "uuid" {
			return true
		}
		return p.resolvesToUUID(aliasForm.Target, data)
	}

	if distinctForm, isDistinct := resolved.Form.(resolution.DistinctForm); isDistinct {
		if distinctForm.Base.Name == "uuid" {
			return true
		}
	}

	return false
}

func (*Plugin) isFixedSizeUint8ArrayType(resolved resolution.Type) bool {
	form, ok := resolved.Form.(resolution.DistinctForm)
	if !ok {
		return false
	}
	if form.Base.Name != "Array" || form.Base.ArraySize == nil {
		return false
	}
	if len(form.Base.TypeArgs) == 0 {
		return false
	}
	elemType := form.Base.TypeArgs[0]
	return resolution.IsPrimitive(elemType.Name) && elemType.Name == "uint8"
}

type templateData struct {
	includes      *includeManager
	table         *resolution.Table
	jsonPaths     set.Set[string]
	OutputPath    string
	Namespace     string
	rawNs         string
	Serializers   []serializerData
	ArrayWrappers []arrayWrapperData
	Unions        []unionDispatchData
}

// jsonInclude returns the header to include for a cross-namespace reference into
// outputPath.
func (d *templateData) jsonInclude(outputPath string) string {
	return IncludeFor(d.jsonPaths, outputPath)
}

type arrayWrapperData struct {
	Name        string
	ElementType string
	// ElementNeedsConvert is true if element is a struct that needs to_json()/parse().
	ElementNeedsConvert bool
}

func (d *templateData) HasIncludes() bool {
	return len(d.includes.system) > 0 || len(d.includes.internal) > 0
}

func (d *templateData) SystemIncludes() []string {
	return d.includes.system
}

func (d *templateData) InternalIncludes() []string {
	return d.includes.internal
}

type serializerData struct {
	Name           string
	TypeParamNames string
	TypeParams     []typeParamData
	Fields         []fieldData
	ParentTypes    []parentTypeData
	HasExtends     bool
	IsGeneric      bool
}

type parentTypeData struct {
	// QualifiedName is the fully qualified C++ name (e.g., "arc::ir::IR").
	QualifiedName string
}

type typeParamData struct {
	Name string
}

type fieldData struct {
	Name            string
	CppType         string
	JSONName        string
	ParseExpr       string
	ToJSONExpr      string
	TypeParamName   string
	JSONParseExpr   string
	StructParseExpr string
	IsGenericField  bool
	IsOptional      bool
}
