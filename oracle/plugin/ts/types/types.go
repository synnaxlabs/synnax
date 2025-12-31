// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package types provides the TypeScript types code generation plugin for Oracle.
// It generates Zod schemas and TypeScript type definitions from Oracle schemas.
package types

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/handwritten"
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/domain/ontology"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	pluginoutput "github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
)

// Plugin generates TypeScript type definitions and Zod schemas from Oracle schemas.
type Plugin struct{ Options Options }

// Options configures the TypeScript types plugin.
type Options struct {
	OutputPath      string
	FileNamePattern string
	GenerateTypes   bool
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		OutputPath:      "{{.Namespace}}",
		FileNamePattern: "types.gen.ts",
		GenerateTypes:   true,
	}
}

// New creates a new TypeScript types plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "ts/types" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return nil }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return nil }

// Check verifies generated files are up-to-date.
func (p *Plugin) Check(req *plugin.Request) error { return nil }

var (
	prettierCmd = []string{"npx", "prettier", "--write"}
	eslintCmd   = []string{"npx", "eslint", "--fix"}
)

// PostWrite runs prettier and eslint on the generated TypeScript files.
// Files are grouped by their package directory (containing package.json).
// Prettier runs first to format, then eslint --fix to sort imports and apply fixes.
func (p *Plugin) PostWrite(files []string) error {
	if len(files) == 0 {
		return nil
	}
	byPackage := make(map[string][]string)
	for _, file := range files {
		if pkgDir := findPackageDir(file); pkgDir != "" {
			byPackage[pkgDir] = append(byPackage[pkgDir], file)
		}
	}
	for pkgDir, pkgFiles := range byPackage {
		if err := exec.OnFiles(prettierCmd, pkgFiles, pkgDir); err != nil {
			return err
		}
		if err := exec.OnFiles(eslintCmd, pkgFiles, pkgDir); err != nil {
			return err
		}
	}
	return nil
}

// findPackageDir finds the nearest directory containing package.json.
func findPackageDir(file string) string {
	dir := filepath.Dir(file)
	for dir != "/" && dir != "." {
		if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return ""
}

// Generate produces TypeScript type definition files from the analyzed schemas.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	outputStructs := make(map[string][]*resolution.Struct)
	outputEnums := make(map[string][]*resolution.Enum)
	for _, entry := range req.Resolutions.AllStructs() {
		if outputPath := pluginoutput.GetPath(entry, "ts"); outputPath != "" {
			if req.RepoRoot != "" {
				if err := req.ValidateOutputPath(outputPath); err != nil {
					return nil, fmt.Errorf("invalid output path for struct %s: %w", entry.Name, err)
				}
			}
			outputStructs[outputPath] = append(outputStructs[outputPath], entry)
		}
	}
	for _, e := range enum.CollectWithOwnOutput(req.Resolutions.AllEnums(), "ts") {
		enumPath := pluginoutput.GetEnumPath(e, "ts")
		if req.RepoRoot != "" {
			if err := req.ValidateOutputPath(enumPath); err != nil {
				return nil, fmt.Errorf("invalid output path for enum %s: %w", e.Name, err)
			}
		}
		outputEnums[enumPath] = append(outputEnums[enumPath], e)
	}
	for outputPath, structs := range outputStructs {
		enums := enum.CollectReferenced(structs)
		if standaloneEnums, ok := outputEnums[outputPath]; ok {
			enums = mergeEnums(enums, standaloneEnums)
			delete(outputEnums, outputPath)
		}
		content, err := p.generateFile(structs[0].Namespace, outputPath, structs, enums, req)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s: %w", outputPath, err)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
			Content: content,
		})
	}
	for outputPath, enums := range outputEnums {
		content, err := p.generateFile(enums[0].Namespace, outputPath, nil, enums, req)
		if err != nil {
			return nil, fmt.Errorf("failed to generate %s: %w", outputPath, err)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
			Content: content,
		})
	}

	return resp, nil
}

func mergeEnums(a, b []*resolution.Enum) []*resolution.Enum {
	seen := make(map[string]bool, len(a))
	for _, e := range a {
		seen[e.QualifiedName] = true
	}
	result := append([]*resolution.Enum{}, a...)
	for _, e := range b {
		if !seen[e.QualifiedName] {
			result = append(result, e)
		}
	}
	return result
}

// packageMapping defines the known TypeScript package mappings in the workspace.
type packageMapping struct {
	pathPrefix     string // e.g., "client/ts/src"
	packageName    string // e.g., "@synnaxlabs/client"
	internalPrefix string // e.g., "@/"
}

