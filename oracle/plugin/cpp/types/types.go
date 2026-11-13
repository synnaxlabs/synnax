// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/domain/resource"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/cpp/internal/defaults"
	"github.com/synnaxlabs/oracle/plugin/cpp/internal/includes"
	"github.com/synnaxlabs/oracle/plugin/cpp/keywords"
	cppnaming "github.com/synnaxlabs/oracle/plugin/cpp/naming"
	cppprimitives "github.com/synnaxlabs/oracle/plugin/cpp/primitives"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/internal/arrays"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// primitiveMapper is the C++-specific primitive type mapper.
var primitiveMapper = cppprimitives.Mapper()

type Plugin struct{ Options Options }

type Options struct {
	FileNamePattern string
}

func DefaultOptions() Options {
	return Options{
		FileNamePattern: "types.gen.h",
	}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string { return "cpp/types" }

func (p *Plugin) Domains() []string { return []string{"cpp"} }

func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	req = req.OmitFields("cpp")
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	typeDefCollector, err := framework.CollectDistinct("cpp", req)
	if err != nil {
		return nil, err
	}

	structCollector, err := framework.CollectStructs("cpp", req)
	if err != nil {
		return nil, err
	}

	aliasCollector, err := framework.CollectAliases("cpp", req)
	if err != nil {
		return nil, err
	}

	var combinedOrder []string
	combinedOrder = append(combinedOrder, structCollector.Paths()...)
	for _, path := range aliasCollector.Paths() {
		if !structCollector.Has(path) && !typeDefCollector.Has(path) {
			combinedOrder = append(combinedOrder, path)
		}
	}

	enumOutputPaths := make(map[string][]resolution.Type)
	for _, e := range req.Resolutions.EnumTypes() {
		if omit.IsSkipped(e, "cpp") {
			continue
		}
		enumPath := enum.FindOutputPath(e, req.Resolutions, "cpp")
		if enumPath == "" {
			continue
		}
		if !structCollector.Has(enumPath) && !aliasCollector.Has(enumPath) &&
			!typeDefCollector.Has(enumPath) {
			enumOutputPaths[enumPath] = append(enumOutputPaths[enumPath], e)
		}
	}
	for path := range enumOutputPaths {
		found := slices.Contains(combinedOrder, path)
		if !found {
			combinedOrder = append(combinedOrder, path)
		}
	}

	for _, outputPath := range combinedOrder {
		structs := structCollector.Get(outputPath)
		enums := enum.CollectReferenced(structs, req.Resolutions)

		if len(structs) > 0 {
			namespace := structs[0].Namespace
			enums = framework.MergeTypes(
				enums,
				enum.CollectNamespaceEnums(
					namespace,
					outputPath,
					req.Resolutions,
					"cpp",
					nil,
				),
			)
		} else {
			if standaloneEnums, ok := enumOutputPaths[outputPath]; ok {
				enums = framework.MergeTypes(enums, standaloneEnums)
			} else {
				aliases := aliasCollector.Get(outputPath)
				var namespace string
				if len(aliases) > 0 {
					namespace = aliases[0].Namespace
				}
				if namespace != "" {
					enums = framework.MergeTypes(
						enums,
						enum.CollectNamespaceEnums(
							namespace,
							outputPath,
							req.Resolutions,
							"cpp",
							nil,
						),
					)
				}
			}
		}

		var typeDefs []resolution.Type
		if typeDefCollector.Has(outputPath) {
			typeDefs = typeDefCollector.Remove(outputPath)
		}
		aliases := aliasCollector.Get(outputPath)
		content, err := p.generateFile(
			outputPath,
			structs,
			enums,
			typeDefs,
			aliases,
			req.Resolutions,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
			Content: content,
		})
	}

	err = typeDefCollector.ForEach(
		func(outputPath string, typeDefs []resolution.Type) error {
			var enums []resolution.Type
			if len(typeDefs) > 0 {
				namespace := typeDefs[0].Namespace
				enums = enum.CollectNamespaceEnums(
					namespace,
					outputPath,
					req.Resolutions,
					"cpp",
					nil,
				)
			}
			content, err := p.generateFile(
				outputPath,
				nil,
				enums,
				typeDefs,
				nil,
				req.Resolutions,
			)
			if err != nil {
				return errors.Wrapf(err, "failed to generate %s", outputPath)
			}
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
				Content: content,
			})
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (p *Plugin) generateFile(
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	typeDefs []resolution.Type,
	aliases []resolution.Type,
	table *resolution.Table,
) ([]byte, error) {
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	} else if len(typeDefs) > 0 {
		namespace = typeDefs[0].Namespace
	} else if len(aliases) > 0 {
		namespace = aliases[0].Namespace
	} else if len(enums) > 0 {
		namespace = enums[0].Namespace
	}

	data := &templateData{
		OutputPath:  outputPath,
		Namespace:   cppnaming.Namespace(outputPath),
		Enums:       make([]enumData, 0, len(enums)),
		SortedDecls: make([]sortedDeclData, 0),
		Manager:     includes.NewManager(),
		table:       table,
		rawNs:       namespace,
	}

	declaredNames := make(set.Set[string])

	for _, e := range enums {
		if e.Namespace == namespace && !omit.IsSkipped(e, "cpp") {
			data.Enums = append(data.Enums, p.processEnum(e))
		}
	}

	data.Resource = p.extractResource(structs, table, data)

	var combinedTypes []resolution.Type
	combinedTypes = append(combinedTypes, typeDefs...)
	combinedTypes = append(combinedTypes, aliases...)
	combinedTypes = append(combinedTypes, structs...)

	if namespace != "" {
		allNamespaceTypes := table.TypesInNamespace(namespace)
		existingQNames := make(set.Set[string])
		for _, t := range combinedTypes {
			existingQNames.Add(t.QualifiedName)
		}
		for _, t := range allNamespaceTypes {
			// Synthetic inline variant payloads flatten into their variant
			// structs; they have no standalone C++ type.
			if existingQNames.Contains(t.QualifiedName) || t.Synthetic {
				continue
			}
			// Types routed to a different file by a per-type @cpp output
			// override belong to that file, not this one.
			if p := output.GetPath(t, "cpp"); p != "" && p != outputPath {
				continue
			}
			combinedTypes = append(combinedTypes, t)
		}
	}

	allSortedTypes := table.TopologicalSort(combinedTypes)

	var sortedTypes []resolution.Type
	for _, typ := range allSortedTypes {
		if !omit.IsSkipped(typ, "cpp") {
			sortedTypes = append(sortedTypes, typ)
		}
	}

	for _, typ := range sortedTypes {
		if form, ok := typ.Form.(resolution.StructForm); ok && !form.IsGeneric() {
			name := domain.GetName(typ, "cpp")
			data.ForwardDecls = append(data.ForwardDecls, name)
		}
	}

	for _, typ := range sortedTypes {
		switch typ.Form.(type) {
		case resolution.DistinctForm:
			tdd := p.processTypeDef(typ, data)
			if !declaredNames.Contains(tdd.Name) {
				declaredNames.Add(tdd.Name)
				data.SortedDecls = append(data.SortedDecls, sortedDeclData{
					IsTypeDef: true,
					TypeDef:   tdd,
				})
			}
		case resolution.AliasForm:
			aliasData := p.processAlias(typ, data)
			data.SortedDecls = append(data.SortedDecls, sortedDeclData{
				IsAlias: true,
				Alias:   aliasData,
			})
		case resolution.StructForm:
			structData := p.processStruct(typ, data)
			data.SortedDecls = append(data.SortedDecls, sortedDeclData{
				IsStruct: true,
				Struct:   structData,
			})
		case resolution.UnionForm:
			variants, ud := p.processUnion(typ, data)
			for _, v := range variants {
				data.SortedDecls = append(data.SortedDecls, sortedDeclData{
					IsStruct: true,
					Struct:   v,
				})
			}
			data.SortedDecls = append(data.SortedDecls, sortedDeclData{
				IsUnion: true,
				Union:   ud,
			})
		}
	}

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Plugin) processEnum(e resolution.Type) enumData {
	form, ok := e.Form.(resolution.EnumForm)
	if !ok {
		return enumData{Name: e.Name}
	}
	values := make([]enumValueData, 0, len(form.Values))
	for _, v := range form.Values {
		// Use snake_case for string enum constants, PascalCase for int enums
		name := lo.PascalCase(v.Name)
		if !form.IsIntEnum {
			name = casing.FieldSnake(v.Name)
		}
		values = append(values, enumValueData{
			Name:     name,
			Value:    v.StringValue(),
			IntValue: v.IntValue(),
		})
	}
	return enumData{
		Name:      e.Name,
		Values:    values,
		IsIntEnum: form.IsIntEnum,
	}
}

