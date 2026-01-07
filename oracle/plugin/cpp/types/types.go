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
	"github.com/synnaxlabs/oracle/domain/key"
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
var primitiveMapper = &cppprimitives.Mapper{}

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

var clangFormatCmd = []string{"clang-format", "-i"}

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

	// Generate files for structs and standalone aliases
	for _, outputPath := range combinedOrder {
		structs := structCollector.Get(outputPath)
		enums := enum.CollectReferenced(structs, req.Resolutions)
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

	// Handle standalone typedef-only outputs
	err = typeDefCollector.ForEach(func(outputPath string, typeDefs []resolution.Type) error {
		content, err := p.generateFile(outputPath, nil, nil, typeDefs, nil, req.Resolutions)
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

func (p *Plugin) PostWrite(files []string) error {
	if p.Options.DisableFormatter || len(files) == 0 {
		return nil
	}

	// Filter to only C++ header files
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
	}

	data := &templateData{
		OutputPath:  outputPath,
		Namespace:   deriveNamespace(outputPath),
		KeyFields:   make([]keyFieldData, 0),
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

	// Collect key fields from structs - these are inferred 'using' declarations
	skip := func(s resolution.Type) bool { return omit.IsType(s, "cpp") }
	keyFields := key.Collect(structs, table, skip)
	for _, kf := range keyFields {
		kfd := p.processKeyField(kf, data)
		// Only add if not already declared by a typedef
		if !declaredNames[kfd.Name] {
			declaredNames[kfd.Name] = true
			data.KeyFields = append(data.KeyFields, kfd)
		}
	}

	// Process typedefs (distinct types) - mark as declared
	for _, td := range typeDefs {
		tdd := p.processTypeDef(td, data)
		if !declaredNames[tdd.Name] {
			declaredNames[tdd.Name] = true
			data.TypeDefs = append(data.TypeDefs, tdd)
		}
	}

	// Process enums that are in the same namespace
	for _, e := range enums {
		if e.Namespace == namespace && !omit.IsType(e, "cpp") {
			data.Enums = append(data.Enums, p.processEnum(e))
		}
	}

	// Combine aliases and structs for topological sorting
	// These are the types that can have cross-dependencies
	var combinedTypes []resolution.Type
	for _, alias := range aliases {
		if !omit.IsType(alias, "cpp") {
			combinedTypes = append(combinedTypes, alias)
		}
	}
	for _, s := range structs {
		if !omit.IsType(s, "cpp") {
			combinedTypes = append(combinedTypes, s)
		}
	}

	// Sort topologically so dependencies come before dependents
	sortedTypes := table.TopologicalSort(combinedTypes)

	// Process in sorted order, creating unified declarations
	for _, typ := range sortedTypes {
		switch typ.Form.(type) {
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
	subNs := parts[len(parts)-1]
	return fmt.Sprintf("synnax::%s", subNs)
}

func (p *Plugin) processEnum(e resolution.Type) enumData {
	form, ok := e.Form.(resolution.EnumForm)
	if !ok {
		return enumData{Name: e.Name}
	}
	values := make([]enumValueData, 0, len(form.Values))
	for _, v := range form.Values {
		values = append(values, enumValueData{
			Name:     toPascalCase(v.Name),
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
	return typeDefData{
		Name:    td.Name,
		CppType: p.typeDefBaseToCpp(form.Base, data),
	}
}

func (p *Plugin) typeDefBaseToCpp(typeRef resolution.TypeRef, data *templateData) string {
	if resolution.IsPrimitive(typeRef.Name) {
		return p.primitiveToCpp(typeRef.Name, data)
	}
	// Try to resolve as another type
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return "void"
	}
	switch resolved.Form.(type) {
	case resolution.DistinctForm:
		// Another typedef - use its name (with namespace if different)
		if resolved.Namespace != data.rawNs {
			targetOutputPath := output.GetPath(resolved, "cpp")
			if targetOutputPath != "" {
				includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
				data.includes.addInternal(includePath)
			}
			// Use namespace-qualified name
			ns := deriveNamespace(targetOutputPath)
			return fmt.Sprintf("%s::%s", ns, resolved.Name)
		}
		return resolved.Name
	default:
		return "void"
	}
}

func (p *Plugin) processAlias(alias resolution.Type, data *templateData) aliasData {
	form, ok := alias.Form.(resolution.AliasForm)
	if !ok {
		return aliasData{Name: alias.Name, Target: "void"}
	}

	// Convert target type to C++ type string
	target := p.aliasTargetToCpp(form.Target, data)

	// Collect type parameters
	var typeParams []string
	for _, tp := range form.TypeParams {
		typeParams = append(typeParams, tp.Name)
	}

	return aliasData{
		Name:       alias.Name,
		Target:     target,
		IsGeneric:  len(typeParams) > 0,
		TypeParams: typeParams,
	}
}

func (p *Plugin) aliasTargetToCpp(typeRef resolution.TypeRef, data *templateData) string {
	// Handle type parameters
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
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
			if resolved.Namespace != "" {
				name = fmt.Sprintf("%s::%s", resolved.Namespace, name)
			}
		} else {
			// Generated type - include the generated header
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.includes.addInternal(includePath)
		}
	}

	// Build with type arguments
	if len(typeRef.TypeArgs) == 0 {
		return name
	}

	args := make([]string, len(typeRef.TypeArgs))
	for i, arg := range typeRef.TypeArgs {
		args[i] = p.aliasTargetToCpp(arg, data)
	}
	return fmt.Sprintf("%s<%s>", name, strings.Join(args, ", "))
}

func (p *Plugin) processKeyField(kf key.Field, data *templateData) keyFieldData {
	cppType := p.primitiveToCpp(kf.Primitive, data)
	return keyFieldData{
		Name:    lo.Capitalize(lo.CamelCase(kf.Name)),
		CppType: cppType,
	}
}

func (p *Plugin) processStruct(entry resolution.Type, data *templateData) structData {
	form, ok := entry.Form.(resolution.StructForm)
	if !ok {
		return structData{Name: entry.Name}
	}

	// Check for @cpp name override
	name := domain.GetName(entry, "cpp")

	// Always generate JSON parse and to_json methods
	data.includes.addInternal("x/cpp/xjson/xjson.h")

	// Check if this is an alias type
	aliasForm, isAlias := entry.Form.(resolution.AliasForm)

	sd := structData{
		Name:           name,
		Doc:            doc.Get(entry.Domains),
		Fields:         make([]fieldData, 0, len(form.Fields)),
		IsGeneric:      form.IsGeneric(),
		IsAlias:        isAlias,
		GenerateParse:  true,
		GenerateToJson: true,
	}

	// Process type parameters
	for _, tp := range form.TypeParams {
		sd.TypeParams = append(sd.TypeParams, p.processTypeParam(tp))
	}

	// Handle alias types
	if isAlias {
		sd.AliasOf = p.typeRefToCpp(aliasForm.Target, data)
		return sd
	}

	// For C++, we always flatten fields (no struct embedding like Go)
	// This handles both extending and non-extending structs uniformly
	for _, field := range resolution.UnifiedFields(entry, data.table) {
		sd.Fields = append(sd.Fields, p.processField(field, entry, data))
	}

	return sd
}

func (p *Plugin) processTypeParam(tp resolution.TypeParam) typeParamData {
	return typeParamData{
		Name: tp.Name,
	}
}

func (p *Plugin) processField(field resolution.Field, entry resolution.Type, data *templateData) fieldData {
	cppType := p.typeRefToCpp(field.Type, data)

	// Apply optional wrappers based on field flags
	if field.IsHardOptional {
		data.includes.addSystem("optional")
		cppType = fmt.Sprintf("std::optional<%s>", cppType)
	}

	// Generate JSON expressions
	parseExpr := p.parseExprForField(field, cppType, data)
	toJsonExpr := p.toJsonExprForField(field, data)

	return fieldData{
		Name:         field.Name,
		CppType:      cppType,
		Doc:          doc.Get(field.Domains),
		JsonName:     field.Name,
		ParseExpr:    parseExpr,
		ToJsonExpr:   toJsonExpr,
		HasDefault:   field.IsOptional,
		DefaultValue: p.defaultValueForType(field.Type, field.IsHardOptional, data),
	}
}

func (p *Plugin) typeRefToCpp(typeRef resolution.TypeRef, data *templateData) string {
	// Handle type parameters first
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
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
		if imp.Category == "system" {
			data.includes.addSystem(imp.Path)
		} else if imp.Category == "internal" {
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
			if cppInclude != "" {
				data.includes.addInternal(cppInclude)
			}
			// Use namespace prefix for handwritten types.
			if resolved.Namespace != "" {
				name = fmt.Sprintf("%s::%s", resolved.Namespace, name)
			}
		} else {
			// Generated type - include the generated header
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.includes.addInternal(includePath)
		}
	}

	return p.buildGenericType(name, typeArgs, data)
}

func (p *Plugin) resolveEnumType(resolved resolution.Type, form resolution.EnumForm, data *templateData) string {
	// For cross-namespace references, we need to add an include
	if resolved.Namespace != data.rawNs {
		targetOutputPath := enum.FindOutputPath(resolved, data.table, "cpp")
		if targetOutputPath != "" {
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.includes.addInternal(includePath)
		}
	}

	return resolved.Name
}

func (p *Plugin) resolveDistinctType(resolved resolution.Type, data *templateData) string {
	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.includes.addInternal(includePath)
		}
		// Use namespace-qualified name
		ns := deriveNamespace(targetOutputPath)
		return fmt.Sprintf("%s::%s", ns, resolved.Name)
	}
	return resolved.Name
}

func (p *Plugin) resolveAliasType(resolved resolution.Type, typeArgs []resolution.TypeRef, data *templateData) string {
	// Similar to struct handling for now
	name := resolved.Name
	if resolved.Namespace != data.rawNs {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			includePath := fmt.Sprintf("%s/%s", targetOutputPath, "types.gen.h")
			data.includes.addInternal(includePath)
		}
	}
	return p.buildGenericType(name, typeArgs, data)
}

func (p *Plugin) buildGenericType(baseName string, typeArgs []resolution.TypeRef, data *templateData) string {
	if len(typeArgs) == 0 {
		return baseName
	}

	args := make([]string, len(typeArgs))
	for i, arg := range typeArgs {
		args[i] = p.typeRefToCpp(arg, data)
	}
	return fmt.Sprintf("%s<%s>", baseName, strings.Join(args, ", "))
}

func toPascalCase(s string) string {
	return lo.PascalCase(s)
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
	OutputPath  string
	Namespace   string
	KeyFields   []keyFieldData
	Structs     []structData
	Enums       []enumData
	TypeDefs    []typeDefData
	Aliases     []aliasData
	SortedDecls []sortedDeclData // Topologically sorted aliases and structs
	includes    *includeManager
	table       *resolution.Table
	rawNs       string // Original Oracle namespace for cross-reference detection
}

type sortedDeclData struct {
	IsAlias  bool
	IsStruct bool
	Alias    aliasData
	Struct   structData
}

type keyFieldData struct {
	Name    string
	CppType string
}

type typeDefData struct {
	Name    string
	CppType string
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
	GenerateParse  bool
	GenerateToJson bool
}

type typeParamData struct {
	Name string
}

type fieldData struct {
	Name         string
	CppType      string
	Doc          string
	JsonName     string
	ParseExpr    string
	ToJsonExpr   string
	HasDefault   bool
	DefaultValue string
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
		return "telem::TimeStamp(0)"
	case "timespan":
		return "telem::TimeSpan(0)"
	case "time_range", "time_range_bounded":
		return "telem::TimeRange{}"
	case "json":
		return "nlohmann::json{}"
	case "bytes":
		return "{}"
	default:
		return "{}"
	}
}

