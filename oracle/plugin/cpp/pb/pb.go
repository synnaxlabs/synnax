// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package pb provides an Oracle plugin that generates protobuf translator functions
// for C++ types. It produces translator.gen.h and translator.gen.cpp files that convert
// between C++ domain types and protobuf types.
package pb

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	cppprimitives "github.com/synnaxlabs/oracle/plugin/cpp/primitives"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

var primitiveMapper = &cppprimitives.Mapper{}

// Plugin generates protobuf translator functions for C++ types.
type Plugin struct{ Options Options }

// Options configures the cpp/pb plugin.
type Options struct {
	// HeaderFileNamePattern is the filename pattern for the translator header.
	HeaderFileNamePattern string
	// SourceFileNamePattern is the filename pattern for the translator implementation.
	SourceFileNamePattern string
	// DisableFormatter disables clang-format post-processing.
	DisableFormatter bool
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		HeaderFileNamePattern: "translator.gen.h",
		SourceFileNamePattern: "translator.gen.cpp",
	}
}

// New creates a new cpp/pb plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "cpp/pb" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"cpp", "pb"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"cpp/types", "pb/types"} }

// Check verifies generated files are up-to-date.
func (p *Plugin) Check(*plugin.Request) error { return nil }

var clangFormatCmd = []string{"clang-format", "-i"}

// PostWrite runs clang-format on all generated C++ files.
func (p *Plugin) PostWrite(files []string) error {
	if p.Options.DisableFormatter || len(files) == 0 {
		return nil
	}
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

// Generate produces translator functions for structs with both @cpp and @pb flags.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	// Group structs by their cpp output path that also have @pb
	outputStructs := make(map[string][]resolution.Type)
	var outputOrder []string

	for _, entry := range req.Resolutions.StructTypes() {
		// Must have both @cpp output and @pb flag
		cppOutputPath := output.GetPath(entry, "cpp")
		if cppOutputPath == "" {
			continue
		}
		if !hasPBFlag(entry) {
			continue
		}
		if omit.IsType(entry, "cpp") || omit.IsType(entry, "pb") {
			continue
		}

		if req.RepoRoot != "" {
			if err := req.ValidateOutputPath(cppOutputPath); err != nil {
				return nil, errors.Wrapf(err, "invalid output path for struct %s", entry.Name)
			}
		}

		if _, exists := outputStructs[cppOutputPath]; !exists {
			outputOrder = append(outputOrder, cppOutputPath)
		}
		outputStructs[cppOutputPath] = append(outputStructs[cppOutputPath], entry)
	}

	for _, outputPath := range outputOrder {
		structs := outputStructs[outputPath]
		namespace := ""
		if len(structs) > 0 {
			namespace = structs[0].Namespace
		}

		// Collect enums referenced by these structs
		enums := enum.CollectReferenced(structs, req.Resolutions)

		// Generate header file
		headerContent, err := p.generateHeader(outputPath, structs, enums, namespace, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate header for %s", outputPath)
		}
		if len(headerContent) > 0 {
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.HeaderFileNamePattern),
				Content: headerContent,
			})
		}

		// Generate source file
		sourceContent, err := p.generateSource(outputPath, structs, enums, namespace, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate source for %s", outputPath)
		}
		if len(sourceContent) > 0 {
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.SourceFileNamePattern),
				Content: sourceContent,
			})
		}
	}

	return resp, nil
}

func hasPBFlag(t resolution.Type) bool {
	_, hasPB := t.Domains["pb"]
	return hasPB
}

