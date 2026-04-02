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
	"path/filepath"
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/typemap"
	"github.com/synnaxlabs/oracle/plugin/output"
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
}

type encoderFileOutput struct {
	Package        string
	ExtraImports   map[string]string
	NeedsMath      bool
	NeedsJSON      bool
	ConcreteCodecs []concreteCodec
	GenericCodecs  []genericCodec
}

// --- Generation entry point ---

func generateEncoderCodecFile(
	packageName string,
	parentPath string,
	entries []CodecEntry,
	table *resolution.Table,
	repoRoot string,
) ([]byte, error) {
	fo := encoderFileOutput{
		Package:      packageName,
		ExtraImports: make(map[string]string),
	}
	for _, e := range entries {
		b := &encoderBuilder{
			table:       table,
			repoRoot:    repoRoot,
			packageName: packageName,
			parentPath:  parentPath,
			imports:     fo.ExtraImports,
		}
		form, ok := e.Type.Form.(resolution.StructForm)
		if !ok {
			continue
		}

		// Collect type params that produce Go generics. A type param becomes a Go
		// generic parameter only when it has NO default value (like Details? in
		// Status<Details?>). Params with any default (like V = Variant or Data? =
		// record) are substituted by the types plugin and the Go type is concrete.
		var typeParams []typeParamData
		if form.IsGeneric() {
			for _, tp := range form.TypeParams {
				if tp.HasDefault() {
					continue
				}
				typeParams = append(typeParams, typeParamData{
					Name:       tp.Name,
					Constraint: typeParamConstraint(tp),
				})
			}
		}

		recv := receiverName(e.GoName)
		if len(typeParams) > 0 {
			// Generic type with non-defaulted params: generate method with
			// type assertion + JSON fallback for type parameter fields.
			b.hasTypeParams = true
			fields := resolution.UnifiedFields(e.Type, b.table)
			if err := b.processFields(fields, recv, recv); err != nil {
				return nil, errors.Wrapf(err, "failed to generate generic codec for %s", e.GoName)
			}
			if b.needsMath {
				fo.NeedsMath = true
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
			})
		} else {
			fields := resolution.UnifiedFields(e.Type, b.table)
			if err := b.processFields(fields, recv, recv); err != nil {
				return nil, errors.Wrapf(err, "failed to generate codec for %s", e.GoName)
			}
			if b.needsMath {
				fo.NeedsMath = true
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
			})
		}
	}
	tmpl, err := template.New("encoder_codec").Funcs(template.FuncMap{
		"tpNames": tpNames,
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

func typeParamConstraint(tp resolution.TypeParam) string {
	if tp.Constraint != nil && resolution.IsConstraint(tp.Constraint.Name) {
		return tp.Constraint.Name
	}
	return "any"
}

// reservedNames contains single-letter variable names used in generated method
// bodies (parameters, loop vars, temporaries) that would conflict with a receiver.
var reservedNames = map[string]bool{
	"w": true, "r": true, // method parameters
	"n": true, "b": true, "v": true, "m": true, // temporaries
	"i": true, "j": true, "k": true, "l": true, // loop indices
}

// receiverName derives a Go-idiomatic short receiver name from a type name.
// It takes the lowercase initials of each word in the PascalCase name
// (e.g., "TimeRange" -> "tr", "XY" -> "xy", "Status" -> "s").
// Names that conflict with generated local variables get a "v" suffix.
func receiverName(goName string) string {
	var initials []byte
	for i, c := range goName {
		if i == 0 || (c >= 'A' && c <= 'Z') {
			initials = append(initials, byte(c|0x20)) // lowercase
		}
	}
	name := string(initials)
	if reservedNames[name] {
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
	table         *resolution.Table
	repoRoot      string
	packageName   string
	parentPath    string
	imports       map[string]string
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
		if domain.GetStringFromField(f, "go", "marshal") == "skip" {
			continue
		}
		goName := naming.GetFieldName(f)
		getPath := getPrefix + "." + goName
		setPath := setPrefix + "." + goName

		// Type parameter fields use a type assertion with JSON fallback,
		// but only when the type param has no default. Defaulted type params
		// (e.g. V extends Variant = Variant) are substituted with their
		// default and encoded concretely.
		if f.Type.IsTypeParam() && b.hasTypeParams {
			if f.Type.TypeParam.HasDefault() {
				f.Type = *f.Type.TypeParam.Default
			} else {
				if err := b.processTypeParamField(getPath, setPath); err != nil {
					return err
				}
				continue
			}
		}

		if f.IsHardOptional {
			if err := b.processHardOptional(f, getPath, setPath); err != nil {
				return err
			}
		} else if f.IsOptional && b.isGoNilable(f.Type) {
			if err := b.processSoftOptionalNilable(f, getPath, setPath); err != nil {
				return err
			}
		} else {
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
		return b.processTypeParamField(getPath, setPath)
	}

	actual, effectiveTypeArgs := typemap.UnwrapTypeRef(resolved, ref, b.table)

	switch form := actual.Form.(type) {
	case resolution.StructForm:
		return b.processStruct(actual, form, effectiveTypeArgs, getPath, setPath)
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

func (b *encoderBuilder) processTypeParamField(getPath, setPath string) error {
	b.needsJSON = true
	ind := b.indent()
	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("if m, ok := any(%s).(orc.SelfEncoder); ok {", getPath),
		ind+"\tif err := m.EncodeOrc(w); err != nil { return err }",
		ind+"} else {",
		ind+fmt.Sprintf("\tb, err := json.Marshal(%s)", getPath),
		ind+"\tif err != nil { return err }",
		ind+"\tw.Uint32(uint32(len(b)))",
		ind+"\tw.Write(b)",
		ind+"}",
	)
	b.decodeLines = append(b.decodeLines,
		ind+fmt.Sprintf("if m, ok := any(&%s).(orc.SelfDecoder); ok {", setPath),
		ind+"\tif err := m.DecodeOrc(r); err != nil { return err }",
		ind+"} else {",
		ind+"\tn, err := r.CollectionLen(); if err != nil { return err }",
		ind+"\tb := make([]byte, n)",
		ind+"\tif _, err = r.Read(b); err != nil { return err }",
		ind+fmt.Sprintf("\tif err = json.Unmarshal(b, &%s); err != nil { return err }", setPath),
		ind+"}",
	)
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
		return b.processFields(substituted, getPath, setPath)
	}

	// Method dispatch: call EncodeOrc/DecodeOrc on the field directly.
	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("if err := %s.EncodeOrc(w); err != nil { return err }", getPath))
	b.decodeWithErr(
		ind + fmt.Sprintf("if err = %s.DecodeOrc(r); err != nil { return err }", setPath))
	return nil
}

func (b *encoderBuilder) processHardOptional(
	f resolution.Field, getPath, setPath string,
) error {
	if f.Type.Name == "nil" {
		return nil
	}
	ind := b.indent()
	resolved, ok := b.resolveTypeRef(f.Type)
	if !ok {
		return errors.Newf("cannot resolve type %q (field=%s)", f.Type.Name, f.Name)
	}

	actual := b.unwrapType(resolved)

	// Hard optional arrays/maps
	if bg, ok := actual.Form.(resolution.BuiltinGenericForm); ok && (bg.Name == "Array" || bg.Name == "Map") {
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
		derefGet := "(*" + getPath + ")"
		if err := b.processValueByType(resolved, f.Type, derefGet, setPath); err != nil {
			return err
		}
		b.skipNilCheck = false
		b.inBlock--
		b.depth--
		b.encodeLines = append(b.encodeLines, ind+"} else {", ind+"\tw.Bool(false)", ind+"}")
		b.decodeLines = append(b.decodeLines, ind+"}", ind+"}")
		return nil
	}

	// Hard optional json/any
	if prim, ok := actual.Form.(resolution.PrimitiveForm); ok && (prim.Name == "record" || prim.Name == "any") {
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
		if err := b.processValueByType(resolved, f.Type, getPath, setPath); err != nil {
			return err
		}
		b.inBlock--
		b.depth--
		b.encodeLines = append(b.encodeLines, ind+"} else {", ind+"\tw.Bool(false)", ind+"}")
		b.decodeLines = append(b.decodeLines, ind+"}", ind+"}")
		return nil
	}

	// Hard optional other (pointer to struct/primitive)
	goType, err := b.goTypeName(resolved)
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
		ind+fmt.Sprintf("\tvar v %s", goType),
	)
	b.depth++
	b.inBlock++
	derefGet := "(*" + getPath + ")"
	if err := b.processValueByType(resolved, f.Type, derefGet, "v"); err != nil {
		return err
	}
	b.inBlock--
	b.depth--
	b.encodeLines = append(b.encodeLines, ind+"} else {", ind+"\tw.Bool(false)", ind+"}")
	b.decodeLines = append(b.decodeLines,
		ind+"\t"+setPath+" = &v",
		ind+"}",
		ind+"}",
	)
	return nil
}

func (b *encoderBuilder) processSoftOptionalNilable(
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
	b.encodeLines = append(b.encodeLines, ind+"} else {", ind+"\tw.Bool(false)", ind+"}")
	b.decodeLines = append(b.decodeLines, ind+"}", ind+"}")
	return nil
}

func (b *encoderBuilder) processArray(
	f resolution.Field, getPath, setPath string,
) error {
	ind := b.indent()
	elemRef := f.Type.TypeArgs[0]
	elemType, ok := elemRef.Resolve(b.table)
	if !ok {
		return errors.Newf("cannot resolve array element type %s", elemRef.Name)
	}
	goType, err := b.resolveGoSliceElemType(elemType)
	if err != nil {
		return err
	}

	// Write a presence bit to distinguish nil from empty slices.
	// When inside a hard-optional guard, the slice is already known non-nil.
	if !b.skipNilCheck {
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("w.Bool(%s != nil)", getPath),
			ind+fmt.Sprintf("if %s != nil {", getPath),
		)
		b.decodeLines = append(b.decodeLines,
			ind+"{ present, err := r.Bool(); if err != nil { return err }",
			ind+"if present {",
		)
	}

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
	if err := b.processValueByType(elemType, elemRef, elemGetPath, elemSetPath); err != nil {
		return err
	}
	b.inBlock--
	b.depth--

	b.encodeLines = append(b.encodeLines, ind+"\t}")
	b.decodeLines = append(b.decodeLines, ind+"\t}")
	if !b.skipNilCheck {
		b.encodeLines = append(b.encodeLines, ind+"}")
		b.decodeLines = append(b.decodeLines, ind+"}", ind+"}")
	}
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
	goKeyType, err := b.goTypeName(keyType)
	if err != nil {
		return err
	}
	goValType, err := b.goTypeName(valType)
	if err != nil {
		return err
	}

	// Write a presence bit to distinguish nil from empty maps.
	// When inside a hard-optional guard, the map is already known non-nil.
	if !b.skipNilCheck {
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("w.Bool(%s != nil)", getPath),
			ind+fmt.Sprintf("if %s != nil {", getPath),
		)
		b.decodeLines = append(b.decodeLines,
			ind+"{ present, err := r.Bool(); if err != nil { return err }",
			ind+"if present {",
		)
	}

	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("\tw.Uint32(uint32(len(%s)))", getPath),
		ind+fmt.Sprintf("\tfor key, val := range %s {", getPath),
	)
	b.decodeLines = append(b.decodeLines,
		ind+"\tn, err := r.CollectionLen(); if err != nil { return err }",
		ind+fmt.Sprintf("\t%s = make(map[%s]%s, n)", setPath, goKeyType, goValType),
		ind+"\tfor range n {",
		ind+fmt.Sprintf("\t\tvar key %s", goKeyType),
		ind+fmt.Sprintf("\t\tvar val %s", goValType),
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
	if !b.skipNilCheck {
		b.encodeLines = append(b.encodeLines, ind+"}")
		b.decodeLines = append(b.decodeLines, ind+"}", ind+"}")
	}
	return nil
}

