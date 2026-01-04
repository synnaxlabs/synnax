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
	"github.com/synnaxlabs/oracle/domain/key"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/domain/ontology"
	"github.com/synnaxlabs/oracle/domain/validation"
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
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
	var (
		resp           = &plugin.Response{Files: make([]plugin.File, 0)}
		outputStructs  = make(map[string][]resolution.Type)
		outputEnums    = make(map[string][]resolution.Type)
		outputTypeDefs = make(map[string][]resolution.Type)
		structOrder    []string
		enumOrder      []string
		typeDefOrder   []string
	)

	for _, entry := range req.Resolutions.DistinctTypes() {
		if outputPath := output.GetPath(entry, "ts"); outputPath != "" {
			if omit.IsType(entry, "ts") {
				continue
			}
			if req.RepoRoot != "" {
				if err := req.ValidateOutputPath(outputPath); err != nil {
					return nil, errors.Wrapf(err, "invalid output path for typedef %s", entry.Name)
				}
			}
			if _, exists := outputTypeDefs[outputPath]; !exists {
				typeDefOrder = append(typeDefOrder, outputPath)
			}
			outputTypeDefs[outputPath] = append(outputTypeDefs[outputPath], entry)
		}
	}
	for _, entry := range req.Resolutions.AliasTypes() {
		if outputPath := output.GetPath(entry, "ts"); outputPath != "" {
			if omit.IsType(entry, "ts") {
				continue
			}
			if req.RepoRoot != "" {
				if err := req.ValidateOutputPath(outputPath); err != nil {
					return nil, errors.Wrapf(err, "invalid output path for alias %s", entry.Name)
				}
			}
			if _, exists := outputTypeDefs[outputPath]; !exists {
				typeDefOrder = append(typeDefOrder, outputPath)
			}
			outputTypeDefs[outputPath] = append(outputTypeDefs[outputPath], entry)
		}
	}

	for _, entry := range req.Resolutions.StructTypes() {
		if outputPath := output.GetPath(entry, "ts"); outputPath != "" {
			if req.RepoRoot != "" {
				if err := req.ValidateOutputPath(outputPath); err != nil {
					return nil, errors.Wrapf(err, "invalid output path for struct %s", entry.Name)
				}
			}
			if _, exists := outputStructs[outputPath]; !exists {
				structOrder = append(structOrder, outputPath)
			}
			outputStructs[outputPath] = append(outputStructs[outputPath], entry)
		}
	}
	for _, e := range enum.CollectWithOwnOutput(req.Resolutions.EnumTypes(), "ts") {
		enumPath := output.GetPath(e, "ts")
		if req.RepoRoot != "" {
			if err := req.ValidateOutputPath(enumPath); err != nil {
				return nil, errors.Wrapf(err, "invalid output path for enum %s", e.Name)
			}
		}
		if _, exists := outputEnums[enumPath]; !exists {
			enumOrder = append(enumOrder, enumPath)
		}
		outputEnums[enumPath] = append(outputEnums[enumPath], e)
	}
	for _, outputPath := range structOrder {
		structs := outputStructs[outputPath]
		enums := enum.CollectReferenced(structs, req.Resolutions)
		if standaloneEnums, ok := outputEnums[outputPath]; ok {
			enums = mergeEnums(enums, standaloneEnums)
			delete(outputEnums, outputPath)
		}
		var typeDefs []resolution.Type
		if tds, ok := outputTypeDefs[outputPath]; ok {
			typeDefs = tds
			delete(outputTypeDefs, outputPath)
		}
		content, err := p.generateFile(structs[0].Namespace, outputPath, structs, enums, typeDefs, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
			Content: content,
		})
	}
	for _, outputPath := range enumOrder {
		enums, ok := outputEnums[outputPath]
		if !ok {
			continue
		}
		var typeDefs []resolution.Type
		if tds, ok := outputTypeDefs[outputPath]; ok {
			typeDefs = tds
			delete(outputTypeDefs, outputPath)
		}
		content, err := p.generateFile(enums[0].Namespace, outputPath, nil, enums, typeDefs, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
			Content: content,
		})
	}
	// Handle standalone typedef-only outputs
	for _, outputPath := range typeDefOrder {
		typeDefs, ok := outputTypeDefs[outputPath]
		if !ok {
			continue
		}
		content, err := p.generateFile(typeDefs[0].Namespace, outputPath, nil, nil, typeDefs, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.FileNamePattern),
			Content: content,
		})
	}

	return resp, nil
}

