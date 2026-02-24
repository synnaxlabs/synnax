// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package marshal provides an Oracle plugin that generates gorp.Codec implementations
// using direct binary encoding for zero-allocation serialization.
package marshal

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/exec"
	"github.com/synnaxlabs/oracle/plugin"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
)

// Plugin generates gorp.Codec implementations for structs annotated with @go marshal.
type Plugin struct{ Options Options }

// Options configures the go/marshal plugin.
type Options struct {
	FileNamePattern string
}

// DefaultOptions returns the default plugin options.
func DefaultOptions() Options {
	return Options{FileNamePattern: "codec.gen.go"}
}

// New creates a new go/marshal plugin with the given options.
func New(opts Options) *Plugin { return &Plugin{Options: opts} }

func (p *Plugin) Name() string                { return "go/marshal" }
func (p *Plugin) Domains() []string           { return []string{"go"} }
func (p *Plugin) Requires() []string          { return []string{"go/types", "go/pb"} }
func (p *Plugin) Check(*plugin.Request) error { return nil }

var goPostWriter = &exec.PostWriter{
	Extensions: []string{".go"},
	Commands:   [][]string{{"gofmt", "-w"}},
}

func (p *Plugin) PostWrite(files []string) error {
	return goPostWriter.PostWrite(files)
}

func hasMarshalAnnotation(typ resolution.Type) bool {
	domain, ok := typ.Domains["go"]
	if !ok {
		return false
	}
	for _, expr := range domain.Expressions {
		if expr.Name == "marshal" {
			return true
		}
	}
	return false
}

func getGoName(s resolution.Type) string {
	if domain, ok := s.Domains["go"]; ok {
		for _, expr := range domain.Expressions {
			if expr.Name == "name" && len(expr.Values) > 0 {
				return expr.Values[0].StringValue
			}
		}
	}
	return ""
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

type codecEntry struct {
	GoName      string
	PBName      string
	ParentAlias string
	ParentPath  string
	Type        resolution.Type
}

func (p *Plugin) Generate(req *plugin.Request) (*plugin.Response, error) {
	resp := &plugin.Response{Files: make([]plugin.File, 0)}
	outputEntries := make(map[string][]codecEntry)
	var outputOrder []string

	for _, entry := range req.Resolutions.StructTypes() {
		if !hasMarshalAnnotation(entry) {
			continue
		}
		if form, ok := entry.Form.(resolution.StructForm); ok && form.IsGeneric() {
			continue
		}
		if !output.HasPB(entry) {
			continue
		}
		goPath := output.GetPath(entry, "go")
		if goPath == "" {
			continue
		}
		pbPath := goPath + "/pb"
		if req.RepoRoot != "" {
			if err := req.ValidateOutputPath(pbPath); err != nil {
				return nil, errors.Wrapf(err, "invalid output path for %s", entry.Name)
			}
		}
		goName := getGoName(entry)
		if goName == "" {
			goName = entry.Name
		}
		pbName := getPBName(entry)
		if pbName == "" {
			pbName = entry.Name
		}
		parentAlias := naming.DerivePackageAlias(goPath, "pb")
		if _, exists := outputEntries[pbPath]; !exists {
			outputOrder = append(outputOrder, pbPath)
		}
		outputEntries[pbPath] = append(outputEntries[pbPath], codecEntry{
			GoName:      goName,
			PBName:      pbName,
			ParentAlias: parentAlias,
			ParentPath:  goPath,
			Type:        entry,
		})
	}

	for _, pbPath := range outputOrder {
		entries := outputEntries[pbPath]
		content, err := p.generateFile(pbPath, entries, req)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to generate %s", pbPath)
		}
		if len(content) > 0 {
			resp.Files = append(resp.Files, plugin.File{
				Path:    fmt.Sprintf("%s/%s", pbPath, p.Options.FileNamePattern),
				Content: content,
			})
		}
	}
	return resp, nil
}

type codecOutput struct {
	GoName        string
	ParentAlias   string
	Constants     string
	FieldCount    int
	EstSize       int
	MarshalBody   string
	UnmarshalBody string
	HelperFuncs   string
}

type fileOutput struct {
	Package          string
	ParentAlias      string
	ParentImportPath string
	ExtraImports     map[string]string
	NeedsMath        bool
	NeedsJSON        bool
	Codecs           []codecOutput
}

