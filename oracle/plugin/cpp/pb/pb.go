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
	"slices"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/cpp/internal/includes"
	"github.com/synnaxlabs/oracle/plugin/cpp/json"
	"github.com/synnaxlabs/oracle/plugin/cpp/keywords"
	"github.com/synnaxlabs/oracle/plugin/cpp/naming"
	cppprimitives "github.com/synnaxlabs/oracle/plugin/cpp/primitives"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/internal/arrays"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

var primitiveMapper = cppprimitives.Mapper()

type Plugin struct{ Options Options }

type Options struct {
	FileNamePattern string
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

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	req = req.OmitFields("cpp", "pb")
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	outputStructs := make(map[string][]resolution.Type)
	outputDistinct := make(map[string][]resolution.Type)
	var outputOrder []string

	for _, entry := range req.Resolutions.StructTypes() {
		cppOutputPath := output.GetPath(entry, "cpp")
		if cppOutputPath == "" {
			continue
		}
		if !hasPBFlag(entry) {
			continue
		}
		if omit.IsSkipped(entry, "pb") {
			continue
		}
		if omit.IsSkipped(entry, "cpp") {
			continue
		}

		if req.RepoRoot != "" {
			if err := req.ValidateOutputPath(cppOutputPath); err != nil {
				return nil, errors.Wrapf(
					err,
					"invalid output path for struct %s",
					entry.Name,
				)
			}
		}

		if _, exists := outputStructs[cppOutputPath]; !exists {
			outputOrder = append(outputOrder, cppOutputPath)
		}
		outputStructs[cppOutputPath] = append(outputStructs[cppOutputPath], entry)
	}

	for _, entry := range req.Resolutions.DistinctTypes() {
		cppOutputPath := output.GetPath(entry, "cpp")
		if cppOutputPath == "" {
			continue
		}
		if omit.IsSkipped(entry, "cpp") || omit.IsSkipped(entry, "pb") {
			continue
		}
		if !hasPBFlag(entry) || !hasExplicitPBName(entry) {
			continue
		}

		form, ok := entry.Form.(resolution.DistinctForm)
		if !ok {
			continue
		}
		if _, wraps := arrays.Wrapped(form, req.Resolutions); !wraps {
			continue
		}

		if _, exists := outputStructs[cppOutputPath]; !exists {
			if _, exists := outputDistinct[cppOutputPath]; !exists {
				outputOrder = append(outputOrder, cppOutputPath)
			}
		}
		outputDistinct[cppOutputPath] = append(outputDistinct[cppOutputPath], entry)
	}

	outputUnions := make(map[string][]resolution.Type)
	for _, u := range req.Resolutions.UnionTypes() {
		cppOutputPath := output.GetPath(u, "cpp")
		if cppOutputPath == "" || !hasPBFlag(u) {
			continue
		}
		if omit.IsSkipped(u, "pb") || omit.IsSkipped(u, "cpp") {
			continue
		}
		if !slices.Contains(outputOrder, cppOutputPath) {
			outputOrder = append(outputOrder, cppOutputPath)
		}
		outputUnions[cppOutputPath] = append(outputUnions[cppOutputPath], u)
	}

	standaloneEnums := make(map[string][]resolution.Type)
	for _, e := range req.Resolutions.EnumTypes() {
		if omit.IsSkipped(e, "cpp") || omit.IsSkipped(e, "pb") {
			continue
		}
		if !hasPBFlag(e) {
			continue
		}
		enumPath := enum.FindOutputPath(e, req.Resolutions, "cpp")
		if enumPath == "" {
			continue
		}
		standaloneEnums[enumPath] = append(standaloneEnums[enumPath], e)
	}
	for path := range standaloneEnums {
		found := slices.Contains(outputOrder, path)
		if !found {
			outputOrder = append(outputOrder, path)
		}
	}

	jsonPaths, err := json.ContentPaths(req)
	if err != nil {
		return nil, err
	}

	protoPaths := make(set.Set[string])
	for _, outputPath := range outputOrder {
		protoPaths.Add(outputPath)
	}

	for _, outputPath := range outputOrder {
		structs := outputStructs[outputPath]
		unions := outputUnions[outputPath]
		namespace := ""
		if len(structs) > 0 {
			namespace = structs[0].Namespace
		} else if len(unions) > 0 {
			namespace = unions[0].Namespace
		}

		enums := enum.CollectReferenced(structs, req.Resolutions)

		// Standalone pb enums must translate even when no sibling pb struct
		// references them: cross-namespace callers resolve the translator here.
		if standalone, ok := standaloneEnums[outputPath]; ok {
			enums = append(enums, standalone...)
			if namespace == "" && len(standalone) > 0 {
				namespace = standalone[0].Namespace
			}
		}

		content, err := p.generateProto(
			outputPath, structs, unions, enums, namespace, jsonPaths, protoPaths, req,
		)
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

func hasExplicitPBName(t resolution.Type) bool {
	if domain, ok := t.Domains["pb"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 &&
				expr.Values[0].StringValue != "" {
				return true
			}
		}
	}
	return false
}

func (p *Plugin) generateProto(
	outputPath string,
	structs []resolution.Type,
	unions []resolution.Type,
	enums []resolution.Type,
	namespace string,
	jsonPaths set.Set[string],
	protoPaths set.Set[string],
	req *plugin.Request,
) ([]byte, error) {
	data := &templateData{
		Namespace:        naming.Namespace(outputPath),
		Translators:      make([]translatorData, 0, len(structs)),
		UnionTranslators: make([]unionTranslatorData, 0, len(unions)),
		EnumTranslators:  make([]enumTranslatorData, 0),
		ArrayWrappers:    make([]arrayWrapperTranslatorData, 0),
		Manager:          includes.NewManager(),
		table:            req.Resolutions,
		jsonPaths:        jsonPaths,
		protoPaths:       protoPaths,
		rawNs:            namespace,
		processedEnums:   make(set.Set[string]),
	}

	data.AddSystem("utility")
	data.AddInternal(fmt.Sprintf("%s/types.gen.h", outputPath))
	data.AddInternal(data.jsonInclude(outputPath))
	data.AddInternal("x/cpp/errors/errors.h")
	data.AddInternal("x/cpp/pb/pb.h")

	pbOutputPaths := make(set.Set[string])
	for _, s := range append(slices.Clone(structs), unions...) {
		pbPath := output.GetPBPath(s)
		if pbPath != "" && !pbOutputPaths.Contains(pbPath) {
			pbOutputPaths.Add(pbPath)
			protoInclude := deriveProtoInclude(pbPath, s.Namespace)
			if protoInclude != "" {
				data.AddInternal(protoInclude)
			}
		}
	}

	for _, s := range structs {
		if omit.IsSkipped(s, "pb") {
			continue
		}
		if omit.IsSkipped(s, "cpp") {
			continue
		}
		form, ok := s.Form.(resolution.StructForm)
		if !ok {
			continue
		}

		translator := p.processStructForTranslation(s, form, data)
		if translator != nil {
			data.Translators = append(data.Translators, *translator)
		}
	}

	for _, u := range unions {
		form, ok := u.Form.(resolution.UnionForm)
		if !ok {
			continue
		}
		data.UnionTranslators = append(
			data.UnionTranslators,
			p.processUnionForTranslation(u, form, data),
		)
	}
	if len(data.UnionTranslators) > 0 {
		data.AddSystem("variant")
		data.AddSystem("type_traits")
	}

	// Add type_traits include only if there are generic translators
	for _, t := range data.Translators {
		if t.IsGeneric {
			data.AddSystem("type_traits")
			break
		}
	}

	for _, e := range enums {
		if omit.IsSkipped(e, "cpp") || omit.IsSkipped(e, "pb") {
			continue
		}
		if e.Namespace != namespace {
			continue
		}
		if data.processedEnums.Contains(e.QualifiedName) {
			continue
		}
		data.processedEnums.Add(e.QualifiedName)

		enumTranslator := p.processEnumForTranslation(e, data)
		if enumTranslator != nil {
			data.EnumTranslators = append(data.EnumTranslators, *enumTranslator)
		}
	}

	// Add includes for enum translators (string enums use unordered_map)
	if len(data.EnumTranslators) > 0 {
		data.AddSystem("string")
		data.AddSystem("unordered_map")
	}

	for _, dt := range req.Resolutions.DistinctTypes() {
		if output.GetPath(dt, "cpp") != outputPath {
			continue
		}
		if omit.IsSkipped(dt, "cpp") || omit.IsSkipped(dt, "pb") {
			continue
		}
		if !hasPBFlag(dt) || !hasExplicitPBName(dt) {
			continue
		}

		form, ok := dt.Form.(resolution.DistinctForm)
		if !ok {
			continue
		}

		arrayWrapper := p.processArrayWrapperForTranslation(dt, form, data)
		if arrayWrapper != nil {
			data.ArrayWrappers = append(data.ArrayWrappers, *arrayWrapper)
		}
	}

	if len(data.Translators) == 0 && len(data.EnumTranslators) == 0 &&
		len(data.ArrayWrappers) == 0 && len(data.UnionTranslators) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	if err := protoTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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
			if data.protoPaths.Contains(targetOutputPath) {
				data.AddInternal(targetOutputPath + "/proto.gen.h")
			}
			ns := naming.Namespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	}

	name += p.typeArgsToCpp(
		resolution.TypeParams(parent.Form),
		extendsRef.TypeArgs,
		data,
	)

	return name
}