func (p *Plugin) processTypeDef(td resolution.Type, data *templateData) typeDefData {
	form, ok := td.Form.(resolution.DistinctForm)
	if !ok {
		return typeDefData{Name: td.Name, CppType: "void"}
	}
	name := domain.GetName(td, "cpp")

	tdd := typeDefData{
		Name:    name,
		CppType: p.typeRefToCpp(form.Base, data),
		Doc:     doc.Get(td.Domains),
	}

	if base, ok := arrays.Wrapped(form, data.table); ok {
		tdd.IsArrayWrapper = true
		elemTypeRef := base.TypeArgs[0]
		tdd.ElementType = p.typeRefToCpp(elemTypeRef, data)
		tdd.ElementIsPrimitive = resolution.IsPrimitive(elemTypeRef.Name)
		if base.ArraySize != nil {
			tdd.IsFixedSizeArray = true
			tdd.ArraySize = *base.ArraySize
			data.AddSystem("array")
		} else {
			data.AddSystem("vector")
		}

		if cppDomain, ok := td.Domains["cpp"]; ok {
			if methodsExpr, found := cppDomain.Expressions.Find("methods"); found {
				for _, v := range methodsExpr.Values {
					if v.StringValue != "" {
						tdd.Methods = append(tdd.Methods, v.StringValue)
					}
				}
			}
			if includesExpr, found := cppDomain.Expressions.Find("includes"); found {
				for _, v := range includesExpr.Values {
					if v.StringValue != "" {
						data.AddInternal(v.StringValue)
					}
				}
			}
			if sysIncludesExpr, found := cppDomain.Expressions.Find(
				"system_includes",
			); found {
				for _, v := range sysIncludesExpr.Values {
					if v.StringValue != "" {
						data.AddSystem(v.StringValue)
					}
				}
			}
		}

		if hasPBFlag(td) && !omit.IsSkipped(td, "pb") && hasExplicitPBName(td) {
			pbOutputPath := output.GetPBPath(td)
			if pbOutputPath != "" {
				pbName := cppnaming.PBName(td)
				pbNamespace := cppnaming.PBNamespace(pbOutputPath, td.Namespace)
				tdd.HasProto = true
				tdd.ProtoType = fmt.Sprintf("%s::%s", pbNamespace, pbName)
				tdd.ProtoNamespace = pbNamespace
				tdd.ProtoClass = pbName
				data.AddSystem("utility")
				data.AddInternal("x/cpp/errors/errors.h")
				protoInclude := fmt.Sprintf("%s/%s.pb.h", pbOutputPath, td.Namespace)
				data.AddInternal(protoInclude)
			}
		}

		data.AddInternal("x/cpp/json/json.h")
	}

	return tdd
}

func (p *Plugin) processAlias(alias resolution.Type, data *templateData) aliasData {
	form, ok := alias.Form.(resolution.AliasForm)
	if !ok {
		return aliasData{Name: alias.Name, Target: "void"}
	}

	name := domain.GetName(alias, "cpp")
	target := p.aliasTargetToCpp(form.Target, data)

	var typeParams []string
	for _, tp := range form.TypeParams {
		if tp.HasDefault() {
			continue
		}
		typeParams = append(typeParams, tp.Name)
	}

	return aliasData{
		Name:       name,
		Target:     target,
		IsGeneric:  len(typeParams) > 0,
		TypeParams: typeParams,
	}
}

