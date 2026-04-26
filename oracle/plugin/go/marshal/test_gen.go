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
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

type testFileOutput struct {
	Package      string
	PkgImport    string
	ExtraImports map[string]string
	NeedsUUID    bool
	Tests        []testEntry
	GenericTests []genericTestEntry
}

type genericTestEntry struct {
	GoName     string
	Receiver   string
	TypeParams []typeParamData
	Cases      []testCase
}

type testCase struct {
	Name      string
	ValueExpr string
}

type testEntry struct {
	GoName   string
	Receiver string
	Cases    []testCase
}

type valueMode int

const (
	modeFullyPopulated valueMode = iota
	modeZeroValue
	modeEmptyCollections
)

func generateTestCodecFile(
	packageName string,
	parentPath string,
	entries []CodecEntry,
	table *resolution.Table,
	repoRoot string,
) ([]byte, error) {
	pkgImport, err := resolveGoImportPath(parentPath, repoRoot)
	if err != nil {
		return nil, err
	}
	fo := testFileOutput{
		Package:      packageName,
		PkgImport:    pkgImport,
		ExtraImports: make(map[string]string),
	}

	for _, e := range entries {
		form, ok := e.Type.Form.(resolution.StructForm)
		if !ok {
			continue
		}

		// Collect non-defaulted type params (same logic as encoder).
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

		recv := ReceiverName(e.GoName)
		// In test files the receiver is used as a local variable, so it must
		// not shadow the package import alias.
		if recv == packageName {
			recv = recv + "v"
		}
		modes := []struct {
			name string
			mode valueMode
		}{
			{"fully populated", modeFullyPopulated},
			{"zero values", modeZeroValue},
		}
		if typeHasCollections(e.Type, table) {
			modes = append(modes, struct {
				name string
				mode valueMode
			}{"empty collections", modeEmptyCollections})
		}

		if len(typeParams) > 0 {
			// Generic type: substitute concrete types for type params.
			typeArgMap := make(map[string]resolution.TypeRef)
			for _, tp := range form.TypeParams {
				if tp.HasDefault() {
					typeArgMap[tp.Name] = *tp.Default
				} else {
					typeArgMap[tp.Name] = concreteTypeRefForConstraint(tp)
				}
			}

			gte := genericTestEntry{GoName: e.GoName, Receiver: recv, TypeParams: typeParams}
			for _, m := range modes {
				b := &testValueBuilder{
					table:       table,
					repoRoot:    repoRoot,
					packageName: packageName,
					parentPath:  parentPath,
					imports:     fo.ExtraImports,
					pkgPrefix:   packageName + ".",
					mode:        m.mode,
				}

				fields := resolution.UnifiedFields(e.Type, table)
				substituted := make([]resolution.Field, len(fields))
				for i, f := range fields {
					substituted[i] = f
					substituted[i].Type = resolution.SubstituteTypeRef(f.Type, typeArgMap)
				}

				var concreteTypeArgStrs []string
				for _, tp := range typeParams {
					concreteTypeArgStrs = append(concreteTypeArgStrs, concreteGoTypeForConstraint(tp.Constraint))
				}
				concreteGoName := e.GoName + "[" + strings.Join(concreteTypeArgStrs, ", ") + "]"

				var fieldExprs []string
				for _, f := range substituted {
					if f.Type.Name == "nil" {
						continue
					}
					if domain.GetStringFromField(f, "go", "marshal") == "omit" {
						continue
					}
					r, ok := f.Type.Resolve(table)
					if !ok {
						continue
					}
					b.fieldIndex++
					fieldGoName := naming.GetFieldName(f)
					var expr string
					var err error
					if f.IsHardOptional {
						expr, err = b.hardOptionalExpr(r, f.Type)
					} else {
						expr, err = b.valueExpr(r, f.Type)
					}
					if err != nil {
						return nil, errors.Wrapf(err, "failed to generate %s test value for %s field %s", m.name, e.GoName, fieldGoName)
					}
					if expr != "" {
						fieldExprs = append(fieldExprs, fieldGoName+": "+expr)
					}
				}
				if b.needsUUID {
					fo.NeedsUUID = true
				}
				valueExpr := b.formatComposite(b.pkgPrefix+concreteGoName, fieldExprs)
				gte.Cases = append(gte.Cases, testCase{Name: m.name, ValueExpr: valueExpr})
			}
			fo.GenericTests = append(fo.GenericTests, gte)
		} else {
			te := testEntry{GoName: e.GoName, Receiver: recv}
			for _, m := range modes {
				b := &testValueBuilder{
					table:       table,
					repoRoot:    repoRoot,
					packageName: packageName,
					parentPath:  parentPath,
					imports:     fo.ExtraImports,
					pkgPrefix:   packageName + ".",
					mode:        m.mode,
				}
				valueExpr, err := b.buildStructLiteral(e.Type, e.GoName)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to generate %s test value for %s", m.name, e.GoName)
				}
				if b.needsUUID {
					fo.NeedsUUID = true
				}
				te.Cases = append(te.Cases, testCase{Name: m.name, ValueExpr: valueExpr})
			}
			fo.Tests = append(fo.Tests, te)
		}
	}

	if len(fo.Tests) == 0 && len(fo.GenericTests) == 0 {
		return nil, nil
	}

	tmpl, err := template.New("codec_test").Funcs(template.FuncMap{
		"concreteGoType": concreteGoTypeForConstraint,
		"tpNames":        tpNames,
		"sortedImports":  sortedImports,
	}).Parse(testCodecTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse test template")
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, fo); err != nil {
		return nil, errors.Wrap(err, "failed to execute test template")
	}
	return buf.Bytes(), nil
}