func (p *Plugin) generateFile(
	pbPath string,
	entries []codecEntry,
	req *plugin.Request,
) ([]byte, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	parentAlias := entries[0].ParentAlias
	parentGoPath := entries[0].ParentPath
	parentImportPath, err := resolveGoImportPath(parentGoPath, req.RepoRoot)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve parent package import")
	}

	fo := fileOutput{
		Package:          "pb",
		ParentAlias:      parentAlias,
		ParentImportPath: parentImportPath,
		ExtraImports:     make(map[string]string),
	}

	for _, e := range entries {
		b := &codeBuilder{
			table:       req.Resolutions,
			repoRoot:    req.RepoRoot,
			parentAlias: e.ParentAlias,
			parentPath:  e.ParentPath,
			imports:     fo.ExtraImports,
		}
		if err := b.processType(e.Type); err != nil {
			return nil, errors.Wrapf(err, "failed to generate codec for %s", e.GoName)
		}
		if b.needsMath {
			fo.NeedsMath = true
		}
		if b.needsJSON {
			fo.NeedsJSON = true
		}

		constName := naming.ToPascalCase(e.GoName)
		var constBuf bytes.Buffer
		for _, c := range b.consts {
			fmt.Fprintf(&constBuf, "\t%s%s = %d\n", constName, c.name, c.index)
		}
		fmt.Fprintf(&constBuf, "\t%sFieldCount = %d", constName, len(b.consts))

		fo.Codecs = append(fo.Codecs, codecOutput{
			GoName:        e.GoName,
			ParentAlias:   e.ParentAlias,
			Constants:     constBuf.String(),
			FieldCount:    len(b.consts),
			EstSize:       b.estSize,
			MarshalBody:   strings.Join(b.marshalLines, "\n"),
			UnmarshalBody: strings.Join(b.unmarshalLines, "\n"),
			HelperFuncs:   strings.Join(b.helperFuncs, "\n"),
		})
	}

	tmpl, err := template.New("codec").Funcs(template.FuncMap{
		"lowerFirst": lowerFirst,
	}).Parse(codecTemplate)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse template")
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, fo); err != nil {
		return nil, errors.Wrap(err, "failed to execute template")
	}
	return buf.Bytes(), nil
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// codeBuilder generates binary marshal/unmarshal code for a single type.
type codeBuilder struct {
	table            *resolution.Table
	repoRoot         string
	parentAlias      string
	parentPath       string
	imports          map[string]string
	consts           []wireConst
	marshalLines     []string
	unmarshalLines   []string
	estSize          int
	needsMath        bool
	needsJSON        bool
	index            int
	depth            int
	varCounter       int
	processingTypes  map[string]bool
	helperFuncs      []string
	generatedHelpers map[string]bool
}

func (b *codeBuilder) nextVar(prefix string) string {
	b.varCounter++
	return fmt.Sprintf("%s%d", prefix, b.varCounter)
}

type wireConst struct {
	name  string
	index int
}

func (b *codeBuilder) indent() string { return strings.Repeat("\t", b.depth+1) }

func (b *codeBuilder) processType(typ resolution.Type) error {
	b.processingTypes = make(map[string]bool)
	b.generatedHelpers = make(map[string]bool)
	fields := resolution.UnifiedFields(typ, b.table)
	return b.processFields(fields, "s", "r", "Field")
}

func (b *codeBuilder) processFields(
	fields []resolution.Field,
	getPrefix, setPrefix, constPrefix string,
) error {
	for _, f := range fields {
		if f.Type.Name == "nil" {
			continue
		}
		goName := naming.GetFieldName(f)
		getPath := getPrefix + "." + goName
		setPath := setPrefix + "." + goName
		cPrefix := constPrefix + goName
		if f.IsHardOptional {
			if err := b.processHardOptional(f, getPath, setPath, cPrefix); err != nil {
				return err
			}
		} else if f.IsOptional && b.isGoNilable(f.Type) {
			if err := b.processSoftOptionalNilable(f, getPath, setPath, cPrefix); err != nil {
				return err
			}
		} else {
			if err := b.processFieldValue(f, getPath, setPath, cPrefix); err != nil {
				return err
			}
		}
	}
	return nil
}

// isGoNilable returns true if the resolved Go type is nilable (slice or map).
func (b *codeBuilder) isGoNilable(ref resolution.TypeRef) bool {
	resolved, ok := ref.Resolve(b.table)
	if !ok {
		return false
	}
	actual := resolved
	for {
		switch form := actual.Form.(type) {
		case resolution.AliasForm:
			target, ok := form.Target.Resolve(b.table)
			if !ok {
				return false
			}
			actual = target
			continue
		case resolution.DistinctForm:
			target, ok := form.Base.Resolve(b.table)
			if !ok {
				return false
			}
			actual = target
			continue
		default:
		}
		break
	}
	if bg, ok := actual.Form.(resolution.BuiltinGenericForm); ok {
		return bg.Name == "Array" || bg.Name == "Map"
	}
	return false
}

