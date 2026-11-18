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
	"strconv"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/cpp/internal/defaults"
	"github.com/synnaxlabs/oracle/plugin/cpp/internal/includes"
	"github.com/synnaxlabs/oracle/plugin/cpp/keywords"
	cppnaming "github.com/synnaxlabs/oracle/plugin/cpp/naming"
	cppprimitives "github.com/synnaxlabs/oracle/plugin/cpp/primitives"
	cpptypes "github.com/synnaxlabs/oracle/plugin/cpp/types"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/internal/arrays"
	"github.com/synnaxlabs/oracle/plugin/internal/telem"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

var primitiveMapper = cppprimitives.Mapper()

type Plugin struct{ Options Options }

type Options struct {
	FileNamePattern string
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

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	req = req.OmitFields("cpp")
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	c, err := collect(req)
	if err != nil {
		return nil, err
	}

	for outputPath := range c.jsonPaths {
		structs := c.structs.Get(outputPath)
		distinctTypes := c.distinct.Get(outputPath)
		unions := c.unions.Get(outputPath)

		var filePath string
		switch {
		case len(structs) > 0:
			filePath = structs[0].FilePath
		case len(distinctTypes) > 0:
			filePath = distinctTypes[0].FilePath
		default:
			filePath = unions[0].FilePath
		}

		content, err := p.generateFile(
			outputPath, structs, distinctTypes, unions, filePath, c.jsonPaths, req,
		)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate json for %s", outputPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
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
		deletions.Add(fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern))
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
		if lo.SomeBy(c.distinct.Get(path), func(t resolution.Type) bool {
			return serializesArray(t, req.Resolutions)
		}) {
			c.jsonPaths.Add(path)
		}
	}
	return c, nil
}

// serializesArray reports whether t is a distinct type wrapping an array, the only
// distinct form that needs JSON serialization. A fixed-size uint8 array is excluded,
// since its own hand-written header serializes it.
func serializesArray(t resolution.Type, table *resolution.Table) bool {
	form, ok := t.Form.(resolution.DistinctForm)
	if !ok {
		return false
	}
	_, ok = arrays.Wrapped(form, table)
	return ok && !isFixedSizeUint8ArrayType(t, table)
}