func (b *encoderBuilder) processLeaf(
	typ resolution.Type, getPath, setPath string,
) error {
	primName, goTypeCast, err := b.resolveLeaf(typ)
	if err != nil {
		return err
	}
	ind := b.indent()

	switch primName {
	case "string":
		if goTypeCast != "" {
			b.encodeLines = append(b.encodeLines,
				ind+fmt.Sprintf("w.String(string(%s))", getPath))
			b.decodeLine(
				ind + fmt.Sprintf("{ v, err := r.String(); if err != nil { return err }; %s = %s(v) }", setPath, goTypeCast))
		} else {
			b.encodeLines = append(b.encodeLines,
				ind+fmt.Sprintf("w.String(%s)", getPath))
			b.decodeWithErr(
				ind + fmt.Sprintf("if %s, err = r.String(); err != nil { return err }", setPath))
		}

	case "uuid":
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("w.Write(%s[:])", getPath))
		b.decodeLine(
			ind + fmt.Sprintf("if _, err := r.Read(%s[:]); err != nil { return err }", setPath))

	case "record", "any":
		b.needsJSON = true
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("{ b, err := json.Marshal(%s)", getPath),
			ind+"\tif err != nil { return err }",
			ind+"\tw.Uint32(uint32(len(b)))",
			ind+"\tw.Write(b) }",
		)
		b.decodeLines = append(b.decodeLines,
			ind+"{ n, err := r.CollectionLen(); if err != nil { return err }",
			ind+"\tb := make([]byte, n)",
			ind+"\tif _, err = r.Read(b); err != nil { return err }",
			ind+fmt.Sprintf("\tif err = json.Unmarshal(b, &%s); err != nil { return err } }", setPath),
		)

	case "bytes":
		// Write a presence bit to distinguish nil from empty byte slices.
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("w.Bool(%s != nil)", getPath),
			ind+fmt.Sprintf("if %s != nil {", getPath),
			ind+fmt.Sprintf("\tw.Uint32(uint32(len(%s)))", getPath),
			ind+fmt.Sprintf("\tw.Write(%s)", getPath),
			ind+"}",
		)
		b.decodeLines = append(b.decodeLines,
			ind+"{ present, err := r.Bool(); if err != nil { return err }",
			ind+"if present {",
			ind+"\tn, err := r.CollectionLen(); if err != nil { return err }",
			ind+fmt.Sprintf("\t%s = make([]byte, n)", setPath),
			ind+fmt.Sprintf("\tif _, err = r.Read(%s); err != nil { return err }", setPath),
			ind+"} }",
		)

	case "bool":
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("w.Bool(%s)", getPath))
		b.decodeWithErr(
			ind + fmt.Sprintf("if %s, err = r.Bool(); err != nil { return err }", setPath))

	case "int8":
		b.addIntLeaf(ind, getPath, setPath, goTypeCast, "Int8", "int8", "int8(%s)")
	case "int16":
		b.addIntLeaf(ind, getPath, setPath, goTypeCast, "Int16", "int16", "int16(%s)")
	case "int32":
		b.addIntLeaf(ind, getPath, setPath, goTypeCast, "Int32", "int32", "int32(%s)")
	case "int64":
		b.addIntLeaf(ind, getPath, setPath, goTypeCast, "Int64", "int64", "int64(%s)")
	case "uint8":
		b.addIntLeaf(ind, getPath, setPath, goTypeCast, "Uint8", "uint8", "uint8(%s)")
	case "uint12", "uint16":
		b.addIntLeaf(ind, getPath, setPath, goTypeCast, "Uint16", "uint16", "uint16(%s)")
	case "uint20", "uint32":
		b.addIntLeaf(ind, getPath, setPath, goTypeCast, "Uint32", "uint32", "uint32(%s)")
	case "uint64":
		b.addIntLeaf(ind, getPath, setPath, goTypeCast, "Uint64", "uint64", "uint64(%s)")
	case "float32":
		b.addIntLeaf(ind, getPath, setPath, goTypeCast, "Float32", "float32", "float32(%s)")
	case "float64":
		b.addIntLeaf(ind, getPath, setPath, goTypeCast, "Float64", "float64", "float64(%s)")

	default:
		return errors.Newf("unsupported primitive type: %s", primName)
	}
	return nil
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
			ind + fmt.Sprintf("{ v, err := r.%s(); if err != nil { return err }; %s = %s(v) }", writerMethod, setPath, cast))
	} else {
		b.decodeWithErr(
			ind + fmt.Sprintf("if %s, err = r.%s(); err != nil { return err }", setPath, writerMethod))
	}
}