// processHardOptional handles ?? (pointer) optional fields with a 1-byte presence flag.
func (b *codeBuilder) processHardOptional(
	f resolution.Field, getPath, setPath, constPrefix string,
) error {
	if f.Type.Name == "nil" {
		return nil
	}
	ind := b.indent()
	resolved, ok := b.resolveTypeRef(f.Type)
	if !ok {
		return fmt.Errorf("cannot resolve type %q (field=%s)", f.Type.Name, f.Name)
	}

	// Hard optional arrays/maps: pointer to slice/map
	actual := b.unwrapType(resolved)
	if bg, ok := actual.Form.(resolution.BuiltinGenericForm); ok && (bg.Name == "Array" || bg.Name == "Map") {
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("if %s != nil {", getPath),
			ind+"\tbuf = append(buf, 1)",
		)
		b.unmarshalLines = append(b.unmarshalLines,
			ind+"if data[0] == 1 {",
			ind+"\tdata = data[1:]",
		)
		b.depth++
		derefGet := "(*" + getPath + ")"
		if err := b.processValueByType(resolved, f.Type, derefGet, setPath, constPrefix); err != nil {
			return err
		}
		b.depth--
		b.marshalLines = append(b.marshalLines, ind+"} else {", ind+"\tbuf = append(buf, 0)", ind+"}")
		b.unmarshalLines = append(b.unmarshalLines, ind+"} else {", ind+"\tdata = data[1:]", ind+"}")
		return nil
	}

	goType, err := b.goTypeName(resolved)
	if err != nil {
		return err
	}
	ov := b.nextVar("_ov")
	b.marshalLines = append(b.marshalLines,
		ind+fmt.Sprintf("if %s != nil {", getPath),
		ind+"\tbuf = append(buf, 1)",
	)
	b.unmarshalLines = append(b.unmarshalLines,
		ind+"if data[0] == 1 {",
		ind+"\tdata = data[1:]",
		ind+fmt.Sprintf("\tvar %s %s", ov, goType),
	)

	b.depth++
	derefGet := "(*" + getPath + ")"
	if err := b.processValueByType(resolved, f.Type, derefGet, ov, constPrefix); err != nil {
		return err
	}
	b.depth--

	b.marshalLines = append(b.marshalLines, ind+"} else {", ind+"\tbuf = append(buf, 0)", ind+"}")
	b.unmarshalLines = append(b.unmarshalLines,
		ind+"\t"+setPath+" = &"+ov,
		ind+"} else {",
		ind+"\tdata = data[1:]",
		ind+"}",
	)
	return nil
}

// processSoftOptionalNilable handles ? optional on slices/maps (nilable without pointer).
func (b *codeBuilder) processSoftOptionalNilable(
	f resolution.Field, getPath, setPath, constPrefix string,
) error {
	ind := b.indent()
	resolved, ok := b.resolveTypeRef(f.Type)
	if !ok {
		return fmt.Errorf("cannot resolve type %q (field=%s)", f.Type.Name, f.Name)
	}

	b.marshalLines = append(b.marshalLines,
		ind+fmt.Sprintf("if %s != nil {", getPath),
		ind+"\tbuf = append(buf, 1)",
	)
	b.unmarshalLines = append(b.unmarshalLines,
		ind+"if data[0] == 1 {",
		ind+"\tdata = data[1:]",
	)

	b.depth++
	if err := b.processValueByType(resolved, f.Type, getPath, setPath, constPrefix); err != nil {
		return err
	}
	b.depth--

	b.marshalLines = append(b.marshalLines, ind+"} else {", ind+"\tbuf = append(buf, 0)", ind+"}")
	b.unmarshalLines = append(b.unmarshalLines, ind+"} else {", ind+"\tdata = data[1:]", ind+"}")
	return nil
}

// unwrapType resolves through aliases and distinct types to find the underlying form.
func (b *codeBuilder) unwrapType(typ resolution.Type) resolution.Type {
	actual := typ
	for {
		switch form := actual.Form.(type) {
		case resolution.AliasForm:
			target, ok := form.Target.Resolve(b.table)
			if !ok {
				return actual
			}
			actual = target
			continue
		case resolution.DistinctForm:
			target, ok := form.Base.Resolve(b.table)
			if !ok {
				return actual
			}
			actual = target
			continue
		default:
		}
		break
	}
	return actual
}