func (p *Plugin) generateFile(
	outputPath string,
	structs []resolution.Type,
	distinctTypes []resolution.Type,
	unions []resolution.Type,
	filePath string,
	jsonPaths set.Set[string],
	req *plugin.Request,
) ([]byte, error) {
	data := &templateData{
		OutputPath:    outputPath,
		Namespace:     cppnaming.Namespace(outputPath),
		Serializers:   make([]serializerData, 0, len(structs)),
		ArrayWrappers: make([]arrayWrapperData, 0),
		Manager:       includes.NewManager(),
		table:         req.Resolutions,
		jsonPaths:     jsonPaths,
		filePath:      filePath,
	}

	data.AddInternal(outputPath + "/types.gen.h")
	data.AddInternal("x/cpp/json/json.h")

	for _, s := range structs {
		serializer, err := p.processStruct(s, data)
		if err != nil {
			return nil, err
		}
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
	if !serializesArray(dt, data.table) {
		return nil
	}
	base, _ := arrays.Wrapped(
		resolution.MustForm[resolution.DistinctForm](dt),
		data.table,
	)
	var arraySize int64
	if base.ArraySize != nil {
		arraySize = *base.ArraySize
		data.AddSystem("algorithm")
		data.AddSystem("string")
	}

	name := domain.GetName(dt, "cpp")
	elemType := base.TypeArgs[0]
	elemCppType := p.typeRefToCpp(elemType, data)

	elemNeedsConversion := false
	if elem, _, ok := resolution.Dealias(elemType, data.table); ok {
		if _, isStruct := elem.Form.(resolution.StructForm); isStruct {
			elemNeedsConversion = true
		}
	}

	return &arrayWrapperData{
		Name:                name,
		ElementType:         elemCppType,
		ElementNeedsConvert: elemNeedsConversion,
		IsFixedSizeArray:    base.ArraySize != nil,
		ArraySize:           arraySize,
	}
}

func (p *Plugin) resolveExtendsType(
	extendsRef resolution.TypeRef,
	parent resolution.Type,
	data *templateData,
) string {
	name := domain.GetName(parent, "cpp")

	if parent.FilePath != data.filePath {
		targetOutputPath := output.GetPath(parent, "cpp")
		if targetOutputPath != "" {
			data.AddInternal(data.jsonInclude(targetOutputPath))
			ns := cppnaming.Namespace(targetOutputPath)
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
) (*serializerData, error) {
	form, ok := s.Form.(resolution.StructForm)
	if !ok {
		return nil, nil
	}
	fields := resolution.UnifiedFields(s, data.table)
	for _, field := range fields {
		if err := checkMapKey(s, field, data.table); err != nil {
			return nil, err
		}
	}
	typeParams := serializerTypeParams(form)
	typeParamNames := lo.Map(typeParams, func(tp typeParamData, _ int) string {
		return tp.Name
	})
	serializer := &serializerData{
		Name:           domain.GetName(s, "cpp"),
		IsGeneric:      len(typeParams) > 0,
		TypeParams:     typeParams,
		TypeParamNames: strings.Join(typeParamNames, ", "),
	}
	if len(typeParams) > 0 {
		data.AddSystem("type_traits")
	}
	if resolver.CanUseInheritance(form, "cpp", data.table) {
		serializer.HasExtends = true
		serializer.ParentTypes = p.parentTypes(form, data)
		fields = form.Fields
	}
	serializer.Fields = make([]fieldData, 0, len(fields))
	for _, field := range fields {
		serializer.Fields = append(serializer.Fields, p.processField(field, s, data))
	}
	return serializer, nil
}

// serializerTypeParams returns the type parameters of form that have no default.
func serializerTypeParams(form resolution.StructForm) []typeParamData {
	typeParams := make([]typeParamData, 0, len(form.TypeParams))
	for _, tp := range form.TypeParams {
		if !tp.HasDefault() {
			typeParams = append(typeParams, typeParamData{Name: tp.Name})
		}
	}
	return typeParams
}

func (p *Plugin) parentTypes(
	form resolution.StructForm,
	data *templateData,
) []parentTypeData {
	var parents []parentTypeData
	for _, extendsRef := range form.Extends {
		parent, ok := extendsRef.Resolve(data.table)
		if !ok {
			continue
		}
		parents = append(parents, parentTypeData{
			QualifiedName: p.resolveExtendsType(extendsRef, parent, data),
		})
	}
	return parents
}

// objectMap returns the map type behind ref, whether its values are structs, and
// whether to_json must build its JSON object by hand: nlohmann_json writes non-string
// keys as an array of pairs and cannot reach the to_json member of a struct.
func objectMap(
	ref resolution.TypeRef,
	table *resolution.Table,
) (resolution.TypeRef, bool, bool) {
	if _, target, ok := resolution.Dealias(ref, table); ok {
		ref = target
	}
	if ref.Name != "Map" || len(ref.TypeArgs) < 2 {
		return resolution.TypeRef{}, false, false
	}
	isStruct := false
	if value, _, ok := resolution.Dealias(ref.TypeArgs[1], table); ok {
		_, isStruct = value.Form.(resolution.StructForm)
	}
	key := ref.TypeArgs[0]
	return ref, isStruct, isStruct || key.IsTypeParam() ||
		mapKeyBase(key, table) != "string"
}

// mapKeyBase returns the primitive that a map key is written as. C++ holds a string
// enum as a std::string, so a string enum key is a string.
func mapKeyBase(key resolution.TypeRef, table *resolution.Table) string {
	if target, _, ok := resolution.Dealias(key, table); ok {
		form, isEnum := target.Form.(resolution.EnumForm)
		if isEnum && !form.IsIntEnum {
			return "string"
		}
	}
	return resolution.PrimitiveBase(key, table)
}

// checkMapKey rejects a map that to_json builds by hand when its key has no exact JSON
// object key: to_json writes string keys as they are, UUID keys through to_string, and
// integer keys through std::to_string.
func checkMapKey(
	s resolution.Type,
	field resolution.Field,
	table *resolution.Table,
) error {
	ref, _, ok := objectMap(field.Type, table)
	if !ok || ref.TypeArgs[0].IsTypeParam() {
		return nil
	}
	base := mapKeyBase(ref.TypeArgs[0], table)
	if base == "string" || base == "uuid" || strings.HasPrefix(base, "int") ||
		strings.HasPrefix(base, "uint") {
		return nil
	}
	return errors.Wrapf(
		validate.ErrValidation,
		"%s.%s: C++ cannot serialize a map keyed by %s",
		s.QualifiedName,
		field.Name,
		ref.TypeArgs[0].Name,
	)
}

func (p *Plugin) processField(
	field resolution.Field,
	parent resolution.Type,
	data *templateData,
) fieldData {
	cppType := p.typeRefToCpp(field.Type, data)
	indirect := cpptypes.Indirect(field, parent, data.table)

	// A string minted by the create sentinel needs the UUID header, which the
	// std::string mapping does not bring in.
	if defaultIdent(field) == "create" &&
		resolution.PrimitiveBase(field.Type, data.table) == "string" {
		data.AddInternal("x/cpp/uuid/uuid.h")
	}

	parseExpr := p.parseExprForField(field, cppType, data, indirect)
	toJSONExpr := toJSONExprForField(field, data, indirect)

	return fieldData{
		Name:       memberName(field),
		ParseExpr:  parseExpr,
		ToJSONExpr: toJSONExpr,
	}
}

// memberName returns the name of the C++ struct member that holds field.
func memberName(field resolution.Field) string {
	name := domain.GetFieldName(field, "cpp")
	if name == field.Name {
		name = casing.FieldSnake(field.Name)
	}
	return keywords.Escape(name)
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
		data.AddSystem("vector")
		return fmt.Sprintf("std::vector<%s>", innerType)
	}
	if typeRef.Name == "Map" {
		return p.mapToCpp(typeRef, data)
	}
	if cppType, ok := primitiveToCpp(typeRef.Name, data); ok {
		return cppType
	}
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return typeRef.Name
	}
	if enumForm, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
		if !enumForm.IsIntEnum {
			data.AddSystem("string")
			return "std::string"
		}
		return resolved.Name
	}
	if _, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		data.addAliasedStructInclude(typeRef)
	}
	name := domain.GetName(resolved, "cpp")
	if outputPath := data.foreignOutputPath(resolved); outputPath != "" {
		data.AddInternal(foreignInclude(resolved, outputPath, data))
		name = qualify(outputPath, name)
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

func (p *Plugin) mapToCpp(typeRef resolution.TypeRef, data *templateData) string {
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

// primitiveToCpp returns the C++ type of the primitive name and adds its headers. It
// returns false when name is not a primitive with a C++ type.
func primitiveToCpp(name string, data *templateData) (string, bool) {
	mapping := primitiveMapper.Map(name)
	if mapping.TargetType == "" || mapping.TargetType == "void" {
		return "", false
	}
	for _, imp := range mapping.Imports {
		if imp.Category == "system" {
			data.AddSystem(imp.Path)
		} else {
			data.AddInternal(imp.Path)
		}
	}
	return mapping.TargetType, true
}

// addAliasedStructInclude adds the JSON include for the struct behind the alias ref
// when that struct lives in another file.
func (d *templateData) addAliasedStructInclude(ref resolution.TypeRef) {
	target, _, ok := resolution.Dealias(ref, d.table)
	if !ok {
		return
	}
	if _, isStruct := target.Form.(resolution.StructForm); !isStruct {
		return
	}
	if outputPath := d.foreignOutputPath(target); outputPath != "" {
		d.AddInternal(d.jsonInclude(outputPath))
	}
}

// foreignInclude returns the header that declares t, which lives at outputPath in
// another file. A cpp include expression on t takes precedence.
func foreignInclude(t resolution.Type, outputPath string, data *templateData) string {
	if cppDomain, ok := t.Domains["cpp"]; ok {
		if expr, found := cppDomain.Expressions.Find("include"); found &&
			len(expr.Values) > 0 && expr.Values[0].StringValue != "" {
			return expr.Values[0].StringValue
		}
	}
	if isFixedSizeUint8ArrayType(t, data.table) {
		return fmt.Sprintf("%s/%s.h", outputPath, lo.SnakeCase(t.Name))
	}
	return data.jsonInclude(outputPath)
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
		out = append(out, p.typeRefToCpp(arg, data))
	}
	if len(out) == 0 {
		return ""
	}
	return "<" + strings.Join(out, ", ") + ">"
}

func (p *Plugin) parseExprForField(
	field resolution.Field,
	cppType string,
	data *templateData,
	indirect bool,
) string {
	f := fieldParser{
		Plugin:     p,
		field:      field,
		cppType:    cppType,
		data:       data,
		indirect:   indirect,
		jsonName:   casing.FieldSnake(field.Name),
		hasDefault: field.Optional || hasRenderableDefault(field, data.table),
	}
	typeRef := field.Type
	if typeRef.TypeParam != nil && !typeRef.TypeParam.HasDefault() {
		return f.optionalOr(typeRef.TypeParam.Name)
	}
	if wrapperType, ok := f.arrayWrapperType(); ok {
		return f.composite(wrapperType)
	}
	if elemType, isArray := arrays.ElementType(typeRef, data.table); isArray {
		return f.array(elemType)
	}
	if resolved, ok := typeRef.Resolve(data.table); ok {
		if expr, ok := f.named(resolved); ok {
			return expr
		}
	}
	if mapping := primitiveMapper.Map(typeRef.Name); mapping.TargetType != "" &&
		mapping.TargetType != "void" {
		return f.primitive()
	}
	return f.other()
}

// fieldParser renders the C++ expression that parses one field from a JSON parser.
type fieldParser struct {
	*Plugin
	field      resolution.Field
	cppType    string
	data       *templateData
	indirect   bool
	jsonName   string
	hasDefault bool
}

func (f fieldParser) plain(cppType string) string {
	return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, f.jsonName)
}

