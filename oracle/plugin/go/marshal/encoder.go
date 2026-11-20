// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package marshal

import (
	"bytes"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/typemap"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// --- Output data structures ---

type concreteCodec struct {
	GoName     string
	Receiver   string
	EncodeBody string
	DecodeBody string
	UsesErr    bool
	// Recursive marks types whose DecodeOrc can re-enter itself; their decode bodies
	// are guarded with PushDepth/PopDepth against malicious nesting.
	Recursive bool
}

type typeParamData struct {
	Name       string
	Constraint string
}

type genericCodec struct {
	GoName     string
	Receiver   string
	TypeParams []typeParamData
	EncodeBody string
	DecodeBody string
	UsesErr    bool
	// Recursive marks types whose DecodeOrc can re-enter itself; their decode bodies
	// are guarded with PushDepth/PopDepth against malicious nesting.
	Recursive bool
}

type encoderFileOutput struct {
	Package        string
	ExtraImports   map[string]string
	NeedsJSON      bool
	NeedsJSONText  bool
	HasFlex        bool
	ConcreteCodecs []concreteCodec
	GenericCodecs  []genericCodec
	FlexMethods    string
}

// --- Generation entry point ---

// generateEncoderCodecFile renders the codec file, then renders it again when it
// imports two packages that derive one alias, so the aliases follow
// naming.ImportAliases.
func generateEncoderCodecFile(
	packageName string,
	parentPath string,
	entries []codecEntry,
	flex []flexCodec,
	table *resolution.Table,
	repoRoot string,
	pins versioning.Pins,
) ([]byte, error) {
	imports := make(map[string]string)
	content, err := renderEncoderCodecFile(
		packageName, parentPath, entries, flex, table, repoRoot, pins, nil,
		imports,
	)
	if err != nil {
		return nil, err
	}
	current := resolveGoImportPath(parentPath, repoRoot)
	aliases := naming.ImportAliases(slices.Collect(maps.Keys(imports)), current)
	if len(aliases) == 0 {
		return content, nil
	}
	return renderEncoderCodecFile(
		packageName, parentPath, entries, flex, table, repoRoot, pins, aliases,
		make(map[string]string),
	)
}

// renderEncoderCodecFile renders the codec file, recording its imports into imports.
// aliases forces the alias of each import path it holds.
func renderEncoderCodecFile(
	packageName string,
	parentPath string,
	entries []codecEntry,
	flex []flexCodec,
	table *resolution.Table,
	repoRoot string,
	pins versioning.Pins,
	aliases map[string]string,
	imports map[string]string,
) ([]byte, error) {
	fo := encoderFileOutput{
		Package:      packageName,
		ExtraImports: imports,
	}

	if len(flex) > 0 {
		if err := fo.addFlex(flex); err != nil {
			return nil, err
		}
	}
	for _, e := range entries {
		b := &encoderBuilder{
			table:       table,
			repoRoot:    repoRoot,
			packageName: packageName,
			parentPath:  parentPath,
			imports:     fo.ExtraImports,
			aliases:     aliases,
			scope:       pinScope{pins: pins, owners: []string{e.Type.QualifiedName}},
		}
		if uform, isUnion := e.Type.Form.(resolution.UnionForm); isUnion {
			if err := fo.addUnionCodec(e, uform, b); err != nil {
				return nil, err
			}
			continue
		}
		form, ok := e.Type.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		if err := fo.addStructCodec(e, form, b); err != nil {
			return nil, err
		}
	}
	tmpl, err := template.New("encoder_codec").Funcs(template.FuncMap{
		"tpNames":       tpNames,
		"sortedImports": sortedImports,
	}).Parse(encoderCodecTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse encoder template")
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, fo); err != nil {
		return nil, errors.Wrap(err, "failed to execute encoder template")
	}
	return buf.Bytes(), nil
}

// addFlex adds the flex methods for distinct scalar types, in sorted order.
func (fo *encoderFileOutput) addFlex(flex []flexCodec) error {
	sort.Slice(flex, func(i, j int) bool { return flex[i].GoName < flex[j].GoName })
	var flexBuf strings.Builder
	for _, fc := range flex {
		methods, err := generateFlexMethods(fc)
		if err != nil {
			return err
		}
		flexBuf.WriteString(methods)
	}
	fo.FlexMethods = flexBuf.String()
	fo.HasFlex = true
	fo.NeedsJSONText = true
	fo.ExtraImports["github.com/vmihailenco/msgpack/v5"] = ""
	fo.ExtraImports["github.com/synnaxlabs/x/encoding/msgpack"] = "xmsgpack"
	fo.ExtraImports["github.com/synnaxlabs/x/encoding/json"] = "xjson"
	return nil
}

// addUnionCodec adds the codec for the union entry e.
func (fo *encoderFileOutput) addUnionCodec(
	e codecEntry,
	uform resolution.UnionForm,
	b *encoderBuilder,
) error {
	uc, err := buildUnionCodec(e.Type, uform, b)
	if err != nil {
		return errors.Wrapf(err, "failed to generate union codec for %s", e.GoName)
	}
	if b.needsJSON {
		fo.NeedsJSON = true
	}
	uc.Recursive = typeIsRecursive(e.Type, b.table)
	fo.ConcreteCodecs = append(fo.ConcreteCodecs, uc)
	return nil
}

// addStructCodec adds the codec for the struct entry e, generic when the struct has
// type params without defaults.
func (fo *encoderFileOutput) addStructCodec(
	e codecEntry,
	form resolution.StructForm,
	b *encoderBuilder,
) error {
	// Collect type params that produce Go generics. A type param becomes a Go generic
	// parameter only when it has NO default value (like Details? in Status<Details?>).
	// Params with any default (like V = Variant or Data? = record) are substituted by
	// the types plugin and the Go type is concrete.
	var typeParams []typeParamData
	if form.IsGeneric() {
		for _, tp := range resolution.NonDefaultedTypeParams(form.TypeParams) {
			typeParams = append(typeParams, typeParamData{
				Name:       tp.Name,
				Constraint: typemap.TypeParamConstraint(tp),
			})
		}
	}
	recv := receiverName(e.GoName)
	fields := resolution.UnifiedFields(e.Type, b.table)
	if len(typeParams) == 0 {
		if err := b.processFields(fields, recv, recv); err != nil {
			return errors.Wrapf(err, "failed to generate codec for %s", e.GoName)
		}
		if b.needsJSON {
			fo.NeedsJSON = true
		}
		fo.ConcreteCodecs = append(fo.ConcreteCodecs, concreteCodec{
			GoName:     e.GoName,
			Receiver:   recv,
			EncodeBody: strings.Join(b.encodeLines, "\n"),
			DecodeBody: strings.Join(b.decodeLines, "\n"),
			UsesErr:    b.usesErr,
			Recursive:  typeIsRecursive(e.Type, b.table),
		})
		return nil
	}
	// Generic type with non-defaulted params: generate method with type assertion +
	// JSON fallback for type parameter fields.
	b.hasTypeParams = true
	if err := b.processFields(fields, recv, recv); err != nil {
		return errors.Wrapf(err, "failed to generate generic codec for %s", e.GoName)
	}
	if b.needsJSON {
		fo.NeedsJSON = true
	}
	fo.GenericCodecs = append(fo.GenericCodecs, genericCodec{
		GoName:     e.GoName,
		Receiver:   recv,
		TypeParams: typeParams,
		EncodeBody: strings.Join(b.encodeLines, "\n"),
		DecodeBody: strings.Join(b.decodeLines, "\n"),
		UsesErr:    b.usesErr,
		Recursive:  typeIsRecursive(e.Type, b.table),
	})
	return nil
}

// reservedNames contains single-letter variable names used in generated method bodies
// (parameters, loop vars, temporaries) that would conflict with a receiver.
var reservedNames = set.New(
	"w", "r", // method parameters
	"n", "b", "v", "m", // temporaries
	"i", "j", "k", "l", // loop indices
	"err", "ok", // error/bool variables in method bodies
)

// receiverName derives a Go-idiomatic short receiver name from a type name. It takes
// the lowercase initials of each word in the PascalCase name (e.g., "TimeRange" ->
// "tr", "XY" -> "xy", "Status" -> "s"). Names that conflict with generated local
// variables get a "v" suffix.
func receiverName(goName string) string {
	var initials []byte
	for i, c := range goName {
		if i == 0 || (c >= 'A' && c <= 'Z') {
			initials = append(initials, byte(c|0x20)) // lowercase
		}
	}
	name := string(initials)
	if reservedNames.Contains(name) {
		return name + "v"
	}
	return name
}

// Template helper for generic type parameter name lists.
func tpNames(tps []typeParamData) string {
	parts := make([]string, len(tps))
	for i, tp := range tps {
		parts[i] = tp.Name
	}
	return strings.Join(parts, ", ")
}

// --- Encoder builder ---

type encoderBuilder struct {
	table       *resolution.Table
	repoRoot    string
	packageName string
	parentPath  string
	imports     map[string]string
	// aliases forces the alias of each import path it holds.
	aliases       map[string]string
	scope         pinScope
	encodeLines   []string
	decodeLines   []string
	needsMath     bool
	needsJSON     bool
	usesErr       bool
	depth         int
	inBlock       int
	skipNilCheck  bool
	hasTypeParams bool
}

var loopIndexVars = []string{"i", "j", "k", "l", "m"}

func (b *encoderBuilder) indent() string { return strings.Repeat("\t", b.depth+1) }

func (b *encoderBuilder) loopIndex() string {
	if b.depth < len(loopIndexVars) {
		return loopIndexVars[b.depth]
	}
	return fmt.Sprintf("i%d", b.depth)
}

func (b *encoderBuilder) decodeLine(line string) {
	b.decodeLines = append(b.decodeLines, line)
}

func (b *encoderBuilder) decodeWithErr(line string) {
	if b.inBlock == 0 {
		b.usesErr = true
	}
	b.decodeLines = append(b.decodeLines, line)
}

func (b *encoderBuilder) processFields(
	fields []resolution.Field,
	getPrefix, setPrefix string,
) error {
	for _, f := range fields {
		if f.Type.Name == "nil" || !b.canResolve(f.Type) {
			continue
		}
		marshalDirective := domain.GetStringFromField(f, "go", "marshal")
		if marshalDirective == "omit" {
			continue
		}
		goName := naming.GetFieldName(f)
		getPath := getPrefix + "." + goName
		setPath := setPrefix + "." + goName

		// Type parameter fields use a type assertion with JSON fallback, but only when
		// the type param has no default. Defaulted type params (e.g. V extends Variant
		// = Variant) are substituted with their default and encoded concretely.
		if f.Type.IsTypeParam() && b.hasTypeParams {
			if f.Type.TypeParam.HasDefault() {
				f.Type = *f.Type.TypeParam.Default
			} else {
				if err := b.processTypeParamField(
					getPath,
					setPath,
					marshalDirective == "json_only",
				); err != nil {
					return err
				}
				continue
			}
		}

		switch {
		case f.Optional && !typemap.Nilable(f.Type, b.table):
			if err := b.processOptional(f, getPath, setPath); err != nil {
				return err
			}
		case f.Optional:
			if err := b.processOptionalNilable(f, getPath, setPath); err != nil {
				return err
			}
		default:
			if err := b.processFieldValue(f, getPath, setPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *encoderBuilder) processFieldValue(
	f resolution.Field, getPath, setPath string,
) error {
	resolved, ok := b.resolveTypeRef(f.Type)
	if !ok {
		return errors.Newf("cannot resolve type %q (field=%s)", f.Type.Name, f.Name)
	}
	return b.processValueByType(resolved, f.Type, getPath, setPath)
}

func (b *encoderBuilder) processValueByType(
	resolved resolution.Type, ref resolution.TypeRef,
	getPath, setPath string,
) error {
	// Type parameter fields use a type assertion with JSON fallback.
	if ref.IsTypeParam() && b.hasTypeParams {
		return b.processTypeParamField(getPath, setPath, false)
	}

	actual, effectiveTypeArgs := typemap.UnwrapTypeRef(resolved, ref, b.table)

	switch form := actual.Form.(type) {
	case resolution.StructForm:
		return b.processStruct(actual, form, effectiveTypeArgs, getPath, setPath)
	case resolution.UnionForm:
		// Union wrappers carry their own generated EncodeOrc/DecodeOrc (a binary
		// discriminator tag plus the variant's struct codecs), so union fields dispatch
		// like struct fields.
		ind := b.indent()
		b.encodeLines = append(
			b.encodeLines,
			ind+fmt.Sprintf(
				"if err := %s.EncodeOrc(w); err != nil { return err }",
				callPath(getPath),
			),
		)
		b.decodeWithErr(
			ind + fmt.Sprintf(
				"if err = %s.DecodeOrc(r); err != nil { return err }",
				setPath,
			),
		)
		return nil
	case resolution.BuiltinGenericForm:
		if form.Name == "Array" {
			typeArgs := ref.TypeArgs
			if len(typeArgs) == 0 {
				typeArgs = effectiveTypeArgs
			}
			if len(typeArgs) > 0 {
				fakeField := resolution.Field{Type: resolution.TypeRef{
					Name:     "Array",
					TypeArgs: typeArgs,
				}}
				return b.processArray(fakeField, getPath, setPath)
			}
		}
		if form.Name == "Map" {
			typeArgs := ref.TypeArgs
			if len(typeArgs) == 0 {
				typeArgs = effectiveTypeArgs
			}
			if len(typeArgs) >= 2 {
				return b.processMap(typeArgs[0], typeArgs[1], getPath, setPath)
			}
		}
		return errors.Newf("unsupported builtin generic: %s", form.Name)
	default:
		return b.processLeaf(resolved, getPath, setPath)
	}
}

func (b *encoderBuilder) processTypeParamField(
	getPath, setPath string,
	jsonOnly bool,
) error {
	b.needsJSON = true
	ind := b.indent()
	if jsonOnly {
		b.encodeLines = append(b.encodeLines,
			ind+"{",
			ind+fmt.Sprintf("\tb, err := json.Marshal(%s)", getPath),
			ind+"\tif err != nil { return err }",
			ind+"\tw.WriteWithLen(b)",
			ind+"}",
		)
		b.decodeLines = append(
			b.decodeLines,
			ind+"{",
			ind+"\tb, err := r.ReadWithLen(); if err != nil { return err }",
			ind+fmt.Sprintf(
				"\tif err = json.Unmarshal(b, &%s); err != nil { return err }",
				setPath,
			),
			ind+"}",
		)
	} else {
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("if m, ok := any(%s).(orc.SelfEncoder); ok {", getPath),
			ind+"\tif err := m.EncodeOrc(w); err != nil { return err }",
			ind+"} else {",
			ind+fmt.Sprintf("\tb, err := json.Marshal(%s)", getPath),
			ind+"\tif err != nil { return err }",
			ind+"\tw.WriteWithLen(b)",
			ind+"}",
		)
		b.decodeLines = append(
			b.decodeLines,
			ind+fmt.Sprintf("if m, ok := any(&%s).(orc.SelfDecoder); ok {", setPath),
			ind+"\tif err := m.DecodeOrc(r); err != nil { return err }",
			ind+"} else {",
			ind+"\tb, err := r.ReadWithLen(); if err != nil { return err }",
			ind+fmt.Sprintf(
				"\tif err = json.Unmarshal(b, &%s); err != nil { return err }",
				setPath,
			),
			ind+"}",
		)
	}
	return nil
}

func (b *encoderBuilder) processStruct(
	actual resolution.Type,
	form resolution.StructForm,
	effectiveTypeArgs []resolution.TypeRef,
	getPath, setPath string,
) error {
	ind := b.indent()

	if form.IsGeneric() && len(effectiveTypeArgs) == 0 {
		// Generic without args - inline fields with defaults substituted.
		innerFields := resolution.UnifiedFields(actual, b.table)
		typeArgMap := make(map[string]resolution.TypeRef)
		for _, tp := range form.TypeParams {
			if tp.HasDefault() {
				typeArgMap[tp.Name] = *tp.Default
			}
		}
		substituted := make([]resolution.Field, len(innerFields))
		for i, f := range innerFields {
			substituted[i] = f
			substituted[i].Type = resolution.SubstituteTypeRef(f.Type, typeArgMap)
		}
		defer b.scope.enter(actual.QualifiedName)()
		return b.processFields(substituted, getPath, setPath)
	}

	// Method dispatch: call EncodeOrc/DecodeOrc on the field directly.
	b.encodeLines = append(
		b.encodeLines,
		ind+fmt.Sprintf(
			"if err := %s.EncodeOrc(w); err != nil { return err }",
			callPath(getPath),
		),
	)
	b.decodeWithErr(
		ind + fmt.Sprintf(
			"if err = %s.DecodeOrc(r); err != nil { return err }",
			setPath,
		),
	)
	return nil
}

// callPath strips the parenthesized deref an optional field's path carries: method
// calls auto-dereference pointers, so (*d.Status).EncodeOrc reads as
// d.Status.EncodeOrc.
func callPath(getPath string) string {
	if strings.HasPrefix(getPath, "(*") && strings.HasSuffix(getPath, ")") {
		return getPath[2 : len(getPath)-1]
	}
	return getPath
}

// valuePath strips the redundant outer parentheses an optional field's deref path
// carries so a scalar value reads as *d.Foo rather than (*d.Foo). Unlike callPath it
// keeps the leading star, since the value itself is passed on.
func valuePath(getPath string) string {
	if strings.HasPrefix(getPath, "(*") && strings.HasSuffix(getPath, ")") {
		return getPath[1 : len(getPath)-1]
	}
	return getPath
}

// typeIsRecursive reports whether decoding typ can re-enter its own codec, through
// struct fields (including inherited ones) or through a union's bases and variant
// payloads. Recursive codecs guard DecodeOrc with a depth limit.
func typeIsRecursive(typ resolution.Type, table *resolution.Table) bool {
	switch form := typ.Form.(type) {
	case resolution.UnionForm:
		for _, ext := range form.Extends {
			if resolution.RefersTo(ext, typ.QualifiedName, table) {
				return true
			}
		}
		for _, v := range form.Variants {
			if resolution.RefersTo(v.Type, typ.QualifiedName, table) {
				return true
			}
		}
	case resolution.StructForm:
		for _, f := range resolution.UnifiedFields(typ, table) {
			if resolution.RefersTo(f.Type, typ.QualifiedName, table) {
				return true
			}
		}
	}
	return false
}

// buildUnionCodec generates the EncodeOrc/DecodeOrc bodies for a discriminated union
// wrapper. The encoding is fully binary: a length-prefixed discriminator string
// followed by the active variant's base and payload structs encoded positionally
// through their own codecs. The discriminator string keeps stored bytes stable under
// variant addition and reordering; variant field changes version through frozen codecs
// like any struct change. indentLines shifts builder-emitted method-body statements one
// tab deeper so they sit inside a union codec's switch case.
func indentLines(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		for sub := range strings.SplitSeq(l, "\n") {
			out = append(out, "\t"+sub)
		}
	}
	return out
}

func buildUnionCodec(
	entry resolution.Type,
	form resolution.UnionForm,
	b *encoderBuilder,
) (concreteCodec, error) {
	table := b.table
	goName := naming.GetGoName(entry)
	recv := receiverName(goName)
	if recv == "tag" {
		recv += "v"
	}
	b.imports["github.com/synnaxlabs/x/errors"] = ""

	var baseEmbeds []string
	for _, ext := range form.Extends {
		parent, ok := ext.Resolve(table)
		if !ok {
			return concreteCodec{}, errors.Newf(
				"union %s: unresolved base %s", entry.Name, ext.Name,
			)
		}
		baseEmbeds = append(baseEmbeds, naming.GetGoName(parent))
	}

	enc := []string{fmt.Sprintf("\tswitch v := %s.Variant.(type) {", recv)}
	dec := []string{
		"\ttag, err := r.StringBytes()",
		"\tif err != nil { return err }",
		"\tswitch string(tag) {",
	}
	for _, v := range form.Variants {
		caseEnc, caseDec, err := b.variantCase(entry, form, v, baseEmbeds, goName, recv)
		if err != nil {
			return concreteCodec{}, err
		}
		enc = append(enc, caseEnc...)
		dec = append(dec, caseDec...)
	}
	enc = append(enc,
		"\tdefault:",
		fmt.Sprintf(
			"\t\treturn errors.Newf(\"%s: nil or unknown variant %%T\", %s.Variant)",
			goName, recv,
		),
		"\t}",
	)
	dec = append(dec,
		"\tdefault:",
		fmt.Sprintf("\t\treturn errors.Newf(\"%s: unknown variant %%q\", tag)", goName),
		"\t}",
	)
	return concreteCodec{
		GoName:     goName,
		Receiver:   recv,
		EncodeBody: strings.Join(enc, "\n"),
		DecodeBody: strings.Join(dec, "\n"),
	}, nil
}

// variantCase returns the encode and decode statements of the switch case for the
// variant v of a union.
func (b *encoderBuilder) variantCase(
	entry resolution.Type,
	form resolution.UnionForm,
	v resolution.UnionVariant,
	baseEmbeds []string,
	goName, recv string,
) ([]string, []string, error) {
	embeds, inlineFields, err := variantEmbeds(entry, form, v, baseEmbeds, b.table)
	if err != nil {
		return nil, nil, err
	}
	variantType := casing.VariantTypeName(goName, v.Name)
	enc := []string{
		fmt.Sprintf("\tcase %s:", variantType),
		fmt.Sprintf("\t\tw.String(%q)", v.Name),
	}
	dec := []string{fmt.Sprintf("\tcase %q:", v.Name), "\t\tvar v " + variantType}
	for _, embed := range embeds {
		enc = append(enc, fmt.Sprintf(
			"\t\tif err := v.%s.EncodeOrc(w); err != nil { return err }", embed,
		))
		dec = append(dec, fmt.Sprintf(
			"\t\tif err := v.%s.DecodeOrc(r); err != nil { return err }", embed,
		))
	}
	if len(inlineFields) > 0 {
		fieldEnc, fieldDec, err := b.inlineVariantLines(inlineFields)
		if err != nil {
			return nil, nil, errors.Wrapf(err,
				"union %s variant %q: inline field codec", entry.Name, v.Name)
		}
		enc = append(enc, fieldEnc...)
		dec = append(dec, fieldDec...)
	}
	dec = append(dec, fmt.Sprintf("\t\t%s.Variant = v", recv))
	return enc, dec, nil
}

// variantEmbeds returns the Go names of the types that the variant v of a union embeds,
// and the fields that an inline variant declares itself.
func variantEmbeds(
	entry resolution.Type,
	form resolution.UnionForm,
	v resolution.UnionVariant,
	baseEmbeds []string,
	table *resolution.Table,
) ([]string, []resolution.Field, error) {
	payload, ok := v.Type.Resolve(table)
	if !ok {
		return nil, nil, errors.Newf(
			"union %s variant %q: unresolved payload %s",
			entry.Name, v.Name, v.Type.Name,
		)
	}
	if !v.Inline {
		return append(slices.Clone(baseEmbeds), naming.GetGoName(payload)), nil, nil
	}
	pform := resolution.MustForm[resolution.StructForm](payload)
	inherited, declared := resolver.VariantBases(form, v, "go", table)
	var embeds []string
	for _, ext := range inherited {
		parent, ok := ext.Resolve(table)
		if !ok {
			return nil, nil, errors.Newf(
				"union %s variant %q: unresolved base %s",
				entry.Name, v.Name, ext.Name,
			)
		}
		embeds = append(embeds, naming.GetGoName(parent))
	}
	inlineFields := append(
		slices.Clone(declared),
		declaredFields(inherited, pform.Fields, table)...,
	)
	return embeds, inlineFields, nil
}

// inlineVariantLines returns the encode and decode statements of the fields of an
// inline union variant, indented to sit inside the union codec's switch case.
func (b *encoderBuilder) inlineVariantLines(
	fields []resolution.Field,
) ([]string, []string, error) {
	fb := &encoderBuilder{
		table:       b.table,
		repoRoot:    b.repoRoot,
		packageName: b.packageName,
		parentPath:  b.parentPath,
		imports:     b.imports,
		aliases:     b.aliases,
		scope: pinScope{
			pins:   b.scope.pins,
			owners: slices.Clone(b.scope.owners),
		},
	}
	if err := fb.processFields(fields, "v", "v"); err != nil {
		return nil, nil, err
	}
	b.needsMath = b.needsMath || fb.needsMath
	b.needsJSON = b.needsJSON || fb.needsJSON
	return indentLines(fb.encodeLines), indentLines(fb.decodeLines), nil
}

func (b *encoderBuilder) processOptional(
	f resolution.Field, getPath, setPath string,
) error {
	if f.Type.Name == "nil" {
		return nil
	}
	resolved, ok := b.resolveTypeRef(f.Type)
	if !ok {
		return errors.Newf("cannot resolve type %q (field=%s)", f.Type.Name, f.Name)
	}

	actual := b.unwrapType(resolved)

	// Optional arrays/maps
	if bg, ok := actual.Form.(resolution.BuiltinGenericForm); ok &&
		(bg.Name == "Array" || bg.Name == "Map") {
		return b.processOptionalValue(
			f,
			resolved,
			"(*"+getPath+")",
			getPath,
			setPath,
			true,
		)
	}

	// Optional json/any
	if prim, ok := actual.Form.(resolution.PrimitiveForm); ok &&
		(prim.Name == "record" || prim.Name == "any") {
		return b.processOptionalValue(f, resolved, getPath, getPath, setPath, false)
	}

	// Optional other (pointer to struct/primitive)
	return b.processOptionalPointer(f, resolved, getPath, setPath)
}

// processOptionalPointer writes a presence bit for the optional pointer field f and,
// when the pointer is not nil, the value it points to.
func (b *encoderBuilder) processOptionalPointer(
	f resolution.Field,
	resolved resolution.Type,
	getPath, setPath string,
) error {
	ind := b.indent()
	goType, err := typemap.GoTypeRefName(f.Type, b.table, b.goTypeName)
	if err != nil {
		return err
	}
	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("if %s != nil {", getPath),
		ind+"\tw.Bool(true)",
	)
	b.decodeLines = append(b.decodeLines,
		ind+"{ present, err := r.Bool(); if err != nil { return err }",
		ind+"if present {",
		ind+"\tvar hv "+goType,
	)
	b.depth++
	b.inBlock++
	derefGet := "(*" + getPath + ")"
	if err := b.processValueByType(resolved, f.Type, derefGet, "hv"); err != nil {
		return err
	}
	b.inBlock--
	b.depth--
	b.encodeLines = append(
		b.encodeLines,
		ind+"} else {",
		ind+"\tw.Bool(false)",
		ind+"}",
	)
	b.decodeLines = append(b.decodeLines,
		ind+"\t"+setPath+" = &hv",
		ind+"}",
		ind+"}",
	)
	return nil
}

// processOptionalValue writes a presence bit for the nilable field f and, when the
// value is present, the value read from valuePath. skipNilCheck marks the value as
// known non-nil inside the guard.
func (b *encoderBuilder) processOptionalValue(
	f resolution.Field,
	resolved resolution.Type,
	valuePath, getPath, setPath string,
	skipNilCheck bool,
) error {
	ind := b.indent()
	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("if %s != nil {", getPath),
		ind+"\tw.Bool(true)",
	)
	b.decodeLines = append(b.decodeLines,
		ind+"{ present, err := r.Bool(); if err != nil { return err }",
		ind+"if present {",
	)
	b.depth++
	b.inBlock++
	if skipNilCheck {
		b.skipNilCheck = true
	}
	if err := b.processValueByType(resolved, f.Type, valuePath, setPath); err != nil {
		return err
	}
	if skipNilCheck {
		b.skipNilCheck = false
	}
	b.inBlock--
	b.depth--
	b.encodeLines = append(
		b.encodeLines,
		ind+"} else {",
		ind+"\tw.Bool(false)",
		ind+"}",
	)
	b.decodeLines = append(b.decodeLines, ind+"}", ind+"}")
	return nil
}

