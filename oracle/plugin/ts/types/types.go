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
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/domain/resource"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/internal/telem"
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
	FileNamePattern string
	GenerateTypes   bool
}

const orNumericString = ".or(z.string().refine((v) => !isNaN(Number(v)))" +
	".transform(Number))"

func DefaultOptions() Options {
	return Options{
		FileNamePattern: "types.gen.ts",
		GenerateTypes:   true,
	}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (*Plugin) Name() string { return "ts/types" }

func (*Plugin) Domains() []string { return nil }

func (*Plugin) Requires() []string { return nil }

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	req = req.OmitFields("ts")
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	c, err := newCollectors(req)
	if err != nil {
		return nil, err
	}

	if err = c.structs.ForEach(
		func(outputPath string, structs []resolution.Type) error {
			return p.appendFile(resp, outputPath, structs[0], fileTypes{
				structs:  structs,
				enums:    fileEnums(structs, outputPath, c.enums, req),
				typeDefs: take(c.typeDefs, outputPath),
				unions:   take(c.unions, outputPath),
			}, req)
		},
	); err != nil {
		return nil, err
	}

	if err = c.unions.ForEach(func(outputPath string, unions []resolution.Type) error {
		return p.appendFile(resp, outputPath, unions[0], fileTypes{
			enums:    fileEnums(unions, outputPath, c.enums, req),
			typeDefs: take(c.typeDefs, outputPath),
			unions:   unions,
		}, req)
	}); err != nil {
		return nil, err
	}

	if err = c.enums.ForEach(func(outputPath string, enums []resolution.Type) error {
		return p.appendFile(resp, outputPath, enums[0], fileTypes{
			enums:    enums,
			typeDefs: take(c.typeDefs, outputPath),
		}, req)
	}); err != nil {
		return nil, err
	}

	if err = c.typeDefs.ForEach(
		func(outputPath string, typeDefs []resolution.Type) error {
			enums := enum.CollectNamespaceEnums(
				typeDefs[0].Namespace,
				outputPath,
				req.Resolutions,
				"ts",
				nil,
			)
			return p.appendFile(resp, outputPath, typeDefs[0], fileTypes{
				enums:    enums,
				typeDefs: typeDefs,
			}, req)
		},
	); err != nil {
		return nil, err
	}

	return resp, nil
}

// collectors group the types of a request by the file that declares them.
type collectors struct {
	typeDefs *framework.Collector
	structs  *framework.Collector
	unions   *framework.Collector
	enums    *framework.Collector
}

func newCollectors(req *plugin.Request) (collectors, error) {
	typeDefs := framework.NewCollector("ts", req)
	if err := typeDefs.AddAll(req.Resolutions.DistinctTypes()); err != nil {
		return collectors{}, err
	}
	if err := typeDefs.AddAll(req.Resolutions.AliasTypes()); err != nil {
		return collectors{}, err
	}
	structs, err := framework.CollectStructs("ts", req)
	if err != nil {
		return collectors{}, err
	}
	unions := framework.NewCollector("ts", req)
	if err := unions.AddAll(req.Resolutions.UnionTypes()); err != nil {
		return collectors{}, err
	}
	enums := framework.NewCollector("ts", req).
		WithPathFunc(func(typ resolution.Type) string { return output.GetPath(typ, "ts") }).
		WithSkipFunc(nil)
	for _, e := range enum.CollectWithOwnOutput(req.Resolutions.EnumTypes(), "ts") {
		if err := enums.Add(e); err != nil {
			return collectors{}, err
		}
	}
	return collectors{
		typeDefs: typeDefs,
		structs:  structs,
		unions:   unions,
		enums:    enums,
	}, nil
}

// fileTypes holds the types that one generated file declares.
type fileTypes struct {
	structs  []resolution.Type
	enums    []resolution.Type
	typeDefs []resolution.Type
	unions   []resolution.Type
}

// take removes and returns the types that c holds for outputPath, or nil.
func take(c *framework.Collector, outputPath string) []resolution.Type {
	if !c.Has(outputPath) {
		return nil
	}
	return c.Remove(outputPath)
}

// fileEnums returns the enums of the file at outputPath: those that types reference,
// those that enums holds for the file, and those of the file's namespace. It removes
// the file's entries from enums.
func fileEnums(
	types []resolution.Type,
	outputPath string,
	enums *framework.Collector,
	req *plugin.Request,
) []resolution.Type {
	result := enum.CollectReferenced(types, req.Resolutions)
	result = framework.MergeTypesByName(result, take(enums, outputPath))
	return framework.MergeTypesByName(
		result,
		enum.CollectNamespaceEnums(
			types[0].Namespace,
			outputPath,
			req.Resolutions,
			"ts",
			nil,
		),
	)
}

// appendFile generates the file at outputPath and appends it to resp. The file takes
// its namespace and source path from first.
func (p *Plugin) appendFile(
	resp *plugin.Response,
	outputPath string,
	first resolution.Type,
	types fileTypes,
	req *plugin.Request,
) error {
	content, err := p.generateFile(
		first.Namespace,
		first.FilePath,
		outputPath,
		types.structs,
		types.enums,
		types.typeDefs,
		types.unions,
		req,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to generate %s", outputPath)
	}
	resp.Files = append(resp.Files, plugin.File{
		Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
		Content: content,
	})
	return nil
}

// hasNonPrimitiveDependency returns true if a type definition has dependencies on
// non-primitive types (i.e., references other schema types that need to be declared
// before this type). This is used to determine whether a distinct type should be
// included in topological sorting.
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

// generateFile renders one types file. When two modules export a name that the file
// imports, it renders the file again with aliases.
func (p *Plugin) generateFile(
	namespace string,
	filePath string,
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	typeDefs []resolution.Type,
	unions []resolution.Type,
	req *plugin.Request,
) ([]byte, error) {
	content, conflicts, err := p.renderFile(
		namespace, filePath, outputPath, structs, enums, typeDefs, unions, req, nil,
	)
	if err != nil || len(conflicts) == 0 {
		return content, err
	}
	aliases := imports.Aliases(conflicts)
	content, conflicts, err = p.renderFile(
		namespace, filePath, outputPath, structs, enums, typeDefs, unions, req, aliases,
	)
	if err != nil {
		return nil, err
	}
	if len(conflicts) > 0 {
		return nil, errors.Newf(
			"unresolvable import name conflicts in %s: %v", outputPath, conflicts,
		)
	}
	return content, nil
}

// renderFile renders one types file, importing each name that aliases holds under its
// alias. It returns the names that two modules export to the file.
func (p *Plugin) renderFile(
	namespace string,
	filePath string,
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	typeDefs []resolution.Type,
	unions []resolution.Type,
	req *plugin.Request,
	aliases map[imports.Key]string,
) ([]byte, map[string][]string, error) {
	data := &templateData{
		Namespace:       namespace,
		FilePath:        filePath,
		OutputPath:      outputPath,
		Request:         req,
		aliases:         aliases,
		Enums:           make([]enumData, 0, len(enums)),
		TypeDefs:        make([]typeDefData, 0, len(typeDefs)),
		SortedDecls:     make([]sortedDeclData, 0),
		GenerateTypes:   p.Options.GenerateTypes,
		Manager:         imports.NewManager(),
		missingIncludes: make(set.Set[string]),
	}
	skip := func(s resolution.Type) bool { return omit.IsSkipped(s, "ts") }
	data.Resource = p.extractResource(
		slices.Concat(structs, unions),
		key.Collect(structs, req.Resolutions, skip),
		skip,
		req.Resolutions,
	)
	if data.Resource != nil {
		data.AddImport("@/resource", "resource")
	}

	primitiveTypeDefs, dependentTypeDefs := splitTypeDefs(typeDefs)
	for _, td := range primitiveTypeDefs {
		data.TypeDefs = append(data.TypeDefs, p.processTypeDef(td, data))
	}

	for _, e := range enums {
		data.Enums = append(data.Enums, p.processEnum(e))
	}

	p.addSortedDecls(data, structs, dependentTypeDefs, unions)

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, nil, err
	}
	if len(data.missingIncludes) > 0 {
		missing := data.missingIncludes.Slice()
		slices.Sort(missing)
		return nil, nil, errors.Newf(
			"types with no generated TypeScript declaration need a @ts include "+
				"module: %s",
			strings.Join(missing, ", "),
		)
	}
	return buf.Bytes(), data.Conflicts(), nil
}

// addSortedDecls adds the declarations of structs, typeDefs, and unions to data, sorted
// topologically so that dependencies come before dependents.
func (p *Plugin) addSortedDecls(
	data *templateData,
	structs, typeDefs, unions []resolution.Type,
) {
	table := data.Request.Resolutions
	// Structs come first so that when there's a cycle, typedefs (like array types) are
	// placed after their element types. Array typedefs can't use getters, so they must
	// come after their element types are defined.
	sortedTypes := table.TopologicalSort(slices.Concat(structs, typeDefs, unions))
	// A struct field that references a type declared later needs a getter for lazy
	// evaluation.
	data.DeclOrder = make(map[string]int, len(sortedTypes))
	for i, typ := range sortedTypes {
		data.DeclOrder[typ.QualifiedName] = i
	}
	for i, typ := range sortedTypes {
		data.CurrentDeclIndex = i
		if decl, ok := p.processSortedDecl(typ, table, data); ok {
			data.SortedDecls = append(data.SortedDecls, decl)
		}
	}
}

// splitTypeDefs splits typeDefs into those with only primitive dependencies, which need
// no sorting, and those that depend on schema types (e.g., Params Param[]), which are
// sorted topologically with aliases and structs.
func splitTypeDefs(
	typeDefs []resolution.Type,
) (primitive, dependent []resolution.Type) {
	for _, td := range typeDefs {
		switch td.Form.(type) {
		case resolution.AliasForm:
			dependent = append(dependent, td)
		case resolution.DistinctForm:
			if hasNonPrimitiveDependency(td) {
				dependent = append(dependent, td)
			} else {
				primitive = append(primitive, td)
			}
		default:
			primitive = append(primitive, td)
		}
	}
	return primitive, dependent
}

// processSortedDecl builds the declaration for one topologically sorted type. It
// returns false for a form that has no sorted declaration.
func (p *Plugin) processSortedDecl(
	typ resolution.Type,
	table *resolution.Table,
	data *templateData,
) (sortedDeclData, bool) {
	switch form := typ.Form.(type) {
	case resolution.AliasForm:
		// Generic aliases need full struct treatment for type params
		if form.IsGeneric() {
			return sortedDeclData{
				IsStruct: true,
				Struct:   p.processStruct(typ, table, data),
			}, true
		}
		return sortedDeclData{
			IsTypeDef: true,
			TypeDef:   p.processTypeDef(typ, data),
		}, true
	case resolution.DistinctForm:
		return sortedDeclData{
			IsTypeDef: true,
			TypeDef:   p.processTypeDef(typ, data),
		}, true
	case resolution.StructForm:
		return sortedDeclData{
			IsStruct: true,
			Struct:   p.processStruct(typ, table, data),
		}, true
	case resolution.UnionForm:
		return sortedDeclData{
			IsUnion: true,
			Union:   p.processUnion(typ, table, data),
		}, true
	}
	return sortedDeclData{}, false
}