var knownPackages = []packageMapping{
	{pathPrefix: "client/ts/src", packageName: "@synnaxlabs/client", internalPrefix: "@/"},
	{pathPrefix: "x/ts/src", packageName: "@synnaxlabs/x", internalPrefix: "@/"},
	{pathPrefix: "pluto/src", packageName: "@synnaxlabs/pluto", internalPrefix: "@/"},
	{pathPrefix: "freighter/ts/src", packageName: "@synnaxlabs/freighter", internalPrefix: "@/"},
	{pathPrefix: "alamos/ts/src", packageName: "@synnaxlabs/alamos", internalPrefix: "@/"},
	{pathPrefix: "drift/src", packageName: "@synnaxlabs/drift", internalPrefix: "@/"},
}

// findPackage finds the package mapping for a given output path.
func findPackage(outputPath string) *packageMapping {
	for i := range knownPackages {
		if strings.HasPrefix(outputPath, knownPackages[i].pathPrefix) {
			return &knownPackages[i]
		}
	}
	return nil
}

func calculateImportPath(fromPath, toPath string) string {
	fromPkg, toPkg := findPackage(fromPath), findPackage(toPath)
	if fromPkg == nil || toPkg == nil {
		return calculateRelativeImport(fromPath, toPath)
	}
	if fromPkg.packageName == toPkg.packageName {
		relativePath := strings.TrimPrefix(strings.TrimPrefix(toPath, toPkg.pathPrefix), "/")
		return toPkg.internalPrefix + relativePath
	}
	return toPkg.packageName
}

func calculateRelativeImport(from, to string) string {
	rel, err := filepath.Rel(from, to)
	if err != nil {
		return "./" + to
	}
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel
}

func (p *Plugin) generateFile(
	namespace string,
	outputPath string,
	structs []*resolution.Struct,
	enums []*resolution.Enum,
	req *plugin.Request,
) ([]byte, error) {
	data := &templateData{
		Namespace:     namespace,
		OutputPath:    outputPath,
		Request:       req,
		KeyFields:     make([]keyFieldData, 0),
		Structs:       make([]structData, 0, len(structs)),
		Enums:         make([]enumData, 0, len(enums)),
		GenerateTypes: p.Options.GenerateTypes,
		Imports:       make(map[string]*importSpec),
	}
	skip := func(s *resolution.Struct) bool { return handwritten.IsStruct(s, "ts") }
	rawKeyFields := key.Collect(structs, skip)
	keyFields := p.convertKeyFields(rawKeyFields, structs, data)
	data.KeyFields = keyFields
	data.Ontology = p.extractOntology(structs, rawKeyFields, skip)
	if data.Ontology != nil {
		data.addNamedImport("@/ontology", "ontology")
	}
	for _, enum := range enums {
		data.Enums = append(data.Enums, p.processEnum(enum))
	}
	for _, entry := range structs {
		data.Structs = append(data.Structs, p.processStruct(entry, req.Resolutions, data, keyFields))
	}
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Plugin) convertKeyFields(fields []key.Field, structs []*resolution.Struct, data *templateData) []keyFieldData {
	result := make([]keyFieldData, 0, len(fields))
	for _, f := range fields {
		primitive := f.Primitive
		if override := findFieldTypeOverride(structs, f.Name, "ts"); override != "" {
			primitive = override
		}
		result = append(result, keyFieldData{
			Name:      f.Name,
			TSName:    lo.CamelCase(f.Name),
			ZodType:   primitiveToZod(primitive, data, false),
			Primitive: primitive,
		})
	}
	return result
}

func findFieldTypeOverride(structs []*resolution.Struct, fieldName, domainName string) string {
	for _, s := range structs {
		for _, f := range s.Fields {
			if f.Name == fieldName {
				if override := getFieldTypeOverride(f, domainName); override != "" {
					return override
				}
			}
		}
	}
	return ""
}

func (p *Plugin) extractOntology(structs []*resolution.Struct, keyFields []key.Field, skip ontology.SkipFunc) *ontologyData {
	data := ontology.Extract(structs, keyFields, skip)
	if data == nil {
		return nil
	}
	keyType := lo.Capitalize(lo.CamelCase(data.KeyField.Name))
	// Check if there's a @ts type override for the key field - if so, use that for the zero value
	primitive := data.KeyField.Primitive
	if override := findFieldTypeOverride(structs, data.KeyField.Name, "ts"); override != "" {
		primitive = override
	}
	keyZeroValue := primitiveZeroValue(primitive)
	return &ontologyData{
		TypeName:     data.TypeName,
		KeyType:      keyType,
		KeyZeroValue: keyZeroValue,
	}
}

func (p *Plugin) processEnum(enum *resolution.Enum) enumData {
	values := make([]enumValueData, 0, len(enum.Values))
	for _, v := range enum.Values {
		values = append(values, enumValueData{
			Name:      v.Name,
			Value:     v.StringValue,
			IntValue:  v.IntValue,
			IsIntEnum: enum.IsIntEnum,
		})
	}
	return enumData{Name: enum.Name, Values: values, IsIntEnum: enum.IsIntEnum}
}

