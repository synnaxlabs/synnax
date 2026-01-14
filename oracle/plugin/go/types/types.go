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

	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/framework"
	gointernal "github.com/synnaxlabs/oracle/plugin/go/internal"
	goprimitives "github.com/synnaxlabs/oracle/plugin/go/primitives"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
)

const goModulePrefix = "github.com/synnaxlabs/synnax/"

// primitiveMapper is the Go-specific primitive type mapper.
var primitiveMapper = &goprimitives.Mapper{}

type Plugin struct{ Options Options }

type Options struct {
	FileNamePattern string
}

func DefaultOptions() Options {
	return Options{
		FileNamePattern: "types.gen.go",
	}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string { return "go/types" }

func (p *Plugin) Domains() []string { return []string{"go"} }

func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) Check(*plugin.Request) error { return nil }

var gofmtCmd = []string{"gofmt", "-w"}

// PostWrite runs gofmt on all generated Go files.
func (p *Plugin) PostWrite(files []string) error {
	if len(files) == 0 {
		return nil
	}
	var goFiles []string
	for _, f := range files {
		if strings.HasSuffix(f, ".go") {
			goFiles = append(goFiles, f)
		}
	}
	if len(goFiles) == 0 {
		return nil
	}
	return exec.OnFiles(gofmtCmd, goFiles, "")
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	gen := &framework.Generator{
		Domain:          "go",
		FilePattern:     p.Options.FileNamePattern,
		FileGenerator:   &goFileGenerator{},
		MergeByName:     false,
		CollectTypeDefs: true,
		CollectEnums:    true,
		ExtraEnumsFunc:  collectNamespaceEnums,
	}
	return gen.Generate(req)
}

// goFileGenerator implements framework.FileGenerator for Go code generation.
type goFileGenerator struct{}

