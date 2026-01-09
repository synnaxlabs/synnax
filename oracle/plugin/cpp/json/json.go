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
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

var primitiveMapper = &cppprimitives.Mapper{}

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

var clangFormatCmd = []string{"clang-format", "-i"}

func (p *Plugin) PostWrite(files []string) error {
	if p.Options.DisableFormatter || len(files) == 0 {
		return nil
	}
	var cppFiles []string
	for _, f := range files {
		if strings.HasSuffix(f, ".h") || strings.HasSuffix(f, ".hpp") ||
			strings.HasSuffix(f, ".cpp") || strings.HasSuffix(f, ".cc") {
			cppFiles = append(cppFiles, f)
		}
	}
	if len(cppFiles) == 0 {
		return nil
	}
	return exec.OnFiles(clangFormatCmd, cppFiles, "")
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	structCollector, err := framework.CollectStructs("cpp", req)
	if err != nil {
		return nil, err
	}

	for _, outputPath := range structCollector.Paths() {
		structs := structCollector.Get(outputPath)
		if len(structs) == 0 {
			continue
		}

		namespace := structs[0].Namespace
		content, err := p.generateFile(outputPath, structs, namespace, req)
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
	namespace string,
	req *plugin.Request,
) ([]byte, error) {
	data := &templateData{
		OutputPath: outputPath,
		Namespace:  deriveNamespace(outputPath),
		Serializers: make([]serializerData, 0, len(structs)),
		includes:   newIncludeManager(),
		table:      req.Resolutions,
		rawNs:      namespace,
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

	if len(data.Serializers) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	if err := jsonTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
		typeParams = append(typeParams, typeParamData{Name: tp.Name})
		typeParamNames = append(typeParamNames, tp.Name)
	}

	serializer := &serializerData{
		Name:           cppName,
		IsGeneric:      form.IsGeneric(),
		TypeParams:     typeParams,
		TypeParamNames: strings.Join(typeParamNames, ", "),
		Fields:         make([]fieldData, 0),
	}

	if form.IsGeneric() {
		data.includes.addSystem("type_traits")
	}

	for _, field := range resolution.UnifiedFields(s, data.table) {
		fieldData := p.processField(field, data)
		serializer.Fields = append(serializer.Fields, fieldData)
	}

	return serializer, nil
}

func (p *Plugin) processField(field resolution.Field, data *templateData) fieldData {
	cppType := p.typeRefToCpp(field.Type, data)
	jsonName := field.Name

	isGenericField := field.Type.IsTypeParam() && field.Type.TypeParam != nil
	typeParamName := ""
	if isGenericField {
		typeParamName = field.Type.TypeParam.Name
	}

	parseExpr := p.parseExprForField(field, cppType, data)
	toJsonExpr := p.toJsonExprForField(field, data)

	var jsonParseExpr, structParseExpr string
	if isGenericField {
		jsonParseExpr, structParseExpr = p.genericParseExprsForField(field, data)
	}

	return fieldData{
		Name:           field.Name,
		CppType:        cppType,
		JsonName:       jsonName,
		ParseExpr:      parseExpr,
		ToJsonExpr:     toJsonExpr,
		IsGenericField: isGenericField,
		TypeParamName:  typeParamName,
		IsHardOptional: field.IsHardOptional,
		JsonParseExpr:  jsonParseExpr,
		StructParseExpr: structParseExpr,
	}
}

func (p *Plugin) typeRefToCpp(typeRef resolution.TypeRef, data *templateData) string {
	if typeRef.TypeParam != nil {
		return typeRef.TypeParam.Name
	}

	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		innerType := p.typeRefToCpp(typeRef.TypeArgs[0], data)
		data.includes.addSystem("vector")
		return fmt.Sprintf("std::vector<%s>", innerType)
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

	name := domain.GetName(resolved, "cpp")

	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			includePath := fmt.Sprintf("%s/json.gen.h", targetOutputPath)
			data.includes.addInternal(includePath)
			ns := deriveNamespace(targetOutputPath)
			name = fmt.Sprintf("%s::%s", ns, name)
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

func (p *Plugin) parseExprForField(field resolution.Field, cppType string, data *templateData) string {
	typeRef := field.Type
	jsonName := field.Name
	hasDefault := field.IsOptional

	if typeRef.TypeParam != nil {
		innerType := typeRef.TypeParam.Name
		if field.IsHardOptional {
			innerExpr := p.parseExprForTypeRef(typeRef, innerType, jsonName, false, data)
			return fmt.Sprintf(`parser.has("%s") ? std::make_optional(%s) : std::nullopt`, jsonName, innerExpr)
		}
		return fmt.Sprintf(`parser.field<%s>("%s")`, typeRef.TypeParam.Name, jsonName)
	}

	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		innerTypeRef := typeRef.TypeArgs[0]
		innerType := p.typeRefToCpp(innerTypeRef, data)

		if innerTypeRef.TypeParam != nil {
			return fmt.Sprintf(`parser.field<std::vector<%s>>("%s")`, innerTypeRef.TypeParam.Name, jsonName)
		}

		return fmt.Sprintf(`parser.field<std::vector<%s>>("%s")`, innerType, jsonName)
	}

	resolved, resolvedOk := typeRef.Resolve(data.table)
	if resolvedOk {
		if enumForm, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
			if !enumForm.IsIntEnum {
				if field.IsHardOptional {
					return fmt.Sprintf(`parser.optional_field<std::string>("%s")`, jsonName)
				}
				if hasDefault {
					return fmt.Sprintf(`parser.field<std::string>("%s", "")`, jsonName)
				}
				return fmt.Sprintf(`parser.field<std::string>("%s")`, jsonName)
			}
		}
		if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
			structType := domain.GetName(resolved, "cpp")
			if resolved.Namespace != data.rawNs {
				targetOutputPath := output.GetPath(resolved, "cpp")
				if targetOutputPath != "" {
					ns := deriveNamespace(targetOutputPath)
					structType = fmt.Sprintf("%s::%s", ns, structType)
				}
			}
			if field.IsHardOptional {
				return fmt.Sprintf(`%s::parse(parser.optional_child("%s"))`, structType, jsonName)
			}
			return fmt.Sprintf(`%s::parse(parser.child("%s"))`, structType, jsonName)
		}
	}

	if mapping := primitiveMapper.Map(typeRef.Name); mapping.TargetType != "" && mapping.TargetType != "void" {
		if field.IsHardOptional {
			return fmt.Sprintf(`parser.optional_field<%s>("%s")`, cppType, jsonName)
		}
		if hasDefault {
			defaultVal := defaultValueForPrimitive(typeRef.Name)
			return fmt.Sprintf(`parser.field<%s>("%s", %s)`, cppType, jsonName, defaultVal)
		}
		return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
	}

	if field.IsHardOptional {
		return fmt.Sprintf(`%s::parse(parser.optional_child("%s"))`, cppType, jsonName)
	}
	return fmt.Sprintf(`%s::parse(parser.child("%s"))`, cppType, jsonName)
}