func (b *codeBuilder) processValueByType(
	resolved resolution.Type, ref resolution.TypeRef,
	getPath, setPath, constPrefix string,
) error {
	actual := resolved
	effectiveTypeArgs := ref.TypeArgs
	for {
		switch form := actual.Form.(type) {
		case resolution.AliasForm:
			target, ok := form.Target.Resolve(b.table)
			if !ok {
				return fmt.Errorf("cannot resolve alias target %s", form.Target.Name)
			}
			if len(form.Target.TypeArgs) > 0 {
				effectiveTypeArgs = form.Target.TypeArgs
			}
			actual = target
			continue
		case resolution.DistinctForm:
			target, ok := form.Base.Resolve(b.table)
			if !ok {
				return fmt.Errorf("cannot resolve distinct base %s", form.Base.Name)
			}
			if len(form.Base.TypeArgs) > 0 {
				effectiveTypeArgs = form.Base.TypeArgs
			}
			actual = target
			continue
		default:
			_ = form
		}
		break
	}

	switch form := actual.Form.(type) {
	case resolution.StructForm:
		if b.processingTypes[actual.QualifiedName] {
			return b.processRecursiveStruct(actual, getPath, setPath, constPrefix)
		}
		b.processingTypes[actual.QualifiedName] = true
		defer func() { delete(b.processingTypes, actual.QualifiedName) }()

		innerFields := resolution.UnifiedFields(actual, b.table)
		if form.IsGeneric() && len(effectiveTypeArgs) > 0 {
			typeArgMap := make(map[string]resolution.TypeRef)
			for i, tp := range form.TypeParams {
				if i < len(effectiveTypeArgs) {
					typeArgMap[tp.Name] = effectiveTypeArgs[i]
				} else if tp.HasDefault() {
					typeArgMap[tp.Name] = *tp.Default
				}
			}
			substituted := make([]resolution.Field, len(innerFields))
			for i, f := range innerFields {
				substituted[i] = f
				substituted[i].Type = resolution.SubstituteTypeRef(f.Type, typeArgMap)
			}
			innerFields = substituted
		}
		return b.processFields(innerFields, getPath, setPath, constPrefix)
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
				return b.processArray(fakeField, getPath, setPath, constPrefix)
			}
		}
		if form.Name == "Map" {
			typeArgs := ref.TypeArgs
			if len(typeArgs) == 0 {
				typeArgs = effectiveTypeArgs
			}
			if len(typeArgs) >= 2 {
				return b.processMap(typeArgs[0], typeArgs[1], getPath, setPath, constPrefix)
			}
		}
		return fmt.Errorf("unsupported builtin generic: %s", form.Name)
	default:
		return b.processLeaf(resolved, getPath, setPath, constPrefix)
	}
}

func (b *codeBuilder) processFieldValue(
	f resolution.Field, getPath, setPath, constPrefix string,
) error {
	resolved, ok := b.resolveTypeRef(f.Type)
	if !ok {
		return fmt.Errorf(
			"cannot resolve type %q (field=%s, isTypeParam=%v)",
			f.Type.Name, f.Name, f.Type.IsTypeParam(),
		)
	}
	if err := b.processValueByType(resolved, f.Type, getPath, setPath, constPrefix); err != nil {
		return fmt.Errorf("field %s (type %s, path %s): %w", f.Name, f.Type.Name, getPath, err)
	}
	return nil
}

func (b *codeBuilder) resolveTypeRef(ref resolution.TypeRef) (resolution.Type, bool) {
	resolved, ok := ref.Resolve(b.table)
	if ok {
		return resolved, true
	}
	if ref.IsTypeParam() && ref.TypeParam != nil && ref.TypeParam.HasDefault() {
		return ref.TypeParam.Default.Resolve(b.table)
	}
	return resolution.Type{}, false
}

func (b *codeBuilder) processRecursiveStruct(
	typ resolution.Type, getPath, setPath, constPrefix string,
) error {
	ind := b.indent()
	goType, err := b.goTypeName(typ)
	if err != nil {
		return err
	}
	helperName := "marshal" + strings.ReplaceAll(goType, ".", "")
	unmarshalHelper := "un" + helperName

	b.marshalLines = append(b.marshalLines,
		ind+fmt.Sprintf("{ _sub, _se := %s(%s)", helperName, getPath),
		ind+"\tif _se != nil { return nil, _se }",
		ind+"\tbuf = binary.BigEndian.AppendUint32(buf, uint32(len(_sub)))",
		ind+"\tbuf = append(buf, _sub...) }",
	)
	b.unmarshalLines = append(b.unmarshalLines,
		ind+"{ _sLen := binary.BigEndian.Uint32(data[:4]); data = data[4:]",
		ind+fmt.Sprintf("\t_sv, _se := %s(data[:_sLen])", unmarshalHelper),
		ind+"\tif _se != nil { return r, _se }",
		ind+fmt.Sprintf("\t%s = _sv", setPath),
		ind+"\tdata = data[_sLen:] }",
	)

	idx := b.index
	b.index++
	b.consts = append(b.consts, wireConst{name: constPrefix, index: idx})
	b.estSize += 64

	helperKey := typ.QualifiedName
	if b.generatedHelpers[helperKey] {
		return nil
	}
	b.generatedHelpers[helperKey] = true

	sub := &codeBuilder{
		table:            b.table,
		repoRoot:         b.repoRoot,
		parentAlias:      b.parentAlias,
		parentPath:       b.parentPath,
		imports:          b.imports,
		processingTypes:  map[string]bool{helperKey: true},
		generatedHelpers: b.generatedHelpers,
	}
	innerFields := resolution.UnifiedFields(typ, b.table)
	if err := sub.processFields(innerFields, "s", "r", "Field"); err != nil {
		return fmt.Errorf("failed to generate recursive helper for %s: %w", typ.Name, err)
	}
	if sub.needsMath {
		b.needsMath = true
	}
	if sub.needsJSON {
		b.needsJSON = true
	}

	helper := fmt.Sprintf(`
func %s(s %s) ([]byte, error) {
	buf := make([]byte, 0, %d)
%s
	return buf, nil
}

func %s(data []byte) (%s, error) {
	var r %s
%s
	return r, nil
}`, helperName, goType, sub.estSize, strings.Join(sub.marshalLines, "\n"),
		unmarshalHelper, goType, goType, strings.Join(sub.unmarshalLines, "\n"))

	b.helperFuncs = append(b.helperFuncs, helper)
	b.helperFuncs = append(b.helperFuncs, sub.helperFuncs...)
	return nil
}