func (g *goFileGenerator) GenerateFile(ctx *framework.GenerateContext) (string, error) {
	content, err := generateGoFile(ctx.OutputPath, ctx.Structs, ctx.Enums, ctx.TypeDefs, ctx.Table, ctx.RepoRoot)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func collectNamespaceEnums(structs []resolution.Type, table *resolution.Table, outputPath string) []resolution.Type {
	if len(structs) == 0 {
		return nil
	}
	namespace := structs[0].Namespace
	var result []resolution.Type
	for _, e := range table.EnumTypes() {
		if e.Namespace != namespace {
			continue
		}
		// Check if this enum's @go output matches (via file-level inheritance)
		if enumPath := enum.FindOutputPath(e, table, "go"); enumPath == outputPath {
			result = append(result, e)
		}
	}
	return result
}

func generateGoFile(
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	typeDefs []resolution.Type,
	table *resolution.Table,
	repoRoot string,
) ([]byte, error) {
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	} else if len(typeDefs) > 0 {
		namespace = typeDefs[0].Namespace
	} else if len(enums) > 0 {
		namespace = enums[0].Namespace
	}

	pkg := gointernal.DerivePackageName(outputPath)
	imports := gointernal.NewImportManager()

	// Create resolver context
	ctx := &resolver.Context{
		Table:                         table,
		OutputPath:                    outputPath,
		Namespace:                     namespace,
		RepoRoot:                      repoRoot,
		DomainName:                    "go",
		SubstituteDefaultedTypeParams: true, // Go doesn't support advanced generics
	}

	// Create resolver with Go-specific components
	r := &resolver.Resolver{
		Formatter:       &GoFormatter{},
		ImportResolver:  &GoImportResolver{RepoRoot: repoRoot, CurrentPackage: pkg},
		ImportAdder:     imports,
		PrimitiveMapper: primitiveMapper,
	}

	data := &templateData{
		Package:    pkg,
		OutputPath: outputPath,
		Namespace:  namespace,
		Structs:    make([]structData, 0, len(structs)),
		Enums:      make([]enumData, 0, len(enums)),
		TypeDefs:   make([]typeDefData, 0, len(typeDefs)),
		imports:    imports,
		table:      table,
		repoRoot:   repoRoot,
		resolver:   r,
		ctx:        ctx,
	}

	// Process typedefs
	for _, td := range typeDefs {
		if !omit.IsType(td, "go") {
			data.TypeDefs = append(data.TypeDefs, processTypeDef(td, data))
		}
	}

	// Process enums that are in the same namespace
	for _, e := range enums {
		if e.Namespace == namespace && !omit.IsType(e, "go") {
			data.Enums = append(data.Enums, processEnum(e))
		}
	}

	// Process structs
	for _, entry := range structs {
		// Skip omitted structs
		if omit.IsType(entry, "go") {
			continue
		}
		data.Structs = append(data.Structs, processStruct(entry, data))
	}

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func resolveGoImportPath(outputPath, repoRoot string) string {
	return gomod.ResolveImportPath(outputPath, repoRoot, goModulePrefix)
}

func processEnum(e resolution.Type) enumData {
	form := e.Form.(resolution.EnumForm)
	values := make([]enumValueData, 0, len(form.Values))
	for _, v := range form.Values {
		values = append(values, enumValueData{
			Name:     gointernal.ToPascalCase(v.Name),
			Value:    v.StringValue(),
			IntValue: v.IntValue(),
		})
	}
	// Check if enum values are continuous starting at 1 (e.g., 1, 2, 3, ...)
	startsAtOne := form.IsIntEnum && len(values) > 0 && values[0].IntValue == 1
	if startsAtOne {
		for i, v := range values {
			if v.IntValue != int64(i+1) {
				startsAtOne = false
				break
			}
		}
	}
	return enumData{
		Name:        e.Name,
		Values:      values,
		IsIntEnum:   form.IsIntEnum,
		StartsAtOne: startsAtOne,
	}
}

func processTypeDef(td resolution.Type, data *templateData) typeDefData {
	// Check for @go name override
	name := getGoName(td)
	if name == "" {
		name = td.Name
	}

	switch form := td.Form.(type) {
	case resolution.DistinctForm:
		result := typeDefData{
			Name:     name,
			BaseType: data.resolver.ResolveTypeRef(form.Base, data.ctx),
			IsAlias:  false, // DistinctForm → "type X Y" (distinct type)
		}
		for _, tp := range form.TypeParams {
			if tp.HasDefault() {
				continue // Skip defaulted type params
			}
			result.TypeParams = append(result.TypeParams, processTypeParam(tp, data))
		}
		result.IsGeneric = len(result.TypeParams) > 0
		return result
	case resolution.AliasForm:
		// Check if target is a generic struct with optional type params
		// that need types.Nil substitution
		targetRef := form.Target
		if targetResolved, ok := targetRef.Resolve(data.table); ok {
			if targetForm, ok := targetResolved.Form.(resolution.StructForm); ok {
				// Count how many non-defaulted type params the target has
				var nonDefaultedParams []resolution.TypeParam
				for _, tp := range targetForm.TypeParams {
					if !tp.HasDefault() {
						nonDefaultedParams = append(nonDefaultedParams, tp)
					}
				}
				// If there are missing type args for optional params, synthesize nil refs
				providedArgs := len(targetRef.TypeArgs)
				if providedArgs < len(nonDefaultedParams) {
					newTypeArgs := make([]resolution.TypeRef, len(nonDefaultedParams))
					copy(newTypeArgs, targetRef.TypeArgs)
					for i := providedArgs; i < len(nonDefaultedParams); i++ {
						if nonDefaultedParams[i].Optional {
							// Synthesize a nil type reference for optional param
							newTypeArgs[i] = resolution.TypeRef{Name: "nil"}
						}
					}
					targetRef = resolution.TypeRef{
						Name:     targetRef.Name,
						TypeArgs: newTypeArgs,
					}
				}
			}
		}
		baseType := data.resolver.ResolveTypeRef(targetRef, data.ctx)
		result := typeDefData{
			Name:     name,
			BaseType: baseType,
			IsAlias:  true, // AliasForm → "type X = Y" (transparent alias)
		}
		for _, tp := range form.TypeParams {
			if tp.HasDefault() {
				continue // Skip defaulted type params
			}
			result.TypeParams = append(result.TypeParams, processTypeParam(tp, data))
		}
		result.IsGeneric = len(result.TypeParams) > 0
		return result
	default:
		return typeDefData{Name: name, BaseType: "any"}
	}
}

func processStruct(entry resolution.Type, data *templateData) structData {
	form := entry.Form.(resolution.StructForm)
	sd := structData{
		Name:    entry.Name,
		Doc:     doc.Get(entry.Domains),
		Fields:  make([]fieldData, 0, len(form.Fields)),
		IsAlias: false,
	}

	// Check for @go name override
	if name := domain.GetStringFromType(entry, "go", "name"); name != "" {
		sd.Name = name
	}

	// Process type parameters, skipping defaulted ones
	// (Go doesn't support advanced generics with defaults)
	for _, tp := range form.TypeParams {
		if tp.HasDefault() {
			continue // Skip defaulted type params
		}
		sd.TypeParams = append(sd.TypeParams, processTypeParam(tp, data))
	}
	sd.IsGeneric = len(sd.TypeParams) > 0

	// Handle struct extension
	if len(form.Extends) > 0 {
		// If omitting fields, fall back to field flattening
		// since Go struct embedding can't exclude individual parent fields
		if len(form.OmittedFields) > 0 {
			// Use UnifiedFields() which respects OmittedFields
			for _, field := range resolution.UnifiedFields(entry, data.table) {
				sd.Fields = append(sd.Fields, processField(field, data))
			}
			// Process @go field and @go imports directives
			sd.ExtraFields = domain.GetAllStringsFromType(entry, "go", "fields")
			for _, imp := range domain.GetAllStringsFromType(entry, "go", "imports") {
				data.imports.AddExternal(imp)
			}
			return sd
		}

		// Check for field conflicts between parents (requires flattening)
		if hasFieldConflicts(form.Extends, data.table) {
			// Use UnifiedFields() which handles conflict resolution
			for _, field := range resolution.UnifiedFields(entry, data.table) {
				sd.Fields = append(sd.Fields, processField(field, data))
			}
			// Process @go field and @go imports directives
			sd.ExtraFields = domain.GetAllStringsFromType(entry, "go", "fields")
			for _, imp := range domain.GetAllStringsFromType(entry, "go", "imports") {
				data.imports.AddExternal(imp)
			}
			return sd
		}

		// Use struct embedding (idiomatic Go pattern)
		sd.HasExtends = true
		for _, extendsRef := range form.Extends {
			parent, ok := extendsRef.Resolve(data.table)
			if ok {
				sd.ExtendsTypes = append(sd.ExtendsTypes, resolveExtendsType(extendsRef, parent, data))
			}
		}

		// Only include child's own fields (parent fields come via embedding)
		for _, field := range form.Fields {
			sd.Fields = append(sd.Fields, processField(field, data))
		}
		// Process @go field and @go imports directives
		sd.ExtraFields = domain.GetAllStringsFromType(entry, "go", "fields")
		for _, imp := range domain.GetAllStringsFromType(entry, "go", "imports") {
			data.imports.AddExternal(imp)
		}
		return sd
	}

	// Process fields for non-extending structs
	for _, field := range resolution.UnifiedFields(entry, data.table) {
		sd.Fields = append(sd.Fields, processField(field, data))
	}

	// Process @go field directives for extra fields
	sd.ExtraFields = domain.GetAllStringsFromType(entry, "go", "fields")

	// Process @go imports directives for extra imports
	for _, imp := range domain.GetAllStringsFromType(entry, "go", "imports") {
		data.imports.AddExternal(imp)
	}

	return sd
}

func processTypeParam(tp resolution.TypeParam, data *templateData) typeParamData {
	tpd := typeParamData{
		Name:       tp.Name,
		Constraint: "any", // Default constraint
	}

	// Map constraint to Go type
	if tp.Constraint != nil {
		tpd.Constraint = constraintToGo(*tp.Constraint, data)
	}

	return tpd
}

func constraintToGo(constraint resolution.TypeRef, data *templateData) string {
	if resolution.IsPrimitive(constraint.Name) {
		switch constraint.Name {
		case "json":
			return "any"
		case "string":
			return "~string"
		case "int", "int8", "int16", "int32", "int64":
			return "~int | ~int8 | ~int16 | ~int32 | ~int64"
		case "uint", "uint8", "uint16", "uint32", "uint64":
			return "~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64"
		default:
			return data.resolver.ResolveTypeRef(constraint, data.ctx)
		}
	}
	return data.resolver.ResolveTypeRef(constraint, data.ctx)
}

func processField(field resolution.Field, data *templateData) fieldData {
	goType := data.resolver.ResolveTypeRef(field.Type, data.ctx)
	// Hard optional (??) fields become pointers in Go to distinguish nil from zero value.
	// Arrays and maps are reference types and don't need pointers (nil is their zero value).
	if field.IsHardOptional && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map[") && !strings.HasPrefix(goType, "binary.MsgpackEncodedJSON") {
		goType = "*" + goType
	}
	return fieldData{
		GoName:         gointernal.ToPascalCase(field.Name),
		GoType:         goType,
		JSONName:       gointernal.ToSnakeCase(field.Name),
		IsOptional:     field.IsOptional || field.IsHardOptional,
		IsHardOptional: field.IsHardOptional,
		Doc:            doc.Get(field.Domains),
	}
}

func buildGenericType(baseName string, typeArgs []resolution.TypeRef, targetType *resolution.Type, data *templateData) string {
	if len(typeArgs) == 0 {
		return baseName
	}

	// Filter type args, skipping those that correspond to defaulted params
	var args []string
	if targetType != nil {
		if form, ok := targetType.Form.(resolution.StructForm); ok {
			for i, arg := range typeArgs {
				if i < len(form.TypeParams) && form.TypeParams[i].HasDefault() {
					continue // Skip defaulted type args
				}
				args = append(args, data.resolver.ResolveTypeRef(arg, data.ctx))
			}
		} else {
			for _, arg := range typeArgs {
				args = append(args, data.resolver.ResolveTypeRef(arg, data.ctx))
			}
		}
	} else {
		for _, arg := range typeArgs {
			args = append(args, data.resolver.ResolveTypeRef(arg, data.ctx))
		}
	}

	if len(args) == 0 {
		return baseName
	}
	return fmt.Sprintf("%s[%s]", baseName, strings.Join(args, ", "))
}

func resolveExtendsType(extendsRef resolution.TypeRef, parent resolution.Type, data *templateData) string {
	targetOutputPath := output.GetPath(parent, "go")

	// Check for @go name override
	name := getGoName(parent)
	if name == "" {
		name = parent.Name
	}

	// Same namespace AND same output path (or no output path) -> use unqualified name
	if parent.Namespace == data.Namespace && (targetOutputPath == "" || targetOutputPath == data.OutputPath) {
		return buildGenericType(name, extendsRef.TypeArgs, &parent, data)
	}

	// Different output path -> need qualified name with import
	if targetOutputPath == "" {
		// No output path but different namespace - can't resolve package, use unqualified
		return name
	}
	alias := gointernal.DerivePackageAlias(targetOutputPath, data.Package)
	data.imports.AddInternal(alias, resolveGoImportPath(targetOutputPath, data.repoRoot))
	return fmt.Sprintf("%s.%s", alias, buildGenericType(name, extendsRef.TypeArgs, &parent, data))
}

// hasFieldConflicts checks if multiple parents have fields with the same name,
// which would cause ambiguity in Go struct embedding.
func hasFieldConflicts(extends []resolution.TypeRef, table *resolution.Table) bool {
	if len(extends) < 2 {
		return false
	}
	seenFields := make(map[string]bool)
	for _, extendsRef := range extends {
		parent, ok := extendsRef.Resolve(table)
		if !ok {
			continue
		}
		for _, field := range resolution.UnifiedFields(parent, table) {
			if seenFields[field.Name] {
				return true // Conflict found
			}
			seenFields[field.Name] = true
		}
	}
	return false
}

type templateData struct {
	Package    string
	OutputPath string
	Namespace  string
	Structs    []structData
	Enums      []enumData
	TypeDefs   []typeDefData
	imports    *gointernal.ImportManager
	table      *resolution.Table
	repoRoot   string
	resolver   *resolver.Resolver
	ctx        *resolver.Context
}

func (d *templateData) HasImports() bool { return d.imports.HasImports() }

func (d *templateData) ExternalImports() []string { return d.imports.ExternalImports() }

func (d *templateData) InternalImports() []gointernal.InternalImportData {
	return d.imports.InternalImports()
}

type structData struct {
	Name       string
	Doc        string
	Fields     []fieldData
	TypeParams []typeParamData
	IsGeneric  bool
	IsAlias    bool
	AliasOf    string
	// Extension support (multiple inheritance via embedding)
	HasExtends   bool
	ExtendsTypes []string // Parent types (may be qualified: "parent.Parent")
	// Extra fields from @go field directives (raw Go field declarations)
	ExtraFields []string
}

type typeParamData struct {
	Name       string
	Constraint string
}

type fieldData struct {
	GoName         string
	GoType         string
	JSONName       string
	IsOptional     bool
	IsHardOptional bool
	Doc            string
}

func (f fieldData) TagSuffix() string {
	if f.IsHardOptional {
		return ",omitempty"
	}
	return ""
}

type enumData struct {
	Name        string
	Values      []enumValueData
	IsIntEnum   bool
	StartsAtOne bool
}

type enumValueData struct {
	Name     string
	Value    string
	IntValue int64
}

type typeDefData struct {
	Name       string
	BaseType   string
	IsAlias    bool // If true, use "type X = Y", otherwise "type X Y"
	TypeParams []typeParamData
	IsGeneric  bool
}

var templateFuncs = template.FuncMap{
	"join": strings.Join,
}

var fileTemplate = template.Must(template.New("go-types").Funcs(templateFuncs).Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}
{{- if .HasImports}}

import (
{{- range .ExternalImports}}
	"{{.}}"
{{- end}}
{{- range .InternalImports}}
{{- if .NeedsAlias}}
	{{.Alias}} "{{.Path}}"
{{- else}}
	"{{.Path}}"
{{- end}}
{{- end}}
)
{{- end}}
{{- range .TypeDefs}}
{{- if .IsAlias}}

type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} = {{.BaseType}}
{{- else}}