func (b *encoderBuilder) processOptionalNilable(
	f resolution.Field, getPath, setPath string,
) error {
	ind := b.indent()
	resolved, ok := b.resolveTypeRef(f.Type)
	if !ok {
		return errors.Newf("cannot resolve type %q (field=%s)", f.Type.Name, f.Name)
	}
	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("if %s != nil {", getPath),
		ind+"\tw.Bool(true)",
	)
	b.decodeLines = append(b.decodeLines,
		ind+"{ present, err := r.Bool(); if err != nil { return err }",
		ind+"if present {",
	)
	b.depth++
	b.inBlock++
	b.skipNilCheck = true
	if err := b.processValueByType(resolved, f.Type, getPath, setPath); err != nil {
		return err
	}
	b.skipNilCheck = false
	b.inBlock--
	b.depth--
	b.encodeLines = append(
		b.encodeLines,
		ind+"} else {",
		ind+"\tw.Bool(false)",
		ind+"}",
	)
	b.decodeLines = append(b.decodeLines, ind+"}", ind+"}")
	return nil
}

func (b *encoderBuilder) processArray(
	f resolution.Field, getPath, setPath string,
) error {
	ind := b.indent()
	elemRef := f.Type.TypeArgs[0]
	elemType, ok := elemRef.Resolve(b.table)
	if !ok && (!elemRef.IsTypeParam() || !b.hasTypeParams) {
		return errors.Newf("cannot resolve array element type %s", elemRef.Name)
	}
	goType, err := typemap.ResolveGoSliceElemType(elemRef, b.table, b.goTypeName)
	if err != nil {
		return err
	}

	// Write a presence bit to distinguish nil from empty slices.
	// When inside a optional guard, the slice is already known non-nil.
	known := b.openNilable(ind, getPath)

	idx := b.loopIndex()
	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("\tw.Uint32(uint32(len(%s)))", getPath),
		ind+fmt.Sprintf("\tfor %s := range %s {", idx, getPath),
	)
	b.decodeLines = append(b.decodeLines,
		ind+"\tn, err := r.CollectionLen(); if err != nil { return err }",
		ind+fmt.Sprintf("\t%s = make([]%s, n)", setPath, goType),
		ind+fmt.Sprintf("\tfor %s := range %s {", idx, setPath),
	)

	b.depth++
	b.inBlock++
	elemGetPath := getPath + "[" + idx + "]"
	elemSetPath := setPath + "[" + idx + "]"
	if err := b.processValueByType(
		elemType,
		elemRef,
		elemGetPath,
		elemSetPath,
	); err != nil {
		return err
	}
	b.inBlock--
	b.depth--

	b.encodeLines = append(b.encodeLines, ind+"\t}")
	b.decodeLines = append(b.decodeLines, ind+"\t}")
	b.closeNilable(ind, known)
	return nil
}

