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
var primitiveMapper = goprimitives.Mapper()

// Plugin generates Go type definitions from Oracle schema definitions.
type Plugin struct{ Options Options }

// Options configures the go/types plugin.
type Options struct {
	// FileNamePattern is the filename pattern for generated type files.
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		FileNamePattern: "types.gen.go",
	}
}

// New creates a new go/types plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "go/types" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"go"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return nil }

// Check verifies generated files are up-to-date. Currently unimplemented.
func (p *Plugin) Check(*plugin.Request) error { return nil }

var goPostWriter = &exec.PostWriter{
	Extensions: []string{".go"},
	Commands:   [][]string{{"gofmt", "-w"}},
}

// PostWrite runs gofmt on all generated Go files.
func (p *Plugin) PostWrite(files []string) error {
	return goPostWriter.PostWrite(files)
}

// Generate produces Go type definitions for structs, enums, and typedefs with @go flag.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	gen := &framework.Generator{
		Domain:          "go",
		FilePattern:     p.Options.FileNamePattern,
		FileGenerator:   &goFileGenerator{},
		MergeByName:     false,
		CollectTypeDefs: true,
		CollectEnums:    true,
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

	ctx := &resolver.Context{
		Table:                         table,
		OutputPath:                    outputPath,
		Namespace:                     namespace,
		RepoRoot:                      repoRoot,
		DomainName:                    "go",
		SubstituteDefaultedTypeParams: true,
	}

	r := &resolver.Resolver{
		Formatter:       GoFormatter(),
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

	for _, td := range typeDefs {
		if !omit.IsType(td, "go") {
			data.TypeDefs = append(data.TypeDefs, processTypeDef(td, data))
		}
	}

	for _, e := range enums {
		if e.Namespace == namespace && !omit.IsType(e, "go") {
			data.Enums = append(data.Enums, processEnum(e))
		}
	}

	for _, entry := range structs {
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
	name := getGoName(td)
	if name == "" {
		name = td.Name
	}

	switch form := td.Form.(type) {
	case resolution.DistinctForm:
		result := typeDefData{
			Name:     name,
			BaseType: data.resolver.ResolveTypeRef(form.Base, data.ctx),
			IsAlias:  false,
		}
		for _, tp := range form.TypeParams {
			if tp.HasDefault() {
				continue
			}
			result.TypeParams = append(result.TypeParams, processTypeParam(tp, data))
		}
		result.IsGeneric = len(result.TypeParams) > 0
		return result
	case resolution.AliasForm:
		targetRef := form.Target
		if targetResolved, ok := targetRef.Resolve(data.table); ok {
			if targetForm, ok := targetResolved.Form.(resolution.StructForm); ok {
				var nonDefaultedParams []resolution.TypeParam
				for _, tp := range targetForm.TypeParams {
					if !tp.HasDefault() {
						nonDefaultedParams = append(nonDefaultedParams, tp)
					}
				}
				providedArgs := len(targetRef.TypeArgs)
				if providedArgs < len(nonDefaultedParams) {
					newTypeArgs := make([]resolution.TypeRef, len(nonDefaultedParams))
					copy(newTypeArgs, targetRef.TypeArgs)
					for i := providedArgs; i < len(nonDefaultedParams); i++ {
						if nonDefaultedParams[i].Optional {
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
			IsAlias:  true,
		}
		for _, tp := range form.TypeParams {
			if tp.HasDefault() {
				continue
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

	if name := domain.GetStringFromType(entry, "go", "name"); name != "" {
		sd.Name = name
	}

	for _, tp := range form.TypeParams {
		if tp.HasDefault() {
			continue
		}
		sd.TypeParams = append(sd.TypeParams, processTypeParam(tp, data))
	}
	sd.IsGeneric = len(sd.TypeParams) > 0

	if len(form.Extends) > 0 {
		if len(form.OmittedFields) > 0 {
			for _, field := range resolution.UnifiedFields(entry, data.table) {
				sd.Fields = append(sd.Fields, processField(field, data))
			}
			sd.ExtraFields = domain.GetAllStringsFromType(entry, "go", "fields")
			for _, imp := range domain.GetAllStringsFromType(entry, "go", "imports") {
				data.imports.AddExternal(imp)
			}
			return sd
		}

		if resolver.HasFieldConflicts(form.Extends, data.table) {
			for _, field := range resolution.UnifiedFields(entry, data.table) {
				sd.Fields = append(sd.Fields, processField(field, data))
			}
			sd.ExtraFields = domain.GetAllStringsFromType(entry, "go", "fields")
			for _, imp := range domain.GetAllStringsFromType(entry, "go", "imports") {
				data.imports.AddExternal(imp)
			}
			return sd
		}

		sd.HasExtends = true
		for _, extendsRef := range form.Extends {
			parent, ok := extendsRef.Resolve(data.table)
			if ok {
				sd.ExtendsTypes = append(sd.ExtendsTypes, resolveExtendsType(extendsRef, parent, data))
			}
		}

		for _, field := range form.Fields {
			sd.Fields = append(sd.Fields, processField(field, data))
		}
		sd.ExtraFields = domain.GetAllStringsFromType(entry, "go", "fields")
		for _, imp := range domain.GetAllStringsFromType(entry, "go", "imports") {
			data.imports.AddExternal(imp)
		}
		return sd
	}

	for _, field := range resolution.UnifiedFields(entry, data.table) {
		sd.Fields = append(sd.Fields, processField(field, data))
	}

	sd.ExtraFields = domain.GetAllStringsFromType(entry, "go", "fields")

	for _, imp := range domain.GetAllStringsFromType(entry, "go", "imports") {
		data.imports.AddExternal(imp)
	}

	return sd
}

func processTypeParam(tp resolution.TypeParam, data *templateData) typeParamData {
	tpd := typeParamData{
		Name:       tp.Name,
		Constraint: "any",
	}

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

	var args []string
	if targetType != nil {
		if form, ok := targetType.Form.(resolution.StructForm); ok {
			for i, arg := range typeArgs {
				if i < len(form.TypeParams) && form.TypeParams[i].HasDefault() {
					continue
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

	name := getGoName(parent)
	if name == "" {
		name = parent.Name
	}

	if parent.Namespace == data.Namespace && (targetOutputPath == "" || targetOutputPath == data.OutputPath) {
		return buildGenericType(name, extendsRef.TypeArgs, &parent, data)
	}

	if targetOutputPath == "" {
		return name
	}
	alias := gointernal.DerivePackageAlias(targetOutputPath, data.Package)
	data.imports.AddInternal(alias, resolveGoImportPath(targetOutputPath, data.repoRoot))
	return fmt.Sprintf("%s.%s", alias, buildGenericType(name, extendsRef.TypeArgs, &parent, data))
}

type templateData struct {
	imports    *gointernal.ImportManager
	table      *resolution.Table
	resolver   *resolver.Resolver
	ctx        *resolver.Context
	Package    string
	OutputPath string
	Namespace  string
	repoRoot   string
	Structs    []structData
	Enums      []enumData
	TypeDefs   []typeDefData
}

// HasImports returns true if any imports are needed.
func (d *templateData) HasImports() bool { return d.imports.HasImports() }

// ExternalImports returns sorted external imports.
func (d *templateData) ExternalImports() []string { return d.imports.ExternalImports() }

// InternalImports returns sorted internal imports.
func (d *templateData) InternalImports() []gointernal.InternalImportData {
	return d.imports.InternalImports()
}

type structData struct {
	Name         string
	Doc          string
	AliasOf      string
	Fields       []fieldData
	TypeParams   []typeParamData
	ExtendsTypes []string
	ExtraFields  []string
	IsGeneric    bool
	IsAlias      bool
	HasExtends   bool
}

type typeParamData struct {
	Name       string
	Constraint string
}

type fieldData struct {
	GoName         string
	GoType         string
	JSONName       string
	Doc            string
	IsOptional     bool
	IsHardOptional bool
}

// TagSuffix returns the JSON/msgpack tag suffix for the field.
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
	TypeParams []typeParamData
	IsAlias    bool
	IsGeneric  bool
}

var templateFuncs = template.FuncMap{
	"join":      strings.Join,
	"formatDoc": doc.FormatGo,
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
{{- if .Doc}}
{{formatDoc .Name .Doc}}
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
	{{formatDoc .GoName .Doc | printf "%s"}}
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
	{{formatDoc .GoName .Doc | printf "%s"}}
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
