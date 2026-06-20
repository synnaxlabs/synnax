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
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	goprimitives "github.com/synnaxlabs/oracle/plugin/go/primitives"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
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

// Generate produces Go type definitions for structs, enums, and typedefs with @go flag.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	gen := &framework.Generator{
		Domain:          "go",
		FilePattern:     p.Options.FileNamePattern,
		FileGenerator:   &goFileGenerator{},
		MergeByName:     false,
		CollectTypeDefs: true,
		CollectEnums:    true,
		CollectUnions:   true,
	}
	return gen.Generate(req)
}

// goFileGenerator implements framework.FileGenerator for Go code generation.
type goFileGenerator struct{}

func (g *goFileGenerator) GenerateFile(ctx *framework.GenerateContext) (string, error) {
	content, err := GenerateGoFile(ctx.OutputPath, ctx.Structs, ctx.Enums, ctx.TypeDefs, ctx.Unions, ctx.Table, ctx.RepoRoot)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// GenerateGoFile generates a Go types file for the given structs, enums, and
// type definitions at the specified output path. Exported for use by the
// migrate plugin to generate frozen type definitions. ImportOverrides maps
// original import paths to replacements (nil for normal operation).
func GenerateGoFile(
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	typeDefs []resolution.Type,
	unions []resolution.Type,
	table *resolution.Table,
	repoRoot string,
	importOverrides ...map[string]string,
) ([]byte, error) {
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	} else if len(unions) > 0 {
		namespace = unions[0].Namespace
	} else if len(typeDefs) > 0 {
		namespace = typeDefs[0].Namespace
	} else if len(enums) > 0 {
		namespace = enums[0].Namespace
	}

	pkg := naming.DerivePackageName(outputPath)
	imports := imports.NewManager()

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

	for _, entry := range unions {
		if omit.IsType(entry, "go") {
			continue
		}
		data.Unions = append(data.Unions, processUnion(entry, data))
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
	name := naming.GetGoName(e)
	form := e.Form.(resolution.EnumForm)
	values := make([]enumValueData, 0, len(form.Values))
	for _, v := range form.Values {
		values = append(values, enumValueData{
			Name:     naming.ToPascalCase(v.Name),
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
		Name:        name,
		Doc:         doc.Get(e.Domains),
		Values:      values,
		IsIntEnum:   form.IsIntEnum,
		StartsAtOne: startsAtOne,
	}
}

func processTypeDef(td resolution.Type, data *templateData) typeDefData {
	name := naming.GetGoName(td)

	switch form := td.Form.(type) {
	case resolution.DistinctForm:
		result := typeDefData{
			Name:     name,
			Doc:      doc.Get(td.Domains),
			BaseType: data.resolver.ResolveTypeRef(form.Base, data.ctx),
			IsAlias:  false,
		}
		for _, tp := range resolution.NonDefaultedTypeParams(form.TypeParams) {
			result.TypeParams = append(result.TypeParams, processTypeParam(tp, data))
		}
		result.IsGeneric = len(result.TypeParams) > 0
		return result
	case resolution.AliasForm:
		targetRef := form.Target
		if targetResolved, ok := targetRef.Resolve(data.table); ok {
			if targetForm, ok := targetResolved.Form.(resolution.StructForm); ok {
				nonDefaultedParams := resolution.NonDefaultedTypeParams(targetForm.TypeParams)
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
			Doc:      doc.Get(td.Domains),
			BaseType: baseType,
			IsAlias:  true,
		}
		for _, tp := range resolution.NonDefaultedTypeParams(form.TypeParams) {
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

	for _, tp := range resolution.NonDefaultedTypeParams(form.TypeParams) {
		sd.TypeParams = append(sd.TypeParams, processTypeParam(tp, data))
	}
	sd.IsGeneric = len(sd.TypeParams) > 0

	if len(form.Extends) > 0 {
		// Flatten (rather than embed the parent) when fields are omitted, parents
		// conflict, or a field removes an inherited domain — none can be expressed
		// through Go struct embedding.
		if len(form.OmittedFields) > 0 ||
			resolver.HasFieldConflicts(form.Extends, data.table) ||
			resolver.HasDomainOmissions(form) {
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

	genMethods := !sd.IsGeneric
	for _, field := range resolution.UnifiedFields(entry, data.table) {
		sd.Fields = append(sd.Fields, processField(field, data))
		if !genMethods {
			continue
		}
		sd.DefaultFills = append(sd.DefaultFills, goDefaultFills(field, data)...)
		if step, ok := goRecurseStep(field, data, defaultsHasOwn, neverSkip); ok {
			sd.DefaultRecurse = append(sd.DefaultRecurse, step)
		}
		if validateSkip(field, data) {
			continue
		}
		if chk, ok := goEnumCheck(field, data); ok {
			sd.EnumChecks = append(sd.EnumChecks, chk)
		}
		sd.ConstraintChecks = append(sd.ConstraintChecks, goConstraintChecks(field, data)...)
		if step, ok := goRecurseStep(field, data, validateHasOwn, validateSkip); ok {
			sd.ValidateRecurse = append(sd.ValidateRecurse, step)
		}
	}
	if len(sd.EnumChecks) > 0 || len(sd.ConstraintChecks) > 0 || len(sd.ValidateRecurse) > 0 {
		data.imports.AddExternal(validateImportPath)
	}
	if hasSliceRecurse(sd.ValidateRecurse) {
		data.imports.AddExternal(strconvImportPath)
	}
	if len(sd.Name) > 0 {
		sd.Receiver = strings.ToLower(sd.Name[:1])
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
	if resolution.IsConstraint(constraint.Name) {
		return constraint.Name
	}
	if resolution.IsPrimitive(constraint.Name) {
		switch constraint.Name {
		case "record":
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
	if field.Optional && !strings.HasPrefix(goType, "[]") && !strings.HasPrefix(goType, "map[") && !strings.HasPrefix(goType, "msgpack.EncodedJSON") {
		goType = "*" + goType
	}
	// An optional slice or map stays a plain nilable container (not a pointer), so a
	// nil slice ("not loaded") must serialize as null, not be omitted; otherwise it is
	// indistinguishable from a present-but-empty slice. So these fields drop omitempty.
	isOptionalContainer := field.Optional &&
		(strings.HasPrefix(goType, "[]") || strings.HasPrefix(goType, "map["))
	return fieldData{
		GoName:              naming.GetFieldName(field),
		GoType:              goType,
		JSONName:            casing.FieldSnake(field.Name),
		IsOptional:          field.Optional,
		IsOptionalContainer: isOptionalContainer,
		Doc:                 doc.Get(field.Domains),
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

	name := naming.GetGoName(parent)

	if parent.Namespace == data.Namespace && (targetOutputPath == "" || targetOutputPath == data.OutputPath) {
		return buildGenericType(name, extendsRef.TypeArgs, &parent, data)
	}

	if targetOutputPath == "" {
		return name
	}
	alias := naming.DerivePackageAlias(targetOutputPath, data.Package)
	data.imports.AddInternal(alias, resolveGoImportPath(targetOutputPath, data.repoRoot))
	return fmt.Sprintf("%s.%s", alias, buildGenericType(name, extendsRef.TypeArgs, &parent, data))
}

type templateData struct {
	imports    *imports.Manager
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
	Unions     []unionData
}

// HasImports returns true if any imports are needed.
func (d *templateData) HasImports() bool { return d.imports.HasImports() }

// ExternalImports returns sorted external imports.
func (d *templateData) ExternalImports() []string { return d.imports.ExternalImports() }

// InternalImports returns sorted internal imports.
func (d *templateData) InternalImports() []imports.InternalImportData {
	return d.imports.InternalImports()
}

type structData struct {
	Name             string
	Doc              string
	AliasOf          string
	Receiver         string
	Fields           []fieldData
	TypeParams       []typeParamData
	ExtendsTypes     []string
	ExtraFields      []string
	DefaultFills     []defaultFillData
	DefaultRecurse   []recurseStepData
	EnumChecks       []enumCheckData
	ConstraintChecks []constraintCheckData
	ValidateRecurse  []recurseStepData
	IsGeneric        bool
	IsAlias          bool
	HasExtends       bool
}

type typeParamData struct {
	Name       string
	Constraint string
}

type fieldData struct {
	GoName              string
	GoType              string
	JSONName            string
	Doc                 string
	IsOptional          bool
	IsOptionalContainer bool
}

// TagSuffix returns the JSON/msgpack tag suffix for the field. Optional containers
// (nilable slices/maps) deliberately omit `,omitempty` so a nil container serializes
// as null (not loaded) distinctly from a present empty container.
func (f fieldData) TagSuffix() string {
	if f.IsOptional && !f.IsOptionalContainer {
		return ",omitempty"
	}
	return ""
}

type enumData struct {
	Name        string
	Doc         string
	Values      []enumValueData
	IsIntEnum   bool
	StartsAtOne bool
}

// Receiver returns the single-letter, lowercased method receiver name for the enum type
// (e.g. "n" for Notation).
func (e enumData) Receiver() string { return strings.ToLower(e.Name[:1]) }

type enumValueData struct {
	Name     string
	Value    string
	IntValue int64
}

type typeDefData struct {
	Name       string
	Doc        string
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
{{- if .Doc}}

{{formatDoc .Name .Doc}}
{{- end}}
{{- if .IsAlias}}
type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} = {{.BaseType}}
{{- else}}
type {{.Name}}{{if .IsGeneric}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{$tp.Name}} {{$tp.Constraint}}{{end}}]{{end}} {{.BaseType}}
{{- end}}
{{- end}}
{{- range $enum := .Enums}}

{{- if $enum.Doc}}
{{formatDoc $enum.Name $enum.Doc}}
{{- end}}
{{- if $enum.IsIntEnum}}
type {{$enum.Name}} uint8

{{"//"}}go:generate stringer -type={{$enum.Name}}

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

{{- if not $enum.Doc}}
{{- end}}
type {{$enum.Name}} string

const (
{{- range $enum.Values}}
	{{$enum.Name}}{{.Name}} {{$enum.Name}} = "{{.Value}}"
{{- end}}
)
{{- if $enum.Values}}

// IsValid reports whether {{$enum.Receiver}} is one of the defined {{$enum.Name}} values.
func ({{$enum.Receiver}} {{$enum.Name}}) IsValid() bool {
	switch {{$enum.Receiver}} {
	case {{range $i, $v := $enum.Values}}{{if $i}}, {{end}}{{$enum.Name}}{{$v.Name}}{{end}}:
		return true
	default:
		return false
	}
}
{{- end}}
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
{{- $s := .}}
{{- if or .DefaultFills .DefaultRecurse}}

func ({{$s.Receiver}} {{$s.Name}}) ApplyDefaults() {{$s.Name}} {
{{- range $s.DefaultFills}}
	if {{$s.Receiver}}.{{.GoName}} == {{.ZeroLit}} {
		{{$s.Receiver}}.{{.GoName}} = {{.Expr}}
	}
{{- end}}
{{- range $s.DefaultRecurse}}
{{- if eq (printf "%s" .Kind) "value"}}
	{{$s.Receiver}}.{{.GoName}} = {{$s.Receiver}}.{{.GoName}}.ApplyDefaults()
{{- else if eq (printf "%s" .Kind) "pointer"}}
	if {{$s.Receiver}}.{{.GoName}} != nil {
		applied := {{$s.Receiver}}.{{.GoName}}.ApplyDefaults()
		{{$s.Receiver}}.{{.GoName}} = &applied
	}
{{- else if eq (printf "%s" .Kind) "slice"}}
	for i := range {{$s.Receiver}}.{{.GoName}} {
		{{$s.Receiver}}.{{.GoName}}[i] = {{$s.Receiver}}.{{.GoName}}[i].ApplyDefaults()
	}
{{- else if eq (printf "%s" .Kind) "map"}}
	for key, value := range {{$s.Receiver}}.{{.GoName}} {
		{{$s.Receiver}}.{{.GoName}}[key] = value.ApplyDefaults()
	}
{{- end}}
{{- end}}
	return {{$s.Receiver}}
}
{{- end}}
{{- if or .EnumChecks .ConstraintChecks .ValidateRecurse}}

func ({{$s.Receiver}} {{$s.Name}}) Validate() error {
	v := validate.New("{{$s.Name}}")
{{- range $s.EnumChecks}}
	v.Ternaryf("{{.FieldName}}", !{{$s.Receiver}}.{{.GoName}}.IsValid(), "invalid {{.FieldName}}: %v", {{$s.Receiver}}.{{.GoName}})
{{- end}}
{{- range $s.ConstraintChecks}}
{{- if eq .Kind "non_empty_string"}}
	validate.NotEmptyString(v, "{{.FieldName}}", {{$s.Receiver}}.{{.GoName}})
{{- else if eq .Kind "non_zero"}}
	validate.NonZero(v, "{{.FieldName}}", {{$s.Receiver}}.{{.GoName}})
{{- else if eq .Kind "min_len"}}
	v.Ternaryf("{{.FieldName}}", len({{$s.Receiver}}.{{.GoName}}) < {{.Arg}}, "must be at least {{.Arg}} characters long")
{{- else if eq .Kind "max_len"}}
	v.Ternaryf("{{.FieldName}}", len({{$s.Receiver}}.{{.GoName}}) > {{.Arg}}, "must be at most {{.Arg}} characters long")
{{- else if eq .Kind "ge"}}
	validate.GreaterThanEq(v, "{{.FieldName}}", {{$s.Receiver}}.{{.GoName}}, {{.Arg}})
{{- else if eq .Kind "le"}}
	validate.LessThanEq(v, "{{.FieldName}}", {{$s.Receiver}}.{{.GoName}}, {{.Arg}})
{{- end}}
{{- end}}
{{- range $s.ValidateRecurse}}
{{- if eq (printf "%s" .Kind) "value"}}
	v.Exec(func() error { return validate.PathedError({{$s.Receiver}}.{{.GoName}}.Validate(), "{{.JSONName}}") })
{{- else if eq (printf "%s" .Kind) "pointer"}}
	if {{$s.Receiver}}.{{.GoName}} != nil {
		v.Exec(func() error { return validate.PathedError({{$s.Receiver}}.{{.GoName}}.Validate(), "{{.JSONName}}") })
	}
{{- else if eq (printf "%s" .Kind) "slice"}}
	for i := range {{$s.Receiver}}.{{.GoName}} {
		v.Exec(func() error { return validate.PathedError({{$s.Receiver}}.{{.GoName}}[i].Validate(), "{{.JSONName}}", strconv.Itoa(i)) })
	}
{{- else if eq (printf "%s" .Kind) "map"}}
	for key, value := range {{$s.Receiver}}.{{.GoName}} {
		v.Exec(func() error { return validate.PathedError(value.Validate(), "{{.JSONName}}", key) })
	}
{{- end}}
{{- end}}
	return v.Error()
}
{{- end}}
{{end -}}
{{- range .Unions}}
{{- $u := .}}

type {{.DiscType}} string

const (
{{- range .Variants}}
	{{.ConstName}} {{$u.DiscType}} = "{{.Value}}"
{{- end}}
)

type {{.InterfaceName}} interface {
	{{.Marker}}()
}
{{range .Variants}}
{{- if .Doc}}
{{formatDoc .TypeName .Doc}}
{{- end}}
type {{.TypeName}} struct {
{{- range .Embeds}}
	{{.}}
{{- end}}
{{- range .Fields}}
{{- if .Doc}}
	{{formatDoc .GoName .Doc | printf "%s"}}
{{- end}}
	{{.GoName}} {{.GoType}} ` + "`" + `json:"{{.JSONName}}{{.TagSuffix}}" msgpack:"{{.JSONName}}{{.TagSuffix}}"` + "`" + `
{{- end}}
}

func ({{.TypeName}}) {{$u.Marker}}() {}
{{- $vt := .}}
{{- if .NeedsApplyDefaults}}

func ({{$vt.Receiver}} {{$vt.TypeName}}) ApplyDefaults() {{$vt.TypeName}} {
{{- range $vt.DefaultRecurse}}
{{- if eq (printf "%s" .Kind) "value"}}
	{{$vt.Receiver}}.{{.GoName}} = {{$vt.Receiver}}.{{.GoName}}.ApplyDefaults()
{{- else if eq (printf "%s" .Kind) "pointer"}}
	if {{$vt.Receiver}}.{{.GoName}} != nil {
		applied := {{$vt.Receiver}}.{{.GoName}}.ApplyDefaults()
		{{$vt.Receiver}}.{{.GoName}} = &applied
	}
{{- else if eq (printf "%s" .Kind) "slice"}}
	for i := range {{$vt.Receiver}}.{{.GoName}} {
		{{$vt.Receiver}}.{{.GoName}}[i] = {{$vt.Receiver}}.{{.GoName}}[i].ApplyDefaults()
	}
{{- else if eq (printf "%s" .Kind) "map"}}
	for key, value := range {{$vt.Receiver}}.{{.GoName}} {
		{{$vt.Receiver}}.{{.GoName}}[key] = value.ApplyDefaults()
	}
{{- end}}
{{- end}}
	return {{$vt.Receiver}}
}
{{- end}}
{{- if .NeedsValidate}}

func ({{$vt.Receiver}} {{$vt.TypeName}}) Validate() error {
	v := validate.New("{{$vt.TypeName}}")
{{- range $vt.ValidateRecurse}}
{{- if eq (printf "%s" .Kind) "value"}}
{{- if .JSONName}}
	v.Exec(func() error { return validate.PathedError({{$vt.Receiver}}.{{.GoName}}.Validate(), "{{.JSONName}}") })
{{- else}}
	v.Exec({{$vt.Receiver}}.{{.GoName}}.Validate)
{{- end}}
{{- else if eq (printf "%s" .Kind) "pointer"}}
	if {{$vt.Receiver}}.{{.GoName}} != nil {
		v.Exec(func() error { return validate.PathedError({{$vt.Receiver}}.{{.GoName}}.Validate(), "{{.JSONName}}") })
	}
{{- else if eq (printf "%s" .Kind) "slice"}}
	for i := range {{$vt.Receiver}}.{{.GoName}} {
		v.Exec(func() error { return validate.PathedError({{$vt.Receiver}}.{{.GoName}}[i].Validate(), "{{.JSONName}}", strconv.Itoa(i)) })
	}
{{- else if eq (printf "%s" .Kind) "map"}}
	for key, value := range {{$vt.Receiver}}.{{.GoName}} {
		v.Exec(func() error { return validate.PathedError(value.Validate(), "{{.JSONName}}", key) })
	}
{{- end}}
{{- end}}
	return v.Error()
}
{{- end}}
{{end -}}
{{if .Doc}}{{formatDoc .Name .Doc}}
{{end -}}
type {{.Name}} struct {
	Variant {{.InterfaceName}}
}

func (u {{.Name}}) MarshalJSON() ([]byte, error) {
	if u.Variant == nil {
		return []byte("null"), nil
	}
	var t {{.DiscType}}
	switch u.Variant.(type) {
{{- range .Variants}}
	case {{.TypeName}}:
		t = {{.ConstName}}
{{- end}}
	default:
		return nil, errors.Newf("{{.Name}}: nil or unknown variant %T", u.Variant)
	}
	raw, err := json.Marshal(u.Variant)
	if err != nil {
		return nil, err
	}
	fields := map[string]json.RawMessage{}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	tag, err := json.Marshal(t)
	if err != nil {
		return nil, err
	}
	fields["{{.DiscJSONName}}"] = tag
	return json.Marshal(fields)
}

func (u *{{.Name}}) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		u.Variant = nil
		return nil
	}
	var disc struct {
		Type {{.DiscType}} ` + "`" + `json:"{{.DiscJSONName}}"` + "`" + `
	}
	if err := json.Unmarshal(data, &disc); err != nil {
		return err
	}
	switch disc.Type {
{{- range .Variants}}
	case {{.ConstName}}:
		var v {{.TypeName}}
		if err := json.Unmarshal(data, &v); err != nil {
			return err
		}
		u.Variant = v
{{- end}}
	default:
		return errors.Newf("{{.Name}}: unknown {{.DiscJSONName}} %q", disc.Type)
	}
	return nil
}
{{- if .NeedsApplyDefaults}}

func (u {{.Name}}) ApplyDefaults() {{.Name}} {
	switch variant := u.Variant.(type) {
{{- range .Variants}}
{{- if .NeedsApplyDefaults}}
	case {{.TypeName}}:
		u.Variant = variant.ApplyDefaults()
{{- end}}
{{- end}}
	}
	return u
}
{{- end}}
{{- if .NeedsValidate}}

func (u {{.Name}}) Validate() error {
	switch variant := u.Variant.(type) {
{{- range .Variants}}
{{- if .NeedsValidate}}
	case {{.TypeName}}:
		return variant.Validate()
{{- end}}
{{- end}}
	}
	return nil
}
{{- end}}
{{end -}}
`))