func (b *encoderBuilder) processMap(
	keyRef, valRef resolution.TypeRef,
	getPath, setPath string,
) error {
	ind := b.indent()
	keyType, ok := keyRef.Resolve(b.table)
	if !ok {
		return errors.Newf("cannot resolve map key type %s", keyRef.Name)
	}
	valType, ok := valRef.Resolve(b.table)
	if !ok {
		return errors.Newf("cannot resolve map value type %s", valRef.Name)
	}
	goKeyType, err := typemap.GoTypeRefName(keyRef, b.table, b.goTypeName)
	if err != nil {
		return err
	}
	goValType, err := typemap.GoTypeRefName(valRef, b.table, b.goTypeName)
	if err != nil {
		return err
	}

	// Write a presence bit to distinguish nil from empty maps. When inside a optional
	// guard, the map is already known non-nil.
	known := b.openNilable(ind, getPath)

	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("\tw.Uint32(uint32(len(%s)))", getPath),
		ind+fmt.Sprintf("\tfor key, val := range %s {", getPath),
	)
	b.decodeLines = append(b.decodeLines,
		ind+"\tn, err := r.CollectionLen(); if err != nil { return err }",
		ind+fmt.Sprintf("\t%s = make(map[%s]%s, n)", setPath, goKeyType, goValType),
		ind+"\tfor range n {",
		ind+"\t\tvar key "+goKeyType,
		ind+"\t\tvar val "+goValType,
	)

	b.depth++
	b.inBlock++
	if err := b.processValueByType(keyType, keyRef, "key", "key"); err != nil {
		return errors.Wrapf(err, "map key")
	}
	if err := b.processValueByType(valType, valRef, "val", "val"); err != nil {
		return errors.Wrapf(err, "map value")
	}
	b.inBlock--
	b.depth--

	b.encodeLines = append(b.encodeLines, ind+"\t}")
	b.decodeLines = append(b.decodeLines,
		ind+"\t\t"+setPath+"[key] = val",
		ind+"\t}",
	)
	b.closeNilable(ind, known)
	return nil
}