// processUnionForTranslation builds the translator view for a discriminated union:
// a member translator per variant struct against its payload message, and the free
// functions that dispatch between the std::variant and the protobuf oneof wrapper.
// The union's extends bases are message fields on the wrapper, so the dispatch
// converts them beside the variant payload.
func (p *Plugin) processUnionForTranslation(
	u resolution.Type,
	form resolution.UnionForm,
	data *templateData,
) unionTranslatorData {
	cppName := domain.GetName(u, "cpp")
	pbNamespace := naming.PBNamespace(output.GetPBPath(u), u.Namespace)
	ut := unionTranslatorData{
		CppName:   cppName,
		SnakeName: casing.FieldSnake(cppName),
		PBType:    fmt.Sprintf("%s::%s", pbNamespace, naming.PBName(u)),
	}
	for _, ext := range form.Extends {
		base, ok := ext.Resolve(data.table)
		if !ok {
			continue
		}
		// An alias base keeps its own field name but translates through the struct it
		// stands for.
		target, targetRef, ok := resolution.Dealias(ext, data.table)
		if !ok {
			continue
		}
		if _, isStruct := target.Form.(resolution.StructForm); !isStruct {
			continue
		}
		ut.Bases = append(ut.Bases, unionBaseTranslatorData{
			CppType:    p.resolveExtendsType(targetRef, target, data),
			PBAccessor: keywords.Escape(casing.FieldSnake(base.Name)),
		})
	}
	for _, v := range form.Variants {
		payload, _, ok := resolution.Dealias(v.Type, data.table)
		if !ok {
			continue
		}
		variantName := naming.VariantTypeName(cppName, v.Name)
		messageNamespace, _ := naming.VariantPBNamespace(u, v, payload)
		translator := translatorData{
			CppName:     variantName,
			PBName:      naming.PBName(payload),
			PBNamespace: messageNamespace,
			Fields:      make([]fieldTranslatorData, 0),
		}
		for _, field := range resolution.UnifiedFields(payload, data.table) {
			translator.Fields = append(
				translator.Fields,
				p.processFieldForTranslation(field, data),
			)
		}
		data.Translators = append(data.Translators, translator)
		accessor := casing.FieldSnake(v.Name)
		ut.Variants = append(ut.Variants, unionVariantTranslatorData{
			TypeName:   variantName,
			PBAccessor: keywords.Escape(accessor),
			CaseName:   lo.PascalCase(accessor),
		})
	}
	return ut
}

// unionTranslatorNames returns the to_proto and from_proto functions for a union.
// Both are namespace-qualified: a call from inside a struct's own to_proto would
// otherwise resolve to that member function.
func (p *Plugin) unionTranslatorNames(
	resolved resolution.Type,
	data *templateData,
) (toProto, fromProto string) {
	fromProto = casing.FieldSnake(domain.GetName(resolved, "cpp")) + "_from_proto"
	ns := data.Namespace
	if resolved.Namespace != data.rawNs {
		if targetOutputPath := output.GetPath(resolved, "cpp"); targetOutputPath != "" {
			data.addConversionIncludes(targetOutputPath)
			ns = naming.Namespace(targetOutputPath)
		}
	}
	return fmt.Sprintf("::%s::to_proto", ns), fmt.Sprintf("::%s::%s", ns, fromProto)
}

func (p *Plugin) generateUnionConversion(
	resolved resolution.Type,
	isOptional bool,
	data *templateData,
	cppFieldName, pbAccessorName string,
) (forward, backward string) {
	toProto, fromProto := p.unionTranslatorNames(resolved, data)
	if isOptional {
		forward = fmt.Sprintf(`if (this->%s.has_value()) {
        auto [v, err] = %s(*this->%s);
        if (err) return {{}, err};
        *pb.mutable_%s() = v;
    }`, cppFieldName, toProto, cppFieldName, pbAccessorName)
		backward = fmt.Sprintf(`if (pb.has_%s()) {
        auto [v, err] = %s(pb.%s());
        if (err) return {{}, err};
        cpp.%s = v;
    }`, pbAccessorName, fromProto, pbAccessorName, cppFieldName)
		return forward, backward
	}
	forward = fmt.Sprintf(`{
        auto [v, err] = %s(this->%s);
        if (err) return {{}, err};
        *pb.mutable_%s() = v;
    }`, toProto, cppFieldName, pbAccessorName)
	backward = fmt.Sprintf(`{
        auto [v, err] = %s(pb.%s());
        if (err) return {{}, err};
        cpp.%s = v;
    }`, fromProto, pbAccessorName, cppFieldName)
	return forward, backward
}

