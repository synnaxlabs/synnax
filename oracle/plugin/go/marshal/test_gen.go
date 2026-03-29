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
	Adapters     []testEntry
}

type testEntry struct {
	GoName    string
	ValueExpr string
}

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
		if !ok || form.IsGeneric() {
			continue
		}
		b := &testValueBuilder{
			table:       table,
			repoRoot:    repoRoot,
			packageName: packageName,
			parentPath:  parentPath,
			imports:     fo.ExtraImports,
			pkgPrefix:   packageName + ".",
		}
		valueExpr, err := b.buildStructLiteral(e.Type, e.GoName)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate test value for %s", e.GoName)
		}
		if b.needsUUID {
			fo.NeedsUUID = true
		}
		te := testEntry{GoName: e.GoName, ValueExpr: valueExpr}
		fo.Tests = append(fo.Tests, te)
		if e.Adapter {
			fo.Adapters = append(fo.Adapters, te)
		}
	}

	if len(fo.Tests) == 0 {
		return nil, nil
	}

	tmpl, err := template.New("codec_test").Parse(testCodecTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse test template")
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, fo); err != nil {
		return nil, errors.Wrap(err, "failed to execute test template")
	}
	return buf.Bytes(), nil
}

// testValueBuilder generates Go expressions for fully populated test values.
type testValueBuilder struct {
	table       *resolution.Table
	repoRoot    string
	packageName string
	parentPath  string
	imports     map[string]string
	pkgPrefix   string
	needsUUID   bool
	depth       int
}

func (b *testValueBuilder) buildStructLiteral(
	typ resolution.Type, goName string,
) (string, error) {
	fieldExprs, err := b.buildStructFieldExprs(typ)
	if err != nil {
		return "", errors.Wrapf(err, "type %s", goName)
	}
	return b.pkgPrefix + goName + "{" + strings.Join(fieldExprs, ", ") + "}", nil
}

func (b *testValueBuilder) buildFieldExprs(fields []resolution.Field) ([]string, error) {
	var fieldExprs []string
	for _, f := range fields {
		if f.Type.Name == "nil" {
			continue
		}
		resolved, ok := f.Type.Resolve(b.table)
		if !ok {
			if f.Type.IsTypeParam() {
				continue
			}
			continue
		}
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
		exprs = append(exprs, parentGoName+": "+parentGoType+"{"+strings.Join(parentFieldExprs, ", ")+"}")
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
				r, ok := f.Type.Resolve(b.table)
				if !ok {
					continue
				}
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
			return prefix + goName + typeArgSuffix + "{" + strings.Join(fieldExprs, ", ") + "}", nil
		}

		fieldExprs, err := b.buildStructFieldExprs(actual)
		if err != nil {
			return "", err
		}
		b.depth--
		return prefix + goName + "{" + strings.Join(fieldExprs, ", ") + "}", nil

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
	var base string
	switch primName {
	case "string":
		base = `"test"`
	case "bool":
		base = "true"
	case "int8":
		base = "1"
	case "int16":
		base = "2"
	case "int32":
		base = "3"
	case "int64":
		base = "4"
	case "uint8":
		base = "5"
	case "uint12", "uint16":
		base = "6"
	case "uint20", "uint32":
		base = "7"
	case "uint64":
		base = "8"
	case "float32":
		base = "1.5"
	case "float64":
		base = "2.5"
	case "uuid":
		b.needsUUID = true
		base = `uuid.MustParse("a1b2c3d4-e5f6-7890-abcd-ef1234567890")`
	case "bytes":
		base = "[]byte{1, 2, 3}"
	case "record", "any":
		base = `map[string]interface{}{"key": "value"}`
	default:
		return "", errors.Newf("unsupported primitive for test value: %s", primName)
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
	if len(form.Values) == 0 {
		return goType + "(0)", nil
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
	elemExpr, err := b.valueExpr(elemType, elemRef)
	if err != nil {
		return "", err
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
	keyExpr, err := b.valueExpr(keyType, keyRef)
	if err != nil {
		return "", err
	}
	valExpr, err := b.valueExpr(valType, valRef)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("map[%s]%s{%s: %s}", goKeyType, goValType, keyExpr, valExpr), nil
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
	"encoding/binary"
	"testing"
{{- if .Adapters}}
	"context"
{{- end}}
{{- if .NeedsUUID}}
	"github.com/google/uuid"
{{- end}}

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xbinary "github.com/synnaxlabs/x/binary"

	"{{.PkgImport}}"
{{- range $path, $alias := .ExtraImports}}
	{{if $alias}}{{$alias}} {{end}}"{{$path}}"
{{- end}}
)

var _ = Describe("Codec", func() {
{{- range .Tests}}
	Describe("{{.GoName}}", func() {
		It("should round-trip encode and decode", func() {
			original := {{.ValueExpr}}
			w := xbinary.NewWriter(0, binary.BigEndian)
			Expect({{$.Package}}.Encode{{.GoName}}(w, &original)).To(Succeed())
			var decoded {{$.Package}}.{{.GoName}}
			r := xbinary.NewReader(nil, binary.BigEndian)
			r.ResetBytes(w.Bytes())
			Expect({{$.Package}}.Decode{{.GoName}}(r, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(original))
		})
	})
{{- end}}
{{- range .Adapters}}
	Describe("{{.GoName}}Codec", func() {
		It("should round-trip through the Codec interface", func() {
			original := {{.ValueExpr}}
			ctx := context.Background()
			data, err := {{$.Package}}.{{.GoName}}Codec.Encode(ctx, original)
			Expect(err).ToNot(HaveOccurred())
			var decoded {{$.Package}}.{{.GoName}}
			Expect({{$.Package}}.{{.GoName}}Codec.Decode(ctx, data, &decoded)).To(Succeed())
			Expect(decoded).To(Equal(original))
		})
	})
{{- end}}
})
{{range .Tests}}
func BenchmarkEncodeDecode{{.GoName}}(b *testing.B) {
	s := {{.ValueExpr}}
	w := xbinary.NewWriter(0, binary.BigEndian)
	for i := 0; i < b.N; i++ {
		w.Reset()
		if err := {{$.Package}}.Encode{{.GoName}}(w, &s); err != nil {
			b.Fatal(err)
		}
		var decoded {{$.Package}}.{{.GoName}}
		r := xbinary.NewReader(nil, binary.BigEndian)
		r.ResetBytes(w.Bytes())
		if err := {{$.Package}}.Decode{{.GoName}}(r, &decoded); err != nil {
			b.Fatal(err)
		}
	}
}
{{end}}`