func (b *encoderBuilder) stringLeaf(ind, valPath, setPath, goTypeCast string) {
	if goTypeCast != "" {
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("w.String(string(%s))", valPath))
		b.decodeLine(
			ind + fmt.Sprintf(
				"{ rawV, err := r.String(); if err != nil { return err }; %s = %s(rawV) }",
				setPath,
				goTypeCast,
			),
		)
	} else {
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("w.String(%s)", valPath))
		b.decodeWithErr(
			ind + fmt.Sprintf(
				"if %s, err = r.String(); err != nil { return err }",
				setPath,
			),
		)
	}
}

func (b *encoderBuilder) bytesLeaf(ind, valPath, setPath string) {
	// Write a presence bit to distinguish nil from empty byte slices. Inside an
	// optional guard, the slice is already known non-nil.
	if b.takeNonNil() {
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("w.WriteWithLen(%s)", valPath),
		)
		b.decodeLines = append(
			b.decodeLines,
			ind+setPath+", err = r.ReadWithLen(); if err != nil { return err }",
		)
		return
	}
	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("w.Bool(%s != nil)", valPath),
		ind+fmt.Sprintf("if %s != nil {", valPath),
		ind+fmt.Sprintf("\tw.WriteWithLen(%s)", valPath),
		ind+"}",
	)
	b.decodeLines = append(
		b.decodeLines,
		ind+"{ present, err := r.Bool(); if err != nil { return err }",
		ind+"if present {",
		ind+fmt.Sprintf(
			"\t%s, err = r.ReadWithLen(); if err != nil { return err }",
			setPath,
		),
		ind+"} }",
	)
}