func (p *Plugin) processStructForTranslation(
	s resolution.Type,
	form resolution.StructForm,
	data *templateData,
) *translatorData {
	cppName := domain.GetName(s, "cpp")

	pbName := naming.PBName(s)
	pbOutputPath := output.GetPBPath(s)
	pbNamespace := naming.PBNamespace(pbOutputPath, s.Namespace)

	typeParams := make([]typeParamData, 0, len(form.TypeParams))
	typeParamNames := make([]string, 0, len(form.TypeParams))
	for _, tp := range form.TypeParams {
		if tp.HasDefault() {
			continue
		}
		typeParams = append(typeParams, typeParamData{Name: tp.Name})
		typeParamNames = append(typeParamNames, tp.Name)
	}

	translator := &translatorData{
		CppName:        cppName,
		PBName:         pbName,
		PBNamespace:    pbNamespace,
		Fields:         make([]fieldTranslatorData, 0),
		IsGeneric:      len(typeParams) > 0,
		TypeParams:     typeParams,
		TypeParamNames: strings.Join(typeParamNames, ", "),
	}

	if len(typeParams) > 0 {
		data.AddInternal("x/cpp/json/any.h")
		data.AddInternal("x/cpp/json/json.h")
	}

	if resolver.CanUseInheritance(form, "cpp", data.table) {
		translator.HasExtends = true
		for _, extendsRef := range form.Extends {
			parent, ok := extendsRef.Resolve(data.table)
			if !ok {
				continue
			}
			qualifiedName := p.resolveExtendsType(extendsRef, parent, data)
			translator.ParentTypes = append(translator.ParentTypes, parentTypeData{
				QualifiedName: qualifiedName,
			})
		}
		for _, field := range form.Fields {
			fieldData := p.processFieldForTranslation(field, data)
			translator.Fields = append(translator.Fields, fieldData)
		}
		for _, field := range resolution.UnifiedFields(s, data.table) {
			fieldData := p.processFieldForTranslation(field, data)
			translator.AllFields = append(translator.AllFields, fieldData)
		}
	} else {
		for _, field := range resolution.UnifiedFields(s, data.table) {
			fieldData := p.processFieldForTranslation(field, data)
			translator.Fields = append(translator.Fields, fieldData)
		}
	}

	return translator
}

func (p *Plugin) processFieldForTranslation(
	field resolution.Field,
	data *templateData,
) fieldTranslatorData {
	pbFieldName := casing.FieldSnake(field.Name)

	cppFieldName := domain.GetFieldName(field, "cpp")
	if cppFieldName == field.Name {
		cppFieldName = casing.FieldSnake(field.Name)
	}
	cppFieldName = keywords.Escape(cppFieldName)

	isGenericField := false
	typeParamName := ""
	if field.Type.TypeParam != nil {
		if !field.Type.TypeParam.HasDefault() {
			isGenericField = true
			typeParamName = field.Type.TypeParam.Name
		}
	}

	forwardExpr, backwardExpr := p.generateFieldConversion(field, cppFieldName, data)
	var forwardJSONExpr, backwardJSONExpr string
	if isGenericField {
		pbAccessorName := keywords.Escape(pbFieldName)
		forwardJSONExpr, backwardJSONExpr = p.generateJSONFieldConversion(
			field,
			cppFieldName,
			pbAccessorName,
		)
	}

	return fieldTranslatorData{
		CppName:          cppFieldName,
		PBName:           pbFieldName,
		ForwardExpr:      forwardExpr,
		BackwardExpr:     backwardExpr,
		ForwardJSONExpr:  forwardJSONExpr,
		BackwardJSONExpr: backwardJSONExpr,
		IsGenericField:   isGenericField,
		TypeParamName:    typeParamName,
	}
}

func (p *Plugin) generateFieldConversion(
	field resolution.Field,
	cppFieldName string,
	data *templateData,
) (forward, backward string) {
	typeRef := field.Type
	pbFieldName := casing.FieldSnake(field.Name)
	pbAccessorName := keywords.Escape(pbFieldName)
	pbSetter := fmt.Sprintf("pb.set_%s", pbAccessorName)

	if p.isFixedSizeUint8Array(typeRef, data.table) {
		return p.generateFixedSizeUint8ArrayConversion(field, data)
	}

	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		return p.generateArrayConversion(field, cppFieldName, data)
	}

	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		return p.generateMapConversion(field, data)
	}

	if resolution.IsPrimitive(typeRef.Name) {
		return p.generatePrimitiveConversion(
			typeRef.Name,
			cppFieldName,
			pbAccessorName,
			pbSetter,
			field.Optional,
			data,
		)
	}

	if typeRef.TypeParam != nil {
		if typeRef.TypeParam.HasDefault() {
			substitutedField := field
			substitutedField.Type = *typeRef.TypeParam.Default
			return p.generateFieldConversion(substitutedField, cppFieldName, data)
		}
		return p.generateTypeParamConversion(field, cppFieldName, pbAccessorName)
	}

	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return fmt.Sprintf("%s(this->%s)", pbSetter, cppFieldName),
			fmt.Sprintf("cpp.%s = pb.%s();", cppFieldName, pbAccessorName)
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		return p.generateStructConversion(
			typeRef,
			resolved,
			field.Optional,
			data,
			cppFieldName,
			pbAccessorName,
		)
	case resolution.UnionForm:
		return p.generateUnionConversion(
			resolved,
			field.Optional,
			data,
			cppFieldName,
			pbAccessorName,
		)
	case resolution.EnumForm:
		return p.generateEnumConversion(
			resolved,
			form,
			cppFieldName,
			pbAccessorName,
			pbSetter,
			data,
		)
	case resolution.DistinctForm:
		if omit.IsSkipped(resolved, "cpp") {
			return p.generateHandWrittenConversion(
				resolved,
				field.Optional,
				cppFieldName,
				pbAccessorName,
				pbSetter,
				data,
			)
		}
		// A generated distinct type converts as its base.
		field.Type = resolution.SubstituteTypeRef(
			form.Base,
			resolution.TypeArgMap(form.TypeParams, typeRef.TypeArgs),
		)
		return p.generateFieldConversion(field, cppFieldName, data)
	case resolution.AliasForm:
		field.Type = resolution.SubstituteTypeRef(
			form.Target,
			resolution.TypeArgMap(form.TypeParams, typeRef.TypeArgs),
		)
		// A C++ alias is a using declaration, so a struct converts through its name.
		if target, ok := field.Type.Resolve(data.table); ok {
			if _, isStruct := target.Form.(resolution.StructForm); isStruct {
				return p.generateStructConversion(
					typeRef,
					target,
					field.Optional,
					data,
					cppFieldName,
					pbAccessorName,
				)
			}
		}
		return p.generateFieldConversion(field, cppFieldName, data)
	default:
		return fmt.Sprintf("%s(this->%s)", pbSetter, cppFieldName),
			fmt.Sprintf("cpp.%s = pb.%s();", cppFieldName, pbAccessorName)
	}
}