// required parses a field with no fallback. A required string or number field also
// refuses its zero value, as Go and TypeScript do.
func (f fieldParser) required() string {
	return fmt.Sprintf(
		`parser.%s<%s>("%s")`,
		fieldFunc(f.field, f.data.table),
		f.cppType,
		f.jsonName,
	)
}

// fieldFunc returns the parser method that reads field.
func fieldFunc(field resolution.Field, table *resolution.Table) string {
	// The Core mints a key, so a client sends its zero value on create.
	_, isKey := field.Domains["key"]
	if isKey || field.Optional ||
		!validation.FieldRules(field, table).Required {
		return "field"
	}
	base := resolution.PrimitiveBase(field.Type, table)
	if resolution.IsPrimitive(base) &&
		(resolution.IsStringPrimitive(base) || resolution.IsNumberPrimitive(base)) {
		return "required_field"
	}
	return "field"
}

func (f fieldParser) withFallback(cppType, fallback string) string {
	return fmt.Sprintf(`parser.field<%s>("%s", %s)`, cppType, f.jsonName, fallback)
}

func (f fieldParser) optionalOr(cppType string) string {
	if f.field.Optional {
		return f.plain(fmt.Sprintf("std::optional<%s>", cppType))
	}
	return f.plain(cppType)
}

