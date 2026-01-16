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
	"path/filepath"
	"strings"
	"text/template"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	pbprimitives "github.com/synnaxlabs/oracle/plugin/pb/primitives"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const defaultModulePrefix = "github.com/synnaxlabs/synnax/"

// primitiveMapper is the Protobuf-specific primitive type mapper.
var primitiveMapper = pbprimitives.Mapper()

type Plugin struct{ Options Options }

type Options struct {
	FileNamePattern string
}

func DefaultOptions() Options {
	return Options{
		FileNamePattern: "types.gen.proto",
	}
}

func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string { return "pb/types" }

func (p *Plugin) Domains() []string { return []string{"pb"} }

func (p *Plugin) Requires() []string { return nil }

func (p *Plugin) Check(req *plugin.Request) error { return nil }

var (
	bufFormatCmd   = []string{"buf", "format", "-w"}
	bufGenerateCmd = []string{"buf", "generate"}
)

func (p *Plugin) PostWrite(files []string) error {
	if len(files) == 0 {
		return nil
	}
	firstFile := files[0]
	repoRoot := gomod.FindRepoRoot(firstFile)
	if repoRoot == "" {
		return errors.New("could not determine repo root from file paths")
	}
	// buf format -w and buf generate don't accept file arguments - they operate
	// on the entire directory. Run both without file arguments from the repo root.
	if err := exec.OnFiles(bufFormatCmd, nil, repoRoot); err != nil {
		return err
	}
	return exec.OnFiles(bufGenerateCmd, nil, repoRoot)
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	structCollector := framework.NewCollector("pb", req).
		WithPathFunc(output.GetPBPath).
		WithSkipFunc(nil)
	if err := structCollector.AddAll(req.Resolutions.StructTypes()); err != nil {
		return nil, err
	}

	pbPathFunc := func(typ resolution.Type, table *resolution.Table) string {
		return enum.FindPBOutputPath(typ, table)
	}
	err := structCollector.ForEach(func(outputPath string, structs []resolution.Type) error {
		namespace := ""
		if len(structs) > 0 {
			namespace = structs[0].Namespace
		}
		enums := enum.CollectReferenced(structs, req.Resolutions)
		enums = framework.MergeTypes(enums, enum.CollectNamespaceEnums(namespace, outputPath, req.Resolutions, "pb", pbPathFunc))
		content, err := p.generateFile(outputPath, structs, enums, req.Resolutions, req.RepoRoot)
		if err != nil {
			return errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		// Use namespace-based filename: {namespace}.proto
		filename := namespace + ".proto"
		resp.Files = append(resp.Files, plugin.File{
			Path:    fmt.Sprintf("%s/%s", outputPath, filename),
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
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	table *resolution.Table,
	repoRoot string,
) ([]byte, error) {
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	}

	data := &templateData{
		Package:    derivePackageName(outputPath, structs),
		GoPackage:  deriveGoPackage(outputPath, structs, repoRoot),
		OutputPath: outputPath,
		Namespace:  namespace,
		Messages:   make([]messageData, 0, len(structs)),
		Enums:      make([]enumData, 0, len(enums)),
		imports:    newImportManager(),
		table:      table,
		repoRoot:   repoRoot,
	}

	for _, e := range enums {
		if e.Namespace == namespace && !omit.IsType(e, "pb") {
			data.Enums = append(data.Enums, p.processEnum(e))
		}
	}

	for _, entry := range structs {
		if omit.IsType(entry, "pb") {
			continue
		}
		msg, err := p.processStruct(entry, data)
		if err != nil {
			return nil, err
		}
		if !msg.Skip {
			data.Messages = append(data.Messages, msg)
		}
	}

	var buf bytes.Buffer
	if err := fileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func derivePackageName(outputPath string, structs []resolution.Type) string {
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	} else {
		// Fallback: derive namespace from output path
		parts := strings.Split(outputPath, "/")
		if len(parts) >= 1 {
			if parts[len(parts)-1] == "pb" && len(parts) >= 2 {
				namespace = parts[len(parts)-2]
			} else {
				namespace = parts[len(parts)-1]
			}
		} else {
			namespace = filepath.Base(outputPath)
		}
	}
	prefix := deriveLayerPrefix(outputPath)
	return prefix + "." + namespace + ".pb"
}

func deriveLayerPrefix(outputPath string) string {
	parts := strings.Split(outputPath, "/")
	// For core/pkg/{layer}/... paths, use the layer as prefix
	if len(parts) >= 3 && parts[0] == "core" && parts[1] == "pkg" {
		return parts[2]
	}
	// For other paths, use the first component
	if len(parts) >= 1 && parts[0] != "" {
		return parts[0]
	}
	return "synnax"
}

func deriveGoPackage(outputPath string, structs []resolution.Type, repoRoot string) string {
	return gomod.ResolveImportPath(outputPath, repoRoot, defaultModulePrefix)
}

func (p *Plugin) processEnum(e resolution.Type) enumData {
	form, ok := e.Form.(resolution.EnumForm)
	if !ok {
		return enumData{Name: e.Name}
	}

	ed := enumData{
		Name:   e.Name,
		Doc:    doc.Get(e.Domains),
		Values: make([]enumValueData, 0, len(form.Values)),
	}

	prefix := toScreamingSnake(e.Name) + "_"

	for i, v := range form.Values {
		number := i // Default to sequential for string enums
		switch n := v.Value.(type) {
		case int64:
			number = int(n)
		case int:
			number = n
		}
		ed.Values = append(ed.Values, enumValueData{
			Name:   prefix + toScreamingSnake(v.Name),
			Number: number,
		})
	}

	return ed
}

func (p *Plugin) processStruct(entry resolution.Type, data *templateData) (messageData, error) {
	_, ok := entry.Form.(resolution.StructForm)
	if !ok {
		if _, isAlias := entry.Form.(resolution.AliasForm); isAlias {
			return messageData{Skip: true}, nil
		}
		return messageData{Skip: true}, nil
	}

	name := getPBName(entry)
	if name == "" {
		name = entry.Name
	}

	md := messageData{
		Name:   name,
		Doc:    doc.Get(entry.Domains),
		Fields: make([]fieldData, 0),
	}

	fieldNumber := 1
	for _, field := range resolution.UnifiedFields(entry, data.table) {
		fd, err := p.processField(field, fieldNumber, data)
		if err != nil {
			return messageData{}, errors.Wrapf(err, "struct %q", entry.Name)
		}
		md.Fields = append(md.Fields, fd)
		fieldNumber++
	}

	return md, nil
}

func (p *Plugin) processField(field resolution.Field, number int, data *templateData) (fieldData, error) {
	if p.isFixedSizeUint8Array(field.Type, data.table) {
		return fieldData{
			Name:       toSnakeCase(field.Name),
			Doc:        doc.Get(field.Domains),
			Type:       "bytes",
			Number:     number,
			IsOptional: field.IsHardOptional,
			IsRepeated: false, // bytes is not repeated
		}, nil
	}

	isArray := p.isArrayType(field.Type, data.table)

	if p.isNestedArrayType(field.Type, data.table) {
		wrapperName, err := p.generateNestedArrayWrapper(field.Type, data)
		if err != nil {
			return fieldData{}, errors.Wrapf(err, "field %q", field.Name)
		}
		return fieldData{
			Name:       toSnakeCase(field.Name),
			Doc:        doc.Get(field.Domains),
			Type:       wrapperName,
			Number:     number,
			IsOptional: field.IsHardOptional,
			IsRepeated: true,
		}, nil
	}

	protoType, err := p.typeToProto(field.Type, data)
	if err != nil {
		return fieldData{}, errors.Wrapf(err, "field %q", field.Name)
	}

	return fieldData{
		Name:       toSnakeCase(field.Name),
		Doc:        doc.Get(field.Domains),
		Type:       protoType,
		Number:     number,
		IsOptional: field.IsHardOptional,
		IsRepeated: isArray,
	}, nil
}

func (p *Plugin) isArrayType(typeRef resolution.TypeRef, table *resolution.Table) bool {
	if typeRef.Name == "Array" {
		return true
	}

	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return false
	}

	switch form := resolved.Form.(type) {
	case resolution.BuiltinGenericForm:
		return form.Name == "Array"
	case resolution.AliasForm:
		return p.isArrayType(form.Target, table)
	case resolution.DistinctForm:
		return p.isArrayType(form.Base, table)
	default:
		return false
	}
}

func (p *Plugin) getArrayElementType(typeRef resolution.TypeRef, table *resolution.Table) (resolution.TypeRef, bool) {
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		return typeRef.TypeArgs[0], true
	}

	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return resolution.TypeRef{}, false
	}

	switch form := resolved.Form.(type) {
	case resolution.BuiltinGenericForm:
		if form.Name == "Array" && len(typeRef.TypeArgs) > 0 {
			return typeRef.TypeArgs[0], true
		}
		return resolution.TypeRef{}, false
	case resolution.AliasForm:
		return p.getArrayElementType(form.Target, table)
	case resolution.DistinctForm:
		return p.getArrayElementType(form.Base, table)
	default:
		return resolution.TypeRef{}, false
	}
}

func (p *Plugin) isFixedSizeUint8Array(typeRef resolution.TypeRef, table *resolution.Table) bool {
	arraySize := p.getArraySize(typeRef, table)
	if arraySize == nil {
		return false
	}

	elemType, ok := p.getArrayElementType(typeRef, table)
	if !ok {
		return false
	}

	resolved, ok := elemType.Resolve(table)
	if !ok {
		return elemType.Name == "uint8"
	}

	if prim, ok := resolved.Form.(resolution.PrimitiveForm); ok {
		return prim.Name == "uint8"
	}
	return false
}

func (p *Plugin) getArraySize(typeRef resolution.TypeRef, table *resolution.Table) *int64 {
	if typeRef.Name == "Array" && typeRef.ArraySize != nil {
		return typeRef.ArraySize
	}

	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return nil
	}

	switch form := resolved.Form.(type) {
	case resolution.AliasForm:
		return p.getArraySize(form.Target, table)
	case resolution.DistinctForm:
		return p.getArraySize(form.Base, table)
	default:
		return nil
	}
}