func (p *Plugin) generateJSONFieldConversion(
	field resolution.Field,
	cppFieldName,
	pbAccessorName string,
) (forward, backward string) {
	if field.Optional {
		forward = fmt.Sprintf(`if (this->%s.has_value() && !this->%s->is_null()) {
        auto [v, err] = x::json::to_any(*this->%s);
        if (err) return {{}, err};
        *pb.mutable_%s() = v;
    }`, cppFieldName, cppFieldName, cppFieldName, pbAccessorName)
		backward = fmt.Sprintf(`if (pb.has_%s()) {
        auto [v, err] = x::json::from_any(pb.%s());
        if (err) return {{}, err};
        cpp.%s = v;
    }`, pbAccessorName, pbAccessorName, cppFieldName)
	} else {
		forward = fmt.Sprintf(`if (!this->%s.is_null()) {
        auto [v, err] = x::json::to_any(this->%s);
        if (err) return {{}, err};
        *pb.mutable_%s() = v;
    }`, cppFieldName, cppFieldName, pbAccessorName)
		backward = fmt.Sprintf(`auto [v, err] = x::json::from_any(pb.%s());
        if (err) return {{}, err};
        cpp.%s = v;`, pbAccessorName, cppFieldName)
	}
	return forward, backward
}

func (p *Plugin) generatePrimitiveConversion(
	primitive, cppFieldName, pbAccessorName, pbSetter string,
	isOptional bool,
	data *templateData,
) (forward, backward string) {
	switch primitive {
	case "uuid":
		if isOptional {
			forward = fmt.Sprintf(
				"if (this->%s.has_value()) %s(this->%s->to_string())",
				cppFieldName,
				pbSetter,
				cppFieldName,
			)
			backward = fmt.Sprintf(`if (!pb.%s().empty()) {
        auto [v, err] = x::uuid::UUID::parse(pb.%s());
        if (err) return {{}, err};
        cpp.%s = v;
    }`, pbAccessorName, pbAccessorName, cppFieldName)
		} else {
			forward = fmt.Sprintf("%s(this->%s.to_string())", pbSetter, cppFieldName)
			backward = fmt.Sprintf(`{
        auto [v, err] = x::uuid::UUID::parse(pb.%s());
        if (err) return {{}, err};
        cpp.%s = v;
    }`, pbAccessorName, cppFieldName)
		}
		return forward, backward
	case "timestamp":
		return fmt.Sprintf("%s(this->%s.nanoseconds())", pbSetter, cppFieldName),
			fmt.Sprintf(
				"cpp.%s = x::telem::TimeStamp(pb.%s());",
				cppFieldName,
				pbAccessorName,
			)
	case "timespan":
		return fmt.Sprintf("%s(this->%s.nanoseconds())", pbSetter, cppFieldName),
			fmt.Sprintf(
				"cpp.%s = x::telem::TimeSpan(pb.%s());",
				cppFieldName,
				pbAccessorName,
			)
	case "data_type":
		return fmt.Sprintf("%s(this->%s.to_proto())", pbSetter, cppFieldName),
			fmt.Sprintf(
				"cpp.%s = x::telem::DataType::from_proto(pb.%s());",
				cppFieldName,
				pbAccessorName,
			)
	case "record":
		data.AddInternal("x/cpp/json/struct.h")
		if isOptional {
			forward = fmt.Sprintf(`if (this->%s.has_value()) {
        auto [v, err] = x::json::to_struct(*this->%s);
        if (err) return {{}, err};
        *pb.mutable_%s() = v;
    }`, cppFieldName, cppFieldName, pbAccessorName)
			backward = fmt.Sprintf(`if (pb.has_%s()) {
        auto [v, err] = x::json::from_struct(pb.%s());
        if (err) return {{}, err};
        cpp.%s = v;
    }`, pbAccessorName, pbAccessorName, cppFieldName)
		} else {
			forward = fmt.Sprintf(`{
        auto [v, err] = x::json::to_struct(this->%s);
        if (err) return {{}, err};
        *pb.mutable_%s() = v;
    }`, cppFieldName, pbAccessorName)
			backward = fmt.Sprintf(`{
        auto [v, err] = x::json::from_struct(pb.%s());
        if (err) return {{}, err};
        cpp.%s = v;
    }`, pbAccessorName, cppFieldName)
		}
		return forward, backward
	case "any":
		data.AddInternal("x/cpp/json/json.h")
		if isOptional {
			forward = fmt.Sprintf(
				"if (this->%s.has_value()) pb.set_%s((*this->%s).dump())",
				cppFieldName,
				pbAccessorName,
				cppFieldName,
			)
			backward = fmt.Sprintf(`if (pb.has_%s()) {
        cpp.%s = x::json::json::parse(pb.%s(), nullptr, false);
    }`, pbAccessorName, cppFieldName, pbAccessorName)
		} else {
			forward = fmt.Sprintf(
				"pb.set_%s(this->%s.dump())",
				pbAccessorName,
				cppFieldName,
			)
			backward = fmt.Sprintf(
				"cpp.%s = x::json::json::parse(pb.%s(), nullptr, false);",
				cppFieldName,
				pbAccessorName,
			)
		}
		return forward, backward
	case "bytes":
		return fmt.Sprintf(
				"pb.set_%s(this->%s.data(), this->%s.size())",
				pbAccessorName,
				cppFieldName,
				cppFieldName,
			),
			fmt.Sprintf(
				"cpp.%s.assign(pb.%s().begin(), pb.%s().end());",
				cppFieldName,
				pbAccessorName,
				pbAccessorName,
			)
	default:
		if isOptional {
			return fmt.Sprintf(
					"if (this->%s.has_value()) %s(*this->%s)",
					cppFieldName,
					pbSetter,
					cppFieldName,
				),
				fmt.Sprintf(
					"if (pb.has_%s()) cpp.%s = pb.%s();",
					pbAccessorName,
					cppFieldName,
					pbAccessorName,
				)
		}
		return fmt.Sprintf("%s(this->%s)", pbSetter, cppFieldName),
			fmt.Sprintf("cpp.%s = pb.%s();", cppFieldName, pbAccessorName)
	}
}

func (p *Plugin) generateStructConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	isOptional bool,
	data *templateData,
	cppFieldName, pbAccessorName string,
) (forward, backward string) {
	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			data.addConversionIncludes(targetOutputPath)
		}
	}

	cppType := p.typeRefToCppForTranslator(typeRef, data)
	if isOptional {
		forward = fmt.Sprintf(`if (this->%s.has_value()) {
        auto [v, err] = this->%s->to_proto();
        if (err) return {{}, err};
        *pb.mutable_%s() = v;
    }`, cppFieldName, cppFieldName, pbAccessorName)
		backward = fmt.Sprintf(`if (pb.has_%s()) {
        auto [v, err] = %s::from_proto(pb.%s());
        if (err) return {{}, err};
        cpp.%s = v;
    }`, pbAccessorName, cppType, pbAccessorName, cppFieldName)
	} else {
		forward = fmt.Sprintf(`{
        auto [v, err] = this->%s.to_proto();
        if (err) return {{}, err};
        *pb.mutable_%s() = v;
    }`, cppFieldName, pbAccessorName)
		backward = fmt.Sprintf(`{
        auto [v, err] = %s::from_proto(pb.%s());
        if (err) return {{}, err};
        cpp.%s = v;
    }`, cppType, pbAccessorName, cppFieldName)
	}

	return forward, backward
}