func findFieldTypeOverride(
	structs []resolution.Type,
	fieldName, domainName string,
) string {
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

func (p *Plugin) extractResource(
	structs []resolution.Type,
	keyFields []key.Field,
	skip resource.SkipFunc,
	table *resolution.Table,
) *resourceData {
	data := resource.Extract(structs, keyFields, skip, table)
	if data == nil {
		return nil
	}
	keyType := lo.Capitalize(FieldName(data.KeyField.Name))
	primitive := data.KeyField.Primitive
	if override := findFieldTypeOverride(
		structs,
		data.KeyField.Name,
		"ts",
	); override != "" {
		primitive = override
	}
	// Also check if the key field's type itself has a @ts type override (e.g., Key
	// uint64 { @ts type string })
	if override := findKeyTypeTypeOverride(
		structs,
		data.KeyField.Name,
		table,
	); override != "" {
		primitive = override
	}
	keyZeroValue := primitiveZeroValue(primitive)
	return &resourceData{
		TypeName:     data.TypeName,
		KeyType:      keyType,
		KeyZeroValue: keyZeroValue,
	}
}

func findKeyTypeTypeOverride(
	structs []resolution.Type,
	keyFieldName string,
	table *resolution.Table,
) string {
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
	switch form := td.Form.(type) {
	case resolution.DistinctForm:
		toNumber, toString := typeDefConversions(td)
		if typeOverride := getTypeTypeOverride(td, "ts"); typeOverride != "" {
			return p.processOverriddenTypeDef(
				td, form, typeOverride, toNumber, toString, data,
			)
		}
		zodType := p.typeDefZod(td, form.Base, data)
		if toNumber {
			zodType += orNumericString
		}
		if toString {
			zodType = fmt.Sprintf(
				"z.union([%s, z.number()]).transform(String)",
				zodType,
			)
		}
		return typeDefData{
			Name:    td.Name,
			TSName:  td.Name,
			TSType:  p.typeDefBaseToTS(&form.Base, data),
			ZodType: zodType,
		}
	case resolution.AliasForm:
		zodType := p.typeDefZod(td, form.Target, data)
		return typeDefData{
			Name:    td.Name,
			TSName:  td.Name,
			TSType:  p.typeDefBaseToTS(&form.Target, data),
			ZodType: zodType,
		}
	default:
		return typeDefData{
			Name:    td.Name,
			TSName:  td.Name,
			TSType:  "unknown",
			ZodType: "z.unknown()",
		}
	}
}

// typeDefConversions reports whether td carries the to_number and to_string ts
// expressions.
func typeDefConversions(td resolution.Type) (toNumber, toString bool) {
	exprs := tsExpressions(td)
	_, toNumber = exprs.Find("to_number")
	_, toString = exprs.Find("to_string")
	return toNumber, toString
}

// processOverriddenTypeDef builds a distinct type whose ts type override replaces its
// base.
func (p *Plugin) processOverriddenTypeDef(
	td resolution.Type,
	form resolution.DistinctForm,
	typeOverride string,
	toNumber, toString bool,
	data *templateData,
) typeDefData {
	zodType := primitiveToZod(typeOverride, data)
	// The override replaces the schema of every type in the chain, so the type applies
	// all of their rules.
	rules := validation.TypeRules(validation.TypeChain(
		resolution.TypeRef{Name: td.QualifiedName},
		data.Request.Resolutions,
	))
	if !validation.IsEmpty(rules) {
		result := p.applyValidation(
			zodType,
			rules,
			nil,
			form.Base,
			td.Name,
			data.Request.Resolutions,
			data,
			typeOverride,
		)
		zodType = result.ZodType
	}
	if toNumber {
		zodType += orNumericString
	}
	if toString {
		zodType = fmt.Sprintf(
			"z.union([%s, z.number(), z.bigint()]).transform(String)",
			zodType,
		)
	}
	return typeDefData{
		Name:    td.Name,
		TSName:  td.Name,
		TSType:  primitiveToTS(typeOverride),
		ZodType: zodType,
	}
}

// typeDefZod returns the Zod schema for a type definition over base, with the
// validation rules of td applied.
func (p *Plugin) typeDefZod(
	td resolution.Type,
	base resolution.TypeRef,
	data *templateData,
) string {
	var zodType string
	if isArrayTypeRef(base) && len(base.TypeArgs) > 0 {
		elemZod := p.typeRefToZod(&base.TypeArgs[0], data.Request.Resolutions, data)
		if base.ArraySize != nil {
			zodType = fmt.Sprintf("z.tuple([%s])", repeatJoin(elemZod, *base.ArraySize))
		} else {
			zodType = elemZod + ".array().default(() => [])"
		}
	} else {
		zodType = p.typeDefBaseToZod(&base, data)
	}
	validateDomain, ok := td.Domains["validate"]
	if !ok {
		return zodType
	}
	return p.applyValidation(
		zodType,
		validation.Parse(validateDomain),
		nil,
		base,
		td.Name,
		data.Request.Resolutions,
		data,
		"",
	).ZodType
}

func (p *Plugin) typeDefBaseToZod(
	typeRef *resolution.TypeRef,
	data *templateData,
) string {
	if typeRef == nil {
		return "z.unknown()"
	}
	return p.typeRefToZod(typeRef, data.Request.Resolutions, data)
}

func (p *Plugin) typeDefBaseToTS(
	typeRef *resolution.TypeRef,
	data *templateData,
) string {
	if typeRef == nil {
		return "unknown"
	}
	return p.typeRefToTS(typeRef, data.Request.Resolutions, data, false)
}

// filterRequests returns the qualified names of the requests derived from filters.
func filterRequests(table *resolution.Table) set.Set[string] {
	names := make(set.Set[string])
	for _, typ := range table.StructTypes() {
		if typ.Filter != nil {
			names.Add(typ.Filter.Request)
		}
	}
	return names
}

// isExtendBase reports whether another type's generated zod schema calls .extend on
// entry's schema: struct extends bases, union shared bases, and union variant payloads.
// Such schemas must stay ZodObjects, since the z.ZodType annotation used to break
// recursive inference has no .extend.
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

func (p *Plugin) processStruct(
	entry resolution.Type,
	table *resolution.Table,
	data *templateData,
) structData {
	if aliasForm, isAlias := entry.Form.(resolution.AliasForm); isAlias {
		return p.processAlias(entry, aliasForm, table, data)
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
		Strict:      resolution.InFilterRequest(entry, table),
		Request:     filterRequests(table).Contains(entry.QualifiedName),
	}
	applyStructTSExpressions(&sd, entry)
	if sd.Handwritten {
		return sd
	}
	if extendsUnion(&sd, form, table, data) {
		return sd
	}
	p.processTypeParams(&sd, form.TypeParams, table, data)
	for _, tp := range sd.TypeParams {
		if tp.IsPrimitiveConstrained {
			addXImport(data, xImport{name: "numeric", submodule: "numeric"})
			break
		}
	}
	if sd.IsGeneric {
		sd.IsPrimitiveConstrainedGeneric = allPrimitiveConstrainedWithDefaults(
			sd.TypeParams,
		)
	}

	// A domain removal (-@domain) cannot be expressed through Zod extend/omit chaining
	// — the parent schema still carries the domain — so flatten via UnifiedFields, as
	// for a parent absent from TypeScript. Typeless overrides are already resolved by
	// the analyzer.
	if len(form.Extends) > 0 && !resolver.HasDomainOmissions(form) &&
		resolver.ParentsExist(form.Extends, "ts", table) &&
		p.collectExtendsParents(&sd, form, table, data) {
		p.processExtendingFields(&sd, entry, form, table, data)
		if sd.ConcreteTypes && len(sd.PartialFields) > 0 {
			addXImport(data, xImport{name: "optional", submodule: "optional"})
		}
	} else {
		p.processFlatFields(&sd, entry, form, table, data)
	}
	if sd.CoalesceTypeParams {
		computeCoalescedTypes(&sd)
	}
	return sd
}

func (p *Plugin) processAlias(
	entry resolution.Type,
	aliasForm resolution.AliasForm,
	table *resolution.Table,
	data *templateData,
) structData {
	sd := structData{
		Name:          entry.Name,
		TSName:        entry.Name,
		Doc:           doc.Get(entry.Domains),
		IsGeneric:     aliasForm.IsGeneric(),
		IsSingleParam: len(aliasForm.TypeParams) == 1,
		IsAlias:       true,
	}
	for _, expr := range tsExpressions(entry) {
		applySharedTSExpression(&sd, expr)
	}
	if sd.Handwritten {
		return sd
	}
	p.processTypeParams(&sd, aliasForm.TypeParams, table, data)
	target := aliasForm.Target
	if !isArrayTypeRef(target) || len(target.TypeArgs) == 0 {
		sd.AliasOf = p.typeRefToZod(&target, table, data)
		return sd
	}
	elemZod := p.typeRefToZod(&target.TypeArgs[0], table, data)
	if target.ArraySize == nil {
		sd.AliasOf = elemZod + ".array().default(() => [])"
		return sd
	}
	elements := make([]string, *target.ArraySize)
	for i := range elements {
		elements[i] = elemZod
	}
	sd.AliasOf = fmt.Sprintf("z.tuple([%s])", strings.Join(elements, ", "))
	return sd
}

func tsExpressions(entry resolution.Type) resolution.Expressions {
	tsDomain, ok := entry.Domains["ts"]
	if !ok {
		return nil
	}
	return tsDomain.Expressions
}

func applyStructTSExpressions(sd *structData, entry resolution.Type) {
	for _, expr := range tsExpressions(entry) {
		if applySharedTSExpression(sd, expr) {
			continue
		}
		switch expr.Name {
		case "concrete_types":
			sd.ConcreteTypes = true
		case "coalesce_type_params":
			sd.CoalesceTypeParams = true
		}
	}
}

// applySharedTSExpression applies a @ts expression that structs and aliases both
// accept, and reports whether expr was one.
func applySharedTSExpression(sd *structData, expr resolution.Expression) bool {
	switch expr.Name {
	case "use_input":
		sd.UseInput = true
	case "type_only":
		sd.TypeOnly = true
	case "omit", "hand":
		sd.Handwritten = true
	case "name":
		if len(expr.Values) > 0 {
			sd.TSName = expr.Values[0].StringValue
		}
	default:
		return false
	}
	return true
}

// extendsUnion handles a @create New whose base is a discriminated union, which
// projects to `type New… = z.input<typeof unionZ>`. The union's fields are not
// flattened, since an interface cannot extend a union. It reports whether the base is a
// union, in which case sd is complete.
func extendsUnion(
	sd *structData,
	form resolution.StructForm,
	table *resolution.Table,
	data *templateData,
) bool {
	if len(form.Extends) != 1 {
		return false
	}
	base, ok := form.Extends[0].Resolve(table)
	if !ok {
		return false
	}
	if _, isUnion := base.Form.(resolution.UnionForm); !isUnion {
		return false
	}
	sd.HasExtends = true
	sd.BaseIsUnion = true
	schemaName := casing.CamelAcronym(domain.GetName(base, "ts")) + "Z"
	importLocalHand(base, schemaName, data)
	sd.ExtendsName = schemaName
	for _, f := range form.OmittedFields {
		sd.OmittedFields = append(sd.OmittedFields, FieldName(f))
	}
	return true
}

func (p *Plugin) processTypeParams(
	sd *structData,
	params []resolution.TypeParam,
	table *resolution.Table,
	data *templateData,
) {
	for _, tp := range params {
		sd.TypeParams = append(sd.TypeParams, p.processTypeParam(tp, table, data))
	}
	for _, tp := range sd.TypeParams {
		if tp.IsJSON || strings.Contains(tp.Constraint, "record.") ||
			strings.Contains(tp.Default, "record.") {
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
}

func allPrimitiveConstrainedWithDefaults(params []typeParamData) bool {
	for _, tp := range params {
		if !tp.IsPrimitiveConstrained || !tp.HasDefault {
			return false
		}
	}
	return true
}

func optionalTypeParamNames(params []resolution.TypeParam) set.Set[string] {
	names := make(set.Set[string])
	for _, tp := range params {
		if tp.Optional {
			names.Add(tp.Name)
		}
	}
	return names
}

// collectExtendsParents appends the schema of each struct parent to sd.ExtendsParents
// for merge chaining. It stops at the first parent that is not a struct and reports
// whether every parent was one.
func (p *Plugin) collectExtendsParents(
	sd *structData,
	form resolution.StructForm,
	table *resolution.Table,
	data *templateData,
) bool {
	for _, extendsRef := range form.Extends {
		target, _, ok := resolution.Dealias(extendsRef, table)
		if !ok {
			return false
		}
		if _, isStruct := target.Form.(resolution.StructForm); !isStruct {
			return false
		}
		parentType, _ := extendsRef.Resolve(table)
		parentTSName := domain.GetName(parentType, "ts")
		schemaName := casing.CamelAcronym(parentTSName) + "Z"
		importLocalHand(parentType, schemaName, data)
		if parentType.FilePath != data.FilePath {
			ns := importNamespace(
				parentType.Namespace,
				output.GetPath(parentType, "ts"),
				data,
			)
			schemaName = ns + "." + schemaName
		}
		sd.ExtendsParents = append(sd.ExtendsParents, extendsParentInfo{
			Name:     schemaName,
			TypeName: parentTSName,
			Schema:   p.typeRefToZodInternal(&extendsRef, table, data, true),
		})
	}
	return len(sd.ExtendsParents) > 0
}

// processExtendingFields fills sd for a struct that extends its parents' schemas,
// keeping only the fields that differ from the parents.
func (p *Plugin) processExtendingFields(
	sd *structData,
	entry resolution.Type,
	form resolution.StructForm,
	table *resolution.Table,
	data *templateData,
) {
	sd.HasExtends = true
	sd.ExtendsName = sd.ExtendsParents[0].Name
	sd.ExtendsTypeName = sd.ExtendsParents[0].TypeName
	for _, f := range form.OmittedFields {
		sd.OmittedFields = append(sd.OmittedFields, FieldName(f))
	}
	parentFields := make(map[string]resolution.Field)
	for _, extendsRef := range form.Extends {
		parentType, _, _ := resolution.Dealias(extendsRef, table)
		for _, pf := range resolution.UnifiedFields(parentType, table) {
			if _, exists := parentFields[pf.Name]; !exists {
				parentFields[pf.Name] = pf // First parent wins
			}
		}
	}
	for _, field := range form.Fields {
		parentField, existsInParent := parentFields[field.Name]
		if existsInParent && isFieldUnchanged(parentField, field) {
			continue
		}
		if existsInParent && isOnlyOptionalityChange(parentField, field) {
			sd.PartialFields = append(
				sd.PartialFields,
				fieldData{TSName: FieldName(field.Name)},
			)
			continue
		}
		if existsInParent {
			sd.OmittedFields = append(sd.OmittedFields, FieldName(field.Name))
		}
		sd.ExtendFields = append(
			sd.ExtendFields,
			p.processField(field, entry, table, data, sd.ConcreteTypes),
		)
	}
	// An input type makes defaulted base fields optional: the default fills any absent
	// value. The non-concrete path gets this from `z.input` plus each field's own
	// `.default()`, but the concrete factory's type is an `optional.Optional<...>`
	// wrapper rather than `z.input`, so the defaulted fields must be folded into the
	// partial set explicitly here.
	if sd.UseInput && sd.ConcreteTypes {
		p.addInputBaseFields(sd, form, table, data)
	}
}

func (p *Plugin) addInputBaseFields(
	sd *structData,
	form resolution.StructForm,
	table *resolution.Table,
	data *templateData,
) {
	handled := make(set.Set[string])
	for _, f := range sd.PartialFields {
		handled.Add(f.TSName)
	}
	for _, f := range sd.OmittedFields {
		handled.Add(f)
	}
	for _, f := range sd.ExtendFields {
		handled.Add(f.TSName)
	}
	optionalTypeParams := optionalTypeParamNames(form.TypeParams)
	for _, extendsRef := range form.Extends {
		parentType, _, _ := resolution.Dealias(extendsRef, table)
		for _, pf := range resolution.UnifiedFields(parentType, table) {
			name := FieldName(pf.Name)
			if handled.Contains(name) {
				continue
			}
			if sd.TypeOnly &&
				p.addTypeOnlyInputField(sd, pf, optionalTypeParams, table, data) {
				handled.Add(name)
				continue
			}
			if pf.Default == nil {
				continue
			}
			handled.Add(name)
			sd.PartialFields = append(sd.PartialFields, fieldData{TSName: name})
		}
	}
}

// addTypeOnlyInputField re-projects base field pf for the input type of a type-only
// struct, and reports whether pf needed it.
func (p *Plugin) addTypeOnlyInputField(
	sd *structData,
	pf resolution.Field,
	optionalTypeParams set.Set[string],
	table *resolution.Table,
	data *templateData,
) bool {
	name := FieldName(pf.Name)
	// A base field whose type is itself a `@create`-d struct must carry that struct's
	// input projection (its `New`), not the output type the optional.Optional wrapper
	// would inherit. The New makes the nested struct's defaulted fields optional too.
	if newType, ok := p.createNewRefForField(pf, table, data); ok {
		sd.OmittedFields = append(sd.OmittedFields, name)
		sd.ExtendFields = append(sd.ExtendFields, fieldData{
			TSName:     name,
			IsOptional: pf.Optional,
			TSType:     newType,
		})
		return true
	}
	// A conditional (type-param-typed) base field projects as z.infer in the base type.
	// The New is an input type, so it must re-project the field as z.input. The base's
	// z.infer projection is stripped via Omit so only the z.input conditional survives.
	if !pf.Type.IsTypeParam() || pf.Type.TypeParam == nil ||
		!optionalTypeParams.Contains(pf.Type.TypeParam.Name) {
		return false
	}
	sd.ConditionalOmittedFields = append(sd.ConditionalOmittedFields, name)
	sd.ConditionalFields = append(sd.ConditionalFields, conditionalFieldData{
		TypeParamName: pf.Type.TypeParam.Name,
		NeverType:     "z.ZodNever",
		UseInput:      true,
		Field:         fieldData{TSName: name, IsOptional: false},
	})
	return true
}

// processFlatFields fills sd with every field of entry, its parents' included.
func (p *Plugin) processFlatFields(
	sd *structData,
	entry resolution.Type,
	form resolution.StructForm,
	table *resolution.Table,
	data *templateData,
) {
	allFields := resolution.UnifiedFields(entry, table)
	sd.Fields = make([]fieldData, 0, len(allFields))
	optionalTypeParams := optionalTypeParamNames(form.TypeParams)
	for _, field := range allFields {
		if sd.Strict && field.Default != nil {
			field.Default = nil
			field.Optional = true
		}
		fd := p.processField(field, entry, table, data, sd.ConcreteTypes)
		sd.Fields = append(sd.Fields, fd)
		if !sd.ConcreteTypes || !field.Type.IsTypeParam() ||
			field.Type.TypeParam == nil ||
			!optionalTypeParams.Contains(field.Type.TypeParam.Name) {
			sd.BaseFields = append(sd.BaseFields, fd)
			continue
		}
		tp := field.Type.TypeParam
		effectiveRef := tp.Constraint
		if effectiveRef == nil && tp.Default != nil && !tp.Optional {
			effectiveRef = tp.Default
		}
		sd.ConditionalFields = append(sd.ConditionalFields, conditionalFieldData{
			Field:         fd,
			TypeParamName: field.Type.TypeParam.Name,
			NeverType:     "z.ZodNever",
			FallbackSchemaType: fallbackSchemaTypeForConstraint(
				effectiveRef,
				table,
				data,
			),
		})
	}
}

func isFieldUnchanged(parent, child resolution.Field) bool {
	childIsOptional := child.Optional
	parentIsOptional := parent.Optional
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
	childIsOptional := child.Optional
	parentIsOptional := parent.Optional
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
		sd.Fields[i].CoalescedTSType = coalesceTSType(
			sd.Fields[i].TSType,
			sd.TypeParams,
		)
	}
	for i := range sd.BaseFields {
		sd.BaseFields[i].CoalescedTSType = coalesceTSType(
			sd.BaseFields[i].TSType,
			sd.TypeParams,
		)
	}
	for i := range sd.ExtendFields {
		sd.ExtendFields[i].CoalescedTSType = coalesceTSType(
			sd.ExtendFields[i].TSType,
			sd.TypeParams,
		)
	}
	for i := range sd.ConditionalFields {
		sd.ConditionalFields[i].Field.CoalescedTSType = coalesceTSType(
			sd.ConditionalFields[i].Field.TSType,
			sd.TypeParams,
		)
	}
}

// importNamespace imports the TS module generated for namespace at outputPath, or at
// namespace when outputPath is empty, and returns the identifier that qualifies its
// members. A module in an ancestor directory is imported from its types.gen file,
// because its barrel re-exports the importing module.
func importNamespace(namespace, outputPath string, data *templateData) string {
	if outputPath == "" {
		outputPath = namespace
	}
	ident := casing.CamelAcronym(namespace)
	if strings.HasPrefix(data.OutputPath, outputPath+"/") {
		path := paths.CalculateImport(data.OutputPath, outputPath+"/types.gen")
		if alias, ok := data.aliases[imports.Key{Path: path, Name: ident}]; ok {
			ident = alias
		}
		data.AddNamespaceImport(path, ident)
		return ident
	}
	path := paths.CalculateImport(data.OutputPath, outputPath)
	if alias, ok := data.aliases[imports.Key{Path: path, Name: ident}]; ok {
		data.AddAliasedImport(path, ident, alias)
		return alias
	}
	data.AddImport(path, ident)
	return ident
}

// qualifier imports the module that declares typ, a type from another schema, and
// returns the name that the file gives the module.
func qualifier(
	typ resolution.Type,
	table *resolution.Table,
	data *templateData,
) string {
	outputPath := output.GetPath(typ, "ts")
	if _, isEnum := typ.Form.(resolution.EnumForm); isEnum {
		outputPath = enum.FindOutputPath(typ, table, "ts")
	}
	return importNamespace(typ.Namespace, outputPath, data)
}

// FieldName converts a field identifier to camelCase using the naive conversion the
// JSON codec's snake/camel round-trip relies on, preserving only a trailing acronym run
// of two or more uppercase letters in the source ("ClientXY" -> "clientXY", "EntityID"
// -> "entityID"). Use this for wire field keys, discriminators, and key-field names,
// never for type or schema-const identifiers.
func FieldName(s string) string {
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

// parentSchemaName resolves a base or payload type reference to its TS schema const
// name (e.g. "baseAIChannelZ"), importing and namespace-qualifying it when it lives in
// another output. It reports false when the reference does not resolve to a struct that
// can be composed. The resolution mirrors the struct-extends path so union variants and
// structs compose identically.
func parentSchemaName(
	ref resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) (string, bool) {
	target, _, ok := resolution.Dealias(ref, table)
	if !ok {
		return "", false
	}
	if _, isStruct := target.Form.(resolution.StructForm); !isStruct {
		return "", false
	}
	// An alias re-exports its target's schema under its own name.
	parent, _ := ref.Resolve(table)
	name := casing.CamelAcronym(domain.GetName(parent, "ts")) + "Z"
	importLocalHand(parent, name, data)
	if parent.FilePath != data.FilePath {
		ns := importNamespace(parent.Namespace, output.GetPath(parent, "ts"), data)
		name = ns + "." + name
	}
	return name, true
}

// createNewRefForField reports whether field's type resolves to a `@create`-d struct
// carrying a synthesized `New`, and if so returns the rendered TS type referencing that
// `New` with the struct's primary details type passed as a zod schema (e.g.
// "status.New<typeof statusDetailsZ>"). It follows aliases (e.g. a `Status<D>` alias to
// `status.Status<...>`), substituting each alias's type params with the field's type
// args so the primary details type is the concrete one the field binds. It reports
// false when the field type does not resolve to a create struct with a synthesized New,
// or when it is a type param, primitive, array, or map.
//
// The details schema ref renders as `typeof <schema>` for a non-generic details struct
// and `ReturnType<typeof <schema>>` for a generic one (whose schema is a factory
// function), matching how the plugin annotates schema types elsewhere.
func (p *Plugin) createNewRefForField(
	field resolution.Field,
	table *resolution.Table,
	data *templateData,
) (string, bool) {
	ref := field.Type
	if ref.IsTypeParam() || ref.Name == "" || resolution.IsPrimitive(ref.Name) ||
		ref.Name == "Array" || ref.Name == "Map" {
		return "", false
	}
	resolved, ref, ok := resolution.Dealias(ref, table)
	if !ok {
		return "", false
	}
	if _, isStruct := resolved.Form.(resolution.StructForm); !isStruct {
		return "", false
	}
	if _, isCreate := resolved.Domains["create"]; !isCreate {
		return "", false
	}
	if _, exists := table.Get(resolved.Namespace + ".New"); !exists {
		return "", false
	}
	if len(ref.TypeArgs) == 0 {
		return "", false
	}
	detailsRef := ref.TypeArgs[0]
	detailsSchema, ok := p.detailsSchemaRef(detailsRef, table, data)
	if !ok {
		return "", false
	}
	newName := "New"
	if resolved.FilePath != data.FilePath {
		ns := importNamespace(resolved.Namespace, output.GetPath(resolved, "ts"), data)
		newName = ns + ".New"
	}
	return fmt.Sprintf("%s<%s>", newName, detailsSchema), true
}

// detailsSchemaRef renders the zod schema reference for a details type bound to a
// `@create` base's primary param. A non-generic struct's schema is a const, so it
// renders as `typeof <schema>`; a generic struct's schema is a factory function,
// so it renders as `ReturnType<typeof <schema>>` (without threading type args, to
// reference the factory's broadest return). It reports false when the details type
// is not a struct.
func (p *Plugin) detailsSchemaRef(
	ref resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) (string, bool) {
	target, _, ok := resolution.Dealias(ref, table)
	if !ok {
		return "", false
	}
	if _, isStruct := target.Form.(resolution.StructForm); !isStruct {
		return "", false
	}
	resolved, _ := ref.Resolve(table)
	params := resolution.TypeParams(resolved.Form)
	importLocalHand(
		resolved,
		casing.CamelAcronym(domain.GetName(resolved, "ts"))+"Z",
		data,
	)
	prefix := ""
	if resolved.FilePath != data.FilePath {
		ns := importNamespace(resolved.Namespace, output.GetPath(resolved, "ts"), data)
		prefix = ns + "."
	}
	schema := prefix + casing.CamelAcronym(domain.GetName(resolved, "ts")) + "Z"
	if len(params) > 0 {
		return fmt.Sprintf("ReturnType<typeof %s>", schema), true
	}
	return "typeof " + schema, true
}

func coalesceTSType(tsType string, typeParams []typeParamData) string {
	sorted := make([]typeParamData, len(typeParams))
	copy(sorted, typeParams)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Name) > len(sorted[j].Name)
	})
	result := tsType
	for _, tp := range sorted {
		// Match the type-param name only as a whole identifier so a name that
		// is a substring of a larger identifier (e.g. `Type` inside `ReturnType`)
		// is left untouched.
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(tp.Name) + `\b`)
		result = re.ReplaceAllString(result, `S["`+casing.CamelAcronym(tp.Name)+`"]`)
	}
	return result
}

