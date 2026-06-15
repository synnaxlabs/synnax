// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package pb provides an Oracle plugin that generates protobuf translator functions
// for the pb/ subdirectory pattern. It produces translator.gen.go files that convert
// between Go domain types and protobuf types.
package pb

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/enum"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Plugin generates protobuf translator functions for the pb/ subdirectory pattern.
type Plugin struct{ Options Options }

// Options configures the go/pb plugin.
type Options struct {
	// TranslatorFileNamePattern is the filename pattern for translator files.
	TranslatorFileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{
		TranslatorFileNamePattern: "translator.gen.go",
	}
}

// New creates a new go/pb plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

// Name returns the plugin identifier.
func (p *Plugin) Name() string { return "go/pb" }

// Domains returns the domains this plugin handles.
func (p *Plugin) Domains() []string { return []string{"pb"} }

// Requires returns plugin dependencies.
func (p *Plugin) Requires() []string { return []string{"go/types", "pb/types"} }

// Check verifies generated files are up-to-date. Currently unimplemented.
func (p *Plugin) Check(*plugin.Request) error { return nil }

// Generate produces translator functions for structs with @pb flag.
func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}

	outputStructs := make(map[string][]resolution.Type)
	outputTypeDefs := make(map[string][]resolution.Type)
	var outputOrder []string

	for _, entry := range req.Resolutions.StructTypes() {
		outputPath := output.GetPBPath(entry)
		if outputPath == "" {
			continue
		}

		if omit.IsType(entry, "pb") {
			continue
		}

		if req.RepoRoot != "" {
			if err := req.ValidateOutputPath(outputPath); err != nil {
				return nil, errors.Wrapf(err, "invalid output path for struct %s", entry.Name)
			}
		}
		if _, exists := outputStructs[outputPath]; !exists {
			outputOrder = append(outputOrder, outputPath)
		}
		outputStructs[outputPath] = append(outputStructs[outputPath], entry)
	}

	for _, entry := range req.Resolutions.DistinctTypes() {
		outputPath := output.GetPBPath(entry)
		if outputPath == "" {
			continue
		}
		if omit.IsType(entry, "pb") {
			continue
		}

		form, ok := entry.Form.(resolution.DistinctForm)
		if !ok {
			continue
		}
		if !p.isStructWrappingTypedef(form.Base, req.Resolutions) {
			continue
		}

		if req.RepoRoot != "" {
			if err := req.ValidateOutputPath(outputPath); err != nil {
				return nil, errors.Wrapf(err, "invalid output path for typedef %s", entry.Name)
			}
		}
		if _, exists := outputStructs[outputPath]; !exists {
			if _, exists := outputTypeDefs[outputPath]; !exists {
				outputOrder = append(outputOrder, outputPath)
			}
		}
		outputTypeDefs[outputPath] = append(outputTypeDefs[outputPath], entry)
	}

	outputUnions := make(map[string][]resolution.Type)
	for _, entry := range req.Resolutions.UnionTypes() {
		outputPath := output.GetPBPath(entry)
		if outputPath == "" {
			continue
		}
		if omit.IsType(entry, "pb") {
			continue
		}
		if req.RepoRoot != "" {
			if err := req.ValidateOutputPath(outputPath); err != nil {
				return nil, errors.Wrapf(err, "invalid output path for union %s", entry.Name)
			}
		}
		_, hasStruct := outputStructs[outputPath]
		_, hasTypeDef := outputTypeDefs[outputPath]
		if _, exists := outputUnions[outputPath]; !exists && !hasStruct && !hasTypeDef {
			outputOrder = append(outputOrder, outputPath)
		}
		outputUnions[outputPath] = append(outputUnions[outputPath], entry)
	}

	// Register pb output paths for schemas that opt into @pb but declare
	// only enums (no structs or distinct typedefs). Cross-namespace fields
	// referencing these enums depend on the foreign translator existing;
	// without this pass the schema produces nothing and the dependent
	// schema fails to compile against its missing import.
	enumOnlyNamespace := make(map[string]string)
	for _, e := range req.Resolutions.EnumTypes() {
		if omit.IsType(e, "pb") {
			continue
		}
		outputPath := enum.FindPBOutputPath(e, req.Resolutions)
		if outputPath == "" {
			continue
		}
		if _, hasStruct := outputStructs[outputPath]; hasStruct {
			continue
		}
		if _, hasTypeDef := outputTypeDefs[outputPath]; hasTypeDef {
			continue
		}
		if _, hasUnion := outputUnions[outputPath]; hasUnion {
			continue
		}
		if _, alreadyRegistered := enumOnlyNamespace[outputPath]; alreadyRegistered {
			continue
		}
		enumOnlyNamespace[outputPath] = e.Namespace
		outputOrder = append(outputOrder, outputPath)
	}

	pbPathFunc := func(typ resolution.Type, table *resolution.Table) string {
		return enum.FindPBOutputPath(typ, table)
	}
	for _, outputPath := range outputOrder {
		structs := outputStructs[outputPath]
		typeDefs := outputTypeDefs[outputPath]
		unions := outputUnions[outputPath]
		var namespace string
		switch {
		case len(structs) > 0:
			namespace = structs[0].Namespace
		case len(typeDefs) > 0:
			namespace = typeDefs[0].Namespace
		case len(unions) > 0:
			namespace = unions[0].Namespace
		default:
			namespace = enumOnlyNamespace[outputPath]
		}
		// CollectNamespaceEnums with FindPBOutputPath respects each enum's
		// own @pb opt-in (HasPB) and FilePath, so an enum declared in a
		// different schema that happens to share this namespace name does
		// not bleed into this output.
		enums := enum.CollectNamespaceEnums(namespace, outputPath, req.Resolutions, "pb", pbPathFunc)
		content, err := p.generateFile(outputPath, structs, typeDefs, unions, enums, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate %s", outputPath)
		}
		if len(content) > 0 {
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/%s", outputPath, p.Options.TranslatorFileNamePattern),
				Content: content,
			})
		}
	}

	return resp, nil
}

// isStructWrappingTypedef checks if a typeRef ultimately resolves to a struct type.
func (p *Plugin) isStructWrappingTypedef(typeRef resolution.TypeRef, table *resolution.Table) bool {
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return false
	}
	// Direct struct
	if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
		return true
	}
	// Alias to struct
	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		return p.isStructWrappingTypedef(aliasForm.Target, table)
	}
	return false
}

func (p *Plugin) generateFile(
	outputPath string,
	structs []resolution.Type,
	typeDefs []resolution.Type,
	unions []resolution.Type,
	enums []resolution.Type,
	req *plugin.Request,
) ([]byte, error) {
	namespace := ""
	if len(structs) > 0 {
		namespace = structs[0].Namespace
	} else if len(typeDefs) > 0 {
		namespace = typeDefs[0].Namespace
	} else if len(unions) > 0 {
		namespace = unions[0].Namespace
	}

	parentGoPath := strings.TrimSuffix(outputPath, "/pb")

	data := &templateData{
		Package:               "pb",
		OutputPath:            outputPath,
		ParentGoPath:          parentGoPath,
		Namespace:             namespace,
		Translators:           make([]translatorData, 0, len(structs)),
		GenericTranslators:    make([]genericTranslatorData, 0),
		EnumTranslators:       make([]enumTranslatorData, 0),
		AnyHelpers:            make([]anyHelperData, 0),
		DelegationTranslators: make([]delegationTranslatorData, 0),
		imports:               imports.NewManager(),
		repoRoot:              req.RepoRoot,
		table:                 req.Resolutions,
		usedEnums:             make(map[string]*resolution.Type),
		generatedAnyHelpers:   make(set.Set[string]),
	}

	parentImportPath, err := resolveGoImportPath(parentGoPath, req.RepoRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve parent package import")
	}
	parentAlias := naming.DerivePackageName(parentGoPath)
	data.imports.AddInternal(parentAlias, parentImportPath)
	data.parentAlias = parentAlias

	for _, s := range structs {
		if omit.IsType(s, "pb") {
			continue
		}
		form, ok := s.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		if form.IsGeneric() {
			// Generate generic translator with type parameters
			genericTranslator, err := p.processGenericStructForTranslation(s, form, data, req)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to process generic struct %s", s.Name)
			}
			if genericTranslator != nil {
				data.GenericTranslators = append(data.GenericTranslators, *genericTranslator)
			}
		} else {
			translator, err := p.processStructForTranslation(s, form, data, req)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to process struct %s", s.Name)
			}
			if translator != nil {
				data.Translators = append(data.Translators, *translator)
			}
		}
	}

	for _, u := range unions {
		form, ok := u.Form.(resolution.UnionForm)
		if !ok {
			return nil, errors.Newf(
				"union %s has unexpected form %T in the resolution table", u.Name, u.Form,
			)
		}
		ut, err := p.processUnionForTranslation(u, form, data)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process union %s", u.Name)
		}
		data.UnionTranslators = append(data.UnionTranslators, *ut)

		// Inline variants have no standalone Go payload type, so their payload
		// message translates against the variant member itself: the member
		// declares the payload's fields directly (and promotes base-embed
		// fields), so field access lines up with the payload message.
		goName := naming.GetGoName(u)
		for _, v := range form.Variants {
			if !v.Inline {
				continue
			}
			payload, ok := v.Type.Resolve(data.table)
			if !ok {
				continue
			}
			pform, ok := payload.Form.(resolution.StructForm)
			if !ok {
				continue
			}
			pt, err := p.processStructForTranslation(payload, pform, data, req)
			if err != nil || pt == nil {
				return nil, errors.Wrapf(err,
					"failed to process inline payload for union %s variant %q",
					u.Name, v.Name)
			}
			// The translator is named after the member, not the payload message:
			// a union composed from this one inherits the variant and generates
			// its own member translator against the same payload message, so
			// payload-message naming would collide.
			memberName := casing.VariantTypeName(goName, v.Name)
			pt.Name = memberName
			pt.GoType = fmt.Sprintf("%s.%s", data.parentAlias, memberName)
			pt.GoTypeShort = memberName
			data.Translators = append(data.Translators, *pt)
		}
	}
	if len(data.UnionTranslators) > 0 {
		data.imports.AddExternal("github.com/synnaxlabs/x/errors")
	}

	for i := range enums {
		e := enums[i]
		if omit.IsType(e, "pb") {
			continue
		}
		enumPBPath := enum.FindPBOutputPath(e, req.Resolutions)
		if enumPBPath != "" && enumPBPath != outputPath {
			continue
		}
		data.usedEnums[e.QualifiedName] = &e
	}

	enumKeys := make([]string, 0, len(data.usedEnums))
	for k := range data.usedEnums {
		enumKeys = append(enumKeys, k)
	}
	sort.Strings(enumKeys)
	for _, k := range enumKeys {
		enumTranslator := p.generateEnumTranslator(data.usedEnums[k], data)
		if enumTranslator != nil {
			data.EnumTranslators = append(data.EnumTranslators, *enumTranslator)
		}
	}

	if len(data.EnumTranslators) > 0 {
		data.imports.AddExternal("github.com/synnaxlabs/x/errors")
	}

	for _, td := range typeDefs {
		if omit.IsType(td, "pb") {
			continue
		}
		form, ok := td.Form.(resolution.DistinctForm)
		if !ok {
			continue
		}
		delegator, err := p.processDelegationTranslator(td, form, data, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to process delegation translator %s", td.Name)
		}
		if delegator != nil {
			data.DelegationTranslators = append(data.DelegationTranslators, *delegator)
		}
	}

	if len(data.Translators) == 0 && len(data.GenericTranslators) == 0 && len(data.EnumTranslators) == 0 && len(data.DelegationTranslators) == 0 && len(data.UnionTranslators) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	if err := translatorFileTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// unionTranslatorData holds data for translating a discriminated union between