type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} {{.BaseType}}
{{- end}}
{{- end}}
{{- range $enum := .Enums}}

{{- if $enum.IsIntEnum}}

type {{$enum.Name}} uint8

const (
{{- range $i, $v := $enum.Values}}
{{- if eq $i 0}}
	{{$enum.Name}}{{$v.Name}} {{$enum.Name}} = iota{{if $enum.StartsAtOne}} + 1{{end}}
{{- else}}
	{{$enum.Name}}{{$v.Name}}
{{- end}}
{{- end}}
)
{{- else}}

type {{$enum.Name}} string

const (
{{- range $enum.Values}}
	{{$enum.Name}}{{.Name}} {{$enum.Name}} = "{{.Value}}"
{{- end}}
)
{{- end}}
{{- end}}
{{range .Structs}}
{{if .Doc -}}
// {{.Doc}}
{{end -}}
{{if .IsAlias -}}
type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} = {{.AliasOf}}
{{else if .HasExtends -}}
type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} struct {
{{- range .ExtendsTypes}}
	{{.}}
{{- end}}
{{- range .Fields}}
{{- if .Doc}}
	// {{.Doc}}
{{- end}}
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JSONName}}{{.TagSuffix}}" msgpack:"{{.JSONName}}{{.TagSuffix}}"` + "`" + `
{{- end}}
{{- range .ExtraFields}}
	{{.}}
{{- end}}
}
{{else -}}
type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} struct {
{{- range .Fields}}
{{- if .Doc}}
	// {{.Doc}}
{{- end}}
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JSONName}}{{.TagSuffix}}" msgpack:"{{.JSONName}}{{.TagSuffix}}"` + "`" + `
{{- end}}
{{- range .ExtraFields}}
	{{.}}
{{- end}}
}
{{end -}}
{{end -}}
`))

func getGoName(t resolution.Type) string {
	return domain.GetStringFromType(t, "go", "name")
}