func (p *Plugin) parseExprForTypeRef(typeRef resolution.TypeRef, cppType, jsonName string, hasDefault bool, data *templateData) string {
	if typeRef.TypeParam != nil {
		return fmt.Sprintf(`%s::parse(parser.child("%s"))`, typeRef.TypeParam.Name, jsonName)
	}

	resolved, resolvedOk := typeRef.Resolve(data.table)
	if resolvedOk {
		if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
			structType := domain.GetName(resolved, "cpp")
			if resolved.Namespace != data.rawNs {
				targetOutputPath := output.GetPath(resolved, "cpp")
				if targetOutputPath != "" {
					ns := deriveNamespace(targetOutputPath)
					structType = fmt.Sprintf("%s::%s", ns, structType)
				}
			}
			return fmt.Sprintf(`%s::parse(parser.child("%s"))`, structType, jsonName)
		}
	}

	if mapping := primitiveMapper.Map(typeRef.Name); mapping.TargetType != "" && mapping.TargetType != "void" {
		return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
	}

	return fmt.Sprintf(`%s::parse(parser.child("%s"))`, cppType, jsonName)
}

func (p *Plugin) genericParseExprsForField(field resolution.Field, data *templateData) (jsonParseExpr, structParseExpr string) {
	jsonName := field.Name
	typeParamName := field.Type.TypeParam.Name

	if field.IsHardOptional {
		jsonParseExpr = fmt.Sprintf(`parser.has("%s") ? std::make_optional(parser.field<x::json::json>("%s")) : std::nullopt`, jsonName, jsonName)
		structParseExpr = fmt.Sprintf(`parser.has("%s") ? std::make_optional(%s::parse(parser.child("%s"))) : std::nullopt`, jsonName, typeParamName, jsonName)
	} else {
		jsonParseExpr = fmt.Sprintf(`parser.field<x::json::json>("%s")`, jsonName)
		structParseExpr = fmt.Sprintf(`%s::parse(parser.child("%s"))`, typeParamName, jsonName)
	}

	return jsonParseExpr, structParseExpr
}

func (p *Plugin) toJsonExprForField(field resolution.Field, data *templateData) string {
	typeRef := field.Type
	jsonName := field.Name
	fieldName := field.Name

	if typeRef.TypeParam != nil {
		typeName := typeRef.TypeParam.Name
		return fmt.Sprintf(`if constexpr (std::is_same_v<%s, x::json::json>)
        j["%s"] = this->%s;
    else
        j["%s"] = this->%s.to_json();`, typeName, jsonName, fieldName, jsonName, fieldName)
	}

	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		innerTypeRef := typeRef.TypeArgs[0]

		if innerTypeRef.TypeParam != nil {
			typeName := innerTypeRef.TypeParam.Name
			return fmt.Sprintf(`{
        auto arr = x::json::json::array();
        for (const auto& item : this->%s)
            if constexpr (std::is_same_v<%s, x::json::json>)
                arr.push_back(item);
            else
                arr.push_back(item.to_json());
        j["%s"] = arr;
    }`, fieldName, typeName, jsonName)
		}

		resolvedInner, resolvedOk := innerTypeRef.Resolve(data.table)
		if resolvedOk {
			if _, isStruct := resolvedInner.Form.(resolution.StructForm); isStruct {
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
			return fmt.Sprintf(`j["%s"] = this->%s.to_json();`, jsonName, fieldName)
		}
	}

	if typeRef.Name == "timestamp" || typeRef.Name == "timespan" {
		return fmt.Sprintf(`j["%s"] = this->%s.nanoseconds();`, jsonName, fieldName)
	}

	return fmt.Sprintf(`j["%s"] = this->%s;`, jsonName, fieldName)
}

func defaultValueForPrimitive(primitive string) string {
	switch primitive {
	case "string", "uuid":
		return `""`
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

type templateData struct {
	OutputPath      string
	Namespace       string
	Serializers     []serializerData
	includes        *includeManager
	table           *resolution.Table
	rawNs           string
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
}

type typeParamData struct {
	Name string
}

type fieldData struct {
	Name            string
	CppType         string
	JsonName        string
	ParseExpr       string
	ToJsonExpr      string
	IsGenericField  bool
	TypeParamName   string
	IsHardOptional  bool
	JsonParseExpr   string
	StructParseExpr string
}