// its Go sealed-interface form and its protobuf oneof wrapper message.
type unionTranslatorData struct {
	Name        string
	GoType      string
	PBType      string
	GoTypeShort string
	PBTypeShort string
	// Bases lists the union's extends bases, which nest as message fields on
	// the wrapper and translate through the bases' own translator functions.
	Bases []unionBaseTranslatorData
	// Variants lists every variant in declaration order.
	Variants []unionVariantTranslatorData
}

// unionBaseTranslatorData holds data for one extends base of a union
// translator.
type unionBaseTranslatorData struct {
	// GoEmbed is the embedded base field on the Go variant struct (e.g. "TabBase").
	GoEmbed string
	// PBGoName is the protoc-generated Go field for the base message on the
	// wrapper (e.g. "TabBase" for proto field "tab_base").
	PBGoName string
	// ToPB and FromPB are the base struct's translator functions.
	ToPB   string
	FromPB string
}

// unionVariantTranslatorData holds data for one variant of a union translator.
type unionVariantTranslatorData struct {
	// GoVariantType is the qualified Go variant struct (e.g. "schematic.NodeConfigCap").
	GoVariantType string
	// PBWrapper is the protoc-generated oneof wrapper type (e.g. "NodeConfig_Cap").
	PBWrapper string
	// PBField is the field name inside the oneof wrapper (e.g. "Cap").
	PBField string
	// IsInline marks an inline variant: the payload translators take the
	// variant member itself rather than an embedded payload field.
	IsInline bool
	// PayloadGoField is the embedded payload field on the Go variant struct.
	// Empty for inline variants.
	PayloadGoField string
	// PayloadToPB and PayloadFromPB are the payload struct's translator functions.
	PayloadToPB   string
	PayloadFromPB string
}

// processUnionForTranslation builds the translator view for a discriminated
// union: a nil-variant-aware ToPB that switches on the Go variant type and
// sets the corresponding protobuf oneof wrapper, and the inverse FromPB.
func (p *Plugin) processUnionForTranslation(
	u resolution.Type,
	form resolution.UnionForm,
	data *templateData,
) (*unionTranslatorData, error) {
	goName := naming.GetGoName(u)
	pbName := getPBName(u)
	if pbName == "" {
		pbName = u.Name
	}
	ut := &unionTranslatorData{
		Name:        pbName,
		GoType:      fmt.Sprintf("%s.%s", data.parentAlias, goName),
		PBType:      pbName,
		GoTypeShort: goName,
		PBTypeShort: pbName,
	}
	for _, ext := range form.Extends {
		base, ok := ext.Resolve(data.table)
		if !ok {
			return nil, errors.Newf("union %s: unresolved base %s", u.Name, ext.Name)
		}
		if _, isStruct := base.Form.(resolution.StructForm); !isStruct {
			continue
		}
		prefix, baseName := p.resolvePBTranslatorInfo(base, data)
		ut.Bases = append(ut.Bases, unionBaseTranslatorData{
			GoEmbed:  naming.GetGoName(base),
			PBGoName: lo.PascalCase(casing.FieldSnake(base.Name)),
			ToPB:     prefix + baseName + "ToPB",
			FromPB:   prefix + baseName + "FromPB",
		})
	}
	for _, v := range form.Variants {
		payload, ok := v.Type.Resolve(data.table)
		if !ok {
			return nil, errors.Newf("union %s variant %q: unresolved payload type", u.Name, v.Name)
		}
		pbField := protocOneofGoName(v.Name)
		vt := unionVariantTranslatorData{
			GoVariantType: fmt.Sprintf("%s.%s", data.parentAlias, casing.VariantTypeName(goName, v.Name)),
			PBWrapper:     fmt.Sprintf("%s_%s", pbName, pbField),
			PBField:       pbField,
			IsInline:      v.Inline,
		}
		if v.Inline {
			// Inline payload translators are member-typed and generated locally
			// for each union that carries the variant, so they are referenced
			// without a package prefix.
			memberName := casing.VariantTypeName(goName, v.Name)
			vt.PayloadToPB = memberName + "ToPB"
			vt.PayloadFromPB = memberName + "FromPB"
		} else {
			prefix, payloadName := p.resolvePBTranslatorInfo(payload, data)
			vt.PayloadGoField = naming.GetGoName(payload)
			vt.PayloadToPB = prefix + payloadName + "ToPB"
			vt.PayloadFromPB = prefix + payloadName + "FromPB"
		}
		ut.Variants = append(ut.Variants, vt)
	}
	return ut, nil
}

// protocOneofGoName mirrors protoc-gen-go's naming for oneof member fields:
// the CamelCase field name gains a trailing underscore when it collides with a
// method protoc generates on every message.
func protocOneofGoName(value string) string {
	name := lo.PascalCase(casing.FieldSnake(value))
	switch name {
	case "Reset", "String", "ProtoMessage", "ProtoReflect", "Descriptor":
		return name + "_"
	}
	return name
}

// resolveUnionTranslatorName returns the package prefix and function base name
// for a union's pb translator, adding the cross-package import when the union
// lives in a different proto package.
func (p *Plugin) resolveUnionTranslatorName(
	u resolution.Type,
	data *templateData,
) (prefix, name string) {
	name = getPBName(u)
	if name == "" {
		name = u.Name
	}
	pbPath := output.GetPBPath(u)
	if u.Namespace != data.Namespace || (pbPath != "" && pbPath != data.OutputPath) {
		if importPath, err := resolveGoImportPath(pbPath, data.repoRoot); err == nil {
			alias := strings.ToLower(u.Namespace) + "pb"
			data.imports.AddInternal(alias, importPath)
			prefix = alias + "."
		}
	}
	return prefix, name
}

func (p *Plugin) processStructForTranslation(
	s resolution.Type,
	form resolution.StructForm,
	data *templateData,
	req *plugin.Request,
) (*translatorData, error) {
	if _, ok := s.Form.(resolution.AliasForm); ok {
		return nil, nil
	}

	goName := naming.GetGoName(s)

	pbName := getPBName(s)
	if pbName == "" {
		pbName = s.Name
	}

	translator := &translatorData{
		Name:           pbName,
		GoType:         fmt.Sprintf("%s.%s", data.parentAlias, goName),
		PBType:         pbName,
		GoTypeShort:    goName,
		PBTypeShort:    pbName,
		Fields:         make([]fieldTranslatorData, 0),
		ErrorFields:    make([]fieldTranslatorData, 0),
		OptionalFields: make([]fieldTranslatorData, 0),
	}

	for _, field := range resolution.UnifiedFields(s, data.table) {
		fieldData := p.processFieldForTranslation(field, data, s)
		if fieldData.IsOptional {
			translator.OptionalFields = append(translator.OptionalFields, fieldData)
		} else if fieldData.HasError || fieldData.HasBackwardError {
			translator.ErrorFields = append(translator.ErrorFields, fieldData)
		} else {
			translator.Fields = append(translator.Fields, fieldData)
		}
	}

	return translator, nil
}