func (p *Plugin) isNestedArrayType(typeRef resolution.TypeRef, table *resolution.Table) bool {
	if !p.isArrayType(typeRef, table) {
		return false
	}
	elemType, ok := p.getArrayElementType(typeRef, table)
	if !ok {
		return false
	}
	return p.isArrayType(elemType, table)
}

func (p *Plugin) getNestedArrayWrapperName(typeRef resolution.TypeRef, table *resolution.Table) string {
	elemType, ok := p.getArrayElementType(typeRef, table)
	if !ok {
		return "ArrayWrapper"
	}

	resolved, ok := elemType.Resolve(table)
	if ok {
		return resolved.Name + "Wrapper"
	}

	if resolution.IsPrimitive(elemType.Name) {
		return cases.Title(language.English).String(elemType.Name) + "Array"
	}

	return "ArrayWrapper"
}

func (p *Plugin) generateNestedArrayWrapper(typeRef resolution.TypeRef, data *templateData) (string, error) {
	elemType, ok := p.getArrayElementType(typeRef, data.table)
	if !ok {
		return "", errors.New("could not get element type for nested array")
	}

	innerElemType, ok := p.getArrayElementType(elemType, data.table)
	if !ok {
		return "", errors.New("could not get inner element type for nested array")
	}

	innerProtoType, err := p.typeToProto(innerElemType, data)
	if err != nil {
		return "", errors.Wrap(err, "could not convert inner element type to proto")
	}

	wrapperName := p.getNestedArrayWrapperName(typeRef, data.table)

	if data.wrapperMessages == nil {
		data.wrapperMessages = make(map[string]bool)
	}
	if !data.wrapperMessages[wrapperName] {
		data.wrapperMessages[wrapperName] = true
		data.Messages = append(data.Messages, messageData{
			Name: wrapperName,
			Fields: []fieldData{
				{Name: "values", Type: innerProtoType, Number: 1, IsRepeated: true},
			},
		})
	}

	return wrapperName, nil
}