func (b *codeBuilder) processArray(
	f resolution.Field, getPath, setPath, constPrefix string,
) error {
	ind := b.indent()
	elemRef := f.Type.TypeArgs[0]
	elemType, ok := elemRef.Resolve(b.table)
	if !ok {
		return fmt.Errorf("cannot resolve array element type %s", elemRef.Name)
	}
	goType, err := b.resolveGoSliceElemType(elemType)
	if err != nil {
		return err
	}

	ev := b.nextVar("_e")
	iv := b.nextVar("_i")
	b.marshalLines = append(b.marshalLines,
		ind+fmt.Sprintf("buf = binary.BigEndian.AppendUint32(buf, uint32(len(%s)))", getPath),
		ind+fmt.Sprintf("for _, %s := range %s {", ev, getPath),
	)
	b.unmarshalLines = append(b.unmarshalLines,
		ind+"{ _n := binary.BigEndian.Uint32(data[:4]); data = data[4:]",
		ind+fmt.Sprintf("\t%s = make([]%s, _n)", setPath, goType),
		ind+fmt.Sprintf("\tfor %s := range %s {", iv, setPath),
	)

	b.depth++
	if err := b.processValueByType(elemType, elemRef, ev, setPath+"["+iv+"]", constPrefix+"Elem"); err != nil {
		return err
	}
	b.depth--

	b.marshalLines = append(b.marshalLines, ind+"}")
	b.unmarshalLines = append(b.unmarshalLines, ind+"\t}", ind+"}")
	return nil
}

func (b *codeBuilder) processMap(
	keyRef, valRef resolution.TypeRef,
	getPath, setPath, constPrefix string,
) error {
	ind := b.indent()
	keyType, ok := keyRef.Resolve(b.table)
	if !ok {
		return fmt.Errorf("cannot resolve map key type %s", keyRef.Name)
	}
	valType, ok := valRef.Resolve(b.table)
	if !ok {
		return fmt.Errorf("cannot resolve map value type %s", valRef.Name)
	}
	goKeyType, err := b.goTypeName(keyType)
	if err != nil {
		return err
	}
	goValType, err := b.goTypeName(valType)
	if err != nil {
		return err
	}

	idx := b.index
	b.index++
	b.consts = append(b.consts, wireConst{name: constPrefix, index: idx})
	b.estSize += 64

	mk := b.nextVar("_mk")
	mv := b.nextVar("_mv")
	mi := b.nextVar("_mi")
	b.marshalLines = append(b.marshalLines,
		ind+fmt.Sprintf("buf = binary.BigEndian.AppendUint32(buf, uint32(len(%s)))", getPath),
		ind+fmt.Sprintf("for %s, %s := range %s {", mk, mv, getPath),
	)
	b.unmarshalLines = append(b.unmarshalLines,
		ind+"{ _n := binary.BigEndian.Uint32(data[:4]); data = data[4:]",
		ind+fmt.Sprintf("\t%s = make(map[%s]%s, _n)", setPath, goKeyType, goValType),
		ind+fmt.Sprintf("\tfor %s := uint32(0); %s < _n; %s++ {", mi, mi, mi),
		ind+fmt.Sprintf("\t\tvar %s %s", mk, goKeyType),
		ind+fmt.Sprintf("\t\tvar %s %s", mv, goValType),
	)

	b.depth++
	if err := b.processValueByType(keyType, keyRef, mk, mk, constPrefix+"Key"); err != nil {
		return fmt.Errorf("map key: %w", err)
	}
	if err := b.processValueByType(valType, valRef, mv, mv, constPrefix+"Val"); err != nil {
		return fmt.Errorf("map value: %w", err)
	}
	b.depth--

	b.marshalLines = append(b.marshalLines, ind+"}")
	b.unmarshalLines = append(b.unmarshalLines,
		ind+fmt.Sprintf("\t\t%s[%s] = %s", setPath, mk, mv),
		ind+"\t}",
		ind+"}",
	)
	return nil
}

