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
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	cppprimitives "github.com/synnaxlabs/oracle/plugin/cpp/primitives"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

var primitiveMapper = cppprimitives.Mapper()

type Plugin struct{ Options Options }

type Options struct {
	FileNamePattern  string
	DisableFormatter bool
}

func DefaultOptions() Options {
	return Options{
		FileNamePattern: "json.gen.h",
	}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string { return "cpp/json" }

func (p *Plugin) Domains() []string { return []string{"cpp"} }

func (p *Plugin) Requires() []string { return []string{"cpp/types"} }

func (p *Plugin) Check(*plugin.Request) error { return nil }

var cppPostWriter = &exec.PostWriter{
	Extensions: []string{".h", ".hpp", ".cpp", ".cc"},
	Commands:   [][]string{{"clang-format", "-i"}},
}

func (p *Plugin) PostWrite(files []string) error {
	if p.Options.DisableFormatter {
		return nil
	}
	return cppPostWriter.PostWrite(files)
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	structCollector, err := framework.CollectStructs("cpp", req)
	if err != nil {
		return nil, err
	}

	distinctCollector, err := framework.CollectDistinct("cpp", req)
	if err != nil {
		return nil, err
	}

	allPaths := make(map[string]bool)
	for _, path := range structCollector.Paths() {
		allPaths[path] = true
	}
	for _, path := range distinctCollector.Paths() {
		allPaths[path] = true
	}

	for outputPath := range allPaths {
		structs := structCollector.Get(outputPath)
		distinctTypes := distinctCollector.Get(outputPath)

		var namespace string
		if len(structs) > 0 {
			namespace = structs[0].Namespace
		} else if len(distinctTypes) > 0 {
			namespace = distinctTypes[0].Namespace
		} else {
			continue
		}

		content, err := p.generateFile(outputPath, structs, distinctTypes, namespace, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate json for %s", outputPath)
		}
		if len(content) > 0 {
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
				Content: content,
			})
		}
	}

	return resp, nil
}