func (b *encoderBuilder) boolLeaf(ind, valPath, setPath, goTypeCast string) {
	if goTypeCast != "" {
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("w.Bool(bool(%s))", valPath))
		b.decodeLine(
			ind + fmt.Sprintf(
				"{ rawV, err := r.Bool(); if err != nil { return err }; "+
					"%s = %s(rawV) }",
				setPath,
				goTypeCast,
			),
		)
	} else {
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("w.Bool(%s)", valPath))
		b.decodeWithErr(
			ind + fmt.Sprintf(
				"if %s, err = r.Bool(); err != nil { return err }",
				setPath,
			),
		)
	}
}

// openNilable writes a presence bit that tells a nil value from an empty one, and opens
// the block that holds a present value. It returns true without writing when an
// optional guard already knows the value is non-nil.
func (b *encoderBuilder) openNilable(ind, getPath string) bool {
	known := b.takeNonNil()
	if !known {
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("w.Bool(%s != nil)", getPath),
			ind+fmt.Sprintf("if %s != nil {", getPath),
		)
		b.decodeLines = append(b.decodeLines,
			ind+"{ present, err := r.Bool(); if err != nil { return err }",
			ind+"if present {",
		)
	}
	return known
}

// closeNilable closes the block that openNilable opened.
func (b *encoderBuilder) closeNilable(ind string, known bool) {
	if !known {
		b.encodeLines = append(b.encodeLines, ind+"}")
		b.decodeLines = append(b.decodeLines, ind+"}", ind+"}")
	}
}