func (p *Plugin) generateHeader(
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	namespace string,
	req *plugin.Request,
) ([]byte, error) {
	data := &headerTemplateData{
		OutputPath:        outputPath,
		Namespace:         deriveNamespace(outputPath),
		PBNamespace:       derivePBNamespace(outputPath),
		Translators:       make([]translatorData, 0, len(structs)),
		EnumTranslators:   make([]enumTranslatorData, 0),
		ForwardDecls:      make([]forwardDeclData, 0),
		includes:          newIncludeManager(),
		table:             req.Resolutions,
		rawNs:             namespace,
		processedEnums:    make(map[string]bool),
		processedStructs:  make(map[string]bool),
	}

	// Add standard includes
	data.includes.addSystem("vector")

	// Include the types header from the same directory
	data.includes.addInternal(fmt.Sprintf("%s/types.gen.h", outputPath))

	// Process structs
	for _, s := range structs {
		if omit.IsType(s, "cpp") || omit.IsType(s, "pb") {
			continue
		}
		form, ok := s.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		// Skip generic structs for now - they require special handling
		if form.IsGeneric() {
			continue
		}

		translator := p.processStructForTranslation(s, form, data, req)
		if translator != nil {
			data.Translators = append(data.Translators, *translator)
		}
	}

	// Process enums
	for _, e := range enums {
		if omit.IsType(e, "cpp") || omit.IsType(e, "pb") {
			continue
		}
		if e.Namespace != namespace {
			continue
		}
		if data.processedEnums[e.QualifiedName] {
			continue
		}
		data.processedEnums[e.QualifiedName] = true

		enumTranslator := p.processEnumForTranslation(e, data)
		if enumTranslator != nil {
			data.EnumTranslators = append(data.EnumTranslators, *enumTranslator)
		}
	}

	if len(data.Translators) == 0 && len(data.EnumTranslators) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	if err := headerTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *Plugin) generateSource(
	outputPath string,
	structs []resolution.Type,
	enums []resolution.Type,
	namespace string,
	req *plugin.Request,
) ([]byte, error) {
	data := &sourceTemplateData{
		OutputPath:         outputPath,
		Namespace:          deriveNamespace(outputPath),
		PBNamespace:        derivePBNamespace(outputPath),
		HeaderFile:         p.Options.HeaderFileNamePattern,
		Translators:        make([]translatorData, 0, len(structs)),
		EnumTranslators:    make([]enumTranslatorData, 0),
		includes:           newIncludeManager(),
		table:              req.Resolutions,
		rawNs:              namespace,
		processedEnums:     make(map[string]bool),
	}

	// Include the header
	data.includes.addInternal(fmt.Sprintf("%s/%s", outputPath, p.Options.HeaderFileNamePattern))

	// Include protobuf headers
	protoInclude := deriveProtoInclude(outputPath, namespace)
	if protoInclude != "" {
		data.includes.addInternal(protoInclude)
	}

	// Process structs
	for _, s := range structs {
		if omit.IsType(s, "cpp") || omit.IsType(s, "pb") {
			continue
		}
		form, ok := s.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		if form.IsGeneric() {
			continue
		}

		translator := p.processStructForTranslation(s, form, data, req)
		if translator != nil {
			data.Translators = append(data.Translators, *translator)
		}
	}

	// Process enums
	for _, e := range enums {
		if omit.IsType(e, "cpp") || omit.IsType(e, "pb") {
			continue
		}
		if e.Namespace != namespace {
			continue
		}
		if data.processedEnums[e.QualifiedName] {
			continue
		}
		data.processedEnums[e.QualifiedName] = true

		enumTranslator := p.processEnumForTranslation(e, data)
		if enumTranslator != nil {
			data.EnumTranslators = append(data.EnumTranslators, *enumTranslator)
		}
	}

	if len(data.Translators) == 0 && len(data.EnumTranslators) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	if err := sourceTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type templateDataBase interface {
	getTable() *resolution.Table
	getRawNs() string
	getIncludes() *includeManager
}

func (p *Plugin) processStructForTranslation(
	s resolution.Type,
	form resolution.StructForm,
	data templateDataBase,
	req *plugin.Request,
) *translatorData {
	cppName := domain.GetName(s, "cpp")
	pbName := getPBName(s)

	translator := &translatorData{
		CppName:  cppName,
		PBName:   pbName,
		Fields:   make([]fieldTranslatorData, 0),
	}

	// Process fields
	for _, field := range resolution.UnifiedFields(s, data.getTable()) {
		fieldData := p.processFieldForTranslation(field, data)
		translator.Fields = append(translator.Fields, fieldData)
	}

	return translator
}

func (p *Plugin) processFieldForTranslation(
	field resolution.Field,
	data templateDataBase,
) fieldTranslatorData {
	fieldName := field.Name
	pbFieldName := field.Name

	// Determine conversion expressions
	forwardExpr, backwardExpr := p.generateFieldConversion(field, data)

	return fieldTranslatorData{
		CppName:      fieldName,
		PBName:       pbFieldName,
		ForwardExpr:  forwardExpr,
		BackwardExpr: backwardExpr,
		IsOptional:   field.IsHardOptional,
		IsArray:      field.Type.Name == "Array",
	}
}

func (p *Plugin) generateFieldConversion(
	field resolution.Field,
	data templateDataBase,
) (forward, backward string) {
	typeRef := field.Type
	fieldName := field.Name
	cppField := fmt.Sprintf("cpp.%s", fieldName)
	pbField := fmt.Sprintf("pb.%s()", fieldName)
	pbSetter := fmt.Sprintf("pb.set_%s", fieldName)

	// Handle arrays
	if typeRef.Name == "Array" && len(typeRef.TypeArgs) > 0 {
		return p.generateArrayConversion(field, data, cppField, pbField, pbSetter)
	}

	// Handle primitives
	if resolution.IsPrimitive(typeRef.Name) {
		return p.generatePrimitiveConversion(typeRef.Name, cppField, pbField, pbSetter, data)
	}

	// Try to resolve the type
	resolved, ok := typeRef.Resolve(data.getTable())
	if !ok {
		// Default: direct copy
		return fmt.Sprintf("%s(%s)", pbSetter, cppField),
			fmt.Sprintf("cpp.%s = %s", fieldName, pbField)
	}

	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		return p.generateStructConversion(typeRef, resolved, field.IsHardOptional, data, fieldName)
	case resolution.EnumForm:
		return p.generateEnumConversion(resolved, form, cppField, pbField, pbSetter, data)
	case resolution.DistinctForm:
		return p.generateDistinctConversion(typeRef, resolved, form, cppField, pbField, pbSetter, data)
	case resolution.AliasForm:
		return p.generateAliasConversion(typeRef, resolved, form, cppField, pbField, pbSetter, data)
	default:
		return fmt.Sprintf("%s(%s)", pbSetter, cppField),
			fmt.Sprintf("cpp.%s = %s", fieldName, pbField)
	}
}

