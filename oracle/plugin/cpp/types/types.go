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
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	cppprimitives "github.com/synnaxlabs/oracle/plugin/cpp/primitives"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// primitiveMapper is the C++-specific primitive type mapper.
var primitiveMapper = cppprimitives.Mapper()

type Plugin struct{ Options Options }

type Options struct {
	FileNamePattern  string
	DisableFormatter bool // If true, skip running clang-format
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

func (p *Plugin) Check(req *plugin.Request) error {
	return nil
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	// Collect types using framework collectors
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

	// Build combined order - structs first, then aliases that don't share path with structs
	var combinedOrder []string
	combinedOrder = append(combinedOrder, structCollector.Paths()...)
	for _, path := range aliasCollector.Paths() {
		if !structCollector.Has(path) && !typeDefCollector.Has(path) {
			combinedOrder = append(combinedOrder, path)
		}
	}

	// Collect standalone enum output paths (for schemas where all structs are omitted but enums exist)
	enumOutputPaths := make(map[string][]resolution.Type)
	for _, e := range req.Resolutions.EnumTypes() {
		if omit.IsType(e, "cpp") {
			continue
		}
		enumPath := enum.FindOutputPath(e, req.Resolutions, "cpp")
		if enumPath == "" {
			continue
		}
		// Only add if not already covered by structs, aliases, or typedefs
		if !structCollector.Has(enumPath) && !aliasCollector.Has(enumPath) && !typeDefCollector.Has(enumPath) {
			enumOutputPaths[enumPath] = append(enumOutputPaths[enumPath], e)
		}
	}
	for path := range enumOutputPaths {
		found := false
		for _, p := range combinedOrder {
			if p == path {
				found = true
				break
			}
		}
		if !found {
			combinedOrder = append(combinedOrder, path)
		}
	}

	for _, outputPath := range combinedOrder {
		structs := structCollector.Get(outputPath)
		enums := enum.CollectReferenced(structs, req.Resolutions)

		if len(structs) > 0 {
			namespace := structs[0].Namespace
			enums = framework.MergeTypes(enums, enum.CollectNamespaceEnums(namespace, outputPath, req.Resolutions, "cpp", nil))
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
					enums = framework.MergeTypes(enums, enum.CollectNamespaceEnums(namespace, outputPath, req.Resolutions, "cpp", nil))
				}
			}
		}

		var typeDefs []resolution.Type
		if typeDefCollector.Has(outputPath) {
			typeDefs = typeDefCollector.Remove(outputPath)
		}
		aliases := aliasCollector.Get(outputPath)
		content, err := p.generateFile(outputPath, structs, enums, typeDefs, aliases, req.Resolutions)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
			Content: content,
		})
	}

	err = typeDefCollector.ForEach(func(outputPath string, typeDefs []resolution.Type) error {
		var enums []resolution.Type
		if len(typeDefs) > 0 {
			namespace := typeDefs[0].Namespace
			enums = enum.CollectNamespaceEnums(namespace, outputPath, req.Resolutions, "cpp", nil)
		}
		content, err := p.generateFile(outputPath, nil, enums, typeDefs, nil, req.Resolutions)
		if err != nil {
			return errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
			Content: content,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

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
		Namespace:   deriveNamespace(outputPath),
		Structs:     make([]structData, 0, len(structs)),
		Enums:       make([]enumData, 0, len(enums)),
		TypeDefs:    make([]typeDefData, 0, len(typeDefs)),
		Aliases:     make([]aliasData, 0, len(aliases)),
		SortedDecls: make([]sortedDeclData, 0),
		includes:    newIncludeManager(),
		table:       table,
		rawNs:       namespace,
	}

	// Track declared type names to avoid duplicates
	declaredNames := make(map[string]bool)

	// Process enums that are in the same namespace
	for _, e := range enums {
		if e.Namespace == namespace && !omit.IsType(e, "cpp") {
			data.Enums = append(data.Enums, p.processEnum(e))
		}
	}

	// Combine typedefs, aliases and structs for topological sorting.
	// We need to include omitted types (like parent structs) to maintain
	// correct dependency ordering for child types.
	var combinedTypes []resolution.Type
	combinedTypes = append(combinedTypes, typeDefs...)
	combinedTypes = append(combinedTypes, aliases...)
	combinedTypes = append(combinedTypes, structs...)

	// For structs that extend omitted parents, we need to add the parent types
	// to the sort so their dependencies are considered. Get all structs in the
	// same namespace from the resolution table.
	if namespace != "" {
		allNamespaceTypes := table.TypesInNamespace(namespace)
		existingQNames := make(map[string]bool)
		for _, t := range combinedTypes {
			existingQNames[t.QualifiedName] = true
		}
		for _, t := range allNamespaceTypes {
			if !existingQNames[t.QualifiedName] {
				combinedTypes = append(combinedTypes, t)
			}
		}
	}

	// Sort topologically so dependencies come before dependents
	allSortedTypes := table.TopologicalSort(combinedTypes)

	// Now filter out omitted types while preserving the sorted order
	var sortedTypes []resolution.Type
	for _, typ := range allSortedTypes {
		if !omit.IsType(typ, "cpp") {
			sortedTypes = append(sortedTypes, typ)
		}
	}

	// Collect forward declarations for non-generic structs only.
	// Template structs cannot use simple forward declarations - they require
	// template<typename T> struct Foo; syntax, and even then std::optional<Foo<T>>
	// needs the full definition. So we skip templates entirely.
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
			if !declaredNames[tdd.Name] {
				declaredNames[tdd.Name] = true
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
		}
	}

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
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

func (p *Plugin) processEnum(e resolution.Type) enumData {
	form, ok := e.Form.(resolution.EnumForm)
	if !ok {
		return enumData{Name: e.Name}
	}
	values := make([]enumValueData, 0, len(form.Values))
	for _, v := range form.Values {
		// Use snake_case for string enum constants, PascalCase for int enums
		name := toPascalCase(v.Name)
		if !form.IsIntEnum {
			name = toSnakeCase(v.Name)
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
	// Check for @cpp name override
	name := domain.GetName(td, "cpp")

	tdd := typeDefData{
		Name:    name,
		CppType: p.typeRefToCpp(form.Base, data),
	}

	// Check if this is an array type - if so, generate a wrapper struct
	if form.Base.Name == "Array" && len(form.Base.TypeArgs) > 0 {
		tdd.IsArrayWrapper = true
		elemTypeRef := form.Base.TypeArgs[0]
		tdd.ElementType = p.typeRefToCpp(elemTypeRef, data)
		// Check if element is primitive - allows inline initializer_list constructor
		tdd.ElementIsPrimitive = resolution.IsPrimitive(elemTypeRef.Name)
		// Check for fixed-size array
		if form.Base.ArraySize != nil {
			tdd.IsFixedSizeArray = true
			tdd.ArraySize = *form.Base.ArraySize
			data.includes.addSystem("array")
		} else {
			data.includes.addSystem("vector")
		}

		// Extract custom methods from @cpp directives
		if cppDomain, ok := td.Domains["cpp"]; ok {
			if methodsExpr, found := cppDomain.Expressions.Find("methods"); found {
				for _, v := range methodsExpr.Values {
					if v.StringValue != "" {
						tdd.Methods = append(tdd.Methods, v.StringValue)
					}
				}
			}
			// Add custom includes required by the methods
			if includesExpr, found := cppDomain.Expressions.Find("includes"); found {
				for _, v := range includesExpr.Values {
					if v.StringValue != "" {
						data.includes.addInternal(v.StringValue)
					}
				}
			}
			// Add system includes
			if sysIncludesExpr, found := cppDomain.Expressions.Find("system_includes"); found {
				for _, v := range sysIncludesExpr.Values {
					if v.StringValue != "" {
						data.includes.addSystem(v.StringValue)
					}
				}
			}
		}

		// Note: Array wrappers do NOT get proto support by default because
		// proto uses repeated fields, not wrapper messages. Proto support for
		// array wrappers requires an explicit @pb name directive indicating
		// a corresponding proto message exists.
		if hasPBFlag(td) && !omit.IsType(td, "pb") {
			// Only generate proto for array wrappers if they have an explicit @pb name
			// (meaning someone defined a proto message for this wrapper type)
			if hasExplicitPBName(td) {
				pbOutputPath := output.GetPBPath(td)
				if pbOutputPath != "" {
					pbName := getPBName(td)
					pbNamespace := derivePBCppNamespace(pbOutputPath)
					tdd.HasProto = true
					tdd.ProtoType = fmt.Sprintf("%s::%s", pbNamespace, pbName)
					tdd.ProtoNamespace = pbNamespace
					tdd.ProtoClass = pbName
					data.includes.addSystem("utility")
					data.includes.addInternal("x/cpp/errors/errors.h")
					// Include the protobuf header
					protoInclude := fmt.Sprintf("%s/%s.pb.h", pbOutputPath, td.Namespace)
					data.includes.addInternal(protoInclude)
				}
			}
		}

		// Array wrappers need json.h for parse/to_json
		data.includes.addInternal("x/cpp/json/json.h")
	}

	return tdd
}

func (p *Plugin) processAlias(alias resolution.Type, data *templateData) aliasData {
	form, ok := alias.Form.(resolution.AliasForm)
	if !ok {
		return aliasData{Name: alias.Name, Target: "void"}
	}

	// Get the C++ name (respects @cpp name directive)
	name := domain.GetName(alias, "cpp")

	// Convert target type to C++ type string
	target := p.aliasTargetToCpp(form.Target, data)

	// Collect type parameters, skipping defaulted ones
	var typeParams []string
	for _, tp := range form.TypeParams {
		if tp.HasDefault() {
			continue // Skip defaulted type params
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

func (p *Plugin) aliasTargetToCpp(typeRef resolution.TypeRef, data *templateData) string {
	// Handle type parameters
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		// For defaulted type params, use the default type instead
		if typeRef.TypeParam.HasDefault() {
			return p.aliasTargetToCpp(*typeRef.TypeParam.Default, data)
		}
		return typeRef.TypeParam.Name
	}

	// Check for Array (built-in generic)
	if typeRef.Name == "Array" {
		data.includes.addSystem("vector")
		elementType := "void"
		if len(typeRef.TypeArgs) > 0 {
			elementType = p.aliasTargetToCpp(typeRef.TypeArgs[0], data)
		}
		return fmt.Sprintf("std::vector<%s>", elementType)
	}

	// Check for Map (built-in generic)
	if typeRef.Name == "Map" {
		data.includes.addSystem("unordered_map")
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

	// Check if it's a primitive
	if resolution.IsPrimitive(typeRef.Name) {
		return p.primitiveToCpp(typeRef.Name, data)
	}

	// Try to resolve the type
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return typeRef.Name
	}

	// Build the base name with namespace handling
	name := resolved.Name
	isOmitted := omit.IsType(resolved, "cpp")
	targetOutputPath := output.GetPath(resolved, "cpp")

	// Check for @cpp include and name overrides
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

	// Handle cross-namespace references
	if resolved.Namespace != data.rawNs {
		if isOmitted || targetOutputPath == "" {
			// Handwritten type - use @cpp include and resolved namespace
			if cppInclude != "" {
				data.includes.addInternal(cppInclude)
			}
			// If there's an output path, derive the namespace from it (handles x/cpp/telem -> x::telem).
			// Otherwise use the schema namespace directly.
			if targetOutputPath != "" {
				ns := deriveNamespace(targetOutputPath)
				name = fmt.Sprintf("::%s::%s", ns, name)
			} else if resolved.Namespace != "" {
				name = fmt.Sprintf("::%s::%s", resolved.Namespace, name)
			}
		} else {
			// Generated type - include the generated header and add namespace prefix
			// Must use :: prefix for absolute namespace resolution
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.includes.addInternal(includePath)
			ns := deriveNamespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	}

	// Build with type arguments, filtering out defaulted params
	// For generic types with all-defaulted/optional params referenced without args, we need <>
	// BUT only if the type is actually a C++ template (has params without explicit defaults)
	if len(typeRef.TypeArgs) == 0 {
		// Check if target is a C++ template with all params defaulted
		if isCppTemplateWithAllDefaults(resolved) {
			return name + "<>" // Need <> to instantiate template with defaults
		}
		return name
	}

	// Filter type args, skipping those for defaulted params
	var args []string
	if form, ok := resolved.Form.(resolution.StructForm); ok {
		for i, arg := range typeRef.TypeArgs {
			if i < len(form.TypeParams) && form.TypeParams[i].HasDefault() {
				continue // Skip type args for defaulted params
			}
			args = append(args, p.aliasTargetToCpp(arg, data))
		}
	} else {
		for _, arg := range typeRef.TypeArgs {
			args = append(args, p.aliasTargetToCpp(arg, data))
		}
	}

	if len(args) == 0 {
		// All args were for defaulted params - check if we need <>
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

	// Process type parameters, skipping those with explicit defaults but keeping optional ones
	// Optional type params get implicit defaults (std::monostate) which C++ can handle
	for _, tp := range form.TypeParams {
		if tp.HasDefault() {
			continue // Skip type params with explicit defaults (substitute default value)
		}
		sd.TypeParams = append(sd.TypeParams, p.processTypeParam(tp, data))
	}
	sd.IsGeneric = len(sd.TypeParams) > 0

	if isAlias {
		sd.AliasOf = p.typeRefToCpp(aliasForm.Target, data)
		return sd
	}

	// Check if we can use C++ multiple inheritance
	if canUseInheritance(form, data.table) {
		sd.HasExtends = true
		for _, extendsRef := range form.Extends {
			parent, ok := extendsRef.Resolve(data.table)
			if !ok {
				continue
			}
			qualifiedName := p.resolveExtendsType(extendsRef, parent, data)
			sd.ExtendsTypes = append(sd.ExtendsTypes, qualifiedName)
		}
		// Only process child's own fields (not inherited)
		for _, field := range form.Fields {
			sd.Fields = append(sd.Fields, p.processField(field, entry, data))
		}
	} else {
		// Fall back to field flattening (current behavior)
		for _, field := range resolution.UnifiedFields(entry, data.table) {
			sd.Fields = append(sd.Fields, p.processField(field, entry, data))
		}
	}

	// Structs with fields or inheritance need json.h for parse/to_json methods
	if len(sd.Fields) > 0 || sd.HasExtends {
		data.includes.addInternal("x/cpp/json/json.h")
	}

	// Generic structs need type_traits for if constexpr checks
	if form.IsGeneric() {
		data.includes.addSystem("type_traits")
	}

	if hasPBFlag(entry) && !omit.IsType(entry, "pb") {
		pbOutputPath := output.GetPBPath(entry)
		if pbOutputPath != "" {
			pbName := getPBName(entry)
			pbNamespace := derivePBCppNamespace(pbOutputPath)
			sd.HasProto = true
			sd.ProtoType = fmt.Sprintf("%s::%s", pbNamespace, pbName)
			sd.ProtoNamespace = pbNamespace
			sd.ProtoClass = pbName
			data.includes.addSystem("utility")
			data.includes.addInternal("x/cpp/errors/errors.h")
			// Include the protobuf header
			protoInclude := fmt.Sprintf("%s/%s.pb.h", pbOutputPath, entry.Namespace)
			data.includes.addInternal(protoInclude)
		}
	}

	// Extract custom method declarations and includes from @cpp directives
	if cppDomain, ok := entry.Domains["cpp"]; ok {
		if methodsExpr, found := cppDomain.Expressions.Find("methods"); found {
			for _, v := range methodsExpr.Values {
				if v.StringValue != "" {
					sd.Methods = append(sd.Methods, v.StringValue)
				}
			}
		}
		// Add custom includes required by the methods
		if includesExpr, found := cppDomain.Expressions.Find("includes"); found {
			for _, v := range includesExpr.Values {
				if v.StringValue != "" {
					data.includes.addInternal(v.StringValue)
				}
			}
		}
		// Add system includes (e.g., <ostream>)
		if sysIncludesExpr, found := cppDomain.Expressions.Find("system_includes"); found {
			for _, v := range sysIncludesExpr.Values {
				if v.StringValue != "" {
					data.includes.addSystem(v.StringValue)
				}
			}
		}
	}

	return sd
}

func (p *Plugin) processTypeParam(tp resolution.TypeParam, data *templateData) typeParamData {
	tpd := typeParamData{Name: tp.Name}
	// Optional type params without explicit defaults get std::monostate as implicit default
	// This mirrors TypeScript's approach of using z.ZodNever for optional type params
	if tp.Optional {
		tpd.HasDefault = true
		tpd.Default = "std::monostate"
		data.includes.addSystem("variant")
	}
	return tpd
}

// cppDefaultValue returns the default initializer for a C++ type based on both
// the C++ type string and the underlying primitive type (if any).
// Returns empty string if no explicit default is needed (e.g., for types with
// proper default constructors like std::string, std::vector, std::optional).
func cppDefaultValue(cppType string, underlyingPrimitive string) string {
	// First check for telem types with explicit constructors (must check before underlying primitive)
	// These types are based on int64 but need explicit constructor calls
	if strings.Contains(cppType, "::telem::TimeStamp") {
		return "x::telem::TimeStamp(0)"
	}
	if strings.Contains(cppType, "::telem::TimeSpan") {
		return "x::telem::TimeSpan(0)"
	}
	if strings.Contains(cppType, "::telem::TimeRange") {
		return "x::telem::TimeRange{}"
	}

	// Then check the underlying primitive type for aliases/distinct types
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

	// Then check the C++ type string for direct primitive types
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

	// Telem types with explicit constructors need constructor syntax
	if strings.HasPrefix(cppType, "x::telem::") {
		switch cppType {
		case "x::telem::TimeStamp":
			return "x::telem::TimeStamp(0)"
		case "x::telem::TimeSpan":
			return "x::telem::TimeSpan(0)"
		case "x::telem::TimeRange":
			return "x::telem::TimeRange{}"
		default:
			// Other telem types default to brace initialization
			return "{}"
		}
	}

	// Types that have proper default constructors don't need explicit defaults:
	// - std::string defaults to ""
	// - std::vector<T> defaults to empty
	// - std::optional<T> defaults to nullopt
	// - std::unordered_map<K,V> defaults to empty
	// - x::mem::indirect<T> defaults to empty/null
	// - User-defined structs have their own field defaults
	return ""
}

// getUnderlyingPrimitive returns the underlying primitive type name for a type reference.
// For primitives, returns the primitive name directly.
// For distinct types (aliases), returns the underlying primitive if the alias is based on a primitive.
// Returns empty string if not based on a primitive.
func getUnderlyingPrimitive(typeRef resolution.TypeRef, table *resolution.Table) string {
	// Direct primitive check
	if resolution.IsPrimitive(typeRef.Name) {
		return typeRef.Name
	}

	// Try to resolve the type
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return ""
	}

	// Check for distinct type (type alias)
	if form, ok := resolved.Form.(resolution.DistinctForm); ok {
		// Recursively check the base type
		return getUnderlyingPrimitive(form.Base, table)
	}

	return ""
}

func (p *Plugin) processField(field resolution.Field, entry resolution.Type, data *templateData) fieldData {
	cppType := p.typeRefToCpp(field.Type, data)
	isSelfRef := isSelfReference(field.Type, entry)

	// Get the underlying primitive type for determining default values
	// This handles type aliases like `Key = uint32` where Key should get `= 0`
	underlyingPrimitive := getUnderlyingPrimitive(field.Type, data.table)

	// Apply optional wrappers based on field flags
	if field.IsHardOptional {
		if isSelfRef {
			// Self-referential types need indirect storage because std::optional<T>
			// requires T to be complete, but T is incomplete inside its own definition.
			// x::mem::indirect provides value semantics with deep copy support.
			data.includes.addInternal("x/cpp/mem/indirect.h")
			cppType = fmt.Sprintf("x::mem::indirect<%s>", cppType)
		} else {
			data.includes.addSystem("optional")
			cppType = fmt.Sprintf("std::optional<%s>", cppType)
		}
		// Optional types don't need default values since std::optional defaults to nullopt
		underlyingPrimitive = ""
	}

	// Get the C++ field name, respecting @cpp name override
	// If there's an override, use it directly; otherwise convert to snake_case
	cppFieldName := domain.GetFieldName(field, "cpp")
	if cppFieldName == field.Name {
		cppFieldName = toSnakeCase(field.Name)
	}

	return fieldData{
		Name:         cppFieldName,
		CppType:      cppType,
		Doc:          doc.Get(field.Domains),
		IsSelfRef:    isSelfRef,
		DefaultValue: cppDefaultValue(cppType, underlyingPrimitive),
	}
}

// isSelfReference checks if a type reference refers to its containing type,
// either directly or through type arguments (for generic recursive references).
func isSelfReference(t resolution.TypeRef, parent resolution.Type) bool {
	if t.Name == parent.QualifiedName {
		return true
	}
	// Check type arguments for generic recursive references
	for _, arg := range t.TypeArgs {
		if isSelfReference(arg, parent) {
			return true
		}
	}
	return false
}

func (p *Plugin) typeRefToCpp(typeRef resolution.TypeRef, data *templateData) string {
	// Handle type parameters first
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		// For defaulted type params, use the default type instead
		if typeRef.TypeParam.HasDefault() {
			return p.typeRefToCpp(*typeRef.TypeParam.Default, data)
		}
		return typeRef.TypeParam.Name
	}

	// Check for Array (built-in generic)
	if typeRef.Name == "Array" {
		data.includes.addSystem("vector")
		elementType := "void"
		if len(typeRef.TypeArgs) > 0 {
			elementType = p.typeRefToCpp(typeRef.TypeArgs[0], data)
		}
		return fmt.Sprintf("std::vector<%s>", elementType)
	}

	// Check for Map (built-in generic)
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

	// Check if it's a primitive
	if resolution.IsPrimitive(typeRef.Name) {
		return p.primitiveToCpp(typeRef.Name, data)
	}

	// Try to resolve the type from the table
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
	default:
		return "void"
	}
}

func (p *Plugin) primitiveToCpp(primitive string, data *templateData) string {
	mapping := primitiveMapper.Map(primitive)
	if mapping.TargetType == "any" {
		return "void"
	}

	for _, imp := range mapping.Imports {
		switch imp.Category {
		case "system":
			data.includes.addSystem(imp.Path)
		case "internal":
			data.includes.addInternal(imp.Path)
		}
	}

	return mapping.TargetType
}

func (p *Plugin) resolveStructType(resolved resolution.Type, typeArgs []resolution.TypeRef, data *templateData) string {
	// Check if struct has a @cpp name override
	name := resolved.Name
	var cppInclude string
	isOmitted := omit.IsType(resolved, "cpp")

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

	// Handle cross-namespace references
	if resolved.Namespace != data.rawNs {
		if isOmitted || targetOutputPath == "" {
			// This struct is omitted or has no @cpp output - it's handwritten.
			// Use the @cpp include path if provided, otherwise we can't include it.
			// Use global namespace prefix :: to avoid ambiguity with synnax::* namespaces
			if cppInclude != "" {
				data.includes.addInternal(cppInclude)
			}
			// Use namespace prefix for handwritten types.
			// If there's an output path, derive the namespace from it (handles x/cpp/telem -> x::telem).
			// Otherwise use the schema namespace directly.
			if targetOutputPath != "" {
				ns := deriveNamespace(targetOutputPath)
				name = fmt.Sprintf("::%s::%s", ns, name)
			} else if resolved.Namespace != "" {
				name = fmt.Sprintf("::%s::%s", resolved.Namespace, name)
			}
		} else {
			// Generated type - include the generated header
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.includes.addInternal(includePath)
			// Use namespace prefix for cross-namespace generated types
			// Must use :: prefix for absolute namespace resolution
			ns := deriveNamespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	}

	return p.buildGenericType(name, typeArgs, &resolved, data)
}

func (p *Plugin) resolveEnumType(resolved resolution.Type, form resolution.EnumForm, data *templateData) string {
	// String enums in C++ are represented as std::string, not as an enum class
	if !form.IsIntEnum {
		data.includes.addSystem("string")
		return "std::string"
	}

	name := resolved.Name

	// For cross-namespace references, we need to add an include and namespace prefix
	if resolved.Namespace != data.rawNs {
		targetOutputPath := enum.FindOutputPath(resolved, data.table, "cpp")
		if targetOutputPath != "" {
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.includes.addInternal(includePath)
			// Use namespace-qualified name with :: prefix for absolute resolution
			ns := deriveNamespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	}

	return name
}

func (p *Plugin) resolveDistinctType(resolved resolution.Type, data *templateData) string {
	// Check for @cpp name override
	name := domain.GetName(resolved, "cpp")

	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.includes.addInternal(includePath)
		}
		// Use namespace-qualified name with :: prefix for absolute resolution
		ns := deriveNamespace(targetOutputPath)
		return fmt.Sprintf("::%s::%s", ns, name)
	}
	return name
}

func (p *Plugin) resolveAliasType(resolved resolution.Type, typeArgs []resolution.TypeRef, data *templateData) string {
	name := domain.GetName(resolved, "cpp")
	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.includes.addInternal(includePath)
			// Use :: prefix for absolute namespace resolution
			ns := deriveNamespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	}
	return p.buildGenericType(name, typeArgs, &resolved, data)
}

func (p *Plugin) buildGenericType(baseName string, typeArgs []resolution.TypeRef, targetType *resolution.Type, data *templateData) string {
	if len(typeArgs) == 0 {
		return baseName
	}

	// Filter type args, skipping those that correspond to defaulted params
	var args []string
	var typeParams []resolution.TypeParam
	if targetType != nil {
		switch form := targetType.Form.(type) {
		case resolution.StructForm:
			typeParams = form.TypeParams
		case resolution.AliasForm:
			typeParams = form.TypeParams
		}

		if len(typeParams) > 0 {
			for i, arg := range typeArgs {
				if i < len(typeParams) && typeParams[i].HasDefault() {
					continue // Skip defaulted type args
				}
				args = append(args, p.typeRefToCpp(arg, data))
			}
		} else {
			for _, arg := range typeArgs {
				args = append(args, p.typeRefToCpp(arg, data))
			}
		}
	} else {
		for _, arg := range typeArgs {
			args = append(args, p.typeRefToCpp(arg, data))
		}
	}

	if len(args) == 0 {
		// All args were for defaulted params - check if we need <>
		// Only add <> if the type is actually a C++ template with all params defaulted
		if targetType != nil && isCppTemplateWithAllDefaults(*targetType) {
			return baseName + "<>"
		}
		return baseName
	}
	return fmt.Sprintf("%s<%s>", baseName, strings.Join(args, ", "))
}

// isCppTemplateWithAllDefaults returns true if the type would be generated as a C++ template
// AND all of its C++ template parameters have defaults.
//
// A struct is a C++ template only if it has type params without explicit defaults.
// Params with explicit defaults are substituted at code generation and don't become template params.
// Optional params (without explicit defaults) DO become template params with implicit std::monostate default.
//
// This function returns true when:
// - The type has at least one type param without explicit default (making it a C++ template)
// - All such params are optional (giving them implicit defaults)
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
	OutputPath   string
	Namespace    string
	ForwardDecls []string
	Structs      []structData
	Enums        []enumData
	TypeDefs     []typeDefData
	Aliases      []aliasData
	SortedDecls  []sortedDeclData
	includes     *includeManager
	table        *resolution.Table
	rawNs        string
}

type sortedDeclData struct {
	IsAlias   bool
	IsStruct  bool
	IsTypeDef bool
	Alias     aliasData
	Struct    structData
	TypeDef   typeDefData
}

type typeDefData struct {
	Name               string
	CppType            string
	IsArrayWrapper     bool     // True if this is an array distinct type that should be a wrapper struct
	IsFixedSizeArray   bool     // True if this is a fixed-size array (uses std::array instead of std::vector)
	ArraySize          int64    // Size of fixed-size array
	ElementType        string   // The element type for array wrappers
	ElementIsPrimitive bool     // True if element type is primitive (allows initializer_list constructor)
	Methods            []string // Custom methods from @cpp methods
	HasProto           bool
	ProtoType          string
	ProtoNamespace     string
	ProtoClass         string
}

type aliasData struct {
	Name       string
	Target     string
	IsGeneric  bool
	TypeParams []string
}

func (d *templateData) HasIncludes() bool {
	return len(d.includes.system) > 0 || len(d.includes.internal) > 0
}

func (d *templateData) SystemIncludes() []string { return d.includes.system }

func (d *templateData) InternalIncludes() []string { return d.includes.internal }

type structData struct {
	Name           string
	Doc            string
	Fields         []fieldData
	TypeParams     []typeParamData
	IsGeneric      bool
	IsAlias        bool
	AliasOf        string
	HasProto       bool
	ProtoType      string
	ProtoNamespace string
	ProtoClass     string
	Methods        []string
	// Inheritance support
	HasExtends   bool
	ExtendsTypes []string // e.g., ["arc::ir::IR", "arc::compiler::Output"]
}

type typeParamData struct {
	Name       string
	HasDefault bool
	Default    string
}

type fieldData struct {
	Name         string
	CppType      string
	Doc          string
	IsSelfRef    bool
	DefaultValue string // Default initializer (e.g., "0", "false", "{}")
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

// hasExplicitPBName returns true if the type has an explicit @pb name directive.
// This is used to determine if an array wrapper has a corresponding proto message.
func hasExplicitPBName(s resolution.Type) bool {
	if domain, ok := s.Domains["pb"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 && expr.Values[0].StringValue != "" {
				return true
			}
		}
	}
	return false
}

// canUseInheritance checks if a struct can use C++ multiple inheritance.
// Returns false if there are omitted fields or field name conflicts between parents.
func canUseInheritance(form resolution.StructForm, table *resolution.Table) bool {
	if len(form.Extends) == 0 {
		return false
	}
	if len(form.OmittedFields) > 0 {
		return false // Can't omit fields with inheritance
	}
	return !hasFieldConflicts(form.Extends, table)
}

// hasFieldConflicts returns true if multiple parents have overlapping field names.
func hasFieldConflicts(extends []resolution.TypeRef, table *resolution.Table) bool {
	if len(extends) < 2 {
		return false
	}
	seen := make(map[string]bool)
	for _, ext := range extends {
		parent, ok := ext.Resolve(table)
		if !ok {
			continue
		}
		for _, f := range resolution.UnifiedFields(parent, table) {
			if seen[f.Name] {
				return true
			}
			seen[f.Name] = true
		}
	}
	return false
}

// resolveExtendsType converts a parent TypeRef to a fully qualified C++ type string.
// It also adds the necessary include for the parent's header.
func (p *Plugin) resolveExtendsType(extendsRef resolution.TypeRef, parent resolution.Type, data *templateData) string {
	name := domain.GetName(parent, "cpp")

	// Check for cross-namespace reference
	if parent.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(parent, "cpp")
		if targetOutputPath != "" {
			// Add include for the parent's header
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.includes.addInternal(includePath)
			// Use namespace-qualified name with :: prefix for absolute resolution
			ns := deriveNamespace(targetOutputPath)
			name = fmt.Sprintf("::%s::%s", ns, name)
		}
	}

	// Handle generic parents with type arguments
	return p.buildGenericType(name, extendsRef.TypeArgs, &parent, data)
}

var templateFuncs = template.FuncMap{
	"join":             strings.Join,
	"toUpper":          strings.ToUpper,
	"toScreamingSnake": toScreamingSnake,
	"toSnakeCase":      toSnakeCase,
}

var fileTemplate = template.Must(template.New("cpp-types").Funcs(templateFuncs).Parse(`// Code generated by oracle. DO NOT EDIT.

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
    {{$td.Name}}() = default;
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
    [[nodiscard]] {{$td.ProtoType}} to_proto() const;
    static std::pair<{{$td.Name}}, x::errors::Error> from_proto(const {{$td.ProtoType}}& pb);
{{- end}}
{{- if $td.Methods}}

    // Custom methods
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
{{- if $s.Doc}}
/// @brief {{$s.Doc}}
{{end}}
{{- if $s.IsAlias}}
{{- if $s.IsGeneric}}template <{{range $j, $p := $s.TypeParams}}{{if $j}}, {{end}}typename {{$p.Name}}{{if $p.HasDefault}} = {{$p.Default}}{{end}}{{end}}>
{{end}}using {{$s.Name}} = {{$s.AliasOf}};
{{- else}}
{{- if $s.IsGeneric}}template <{{range $j, $p := $s.TypeParams}}{{if $j}}, {{end}}typename {{$p.Name}}{{if $p.HasDefault}} = {{$p.Default}}{{end}}{{end}}>
{{end}}struct {{$s.Name}}{{if $s.HasExtends}} : {{range $i, $parent := $s.ExtendsTypes}}{{if $i}}, {{end}}public {{$parent}}{{end}}{{end}} {
{{- range $s.Fields}}
{{- if .Doc}}
    /// @brief {{.Doc}}
{{- end}}
    {{.CppType}} {{.Name}}{{if .DefaultValue}} = {{.DefaultValue}}{{end}};
{{- end}}

    static {{$s.Name}} parse(x::json::Parser parser);
    [[nodiscard]] x::json::json to_json() const;
{{- if $s.HasProto}}

    using proto_type = {{$s.ProtoType}};
    [[nodiscard]] {{$s.ProtoType}} to_proto() const;
    static std::pair<{{$s.Name}}, x::errors::Error> from_proto(const {{$s.ProtoType}}& pb);
{{- end}}
{{- if $s.Methods}}

    // Custom methods
{{- range $s.Methods}}
    {{.}};
{{- end}}
{{- end}}
};
{{- end}}
{{- end}}
{{- end}}
}
`))