func (p *Plugin) processTypeParam(
	tp resolution.TypeParam,
	table *resolution.Table,
	data *templateData,
) typeParamData {
	tpd := typeParamData{Name: tp.Name, Constraint: "z.ZodType"}
	if tp.Constraint != nil {
		if resolution.IsPrimitive(tp.Constraint.Name) &&
			tp.Constraint.Name == "record" {
			tpd.IsJSON = true
			tpd.Constraint = "z.ZodType<record.Unknown>"
		}
		if resolution.IsPrimitive(tp.Constraint.Name) &&
			tp.Constraint.Name == "string" {
			tpd.Constraint = "z.ZodType<string>"
		}
		if tp.Constraint.Name == "numeric" {
			tpd.IsPrimitiveConstrained = true
			tpd.BareConstraint = "numeric.Value"
		}
		if typeName, _, ok := enumNames(*tp.Constraint, table, data); ok {
			tpd.Constraint = "z.ZodType<" + typeName + ">"
		}
	}
	if tp.Optional {
		tpd.HasDefault = true
		tpd.Default = "z.ZodNever"
		tpd.DefaultValue = "z.unknown()"
	} else if tp.Default != nil {
		tpd.HasDefault = true
		if _, schema, ok := enumNames(*tp.Default, table, data); ok {
			tpd.Default = "typeof " + schema
			tpd.DefaultValue = schema
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
	"record": {
		zodType:  "z.ZodType<record.Unknown>",
		zodValue: "record.unknownZ().default(() => ({}))",
	},
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

func fallbackForConstraint(
	constraint *resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	if constraint == nil {
		return "z.unknown().optional()"
	}
	if _, schema, ok := enumNames(*constraint, table, data); ok {
		return schema
	}
	return defaultValueToTS(constraint.Name)
}

// enumNames returns the type and schema names that ref, an enum or an alias of one, has
// in the file data generates, and imports the namespace that declares ref. ok is false
// when ref is not an enum.
func enumNames(
	ref resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) (typeName, schema string, ok bool) {
	target, _, ok := resolution.Dealias(ref, table)
	if !ok {
		return "", "", false
	}
	if _, isEnum := target.Form.(resolution.EnumForm); !isEnum {
		return "", "", false
	}
	declared, _ := ref.Resolve(table)
	typeName = domain.GetName(declared, "ts")
	schema = casing.CamelAcronym(typeName) + "Z"
	if declared.FilePath == data.FilePath {
		return typeName, schema, true
	}
	outputPath := output.GetPath(declared, "ts")
	if _, isEnum := declared.Form.(resolution.EnumForm); isEnum {
		outputPath = enum.FindOutputPath(declared, table, "ts")
	}
	ns := importNamespace(declared.Namespace, outputPath, data)
	return ns + "." + typeName, ns + "." + schema, true
}

func fallbackSchemaTypeForConstraint(
	constraint *resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	if constraint == nil {
		return "z.ZodOptional<z.ZodUnknown>"
	}
	if _, schema, ok := enumNames(*constraint, table, data); ok {
		return "typeof " + schema
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
func isForwardReference(
	t resolution.TypeRef,
	data *templateData,
	table *resolution.Table,
) bool {
	if data.DeclOrder == nil {
		return false
	}

	var checkRef func(ref resolution.TypeRef) bool
	checkRef = func(ref resolution.TypeRef) bool {
		if ref.Name == "" || ref.IsTypeParam() {
			return false
		}
		if resolution.IsPrimitive(ref.Name) || ref.Name == "Array" ||
			ref.Name == "Map" {
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

// overrideSchema sets the schemas of a field with a `@ts type` override.
func (p *Plugin) overrideSchema(
	fd *fieldData,
	field resolution.Field,
	parentType resolution.Type,
	typeOverride string,
	table *resolution.Table,
	data *templateData,
	needsTypeImports bool,
) {
	// A `@ts type` override may name either a primitive (e.g. `string`) or another
	// schema type (e.g. `telem.TimeRangeBounded`). When it resolves to a known
	// non-primitive type, route it through the normal type-ref machinery so its schema
	// reference and import are emitted correctly; otherwise treat it as a primitive.
	// The override may be qualified (cross-namespace, e.g. telem.TimeRangeBounded) or
	// unqualified (same namespace as the field): try the qualified name first, then
	// resolve against the field's own namespace.
	overrideType, overrideResolves := table.Get(typeOverride)
	if !overrideResolves {
		overrideType, overrideResolves = table.Lookup(
			parentType.Namespace,
			typeOverride,
		)
	}
	if overrideResolves && !resolution.IsPrimitive(typeOverride) &&
		typeOverride != "record" {
		overrideRef := resolution.TypeRef{Name: overrideType.QualifiedName}
		fd.ZodType = p.typeRefToZod(&overrideRef, table, data)
		fd.TSType = p.typeRefToTS(&overrideRef, table, data, needsTypeImports)
		fd.ZodSchemaType = p.typeRefToZodSchemaType(&overrideRef, table, data)
	} else {
		fd.ZodType = primitiveToZod(typeOverride, data)
		fd.TSType = primitiveToTS(typeOverride)
		fd.ZodSchemaType = primitiveToZodSchemaType(typeOverride)
	}
	// The override replaces the schema of every type in the chain, so the field applies
	// all of their rules.
	rules := validation.FieldRules(field, table)
	if !validation.IsEmpty(rules) || field.Default != nil {
		result := p.applyValidation(
			fd.ZodType,
			rules,
			field.Default,
			field.Type,
			field.Name,
			table,
			data,
			typeOverride,
		)
		fd.ZodType = result.ZodType
		if result.HasDefault {
			fd.ZodSchemaType = defaultSchemaWrapper(result, fd.ZodSchemaType)
		}
	}
}

// fieldSchema sets the schemas of a field from its own type.
func (p *Plugin) fieldSchema(
	fd *fieldData,
	field resolution.Field,
	table *resolution.Table,
	data *templateData,
	needsTypeImports bool,
) {
	isArray := field.Type.Name == "Array"
	typeRefToProcess := &field.Type
	if isArray && len(field.Type.TypeArgs) > 0 {
		typeRefToProcess = &field.Type.TypeArgs[0]
	}
	fd.ZodType = p.typeRefToZod(typeRefToProcess, table, data)
	fd.TSType = p.typeRefToTS(typeRefToProcess, table, data, needsTypeImports)
	fd.ZodSchemaType = p.typeRefToZodSchemaType(typeRefToProcess, table, data)
	// An array field's default applies to the array that wrapArray builds, not to the
	// element schema processed here.
	elemDefault := field.Default
	if isArray {
		elemDefault = nil
	}
	rules := validation.Merge(
		validation.Parse(field.Domains["validate"]),
		handTypeRules(field.Type, table),
	)
	if validation.IsEmpty(rules) && elemDefault == nil {
		return
	}
	// A type-param field's schema is "param ?? fallback", and only the fallback takes
	// the rules.
	paramPart, target := "", fd.ZodType
	if sepIndex := strings.Index(fd.ZodType, " ?? "); sepIndex > 0 {
		paramPart, target = fd.ZodType[:sepIndex+4], fd.ZodType[sepIndex+4:]
	}
	result := p.applyValidation(
		target,
		rules,
		elemDefault,
		field.Type,
		field.Name,
		table,
		data,
		"",
	)
	fd.ZodType = paramPart + result.ZodType
	if result.HasDefault {
		fd.ZodSchemaType = defaultSchemaWrapper(result, fd.ZodSchemaType)
	}
}

// wrapArray wraps the element schemas of an array field in an array schema.
func (p *Plugin) wrapArray(
	fd *fieldData,
	field resolution.Field,
	table *resolution.Table,
	data *templateData,
) {
	isAnyOptional := field.Optional
	if isAnyOptional {
		addXImport(data, xImport{name: "zod", submodule: "zod"})
		fd.ZodType = fmt.Sprintf("zod.nullToUndefined(%s.array())", fd.ZodType)
		fd.ZodSchemaType = fmt.Sprintf(
			"ReturnType<typeof zod.nullToUndefined<z.ZodArray<%s>>>",
			fd.ZodSchemaType,
		)
	} else {
		// The server omits a nil ("not loaded") array, so a missing array defaults to
		// []; an allocated empty array still arrives as [].
		arrayZod := fd.ZodType + ".array()"
		fd.ZodSchemaType = fmt.Sprintf(
			"z.ZodDefault<z.ZodArray<%s>>",
			fd.ZodSchemaType,
		)
		if field.Default != nil &&
			field.Default.Kind == resolution.ValueKindArray &&
			len(field.Default.Elements) > 0 {
			fd.ZodType = fmt.Sprintf(
				"%s.default(%s)",
				arrayZod,
				p.tsDefaultLiteral(field.Type, *field.Default, table, data),
			)
		} else {
			fd.ZodType = arrayZod + ".default(() => [])"
		}
	}
}

// wrapMap sets the record schemas of a map field.
func (p *Plugin) wrapMap(
	fd *fieldData,
	field resolution.Field,
	table *resolution.Table,
	data *templateData,
) {
	isAnyOptional := field.Optional
	keyZ := p.typeRefToZod(&field.Type.TypeArgs[0], table, data)
	valueZ := p.typeRefToZod(&field.Type.TypeArgs[1], table, data)
	keySchemaType := p.typeRefToZodSchemaType(&field.Type.TypeArgs[0], table, data)
	valueSchemaType := p.typeRefToZodSchemaType(
		&field.Type.TypeArgs[1],
		table,
		data,
	)
	record, keySchemaType := recordFunc(
		&field.Type.TypeArgs[0],
		keySchemaType,
		table,
	)
	if isAnyOptional {
		addXImport(data, xImport{name: "zod", submodule: "zod"})
		fd.ZodType = fmt.Sprintf(
			"zod.nullToUndefined(z.%s(%s, %s))",
			record,
			keyZ,
			valueZ,
		)
		fd.ZodSchemaType = fmt.Sprintf(
			"ReturnType<typeof zod.nullToUndefined<z.ZodRecord<%s, %s>>>",
			keySchemaType,
			valueSchemaType,
		)
	} else {
		fd.ZodType = fmt.Sprintf(
			"z.%s(%s, %s).default(() => ({}))",
			record,
			keyZ,
			valueZ,
		)
		fd.ZodSchemaType = fmt.Sprintf(
			"z.ZodDefault<z.ZodRecord<%s, %s>>",
			keySchemaType,
			valueSchemaType,
		)
	}
}

// nullToUndefined wraps the schemas of fd so a null value parses as undefined.
func nullToUndefined(fd *fieldData, data *templateData) {
	addXImport(data, xImport{name: "zod", submodule: "zod"})
	fd.ZodType = fmt.Sprintf("zod.nullToUndefined(%s)", fd.ZodType)
	fd.ZodSchemaType = fmt.Sprintf(
		"ReturnType<typeof zod.nullToUndefined<%s>>",
		fd.ZodSchemaType,
	)
}

// wrapOptional makes the schemas of the optional field optional.
func wrapOptional(
	fd *fieldData,
	field resolution.Field,
	table *resolution.Table,
	data *templateData,
) {
	switch {
	case isUnionField(field, table):
		// Optional union-typed fields tolerate null: the Go side marshals a nil-variant
		// union as null.
		nullToUndefined(fd, data)
	case !field.Type.IsTypeParam():
		fd.ZodType += ".optional()"
	case field.Optional:
		// Optional (?) on a type-param field: the field is ALWAYS optional, even when a
		// caller passes a concrete schema. Wrap the whole "param ?? fallback"
		// expression in .optional() so caller schemas get wrapped too. Strip a trailing
		// .optional() from the fallback first so we don't emit redundant nesting.
		fd.ZodType = fmt.Sprintf(
			"(%s).optional()",
			strings.TrimSuffix(fd.ZodType, ".optional()"),
		)
	}
	fd.ZodSchemaType = fmt.Sprintf("z.ZodOptional<%s>", fd.ZodSchemaType)
}

func (p *Plugin) processField(
	field resolution.Field,
	parentType resolution.Type,
	table *resolution.Table,
	data *templateData,
	needsTypeImports bool,
) fieldData {
	isArray := field.Type.Name == "Array"
	needsGetter := isSelfReference(field.Type, parentType) ||
		isForwardReference(field.Type, data, table)

	fd := fieldData{
		Name:       field.Name,
		TSName:     FieldName(field.Name),
		Doc:        doc.Get(field.Domains),
		IsOptional: field.Optional,
		IsArray:    isArray,
		IsSelfRef:  needsGetter,
	}
	if typeOverride := getFieldTypeOverride(field, "ts"); typeOverride != "" {
		p.overrideSchema(
			&fd,
			field,
			parentType,
			typeOverride,
			table,
			data,
			needsTypeImports,
		)
	} else {
		p.fieldSchema(&fd, field, table, data, needsTypeImports)
	}
	// `@ts pick <field>` narrows a struct-typed field to a subset of the referenced
	// type's fields (e.g. a parent referenced by key alone), emitting a Zod .pick() and
	// a TS Pick<> rather than a standalone reference type.
	if pickField := domain.GetStringFromField(field, "ts", "pick"); pickField != "" {
		camel := FieldName(pickField)
		fd.ZodType = fmt.Sprintf("%s.pick({ %s: true })", fd.ZodType, camel)
		fd.TSType = fmt.Sprintf("Pick<%s, %q>", fd.TSType, camel)
	}
	p.wrapCollection(&fd, field, table, data)
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

// wrapCollection applies the array, map, or record handling that the type of field
// needs, or makes fd optional when field is optional.
func (p *Plugin) wrapCollection(
	fd *fieldData,
	field resolution.Field,
	table *resolution.Table,
	data *templateData,
) {
	typeOverride := getFieldTypeOverride(field, "ts")
	switch {
	case fd.IsArray:
		p.wrapArray(fd, field, table, data)
	case field.Type.Name == "Map" && len(field.Type.TypeArgs) >= 2:
		p.wrapMap(fd, field, table, data)
	case field.Type.Name == "record" || typeOverride == "record":
		wrapRecord(fd, field, data)
	default:
		if empty, ok := aliasedCollectionEmpty(field.Type, typeOverride, table); ok {
			wrapAliasedCollection(fd, field, empty, data)
		} else if field.Optional {
			wrapOptional(fd, field, table, data)
		}
	}
}

func wrapRecord(fd *fieldData, field resolution.Field, data *templateData) {
	if field.Optional {
		nullToUndefined(fd, data)
		return
	}
	addXImport(data, xImport{name: "record", submodule: "record"})
	fd.ZodType = "record.unknownZ().default(() => ({}))"
	fd.ZodSchemaType = "z.ZodDefault<ReturnType<typeof record.unknownZ>>"
}

// wrapAliasedCollection makes an alias of a literal collection parse as that collection
// does.
func wrapAliasedCollection(
	fd *fieldData,
	field resolution.Field,
	empty string,
	data *templateData,
) {
	if field.Optional {
		nullToUndefined(fd, data)
	} else if field.Default == nil && empty != "" {
		fd.ZodType = fmt.Sprintf("%s.default(() => %s)", fd.ZodType, empty)
		fd.ZodSchemaType = fmt.Sprintf("z.ZodDefault<%s>", fd.ZodSchemaType)
	}
}

// aliasedCollectionEmpty returns the empty value of the array, map, or record that ref
// is an alias of. empty is blank for an array, whose alias declaration sets its own
// default. ok is false when ref is not such an alias, or when a type override replaces
// its form.
func aliasedCollectionEmpty(
	ref resolution.TypeRef,
	typeOverride string,
	table *resolution.Table,
) (empty string, ok bool) {
	if typeOverride != "" {
		return "", false
	}
	_, target, ok := resolution.Dealias(ref, table)
	if !ok {
		return "", false
	}
	switch target.Name {
	case "Array":
		return "", true
	case "Map", "record":
		return "({})", true
	}
	return "", false
}

// isUnionField reports whether a field's type resolves to a discriminated union.
func isUnionField(field resolution.Field, table *resolution.Table) bool {
	resolved, _, ok := resolution.Dealias(field.Type, table)
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

func (p *Plugin) typeRefToZod(
	typeRef *resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	return p.typeRefToZodInternal(typeRef, table, data, false)
}

func (p *Plugin) typeRefToZodInternal(
	typeRef *resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
	forStructArg bool,
) string {
	if typeRef == nil {
		return "z.unknown()"
	}
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		return typeParamToZod(typeRef.TypeParam, table, data, forStructArg)
	}
	if resolution.IsPrimitive(typeRef.Name) {
		return primitiveToZod(typeRef.Name, data)
	}
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		elemZod := p.typeRefToZodInternal(&typeRef.TypeArgs[0], table, data, false)
		if typeRef.ArraySize != nil {
			return fmt.Sprintf("z.tuple([%s])", repeatJoin(elemZod, *typeRef.ArraySize))
		}
		return fmt.Sprintf("z.array(%s)", elemZod)
	}
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		keyZ := p.typeRefToZodInternal(&typeRef.TypeArgs[0], table, data, false)
		valueZ := p.typeRefToZodInternal(&typeRef.TypeArgs[1], table, data, false)
		record, _ := recordFunc(&typeRef.TypeArgs[0], "", table)
		return fmt.Sprintf("z.%s(%s, %s)", record, keyZ, valueZ)
	}
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return "z.unknown()"
	}
	schemaName := casing.CamelAcronym(domain.GetName(resolved, "ts")) + "Z"
	if alias, isAlias := resolved.Form.(resolution.AliasForm); !isAlias ||
		!alias.IsGeneric() {
		importLocalHand(resolved, schemaName, data)
	}
	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		if form.IsGeneric() {
			schemaName = p.genericSchemaCall(
				schemaName,
				form.TypeParams,
				typeRef.TypeArgs,
				table,
				data,
			)
		}
		return referenceName(resolved, schemaName, table, data)
	case resolution.EnumForm, resolution.DistinctForm, resolution.UnionForm:
		return referenceName(resolved, schemaName, table, data)
	case resolution.AliasForm:
		if !form.IsGeneric() {
			return referenceName(resolved, schemaName, table, data)
		}
		target := aliasTarget(form, typeRef.TypeArgs)
		return p.typeRefToZodInternal(&target, table, data, forStructArg)
	default:
		return "z.unknown()"
	}
}

// aliasTarget returns the target of a generic alias with args substituted for its type
// parameters.
func aliasTarget(
	form resolution.AliasForm,
	args []resolution.TypeRef,
) resolution.TypeRef {
	if len(args) == 0 {
		return form.Target
	}
	return resolution.SubstituteTypeRef(
		form.Target,
		resolution.TypeArgMap(form.TypeParams, args),
	)
}

// typeParamToZod returns the Zod schema for a reference to a type parameter. Outside a
// struct argument, an optional, defaulted, or numeric parameter falls back to a schema
// when the caller omits it.
func typeParamToZod(
	param *resolution.TypeParam,
	table *resolution.Table,
	data *templateData,
	forStructArg bool,
) string {
	paramName := casing.CamelAcronym(param.Name)
	if forStructArg {
		return paramName
	}
	if param.Constraint != nil && param.Constraint.Name == "numeric" {
		return fmt.Sprintf("%s ?? %s", paramName, defaultValueToTS(param.Default.Name))
	}
	if param.Default == nil && !param.Optional {
		return paramName
	}
	effectiveRef := param.Constraint
	if effectiveRef == nil && param.Default != nil && !param.Optional {
		effectiveRef = param.Default
	}
	if effectiveRef != nil && effectiveRef.Name == "record" {
		addXImport(data, xImport{name: "record", submodule: "record"})
	}
	return fmt.Sprintf(
		"%s ?? %s",
		paramName,
		fallbackForConstraint(effectiveRef, table, data),
	)
}

// genericSchemaCall returns the call of the generic schema function schemaName with
// args. nil arguments are left out. A schema with one type parameter takes its argument
// bare, and one with more takes an object of named arguments.
func (p *Plugin) genericSchemaCall(
	schemaName string,
	params []resolution.TypeParam,
	args []resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	type indexedArg struct {
		index int
		value string
	}
	nonNilArgs := make([]indexedArg, 0, len(args))
	for i, arg := range args {
		if arg.Name == "nil" {
			continue
		}
		nonNilArgs = append(
			nonNilArgs,
			indexedArg{i, p.typeRefToZodInternal(&arg, table, data, true)},
		)
	}
	if len(nonNilArgs) == 0 {
		return schemaName + "()"
	}
	if len(params) == 1 {
		return fmt.Sprintf("%s(%s)", schemaName, nonNilArgs[0].value)
	}
	namedArgs := make([]string, len(nonNilArgs))
	for i, arg := range nonNilArgs {
		namedArgs[i] = fmt.Sprintf(
			"%s: %s",
			casing.CamelAcronym(params[arg.index].Name),
			arg.value,
		)
	}
	return fmt.Sprintf("%s({%s})", schemaName, strings.Join(namedArgs, ", "))
}

// referenceName returns name as the file being generated refers to it: bare when typ is
// declared in that file, and under the namespace that imports it otherwise.
func referenceName(
	typ resolution.Type,
	name string,
	table *resolution.Table,
	data *templateData,
) string {
	if typ.FilePath == data.FilePath {
		return name
	}
	return qualifier(typ, table, data) + "." + name
}

// repeatJoin returns n copies of elem, joined with commas.
func repeatJoin(elem string, n int64) string {
	elements := make([]string, n)
	for i := range elements {
		elements[i] = elem
	}
	return strings.Join(elements, ", ")
}

func (p *Plugin) typeRefToTS(
	typeRef *resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
	needsTypeImports bool,
) string {
	return p.typeRefToTSInternal(typeRef, table, data, false, needsTypeImports)
}

func (p *Plugin) typeRefToTSInternal(
	typeRef *resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
	forStructArg, needsTypeImports bool,
) string {
	if typeRef == nil {
		return "unknown"
	}
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		if forStructArg ||
			(typeRef.TypeParam.Constraint != nil &&
				typeRef.TypeParam.Constraint.Name == "numeric") {
			return typeRef.TypeParam.Name
		}
		return fmt.Sprintf("z.infer<%s>", typeRef.TypeParam.Name)
	}
	if resolution.IsPrimitive(typeRef.Name) {
		if needsTypeImports {
			importTelemType(typeRef.Name, data)
		}
		return primitiveToTS(typeRef.Name)
	}
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		elemTS := p.typeRefToTSInternal(
			&typeRef.TypeArgs[0],
			table,
			data,
			forStructArg,
			needsTypeImports,
		)
		if typeRef.ArraySize != nil {
			return fmt.Sprintf("[%s]", repeatJoin(elemTS, *typeRef.ArraySize))
		}
		return elemTS + "[]"
	}
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		return p.mapToTS(typeRef, table, data, forStructArg, needsTypeImports)
	}
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return "unknown"
	}
	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		return p.typeReference(
			resolved, form.TypeParams, typeRef.TypeArgs, table, data, needsTypeImports,
		)
	case resolution.AliasForm:
		return p.typeReference(
			resolved, form.TypeParams, typeRef.TypeArgs, table, data, needsTypeImports,
		)
	case resolution.EnumForm, resolution.DistinctForm, resolution.UnionForm:
		return referenceName(resolved, domain.GetName(resolved, "ts"), table, data)
	default:
		return "unknown"
	}
}