func (p *Plugin) generatePrimitiveConversion(
	primitive, cppField, pbField, pbSetter string,
	data templateDataBase,
) (forward, backward string) {
	switch primitive {
	case "uuid":
		return fmt.Sprintf("%s(%s)", pbSetter, cppField),
			fmt.Sprintf("cpp.%s = %s", strings.TrimPrefix(cppField, "cpp."), pbField)
	case "timestamp":
		return fmt.Sprintf("%s(static_cast<std::int64_t>(%s))", pbSetter, cppField),
			fmt.Sprintf("cpp.%s = telem::TimeStamp(%s)", strings.TrimPrefix(cppField, "cpp."), pbField)
	case "timespan":
		return fmt.Sprintf("%s(static_cast<std::int64_t>(%s))", pbSetter, cppField),
			fmt.Sprintf("cpp.%s = telem::TimeSpan(%s)", strings.TrimPrefix(cppField, "cpp."), pbField)
	case "data_type":
		return fmt.Sprintf("%s(std::string(%s))", pbSetter, cppField),
			fmt.Sprintf("cpp.%s = telem::DataType(%s)", strings.TrimPrefix(cppField, "cpp."), pbField)
	default:
		return fmt.Sprintf("%s(%s)", pbSetter, cppField),
			fmt.Sprintf("cpp.%s = %s", strings.TrimPrefix(cppField, "cpp."), pbField)
	}
}

func (p *Plugin) generateStructConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	isOptional bool,
	data templateDataBase,
	fieldName string,
) (forward, backward string) {
	cppField := fmt.Sprintf("cpp.%s", fieldName)

	// Get the pb namespace for the struct
	structPBName := getPBName(resolved)

	if isOptional {
		forward = fmt.Sprintf("if (%s.has_value()) *pb.mutable_%s() = to_pb(*%s)", cppField, fieldName, cppField)
		backward = fmt.Sprintf("if (pb.has_%s()) cpp.%s = from_pb(pb.%s())", fieldName, fieldName, fieldName)
	} else {
		forward = fmt.Sprintf("*pb.mutable_%s() = to_pb(%s)", fieldName, cppField)
		backward = fmt.Sprintf("cpp.%s = from_pb(pb.%s())", fieldName, fieldName)
	}

	// Check if we need to qualify with namespace
	if resolved.Namespace != data.getRawNs() {
		targetOutputPath := output.GetPath(resolved, "cpp")
		if targetOutputPath != "" {
			ns := derivePBNamespace(targetOutputPath)
			if isOptional {
				forward = fmt.Sprintf("if (%s.has_value()) *pb.mutable_%s() = %s::to_pb(*%s)", cppField, fieldName, ns, cppField)
				backward = fmt.Sprintf("if (pb.has_%s()) cpp.%s = %s::from_pb(pb.%s())", fieldName, fieldName, ns, fieldName)
			} else {
				forward = fmt.Sprintf("*pb.mutable_%s() = %s::to_pb(%s)", fieldName, ns, cppField)
				backward = fmt.Sprintf("cpp.%s = %s::from_pb(pb.%s())", fieldName, ns, fieldName)
			}
			// Add include for cross-namespace translator
			data.getIncludes().addInternal(fmt.Sprintf("%s/translator.gen.h", targetOutputPath))
		}
	}

	_ = structPBName // unused for now
	return forward, backward
}