func mergeEnums(a, b []resolution.Type) []resolution.Type {
	seen := make(map[string]bool, len(a))
	for _, e := range a {
		seen[e.QualifiedName] = true
	}
	result := append([]resolution.Type{}, a...)
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
	structs []resolution.Type,
	enums []resolution.Type,
	typeDefs []resolution.Type,
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
		Imports:       make(map[string]*importSpec),
	}
	skip := func(s resolution.Type) bool { return omit.IsType(s, "ts") }
	rawKeyFields := key.Collect(structs, req.Resolutions, skip)
	data.Ontology = p.extractOntology(structs, rawKeyFields, skip)
	if data.Ontology != nil {
		data.addNamedImport("@/ontology", "ontology")
	}

	// Separate distinct types (which don't have dependencies on schema types)
	// from alias types (which might depend on structs)
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
	for _, td := range distinctTypeDefs {
		data.TypeDefs = append(data.TypeDefs, p.processTypeDef(td, data))
	}

	for _, e := range enums {
		data.Enums = append(data.Enums, p.processEnum(e))
	}

	// Combine aliases and structs for topological sorting
	var combinedTypes []resolution.Type
	combinedTypes = append(combinedTypes, aliasTypeDefs...)
	combinedTypes = append(combinedTypes, structs...)

	// Sort topologically so dependencies come before dependents
	sortedTypes := req.Resolutions.TopologicalSort(combinedTypes)

	// Process in sorted order
	for _, typ := range sortedTypes {
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
		case resolution.StructForm:
			data.SortedDecls = append(data.SortedDecls, sortedDeclData{
				IsStruct: true,
				Struct:   p.processStruct(typ, req.Resolutions, data),
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
) *ontologyData {
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
		// Check for @ts type override on the distinct type itself
		if typeOverride := getTypeTypeOverride(td, "ts"); typeOverride != "" {
			return typeDefData{
				Name:    td.Name,
				TSName:  td.Name,
				TSType:  primitiveToTS(typeOverride),
				ZodType: primitiveToZod(typeOverride, data, false),
			}
		}
		return typeDefData{
			Name:    td.Name,
			TSName:  td.Name,
			TSType:  p.typeDefBaseToTS(&form.Base, data),
			ZodType: p.typeDefBaseToZod(&form.Base, data),
		}
	case resolution.AliasForm:
		return typeDefData{
			Name:    td.Name,
			TSName:  td.Name,
			TSType:  p.typeDefBaseToTS(&form.Target, data),
			ZodType: p.typeDefBaseToZod(&form.Target, data),
		}
	default:
		return typeDefData{Name: td.Name, TSName: td.Name, TSType: "unknown", ZodType: "z.unknown()"}
	}
}

// typeDefBaseToZod converts a TypeDef's base type to a Zod schema string.
// This handles primitives, distinct types, and generic struct references with type args.
func (p *Plugin) typeDefBaseToZod(typeRef *resolution.TypeRef, data *templateData) string {
	if typeRef == nil {
		return "z.unknown()"
	}
	// Use the full typeRefToZod logic which handles all cases:
	// - primitives
	// - generic struct references with type args (like status.Status<StatusDetails>)
	// - distinct types
	// - enums
	return p.typeRefToZod(typeRef, data.Request.Resolutions, data, false)
}

// typeDefBaseToTS converts a TypeDef's base type to a TypeScript type string.
// This handles primitives, distinct types, and generic struct references with type args.
// TypeDefs use z.infer so they don't need concrete type imports.
func (p *Plugin) typeDefBaseToTS(typeRef *resolution.TypeRef, data *templateData) string {
	if typeRef == nil {
		return "unknown"
	}
	// Use the full typeRefToTS logic which handles all cases
	// TypeDefs don't need type imports since they use z.infer<typeof ...Z>
	return p.typeRefToTS(typeRef, data.Request.Resolutions, data, false)
}

func (p *Plugin) processStruct(entry resolution.Type, table *resolution.Table, data *templateData) structData {
	// Handle alias types (which don't have StructForm but can be generic)
	if aliasForm, isAlias := entry.Form.(resolution.AliasForm); isAlias {
		sd := structData{
			Name:          entry.Name,
			TSName:        entry.Name,
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
			sd.TypeParams = append(sd.TypeParams, p.processTypeParam(tp, table, data))
		}
		// Check if all type params are optional (have defaults or are marked optional)
		sd.AllParamsOptional = true
		for _, tp := range sd.TypeParams {
			if !tp.HasDefault {
				sd.AllParamsOptional = false
				break
			}
		}
		sd.AliasOf = p.typeRefToZod(&aliasForm.Target, table, data, sd.UseInput)
		// For non-generic aliases to parameterized generic types, generate an explicit type reference
		// This is needed because z.infer doesn't work well with custom ZodObject types
		// We check len(aliasForm.TypeParams) == 0 because IsGeneric() checks the target, not the alias
		if len(aliasForm.TypeParams) == 0 && len(aliasForm.Target.TypeArgs) > 0 {
			sd.AliasTypeRef = p.typeRefToTSType(&aliasForm.Target, table, data)
			fmt.Printf("DEBUG SET: %s -> AliasTypeRef: '%s'\n", sd.Name, sd.AliasTypeRef)
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
		IsGeneric:     form.IsGeneric(),
		IsSingleParam: len(form.TypeParams) == 1,
		IsAlias:       false,
		IsRecursive:   form.IsRecursive,
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
		sd.TypeParams = append(sd.TypeParams, p.processTypeParam(tp, table, data))
	}

	// Check if all type params are optional (have defaults or are marked optional)
	sd.AllParamsOptional = true
	for _, tp := range sd.TypeParams {
		if !tp.HasDefault {
			sd.AllParamsOptional = false
			break
		}
	}

	// Handle struct extension with Zod's .omit().partial().extend() pattern
	if form.Extends != nil {
		parentType, ok := form.Extends.Resolve(table)
		if ok {
			parentForm, isStruct := parentType.Form.(resolution.StructForm)
			if isStruct {
				sd.HasExtends = true

				// Get parent's TSName (respecting @ts name domain)
				parentTSName := parentType.Name
				if tsDomain, ok := parentType.Domains["ts"]; ok {
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
				for _, f := range form.OmittedFields {
					sd.OmittedFields = append(sd.OmittedFields, lo.CamelCase(f))
				}

				// Handle generic parent: need to call parent function with schema args
				if parentForm.IsGeneric() {
					sd.ExtendsParentIsGeneric = true
					// Build list of schema param names from parent's type params
					for _, tp := range parentForm.TypeParams {
						sd.ExtendsParentSchemaArgs = append(sd.ExtendsParentSchemaArgs, lo.CamelCase(tp.Name))
					}
				}

				// Build map of parent fields for comparison
				parentFields := make(map[string]resolution.Field)
				for _, pf := range resolution.UnifiedFields(parentType, table) {
					parentFields[pf.Name] = pf
				}

				// Categorize child fields into partial vs extend
				for _, field := range form.Fields {
					parentField, existsInParent := parentFields[field.Name]
					if existsInParent {
						if isOnlyOptionalityChange(parentField, field) {
							// Same base type, just made optional -> use .partial()
							sd.PartialFields = append(sd.PartialFields, p.processField(field, entry, table, data, sd.UseInput, sd.ConcreteTypes))
						} else {
							// Different type -> omit parent's version AND extend with new type
							sd.OmittedFields = append(sd.OmittedFields, lo.CamelCase(field.Name))
							sd.ExtendFields = append(sd.ExtendFields, p.processField(field, entry, table, data, sd.UseInput, sd.ConcreteTypes))
						}
					} else {
						// New field -> just extend
						sd.ExtendFields = append(sd.ExtendFields, p.processField(field, entry, table, data, sd.UseInput, sd.ConcreteTypes))
					}
				}
				// Add optional import for concrete types with partial fields
				if sd.ConcreteTypes && len(sd.PartialFields) > 0 {
					addXImport(data, xImport{name: "optional", submodule: "optional"})
				}
				return sd
			}
		}
	}

	// Non-extending struct: use all fields (flattened for compatibility)
	allFields := resolution.UnifiedFields(entry, table)
	sd.Fields = make([]fieldData, 0, len(allFields))

	// Build map of optional type params for conditional field detection
	optionalTypeParams := make(map[string]bool)
	for _, tp := range form.TypeParams {
		if tp.Optional {
			optionalTypeParams[tp.Name] = true
		}
	}

	for _, field := range allFields {
		fd := p.processField(field, entry, table, data, sd.UseInput, sd.ConcreteTypes)
		sd.Fields = append(sd.Fields, fd)

		// For concrete_types: detect fields that reference optional type params
		if sd.ConcreteTypes && field.Type.IsTypeParam() &&
			field.Type.TypeParam != nil &&
			optionalTypeParams[field.Type.TypeParam.Name] {
			// This field depends on an optional type param - add to conditional fields
			sd.ConditionalFields = append(sd.ConditionalFields, conditionalFieldData{
				Field:         fd,
				TypeParamName: field.Type.TypeParam.Name,
				NeverType:     "z.ZodNever",
			})
		} else {
			// Regular field - always present
			sd.BaseFields = append(sd.BaseFields, fd)
		}
	}
	return sd
}

// isAliasType checks if a type has AliasForm.
func isAliasType(typ resolution.Type) bool {
	_, ok := typ.Form.(resolution.AliasForm)
	return ok
}

// isOnlyOptionalityChange returns true if the child field is the same as the parent field
// except for being optional. This allows using .partial() instead of .extend().
func isOnlyOptionalityChange(parent, child resolution.Field) bool {
	// Child must be optional and parent must NOT be optional
	childIsOptional := child.IsOptional || child.IsHardOptional
	parentIsOptional := parent.IsOptional || parent.IsHardOptional
	if !childIsOptional || parentIsOptional {
		return false
	}
	// Compare base types (must be the same)
	return sameBaseType(parent.Type, child.Type)
}

// isArrayTypeRef checks if a TypeRef is an Array type.
func isArrayTypeRef(r resolution.TypeRef) bool {
	return r.Name == "Array"
}

// sameBaseType compares two TypeRefs ignoring optionality.
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

func (p *Plugin) processTypeParam(tp resolution.TypeParam, table *resolution.Table, data *templateData) typeParamData {
	tpd := typeParamData{Name: tp.Name, Constraint: "z.ZodType"}
	if tp.Constraint != nil {
		if resolution.IsPrimitive(tp.Constraint.Name) && tp.Constraint.Name == "json" {
			tpd.IsJSON = true
		}
		// Enum constraints use z.ZodType<EnumType> to allow literals and subset enums
		resolved, ok := tp.Constraint.Resolve(table)
		if ok {
			if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
				enumTypeName := lo.Capitalize(lo.CamelCase(resolved.Name))
				tpd.Constraint = "z.ZodType<" + enumTypeName + ">"
			}
		}
	}
	if tp.Default != nil {
		tpd.HasDefault = true
		// Handle enum defaults
		resolved, ok := tp.Default.Resolve(table)
		if ok {
			if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
				enumZodName := lo.CamelCase(resolved.Name) + "Z"
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
	} else if tp.Optional {
		// Optional type param with no explicit default -> default to never
		tpd.HasDefault = true
		tpd.Default = "z.ZodNever"
		tpd.DefaultValue = "z.unknown()"
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

func fallbackForConstraint(constraint *resolution.TypeRef, table *resolution.Table) string {
	if constraint == nil {
		return "z.unknown()"
	}
	// Handle enum constraints
	resolved, ok := constraint.Resolve(table)
	if ok {
		if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
			return lo.CamelCase(resolved.Name) + "Z"
		}
	}
	return defaultValueToTS(constraint.Name)
}

// isSelfReference checks if a type reference points to the parent struct,
// either directly or through arrays/optionals.
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

func (p *Plugin) processField(field resolution.Field, parentType resolution.Type, table *resolution.Table, data *templateData, useInput bool, needsTypeImports bool) fieldData {
	isArray := field.Type.Name == "Array"
	fd := fieldData{
		Name:           field.Name,
		TSName:         lo.CamelCase(field.Name),
		IsOptional:     field.IsOptional,
		IsHardOptional: field.IsHardOptional,
		IsArray:        isArray,
		IsSelfRef:      isSelfReference(field.Type, parentType),
	}
	if typeOverride := getFieldTypeOverride(field, "ts"); typeOverride != "" {
		fd.ZodType = primitiveToZod(typeOverride, data, useInput)
		fd.TSType = primitiveToTS(typeOverride)
		if validateDomain, ok := field.Domains["validate"]; ok {
			fd.ZodType = p.applyValidation(fd.ZodType, validateDomain, field.Type, field.Name, table)
		}
	} else {
		// For arrays, process just the element type - the array wrapper is added later
		// by array.nullishToEmpty() or array.nullToUndefined()
		typeRefToProcess := &field.Type
		if isArray && len(field.Type.TypeArgs) > 0 {
			typeRefToProcess = &field.Type.TypeArgs[0]
		}
		fd.ZodType = p.typeRefToZod(typeRefToProcess, table, data, useInput)
		fd.TSType = p.typeRefToTS(typeRefToProcess, table, data, needsTypeImports)
		if validateDomain, ok := field.Domains["validate"]; ok {
			fd.ZodType = p.applyValidation(fd.ZodType, validateDomain, field.Type, field.Name, table)
		}
	}
	// Handle @key generate for auto-generating IDs
	if key.HasGenerate(field) {
		primitive := key.ResolvePrimitive(field.Type, table)
		switch primitive {
		case "string":
			addXImport(data, xImport{name: "id", submodule: "id"})
			fd.ZodType = fmt.Sprintf("%s.default(() => id.create())", fd.ZodType)
		case "uuid":
			addXImport(data, xImport{name: "uuid", submodule: "uuid"})
			fd.ZodType = fmt.Sprintf("%s.default(() => uuid.create())", fd.ZodType)
		}
	}
	isAnyOptional := field.IsOptional || field.IsHardOptional
	if isArray {
		addXImport(data, xImport{name: "array", submodule: "array"})
		if isAnyOptional {
			// Optional array: null/undefined -> undefined, [] stays []
			fd.ZodType = fmt.Sprintf("array.nullToUndefined(%s)", fd.ZodType)
		} else {
			// Required array: coerce nullish -> []
			fd.ZodType = fmt.Sprintf("array.nullishToEmpty(%s)", fd.ZodType)
		}
	} else if isAnyOptional {
		fd.ZodType += ".optional()"
	}
	return fd
}

func getFieldTypeOverride(field resolution.Field, domainName string) string {
	domain, ok := field.Domains[domainName]
	if !ok {
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

func getTypeTypeOverride(typ resolution.Type, domainName string) string {
	domain, ok := typ.Domains[domainName]
	if !ok {
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

func (p *Plugin) typeRefToZod(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData, useInput bool) string {
	return p.typeRefToZodInternal(typeRef, table, data, useInput, false)
}

// typeRefToTSType generates an explicit TypeScript type reference for a type reference.
// For example, status.Status<StatusDetails> becomes status.Status<typeof statusDetailsZ>
func (p *Plugin) typeRefToTSType(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData) string {
	if typeRef == nil {
		return "unknown"
	}

	// Handle primitives
	if resolution.IsPrimitive(typeRef.Name) {
		return primitiveToTSType(typeRef.Name)
	}

	// Try to resolve the type
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return "unknown"
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		typeName := resolved.Name
		if form.IsGeneric() && len(typeRef.TypeArgs) > 0 {
			args := make([]string, len(typeRef.TypeArgs))
			for i, arg := range typeRef.TypeArgs {
				// For type args that are structs/types, use typeof schemaZ
				args[i] = p.typeArgToTSType(&arg, table, data)
			}
			typeName = fmt.Sprintf("%s<%s>", typeName, strings.Join(args, ", "))
		}
		if resolved.Namespace != data.Namespace {
			ns := resolved.Namespace
			targetOutputPath := output.GetPath(resolved, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.addNamedImport(calculateImportPath(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("%s.%s", ns, typeName)
		}
		return typeName

	case resolution.EnumForm:
		typeName := resolved.Name
		if resolved.Namespace != data.Namespace {
			ns := resolved.Namespace
			targetOutputPath := enum.FindOutputPath(resolved, table, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.addNamedImport(calculateImportPath(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("%s.%s", ns, typeName)
		}
		return typeName
	}

	return "unknown"
}

// typeArgToTSType converts a type argument to its TypeScript type reference.
// For struct types, it returns typeof schemaZ to get the Zod schema type.
func (p *Plugin) typeArgToTSType(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData) string {
	if typeRef == nil {
		return "unknown"
	}

	// Handle primitives - not wrapped in typeof
	if resolution.IsPrimitive(typeRef.Name) {
		return primitiveToTSType(typeRef.Name)
	}

	// Try to resolve the type
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return "unknown"
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		schemaName := lo.CamelCase(resolved.Name) + "Z"
		if form.IsGeneric() && len(typeRef.TypeArgs) > 0 {
			// For generic types with args, recursively get the full schema call
			args := make([]string, len(typeRef.TypeArgs))
			for i, arg := range typeRef.TypeArgs {
				args[i] = p.typeArgToTSType(&arg, table, data)
			}
			if len(form.TypeParams) == 1 {
				schemaName = fmt.Sprintf("%s(%s)", schemaName, args[0])
			} else {
				namedArgs := make([]string, len(typeRef.TypeArgs))
				for i, arg := range args {
					namedArgs[i] = fmt.Sprintf("%s: %s", lo.CamelCase(form.TypeParams[i].Name), arg)
				}
				schemaName = fmt.Sprintf("%s({%s})", schemaName, strings.Join(namedArgs, ", "))
			}
		}
		if resolved.Namespace != data.Namespace {
			ns := resolved.Namespace
			targetOutputPath := output.GetPath(resolved, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.addNamedImport(calculateImportPath(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("typeof %s.%s", ns, schemaName)
		}
		return fmt.Sprintf("typeof %s", schemaName)

	case resolution.EnumForm:
		schemaName := lo.CamelCase(resolved.Name) + "Z"
		if resolved.Namespace != data.Namespace {
			ns := resolved.Namespace
			targetOutputPath := enum.FindOutputPath(resolved, table, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.addNamedImport(calculateImportPath(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("typeof %s.%s", ns, schemaName)
		}
		return fmt.Sprintf("typeof %s", schemaName)
	}

	return "unknown"
}

// primitiveToTSType converts a primitive type name to its TypeScript type equivalent.
func primitiveToTSType(name string) string {
	switch name {
	case "string", "uuid":
		return "string"
	case "bool":
		return "boolean"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return "number"
	case "json":
		return "unknown"
	default:
		return "unknown"
	}
}

func (p *Plugin) typeRefToZodInternal(typeRef *resolution.TypeRef, table *resolution.Table, data *templateData, useInput bool, forStructArg bool) string {
	if typeRef == nil {
		return "z.unknown()"
	}

	// Handle type parameter reference
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		paramName := lo.CamelCase(typeRef.TypeParam.Name)
		if forStructArg {
			return paramName
		}
		if typeRef.TypeParam.Constraint != nil && typeRef.TypeParam.Constraint.Name == "json" {
			addXImport(data, xImport{name: "zod", submodule: "zod"})
			if useInput {
				return fmt.Sprintf("zod.jsonStringifier(%s)", paramName)
			}
			return fmt.Sprintf("zod.stringifiedJSON(%s)", paramName)
		}
		// Only add fallback if the type param has a default or is optional
		if typeRef.TypeParam.Default != nil || typeRef.TypeParam.Optional {
			return fmt.Sprintf("%s ?? %s", paramName, fallbackForConstraint(typeRef.TypeParam.Constraint, table))
		}
		return paramName
	}

	// Handle primitives
	if resolution.IsPrimitive(typeRef.Name) {
		return primitiveToZod(typeRef.Name, data, useInput)
	}

	// Handle Array type
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		elemZod := p.typeRefToZodInternal(&typeRef.TypeArgs[0], table, data, useInput, false)
		return fmt.Sprintf("z.array(%s)", elemZod)
	}

	// Handle Map type
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		keyZ := p.typeRefToZodInternal(&typeRef.TypeArgs[0], table, data, useInput, false)
		valueZ := p.typeRefToZodInternal(&typeRef.TypeArgs[1], table, data, useInput, false)
		return fmt.Sprintf("z.record(%s, %s)", keyZ, valueZ)
	}

	// Try to resolve the type
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return "z.unknown()"
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		schemaName := lo.CamelCase(resolved.Name) + "Z"
		if form.IsGeneric() {
			if len(typeRef.TypeArgs) > 0 {
				args := make([]string, len(typeRef.TypeArgs))
				for i, arg := range typeRef.TypeArgs {
					args[i] = p.typeRefToZodInternal(&arg, table, data, useInput, true)
				}
				if len(form.TypeParams) == 1 {
					schemaName = fmt.Sprintf("%s(%s)", schemaName, args[0])
				} else {
					namedArgs := make([]string, len(typeRef.TypeArgs))
					for i, arg := range args {
						namedArgs[i] = fmt.Sprintf("%s: %s", lo.CamelCase(form.TypeParams[i].Name), arg)
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
			data.addNamedImport(calculateImportPath(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("%s.%s", ns, schemaName)
		}
		return schemaName

	case resolution.EnumForm:
		enumName := lo.CamelCase(resolved.Name) + "Z"
		if resolved.Namespace != data.Namespace {
			ns := resolved.Namespace
			targetOutputPath := enum.FindOutputPath(resolved, table, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.addNamedImport(calculateImportPath(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("%s.%s", ns, enumName)
		}
		return enumName

	case resolution.DistinctForm:
		schemaName := lo.CamelCase(resolved.Name) + "Z"
		if resolved.Namespace != data.Namespace {
			ns := resolved.Namespace
			targetOutputPath := output.GetPath(resolved, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.addNamedImport(calculateImportPath(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("%s.%s", ns, schemaName)
		}
		return schemaName

	case resolution.AliasForm:
		// For non-generic aliases, just use the schema name
		if !form.IsGeneric() {
			schemaName := lo.CamelCase(resolved.Name) + "Z"
			if resolved.Namespace != data.Namespace {
				ns := resolved.Namespace
				targetOutputPath := output.GetPath(resolved, "ts")
				if targetOutputPath == "" {
					targetOutputPath = ns
				}
				data.addNamedImport(calculateImportPath(data.OutputPath, targetOutputPath), ns)
				return fmt.Sprintf("%s.%s", ns, schemaName)
			}
			return schemaName
		}
		// For generic aliases, substitute type args and generate inline
		target := form.Target
		if len(typeRef.TypeArgs) > 0 {
			// Build substitution map from alias type params to caller's type args
			typeArgMap := make(map[string]resolution.TypeRef)
			for i, tp := range form.TypeParams {
				if i < len(typeRef.TypeArgs) {
					typeArgMap[tp.Name] = typeRef.TypeArgs[i]
				}
			}
			target = resolution.SubstituteTypeRef(target, typeArgMap)
		}
		return p.typeRefToZodInternal(&target, table, data, useInput, forStructArg)

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

	// Handle type parameter reference
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		if forStructArg {
			return typeRef.TypeParam.Name
		}
		return fmt.Sprintf("z.infer<%s>", typeRef.TypeParam.Name)
	}

	// Handle primitives
	if resolution.IsPrimitive(typeRef.Name) {
		// Add imports for special primitive types used in concrete type generation
		if needsTypeImports {
			switch typeRef.Name {
			case "timestamp":
				addXImport(data, xImport{name: "TimeStamp", submodule: "telem"})
			case "timespan":
				addXImport(data, xImport{name: "TimeSpan", submodule: "telem"})
			case "color":
				addXImport(data, xImport{name: "Color", submodule: "color"})
			}
		}
		return primitiveToTS(typeRef.Name)
	}

	// Handle Array type
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		elemTS := p.typeRefToTSInternal(&typeRef.TypeArgs[0], table, data, forStructArg, needsTypeImports)
		return elemTS + "[]"
	}

	// Handle Map type
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) >= 2 {
		keyType := p.typeRefToTSInternal(&typeRef.TypeArgs[0], table, data, forStructArg, needsTypeImports)
		valueType := p.typeRefToTSInternal(&typeRef.TypeArgs[1], table, data, forStructArg, needsTypeImports)
		return fmt.Sprintf("Record<%s, %s>", keyType, valueType)
	}

	// Try to resolve the type
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return "unknown"
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		typeName := resolved.Name
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
		if resolved.Namespace != data.Namespace {
			return fmt.Sprintf("%s.%s", resolved.Namespace, resolved.Name)
		}
		return resolved.Name

	case resolution.DistinctForm:
		if resolved.Namespace != data.Namespace {
			ns := resolved.Namespace
			targetOutputPath := output.GetPath(resolved, "ts")
			if targetOutputPath == "" {
				targetOutputPath = ns
			}
			data.addNamedImport(calculateImportPath(data.OutputPath, targetOutputPath), ns)
			return fmt.Sprintf("%s.%s", ns, resolved.Name)
		}
		return resolved.Name

	case resolution.AliasForm:
		// For aliases, use the alias name with type args if present
		typeName := resolved.Name
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

	default:
		return "unknown"
	}
}

var primitiveTSTypes = map[string]string{
	"string": "string", "uuid": "string",
	"bool": "boolean",
	"int8": "number", "int16": "number", "int32": "number", "int64": "number",
	"uint8": "number", "uint12": "number", "uint16": "number", "uint20": "number", "uint32": "number", "uint64": "number",
	"float32": "number", "float64": "number",
	"timestamp": "TimeStamp", "timespan": "TimeSpan", "data_type": "DataType",
	"color": "Color",
	"json":  "unknown", "bytes": "Uint8Array",
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
	"int32":              {schema: "z.int32()"},
	"int64":              {schema: "z.int64()"},
	"uint8":              {schema: "zod.uint8Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"uint12":             {schema: "zod.uint12Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"uint16":             {schema: "zod.uint16Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"uint20":             {schema: "zod.uint20Z", xImports: []xImport{{name: "zod", submodule: "zod"}}},
	"uint32":             {schema: "z.uint32()"},
	"uint64":             {schema: "z.uint64()"},
	"float32":            {schema: "z.number()"},
	"float64":            {schema: "z.number()"},
	"timestamp":          {schema: "TimeStamp.z", xImports: []xImport{{name: "TimeStamp", submodule: "telem"}}},
	"timespan":           {schema: "TimeSpan.z", xImports: []xImport{{name: "TimeSpan", submodule: "telem"}}},
	"time_range":         {schema: "TimeRange.z", xImports: []xImport{{name: "TimeRange", submodule: "telem"}}},
	"time_range_bounded": {schema: "TimeRange.boundedZ", xImports: []xImport{{name: "TimeRange", submodule: "telem"}}},
	"data_type":          {schema: "DataType.z", xImports: []xImport{{name: "DataType", submodule: "telem"}}},
	"color":              {schema: "color.colorZ", xImports: []xImport{{name: "color", submodule: "color"}}},
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

func (p *Plugin) applyValidation(zodType string, domain resolution.Domain, typeRef resolution.TypeRef, fieldName string, table *resolution.Table) string {
	rules := validation.Parse(domain)
	if validation.IsEmpty(rules) {
		return zodType
	}
	isString := resolution.IsPrimitive(typeRef.Name) && resolution.IsStringPrimitive(typeRef.Name)
	isNumber := resolution.IsPrimitive(typeRef.Name) && resolution.IsNumberPrimitive(typeRef.Name)
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
			// Special handling for timestamp/timespan with default of 0
			if rules.Default.IntValue == 0 {
				switch typeRef.Name {
				case "timestamp":
					zodType = fmt.Sprintf("%s.default(TimeStamp.ZERO)", zodType)
				case "timespan":
					zodType = fmt.Sprintf("%s.default(TimeSpan.ZERO)", zodType)
				default:
					zodType = fmt.Sprintf("%s.default(%d)", zodType, rules.Default.IntValue)
				}
			} else {
				zodType = fmt.Sprintf("%s.default(%d)", zodType, rules.Default.IntValue)
			}
		case resolution.ValueKindFloat:
			zodType = fmt.Sprintf("%s.default(%f)", zodType, rules.Default.FloatValue)
		case resolution.ValueKindBool:
			zodType = fmt.Sprintf("%s.default(%t)", zodType, rules.Default.BoolValue)
		case resolution.ValueKindIdent:
			// Handle identifier-based defaults like "now" for timestamps
			if rules.Default.IdentValue == "now" && typeRef.Name == "timestamp" {
				zodType = fmt.Sprintf("%s.default(() => TimeStamp.now())", zodType)
			}
		}
	}
	return zodType
}

type templateData struct {
	Namespace, OutputPath string
	Request               *plugin.Request
	Structs               []structData
	Enums                 []enumData
	TypeDefs              []typeDefData
	SortedDecls           []sortedDeclData // Topologically sorted aliases and structs
	GenerateTypes         bool
	Imports               map[string]*importSpec
	Ontology              *ontologyData
}

// sortedDeclData holds a single declaration (typedef or struct) for sorted output.
type sortedDeclData struct {
	IsTypeDef bool
	IsStruct  bool
	TypeDef   typeDefData
	Struct    structData
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
	Name, TSName, AliasOf, AliasTypeRef            string
	Fields                                         []fieldData
	TypeParams                                     []typeParamData
	UseInput, Handwritten, ConcreteTypes           bool
	IsGeneric, IsSingleParam, IsAlias, IsRecursive bool
	AllParamsOptional                              bool // True if all type params have defaults or are optional

	// Extension support
	HasExtends              bool
	ExtendsName             string      // Parent schema name (e.g., "payloadZ")
	ExtendsTypeName         string      // Parent type name (e.g., "Payload")
	ExtendsParentIsGeneric  bool        // True if parent has type params
	ExtendsParentSchemaArgs []string    // Schema param names for calling generic parent (e.g., ["properties", "make", "model"])
	OmittedFields           []string    // Fields omitted from parent via -fieldName
	PartialFields           []fieldData // Fields that only need .partial() (just optionality change)
	ExtendFields            []fieldData // Fields that need .extend() (new fields or type changes)

	// Conditional field support for optional type params
	ConditionalFields []conditionalFieldData // Fields to include conditionally based on type param
	BaseFields        []fieldData            // Fields that are always present (non-conditional)
}

type typeParamData struct {
	Name, Constraint, Default, DefaultValue string
	HasDefault, IsJSON                      bool
}

type fieldData struct {
	Name, TSName, ZodType, TSType                  string
	IsOptional, IsHardOptional, IsArray, IsSelfRef bool
}

type conditionalFieldData struct {
	Field         fieldData // The field data
	TypeParamName string    // The type param this field depends on (e.g., "D")
	NeverType     string    // The "never" type for this param (e.g., "z.ZodNever")
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
{{- range .TypeDefs }}

export const {{ .TSName | camelCase }}Z = {{ .ZodType }};
export type {{ .TSName }} = z.infer<typeof {{ .TSName | camelCase }}Z>;
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

export interface {{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> {
{{- range $i, $p := .TypeParams }}
  {{ $p.Name | camelCase }}{{ if $p.HasDefault }}?{{ end }}: {{ $p.Name }};
{{- end }}
}

export const {{ camelCase .TSName }}Z = <{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>({
{{- range $i, $p := .TypeParams }}
  {{ $p.Name | camelCase }},
{{- end }}
}: {{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>{{ if .AllParamsOptional }} = {}{{ end }}) =>
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
{{- if .AliasTypeRef }}
export type {{ .TSName }} = {{ .AliasTypeRef }};
{{- else }}
export interface {{ .TSName }} extends z.{{ if .UseInput }}input{{ else }}infer{{ end }}<typeof {{ camelCase .TSName }}Z> {}
{{- end }}
{{- end }}
{{- end }}
{{- else if .IsGeneric }}
{{- if .IsSingleParam }}
{{- if and .ConcreteTypes .ConditionalFields }}

export type {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = Omit<z.ZodObject<z.ZodRawShape>, "_output"> & { _output: {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ $p.Name }}{{ end }}> };
{{- end }}

export const {{ camelCase .TSName }}Z = <{{ range $i, $p := .TypeParams }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>({{ range $i, $p := .TypeParams }}{{ $p.Name | camelCase }}{{ if $p.HasDefault }}?{{ end }}: {{ $p.Name }}{{ end }}){{ if and .ConcreteTypes .ConditionalFields }}: {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ $p.Name }}{{ end }}>{{ end }} =>
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
    }){{ if and .ConcreteTypes .ConditionalFields }} as unknown as {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ $p.Name }}{{ end }}>{{ end }}
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
  }){{ if and .ConcreteTypes .ConditionalFields }} as unknown as {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ $p.Name }}{{ end }}>{{ end }};
{{- end }}
{{- else }}

export interface {{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> {
{{- range $i, $p := .TypeParams }}
  {{ $p.Name | camelCase }}{{ if $p.HasDefault }}?{{ end }}: {{ $p.Name }};
{{- end }}
}
{{- if and .ConcreteTypes .ConditionalFields }}

export type {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = Omit<z.ZodObject<z.ZodRawShape>, "_output"> & { _output: {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}> };
{{- end }}

export const {{ camelCase .TSName }}Z = <{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}>({
{{- range $i, $p := .TypeParams }}
  {{ $p.Name | camelCase }},
{{- end }}
}: {{ .TSName }}Schemas<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>{{ if .AllParamsOptional }} = {}{{ end }}){{ if and .ConcreteTypes .ConditionalFields }}: {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>{{ end }} =>
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
    }){{ if and .ConcreteTypes .ConditionalFields }} as unknown as {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>{{ end }}
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
  }){{ if and .ConcreteTypes .ConditionalFields }} as unknown as {{ .TSName }}ZodObject<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>{{ end }};
{{- end }}
{{- end }}
{{- if $.GenerateTypes }}
{{- if .ConcreteTypes }}
{{- if .HasExtends }}
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = {{ if .PartialFields }}optional.Optional<{{ end }}{{ if .OmittedFields }}Omit<{{ end }}{{ .ExtendsTypeName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }}{{ end }}>{{ if .OmittedFields }}, {{ range $i, $f := .OmittedFields }}{{ if $i }} | {{ end }}"{{ $f }}"{{ end }}>{{ end }}{{ if .PartialFields }}, {{ range $i, $f := .PartialFields }}{{ if $i }} | {{ end }}"{{ $f.TSName }}"{{ end }}>{{ end }}{{ if .ExtendFields }} & {
{{- range .ExtendFields }}
  {{ .TSName }}{{ if or .IsOptional .IsHardOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}{{ end }};
{{- else }}
{{- if .ConditionalFields }}
export type {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> = {
{{- range .BaseFields }}
  {{ .TSName }}{{ if or .IsOptional .IsHardOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}{{ range .ConditionalFields }} & ([{{ .TypeParamName }}] extends [{{ .NeverType }}] ? {} : { {{ .Field.TSName }}: {{ .Field.TSType }}{{ if .Field.IsArray }}[]{{ end }} }){{ end }};
{{- else }}
export interface {{ .TSName }}<{{ range $i, $p := .TypeParams }}{{ if $i }}, {{ end }}{{ $p.Name }} extends {{ $p.Constraint }}{{ if $p.HasDefault }} = {{ $p.Default }}{{ end }}{{ end }}> {
{{- range .Fields }}
  {{ .TSName }}{{ if or .IsOptional .IsHardOptional }}?{{ end }}: {{ .TSType }}{{ if .IsArray }}[]{{ end }};
{{- end }}
}
{{- end }}
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
{{- end }}
{{- end }}
{{- if .Ontology }}

export const ontologyID = ontology.createIDFactory<{{ .Ontology.KeyType }}>("{{ .Ontology.TypeName }}");
export const TYPE_ONTOLOGY_ID = ontologyID({{ .Ontology.KeyZeroValue }});
{{- end }}
`))