// importTelemType imports the telem class that the TypeScript type of primitive names.
func importTelemType(primitive string, data *templateData) {
	switch primitive {
	case "timestamp":
		addXImport(data, xImport{name: "TimeStamp", submodule: "telem"})
	case "timespan":
		addXImport(data, xImport{name: "TimeSpan", submodule: "telem"})
	}
}

func (p *Plugin) mapToTS(
	typeRef *resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
	forStructArg, needsTypeImports bool,
) string {
	keyType := p.typeRefToTSInternal(
		&typeRef.TypeArgs[0],
		table,
		data,
		forStructArg,
		needsTypeImports,
	)
	valueType := p.typeRefToTSInternal(
		&typeRef.TypeArgs[1],
		table,
		data,
		forStructArg,
		needsTypeImports,
	)
	if record, _ := recordFunc(&typeRef.TypeArgs[0], "", table); record != "record" {
		return fmt.Sprintf("Partial<Record<%s, %s>>", keyType, valueType)
	}
	return fmt.Sprintf("Record<%s, %s>", keyType, valueType)
}

// typeReference returns the TypeScript type that refers to typ with args. A generic typ
// takes its args as type arguments.
func (p *Plugin) typeReference(
	typ resolution.Type,
	params []resolution.TypeParam,
	args []resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
	needsTypeImports bool,
) string {
	typeName := domain.GetName(typ, "ts")
	if len(params) > 0 && len(args) > 0 {
		typeName = fmt.Sprintf(
			"%s<%s>",
			typeName,
			p.genericArgs(params, args, table, data, needsTypeImports),
		)
	}
	return referenceName(typ, typeName, table, data)
}

