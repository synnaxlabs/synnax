// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb

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
	"github.com/synnaxlabs/oracle/plugin/enum"
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
		FileNamePattern: "proto.gen.h",
	}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string { return "cpp/pb" }

func (p *Plugin) Domains() []string { return []string{"cpp", "pb"} }

func (p *Plugin) Requires() []string { return []string{"cpp/types", "pb/types"} }

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

	outputStructs := make(map[string][]resolution.Type)
	var outputOrder []string

	for _, entry := range req.Resolutions.StructTypes() {
		cppOutputPath := output.GetPath(entry, "cpp")
		if cppOutputPath == "" {
			continue
		}
		if omit.IsType(entry, "cpp") {
			continue
		}
		if !hasPBFlag(entry) {
			continue
		}
		if omit.IsType(entry, "pb") {
			continue
		}

		if req.RepoRoot != "" {
			if err := req.ValidateOutputPath(cppOutputPath); err != nil {
				return nil, errors.Wrapf(err, "invalid output path for struct %s", entry.Name)
			}
		}

		if _, exists := outputStructs[cppOutputPath]; !exists {
			outputOrder = append(outputOrder, cppOutputPath)
		}
		outputStructs[cppOutputPath] = append(outputStructs[cppOutputPath], entry)
	}

	standaloneEnums := make(map[string][]resolution.Type)
	for _, e := range req.Resolutions.EnumTypes() {
		if omit.IsType(e, "cpp") || omit.IsType(e, "pb") {
			continue
		}
		if !hasPBFlag(e) {
			continue
		}
		enumPath := enum.FindOutputPath(e, req.Resolutions, "cpp")
		if enumPath == "" {
			continue
		}
		if _, exists := outputStructs[enumPath]; !exists {
			standaloneEnums[enumPath] = append(standaloneEnums[enumPath], e)
		}
	}
	for path := range standaloneEnums {
		found := false
		for _, p := range outputOrder {
			if p == path {
				found = true
				break
			}
		}
		if !found {
			outputOrder = append(outputOrder, path)
		}
	}

	for _, outputPath := range outputOrder {
		structs := outputStructs[outputPath]
		namespace := ""
		if len(structs) > 0 {
			namespace = structs[0].Namespace
		}

		enums := enum.CollectReferenced(structs, req.Resolutions)

		if len(structs) == 0 {
			if standalone, ok := standaloneEnums[outputPath]; ok {
				enums = append(enums, standalone...)
				if len(standalone) > 0 {
					namespace = standalone[0].Namespace
				}
			}
		}

		content, err := p.generateProto(outputPath, structs, enums, namespace, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate proto for %s", outputPath)
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

func hasPBFlag(t resolution.Type) bool {
	_, hasPB := t.Domains["pb"]
	return hasPB
}

func (p *Plugin) generateProto(
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	namespace string,
	req *plugin.Request,
) ([]byte, error) {
	data := &templateData{
		OutputPath:       outputPath,
		Namespace:        deriveNamespace(outputPath),
		Translators:      make([]translatorData, 0, len(structs)),
		EnumTranslators:  make([]enumTranslatorData, 0),
		includes:         newIncludeManager(),
		table:            req.Resolutions,
		rawNs:            namespace,
		processedEnums:   make(map[string]bool),
		processedStructs: make(map[string]bool),
	}

	data.includes.addSystem("type_traits")
	data.includes.addSystem("utility")
	data.includes.addInternal(fmt.Sprintf("%s/types.gen.h", outputPath))
	data.includes.addInternal("x/cpp/errors/errors.h")

	pbOutputPaths := make(map[string]bool)
	for _, s := range structs {
		pbPath := output.GetPBPath(s)
		if pbPath != "" && !pbOutputPaths[pbPath] {
			pbOutputPaths[pbPath] = true
			protoInclude := deriveProtoInclude(pbPath, s.Namespace)
			if protoInclude != "" {
				data.includes.addInternal(protoInclude)
			}
		}
	}

	for _, s := range structs {
		if omit.IsType(s, "cpp") || omit.IsType(s, "pb") {
			continue
		}
		form, ok := s.Form.(resolution.StructForm)
		if !ok {
			continue
		}

		translator := p.processStructForTranslation(s, form, data, req)
		if translator != nil {
			data.Translators = append(data.Translators, *translator)
		}
	}

	for _, e := range enums {
		if omit.IsType(e, "cpp") || omit.IsType(e, "pb") {
			continue
		}
		if e.Namespace != namespace {
			continue
		}
		if data.processedEnums[e.QualifiedName] {
			continue
		}
		data.processedEnums[e.QualifiedName] = true

		enumTranslator := p.processEnumForTranslation(e, data)
		if enumTranslator != nil {
			data.EnumTranslators = append(data.EnumTranslators, *enumTranslator)
		}
	}

	if len(data.Translators) == 0 && len(data.EnumTranslators) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	if err := protoTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Plugin) processStructForTranslation(
	s resolution.Type,
	form resolution.StructForm,
	data *templateData,
	req *plugin.Request,
) *translatorData {
	cppName := domain.GetName(s, "cpp")

	pbName := getPBName(s)
	pbOutputPath := output.GetPBPath(s)
	pbNamespace := derivePBCppNamespace(pbOutputPath)

	typeParams := make([]typeParamData, 0, len(form.TypeParams))
	typeParamNames := make([]string, 0, len(form.TypeParams))
	for _, tp := range form.TypeParams {
		typeParams = append(typeParams, typeParamData{Name: tp.Name})
		typeParamNames = append(typeParamNames, tp.Name)
	}

	translator := &translatorData{
		CppName:        cppName,
		PBName:         pbName,
		PBNamespace:    pbNamespace,
		Fields:         make([]fieldTranslatorData, 0),
		IsGeneric:      form.IsGeneric(),
		TypeParams:     typeParams,
		TypeParamNames: strings.Join(typeParamNames, ", "),
	}

	if form.IsGeneric() {
		data.includes.addInternal("x/cpp/json/any.h")
	}

	for _, field := range resolution.UnifiedFields(s, data.table) {
		fieldData := p.processFieldForTranslation(field, form, data)
		translator.Fields = append(translator.Fields, fieldData)
	}

	return translator
}

func (p *Plugin) processFieldForTranslation(
	field resolution.Field,
	form resolution.StructForm,
	data *templateData,
) fieldTranslatorData {
	fieldName := field.Name
	pbFieldName := field.Name

	isGenericField := false
	typeParamName := ""
	if field.Type.TypeParam != nil {
		isGenericField = true
		typeParamName = field.Type.TypeParam.Name
	}

	forwardExpr, backwardExpr := p.generateFieldConversion(field, data)
	forwardJsonExpr, backwardJsonExpr := "", ""
	if isGenericField {
		forwardJsonExpr, backwardJsonExpr = p.generateJsonFieldConversion(field, data)
	}

	return fieldTranslatorData{
		CppName:          fieldName,
		PBName:           pbFieldName,
		ForwardExpr:      forwardExpr,
		BackwardExpr:     backwardExpr,
		ForwardJsonExpr:  forwardJsonExpr,
		BackwardJsonExpr: backwardJsonExpr,
		IsOptional:       field.IsHardOptional,
		IsArray:          field.Type.Name == "Array",
		IsGenericField:   isGenericField,
		TypeParamName:    typeParamName,
	}
}

func (p *Plugin) generateFieldConversion(
	field resolution.Field,
	data *templateData,
) (forward, backward string) {
	typeRef := field.Type
	fieldName := field.Name
	pbSetter := fmt.Sprintf("pb.set_%s", fieldName)

	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		return p.generateArrayConversion(field, data)
	}

	if resolution.IsPrimitive(typeRef.Name) {
		return p.generatePrimitiveConversion(typeRef.Name, fieldName, pbSetter, field.IsHardOptional, data)
	}

	if typeRef.TypeParam != nil {
		return p.generateTypeParamConversion(field, data, fieldName)
	}

	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return fmt.Sprintf("%s(this->%s)", pbSetter, fieldName),
			fmt.Sprintf("cpp.%s = pb.%s();", fieldName, fieldName)
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		return p.generateStructConversion(typeRef, resolved, field.IsHardOptional, data, fieldName)
	case resolution.EnumForm:
		return p.generateEnumConversion(resolved, form, fieldName, pbSetter, data)
	case resolution.DistinctForm:
		return p.generateDistinctConversion(resolved, form, fieldName, pbSetter, data)
	case resolution.AliasForm:
		return p.generateAliasConversion(resolved, form, field.IsHardOptional, fieldName, pbSetter, data)
	default:
		return fmt.Sprintf("%s(this->%s)", pbSetter, fieldName),
			fmt.Sprintf("cpp.%s = pb.%s();", fieldName, fieldName)
	}
}

