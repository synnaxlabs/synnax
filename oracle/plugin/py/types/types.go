// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/domain/ontology"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/primitives"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

type Plugin struct{ Options Options }

type Options struct {
	OutputPath      string
	FileNamePattern string
}

func DefaultOptions() Options {
	return Options{
		OutputPath:      "{{.Namespace}}",
		FileNamePattern: "types_gen.py",
	}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string { return "py/types" }

func (p *Plugin) Domains() []string { return nil }

func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) Check(req *plugin.Request) error { return nil }

var postWriter = &exec.PostWriter{
	ConfigFile: "pyproject.toml",
	Commands: [][]string{
		{"poetry", "run", "isort"},
		{"poetry", "run", "black"},
	},
}

func (p *Plugin) PostWrite(files []string) error {
	return postWriter.PostWrite(files)
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	// Collect types using framework collectors
	typeDefCollector := framework.NewCollector("py", req)
	if err := typeDefCollector.AddAll(req.Resolutions.DistinctTypes()); err != nil {
		return nil, err
	}
	if err := typeDefCollector.AddAll(req.Resolutions.AliasTypes()); err != nil {
		return nil, err
	}

	structCollector, err := framework.CollectStructs("py", req)
	if err != nil {
		return nil, err
	}

	enumCollector := framework.NewCollector("py", req).
		WithPathFunc(func(typ resolution.Type) string { return output.GetPath(typ, "py") }).
		WithSkipFunc(nil)
	for _, e := range enum.CollectWithOwnOutput(req.Resolutions.EnumTypes(), "py") {
		if err := enumCollector.Add(e); err != nil {
			return nil, err
		}
	}

	// Generate files for structs (merging in enums and typedefs from same output path)
	err = structCollector.ForEach(func(outputPath string, structs []resolution.Type) error {
		enums := enum.CollectReferenced(structs, req.Resolutions)
		// Merge standalone enums that share the same output path
		if enumCollector.Has(outputPath) {
			enums = framework.MergeTypesByName(enums, enumCollector.Remove(outputPath))
		}
		var typeDefs []resolution.Type
		if typeDefCollector.Has(outputPath) {
			typeDefs = typeDefCollector.Remove(outputPath)
		}
		content, err := p.generateFile(structs[0].Namespace, outputPath, structs, enums, typeDefs, req.Resolutions)
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

	// Handle standalone typedef-only outputs
	err = typeDefCollector.ForEach(func(outputPath string, typeDefs []resolution.Type) error {
		content, err := p.generateFile(typeDefs[0].Namespace, outputPath, nil, nil, typeDefs, req.Resolutions)
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

	// Handle standalone enum-only outputs
	err = enumCollector.ForEach(func(outputPath string, enums []resolution.Type) error {
		content, err := p.generateFile(enums[0].Namespace, outputPath, nil, enums, nil, req.Resolutions)
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


func (p *Plugin) generateFile(
	namespace string,
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	typeDefs []resolution.Type,
	table *resolution.Table,
) ([]byte, error) {
	data := &templateData{
		Namespace:   namespace,
		OutputPath:  outputPath,
		KeyFields:   make([]keyFieldData, 0),
		Structs:     make([]structData, 0, len(structs)),
		Enums:       make([]enumData, 0, len(enums)),
		TypeDefs:    make([]typeDefData, 0, len(typeDefs)),
		SortedDecls: make([]sortedDeclData, 0),
		imports:     newImportManager(),
	}
	if len(structs) > 0 {
		data.imports.addPydantic("BaseModel")
	}

	// Track declared names to avoid duplicates
	declaredNames := make(map[string]bool)

	skip := func(typ resolution.Type) bool { return omit.IsType(typ, "py") }
	rawKeyFields := key.Collect(structs, table, skip)
	keyFields := p.convertKeyFields(rawKeyFields, data)
	data.KeyFields = keyFields
	data.Ontology = p.extractOntology(structs, rawKeyFields, keyFields, skip)
	if data.Ontology != nil {
		data.imports.addOntology("ID")
	}

	// Mark key fields as declared
	for _, kf := range keyFields {
		name := lo.Capitalize(lo.CamelCase(kf.Name))
		declaredNames[name] = true
	}

	// Separate distinct types from alias types
	var distinctTypeDefs []resolution.Type
	var aliasTypeDefs []resolution.Type
	for _, td := range typeDefs {
		switch td.Form.(type) {
		case resolution.AliasForm:
			aliasTypeDefs = append(aliasTypeDefs, td)
		default:
			distinctTypeDefs = append(distinctTypeDefs, td)
		}
	}

	// Process distinct types first (they don't depend on other schema types)
	// Skip if already declared by key fields
	for _, td := range distinctTypeDefs {
		if !declaredNames[td.Name] {
			declaredNames[td.Name] = true
			data.TypeDefs = append(data.TypeDefs, p.processTypeDef(td, table, data))
		}
	}

	for _, e := range enums {
		data.Enums = append(data.Enums, p.processEnum(e, data))
	}

	// Combine aliases and structs for topological sorting
	var combinedTypes []resolution.Type
	combinedTypes = append(combinedTypes, aliasTypeDefs...)
	combinedTypes = append(combinedTypes, structs...)

	// Sort topologically so dependencies come before dependents
	sortedTypes := table.TopologicalSort(combinedTypes)

	// Process in sorted order
	for _, typ := range sortedTypes {
		switch typ.Form.(type) {
		case resolution.AliasForm:
			data.SortedDecls = append(data.SortedDecls, sortedDeclData{
				IsTypeDef: true,
				TypeDef:   p.processTypeDef(typ, table, data),
			})
		case resolution.StructForm:
			data.SortedDecls = append(data.SortedDecls, sortedDeclData{
				IsStruct: true,
				Struct:   p.processStruct(typ, table, data, keyFields),
			})
		}
	}

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Plugin) convertKeyFields(fields []key.Field, data *templateData) []keyFieldData {
	result := make([]keyFieldData, 0, len(fields))
	for _, f := range fields {
		result = append(result, keyFieldData{
			Name:   f.Name,
			PyType: primitiveToPython(f.Primitive, data),
		})
	}
	return result
}

func (p *Plugin) extractOntology(types []resolution.Type, rawFields []key.Field, keyFields []keyFieldData, skip ontology.SkipFunc) *ontologyData {
	data := ontology.Extract(types, rawFields, skip)
	if data == nil || len(keyFields) == 0 {
		return nil
	}
	return &ontologyData{
		TypeName:   data.TypeName,
		KeyType:    keyFields[0].PyType,
		StructName: data.StructName,
	}
}

func (p *Plugin) processEnum(typ resolution.Type, data *templateData) enumData {
	form, ok := typ.Form.(resolution.EnumForm)
	if !ok {
		return enumData{Name: typ.Name}
	}
	values := make([]enumValueData, 0, len(form.Values))
	var literalValues []string
	for _, v := range form.Values {
		values = append(values, enumValueData{
			Name:      v.Name,
			Value:     v.StringValue(),
			IntValue:  v.IntValue(),
			IsIntEnum: form.IsIntEnum,
		})
		if !form.IsIntEnum {
			literalValues = append(literalValues, fmt.Sprintf("%q", v.StringValue()))
		}
	}
	if form.IsIntEnum {
		data.imports.addEnum("IntEnum")
	} else {
		data.imports.addTyping("Literal")
	}
	return enumData{
		Name:          typ.Name,
		Values:        values,
		IsIntEnum:     form.IsIntEnum,
		LiteralValues: strings.Join(literalValues, ", "),
	}
}

func (p *Plugin) processTypeDef(typ resolution.Type, table *resolution.Table, data *templateData) typeDefData {
	switch form := typ.Form.(type) {
	case resolution.DistinctForm:
		data.imports.addTyping("NewType")
		return typeDefData{
			Name:       typ.Name,
			BaseType:   p.typeDefBaseToPython(form.Base, typ.Namespace, table, data),
			IsDistinct: true,
		}
	case resolution.AliasForm:
		// For alias types, use typeRefToPythonAlias to properly handle struct references
		// with type arguments (e.g., status.Status<StatusDetails> -> status.Status[StatusDetails])
		data.imports.addTyping("TypeAlias")
		return typeDefData{
			Name:       typ.Name,
			BaseType:   p.typeRefToPythonAlias(form.Target, table, data),
			IsDistinct: false,
		}
	default:
		data.imports.addTyping("NewType")
		return typeDefData{Name: typ.Name, BaseType: "Any", IsDistinct: true}
	}
}

func (p *Plugin) typeDefBaseToPython(typeRef resolution.TypeRef, currentNamespace string, table *resolution.Table, data *templateData) string {
	if resolution.IsPrimitive(typeRef.Name) {
		return primitiveToPython(typeRef.Name, data)
	}
	// Try to resolve another typedef
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		data.imports.addTyping("Any")
		return "Any"
	}
	if _, isDistinct := resolved.Form.(resolution.DistinctForm); isDistinct {
		if resolved.Namespace != currentNamespace {
			ns := resolved.Namespace
			outputPath := output.GetPath(resolved, "py")
			if outputPath == "" {
				outputPath = ns
			}
			modulePath := toPythonModulePath(outputPath)
			data.imports.addNamespace(ns, modulePath)
			return fmt.Sprintf("%s.%s", ns, resolved.Name)
		}
		return resolved.Name
	}
	data.imports.addTyping("Any")
	return "Any"
}

func (p *Plugin) processStruct(
	entry resolution.Type,
	table *resolution.Table,
	data *templateData,
	keyFields []keyFieldData,
) structData {
	sd := structData{
		Name:   entry.Name,
		PyName: entry.Name, // Default to original name
	}

	form, ok := entry.Form.(resolution.StructForm)
	if !ok {
		sd.Skip = true
		return sd
	}

	// Check for py domain omit
	if domain.HasExprFromType(entry, "py", "omit") {
		sd.Skip = true
		return sd
	}
	// Check for py domain name override
	if name := domain.GetStringFromType(entry, "py", "name"); name != "" {
		sd.PyName = name
	}

	// Handle struct aliases (AliasForm)
	if aliasForm, isAlias := entry.Form.(resolution.AliasForm); isAlias {
		sd.IsAlias = true
		sd.AliasOf = p.typeRefToPythonAlias(aliasForm.Target, table, data)
		return sd
	}

	// Handle struct extension with Pydantic inheritance
	if form.Extends != nil {
		parent, parentOk := form.Extends.Resolve(table)
		if parentOk {
			sd.HasExtends = true
			sd.ExtendsName = getPyName(parent)
			parentFields := resolution.UnifiedFields(parent, table)

			// For extends, only include child's own fields (not inherited)
			// Pass OmittedFields so excluded fields get Field(exclude=True)
			sd.Fields = make([]fieldData, 0, len(form.Fields)+len(form.OmittedFields))
			redefinedFields := make(map[string]bool)
			for _, field := range form.Fields {
				redefinedFields[field.Name] = true
				sd.Fields = append(sd.Fields, p.processField(field, table, data, keyFields, form.OmittedFields))
				// Check if this field has @key annotation for __hash__ generation
				if key.HasKey(field) {
					sd.KeyField = field.Name
				}
			}
			// Also check parent fields for @key if not redefined
			for _, pf := range parentFields {
				if !redefinedFields[pf.Name] && key.HasKey(pf) {
					sd.KeyField = pf.Name
				}
			}

			// For omitted fields that aren't redefined, we need to explicitly
			// add them with Field(exclude=True) to exclude from serialization
			for _, omittedName := range form.OmittedFields {
				if redefinedFields[omittedName] {
					continue // Already handled above
				}
				// Find the field type from parent
				for _, pf := range parentFields {
					if pf.Name == omittedName {
						data.imports.addPydantic("Field")
						fd := fieldData{
							Name:   pf.Name,
							PyType: p.typeToPython(pf.Type, table, data),
						}
						if pf.IsOptional || pf.IsHardOptional {
							fd.PyType = fd.PyType + " | None"
							fd.Default = " = Field(default=None, exclude=True)"
						} else {
							fd.Default = " = Field(exclude=True)"
						}
						sd.Fields = append(sd.Fields, fd)
						break
					}
				}
			}
			return sd
		}
	}

	// Non-extending struct: use all fields (flattened for compatibility)
	allFields := resolution.UnifiedFields(entry, table)
	sd.Fields = make([]fieldData, 0, len(allFields))
	for _, field := range allFields {
		sd.Fields = append(sd.Fields, p.processField(field, table, data, keyFields, nil))
		// Check if this field has @key annotation for __hash__ generation
		if key.HasKey(field) {
			sd.KeyField = field.Name
		}
	}
	return sd
}

func getPyName(typ resolution.Type) string {
	return domain.GetName(typ, "py")
}

func (p *Plugin) processField(
	field resolution.Field,
	table *resolution.Table,
	data *templateData,
	keyFields []keyFieldData,
	excludedFields []string,
) fieldData {
	fd := fieldData{
		Name:           field.Name,
		IsOptional:     field.IsOptional,
		IsHardOptional: field.IsHardOptional,
		IsArray:        field.Type.Name == "Array",
	}

	baseType := p.typeToPython(field.Type, table, data)
	var fieldConstraints []string
	if validateDomain, ok := field.Domains["validate"]; ok {
		fieldConstraints = p.collectValidation(validateDomain, field.Type, table, data)
	}

	// Check if this field should be excluded from serialization (Pydantic v2)
	isExcluded := lo.Contains(excludedFields, field.Name)
	if isExcluded {
		fieldConstraints = append(fieldConstraints, "exclude=True")
	}

	if fd.IsArray {
		fd.PyType = fmt.Sprintf("list[%s]", baseType)
	} else {
		fd.PyType = baseType
	}

	// Both soft optional (?) and hard optional (??) become T | None in Python
	if field.IsOptional || field.IsHardOptional {
		fd.PyType = fd.PyType + " | None"
	}

	fd.Default = p.buildDefault(field, fieldConstraints, data)

	return fd
}

func (p *Plugin) buildDefault(
	field resolution.Field,
	constraints []string,
	data *templateData,
) string {
	hasConstraints := len(constraints) > 0

	isAnyOptional := field.IsOptional || field.IsHardOptional
	if isAnyOptional {
		if hasConstraints {
			data.imports.addPydantic("Field")
			return fmt.Sprintf(" = Field(default=None, %s)", strings.Join(constraints, ", "))
		}
		return " = None"
	}

	if hasConstraints {
		data.imports.addPydantic("Field")
		return fmt.Sprintf(" = Field(%s)", strings.Join(constraints, ", "))
	}

	return ""
}

func (p *Plugin) collectValidation(
	domain resolution.Domain,
	typeRef resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) []string {
	rules := validation.Parse(domain)
	if validation.IsEmpty(rules) {
		return nil
	}
	var constraints []string
	isString := resolution.IsPrimitive(typeRef.Name) && resolution.IsStringPrimitive(typeRef.Name)
	isNumber := resolution.IsPrimitive(typeRef.Name) && resolution.IsNumberPrimitive(typeRef.Name)
	if isString {
		if rules.MinLength != nil {
			constraints = append(constraints, fmt.Sprintf("min_length=%d", *rules.MinLength))
		}
		if rules.MaxLength != nil {
			constraints = append(constraints, fmt.Sprintf("max_length=%d", *rules.MaxLength))
		}
	}
	if isNumber {
		if rules.Min != nil {
			if rules.Min.IsInt {
				constraints = append(constraints, fmt.Sprintf("ge=%d", rules.Min.Int))
			} else {
				constraints = append(constraints, fmt.Sprintf("ge=%f", rules.Min.Float))
			}
		}
		if rules.Max != nil {
			if rules.Max.IsInt {
				constraints = append(constraints, fmt.Sprintf("le=%d", rules.Max.Int))
			} else {
				constraints = append(constraints, fmt.Sprintf("le=%f", rules.Max.Float))
			}
		}
	}
	if rules.Default != nil {
		switch rules.Default.Kind {
		case resolution.ValueKindBool:
			if rules.Default.BoolValue {
				constraints = append(constraints, "default=True")
			} else {
				constraints = append(constraints, "default=False")
			}
		case resolution.ValueKindInt:
			constraints = append(constraints, fmt.Sprintf("default=%d", rules.Default.IntValue))
		case resolution.ValueKindFloat:
			constraints = append(constraints, fmt.Sprintf("default=%f", rules.Default.FloatValue))
		case resolution.ValueKindString:
			constraints = append(constraints, fmt.Sprintf("default=%q", rules.Default.StringValue))
		}
	}
	return constraints
}

func addCrossNamespaceImport(modulePath string, typeName string, data *templateData) string {
	parts := strings.Split(modulePath, ".")
	if len(parts) >= 2 {
		parentPath := strings.Join(parts[:len(parts)-1], ".")
		moduleName := parts[len(parts)-1]
		data.imports.addModuleImport(parentPath, moduleName)
		return fmt.Sprintf("%s.%s", moduleName, typeName)
	}
	// Fallback for single-level module path (rare case)
	data.imports.addModuleImport("", modulePath)
	return fmt.Sprintf("%s.%s", modulePath, typeName)
}

func (p *Plugin) typeToPython(
	typeRef resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	// Handle primitives directly
	if resolution.IsPrimitive(typeRef.Name) {
		return primitiveToPython(typeRef.Name, data)
	}

	// Handle Array type
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		elemType := p.typeToPython(typeRef.TypeArgs[0], table, data)
		return elemType
	}

	// Try to resolve the type
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		data.imports.addTyping("Any")
		return "Any"
	}

	switch resolved.Form.(type) {
	case resolution.StructForm:
		if resolved.Namespace != data.Namespace {
			outputPath := output.GetPath(resolved, "py")
			if outputPath == "" {
				outputPath = resolved.Namespace
			}
			modulePath := toPythonModulePath(outputPath)
			return addCrossNamespaceImport(modulePath, resolved.Name, data)
		}
		return resolved.Name

	case resolution.EnumForm:
		if resolved.Namespace != data.Namespace {
			outputPath := enum.FindOutputPath(resolved, table, "py")
			if outputPath == "" {
				outputPath = resolved.Namespace
			}
			modulePath := toPythonModulePath(outputPath)
			return addCrossNamespaceImport(modulePath, resolved.Name, data)
		}
		return resolved.Name

	case resolution.DistinctForm:
		if resolved.Namespace != data.Namespace {
			outputPath := output.GetPath(resolved, "py")
			if outputPath == "" {
				outputPath = resolved.Namespace
			}
			modulePath := toPythonModulePath(outputPath)
			return addCrossNamespaceImport(modulePath, resolved.Name, data)
		}
		return resolved.Name

	case resolution.AliasForm:
		if resolved.Namespace != data.Namespace {
			outputPath := output.GetPath(resolved, "py")
			if outputPath == "" {
				outputPath = resolved.Namespace
			}
			modulePath := toPythonModulePath(outputPath)
			return addCrossNamespaceImport(modulePath, resolved.Name, data)
		}
		return resolved.Name

	default:
		data.imports.addTyping("Any")
		return "Any"
	}
}

func (p *Plugin) typeRefToPythonAlias(
	typeRef resolution.TypeRef,
	table *resolution.Table,
	data *templateData,
) string {
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return p.typeToPython(typeRef, table, data)
	}

	_, isStruct := resolved.Form.(resolution.StructForm)
	if !isStruct {
		return p.typeToPython(typeRef, table, data)
	}

	// Get the base type name with proper import handling
	var baseName string
	if resolved.Namespace != data.Namespace {
		outputPath := output.GetPath(resolved, "py")
		if outputPath == "" {
			outputPath = resolved.Namespace
		}
		modulePath := toPythonModulePath(outputPath)
		baseName = addCrossNamespaceImport(modulePath, resolved.Name, data)
	} else {
		baseName = resolved.Name
	}

	// If no type arguments, just return the base name
	if len(typeRef.TypeArgs) == 0 {
		return baseName
	}

	// Build type arguments for generic: Status[Details] or Status[Details, Data]
	var typeArgs []string
	for _, arg := range typeRef.TypeArgs {
		typeArgs = append(typeArgs, p.typeToPython(arg, table, data))
	}
	return fmt.Sprintf("%s[%s]", baseName, strings.Join(typeArgs, ", "))
}

func toPythonModulePath(repoPath string) string {
	prefixes := []string{
		"client/py/",
		"alamos/py/",
		"freighter/py/",
	}

	path := repoPath
	for _, prefix := range prefixes {
		if strings.HasPrefix(path, prefix) {
			path = strings.TrimPrefix(path, prefix)
			break
		}
	}

	return strings.ReplaceAll(path, "/", ".")
}

func primitiveToPython(primitive string, data *templateData) string {
	mapping := primitives.GetMapping(primitive, "py")
	if mapping.TargetType == "any" {
		data.imports.addTyping("Any")
		return "Any"
	}
	for _, imp := range mapping.Imports {
		switch imp.Category {
		case "uuid":
			data.imports.addUUID(imp.Name)
		case "typing":
			data.imports.addTyping(imp.Name)
		case "synnax":
			data.imports.addSynnax(imp.Name)
		}
	}
	return mapping.TargetType
}

type importManager struct {
	uuid       []string
	typing     []string
	enum       []string
	pydantic   []string
	synnax     []string
	ontology   []string              // imports from synnax.ontology.payload
	namespaces []namespaceImportData // alias -> path
	modules    []moduleImportData    // module name -> parent path (for "from parent import module")
}

func newImportManager() *importManager {
	return &importManager{}
}

func (m *importManager) addUUID(name string) {
	if !lo.Contains(m.uuid, name) {
		m.uuid = append(m.uuid, name)
	}
}
func (m *importManager) addTyping(name string) {
	if !lo.Contains(m.typing, name) {
		m.typing = append(m.typing, name)
	}
}
func (m *importManager) addEnum(name string) {
	if !lo.Contains(m.enum, name) {
		m.enum = append(m.enum, name)
	}
}
func (m *importManager) addPydantic(name string) {
	if !lo.Contains(m.pydantic, name) {
		m.pydantic = append(m.pydantic, name)
	}
}
func (m *importManager) addSynnax(name string) {
	if !lo.Contains(m.synnax, name) {
		m.synnax = append(m.synnax, name)
	}
}
func (m *importManager) addOntology(name string) {
	if !lo.Contains(m.ontology, name) {
		m.ontology = append(m.ontology, name)
	}
}
func (m *importManager) addNamespace(alias, path string) {
	if !lo.ContainsBy(m.namespaces, func(n namespaceImportData) bool { return n.Alias == alias }) {
		m.namespaces = append(m.namespaces, namespaceImportData{Alias: alias, Path: path})
	}
}
func (m *importManager) addModuleImport(parentPath, moduleName string) {
	if !lo.ContainsBy(m.modules, func(mod moduleImportData) bool { return mod.Module == moduleName }) {
		m.modules = append(m.modules, moduleImportData{Module: moduleName, Parent: parentPath})
	}
}

type templateData struct {
	Namespace   string
	OutputPath  string
	KeyFields   []keyFieldData
	Structs     []structData
	Enums       []enumData
	TypeDefs    []typeDefData
	SortedDecls []sortedDeclData // Topologically sorted aliases and structs
	imports     *importManager
	Ontology    *ontologyData // Ontology data if domain ontology is present
}

type sortedDeclData struct {
	IsTypeDef bool
	IsStruct  bool
	TypeDef   typeDefData
	Struct    structData
}

type typeDefData struct {
	Name       string // Type definition name
	BaseType   string // Python base type (e.g., "int", "str")
	IsDistinct bool   // If true, use NewType; if false, use TypeAlias
}

type ontologyData struct {
	TypeName   string
	KeyType    string
	StructName string
}

func (d *templateData) UUIDImports() []string     { return d.imports.uuid }
func (d *templateData) TypingImports() []string   { return d.imports.typing }
func (d *templateData) EnumImports() []string     { return d.imports.enum }
func (d *templateData) PydanticImports() []string { return d.imports.pydantic }
func (d *templateData) SynnaxImports() []string   { return d.imports.synnax }
func (d *templateData) OntologyImports() []string { return d.imports.ontology }

func (d *templateData) NamespaceImports() []namespaceImportData { return d.imports.namespaces }

func (d *templateData) ModuleImports() []moduleImportData { return d.imports.modules }

type keyFieldData struct {
	Name   string
	PyType string
}

type structData struct {
	Name    string // Original struct name from schema
	PyName  string // Python name (can be overridden via py domain { name "..." })
	Fields  []fieldData
	Skip    bool   // If true, skip generating this struct (omit)
	IsAlias bool   // If true, this struct is a type alias
	AliasOf string // Python expression for the aliased type (e.g., "status.Status[StatusDetails]")

	// Extension support
	HasExtends  bool
	ExtendsName string // Parent class name

	// Key field support for __hash__ generation
	KeyField string // Name of the key field (if any) for generating __hash__
}

type fieldData struct {
	Name           string
	PyType         string
	Default        string
	IsOptional     bool
	IsHardOptional bool
	IsArray        bool
}

type enumData struct {
	Name          string
	Values        []enumValueData
	IsIntEnum     bool
	LiteralValues string
}

type enumValueData struct {
	Name      string
	Value     string
	IntValue  int64
	IsIntEnum bool
}

type namespaceImportData struct {
	Alias string
	Path  string
}

type moduleImportData struct {
	Module string // module name to import (e.g., "status")
	Parent string // parent path to import from (e.g., "synnax")
}

var templateFuncs = template.FuncMap{
	"title": lo.Capitalize,
	"join":  strings.Join,
	"upper": strings.ToUpper,
}

var fileTemplate = template.Must(template.New("python").Funcs(templateFuncs).Parse(`# Code generated by Oracle. DO NOT EDIT.

from __future__ import annotations
{{- if .UUIDImports }}
from uuid import {{ join .UUIDImports ", " }}
{{- end }}
{{- if .TypingImports }}
from typing import {{ join .TypingImports ", " }}
{{- end }}
{{- if .EnumImports }}
from enum import {{ join .EnumImports ", " }}
{{- end }}
{{- if .PydanticImports }}
from pydantic import {{ join .PydanticImports ", " }}
{{- end }}
{{- if .SynnaxImports }}
from synnax.telem import {{ join .SynnaxImports ", " }}
{{- end }}
{{- if .OntologyImports }}
from synnax.ontology.payload import {{ join .OntologyImports ", " }}
{{- end }}
{{- range .NamespaceImports }}
from {{ .Path }} import {{ .Alias }}
{{- end }}
{{- range .ModuleImports }}
from {{ .Parent }} import {{ .Module }}
{{- end }}
{{- range .KeyFields }}

{{ .Name | title }} = {{ .PyType }}
{{- end }}
{{- range .TypeDefs }}
{{- if .IsDistinct }}

{{ .Name }} = NewType("{{ .Name }}", {{ .BaseType }})
{{- else }}

{{ .Name }}: TypeAlias = {{ .BaseType }}
{{- end }}
{{- end }}
{{- range .Enums }}
{{- if .IsIntEnum }}


class {{ .Name }}(IntEnum):
{{- range .Values }}
    {{ .Name }} = {{ .IntValue }}
{{- end }}
{{- else }}


{{ .Name }} = Literal[{{ .LiteralValues }}]
{{- end }}
{{- end }}
{{- range .SortedDecls }}
{{- if .IsTypeDef }}
{{- if .TypeDef.IsDistinct }}

{{ .TypeDef.Name }} = NewType("{{ .TypeDef.Name }}", {{ .TypeDef.BaseType }})
{{- else }}

{{ .TypeDef.Name }}: TypeAlias = {{ .TypeDef.BaseType }}
{{- end }}
{{- else if .IsStruct }}
{{- with .Struct }}
{{- if not .Skip }}
{{- if .IsAlias }}


{{ .PyName }} = {{ .AliasOf }}
{{- else if .HasExtends }}


class {{ .PyName }}({{ .ExtendsName }}):
{{- if or .Fields .KeyField }}
{{- range .Fields }}
    {{ .Name }}: {{ .PyType }}{{ .Default }}
{{- end }}
{{- if .KeyField }}

    def __hash__(self) -> int:
        return hash(self.{{ .KeyField }})
{{- end }}
{{- else }}
    pass
{{- end }}
{{- else }}


class {{ .PyName }}(BaseModel):
{{- range .Fields }}
    {{ .Name }}: {{ .PyType }}{{ .Default }}
{{- end }}
{{- if .KeyField }}

    def __hash__(self) -> int:
        return hash(self.{{ .KeyField }})
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- if .Ontology }}


ONTOLOGY_TYPE = ID(type="{{ .Ontology.TypeName }}")


def ontology_id(key: {{ .Ontology.KeyType }}) -> ID:
    return ID(type="{{ .Ontology.TypeName }}", key=str(key))
{{- end }}
`))