func (p *Plugin) defaultValueForType(typeRef resolution.TypeRef, isHardOptional bool, data *templateData) string {
	// Check for Array
	if typeRef.Name == "Array" {
		return "{}"
	}
	if isHardOptional {
		return "std::nullopt"
	}
	// Check if it's a primitive
	if resolution.IsPrimitive(typeRef.Name) {
		return defaultValueForPrimitive(typeRef.Name)
	}
	// Try to resolve the type
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return "{}"
	}
	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		return "{}"
	case resolution.EnumForm:
		if form.IsIntEnum {
			return "static_cast<" + resolved.Name + ">(0)"
		}
		return `""`
	case resolution.DistinctForm, resolution.AliasForm:
		return "{}"
	default:
		return "{}"
	}
}

func (p *Plugin) parseExprForField(field resolution.Field, cppType string, data *templateData) string {
	typeRef := field.Type
	jsonName := field.Name
	hasDefault := field.IsOptional

	// Handle hard optional
	if field.IsHardOptional {
		// Hard optional uses std::optional, check if field exists
		innerType := p.typeRefToCpp(typeRef, data)
		innerExpr := p.parseExprForTypeRef(typeRef, innerType, jsonName, false, data)
		return fmt.Sprintf(`parser.has("%s") ? std::make_optional(%s) : std::nullopt`, jsonName, innerExpr)
	}

	// Handle arrays
	if typeRef.Name == "Array" {
		if hasDefault {
			return fmt.Sprintf(`parser.field<%s>("%s", {})`, cppType, jsonName)
		}
		return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
	}

	// Check if primitive
	if resolution.IsPrimitive(typeRef.Name) {
		if hasDefault {
			defaultVal := p.defaultValueForType(typeRef, false, data)
			return fmt.Sprintf(`parser.field<%s>("%s", %s)`, cppType, jsonName, defaultVal)
		}
		return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
	}

	// Try to resolve the type
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		if hasDefault {
			return fmt.Sprintf(`parser.field<%s>("%s", {})`, cppType, jsonName)
		}
		return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		// Nested structs call their parse method via optional_child
		structType := p.resolveStructType(resolved, typeRef.TypeArgs, data)
		if hasDefault {
			return fmt.Sprintf(`%s::parse(parser.optional_child("%s"))`, structType, jsonName)
		}
		return fmt.Sprintf(`%s::parse(parser.child("%s"))`, structType, jsonName)

	case resolution.EnumForm:
		// Enums need special handling based on int vs string
		if form.IsIntEnum {
			if hasDefault {
				return fmt.Sprintf(`static_cast<%s>(parser.field<int>("%s", 0))`, resolved.Name, jsonName)
			}
			return fmt.Sprintf(`static_cast<%s>(parser.field<int>("%s"))`, resolved.Name, jsonName)
		}
		// String enum - parse as string
		if hasDefault {
			return fmt.Sprintf(`parser.field<std::string>("%s", "")`, jsonName)
		}
		return fmt.Sprintf(`parser.field<std::string>("%s")`, jsonName)

	default:
		if hasDefault {
			defaultVal := p.defaultValueForType(typeRef, false, data)
			return fmt.Sprintf(`parser.field<%s>("%s", %s)`, cppType, jsonName, defaultVal)
		}
		return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
	}
}