// genericArgs renders the type arguments of a generic struct or alias. Its type
// parameters take Zod types, except numeric ones, which take TypeScript types.
func (p *Plugin) genericArgs(
	params []resolution.TypeParam,
	args []resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
	needsTypeImports bool,
) string {
	out := make([]string, len(args))
	for i := range args {
		if i < len(params) && params[i].Constraint != nil &&
			params[i].Constraint.Name == "numeric" {
			out[i] = p.typeRefToTSInternal(
				&args[i],
				table,
				data,
				true,
				needsTypeImports,
			)
			continue
		}
		out[i] = p.typeRefToZodSchemaType(&args[i], table, data)
	}
	return strings.Join(out, ", ")
}

var primitiveTSTypes = map[string]string{
	"string":    "string",
	"uuid":      "string",
	"bool":      "boolean",
	"int8":      "number",
	"int16":     "number",
	"int32":     "number",
	"int64":     "number",
	"uint8":     "number",
	"uint12":    "number",
	"uint16":    "number",
	"uint20":    "number",
	"uint32":    "number",
	"uint64":    "number",
	"float32":   "number",
	"float64":   "number",
	"number":    "number",
	"timestamp": "TimeStamp",
	"timespan":  "TimeSpan",
	"data_type": "DataType",
	"record":    "unknown",
	"bytes":     "Uint8Array",
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
	"uuid":   {schema: "z.uuid()"},
	"string": {schema: "z.string()"},
	"bool":   {schema: "z.boolean()"},
	"int8": {
		schema:   "zod.int8",
		xImports: []xImport{{name: "zod", submodule: "zod"}},
	},
	"int16": {
		schema:   "zod.int16",
		xImports: []xImport{{name: "zod", submodule: "zod"}},
	},
	"int32": {schema: "z.int32()"},
	"int64": {schema: "z.int64()"},
	"uint8": {
		schema:   "zod.uint8",
		xImports: []xImport{{name: "zod", submodule: "zod"}},
	},
	"uint12": {
		schema:   "zod.uint12",
		xImports: []xImport{{name: "zod", submodule: "zod"}},
	},
	"uint16": {
		schema:   "zod.uint16",
		xImports: []xImport{{name: "zod", submodule: "zod"}},
	},
	"uint20": {
		schema:   "zod.uint20",
		xImports: []xImport{{name: "zod", submodule: "zod"}},
	},
	"uint32":  {schema: "z.uint32()"},
	"uint64":  {schema: "z.uint64()"},
	"float32": {schema: "z.number()"},
	"float64": {schema: "z.number()"},
	"number":  {schema: "z.number()"},
	"timestamp": {
		schema:   "TimeStamp.z",
		xImports: []xImport{{name: "TimeStamp", submodule: "telem"}},
	},
	"timespan": {
		schema:   "TimeSpan.z",
		xImports: []xImport{{name: "TimeSpan", submodule: "telem"}},
	},
	"time_range": {
		schema:   "TimeRange.z",
		xImports: []xImport{{name: "TimeRange", submodule: "telem"}},
	},
	"time_range_bounded": {
		schema:   "TimeRange.boundedZ",
		xImports: []xImport{{name: "TimeRange", submodule: "telem"}},
	},
	"data_type": {
		schema:   "DataType.z",
		xImports: []xImport{{name: "DataType", submodule: "telem"}},
	},
	"record": {
		schema:   "record.unknownZ().or(z.string().transform((s) => JSON.parse(s)))",
		xImports: []xImport{{name: "record", submodule: "record"}},
	},
	"bytes": {schema: "z.instanceof(Uint8Array)"},
}