func (p *Plugin) typeRefToCppForTranslator(
	typeRef resolution.TypeRef,
	data *templateData,
) string {
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		// For defaulted type params, substitute the default
		if typeRef.TypeParam.HasDefault() {
			return p.typeRefToCppForTranslator(*typeRef.TypeParam.Default, data)
		}
		return typeRef.TypeParam.Name
	}

	if typeRef.Name == "Map" && len(typeRef.TypeArgs) == 2 {
		return fmt.Sprintf(
			"std::unordered_map<%s, %s>",
			p.typeRefToCppForTranslator(typeRef.TypeArgs[0], data),
			p.typeRefToCppForTranslator(typeRef.TypeArgs[1], data),
		)
	}
	if mapping := primitiveMapper.Map(typeRef.Name); mapping.TargetType != "" &&
		mapping.TargetType != "void" {
		return mapping.TargetType
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
			ns := naming.Namespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	}

	return name + p.typeArgsToCpp(
		resolution.TypeParams(resolved.Form),
		typeRef.TypeArgs,
		data,
	)
}

// typeArgsToCpp returns args as a C++ template argument list, leaving out each argument
// bound to a defaulted parameter in params. It returns "" when no argument is left.
func (p *Plugin) typeArgsToCpp(
	params []resolution.TypeParam,
	args []resolution.TypeRef,
	data *templateData,
) string {
	var out []string
	for i, arg := range args {
		if i < len(params) && params[i].HasDefault() {
			continue
		}
		out = append(out, p.typeRefToCppForTranslator(arg, data))
	}
	if len(out) == 0 {
		return ""
	}
	return "<" + strings.Join(out, ", ") + ">"
}

func (p *Plugin) generateTypeParamConversion(
	field resolution.Field,
	cppFieldName, pbAccessorName string,
) (forward, backward string) {
	typeParamName := field.Type.TypeParam.Name
	// Always use JSON serialization for generic type parameters.
	// This ensures compatibility with the Go server which stores details as JSON.
	// Handle monostate specially since it doesn't have to_json()/parse() methods.
	if field.Optional {
		forward = fmt.Sprintf(`if constexpr (!std::is_same_v<%s, std::monostate>) {
        if (this->%s.has_value()) {
            if (const auto j = this->%s->to_json(); !j.is_null()) {
                auto [v, err] = x::json::to_any(j);
                if (err) return {{}, err};
                *pb.mutable_%s() = v;
            }
        }
    }`, typeParamName, cppFieldName, cppFieldName, pbAccessorName)
		backward = fmt.Sprintf(`if (pb.has_%s()) {
        auto [v, err] = x::json::from_any(pb.%s());
        if (err) return {{}, err};
        if constexpr (std::is_same_v<%s, std::monostate>)
            cpp.%s = std::monostate{};
        else
            cpp.%s = %s::parse(x::json::Parser(v));
    }`,
			pbAccessorName,
			pbAccessorName,
			typeParamName,
			cppFieldName,
			cppFieldName,
			typeParamName,
		)
	} else {
		forward = fmt.Sprintf(`if constexpr (!std::is_same_v<%s, std::monostate>) {
        if (const auto j = this->%s.to_json(); !j.is_null()) {
            auto [v, err] = x::json::to_any(j);
            if (err) return {{}, err};
            *pb.mutable_%s() = v;
        }
    }`, typeParamName, cppFieldName, pbAccessorName)
		backward = fmt.Sprintf(`auto [v, err] = x::json::from_any(pb.%s());
        if (err) return {{}, err};
        if constexpr (std::is_same_v<%s, std::monostate>)
            cpp.%s = std::monostate{};
        else
            cpp.%s = %s::parse(x::json::Parser(v));`,
			pbAccessorName,
			typeParamName,
			cppFieldName,
			cppFieldName,
			typeParamName,
		)
	}
	return forward, backward
}

func (p *Plugin) generateEnumConversion(
	resolved resolution.Type,
	form resolution.EnumForm,
	cppFieldName, pbAccessorName, pbSetter string,
	data *templateData,
) (forward, backward string) {
	enumName := resolved.Name

	cppEnumType := domain.GetName(resolved, "cpp")
	funcNamespace := ""
	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			ns := naming.Namespace(targetOutputPath)
			cppEnumType = fmt.Sprintf("::%s::%s", ns, cppEnumType)
			funcNamespace = fmt.Sprintf("::%s::", ns)
		}
	}

	pbOutputPath := enum.FindPBOutputPath(resolved, data.table)
	pbNamespace := naming.PBNamespace(pbOutputPath, resolved.Namespace)

	if form.IsIntEnum {
		forward = fmt.Sprintf(
			"%s(static_cast<%s::%s>(this->%s))",
			pbSetter,
			pbNamespace,
			enumName,
			cppFieldName,
		)
		backward = fmt.Sprintf(
			"cpp.%s = static_cast<%s>(pb.%s());",
			cppFieldName,
			cppEnumType,
			pbAccessorName,
		)
	} else {
		funcName := funcNamespace + casing.TypeSnake(enumName)
		forward = fmt.Sprintf(`{
        auto [v, err] = %s_to_pb(this->%s);
        if (err) return {{}, err};
        %s(v);
    }`, funcName, cppFieldName, pbSetter)
		backward = fmt.Sprintf(`{
        auto [v, err] = %s_from_pb(pb.%s());
        if (err) return {{}, err};
        cpp.%s = v;
    }`, funcName, pbAccessorName, cppFieldName)
	}

	return forward, backward
}

// generateHandWrittenConversion converts a type hand-written in C++ through its
// to_proto and from_proto members.
func (p *Plugin) generateHandWrittenConversion(
	resolved resolution.Type,
	isOptional bool,
	cppFieldName, pbAccessorName, pbSetter string,
	data *templateData,
) (forward, backward string) {
	cppName := domain.GetName(resolved, "cpp")
	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			ns := naming.Namespace(targetOutputPath)
			cppName = fmt.Sprintf("::%s::%s", ns, cppName)
		}
	}
	if isOptional {
		return fmt.Sprintf(
				"if (this->%s.has_value()) %s(this->%s->to_proto())",
				cppFieldName,
				pbSetter,
				cppFieldName,
			),
			fmt.Sprintf(
				"if (pb.has_%s()) cpp.%s = %s::from_proto(pb.%s());",
				pbAccessorName,
				cppFieldName,
				cppName,
				pbAccessorName,
			)
	}
	return fmt.Sprintf("%s(this->%s.to_proto())", pbSetter, cppFieldName),
		fmt.Sprintf(
			"cpp.%s = %s::from_proto(pb.%s());",
			cppFieldName,
			cppName,
			pbAccessorName,
		)
}