func (p *Plugin) generateEnumConversion(
	resolved resolution.Type,
	form resolution.EnumForm,
	cppField, pbField, pbSetter string,
	data templateDataBase,
) (forward, backward string) {
	enumName := resolved.Name
	fieldName := strings.TrimPrefix(cppField, "cpp.")

	if form.IsIntEnum {
		forward = fmt.Sprintf("%s(static_cast<api::v1::%s>(%s))", pbSetter, enumName, cppField)
		backward = fmt.Sprintf("cpp.%s = static_cast<%s>(%s)", fieldName, enumName, pbField)
	} else {
		// String enum - use translator functions
		forward = fmt.Sprintf("%s(%sToPB(%s))", pbSetter, enumName, cppField)
		backward = fmt.Sprintf("cpp.%s = %sFromPB(%s)", fieldName, enumName, pbField)
	}

	return forward, backward
}

func (p *Plugin) generateDistinctConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	form resolution.DistinctForm,
	cppField, pbField, pbSetter string,
	data templateDataBase,
) (forward, backward string) {
	fieldName := strings.TrimPrefix(cppField, "cpp.")

	// Get the @cpp name if defined, otherwise use the original name
	cppName := domain.GetName(resolved, "cpp")

	// Get the underlying primitive type for casting
	if resolution.IsPrimitive(form.Base.Name) {
		protoType := primitiveToProtoType(form.Base.Name)
		return fmt.Sprintf("%s(static_cast<%s>(%s))", pbSetter, protoType, cppField),
			fmt.Sprintf("cpp.%s = %s(%s)", fieldName, cppName, pbField)
	}

	return fmt.Sprintf("%s(%s)", pbSetter, cppField),
		fmt.Sprintf("cpp.%s = %s", fieldName, pbField)
}

func (p *Plugin) generateAliasConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	form resolution.AliasForm,
	cppField, pbField, pbSetter string,
	data templateDataBase,
) (forward, backward string) {
	fieldName := strings.TrimPrefix(cppField, "cpp.")

	if resolution.IsPrimitive(form.Target.Name) {
		return p.generatePrimitiveConversion(form.Target.Name, cppField, pbField, pbSetter, data)
	}

	return fmt.Sprintf("%s(%s)", pbSetter, cppField),
		fmt.Sprintf("cpp.%s = %s", fieldName, pbField)
}

func (p *Plugin) generateArrayConversion(
	field resolution.Field,
	data templateDataBase,
	cppField, pbField, pbSetter string,
) (forward, backward string) {
	fieldName := field.Name
	typeRef := field.Type

	if len(typeRef.TypeArgs) == 0 {
		return fmt.Sprintf("// TODO: array without type args"), fmt.Sprintf("// TODO: array without type args")
	}

	elemType := typeRef.TypeArgs[0]

	// Handle arrays of structs
	if !resolution.IsPrimitive(elemType.Name) {
		if resolved, ok := elemType.Resolve(data.getTable()); ok {
			if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
				forward = fmt.Sprintf("for (const auto& item : %s) *pb.add_%s() = to_pb(item)", cppField, fieldName)
				backward = fmt.Sprintf("for (const auto& item : pb.%s()) cpp.%s.push_back(from_pb(item))", fieldName, fieldName)
				return forward, backward
			}
		}
	}

	// Array of primitives
	forward = fmt.Sprintf("for (const auto& item : %s) pb.add_%s(item)", cppField, fieldName)
	backward = fmt.Sprintf("for (const auto& item : pb.%s()) cpp.%s.push_back(item)", fieldName, fieldName)

	return forward, backward
}

func (p *Plugin) processEnumForTranslation(
	e resolution.Type,
	data templateDataBase,
) *enumTranslatorData {
	form, ok := e.Form.(resolution.EnumForm)
	if !ok {
		return nil
	}

	// String enums need translator functions
	if form.IsIntEnum {
		return nil
	}

	values := make([]enumValueTranslatorData, 0, len(form.Values))
	for _, v := range form.Values {
		valueName := toPascalCase(v.Name)
		pbValueName := fmt.Sprintf("%s_%s", e.Name, toScreamingSnake(v.Name))

		values = append(values, enumValueTranslatorData{
			CppValue: fmt.Sprintf("%s%s", e.Name, valueName),
			PBValue:  pbValueName,
		})
	}

	return &enumTranslatorData{
		Name:      e.Name,
		Values:    values,
		PBDefault: fmt.Sprintf("%s_UNSPECIFIED", e.Name),
		CppDefault: fmt.Sprintf("%s%s", e.Name, toPascalCase(form.Values[0].Name)),
	}
}