func (p *Plugin) processFieldForTranslation(
	field resolution.Field,
	data *templateData,
	parentStruct resolution.Type,
) fieldTranslatorData {
	goName := naming.GetFieldName(field)
	pbName := lo.PascalCase(lo.SnakeCase(field.Name))

	isHardOptional := field.IsHardOptional
	isOptional := isHardOptional
	isOptionalStruct := isOptional &&
		(isStructType(field.Type, data.table) || isUnionType(field.Type, data.table))
	isOptionalEnum := isOptional && isEnumType(field.Type, data.table)

	forwardExpr, backwardExpr, backwardCast, hasError, hasBackwardError := p.generateFieldConversion(field, data, parentStruct)

	fd := fieldTranslatorData{
		GoName:           goName,
		PBName:           pbName,
		ForwardExpr:      forwardExpr,
		BackwardExpr:     backwardExpr,
		BackwardCast:     backwardCast,
		IsOptional:       isOptional,
		IsOptionalStruct: isOptionalStruct,
		IsOptionalEnum:   isOptionalEnum,
		HasError:         hasError,
		HasBackwardError: hasBackwardError,
	}

	typeRef := field.Type

	// Hard optional primitives that need type conversion (e.g., *uint8 <-> *uint32)
	// require pointer dereference before casting and re-addressing after.
	if isHardOptional && resolution.IsPrimitive(typeRef.Name) && primitiveNeedsConversion(typeRef.Name) {
		fd.NeedsPtrConversion = true
		goFieldDeref := "*r." + goName
		pbFieldDeref := "*pb." + pbName
		fd.ForwardExpr, fd.BackwardExpr, _, _ = p.generatePrimitiveConversion(typeRef.Name, goFieldDeref, pbFieldDeref, data)
	}

	// Hard optional typedefs over primitives (e.g. *channel.Key <-> *uint32)
	// need the same deref-convert-readdress treatment.
	if isHardOptional && !resolution.IsPrimitive(typeRef.Name) {
		if resolved, ok := typeRef.Resolve(data.table); ok {
			if form, isDistinct := resolved.Form.(resolution.DistinctForm); isDistinct &&
				resolution.IsPrimitive(form.Base.Name) {
				fd.NeedsPtrConversion = true
				fd.ForwardExpr, fd.BackwardExpr, _, _ = p.generateTypeDefConversion(
					typeRef, resolved, form, data, "*r."+goName, "*pb."+pbName,
				)
			}
		}
	}

	// Maps whose value type does not survive a direct copy (numeric primitives that
	// widen, opaque records that bridge through structpb.Struct, struct values that
	// have their own pb translator) require element-wise conversion loops. Force
	// into OptionalFields so the template renders a nil-guarded loop rather than a
	// direct struct initializer assignment.
	if typeRef.Name == "Map" && len(typeRef.TypeArgs) == 2 {
		if mvc := p.buildMapValueConversion(typeRef, data); mvc != nil {
			fd.MapValueConversion = mvc
			fd.IsOptional = true
			fd.ForwardExpr = ""
			fd.BackwardExpr = ""
		}
	}

	return fd
}

// buildMapValueConversion returns the per-element conversion data for a Map
// field whose value type does not round-trip directly between the Go domain
// type and the proto wire type. Returns nil when no conversion is needed (the
// caller should fall back to direct field copy).
func (p *Plugin) buildMapValueConversion(
	typeRef resolution.TypeRef, data *templateData,
) *mapValueConversionData {
	keyType := primitiveToProtoType(typeRef.TypeArgs[0].Name)
	valArg := typeRef.TypeArgs[1]

	if resolution.IsPrimitive(valArg.Name) {
		switch valArg.Name {
		case "record":
			data.imports.AddInternal("msgpack", "github.com/synnaxlabs/x/encoding/msgpack")
			data.imports.AddExternal("google.golang.org/protobuf/types/known/structpb")
			return &mapValueConversionData{
				GoMapType:         fmt.Sprintf("map[%s]msgpack.EncodedJSON", keyType),
				PBMapType:         fmt.Sprintf("map[%s]*structpb.Struct", keyType),
				ForwardValueExpr:  "structpb.NewStruct(v)",
				BackwardValueExpr: "msgpack.EncodedJSON(v.AsMap())",
				ForwardHasError:   true,
			}
		default:
			if primitiveNeedsConversion(valArg.Name) {
				fwd, bwd, _, _ := p.generatePrimitiveConversion(valArg.Name, "v", "v", data)
				return &mapValueConversionData{
					GoMapType:         fmt.Sprintf("map[%s]%s", keyType, valArg.Name),
					PBMapType:         fmt.Sprintf("map[%s]%s", keyType, primitiveToProtoType(valArg.Name)),
					ForwardValueExpr:  fwd,
					BackwardValueExpr: bwd,
				}
			}
			return nil
		}
	}

	resolved, ok := valArg.Resolve(data.table)
	if !ok {
		return nil
	}
	if _, isUnion := resolved.Form.(resolution.UnionForm); isUnion {
		goValType := p.resolveGoTypeLiteral(valArg, data)
		prefix, name := p.resolveUnionTranslatorName(resolved, data)
		return &mapValueConversionData{
			GoMapType:         fmt.Sprintf("map[%s]%s", keyType, goValType),
			PBMapType:         fmt.Sprintf("map[%s]*%s%s", keyType, prefix, name),
			ForwardValueExpr:  fmt.Sprintf("%s%sToPB(v)", prefix, name),
			BackwardValueExpr: fmt.Sprintf("%s%sFromPB(v)", prefix, name),
			ForwardHasError:   true,
			BackwardHasError:  true,
		}
	}
	if _, isStruct := resolved.Form.(resolution.StructForm); !isStruct {
		return nil
	}
	goValType := p.resolveGoTypeLiteral(valArg, data)
	translatorPrefix, structName := p.resolvePBTranslatorInfo(resolved, data)
	return &mapValueConversionData{
		GoMapType:         fmt.Sprintf("map[%s]%s", keyType, goValType),
		PBMapType:         fmt.Sprintf("map[%s]*%s%s", keyType, translatorPrefix, structName),
		ForwardValueExpr:  fmt.Sprintf("%s%sToPB(v)", translatorPrefix, structName),
		BackwardValueExpr: fmt.Sprintf("%s%sFromPB(v)", translatorPrefix, structName),
		ForwardHasError:   true,
		BackwardHasError:  true,
	}
}

func (p *Plugin) processGenericStructForTranslation(
	s resolution.Type,
	form resolution.StructForm,
	data *templateData,
	req *plugin.Request,
) (*genericTranslatorData, error) {
	if _, ok := s.Form.(resolution.AliasForm); ok {
		return nil, nil
	}

	goName := naming.GetGoName(s)

	pbName := getPBName(s)
	if pbName == "" {
		pbName = s.Name
	}

	typeParams := make([]typeParamData, 0, len(form.TypeParams))
	typeParamNames := make([]string, 0, len(form.TypeParams))
	for _, tp := range resolution.NonDefaultedTypeParams(form.TypeParams) {
		typeParams = append(typeParams, typeParamData{Name: tp.Name, Constraint: typeParamConstraint(tp)})
		typeParamNames = append(typeParamNames, tp.Name)
	}

	// anypb is only referenced from translator signatures of structs whose
	// generics survive default-substitution; for fully-defaulted generics the
	// emitted translator is concrete and the import would be unused.
	if len(typeParamNames) > 0 {
		data.imports.AddExternal("google.golang.org/protobuf/types/known/anypb")
	}

	goTypeBase := fmt.Sprintf("%s.%s", data.parentAlias, goName)
	goTypeWithParams := goTypeBase
	if len(typeParamNames) > 0 {
		goTypeWithParams = fmt.Sprintf("%s[%s]", goTypeBase, strings.Join(typeParamNames, ", "))
	}

	translator := &genericTranslatorData{
		Name:            pbName,
		GoType:          goTypeWithParams,
		GoTypeBase:      goTypeBase,
		PBType:          pbName,
		GoTypeShort:     goName,
		PBTypeShort:     pbName,
		TypeParams:      typeParams,
		Fields:          make([]fieldTranslatorData, 0),
		ErrorFields:     make([]fieldTranslatorData, 0),
		TypeParamFields: make([]fieldTranslatorData, 0),
		OptionalFields:  make([]fieldTranslatorData, 0),
	}

	for _, field := range resolution.UnifiedFields(s, data.table) {
		fieldData, isTypeParam := p.processGenericFieldForTranslation(field, data, s, form, typeParams)
		if isTypeParam {
			translator.TypeParamFields = append(translator.TypeParamFields, fieldData)
		} else if fieldData.IsOptional {
			translator.OptionalFields = append(translator.OptionalFields, fieldData)
		} else if fieldData.HasError || fieldData.HasBackwardError {
			translator.ErrorFields = append(translator.ErrorFields, fieldData)
		} else {
			translator.Fields = append(translator.Fields, fieldData)
		}
	}

	return translator, nil
}