// composite parses a struct-like field, falling back to its struct or array default.
func (f fieldParser) composite(cppType string) string {
	if f.field.Optional {
		return f.plain(fmt.Sprintf("std::optional<%s>", cppType))
	}
	if fallback := f.compositeFallback(f.field, cppType, f.data); fallback != "" {
		return f.withFallback(cppType, fallback)
	}
	return f.plain(cppType)
}

func (f fieldParser) vector(elemType string) string {
	if f.field.Optional {
		return f.plain(fmt.Sprintf("std::optional<std::vector<%s>>", elemType))
	}
	vec := fmt.Sprintf("std::vector<%s>", elemType)
	return f.withFallback(vec, vec+"{}")
}

// arrayWrapperType returns the C++ name of the field's type when it is a distinct type
// that wraps an array.
func (f fieldParser) arrayWrapperType() (string, bool) {
	target, _, ok := resolution.Dealias(f.field.Type, f.data.table)
	if !ok {
		return "", false
	}
	form, isDistinct := target.Form.(resolution.DistinctForm)
	if !isDistinct {
		return "", false
	}
	if _, wraps := arrays.Wrapped(form, f.data.table); !wraps {
		return "", false
	}
	resolved, _ := f.field.Type.Resolve(f.data.table)
	return f.data.qualifiedName(resolved), true
}

func (f fieldParser) array(elemType resolution.TypeRef) string {
	innerType := f.typeRefToCpp(elemType, f.data)
	if elemType.TypeParam != nil {
		return f.vector(elemType.TypeParam.Name)
	}
	elemResolved, ok := elemType.Resolve(f.data.table)
	if !ok {
		return f.vector(innerType)
	}
	if structForm, isStruct := elemResolved.Form.(resolution.StructForm); isStruct {
		structType := f.data.qualifiedName(elemResolved) +
			f.typeArgsToCpp(structForm.TypeParams, elemType.TypeArgs, f.data)
		return f.vector(structType)
	}
	if union, ok := dealiasUnion(elemType, f.data.table); ok {
		_, parseFn := f.unionNames(union)
		return fmt.Sprintf(
			`[&] {
        std::vector<%s> result;
        if (parser.has("%s"))
            parser.iter("%s", [&result](x::json::Parser& p) { result.push_back(%s(p)); });
        return result;
    }()`,
			innerType, f.jsonName, f.jsonName, parseFn,
		)
	}
	return f.vector(innerType)
}