func (p *Plugin) generateArrayConversion(
	field resolution.Field,
	cppFieldName string,
	data *templateData,
) (forward, backward string) {
	pbFieldName := casing.FieldSnake(field.Name)
	pbAccessorName := keywords.Escape(pbFieldName)
	typeRef := field.Type

	if len(typeRef.TypeArgs) == 0 {
		return "// TODO: array without type args", "// TODO: array without type args"
	}

	if arrays.IsNested(typeRef, data.table) {
		return p.generateNestedArrayConversion(
			cppFieldName,
			pbAccessorName,
			typeRef,
			data,
		)
	}

	elemType := typeRef.TypeArgs[0]
	return p.generateArrayElementConversion(
		cppFieldName,
		pbAccessorName,
		elemType,
		field.Optional,
		data,
	)
}

// generateArrayElementConversion renders the proto conversion for an array field.
// A nullable array maps to a wrapper message (a list message whose repeated field
// is named values) so absence is distinguishable from emptiness; the generated
// code unwraps the std::optional and gates the wrapper on presence. A required
// array maps to a plain repeated field.
func (p *Plugin) generateArrayElementConversion(
	cppFieldName, pbAccessorName string,
	elemType resolution.TypeRef,
	isOptional bool,
	data *templateData,
) (forward, backward string) {
	if !resolution.IsPrimitive(elemType.Name) {
		if resolved, _, ok := resolution.Dealias(elemType, data.table); ok {
			if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
				if resolved.Namespace != data.rawNs {
					targetOutputPath := output.GetPath(resolved, "cpp")
					if targetOutputPath != "" {
						data.addConversionIncludes(targetOutputPath)
					}
				}
				elemCppType := p.typeRefToCppForTranslator(elemType, data)
				if isOptional {
					forward = fmt.Sprintf(`if (this->%s.has_value()) {
        auto* wrapper = pb.mutable_%s();
        for (const auto& item : *this->%s) {
            auto [v, err] = item.to_proto();
            if (err) return {{}, err};
            *wrapper->add_values() = v;
        }
    }`, cppFieldName, pbAccessorName, cppFieldName)
					backward = fmt.Sprintf(`if (pb.has_%s()) {
        cpp.%s.emplace();
        if (auto err = x::pb::from_proto_repeated<%s>(*cpp.%s, pb.%s().values())) return {{}, err};
    }`, pbAccessorName, cppFieldName, elemCppType, cppFieldName, pbAccessorName)
					return forward, backward
				}
				forward = fmt.Sprintf(`for (const auto& item : this->%s) {
        auto [v, err] = item.to_proto();
        if (err) return {{}, err};
        *pb.add_%s() = v;
    }`, cppFieldName, pbAccessorName)
				backward = fmt.Sprintf(
					"if (auto err = x::pb::from_proto_repeated<%s>(cpp.%s, pb.%s())) return {{}, err};",
					elemCppType,
					cppFieldName,
					pbAccessorName,
				)
				return forward, backward
			}
			if _, isUnion := resolved.Form.(resolution.UnionForm); isUnion {
				return p.generateUnionArrayConversion(
					resolved,
					cppFieldName,
					pbAccessorName,
					isOptional,
					data,
				)
			}
		}
	}

	if toPB, fromPB, ok := p.elementConversion(elemType, data); ok {
		if isOptional {
			forward = fmt.Sprintf(`if (this->%s.has_value()) {
        auto* wrapper = pb.mutable_%s();
        for (const auto& item : *this->%s) {
            %s
        }
    }`, cppFieldName, pbAccessorName, cppFieldName, toPB("wrapper->add_values"))
			backward = fmt.Sprintf(`if (pb.has_%s()) {
        cpp.%s.emplace();
        for (const auto& item : pb.%s().values()) {
            %s
        }
    }`, pbAccessorName, cppFieldName, pbAccessorName, fromPB("cpp."+cppFieldName+"->"))
			return forward, backward
		}
		forward = fmt.Sprintf(`for (const auto& item : this->%s) {
        %s
    }`, cppFieldName, toPB("pb.add_"+pbAccessorName))
		backward = fmt.Sprintf(`for (const auto& item : pb.%s()) {
        %s
    }`, pbAccessorName, fromPB("cpp."+cppFieldName+"."))
		return forward, backward
	}

	if isOptional {
		forward = fmt.Sprintf(`if (this->%s.has_value()) {
        auto* wrapper = pb.mutable_%s();
        for (const auto& item : *this->%s) wrapper->add_values(item);
    }`, cppFieldName, pbAccessorName, cppFieldName)
		backward = fmt.Sprintf(`if (pb.has_%s()) {
        cpp.%s.emplace();
        for (const auto& item : pb.%s().values()) cpp.%s->push_back(item);
    }`, pbAccessorName, cppFieldName, pbAccessorName, cppFieldName)
		return forward, backward
	}

	forward = fmt.Sprintf(
		"for (const auto& item : this->%s) pb.add_%s(item)",
		cppFieldName,
		pbAccessorName,
	)
	backward = fmt.Sprintf(
		"for (const auto& item : pb.%s()) cpp.%s.push_back(item);",
		pbAccessorName,
		cppFieldName,
	)

	return forward, backward
}

// elementConversion returns the per-item statements that translate an array element
// that proto cannot hold as is: a uuid (directly or through aliases and distinct types)
// or an enum. toPB takes the proto adder and fromPB takes the C++ container prefix
// ("cpp.field." or "cpp.field->"). Both read the loop variable item.
func (p *Plugin) elementConversion(
	elemType resolution.TypeRef,
	data *templateData,
) (toPB, fromPB func(string) string, ok bool) {
	if key.ResolvePrimitive(elemType, data.table) == "uuid" {
		toPB = func(add string) string {
			return fmt.Sprintf("%s(item.to_string());", add)
		}
		fromPB = func(dst string) string {
			return fmt.Sprintf(`auto [v, err] = x::uuid::UUID::parse(item);
            if (err) return {{}, err};
            %spush_back(v);`, dst)
		}
		return toPB, fromPB, true
	}
	resolved, _, ok := resolution.Dealias(elemType, data.table)
	if !ok {
		return nil, nil, false
	}
	form, ok := resolved.Form.(resolution.EnumForm)
	if !ok {
		return nil, nil, false
	}
	cppEnumType := domain.GetName(resolved, "cpp")
	funcNamespace := ""
	if resolved.Namespace != data.rawNs {
		if target := output.GetPath(resolved, "cpp"); target != "" {
			data.addConversionIncludes(target)
			ns := naming.Namespace(target)
			cppEnumType = fmt.Sprintf("::%s::%s", ns, cppEnumType)
			funcNamespace = fmt.Sprintf("::%s::", ns)
		}
	}
	pbEnumType := fmt.Sprintf(
		"%s::%s",
		naming.PBNamespace(
			enum.FindPBOutputPath(resolved, data.table),
			resolved.Namespace,
		),
		resolved.Name,
	)
	if form.IsIntEnum {
		toPB = func(add string) string {
			return fmt.Sprintf("%s(static_cast<%s>(item));", add, pbEnumType)
		}
		fromPB = func(dst string) string {
			return fmt.Sprintf("%spush_back(static_cast<%s>(item));", dst, cppEnumType)
		}
		return toPB, fromPB, true
	}
	funcName := funcNamespace + casing.TypeSnake(resolved.Name)
	toPB = func(add string) string {
		return fmt.Sprintf(`auto [v, err] = %s_to_pb(item);
            if (err) return {{}, err};
            %s(v);`, funcName, add)
	}
	fromPB = func(dst string) string {
		return fmt.Sprintf(`auto [v, err] = %s_from_pb(static_cast<%s>(item));
            if (err) return {{}, err};
            %spush_back(v);`, funcName, pbEnumType, dst)
	}
	return toPB, fromPB, true
}