func (p *Plugin) processGenericFieldForTranslation(
	field resolution.Field,
	data *templateData,
	parentStruct resolution.Type,
	parentForm resolution.StructForm,
	typeParams []typeParamData,
) (fieldTranslatorData, bool) {
	goName := naming.GetFieldName(field)
	pbName := lo.PascalCase(lo.SnakeCase(field.Name))
	typeRef := field.Type

	isHardOptional := field.IsHardOptional
	isOptional := isHardOptional

	goFieldName := "r." + goName
	pbFieldName := "pb." + pbName

	if typeRef.IsTypeParam() && typeRef.TypeParam != nil {
		if typeRef.TypeParam.HasDefault() {
			forwardExpr, backwardExpr, backwardCast, hasError, hasBackwardError := p.generateFieldConversion(
				resolution.Field{
					Name:           field.Name,
					Type:           *typeRef.TypeParam.Default,
					IsOptional:     field.IsOptional,
					IsHardOptional: field.IsHardOptional,
				},
				data, parentStruct,
			)
			return fieldTranslatorData{
				GoName:           goName,
				PBName:           pbName,
				ForwardExpr:      forwardExpr,
				BackwardExpr:     backwardExpr,
				BackwardCast:     backwardCast,
				IsOptional:       isOptional,
				IsOptionalStruct: isOptional && isStructType(*typeRef.TypeParam.Default, data.table),
				IsOptionalEnum:   isOptional && isEnumType(*typeRef.TypeParam.Default, data.table),
				HasError:         hasError,
				HasBackwardError: hasBackwardError,
			}, false
		}

		paramName := typeRef.TypeParam.Name
		converterFunc := fmt.Sprintf("translate%s", paramName)

		forwardExpr := fmt.Sprintf("%s(%s)", converterFunc, goFieldName)
		backwardExpr := fmt.Sprintf("%s(%s)", converterFunc, pbFieldName)

		return fieldTranslatorData{
			GoName:           goName,
			PBName:           pbName,
			ForwardExpr:      forwardExpr,
			BackwardExpr:     backwardExpr,
			IsOptional:       isOptional,
			IsOptionalStruct: false,
			HasError:         true,
			HasBackwardError: true,
		}, true
	}

	forwardExpr, backwardExpr, backwardCast, hasError, hasBackwardError := p.generateFieldConversion(field, data, parentStruct)

	return fieldTranslatorData{
		GoName:           goName,
		PBName:           pbName,
		ForwardExpr:      forwardExpr,
		BackwardExpr:     backwardExpr,
		BackwardCast:     backwardCast,
		IsOptional:       isOptional,
		IsOptionalStruct: isHardOptional && isStructType(typeRef, data.table),
		IsOptionalEnum:   isHardOptional && isEnumType(typeRef, data.table),
		HasError:         hasError,
		HasBackwardError: hasBackwardError,
	}, false
}

func (p *Plugin) processDelegationTranslator(
	td resolution.Type,
	form resolution.DistinctForm,
	data *templateData,
	req *plugin.Request,
) (*delegationTranslatorData, error) {
	goName := naming.GetGoName(td)

	typeParams := make([]typeParamData, 0, len(form.TypeParams))
	typeParamNames := make([]string, 0, len(form.TypeParams))
	for _, tp := range resolution.NonDefaultedTypeParams(form.TypeParams) {
		typeParams = append(typeParams, typeParamData{Name: tp.Name, Constraint: typeParamConstraint(tp)})
		typeParamNames = append(typeParamNames, tp.Name)
	}

	goType := fmt.Sprintf("%s.%s", data.parentAlias, goName)
	if len(typeParamNames) > 0 {
		goType = fmt.Sprintf("%s.%s[%s]", data.parentAlias, goName, strings.Join(typeParamNames, ", "))
	}

	underlyingResolved, ok := form.Base.Resolve(data.table)
	if !ok {
		return nil, nil
	}

	actualStruct := underlyingResolved
	for {
		if aliasForm, isAlias := actualStruct.Form.(resolution.AliasForm); isAlias {
			if target, ok := aliasForm.Target.Resolve(data.table); ok {
				actualStruct = target
				continue
			}
		}
		break
	}

	underlyingGoName := naming.GetGoName(actualStruct)

	underlyingPBPath := output.GetPBPath(actualStruct)
	if underlyingPBPath == "" {
		_, underlyingPBPath = findStructWithPB(actualStruct, data.table)
		if underlyingPBPath == "" {
			return nil, nil
		}
	}

	underlyingGoPath := output.GetPath(actualStruct, "go")
	if underlyingGoPath == "" {
		return nil, nil
	}
	underlyingGoImportPath, err := resolveGoImportPath(underlyingGoPath, data.repoRoot)
	if err != nil {
		return nil, err
	}
	underlyingGoAlias := naming.DerivePackageAlias(underlyingGoPath, data.parentAlias)
	data.imports.AddInternal(underlyingGoAlias, underlyingGoImportPath)

	underlyingPBImportPath, err := resolveGoImportPath(underlyingPBPath, data.repoRoot)
	if err != nil {
		return nil, err
	}
	underlyingPBAlias := underlyingGoAlias + "_pb"
	data.imports.AddInternal(underlyingPBAlias, underlyingPBImportPath)

	if len(typeParams) > 0 {
		data.imports.AddExternal("google.golang.org/protobuf/types/known/anypb")
	}

	underlyingGoType := fmt.Sprintf("%s.%s", underlyingGoAlias, underlyingGoName)
	if len(typeParamNames) > 0 {
		underlyingGoType = fmt.Sprintf("%s.%s[%s]", underlyingGoAlias, underlyingGoName, strings.Join(typeParamNames, ", "))
	}

	underlyingPBName := getPBName(actualStruct)
	if underlyingPBName == "" {
		underlyingPBName = actualStruct.Name
	}

	return &delegationTranslatorData{
		Name:                       goName,
		GoType:                     goType,
		TypeParams:                 typeParams,
		UnderlyingName:             underlyingPBName,
		UnderlyingGoType:           underlyingGoType,
		UnderlyingPBType:           fmt.Sprintf("%s.%s", underlyingPBAlias, underlyingPBName),
		UnderlyingTranslatorPrefix: underlyingPBAlias + ".",
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

func (p *Plugin) generateFixedSizeUint8ArrayConversion(
	typeRef resolution.TypeRef,
	data *templateData,
	goField, pbField string,
) (forward, backward string) {
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return fmt.Sprintf("%s[:]", goField), pbField
	}

	goOutput := output.GetPath(resolved, "go")
	if goOutput == "" {
		return fmt.Sprintf("%s[:]", goField), pbField
	}

	importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
	if err != nil {
		return fmt.Sprintf("%s[:]", goField), pbField
	}

	alias := naming.DerivePackageName(goOutput)
	data.imports.AddInternal(alias, importPath)

	forward = fmt.Sprintf("%s.Bytes()", goField)
	backward = fmt.Sprintf("%s.FromBytes(%s)", alias, pbField)

	return forward, backward
}

func (p *Plugin) generateFieldConversion(
	field resolution.Field,
	data *templateData,
	parentStruct resolution.Type,
) (forward, backward, backwardCast string, hasError, hasBackwardError bool) {
	typeRef := field.Type
	goFieldName := "r." + naming.GetFieldName(field)
	pbFieldName := "pb." + lo.PascalCase(lo.SnakeCase(field.Name))

	if p.isFixedSizeUint8Array(typeRef, data.table) {
		f, b := p.generateFixedSizeUint8ArrayConversion(typeRef, data, goFieldName, pbFieldName)
		return f, b, "", false, false
	}

	if p.isArrayType(typeRef, data.table) {
		f, b, e, be := p.generateArrayConversion(field, data, goFieldName, pbFieldName)
		return f, b, "", e, be
	}

	if hasKeyDomain(field) && resolution.IsPrimitive(typeRef.Name) && isNumericPrimitive(typeRef.Name) {
		protoType := primitiveToProtoType(typeRef.Name)
		keyPkgAlias := data.parentAlias
		goOutput := output.GetPath(parentStruct, "go")
		if goOutput != "" && goOutput != data.ParentGoPath {
			importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
			if err == nil {
				keyPkgAlias = naming.DerivePackageName(goOutput)
				data.imports.AddInternal(keyPkgAlias, importPath)
			}
		}
		return fmt.Sprintf("%s(%s)", protoType, goFieldName),
			fmt.Sprintf("%s.Key(%s)", keyPkgAlias, pbFieldName),
			"", false, false
	}

	if resolution.IsPrimitive(typeRef.Name) {
		f, b, e, be := p.generatePrimitiveConversion(typeRef.Name, goFieldName, pbFieldName, data)
		return f, b, "", e, be
	}

	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return goFieldName, pbFieldName, "", false, false
	}

	if _, isUnion := resolved.Form.(resolution.UnionForm); isUnion {
		prefix, name := p.resolveUnionTranslatorName(resolved, data)
		if field.IsHardOptional {
			return fmt.Sprintf("%s%sToPB(*%s)", prefix, name, goFieldName),
				fmt.Sprintf("%s%sFromPB(%s)", prefix, name, pbFieldName),
				"", true, true
		}
		return fmt.Sprintf("%s%sToPB(%s)", prefix, name, goFieldName),
			fmt.Sprintf("%s%sFromPB(%s)", prefix, name, pbFieldName),
			"", true, true
	}

	if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
		f, b, c, hasErr := p.generateStructConversion(typeRef, resolved, field.IsHardOptional, data, goFieldName, pbFieldName)
		return f, b, c, hasErr, hasErr
	}

	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		if target, ok := aliasForm.Target.Resolve(data.table); ok {
			if _, isStruct := target.Form.(resolution.StructForm); isStruct {
				f, b, c, hasErr := p.generateStructConversion(typeRef, resolved, field.IsHardOptional, data, goFieldName, pbFieldName)
				return f, b, c, hasErr, hasErr
			}
		}
	}

	if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
		f, b := p.generateEnumConversion(typeRef, resolved, data, goFieldName, pbFieldName, field.IsHardOptional)
		return f, b, "", true, true
	}

	if form, isDistinct := resolved.Form.(resolution.DistinctForm); isDistinct {
		if baseResolved, ok := form.Base.Resolve(data.table); ok {
			if _, isStruct := baseResolved.Form.(resolution.StructForm); isStruct {
				f, b, c, hasErr := p.generateStructConversion(typeRef, resolved, field.IsHardOptional, data, goFieldName, pbFieldName)
				return f, b, c, hasErr, hasErr
			}
			// Also check if base is an alias to a struct
			if aliasForm, isAlias := baseResolved.Form.(resolution.AliasForm); isAlias {
				if target, ok := aliasForm.Target.Resolve(data.table); ok {
					if _, isStruct := target.Form.(resolution.StructForm); isStruct {
						f, b, c, hasErr := p.generateStructConversion(typeRef, resolved, field.IsHardOptional, data, goFieldName, pbFieldName)
						return f, b, c, hasErr, hasErr
					}
				}
			}
		}
		f, b, c, be := p.generateTypeDefConversion(typeRef, resolved, form, data, goFieldName, pbFieldName)
		return f, b, c, false, be
	}

	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		if resolution.IsPrimitive(aliasForm.Target.Name) {
			f, b, c, be := p.generateAliasConversion(typeRef, resolved, aliasForm, data, goFieldName, pbFieldName)
			return f, b, c, false, be
		}
	}

	return goFieldName, pbFieldName, "", false, false
}