// uuidLeaf adds the statements that write and read a UUID as its 16 raw bytes.
func (b *encoderBuilder) uuidLeaf(ind, getPath, setPath string) {
	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("w.Write(%s[:])", getPath))
	b.decodeLine(
		ind + fmt.Sprintf(
			"if _, err := r.Read(%s[:]); err != nil { return err }",
			setPath,
		),
	)
}

// jsonLeaf adds the statements that write and read a value as length-prefixed JSON.
func (b *encoderBuilder) jsonLeaf(ind, valPath, setPath string) {
	b.needsJSON = true
	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("{ b, err := json.Marshal(%s)", valPath),
		ind+"\tif err != nil { return err }",
		ind+"\tw.WriteWithLen(b) }",
	)
	b.decodeLines = append(
		b.decodeLines,
		ind+"{ b, err := r.ReadWithLen(); if err != nil { return err }",
		ind+fmt.Sprintf(
			"\tif err = json.Unmarshal(b, &%s); err != nil { return err } }",
			setPath,
		),
	)
}

func (b *encoderBuilder) processLeaf(
	typ resolution.Type, getPath, setPath string,
) error {
	primName, goTypeCast, err := b.resolveLeaf(typ)
	if err != nil {
		return err
	}
	ind := b.indent()
	// Scalar leaves pass the value on directly, so a parenthesized optional deref reads
	// better without the redundant parens (*d.Foo, not (*d.Foo)). uuid keeps getPath
	// because it slice-indexes the value: (*d.Foo)[:] must stay parenthesized.
	valPath := valuePath(getPath)

	switch primName {
	case "string":
		b.stringLeaf(ind, valPath, setPath, goTypeCast)

	case "uuid":
		b.uuidLeaf(ind, getPath, setPath)

	case "record", "any":
		b.jsonLeaf(ind, valPath, setPath)

	case "bytes":
		b.bytesLeaf(ind, valPath, setPath)

	case "bool":
		b.boolLeaf(ind, valPath, setPath, goTypeCast)

	default:
		goPrim, ok := numericGoType(primName)
		if !ok {
			return errors.Newf("unsupported primitive type: %s", primName)
		}
		writerMethod := strings.ToUpper(goPrim[:1]) + goPrim[1:]
		b.addIntLeaf(
			ind,
			valPath,
			setPath,
			goTypeCast,
			writerMethod,
			goPrim,
			goPrim+"(%s)",
		)
	}
	return nil
}