// --- Type dependency walk ---

func collectSerializableTypes(
	entryType resolution.Type,
	table *resolution.Table,
) (byPackage map[string][]resolution.Type, reachable set.Set[string]) {
	result := make(map[string][]resolution.Type)
	visited := make(set.Set[string])
	walkSerializableTypes(entryType, table, result, visited)
	return result, visited
}

func walkSerializableTypes(
	typ resolution.Type,
	table *resolution.Table,
	result map[string][]resolution.Type,
	visited set.Set[string],
) {
	if visited.Contains(typ.QualifiedName) {
		return
	}
	visited.Add(typ.QualifiedName)
	goPath := output.GetPath(typ, "go")
	if goPath != "" {
		if _, ok := typ.Form.(resolution.StructForm); ok {
			result[goPath] = append(result[goPath], typ)
		}
	}
	if sf, ok := typ.Form.(resolution.StructForm); ok {
		for _, ext := range sf.Extends {
			walkSerializableRef(ext, table, result, visited)
		}
	}
	fields := resolution.UnifiedFields(typ, table)
	for _, f := range fields {
		walkSerializableRef(f.Type, table, result, visited)
	}
}

func walkSerializableRef(
	ref resolution.TypeRef,
	table *resolution.Table,
	result map[string][]resolution.Type,
	visited set.Set[string],
) {
	resolved, ok := ref.Resolve(table)
	if !ok {
		return
	}
	for _, arg := range ref.TypeArgs {
		walkSerializableRef(arg, table, result, visited)
	}
	switch form := resolved.Form.(type) {
	case resolution.StructForm:
		walkSerializableTypes(resolved, table, result, visited)
	case resolution.AliasForm:
		walkSerializableRef(form.Target, table, result, visited)
	case resolution.DistinctForm:
		walkSerializableRef(form.Base, table, result, visited)
	}
}