func (p *Plugin) generatePrimitiveConversion(
	primitive, goField, pbField string,
	data *templateData,
) (forward, backward string, hasError, hasBackwardError bool) {
	switch primitive {
	case "uuid":
		data.imports.AddExternal("github.com/google/uuid")
		return fmt.Sprintf("%s.String()", goField),
			fmt.Sprintf("uuid.Parse(%s)", pbField), false, true
	case "timestamp":
		data.imports.AddExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("int64(%s)", goField),
			fmt.Sprintf("telem.TimeStamp(%s)", pbField), false, false
	case "timespan":
		data.imports.AddExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("int64(%s)", goField),
			fmt.Sprintf("telem.TimeSpan(%s)", pbField), false, false
	case "time_range", "time_range_bounded":
		data.imports.AddExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("telem.TranslateTimeRangeForward(%s)", goField),
			fmt.Sprintf("telem.TranslateTimeRangeBackward(%s)", pbField), false, false
	case "record":
		data.imports.AddExternal("google.golang.org/protobuf/types/known/structpb")
		return fmt.Sprintf("structpb.NewStruct(%s)", goField),
			fmt.Sprintf("%s.AsMap()", pbField), true, false
	case "uint12":
		data.imports.AddExternal("github.com/synnaxlabs/x/types")
		return fmt.Sprintf("uint32(%s)", goField),
			fmt.Sprintf("types.Uint12(%s)", pbField), false, false
	case "uint20":
		data.imports.AddExternal("github.com/synnaxlabs/x/types")
		return fmt.Sprintf("uint32(%s)", goField),
			fmt.Sprintf("types.Uint20(%s)", pbField), false, false
	case "data_type":
		data.imports.AddExternal("github.com/synnaxlabs/x/telem")
		return fmt.Sprintf("string(%s)", goField),
			fmt.Sprintf("telem.DataType(%s)", pbField), false, false
	case "any":
		data.imports.AddExternal("encoding/json")
		return fmt.Sprintf("json.Marshal(%s)", goField),
			fmt.Sprintf("func() any { var v any; _ = json.Unmarshal(%s, &v); return v }()", pbField), true, false
	case "int8":
		return fmt.Sprintf("int32(%s)", goField),
			fmt.Sprintf("int8(%s)", pbField), false, false
	case "int16":
		return fmt.Sprintf("int32(%s)", goField),
			fmt.Sprintf("int16(%s)", pbField), false, false
	case "uint8":
		return fmt.Sprintf("uint32(%s)", goField),
			fmt.Sprintf("uint8(%s)", pbField), false, false
	case "uint16":
		return fmt.Sprintf("uint32(%s)", goField),
			fmt.Sprintf("uint16(%s)", pbField), false, false
	default:
		return goField, pbField, false, false
	}
}

func (p *Plugin) generateStructConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	isHardOptional bool,
	data *templateData,
	goField, pbField string,
) (forward, backward, backwardCast string, hasError bool) {
	actualStruct := resolved
	var typeArgs []resolution.TypeRef

	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		typeArgs = aliasForm.Target.TypeArgs
		if target, ok := aliasForm.Target.Resolve(data.table); ok {
			actualStruct = target
		}
	} else {
		typeArgs = typeRef.TypeArgs
	}

	for {
		aliasForm, isAlias := actualStruct.Form.(resolution.AliasForm)
		if !isAlias {
			break
		}
		if len(typeArgs) == 0 && len(aliasForm.Target.TypeArgs) > 0 {
			typeArgs = aliasForm.Target.TypeArgs
		}
		if target, ok := aliasForm.Target.Resolve(data.table); ok {
			actualStruct = target
		} else {
			break
		}
	}

	actualForm, ok := actualStruct.Form.(resolution.StructForm)
	if !ok {
		return goField, pbField, "", false
	}

	if actualForm.IsGeneric() {
		nonDefaultedParams := resolution.NonDefaultedTypeParams(actualForm.TypeParams)
		providedArgs := len(typeArgs)
		if providedArgs < len(nonDefaultedParams) {
			newTypeArgs := make([]resolution.TypeRef, len(nonDefaultedParams))
			copy(newTypeArgs, typeArgs)
			for i := providedArgs; i < len(nonDefaultedParams); i++ {
				if nonDefaultedParams[i].Optional {
					newTypeArgs[i] = resolution.TypeRef{Name: "nil"}
				}
			}
			typeArgs = newTypeArgs
		}
	}

	if actualForm.IsGeneric() && len(typeArgs) > 0 {
		return p.generateGenericStructConversion(typeRef, resolved, actualStruct, actualForm, typeArgs, data, goField, pbField, isHardOptional)
	}

	// A fully-defaulted generic struct (every type param has a default and the
	// caller supplied no args) emits a non-generic Go pb translator under its
	// bare name, so the call site falls through to the regular non-generic
	// branch below instead of returning the raw field names.
	translatorPrefix, translatorStructName := p.resolvePBTranslatorInfo(actualStruct, data)

	if isHardOptional {
		return fmt.Sprintf("%s%sToPB(*%s)", translatorPrefix, translatorStructName, goField),
			fmt.Sprintf("%s%sFromPB(%s)", translatorPrefix, translatorStructName, pbField), "", true
	}

	return fmt.Sprintf("%s%sToPB(%s)", translatorPrefix, translatorStructName, goField),
		fmt.Sprintf("%s%sFromPB(%s)", translatorPrefix, translatorStructName, pbField), "", true
}

func (p *Plugin) generateGenericStructConversion(
	typeRef resolution.TypeRef,
	originalResolved resolution.Type,
	actualStruct resolution.Type,
	actualForm resolution.StructForm,
	typeArgs []resolution.TypeRef,
	data *templateData,
	goField, pbField string,
	isHardOptional bool,
) (forward, backward, backwardCast string, hasError bool) {
	translatorPrefix, structName := p.resolvePBTranslatorInfo(actualStruct, data)

	var forwardConverters, backwardConverters []string
	var explicitTypeArgs []string
	for i, typeArg := range typeArgs {
		if i < len(actualForm.TypeParams) && actualForm.TypeParams[i].HasDefault() {
			continue
		}

		if typeArg.IsTypeParam() && typeArg.TypeParam != nil && !typeArg.TypeParam.HasDefault() {
			paramName := typeArg.TypeParam.Name
			forwardConverters = append(forwardConverters, fmt.Sprintf("translate%s", paramName))
			backwardConverters = append(backwardConverters, fmt.Sprintf("translate%s", paramName))
			explicitTypeArgs = append(explicitTypeArgs, paramName)
			continue
		}

		argResolved, ok := typeArg.Resolve(data.table)
		if ok {
			if _, isStruct := argResolved.Form.(resolution.StructForm); isStruct {
				argGoName := naming.GetGoName(argResolved)

				p.ensureAnyHelper(argResolved, data)

				forwardConverters = append(forwardConverters, fmt.Sprintf("%sToPBAny", argGoName))
				backwardConverters = append(backwardConverters, fmt.Sprintf("%sFromPBAny", argGoName))

				explicitTypeArgs = append(explicitTypeArgs, fmt.Sprintf("%s.%s", data.parentAlias, argGoName))
				continue
			}
		}
		forwardConverters = append(forwardConverters, "nil")
		backwardConverters = append(backwardConverters, "nil")
		if typeArg.Name == "nil" {
			data.imports.AddInternal("gotypes", "go/types")
			explicitTypeArgs = append(explicitTypeArgs, "gotypes.Nil")
		} else {
			explicitTypeArgs = append(explicitTypeArgs, "any")
		}
	}

	forwardArgs := strings.Join(forwardConverters, ", ")
	backwardArgs := strings.Join(backwardConverters, ", ")
	typeArgsStr := "[" + strings.Join(explicitTypeArgs, ", ") + "]"

	var genericGoType string
	goOutput := output.GetPath(actualStruct, "go")
	if goOutput != "" {
		importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
		if err == nil {
			alias := naming.DerivePackageName(goOutput)
			data.imports.AddInternal(alias, importPath)
			genericGoType = fmt.Sprintf("%s.%s[%s]", alias, structName, strings.Join(explicitTypeArgs, ", "))
		}
	}

	aliasGoName := naming.GetGoName(originalResolved)
	if isHardOptional {
		if genericGoType != "" {
			forward = fmt.Sprintf("%s%sToPB%s((%s)(*%s), %s)", translatorPrefix, structName, typeArgsStr, genericGoType, goField, forwardArgs)
		} else {
			forward = fmt.Sprintf("%s%sToPB%s(*%s, %s)", translatorPrefix, structName, typeArgsStr, goField, forwardArgs)
		}
		backward = fmt.Sprintf("%s%sFromPB%s(%s, %s)", translatorPrefix, structName, typeArgsStr, pbField, backwardArgs)
		_, isAlias := originalResolved.Form.(resolution.AliasForm)
		if !isAlias && len(explicitTypeArgs) > 0 {
			backwardCast = fmt.Sprintf("(*%s.%s[%s])", data.parentAlias, aliasGoName, strings.Join(explicitTypeArgs, ", "))
		} else {
			backwardCast = fmt.Sprintf("(*%s.%s)", data.parentAlias, aliasGoName)
		}
	} else {
		if genericGoType != "" {
			forward = fmt.Sprintf("%s%sToPB%s((%s)(%s), %s)", translatorPrefix, structName, typeArgsStr, genericGoType, goField, forwardArgs)
		} else {
			forward = fmt.Sprintf("%s%sToPB%s(%s, %s)", translatorPrefix, structName, typeArgsStr, goField, forwardArgs)
		}
		backward = fmt.Sprintf("%s%sFromPB%s(%s, %s)", translatorPrefix, structName, typeArgsStr, pbField, backwardArgs)
	}

	return forward, backward, backwardCast, true
}