const (
	xPackageName = "@synnaxlabs/x"
	xPathPrefix  = "x/ts/src"
)

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

func (p *Plugin) typeRefToZodSchemaType(
	typeRef *resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
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
			return fmt.Sprintf(
				"z.ZodTuple<[%s]>",
				repeatJoin(elemZodType, *typeRef.ArraySize),
			)
		}
		return fmt.Sprintf("z.ZodArray<%s>", elemZodType)
	}
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		keyType := p.typeRefToZodSchemaType(&typeRef.TypeArgs[0], table, data)
		valueType := p.typeRefToZodSchemaType(&typeRef.TypeArgs[1], table, data)
		_, keyType = recordFunc(&typeRef.TypeArgs[0], keyType, table)
		return fmt.Sprintf("z.ZodRecord<%s, %s>", keyType, valueType)
	}
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		resolved, ok = table.Lookup(data.Namespace, typeRef.Name)
	}
	if !ok {
		return "z.ZodType"
	}

	var prefix string
	if resolved.FilePath != data.FilePath {
		prefix = qualifier(resolved, table, data) + "."
	}

	tsName := domain.GetName(resolved, "ts")
	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		if form.IsGeneric() && len(typeRef.TypeArgs) > 0 {
			return p.genericStructSchemaType(
				resolved, prefix, tsName, typeRef.TypeArgs, table, data,
			)
		}
		return fmt.Sprintf("typeof %s%sZ", prefix, casing.CamelAcronym(tsName))

	case resolution.EnumForm, resolution.DistinctForm, resolution.AliasForm:
		return fmt.Sprintf("typeof %s%sZ", prefix, casing.CamelAcronym(tsName))

	case resolution.UnionForm:
		// A union schema's inferred type depends on every variant schema, so a
		// `typeof xZ` annotation in a getter would re-enter the inference cycle the
		// getter exists to break. z.ZodType<Name> references only the declared
		// union type alias, which TypeScript resolves lazily.
		return fmt.Sprintf("z.ZodType<%s%s>", prefix, tsName)
	}

	return "z.ZodType"
}

// genericStructSchemaType returns the Zod schema type of the generic struct typ with
// args. prefix is the namespace that the file refers to typ under, if any.
func (p *Plugin) genericStructSchemaType(
	typ resolution.Type,
	prefix, tsName string,
	args []resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	argTypes := make([]string, len(args))
	for i, arg := range args {
		argTypes[i] = p.typeRefToZodSchemaType(&arg, table, data)
	}
	joined := strings.Join(argTypes, ", ")
	if _, ok := tsExpressions(typ).Find("concrete_types"); ok {
		return fmt.Sprintf("%s%sZodObject<%s>", prefix, tsName, joined)
	}
	return fmt.Sprintf(
		"ReturnType<typeof %s%sZ<%s>>",
		prefix,
		casing.CamelAcronym(tsName),
		joined,
	)
}

func isInXPackage(outputPath string) bool {
	return strings.HasPrefix(outputPath, xPathPrefix)
}