func (p *Plugin) processStruct(entry *resolution.Struct, table *resolution.Table, data *templateData, keyFields []keyFieldData) structData {
	sd := structData{
		Name:          entry.Name,
		TSName:        entry.Name,
		IsGeneric:     entry.IsGeneric(),
		IsSingleParam: len(entry.TypeParams) == 1,
		IsAlias:       entry.IsAlias(),
		IsRecursive:   entry.IsRecursive,
	}
	if tsDomain, ok := entry.Domains["ts"]; ok {
		for _, expr := range tsDomain.Expressions {
			switch expr.Name {
			case "use_input":
				sd.UseInput = true
			case "handwritten":
				sd.Handwritten = true
			case "concrete_types":
				sd.ConcreteTypes = true
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
	for _, tp := range entry.TypeParams {
		sd.TypeParams = append(sd.TypeParams, p.processTypeParam(tp, data))
	}
	if entry.IsAlias() {
		sd.AliasOf = p.typeToZod(entry.AliasOf, table, data, sd.UseInput)
		return sd
	}

	// Handle struct extension with Zod's .omit().partial().extend() pattern
	if entry.HasExtends() && entry.Extends.StructRef != nil {
		sd.HasExtends = true
		parentStruct := entry.Extends.StructRef

		// Get parent's TSName (respecting @ts name domain)
		parentTSName := parentStruct.Name
		if tsDomain, ok := parentStruct.Domains["ts"]; ok {
			for _, expr := range tsDomain.Expressions {
				if expr.Name == "name" && len(expr.Values) > 0 {
					parentTSName = expr.Values[0].StringValue
					break
				}
			}
		}
		sd.ExtendsName = lo.CamelCase(parentTSName) + "Z"
		sd.ExtendsTypeName = parentTSName
		// Convert omitted field names to camelCase for TypeScript
		for _, f := range entry.OmittedFields {
			sd.OmittedFields = append(sd.OmittedFields, lo.CamelCase(f))
		}

		// Handle generic parent: need to call parent function with schema args
		if parentStruct.IsGeneric() {
			sd.ExtendsParentIsGeneric = true
			// Build list of schema param names from parent's type params
			for _, tp := range parentStruct.TypeParams {
				sd.ExtendsParentSchemaArgs = append(sd.ExtendsParentSchemaArgs, lo.CamelCase(tp.Name))
			}
		}

		// Build map of parent fields for comparison
		parentFields := make(map[string]*resolution.Field)
		for _, pf := range parentStruct.UnifiedFields() {
			parentFields[pf.Name] = pf
		}

		// Categorize child fields into partial vs extend
		for _, field := range entry.Fields {
			parentField, existsInParent := parentFields[field.Name]
			if existsInParent && isOnlyOptionalityChange(parentField.TypeRef, field.TypeRef) {
				// Same base type, just made optional → use .partial()
				sd.PartialFields = append(sd.PartialFields, p.processField(field, entry, table, data, keyFields, sd.UseInput))
			} else {
				// New field or type change → use .extend()
				sd.ExtendFields = append(sd.ExtendFields, p.processField(field, entry, table, data, keyFields, sd.UseInput))
			}
		}
		return sd
	}

	// Non-extending struct: use all fields (flattened for compatibility)
	allFields := entry.UnifiedFields()
	sd.Fields = make([]fieldData, 0, len(allFields))
	for _, field := range allFields {
		sd.Fields = append(sd.Fields, p.processField(field, entry, table, data, keyFields, sd.UseInput))
	}
	return sd
}

// isOnlyOptionalityChange returns true if the child type is the same as the parent type
// except for being optional. This allows using .partial() instead of .extend().
func isOnlyOptionalityChange(parent, child *resolution.TypeRef) bool {
	if parent == nil || child == nil {
		return false
	}
	// Child must be optional (or more optional than parent)
	if !child.IsOptional && !child.IsHardOptional {
		return false
	}
	// If parent is already optional at same level, no change needed
	if parent.IsOptional == child.IsOptional && parent.IsHardOptional == child.IsHardOptional {
		return false
	}
	// Compare base types (ignoring optionality)
	return sameBaseType(parent, child)
}

// sameBaseType compares two TypeRefs ignoring optionality.
func sameBaseType(a, b *resolution.TypeRef) bool {
	if a.Kind != b.Kind {
		return false
	}
	if a.Primitive != b.Primitive {
		return false
	}
	if a.IsArray != b.IsArray {
		return false
	}
	// For struct refs, compare by name
	if a.StructRef != nil && b.StructRef != nil {
		if a.StructRef.QualifiedName != b.StructRef.QualifiedName {
			return false
		}
	} else if a.StructRef != nil || b.StructRef != nil {
		return false
	}
	// For enum refs, compare by name
	if a.EnumRef != nil && b.EnumRef != nil {
		if a.EnumRef.QualifiedName != b.EnumRef.QualifiedName {
			return false
		}
	} else if a.EnumRef != nil || b.EnumRef != nil {
		return false
	}
	return true
}

func (p *Plugin) processTypeParam(tp *resolution.TypeParam, data *templateData) typeParamData {
	tpd := typeParamData{Name: tp.Name, Constraint: "z.ZodType"}
	if tp.Constraint != nil {
		tpd.IsJSON = tp.Constraint.Primitive == "json"
	}
	if tp.Default != nil {
		tpd.HasDefault = true
		tpd.Default = defaultToTS(tp.Default.RawType)
		tpd.DefaultValue = defaultValueToTS(tp.Default.RawType)
	}
	return tpd
}

type typeParamMapping struct {
	zodType  string // e.g., "z.ZodNumber" for type constraints
	zodValue string // e.g., "z.number()" for runtime values
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
	"json":      {zodType: "z.ZodType", zodValue: ""},
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

func fallbackForConstraint(constraint *resolution.TypeRef) string {
	if constraint == nil {
		return "z.unknown()"
	}
	return defaultValueToTS(constraint.Primitive)
}

// isSelfReference checks if a type reference points to the parent struct,
// either directly or through arrays/optionals.
func isSelfReference(t *resolution.TypeRef, parent *resolution.Struct) bool {
	if t == nil || parent == nil {
		return false
	}
	switch t.Kind {
	case resolution.TypeKindStruct:
		if t.StructRef == parent {
			return true
		}
		// Check type arguments for generic recursive references
		for _, arg := range t.TypeArgs {
			if isSelfReference(arg, parent) {
				return true
			}
		}
	}
	return false
}

func (p *Plugin) processField(field *resolution.Field, parentStruct *resolution.Struct, table *resolution.Table, data *templateData, keyFields []keyFieldData, useInput bool) fieldData {
	fd := fieldData{
		Name:           field.Name,
		TSName:         lo.CamelCase(field.Name),
		IsOptional:     field.TypeRef.IsOptional,
		IsHardOptional: field.TypeRef.IsHardOptional,
		IsArray:        field.TypeRef.IsArray,
		IsSelfRef:      isSelfReference(field.TypeRef, parentStruct),
	}
	if typeOverride := getFieldTypeOverride(field, "ts"); typeOverride != "" {
		fd.ZodType = primitiveToZod(typeOverride, data, useInput)
		fd.TSType = primitiveToTS(typeOverride)
		if validateDomain := plugin.GetFieldDomain(field, "validate"); validateDomain != nil {
			fd.ZodType = p.applyValidation(fd.ZodType, validateDomain, field.TypeRef, field.Name)
		}
	} else if _, hasKey := field.Domains["key"]; hasKey {
		if keyField := findKeyField(field.Name, keyFields); keyField != nil {
			fd.ZodType = keyField.TSName + "Z"
			fd.TSType = lo.Capitalize(lo.CamelCase(keyField.Name))
		} else {
			fd.ZodType = p.typeToZod(field.TypeRef, table, data, useInput)
			fd.TSType = p.typeToTS(field.TypeRef, table, data)
		}
	} else {
		fd.ZodType = p.typeToZod(field.TypeRef, table, data, useInput)
		fd.TSType = p.typeToTS(field.TypeRef, table, data)
		if validateDomain := plugin.GetFieldDomain(field, "validate"); validateDomain != nil {
			fd.ZodType = p.applyValidation(fd.ZodType, validateDomain, field.TypeRef, field.Name)
		}
	}
	isAnyOptional := field.TypeRef.IsOptional || field.TypeRef.IsHardOptional
	if field.TypeRef.IsArray {
		addXImport(data, xImport{name: "array", submodule: "array"})
		if isAnyOptional {
			// Optional array: null/undefined → undefined, [] stays []
			fd.ZodType = fmt.Sprintf("array.nullToUndefined(%s)", fd.ZodType)
		} else {
			// Required array: coerce nullish → []
			fd.ZodType = fmt.Sprintf("array.nullishToEmpty(%s)", fd.ZodType)
		}
	} else if isAnyOptional {
		fd.ZodType += ".optional()"
	}
	return fd
}

func findKeyField(name string, keyFields []keyFieldData) *keyFieldData {
	for i := range keyFields {
		if keyFields[i].Name == name {
			return &keyFields[i]
		}
	}
	return nil
}

func getFieldTypeOverride(field *resolution.Field, domainName string) string {
	domain := plugin.GetFieldDomain(field, domainName)
	if domain == nil {
		return ""
	}
	for _, expr := range domain.Expressions {
		if expr.Name == "type" && len(expr.Values) > 0 {
			if v := expr.Values[0].StringValue; v != "" {
				return v
			}
			return expr.Values[0].IdentValue
		}
	}
	return ""
}

func (p *Plugin) typeToZod(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData, useInput bool) string {
	return p.typeToZodInternal(typeRef, table, data, useInput, false)
}

func (p *Plugin) typeToZodInternal(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData, useInput bool, forStructArg bool) string {
	switch typeRef.Kind {
	case resolution.TypeKindPrimitive:
		return primitiveToZod(typeRef.Primitive, data, useInput)
	case resolution.TypeKindTypeParam:
		if typeRef.TypeParamRef != nil {
			paramName := lo.CamelCase(typeRef.TypeParamRef.Name)
			if forStructArg {
				return paramName
			}
			if typeRef.TypeParamRef.Constraint != nil && typeRef.TypeParamRef.Constraint.Primitive == "json" {
				addXImport(data, xImport{name: "zod", submodule: "zod"})
				if useInput {
					return fmt.Sprintf("zod.jsonStringifier(%s)", paramName)
				}
				return fmt.Sprintf("zod.stringifiedJSON(%s)", paramName)
			}
			return fmt.Sprintf("%s ?? %s", paramName, fallbackForConstraint(typeRef.TypeParamRef.Constraint))
		}
		return "z.unknown()"
	case resolution.TypeKindStruct:
		if typeRef.StructRef == nil {
			return "z.unknown()"
		}
		schemaName := lo.CamelCase(typeRef.StructRef.Name) + "Z"
		if typeRef.StructRef.IsGeneric() {
			if len(typeRef.TypeArgs) > 0 {
				structParams := typeRef.StructRef.TypeParams
				args := make([]string, len(typeRef.TypeArgs))
				for i, arg := range typeRef.TypeArgs {
					args[i] = p.typeToZodInternal(arg, table, data, useInput, true)
				}
				if len(structParams) == 1 {
					schemaName = fmt.Sprintf("%s(%s)", schemaName, args[0])
				} else {
					namedArgs := make([]string, len(typeRef.TypeArgs))
					for i, arg := range args {
						namedArgs[i] = fmt.Sprintf("%s: %s", lo.CamelCase(structParams[i].Name), arg)
					}
					schemaName = fmt.Sprintf("%s({%s})", schemaName, strings.Join(namedArgs, ", "))
				}
			} else {
				schemaName += "()"
			}
		}
		if typeRef.StructRef.Namespace != data.Namespace {
			ns := typeRef.StructRef.Namespace
			targetOutputPath := pluginoutput.GetPath(typeRef.StructRef, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.addNamedImport(calculateImportPath(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("%s.%s", ns, schemaName)
		}
		return schemaName
	case resolution.TypeKindEnum:
		if typeRef.EnumRef == nil {
			return "z.unknown()"
		}
		enumName := lo.CamelCase(typeRef.EnumRef.Name) + "Z"
		if typeRef.EnumRef.Namespace != data.Namespace {
			ns := typeRef.EnumRef.Namespace
			targetOutputPath := enum.FindOutputPath(typeRef.EnumRef, table, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.addNamedImport(calculateImportPath(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("%s.%s", ns, enumName)
		}
		return enumName
	case resolution.TypeKindMap:
		keyZ, valueZ := "z.string()", "z.unknown()"
		if typeRef.MapKeyType != nil {
			keyZ = p.typeToZodInternal(typeRef.MapKeyType, table, data, useInput, false)
		}
		if typeRef.MapValueType != nil {
			valueZ = p.typeToZodInternal(typeRef.MapValueType, table, data, useInput, false)
		}
		return fmt.Sprintf("z.record(%s, %s)", keyZ, valueZ)
	default:
		return "z.unknown()"
	}
}

func (p *Plugin) typeToTS(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData) string {
	return p.typeToTSInternal(typeRef, table, data, false)
}

func (p *Plugin) typeToTSInternal(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData, forStructArg bool) string {
	switch typeRef.Kind {
	case resolution.TypeKindPrimitive:
		return primitiveToTS(typeRef.Primitive)
	case resolution.TypeKindTypeParam:
		if typeRef.TypeParamRef != nil {
			if forStructArg {
				return typeRef.TypeParamRef.Name
			}
			return fmt.Sprintf("z.infer<%s>", typeRef.TypeParamRef.Name)
		}
		return "unknown"
	case resolution.TypeKindStruct:
		if typeRef.StructRef == nil {
			return "unknown"
		}
		typeName := typeRef.StructRef.Name
		if typeRef.StructRef.IsGeneric() && len(typeRef.TypeArgs) > 0 {
			args := make([]string, len(typeRef.TypeArgs))
			for i, arg := range typeRef.TypeArgs {
				args[i] = p.typeToTSInternal(arg, table, data, true)
			}
			typeName = fmt.Sprintf("%s<%s>", typeName, strings.Join(args, ", "))
		}
		if typeRef.StructRef.Namespace != data.Namespace {
			return fmt.Sprintf("%s.%s", typeRef.StructRef.Namespace, typeName)
		}
		return typeName
	case resolution.TypeKindEnum:
		if typeRef.EnumRef == nil {
			return "unknown"
		}
		if typeRef.EnumRef.Namespace != data.Namespace {
			return fmt.Sprintf("%s.%s", typeRef.EnumRef.Namespace, typeRef.EnumRef.Name)
		}
		return typeRef.EnumRef.Name
	case resolution.TypeKindMap:
		keyType, valueType := "string", "unknown"
		if typeRef.MapKeyType != nil {
			keyType = p.typeToTSInternal(typeRef.MapKeyType, table, data, forStructArg)
		}
		if typeRef.MapValueType != nil {
			valueType = p.typeToTSInternal(typeRef.MapValueType, table, data, forStructArg)
		}
		return fmt.Sprintf("Record<%s, %s>", keyType, valueType)
	default:
		return "unknown"
	}
}

var primitiveTSTypes = map[string]string{
	"string": "string", "uuid": "string",
	"bool": "boolean",
	"int8": "number", "int16": "number", "int32": "number", "int64": "number",
	"uint8": "number", "uint16": "number", "uint32": "number", "uint64": "number",
	"float32": "number", "float64": "number",
	"timestamp": "number", "timespan": "number",
	"json": "unknown", "bytes": "Uint8Array",
}

func primitiveToTS(primitive string) string {
	if t, ok := primitiveTSTypes[primitive]; ok {
		return t
	}
	return "unknown"
}

type xImport struct {
	name      string // e.g., "TimeStamp", "zod", "array"
	submodule string // e.g., "telem", "zod", "array" - submodule within x package
}

type primitiveMapping struct {
	schema   string
	xImports []xImport // imports needed from @synnaxlabs/x
}

var primitiveZodTypes = map[string]primitiveMapping{
	"uuid":               {schema: "z.uuid()"},
	"string":             {schema: "z.string()"},
	"bool":               {schema: "z.boolean()"},
	"int8":               {schema: "zod.int8Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"int16":              {schema: "zod.int16Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"int32":              {schema: "zod.int32Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"int64":              {schema: "zod.int64Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"uint8":              {schema: "zod.uint8Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"uint16":             {schema: "zod.uint16Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"uint32":             {schema: "zod.uint32Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"uint64":             {schema: "zod.uint64Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"float32":            {schema: "zod.float32Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"float64":            {schema: "zod.float64Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"timestamp":          {schema: "TimeStamp.z", xImports: []xImport{{name: "TimeStamp", submodule: "telem"}}},
	"timespan":           {schema: "TimeSpan.z", xImports: []xImport{{name: "TimeSpan", submodule: "telem"}}},
	"time_range":         {schema: "TimeRange.z", xImports: []xImport{{name: "TimeRange", submodule: "telem"}}},
	"time_range_bounded": {schema: "TimeRange.boundedZ", xImports: []xImport{{name: "TimeRange", submodule: "telem"}}},
	"json":               {schema: "record.unknownZ.or(z.string().transform((s) => JSON.parse(s)))", xImports: []xImport{{name: "record", submodule: "record"}}},
	"bytes":              {schema: "z.instanceof(Uint8Array)"},
}

const xPackageName = "@synnaxlabs/x"
const xPathPrefix = "x/ts/src"

// isInXPackage checks if the output path is within the x package.
func isInXPackage(outputPath string) bool {
	return strings.HasPrefix(outputPath, xPathPrefix)
}

// addXImport adds an import from the x package, using the correct path based on output location.
func addXImport(data *templateData, imp xImport) {
	if isInXPackage(data.OutputPath) {
		// Internal import: @/submodule
		data.addNamedImport("@/"+imp.submodule, imp.name)
	} else {
		// External import: @synnaxlabs/x
		data.addNamedImport(xPackageName, imp.name)
	}
}

func primitiveToZod(primitive string, data *templateData, useInput bool) string {
	// Special handling for JSON based on useInput
	if primitive == "json" {
		addXImport(data, xImport{name: "zod", submodule: "zod"})
		if useInput {
			// For input types, stringify JSON when sending to server
			return "zod.jsonStringifier()"
		}
		// For output types, parse JSON when receiving from server
		return "zod.stringifiedJSON()"
	}
	if mapping, ok := primitiveZodTypes[primitive]; ok {
		for _, imp := range mapping.xImports {
			addXImport(data, imp)
		}
		return mapping.schema
	}
	return "z.unknown()"
}

func (p *Plugin) applyValidation(zodType string, domain *resolution.Domain, typeRef *resolution.TypeRef, fieldName string) string {
	rules := validation.Parse(domain)
	if validation.IsEmpty(rules) {
		return zodType
	}
	isString := typeRef.Kind == resolution.TypeKindPrimitive && resolution.IsStringPrimitive(typeRef.Primitive)
	isNumber := typeRef.Kind == resolution.TypeKindPrimitive && resolution.IsNumberPrimitive(typeRef.Primitive)
	if isString {
		if rules.Required {
			humanName := lo.Capitalize(strings.ReplaceAll(fieldName, "_", " "))
			zodType = fmt.Sprintf("%s.min(1, \"%s is required\")", zodType, humanName)
		}
		if rules.Email {
			zodType += ".email()"
		}
		if rules.URL {
			zodType += ".url()"
		}
		if rules.MinLength != nil {
			zodType = fmt.Sprintf("%s.min(%d)", zodType, *rules.MinLength)
		}
		if rules.MaxLength != nil {
			zodType = fmt.Sprintf("%s.max(%d)", zodType, *rules.MaxLength)
		}
		if rules.Pattern != nil {
			zodType = fmt.Sprintf("%s.regex(/%s/)", zodType, *rules.Pattern)
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
	if rules.Default != nil {
		switch rules.Default.Kind {
		case resolution.ValueKindString:
			zodType = fmt.Sprintf("%s.default(%q)", zodType, rules.Default.StringValue)
		case resolution.ValueKindInt:
			zodType = fmt.Sprintf("%s.default(%d)", zodType, rules.Default.IntValue)
		case resolution.ValueKindFloat:
			zodType = fmt.Sprintf("%s.default(%f)", zodType, rules.Default.FloatValue)
		case resolution.ValueKindBool:
			zodType = fmt.Sprintf("%s.default(%t)", zodType, rules.Default.BoolValue)
		}
	}
	return zodType
}

type templateData struct {
	Namespace, OutputPath string
	Request               *plugin.Request
	KeyFields             []keyFieldData
	Structs               []structData
	Enums                 []enumData
	GenerateTypes         bool
	Imports               map[string]*importSpec
	Ontology              *ontologyData
}

type ontologyData struct {
	TypeName, KeyType, KeyZeroValue string
}

type keyFieldData struct {
	Name, TSName, ZodType, Primitive string
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

type importSpec struct {
	Names map[string]bool
}

func (d *templateData) addNamedImport(path, name string) {
	if d.Imports[path] == nil {
		d.Imports[path] = &importSpec{Names: make(map[string]bool)}
	}
	d.Imports[path].Names[name] = true
}

func (d *templateData) filterImports(filter func(string) bool) []namedImportData {
	var result []namedImportData
	for path, spec := range d.Imports {
		if len(spec.Names) > 0 && filter(path) {
			names := make([]string, 0, len(spec.Names))
			for name := range spec.Names {
				names = append(names, name)
			}
			sort.Strings(names)
			result = append(result, namedImportData{Path: path, Names: names})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Path < result[j].Path })
	return result
}

func (d *templateData) SynnaxImports() []namedImportData {
	return d.filterImports(func(p string) bool { return strings.HasPrefix(p, "@synnaxlabs/") })
}

func (d *templateData) ExternalNamedImports() []namedImportData {
	return d.filterImports(func(p string) bool {
		return !strings.HasPrefix(p, "@/") && !strings.HasPrefix(p, "@synnaxlabs/")
	})
}

func (d *templateData) InternalNamedImports() []namedImportData {
	return d.filterImports(func(p string) bool { return strings.HasPrefix(p, "@/") })
}

type namedImportData struct {
	Path  string
	Names []string
}

type structData struct {
	Name, TSName, AliasOf                          string
	Fields                                         []fieldData
	TypeParams                                     []typeParamData
	UseInput, Handwritten, ConcreteTypes           bool
	IsGeneric, IsSingleParam, IsAlias, IsRecursive bool

	// Extension support
	HasExtends              bool
	ExtendsName             string      // Parent schema name (e.g., "payloadZ")
	ExtendsTypeName         string      // Parent type name (e.g., "Payload")
	ExtendsParentIsGeneric  bool        // True if parent has type params
	ExtendsParentSchemaArgs []string    // Schema param names for calling generic parent (e.g., ["properties", "make", "model"])
	OmittedFields           []string    // Fields omitted from parent via -fieldName
	PartialFields           []fieldData // Fields that only need .partial() (just optionality change)
	ExtendFields            []fieldData // Fields that need .extend() (new fields or type changes)
}

type typeParamData struct {
	Name, Constraint, Default, DefaultValue string
	HasDefault, IsJSON                      bool
}

type fieldData struct {
	Name, TSName, ZodType, TSType                  string
	IsOptional, IsHardOptional, IsArray, IsSelfRef bool
}

type enumData struct {
	Name      string
	Values    []enumValueData
	IsIntEnum bool
}

type enumValueData struct {
	Name, Value string
	IntValue    int64
	IsIntEnum   bool
}

var templateFuncs = template.FuncMap{
	"camelCase":   lo.CamelCase,
	"title":       lo.Capitalize,
	"lower":       strings.ToLower,
	"pluralUpper": func(name string) string { return strings.ToUpper(lo.SnakeCase(name)) + "S" },
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
{{- range .KeyFields }}

export const {{ .TSName }}Z = {{ .ZodType }};
{{- if $.GenerateTypes }}
export type {{ .Name | camelCase | title }} = z.infer<typeof {{ .TSName }}Z>;
{{- end }}
{{- end }}
{{- range .Enums }}

{{- if .IsIntEnum }}
export enum {{ .Name }} {
{{- range $i, $v := .Values }}
  {{ $v.Name }} = {{ $v.IntValue }},
{{- end }}
}
export const {{ camelCase .Name }}Z = z.enum({{ .Name }});
{{- else }}
export const {{ pluralUpper .Name }} = [{{ range $i, $v := .Values }}{{ if $i }}, {{ end }}"{{ $v.Value }}"{{ end }}] as const;
export const {{ camelCase .Name }}Z = z.enum([...{{ pluralUpper .Name }}]);
{{- end }}
{{- if and $.GenerateTypes (not .IsIntEnum) }}
export type {{ .Name }} = z.infer<typeof {{ camelCase .Name }}Z>;
{{- end }}
{{- end }}
{{- range .Structs }}
{{- if not .Handwritten }}
{{- if .IsAlias }}
{{- if .IsGeneric }}
{{- if .IsSingleParam }}

export const {{ camelCase .TSName }}Z = <{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>({{ range $i, $p := .TypeParams }}{{ $p.Name | camelCase }}?: {{ $p.Name }}{{ end }}) =>
  {{ .AliasOf }};
{{- if $.GenerateTypes }}
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = z.{{ if .UseInput }}input{{ else }}infer{{ end }}<
  ReturnType<typeof {{ camelCase .TSName }}Z<{{ range $i, $p := .TypeParams }}{{ $p.Name }}{{ end }}>>
>;
{{- end }}
{{- else }}

export interface {{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> {
{{- range $i, $p := .TypeParams }}
  {{ $p.Name | camelCase }}?: {{ $p.Name }};
{{- end }}
}

export const {{ camelCase .TSName }}Z = <{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>({
{{- range $i, $p := .TypeParams }}
  {{ $p.Name | camelCase }},
{{- end }}
}: {{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}> = {}) =>
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
{{- else if .IsGeneric }}
{{- if .IsSingleParam }}

export const {{ camelCase .TSName }}Z = <{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>({{ range $i, $p := .TypeParams }}{{ $p.Name | camelCase }}?: {{ $p.Name }}{{ end }}) =>
{{- if .HasExtends }}
  {{ .ExtendsName }}({{ if .ExtendsParentIsGeneric }}{{ range $i, $a := .ExtendsParentSchemaArgs }}{{ if $i }}, {{ end }}{{ $a }}{{ end }}{{ end }})
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
      get {{ .TSName }}() {
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
    get {{ .TSName }}() {
      return {{ .ZodType }};
    },
{{- else }}
    {{ .TSName }}: {{ .ZodType }},
{{- end }}
{{- end }}
  });
{{- end }}
{{- else }}

export interface {{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> {
{{- range $i, $p := .TypeParams }}
  {{ $p.Name | camelCase }}?: {{ $p.Name }};
{{- end }}
}

export const {{ camelCase .TSName }}Z = <{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>({
{{- range $i, $p := .TypeParams }}
  {{ $p.Name | camelCase }},
{{- end }}
}: {{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}> = {}) =>
{{- if .HasExtends }}
  {{ .ExtendsName }}({{ if .ExtendsParentIsGeneric }}{ {{ range $i, $a := .ExtendsParentSchemaArgs }}{{ if $i }}, {{ end }}{{ $a }}{{ end }} }{{ end }})
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
      get {{ .TSName }}() {
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
    get {{ .TSName }}() {
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
export interface {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> extends Omit<{{ .ExtendsTypeName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>, {{ range $i, $f := .OmittedFields }}{{ if $i }} | {{ end }}"{{ $f }}"{{ end }}{{ if and .OmittedFields (or .PartialFields .ExtendFields) }} | {{ end }}{{ range $i, $f := .PartialFields }}{{ if $i }} | {{ end }}"{{ $f.TSName }}"{{ end }}{{ if and .PartialFields .ExtendFields }} | {{ end }}{{ range $i, $f := .ExtendFields }}{{ if $i }} | {{ end }}"{{ $f.TSName }}"{{ end }}> {
{{- range .PartialFields }}
  {{ .TSName }}?: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
{{- range .ExtendFields }}
  {{ .TSName }}{{ if or .IsOptional .IsHardOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}
{{- else }}
export interface {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> {
{{- range .Fields }}
  {{ .TSName }}{{ if or .IsOptional .IsHardOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}
{{- end }}
{{- else }}
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = z.{{ if .UseInput }}input{{ else }}infer{{ end }}<
  ReturnType<typeof {{ camelCase .TSName }}Z<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>>
>;
{{- end }}
{{- end }}
{{- else if .HasExtends }}

export const {{ camelCase .TSName }}Z = {{ .ExtendsName }}
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
    get {{ .TSName }}() {
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

export const {{ camelCase .TSName }}Z = z.object({
{{- range .Fields }}
{{- if .IsSelfRef }}
  get {{ .TSName }}() {
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
{{- if .Ontology }}

export const ontologyID = ontology.createIDFactory<{{ .Ontology.KeyType }}>("{{ .Ontology.TypeName }}");
export const TYPE_ONTOLOGY_ID = ontologyID({{ .Ontology.KeyZeroValue }});
{{- end }}
`))