func (p *Plugin) ensureAnyHelper(s resolution.Type, data *templateData) {
	key := s.QualifiedName
	if data.generatedAnyHelpers.Contains(key) {
		return
	}
	data.generatedAnyHelpers.Add(key)

	data.imports.AddExternal("google.golang.org/protobuf/types/known/anypb")
	data.imports.AddExternal("google.golang.org/protobuf/types/known/structpb")
	data.imports.AddExternal("google.golang.org/protobuf/encoding/protojson")
	data.imports.AddExternal("encoding/json")

	goName := naming.GetGoName(s)

	pbName := getPBName(s)
	if pbName == "" {
		pbName = s.Name
	}

	data.AnyHelpers = append(data.AnyHelpers, anyHelperData{
		TypeName: goName,
		GoType:   fmt.Sprintf("%s.%s", data.parentAlias, goName),
		PBType:   pbName,
	})
}

func (p *Plugin) generateEnumConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	data *templateData,
	goField, pbField string,
	isHardOptional bool,
) (forward, backward string) {
	enumName := resolved.Name
	forwardArg := goField
	backwardArg := pbField
	if isHardOptional {
		forwardArg = "*" + goField
		backwardArg = "*" + pbField
	}

	pbPath := enum.FindPBOutputPath(resolved, data.table)
	if pbPath != "" && pbPath != data.OutputPath {
		importPath, err := resolveGoImportPath(pbPath, data.repoRoot)
		if err == nil {
			alias := strings.ToLower(resolved.Namespace) + "pb"
			data.imports.AddInternal(alias, importPath)
			return fmt.Sprintf("%s.%sToPB(%s)", alias, enumName, forwardArg),
				fmt.Sprintf("%s.%sFromPB(%s)", alias, enumName, backwardArg)
		}
	}

	if _, exists := data.usedEnums[resolved.QualifiedName]; !exists {
		data.usedEnums[resolved.QualifiedName] = &resolved
	}

	return fmt.Sprintf("%sToPB(%s)", enumName, forwardArg),
		fmt.Sprintf("%sFromPB(%s)", enumName, backwardArg)
}

func (p *Plugin) generateTypeDefConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	form resolution.DistinctForm,
	data *templateData,
	goField, pbField string,
) (forward, backward, backwardCast string, hasBackwardError bool) {
	baseType := form.Base
	if !resolution.IsPrimitive(baseType.Name) {
		return goField, pbField, "", false
	}

	typedefPrefix := ""
	goOutput := output.GetPath(resolved, "go")
	if resolved.Namespace != data.Namespace || (goOutput != "" && goOutput != data.ParentGoPath) {
		if goOutput != "" {
			importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
			if err == nil {
				alias := naming.DerivePackageAlias(goOutput, data.parentAlias)
				data.imports.AddInternal(alias, importPath)
				typedefPrefix = alias + "."
			}
		}
	} else {
		typedefPrefix = data.parentAlias + "."
	}

	resolvedGoName := naming.GetGoName(resolved)

	if baseType.Name == "uuid" {
		data.imports.AddExternal("github.com/google/uuid")
		forward = fmt.Sprintf("%s.String()", goField)
		backward = fmt.Sprintf("uuid.Parse(%s)", pbField)
		backwardCast = fmt.Sprintf("%s%s", typedefPrefix, resolvedGoName)
		return forward, backward, backwardCast, true
	}

	protoType := primitiveToProtoType(baseType.Name)

	forward = fmt.Sprintf("%s(%s)", protoType, goField)
	backward = fmt.Sprintf("%s%s(%s)", typedefPrefix, resolvedGoName, pbField)

	return forward, backward, "", false
}

func (p *Plugin) generateAliasConversion(
	typeRef resolution.TypeRef,
	resolved resolution.Type,
	form resolution.AliasForm,
	data *templateData,
	goField, pbField string,
) (forward, backward, backwardCast string, hasBackwardError bool) {
	primitiveName := form.Target.Name

	aliasPrefix := ""
	goOutput := output.GetPath(resolved, "go")
	if resolved.Namespace != data.Namespace || (goOutput != "" && goOutput != data.ParentGoPath) {
		if goOutput != "" {
			importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
			if err == nil {
				alias := naming.DerivePackageAlias(goOutput, data.parentAlias)
				data.imports.AddInternal(alias, importPath)
				aliasPrefix = alias + "."
			}
		}
	} else {
		aliasPrefix = data.parentAlias + "."
	}

	resolvedGoName := naming.GetGoName(resolved)

	// Handle uuid specially
	if primitiveName == "uuid" {
		data.imports.AddExternal("github.com/google/uuid")
		forward = fmt.Sprintf("%s.String()", goField)
		backward = fmt.Sprintf("uuid.Parse(%s)", pbField)
		backwardCast = fmt.Sprintf("%s%s", aliasPrefix, resolvedGoName)
		return forward, backward, backwardCast, true
	}

	protoType := primitiveToProtoType(primitiveName)
	forward = fmt.Sprintf("%s(%s)", protoType, goField)
	backward = fmt.Sprintf("%s%s(%s)", aliasPrefix, resolvedGoName, pbField)
	return forward, backward, "", false
}

func (p *Plugin) generateArrayConversion(
	field resolution.Field,
	data *templateData,
	goField, pbField string,
) (forward, backward string, hasError, hasBackwardError bool) {
	typeRef := field.Type

	if p.isNestedArrayType(typeRef, data.table) {
		f, b, e := p.generateNestedArrayConversion(typeRef, data, goField, pbField)
		return f, b, e, e
	}

	elemType, ok := p.getArrayElementType(typeRef, data.table)
	if !ok {
		return goField, pbField, false, false
	}

	if elemType.Name == "record" {
		data.NeedsRecordArrayHelpers = true
		data.imports.AddExternal("google.golang.org/protobuf/types/known/structpb")
		data.imports.AddExternal("github.com/synnaxlabs/x/encoding/msgpack")
		return fmt.Sprintf("recordsToPB(%s)", goField),
			fmt.Sprintf("recordsFromPB(%s)", pbField),
			true, false
	}

	elemResolved, ok := elemType.Resolve(data.table)
	if ok {
		if _, isUnion := elemResolved.Form.(resolution.UnionForm); isUnion {
			prefix, name := p.resolveUnionTranslatorName(elemResolved, data)
			pluralName := pluralizeDistinct(name)
			return fmt.Sprintf("%s%sToPB(%s)", prefix, pluralName, goField),
				fmt.Sprintf("%s%sFromPB(%s)", prefix, pluralName, pbField),
				true, true
		}
		if structForm, isStruct := elemResolved.Form.(resolution.StructForm); isStruct {
			translatorPrefix, translatorStructName := p.resolvePBTranslatorInfo(elemResolved, data)
			pluralName := pluralizeDistinct(translatorStructName)

			if structForm.IsGeneric() && len(elemType.TypeArgs) > 0 {
				var typeParamArgs, converterArgs []string
				for _, ta := range elemType.TypeArgs {
					if ta.IsTypeParam() && ta.TypeParam != nil && !ta.TypeParam.HasDefault() {
						typeParamArgs = append(typeParamArgs, ta.TypeParam.Name)
						converterArgs = append(converterArgs, fmt.Sprintf("translate%s", ta.TypeParam.Name))
					}
				}
				if len(typeParamArgs) > 0 {
					typeArgsStr := "[" + strings.Join(typeParamArgs, ", ") + "]"
					converterArgsStr := strings.Join(converterArgs, ", ")
					return fmt.Sprintf("%s%sToPB%s(%s, %s)", translatorPrefix, pluralName, typeArgsStr, goField, converterArgsStr),
						fmt.Sprintf("%s%sFromPB%s(%s, %s)", translatorPrefix, pluralName, typeArgsStr, pbField, converterArgsStr),
						true, true
				}
			}

			return fmt.Sprintf("%s%sToPB(%s)", translatorPrefix, pluralName, goField),
				fmt.Sprintf("%s%sFromPB(%s)", translatorPrefix, pluralName, pbField),
				true, true
		}
	}

	if resolution.IsPrimitive(elemType.Name) {
		switch elemType.Name {
		case "uuid":
			data.imports.AddExternal("github.com/google/uuid")
			data.imports.AddExternal("github.com/samber/lo")
			// Forward conversion uses lo.Map (no error possible)
			// Backward conversion uses IIFE with proper error handling
			backward = fmt.Sprintf(`func() ([]uuid.UUID, error) {
		result := make([]uuid.UUID, len(%s))
		for i, s := range %s {
			parsed, err := uuid.Parse(s)
			if err != nil {
				return nil, err
			}
			result[i] = parsed
		}
		return result, nil
	}()`, pbField, pbField)
			return fmt.Sprintf("lo.Map(%s, func(u uuid.UUID, _ int) string { return u.String() })", goField),
				backward, false, true
		}
	}

	// Distinct element types over primitive bases require per-element casts:
	// `[]channel.Key` cannot be assigned to `[]uint32` (and vice-versa), so emit
	// lo.Map conversions in both directions using the proto/distinct type names.
	if ok {
		if distinctForm, isDistinct := elemResolved.Form.(resolution.DistinctForm); isDistinct {
			if resolution.IsPrimitive(distinctForm.Base.Name) && distinctForm.Base.Name != "uuid" {
				data.imports.AddExternal("github.com/samber/lo")
				elemGoType, ok := p.qualifiedDistinctGoName(elemResolved, data)
				if ok {
					protoType := primitiveToProtoType(distinctForm.Base.Name)
					forward := fmt.Sprintf("lo.Map(%s, func(v %s, _ int) %s { return %s(v) })",
						goField, elemGoType, protoType, protoType)
					backward := fmt.Sprintf("lo.Map(%s, func(v %s, _ int) %s { return %s(v) })",
						pbField, protoType, elemGoType, elemGoType)
					return forward, backward, false, false
				}
			}
		}
	}

	return goField, pbField, false, false
}