func (p *Plugin) generateFile(
	outputPath string,
	structs []resolution.Type,
	distinctTypes []resolution.Type,
	namespace string,
	req *plugin.Request,
) ([]byte, error) {
	data := &templateData{
		OutputPath:    outputPath,
		Namespace:     deriveNamespace(outputPath),
		Serializers:   make([]serializerData, 0, len(structs)),
		ArrayWrappers: make([]arrayWrapperData, 0),
		includes:      newIncludeManager(),
		table:         req.Resolutions,
		rawNs:         namespace,
	}

	data.includes.addInternal(fmt.Sprintf("%s/types.gen.h", outputPath))
	data.includes.addInternal("x/cpp/json/json.h")

	for _, s := range structs {
		if omit.IsType(s, "cpp") {
			continue
		}
		serializer, err := p.processStruct(s, data, req)
		if err != nil {
			return nil, err
		}
		if serializer != nil {
			data.Serializers = append(data.Serializers, *serializer)
		}
	}

	for _, dt := range distinctTypes {
		if omit.IsType(dt, "cpp") {
			continue
		}
		wrapper := p.processArrayWrapper(dt, data)
		if wrapper != nil {
			data.ArrayWrappers = append(data.ArrayWrappers, *wrapper)
		}
	}

	if len(data.Serializers) == 0 && len(data.ArrayWrappers) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	if err := jsonTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Plugin) processArrayWrapper(dt resolution.Type, data *templateData) *arrayWrapperData {
	form, ok := dt.Form.(resolution.DistinctForm)
	if !ok {
		return nil
	}

	if form.Base.Name != "Array" || len(form.Base.TypeArgs) == 0 {
		return nil
	}

	if form.Base.ArraySize != nil {
		return nil
	}

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

func (p *Plugin) resolveExtendsType(extendsRef resolution.TypeRef, parent resolution.Type, data *templateData) string {
	name := domain.GetName(parent, "cpp")

	if parent.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(parent, "cpp")
		if targetOutputPath != "" {
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "json.gen.h")
			data.includes.addInternal(includePath)
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
	req *plugin.Request,
) (*serializerData, error) {
	form, ok := s.Form.(resolution.StructForm)
	if !ok {
		return nil, nil
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

	return serializer, nil
}

func isSelfReference(t resolution.TypeRef, parent resolution.Type) bool {
	if t.Name == parent.QualifiedName {
		return true
	}
	for _, arg := range t.TypeArgs {
		if isSelfReference(arg, parent) {
			return true
		}
	}
	return false
}

func (p *Plugin) resolveToArrayElement(typeRef resolution.TypeRef, data *templateData) (resolution.TypeRef, bool) {
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

func (p *Plugin) processField(field resolution.Field, parent resolution.Type, data *templateData) fieldData {
	cppType := p.typeRefToCpp(field.Type, data)
	jsonName := toSnakeCase(field.Name)

	cppFieldName := domain.GetFieldName(field, "cpp")
	if cppFieldName == field.Name {
		cppFieldName = toSnakeCase(field.Name)
	}

	isGenericField := field.Type.IsTypeParam() && field.Type.TypeParam != nil && !field.Type.TypeParam.HasDefault()
	typeParamName := ""
	if isGenericField {
		typeParamName = field.Type.TypeParam.Name
	}

	isSelfRef := field.IsHardOptional && isSelfReference(field.Type, parent)

	parseExpr := p.parseExprForField(field, parent, cppType, data, isSelfRef)
	toJSONExpr := p.toJSONExprForField(field, parent, data, isSelfRef)

	var jsonParseExpr, structParseExpr string
	if isGenericField {
		jsonParseExpr, structParseExpr = p.genericParseExprsForField(field, data)
	}

	return fieldData{
		Name:            cppFieldName,
		CppType:         cppType,
		JSONName:        jsonName,
		ParseExpr:       parseExpr,
		ToJSONExpr:      toJSONExpr,
		IsGenericField:  isGenericField,
		TypeParamName:   typeParamName,
		IsHardOptional:  field.IsHardOptional,
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

	if mapping := primitiveMapper.Map(typeRef.Name); mapping.TargetType != "" && mapping.TargetType != "void" {
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
						data.includes.addInternal(fmt.Sprintf("%s/json.gen.h", targetOutputPath))
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
				includePath = fmt.Sprintf("%s/json.gen.h", targetOutputPath)
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

func (p *Plugin) parseExprForField(field resolution.Field, parent resolution.Type, cppType string, data *templateData, isSelfRef bool) string {
	typeRef := field.Type
	jsonName := toSnakeCase(field.Name)
	hasDefault := field.IsOptional

	if typeRef.TypeParam != nil && !typeRef.TypeParam.HasDefault() {
		innerType := typeRef.TypeParam.Name
		if field.IsHardOptional {
			innerExpr := p.parseExprForTypeRef(typeRef, innerType, jsonName, false, data)
			return fmt.Sprintf(`parser.has("%s") ? std::make_optional(%s) : std::nullopt`, jsonName, innerExpr)
		}
		return fmt.Sprintf(`parser.field<%s>("%s")`, typeRef.TypeParam.Name, jsonName)
	}

	if resolved, ok := typeRef.Resolve(data.table); ok {
		if distinctForm, isDistinct := resolved.Form.(resolution.DistinctForm); isDistinct {
			if distinctForm.Base.Name == "Array" && len(distinctForm.Base.TypeArgs) > 0 {
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
			return fmt.Sprintf(`parser.field<std::vector<%s>>("%s")`, elemType.TypeParam.Name, jsonName)
		}

		if elemResolved, ok := elemType.Resolve(data.table); ok {
			if _, isStruct := elemResolved.Form.(resolution.StructForm); isStruct {
				structType := domain.GetName(elemResolved, "cpp")
				if elemResolved.Namespace != data.rawNs {
					targetOutputPath := output.GetPath(elemResolved, "cpp")
					if targetOutputPath != "" {
						ns := deriveNamespace(targetOutputPath)
						structType = fmt.Sprintf("::%s::%s", ns, structType)
					}
				}
				return fmt.Sprintf(`parser.field<std::vector<%s>>("%s")`, structType, jsonName)
			}
		}

		return fmt.Sprintf(`parser.field<std::vector<%s>>("%s")`, innerType, jsonName)
	}

	resolved, resolvedOk := typeRef.Resolve(data.table)
	if resolvedOk {
		if enumForm, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
			if !enumForm.IsIntEnum {
				if field.IsHardOptional {
					return fmt.Sprintf(`parser.has("%s") ? std::make_optional(parser.field<std::string>("%s")) : std::nullopt`, jsonName, jsonName)
				}
				if hasDefault {
					return fmt.Sprintf(`parser.field<std::string>("%s", "")`, jsonName)
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
			if field.IsHardOptional {
				return fmt.Sprintf(`parser.has("%s") ? std::make_optional(parser.field<%s>("%s")) : std::nullopt`, jsonName, enumType, jsonName)
			}
			return fmt.Sprintf(`parser.field<%s>("%s")`, enumType, jsonName)
		}
		if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
			structType := domain.GetName(resolved, "cpp")
			if resolved.Namespace != data.rawNs {
				targetOutputPath := output.GetPath(resolved, "cpp")
				if targetOutputPath != "" {
					ns := deriveNamespace(targetOutputPath)
					structType = fmt.Sprintf("::%s::%s", ns, structType)
				}
			}
			if field.IsHardOptional {
				if isSelfRef {
					return fmt.Sprintf(`parser.has("%s") ? x::mem::indirect<%s>(parser.field<%s>("%s")) : nullptr`, jsonName, structType, structType, jsonName)
				}
				return fmt.Sprintf(`parser.has("%s") ? std::make_optional(parser.field<%s>("%s")) : std::nullopt`, jsonName, structType, jsonName)
			}
			return fmt.Sprintf(`parser.field<%s>("%s")`, structType, jsonName)
		}
		if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
			if targetResolved, targetOk := aliasForm.Target.Resolve(data.table); targetOk {
				if _, isStruct := targetResolved.Form.(resolution.StructForm); isStruct {
					aliasType := domain.GetName(resolved, "cpp")
					if resolved.Namespace != data.rawNs {
						targetOutputPath := output.GetPath(resolved, "cpp")
						if targetOutputPath != "" {
							ns := deriveNamespace(targetOutputPath)
							aliasType = fmt.Sprintf("::%s::%s", ns, aliasType)
						}
					}
					if field.IsHardOptional {
						return fmt.Sprintf(`parser.has("%s") ? std::make_optional(parser.field<%s>("%s")) : std::nullopt`, jsonName, aliasType, jsonName)
					}
					return fmt.Sprintf(`parser.field<%s>("%s")`, aliasType, jsonName)
				}
			}
		}
	}

	if mapping := primitiveMapper.Map(typeRef.Name); mapping.TargetType != "" && mapping.TargetType != "void" {
		if field.IsHardOptional {
			return fmt.Sprintf(`parser.has("%s") ? std::make_optional(parser.field<%s>("%s")) : std::nullopt`, jsonName, cppType, jsonName)
		}
		if hasDefault {
			defaultVal := defaultValueForPrimitive(typeRef.Name)
			return fmt.Sprintf(`parser.field<%s>("%s", %s)`, cppType, jsonName, defaultVal)
		}
		return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
	}

	if field.IsHardOptional {
		if isSelfRef {
			return fmt.Sprintf(`parser.has("%s") ? x::mem::indirect<%s>(parser.field<%s>("%s")) : nullptr`, jsonName, cppType, cppType, jsonName)
		}
		return fmt.Sprintf(`parser.has("%s") ? std::make_optional(parser.field<%s>("%s")) : std::nullopt`, jsonName, cppType, jsonName)
	}
	return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
}

func (p *Plugin) parseExprForTypeRef(typeRef resolution.TypeRef, cppType, jsonName string, hasDefault bool, data *templateData) string {
	if typeRef.TypeParam != nil {
		return fmt.Sprintf(`parser.field<%s>("%s")`, typeRef.TypeParam.Name, jsonName)
	}

	resolved, resolvedOk := typeRef.Resolve(data.table)
	if resolvedOk {
		if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
			structType := domain.GetName(resolved, "cpp")
			if resolved.Namespace != data.rawNs {
				targetOutputPath := output.GetPath(resolved, "cpp")
				if targetOutputPath != "" {
					ns := deriveNamespace(targetOutputPath)
					structType = fmt.Sprintf("::%s::%s", ns, structType)
				}
			}
			return fmt.Sprintf(`parser.field<%s>("%s")`, structType, jsonName)
		}
	}

	if mapping := primitiveMapper.Map(typeRef.Name); mapping.TargetType != "" && mapping.TargetType != "void" {
		return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
	}

	return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
}

func (p *Plugin) genericParseExprsForField(field resolution.Field, data *templateData) (jsonParseExpr, structParseExpr string) {
	jsonName := toSnakeCase(field.Name)
	typeParamName := field.Type.TypeParam.Name

	if field.IsHardOptional {
		jsonParseExpr = fmt.Sprintf(`parser.has("%s") ? std::make_optional(parser.field<x::json::json>("%s")) : std::nullopt`, jsonName, jsonName)
		structParseExpr = fmt.Sprintf(`parser.has("%s") ? std::make_optional(parser.field<%s>("%s")) : std::nullopt`, jsonName, typeParamName, jsonName)
	} else {
		jsonParseExpr = fmt.Sprintf(`parser.field<x::json::json>("%s")`, jsonName)
		structParseExpr = fmt.Sprintf(`parser.field<%s>("%s")`, typeParamName, jsonName)
	}

	return jsonParseExpr, structParseExpr
}

func (p *Plugin) toJSONExprForField(field resolution.Field, parent resolution.Type, data *templateData, isSelfRef bool) string {
	typeRef := field.Type
	jsonName := toSnakeCase(field.Name)

	fieldName := domain.GetFieldName(field, "cpp")
	if fieldName == field.Name {
		fieldName = toSnakeCase(field.Name)
	}

	if typeRef.TypeParam != nil && !typeRef.TypeParam.HasDefault() {
		typeName := typeRef.TypeParam.Name
		return fmt.Sprintf(`if constexpr (std::is_same_v<%s, x::json::json>)
        j["%s"] = this->%s;
    else if constexpr (std::is_same_v<%s, std::monostate>)
        j["%s"] = nullptr;
    else
        j["%s"] = this->%s.to_json();`, typeName, jsonName, fieldName, typeName, jsonName, jsonName, fieldName)
	}

	if resolved, ok := typeRef.Resolve(data.table); ok {
		if distinctForm, isDistinct := resolved.Form.(resolution.DistinctForm); isDistinct {
			if distinctForm.Base.Name == "Array" && len(distinctForm.Base.TypeArgs) > 0 {
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
            if constexpr (std::is_same_v<%s, x::json::json>)
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
				return fmt.Sprintf(`{
        auto arr = x::json::json::array();
        for (const auto& item : this->%s) arr.push_back(item.to_json());
        j["%s"] = arr;
    }`, fieldName, jsonName)
			}
		}

		return fmt.Sprintf(`j["%s"] = this->%s;`, jsonName, fieldName)
	}

	resolved, resolvedOk := typeRef.Resolve(data.table)
	if resolvedOk {
		if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
			if isSelfRef {
				return fmt.Sprintf(`if (this->%s.has_value()) j["%s"] = this->%s->to_json();`, fieldName, jsonName, fieldName)
			}
			if field.IsHardOptional {
				return fmt.Sprintf(`if (this->%s.has_value()) j["%s"] = this->%s->to_json();`, fieldName, jsonName, fieldName)
			}
			return fmt.Sprintf(`j["%s"] = this->%s.to_json();`, jsonName, fieldName)
		}
		if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
			if targetResolved, targetOk := aliasForm.Target.Resolve(data.table); targetOk {
				if _, isStruct := targetResolved.Form.(resolution.StructForm); isStruct {
					if field.IsHardOptional {
						return fmt.Sprintf(`if (this->%s.has_value()) j["%s"] = this->%s->to_json();`, fieldName, jsonName, fieldName)
					}
					return fmt.Sprintf(`j["%s"] = this->%s.to_json();`, jsonName, fieldName)
				}
			}
		}
	}

	lowerName := strings.ToLower(typeRef.Name)
	if strings.HasSuffix(lowerName, "timestamp") || strings.HasSuffix(lowerName, "timespan") {
		return fmt.Sprintf(`j["%s"] = this->%s.nanoseconds();`, jsonName, fieldName)
	}

	if typeRef.Name == "uuid" || p.resolvesToUUID(typeRef, data) {
		if field.IsHardOptional {
			return fmt.Sprintf(`if (this->%s.has_value()) j["%s"] = this->%s->to_json();`, fieldName, jsonName, fieldName)
		}
		return fmt.Sprintf(`j["%s"] = this->%s.to_json();`, jsonName, fieldName)
	}

	return fmt.Sprintf(`j["%s"] = this->%s;`, jsonName, fieldName)
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
	case "json":
		return "x::json::json{}"
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

func (p *Plugin) isFixedSizeUint8ArrayType(resolved resolution.Type) bool {
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
	OutputPath    string
	Namespace     string
	Serializers   []serializerData
	ArrayWrappers []arrayWrapperData
	includes      *includeManager
	table         *resolution.Table
	rawNs         string
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
	IsGeneric      bool
	TypeParams     []typeParamData
	TypeParamNames string
	Fields         []fieldData
	// HasExtends indicates inheritance support.
	HasExtends  bool
	ParentTypes []parentTypeData
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
	IsGenericField  bool
	TypeParamName   string
	IsHardOptional  bool
	JSONParseExpr   string
	StructParseExpr string
}