func concreteTypeRefForConstraint(tp resolution.TypeParam) resolution.TypeRef {
	return resolution.TypeRef{Name: concreteOracleTypeForConstraint(tp)}
}

func concreteOracleTypeForConstraint(tp resolution.TypeParam) string {
	if tp.Constraint != nil {
		switch tp.Constraint.Name {
		case "comparable":
			return "string"
		case "integer":
			return "int64"
		case "float":
			return "float64"
		case "number":
			return "int64"
		}
	}
	return "string"
}

func concreteGoTypeForConstraint(constraint string) string {
	switch constraint {
	case "comparable":
		return "string"
	case "integer":
		return "int64"
	case "float":
		return "float64"
	case "number":
		return "int64"
	case "any":
		return "string"
	default:
		return "string"
	}
}

func typeHasCollections(typ resolution.Type, table *resolution.Table) bool {
	fields := resolution.UnifiedFields(typ, table)
	for _, f := range fields {
		resolved, ok := f.Type.Resolve(table)
		if !ok {
			continue
		}
		actual, _ := typemap.UnwrapTypeRef(resolved, f.Type, table)
		if bg, ok := actual.Form.(resolution.BuiltinGenericForm); ok {
			if bg.Name == "Array" || bg.Name == "Map" {
				return true
			}
		}
	}
	return false
}

// testValueBuilder generates Go expressions for test values.
type testValueBuilder struct {
	table       *resolution.Table
	repoRoot    string
	packageName string
	parentPath  string
	imports     map[string]string
	pkgPrefix   string
	needsUUID   bool
	depth       int
	fieldIndex  int
	mode        valueMode
}

func (b *testValueBuilder) buildStructLiteral(
	typ resolution.Type, goName string,
) (string, error) {
	fieldExprs, err := b.buildStructFieldExprs(typ)
	if err != nil {
		return "", errors.Wrapf(err, "type %s", goName)
	}
	return b.formatComposite(b.pkgPrefix+goName, fieldExprs), nil
}

func (b *testValueBuilder) formatComposite(typeName string, entries []string) string {
	if len(entries) == 0 {
		return typeName + "{}"
	}
	singleLine := typeName + "{" + strings.Join(entries, ", ") + "}"
	if len(entries) <= 2 && len(singleLine) <= 80 {
		return singleLine
	}
	return typeName + "{\n" + strings.Join(entries, ",\n") + ",\n}"
}

