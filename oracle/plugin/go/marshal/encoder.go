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
	TypeParams []typeParamData
	EncodeBody string
	DecodeBody string
	UsesErr    bool
}

type adapterInfo struct {
	GoName string
}

type encoderFileOutput struct {
	Package        string
	ExtraImports   map[string]string
	NeedsMath      bool
	NeedsJSON      bool
	ConcreteCodecs []concreteCodec
	GenericCodecs  []genericCodec
	Adapters       []adapterInfo
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

		if len(typeParams) > 0 {
			// Generic type with non-defaulted params: generate generic function.
			b.typeParamConverters = make(map[string]string)
			for _, tp := range typeParams {
				b.typeParamConverters[tp.Name] = "encode" + naming.ToPascalCase(tp.Name)
			}
			fields := resolution.UnifiedFields(e.Type, b.table)
			if err := b.processFields(fields, "s", "s"); err != nil {
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
				TypeParams: typeParams,
				EncodeBody: strings.Join(b.encodeLines, "\n"),
				DecodeBody: strings.Join(b.decodeLines, "\n"),
				UsesErr:    b.usesErr,
			})
		} else {
			fields := resolution.UnifiedFields(e.Type, b.table)
			if err := b.processFields(fields, "s", "s"); err != nil {
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
				EncodeBody: strings.Join(b.encodeLines, "\n"),
				DecodeBody: strings.Join(b.decodeLines, "\n"),
				UsesErr:    b.usesErr,
			})
		}
		if e.Adapter {
			fo.Adapters = append(fo.Adapters, adapterInfo{GoName: e.GoName})
		}
	}
	tmpl, err := template.New("encoder_codec").Funcs(template.FuncMap{
		"lowerFirst": naming.LowerFirst,
		"tpList":     tpList,
		"tpNames":    tpNames,
		"encodeArgs": encodeArgs,
		"decodeArgs": decodeArgs,
		"fwdEncArgs": fwdEncArgs,
		"fwdDecArgs": fwdDecArgs,
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

// Template helpers for generic type parameter lists.
func tpList(tps []typeParamData) string {
	parts := make([]string, len(tps))
	for i, tp := range tps {
		parts[i] = tp.Name + " " + tp.Constraint
	}
	return strings.Join(parts, ", ")
}

func tpNames(tps []typeParamData) string {
	parts := make([]string, len(tps))
	for i, tp := range tps {
		parts[i] = tp.Name
	}
	return strings.Join(parts, ", ")
}

func encodeArgs(tps []typeParamData) string {
	parts := make([]string, len(tps))
	for i, tp := range tps {
		parts[i] = "encode" + naming.ToPascalCase(tp.Name) + " func(*orc.Writer, *" + tp.Name + ") error"
	}
	return strings.Join(parts, ", ")
}

func decodeArgs(tps []typeParamData) string {
	parts := make([]string, len(tps))
	for i, tp := range tps {
		parts[i] = "decode" + naming.ToPascalCase(tp.Name) + " func(*orc.Reader, *" + tp.Name + ") error"
	}
	return strings.Join(parts, ", ")
}

func fwdEncArgs(tps []typeParamData) string {
	parts := make([]string, len(tps))
	for i, tp := range tps {
		parts[i] = "encode" + naming.ToPascalCase(tp.Name)
	}
	return strings.Join(parts, ", ")
}

func fwdDecArgs(tps []typeParamData) string {
	parts := make([]string, len(tps))
	for i, tp := range tps {
		parts[i] = "decode" + naming.ToPascalCase(tp.Name)
	}
	return strings.Join(parts, ", ")
}

// --- Encoder builder ---

type encoderBuilder struct {
	table               *resolution.Table
	repoRoot            string
	packageName         string
	parentPath          string
	imports             map[string]string
	encodeLines         []string
	decodeLines         []string
	needsMath           bool
	needsJSON           bool
	usesErr             bool
	depth               int
	inBlock             int
	typeParamConverters map[string]string // typeParamName -> converter func name
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
		goName := naming.GetFieldName(f)
		getPath := getPrefix + "." + goName
		setPath := setPrefix + "." + goName

		// Type parameter fields are always processed via their converter,
		// regardless of optional flags.
		if f.Type.IsTypeParam() && b.typeParamConverters != nil {
			if converter, ok := b.typeParamConverters[f.Type.TypeParam.Name]; ok {
				if err := b.processTypeParamField(converter, getPath, setPath); err != nil {
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
	// Check if this is a type parameter field - delegate to converter function.
	if ref.IsTypeParam() && b.typeParamConverters != nil {
		if converter, ok := b.typeParamConverters[ref.TypeParam.Name]; ok {
			return b.processTypeParamField(converter, getPath, setPath)
		}
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

func (b *encoderBuilder) processTypeParamField(converter, getPath, setPath string) error {
	ind := b.indent()
	encConverter := converter
	decConverter := strings.Replace(converter, "encode", "decode", 1)
	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("if err := %s(w, &%s); err != nil { return err }", encConverter, getPath))
	b.decodeWithErr(
		ind + fmt.Sprintf("if err = %s(r, &%s); err != nil { return err }", decConverter, setPath))
	return nil
}

func (b *encoderBuilder) processStruct(
	actual resolution.Type,
	form resolution.StructForm,
	effectiveTypeArgs []resolution.TypeRef,
	getPath, setPath string,
) error {
	ind := b.indent()
	structPath := output.GetPath(actual, "go")
	localName := naming.GetGoName(actual)

	// Determine function prefix (same-package vs cross-package).
	var fnPrefix string
	if structPath != "" && structPath != b.parentPath {
		goType, err := b.goTypeName(actual)
		if err != nil {
			return err
		}
		parts := strings.SplitN(goType, ".", 2)
		if len(parts) == 2 {
			fnPrefix = parts[0] + "."
			localName = parts[1]
		}
	}

	if !form.IsGeneric() {
		// Concrete struct - simple delegation.
		encodeFn := fnPrefix + "Encode" + localName
		decodeFn := fnPrefix + "Decode" + localName
		b.encodeLines = append(b.encodeLines,
			ind+fmt.Sprintf("if err := %s(w, &%s); err != nil { return err }", encodeFn, getPath))
		b.decodeWithErr(
			ind + fmt.Sprintf("if err = %s(r, &%s); err != nil { return err }", decodeFn, setPath))
		return nil
	}

	// Generic struct - delegate with type args and converter functions.
	if len(effectiveTypeArgs) == 0 {
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

	// Build type arg list and converter expressions for non-defaulted params.
	var goTypeArgs []string
	var encConverters, decConverters []string
	for i, typeArg := range effectiveTypeArgs {
		if i >= len(form.TypeParams) {
			break
		}
		tp := form.TypeParams[i]
		if tp.HasDefault() {
			continue
		}

		// Resolve the concrete type argument.
		if typeArg.IsTypeParam() {
			// Forwarding a type param from the enclosing generic function.
			goTypeArgs = append(goTypeArgs, typeArg.Name)
			encConverters = append(encConverters, "encode"+naming.ToPascalCase(typeArg.Name))
			decConverters = append(decConverters, "decode"+naming.ToPascalCase(typeArg.Name))
			continue
		}

		argResolved, ok := typeArg.Resolve(b.table)
		if !ok {
			if typeArg.Name == "nil" {
				continue
			}
			return errors.Newf("cannot resolve type argument %q for generic struct %s", typeArg.Name, actual.Name)
		}
		argGoType, err := b.goTypeName(argResolved)
		if err != nil {
			return err
		}
		goTypeArgs = append(goTypeArgs, argGoType)

		// Build encoder name for this concrete type argument.
		argLocalName := naming.GetGoName(argResolved)
		argPath := output.GetPath(argResolved, "go")
		var argFnPrefix string
		if argPath != "" && argPath != b.parentPath {
			parts := strings.SplitN(argGoType, ".", 2)
			if len(parts) == 2 {
				argFnPrefix = parts[0] + "."
				argLocalName = parts[1]
			}
		}
		encConverters = append(encConverters, argFnPrefix+"Encode"+argLocalName)
		decConverters = append(decConverters, argFnPrefix+"Decode"+argLocalName)
	}

	typeArgStr := ""
	if len(goTypeArgs) > 0 {
		typeArgStr = "[" + strings.Join(goTypeArgs, ", ") + "]"
	}
	encConvStr := ""
	decConvStr := ""
	if len(encConverters) > 0 {
		encConvStr = ", " + strings.Join(encConverters, ", ")
		decConvStr = ", " + strings.Join(decConverters, ", ")
	}

	encodeFn := fnPrefix + "Encode" + localName + typeArgStr
	decodeFn := fnPrefix + "Decode" + localName + typeArgStr

	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("if err := %s(w, &%s%s); err != nil { return err }", encodeFn, getPath, encConvStr))
	b.decodeWithErr(
		ind + fmt.Sprintf("if err = %s(r, &%s%s); err != nil { return err }", decodeFn, setPath, decConvStr))
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
		derefGet := "(*" + getPath + ")"
		if err := b.processValueByType(resolved, f.Type, derefGet, setPath); err != nil {
			return err
		}
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
	if err := b.processValueByType(resolved, f.Type, getPath, setPath); err != nil {
		return err
	}
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
	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("w.Bool(%s != nil)", getPath),
		ind+fmt.Sprintf("if %s != nil {", getPath),
	)
	b.decodeLines = append(b.decodeLines,
		ind+"{ present, err := r.Bool(); if err != nil { return err }",
		ind+"if present {",
	)

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

	b.encodeLines = append(b.encodeLines, ind+"\t}", ind+"}")
	b.decodeLines = append(b.decodeLines, ind+"\t}", ind+"}", ind+"}")
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
	b.encodeLines = append(b.encodeLines,
		ind+fmt.Sprintf("w.Bool(%s != nil)", getPath),
		ind+fmt.Sprintf("if %s != nil {", getPath),
	)
	b.decodeLines = append(b.decodeLines,
		ind+"{ present, err := r.Bool(); if err != nil { return err }",
		ind+"if present {",
	)

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

	b.encodeLines = append(b.encodeLines, ind+"\t}", ind+"}")
	b.decodeLines = append(b.decodeLines,
		ind+"\t\t"+setPath+"[key] = val",
		ind+"\t}",
		ind+"}", ind+"}",
	)
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
			ind + fmt.Sprintf("if _, err = r.Read(%s[:]); err != nil { return err }", setPath))

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
	if ref.IsTypeParam() && b.typeParamConverters != nil {
		if _, ok := b.typeParamConverters[ref.TypeParam.Name]; ok {
			return true
		}
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
{{- if .Adapters}}
	"context"
	"io"
	"sync"
{{- end}}
{{- if .NeedsJSON}}
	"encoding/json"
{{- end}}

{{- if .Adapters}}
	xencoding "github.com/synnaxlabs/x/encoding"
{{- end}}
	"github.com/synnaxlabs/x/encoding/orc"
{{- range $path, $alias := .ExtraImports}}
	{{if $alias}}{{$alias}} {{end}}"{{$path}}"
{{- end}}
)
{{range .ConcreteCodecs}}
func Encode{{.GoName}}(w *orc.Writer, s *{{.GoName}}) error {
{{.EncodeBody}}
	return nil
}

func Decode{{.GoName}}(r *orc.Reader, s *{{.GoName}}) error {
{{- if .UsesErr}}
	var err error
{{- end}}
{{.DecodeBody}}
	return nil
}
{{end}}{{range .GenericCodecs}}
func Encode{{.GoName}}[{{tpList .TypeParams}}](w *orc.Writer, s *{{.GoName}}[{{tpNames .TypeParams}}], {{encodeArgs .TypeParams}}) error {
{{.EncodeBody}}
	return nil
}

func Decode{{.GoName}}[{{tpList .TypeParams}}](r *orc.Reader, s *{{.GoName}}[{{tpNames .TypeParams}}], {{decodeArgs .TypeParams}}) error {
{{- if .UsesErr}}
	var err error
{{- end}}
{{.DecodeBody}}
	return nil
}
{{end}}{{if .Adapters}}
var writerPool = sync.Pool{New: func() any { return orc.NewWriter(0) }}
var readerPool = sync.Pool{New: func() any { return orc.NewReader(nil) }}
{{range .Adapters}}
type {{lowerFirst .GoName}}Codec struct{}

var {{.GoName}}Codec xencoding.Codec = {{lowerFirst .GoName}}Codec{}

func ({{lowerFirst .GoName}}Codec) Encode(ctx context.Context, value any) ([]byte, error) {
	s := value.({{.GoName}})
	w := writerPool.Get().(*orc.Writer)
	defer writerPool.Put(w)
	w.Reset()
	if err := Encode{{.GoName}}(w, &s); err != nil {
		return nil, err
	}
	return w.Copy(), nil
}

func (c {{lowerFirst .GoName}}Codec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	b, err := c.Encode(ctx, value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func ({{lowerFirst .GoName}}Codec) Decode(ctx context.Context, data []byte, value any) error {
	s := value.(*{{.GoName}})
	r := readerPool.Get().(*orc.Reader)
	defer readerPool.Put(r)
	r.ResetBytes(data)
	return Decode{{.GoName}}(r, s)
}

func (c {{lowerFirst .GoName}}Codec) DecodeStream(ctx context.Context, rd io.Reader, value any) error {
	data, err := io.ReadAll(rd)
	if err != nil {
		return err
	}
	return c.Decode(ctx, data, value)
}
{{end}}{{end}}`