// --- Helper methods for type resolution ---

func (b *encoderBuilder) resolveTypeRef(ref resolution.TypeRef) (resolution.Type, bool) {
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

func (b *encoderBuilder) isGoNilable(ref resolution.TypeRef) bool {
	resolved, ok := ref.Resolve(b.table)
	if !ok {
		return false
	}
	actual := b.unwrapType(resolved)
	if bg, ok := actual.Form.(resolution.BuiltinGenericForm); ok {
		return bg.Name == "Array" || bg.Name == "Map"
	}
	return false
}

func (b *encoderBuilder) resolveLeaf(typ resolution.Type) (primName, goTypeCast string, err error) {
	return typemap.ResolveLeafPrimitive(typ, b.table, b.goTypeName)
}

func (b *encoderBuilder) goTypeName(typ resolution.Type) (string, error) {
	if prim, ok := typ.Form.(resolution.PrimitiveForm); ok {
		goType, ok := typemap.PrimitiveGoType(prim.Name)
		if !ok {
			return "", errors.Newf("unsupported primitive type: %s", prim.Name)
		}
		if typemap.IsUUID(prim.Name) {
			b.imports["github.com/google/uuid"] = "uuid"
		}
		return goType, nil
	}
	goName := naming.GetGoName(typ)
	goPath := output.GetPath(typ, "go")
	if goPath == "" || goPath == b.parentPath {
		return goName, nil
	}
	importPath, err := resolveGoImportPath(goPath, b.repoRoot)
	if err != nil {
		return "", err
	}
	if existingAlias, ok := b.imports[importPath]; ok {
		qualifier := existingAlias
		if qualifier == "" {
			qualifier = filepath.Base(importPath)
		}
		return qualifier + "." + goName, nil
	}
	alias := naming.DerivePackageAlias(goPath, b.packageName)
	if alias == b.packageName || b.aliasUsed(alias) {
		parent := filepath.Base(filepath.Dir(goPath))
		alias = parent + alias
	}
	for b.aliasUsed(alias) {
		alias = alias + "x"
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

func (b *encoderBuilder) resolveGoSliceElemType(typ resolution.Type) (string, error) {
	return typemap.ResolveGoSliceElemType(typ, b.table, b.goTypeName)
}

// --- Template ---

const encoderCodecTemplate = `// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
{{- if .NeedsJSON}}
	"encoding/json"
{{- end}}

	"github.com/synnaxlabs/x/encoding/orc"
{{- range $path, $alias := .ExtraImports}}
	{{if $alias}}{{$alias}} {{end}}"{{$path}}"
{{- end}}
)
{{range .ConcreteCodecs}}
func ({{.Receiver}} {{.GoName}}) EncodeOrc(w *orc.Writer) error {
{{.EncodeBody}}
	return nil
}

func ({{.Receiver}} *{{.GoName}}) DecodeOrc(r *orc.Reader) error {
{{- if .UsesErr}}
	var err error
{{- end}}
{{.DecodeBody}}
	return nil
}
{{end}}{{range .GenericCodecs}}
func ({{.Receiver}} {{.GoName}}[{{tpNames .TypeParams}}]) EncodeOrc(w *orc.Writer) error {
{{.EncodeBody}}
	return nil
}

func ({{.Receiver}} *{{.GoName}}[{{tpNames .TypeParams}}]) DecodeOrc(r *orc.Reader) error {
{{- if .UsesErr}}
	var err error
{{- end}}
{{.DecodeBody}}
	return nil
}
{{end}}`