func (b *testValueBuilder) buildFieldExprs(fields []resolution.Field) ([]string, error) {
	var fieldExprs []string
	for _, f := range fields {
		if f.Type.Name == "nil" {
			continue
		}
		if domain.GetStringFromField(f, "go", "marshal") == "omit" {
			continue
		}
		resolved, ok := f.Type.Resolve(b.table)
		if !ok {
			if f.Type.IsTypeParam() {
				continue
			}
			continue
		}
		b.fieldIndex++
		fieldGoName := naming.GetFieldName(f)
		var expr string
		var err error
		if f.IsHardOptional {
			expr, err = b.hardOptionalExpr(resolved, f.Type)
		} else {
			expr, err = b.valueExpr(resolved, f.Type)
		}
		if err != nil {
			return nil, errors.Wrapf(err, "field %s", fieldGoName)
		}
		if expr != "" {
			fieldExprs = append(fieldExprs, fieldGoName+": "+expr)
		}
	}
	return fieldExprs, nil
}

func (b *testValueBuilder) buildStructFieldExprs(typ resolution.Type) ([]string, error) {
	form, ok := typ.Form.(resolution.StructForm)
	if !ok {
		return nil, nil
	}
	if resolver.CanUseInheritance(form, b.table) {
		return b.buildEmbeddedStructFieldExprs(form)
	}
	fields := resolution.UnifiedFields(typ, b.table)
	return b.buildFieldExprs(fields)
}

func (b *testValueBuilder) buildEmbeddedStructFieldExprs(
	form resolution.StructForm,
) ([]string, error) {
	var exprs []string
	for _, extendsRef := range form.Extends {
		parent, ok := extendsRef.Resolve(b.table)
		if !ok {
			continue
		}
		parentGoName := naming.GetGoName(parent)
		parentGoType, err := b.goTypeName(parent)
		if err != nil {
			return nil, err
		}
		parentFieldExprs, err := b.buildStructFieldExprs(parent)
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, parentGoName+": "+b.formatComposite(parentGoType, parentFieldExprs))
	}
	childFieldExprs, err := b.buildFieldExprs(form.Fields)
	if err != nil {
		return nil, err
	}
	exprs = append(exprs, childFieldExprs...)
	return exprs, nil
}

func (b *testValueBuilder) hardOptionalExpr(
	resolved resolution.Type, ref resolution.TypeRef,
) (string, error) {
	if b.mode == modeZeroValue {
		return "nil", nil
	}
	inner, err := b.valueExpr(resolved, ref)
	if err != nil {
		return "", err
	}
	if inner == "" {
		return "nil", nil
	}
	// For struct/map/slice literals, extract the type from "Type{...}".
	// For primitives, cast the value to the correct type so that Go's
	// type inference assigns the right type to v (e.g., uint8(5) not just 5).
	var goType string
	if idx := strings.Index(inner, "{"); idx >= 0 {
		goType = inner[:idx]
		return fmt.Sprintf("func() *%s { v := %s; return &v }()", goType, inner), nil
	}
	goType, err = b.goTypeName(resolved)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("func() *%s { v := %s(%s); return &v }()", goType, goType, inner), nil
}