func (p *Plugin) aliasTargetToCpp(
	typeRef resolution.TypeRef,
	data *templateData,
) string {
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		if typeRef.TypeParam.HasDefault() {
			return p.aliasTargetToCpp(*typeRef.TypeParam.Default, data)
		}
		return typeRef.TypeParam.Name
	}

	if typeRef.Name == "Array" {
		data.AddSystem("vector")
		elementType := "void"
		if len(typeRef.TypeArgs) > 0 {
			elementType = p.aliasTargetToCpp(typeRef.TypeArgs[0], data)
		}
		return fmt.Sprintf("std::vector<%s>", elementType)
	}

	if typeRef.Name == "Map" {
		data.AddSystem("unordered_map")
		keyType := "std::string"
		valueType := "void"
		if len(typeRef.TypeArgs) > 0 {
			keyType = p.aliasTargetToCpp(typeRef.TypeArgs[0], data)
		}
		if len(typeRef.TypeArgs) > 1 {
			valueType = p.aliasTargetToCpp(typeRef.TypeArgs[1], data)
		}
		return fmt.Sprintf("std::unordered_map<%s, %s>", keyType, valueType)
	}

	if resolution.IsPrimitive(typeRef.Name) {
		return p.primitiveToCpp(typeRef.Name, data)
	}

	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return typeRef.Name
	}
	// A string enum declares no C++ type, only its member constants.
	if form, ok := resolved.Form.(resolution.EnumForm); ok && !form.IsIntEnum {
		return p.resolveEnumType(resolved, form, data)
	}

	name := resolved.Name
	isOmitted := omit.IsSkipped(resolved, "cpp")
	targetOutputPath := output.GetPath(resolved, "cpp")

	var cppInclude string
	if cppDomain, ok := resolved.Domains["cpp"]; ok {
		for _, expr := range cppDomain.Expressions {
			switch expr.Name {
			case "include":
				if len(expr.Values) > 0 {
					cppInclude = expr.Values[0].StringValue
				}
			case "name":
				if len(expr.Values) > 0 {
					name = expr.Values[0].StringValue
				}
			}
		}
	}

	if resolved.Namespace != data.rawNs {
		if isOmitted || targetOutputPath == "" {
			if cppInclude != "" {
				data.AddInternal(cppInclude)
			}
			if targetOutputPath != "" {
				ns := cppnaming.Namespace(targetOutputPath)
				name = fmt.Sprintf("::%s::%s", ns, name)
			} else if resolved.Namespace != "" {
				name = fmt.Sprintf("::%s::%s", resolved.Namespace, name)
			}
		} else {
			includePath := cppInclude
			if includePath == "" {
				includePath = fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			}
			data.AddInternal(includePath)
			ns := cppnaming.Namespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	} else {
		addLocalHandInclude(resolved, data)
	}

	if len(typeRef.TypeArgs) == 0 {
		if isCppTemplateWithAllDefaults(resolved) {
			return name + "<>"
		}
		return name
	}

	var args []string
	if form, ok := resolved.Form.(resolution.StructForm); ok {
		for i, arg := range typeRef.TypeArgs {
			if i < len(form.TypeParams) && form.TypeParams[i].HasDefault() {
				continue
			}
			args = append(args, p.aliasTargetToCpp(arg, data))
		}
	} else {
		for _, arg := range typeRef.TypeArgs {
			args = append(args, p.aliasTargetToCpp(arg, data))
		}
	}

	if len(args) == 0 {
		if isCppTemplateWithAllDefaults(resolved) {
			return name + "<>"
		}
		return name
	}
	return fmt.Sprintf("%s<%s>", name, strings.Join(args, ", "))
}

func (p *Plugin) processStruct(entry resolution.Type, data *templateData) structData {
	form, ok := entry.Form.(resolution.StructForm)
	if !ok {
		return structData{Name: entry.Name}
	}

	name := domain.GetName(entry, "cpp")
	aliasForm, isAlias := entry.Form.(resolution.AliasForm)

	sd := structData{
		Name:    name,
		Doc:     doc.Get(entry.Domains),
		Fields:  make([]fieldData, 0, len(form.Fields)),
		IsAlias: isAlias,
	}

	for _, tp := range form.TypeParams {
		if tp.HasDefault() {
			continue
		}
		sd.TypeParams = append(sd.TypeParams, p.processTypeParam(tp, data))
	}
	sd.IsGeneric = len(sd.TypeParams) > 0

	if isAlias {
		sd.AliasOf = p.typeRefToCpp(aliasForm.Target, data)
		return sd
	}

	if resolver.CanUseInheritance(form, "cpp", data.table) {
		sd.HasExtends = true
		for _, extendsRef := range form.Extends {
			parent, ok := extendsRef.Resolve(data.table)
			if !ok {
				continue
			}
			qualifiedName := p.resolveExtendsType(extendsRef, parent, data)
			sd.ExtendsTypes = append(sd.ExtendsTypes, qualifiedName)
		}
		// A field that only restates an inherited default keeps the base's member.
		// C++ cannot replace a base member's initializer, so the new default moves
		// into a generated constructor; declaring the member again would hide the
		// base's, which every reference through a base type would still see.
		defaultOnly := resolver.DefaultOnlyOverrides(
			form.Extends, form.Fields, data.table,
		)
		for _, field := range form.Fields {
			fd := p.processField(field, entry, data)
			if !defaultOnly.Contains(field.Name) {
				sd.Fields = append(sd.Fields, fd)
				continue
			}
			if fd.DefaultValue == "" {
				continue
			}
			sd.InheritedDefaults = append(
				sd.InheritedDefaults,
				inheritedDefaultData{Name: fd.Name, Value: fd.DefaultValue},
			)
		}
	} else {
		for _, field := range resolution.UnifiedFields(entry, data.table) {
			sd.Fields = append(sd.Fields, p.processField(field, entry, data))
		}
	}

	if len(sd.Fields) > 0 || sd.HasExtends {
		data.AddInternal("x/cpp/json/json.h")
	}

	if entry.Filter != nil {
		sd.IsFilter = true
		data.AddSystem("utility")
	}

	if where, ok := requestFilter(entry, data.table); ok {
		sd.Where = where.Name
		sd.OptionsName = strings.TrimSuffix(sd.Name, "Request") + "Options"
		sd.Options = lo.Filter(sd.Fields, func(f fieldData, _ int) bool {
			return f.Name != "where"
		})
		data.AddSystem("utility")
	}

	if form.IsGeneric() {
		data.AddSystem("type_traits")
	}

	if hasPBFlag(entry) && !omit.IsSkipped(entry, "pb") {
		pbOutputPath := output.GetPBPath(entry)
		if pbOutputPath != "" {
			pbName := cppnaming.PBName(entry)
			pbNamespace := cppnaming.PBNamespace(pbOutputPath, entry.Namespace)
			sd.HasProto = true
			sd.ProtoType = fmt.Sprintf("%s::%s", pbNamespace, pbName)
			sd.ProtoNamespace = pbNamespace
			sd.ProtoClass = pbName
			data.AddSystem("utility")
			data.AddInternal("x/cpp/errors/errors.h")
			protoInclude := fmt.Sprintf("%s/%s.pb.h", pbOutputPath, entry.Namespace)
			data.AddInternal(protoInclude)
		}
	}

	if cppDomain, ok := entry.Domains["cpp"]; ok {
		if methodsExpr, found := cppDomain.Expressions.Find("methods"); found {
			for _, v := range methodsExpr.Values {
				if v.StringValue != "" {
					sd.Methods = append(sd.Methods, v.StringValue)
				}
			}
		}
		if includesExpr, found := cppDomain.Expressions.Find("includes"); found {
			for _, v := range includesExpr.Values {
				if v.StringValue != "" {
					data.AddInternal(v.StringValue)
				}
			}
		}
		if sysIncludesExpr, found := cppDomain.Expressions.Find(
			"system_includes",
		); found {
			for _, v := range sysIncludesExpr.Values {
				if v.StringValue != "" {
					data.AddSystem(v.StringValue)
				}
			}
		}
	}

	return sd
}