func (p *Plugin) typeToProto(typeRef resolution.TypeRef, data *templateData) (string, error) {
	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		if typeRef.TypeParam.HasDefault() {
			return p.typeToProto(*typeRef.TypeParam.Default, data)
		}
		data.imports.add("google/protobuf/any.proto")
		return "google.protobuf.Any", nil
	}

	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return "", errors.Newf("failed to resolve type reference %q", typeRef.Name)
	}

	switch form := resolved.Form.(type) {
	case resolution.PrimitiveForm:
		mapping := primitiveMapper.Map(form.Name)
		if mapping.TargetType == "any" {
			return "", errors.Newf("primitive type %q has no protobuf mapping", form.Name)
		}
		for _, imp := range mapping.Imports {
			data.imports.add(imp.Path)
		}
		return mapping.TargetType, nil

	case resolution.BuiltinGenericForm:
		if form.Name == "Array" {
			if len(typeRef.TypeArgs) > 0 {
				return p.typeToProto(typeRef.TypeArgs[0], data)
			}
			return "", errors.New("Array type has no type arguments")
		}
		if form.Name == "Map" {
			return p.resolveMapType(typeRef, data)
		}
		return "", errors.Newf("unknown builtin generic type %q", form.Name)

	case resolution.StructForm:
		return p.resolveStructType(typeRef, resolved, data)

	case resolution.EnumForm:
		return p.resolveEnumType(resolved, data), nil

	case resolution.AliasForm:
		return p.typeToProto(form.Target, data)

	case resolution.DistinctForm:
		return p.typeToProto(form.Base, data)

	default:
		return "", errors.Newf("unknown type form %T for type %q", resolved.Form, resolved.Name)
	}
}