func (b *testValueBuilder) valueExpr(
	resolved resolution.Type, ref resolution.TypeRef,
) (string, error) {
	actual, effectiveTypeArgs := typemap.UnwrapTypeRef(resolved, ref, b.table)

	switch form := actual.Form.(type) {
	case resolution.StructForm:
		if b.depth > 2 {
			goType, err := b.goTypeName(actual)
			if err != nil {
				return "", err
			}
			return goType + "{}", nil
		}
		b.depth++
		goName := naming.GetGoName(actual)
		structPath := output.GetPath(actual, "go")
		prefix := b.pkgPrefix
		if structPath != "" && structPath != b.parentPath {
			goType, err := b.goTypeName(actual)
			if err != nil {
				return "", err
			}
			prefix = strings.TrimSuffix(goType, goName)
		}

		if form.IsGeneric() && len(effectiveTypeArgs) > 0 {
			typeArgMap := make(map[string]resolution.TypeRef)
			for i, tp := range form.TypeParams {
				if i < len(effectiveTypeArgs) {
					typeArgMap[tp.Name] = effectiveTypeArgs[i]
				} else if tp.HasDefault() {
					typeArgMap[tp.Name] = *tp.Default
				}
			}
			fields := resolution.UnifiedFields(actual, b.table)
			substituted := make([]resolution.Field, len(fields))
			for i, f := range fields {
				substituted[i] = f
				substituted[i].Type = resolution.SubstituteTypeRef(f.Type, typeArgMap)
			}
			var fieldExprs []string
			for _, f := range substituted {
				if f.Type.Name == "nil" {
					continue
				}
				if domain.GetStringFromField(f, "go", "marshal") == "omit" {
					continue
				}
				r, ok := f.Type.Resolve(b.table)
				if !ok {
					continue
				}
				b.fieldIndex++
				fieldGoName := naming.GetFieldName(f)
				var expr string
				var err error
				if f.IsHardOptional {
					expr, err = b.hardOptionalExpr(r, f.Type)
				} else {
					expr, err = b.valueExpr(r, f.Type)
				}
				if err != nil {
					return "", err
				}
				if expr != "" {
					fieldExprs = append(fieldExprs, fieldGoName+": "+expr)
				}
			}
			// Build type argument list for the generic struct literal.
			var typeArgStrs []string
			for _, arg := range effectiveTypeArgs {
				argResolved, ok := arg.Resolve(b.table)
				if !ok || arg.Name == "nil" {
					continue
				}
				argGoType, err := b.goTypeName(argResolved)
				if err != nil {
					return "", err
				}
				typeArgStrs = append(typeArgStrs, argGoType)
			}
			typeArgSuffix := ""
			if len(typeArgStrs) > 0 {
				typeArgSuffix = "[" + strings.Join(typeArgStrs, ", ") + "]"
			}
			b.depth--
			return b.formatComposite(prefix+goName+typeArgSuffix, fieldExprs), nil
		}

		fieldExprs, err := b.buildStructFieldExprs(actual)
		if err != nil {
			return "", err
		}
		b.depth--
		return b.formatComposite(prefix+goName, fieldExprs), nil

	case resolution.BuiltinGenericForm:
		if form.Name == "Array" && len(effectiveTypeArgs) > 0 {
			return b.arrayExpr(effectiveTypeArgs[0])
		}
		if form.Name == "Map" && len(effectiveTypeArgs) >= 2 {
			return b.mapExpr(effectiveTypeArgs[0], effectiveTypeArgs[1])
		}
		return "", nil

	case resolution.EnumForm:
		return b.enumExpr(resolved, form)

	default:
		return b.primitiveExpr(resolved)
	}
}

func (b *testValueBuilder) primitiveExpr(typ resolution.Type) (string, error) {
	primName, goTypeCast, err := b.resolveLeafPrim(typ)
	if err != nil {
		return "", err
	}
	if b.mode == modeZeroValue {
		return b.zeroPrimitiveExpr(primName, goTypeCast)
	}
	idx := b.fieldIndex
	var base string
	switch primName {
	case "string":
		base = fmt.Sprintf(`"test_%d"`, idx)
	case "bool":
		if idx%2 == 0 {
			base = "false"
		} else {
			base = "true"
		}
	case "int8":
		base = fmt.Sprintf("%d", (idx%126)+1)
	case "int16":
		base = fmt.Sprintf("%d", idx+1)
	case "int32":
		base = fmt.Sprintf("%d", idx+1)
	case "int64":
		base = fmt.Sprintf("%d", idx+1)
	case "uint8":
		base = fmt.Sprintf("%d", (idx%254)+1)
	case "uint12", "uint16":
		base = fmt.Sprintf("%d", idx+1)
	case "uint20", "uint32":
		base = fmt.Sprintf("%d", idx+1)
	case "uint64":
		base = fmt.Sprintf("%d", idx+1)
	case "float32":
		base = fmt.Sprintf("%d.5", idx)
	case "float64":
		base = fmt.Sprintf("%d.5", idx)
	case "uuid":
		b.needsUUID = true
		base = fmt.Sprintf(`uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef12345678%02x")`, idx%256)
	case "bytes":
		base = fmt.Sprintf("[]byte{%d, %d, %d}", idx%256, (idx+1)%256, (idx+2)%256)
	case "record", "any":
		base = fmt.Sprintf(`map[string]interface{}{"key_%d": "value_%d"}`, idx, idx)
	default:
		return "", errors.Newf("unsupported primitive for test value: %s", primName)
	}
	if goTypeCast != "" {
		return goTypeCast + "(" + base + ")", nil
	}
	return base, nil
}