func (b *codeBuilder) resolveGoSliceElemType(typ resolution.Type) (string, error) {
	actual := typ
	for {
		var baseRef resolution.TypeRef
		switch form := actual.Form.(type) {
		case resolution.AliasForm:
			baseRef = form.Target
		case resolution.DistinctForm:
			baseRef = form.Base
		default:
			return b.goTypeName(actual)
		}
		target, ok := baseRef.Resolve(b.table)
		if !ok {
			return "", fmt.Errorf("cannot resolve type %s", baseRef.Name)
		}
		if bg, ok := target.Form.(resolution.BuiltinGenericForm); ok && bg.Name == "Array" && len(baseRef.TypeArgs) > 0 {
			innerElem, ok := baseRef.TypeArgs[0].Resolve(b.table)
			if !ok {
				return "", fmt.Errorf("cannot resolve array element %s", baseRef.TypeArgs[0].Name)
			}
			innerGoType, err := b.resolveGoSliceElemType(innerElem)
			if err != nil {
				return "", err
			}
			return "[]" + innerGoType, nil
		}
		actual = target
	}
}

func (b *codeBuilder) processLeaf(
	typ resolution.Type, getPath, setPath, constName string,
) error {
	primName, goTypeCast, err := b.resolveLeaf(typ)
	if err != nil {
		return err
	}

	idx := b.index
	b.index++
	b.consts = append(b.consts, wireConst{name: constName, index: idx})
	ind := b.indent()

	switch primName {
	case "string":
		b.estSize += 32
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("buf = binary.BigEndian.AppendUint32(buf, uint32(len(%s)))", getPath),
			ind+fmt.Sprintf("buf = append(buf, %s...)", getPath),
		)
		if goTypeCast != "" {
			b.unmarshalLines = append(b.unmarshalLines,
				ind+"{ _n := binary.BigEndian.Uint32(data[:4]); data = data[4:]",
				ind+fmt.Sprintf("\t%s = %s(data[:_n]); data = data[_n:] }", setPath, goTypeCast),
			)
		} else {
			b.unmarshalLines = append(b.unmarshalLines,
				ind+"{ _n := binary.BigEndian.Uint32(data[:4]); data = data[4:]",
				ind+fmt.Sprintf("\t%s = string(data[:_n]); data = data[_n:] }", setPath),
			)
		}

	case "uuid":
		b.estSize += 16
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("buf = append(buf, %s[:]...)", getPath))
		b.unmarshalLines = append(b.unmarshalLines,
			ind+fmt.Sprintf("copy(%s[:], data[:16])", setPath),
			ind+"data = data[16:]",
		)

	case "json", "any":
		b.estSize += 64
		b.needsJSON = true
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("{ _jb, _je := json.Marshal(%s)", getPath),
			ind+"\tif _je != nil { return nil, _je }",
			ind+"\tbuf = binary.BigEndian.AppendUint32(buf, uint32(len(_jb)))",
			ind+"\tbuf = append(buf, _jb...) }",
		)
		b.unmarshalLines = append(b.unmarshalLines,
			ind+"{ _n := binary.BigEndian.Uint32(data[:4]); data = data[4:]",
			ind+fmt.Sprintf("\tif err := json.Unmarshal(data[:_n], &%s); err != nil { return r, err }", setPath),
			ind+"\tdata = data[_n:] }",
		)

	case "bytes":
		b.estSize += 32
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("buf = binary.BigEndian.AppendUint32(buf, uint32(len(%s)))", getPath),
			ind+fmt.Sprintf("buf = append(buf, %s...)", getPath),
		)
		b.unmarshalLines = append(b.unmarshalLines,
			ind+"{ _n := binary.BigEndian.Uint32(data[:4]); data = data[4:]",
			ind+fmt.Sprintf("\t%s = append([]byte(nil), data[:_n]...); data = data[_n:] }", setPath),
		)

	case "bool":
		b.estSize += 1
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("if %s { buf = append(buf, 1) } else { buf = append(buf, 0) }", getPath))
		b.unmarshalLines = append(b.unmarshalLines,
			ind+fmt.Sprintf("%s = data[0] != 0", setPath),
			ind+"data = data[1:]",
		)

	case "int8":
		b.estSize += 1
		cast := "int8"
		if goTypeCast != "" {
			cast = goTypeCast
		}
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("buf = append(buf, byte(%s))", getPath))
		b.unmarshalLines = append(b.unmarshalLines,
			ind+fmt.Sprintf("%s = %s(data[0])", setPath, cast),
			ind+"data = data[1:]",
		)

	case "int16":
		b.estSize += 2
		cast := "int16"
		if goTypeCast != "" {
			cast = goTypeCast
		}
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("buf = binary.BigEndian.AppendUint16(buf, uint16(%s))", getPath))
		b.unmarshalLines = append(b.unmarshalLines,
			ind+fmt.Sprintf("%s = %s(binary.BigEndian.Uint16(data[:2]))", setPath, cast),
			ind+"data = data[2:]",
		)

	case "int32":
		b.estSize += 4
		cast := "int32"
		if goTypeCast != "" {
			cast = goTypeCast
		}
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("buf = binary.BigEndian.AppendUint32(buf, uint32(%s))", getPath))
		b.unmarshalLines = append(b.unmarshalLines,
			ind+fmt.Sprintf("%s = %s(binary.BigEndian.Uint32(data[:4]))", setPath, cast),
			ind+"data = data[4:]",
		)

	case "int64":
		b.estSize += 8
		cast := "int64"
		if goTypeCast != "" {
			cast = goTypeCast
		}
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("buf = binary.BigEndian.AppendUint64(buf, uint64(%s))", getPath))
		b.unmarshalLines = append(b.unmarshalLines,
			ind+fmt.Sprintf("%s = %s(binary.BigEndian.Uint64(data[:8]))", setPath, cast),
			ind+"data = data[8:]",
		)

	case "uint8":
		b.estSize += 1
		cast := "uint8"
		if goTypeCast != "" {
			cast = goTypeCast
		}
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("buf = append(buf, byte(%s))", getPath))
		b.unmarshalLines = append(b.unmarshalLines,
			ind+fmt.Sprintf("%s = %s(data[0])", setPath, cast),
			ind+"data = data[1:]",
		)

	case "uint12", "uint16":
		b.estSize += 2
		cast := "uint16"
		if goTypeCast != "" {
			cast = goTypeCast
		}
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("buf = binary.BigEndian.AppendUint16(buf, uint16(%s))", getPath))
		b.unmarshalLines = append(b.unmarshalLines,
			ind+fmt.Sprintf("%s = %s(binary.BigEndian.Uint16(data[:2]))", setPath, cast),
			ind+"data = data[2:]",
		)

	case "uint20", "uint32":
		b.estSize += 4
		cast := "uint32"
		if goTypeCast != "" {
			cast = goTypeCast
		}
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("buf = binary.BigEndian.AppendUint32(buf, uint32(%s))", getPath))
		b.unmarshalLines = append(b.unmarshalLines,
			ind+fmt.Sprintf("%s = %s(binary.BigEndian.Uint32(data[:4]))", setPath, cast),
			ind+"data = data[4:]",
		)

	case "uint64":
		b.estSize += 8
		cast := "uint64"
		if goTypeCast != "" {
			cast = goTypeCast
		}
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("buf = binary.BigEndian.AppendUint64(buf, uint64(%s))", getPath))
		b.unmarshalLines = append(b.unmarshalLines,
			ind+fmt.Sprintf("%s = %s(binary.BigEndian.Uint64(data[:8]))", setPath, cast),
			ind+"data = data[8:]",
		)

	case "float32":
		b.estSize += 4
		b.needsMath = true
		cast := "float32"
		if goTypeCast != "" {
			cast = goTypeCast
		}
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("buf = binary.BigEndian.AppendUint32(buf, math.Float32bits(float32(%s)))", getPath))
		b.unmarshalLines = append(b.unmarshalLines,
			ind+fmt.Sprintf("%s = %s(math.Float32frombits(binary.BigEndian.Uint32(data[:4])))", setPath, cast),
			ind+"data = data[4:]",
		)

	case "float64":
		b.estSize += 8
		b.needsMath = true
		cast := "float64"
		if goTypeCast != "" {
			cast = goTypeCast
		}
		b.marshalLines = append(b.marshalLines,
			ind+fmt.Sprintf("buf = binary.BigEndian.AppendUint64(buf, math.Float64bits(float64(%s)))", getPath))
		b.unmarshalLines = append(b.unmarshalLines,
			ind+fmt.Sprintf("%s = %s(math.Float64frombits(binary.BigEndian.Uint64(data[:8])))", setPath, cast),
			ind+"data = data[8:]",
		)

	default:
		return fmt.Errorf("unsupported primitive type: %s", primName)
	}
	return nil
}