// qualifiedDistinctGoName returns the Go identifier for a distinct type as it
// should appear in the generated translator file, including any package prefix
// and registering the import if the type lives in a different package.
func (p *Plugin) qualifiedDistinctGoName(resolved resolution.Type, data *templateData) (string, bool) {
	prefix := ""
	goOutput := output.GetPath(resolved, "go")
	if resolved.Namespace != data.Namespace || (goOutput != "" && goOutput != data.ParentGoPath) {
		if goOutput == "" {
			return "", false
		}
		importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
		if err != nil {
			return "", false
		}
		alias := naming.DerivePackageAlias(goOutput, data.parentAlias)
		data.imports.AddInternal(alias, importPath)
		prefix = alias + "."
	} else {
		prefix = data.parentAlias + "."
	}
	return prefix + naming.GetGoName(resolved), true
}

func (p *Plugin) generateNestedArrayConversion(
	typeRef resolution.TypeRef,
	data *templateData,
	goField, pbField string,
) (forward, backward string, hasError bool) {
	wrapperName := p.getNestedArrayWrapperName(typeRef, data.table)

	// Delegate per-element conversion to the inner slice's existing
	// XYZToPB / XYZFromPB helpers. This preserves type safety and error
	// propagation for nested named-slice fields (e.g., Strata []Members).
	// Falls back to the earlier broken lo.Map form only for [][]primitive,
	// which has no struct helper to call — that path was not used by any
	// schema at the time this fix landed.
	if f, b, ok := p.generateStructNestedArrayConversion(typeRef, data, goField, pbField, wrapperName); ok {
		return f, b, true
	}

	data.imports.AddExternal("github.com/samber/lo")
	forward = fmt.Sprintf("lo.Map(%s, func(inner []string, _ int) *%s { return &%s{Values: inner} })", goField, wrapperName, wrapperName)
	backward = fmt.Sprintf("lo.Map(%s, func(w *%s, _ int) []string { return w.Values })", pbField, wrapperName)
	return forward, backward, false
}

// generateStructNestedArrayConversion emits the nested-array translation for
// the common case of a slice-of-named-slice-of-struct (e.g., field type
// []Members where Members = []Member). Returns ok=false if the schema does
// not match this shape (e.g., [][]primitive), in which case the caller
// should fall back to a simpler emission.
//
// The emitted forward expression has signature `([]*<Wrapper>, error)` and
// the backward expression has signature `(<outer-go-type>, error)`. Both
// delegate to the pre-existing XYZToPB / XYZFromPB helpers that the
// generator emits for every named array type, so per-element error handling
// and type conversions stay in one place.
func (p *Plugin) generateStructNestedArrayConversion(
	typeRef resolution.TypeRef,
	data *templateData,
	goField, pbField, wrapperName string,
) (forward, backward string, ok bool) {
	elemType, ok := p.getArrayElementType(typeRef, data.table)
	if !ok {
		return "", "", false
	}
	elemResolved, ok := elemType.Resolve(data.table)
	if !ok {
		return "", "", false
	}
	innerElem, ok := p.getArrayElementType(elemType, data.table)
	if !ok {
		return "", "", false
	}
	innerElemResolved, ok := innerElem.Resolve(data.table)
	if !ok {
		return "", "", false
	}
	if _, isStruct := innerElemResolved.Form.(resolution.StructForm); !isStruct {
		return "", "", false
	}

	translatorPrefix, translatorStructName := p.resolvePBTranslatorInfo(innerElemResolved, data)
	pluralName := pluralizeDistinct(translatorStructName)

	// If the outer typeRef resolves to a distinct named type (e.g., Strata),
	// use its qualified Go name so the IIFE's make() and return types match
	// the field exactly. Otherwise, use []<elem-go-type>, which is assignable
	// to an unnamed outer slice field.
	outerGoType := ""
	if outerResolved, ok := typeRef.Resolve(data.table); ok {
		if _, isDistinct := outerResolved.Form.(resolution.DistinctForm); isDistinct {
			outerGoType = data.parentAlias + "." + outerResolved.Name
		}
	}
	if outerGoType == "" {
		outerGoType = "[]" + data.parentAlias + "." + elemResolved.Name
	}

	forward = fmt.Sprintf(`func() ([]*%s, error) {
		result := make([]*%s, len(%s))
		for i, inner := range %s {
			vals, err := %s%sToPB(inner)
			if err != nil {
				return nil, err
			}
			result[i] = &%s{Values: vals}
		}
		return result, nil
	}()`, wrapperName, wrapperName, goField, goField, translatorPrefix, pluralName, wrapperName)

	backward = fmt.Sprintf(`func() (%s, error) {
		result := make(%s, len(%s))
		for i, w := range %s {
			vals, err := %s%sFromPB(w.Values)
			if err != nil {
				return nil, err
			}
			result[i] = vals
		}
		return result, nil
	}()`, outerGoType, outerGoType, pbField, pbField, translatorPrefix, pluralName)

	return forward, backward, true
}

func (p *Plugin) generateEnumTranslator(
	enumRef *resolution.Type,
	data *templateData,
) *enumTranslatorData {
	form, ok := enumRef.Form.(resolution.EnumForm)
	if !ok {
		return nil
	}

	goName := naming.GetGoName(*enumRef)

	values := make([]enumValueTranslatorData, 0, len(form.Values))

	goAlias := data.parentAlias

	for _, v := range form.Values {
		valueName := naming.ToPascalCase(v.Name)

		goValue := fmt.Sprintf("%s.%s%s", goAlias, goName, valueName)

		enumPrefix := toScreamingSnake(enumRef.Name) + "_"
		pbValueName := fmt.Sprintf("%s_%s%s", enumRef.Name, enumPrefix, toScreamingSnake(v.Name))

		values = append(values, enumValueTranslatorData{
			GoValue: goValue,
			PBValue: pbValueName,
		})
	}

	return &enumTranslatorData{
		Name:      enumRef.Name,
		GoType:    fmt.Sprintf("%s.%s", goAlias, goName),
		PBType:    enumRef.Name,
		IsIntEnum: form.IsIntEnum,
		Values:    values,
	}
}

func resolveGoImportPath(outputPath, repoRoot string) (string, error) {
	return gomod.ResolveImportPath(outputPath, repoRoot, gomod.DefaultModulePrefix), nil
}