func (b *testValueBuilder) zeroPrimitiveExpr(primName, goTypeCast string) (string, error) {
	var base string
	switch primName {
	case "string":
		base = `""`
	case "bool":
		base = "false"
	case "int8", "int16", "int32", "int64",
		"uint8", "uint12", "uint16", "uint20", "uint32", "uint64":
		base = "0"
	case "float32", "float64":
		base = "0"
	case "uuid":
		b.needsUUID = true
		base = "uuid.Nil"
	case "bytes":
		return "nil", nil
	case "record", "any":
		return "nil", nil
	default:
		return "", errors.Newf("unsupported primitive for zero test value: %s", primName)
	}
	if goTypeCast != "" {
		return goTypeCast + "(" + base + ")", nil
	}
	return base, nil
}

func (b *testValueBuilder) enumExpr(
	typ resolution.Type, form resolution.EnumForm,
) (string, error) {
	goType, err := b.goTypeName(typ)
	if err != nil {
		return "", err
	}
	if b.mode == modeZeroValue || len(form.Values) == 0 {
		if form.IsIntEnum {
			return goType + "(0)", nil
		}
		return goType + `("")`, nil
	}
	v := form.Values[0]
	if form.IsIntEnum {
		return fmt.Sprintf("%s(%v)", goType, v.Value), nil
	}
	return fmt.Sprintf("%s(%q)", goType, v.Value), nil
}

func (b *testValueBuilder) arrayExpr(elemRef resolution.TypeRef) (string, error) {
	elemType, ok := elemRef.Resolve(b.table)
	if !ok {
		return "nil", nil
	}
	goType, err := b.goSliceElemType(elemType)
	if err != nil {
		return "", err
	}
	if b.mode == modeZeroValue {
		return "nil", nil
	}
	if b.mode == modeEmptyCollections {
		return fmt.Sprintf("[]%s{}", goType), nil
	}
	elemExpr, err := b.valueExpr(elemType, elemRef)
	if err != nil {
		return "", err
	}
	if strings.Contains(elemExpr, "\n") {
		return b.formatComposite("[]"+goType, []string{elemExpr}), nil
	}
	return fmt.Sprintf("[]%s{%s}", goType, elemExpr), nil
}

func (b *testValueBuilder) mapExpr(keyRef, valRef resolution.TypeRef) (string, error) {
	keyType, ok := keyRef.Resolve(b.table)
	if !ok {
		return "nil", nil
	}
	valType, ok := valRef.Resolve(b.table)
	if !ok {
		return "nil", nil
	}
	goKeyType, err := b.goTypeName(keyType)
	if err != nil {
		return "", err
	}
	goValType, err := b.goTypeName(valType)
	if err != nil {
		return "", err
	}
	mapType := fmt.Sprintf("map[%s]%s", goKeyType, goValType)
	if b.mode == modeZeroValue {
		return "nil", nil
	}
	if b.mode == modeEmptyCollections {
		return mapType + "{}", nil
	}
	keyExpr, err := b.valueExpr(keyType, keyRef)
	if err != nil {
		return "", err
	}
	valExpr, err := b.valueExpr(valType, valRef)
	if err != nil {
		return "", err
	}
	entry := keyExpr + ": " + valExpr
	if strings.Contains(valExpr, "\n") {
		return b.formatComposite(mapType, []string{entry}), nil
	}
	return fmt.Sprintf("%s{%s}", mapType, entry), nil
}

func (b *testValueBuilder) resolveLeafPrim(typ resolution.Type) (string, string, error) {
	return typemap.ResolveLeafPrimitive(typ, b.table, b.goTypeName)
}