func (p *Plugin) generateJsonFieldConversion(
	field resolution.Field,
	data *templateData,
) (forward, backward string) {
	fieldName := field.Name
	if field.IsHardOptional {
		forward = fmt.Sprintf("if (this->%s.has_value()) *pb.mutable_%s() = x::json::to_any(*this->%s)", fieldName, fieldName, fieldName)
		backward = fmt.Sprintf(`if (pb.has_%s()) {
        auto [val, err] = x::json::from_any(pb.%s());
        if (err) return {{}, err};
        cpp.%s = val;
    }`, fieldName, fieldName, fieldName)
	} else {
		forward = fmt.Sprintf("*pb.mutable_%s() = x::json::to_any(this->%s)", fieldName, fieldName)
		backward = fmt.Sprintf(`{
        auto [val, err] = x::json::from_any(pb.%s());
        if (err) return {{}, err};
        cpp.%s = val;
    }`, fieldName, fieldName)
	}
	return forward, backward
}

func (p *Plugin) generatePrimitiveConversion(
	primitive, fieldName, pbSetter string,
	isOptional bool,
	data *templateData,
) (forward, backward string) {
	switch primitive {
	case "uuid":
		return fmt.Sprintf("%s(this->%s)", pbSetter, fieldName),
			fmt.Sprintf("cpp.%s = pb.%s();", fieldName, fieldName)
	case "timestamp":
		return fmt.Sprintf("%s(this->%s.nanoseconds())", pbSetter, fieldName),
			fmt.Sprintf("cpp.%s = x::telem::TimeStamp(pb.%s());", fieldName, fieldName)
	case "timespan":
		return fmt.Sprintf("%s(this->%s.nanoseconds())", pbSetter, fieldName),
			fmt.Sprintf("cpp.%s = x::telem::TimeSpan(pb.%s());", fieldName, fieldName)
	case "data_type":
		return fmt.Sprintf("%s(std::string(this->%s))", pbSetter, fieldName),
			fmt.Sprintf("cpp.%s = x::telem::DataType(pb.%s());", fieldName, fieldName)
	case "json":
		data.includes.addInternal("x/cpp/json/struct.h")
		if isOptional {
			forward = fmt.Sprintf("if (this->%s.has_value()) *pb.mutable_%s() = x::json::to_struct(*this->%s).first", fieldName, fieldName, fieldName)
			backward = fmt.Sprintf(`if (pb.has_%s()) {
        auto [val, err] = x::json::from_struct(pb.%s());
        if (err) return {{}, err};
        cpp.%s = val;
    }`, fieldName, fieldName, fieldName)
		} else {
			forward = fmt.Sprintf("*pb.mutable_%s() = x::json::to_struct(this->%s).first", fieldName, fieldName)
			backward = fmt.Sprintf(`{
        auto [val, err] = x::json::from_struct(pb.%s());
        if (err) return {{}, err};
        cpp.%s = val;
    }`, fieldName, fieldName)
		}
		return forward, backward
	default:
		return fmt.Sprintf("%s(this->%s)", pbSetter, fieldName),
			fmt.Sprintf("cpp.%s = pb.%s();", fieldName, fieldName)
	}
}