func (p *Plugin) processTypeParam(
	tp resolution.TypeParam,
	data *templateData,
) typeParamData {
	tpd := typeParamData{Name: tp.Name}
	if tp.Optional {
		tpd.HasDefault = true
		tpd.Default = "std::monostate"
		data.AddSystem("variant")
	}
	return tpd
}

// cppDefaultValue returns the default initializer for a C++ type based on both
// the C++ type string and the underlying primitive type (if any).
// Returns empty string if no explicit default is needed (e.g., for types with
// proper default constructors like std::string, std::vector, std::optional).
func cppDefaultValue(cppType, underlyingPrimitive string) string {
	if strings.Contains(cppType, "::telem::TimeStamp") {
		return "x::telem::TimeStamp(0)"
	}
	if strings.Contains(cppType, "::telem::TimeSpan") {
		return "x::telem::TimeSpan(0)"
	}
	if strings.Contains(cppType, "::telem::TimeRange") {
		return "x::telem::TimeRange{}"
	}

	if underlyingPrimitive != "" {
		switch underlyingPrimitive {
		case "bool":
			return "false"
		case "float32", "float64":
			return "0"
		case "int8", "int16", "int32", "int64":
			return "0"
		case "uint8", "uint12", "uint16", "uint20", "uint32", "uint64":
			return "0"
		case "timestamp":
			return "x::telem::TimeStamp(0)"
		case "timespan":
			return "x::telem::TimeSpan(0)"
		case "time_range", "time_range_bounded":
			return "x::telem::TimeRange{}"
		}
	}

	switch cppType {
	case "bool":
		return "false"
	case "float", "double":
		return "0"
	case "std::int8_t", "std::int16_t", "std::int32_t", "std::int64_t":
		return "0"
	case "std::uint8_t", "std::uint16_t", "std::uint32_t", "std::uint64_t":
		return "0"
	}

	if strings.HasPrefix(cppType, "x::telem::") {
		switch cppType {
		case "x::telem::TimeStamp":
			return "x::telem::TimeStamp(0)"
		case "x::telem::TimeSpan":
			return "x::telem::TimeSpan(0)"
		case "x::telem::TimeRange":
			return "x::telem::TimeRange{}"
		default:
			return "{}"
		}
	}

	return ""
}

func getUnderlyingPrimitive(
	typeRef resolution.TypeRef,
	table *resolution.Table,
) string {
	if resolution.IsPrimitive(typeRef.Name) {
		return typeRef.Name
	}

	resolved, _, ok := resolution.Dealias(typeRef, table)
	if !ok {
		return ""
	}

	switch form := resolved.Form.(type) {
	case resolution.PrimitiveForm:
		return form.Name
	case resolution.DistinctForm:
		return getUnderlyingPrimitive(form.Base, table)
	}

	return ""
}

// zeroValue returns the member initializer of a required field of typeRef, or "" when
// the type's default constructor gives the zero value.
func (p *Plugin) zeroValue(typeRef resolution.TypeRef, data *templateData) string {
	if isIntEnum(typeRef, data.table) {
		return "{}"
	}
	held := typeRef
	if hand, ok := handWritten(typeRef, data.table); ok {
		held = hand
	}
	return cppDefaultValue(
		p.typeRefToCpp(held, data),
		getUnderlyingPrimitive(typeRef, data.table),
	)
}

// handWritten returns the type that ref stands for through aliases and distinct types
// when that type is hand-written in C++. Both are using declarations in C++, so a field
// of ref holds the hand-written type.
func handWritten(
	ref resolution.TypeRef,
	table *resolution.Table,
) (resolution.TypeRef, bool) {
	resolved, ok := ref.Resolve(table)
	if !ok {
		return resolution.TypeRef{}, false
	}
	if omit.IsHand(resolved, "cpp") {
		return ref, true
	}
	switch form := resolved.Form.(type) {
	case resolution.AliasForm:
		return handWritten(form.Target, table)
	case resolution.DistinctForm:
		return handWritten(form.Base, table)
	}
	return resolution.TypeRef{}, false
}

func isIntEnum(typeRef resolution.TypeRef, table *resolution.Table) bool {
	resolved, _, ok := resolution.Dealias(typeRef, table)
	if !ok {
		return false
	}
	form, ok := resolved.Form.(resolution.EnumForm)
	return ok && form.IsIntEnum
}

// Indirect reports whether field is an x::mem::indirect: an optional field whose value
// holds parent, which std::optional cannot hold before parent is complete. A
// std::vector accepts an incomplete element, so a list field is never indirect.
func Indirect(
	field resolution.Field,
	parent resolution.Type,
	table *resolution.Table,
) bool {
	return field.Optional && field.Type.Name != "Array" &&
		resolution.RefersTo(field.Type, parent.QualifiedName, table)
}

func (p *Plugin) processField(
	field resolution.Field,
	entry resolution.Type,
	data *templateData,
) fieldData {
	cppType := p.typeRefToCpp(field.Type, data)

	var defaultValue string
	if !field.Optional {
		defaultValue = p.zeroValue(field.Type, data)
	} else if lineType, ok := p.filterLineType(field, entry, data); ok {
		cppType = lineType
	} else if Indirect(field, entry, data.table) {
		data.AddInternal("x/cpp/mem/indirect.h")
		cppType = fmt.Sprintf("x::mem::indirect<%s>", cppType)
	} else {
		data.AddSystem("optional")
		cppType = fmt.Sprintf("std::optional<%s>", cppType)
	}

	cppFieldName := domain.GetFieldName(field, "cpp")
	if cppFieldName == field.Name {
		cppFieldName = casing.FieldSnake(field.Name)
	}
	cppFieldName = keywords.Escape(cppFieldName)

	if field.Default != nil {
		if lit := p.defaultRenderer(data).
			Literal(field.Type, *field.Default); lit != "" {
			defaultValue = lit
		}
	}

	return fieldData{
		Name:         cppFieldName,
		CppType:      cppType,
		Doc:          doc.Get(field.Domains),
		DefaultValue: defaultValue,
	}
}