func (p *Plugin) generateUnionArrayConversion(
	resolved resolution.Type,
	cppFieldName, pbAccessorName string,
	isOptional bool,
	data *templateData,
) (forward, backward string) {
	toProto, fromProto := p.unionTranslatorNames(resolved, data)
	if isOptional {
		forward = fmt.Sprintf(`if (this->%s.has_value()) {
        auto* wrapper = pb.mutable_%s();
        for (const auto& item : *this->%s) {
            auto [v, err] = %s(item);
            if (err) return {{}, err};
            *wrapper->add_values() = v;
        }
    }`, cppFieldName, pbAccessorName, cppFieldName, toProto)
		backward = fmt.Sprintf(`if (pb.has_%s()) {
        cpp.%s.emplace();
        for (const auto& item : pb.%s().values()) {
            auto [v, err] = %s(item);
            if (err) return {{}, err};
            cpp.%s->push_back(v);
        }
    }`, pbAccessorName, cppFieldName, pbAccessorName, fromProto, cppFieldName)
		return forward, backward
	}
	forward = fmt.Sprintf(`for (const auto& item : this->%s) {
        auto [v, err] = %s(item);
        if (err) return {{}, err};
        *pb.add_%s() = v;
    }`, cppFieldName, toProto, pbAccessorName)
	backward = fmt.Sprintf(`for (const auto& item : pb.%s()) {
        auto [v, err] = %s(item);
        if (err) return {{}, err};
        cpp.%s.push_back(v);
    }`, pbAccessorName, fromProto, cppFieldName)
	return forward, backward
}

func (p *Plugin) generateMapConversion(
	field resolution.Field,
	data *templateData,
) (forward, backward string) {
	fieldName := casing.FieldSnake(field.Name)
	accessorName := keywords.Escape(fieldName)
	typeRef := field.Type

	if len(typeRef.TypeArgs) < 2 {
		return "// TODO: map without enough type args", "// TODO: map without enough type args"
	}

	valueType := typeRef.TypeArgs[1]
	// An alias value converts as the type it stands for.
	value, valueRef, dealiased := resolution.Dealias(valueType, data.table)

	if dealiased && valueRef.Name == "record" {
		data.AddInternal("x/cpp/json/struct.h")
		forward = fmt.Sprintf(`for (const auto& [k, v] : this->%s) {
        auto [pb_v, err] = x::json::to_struct(v);
        if (err) return {{}, err};
        (*pb.mutable_%s())[k] = pb_v;
    }`, fieldName, accessorName)
		backward = fmt.Sprintf(`for (const auto& [k, v] : pb.%s()) {
        auto [cpp_v, err] = x::json::from_struct(v);
        if (err) return {{}, err};
        cpp.%s[k] = cpp_v;
    }`, accessorName, fieldName)
		return forward, backward
	}

	if dealiased {
		if _, isStruct := value.Form.(resolution.StructForm); isStruct {
			if value.Namespace != data.rawNs {
				targetOutputPath := output.GetPath(value, "cpp")
				if targetOutputPath != "" {
					data.addConversionIncludes(targetOutputPath)
				}
			}
			valueCppType := p.typeRefToCppForTranslator(valueType, data)
			forward = fmt.Sprintf(`for (const auto& [k, v] : this->%s) {
        auto [pb_v, err] = v.to_proto();
        if (err) return {{}, err};
        (*pb.mutable_%s())[k] = pb_v;
    }`, fieldName, accessorName)
			backward = fmt.Sprintf(`for (const auto& [k, v] : pb.%s()) {
        auto [cpp_v, err] = %s::from_proto(v);
        if (err) return {{}, err};
        cpp.%s[k] = cpp_v;
    }`, accessorName, valueCppType, fieldName)
			return forward, backward
		}
	}

	forward = fmt.Sprintf(
		"for (const auto& [k, v] : this->%s) (*pb.mutable_%s())[k] = v",
		fieldName,
		accessorName,
	)
	backward = fmt.Sprintf(
		"for (const auto& [k, v] : pb.%s()) cpp.%s[k] = v;",
		accessorName,
		fieldName,
	)

	return forward, backward
}

func (p *Plugin) isFixedSizeUint8Array(
	typeRef resolution.TypeRef,
	table *resolution.Table,
) bool {
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return false
	}

	form, ok := resolved.Form.(resolution.DistinctForm)
	if !ok {
		return false
	}
	base, ok := arrays.Wrapped(form, table)
	return ok && base.ArraySize != nil && base.TypeArgs[0].Name == "uint8"
}

func (p *Plugin) generateFixedSizeUint8ArrayConversion(
	field resolution.Field,
	data *templateData,
) (forward, backward string) {
	fieldName := casing.FieldSnake(field.Name)
	accessorName := keywords.Escape(fieldName)

	forward = fmt.Sprintf(
		"pb.set_%s(this->%s.data(), this->%s.size())",
		accessorName,
		fieldName,
		fieldName,
	)
	backward = fmt.Sprintf(
		"std::copy(pb.%s().begin(), pb.%s().end(), cpp.%s.begin());",
		accessorName,
		accessorName,
		fieldName,
	)
	data.AddSystem("algorithm")

	return forward, backward
}