func getPBName(s resolution.Type) string {
	if domain, ok := s.Domains["pb"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
}

func findStructWithPB(s resolution.Type, table *resolution.Table) (*resolution.Type, string) {
	current := &s
	for current != nil {
		pbPath := output.GetPBPath(*current)
		if pbPath != "" {
			return current, pbPath
		}
		form, ok := current.Form.(resolution.StructForm)
		if !ok || len(form.Extends) == 0 {
			break
		}
		parent, ok := form.Extends[0].Resolve(table)
		if !ok {
			break
		}
		current = &parent
	}
	return nil, ""
}

func (p *Plugin) resolvePBTranslatorInfo(
	structRef resolution.Type,
	data *templateData,
) (translatorPrefix, translatorStructName string) {
	pbStruct, pbPath := findStructWithPB(structRef, data.table)
	if pbStruct == nil {
		return "", structRef.Name
	}

	translatorStructName = getPBName(*pbStruct)
	if translatorStructName == "" {
		translatorStructName = pbStruct.Name
	}

	if pbStruct.Namespace != data.Namespace || (pbPath != "" && pbPath != data.OutputPath) {
		importPath, err := resolveGoImportPath(pbPath, data.repoRoot)
		if err == nil {
			alias := strings.ToLower(pbStruct.Namespace) + "pb"
			data.imports.AddInternal(alias, importPath)
			translatorPrefix = alias + "."
		}
	}

	return translatorPrefix, translatorStructName
}

func hasKeyDomain(field resolution.Field) bool {
	_, hasKey := field.Domains["key"]
	return hasKey
}

func isNumericPrimitive(primitive string) bool {
	switch primitive {
	case "uint8", "uint16", "uint32", "uint64",
		"int8", "int16", "int32", "int64":
		return true
	default:
		return false
	}
}

func primitiveToProtoType(primitive string) string {
	switch primitive {
	case "uint8", "uint12", "uint16", "uint20", "uint32":
		return "uint32"
	case "uint64":
		return "uint64"
	case "int8", "int16", "int32":
		return "int32"
	case "int64":
		return "int64"
	default:
		return primitive
	}
}

func primitiveNeedsConversion(primitive string) bool {
	return primitiveToProtoType(primitive) != primitive
}

func toScreamingSnake(s string) string {
	return strings.ToUpper(lo.SnakeCase(s))
}

// isUnionType reports whether the type reference resolves to a discriminated
// union.
func isUnionType(typeRef resolution.TypeRef, table *resolution.Table) bool {
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return false
	}
	_, isUnion := resolved.Form.(resolution.UnionForm)
	return isUnion
}

func isStructType(typeRef resolution.TypeRef, table *resolution.Table) bool {
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return false
	}
	if _, isStruct := resolved.Form.(resolution.StructForm); isStruct {
		return true
	}
	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		if target, ok := aliasForm.Target.Resolve(table); ok {
			_, isStruct := target.Form.(resolution.StructForm)
			return isStruct
		}
	}
	return false
}

func isEnumType(typeRef resolution.TypeRef, table *resolution.Table) bool {
	resolved, ok := typeRef.Resolve(table)
	if !ok {
		return false
	}
	if _, isEnum := resolved.Form.(resolution.EnumForm); isEnum {
		return true
	}
	if aliasForm, isAlias := resolved.Form.(resolution.AliasForm); isAlias {
		if target, ok := aliasForm.Target.Resolve(table); ok {
			_, isEnum := target.Form.(resolution.EnumForm)
			return isEnum
		}
	}
	return false
}

func (p *Plugin) resolveGoTypeLiteral(typeRef resolution.TypeRef, data *templateData) string {
	resolved, ok := typeRef.Resolve(data.table)
	if !ok {
		return ""
	}
	goName := naming.GetGoName(resolved)
	goOutput := output.GetPath(resolved, "go")
	if goOutput == "" || goOutput == data.ParentGoPath {
		return fmt.Sprintf("%s.%s", data.parentAlias, goName)
	}
	importPath, err := resolveGoImportPath(goOutput, data.repoRoot)
	if err != nil {
		return fmt.Sprintf("%s.%s", data.parentAlias, goName)
	}
	alias := naming.DerivePackageAlias(goOutput, data.parentAlias)
	data.imports.AddInternal(alias, importPath)
	return fmt.Sprintf("%s.%s", alias, goName)
}

type templateData struct {
	usedEnums             map[string]*resolution.Type
	table                 *resolution.Table
	imports               *imports.Manager
	generatedAnyHelpers   set.Set[string]
	ParentGoPath          string
	Package               string
	OutputPath            string
	Namespace             string
	repoRoot              string
	parentAlias           string
	DelegationTranslators []delegationTranslatorData
	AnyHelpers            []anyHelperData
	EnumTranslators       []enumTranslatorData
	GenericTranslators    []genericTranslatorData
	Translators           []translatorData
	UnionTranslators      []unionTranslatorData
	// NeedsRecordArrayHelpers reports whether any field converts a []record,
	// requiring the shared recordsToPB/recordsFromPB helpers.
	NeedsRecordArrayHelpers bool
}

// HasImports returns true if any imports are needed.
func (d *templateData) HasImports() bool { return d.imports.HasImports() }

// ExternalImports returns sorted external imports.
func (d *templateData) ExternalImports() []string { return d.imports.ExternalImports() }

// InternalImports returns sorted internal imports.
func (d *templateData) InternalImports() []imports.InternalImportData {
	return d.imports.InternalImports()
}

// translatorData holds data for a single type's translators.
type translatorData struct {
	Name        string
	GoType      string
	PBType      string
	GoTypeShort string
	PBTypeShort string
	Fields      []fieldTranslatorData
	// ErrorFields holds fields with error-returning conversions.
	ErrorFields    []fieldTranslatorData
	OptionalFields []fieldTranslatorData
}

// fieldTranslatorData holds data for a single field translation.
type fieldTranslatorData struct {
	GoName       string
	PBName       string
	ForwardExpr  string
	BackwardExpr string
	// BackwardCast is an optional cast for the backward assignment (e.g., "(*rack.Status)").
	BackwardCast     string
	IsOptional       bool
	IsOptionalStruct bool
	// IsOptionalEnum is true for a hard-optional field whose underlying type is an
	// enum. Triggers the same val/&val backward dance as IsOptionalStruct so the
	// pointer-typed Go field is populated from the value-returning EnumFromPB call.
	IsOptionalEnum bool
	// NeedsPtrConversion is true when a hard-optional primitive needs type conversion
	// (e.g., *uint8 <-> *uint32). The template must dereference, convert, and re-address.
	NeedsPtrConversion bool
	// MapValueConversion holds the forward and backward conversion expressions for map
	// value types that need conversion (e.g., uint8 <-> uint32). When set, the template
	// generates an element-wise conversion loop instead of a direct assignment.
	MapValueConversion *mapValueConversionData
	// HasError is true if forward conversion returns (result, error).
	HasError bool
	// HasBackwardError is true if backward conversion returns (result, error).
	HasBackwardError bool
}

type mapValueConversionData struct {
	GoMapType string // e.g., "map[uint32]uint8"
	PBMapType string // e.g., "map[uint32]uint32"
	// ForwardValueExpr is the conversion for a single value, using "v" as placeholder.
	ForwardValueExpr string // e.g., "uint32(v)"
	// BackwardValueExpr is the conversion for a single value, using "v" as placeholder.
	BackwardValueExpr string // e.g., "uint8(v)"
	// ForwardHasError is true when ForwardValueExpr returns (T, error). The
	// template must capture both, propagate err on failure, and assign T into
	// the map.
	ForwardHasError bool
	// BackwardHasError is true when BackwardValueExpr returns (T, error). The
	// template emits the same shape on the FromPB side.
	BackwardHasError bool
}

// enumTranslatorData holds data for enum translator functions.
type enumTranslatorData struct {
	Name      string
	GoType    string
	PBType    string
	IsIntEnum bool
	Values    []enumValueTranslatorData
}

// enumValueTranslatorData holds data for a single enum value translation.
type enumValueTranslatorData struct {
	GoValue string
	PBValue string
}

// genericTranslatorData holds data for a generic type's translators.
// These are translator functions with type parameters that accept converter
// functions for each type parameter.
type genericTranslatorData struct {
	Name string
	// GoType is the full generic type with parameters (e.g., "status.Status[D]").
	GoType string
	// GoTypeBase is the type without parameters (e.g., "status.Status").
	GoTypeBase  string
	PBType      string
	GoTypeShort string
	PBTypeShort string
	TypeParams  []typeParamData
	// Fields holds regular fields that don't use type parameters.
	Fields []fieldTranslatorData
	// ErrorFields holds fields with error-returning conversions.
	ErrorFields []fieldTranslatorData
	// TypeParamFields holds fields that use type parameters and need error handling.
	TypeParamFields []fieldTranslatorData
	OptionalFields  []fieldTranslatorData
}

// typeParamData holds data for a type parameter in a generic translator.
type typeParamData struct {
	// Name is the type parameter name (e.g., "D").
	Name string
	// Constraint is the Go type constraint (e.g., "any").
	Constraint string
}

func typeParamConstraint(tp resolution.TypeParam) string {
	if tp.Constraint != nil && resolution.IsConstraint(tp.Constraint.Name) {
		return tp.Constraint.Name
	}
	return "any"
}

// anyHelperData holds data for ToPBAny/FromPBAny helper functions.
// These are generated for concrete types that are used as type arguments
// to generic structs.
type anyHelperData struct {
	// TypeName is the unqualified type name (e.g., "StatusDetails").
	TypeName string
	// GoType is the fully qualified Go type (e.g., "rack.StatusDetails").
	GoType string
	// PBType is the protobuf message type name (e.g., "PBStatusDetails").
	PBType string
}

// delegationTranslatorData holds data for translators that delegate to an underlying type.
// Used for DistinctForm types that wrap struct types - instead of generating independent
// translators, we generate thin wrappers that cast and delegate.
type delegationTranslatorData struct {
	Name                       string
	GoType                     string
	UnderlyingName             string
	UnderlyingGoType           string
	UnderlyingPBType           string
	UnderlyingTranslatorPrefix string
	TypeParams                 []typeParamData
}