func (b *testValueBuilder) goTypeName(typ resolution.Type) (string, error) {
	if prim, ok := typ.Form.(resolution.PrimitiveForm); ok {
		goType, ok := typemap.PrimitiveGoType(prim.Name)
		if !ok {
			return "", errors.Newf("unsupported primitive: %s", prim.Name)
		}
		if typemap.IsUUID(prim.Name) {
			b.needsUUID = true
			b.imports["github.com/google/uuid"] = "uuid"
		}
		if typemap.IsRecord(prim.Name) {
			b.imports["github.com/synnaxlabs/x/encoding/msgpack"] = "msgpack"
		}
		return goType, nil
	}
	goName := naming.GetGoName(typ)
	goPath := output.GetPath(typ, "go")
	if goPath == "" || goPath == b.parentPath {
		return b.pkgPrefix + goName, nil
	}
	importPath, err := resolveGoImportPath(goPath, b.repoRoot)
	if err != nil {
		return "", err
	}
	alias := naming.DerivePackageAlias(goPath, b.packageName)
	actualPkg := filepath.Base(importPath)
	if alias == actualPkg {
		if _, ok := b.imports[importPath]; !ok {
			b.imports[importPath] = ""
		}
	} else {
		b.imports[importPath] = alias
	}
	qualifier := alias
	if existing, ok := b.imports[importPath]; ok && existing != "" {
		qualifier = existing
	}
	return qualifier + "." + goName, nil
}

func (b *testValueBuilder) goSliceElemType(typ resolution.Type) (string, error) {
	return typemap.ResolveGoSliceElemType(typ, b.table, b.goTypeName)
}