func (p *Plugin) parseExprForTypeRef(typeRef resolution.TypeRef, cppType, jsonName string, hasDefault bool, data *templateData) string {
	// Handle type parameters
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		return fmt.Sprintf(`%s::parse(parser.child("%s"))`, typeRef.TypeParam.Name, jsonName)
	}

	// Check if primitive
	if resolution.IsPrimitive(typeRef.Name) {
		return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
	}

	// Try to resolve the type
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		structType := p.resolveStructType(resolved, typeRef.TypeArgs, data)
		return fmt.Sprintf(`%s::parse(parser.child("%s"))`, structType, jsonName)
	case resolution.EnumForm:
		if form.IsIntEnum {
			return fmt.Sprintf(`static_cast<%s>(parser.field<int>("%s"))`, resolved.Name, jsonName)
		}
		return fmt.Sprintf(`parser.field<std::string>("%s")`, jsonName)
	default:
		return fmt.Sprintf(`parser.field<%s>("%s")`, cppType, jsonName)
	}
}

func (p *Plugin) toJsonExprForField(field resolution.Field, data *templateData) string {
	typeRef := field.Type
	jsonName := field.Name
	fieldName := field.Name

	// Handle hard optional
	if field.IsHardOptional {
		innerExpr := p.toJsonValueExpr(typeRef, fieldName, data)
		return fmt.Sprintf(`if (this->%s.has_value()) j["%s"] = %s;`, fieldName, jsonName, innerExpr)
	}

	// Handle arrays of structs
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		elementType := typeRef.TypeArgs[0]
		if !resolution.IsPrimitive(elementType.Name) {
			if resolved, ok := elementType.Resolve(data.table); ok {
				if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
					return p.toJsonArrayOfStructsExpr(jsonName, fieldName)
				}
			}
		}
		// Array of primitives or enums - direct assignment
		return fmt.Sprintf(`j["%s"] = this->%s;`, jsonName, fieldName)
	}

	// Check if primitive
	if resolution.IsPrimitive(typeRef.Name) {
		return fmt.Sprintf(`j["%s"] = this->%s;`, jsonName, fieldName)
	}

	// Try to resolve the type
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return fmt.Sprintf(`j["%s"] = this->%s;`, jsonName, fieldName)
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		return fmt.Sprintf(`j["%s"] = this->%s.to_json();`, jsonName, fieldName)
	case resolution.EnumForm:
		if form.IsIntEnum {
			return fmt.Sprintf(`j["%s"] = static_cast<int>(this->%s);`, jsonName, fieldName)
		}
		// String enum - serialize as string directly
		return fmt.Sprintf(`j["%s"] = this->%s;`, jsonName, fieldName)
	default:
		return fmt.Sprintf(`j["%s"] = this->%s;`, jsonName, fieldName)
	}
}