func (b *codeBuilder) resolveLeaf(typ resolution.Type) (primName, goTypeCast string, err error) {
	switch form := typ.Form.(type) {
	case resolution.PrimitiveForm:
		return form.Name, "", nil
	case resolution.DistinctForm:
		base, ok := form.Base.Resolve(b.table)
		if !ok {
			return "", "", fmt.Errorf("cannot resolve distinct base %s", form.Base.Name)
		}
		basePrim, _, err := b.resolveLeaf(base)
		if err != nil {
			return "", "", err
		}
		goType, err := b.goTypeName(typ)
		if err != nil {
			return "", "", err
		}
		return basePrim, goType, nil
	case resolution.EnumForm:
		if form.IsIntEnum {
			goType, err := b.goTypeName(typ)
			if err != nil {
				return "", "", err
			}
			return "int64", goType, nil
		}
		goType, err := b.goTypeName(typ)
		if err != nil {
			return "", "", err
		}
		return "string", goType, nil
	case resolution.AliasForm:
		target, ok := form.Target.Resolve(b.table)
		if !ok {
			return "", "", fmt.Errorf("cannot resolve alias target %s", form.Target.Name)
		}
		return b.resolveLeaf(target)
	default:
		return "", "", fmt.Errorf("unsupported type form for leaf: %T (%s)", form, typ.QualifiedName)
	}
}