// named parses a field whose type resolves to an enum, struct, union, or struct alias.
// It returns false for any other type.
func (f fieldParser) named(resolved resolution.Type) (string, bool) {
	if enumForm, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
		if !enumForm.IsIntEnum {
			return f.stringEnum(), true
		}
		return f.intEnum(f.data.qualifiedName(resolved)), true
	}
	if structForm, isStruct := resolved.Form.(resolution.StructForm); isStruct {
		structType := f.data.qualifiedName(resolved) +
			f.typeArgsToCpp(structForm.TypeParams, f.field.Type.TypeArgs, f.data)
		if f.field.Optional && f.indirect {
			return f.plain(fmt.Sprintf("x::mem::indirect<%s>", structType)), true
		}
		return f.composite(structType), true
	}
	if union, ok := dealiasUnion(f.field.Type, f.data.table); ok {
		return f.union(union), true
	}
	aliasForm, isAlias := resolved.Form.(resolution.AliasForm)
	if !isAlias {
		return "", false
	}
	target, _, ok := resolution.Dealias(f.field.Type, f.data.table)
	if !ok {
		return "", false
	}
	if _, isStruct := target.Form.(resolution.StructForm); !isStruct {
		return "", false
	}
	aliasType := f.data.qualifiedName(resolved) +
		f.typeArgsToCpp(aliasForm.TypeParams, f.field.Type.TypeArgs, f.data)
	return f.composite(aliasType), true
}

func (f fieldParser) stringEnum() string {
	if f.field.Optional {
		return f.plain("std::optional<std::string>")
	}
	if !f.hasDefault {
		return f.plain("std::string")
	}
	defaultVal := jsonDefaultLiteral(f.field, f.data.table)
	if defaultVal == "" {
		defaultVal = `""`
	}
	return f.withFallback("std::string", defaultVal)
}

func (f fieldParser) intEnum(enumType string) string {
	if f.field.Optional {
		return f.plain(fmt.Sprintf("std::optional<%s>", enumType))
	}
	if f.field.Default != nil {
		if literal := f.defaultRenderer(f.data).Literal(
			f.field.Type,
			*f.field.Default,
		); literal != "" {
			return f.withFallback(enumType, literal)
		}
	}
	return f.plain(enumType)
}

// unionNames returns the C++ type name of union and the name of its parse function,
// adding the JSON include when union lives in another namespace.
func (f fieldParser) unionNames(union resolution.Type) (string, string) {
	unionType := domain.GetName(union, "cpp")
	parseFn := "parse_" + lo.SnakeCase(unionType)
	outputPath := f.data.foreignOutputPath(union)
	if outputPath == "" {
		return unionType, parseFn
	}
	f.data.AddInternal(f.data.jsonInclude(outputPath))
	return qualify(outputPath, unionType), qualify(outputPath, parseFn)
}

func (f fieldParser) union(union resolution.Type) string {
	unionType, parseFn := f.unionNames(union)
	if f.field.Optional {
		return fmt.Sprintf(
			`parser.has("%s") ? std::optional<%s>(%s(parser.child("%s"))) : std::nullopt`,
			f.jsonName,
			f.cppType,
			parseFn,
			f.jsonName,
		)
	}
	if !f.hasDefault {
		return fmt.Sprintf(`%s(parser.child("%s"))`, parseFn, f.jsonName)
	}
	// std::variant default-constructs its first alternative, so a default naming any
	// other variant must be written out.
	fallback := f.cppType + "{}"
	if uv, ok := validation.ResolveUnionVariant(
		defaultIdent(f.field),
		f.field.Type,
		f.data.table,
	); ok {
		fallback = fmt.Sprintf(
			"%s{%s{}}",
			f.cppType,
			cppnaming.QualifiedVariantTypeName(unionType, uv.Variant.Name),
		)
	}
	return fmt.Sprintf(
		`parser.has("%s") ? %s(parser.child("%s")) : %s`,
		f.jsonName, parseFn, f.jsonName, fallback,
	)
}

func (f fieldParser) primitive() string {
	if f.field.Optional {
		return f.plain(fmt.Sprintf("std::optional<%s>", f.cppType))
	}
	if f.hasDefault {
		if defaultVal := jsonDefaultLiteral(f.field, f.data.table); defaultVal != "" {
			return f.withFallback(f.cppType, defaultVal)
		}
	}
	return f.required()
}