func (p *Plugin) toJsonValueExpr(typeRef resolution.TypeRef, fieldName string, data *templateData) string {
	// Handle type parameters
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		return fmt.Sprintf("this->%s->to_json()", fieldName)
	}

	// Check if primitive
	if resolution.IsPrimitive(typeRef.Name) {
		return fmt.Sprintf("*this->%s", fieldName)
	}

	// Try to resolve the type
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return fmt.Sprintf("*this->%s", fieldName)
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		return fmt.Sprintf("this->%s->to_json()", fieldName)
	case resolution.EnumForm:
		if form.IsIntEnum {
			return fmt.Sprintf("static_cast<int>(*this->%s)", fieldName)
		}
		return fmt.Sprintf("*this->%s", fieldName)
	default:
		return fmt.Sprintf("*this->%s", fieldName)
	}
}

func (p *Plugin) toJsonArrayOfStructsExpr(jsonName, fieldName string) string {
	// For arrays of structs, we need to transform each element
	return fmt.Sprintf(`{
        auto arr = nlohmann::json::array();
        for (const auto& item : this->%s) arr.push_back(item.to_json());
        j["%s"] = arr;
    }`, fieldName, jsonName)
}

var templateFuncs = template.FuncMap{
	"join": strings.Join,
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
{{- range $i, $kf := .KeyFields}}
using {{$kf.Name}} = {{$kf.CppType}};
{{- end}}
{{- range $i, $td := .TypeDefs}}
using {{$td.Name}} = {{$td.CppType}};
{{- end}}
{{- range $i, $enum := .Enums}}
{{if $i}}
{{end}}
{{if $enum.IsIntEnum}}
enum class {{$enum.Name}} {
{{- range $j, $v := $enum.Values}}
    {{$v.Name}} = {{$v.IntValue}},
{{- end}}
};
{{- else}}
{{- range $enum.Values}}
constexpr const char* {{$enum.Name}}{{.Name}} = "{{.Value}}";
{{- end}}
{{- end}}
{{- end}}
{{- range $i, $d := .SortedDecls}}
{{- if $d.IsAlias}}
{{- $a := $d.Alias}}
{{if or $i (gt (len $.KeyFields) 0) (gt (len $.TypeDefs) 0) (gt (len $.Enums) 0)}}
{{end}}
{{- if $a.IsGeneric}}template <{{range $j, $p := $a.TypeParams}}{{if $j}}, {{end}}typename {{$p}}{{end}}>
{{end}}using {{$a.Name}} = {{$a.Target}};
{{- else if $d.IsStruct}}
{{- $s := $d.Struct}}
{{if or $i (gt (len $.KeyFields) 0) (gt (len $.TypeDefs) 0) (gt (len $.Enums) 0)}}
{{end}}
{{- if $s.Doc}}
/// @brief {{$s.Doc}}
{{end}}
{{- if $s.IsAlias}}
{{- if $s.IsGeneric}}template <{{range $j, $p := $s.TypeParams}}{{if $j}}, {{end}}typename {{$p.Name}}{{end}}>
{{end}}using {{$s.Name}} = {{$s.AliasOf}};
{{- else}}
{{- if $s.IsGeneric}}template <{{range $j, $p := $s.TypeParams}}{{if $j}}, {{end}}typename {{$p.Name}}{{end}}>
{{end}}struct {{$s.Name}} {
{{- range $s.Fields}}
{{- if .Doc}}
    /// @brief {{.Doc}}
{{- end}}
    {{.CppType}} {{.Name}};
{{- end}}
{{- if $s.GenerateParse}}

    static {{$s.Name}} parse(xjson::Parser parser) {
        return {{$s.Name}}{
{{- range $j, $f := $s.Fields}}
            .{{$f.Name}} = {{$f.ParseExpr}},
{{- end}}
        };
    }
{{- end}}
{{- if $s.GenerateToJson}}

    [[nodiscard]] json to_json() const {
        json j;
{{- range $s.Fields}}
        {{.ToJsonExpr}}
{{- end}}
        return j;
    }
{{- end}}
};
{{- end}}
{{- end}}
{{- end}}
}
`))