func (p *Plugin) generateStructConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	isOptional bool,
	data *templateData,
	fieldName string,
) (forward, backward string) {
	nsPrefix := ""
	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			data.includes.addInternal(fmt.Sprintf("%s/proto.gen.h", targetOutputPath))
		}
	}

	if isOptional {
		forward = fmt.Sprintf("if (this->%s.has_value()) *pb.mutable_%s() = this->%s->to_proto()", fieldName, fieldName, fieldName)
		backward = fmt.Sprintf(`if (pb.has_%s()) {
        auto [val, err] = %s::from_proto(pb.%s());
        if (err) return {{}, err};
        cpp.%s = val;
    }`, fieldName, p.typeRefToCppForTranslator(typeRef, data), fieldName, fieldName)
	} else {
		forward = fmt.Sprintf("*pb.mutable_%s() = this->%s.to_proto()", fieldName, fieldName)
		backward = fmt.Sprintf(`{
        auto [val, err] = %s::from_proto(pb.%s());
        if (err) return {{}, err};
        cpp.%s = val;
    }`, p.typeRefToCppForTranslator(typeRef, data), fieldName, fieldName)
	}

	_ = nsPrefix
	return forward, backward
}

func (p *Plugin) typeRefToCppForTranslator(typeRef resolution.TypeRef, data *templateData) string {
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		return typeRef.TypeParam.Name
	}

	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return typeRef.Name
	}

	name := resolved.Name
	if cppDomain, ok := resolved.Domains["cpp"]; ok {
		for _, expr := range cppDomain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				name = expr.Values[0].StringValue
			}
		}
	}

	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			ns := deriveNamespace(targetOutputPath)
			name = fmt.Sprintf("%s::%s", ns, name)
		}
	}

	if len(typeRef.TypeArgs) > 0 {
		args := make([]string, len(typeRef.TypeArgs))
		for i, arg := range typeRef.TypeArgs {
			args[i] = p.typeRefToCppForTranslator(arg, data)
		}
		name = fmt.Sprintf("%s<%s>", name, strings.Join(args, ", "))
	}

	return name
}