func (f fieldParser) other() string {
	if f.field.Optional {
		if f.indirect {
			return f.plain(fmt.Sprintf("x::mem::indirect<%s>", f.cppType))
		}
		return f.plain(fmt.Sprintf("std::optional<%s>", f.cppType))
	}
	if !f.hasDefault {
		return f.required()
	}
	if defaultVal := jsonDefaultLiteral(f.field, f.data.table); defaultVal != "" {
		// Hand-written distinct types such as x::telem::Rate and x::telem::DataType
		// expose only explicit constructors, so a bare literal fails to bind as the
		// fallback value.
		if isScalarDefault(f.field.Default) &&
			resolution.IsDistinct(f.field.Type, f.data.table) {
			defaultVal = fmt.Sprintf("%s(%s)", f.cppType, defaultVal)
		}
		return f.withFallback(f.cppType, defaultVal)
	}
	if fallback := f.compositeFallback(f.field, f.cppType, f.data); fallback != "" {
		return f.withFallback(f.cppType, fallback)
	}
	return f.required()
}

// compositeFallback renders a struct or array default as a value of cppType. It returns
// "" when the field has no such default.
func (p *Plugin) compositeFallback(
	field resolution.Field,
	cppType string,
	data *templateData,
) string {
	if field.Default == nil {
		return ""
	}
	switch field.Default.Kind {
	case resolution.ValueKindStruct, resolution.ValueKindArray:
		return cppType + p.defaultRenderer(data).Literal(field.Type, *field.Default)
	case resolution.ValueKindString,
		resolution.ValueKindInt,
		resolution.ValueKindFloat,
		resolution.ValueKindBool,
		resolution.ValueKindIdent:
	}
	return ""
}

func (p *Plugin) defaultRenderer(data *templateData) defaults.Renderer {
	return defaults.Renderer{
		Table:    data.table,
		FilePath: data.filePath,
		TypeName: func(typeRef resolution.TypeRef) string {
			return p.typeRefToCpp(typeRef, data)
		},
	}
}

func toJSONExprForField(
	field resolution.Field,
	data *templateData,
	indirect bool,
) string {
	w := fieldWriter{
		field:    field,
		data:     data,
		member:   "this->" + memberName(field),
		jsonName: casing.FieldSnake(field.Name),
	}
	typeRef := field.Type
	if typeRef.TypeParam != nil && !typeRef.TypeParam.HasDefault() {
		return w.call("x::json::to_json")
	}
	// An indirect<T> has the same has_value() and -> interface as std::optional<T>,
	// and its T (a struct, or a distinct or alias of one) always has to_json().
	// Without this case, a cycle through a distinct falls through to the default
	// assignment below, which is ill-typed against indirect<T>.
	if indirect {
		return w.ifPresent(w.set(w.member + "->to_json()"))
	}
	if wrapsArray(typeRef, data.table) {
		return w.method("to_json")
	}
	if mapRef, isStruct, ok := objectMap(typeRef, data.table); ok {
		return w.objectMap(mapRef, isStruct)
	}
	if elemType, isArray := arrays.ElementType(typeRef, data.table); isArray {
		return w.array(elemType)
	}
	if resolved, ok := typeRef.Resolve(data.table); ok {
		if expr, ok := w.named(resolved); ok {
			return expr
		}
	}
	if telem.IsTimeStamp(typeRef, data.table, "cpp") ||
		telem.IsTimeSpan(typeRef, data.table, "cpp") {
		return w.method("nanoseconds")
	}
	if resolution.PrimitiveBase(typeRef, data.table) == "uuid" {
		return w.method("to_json")
	}
	return w.value()
}

// fieldWriter renders the C++ statement that writes one field into the JSON object j.
type fieldWriter struct {
	field    resolution.Field
	data     *templateData
	member   string
	jsonName string
}

func (w fieldWriter) set(value string) string {
	return fmt.Sprintf(`j["%s"] = %s;`, w.jsonName, value)
}

func (w fieldWriter) ifPresent(stmt string) string {
	return fmt.Sprintf(`if (%s.has_value()) %s`, w.member, stmt)
}

// source returns the expression for the field's value, dereferenced when the field is
// optional.
func (w fieldWriter) source() string {
	if w.field.Optional {
		return "*" + w.member
	}
	return w.member
}

// guard runs stmt only when an optional field holds a value.
func (w fieldWriter) guard(stmt string) string {
	if w.field.Optional {
		return w.ifPresent(stmt)
	}
	return stmt
}

func (w fieldWriter) value() string { return w.guard(w.set(w.source())) }

// call writes the result of the free function fn applied to the field's value.
func (w fieldWriter) call(fn string) string {
	return w.guard(w.set(fmt.Sprintf("%s(%s)", fn, w.source())))
}