// numericGoType returns the Go type that holds the numeric primitive primName, and
// false if primName is not numeric.
func numericGoType(primName string) (string, bool) {
	switch primName {
	case "int8", "int16", "int32", "int64", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return primName, true
	case "uint12":
		return "uint16", true
	case "uint20":
		return "uint32", true
	}
	return "", false
}

func (b *encoderBuilder) addIntLeaf(
	ind, getPath, setPath, goTypeCast, writerMethod, goPrimType, encodeCast string,
) {
	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("w.%s(%s)", writerMethod, fmt.Sprintf(encodeCast, getPath)))
	cast := goPrimType
	if goTypeCast != "" {
		cast = goTypeCast
	}
	if goTypeCast != "" {
		b.decodeLine(
			ind + fmt.Sprintf(
				"{ rawV, err := r.%s(); if err != nil { return err }; %s = %s(rawV) }",
				writerMethod,
				setPath,
				cast,
			),
		)
	} else {
		b.decodeWithErr(
			ind + fmt.Sprintf(
				"if %s, err = r.%s(); err != nil { return err }",
				setPath,
				writerMethod,
			),
		)
	}
}

// --- Helper methods for type resolution ---

func (b *encoderBuilder) resolveTypeRef(
	ref resolution.TypeRef,
) (resolution.Type, bool) {
	resolved, ok := ref.Resolve(b.table)
	if ok {
		return resolved, true
	}
	if ref.IsTypeParam() && ref.TypeParam != nil && ref.TypeParam.HasDefault() {
		return ref.TypeParam.Default.Resolve(b.table)
	}
	return resolution.Type{}, false
}

