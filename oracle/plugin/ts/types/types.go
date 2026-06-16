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
	"sort"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/domain/ontology"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/plugin/ts/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/ts/internal/paths"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

type Plugin struct{ Options Options }

type Options struct {
	OutputPath      string
	FileNamePattern string
	GenerateTypes   bool
}

func DefaultOptions() Options {
	return Options{
		OutputPath:      "{{.Namespace}}",
		FileNamePattern: "types.gen.ts",
		GenerateTypes:   true,
	}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string { return "ts/types" }

func (p *Plugin) Domains() []string { return nil }

func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) Check(req *plugin.Request) error { return nil }

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	typeDefCollector := framework.NewCollector("ts", req)
	if err := typeDefCollector.AddAll(req.Resolutions.DistinctTypes()); err != nil {
		return nil, err
	}
	if err := typeDefCollector.AddAll(req.Resolutions.AliasTypes()); err != nil {
		return nil, err
	}

	structCollector, err := framework.CollectStructs("ts", req)
	if err != nil {
		return nil, err
	}

	unionCollector := framework.NewCollector("ts", req)
	if err := unionCollector.AddAll(req.Resolutions.UnionTypes()); err != nil {
		return nil, err
	}

	enumCollector := framework.NewCollector("ts", req).
		WithPathFunc(func(typ resolution.Type) string { return output.GetPath(typ, "ts") }).
		WithSkipFunc(nil)
	for _, e := range enum.CollectWithOwnOutput(req.Resolutions.EnumTypes(), "ts") {
		if err := enumCollector.Add(e); err != nil {
			return nil, err
		}
	}

	err = structCollector.ForEach(func(outputPath string, structs []resolution.Type) error {
		enums := enum.CollectReferenced(structs, req.Resolutions)
		if enumCollector.Has(outputPath) {
			enums = framework.MergeTypesByName(enums, enumCollector.Remove(outputPath))
		}
		if len(structs) > 0 {
			namespace := structs[0].Namespace
			enums = framework.MergeTypesByName(enums, enum.CollectNamespaceEnums(namespace, outputPath, req.Resolutions, "ts", nil))
		}
		var typeDefs []resolution.Type
		if typeDefCollector.Has(outputPath) {
			typeDefs = typeDefCollector.Remove(outputPath)
		}
		var unions []resolution.Type
		if unionCollector.Has(outputPath) {
			unions = unionCollector.Remove(outputPath)
		}
		content, err := p.generateFile(structs[0].Namespace, outputPath, structs, enums, typeDefs, unions, req)
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

	err = unionCollector.ForEach(func(outputPath string, unions []resolution.Type) error {
		var typeDefs []resolution.Type
		if typeDefCollector.Has(outputPath) {
			typeDefs = typeDefCollector.Remove(outputPath)
		}
		namespace := unions[0].Namespace
		enums := enum.CollectReferenced(unions, req.Resolutions)
		if enumCollector.Has(outputPath) {
			enums = framework.MergeTypesByName(enums, enumCollector.Remove(outputPath))
		}
		enums = framework.MergeTypesByName(enums, enum.CollectNamespaceEnums(namespace, outputPath, req.Resolutions, "ts", nil))
		content, err := p.generateFile(namespace, outputPath, nil, enums, typeDefs, unions, req)
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

	err = enumCollector.ForEach(func(outputPath string, enums []resolution.Type) error {
		var typeDefs []resolution.Type
		if typeDefCollector.Has(outputPath) {
			typeDefs = typeDefCollector.Remove(outputPath)
		}
		content, err := p.generateFile(enums[0].Namespace, outputPath, nil, enums, typeDefs, nil, req)
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

	err = typeDefCollector.ForEach(func(outputPath string, typeDefs []resolution.Type) error {
		var enums []resolution.Type
		if len(typeDefs) > 0 {
			namespace := typeDefs[0].Namespace
			enums = enum.CollectNamespaceEnums(namespace, outputPath, req.Resolutions, "ts", nil)
		}
		content, err := p.generateFile(typeDefs[0].Namespace, outputPath, nil, enums, typeDefs, nil, req)
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

// hasNonPrimitiveDependency returns true if a type definition has dependencies
// on non-primitive types (i.e., references other schema types that need to be
// declared before this type). This is used to determine whether a distinct type
// should be included in topological sorting.
func hasNonPrimitiveDependency(typ resolution.Type) bool {
	var checkRef func(ref resolution.TypeRef) bool
	checkRef = func(ref resolution.TypeRef) bool {
		if ref.Name == "" || ref.IsTypeParam() {
			return false
		}
		// Primitives have no schema dependencies
		if resolution.IsPrimitive(ref.Name) {
			return false
		}
		// For Array and Map, check if their type arguments have dependencies
		if ref.Name == "Array" || ref.Name == "Map" {
			return slices.ContainsFunc(ref.TypeArgs, checkRef)
		}
		// Any other named type is a schema dependency
		return true
	}

	switch form := typ.Form.(type) {
	case resolution.DistinctForm:
		return checkRef(form.Base)
	case resolution.AliasForm:
		return checkRef(form.Target)
	}
	return false
}

func (p *Plugin) generateFile(
	namespace string,
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	typeDefs []resolution.Type,
	unions []resolution.Type,
	req *plugin.Request,
) ([]byte, error) {
	data := &templateData{
		Namespace:     namespace,
		OutputPath:    outputPath,
		Request:       req,
		Structs:       make([]structData, 0, len(structs)),
		Enums:         make([]enumData, 0, len(enums)),
		TypeDefs:      make([]typeDefData, 0, len(typeDefs)),
		SortedDecls:   make([]sortedDeclData, 0),
		GenerateTypes: p.Options.GenerateTypes,
		Manager:       imports.NewManager(),
	}
	skip := func(s resolution.Type) bool { return omit.IsType(s, "ts") }
	rawKeyFields := key.Collect(structs, req.Resolutions, skip)
	data.Ontology = p.extractOntology(structs, rawKeyFields, skip, req.Resolutions)
	if data.Ontology != nil {
		data.AddImport("@/ontology", "ontology")
	}

	// Separate type definitions based on whether they have dependencies on schema types.
	// Distinct types with only primitive bases can be output first (no sorting needed).
	// Distinct types with non-primitive bases (e.g., Params Param[]) must be included
	// in topological sorting along with aliases and structs.
	var primitiveTypeDefs []resolution.Type
	var dependentTypeDefs []resolution.Type
	for _, td := range typeDefs {
		switch td.Form.(type) {
		case resolution.AliasForm:
			dependentTypeDefs = append(dependentTypeDefs, td)
		case resolution.DistinctForm:
			if hasNonPrimitiveDependency(td) {
				dependentTypeDefs = append(dependentTypeDefs, td)
			} else {
				primitiveTypeDefs = append(primitiveTypeDefs, td)
			}
		default:
			primitiveTypeDefs = append(primitiveTypeDefs, td)
		}
	}

	for _, td := range primitiveTypeDefs {
		data.TypeDefs = append(data.TypeDefs, p.processTypeDef(td, data))
	}

	for _, e := range enums {
		data.Enums = append(data.Enums, p.processEnum(e))
	}

	// Combine structs and dependent typedefs for topological sorting.
	// IMPORTANT: Structs come first so that when there's a cycle, typedefs
	// (like array types) are placed after their element types. Array typedefs
	// can't use getters, so they must come after their element types are defined.
	var combinedTypes []resolution.Type
	combinedTypes = append(combinedTypes, structs...)
	combinedTypes = append(combinedTypes, dependentTypeDefs...)
	combinedTypes = append(combinedTypes, unions...)

	// Sort topologically so dependencies come before dependents
	sortedTypes := req.Resolutions.TopologicalSort(combinedTypes)

	// Build declaration order map for forward reference detection.
	// When a struct field references a type declared later, we need to use
	// a getter for lazy evaluation.
	declOrder := make(map[string]int, len(sortedTypes))
	for i, typ := range sortedTypes {
		declOrder[typ.QualifiedName] = i
	}
	data.DeclOrder = declOrder

	// Process in sorted order
	for i, typ := range sortedTypes {
		data.CurrentDeclIndex = i
		switch form := typ.Form.(type) {
		case resolution.AliasForm:
			// Generic aliases need full struct treatment for type params
			if form.IsGeneric() {
				data.SortedDecls = append(data.SortedDecls, sortedDeclData{
					IsStruct: true,
					Struct:   p.processStruct(typ, req.Resolutions, data),
				})
			} else {
				data.SortedDecls = append(data.SortedDecls, sortedDeclData{
					IsTypeDef: true,
					TypeDef:   p.processTypeDef(typ, data),
				})
			}
		case resolution.DistinctForm:
			// Distinct types with non-primitive dependencies (e.g., Params Param[])
			data.SortedDecls = append(data.SortedDecls, sortedDeclData{
				IsTypeDef: true,
				TypeDef:   p.processTypeDef(typ, data),
			})
		case resolution.StructForm:
			data.SortedDecls = append(data.SortedDecls, sortedDeclData{
				IsStruct: true,
				Struct:   p.processStruct(typ, req.Resolutions, data),
			})
		case resolution.UnionForm:
			data.SortedDecls = append(data.SortedDecls, sortedDeclData{
				IsUnion: true,
				Union:   p.processUnion(typ, req.Resolutions, data),
			})
		}
	}

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func findFieldTypeOverride(structs []resolution.Type, fieldName, domainName string) string {
	for _, s := range structs {
		form, ok := s.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		for _, f := range form.Fields {
			if f.Name == fieldName {
				if override := getFieldTypeOverride(f, domainName); override != "" {
					return override
				}
			}
		}
	}
	return ""
}

func (p *Plugin) extractOntology(
	structs []resolution.Type,
	keyFields []key.Field,
	skip ontology.SkipFunc,
	table *resolution.Table,
) *ontologyData {
	data := ontology.Extract(structs, keyFields, skip)
	if data == nil {
		return nil
	}
	keyType := lo.Capitalize(fieldCamel(data.KeyField.Name))
	primitive := data.KeyField.Primitive
	if override := findFieldTypeOverride(structs, data.KeyField.Name, "ts"); override != "" {
		primitive = override
	}
	// Also check if the key field's type itself has a @ts type override (e.g., Key uint64 { @ts type string })
	if override := findKeyTypeTypeOverride(structs, data.KeyField.Name, table); override != "" {
		primitive = override
	}
	keyZeroValue := primitiveZeroValue(primitive)
	return &ontologyData{
		TypeName:     data.TypeName,
		KeyType:      keyType,
		KeyZeroValue: keyZeroValue,
	}
}

func findKeyTypeTypeOverride(structs []resolution.Type, keyFieldName string, table *resolution.Table) string {
	for _, s := range structs {
		form, ok := s.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		for _, f := range form.Fields {
			if f.Name == keyFieldName {
				if typ, ok := table.Get(f.Type.Name); ok {
					// Check if the type has a @ts type override
					return getTypeTypeOverride(typ, "ts")
				}
			}
		}
	}
	return ""
}

func (p *Plugin) processEnum(e resolution.Type) enumData {
	form, ok := e.Form.(resolution.EnumForm)
	if !ok {
		return enumData{Name: e.Name}
	}
	values := make([]enumValueData, 0, len(form.Values))
	for _, v := range form.Values {
		values = append(values, enumValueData{
			Name:      v.Name,
			Value:     v.StringValue(),
			IntValue:  v.IntValue(),
			IsIntEnum: form.IsIntEnum,
		})
	}
	ed := enumData{Name: e.Name, Values: values, IsIntEnum: form.IsIntEnum}
	if tsDomain, ok := e.Domains["ts"]; ok {
		for _, expr := range tsDomain.Expressions {
			if expr.Name == "literals" {
				ed.GenerateLiterals = true
			}
		}
	}
	return ed
}

func (p *Plugin) processTypeDef(td resolution.Type, data *templateData) typeDefData {
	toNumber := false
	toString := false
	if tsDomain, ok := td.Domains["ts"]; ok {
		for _, expr := range tsDomain.Expressions {
			switch expr.Name {
			case "to_number":
				toNumber = true
			case "to_string":
				toString = true
			}
		}
	}

	switch form := td.Form.(type) {
	case resolution.DistinctForm:
		if typeOverride := getTypeTypeOverride(td, "ts"); typeOverride != "" {
			zodType := primitiveToZod(typeOverride, data)
			if validateDomain, ok := td.Domains["validate"]; ok {
				result := p.applyValidation(zodType, validateDomain, nil, form.Base, td.Name, data.Request.Resolutions, data, typeOverride)
				zodType = result.ZodType
			}
			if toNumber {
				zodType = fmt.Sprintf("%s.or(z.string().refine((v) => !isNaN(Number(v))).transform(Number))", zodType)
			}
			if toString {
				zodType = fmt.Sprintf("%s.or(z.number().transform(String).or(z.bigint().transform(String)))", zodType)
			}
			return typeDefData{
				Name:    td.Name,
				TSName:  td.Name,
				TSType:  primitiveToTS(typeOverride),
				ZodType: zodType,
			}
		}
		var zodType string
		if isArrayTypeRef(form.Base) && len(form.Base.TypeArgs) > 0 {
			elemZod := p.typeRefToZod(&form.Base.TypeArgs[0], data.Request.Resolutions, data)
			if form.Base.ArraySize != nil {
				elements := make([]string, *form.Base.ArraySize)
				for i := range elements {
					elements[i] = elemZod
				}
				zodType = fmt.Sprintf("z.tuple([%s])", strings.Join(elements, ", "))
			} else {
				addXImport(data, xImport{name: "array", submodule: "array"})
				zodType = fmt.Sprintf("array.nullishToEmpty(%s)", elemZod)
			}
		} else {
			zodType = p.typeDefBaseToZod(&form.Base, data)
		}
		if validateDomain, ok := td.Domains["validate"]; ok {
			result := p.applyValidation(zodType, validateDomain, nil, form.Base, td.Name, data.Request.Resolutions, data, "")
			zodType = result.ZodType
		}
		if toNumber {
			zodType = fmt.Sprintf("%s.or(z.string().refine((v) => !isNaN(Number(v))).transform(Number))", zodType)
		}
		if toString {
			zodType = fmt.Sprintf("%s.or(z.number().transform(String))", zodType)
		}
		return typeDefData{
			Name:    td.Name,
			TSName:  td.Name,
			TSType:  p.typeDefBaseToTS(&form.Base, data),
			ZodType: zodType,
		}
	case resolution.AliasForm:
		var zodType string
		if isArrayTypeRef(form.Target) && len(form.Target.TypeArgs) > 0 {
			elemZod := p.typeRefToZod(&form.Target.TypeArgs[0], data.Request.Resolutions, data)
			if form.Target.ArraySize != nil {
				elements := make([]string, *form.Target.ArraySize)
				for i := range elements {
					elements[i] = elemZod
				}
				zodType = fmt.Sprintf("z.tuple([%s])", strings.Join(elements, ", "))
			} else {
				addXImport(data, xImport{name: "array", submodule: "array"})
				zodType = fmt.Sprintf("array.nullishToEmpty(%s)", elemZod)
			}
		} else {
			zodType = p.typeDefBaseToZod(&form.Target, data)
		}
		if validateDomain, ok := td.Domains["validate"]; ok {
			result := p.applyValidation(zodType, validateDomain, nil, form.Target, td.Name, data.Request.Resolutions, data, "")
			zodType = result.ZodType
		}
		return typeDefData{
			Name:    td.Name,
			TSName:  td.Name,
			TSType:  p.typeDefBaseToTS(&form.Target, data),
			ZodType: zodType,
		}
	default:
		return typeDefData{Name: td.Name, TSName: td.Name, TSType: "unknown", ZodType: "z.unknown()"}
	}
}

func (p *Plugin) typeDefBaseToZod(typeRef *resolution.TypeRef, data *templateData) string {
	if typeRef == nil {
		return "z.unknown()"
	}
	return p.typeRefToZod(typeRef, data.Request.Resolutions, data)
}

func (p *Plugin) typeDefBaseToTS(typeRef *resolution.TypeRef, data *templateData) string {
	if typeRef == nil {
		return "unknown"
	}
	return p.typeRefToTS(typeRef, data.Request.Resolutions, data, false)
}

// isExtendBase reports whether another type's generated zod schema calls
// .extend on entry's schema: struct extends bases, union shared bases, and
// union variant payloads. Such schemas must stay ZodObjects, since the
// z.ZodType annotation used to break recursive inference has no .extend.
func isExtendBase(entry resolution.Type, table *resolution.Table) bool {
	matches := func(ref resolution.TypeRef) bool {
		resolved, ok := ref.Resolve(table)
		return ok && resolved.QualifiedName == entry.QualifiedName
	}
	for _, typ := range table.Types {
		switch form := typ.Form.(type) {
		case resolution.StructForm:
			if slices.ContainsFunc(form.Extends, matches) {
				return true
			}
		case resolution.UnionForm:
			if slices.ContainsFunc(form.Extends, matches) {
				return true
			}
			if slices.ContainsFunc(
				form.Variants,
				func(v resolution.UnionVariant) bool { return matches(v.Type) },
			) {
				return true
			}
		}
	}
	return false
}

func (p *Plugin) processStruct(entry resolution.Type, table *resolution.Table, data *templateData) structData {
	if aliasForm, isAlias := entry.Form.(resolution.AliasForm); isAlias {
		sd := structData{
			Name:          entry.Name,
			TSName:        entry.Name,
			Doc:           doc.Get(entry.Domains),
			IsGeneric:     aliasForm.IsGeneric(),
			IsSingleParam: len(aliasForm.TypeParams) == 1,
			IsAlias:       true,
		}
		if tsDomain, ok := entry.Domains["ts"]; ok {
			for _, expr := range tsDomain.Expressions {
				switch expr.Name {
				case "use_input":
					sd.UseInput = true
				case "omit":
					sd.Handwritten = true
				case "name":
					if len(expr.Values) > 0 {
						sd.TSName = expr.Values[0].StringValue
					}
				}
			}
		}
		if sd.Handwritten {
			return sd
		}
		for _, tp := range aliasForm.TypeParams {
			sd.TypeParams = append(sd.TypeParams, p.processTypeParam(tp, table))
		}
		for _, tp := range sd.TypeParams {
			if tp.IsJSON || strings.Contains(tp.Constraint, "record.") || strings.Contains(tp.Default, "record.") {
				addXImport(data, xImport{name: "record", submodule: "record"})
				break
			}
		}
		sd.AllParamsOptional = true
		for _, tp := range sd.TypeParams {
			if !tp.HasDefault {
				sd.AllParamsOptional = false
				break
			}
		}
		if isArrayTypeRef(aliasForm.Target) && len(aliasForm.Target.TypeArgs) > 0 {
			elemZod := p.typeRefToZod(&aliasForm.Target.TypeArgs[0], table, data)
			if aliasForm.Target.ArraySize != nil {
				elements := make([]string, *aliasForm.Target.ArraySize)
				for i := range elements {
					elements[i] = elemZod
				}
				sd.AliasOf = fmt.Sprintf("z.tuple([%s])", strings.Join(elements, ", "))
			} else {
				addXImport(data, xImport{name: "array", submodule: "array"})
				sd.AliasOf = fmt.Sprintf("array.nullishToEmpty(%s)", elemZod)
			}
		} else {
			sd.AliasOf = p.typeRefToZod(&aliasForm.Target, table, data)
		}
		return sd
	}

	form, ok := entry.Form.(resolution.StructForm)
	if !ok {
		return structData{Name: entry.Name, TSName: entry.Name}
	}

	sd := structData{
		Name:          entry.Name,
		TSName:        entry.Name,
		Doc:           doc.Get(entry.Domains),
		IsGeneric:     form.IsGeneric(),
		IsSingleParam: len(form.TypeParams) == 1,
		IsAlias:       false,
		// Extend bases must keep a ZodObject schema so extenders can .extend
		// it; the recursive-interface branch would annotate the schema as
		// z.ZodType, which has no .extend. Their inference cycles are broken
		// by annotated forward-reference getters instead.
		IsRecursive: form.IsRecursive && !isExtendBase(entry, table),
	}
	if tsDomain, ok := entry.Domains["ts"]; ok {
		for _, expr := range tsDomain.Expressions {
			switch expr.Name {
			case "use_input":
				sd.UseInput = true
			case "omit":
				sd.Handwritten = true
			case "concrete_types":
				sd.ConcreteTypes = true
			case "coalesce_type_params":
				sd.CoalesceTypeParams = true
			case "name":
				if len(expr.Values) > 0 {
					sd.TSName = expr.Values[0].StringValue
				}
			}
		}
	}
	if sd.Handwritten {
		return sd
	}
	for _, tp := range form.TypeParams {
		sd.TypeParams = append(sd.TypeParams, p.processTypeParam(tp, table))
	}
	for _, tp := range sd.TypeParams {
		if tp.IsJSON || strings.Contains(tp.Constraint, "record.") || strings.Contains(tp.Default, "record.") {
			addXImport(data, xImport{name: "record", submodule: "record"})
			break
		}
	}
	for _, tp := range sd.TypeParams {
		if tp.IsPrimitiveConstrained {
			addXImport(data, xImport{name: "numeric", submodule: "numeric"})
			break
		}
	}

	sd.AllParamsOptional = true
	for _, tp := range sd.TypeParams {
		if !tp.HasDefault {
			sd.AllParamsOptional = false
			break
		}
	}

	if sd.IsGeneric {
		sd.IsPrimitiveConstrainedGeneric = true
		for _, tp := range sd.TypeParams {
			if !tp.IsPrimitiveConstrained || !tp.HasDefault {
				sd.IsPrimitiveConstrainedGeneric = false
				break
			}
		}
	}

	// A domain removal (-@domain) cannot be expressed through Zod extend/omit
	// chaining — the parent schema still carries the domain — so flatten via
	// UnifiedFields. Typeless overrides are already resolved by the analyzer.
	if len(form.Extends) > 0 && !resolver.HasDomainOmissions(form) {
		// Collect all parent schema names for merge chaining
		var allParentsValid = true
		for _, extendsRef := range form.Extends {
			parentType, ok := extendsRef.Resolve(table)
			if !ok {
				allParentsValid = false
				break
			}
			parentForm, isStruct := parentType.Form.(resolution.StructForm)
			if !isStruct {
				allParentsValid = false
				break
			}

			parentTSName := domain.GetName(parentType, "ts")
			schemaName := camelCase(parentTSName) + "Z"

			if parentType.Namespace != data.Namespace {
				ns := parentType.Namespace
				targetOutputPath := output.GetPath(parentType, "ts")
				if targetOutputPath == "" {
					targetOutputPath = ns
				}
				data.AddImport(paths.CalculateImport(data.OutputPath, targetOutputPath), ns)
				schemaName = ns + "." + schemaName
			}

			parentInfo := extendsParentInfo{
				Name:     schemaName,
				TypeName: parentTSName,
			}

			if parentForm.IsGeneric() {
				parentInfo.IsGeneric = true
				for _, tp := range parentForm.TypeParams {
					parentInfo.SchemaArgs = append(parentInfo.SchemaArgs, camelCase(tp.Name))
				}
			}

			sd.ExtendsParents = append(sd.ExtendsParents, parentInfo)
		}

		if allParentsValid && len(sd.ExtendsParents) > 0 {
			sd.HasExtends = true
			sd.ExtendsName = sd.ExtendsParents[0].Name
			sd.ExtendsTypeName = sd.ExtendsParents[0].TypeName
			sd.ExtendsParentIsGeneric = sd.ExtendsParents[0].IsGeneric
			sd.ExtendsParentSchemaArgs = sd.ExtendsParents[0].SchemaArgs

			for _, f := range form.OmittedFields {
				sd.OmittedFields = append(sd.OmittedFields, fieldCamel(f))
			}

			parentFields := make(map[string]resolution.Field)
			for _, extendsRef := range form.Extends {
				parentType, _ := extendsRef.Resolve(table)
				for _, pf := range resolution.UnifiedFields(parentType, table) {
					if _, exists := parentFields[pf.Name]; !exists {
						parentFields[pf.Name] = pf // First parent wins
					}
				}
			}

			for _, field := range form.Fields {
				parentField, existsInParent := parentFields[field.Name]
				if existsInParent {
					if isFieldUnchanged(parentField, field) {
						continue
					} else if isOnlyOptionalityChange(parentField, field) {
						sd.PartialFields = append(sd.PartialFields, fieldData{TSName: fieldCamel(field.Name)})
					} else {
						sd.OmittedFields = append(sd.OmittedFields, fieldCamel(field.Name))
						sd.ExtendFields = append(sd.ExtendFields, p.processField(field, entry, table, data, sd.ConcreteTypes))
					}
				} else {
					sd.ExtendFields = append(sd.ExtendFields, p.processField(field, entry, table, data, sd.ConcreteTypes))
				}
			}

			if sd.ConcreteTypes && len(sd.PartialFields) > 0 {
				addXImport(data, xImport{name: "optional", submodule: "optional"})
			}
			if sd.CoalesceTypeParams {
				computeCoalescedTypes(&sd)
			}
			return sd
		}
	}

	allFields := resolution.UnifiedFields(entry, table)
	sd.Fields = make([]fieldData, 0, len(allFields))
	optionalTypeParams := make(set.Set[string])
	for _, tp := range form.TypeParams {
		if tp.Optional {
			optionalTypeParams.Add(tp.Name)
		}
	}

	for _, field := range allFields {
		fd := p.processField(field, entry, table, data, sd.ConcreteTypes)
		sd.Fields = append(sd.Fields, fd)

		if sd.ConcreteTypes && field.Type.IsTypeParam() &&
			field.Type.TypeParam != nil &&
			optionalTypeParams.Contains(field.Type.TypeParam.Name) {
			tp := field.Type.TypeParam
			effectiveRef := tp.Constraint
			if effectiveRef == nil && tp.Default != nil && !tp.Optional {
				effectiveRef = tp.Default
			}
			sd.ConditionalFields = append(sd.ConditionalFields, conditionalFieldData{
				Field:              fd,
				TypeParamName:      field.Type.TypeParam.Name,
				NeverType:          "z.ZodNever",
				FallbackSchemaType: fallbackSchemaTypeForConstraint(effectiveRef, table),
			})
		} else {
			sd.BaseFields = append(sd.BaseFields, fd)
		}
	}
	if sd.CoalesceTypeParams {
		computeCoalescedTypes(&sd)
	}
	return sd
}

func isFieldUnchanged(parent, child resolution.Field) bool {
	childIsOptional := child.IsOptional || child.IsHardOptional
	parentIsOptional := parent.IsOptional || parent.IsHardOptional
	if childIsOptional != parentIsOptional {
		return false
	}
	if hasPreserveCase(parent) != hasPreserveCase(child) {
		return false
	}
	if hasPreserveKeys(parent) != hasPreserveKeys(child) {
		return false
	}
	if domain.GetStringFromField(parent, "ts", "pick") !=
		domain.GetStringFromField(child, "ts", "pick") {
		return false
	}
	if !sameDefault(parent.Default, child.Default) {
		return false
	}
	return sameBaseType(parent.Type, child.Type)
}

// sameDefault reports whether two field defaults are structurally equal. A child
// override that introduces or changes a default relative to its parent is a
// real change, so the field must be re-emitted rather than inherited verbatim.
func sameDefault(a, b *resolution.ExpressionValue) bool {
	if a == nil || b == nil {
		return a == b
	}
	if a.Kind != b.Kind {
		return false
	}
	switch a.Kind {
	case resolution.ValueKindString:
		return a.StringValue == b.StringValue
	case resolution.ValueKindInt:
		return a.IntValue == b.IntValue
	case resolution.ValueKindFloat:
		return a.FloatValue == b.FloatValue
	case resolution.ValueKindBool:
		return a.BoolValue == b.BoolValue
	case resolution.ValueKindIdent:
		return a.IdentValue == b.IdentValue
	case resolution.ValueKindArray:
		if len(a.Elements) != len(b.Elements) {
			return false
		}
		for i := range a.Elements {
			if !sameDefault(&a.Elements[i], &b.Elements[i]) {
				return false
			}
		}
		return true
	case resolution.ValueKindStruct:
		if len(a.Fields) != len(b.Fields) {
			return false
		}
		for i := range a.Fields {
			if a.Fields[i].Name != b.Fields[i].Name {
				return false
			}
			if !sameDefault(&a.Fields[i].Value, &b.Fields[i].Value) {
				return false
			}
		}
		return true
	}
	return false
}

func isOnlyOptionalityChange(parent, child resolution.Field) bool {
	childIsOptional := child.IsOptional || child.IsHardOptional
	parentIsOptional := parent.IsOptional || parent.IsHardOptional
	if !childIsOptional || parentIsOptional {
		return false
	}
	if hasPreserveCase(parent) != hasPreserveCase(child) {
		return false
	}
	if hasPreserveKeys(parent) != hasPreserveKeys(child) {
		return false
	}
	return sameBaseType(parent.Type, child.Type)
}

func isArrayTypeRef(r resolution.TypeRef) bool {
	return r.Name == "Array"
}

func hasPreserveCase(field resolution.Field) bool {
	return hasTSExpression(field, "preserve_case", "no_preserve_case")
}

// hasPreserveKeys reports whether a map field opts into key-only case
// preservation: the map's keys stay verbatim while its values still undergo
// wire case conversion.
func hasPreserveKeys(field resolution.Field) bool {
	return hasTSExpression(field, "preserve_keys", "no_preserve_keys")
}

func hasTSExpression(field resolution.Field, name, negation string) bool {
	tsDomain, ok := field.Domains["ts"]
	if !ok {
		return false
	}
	for _, expr := range tsDomain.Expressions {
		if expr.Name == negation {
			return false
		}
		if expr.Name == name {
			return true
		}
	}
	return false
}

func sameBaseType(a, b resolution.TypeRef) bool {
	if a.Name != b.Name {
		return false
	}
	if len(a.TypeArgs) != len(b.TypeArgs) {
		return false
	}
	for i := range a.TypeArgs {
		if !sameBaseType(a.TypeArgs[i], b.TypeArgs[i]) {
			return false
		}
	}
	return true
}

func computeCoalescedTypes(sd *structData) {
	for i := range sd.Fields {
		sd.Fields[i].CoalescedTSType = coalesceTSType(sd.Fields[i].TSType, sd.TypeParams)
	}
	for i := range sd.BaseFields {
		sd.BaseFields[i].CoalescedTSType = coalesceTSType(sd.BaseFields[i].TSType, sd.TypeParams)
	}
	for i := range sd.ExtendFields {
		sd.ExtendFields[i].CoalescedTSType = coalesceTSType(sd.ExtendFields[i].TSType, sd.TypeParams)
	}
	for i := range sd.ConditionalFields {
		sd.ConditionalFields[i].Field.CoalescedTSType = coalesceTSType(sd.ConditionalFields[i].Field.TSType, sd.TypeParams)
	}
}

// camelCase converts a generated type or schema-const identifier to camelCase,
// keeping known acronyms upper-cased after the first word ("BaseAOChannel" ->
// "baseAOChannel", "AIVoltageRMSChannel" -> "aiVoltageRMSChannel"). It is the
// template helper behind every "<name>Z" const, so const names stay consistent
// with their acronym-aware type names. Wire field keys must NOT use this; they go
// through fieldCamel to match the JSON codec's naive snake/camel conversion.
func camelCase(s string) string {
	return casing.CamelAcronym(s)
}

// fieldCamel converts a field identifier to camelCase using the naive conversion
// the JSON codec's snake/camel round-trip relies on, preserving only a trailing
// acronym run of two or more uppercase letters in the source ("ClientXY" ->
// "clientXY", "EntityID" -> "entityID"). Use this for wire field keys,
// discriminators, and key-field names, never for type or schema-const identifiers.
func fieldCamel(s string) string {
	if s == "" {
		return s
	}
	base := lo.CamelCase(s)
	end := len(s)
	runStart := end
	for runStart > 0 {
		c := s[runStart-1]
		if c >= 'A' && c <= 'Z' {
			runStart--
			continue
		}
		break
	}
	runLen := end - runStart
	if runLen < 2 || runStart == 0 {
		return base
	}
	if len(base) < runLen {
		return base
	}
	return base[:len(base)-runLen] + s[runStart:]
}

// parentSchemaName resolves a base or payload type reference to its TS schema
// const name (e.g. "baseAIChannelZ"), importing and namespace-qualifying it when
// it lives in another output. It reports false when the reference does not
// resolve to a struct that can be composed. The resolution mirrors the
// struct-extends path so union variants and structs compose identically.
func parentSchemaName(ref resolution.TypeRef, table *resolution.Table, data *templateData) (string, bool) {
	parent, ok := ref.Resolve(table)
	if !ok {
		return "", false
	}
	if _, isStruct := parent.Form.(resolution.StructForm); !isStruct {
		return "", false
	}
	name := camelCase(domain.GetName(parent, "ts")) + "Z"
	if parent.Namespace != data.Namespace {
		ns := parent.Namespace
		targetOutputPath := output.GetPath(parent, "ts")
		if targetOutputPath == "" {
			targetOutputPath = ns
		}
		data.AddImport(paths.CalculateImport(data.OutputPath, targetOutputPath), ns)
		name = ns + "." + name
	}
	return name, true
}

func coalesceTSType(tsType string, typeParams []typeParamData) string {
	sorted := make([]typeParamData, len(typeParams))
	copy(sorted, typeParams)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Name) > len(sorted[j].Name)
	})
	result := tsType
	for _, tp := range sorted {
		result = strings.ReplaceAll(result, tp.Name, `S["`+camelCase(tp.Name)+`"]`)
	}
	return result
}

func (p *Plugin) processTypeParam(tp resolution.TypeParam, table *resolution.Table) typeParamData {
	tpd := typeParamData{Name: tp.Name, Constraint: "z.ZodType"}
	if tp.Constraint != nil {
		if resolution.IsPrimitive(tp.Constraint.Name) && tp.Constraint.Name == "record" {
			tpd.IsJSON = true
			tpd.Constraint = "z.ZodType<record.Unknown>"
		}
		if resolution.IsPrimitive(tp.Constraint.Name) && tp.Constraint.Name == "string" {
			tpd.Constraint = "z.ZodType<string>"
		}
		if tp.Constraint.Name == "numeric" {
			tpd.IsPrimitiveConstrained = true
			tpd.BareConstraint = "numeric.Value"
		}
		resolved, ok := tp.Constraint.Resolve(table)
		if ok {
			if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
				enumTypeName := lo.Capitalize(fieldCamel(resolved.Name))
				tpd.Constraint = "z.ZodType<" + enumTypeName + ">"
			}
		}
	}
	if tp.Optional {
		tpd.HasDefault = true
		tpd.Default = "z.ZodNever"
		tpd.DefaultValue = "z.unknown()"
	} else if tp.Default != nil {
		tpd.HasDefault = true
		// Handle enum defaults
		resolved, ok := tp.Default.Resolve(table)
		if ok {
			if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
				enumZodName := camelCase(resolved.Name) + "Z"
				tpd.Default = "typeof " + enumZodName
				tpd.DefaultValue = enumZodName
			} else {
				tpd.Default = defaultToTS(tp.Default.Name)
				tpd.DefaultValue = defaultValueToTS(tp.Default.Name)
			}
		} else {
			tpd.Default = defaultToTS(tp.Default.Name)
			tpd.DefaultValue = defaultValueToTS(tp.Default.Name)
		}
		if tpd.IsPrimitiveConstrained {
			tpd.BareDefault = primitiveToBareTS(tp.Default.Name)
		}
	}
	return tpd
}

// primitiveToBareTS maps an Oracle primitive name to its bare TypeScript type
// (e.g. number, bigint, string, boolean) for use in interface signatures of
// primitive-constrained generics. Returns the empty string for primitives with
// no bare-TS mapping.
func primitiveToBareTS(name string) string {
	switch name {
	case "string", "uuid":
		return "string"
	case "bool":
		return "boolean"
	case "int8", "int16", "int32", "int64",
		"uint8", "uint12", "uint16", "uint20", "uint32", "uint64",
		"float32", "float64":
		return "number"
	}
	return ""
}

type typeParamMapping struct {
	zodType  string
	zodValue string
}

var typeParamMappings = map[string]typeParamMapping{
	"never":     {zodType: "z.ZodNever", zodValue: "z.unknown()"},
	"string":    {zodType: "z.ZodString", zodValue: "z.string()"},
	"bool":      {zodType: "z.ZodBoolean", zodValue: "z.boolean()"},
	"int8":      {zodType: "z.ZodNumber", zodValue: "z.number()"},
	"int16":     {zodType: "z.ZodNumber", zodValue: "z.number()"},
	"int32":     {zodType: "z.ZodNumber", zodValue: "z.number()"},
	"int64":     {zodType: "z.ZodNumber", zodValue: "z.number()"},
	"uint8":     {zodType: "z.ZodNumber", zodValue: "z.number()"},
	"uint16":    {zodType: "z.ZodNumber", zodValue: "z.number()"},
	"uint32":    {zodType: "z.ZodNumber", zodValue: "z.number()"},
	"uint64":    {zodType: "z.ZodNumber", zodValue: "z.number()"},
	"float32":   {zodType: "z.ZodNumber", zodValue: "z.number()"},
	"float64":   {zodType: "z.ZodNumber", zodValue: "z.number()"},
	"uuid":      {zodType: "z.ZodString", zodValue: "z.string()"},
	"timestamp": {zodType: "z.ZodNumber", zodValue: "z.number()"},
	"timespan":  {zodType: "z.ZodNumber", zodValue: "z.number()"},
	"record":    {zodType: "z.ZodType<record.Unknown>", zodValue: "record.nullishToEmpty()"},
}

func defaultToTS(rawType string) string {
	if m, ok := typeParamMappings[rawType]; ok {
		return m.zodType
	}
	return "z.ZodType"
}

func defaultValueToTS(rawType string) string {
	if m, ok := typeParamMappings[rawType]; ok && m.zodValue != "" {
		return m.zodValue
	}
	return "z.unknown()"
}

func fallbackForConstraint(constraint *resolution.TypeRef, table *resolution.Table) string {
	if constraint == nil {
		return "z.unknown().optional()"
	}
	resolved, ok := constraint.Resolve(table)
	if ok {
		if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
			return camelCase(resolved.Name) + "Z"
		}
	}
	return defaultValueToTS(constraint.Name)
}

func fallbackSchemaTypeForConstraint(constraint *resolution.TypeRef, table *resolution.Table) string {
	if constraint == nil {
		return "z.ZodOptional<z.ZodUnknown>"
	}
	resolved, ok := constraint.Resolve(table)
	if ok {
		if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
			return "typeof " + camelCase(resolved.Name) + "Z"
		}
	}
	if m, ok := typeParamMappings[constraint.Name]; ok {
		return m.zodType
	}
	return "z.ZodType"
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

// isForwardReference checks if a type reference points to a type that will be
// declared later in the output. This is used to detect when we need getters
// for lazy evaluation in circular dependency cycles.
func isForwardReference(t resolution.TypeRef, data *templateData, table *resolution.Table) bool {
	if data.DeclOrder == nil {
		return false
	}

	var checkRef func(ref resolution.TypeRef) bool
	checkRef = func(ref resolution.TypeRef) bool {
		if ref.Name == "" || ref.IsTypeParam() {
			return false
		}
		if resolution.IsPrimitive(ref.Name) || ref.Name == "Array" || ref.Name == "Map" {
			return slices.ContainsFunc(ref.TypeArgs, checkRef)
		}
		resolved, ok := table.Get(ref.Name)
		if !ok {
			// Try namespace-qualified lookup
			resolved, ok = table.Lookup(data.Namespace, ref.Name)
		}
		if !ok {
			return false
		}
		if declIdx, exists := data.DeclOrder[resolved.QualifiedName]; exists {
			if declIdx > data.CurrentDeclIndex {
				return true
			}
		}
		return slices.ContainsFunc(ref.TypeArgs, checkRef)
	}

	return checkRef(t)
}

func (p *Plugin) processField(field resolution.Field, parentType resolution.Type, table *resolution.Table, data *templateData, needsTypeImports bool) fieldData {
	isArray := field.Type.Name == "Array"
	needsGetter := isSelfReference(field.Type, parentType) || isForwardReference(field.Type, data, table)

	fd := fieldData{
		Name:           field.Name,
		TSName:         fieldCamel(field.Name),
		Doc:            doc.Get(field.Domains),
		IsOptional:     field.IsOptional,
		IsHardOptional: field.IsHardOptional,
		IsArray:        isArray,
		IsSelfRef:      needsGetter,
	}
	if typeOverride := getFieldTypeOverride(field, "ts"); typeOverride != "" {
		// A `@ts type` override may name either a primitive (e.g. `string`) or
		// another schema type (e.g. `telem.TimeRangeBounded`). When it resolves to
		// a known non-primitive type, route it through the normal type-ref
		// machinery so its schema reference and import are emitted correctly;
		// otherwise treat it as a primitive. The override may be qualified
		// (cross-namespace, e.g. telem.TimeRangeBounded) or unqualified (same
		// namespace as the field): try the qualified name first, then resolve
		// against the field's own namespace.
		overrideType, overrideResolves := table.Get(typeOverride)
		if !overrideResolves {
			overrideType, overrideResolves = table.Lookup(parentType.Namespace, typeOverride)
		}
		if overrideResolves && !resolution.IsPrimitive(typeOverride) && typeOverride != "record" {
			overrideRef := resolution.TypeRef{Name: overrideType.QualifiedName}
			fd.ZodType = p.typeRefToZod(&overrideRef, table, data)
			fd.TSType = p.typeRefToTS(&overrideRef, table, data, needsTypeImports)
			fd.ZodSchemaType = p.typeRefToZodSchemaType(&overrideRef, table, data)
		} else {
			fd.ZodType = primitiveToZod(typeOverride, data)
			fd.TSType = primitiveToTS(typeOverride)
			fd.ZodSchemaType = primitiveToZodSchemaType(typeOverride)
		}
		if validateDomain, ok := field.Domains["validate"]; ok || field.Default != nil {
			result := p.applyValidation(fd.ZodType, validateDomain, field.Default, field.Type, field.Name, table, data, typeOverride)
			fd.ZodType = result.ZodType
			if result.HasDefault {
				fd.ZodSchemaType = defaultSchemaWrapper(result, fd.ZodSchemaType)
			}
		}
	} else {
		typeRefToProcess := &field.Type
		if isArray && len(field.Type.TypeArgs) > 0 {
			typeRefToProcess = &field.Type.TypeArgs[0]
		}
		fd.ZodType = p.typeRefToZod(typeRefToProcess, table, data)
		fd.TSType = p.typeRefToTS(typeRefToProcess, table, data, needsTypeImports)
		fd.ZodSchemaType = p.typeRefToZodSchemaType(typeRefToProcess, table, data)
		// An array field's default applies to the wrapped array (see the isArray
		// block below), not to the element schema processed here.
		elemDefault := field.Default
		if isArray {
			elemDefault = nil
		}
		if validateDomain, ok := field.Domains["validate"]; ok || elemDefault != nil {
			if sepIndex := strings.Index(fd.ZodType, " ?? "); sepIndex > 0 {
				paramPart := fd.ZodType[:sepIndex]
				fallbackPart := fd.ZodType[sepIndex+4:]
				result := p.applyValidation(fallbackPart, validateDomain, elemDefault, field.Type, field.Name, table, data, "")
				fd.ZodType = paramPart + " ?? " + result.ZodType
				if result.HasDefault {
					fd.ZodSchemaType = defaultSchemaWrapper(result, fd.ZodSchemaType)
				}
			} else {
				result := p.applyValidation(fd.ZodType, validateDomain, elemDefault, field.Type, field.Name, table, data, "")
				fd.ZodType = result.ZodType
				if result.HasDefault {
					fd.ZodSchemaType = defaultSchemaWrapper(result, fd.ZodSchemaType)
				}
			}
		}
	}
	// `@ts pick <field>` narrows a struct-typed field to a subset of the referenced
	// type's fields (e.g. a parent referenced by key alone), emitting a Zod .pick()
	// and a TS Pick<> rather than a standalone reference type.
	if pickField := domain.GetStringFromField(field, "ts", "pick"); pickField != "" {
		camel := fieldCamel(pickField)
		fd.ZodType = fmt.Sprintf("%s.pick({ %s: true })", fd.ZodType, camel)
		fd.TSType = fmt.Sprintf("Pick<%s, %q>", fd.TSType, camel)
	}
	isAnyOptional := field.IsOptional || field.IsHardOptional
	typeOverride := getFieldTypeOverride(field, "ts")
	isJSON := field.Type.Name == "record" || typeOverride == "record"
	isMap := field.Type.Name == "Map" && len(field.Type.TypeArgs) >= 2
	if isArray {
		if isAnyOptional {
			addXImport(data, xImport{name: "zod", submodule: "zod"})
			fd.ZodType = fmt.Sprintf("zod.nullToUndefined(%s.array())", fd.ZodType)
			fd.ZodSchemaType = fmt.Sprintf("ReturnType<typeof zod.nullToUndefined<z.ZodArray<%s>>>", fd.ZodSchemaType)
		} else {
			addXImport(data, xImport{name: "array", submodule: "array"})
			fd.ZodType = fmt.Sprintf("array.nullishToEmpty(%s)", fd.ZodType)
			fd.ZodSchemaType = fmt.Sprintf("ReturnType<typeof array.nullishToEmpty<%s>>", fd.ZodSchemaType)
			// nullishToEmpty already defaults a missing array to []; only a
			// non-empty declared default needs an explicit .default().
			if field.Default != nil && field.Default.Kind == resolution.ValueKindArray && len(field.Default.Elements) > 0 {
				fd.ZodType = fmt.Sprintf("%s.default(%s)", fd.ZodType, p.tsDefaultLiteral(field.Type, *field.Default, table, data))
				fd.ZodSchemaType = fmt.Sprintf("z.ZodDefault<%s>", fd.ZodSchemaType)
			}
		}
	} else if isMap {
		keyZ := p.typeRefToZod(&field.Type.TypeArgs[0], table, data)
		valueZ := p.typeRefToZod(&field.Type.TypeArgs[1], table, data)
		keySchemaType := p.typeRefToZodSchemaType(&field.Type.TypeArgs[0], table, data)
		valueSchemaType := p.typeRefToZodSchemaType(&field.Type.TypeArgs[1], table, data)
		if isAnyOptional {
			addXImport(data, xImport{name: "zod", submodule: "zod"})
			fd.ZodType = fmt.Sprintf("zod.nullToUndefined(z.record(%s, %s))", keyZ, valueZ)
			fd.ZodSchemaType = fmt.Sprintf("ReturnType<typeof zod.nullToUndefined<z.ZodRecord<%s, %s>>>", keySchemaType, valueSchemaType)
		} else {
			addXImport(data, xImport{name: "record", submodule: "record"})
			fd.ZodType = fmt.Sprintf("record.nullishToEmpty(%s, %s)", keyZ, valueZ)
			fd.ZodSchemaType = fmt.Sprintf("ReturnType<typeof record.nullishToEmpty<%s, %s>>", keySchemaType, valueSchemaType)
		}
	} else if isJSON {
		if isAnyOptional {
			addXImport(data, xImport{name: "zod", submodule: "zod"})
			fd.ZodType = fmt.Sprintf("zod.nullToUndefined(%s)", fd.ZodType)
			fd.ZodSchemaType = fmt.Sprintf("ReturnType<typeof zod.nullToUndefined<%s>>", fd.ZodSchemaType)
		} else {
			addXImport(data, xImport{name: "record", submodule: "record"})
			fd.ZodType = "record.nullishToEmpty()"
			fd.ZodSchemaType = "typeof record.nullishToEmpty()"
		}
	} else if isAnyOptional {
		if isUnionField(field, table) {
			// Optional union-typed fields tolerate null: the Go side
			// marshals a nil-variant union as null.
			addXImport(data, xImport{name: "zod", submodule: "zod"})
			fd.ZodType = fmt.Sprintf("zod.nullToUndefined(%s)", fd.ZodType)
			fd.ZodSchemaType = fmt.Sprintf("ReturnType<typeof zod.nullToUndefined<%s>>", fd.ZodSchemaType)
		} else if !field.Type.IsTypeParam() {
			fd.ZodType += ".optional()"
		} else if field.IsHardOptional {
			// Hard-optional (??) on a type-param field: the field is ALWAYS
			// optional, even when a caller passes a concrete schema. Wrap the
			// whole "param ?? fallback" expression in .optional() so caller
			// schemas get wrapped too. Strip a trailing .optional() from the
			// fallback first so we don't emit redundant nesting.
			fd.ZodType = fmt.Sprintf("(%s).optional()", strings.TrimSuffix(fd.ZodType, ".optional()"))
		}
		fd.ZodSchemaType = fmt.Sprintf("z.ZodOptional<%s>", fd.ZodSchemaType)
	}
	if hasPreserveCase(field) {
		addXImport(data, xImport{name: "caseconv", submodule: "caseconv"})
		fd.ZodType = fmt.Sprintf("caseconv.preserveCase(%s)", fd.ZodType)
	}
	if hasPreserveKeys(field) {
		addXImport(data, xImport{name: "caseconv", submodule: "caseconv"})
		fd.ZodType = fmt.Sprintf("caseconv.preserveKeys(%s)", fd.ZodType)
	}
	return fd
}

// isUnionField reports whether a field's type resolves to a discriminated
// union.
func isUnionField(field resolution.Field, table *resolution.Table) bool {
	resolved, ok := field.Type.Resolve(table)
	if !ok {
		return false
	}
	_, isUnion := resolved.Form.(resolution.UnionForm)
	return isUnion
}

func getFieldTypeOverride(field resolution.Field, domainName string) string {
	return domain.GetFieldType(field, domainName)
}

func getTypeTypeOverride(typ resolution.Type, domainName string) string {
	return domain.GetType(typ, domainName)
}

func (p *Plugin) typeRefToZod(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData) string {
	return p.typeRefToZodInternal(typeRef, table, data, false)
}

func (p *Plugin) typeRefToZodInternal(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData, forStructArg bool) string {
	if typeRef == nil {
		return "z.unknown()"
	}
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		paramName := camelCase(typeRef.TypeParam.Name)
		if forStructArg {
			return paramName
		}
		if typeRef.TypeParam.Constraint != nil && typeRef.TypeParam.Constraint.Name == "numeric" {
			return fmt.Sprintf("%s ?? %s", paramName, defaultValueToTS(typeRef.TypeParam.Default.Name))
		}
		if typeRef.TypeParam.Default != nil || typeRef.TypeParam.Optional {
			effectiveRef := typeRef.TypeParam.Constraint
			if effectiveRef == nil && typeRef.TypeParam.Default != nil && !typeRef.TypeParam.Optional {
				effectiveRef = typeRef.TypeParam.Default
			}
			if effectiveRef != nil && effectiveRef.Name == "record" {
				addXImport(data, xImport{name: "record", submodule: "record"})
			}
			return fmt.Sprintf("%s ?? %s", paramName, fallbackForConstraint(effectiveRef, table))
		}
		return paramName
	}
	if resolution.IsPrimitive(typeRef.Name) {
		return primitiveToZod(typeRef.Name, data)
	}
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		elemZod := p.typeRefToZodInternal(&typeRef.TypeArgs[0], table, data, false)
		if typeRef.ArraySize != nil {
			elements := make([]string, *typeRef.ArraySize)
			for i := range elements {
				elements[i] = elemZod
			}
			return fmt.Sprintf("z.tuple([%s])", strings.Join(elements, ", "))
		}
		return fmt.Sprintf("z.array(%s)", elemZod)
	}
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		keyZ := p.typeRefToZodInternal(&typeRef.TypeArgs[0], table, data, false)
		valueZ := p.typeRefToZodInternal(&typeRef.TypeArgs[1], table, data, false)
		return fmt.Sprintf("z.record(%s, %s)", keyZ, valueZ)
	}
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return "z.unknown()"
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		schemaName := camelCase(domain.GetName(resolved, "ts")) + "Z"
		if form.IsGeneric() {
			nonNilArgs := make([]struct {
				index int
				value string
			}, 0, len(typeRef.TypeArgs))
			for i, arg := range typeRef.TypeArgs {
				if arg.Name == "nil" {
					continue
				}
				nonNilArgs = append(nonNilArgs, struct {
					index int
					value string
				}{i, p.typeRefToZodInternal(&arg, table, data, true)})
			}
			if len(nonNilArgs) > 0 {
				if len(form.TypeParams) == 1 {
					schemaName = fmt.Sprintf("%s(%s)", schemaName, nonNilArgs[0].value)
				} else {
					namedArgs := make([]string, len(nonNilArgs))
					for i, arg := range nonNilArgs {
						namedArgs[i] = fmt.Sprintf("%s: %s", camelCase(form.TypeParams[arg.index].Name), arg.value)
					}
					schemaName = fmt.Sprintf("%s({%s})", schemaName, strings.Join(namedArgs, ", "))
				}
			} else {
				schemaName += "()"
			}
		}
		if resolved.Namespace != data.Namespace {
			ns := resolved.Namespace
			targetOutputPath := output.GetPath(resolved, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.AddImport(paths.CalculateImport(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("%s.%s", ns, schemaName)
		}
		return schemaName

	case resolution.EnumForm:
		enumName := camelCase(domain.GetName(resolved, "ts")) + "Z"
		if resolved.Namespace != data.Namespace {
			ns := resolved.Namespace
			targetOutputPath := enum.FindOutputPath(resolved, table, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.AddImport(paths.CalculateImport(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("%s.%s", ns, enumName)
		}
		return enumName

	case resolution.DistinctForm:
		schemaName := camelCase(domain.GetName(resolved, "ts")) + "Z"
		if resolved.Namespace != data.Namespace {
			ns := resolved.Namespace
			targetOutputPath := output.GetPath(resolved, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.AddImport(paths.CalculateImport(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("%s.%s", ns, schemaName)
		}
		return schemaName

	case resolution.AliasForm:
		if !form.IsGeneric() {
			schemaName := camelCase(domain.GetName(resolved, "ts")) + "Z"
			if resolved.Namespace != data.Namespace {
				ns := resolved.Namespace
				targetOutputPath := output.GetPath(resolved, "ts")
				if targetOutputPath == "" {
					targetOutputPath = ns
				}
				data.AddImport(paths.CalculateImport(data.OutputPath, targetOutputPath), ns)
				return fmt.Sprintf("%s.%s", ns, schemaName)
			}
			return schemaName
		}
		target := form.Target
		if len(typeRef.TypeArgs) > 0 {
			typeArgMap := make(map[string]resolution.TypeRef)
			for i, tp := range form.TypeParams {
				if i < len(typeRef.TypeArgs) {
					typeArgMap[tp.Name] = typeRef.TypeArgs[i]
				}
			}
			target = resolution.SubstituteTypeRef(target, typeArgMap)
		}
		return p.typeRefToZodInternal(&target, table, data, forStructArg)

	case resolution.UnionForm:
		schemaName := camelCase(domain.GetName(resolved, "ts")) + "Z"
		if resolved.Namespace != data.Namespace {
			ns := resolved.Namespace
			targetOutputPath := output.GetPath(resolved, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.AddImport(paths.CalculateImport(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("%s.%s", ns, schemaName)
		}
		return schemaName

	default:
		return "z.unknown()"
	}
}

func (p *Plugin) typeRefToTS(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData, needsTypeImports bool) string {
	return p.typeRefToTSInternal(typeRef, table, data, false, needsTypeImports)
}

func (p *Plugin) typeRefToTSInternal(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData, forStructArg bool, needsTypeImports bool) string {
	if typeRef == nil {
		return "unknown"
	}
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		if forStructArg {
			return typeRef.TypeParam.Name
		}
		if typeRef.TypeParam.Constraint != nil && typeRef.TypeParam.Constraint.Name == "numeric" {
			return typeRef.TypeParam.Name
		}
		return fmt.Sprintf("z.infer<%s>", typeRef.TypeParam.Name)
	}
	if resolution.IsPrimitive(typeRef.Name) {
		if needsTypeImports {
			switch typeRef.Name {
			case "timestamp":
				addXImport(data, xImport{name: "TimeStamp", submodule: "telem"})
			case "timespan":
				addXImport(data, xImport{name: "TimeSpan", submodule: "telem"})
			}
		}
		return primitiveToTS(typeRef.Name)
	}
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		elemTS := p.typeRefToTSInternal(&typeRef.TypeArgs[0], table, data, forStructArg, needsTypeImports)
		if typeRef.ArraySize != nil {
			elements := make([]string, *typeRef.ArraySize)
			for i := range elements {
				elements[i] = elemTS
			}
			return fmt.Sprintf("[%s]", strings.Join(elements, ", "))
		}
		return elemTS + "[]"
	}
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		keyType := p.typeRefToTSInternal(&typeRef.TypeArgs[0], table, data, forStructArg, needsTypeImports)
		valueType := p.typeRefToTSInternal(&typeRef.TypeArgs[1], table, data, forStructArg, needsTypeImports)
		return fmt.Sprintf("Record<%s, %s>", keyType, valueType)
	}
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return "unknown"
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		typeName := domain.GetName(resolved, "ts")
		if form.IsGeneric() && len(typeRef.TypeArgs) > 0 {
			args := make([]string, len(typeRef.TypeArgs))
			for i, arg := range typeRef.TypeArgs {
				args[i] = p.typeRefToTSInternal(&arg, table, data, true, needsTypeImports)
			}
			typeName = fmt.Sprintf("%s<%s>", typeName, strings.Join(args, ", "))
		}
		if resolved.Namespace != data.Namespace {
			return fmt.Sprintf("%s.%s", resolved.Namespace, typeName)
		}
		return typeName

	case resolution.EnumForm:
		enumName := domain.GetName(resolved, "ts")
		if resolved.Namespace != data.Namespace {
			return fmt.Sprintf("%s.%s", resolved.Namespace, enumName)
		}
		return enumName

	case resolution.DistinctForm:
		distinctName := domain.GetName(resolved, "ts")
		if resolved.Namespace != data.Namespace {
			ns := resolved.Namespace
			targetOutputPath := output.GetPath(resolved, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.AddImport(paths.CalculateImport(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("%s.%s", ns, distinctName)
		}
		return distinctName

	case resolution.AliasForm:
		typeName := domain.GetName(resolved, "ts")
		if form.IsGeneric() && len(typeRef.TypeArgs) > 0 {
			args := make([]string, len(typeRef.TypeArgs))
			for i, arg := range typeRef.TypeArgs {
				args[i] = p.typeRefToTSInternal(&arg, table, data, true, needsTypeImports)
			}
			typeName = fmt.Sprintf("%s<%s>", typeName, strings.Join(args, ", "))
		}
		if resolved.Namespace != data.Namespace {
			return fmt.Sprintf("%s.%s", resolved.Namespace, typeName)
		}
		return typeName

	case resolution.UnionForm:
		typeName := domain.GetName(resolved, "ts")
		if resolved.Namespace != data.Namespace {
			return fmt.Sprintf("%s.%s", resolved.Namespace, typeName)
		}
		return typeName

	default:
		return "unknown"
	}
}

var primitiveTSTypes = map[string]string{
	"string": "string", "uuid": "string",
	"bool": "boolean",
	"int8": "number", "int16": "number", "int32": "number", "int64": "number",
	"uint8": "number", "uint12": "number", "uint16": "number", "uint20": "number", "uint32": "number", "uint64": "number",
	"float32": "number", "float64": "number", "number": "number",
	"timestamp": "TimeStamp", "timespan": "TimeSpan", "data_type": "DataType",
	"record": "unknown", "bytes": "Uint8Array",
}

func primitiveToTS(primitive string) string {
	if t, ok := primitiveTSTypes[primitive]; ok {
		return t
	}
	return "unknown"
}

type xImport struct {
	name      string
	submodule string
}

type primitiveMapping struct {
	schema   string
	xImports []xImport
}

var primitiveZodTypes = map[string]primitiveMapping{
	"uuid":               {schema: "z.uuid()"},
	"string":             {schema: "z.string()"},
	"bool":               {schema: "z.boolean()"},
	"int8":               {schema: "zod.int8", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"int16":              {schema: "zod.int16", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"int32":              {schema: "z.int32()"},
	"int64":              {schema: "z.int64()"},
	"uint8":              {schema: "zod.uint8", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"uint12":             {schema: "zod.uint12", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"uint16":             {schema: "zod.uint16", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"uint20":             {schema: "zod.uint20", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"uint32":             {schema: "z.uint32()"},
	"uint64":             {schema: "z.uint64()"},
	"float32":            {schema: "z.number()"},
	"float64":            {schema: "z.number()"},
	"number":             {schema: "z.number()"},
	"timestamp":          {schema: "TimeStamp.z", xImports: []xImport{{name: "TimeStamp", submodule: "telem"}}},
	"timespan":           {schema: "TimeSpan.z", xImports: []xImport{{name: "TimeSpan", submodule: "telem"}}},
	"time_range":         {schema: "TimeRange.z", xImports: []xImport{{name: "TimeRange", submodule: "telem"}}},
	"time_range_bounded": {schema: "TimeRange.boundedZ", xImports: []xImport{{name: "TimeRange", submodule: "telem"}}},
	"data_type":          {schema: "DataType.z", xImports: []xImport{{name: "DataType", submodule: "telem"}}},
	"record":             {schema: "record.unknownZ().or(z.string().transform((s) => JSON.parse(s)))", xImports: []xImport{{name: "record", submodule: "record"}}},
	"bytes":              {schema: "z.instanceof(Uint8Array)"},
}

const xPackageName = "@synnaxlabs/x"
const xPathPrefix = "x/ts/src"

var primitiveZodSchemaTypes = map[string]string{
	"uuid":               "z.ZodString",
	"string":             "z.ZodString",
	"bool":               "z.ZodBoolean",
	"int8":               "z.ZodNumber",
	"int16":              "z.ZodNumber",
	"int32":              "z.ZodNumber",
	"int64":              "z.ZodBigInt",
	"uint8":              "z.ZodNumber",
	"uint12":             "z.ZodNumber",
	"uint16":             "z.ZodNumber",
	"uint20":             "z.ZodNumber",
	"uint32":             "z.ZodNumber",
	"uint64":             "z.ZodBigInt",
	"float32":            "z.ZodNumber",
	"float64":            "z.ZodNumber",
	"number":             "z.ZodNumber",
	"timestamp":          "typeof TimeStamp.z",
	"timespan":           "typeof TimeSpan.z",
	"time_range":         "typeof TimeRange.z",
	"time_range_bounded": "typeof TimeRange.boundedZ",
	"data_type":          "typeof DataType.z",
	"record":             "z.ZodType",
	"bytes":              "z.ZodType<Uint8Array>",
}

func primitiveToZodSchemaType(primitive string) string {
	if t, ok := primitiveZodSchemaTypes[primitive]; ok {
		return t
	}
	return "z.ZodType"
}

func (p *Plugin) typeRefToZodSchemaType(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData) string {
	if typeRef == nil {
		return "z.ZodType"
	}
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		return typeRef.TypeParam.Name
	}
	if resolution.IsPrimitive(typeRef.Name) {
		return primitiveToZodSchemaType(typeRef.Name)
	}
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		elemZodType := p.typeRefToZodSchemaType(&typeRef.TypeArgs[0], table, data)
		if typeRef.ArraySize != nil {
			elements := make([]string, *typeRef.ArraySize)
			for i := range elements {
				elements[i] = elemZodType
			}
			return fmt.Sprintf("z.ZodTuple<[%s]>", strings.Join(elements, ", "))
		}
		return fmt.Sprintf("z.ZodArray<%s>", elemZodType)
	}
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		keyType := p.typeRefToZodSchemaType(&typeRef.TypeArgs[0], table, data)
		valueType := p.typeRefToZodSchemaType(&typeRef.TypeArgs[1], table, data)
		return fmt.Sprintf("z.ZodRecord<%s, %s>", keyType, valueType)
	}
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		resolved, ok = table.Lookup(data.Namespace, typeRef.Name)
	}
	if !ok {
		return "z.ZodType"
	}

	prefix := ""
	if resolved.Namespace != data.Namespace {
		prefix = resolved.Namespace + "."
	}

	tsName := domain.GetName(resolved, "ts")
	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		if form.IsGeneric() && len(typeRef.TypeArgs) > 0 {
			hasConcreteTypes := false
			if dom, ok := resolved.Domains["ts"]; ok {
				for _, expr := range dom.Expressions {
					if expr.Name == "concrete_types" {
						hasConcreteTypes = true
						break
					}
				}
			}

			if hasConcreteTypes {
				args := make([]string, len(typeRef.TypeArgs))
				for i, arg := range typeRef.TypeArgs {
					args[i] = p.typeRefToZodSchemaType(&arg, table, data)
				}
				return fmt.Sprintf("%s%sZodObject<%s>", prefix, tsName, strings.Join(args, ", "))
			}
			args := make([]string, len(typeRef.TypeArgs))
			for i, arg := range typeRef.TypeArgs {
				args[i] = p.typeRefToZodSchemaType(&arg, table, data)
			}
			return fmt.Sprintf("ReturnType<typeof %s%sZ<%s>>", prefix, camelCase(tsName), strings.Join(args, ", "))
		}
		return fmt.Sprintf("typeof %s%sZ", prefix, camelCase(tsName))

	case resolution.EnumForm:
		return fmt.Sprintf("typeof %s%sZ", prefix, camelCase(tsName))

	case resolution.DistinctForm:
		return fmt.Sprintf("typeof %s%sZ", prefix, camelCase(tsName))

	case resolution.UnionForm:
		// A union schema's inferred type depends on every variant schema, so a
		// `typeof xZ` annotation in a getter would re-enter the inference cycle the
		// getter exists to break. z.ZodType<Name> references only the declared
		// union type alias, which TypeScript resolves lazily.
		return fmt.Sprintf("z.ZodType<%s%s>", prefix, tsName)
	}

	return "z.ZodType"
}

func isInXPackage(outputPath string) bool {
	return strings.HasPrefix(outputPath, xPathPrefix)
}

func addXImport(data *templateData, imp xImport) {
	if isInXPackage(data.OutputPath) {
		data.AddImport("@/"+imp.submodule, imp.name)
	} else {
		data.AddImport(xPackageName, imp.name)
	}
}

func primitiveToZod(primitive string, data *templateData) string {
	if primitive == "record" {
		addXImport(data, xImport{name: "record", submodule: "record"})
		return "record.unknownZ()"
	}
	if mapping, ok := primitiveZodTypes[primitive]; ok {
		for _, imp := range mapping.xImports {
			addXImport(data, imp)
		}
		return mapping.schema
	}
	return "z.unknown()"
}

type validationResult struct {
	ZodType    string
	HasDefault bool
	// IsPrefault is true when the default was emitted as .prefault() rather than
	// .default(). Struct defaults use .prefault() so the literal is re-parsed,
	// filling each field's own default when a caller supplies a partial value.
	IsPrefault bool
}

// defaultSchemaWrapper wraps a zod schema type for a defaulted field, picking
// ZodPrefault for struct defaults (.prefault) and ZodDefault otherwise.
func defaultSchemaWrapper(r validationResult, inner string) string {
	if r.IsPrefault {
		return fmt.Sprintf("z.ZodPrefault<%s>", inner)
	}
	return fmt.Sprintf("z.ZodDefault<%s>", inner)
}

func (p *Plugin) applyValidation(zodType string, domain resolution.Domain, defaultVal *resolution.ExpressionValue, typeRef resolution.TypeRef, fieldName string, table *resolution.Table, data *templateData, tsTypeOverride string) validationResult {
	rules := validation.Parse(domain)
	if validation.IsEmpty(rules) && defaultVal == nil {
		return validationResult{ZodType: zodType, HasDefault: false}
	}
	hasDefault := defaultVal != nil
	isPrefault := false
	effectiveType := typeRef.Name
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil && typeRef.TypeParam.Constraint != nil {
		effectiveType = typeRef.TypeParam.Constraint.Name
	}
	// A @ts type override (e.g. a telem.Rate field surfaced as a plain "number"
	// in TypeScript) drives both the validation rules and the default literal off
	// the overridden primitive, not the underlying schema type.
	if tsTypeOverride != "" {
		effectiveType = tsTypeOverride
	}
	isString := resolution.IsPrimitive(effectiveType) && resolution.IsStringPrimitive(effectiveType)
	isNumber := resolution.IsPrimitive(effectiveType) && resolution.IsNumberPrimitive(effectiveType)
	if isString {
		if rules.Required {
			humanName := lo.Capitalize(strings.ReplaceAll(fieldName, "_", " "))
			zodType = fmt.Sprintf("%s.min(1, \"%s is required\")", zodType, humanName)
		}
		if rules.MinLength != nil {
			zodType = fmt.Sprintf("%s.min(%d)", zodType, *rules.MinLength)
		}
		if rules.MaxLength != nil {
			zodType = fmt.Sprintf("%s.max(%d)", zodType, *rules.MaxLength)
		}
		if rules.Pattern != nil {
			if rules.PatternMessage != nil {
				zodType = fmt.Sprintf("%s.regex(/%s/, %s)", zodType, *rules.Pattern, tsStringLiteral(*rules.PatternMessage))
			} else {
				zodType = fmt.Sprintf("%s.regex(/%s/)", zodType, *rules.Pattern)
			}
		}
	}
	if isNumber {
		if rules.Min != nil {
			if rules.Min.IsInt {
				zodType = fmt.Sprintf("%s.min(%d)", zodType, rules.Min.Int)
			} else {
				zodType = fmt.Sprintf("%s.min(%f)", zodType, rules.Min.Float)
			}
		}
		if rules.Max != nil {
			if rules.Max.IsInt {
				zodType = fmt.Sprintf("%s.max(%d)", zodType, rules.Max.Int)
			} else {
				zodType = fmt.Sprintf("%s.max(%f)", zodType, rules.Max.Float)
			}
		}
	}
	if defaultVal != nil {
		switch defaultVal.Kind {
		case resolution.ValueKindString:
			zodType = fmt.Sprintf("%s.default(%s)", zodType, tsStringLiteral(defaultVal.StringValue))
		case resolution.ValueKindInt:
			expr, ok := "", false
			if tsTypeOverride == "" {
				expr, ok = tsTelemNumericDefault(
					typeRef, data, fmt.Sprintf("%d", defaultVal.IntValue), defaultVal.IntValue == 0,
				)
			}
			if ok {
				zodType = fmt.Sprintf("%s.default(%s)", zodType, expr)
			} else {
				zodType = fmt.Sprintf("%s.default(%d)", zodType, defaultVal.IntValue)
			}
		case resolution.ValueKindFloat:
			expr, ok := "", false
			if tsTypeOverride == "" {
				expr, ok = tsTelemNumericDefault(
					typeRef, data, fmt.Sprintf("%g", defaultVal.FloatValue), defaultVal.FloatValue == 0,
				)
			}
			if ok {
				zodType = fmt.Sprintf("%s.default(%s)", zodType, expr)
			} else {
				zodType = fmt.Sprintf("%s.default(%f)", zodType, defaultVal.FloatValue)
			}
		case resolution.ValueKindBool:
			zodType = fmt.Sprintf("%s.default(%t)", zodType, defaultVal.BoolValue)
		case resolution.ValueKindIdent:
			// Handle identifier-based defaults like "now" for timestamps
			if defaultVal.IdentValue == "now" && (typeRef.Name == "TimeStamp" || strings.HasSuffix(typeRef.Name, ".TimeStamp")) {
				addXImport(data, xImport{name: "TimeStamp", submodule: "telem"})
				zodType = fmt.Sprintf("%s.default(() => TimeStamp.now())", zodType)
			}
			// Handle "create" for auto-generating keys. uuid keys generate a UUID
			// via uuid.create(); string keys generate a short id via id.create().
			// uuid is a string primitive, so check it first.
			// Use key.ResolvePrimitive to handle type aliases like `Key distinct string`.
			primitive := key.ResolvePrimitive(typeRef, table)
			if defaultVal.IdentValue == "create" {
				if primitive == "uuid" {
					addXImport(data, xImport{name: "uuid", submodule: "uuid"})
					zodType = fmt.Sprintf("%s.default(() => uuid.create())", zodType)
				} else if isString || primitive == "string" {
					addXImport(data, xImport{name: "id", submodule: "id"})
					zodType = fmt.Sprintf("%s.default(() => id.create())", zodType)
				}
			}
			if ev, ok := validation.ResolveEnumVariant(defaultVal.IdentValue, typeRef, table); ok {
				zodType = fmt.Sprintf("%s.default(%s)", zodType, p.enumVariantToTS(ev, data))
			}
		case resolution.ValueKindArray:
			zodType = fmt.Sprintf("%s.default(%s)", zodType, p.tsDefaultLiteral(typeRef, *defaultVal, table, data))
		case resolution.ValueKindStruct:
			// .prefault re-parses the literal, so a partial value a caller supplies
			// is merged with each field's own default rather than rejected.
			zodType = fmt.Sprintf("%s.prefault(%s)", zodType, p.tsDefaultLiteral(typeRef, *defaultVal, table, data))
			isPrefault = true
		}
	}
	return validationResult{ZodType: zodType, HasDefault: hasDefault, IsPrefault: isPrefault}
}

// tsTelemNumericDefault returns the .default(...) argument for a numeric default
// on a telem class field, registering the import, or ("", false) when typeRef is
// not such a type. The telem zod schemas output class instances, so a bare number
// would not type-check as the default; the literal is wrapped in the constructor
// ("new Rate(10)"). A zero TimeStamp or TimeSpan keeps the canonical .ZERO
// constant.
func tsTelemNumericDefault(
	typeRef resolution.TypeRef, data *templateData, literal string, isZero bool,
) (string, bool) {
	name := typeRef.Name
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	switch name {
	case "TimeStamp", "TimeSpan", "Rate":
		addXImport(data, xImport{name: name, submodule: "telem"})
		if isZero && name != "Rate" {
			return name + ".ZERO", true
		}
		return fmt.Sprintf("new %s(%s)", name, literal), true
	}
	return "", false
}

// tsStringLiteral renders a Go string as a double-quoted TypeScript string
// literal. Unlike Go's %q verb, supplementary-plane code points (> U+FFFF) are
// emitted as a surrogate pair (\uD8xx\uDCxx) rather than \U0001XXXX, which is
// not valid JavaScript/TypeScript syntax.
func tsStringLiteral(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			switch {
			case r < 0x20:
				fmt.Fprintf(&b, `\u%04X`, r)
			case r > 0xFFFF:
				v := r - 0x10000
				fmt.Fprintf(&b, `\u%04X\u%04X`, 0xD800+(v>>10), 0xDC00+(v&0x3FF))
			default:
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}

// tsDefaultLiteral renders a default value as a TypeScript literal. typeRef is
// the declared type of the value, used to resolve enum variants, array element
// types, and nested struct field types. Arrays and structs recurse.
func (p *Plugin) tsDefaultLiteral(typeRef resolution.TypeRef, val resolution.ExpressionValue, table *resolution.Table, data *templateData) string {
	switch val.Kind {
	case resolution.ValueKindString:
		return tsStringLiteral(val.StringValue)
	case resolution.ValueKindInt:
		return fmt.Sprintf("%d", val.IntValue)
	case resolution.ValueKindFloat:
		return fmt.Sprintf("%f", val.FloatValue)
	case resolution.ValueKindBool:
		return fmt.Sprintf("%t", val.BoolValue)
	case resolution.ValueKindIdent:
		if ev, ok := validation.ResolveEnumVariant(val.IdentValue, typeRef, table); ok {
			return p.enumVariantToTS(ev, data)
		}
		return val.IdentValue
	case resolution.ValueKindArray:
		elem := arrayElementType(typeRef)
		parts := make([]string, 0, len(val.Elements))
		for _, el := range val.Elements {
			parts = append(parts, p.tsDefaultLiteral(elem, el, table, data))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case resolution.ValueKindStruct:
		return p.tsStructLiteral(typeRef, val, table, data)
	}
	return ""
}

// tsStructLiteral renders a struct default as a TypeScript object literal,
// resolving each field's value against its declared type in the struct named by
// typeRef.
func (p *Plugin) tsStructLiteral(typeRef resolution.TypeRef, val resolution.ExpressionValue, table *resolution.Table, data *templateData) string {
	fieldsByName := structFieldsByName(typeRef, table)
	parts := make([]string, 0, len(val.Fields))
	for _, fv := range val.Fields {
		parts = append(parts, fmt.Sprintf("%s: %s", fieldCamel(fv.Name), p.tsDefaultLiteral(fieldsByName[fv.Name].Type, fv.Value, table, data)))
	}
	return "{ " + strings.Join(parts, ", ") + " }"
}

// arrayElementType returns the element type of an array type reference, or the
// reference itself when it is not an array.
func arrayElementType(typeRef resolution.TypeRef) resolution.TypeRef {
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		return typeRef.TypeArgs[0]
	}
	return typeRef
}

// structFieldsByName resolves typeRef to a struct type and maps its unified
// fields by unqualified name. It returns an empty map when typeRef does not
// resolve to a struct.
func structFieldsByName(typeRef resolution.TypeRef, table *resolution.Table) map[string]resolution.Field {
	out := map[string]resolution.Field{}
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return out
	}
	for _, f := range resolution.UnifiedFields(resolved, table) {
		out[f.Name] = f
	}
	return out
}

func (p *Plugin) enumVariantToTS(ev validation.EnumVariant, data *templateData) string {
	// String-valued enums are emitted as `z.enum([...])` plus a type alias and
	// have no runtime object to dot into. Emit the raw string literal instead;
	// only numeric enums get the `Type.variant` form, since those emit as TS
	// runtime enums.
	if form, ok := ev.Type.Form.(resolution.EnumForm); ok && !form.IsIntEnum {
		return tsStringLiteral(ev.Variant.StringValue())
	}
	enumName := domain.GetName(ev.Type, "ts")
	variantRef := fmt.Sprintf("%s.%s", enumName, ev.Variant.Name)
	if ev.Type.Namespace != data.Namespace {
		variantRef = fmt.Sprintf("%s.%s", ev.Type.Namespace, variantRef)
	}
	return variantRef
}

type templateData struct {
	*imports.Manager
	Request          *plugin.Request
	Ontology         *ontologyData
	DeclOrder        map[string]int
	Namespace        string
	OutputPath       string
	Structs          []structData
	Enums            []enumData
	TypeDefs         []typeDefData
	SortedDecls      []sortedDeclData
	CurrentDeclIndex int
	GenerateTypes    bool
}

type sortedDeclData struct {
	TypeDef   typeDefData
	Struct    structData
	Union     unionData
	IsTypeDef bool
	IsStruct  bool
	IsUnion   bool
}

type typeDefData struct {
	Name    string
	TSName  string
	TSType  string
	ZodType string
}

type ontologyData struct {
	TypeName, KeyType, KeyZeroValue string
}

func primitiveZeroValue(primitive string) string {
	switch primitive {
	case "uuid", "string":
		return `""`
	case "bool":
		return "false"
	case "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return "0"
	default:
		return `""`
	}
}

type structData struct {
	ExtendsName             string
	TSName                  string
	AliasOf                 string
	Doc                     string
	ExtendsTypeName         string
	Name                    string
	TypeParams              []typeParamData
	ExtendsParentSchemaArgs []string
	BaseFields              []fieldData
	ConditionalFields       []conditionalFieldData
	ExtendFields            []fieldData
	PartialFields           []fieldData
	OmittedFields           []string
	Fields                  []fieldData
	ExtendsParents          []extendsParentInfo
	HasExtends              bool
	UseInput                bool
	AllParamsOptional       bool
	ExtendsParentIsGeneric  bool
	Handwritten             bool
	IsRecursive             bool
	IsAlias                 bool
	IsSingleParam           bool
	IsGeneric               bool
	ConcreteTypes           bool
	CoalesceTypeParams      bool
	// IsPrimitiveConstrainedGeneric is true when every type param is constrained
	// to a primitive set (e.g. T extends numeric) and has a default. In this mode
	// the runtime zod schema is emitted as a plain z.object (defaults substituted
	// for type-param-typed fields), but the TS interface keeps the generic shape
	// with primitive constraints (e.g. T extends number | bigint = number).
	IsPrimitiveConstrainedGeneric bool
}

type extendsParentInfo struct {
	Name       string
	TypeName   string
	SchemaArgs []string
	IsGeneric  bool
}

type typeParamData struct {
	Name, Constraint, Default, DefaultValue string
	// BareConstraint and BareDefault are populated for primitive-constrained
	// generics (e.g. T extends numeric). Constraint/Default carry the zod-side
	// values used in runtime-builder positions; BareConstraint/BareDefault carry
	// the TS-side values used in interface signatures (e.g. "number | bigint",
	// "number"). Empty for non-primitive-constrained params.
	BareConstraint, BareDefault string
	HasDefault, IsJSON          bool
	IsPrimitiveConstrained      bool
}

type fieldData struct {
	Name, TSName, ZodType, TSType, ZodSchemaType   string
	CoalescedTSType                                string
	Doc                                            string
	IsOptional, IsHardOptional, IsArray, IsSelfRef bool
}

type conditionalFieldData struct {
	TypeParamName      string
	NeverType          string
	FallbackSchemaType string
	Field              fieldData
}

type enumData struct {
	Name             string
	Values           []enumValueData
	IsIntEnum        bool
	GenerateLiterals bool
}

type enumValueData struct {
	Name, Value string
	IntValue    int64
	IsIntEnum   bool
}

var templateFuncs = template.FuncMap{
	"camelCase": camelCase,
	"title":     lo.Capitalize,
	"lower":     strings.ToLower,
	"pluralUpper": func(name string) string {
		s := strings.ToUpper(lo.SnakeCase(name))
		if strings.HasSuffix(s, "S") {
			return s
		}
		return s + "S"
	},
	"formatDoc": doc.FormatTS,
}

var fileTemplate = template.Must(template.New("zod").Funcs(templateFuncs).Parse(`// Code generated by Oracle. DO NOT EDIT.
{{range .SynnaxImports }}
import { {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} } from "{{ .Path }}";
{{- end }}
import { z } from "zod";
{{- range .ExternalNamedImports }}
import { {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} } from "{{ .Path }}";
{{- end }}
{{ if .InternalNamedImports }}
{{- range .InternalNamedImports }}
import { {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} } from "{{ .Path }}";
{{- end }}
{{- end }}
{{- range .TypeDefs }}

export const {{ .TSName | camelCase }}Z = {{ .ZodType }};
export type {{ .TSName }} = z.infer<typeof {{ .TSName | camelCase }}Z>;
{{- end }}
{{- range .Enums }}

{{ if .IsIntEnum }}
export enum {{ .Name }} {
{{- range $i, $v := .Values }}
  {{ $v.Name }} = {{ $v.IntValue }},
{{- end }}
}
export const {{ camelCase .Name }}Z = z.enum({{ .Name }});
{{- else }}
export const {{ pluralUpper .Name }} = [{{ range $i, $v := .Values }}{{ if $i }}, {{ end }}"{{ $v.Value }}"{{ end }}] as const;
export const {{ camelCase .Name }}Z = z.enum({{ pluralUpper .Name }});
{{- if .GenerateLiterals }}
{{- $enumName := .Name }}
{{- range $i, $v := .Values }}
export const {{ camelCase $v.Name }}{{ $enumName }}Z = z.literal("{{ $v.Value }}");
{{- end }}
{{- end }}
{{- end }}
{{- if and $.GenerateTypes (not .IsIntEnum) }}
export type {{ .Name }} = z.infer<typeof {{ camelCase .Name }}Z>;
{{- end }}
{{- end }}
{{- range .SortedDecls }}
{{- if .IsTypeDef }}

export const {{ .TypeDef.TSName | camelCase }}Z = {{ .TypeDef.ZodType }};
export type {{ .TypeDef.TSName }} = z.infer<typeof {{ .TypeDef.TSName | camelCase }}Z>;
{{- else if .IsStruct }}
{{- with .Struct }}
{{- if not .Handwritten }}
{{- if .IsAlias }}
{{- if and .IsGeneric (gt (len .TypeParams) 0) }}
{{- if .IsSingleParam }}

export const {{ camelCase .TSName }}Z = <{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>({{ range $i, $p := .TypeParams }}{{ $p.Name | camelCase }}{{ if $p.HasDefault }}?{{ end }}: {{ $p.Name }}{{ end }}) =>
  {{ .AliasOf }};
{{- if $.GenerateTypes }}
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = z.{{ if .UseInput }}input{{ else }}infer{{ end }}<
  ReturnType<typeof {{ camelCase .TSName }}Z<{{ range $i, $p := .TypeParams }}{{ $p.Name }}{{ end }}>>
>;
{{- end }}
{{- else }}

export interface {{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Constraint }}{{ end }}{{ end }}> {
{{- range $i, $p := .TypeParams }}
  {{ $p.Name | camelCase }}: {{ $p.Name }};
{{- end }}
}

export const {{ camelCase .TSName }}Z = <{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>({
{{- range $i, $p := .TypeParams }}
  {{ $p.Name | camelCase }},
{{- end }}
}: Partial<{{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>>{{ if .AllParamsOptional }} = {}{{ end }}) =>
  {{ .AliasOf }};
{{- if $.GenerateTypes }}
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = z.{{ if .UseInput }}input{{ else }}infer{{ end }}<
  ReturnType<typeof {{ camelCase .TSName }}Z<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>>
>;
{{- end }}
{{- end }}
{{- else }}

export const {{ camelCase .TSName }}Z = {{ .AliasOf }};
{{- if $.GenerateTypes }}
export interface {{ .TSName }} extends z.{{ if .UseInput }}input{{ else }}infer{{ end }}<typeof {{ camelCase .TSName }}Z> {}
{{- end }}
{{- end }}
{{- else if .IsPrimitiveConstrainedGeneric }}
{{- if .Doc }}

{{ formatDoc .TSName .Doc }}
{{- end }}
export const {{ camelCase .TSName }}Z = <{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.BareConstraint }}{{ if $p.HasDefault }} = {{ $p.BareDefault }}{{ end }}{{ end }}>({{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name | camelCase }}?: z.ZodType<{{ $p.Name }}>{{ end }}) =>
  z.object({
{{- range .Fields }}
{{- if .Doc }}
  {{ formatDoc .TSName .Doc }}
{{- end }}
    {{ .TSName }}: {{ .ZodType }},
{{- end }}
  });
{{- if $.GenerateTypes }}
export interface {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.BareConstraint }}{{ if $p.HasDefault }} = {{ $p.BareDefault }}{{ end }}{{ end }}> {
{{- range .Fields }}
  {{ .TSName }}{{ if or .IsOptional .IsHardOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}
{{- end }}
{{- else if .IsGeneric }}
{{- if .IsSingleParam }}
{{- if and .ConcreteTypes .ConditionalFields }}

export type {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = z.ZodObject<{
{{- range .BaseFields }}
    {{ .TSName }}: {{ .ZodSchemaType }};
{{- end }}
{{- range .ConditionalFields }}
    {{ .Field.TSName }}: [{{ .TypeParamName }}] extends [{{ .NeverType }}] ? {{ .FallbackSchemaType }} : {{ if .Field.IsHardOptional }}z.ZodOptional<{{ .TypeParamName }}>{{ else }}{{ .TypeParamName }}{{ end }};
{{- end }}
}>;

export interface {{ .TSName }}ZFunction {
  <{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ end }}>(
    {{ range $i, $p := .TypeParams }}{{ $p.Name | camelCase }}: {{ $p.Name }}{{ end }}
  ): {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ $p.Name }}{{ end }}>;
  <{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>(
    {{ range $i, $p := .TypeParams }}{{ $p.Name | camelCase }}?: {{ $p.Name }}{{ end }}
  ): {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ $p.Name }}{{ end }}>;
}

export const {{ camelCase .TSName }}Z: {{ .TSName }}ZFunction = <{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ end }}>({{ range $i, $p := .TypeParams }}{{ $p.Name | camelCase }}?: {{ $p.Name }}{{ end }}) =>
{{- else }}

export const {{ camelCase .TSName }}Z = <{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>({{ range $i, $p := .TypeParams }}{{ $p.Name | camelCase }}{{ if $p.HasDefault }}?{{ end }}: {{ $p.Name }}{{ end }}) =>
{{- end }}
{{- if .HasExtends }}
  {{ range $i, $p := .ExtendsParents }}{{ if $i }}.extend({{ end }}{{ $p.Name }}({{ if $p.IsGeneric }}{{ range $j, $a := $p.SchemaArgs }}{{ if $j }}, {{ end }}{{ $a }}{{ end }}{{ end }}){{ if $i }}.shape){{ end }}{{ end }}
{{- if .OmittedFields }}
    .omit({ {{ range $i, $f := .OmittedFields }}{{ if $i }}, {{ end }}{{ $f }}: true{{ end }} })
{{- end }}
{{- if .PartialFields }}
    .partial({ {{ range $i, $f := .PartialFields }}{{ if $i }}, {{ end }}{{ $f.TSName }}: true{{ end }} })
{{- end }}
{{- if .ExtendFields }}
    .extend({
{{- range .ExtendFields }}
{{- if .IsSelfRef }}
      get {{ .TSName }}(): {{ .ZodSchemaType }} {
        return {{ .ZodType }};
      },
{{- else }}
      {{ .TSName }}: {{ .ZodType }},
{{- end }}
{{- end }}
    })
{{- end }};
{{- else }}
  z.object({
{{- range .Fields }}
{{- if .IsSelfRef }}
    get {{ .TSName }}(): {{ .ZodSchemaType }} {
      return {{ .ZodType }};
    },
{{- else }}
    {{ .TSName }}: {{ .ZodType }},
{{- end }}
{{- end }}
  });
{{- end }}
{{- else }}

export interface {{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Constraint }}{{ end }}{{ end }}> {
{{- range $i, $p := .TypeParams }}
  {{ $p.Name | camelCase }}: {{ $p.Name }};
{{- end }}
}
{{- if and .ConcreteTypes .ConditionalFields }}

export type {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = z.ZodObject<{
{{- range .BaseFields }}
    {{ .TSName }}: {{ .ZodSchemaType }};
{{- end }}
{{- range .ConditionalFields }}
    {{ .Field.TSName }}: [{{ .TypeParamName }}] extends [{{ .NeverType }}] ? {{ .FallbackSchemaType }} : {{ if .Field.IsHardOptional }}z.ZodOptional<{{ .TypeParamName }}>{{ else }}{{ .TypeParamName }}{{ end }};
{{- end }}
}>;

export interface {{ .TSName }}ZFunction {
  <{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if not $p.HasDefault }}{{ else if $p.Default }} = {{ $p.Default }}{{ end }}{{ end }}>(
    args: { {{ range $i, $p := .TypeParams }}{{ if $i }}; {{ end }}{{ $p.Name | camelCase }}{{ if $p.HasDefault }}?{{ end }}: {{ $p.Name }}{{ end }} }
  ): {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>;
  <{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>(
    args?: Partial<{{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>>
  ): {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>;
}

export const {{ camelCase .TSName }}Z: {{ .TSName }}ZFunction = <{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ end }}>({
{{- range $i, $p := .TypeParams }}
  {{ $p.Name | camelCase }},
{{- end }}
}: Partial<{{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>> = {}) =>
{{- else }}

export const {{ camelCase .TSName }}Z = <{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>({
{{- range $i, $p := .TypeParams }}
  {{ $p.Name | camelCase }},
{{- end }}
}: Partial<{{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>>{{ if .AllParamsOptional }} = {}{{ end }}) =>
{{- end }}
{{- if .HasExtends }}
  {{ range $i, $p := .ExtendsParents }}{{ if $i }}.extend({{ end }}{{ $p.Name }}({{ if $p.IsGeneric }}{ {{ range $j, $a := $p.SchemaArgs }}{{ if $j }}, {{ end }}{{ $a }}{{ end }} }{{ end }}){{ if $i }}.shape){{ end }}{{ end }}
{{- if .OmittedFields }}
    .omit({ {{ range $i, $f := .OmittedFields }}{{ if $i }}, {{ end }}{{ $f }}: true{{ end }} })
{{- end }}
{{- if .PartialFields }}
    .partial({ {{ range $i, $f := .PartialFields }}{{ if $i }}, {{ end }}{{ $f.TSName }}: true{{ end }} })
{{- end }}
{{- if .ExtendFields }}
    .extend({
{{- range .ExtendFields }}
{{- if .IsSelfRef }}
      get {{ .TSName }}(): {{ .ZodSchemaType }} {
        return {{ .ZodType }};
      },
{{- else }}
      {{ .TSName }}: {{ .ZodType }},
{{- end }}
{{- end }}
    })
{{- end }};
{{- else }}
  z.object({
{{- range .Fields }}
{{- if .IsSelfRef }}
    get {{ .TSName }}(): {{ .ZodSchemaType }} {
      return {{ .ZodType }};
    },
{{- else }}
    {{ .TSName }}: {{ .ZodType }},
{{- end }}
{{- end }}
  });
{{- end }}
{{- end }}
{{- if $.GenerateTypes }}
{{- if .ConcreteTypes }}
{{- if .HasExtends }}
{{- if .CoalesceTypeParams }}
export type {{ .TSName }}<S extends {{ .TSName }}Schemas = {{ .TSName }}Schemas> = {{ if .PartialFields }}optional.Optional<{{ end }}{{ if .OmittedFields }}Omit<{{ end }}{{ .ExtendsTypeName }}<S>{{ if .OmittedFields }}, {{ range $i, $f := .OmittedFields }}{{ if $i }} | {{ end }}"{{ $f }}"{{ end }}>{{ end }}{{ if .PartialFields }}, {{ range $i, $f := .PartialFields }}{{ if $i }} | {{ end }}"{{ $f.TSName }}"{{ end }}>{{ end }}{{ if .ExtendFields }} & {
{{- range .ExtendFields }}
  {{ .TSName }}{{ if or .IsOptional .IsHardOptional }}?{{ end }}: {{ .CoalescedTSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}{{ end }};
{{- else }}
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = {{ if .PartialFields }}optional.Optional<{{ end }}{{ if .OmittedFields }}Omit<{{ end }}{{ .ExtendsTypeName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>{{ if .OmittedFields }}, {{ range $i, $f := .OmittedFields }}{{ if $i }} | {{ end }}"{{ $f }}"{{ end }}>{{ end }}{{ if .PartialFields }}, {{ range $i, $f := .PartialFields }}{{ if $i }} | {{ end }}"{{ $f.TSName }}"{{ end }}>{{ end }}{{ if .ExtendFields }} & {
{{- range .ExtendFields }}
  {{ .TSName }}{{ if or .IsOptional .IsHardOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}{{ end }};
{{- end }}
{{- else }}
{{- if .ConditionalFields }}
{{- if .CoalesceTypeParams }}
export type {{ .TSName }}<S extends {{ .TSName }}Schemas = {{ .TSName }}Schemas> = {
{{- range .BaseFields }}
  {{ .TSName }}{{ if or .IsOptional .IsHardOptional }}?{{ end }}: {{ .CoalescedTSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}{{ range .ConditionalFields }} & ([S["{{ .TypeParamName | camelCase }}"]] extends [{{ .NeverType }}] ? {} : { {{ .Field.TSName }}{{ if .Field.IsHardOptional }}?{{ end }}: {{ .Field.CoalescedTSType }}{{ if .Field.IsArray }}[]{{ end }} }){{ end }};
{{- else }}
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = {
{{- range .BaseFields }}
  {{ .TSName }}{{ if or .IsOptional .IsHardOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}{{ range .ConditionalFields }} & ([{{ .TypeParamName }}] extends [{{ .NeverType }}] ? {} : { {{ .Field.TSName }}{{ if .Field.IsHardOptional }}?{{ end }}: {{ .Field.TSType }}{{ if .Field.IsArray }}[]{{ end }} }){{ end }};
{{- end }}
{{- else }}
{{- if .CoalesceTypeParams }}
export interface {{ .TSName }}<S extends {{ .TSName }}Schemas = {{ .TSName }}Schemas> {
{{- range .Fields }}
  {{ .TSName }}{{ if or .IsOptional .IsHardOptional }}?{{ end }}: {{ .CoalescedTSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}
{{- else }}
export interface {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> {
{{- range .Fields }}
  {{ .TSName }}{{ if or .IsOptional .IsHardOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}
{{- end }}
{{- end }}
{{- end }}
{{- else }}
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = z.{{ if .UseInput }}input{{ else }}infer{{ end }}<
  ReturnType<typeof {{ camelCase .TSName }}Z<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>>
>;
{{- end }}
{{- end }}
{{- else if .HasExtends }}

export const {{ camelCase .TSName }}Z = {{ range $i, $p := .ExtendsParents }}{{ if $i }}.extend({{ end }}{{ $p.Name }}{{ if $i }}.shape){{ end }}{{ end }}
{{- if .OmittedFields }}
  .omit({ {{ range $i, $f := .OmittedFields }}{{ if $i }}, {{ end }}{{ $f }}: true{{ end }} })
{{- end }}
{{- if .PartialFields }}
  .partial({ {{ range $i, $f := .PartialFields }}{{ if $i }}, {{ end }}{{ $f.TSName }}: true{{ end }} })
{{- end }}
{{- if .ExtendFields }}
  .extend({
{{- range .ExtendFields }}
{{- if .IsSelfRef }}
    get {{ .TSName }}(): {{ .ZodSchemaType }} {
      return {{ .ZodType }};
    },
{{- else }}
    {{ .TSName }}: {{ .ZodType }},
{{- end }}
{{- end }}
  })
{{- end }};
{{- if $.GenerateTypes }}
export interface {{ .TSName }} extends z.{{ if .UseInput }}input{{ else }}infer{{ end }}<typeof {{ camelCase .TSName }}Z> {}
{{- end }}
{{- else }}
{{- if .Doc }}

{{ formatDoc .TSName .Doc }}
{{- end }}
{{- if and .IsRecursive $.GenerateTypes }}
export interface {{ .TSName }} {
{{- range .Fields }}
  {{ .TSName }}{{ if or .IsOptional .IsHardOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}
export const {{ camelCase .TSName }}Z: z.ZodType<{{ .TSName }}> = z.object({
{{- range .Fields }}
{{- if .Doc }}
  {{ formatDoc .TSName .Doc }}
{{- end }}
{{- if .IsSelfRef }}
  get {{ .TSName }}() {
    return {{ .ZodType }};
  },
{{- else }}
  {{ .TSName }}: {{ .ZodType }},
{{- end }}
{{- end }}
});
{{- else }}
export const {{ camelCase .TSName }}Z = z.object({
{{- range .Fields }}
{{- if .Doc }}
  {{ formatDoc .TSName .Doc }}
{{- end }}
{{- if .IsSelfRef }}
  get {{ .TSName }}(): {{ .ZodSchemaType }} {
    return {{ .ZodType }};
  },
{{- else }}
  {{ .TSName }}: {{ .ZodType }},
{{- end }}
{{- end }}
});
{{- if $.GenerateTypes }}
export interface {{ .TSName }} extends z.{{ if .UseInput }}input{{ else }}infer{{ end }}<typeof {{ camelCase .TSName }}Z> {}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- else if .IsUnion }}
{{- with .Union }}
{{- $disc := .Discriminator }}
{{- range .Variants }}

{{ if .Doc -}}
{{ formatDoc .TypeName .Doc }}
{{ end -}}
export const {{ .SchemaName }} = {{ if .ParentSchemas }}{{ range $i, $p := .ParentSchemas }}{{ if $i }}.extend({{ end }}{{ $p }}{{ if $i }}.shape){{ end }}{{ end }}.extend({{ else }}z.object({{ end }}{
  {{ $disc }}: z.literal("{{ .Value }}"),
{{- range .Fields }}
{{- if .Doc }}
  {{ formatDoc .TSName .Doc }}
{{- end }}
  {{ .TSName }}: {{ .ZodType }},
{{- end }}
});
{{- if $.GenerateTypes }}
export interface {{ .TypeName }} extends z.infer<typeof {{ .SchemaName }}> {}
{{- end }}
{{- end }}

export const {{ .TypesConst }} = [{{ range $i, $v := .Variants }}{{ if $i }}, {{ end }}"{{ $v.Value }}"{{ end }}] as const;
export const {{ .TypeSchemaName }} = z.enum({{ .TypesConst }});
{{- if $.GenerateTypes }}
export type {{ .TypeName }} = z.infer<typeof {{ .TypeSchemaName }}>;
{{- end }}

{{ if .Doc -}}
{{ formatDoc .TSName .Doc }}
{{ end -}}
export const {{ .SchemaName }} = z.discriminatedUnion("{{ .Discriminator }}", [
{{- range .Variants }}
  {{ .SchemaName }},
{{- end }}
]);
{{- if $.GenerateTypes }}
export type {{ .TSName }} = {{ range $i, $v := .Variants }}{{ if $i }} | {{ end }}{{ $v.TypeName }}{{ end }};
{{- end }}

export const {{ .SchemasConst }}: {
  [K in {{ .TypeName }}]: z.ZodType<Extract<{{ .TSName }}, { {{ .Discriminator }}: K }>>;
} = {
{{- range .Variants }}
  {{ .Value }}: {{ .SchemaName }},
{{- end }}
};
{{- end }}
{{- end }}
{{- end }}
{{- if .Ontology }}

export const ontologyID = ontology.createIDFactory<{{ .Ontology.KeyType }}>("{{ .Ontology.TypeName }}");
export const TYPE_ONTOLOGY_ID = ontologyID({{ .Ontology.KeyZeroValue }});
{{- end }}
`))