// method writes the result of the member function name of the field's value.
func (w fieldWriter) method(name string) string {
	if w.field.Optional {
		return w.ifPresent(w.set(fmt.Sprintf("%s->%s()", w.member, name)))
	}
	return w.set(fmt.Sprintf("%s.%s()", w.member, name))
}

func (w fieldWriter) objectMap(mapRef resolution.TypeRef, isStruct bool) string {
	value := "item"
	if isStruct {
		value = "item.to_json()"
	}
	return w.guard(fmt.Sprintf(`{
        auto obj = x::json::json::object();
        for (const auto& [key, item] : %s) obj[%s] = %s;
        j["%s"] = obj;
    }`, w.source(), w.mapKey(mapRef.TypeArgs[0]), value, w.jsonName))
}

// mapKey returns the JSON object key for the map key named key. JSON object keys are
// strings, and obj[key] with an integer key indexes an array.
func (w fieldWriter) mapKey(key resolution.TypeRef) string {
	switch base := mapKeyBase(key, w.data.table); {
	case key.IsTypeParam():
		return "x::json::map_key(key)"
	case base == "string":
		return "key"
	case base == "uuid":
		return "key.to_string()"
	default:
		w.data.AddSystem("string")
		return "std::to_string(key)"
	}
}

func (w fieldWriter) array(elemType resolution.TypeRef) string {
	if elemType.TypeParam != nil {
		return w.call("x::json::to_array")
	}
	// UUIDs serialize through their to_json member, which nlohmann_json cannot reach
	// inside a vector.
	if resolution.PrimitiveBase(elemType, w.data.table) == "uuid" {
		return w.guard(fmt.Sprintf(`{
        auto arr = x::json::json::array();
        for (const auto& item : %s) arr.push_back(item.to_json());
        j["%s"] = arr;
    }`, w.source(), w.jsonName))
	}
	elem, _, ok := resolution.Dealias(elemType, w.data.table)
	if !ok {
		return w.value()
	}
	if _, isStruct := elem.Form.(resolution.StructForm); isStruct {
		return w.call("x::json::to_array")
	}
	if _, isUnion := elem.Form.(resolution.UnionForm); isUnion {
		return fmt.Sprintf(`{
        auto arr = x::json::json::array();
        for (const auto& item : %s) arr.push_back(%s::to_json(item));
        j["%s"] = arr;
    }`, w.member, w.data.unionQualifier(elem), w.jsonName)
	}
	// nlohmann_json cannot serialize vector<vector<Struct>>, so write each inner array
	// through to_array.
	if innerElem, ok := arrays.ElementType(elemType, w.data.table); ok &&
		isStructRef(innerElem, w.data.table) {
		return fmt.Sprintf(`{
        auto arr = x::json::json::array();
        for (const auto& inner : %s) arr.push_back(x::json::to_array(inner));
        j["%s"] = arr;
    }`, w.member, w.jsonName)
	}
	return w.value()
}

// named writes a field whose type resolves to a struct, union, or struct alias. It
// returns false for any other type.
func (w fieldWriter) named(resolved resolution.Type) (string, bool) {
	if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
		return w.method("to_json"), true
	}
	if union, ok := dealiasUnion(w.field.Type, w.data.table); ok {
		return w.call(w.data.unionQualifier(union) + "::to_json"), true
	}
	if _, isAlias := resolved.Form.(resolution.AliasForm); isAlias &&
		isStructRef(w.field.Type, w.data.table) {
		return w.method("to_json"), true
	}
	return "", false
}

// unionQualifier returns the namespace that holds the free to_json of union, adding
// its JSON include when union lives in another file. Unqualified lookup inside a
// member to_json() finds the member and never reaches the free overload.
func (d *templateData) unionQualifier(union resolution.Type) string {
	outputPath := d.foreignOutputPath(union)
	if outputPath == "" {
		return "::" + d.Namespace
	}
	d.AddInternal(d.jsonInclude(outputPath))
	return "::" + cppnaming.Namespace(outputPath)
}

// wrapsArray reports whether ref stands for a distinct type that wraps an array.
func wrapsArray(ref resolution.TypeRef, table *resolution.Table) bool {
	target, _, ok := resolution.Dealias(ref, table)
	if !ok {
		return false
	}
	form, isDistinct := target.Form.(resolution.DistinctForm)
	if !isDistinct {
		return false
	}
	_, wraps := arrays.Wrapped(form, table)
	return wraps
}

// isStructRef reports whether ref stands for a struct, following aliases.
func isStructRef(ref resolution.TypeRef, table *resolution.Table) bool {
	t, _, ok := resolution.Dealias(ref, table)
	if !ok {
		return false
	}
	_, isStruct := t.Form.(resolution.StructForm)
	return isStruct
}