// requestFilter returns the filter that derives entry as its retrieve request.
func requestFilter(
	entry resolution.Type,
	table *resolution.Table,
) (resolution.Type, bool) {
	return lo.Find(table.StructTypes(), func(t resolution.Type) bool {
		return t.Filter != nil && t.Filter.Request == entry.QualifiedName
	})
}

// filterLineType returns the C++ type of a line of the filter entry, which takes a
// plain value or a list in a designated initializer. A bare line holds an operator in
// an x::filter::Field, and a typed list line is an x::filter::List. It returns false
// for any other field.
func (p *Plugin) filterLineType(
	field resolution.Field,
	entry resolution.Type,
	data *templateData,
) (string, bool) {
	if entry.Filter == nil {
		return "", false
	}
	line, ok := lo.Find(entry.Filter.Lines, func(l resolution.Field) bool {
		return l.Name == field.Name
	})
	if !ok {
		return "", false
	}
	if !line.HasType() {
		data.AddInternal("x/cpp/filter/field.h")
		op := p.typeRefToCpp(field.Type, data)
		return fmt.Sprintf("::x::filter::Field<%s>", op), true
	}
	if field.Type.Name == "Array" && field.Type.ArraySize == nil {
		data.AddInternal("x/cpp/filter/field.h")
		elem := p.typeRefToCpp(field.Type.TypeArgs[0], data)
		return fmt.Sprintf("::x::filter::List<%s>", elem), true
	}
	return "", false
}

func (p *Plugin) defaultRenderer(data *templateData) defaults.Renderer {
	return defaults.Renderer{
		Table:     data.table,
		Namespace: data.rawNs,
		TypeName: func(typeRef resolution.TypeRef) string {
			return p.typeRefToCpp(typeRef, data)
		},
	}
}

func (p *Plugin) typeRefToCpp(typeRef resolution.TypeRef, data *templateData) string {
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		if typeRef.TypeParam.HasDefault() {
			return p.typeRefToCpp(*typeRef.TypeParam.Default, data)
		}
		return typeRef.TypeParam.Name
	}

	if typeRef.Name == "Array" {
		data.AddSystem("vector")
		elementType := "void"
		if len(typeRef.TypeArgs) > 0 {
			elementType = p.typeRefToCpp(typeRef.TypeArgs[0], data)
		}
		return fmt.Sprintf("std::vector<%s>", elementType)
	}

	if typeRef.Name == "Map" {
		data.AddSystem("unordered_map")
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

	if resolution.IsPrimitive(typeRef.Name) {
		return p.primitiveToCpp(typeRef.Name, data)
	}

	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return "void"
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		return p.resolveStructType(resolved, typeRef.TypeArgs, data)
	case resolution.EnumForm:
		return p.resolveEnumType(resolved, form, data)
	case resolution.DistinctForm:
		return p.resolveDistinctType(resolved, data)
	case resolution.AliasForm:
		return p.resolveAliasType(resolved, typeRef.TypeArgs, data)
	case resolution.UnionForm:
		return p.resolveUnionType(resolved, data)
	default:
		return "void"
	}
}

// resolveUnionType resolves a discriminated union to its C++ std::variant alias
// name, adding the cross-namespace include when the union lives elsewhere.
func (p *Plugin) resolveUnionType(resolved resolution.Type, data *templateData) string {
	name := domain.GetName(resolved, "cpp")
	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.AddInternal(includePath)
			ns := cppnaming.Namespace(targetOutputPath)
			return fmt.Sprintf("::%s::%s", ns, name)
		}
	}
	return name
}

func (p *Plugin) primitiveToCpp(primitive string, data *templateData) string {
	mapping := primitiveMapper.Map(primitive)
	if mapping.TargetType == "any" {
		return "void"
	}

	for _, imp := range mapping.Imports {
		switch imp.Category {
		case "system":
			data.AddSystem(imp.Path)
		case "internal":
			data.AddInternal(imp.Path)
		}
	}

	return mapping.TargetType
}

func (p *Plugin) resolveStructType(
	resolved resolution.Type,
	typeArgs []resolution.TypeRef,
	data *templateData,
) string {
	name := resolved.Name
	var cppInclude string
	isOmitted := omit.IsSkipped(resolved, "cpp")

	if cppDomain, ok := resolved.Domains["cpp"]; ok {
		for _, expr := range cppDomain.Expressions {
			switch expr.Name {
			case "name":
				if len(expr.Values) > 0 {
					name = expr.Values[0].StringValue
				}
			case "include":
				if len(expr.Values) > 0 {
					cppInclude = expr.Values[0].StringValue
				}
			}
		}
	}

	targetOutputPath := output.GetPath(resolved, "cpp")

	if resolved.Namespace != data.rawNs {
		if isOmitted || targetOutputPath == "" {
			if cppInclude != "" {
				data.AddInternal(cppInclude)
			}
			if targetOutputPath != "" {
				ns := cppnaming.Namespace(targetOutputPath)
				name = fmt.Sprintf("::%s::%s", ns, name)
			} else if resolved.Namespace != "" {
				name = fmt.Sprintf("::%s::%s", resolved.Namespace, name)
			}
		} else {
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.AddInternal(includePath)
			ns := cppnaming.Namespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	} else {
		addLocalHandInclude(resolved, data)
	}

	return p.buildGenericType(name, typeArgs, &resolved, data)
}

func (p *Plugin) resolveEnumType(
	resolved resolution.Type,
	form resolution.EnumForm,
	data *templateData,
) string {
	name := resolved.Name
	if resolved.Namespace != data.rawNs {
		targetOutputPath := enum.FindOutputPath(resolved, data.table, "cpp")
		if targetOutputPath != "" {
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.AddInternal(includePath)
			ns := cppnaming.Namespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	}
	if !form.IsIntEnum {
		data.AddSystem("string")
		return "std::string"
	}
	return name
}

// explicitInclude returns the header declared via `@cpp include` on the type,
// letting a type override the default <output>/types.gen.h include (e.g. to break
// a header cycle with a small hand-written header).
func explicitInclude(resolved resolution.Type) string {
	cppDomain, ok := resolved.Domains["cpp"]
	if !ok {
		return ""
	}
	expr, found := cppDomain.Expressions.Find("include")
	if !found || len(expr.Values) == 0 {
		return ""
	}
	return expr.Values[0].StringValue
}

// addLocalHandInclude adds the `@cpp include` header for a reference to a
// hand-written type in the same namespace. The generated file carries no declaration
// for it, so the reference dangles without the hand header.
func addLocalHandInclude(resolved resolution.Type, data *templateData) {
	if !omit.IsSkipped(resolved, "cpp") {
		return
	}
	if include := explicitInclude(resolved); include != "" {
		data.AddInternal(include)
	}
}

func (p *Plugin) resolveDistinctType(
	resolved resolution.Type,
	data *templateData,
) string {
	name := domain.GetName(resolved, "cpp")

	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			includePath := explicitInclude(resolved)
			if includePath == "" {
				includePath = fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			}
			data.AddInternal(includePath)
		}
		ns := cppnaming.Namespace(targetOutputPath)
		return fmt.Sprintf("::%s::%s", ns, name)
	}
	addLocalHandInclude(resolved, data)
	return name
}