func (b *codeBuilder) goTypeName(typ resolution.Type) (string, error) {
	if prim, ok := typ.Form.(resolution.PrimitiveForm); ok {
		switch prim.Name {
		case "string":
			return "string", nil
		case "bool":
			return "bool", nil
		case "int8":
			return "int8", nil
		case "int16":
			return "int16", nil
		case "int32":
			return "int32", nil
		case "int64":
			return "int64", nil
		case "uint8":
			return "uint8", nil
		case "uint12", "uint16":
			return "uint16", nil
		case "uint20", "uint32":
			return "uint32", nil
		case "uint64":
			return "uint64", nil
		case "float32":
			return "float32", nil
		case "float64":
			return "float64", nil
		case "uuid":
			b.imports["github.com/google/uuid"] = "uuid"
			return "uuid.UUID", nil
		case "bytes":
			return "[]byte", nil
		case "json", "any":
			return "interface{}", nil
		default:
			return "", fmt.Errorf("unsupported primitive type for goTypeName: %s", prim.Name)
		}
	}
	goName := getGoName(typ)
	if goName == "" {
		goName = naming.ToPascalCase(typ.Name)
	}
	goPath := output.GetPath(typ, "go")
	if goPath == "" || goPath == b.parentPath {
		return b.parentAlias + "." + goName, nil
	}
	importPath, err := resolveGoImportPath(goPath, b.repoRoot)
	if err != nil {
		return "", err
	}
	// Check if this import path already has an alias
	if existingAlias, ok := b.imports[importPath]; ok {
		return existingAlias + "." + goName, nil
	}
	alias := naming.DerivePackageAlias(goPath, "pb")
	// Disambiguate alias if it collides with parent alias or existing imports
	if alias == b.parentAlias || b.aliasUsed(alias) {
		parent := filepath.Base(filepath.Dir(goPath))
		alias = parent + alias
	}
	for b.aliasUsed(alias) {
		alias = "_" + alias
	}
	b.imports[importPath] = alias
	return alias + "." + goName, nil
}

func (b *codeBuilder) aliasUsed(alias string) bool {
	for _, a := range b.imports {
		if a == alias {
			return true
		}
	}
	return false
}

func resolveGoImportPath(outputPath, repoRoot string) (string, error) {
	if repoRoot == "" {
		return "github.com/synnaxlabs/synnax/" + outputPath, nil
	}
	absPath := filepath.Join(repoRoot, outputPath)
	dir := absPath
	for {
		modPath := filepath.Join(dir, "go.mod")
		if fileExists(modPath) {
			moduleName, err := parseModuleName(modPath)
			if err != nil {
				return "", errors.Wrapf(err, "failed to parse go.mod at %s", modPath)
			}
			relPath, err := filepath.Rel(dir, absPath)
			if err != nil {
				return "", errors.Wrapf(err, "failed to compute relative path")
			}
			if relPath == "." {
				return moduleName, nil
			}
			return moduleName + "/" + filepath.ToSlash(relPath), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "github.com/synnaxlabs/synnax/" + outputPath, nil
}

func parseModuleName(modPath string) (string, error) {
	file, err := os.Open(modPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1], nil
			}
		}
	}
	return "", errors.New("module name not found in go.mod")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

const codecTemplate = `// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
	"context"
	"encoding/binary"
{{if .NeedsMath}}	"math"
{{end}}{{if .NeedsJSON}}	"encoding/json"
{{end}}
	"github.com/synnaxlabs/x/gorp"

	{{.ParentAlias}} "{{.ParentImportPath}}"
{{range $path, $alias := .ExtraImports}}	{{$alias}} "{{$path}}"
{{end}})

var _ = binary.BigEndian
{{range .Codecs}}
const (
{{.Constants}}
)

type {{lowerFirst .GoName}}Codec struct{}

func ({{lowerFirst .GoName}}Codec) Marshal(
	_ context.Context,
	s {{.ParentAlias}}.{{.GoName}},
) ([]byte, error) {
	buf := make([]byte, 0, {{.EstSize}})
{{.MarshalBody}}
	return buf, nil
}

func ({{lowerFirst .GoName}}Codec) Unmarshal(
	_ context.Context,
	data []byte,
) ({{.ParentAlias}}.{{.GoName}}, error) {
	var r {{.ParentAlias}}.{{.GoName}}
{{.UnmarshalBody}}
	return r, nil
}

var {{.GoName}}Codec gorp.Codec[{{.ParentAlias}}.{{.GoName}}] = {{lowerFirst .GoName}}Codec{}
{{if .HelperFuncs}}{{.HelperFuncs}}{{end}}
{{end}}`