func (p *Plugin) resolveStructType(typeRef resolution.TypeRef, resolved resolution.Type, data *templateData) (string, error) {
	form, ok := resolved.Form.(resolution.StructForm)
	if !ok {
		if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
			return p.typeToProto(aliasForm.Target, data)
		}
		return "", errors.Newf("expected struct form for type %q, got %T", resolved.Name, resolved.Form)
	}
	_ = form

	pbName := getPBName(resolved)
	if pbName == "" {
		pbName = resolved.Name
	}

	targetOutputPath := output.GetPBPath(resolved)

	if targetOutputPath == data.OutputPath {
		return pbName, nil
	}

	if targetOutputPath == "" {
		return "", errors.Newf("struct %q has no @pb output defined", resolved.Name)
	}

	importPath := targetOutputPath + "/" + resolved.Namespace + ".proto"
	data.imports.add(importPath)

	pkg := derivePackageName(targetOutputPath, []resolution.Type{resolved})
	return fmt.Sprintf(".%s.%s", pkg, pbName), nil
}

func (p *Plugin) resolveEnumType(resolved resolution.Type, data *templateData) string {
	if resolved.Namespace == data.Namespace {
		return resolved.Name
	}

	targetOutputPath := enum.FindPBOutputPath(resolved, data.table)
	if targetOutputPath == "" {
		return "int32"
	}

	importPath := targetOutputPath + "/" + resolved.Namespace + ".proto"
	data.imports.add(importPath)

	pkg := deriveLayerPrefix(targetOutputPath) + "." + resolved.Namespace + ".pb"
	return fmt.Sprintf(".%s.%s", pkg, resolved.Name)
}

func (p *Plugin) resolveMapType(typeRef resolution.TypeRef, data *templateData) (string, error) {
	if len(typeRef.TypeArgs) < 2 {
		return "", errors.Newf("Map type requires 2 type arguments, got %d", len(typeRef.TypeArgs))
	}

	keyType, err := p.typeToProto(typeRef.TypeArgs[0], data)
	if err != nil {
		return "", errors.Wrap(err, "map key type")
	}

	valueType, err := p.typeToProto(typeRef.TypeArgs[1], data)
	if err != nil {
		return "", errors.Wrap(err, "map value type")
	}

	return fmt.Sprintf("map<%s, %s>", keyType, valueType), nil
}

func getPBName(typ resolution.Type) string {
	return domain.GetStringFromType(typ, "pb", "name")
}

func toSnakeCase(s string) string {
	return lo.SnakeCase(s)
}

func toScreamingSnake(s string) string {
	return strings.ToUpper(lo.SnakeCase(s))
}

type importManager struct {
	imports []string
}

func newImportManager() *importManager {
	return &importManager{}
}

func (m *importManager) add(path string) {
	if !lo.Contains(m.imports, path) {
		m.imports = append(m.imports, path)
	}
}

type templateData struct {
	Package         string
	GoPackage       string
	OutputPath      string
	Namespace       string
	Messages        []messageData
	Enums           []enumData
	imports         *importManager
	table           *resolution.Table
	repoRoot        string
	wrapperMessages map[string]bool
}

func (d *templateData) Imports() []string {
	return d.imports.imports
}

type messageData struct {
	Name   string
	Doc    string
	Fields []fieldData
	Skip   bool
}

type fieldData struct {
	Name       string
	Doc        string
	Type       string
	Number     int
	IsOptional bool
	IsRepeated bool
}

type enumData struct {
	Name   string
	Doc    string
	Values []enumValueData
}

type enumValueData struct {
	Name   string
	Number int
}

var templateFuncs = template.FuncMap{
	"formatDoc": doc.FormatProto,
}

var fileTemplate = template.Must(template.New("proto").Funcs(templateFuncs).Parse(`// Code generated by oracle. DO NOT EDIT.

syntax = "proto3";

package {{.Package}};
{{- if .Imports}}
{{range .Imports}}
import "{{.}}";
{{- end}}
{{- end}}

option go_package = "{{.GoPackage}}";
{{- range .Enums}}
{{- if .Doc}}

{{formatDoc .Name .Doc}}
{{- end}}
enum {{.Name}} {
{{- range .Values}}
  {{.Name}} = {{.Number}};
{{- end}}
}
{{- end}}
{{- range .Messages}}
{{- if .Doc}}

{{formatDoc .Name .Doc}}
{{- end}}
message {{.Name}} {
{{- range .Fields}}
{{- if .Doc}}
  {{formatDoc .Name .Doc}}
{{- end}}
  {{if .IsRepeated}}repeated {{else if .IsOptional}}optional {{end}}{{.Type}} {{.Name}} = {{.Number}};
{{- end}}
}
{{- end}}
`))