func (p *Plugin) generateTypeParamConversion(
	field resolution.Field,
	data *templateData,
	fieldName string,
) (forward, backward string) {
	typeParamName := field.Type.TypeParam.Name
	if field.IsHardOptional {
		forward = fmt.Sprintf("if (this->%s.has_value()) pb.mutable_%s()->PackFrom(this->%s->to_proto())", fieldName, fieldName, fieldName)
		backward = fmt.Sprintf(`if (pb.has_%s()) {
        typename %s::proto_type pb_val;
        if (!pb.%s().UnpackTo(&pb_val)) return {{}, x::errors::Error("failed to unpack %s")};
        auto [val, err] = %s::from_proto(pb_val);
        if (err) return {{}, err};
        cpp.%s = val;
    }`, fieldName, typeParamName, fieldName, fieldName, typeParamName, fieldName)
	} else {
		forward = fmt.Sprintf("pb.mutable_%s()->PackFrom(this->%s.to_proto())", fieldName, fieldName)
		backward = fmt.Sprintf(`{
        typename %s::proto_type pb_val;
        if (!pb.%s().UnpackTo(&pb_val)) return {{}, x::errors::Error("failed to unpack %s")};
        auto [val, err] = %s::from_proto(pb_val);
        if (err) return {{}, err};
        cpp.%s = val;
    }`, typeParamName, fieldName, fieldName, typeParamName, fieldName)
	}
	return forward, backward
}

func (p *Plugin) generateEnumConversion(
	resolved resolution.Type,
	form resolution.EnumForm,
	fieldName, pbSetter string,
	data *templateData,
) (forward, backward string) {
	enumName := resolved.Name

	// Derive the pb namespace for the enum
	pbOutputPath := enum.FindPBOutputPath(resolved, data.table)
	pbNamespace := derivePBCppNamespace(pbOutputPath)

	if form.IsIntEnum {
		forward = fmt.Sprintf("%s(static_cast<%s::%s>(this->%s))", pbSetter, pbNamespace, enumName, fieldName)
		backward = fmt.Sprintf("cpp.%s = static_cast<%s>(pb.%s());", fieldName, enumName, fieldName)
	} else {
		forward = fmt.Sprintf("%s(%sToPB(this->%s))", pbSetter, enumName, fieldName)
		backward = fmt.Sprintf("cpp.%s = %sFromPB(pb.%s());", fieldName, enumName, fieldName)
	}

	return forward, backward
}

func (p *Plugin) generateDistinctConversion(
	resolved resolution.Type,
	form resolution.DistinctForm,
	fieldName, pbSetter string,
	data *templateData,
) (forward, backward string) {
	cppName := domain.GetName(resolved, "cpp")

	if resolution.IsPrimitive(form.Base.Name) {
		protoType := primitiveToProtoType(form.Base.Name)
		return fmt.Sprintf("%s(static_cast<%s>(this->%s))", pbSetter, protoType, fieldName),
			fmt.Sprintf("cpp.%s = %s(pb.%s());", fieldName, cppName, fieldName)
	}

	return fmt.Sprintf("%s(this->%s)", pbSetter, fieldName),
		fmt.Sprintf("cpp.%s = pb.%s();", fieldName, fieldName)
}