func (p *Plugin) resolveAliasType(
	resolved resolution.Type,
	typeArgs []resolution.TypeRef,
	data *templateData,
) string {
	name := domain.GetName(resolved, "cpp")
	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			includePath := explicitInclude(resolved)
			if includePath == "" {
				includePath = fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			}
			data.AddInternal(includePath)
			ns := cppnaming.Namespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	} else {
		addLocalHandInclude(resolved, data)
	}
	return p.buildGenericType(name, typeArgs, &resolved, data)
}

func (p *Plugin) buildGenericType(
	baseName string,
	typeArgs []resolution.TypeRef,
	targetType *resolution.Type,
	data *templateData,
) string {
	if len(typeArgs) == 0 {
		return baseName
	}

	var typeParams []resolution.TypeParam
	if targetType != nil {
		typeParams = resolution.TypeParams(targetType.Form)
	}
	var args []string
	for i, arg := range typeArgs {
		if i < len(typeParams) && typeParams[i].HasDefault() {
			continue
		}
		args = append(args, p.typeRefToCpp(arg, data))
	}

	if len(args) == 0 {
		if targetType != nil && isCppTemplateWithAllDefaults(*targetType) {
			return baseName + "<>"
		}
		return baseName
	}
	return fmt.Sprintf("%s<%s>", baseName, strings.Join(args, ", "))
}

// isCppTemplateWithAllDefaults returns true if the type would be generated as a C++
// template AND all of its C++ template parameters have defaults.
//
// A struct is a C++ template only if it has type params without explicit defaults.
// Params with explicit defaults are substituted at code generation and don't become
// template params. Optional params (without explicit defaults) DO become template
// params with implicit std::monostate default.
//
// This function returns true when:
//   - The type has at least one type param without explicit default (making it a C++
//     template)
//   - All such params are optional (giving them implicit defaults)
func isCppTemplateWithAllDefaults(t resolution.Type) bool {
	form, ok := t.Form.(resolution.StructForm)
	if !ok {
		return false
	}

	hasCppTemplateParams := false
	for _, tp := range form.TypeParams {
		if tp.HasDefault() {
			continue // This param is NOT part of C++ template (substituted)
		}
		// This param IS part of the C++ template
		hasCppTemplateParams = true
		// Check if it has a default in C++ (only optional params get implicit defaults)
		if !tp.Optional {
			return false // Has a C++ template param without default
		}
	}
	return hasCppTemplateParams // True only if has template params AND all have defaults
}

type templateData struct {
	Resource *resourceData
	*includes.Manager
	table        *resolution.Table
	OutputPath   string
	Namespace    string
	rawNs        string
	ForwardDecls []string
	Enums        []enumData
	SortedDecls  []sortedDeclData
}

// resourceData contains information for generating resource ID support.
type resourceData struct {
	// TypeName is the resource type name (e.g., "channel", "device", "task").
	TypeName string
	// KeyType is the C++ type name (e.g., "Key").
	KeyType string
	// KeyConversion is the expression to convert key to string (e.g.,
	// "std::to_string(key)" or "key").
	KeyConversion string
}

type sortedDeclData struct {
	Alias     aliasData
	Struct    structData
	TypeDef   typeDefData
	Union     unionData
	IsAlias   bool
	IsStruct  bool
	IsTypeDef bool
	IsUnion   bool
}

type typeDefData struct {
	Name               string
	CppType            string
	Doc                string
	ElementType        string
	ProtoType          string
	ProtoNamespace     string
	ProtoClass         string
	Methods            []string
	ArraySize          int64
	IsArrayWrapper     bool
	IsFixedSizeArray   bool
	ElementIsPrimitive bool
	HasProto           bool
}

type aliasData struct {
	Name       string
	Target     string
	TypeParams []string
	IsGeneric  bool
}

type structData struct {
	ProtoClass     string
	ProtoType      string
	Name           string
	Doc            string
	ProtoNamespace string
	AliasOf        string
	Methods        []string
	ExtendsTypes   []string
	TypeParams     []typeParamData
	Fields         []fieldData
	// InheritedDefaults are inherited members the struct redeclares to change only
	// their default. A generated constructor assigns them.
	InheritedDefaults []inheritedDefaultData
	HasProto          bool
	IsAlias           bool
	IsGeneric         bool
	HasExtends        bool
	// IsFilter is true for the Where of a filter, which combines with &&, ||, and !.
	IsFilter bool
	// Where is the filter of a retrieve request, and is empty for any other struct.
	Where string
	// OptionsName names the struct that holds Options.
	OptionsName string
	// Options are the fields of a retrieve request other than where.
	Options []fieldData
}

// inheritedDefaultData assigns a new default to a member declared by a base struct.
type inheritedDefaultData struct {
	Name  string
	Value string
}

type typeParamData struct {
	Name       string
	Default    string
	HasDefault bool
}

type fieldData struct {
	Name string
	// DefaultValue is the default initializer (e.g., "0", "false", "{}").
	DefaultValue string
	CppType      string
	Doc          string
}

type enumData struct {
	Name      string
	Values    []enumValueData
	IsIntEnum bool
}

type enumValueData struct {
	Name     string
	Value    string
	IntValue int64
}

func hasPBFlag(t resolution.Type) bool {
	_, hasPB := t.Domains["pb"]
	return hasPB
}

// hasExplicitPBName returns true if the type has an explicit @pb name directive.
// This is used to determine if an array wrapper has a corresponding proto message.
func hasExplicitPBName(s resolution.Type) bool {
	if domain, ok := s.Domains["pb"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 &&
				expr.Values[0].StringValue != "" {
				return true
			}
		}
	}
	return false
}

