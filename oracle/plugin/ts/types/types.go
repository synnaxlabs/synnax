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
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
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

// PostWrite runs eslint --fix on the generated TypeScript files to sort imports and format.
// Files are grouped by their package directory (containing package.json).
func (p *Plugin) PostWrite(files []string) error {
	if len(files) == 0 {
		return nil
	}
	// Group files by their package directory
	byPackage := make(map[string][]string)
	for _, file := range files {
		pkgDir := findPackageDir(file)
		if pkgDir != "" {
			byPackage[pkgDir] = append(byPackage[pkgDir], file)
		}
	}
	// Run eslint --fix for each package
	for pkgDir, pkgFiles := range byPackage {
		args := append([]string{"eslint", "--fix"}, pkgFiles...)
		cmd := exec.Command("npx", args...)
		cmd.Dir = pkgDir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("eslint failed in %s: %w", pkgDir, err)
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

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	outputStructs := make(map[string][]*resolution.StructEntry)
	for _, entry := range req.Resolutions.AllStructs() {
		if outputPath := output.GetPath(entry, "ts"); outputPath != "" {
			if req.RepoRoot != "" {
				if err := req.ValidateOutputPath(outputPath); err != nil {
					return nil, fmt.Errorf("invalid output path for struct %s: %w", entry.Name, err)
				}
			}
			outputStructs[outputPath] = append(outputStructs[outputPath], entry)
		}
	}
	for outputPath, structs := range outputStructs {
		enums := enum.CollectReferenced(structs)
		content, err := p.generateFile(structs[0].Namespace, outputPath, structs, enums, req)
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

// calculateImportPath determines the correct import path for cross-namespace references.
// Returns the import path and whether to use a named import (vs namespace import).
func calculateImportPath(fromPath, toPath string) string {
	fromPkg := findPackage(fromPath)
	toPkg := findPackage(toPath)

	if fromPkg == nil || toPkg == nil {
		// Fallback to relative import if packages not recognized
		return calculateRelativeImport(fromPath, toPath)
	}

	if fromPkg.packageName == toPkg.packageName {
		// Same package - use internal absolute import
		// e.g., "client/ts/src/ranger" importing "client/ts/src/label" → "@/label"
		relativePath := strings.TrimPrefix(toPath, toPkg.pathPrefix)
		relativePath = strings.TrimPrefix(relativePath, "/")
		return toPkg.internalPrefix + relativePath
	}

	// Different package - use package name
	// e.g., "client/ts/src/ranger" importing "x/ts/src/label" → "@synnaxlabs/x"
	// The module path within the package (e.g., "/label") is handled by the package's exports
	return toPkg.packageName
}

// calculateRelativeImport calculates the relative import path from one output
// directory to another (fallback for unknown packages).
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
	structs []*resolution.StructEntry,
	enums []*resolution.EnumEntry,
	req *plugin.Request,
) ([]byte, error) {
	data := &templateData{
		Namespace:     namespace,
		OutputPath:    outputPath,
		Request:       req,
		IDFields:      make([]idFieldData, 0),
		Structs:       make([]structData, 0, len(structs)),
		Enums:         make([]enumData, 0, len(enums)),
		GenerateTypes: p.Options.GenerateTypes,
		Imports:       make(map[string]*importSpec),
	}
	idFields := collectIDFields(structs, data)
	data.IDFields = idFields
	data.Ontology = extractOntology(structs, idFields)
	if data.Ontology != nil {
		data.addNamedImport("@/ontology", "ontology")
	}

	for _, enum := range enums {
		data.Enums = append(data.Enums, p.processEnum(enum))
	}
	for _, entry := range structs {
		data.Structs = append(data.Structs, p.processStruct(entry, req.Resolutions, data, idFields))
	}
	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func collectIDFields(structs []*resolution.StructEntry, data *templateData) []idFieldData {
	seen := make(map[string]bool)
	var result []idFieldData
	for _, s := range structs {
		// Skip handwritten structs
		if isHandwritten(s) {
			continue
		}
		for _, f := range s.Fields {
			if _, hasID := f.Domains["id"]; hasID {
				if !seen[f.Name] {
					seen[f.Name] = true
					result = append(result, idFieldData{
						Name:      f.Name,
						TSName:    lo.CamelCase(f.Name),
						ZodType:   primitiveToZod(f.TypeRef.Primitive, data, false),
						Primitive: f.TypeRef.Primitive,
					})
				}
			}
		}
	}
	return result
}

// isHandwritten checks if a struct has the "handwritten" expression in its ts domain.
func isHandwritten(s *resolution.StructEntry) bool {
	if tsDomain, ok := s.Domains["ts"]; ok {
		for _, expr := range tsDomain.Expressions {
			if expr.Name == "handwritten" {
				return true
			}
		}
	}
	return false
}

// extractOntology extracts ontology data from structs if any has the ontology domain.
// Returns nil if no struct has an ontology domain or if no ID field is found.
func extractOntology(structs []*resolution.StructEntry, idFields []idFieldData) *ontologyData {
	for _, s := range structs {
		// Skip handwritten structs
		if isHandwritten(s) {
			continue
		}
		domain, ok := s.Domains["ontology"]
		if !ok {
			continue
		}

		// Find the type expression
		var typeName string
		for _, expr := range domain.Expressions {
			if expr.Name == "type" && len(expr.Values) > 0 {
				typeName = expr.Values[0].StringValue
				break
			}
		}
		if typeName == "" {
			continue
		}

		// Find the ID field's TypeScript type name (capitalized) and zero value
		if len(idFields) == 0 {
			continue
		}
		idField := idFields[0]
		keyType := lo.Capitalize(lo.CamelCase(idField.Name))
		keyZeroValue := primitiveZeroValue(idField.Primitive)

		return &ontologyData{
			TypeName:     typeName,
			KeyType:      keyType,
			KeyZeroValue: keyZeroValue,
		}
	}
	return nil
}

func (p *Plugin) processEnum(enum *resolution.EnumEntry) enumData {
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

func (p *Plugin) processStruct(entry *resolution.StructEntry, table *resolution.Table, data *templateData, idFields []idFieldData) structData {
	sd := structData{
		Name:          entry.Name,
		TSName:        entry.Name,
		Fields:        make([]fieldData, 0, len(entry.Fields)),
		IsGeneric:     entry.IsGeneric(),
		IsSingleParam: len(entry.TypeParams) == 1,
		IsAlias:       entry.IsAlias(),
	}
	// Check if ts domain has use_input, handwritten, concrete_types, or name expressions
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
	// Skip processing fields for handwritten structs (no schema/types generated, no imports needed)
	if sd.Handwritten {
		return sd
	}
	// Process type parameters for generic structs (must happen before alias check)
	for _, tp := range entry.TypeParams {
		sd.TypeParams = append(sd.TypeParams, p.processTypeParam(tp, data))
	}
	// Handle struct aliases: generate the aliased type expression
	if entry.IsAlias() {
		sd.AliasOf = p.typeToZod(entry.AliasOf, table, data, sd.UseInput)
		return sd
	}
	for _, field := range entry.Fields {
		sd.Fields = append(sd.Fields, p.processField(field, table, data, idFields, sd.UseInput))
	}
	return sd
}

// processTypeParam converts a resolution.TypeParam to template data.
func (p *Plugin) processTypeParam(tp *resolution.TypeParam, data *templateData) typeParamData {
	tpd := typeParamData{Name: tp.Name}
	// Map constraint to TypeScript
	if tp.Constraint != nil {
		tpd.Constraint = constraintToTS(tp.Constraint.RawType)
		tpd.IsJSON = tp.Constraint.Primitive == "json"
	} else {
		tpd.Constraint = "z.ZodType" // Default constraint
	}
	// Map default to TypeScript (both type and runtime value)
	if tp.Default != nil {
		tpd.HasDefault = true
		tpd.Default = defaultToTS(tp.Default.RawType)
		tpd.DefaultValue = defaultValueToTS(tp.Default.RawType)
	}
	return tpd
}

// constraintToTS maps Oracle constraint types to TypeScript.
func constraintToTS(rawType string) string {
	switch rawType {
	case "schema":
		return "z.ZodType"
	default:
		return "z.ZodType"
	}
}

// defaultToTS maps Oracle default types to TypeScript Zod type names.
func defaultToTS(rawType string) string {
	switch rawType {
	case "never":
		return "z.ZodNever"
	case "string":
		return "z.ZodString"
	case "bool":
		return "z.ZodBoolean"
	case "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return "z.ZodNumber"
	case "json":
		return "z.ZodType"
	case "uuid":
		return "z.ZodString"
	case "timestamp", "timespan":
		return "z.ZodNumber"
	default:
		return "z.ZodType"
	}
}

// defaultValueToTS maps Oracle default types to TypeScript Zod runtime schema values.
// This is used for default parameter values in destructuring (e.g., { make = z.string() }).
func defaultValueToTS(rawType string) string {
	switch rawType {
	case "string":
		return "z.string()"
	case "bool":
		return "z.boolean()"
	case "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return "z.number()"
	case "uuid":
		return "z.string()"
	case "json":
		return "" // json params don't get a default value, they're wrapped with stringifiedJSON
	default:
		return "z.unknown()"
	}
}

// fallbackForConstraint returns the Zod schema fallback value for a type parameter constraint.
// This is used when the caller doesn't provide a schema for the type parameter.
func fallbackForConstraint(constraint *resolution.TypeRef) string {
	if constraint == nil {
		return "z.unknown()"
	}
	switch constraint.Primitive {
	case "string":
		return "z.string()"
	case "bool":
		return "z.boolean()"
	case "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return "z.number()"
	case "uuid":
		return "z.string()"
	case "json":
		// json constraints are handled specially in typeToZod, but provide fallback just in case
		return "z.unknown()"
	default:
		return "z.unknown()"
	}
}

func (p *Plugin) processField(field *resolution.FieldEntry, table *resolution.Table, data *templateData, idFields []idFieldData, useInput bool) fieldData {
	fd := fieldData{
		Name:       field.Name,
		TSName:     lo.CamelCase(field.Name),
		IsOptional: field.TypeRef.IsOptional,
		IsNullable: field.TypeRef.IsNullable,
		IsArray:    field.TypeRef.IsArray,
	}
	if idField := findIDField(field.Name, idFields); idField != nil {
		fd.ZodType = idField.TSName + "Z"
		fd.TSType = lo.Capitalize(lo.CamelCase(idField.Name)) // e.g., "Key"
	} else {
		fd.ZodType = p.typeToZod(field.TypeRef, table, data, useInput)
		fd.TSType = p.typeToTS(field.TypeRef, table, data)
		if validateDomain := plugin.GetFieldDomain(field, "validate"); validateDomain != nil {
			fd.ZodType = p.applyValidation(fd.ZodType, validateDomain, field.TypeRef, field.Name)
		}
	}

	if field.TypeRef.IsArray {
		if field.TypeRef.IsNullable {
			// array.nullableZ transforms null/undefined to empty array
			addXImport(data, xImport{name: "array", submodule: "array"})
			fd.ZodType = fmt.Sprintf("array.nullableZ(%s)", fd.ZodType)
			// If also optional, add .optional() after
			if field.TypeRef.IsOptional {
				fd.ZodType = fd.ZodType + ".optional()"
			}
		} else {
			fd.ZodType = fmt.Sprintf("z.array(%s)", fd.ZodType)
			if field.TypeRef.IsOptional {
				fd.ZodType = fd.ZodType + ".optional()"
			}
		}
	} else {
		if field.TypeRef.IsOptional && field.TypeRef.IsNullable {
			fd.ZodType = fd.ZodType + ".nullish()"
		} else if field.TypeRef.IsNullable {
			fd.ZodType = fd.ZodType + ".nullable()"
		} else if field.TypeRef.IsOptional {
			fd.ZodType = fd.ZodType + ".optional()"
		}
	}
	return fd
}

func findIDField(name string, idFields []idFieldData) *idFieldData {
	for i := range idFields {
		if idFields[i].Name == name {
			return &idFields[i]
		}
	}
	return nil
}

func (p *Plugin) typeToZod(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData, useInput bool) string {
	return p.typeToZodInternal(typeRef, table, data, useInput, false)
}

// typeToZodInternal generates the Zod schema expression for a type reference.
// forStructArg indicates if this is being generated as an argument to pass to another struct's schema function.
// When true, we don't wrap json type params - the wrapping happens inside the called struct.
func (p *Plugin) typeToZodInternal(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData, useInput bool, forStructArg bool) string {
	switch typeRef.Kind {
	case resolution.TypeKindPrimitive:
		return primitiveToZod(typeRef.Primitive, data, useInput)
	case resolution.TypeKindTypeParam:
		// Type parameter reference (e.g., field details D? in struct Status<D>)
		if typeRef.TypeParamRef != nil {
			paramName := lo.CamelCase(typeRef.TypeParamRef.Name)
			// When passing as arg to another struct, don't wrap - the struct handles wrapping
			if forStructArg {
				return paramName
			}
			// Check if the constraint is json - if so, wrap with stringifiedJSON/jsonStringifier
			if typeRef.TypeParamRef.Constraint != nil && typeRef.TypeParamRef.Constraint.Primitive == "json" {
				addXImport(data, xImport{name: "zod", submodule: "zod"})
				if useInput {
					return fmt.Sprintf("zod.jsonStringifier(%s)", paramName)
				}
				return fmt.Sprintf("zod.stringifiedJSON(%s)", paramName)
			}
			// For non-json type params, use ?? with fallback based on constraint
			fallback := fallbackForConstraint(typeRef.TypeParamRef.Constraint)
			return fmt.Sprintf("%s ?? %s", paramName, fallback)
		}
		return "z.unknown()"
	case resolution.TypeKindStruct:
		if typeRef.StructRef == nil {
			return "z.unknown()"
		}
		schemaName := lo.CamelCase(typeRef.StructRef.Name) + "Z"
		// Handle generic struct references
		if typeRef.StructRef.IsGeneric() {
			if len(typeRef.TypeArgs) > 0 {
				structParams := typeRef.StructRef.TypeParams
				isSingleParam := len(structParams) == 1
				// Generate type arguments, passing forStructArg=true to avoid double-wrapping
				args := make([]string, len(typeRef.TypeArgs))
				for i, arg := range typeRef.TypeArgs {
					args[i] = p.typeToZodInternal(arg, table, data, useInput, true)
				}
				if isSingleParam {
					// Single param: positional argument
					schemaName = fmt.Sprintf("%s(%s)", schemaName, args[0])
				} else {
					// Multiple params: object with named properties
					namedArgs := make([]string, len(typeRef.TypeArgs))
					for i, arg := range args {
						paramName := lo.CamelCase(structParams[i].Name)
						namedArgs[i] = fmt.Sprintf("%s: %s", paramName, arg)
					}
					schemaName = fmt.Sprintf("%s({%s})", schemaName, strings.Join(namedArgs, ", "))
				}
			} else {
				// Generic struct without type args: call with no arguments
				schemaName = schemaName + "()"
			}
		}
		if typeRef.StructRef.Namespace != data.Namespace {
			ns := typeRef.StructRef.Namespace
			// Get target output path from the referenced struct's domain
			targetOutputPath := output.GetPath(typeRef.StructRef, "ts")
			if targetOutputPath == "" {
				// Fallback if no output path defined
				targetOutputPath = ns
			}
			// Calculate import path using package-aware logic
			importPath := calculateImportPath(data.OutputPath, targetOutputPath)
			// Use named import with namespace as the import name
			data.addNamedImport(importPath, ns)
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
			// For enums, look up the struct that references this enum to find its output path
			// Or use namespace as fallback
			targetOutputPath := enum.FindOutputPath(typeRef.EnumRef, table, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			// Calculate import path using package-aware logic
			importPath := calculateImportPath(data.OutputPath, targetOutputPath)
			// Use named import with namespace as the import name
			data.addNamedImport(importPath, ns)
			return fmt.Sprintf("%s.%s", ns, enumName)
		}
		return enumName
	default:
		return "z.unknown()"
	}
}

// typeToTS generates the concrete TypeScript type for a type reference.
// Used when ConcreteTypes is enabled to generate explicit type definitions.
func (p *Plugin) typeToTS(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData) string {
	return p.typeToTSInternal(typeRef, table, data, false)
}

// typeToTSInternal generates the concrete TypeScript type.
// forStructArg indicates this is being used as a type argument to another struct,
// in which case type params should be passed as-is (not wrapped in z.infer).
func (p *Plugin) typeToTSInternal(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData, forStructArg bool) string {
	switch typeRef.Kind {
	case resolution.TypeKindPrimitive:
		return primitiveToTS(typeRef.Primitive)
	case resolution.TypeKindTypeParam:
		if typeRef.TypeParamRef != nil {
			// When passing to another struct, pass the ZodType directly
			if forStructArg {
				return typeRef.TypeParamRef.Name
			}
			// Type params are Zod schemas, so we need z.infer to get the value type
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
				// Pass true for forStructArg since these are arguments to a struct
				args[i] = p.typeToTSInternal(arg, table, data, true)
			}
			typeName = fmt.Sprintf("%s<%s>", typeName, strings.Join(args, ", "))
		}
		if typeRef.StructRef.Namespace != data.Namespace {
			ns := typeRef.StructRef.Namespace
			return fmt.Sprintf("%s.%s", ns, typeName)
		}
		return typeName
	case resolution.TypeKindEnum:
		if typeRef.EnumRef == nil {
			return "unknown"
		}
		enumName := typeRef.EnumRef.Name
		if typeRef.EnumRef.Namespace != data.Namespace {
			ns := typeRef.EnumRef.Namespace
			return fmt.Sprintf("%s.%s", ns, enumName)
		}
		return enumName
	default:
		return "unknown"
	}
}

// primitiveToTS converts a primitive type to its TypeScript equivalent.
func primitiveToTS(primitive string) string {
	switch primitive {
	case "string", "uuid":
		return "string"
	case "bool":
		return "boolean"
	case "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64", "float32", "float64":
		return "number"
	case "timestamp", "timespan":
		return "number"
	case "json":
		return "unknown"
	case "bytes":
		return "Uint8Array"
	default:
		return "unknown"
	}
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

func (p *Plugin) applyValidation(zodType string, domain *resolution.DomainEntry, typeRef *resolution.TypeRef, fieldName string) string {
	isString := typeRef.Kind == resolution.TypeKindPrimitive && resolution.IsStringPrimitive(typeRef.Primitive)
	isNumber := typeRef.Kind == resolution.TypeKindPrimitive && resolution.IsNumberPrimitive(typeRef.Primitive)
	for _, expr := range domain.Expressions {
		if len(expr.Values) == 0 {
			switch expr.Name {
			case "required":
				if isString {
					// Generate human-readable field name from snake_case
					humanName := lo.Capitalize(strings.ReplaceAll(fieldName, "_", " "))
					zodType = fmt.Sprintf("%s.min(1, \"%s is required\")", zodType, humanName)
				}
			case "email":
				if isString {
					zodType += ".email()"
				}
			case "url":
				if isString {
					zodType += ".url()"
				}
			}
			continue
		}
		v := expr.Values[0]
		switch expr.Name {
		case "min_length":
			if isString {
				zodType = fmt.Sprintf("%s.min(%d)", zodType, v.IntValue)
			}
		case "max_length":
			if isString {
				zodType = fmt.Sprintf("%s.max(%d)", zodType, v.IntValue)
			}
		case "pattern":
			if isString {
				zodType = fmt.Sprintf("%s.regex(/%s/)", zodType, v.StringValue)
			}
		case "min":
			if isNumber {
				if v.Kind == resolution.ValueKindInt {
					zodType = fmt.Sprintf("%s.min(%d)", zodType, v.IntValue)
				} else {
					zodType = fmt.Sprintf("%s.min(%f)", zodType, v.FloatValue)
				}
			}
		case "max":
			if isNumber {
				if v.Kind == resolution.ValueKindInt {
					zodType = fmt.Sprintf("%s.max(%d)", zodType, v.IntValue)
				} else {
					zodType = fmt.Sprintf("%s.max(%f)", zodType, v.FloatValue)
				}
			}
		case "default":
			switch v.Kind {
			case resolution.ValueKindString:
				zodType = fmt.Sprintf("%s.default(%q)", zodType, v.StringValue)
			case resolution.ValueKindInt:
				zodType = fmt.Sprintf("%s.default(%d)", zodType, v.IntValue)
			case resolution.ValueKindFloat:
				zodType = fmt.Sprintf("%s.default(%f)", zodType, v.FloatValue)
			case resolution.ValueKindBool:
				zodType = fmt.Sprintf("%s.default(%t)", zodType, v.BoolValue)
			}
		}
	}
	return zodType
}

type templateData struct {
	Namespace     string
	OutputPath    string          // Current output path for calculating relative imports
	Request       *plugin.Request // Request for access to helper methods
	IDFields      []idFieldData
	Structs       []structData
	Enums         []enumData
	GenerateTypes bool
	Imports       map[string]*importSpec
	Ontology      *ontologyData // Ontology data if domain ontology is present
}

// ontologyData holds data for generating ontology ID factory and constant.
type ontologyData struct {
	TypeName     string // e.g., "user" - from domain ontology { type "user" }
	KeyType      string // e.g., "Key" - derived from the ID field
	KeyZeroValue string // e.g., "\"00000000-0000-0000-0000-000000000000\"" for UUID, "0" for uint32
}

type idFieldData struct {
	Name      string // e.g., "key"
	TSName    string // e.g., "key"
	ZodType   string // e.g., "z.uuid()"
	Primitive string // e.g., "uuid", "uint32" - for zero value computation
}

// primitiveZeroValue returns the TypeScript zero value literal for a primitive type.
// For ontology IDs, uuid uses empty string since ontologyID("") represents the type itself.
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
	Names map[string]bool // named imports (e.g., "zod", "TimeStamp", "label")
}

func (d *templateData) addNamedImport(path, name string) {
	if d.Imports[path] == nil {
		d.Imports[path] = &importSpec{Names: make(map[string]bool)}
	}
	d.Imports[path].Names[name] = true
}

func (d *templateData) NamedImports() []namedImportData {
	var result []namedImportData
	for path, spec := range d.Imports {
		if spec.Names != nil && len(spec.Names) > 0 {
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

// SynnaxImports returns named imports from @synnaxlabs/* packages
func (d *templateData) SynnaxImports() []namedImportData {
	var result []namedImportData
	for _, imp := range d.NamedImports() {
		if strings.HasPrefix(imp.Path, "@synnaxlabs/") {
			result = append(result, imp)
		}
	}
	return result
}

// ExternalNamedImports returns named imports from external packages (not @/ or @synnaxlabs/)
func (d *templateData) ExternalNamedImports() []namedImportData {
	var result []namedImportData
	for _, imp := range d.NamedImports() {
		if !strings.HasPrefix(imp.Path, "@/") && !strings.HasPrefix(imp.Path, "@synnaxlabs/") {
			result = append(result, imp)
		}
	}
	return result
}

// InternalNamedImports returns named imports from internal packages (starting with @/)
func (d *templateData) InternalNamedImports() []namedImportData {
	var result []namedImportData
	for _, imp := range d.NamedImports() {
		if strings.HasPrefix(imp.Path, "@/") {
			result = append(result, imp)
		}
	}
	return result
}

type namedImportData struct {
	Path  string
	Names []string
}

type structData struct {
	Name          string // Original struct name from schema
	TSName        string // TypeScript name (can be overridden via ts domain { name "..." })
	Fields        []fieldData
	UseInput      bool            // If true, use z.input instead of z.infer for type derivation
	Handwritten   bool            // If true, skip generating both schema and type (struct is handwritten)
	ConcreteTypes bool            // If true, generate explicit type definitions instead of z.infer
	TypeParams    []typeParamData // Generic type parameters
	IsGeneric     bool            // True if struct has type parameters
	IsSingleParam bool            // True if struct has exactly one type parameter (use positional arg)
	IsAlias       bool            // True if struct is an alias of another type
	AliasOf       string          // TypeScript expression for the aliased type schema
}

// typeParamData holds data for a generic type parameter.
type typeParamData struct {
	Name         string // e.g., "D"
	Constraint   string // e.g., "z.ZodType" (TypeScript constraint)
	Default      string // e.g., "z.ZodNever" (TypeScript default type)
	DefaultValue string // e.g., "z.string()" (runtime default value for destructuring)
	HasDefault   bool   // True if a default value is specified
	IsJSON       bool   // True if the constraint is json (needs special wrapping)
}

type fieldData struct {
	Name       string
	TSName     string
	ZodType    string
	TSType     string // Concrete TypeScript type (e.g., "z.infer<Type>", "string")
	IsOptional bool
	IsNullable bool
	IsArray    bool
}

type enumData struct {
	Name      string
	Values    []enumValueData
	IsIntEnum bool
}

type enumValueData struct {
	Name      string
	Value     string
	IntValue  int64
	IsIntEnum bool
}

var templateFuncs = template.FuncMap{
	"camelCase": lo.CamelCase,
	"title":     lo.Capitalize,
	"lower":     strings.ToLower,
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
{{- range .IDFields }}

export const {{ .TSName }}Z = {{ .ZodType }};
{{- if $.GenerateTypes }}
export type {{ .Name | camelCase | title }} = z.infer<typeof {{ .TSName }}Z>;
{{- end }}
{{- end }}
{{- range .Enums }}

{{- if .IsIntEnum }}
export const {{ camelCase .Name }}Values = [{{ range $i, $v := .Values }}{{ if $i }}, {{ end }}{{ $v.IntValue }}{{ end }}] as const;
export const {{ camelCase .Name }}Z = z.enum([{{ range $i, $v := .Values }}{{ if $i }}, {{ end }}"{{ $v.Name }}"{{ end }}]);
{{- else }}
export const {{ camelCase .Name }}Z = z.enum([{{ range $i, $v := .Values }}{{ if $i }}, {{ end }}"{{ $v.Value }}"{{ end }}]);
{{- end }}
{{- if $.GenerateTypes }}
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
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = z.infer<
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
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = z.infer<
  ReturnType<typeof {{ camelCase .TSName }}Z<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>>
>;
{{- end }}
{{- end }}
{{- else }}

export const {{ camelCase .TSName }}Z = {{ .AliasOf }};
{{- if $.GenerateTypes }}
export type {{ .TSName }} = z.infer<typeof {{ camelCase .TSName }}Z>;
{{- end }}
{{- end }}
{{- else if .IsGeneric }}
{{- if .IsSingleParam }}

export const {{ camelCase .TSName }}Z = <{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>({{ range $i, $p := .TypeParams }}{{ $p.Name | camelCase }}?: {{ $p.Name }}{{ end }}) =>
  z.object({
{{- range .Fields }}
    {{ .TSName }}: {{ .ZodType }},
{{- end }}
  });
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
  z.object({
{{- range .Fields }}
    {{ .TSName }}: {{ .ZodType }},
{{- end }}
  });
{{- end }}
{{- if $.GenerateTypes }}
{{- if .ConcreteTypes }}
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = {
{{- range .Fields }}
  {{ .TSName }}{{ if .IsOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }}{{ if .IsNullable }} | null{{ end }};
{{- end }}
};
{{- else }}
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = z.infer<
  ReturnType<typeof {{ camelCase .TSName }}Z<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>>
>;
{{- end }}
{{- end }}
{{- else }}

export const {{ camelCase .TSName }}Z = z.object({
{{- range .Fields }}
  {{ .TSName }}: {{ .ZodType }},
{{- end }}
});
{{- if $.GenerateTypes }}
export type {{ .TSName }} = z.{{ if .UseInput }}input{{ else }}infer{{ end }}<typeof {{ camelCase .TSName }}Z>;
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- if .Ontology }}

export const ontologyID = ontology.createIDFactory<{{ .Ontology.KeyType }}>("{{ .Ontology.TypeName }}");
export const TYPE_ONTOLOGY_ID = ontologyID({{ .Ontology.KeyZeroValue }});
{{- end }}
`))