func (p *Plugin) generateAliasConversion(
	resolved resolution.Type,
	form resolution.AliasForm,
	isOptional bool,
	fieldName, pbSetter string,
	data *templateData,
) (forward, backward string) {
	if resolution.IsPrimitive(form.Target.Name) {
		return p.generatePrimitiveConversion(form.Target.Name, fieldName, pbSetter, isOptional, data)
	}

	// Follow through to the target type
	targetResolved, ok := form.Target.Resolve(data.table)
	if !ok {
		return fmt.Sprintf("%s(this->%s)", pbSetter, fieldName),
			fmt.Sprintf("cpp.%s = pb.%s();", fieldName, fieldName)
	}

	// If the target is a struct, generate struct conversion
	if _, isStruct := targetResolved.Form.(resolution.StructForm); isStruct {
		// Use the alias type name for from_proto since that's what the field type is
		// Get the C++ name directly from the resolved type (respects @cpp name directive)
		cppType := domain.GetName(resolved, "cpp")
		if resolved.Namespace != data.rawNs {
			targetOutputPath := output.GetPath(resolved, "cpp")
			if targetOutputPath != "" {
				ns := deriveNamespace(targetOutputPath)
				cppType = fmt.Sprintf("%s::%s", ns, cppType)
			}
		}
		if isOptional {
			forward = fmt.Sprintf("if (this->%s.has_value()) *pb.mutable_%s() = this->%s->to_proto()", fieldName, fieldName, fieldName)
			backward = fmt.Sprintf(`if (pb.has_%s()) {
        auto [val, err] = %s::from_proto(pb.%s());
        if (err) return {{}, err};
        cpp.%s = val;
    }`, fieldName, cppType, fieldName, fieldName)
		} else {
			forward = fmt.Sprintf("*pb.mutable_%s() = this->%s.to_proto()", fieldName, fieldName)
			backward = fmt.Sprintf(`{
        auto [val, err] = %s::from_proto(pb.%s());
        if (err) return {{}, err};
        cpp.%s = val;
    }`, cppType, fieldName, fieldName)
		}
		return forward, backward
	}

	// If the target is another alias, recursively handle it
	if targetForm, isAlias := targetResolved.Form.(resolution.AliasForm); isAlias {
		return p.generateAliasConversion(targetResolved, targetForm, isOptional, fieldName, pbSetter, data)
	}

	return fmt.Sprintf("%s(this->%s)", pbSetter, fieldName),
		fmt.Sprintf("cpp.%s = pb.%s();", fieldName, fieldName)
}

func (p *Plugin) generateArrayConversion(
	field resolution.Field,
	data *templateData,
) (forward, backward string) {
	fieldName := field.Name
	typeRef := field.Type

	if len(typeRef.TypeArgs) == 0 {
		return "// TODO: array without type args", "// TODO: array without type args"
	}

	elemType := typeRef.TypeArgs[0]

	if !resolution.IsPrimitive(elemType.Name) {
		if resolved, ok := elemType.Resolve(data.table); ok {
			if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
				if resolved.Namespace != data.rawNs {
					targetOutputPath := output.GetPath(resolved, "cpp")
					if targetOutputPath != "" {
						data.includes.addInternal(fmt.Sprintf("%s/proto.gen.h", targetOutputPath))
					}
				}
				forward = fmt.Sprintf("for (const auto& item : this->%s) *pb.add_%s() = item.to_proto()", fieldName, fieldName)
				backward = fmt.Sprintf(`for (const auto& item : pb.%s()) {
        auto [v, err] = %s::from_proto(item);
        if (err) return {{}, err};
        cpp.%s.push_back(v);
    }`, fieldName, p.typeRefToCppForTranslator(elemType, data), fieldName)
				return forward, backward
			}
		}
	}

	forward = fmt.Sprintf("for (const auto& item : this->%s) pb.add_%s(item)", fieldName, fieldName)
	backward = fmt.Sprintf("for (const auto& item : pb.%s()) cpp.%s.push_back(item);", fieldName, fieldName)

	return forward, backward
}

func (p *Plugin) processEnumForTranslation(
	e resolution.Type,
	data *templateData,
) *enumTranslatorData {
	form, ok := e.Form.(resolution.EnumForm)
	if !ok {
		return nil
	}

	if form.IsIntEnum {
		return nil
	}

	// Derive the pb namespace from the enum's pb output path
	pbOutputPath := enum.FindPBOutputPath(e, data.table)
	pbNamespace := derivePBCppNamespace(pbOutputPath)

	values := make([]enumValueTranslatorData, 0, len(form.Values))
	for _, v := range form.Values {
		cppValueName := fmt.Sprintf("%s_%s", toScreamingSnake(e.Name), toScreamingSnake(v.Name))
		pbValueName := fmt.Sprintf("%s_%s", toScreamingSnake(e.Name), toScreamingSnake(v.Name))

		values = append(values, enumValueTranslatorData{
			CppValue: cppValueName,
			PBValue:  pbValueName,
		})
	}

	return &enumTranslatorData{
		Name:        e.Name,
		PBNamespace: pbNamespace,
		Values:      values,
		PBDefault:   fmt.Sprintf("%s_UNSPECIFIED", toScreamingSnake(e.Name)),
		CppDefault:  fmt.Sprintf("%s_%s", toScreamingSnake(e.Name), toScreamingSnake(form.Values[0].Name)),
	}
}