// resolveExtendsType converts a parent TypeRef to a fully qualified C++ type string.
// It also adds the necessary include for the parent's header.
func (p *Plugin) resolveExtendsType(
	extendsRef resolution.TypeRef,
	parent resolution.Type,
	data *templateData,
) string {
	name := domain.GetName(parent, "cpp")

	// Check for cross-namespace reference
	if parent.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(parent, "cpp")
		if targetOutputPath != "" {
			// Add include for the parent's header
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.AddInternal(includePath)
			// Use namespace-qualified name with :: prefix for absolute resolution
			ns := cppnaming.Namespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	}

	// Handle generic parents with type arguments
	return p.buildGenericType(name, extendsRef.TypeArgs, &parent, data)
}

// extractResource extracts resource metadata from the struct or union with a @resource
// domain, keyed by a struct field with an @key annotation. Returns nil if no suitable
// type is found.
func (p *Plugin) extractResource(
	structs []resolution.Type,
	table *resolution.Table,
	data *templateData,
) *resourceData {
	skip := func(typ resolution.Type) bool { return omit.IsSkipped(typ, "cpp") }
	rawKeyFields := key.Collect(structs, table, skip)
	unions := lo.Filter(table.UnionTypes(), func(typ resolution.Type, _ int) bool {
		return output.GetPath(typ, "cpp") == data.OutputPath
	})
	resData := resource.Extract(
		slices.Concat(structs, unions),
		rawKeyFields,
		skip,
		table,
	)
	if resData == nil {
		return nil
	}
	keyType := p.typeRefToCpp(resData.KeyField.Type, data)

	// Determine key conversion based on the underlying primitive type
	primitive := resData.KeyField.Primitive
	var keyConversion string
	switch primitive {
	case "string":
		// String types don't need conversion
		keyConversion = "key"
	case "uuid":
		// UUID has a to_string() method
		keyConversion = "key.to_string()"
	default:
		// Numeric types need std::to_string
		keyConversion = "std::to_string(key)"
	}

	// Add the resource ID include
	data.AddInternal("client/cpp/resource/id.h")

	return &resourceData{
		TypeName:      resData.TypeName,
		KeyType:       keyType,
		KeyConversion: keyConversion,
	}
}

var templateFuncs = template.FuncMap{
	"join":             strings.Join,
	"toUpper":          strings.ToUpper,
	"toScreamingSnake": cppnaming.ScreamingSnake,
	"toSnakeCase":      casing.FieldSnake,
	"formatDoc":        doc.FormatCpp,
}