// dealiasUnion returns the union ref stands for, following aliases.
func dealiasUnion(
	ref resolution.TypeRef,
	table *resolution.Table,
) (resolution.Type, bool) {
	t, _, ok := resolution.Dealias(ref, table)
	if !ok {
		return resolution.Type{}, false
	}
	_, isUnion := t.Form.(resolution.UnionForm)
	return t, isUnion
}

// hasRenderableDefault reports whether the field declares a default the parse
// expression can honor. Struct and array defaults count (their branches render them);
// identifier defaults count only when they resolve to an enum variant, a boolean
// literal, the create sentinel on a UUID or string field, or now on a timestamp.
func hasRenderableDefault(field resolution.Field, table *resolution.Table) bool {
	if field.Default == nil {
		return false
	}
	if field.Default.Kind != resolution.ValueKindIdent {
		return true
	}
	return jsonDefaultLiteral(field, table) != ""
}

// defaultIdent returns the field's identifier default, or "" when it has no default or
// the default is not an identifier.
func defaultIdent(field resolution.Field) string {
	if field.Default == nil || field.Default.Kind != resolution.ValueKindIdent {
		return ""
	}
	return field.Default.IdentValue
}

// isScalarDefault reports whether the default renders as a bare string, integer, or
// float literal, the three kinds a distinct type's constructor must wrap.
func isScalarDefault(val *resolution.ExpressionValue) bool {
	if val == nil {
		return false
	}
	switch val.Kind {
	case resolution.ValueKindString,
		resolution.ValueKindInt,
		resolution.ValueKindFloat:
		return true
	case resolution.ValueKindBool,
		resolution.ValueKindIdent,
		resolution.ValueKindArray,
		resolution.ValueKindStruct:
	}
	return false
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
		return strconv.FormatInt(v.IntValue, 10)
	case resolution.ValueKindFloat:
		return fmt.Sprintf("%f", v.FloatValue)
	case resolution.ValueKindBool:
		return strconv.FormatBool(v.BoolValue)
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
		// The create sentinel mints a fresh identifier on parse, matching the TS
		// (id.create) and Python (uuid4) generators.
		if v.IdentValue == "create" {
			switch resolution.PrimitiveBase(field.Type, table) {
			case "uuid":
				return "x::uuid::create()"
			case "string":
				return "x::uuid::create().to_string()"
			}
		}
		// A union default renders in the union parse branch, not as a parser.field
		// fallback, so report it renderable without a literal here.
		if _, ok := validation.ResolveUnionVariant(
			v.IdentValue,
			field.Type,
			table,
		); ok {
			return "{}"
		}
		if v.IdentValue == "now" && telem.IsTimeStamp(field.Type, table, "cpp") {
			return "x::telem::TimeStamp::now()"
		}
		// Other unresolvable idents have no C++ rendering; the field stays required.
		return ""
	case resolution.ValueKindArray:
		return ""
	}
	return ""
}

func isFixedSizeUint8ArrayType(resolved resolution.Type, table *resolution.Table) bool {
	form, ok := resolved.Form.(resolution.DistinctForm)
	if !ok {
		return false
	}
	base, ok := arrays.Wrapped(form, table)
	return ok && base.ArraySize != nil && base.TypeArgs[0].Name == "uint8"
}

type templateData struct {
	*includes.Manager
	table         *resolution.Table
	jsonPaths     set.Set[string]
	OutputPath    string
	Namespace     string
	filePath      string
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
	// IsFixedSizeArray is true when the wrapper holds exactly ArraySize elements.
	IsFixedSizeArray bool
	ArraySize        int64
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
	Name       string
	ParseExpr  string
	ToJSONExpr string
}

// foreignOutputPath returns the C++ output path of t when t lives in another file, and
// "" otherwise.
func (d *templateData) foreignOutputPath(t resolution.Type) string {
	if t.FilePath == d.filePath {
		return ""
	}
	return output.GetPath(t, "cpp")
}

// qualifiedName returns the C++ name of t, qualified by its namespace when t lives in
// another file.
func (d *templateData) qualifiedName(t resolution.Type) string {
	name := domain.GetName(t, "cpp")
	if outputPath := d.foreignOutputPath(t); outputPath != "" {
		return qualify(outputPath, name)
	}
	return name
}

func qualify(outputPath, name string) string {
	return fmt.Sprintf("::%s::%s", cppnaming.Namespace(outputPath), name)
}