func deriveNamespace(outputPath string) string {
	parts := strings.Split(outputPath, "/")
	if len(parts) == 0 {
		return "synnax"
	}

	// Determine the top-level namespace based on the path prefix
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

// derivePBCppNamespace converts a pb output path to a fully qualified C++ namespace.
// This mirrors the package derivation logic in pb/types plugin:
// - For "core/pkg/{layer}/{service}/pb" -> "::{layer}::{service}::pb"
// - For "x/go/{service}/pb" -> "::x::{service}::pb"
// - For other paths -> "::{first}::{last-before-pb}::pb"
func derivePBCppNamespace(pbOutputPath string) string {
	if pbOutputPath == "" {
		return "pb"
	}
	parts := strings.Split(pbOutputPath, "/")
	if len(parts) == 0 {
		return "pb"
	}

	// Derive namespace (directory before /pb, or last component)
	namespace := parts[len(parts)-1]
	if namespace == "pb" && len(parts) >= 2 {
		namespace = parts[len(parts)-2]
	}

	// Derive layer prefix (mirrors deriveLayerPrefix in pb/types)
	var prefix string
	if len(parts) >= 3 && parts[0] == "core" && parts[1] == "pkg" {
		prefix = parts[2] // e.g., "distribution", "service", "api"
	} else if len(parts) >= 1 && parts[0] != "" {
		prefix = parts[0] // e.g., "x", "freighter"
	} else {
		prefix = "synnax"
	}

	return fmt.Sprintf("::%s::%s::pb", prefix, namespace)
}

func deriveProtoInclude(pbOutputPath, namespace string) string {
	if pbOutputPath == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s.pb.h", pbOutputPath, namespace)
}

func getPBName(s resolution.Type) string {
	if domain, ok := s.Domains["pb"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return s.Name
}

func primitiveToProtoType(primitive string) string {
	switch primitive {
	case "uint8", "uint12", "uint16", "uint20", "uint32":
		return "uint32_t"
	case "uint64":
		return "uint64_t"
	case "int8", "int16", "int32":
		return "int32_t"
	case "int64":
		return "int64_t"
	case "float32":
		return "float"
	case "float64":
		return "double"
	case "bool":
		return "bool"
	case "string":
		return "std::string"
	default:
		return primitive
	}
}

func toPascalCase(s string) string {
	return lo.PascalCase(s)
}

func toScreamingSnake(s string) string {
	return strings.ToUpper(lo.SnakeCase(s))
}

func toSnakeCase(s string) string {
	return lo.SnakeCase(s)
}

type includeManager struct {
	system   []string
	internal []string
}

func newIncludeManager() *includeManager {
	return &includeManager{}
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
	OutputPath       string
	Namespace        string
	Translators      []translatorData
	EnumTranslators  []enumTranslatorData
	includes         *includeManager
	table            *resolution.Table
	rawNs            string
	processedEnums   map[string]bool
	processedStructs map[string]bool
}

func (d *templateData) HasIncludes() bool {
	return len(d.includes.system) > 0 || len(d.includes.internal) > 0
}
func (d *templateData) SystemIncludes() []string   { return d.includes.system }
func (d *templateData) InternalIncludes() []string { return d.includes.internal }

type translatorData struct {
	CppName        string
	PBName         string
	PBNamespace    string
	Fields         []fieldTranslatorData
	IsGeneric      bool
	TypeParams     []typeParamData
	TypeParamNames string
}

type typeParamData struct {
	Name string
}

type fieldTranslatorData struct {
	CppName          string
	PBName           string
	ForwardExpr      string
	BackwardExpr     string
	ForwardJsonExpr  string
	BackwardJsonExpr string
	IsOptional       bool
	IsArray          bool
	IsGenericField   bool
	TypeParamName    string
}

type enumTranslatorData struct {
	Name        string
	PBNamespace string
	Values      []enumValueTranslatorData
	PBDefault   string
	CppDefault  string
}

type enumValueTranslatorData struct {
	CppValue string
	PBValue  string
}