const testCodecTemplate = `// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Code generated by oracle. DO NOT EDIT.

package {{.Package}}_test

import (
	"reflect"
	"testing"
{{- if .NeedsUUID}}
	"github.com/google/uuid"
{{- end}}

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/encoding/orc"

	"{{.PkgImport}}"
{{- range sortedImports .ExtraImports}}
	{{if .Alias}}{{.Alias}} {{end}}"{{.Path}}"
{{- end}}
)

var _ = Describe("Codec", func() {
{{- range .Tests}}
	Describe("{{.GoName}}", func() {
		DescribeTable("should round-trip encode and decode",
			func(original {{$.Package}}.{{.GoName}}) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded {{$.Package}}.{{.GoName}}
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			{{- range .Cases}}
			Entry("{{.Name}}", {{.ValueExpr}}),
			{{- end}}
		)
	})
{{- end}}
{{- range .GenericTests}}
	Describe("{{.GoName}}", func() {
		DescribeTable("should round-trip encode and decode",
			func(original {{$.Package}}.{{.GoName}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{concreteGoType $tp.Constraint}}{{end}}]) {
				w := orc.NewWriter(0)
				Expect(original.EncodeOrc(w)).To(Succeed())
				var decoded {{$.Package}}.{{.GoName}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{concreteGoType $tp.Constraint}}{{end}}]
				r := orc.NewReader(nil)
				r.ResetBytes(w.Bytes())
				Expect(decoded.DecodeOrc(r)).To(Succeed())
				Expect(decoded).To(Equal(original))
			},
			{{- range .Cases}}
			Entry("{{.Name}}", {{.ValueExpr}}),
			{{- end}}
		)
	})
{{- end}}
})
{{range .GenericTests}}
func BenchmarkEncodeDecode{{.GoName}}(b *testing.B) {
	{{.Receiver}} := {{(index .Cases 0).ValueExpr}}
	w := orc.NewWriter(0)
	r := orc.NewReader(nil)
	for i := 0; i < b.N; i++ {
		w.Reset()
		if err := {{.Receiver}}.EncodeOrc(w); err != nil {
			b.Fatal(err)
		}
		var decoded {{$.Package}}.{{.GoName}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{concreteGoType $tp.Constraint}}{{end}}]
		r.ResetBytes(w.Bytes())
		if err := decoded.DecodeOrc(r); err != nil {
			b.Fatal(err)
		}
	}
}
{{end}}{{range .GenericTests}}
func FuzzDecode{{.GoName}}(f *testing.F) {
	{{- $goName := .GoName}}
	{{- $typeParams := .TypeParams}}
	{{- $recv := .Receiver}}
	{{- range .Cases}}
	{
		seed := {{.ValueExpr}}
		w := orc.NewWriter(0)
		if err := seed.EncodeOrc(w); err != nil {
			f.Fatal(err)
		}
		f.Add(w.Bytes())
	}
	{{- end}}
	f.Fuzz(func(t *testing.T, data []byte) {
		var decoded {{$.Package}}.{{.GoName}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{concreteGoType $tp.Constraint}}{{end}}]
		r := orc.NewReader(nil)
		r.ResetBytes(data)
		if err := decoded.DecodeOrc(r); err != nil {
			return
		}
		w1 := orc.NewWriter(len(data))
		if err := decoded.EncodeOrc(w1); err != nil {
			t.Fatalf("encode after successful decode failed: %v", err)
		}
		var redecoded {{$.Package}}.{{.GoName}}[{{range $i, $tp := .TypeParams}}{{if $i}}, {{end}}{{concreteGoType $tp.Constraint}}{{end}}]
		r.ResetBytes(w1.Bytes())
		if err := redecoded.DecodeOrc(r); err != nil {
			t.Fatalf("re-decode failed: %v", err)
		}
		w2 := orc.NewWriter(w1.Len())
		if err := redecoded.EncodeOrc(w2); err != nil {
			t.Fatalf("re-encode failed: %v", err)
		}
		if w1.Len() != w2.Len() {
			t.Fatalf("encoded length differs between cycles: w1=%d w2=%d", w1.Len(), w2.Len())
		}
		if !reflect.DeepEqual(decoded, redecoded) {
			t.Fatal("round-trip mismatch: decoded values differ after re-encode/re-decode cycle")
		}
	})
}
{{end}}{{range .Tests}}
func BenchmarkEncodeDecode{{.GoName}}(b *testing.B) {
	{{.Receiver}} := {{(index .Cases 0).ValueExpr}}
	w := orc.NewWriter(0)
	r := orc.NewReader(nil)
	for i := 0; i < b.N; i++ {
		w.Reset()
		if err := {{.Receiver}}.EncodeOrc(w); err != nil {
			b.Fatal(err)
		}
		var decoded {{$.Package}}.{{.GoName}}
		r.ResetBytes(w.Bytes())
		if err := decoded.DecodeOrc(r); err != nil {
			b.Fatal(err)
		}
	}
}
{{end}}{{range .Tests}}
func FuzzDecode{{.GoName}}(f *testing.F) {
	{{- $goName := .GoName}}
	{{- range .Cases}}
	{
		seed := {{.ValueExpr}}
		w := orc.NewWriter(0)
		if err := seed.EncodeOrc(w); err != nil {
			f.Fatal(err)
		}
		f.Add(w.Bytes())
	}
	{{- end}}
	f.Fuzz(func(t *testing.T, data []byte) {
		var decoded {{$.Package}}.{{.GoName}}
		r := orc.NewReader(nil)
		r.ResetBytes(data)
		if err := decoded.DecodeOrc(r); err != nil {
			return
		}
		w1 := orc.NewWriter(len(data))
		if err := decoded.EncodeOrc(w1); err != nil {
			t.Fatalf("encode after successful decode failed: %v", err)
		}
		var redecoded {{$.Package}}.{{.GoName}}
		r.ResetBytes(w1.Bytes())
		if err := redecoded.DecodeOrc(r); err != nil {
			t.Fatalf("re-decode failed: %v", err)
		}
		w2 := orc.NewWriter(w1.Len())
		if err := redecoded.EncodeOrc(w2); err != nil {
			t.Fatalf("re-encode failed: %v", err)
		}
		if w1.Len() != w2.Len() {
			t.Fatalf("encoded length differs between cycles: w1=%d w2=%d", w1.Len(), w2.Len())
		}
		if !reflect.DeepEqual(decoded, redecoded) {
			t.Fatal("round-trip mismatch: decoded values differ after re-encode/re-decode cycle")
		}
	})
}
{{end}}`