func deriveNamespace(outputPath string) string {
	parts := strings.Split(outputPath, "/")
	if len(parts) == 0 {
		return "synnax"
	}
	subNs := parts[len(parts)-1]
	return fmt.Sprintf("synnax::%s", subNs)
}

func derivePBNamespace(outputPath string) string {
	parts := strings.Split(outputPath, "/")
	if len(parts) == 0 {
		return "synnax::pb"
	}
	subNs := parts[len(parts)-1]
	return fmt.Sprintf("synnax::%s::pb", subNs)
}

func deriveProtoInclude(outputPath, namespace string) string {
	// Map from C++ output path to proto include
	// Extract the module name from the output path (e.g., "client/cpp/channel" -> "channel")
	parts := strings.Split(outputPath, "/")
	if len(parts) == 0 {
		return ""
	}
	moduleName := parts[len(parts)-1]
	return fmt.Sprintf("core/pkg/api/grpc/v1/%s.pb.h", moduleName)
}

func getPBName(s resolution.Type) string {
	if domain, ok := s.Domains["pb"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return s.Name
}

func primitiveToProtoType(primitive string) string {
	switch primitive {
	case "uint8", "uint12", "uint16", "uint20", "uint32":
		return "uint32_t"
	case "uint64":
		return "uint64_t"
	case "int8", "int16", "int32":
		return "int32_t"
	case "int64":
		return "int64_t"
	case "float32":
		return "float"
	case "float64":
		return "double"
	case "bool":
		return "bool"
	case "string":
		return "std::string"
	default:
		return primitive
	}
}

func toPascalCase(s string) string {
	return lo.PascalCase(s)
}

func toScreamingSnake(s string) string {
	return strings.ToUpper(lo.SnakeCase(s))
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

// Template data types

type headerTemplateData struct {
	OutputPath       string
	Namespace        string
	PBNamespace      string
	Translators      []translatorData
	EnumTranslators  []enumTranslatorData
	ForwardDecls     []forwardDeclData
	includes         *includeManager
	table            *resolution.Table
	rawNs            string
	processedEnums   map[string]bool
	processedStructs map[string]bool
}

func (d *headerTemplateData) getTable() *resolution.Table  { return d.table }
func (d *headerTemplateData) getRawNs() string             { return d.rawNs }
func (d *headerTemplateData) getIncludes() *includeManager { return d.includes }

func (d *headerTemplateData) HasIncludes() bool {
	return len(d.includes.system) > 0 || len(d.includes.internal) > 0
}
func (d *headerTemplateData) SystemIncludes() []string   { return d.includes.system }
func (d *headerTemplateData) InternalIncludes() []string { return d.includes.internal }

type sourceTemplateData struct {
	OutputPath      string
	Namespace       string
	PBNamespace     string
	HeaderFile      string
	Translators     []translatorData
	EnumTranslators []enumTranslatorData
	includes        *includeManager
	table           *resolution.Table
	rawNs           string
	processedEnums  map[string]bool
}

func (d *sourceTemplateData) getTable() *resolution.Table  { return d.table }
func (d *sourceTemplateData) getRawNs() string             { return d.rawNs }
func (d *sourceTemplateData) getIncludes() *includeManager { return d.includes }

func (d *sourceTemplateData) HasIncludes() bool {
	return len(d.includes.system) > 0 || len(d.includes.internal) > 0
}
func (d *sourceTemplateData) SystemIncludes() []string   { return d.includes.system }
func (d *sourceTemplateData) InternalIncludes() []string { return d.includes.internal }

type translatorData struct {
	CppName string
	PBName  string
	Fields  []fieldTranslatorData
}

type fieldTranslatorData struct {
	CppName      string
	PBName       string
	ForwardExpr  string
	BackwardExpr string
	IsOptional   bool
	IsArray      bool
}

type enumTranslatorData struct {
	Name       string
	Values     []enumValueTranslatorData
	PBDefault  string
	CppDefault string
}

type enumValueTranslatorData struct {
	CppValue string
	PBValue  string
}

type forwardDeclData struct {
	Namespace string
	Name      string
}