// importLocalHand imports ident from typ's `@ts include` module when typ is
// hand-written or omitted in the namespace being generated, since the generated file
// declares no such type. A missing module is recorded so generation fails.
func importLocalHand(typ resolution.Type, ident string, data *templateData) {
	if typ.FilePath != data.FilePath || !omit.IsSkipped(typ, "ts") {
		return
	}
	module := domain.GetStringFromType(typ, "ts", "include")
	if module == "" {
		data.missingIncludes.Add(typ.QualifiedName)
		return
	}
	data.AddImport(paths.CalculateImport(data.OutputPath, module), ident)
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
	// .default(). Struct defaults use .prefault() so the literal is re-parsed, filling
	// each field's own default when a caller supplies a partial value.
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

// handTypeRules returns the rules of the first hand-written type in ref's chain and of
// each type it reaches. A generated schema applies its type's rules itself, but a
// hand-written one does not.
func handTypeRules(ref resolution.TypeRef, table *resolution.Table) *validation.Rules {
	chain := validation.TypeChain(ref, table)
	i := slices.IndexFunc(chain, func(t resolution.Type) bool {
		return omit.IsSkipped(t, "ts")
	})
	if i < 0 {
		return &validation.Rules{}
	}
	return validation.TypeRules(chain[i:])
}

func (p *Plugin) applyValidation(
	zodType string,
	rules *validation.Rules,
	defaultVal *resolution.ExpressionValue,
	typeRef resolution.TypeRef,
	fieldName string,
	table *resolution.Table,
	data *templateData,
	tsTypeOverride string,
) validationResult {
	if validation.IsEmpty(rules) && defaultVal == nil {
		return validationResult{ZodType: zodType, HasDefault: false}
	}
	effectiveType := validationType(typeRef, table, tsTypeOverride)
	isString := resolution.IsPrimitive(effectiveType) &&
		resolution.IsStringPrimitive(effectiveType)
	isNumber := resolution.IsPrimitive(effectiveType) &&
		resolution.IsNumberPrimitive(effectiveType)
	if isString {
		zodType = applyStringRules(zodType, rules, fieldName, data)
	}
	if isNumber {
		zodType = applyNumberRules(zodType, rules, fieldName)
	}
	// A hand-written number schema such as telem.numericTimeSpanZ, used directly or
	// through a @ts type override, has no .min or .max, so its bounds go through
	// checks.
	if !isNumber && tsTypeOverride != "number" &&
		!resolution.IsPrimitive(effectiveType) &&
		resolution.IsNumberPrimitive(resolution.PrimitiveBase(typeRef, table)) {
		zodType = applyBoundChecks(zodType, rules)
	}
	isPrefault := false
	if defaultVal != nil {
		zodType, isPrefault = p.applyDefault(
			zodType,
			*defaultVal,
			typeRef,
			table,
			data,
			tsTypeOverride,
			isString,
		)
	}
	return validationResult{
		ZodType:    zodType,
		HasDefault: defaultVal != nil,
		IsPrefault: isPrefault,
	}
}

// validationType returns the type name that classifies typeRef's validation rules and
// default literal.
func validationType(
	typeRef resolution.TypeRef,
	table *resolution.Table,
	tsTypeOverride string,
) string {
	effectiveType := typeRef.Name
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil &&
		typeRef.TypeParam.Constraint != nil {
		effectiveType = typeRef.TypeParam.Constraint.Name
	}
	// A @ts type override (e.g. a telem.Rate field surfaced as a plain "number" in
	// TypeScript) drives both the validation rules and the default literal off the
	// overridden primitive, not the underlying schema type.
	switch {
	case tsTypeOverride == "number":
		effectiveType = "float64"
	case tsTypeOverride != "":
		effectiveType = tsTypeOverride
	case !resolution.IsPrimitive(effectiveType):
		// A named numeric or string type (e.g. a distinct Key over uint32) carries its
		// constraints on the underlying primitive, so classify by that base. A
		// hand-written schema may not be a zod number, so it keeps its own name.
		resolved, ok := typeRef.Resolve(table)
		handWritten := ok && omit.IsSkipped(resolved, "ts")
		base := resolution.PrimitiveBase(typeRef, table)
		if base != "" && !handWritten {
			effectiveType = base
		}
	}
	return effectiveType
}

func applyStringRules(
	zodType string,
	rules *validation.Rules,
	fieldName string,
	data *templateData,
) string {
	if rules.Required {
		zodType = fmt.Sprintf("%s.min(1, \"%s is required\")", zodType, fieldName)
	}
	// Zod's .min and .max count UTF-16 code units, but the other languages count code
	// points. The two agree on whether a string is empty.
	if rules.MinLength != nil {
		if *rules.MinLength > 1 {
			addXImport(data, xImport{name: "zod", submodule: "zod"})
			zodType = fmt.Sprintf(
				"%s.check(zod.minLength(%d))",
				zodType,
				*rules.MinLength,
			)
		} else {
			zodType = fmt.Sprintf("%s.min(%d)", zodType, *rules.MinLength)
		}
	}
	if rules.MaxLength != nil {
		addXImport(data, xImport{name: "zod", submodule: "zod"})
		zodType = fmt.Sprintf(
			"%s.check(zod.maxLength(%d))",
			zodType,
			*rules.MaxLength,
		)
	}
	if rules.Pattern != nil {
		if rules.PatternMessage != nil {
			zodType = fmt.Sprintf(
				"%s.regex(/%s/, %s)",
				zodType,
				*rules.Pattern,
				tsStringLiteral(*rules.PatternMessage),
			)
		} else {
			zodType = fmt.Sprintf("%s.regex(/%s/)", zodType, *rules.Pattern)
		}
	}
	return zodType
}

func applyNumberRules(
	zodType string,
	rules *validation.Rules,
	fieldName string,
) string {
	if rules.Required {
		zodType = fmt.Sprintf(
			"%s.refine((v) => v !== 0, \"%s is required\")",
			zodType,
			fieldName,
		)
	}
	if rules.Min != nil {
		zodType = numberBoundCall(zodType, "min", *rules.Min)
	}
	if rules.GT != nil {
		zodType = numberBoundCall(zodType, "gt", *rules.GT)
	}
	if rules.Max != nil {
		zodType = numberBoundCall(zodType, "max", *rules.Max)
	}
	return zodType
}

// numberBoundCall appends a call to the zod number method with the bound n.
func numberBoundCall(zodType, method string, n validation.Number) string {
	if n.IsInt {
		return fmt.Sprintf("%s.%s(%d)", zodType, method, n.Int)
	}
	return fmt.Sprintf("%s.%s(%f)", zodType, method, n.Float)
}

// applyBoundChecks appends the numeric bounds of rules as zod checks.
func applyBoundChecks(zodType string, rules *validation.Rules) string {
	var checks []string
	if rules.Min != nil {
		checks = append(checks, "z.gte("+boundLiteral(*rules.Min)+")")
	}
	if rules.GT != nil {
		checks = append(checks, "z.gt("+boundLiteral(*rules.GT)+")")
	}
	if rules.Max != nil {
		checks = append(checks, "z.lte("+boundLiteral(*rules.Max)+")")
	}
	if len(checks) == 0 {
		return zodType
	}
	return fmt.Sprintf("%s.check(%s)", zodType, strings.Join(checks, ", "))
}

// applyDefault appends the default of defaultVal to zodType. It reports whether the
// default is a .prefault.
func (p *Plugin) applyDefault(
	zodType string,
	defaultVal resolution.ExpressionValue,
	typeRef resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
	tsTypeOverride string,
	isString bool,
) (string, bool) {
	switch defaultVal.Kind {
	case resolution.ValueKindString:
		return fmt.Sprintf(
			"%s.default(%s)",
			zodType,
			tsStringLiteral(defaultVal.StringValue),
		), false
	case resolution.ValueKindInt:
		return intDefault(
				zodType,
				defaultVal.IntValue,
				typeRef,
				table,
				data,
				tsTypeOverride,
			),
			false
	case resolution.ValueKindFloat:
		return floatDefault(
				zodType,
				defaultVal.FloatValue,
				typeRef,
				table,
				data,
				tsTypeOverride,
			),
			false
	case resolution.ValueKindBool:
		return fmt.Sprintf("%s.default(%t)", zodType, defaultVal.BoolValue), false
	case resolution.ValueKindIdent:
		return p.identDefault(
			zodType,
			defaultVal.IdentValue,
			typeRef,
			table,
			data,
			isString,
		)
	case resolution.ValueKindArray:
		return fmt.Sprintf(
			"%s.default(%s)",
			zodType,
			p.tsDefaultLiteral(typeRef, defaultVal, table, data),
		), false
	case resolution.ValueKindStruct:
		// .prefault re-parses the literal, so a partial value a caller supplies is
		// merged with each field's own default rather than rejected.
		return fmt.Sprintf(
			"%s.prefault(%s)",
			zodType,
			p.tsDefaultLiteral(typeRef, defaultVal, table, data),
		), true
	}
	return zodType, false
}

func intDefault(
	zodType string,
	value int64,
	typeRef resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
	tsTypeOverride string,
) string {
	stringTyped := tsTypeOverride == "string"
	if !stringTyped {
		if resolved, ok := typeRef.Resolve(table); ok &&
			getTypeTypeOverride(resolved, "ts") == "string" {
			stringTyped = true
		}
	}
	if stringTyped {
		// A uint64 surfaced as a string (e.g. Key uint64 { @ts type string }) accepts
		// string input, so an integer default must be emitted as a string literal.
		return fmt.Sprintf(
			"%s.default(%s)",
			zodType,
			tsStringLiteral(strconv.FormatInt(value, 10)),
		)
	}
	expr, ok := "", false
	if tsTypeOverride == "" {
		expr, ok = tsTelemNumericDefault(
			typeRef,
			table,
			data,
			strconv.FormatInt(value, 10),
			value == 0,
		)
	}
	if ok {
		return fmt.Sprintf("%s.default(%s)", zodType, expr)
	}
	return fmt.Sprintf("%s.default(%d)", zodType, value)
}

func floatDefault(
	zodType string,
	value float64,
	typeRef resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
	tsTypeOverride string,
) string {
	expr, ok := "", false
	if tsTypeOverride == "" {
		expr, ok = tsTelemNumericDefault(
			typeRef,
			table,
			data,
			fmt.Sprintf("%g", value),
			value == 0,
		)
	}
	if ok {
		return fmt.Sprintf("%s.default(%s)", zodType, expr)
	}
	return fmt.Sprintf("%s.default(%f)", zodType, value)
}

// identDefault appends an identifier default such as now, create, an enum member, or a
// union variant. It reports whether the default is a .prefault.
func (p *Plugin) identDefault(
	zodType string,
	ident string,
	typeRef resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
	isString bool,
) (string, bool) {
	isPrefault := false
	if ident == "now" && telem.IsTimeStamp(typeRef, table, "ts") {
		addXImport(data, xImport{name: "TimeStamp", submodule: "telem"})
		zodType += ".default(() => TimeStamp.now())"
	}
	// Handle "create" for auto-generating keys. uuid keys generate a UUID via
	// uuid.create(); string keys generate a short id via id.create(). uuid is a string
	// primitive, so check it first. Use key.ResolvePrimitive to handle type aliases
	// like `Key distinct string`.
	primitive := key.ResolvePrimitive(typeRef, table)
	if ident == "create" {
		if primitive == "uuid" {
			addXImport(data, xImport{name: "uuid", submodule: "uuid"})
			zodType += ".default(uuid.create)"
		} else if isString || primitive == "string" {
			addXImport(data, xImport{name: "id", submodule: "id"})
			zodType += ".default(id.create)"
		}
	}
	if ev, ok := validation.ResolveEnumVariant(ident, typeRef, table); ok {
		zodType = fmt.Sprintf("%s.default(%s)", zodType, p.enumVariantToTS(ev, data))
	}
	if uv, ok := validation.ResolveUnionVariant(ident, typeRef, table); ok {
		// .prefault re-parses the discriminator literal, so the variant's own field
		// defaults fill in rather than being demanded of the caller.
		zodType = fmt.Sprintf(
			"%s.prefault({ %s: %q })",
			zodType,
			FieldName(uv.Union.Discriminator),
			uv.Variant.Name,
		)
		isPrefault = true
	}
	return zodType, isPrefault
}

// tsTelemNumericDefault returns the .default(...) argument for a numeric default
// on a telem class field, registering the import, or ("", false) when typeRef is
// not such a type. The telem zod schemas output class instances, so a bare number
// would not type-check as the default; the literal is wrapped in the constructor
// ("new Rate(10)"). A zero TimeStamp or TimeSpan keeps the canonical .ZERO
// constant.
func tsTelemNumericDefault(
	typeRef resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
	literal string,
	isZero bool,
) (string, bool) {
	var name string
	switch {
	case telem.IsTimeStamp(typeRef, table, "ts"):
		name = "TimeStamp"
	case telem.IsTimeSpan(typeRef, table, "ts"):
		name = "TimeSpan"
	case telem.IsRate(typeRef, table, "ts"):
		name = "Rate"
	default:
		return "", false
	}
	addXImport(data, xImport{name: name, submodule: "telem"})
	if isZero && name != "Rate" {
		return name + ".ZERO", true
	}
	return fmt.Sprintf("new %s(%s)", name, literal), true
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

// tsDefaultLiteral renders a default value as a TypeScript literal. typeRef is the
// declared type of the value, used to resolve enum variants, array element types, and
// nested struct field types. Arrays and structs recurse.
func (p *Plugin) tsDefaultLiteral(
	typeRef resolution.TypeRef,
	val resolution.ExpressionValue,
	table *resolution.Table,
	data *templateData,
) string {
	switch val.Kind {
	case resolution.ValueKindString:
		return tsStringLiteral(val.StringValue)
	case resolution.ValueKindInt:
		return strconv.FormatInt(val.IntValue, 10)
	case resolution.ValueKindFloat:
		return fmt.Sprintf("%f", val.FloatValue)
	case resolution.ValueKindBool:
		return strconv.FormatBool(val.BoolValue)
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

// tsStructLiteral renders a struct default as a TypeScript object literal, resolving
// each field's value against its declared type in the struct named by typeRef.
func (p *Plugin) tsStructLiteral(
	typeRef resolution.TypeRef,
	val resolution.ExpressionValue,
	table *resolution.Table,
	data *templateData,
) string {
	fieldsByName := structFieldsByName(typeRef, table)
	parts := make([]string, 0, len(val.Fields))
	for _, fv := range val.Fields {
		parts = append(
			parts,
			fmt.Sprintf(
				"%s: %s",
				FieldName(fv.Name),
				p.tsDefaultLiteral(fieldsByName[fv.Name].Type, fv.Value, table, data),
			),
		)
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
func structFieldsByName(
	typeRef resolution.TypeRef,
	table *resolution.Table,
) map[string]resolution.Field {
	out := map[string]resolution.Field{}
	resolved, _, ok := resolution.Dealias(typeRef, table)
	if !ok {
		return out
	}
	for _, f := range resolution.UnifiedFields(resolved, table) {
		out[f.Name] = f
	}
	return out
}

func (p *Plugin) enumVariantToTS(ev validation.EnumVariant, data *templateData) string {
	// String-valued enums are emitted as `z.enum([...])` plus a type alias and have no
	// runtime object to dot into. Emit the raw string literal instead; only numeric
	// enums get the `Type.variant` form, since those emit as TS runtime enums.
	if form, ok := ev.Type.Form.(resolution.EnumForm); ok && !form.IsIntEnum {
		return tsStringLiteral(ev.Variant.StringValue())
	}
	enumName := domain.GetName(ev.Type, "ts")
	variantRef := fmt.Sprintf("%s.%s", enumName, ev.Variant.Name)
	if ev.Type.FilePath != data.FilePath {
		ns := qualifier(ev.Type, data.Request.Resolutions, data)
		variantRef = ns + "." + variantRef
	}
	return variantRef
}

type templateData struct {
	*imports.Manager
	Request          *plugin.Request
	Resource         *resourceData
	DeclOrder        map[string]int
	Namespace        string
	OutputPath       string
	Enums            []enumData
	TypeDefs         []typeDefData
	SortedDecls      []sortedDeclData
	CurrentDeclIndex int
	GenerateTypes    bool
	// FilePath is the schema file that this file renders. A type is local when it comes
	// from this file: two schemas with one file name share a namespace.
	FilePath string
	// missingIncludes holds the hand-written types referenced in this namespace that
	// declare no `@ts include` module.
	missingIncludes set.Set[string]
	// aliases holds the alias of each imported name that two modules export.
	aliases map[imports.Key]string
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

type resourceData struct {
	TypeName, KeyType, KeyZeroValue string
}

func primitiveZeroValue(primitive string) string {
	switch primitive {
	case "uuid", "string":
		return `""`
	case "bool":
		return "false"
	case "int8",
		"int16",
		"int32",
		"int64",
		"uint8",
		"uint16",
		"uint32",
		"uint64",
		"float32",
		"float64":
		return "0"
	default:
		return `""`
	}
}

type structData struct {
	ExtendsName       string
	TSName            string
	AliasOf           string
	Doc               string
	ExtendsTypeName   string
	Name              string
	TypeParams        []typeParamData
	BaseFields        []fieldData
	ConditionalFields []conditionalFieldData
	ExtendFields      []fieldData
	PartialFields     []fieldData
	OmittedFields     []string
	// Request is true for a request derived from a filter. A query cache keys on it, so
	// it is a type alias, since an interface has no index signature.
	Request bool
	// ConditionalOmittedFields names the conditional (type-param-typed) base fields a
	// synthesized @create New strips from its inner optional.Optional base before
	// re-appending them as z.input conditionals. The New's template wraps the base in
	// Omit<..., these> so the field's z.infer projection from the base type does not
	// collide with the re-appended z.input conditional.
	ConditionalOmittedFields []string
	Fields                   []fieldData
	ExtendsParents           []extendsParentInfo
	HasExtends               bool
	UseInput                 bool
	AllParamsOptional        bool
	Handwritten              bool
	IsRecursive              bool
	// Strict is true for a filter, its request, and its operator types. Each rejects
	// unknown keys, so a stale field fails instead of selecting every record. Each
	// leaves its defaults to the Core, so the cache sees only the fields a caller set.
	Strict             bool
	IsAlias            bool
	IsSingleParam      bool
	IsGeneric          bool
	ConcreteTypes      bool
	CoalesceTypeParams bool
	// IsPrimitiveConstrainedGeneric is true when every type param is constrained to a
	// primitive set (e.g. T extends numeric) and has a default. In this mode the
	// runtime zod schema is emitted as a plain z.object (defaults substituted for
	// type-param-typed fields), but the TS interface keeps the generic shape with
	// primitive constraints (e.g. T extends number | bigint = number).
	IsPrimitiveConstrainedGeneric bool
	// TypeOnly is set for @create-synthesized New types. The runtime zod schema const
	// is suppressed (the base schema already accepts the same input via its own
	// .default()s), and the New type references the base schema directly rather than
	// the now-absent newZ.
	TypeOnly bool
	// BaseIsUnion is set for a @create-synthesized New whose base is a discriminated
	// union. Its input projection cannot be an `interface … extends z.input<…>` (a
	// union is not extendable), so it emits as a `type … = z.input<typeof unionZ>`
	// alias instead.
	BaseIsUnion bool
}

type extendsParentInfo struct {
	Name     string
	TypeName string
	// Schema instantiates the parent's schema with the type arguments the child passes,
	// e.g. `boxZ(z.string())`.
	Schema string
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
	Name, TSName, ZodType, TSType, ZodSchemaType string
	CoalescedTSType                              string
	Doc                                          string
	IsOptional, IsArray, IsSelfRef               bool
}

type conditionalFieldData struct {
	TypeParamName      string
	NeverType          string
	FallbackSchemaType string
	Field              fieldData
	// UseInput renders the conditional field's value type as z.input<Param>
	// rather than z.infer<Param>. It is set for a synthesized @create New, whose
	// conditional field accepts the schema's input (pre-default) shape.
	UseInput bool
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
	"camelCase": casing.CamelAcronym,
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

var fileTemplate = template.Must(
	template.New("zod").
		Funcs(templateFuncs).
		Parse(`// Code generated by Oracle. DO NOT EDIT.
{{range .SynnaxImports }}
import {{ if .Namespace }}* as {{ .Namespace }}{{ else }}{ {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} }{{ end }} from "{{ .Path }}";
{{- end }}
import { z } from "zod";
{{- range .ExternalNamedImports }}
import {{ if .Namespace }}* as {{ .Namespace }}{{ else }}{ {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} }{{ end }} from "{{ .Path }}";
{{- end }}
{{ if .InternalNamedImports }}
{{- range .InternalNamedImports }}
import {{ if .Namespace }}* as {{ .Namespace }}{{ else }}{ {{ range $i, $name := .Names }}{{ if $i }}, {{ end }}{{ $name }}{{ end }} }{{ end }} from "{{ .Path }}";
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
  z.{{ if .Strict }}strictObject{{ else }}object{{ end }}({
{{- range .Fields }}
{{- if .Doc }}
  {{ formatDoc .TSName .Doc 2 }}
{{- end }}
    {{ .TSName }}: {{ .ZodType }},
{{- end }}
  });
{{- if $.GenerateTypes }}
export interface {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.BareConstraint }}{{ if $p.HasDefault }} = {{ $p.BareDefault }}{{ end }}{{ end }}> {
{{- range .Fields }}
  {{ .TSName }}{{ if .IsOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}
{{- end }}
{{- else if .IsGeneric }}
{{- if not .TypeOnly }}
{{- if .IsSingleParam }}
{{- if and .ConcreteTypes .ConditionalFields }}

export type {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = z.ZodObject<{
{{- range .BaseFields }}
    {{ .TSName }}: {{ .ZodSchemaType }};
{{- end }}
{{- range .ConditionalFields }}
    {{ .Field.TSName }}: [{{ .TypeParamName }}] extends [{{ .NeverType }}] ? {{ .FallbackSchemaType }} : {{ if .Field.IsOptional }}z.ZodOptional<{{ .TypeParamName }}>{{ else }}{{ .TypeParamName }}{{ end }};
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
  {{ range $i, $p := .ExtendsParents }}{{ if $i }}.extend({{ end }}{{ $p.Schema }}{{ if $i }}.shape){{ end }}{{ end }}
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
  z.{{ if .Strict }}strictObject{{ else }}object{{ end }}({
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
    {{ .Field.TSName }}: [{{ .TypeParamName }}] extends [{{ .NeverType }}] ? {{ .FallbackSchemaType }} : {{ if .Field.IsOptional }}z.ZodOptional<{{ .TypeParamName }}>{{ else }}{{ .TypeParamName }}{{ end }};
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
  {{ range $i, $p := .ExtendsParents }}{{ if $i }}.extend({{ end }}{{ $p.Schema }}{{ if $i }}.shape){{ end }}{{ end }}
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
  z.{{ if .Strict }}strictObject{{ else }}object{{ end }}({
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
{{- end }}
{{- if $.GenerateTypes }}
{{- if .ConcreteTypes }}
{{- if .HasExtends }}
{{- if .CoalesceTypeParams }}
export type {{ .TSName }}<S extends {{ if .TypeOnly }}{{ .ExtendsTypeName }}{{ else }}{{ .TSName }}{{ end }}Schemas = {{ if .TypeOnly }}{{ .ExtendsTypeName }}{{ else }}{{ .TSName }}{{ end }}Schemas> = {{ if .ConditionalOmittedFields }}Omit<{{ end }}{{ if .PartialFields }}optional.Optional<{{ end }}{{ if .OmittedFields }}Omit<{{ end }}{{ .ExtendsTypeName }}<S>{{ if .OmittedFields }}, {{ range $i, $f := .OmittedFields }}{{ if $i }} | {{ end }}"{{ $f }}"{{ end }}>{{ end }}{{ if .PartialFields }}, {{ range $i, $f := .PartialFields }}{{ if $i }} | {{ end }}"{{ $f.TSName }}"{{ end }}>{{ end }}{{ if .ConditionalOmittedFields }}, {{ range $i, $f := .ConditionalOmittedFields }}{{ if $i }} | {{ end }}"{{ $f }}"{{ end }}>{{ end }}{{ if .ExtendFields }} & {
{{- range .ExtendFields }}
  {{ .TSName }}{{ if .IsOptional }}?{{ end }}: {{ .CoalescedTSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}{{ end }}{{ range .ConditionalFields }} & ([S["{{ .TypeParamName | camelCase }}"]] extends [{{ .NeverType }}] ? {} : { {{ .Field.TSName }}{{ if .Field.IsOptional }}?{{ end }}: {{ if .UseInput }}z.input{{ else }}z.infer{{ end }}<S["{{ .TypeParamName | camelCase }}"]>{{ if .Field.IsArray }}[]{{ end }} }){{ end }};
{{- else }}
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = {{ if .ConditionalOmittedFields }}Omit<{{ end }}{{ if .PartialFields }}optional.Optional<{{ end }}{{ if .OmittedFields }}Omit<{{ end }}{{ .ExtendsTypeName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>{{ if .OmittedFields }}, {{ range $i, $f := .OmittedFields }}{{ if $i }} | {{ end }}"{{ $f }}"{{ end }}>{{ end }}{{ if .PartialFields }}, {{ range $i, $f := .PartialFields }}{{ if $i }} | {{ end }}"{{ $f.TSName }}"{{ end }}>{{ end }}{{ if .ConditionalOmittedFields }}, {{ range $i, $f := .ConditionalOmittedFields }}{{ if $i }} | {{ end }}"{{ $f }}"{{ end }}>{{ end }}{{ if .ExtendFields }} & {
{{- range .ExtendFields }}
  {{ .TSName }}{{ if .IsOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}{{ end }}{{ range .ConditionalFields }} & ([{{ .TypeParamName }}] extends [{{ .NeverType }}] ? {} : { {{ .Field.TSName }}{{ if .Field.IsOptional }}?{{ end }}: {{ if .UseInput }}z.input{{ else }}z.infer{{ end }}<{{ .TypeParamName }}>{{ if .Field.IsArray }}[]{{ end }} }){{ end }};
{{- end }}
{{- else }}
{{- if .ConditionalFields }}
{{- if .CoalesceTypeParams }}
export type {{ .TSName }}<S extends {{ .TSName }}Schemas = {{ .TSName }}Schemas> = {
{{- range .BaseFields }}
  {{ .TSName }}{{ if .IsOptional }}?{{ end }}: {{ .CoalescedTSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}{{ range .ConditionalFields }} & ([S["{{ .TypeParamName | camelCase }}"]] extends [{{ .NeverType }}] ? {} : { {{ .Field.TSName }}{{ if .Field.IsOptional }}?{{ end }}: {{ .Field.CoalescedTSType }}{{ if .Field.IsArray }}[]{{ end }} }){{ end }};
{{- else }}
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = {
{{- range .BaseFields }}
  {{ .TSName }}{{ if .IsOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}{{ range .ConditionalFields }} & ([{{ .TypeParamName }}] extends [{{ .NeverType }}] ? {} : { {{ .Field.TSName }}{{ if .Field.IsOptional }}?{{ end }}: {{ .Field.TSType }}{{ if .Field.IsArray }}[]{{ end }} }){{ end }};
{{- end }}
{{- else }}
{{- if .CoalesceTypeParams }}
export interface {{ .TSName }}<S extends {{ .TSName }}Schemas = {{ .TSName }}Schemas> {
{{- range .Fields }}
  {{ .TSName }}{{ if .IsOptional }}?{{ end }}: {{ .CoalescedTSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}
{{- else }}
export interface {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> {
{{- range .Fields }}
  {{ .TSName }}{{ if .IsOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
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
{{- if not .TypeOnly }}

export const {{ camelCase .TSName }}Z = {{ range $i, $p := .ExtendsParents }}{{ if $i }}.extend({{ end }}{{ $p.Schema }}{{ if $i }}.shape){{ end }}{{ end }}
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
{{- end }}
{{- if $.GenerateTypes }}
{{- if .TypeOnly }}
{{- if .BaseIsUnion }}
export type {{ .TSName }} = {{ if .OmittedFields }}Omit<{{ end }}z.input<typeof {{ .ExtendsName }}>{{ if .OmittedFields }}, {{ range $i, $f := .OmittedFields }}{{ if $i }} | {{ end }}"{{ $f }}"{{ end }}>{{ end }};
{{- else }}
export interface {{ .TSName }} extends {{ if .OmittedFields }}Omit<{{ end }}z.input<typeof {{ .ExtendsName }}>{{ if .OmittedFields }}, {{ range $i, $f := .OmittedFields }}{{ if $i }} | {{ end }}"{{ $f }}"{{ end }}>{{ end }} {}
{{- end }}
{{- else }}
export interface {{ .TSName }} extends z.{{ if .UseInput }}input{{ else }}infer{{ end }}<typeof {{ camelCase .TSName }}Z> {}
{{- end }}
{{- end }}
{{- else }}
{{- if .Doc }}

{{ formatDoc .TSName .Doc }}
{{- end }}
{{- if and .IsRecursive $.GenerateTypes }}
export type {{ .TSName }} = {
{{- range .Fields }}
  {{ .TSName }}{{ if .IsOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
};
export const {{ camelCase .TSName }}Z: z.ZodType<{{ .TSName }}> = z.{{ if .Strict }}strictObject{{ else }}object{{ end }}({
{{- range .Fields }}
{{- if .Doc }}
  {{ formatDoc .TSName .Doc 2 }}
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
export const {{ camelCase .TSName }}Z = z.{{ if .Strict }}strictObject{{ else }}object{{ end }}({
{{- range .Fields }}
{{- if .Doc }}
  {{ formatDoc .TSName .Doc 2 }}
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
{{- if .Request }}
export type {{ .TSName }} = z.infer<typeof {{ camelCase .TSName }}Z>;
{{- else }}
export interface {{ .TSName }} extends z.{{ if .UseInput }}input{{ else }}infer{{ end }}<typeof {{ camelCase .TSName }}Z> {}
{{- end }}
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
  {{ formatDoc .TSName .Doc 2 }}
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
export interface {{ .TypeName }} extends z.infer<typeof {{ .SchemaName }}> {}
{{- end }}
{{- end }}
{{- if not .BorrowsEnum }}

export const {{ .TypesConst }} = [{{ range $i, $v := .Variants }}{{ if $i }}, {{ end }}"{{ $v.Value }}"{{ end }}] as const;
export const {{ .TypeSchemaName }} = z.enum({{ .TypesConst }});
{{- if $.GenerateTypes }}
export type {{ .TypeName }} = z.infer<typeof {{ .TypeSchemaName }}>;
{{- end }}
{{- end }}

{{ if .Doc -}}
{{ formatDoc .TSName .Doc }}
{{ end -}}
export const {{ .SchemaName }} = z.discriminatedUnion("{{ .Discriminator }}", [
{{- $lazy := .LazyVariants }}
{{- range .Variants }}
{{- if $lazy }}
  z.lazy(() => {{ .SchemaName }}),
{{- else }}
  {{ .SchemaName }},
{{- end }}
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
{{- if .Resource }}

export const resourceID = resource.createIDFactory<{{ .Resource.KeyType }}>("{{ .Resource.TypeName }}");
export const TYPE_RESOURCE_ID = resourceID({{ .Resource.KeyZeroValue }});
{{- end }}
`),
)

func boundLiteral(n validation.Number) string {
	if n.IsInt {
		return strconv.FormatInt(n.Int, 10)
	}
	return fmt.Sprint(n.Float)
}

// recordFunc returns the Zod record function for key and the key schema type that the
// record carries. z.record requires every member of an enum key, so an enum key takes
// z.partialRecord.
func recordFunc(
	key *resolution.TypeRef,
	keySchemaType string,
	table *resolution.Table,
) (string, string) {
	target, _, ok := resolution.Dealias(*key, table)
	if !ok {
		return "record", keySchemaType
	}
	if _, isEnum := target.Form.(resolution.EnumForm); !isEnum {
		return "record", keySchemaType
	}
	return "partialRecord", keySchemaType + " & z.core.$partial"
}