func (b *encoderBuilder) canResolve(ref resolution.TypeRef) bool {
	if ref.IsTypeParam() && b.hasTypeParams {
		return true
	}
	_, ok := b.resolveTypeRef(ref)
	return ok
}

func (b *encoderBuilder) unwrapType(typ resolution.Type) resolution.Type {
	return typemap.UnwrapType(typ, b.table)
}

// takeNonNil reports whether the value about to be encoded is known non-nil, and clears
// the flag so the value's elements write their own presence bits.
func (b *encoderBuilder) takeNonNil() bool {
	known := b.skipNilCheck
	b.skipNilCheck = false
	return known
}

func (b *encoderBuilder) resolveLeaf(
	typ resolution.Type,
) (primName, goTypeCast string, err error) {
	return typemap.ResolveLeafPrimitive(typ, b.table, b.goTypeName)
}

// recordGoType returns the Go type of a `record` nested in a generic container, and
// adds its import to imports. A `record` field is a msgpack.EncodedJSON (a typed
// map[string]any), and a nested one must match it, or the codec emits a map[string]any
// that the declared type cannot take. The import keeps any alias the file already gave
// it (flex methods register "xmsgpack"), so the file never imports it twice.
func recordGoType(imports map[string]string) string {
	const importPath = "github.com/synnaxlabs/x/encoding/msgpack"
	alias, registered := imports[importPath]
	if !registered {
		imports[importPath] = ""
	}
	if alias == "" {
		alias = "msgpack"
	}
	return alias + ".EncodedJSON"
}

func (b *encoderBuilder) goTypeName(typ resolution.Type) (string, error) {
	if prim, ok := typ.Form.(resolution.PrimitiveForm); ok {
		if prim.Name == "record" {
			return recordGoType(b.imports), nil
		}
		goType, ok := typemap.PrimitiveGoType(prim.Name)
		if !ok {
			return "", errors.Newf("unsupported primitive type: %s", prim.Name)
		}
		if typemap.IsUUID(prim.Name) {
			b.imports["uuid"] = ""
		}
		return goType, nil
	}
	goName := naming.GetGoName(typ)
	goPath := output.GetPath(typ, "go")
	if goPath == "" || goPath == b.parentPath {
		return goName, nil
	}
	if p, ok := b.scope.path(typ); ok {
		goPath = p
	}
	importPath := resolveGoImportPath(goPath, b.repoRoot)
	if existingAlias, ok := b.imports[importPath]; ok {
		qualifier := existingAlias
		if qualifier == "" {
			qualifier = filepath.Base(importPath)
		}
		return qualifier + "." + goName, nil
	}
	alias, forced := b.aliases[importPath]
	if !forced {
		alias = naming.DerivePackageAlias(goPath, b.packageName)
		if alias == b.packageName || b.aliasUsed(alias) {
			parent := filepath.Base(filepath.Dir(goPath))
			alias = parent + alias
		}
		for b.aliasUsed(alias) {
			alias += "x"
		}
	}
	// Only store an alias when it differs from the package's natural name.
	actualPkg := filepath.Base(importPath)
	if alias == actualPkg {
		b.imports[importPath] = ""
	} else {
		b.imports[importPath] = alias
	}
	return alias + "." + goName, nil
}

func (b *encoderBuilder) aliasUsed(alias string) bool {
	for _, a := range b.imports {
		if a == alias {
			return true
		}
	}
	return false
}

// --- Template ---

const encoderCodecTemplate = `// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
{{- if .NeedsJSON}}
	"encoding/json/v2"
{{- end}}
{{- if .NeedsJSONText}}
	"encoding/json/jsontext"
{{- end}}
{{- if or .ConcreteCodecs .GenericCodecs}}

	"github.com/synnaxlabs/x/encoding/orc"
{{- end}}
{{- range sortedImports .ExtraImports}}
	{{if .Alias}}{{.Alias}} {{end}}"{{.Path}}"
{{- end}}
)
{{range .ConcreteCodecs}}
// EncodeOrc writes the value to w in the Orc binary format.
func ({{.Receiver}} {{.GoName}}) EncodeOrc(w *orc.Writer) error {
{{.EncodeBody}}
	return nil
}

// DecodeOrc reads the value from r in the Orc binary format.
func ({{.Receiver}} *{{.GoName}}) DecodeOrc(r *orc.Reader) error {
{{- if .Recursive}}
	if err := r.PushDepth(orc.MaxDecodeDepth); err != nil {
		return err
	}
	defer r.PopDepth()
{{- end}}
{{- if .UsesErr}}
	var err error
{{- end}}
{{.DecodeBody}}
	return nil
}
{{end}}{{range .GenericCodecs}}
// EncodeOrc writes the value to w in the Orc binary format.
func ({{.Receiver}} {{.GoName}}[{{tpNames .TypeParams}}]) EncodeOrc(w *orc.Writer) error {
{{.EncodeBody}}
	return nil
}

// DecodeOrc reads the value from r in the Orc binary format.
func ({{.Receiver}} *{{.GoName}}[{{tpNames .TypeParams}}]) DecodeOrc(r *orc.Reader) error {
{{- if .Recursive}}
	if err := r.PushDepth(orc.MaxDecodeDepth); err != nil {
		return err
	}
	defer r.PopDepth()
{{- end}}
{{- if .UsesErr}}
	var err error
{{- end}}
{{.DecodeBody}}
	return nil
}
{{end}}{{if .HasFlex}}{{.FlexMethods}}{{end}}`