var fileTemplate = template.Must(
	template.New("cpp-types").
		Funcs(templateFuncs).
		Parse(`// Code generated by oracle. DO NOT EDIT.

#pragma once
{{- if .HasIncludes}}

{{range .SystemIncludes -}}
#include <{{.}}>
{{end -}}
{{range .InternalIncludes -}}
#include "{{.}}"
{{end}}
{{- end}}

namespace {{.Namespace}} {
{{- if .ForwardDecls}}
{{range .ForwardDecls}}
struct {{.}};
{{- end}}
{{end}}
{{- range $i, $enum := .Enums}}
{{if $i}}
{{end}}
{{if $enum.IsIntEnum}}
enum class {{$enum.Name}} : std::uint8_t {
{{- range $j, $v := $enum.Values}}
    {{$v.Name}} = {{$v.IntValue}},
{{- end}}
};
{{- else}}
{{- range $enum.Values}}
constexpr const char* {{$enum.Name | toScreamingSnake}}_{{.Name | toScreamingSnake}} = "{{.Value}}";
{{- end}}
{{- end}}
{{- end}}
{{- range $i, $d := .SortedDecls}}
{{- if $d.IsTypeDef}}
{{- $td := $d.TypeDef}}
{{if or $i (gt (len $.Enums) 0)}}
{{end}}
{{- if $td.Doc}}
{{formatDoc $td.Name $td.Doc}}
{{- end}}
{{- if $td.IsArrayWrapper}}
{{- if $td.IsFixedSizeArray}}
struct {{$td.Name}} : private std::array<{{$td.ElementType}}, {{$td.ArraySize}}> {
    using Base = std::array<{{$td.ElementType}}, {{$td.ArraySize}}>;

    // Inherit constructors
    using Base::Base;
    {{$td.Name}}() = default;

    // Container interface
    using Base::value_type;
    using Base::iterator;
    using Base::const_iterator;
    using Base::reverse_iterator;
    using Base::const_reverse_iterator;
    using Base::size_type;
    using Base::difference_type;
    using Base::reference;
    using Base::const_reference;
    using Base::begin;
    using Base::end;
    using Base::cbegin;
    using Base::cend;
    using Base::rbegin;
    using Base::rend;
    using Base::crbegin;
    using Base::crend;
    using Base::size;
    using Base::empty;
    using Base::max_size;
    using Base::operator[];
    using Base::at;
    using Base::front;
    using Base::back;
    using Base::data;
    using Base::fill;
    using Base::swap;

    static {{$td.Name}} parse(x::json::Parser parser);
    [[nodiscard]] x::json::json to_json() const;
{{- else}}
struct {{$td.Name}} : private std::vector<{{$td.ElementType}}> {
    using Base = std::vector<{{$td.ElementType}}>;

    // Inherit constructors - these are instantiated at point of use, not declaration
    using Base::Base;
    // The default constructor is defined out-of-line below so it instantiates the
    // element type's destructor only after the element type is complete; the element
    // may be forward-declared here to break a reference cycle.
    {{$td.Name}}();
{{- if $td.ElementIsPrimitive}}
    {{$td.Name}}(std::initializer_list<{{$td.ElementType}}> init) : Base(init) {}
{{- end}}

    // Container interface
    using Base::value_type;
    using Base::iterator;
    using Base::const_iterator;
    using Base::reverse_iterator;
    using Base::const_reverse_iterator;
    using Base::size_type;
    using Base::difference_type;
    using Base::reference;
    using Base::const_reference;
    using Base::begin;
    using Base::end;
    using Base::cbegin;
    using Base::cend;
    using Base::rbegin;
    using Base::rend;
    using Base::crbegin;
    using Base::crend;
    using Base::size;
    using Base::empty;
    using Base::max_size;
    using Base::capacity;
    using Base::reserve;
    using Base::shrink_to_fit;
    using Base::operator[];
    using Base::at;
    using Base::front;
    using Base::back;
    using Base::data;
    using Base::push_back;
    using Base::emplace_back;
    using Base::pop_back;
    using Base::insert;
    using Base::emplace;
    using Base::erase;
    using Base::clear;
    using Base::resize;
    using Base::swap;
    using Base::assign;

    static {{$td.Name}} parse(x::json::Parser parser);
    [[nodiscard]] x::json::json to_json() const;
{{- end}}
{{- if $td.HasProto}}

    using proto_type = {{$td.ProtoType}};
    [[nodiscard]] std::pair<{{$td.ProtoType}}, x::errors::Error> to_proto() const;
    static std::pair<{{$td.Name}}, x::errors::Error> from_proto(const {{$td.ProtoType}}& pb);
{{- end}}
{{- if $td.Methods}}

{{- range $td.Methods}}
    {{.}};
{{- end}}
{{- end}}
};
{{- else}}
using {{$td.Name}} = {{$td.CppType}};
{{- end}}
{{- else if $d.IsAlias}}
{{- $a := $d.Alias}}
{{if or $i (gt (len $.Enums) 0)}}
{{end}}
{{- if $a.IsGeneric}}template <{{range $j, $p := $a.TypeParams}}{{if $j}}, {{end}}typename {{$p}}{{end}}>
{{end}}using {{$a.Name}} = {{$a.Target}};
{{- else if $d.IsStruct}}
{{- $s := $d.Struct}}
{{if or $i (gt (len $.Enums) 0)}}
{{end}}
{{- if $s.Options}}
/// @brief {{$s.OptionsName}} holds every field of a {{$s.Name}} except where.
struct {{$s.OptionsName}} {
{{- range $s.Options}}
{{- if .Doc}}
    {{formatDoc .Name .Doc 4 | printf "%s"}}
{{- end}}
    {{.CppType}} {{.Name}}{{if .DefaultValue}} = {{.DefaultValue}}{{end}};
{{- end}}
};

{{end}}
{{- if $s.Doc}}
{{formatDoc $s.Name $s.Doc}}
{{end}}
{{- if $s.IsAlias}}
{{- if $s.IsGeneric}}template <{{range $j, $p := $s.TypeParams}}{{if $j}}, {{end}}typename {{$p.Name}}{{if $p.HasDefault}} = {{$p.Default}}{{end}}{{end}}>
{{end}}using {{$s.Name}} = {{$s.AliasOf}};
{{- else}}
{{- if $s.IsGeneric}}template <{{range $j, $p := $s.TypeParams}}{{if $j}}, {{end}}typename {{$p.Name}}{{if $p.HasDefault}} = {{$p.Default}}{{end}}{{end}}>
{{end}}struct {{$s.Name}}{{if $s.HasExtends}} : {{range $i, $parent := $s.ExtendsTypes}}{{if $i}}, {{end}}public {{$parent}}{{end}}{{end}} {
{{- range $s.Fields}}
{{- if .Doc}}
    {{formatDoc .Name .Doc 4 | printf "%s"}}
{{- end}}
    {{.CppType}} {{.Name}}{{if .DefaultValue}} = {{.DefaultValue}}{{end}};
{{- end}}
{{- if $s.InheritedDefaults}}

    {{$s.Name}}() {
{{- range $s.InheritedDefaults}}
        this->{{.Name}} = {{.Value}};
{{- end}}
    }
{{- end}}

{{- if $s.IsFilter}}

    /// @brief returns a filter that matches when both a and b match.
    friend {{$s.Name}} operator&&({{$s.Name}} a, {{$s.Name}} b) {
        {{$s.Name}} w;
        w.and_.emplace();
        w.and_->push_back(std::move(a));
        w.and_->push_back(std::move(b));
        return w;
    }

    /// @brief returns a filter that matches when a or b matches.
    friend {{$s.Name}} operator||({{$s.Name}} a, {{$s.Name}} b) {
        {{$s.Name}} w;
        w.or_.emplace();
        w.or_->push_back(std::move(a));
        w.or_->push_back(std::move(b));
        return w;
    }

    /// @brief returns a filter that matches when a does not match.
    friend {{$s.Name}} operator!({{$s.Name}} a) {
        {{$s.Name}} w;
        w.not_ = std::move(a);
        return w;
    }
{{- end}}
{{- if $s.Options}}

    /// @brief returns a request for the records that where selects.
    static {{$s.Name}}
    from({{$s.Where}} where, const {{$s.OptionsName}} &options = {}) {
        return {
            .where = std::move(where),
{{- range $s.Options}}
            .{{.Name}} = options.{{.Name}},
{{- end}}
        };
    }
{{- end}}

    static {{$s.Name}} parse(x::json::Parser parser);
    [[nodiscard]] x::json::json to_json() const;
{{- if $s.HasProto}}

    using proto_type = {{$s.ProtoType}};
    [[nodiscard]] std::pair<{{$s.ProtoType}}, x::errors::Error> to_proto() const;
    static std::pair<{{$s.Name}}, x::errors::Error> from_proto(const {{$s.ProtoType}}& pb);
{{- end}}
{{- if $s.Methods}}

{{- range $s.Methods}}
    {{.}};
{{- end}}
{{- end}}
};
{{- end}}
{{- else if $d.IsUnion}}
{{- $u := $d.Union}}
{{if or $i (gt (len $.Enums) 0)}}
{{end}}
{{- if $u.Doc}}
{{formatDoc $u.Name $u.Doc}}
{{- end}}
using {{$u.Name}} = std::variant<{{range $j, $v := $u.Variants}}{{if $j}}, {{end}}{{$v.TypeName}}{{end}}>;

{{$u.Name}} parse_{{$u.SnakeName}}(x::json::Parser parser);
[[nodiscard]] x::json::json to_json(const {{$u.Name}}& value);
{{- if $u.HasProto}}
[[nodiscard]] std::pair<{{$u.ProtoType}}, x::errors::Error> to_proto(const {{$u.Name}}& value);
std::pair<{{$u.Name}}, x::errors::Error> {{$u.SnakeName}}_from_proto(const {{$u.ProtoType}}& pb);
{{- end}}
{{- if $u.Base}}

/// @brief returns the fields every variant of the union shares.
[[nodiscard]] inline const {{$u.Base}}& base(const {{$u.Name}}& value) {
    return std::visit([](const auto& v) -> const {{$u.Base}}& { return v; }, value);
}
{{- end}}
{{- end}}
{{- end}}
{{- range $d := .SortedDecls}}
{{- if and $d.IsTypeDef $d.TypeDef.IsArrayWrapper (not $d.TypeDef.IsFixedSizeArray)}}

inline {{$d.TypeDef.Name}}::{{$d.TypeDef.Name}}() = default;
{{- end}}
{{- end}}
{{- if .Resource}}

const synnax::resource::ID TYPE_RESOURCE_ID("{{.Resource.TypeName}}", "");

inline synnax::resource::ID resource_id(const {{.Resource.KeyType}}& key) {
    return synnax::resource::ID("{{.Resource.TypeName}}", {{.Resource.KeyConversion}});
}
{{- end}}
}
`),
)