func (p *Plugin) generateNestedArrayConversion(
	cppFieldName, pbAccessorName string,
	typeRef resolution.TypeRef,
	data *templateData,
) (forward, backward string) {
	// Inner element type (e.g., Member for []Members where Members = []Member).
	// When it's a struct, delegate to its to_proto/from_proto helpers so the
	// nested conversion type-checks and propagates errors correctly. The
	// primitive-inner path preserves the original `add_values(v)` form.
	if elemType, ok := arrays.ElementType(typeRef, data.table); ok {
		if innerElem, ok := arrays.ElementType(elemType, data.table); ok {
			if innerResolved, _, ok := resolution.Dealias(innerElem, data.table); ok {
				if _, isStruct := innerResolved.Form.(resolution.StructForm); isStruct {
					if innerResolved.Namespace != data.rawNs {
						targetOutputPath := output.GetPath(innerResolved, "cpp")
						if targetOutputPath != "" {
							data.addConversionIncludes(targetOutputPath)
						}
					}
					innerCppType := p.typeRefToCppForTranslator(innerElem, data)
					forward = fmt.Sprintf(`for (const auto& item : this->%s) {
        auto* wrapper = pb.add_%s();
        for (const auto& v : item) {
            auto [v_pb, err] = v.to_proto();
            if (err) return {{}, err};
            *wrapper->add_values() = v_pb;
        }
    }`, cppFieldName, pbAccessorName)
					backward = fmt.Sprintf(`for (const auto& wrapper : pb.%s()) {
        std::vector<%s> inner;
        if (auto err = x::pb::from_proto_repeated<%s>(inner, wrapper.values())) return {{}, err};
        cpp.%s.push_back(std::move(inner));
    }`, pbAccessorName, innerCppType, innerCppType, cppFieldName)
					return forward, backward
				}
			}
		}
	}

	forward = fmt.Sprintf(`for (const auto& item : this->%s) {
        auto* wrapper = pb.add_%s();
        for (const auto& v : item) wrapper->add_values(v);
    }`, cppFieldName, pbAccessorName)

	backward = fmt.Sprintf(`for (const auto& wrapper : pb.%s())
        cpp.%s.push_back({wrapper.values().begin(), wrapper.values().end()});`, pbAccessorName, cppFieldName)

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

	pbOutputPath := enum.FindPBOutputPath(e, data.table)
	pbNamespace := naming.PBNamespace(pbOutputPath, e.Namespace)

	values := make([]enumValueTranslatorData, 0, len(form.Values))
	for _, v := range form.Values {
		cppValueName := fmt.Sprintf(
			"%s_%s",
			naming.ScreamingSnake(e.Name),
			naming.ScreamingSnake(v.Name),
		)
		pbValueName := fmt.Sprintf(
			"%s_%s",
			naming.ScreamingSnake(e.Name),
			naming.ScreamingSnake(v.Name),
		)

		values = append(values, enumValueTranslatorData{
			CppValue: cppValueName,
			PBValue:  pbValueName,
		})
	}

	return &enumTranslatorData{
		Name:        e.Name,
		FuncName:    casing.TypeSnake(e.Name),
		PBNamespace: pbNamespace,
		Values:      values,
	}
}

func (p *Plugin) processArrayWrapperForTranslation(
	dt resolution.Type,
	form resolution.DistinctForm,
	data *templateData,
) *arrayWrapperTranslatorData {
	base, ok := arrays.Wrapped(form, data.table)
	if !ok {
		return nil
	}

	cppName := domain.GetName(dt, "cpp")
	pbName := naming.PBName(dt)
	pbOutputPath := output.GetPBPath(dt)
	pbNamespace := naming.PBNamespace(pbOutputPath, dt.Namespace)

	elemType := base.TypeArgs[0]
	elemCppType := p.typeRefToCppForTranslator(elemType, data)

	elementNeedsConvert := false
	if !resolution.IsPrimitive(elemType.Name) {
		if resolved, _, ok := resolution.Dealias(elemType, data.table); ok {
			_, elementNeedsConvert = resolved.Form.(resolution.StructForm)
		}
	}

	return &arrayWrapperTranslatorData{
		CppName:             cppName,
		PBName:              pbName,
		PBNamespace:         pbNamespace,
		ElementType:         elemCppType,
		ElementNeedsConvert: elementNeedsConvert,
	}
}

func deriveProtoInclude(pbOutputPath, namespace string) string {
	if pbOutputPath == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s.pb.h", pbOutputPath, namespace)
}

type templateData struct {
	processedEnums set.Set[string]
	*includes.Manager
	table            *resolution.Table
	jsonPaths        set.Set[string]
	protoPaths       set.Set[string]
	Namespace        string
	rawNs            string
	Translators      []translatorData
	UnionTranslators []unionTranslatorData
	EnumTranslators  []enumTranslatorData
	ArrayWrappers    []arrayWrapperTranslatorData
}

// unionTranslatorData is the template view of the free functions that move a
// discriminated union between its std::variant alias and its protobuf wrapper.
type unionTranslatorData struct {
	// CppName is the std::variant alias (e.g. "Target").
	CppName string
	// SnakeName names the from_proto function (e.g. target_from_proto).
	SnakeName string
	// PBType is the qualified protobuf wrapper message.
	PBType string
	// Bases are the union's extends bases, nested as message fields on the wrapper.
	Bases []unionBaseTranslatorData
	// Variants lists every variant in declaration order.
	Variants []unionVariantTranslatorData
}

type unionBaseTranslatorData struct {
	// CppType is the qualified base struct the variant structs inherit.
	CppType string
	// PBAccessor is the wrapper's message field for the base.
	PBAccessor string
}

type unionVariantTranslatorData struct {
	// TypeName is the variant struct (e.g. "ExitTarget").
	TypeName string
	// PBAccessor is the oneof member field (e.g. "exit").
	PBAccessor string
	// CaseName is the protoc oneof case without its k prefix (e.g. "Exit").
	CaseName string
}

// jsonInclude returns the header to include for a reference to types generated at
// outputPath.
func (d *templateData) jsonInclude(outputPath string) string {
	return json.IncludeFor(d.jsonPaths, outputPath)
}

// addConversionIncludes registers the headers a cross-package conversion needs: the
// target's proto.gen.h when the plugin emits one (otherwise the conversions are
// hand-written next to the type) and its JSON header.
func (d *templateData) addConversionIncludes(targetOutputPath string) {
	if d.protoPaths.Contains(targetOutputPath) {
		d.AddInternal(targetOutputPath + "/proto.gen.h")
	}
	d.AddInternal(d.jsonInclude(targetOutputPath))
}

type arrayWrapperTranslatorData struct {
	CppName             string
	PBName              string
	PBNamespace         string
	ElementType         string
	ElementNeedsConvert bool
}

type translatorData struct {
	CppName        string
	PBName         string
	PBNamespace    string
	TypeParamNames string
	Fields         []fieldTranslatorData
	TypeParams     []typeParamData
	ParentTypes    []parentTypeData
	AllFields      []fieldTranslatorData
	IsGeneric      bool
	HasExtends     bool
}

type parentTypeData struct {
	QualifiedName string
}

type typeParamData struct {
	Name string
}

type fieldTranslatorData struct {
	CppName          string
	PBName           string
	ForwardExpr      string
	BackwardExpr     string
	ForwardJSONExpr  string
	BackwardJSONExpr string
	TypeParamName    string
	IsGenericField   bool
}

type enumTranslatorData struct {
	Name        string
	FuncName    string
	PBNamespace string
	Values      []enumValueTranslatorData
}

type enumValueTranslatorData struct {
	CppValue string
	PBValue  string
}
