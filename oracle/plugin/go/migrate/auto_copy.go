// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// auto_copy.go generates migrate_auto.gen.go files containing autoMigrateX
// functions that convert between frozen and live type versions.
package migrate

import (
	"bytes"
	"fmt"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/schemadiff"
	"github.com/synnaxlabs/oracle/plugin/go/internal/typemap"
	"github.com/synnaxlabs/oracle/plugin/gomod"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/oracle/versions"
	"github.com/synnaxlabs/x/set"
)

// generateAutoCopy generates a migrate.gen.go file for the given types. mapping
// relates the two versions' surfaces.
func generateAutoCopy(
	pkg, outputPath, repoRoot string,
	types []resolution.Type,
	diff map[string]schemadiff.TypeDiff,
	oldTable, newTable *resolution.Table,
	mapping chainMapping,
	wrappers map[string]string,
) ([]byte, error) {
	newCollector := func(aliases map[string]string) *collector {
		return &collector{
			pkg: pkg, outputPath: outputPath, repoRoot: repoRoot,
			diff: diff, oldTable: oldTable, newTable: newTable,
			mapping: mapping,
			imports: make(map[string]importEntry), generated: make(set.Set[string]),
			wrappers: wrappers, aliases: aliases,
		}
	}
	data := newCollector(nil).collect(types)
	if len(data.Funcs) == 0 {
		return nil, nil
	}
	paths := make([]string, len(data.Imports))
	for i, imp := range data.Imports {
		paths[i] = imp.Path
	}
	current := gomod.ResolveImportPath(outputPath, repoRoot, gomod.DefaultModulePrefix)
	if aliases := naming.VersionAliases(paths, current); len(aliases) > 0 {
		data = newCollector(aliases).collect(types)
	}
	var buf bytes.Buffer
	if err := autoCopyTmpl.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const errorsImportPath = "github.com/synnaxlabs/x/errors"

// --- Data types ---

type fileData struct {
	Package string
	Imports []importEntry
	Funcs   []funcData
}

type importEntry struct {
	Alias string
	Path  string
}

// NeedsAlias reports whether the import renders with an explicit alias. The alias
// always qualifies references in the body; it reaches the import line only when it
// differs from the name the path is already assumed to bind.
func (i importEntry) NeedsAlias() bool {
	return i.Alias != naming.AssumedImportName(i.Path)
}

type funcData struct {
	GoName         string
	TypeParamsDecl string // "" for non-generic, "[Details any]" for generic
	OldTypeName    string
	NewTypeName    string
	UsesCtx        bool
	ZeroValue      string
	Kind           string // "struct", "slice", "cast", "union"
	Preamble       []step
	Fields         []field
	SliceElemExpr  string
	SliceOldElem   string
	SliceNewElem   string
	SliceHasErr    bool
	// Cases holds one conversion per variant a "union" function carries across.
	Cases []unionCase
	// recv is the identifier the function reads old fields through. No preamble
	// variable may take it.
	recv string
}

// unionCase converts one variant of a union. Its preamble and fields read the old
// variant through the bound variable of the generated type switch.
type unionCase struct {
	OldType   string
	NewType   string
	ZeroValue string
	Preamble  []step
	Fields    []field
}

type step struct {
	// Kind is one of "migrate", "optionalMigrate", "optionalCast",
	// "sliceMigrate", "sliceCast", or "mapMigrate".
	Kind     string
	VarName  string
	Accessor string
	Call     string
	Type     string
	ElemExpr string
	OldElem  string
	NewElem  string
}

type field struct {
	Name  string
	Value string
}

type classification struct {
	needsPreamble bool
	inline        string
	step          step
	usesCtx       bool
	// omit leaves the field to the hand-written migration: no copy of it
	// type-checks.
	omit bool
}

// --- Template ---

var autoCopyTmpl = template.Must(
	template.New("autoCopy").Parse(`{{define "preamble"}}
{{- $zero := .ZeroValue}}
{{- range .Preamble}}
{{- if eq .Kind "migrate"}}
	{{.VarName}}, err := {{.Call}}
	if err != nil {
		return {{$zero}}, err
	}
{{- else if eq .Kind "optionalMigrate"}}
	var {{.VarName}} *{{.Type}}
	if {{.Accessor}} != nil {
		v, err := {{.Call}}
		if err != nil {
			return {{$zero}}, err
		}
		{{.VarName}} = &v
	}
{{- else if eq .Kind "optionalCast"}}
	var {{.VarName}} *{{.Type}}
	if {{.Accessor}} != nil {
		v := {{.Type}}(*{{.Accessor}})
		{{.VarName}} = &v
	}
{{- else if eq .Kind "sliceMigrate"}}
	{{.VarName}}, err := lo.MapErr({{.Accessor}}, func(v {{.OldElem}}, _ int) ({{.NewElem}}, error) {
		return {{.ElemExpr}}
	})
	if err != nil {
		return {{$zero}}, err
	}
{{- else if eq .Kind "mapMigrate"}}
	{{.VarName}} := make({{.Type}}, len({{.Accessor}}))
	for k, v := range {{.Accessor}} {
		nv, err := {{.ElemExpr}}
		if err != nil {
			return {{$zero}}, err
		}
		{{.VarName}}[k] = nv
	}
{{- else if eq .Kind "sliceCast"}}
	{{.VarName}} := lo.Map({{.Accessor}}, func(v {{.OldElem}}, _ int) {{.NewElem}} {
		return {{.ElemExpr}}
	})
{{- end}}
{{- end}}
{{- end}}
{{- define "fields"}}
{{- range .Fields}}
		{{.Name}}: {{.Value}},
{{- end}}
{{- end}}
{{- /* file */ -}}
// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import (
	"context"
{{- if .Imports}}
{{end}}
{{- range .Imports}}
	{{if .NeedsAlias}}{{.Alias}} {{end}}"{{.Path}}"
{{- end}}
)
{{range $fn := .Funcs}}
{{- if eq $fn.Kind "cast"}}
func autoMigrate{{$fn.GoName}}{{$fn.TypeParamsDecl}}({{- if $fn.UsesCtx}}ctx{{else}}_{{end}} context.Context, old {{$fn.OldTypeName}}) ({{$fn.NewTypeName}}, error) {
	return {{$fn.NewTypeName}}(old), nil
}
{{else if eq $fn.Kind "slice"}}
func autoMigrate{{$fn.GoName}}{{$fn.TypeParamsDecl}}({{- if $fn.UsesCtx}}ctx{{else}}_{{end}} context.Context, old {{$fn.OldTypeName}}) ({{$fn.NewTypeName}}, error) {
{{- if $fn.SliceHasErr}}
	return lo.MapErr(old, func(v {{$fn.SliceOldElem}}, _ int) ({{$fn.SliceNewElem}}, error) {
		return {{$fn.SliceElemExpr}}
	})
{{- else}}
	return lo.Map(old, func(v {{$fn.SliceOldElem}}, _ int) {{$fn.SliceNewElem}} {
		return {{$fn.SliceElemExpr}}
	}), nil
{{- end}}
}
{{else if eq $fn.Kind "struct"}}
func autoMigrate{{$fn.GoName}}{{$fn.TypeParamsDecl}}({{- if $fn.UsesCtx}}ctx{{else}}_{{end}} context.Context, old {{$fn.OldTypeName}}) ({{$fn.NewTypeName}}, error) {
{{- template "preamble" $fn}}
	return {{$fn.NewTypeName}}{
{{- template "fields" $fn}}
	}, nil
}
{{else if eq $fn.Kind "union"}}
func autoMigrate{{$fn.GoName}}({{- if $fn.UsesCtx}}ctx{{else}}_{{end}} context.Context, old {{$fn.OldTypeName}}) ({{$fn.NewTypeName}}, error) {
	switch variant := old.Variant.(type) {
	case nil:
		return {{$fn.ZeroValue}}, nil
{{- range $fn.Cases}}
	case {{.OldType}}:
{{- template "preamble" .}}
		return {{$fn.NewTypeName}}{Variant: {{.NewType}}{
{{- template "fields" .}}
		}}, nil
{{- end}}
	default:
		return {{$fn.ZeroValue}}, errors.Newf("unknown {{$fn.GoName}} variant %T", variant)
	}
}
{{end}}
{{- end}}`),
)

// --- Collector ---

type collector struct {
	pkg, outputPath, repoRoot string
	diff                      map[string]schemadiff.TypeDiff
	oldTable, newTable        *resolution.Table
	// mapping relates a qualified name in oldTable to the incoming version's
	// surface.
	mapping   chainMapping
	imports   map[string]importEntry
	generated set.Set[string]
	pending   []resolution.Type
	funcs     []funcData
	// wrappers maps qualified type names to their developer-wrapper names,
	// deciding exported vs unexported routing (see wrapperNames).
	wrappers map[string]string
	// usesLo marks that an emitted function body calls a samber/lo helper.
	usesLo bool
	// aliases forces the alias of each import path it holds.
	aliases map[string]string
}

func (c *collector) collect(types []resolution.Type) fileData {
	for _, typ := range types {
		td, hasDiff := c.diff[typ.QualifiedName]
		if !hasDiff || td.Kind == schemadiff.TypeUnchanged {
			continue
		}
		c.ensureFunc(typ)
	}
	for len(c.pending) > 0 {
		typ := c.pending[0]
		c.pending = c.pending[1:]
		c.ensureFunc(typ)
	}
	if c.usesLo {
		c.imports["github.com/samber/lo"] = importEntry{Path: "github.com/samber/lo"}
	}
	// context renders as its own standard-library group in the template;
	// every function signature includes context.Context.
	imps := make([]importEntry, 0, len(c.imports))
	for _, imp := range c.imports {
		imps = append(imps, imp)
	}
	sort.Slice(imps, func(i, j int) bool { return imps[i].Path < imps[j].Path })
	return fileData{Package: c.pkg, Imports: imps, Funcs: c.funcs}
}

func (c *collector) ensureFunc(typ resolution.Type) {
	if c.generated.Contains(typ.QualifiedName) {
		return
	}
	c.generated.Add(typ.QualifiedName)
	switch form := typ.Form.(type) {
	case resolution.StructForm:
		c.funcs = append(c.funcs, c.structFunc(typ, form))
	case resolution.AliasForm:
		c.funcs = append(c.funcs, c.aliasFunc(typ, form))
	case resolution.DistinctForm:
		c.funcs = append(c.funcs, c.distinctFunc(typ, form))
	case resolution.UnionForm:
		c.funcs = append(c.funcs, c.unionFunc(typ, form))
	default:
		c.funcs = append(c.funcs, c.castFunc(typ))
	}
}

func (c *collector) structFunc(typ resolution.Type, sf resolution.StructForm) funcData {
	newType, _ := c.newCounterpart(typ)
	newSF, _ := newType.Form.(resolution.StructForm)
	// Substitute defaulted type params (e.g., V extends Variant = Variant) so
	// fields referencing them resolve to concrete types. The types plugin
	// performs the same substitution for the struct definition; matching it
	// here keeps the migration's per-field classification aligned with the
	// generated struct shape.
	return decorateWithTypeParams(c.structFuncFromForms(
		naming.GetGoName(typ),
		c.resolveTypeName(typ, c.oldTable),
		c.resolveNewTypeName(typ),
		"old",
		substituteTypeParams(sf, nil),
		substituteTypeParams(newSF, nil),
	), sf.TypeParams)
}

func (c *collector) aliasFunc(typ resolution.Type, form resolution.AliasForm) funcData {
	goName := naming.GetGoName(typ)
	oldName := c.resolveTypeName(typ, c.oldTable)
	newName := c.resolveNewTypeName(typ)
	if elemRef, isArr := arrayBaseRef(typ, c.oldTable); isArr {
		return decorateWithTypeParams(
			c.sliceFunc(typ, goName, oldName, newName, elemRef), form.TypeParams,
		)
	}
	targetResolved, targetRef, ok := resolution.Dealias(form.Target, c.oldTable)
	if !ok {
		targetRef = form.Target
		targetResolved, ok = c.oldTable.Lookup(typ.Namespace, form.Target.Name)
	}
	if ok {
		if sf, isStruct := targetResolved.Form.(resolution.StructForm); isStruct {
			newTarget, _ := c.newCounterpart(targetResolved)
			newSF, _ := newTarget.Form.(resolution.StructForm)
			oldSF := substituteTypeParams(sf, targetRef.TypeArgs)
			if newAlias, ok := c.newCounterpart(typ); ok {
				if af, ok := newAlias.Form.(resolution.AliasForm); ok {
					if _, newTargetRef, ok := resolution.Dealias(
						af.Target, c.newTable,
					); ok {
						newSF = substituteTypeParams(newSF, newTargetRef.TypeArgs)
					}
				}
			}
			return decorateWithTypeParams(
				c.structFuncFromForms(goName, oldName, newName, "old", oldSF, newSF),
				form.TypeParams,
			)
		}
		// A Go alias is its target's type, so the variants keep the union's names.
		if uf, isUnion := targetResolved.Form.(resolution.UnionForm); isUnion {
			fn := c.unionFunc(targetResolved, uf)
			fn.GoName, fn.OldTypeName, fn.NewTypeName = goName, oldName, newName
			return fn
		}
	}
	return decorateWithTypeParams(c.castFunc(typ), form.TypeParams)
}

func (c *collector) distinctFunc(
	typ resolution.Type,
	form resolution.DistinctForm,
) funcData {
	goName := naming.GetGoName(typ)
	oldName := c.resolveTypeName(typ, c.oldTable)
	newName := c.resolveNewTypeName(typ)
	if elemRef, isArr := arrayBaseRef(typ, c.oldTable); isArr {
		return decorateWithTypeParams(
			c.sliceFunc(typ, goName, oldName, newName, elemRef), form.TypeParams,
		)
	}
	return decorateWithTypeParams(c.castFunc(typ), form.TypeParams)
}

func (c *collector) castFunc(typ resolution.Type) funcData {
	return funcData{
		GoName:      naming.GetGoName(typ),
		OldTypeName: c.resolveTypeName(typ, c.oldTable),
		NewTypeName: c.resolveNewTypeName(typ),
		Kind:        "cast",
	}
}

// structFuncFromForms builds the conversion from oldSF to newSF, reading each old field
// through recv.
func (c *collector) structFuncFromForms(
	goName, oldTypeName, newTypeName, recv string,
	oldSF, newSF resolution.StructForm,
) funcData {
	fn := funcData{
		GoName: goName, OldTypeName: oldTypeName, NewTypeName: newTypeName,
		ZeroValue: newTypeName + "{}", Kind: "struct", recv: recv,
	}
	newExtends := make(set.Set[string], len(newSF.Extends))
	for _, ext := range newSF.Extends {
		if newParent, ok := ext.Resolve(c.newTable); ok {
			newExtends.Add(naming.GetGoName(newParent))
		}
	}
	for _, ext := range oldSF.Extends {
		oldParent, ok := c.resolveRef(ext)
		if !ok {
			continue
		}
		name := naming.GetGoName(oldParent)
		if !newExtends.Contains(name) {
			continue
		}
		c.addField(&fn, ext, recv+"."+name, name, false)
	}
	for _, oldField := range oldSF.Fields {
		// A field omitted on either side never reaches the wire: a decoded old
		// entry carries the zero value, and the encoder drops the new field.
		// Copying would also couple versions whose memory-only types diverged.
		if domain.GetStringFromField(oldField, "go", "marshal") == "omit" {
			continue
		}
		newField := findField(newSF.Fields, oldField.Name)
		if newField == nil {
			continue
		}
		if domain.GetStringFromField(*newField, "go", "marshal") == "omit" {
			continue
		}
		// A field the two versions shape differently — an optional appearing, a
		// union replacing a struct, a declaration replacing a raw record, a type
		// moving to another resource, a leaf primitive Go cannot cast across — has
		// no copy that type-checks; the hand-written migration carries it.
		if oldField.Optional != newField.Optional ||
			c.unionMismatch(oldField.Type, newField.Type) ||
			c.declarationMismatch(oldField.Type, newField.Type) ||
			c.castMismatch(oldField.Type, newField.Type) {
			continue
		}
		name := naming.GetFieldName(oldField)
		c.addField(&fn, oldField.Type, recv+"."+name, name, oldField.Optional)
	}
	return fn
}

// unionFunc converts a union variant by variant. A variant the incoming version drops
// or reshapes between inline and named has no case, so the generated default returns
// an error naming it until the hand-written migration handles it.
func (c *collector) unionFunc(
	typ resolution.Type,
	oldUF resolution.UnionForm,
) funcData {
	goName := naming.GetGoName(typ)
	oldName := c.resolveTypeName(typ, c.oldTable)
	newName := c.resolveNewTypeName(typ)
	fn := funcData{
		GoName: goName, OldTypeName: oldName, NewTypeName: newName,
		ZeroValue: newName + "{}", Kind: "union",
	}
	c.imports[errorsImportPath] = importEntry{Path: errorsImportPath}
	newType, _ := c.newCounterpart(typ)
	newUF, _ := newType.Form.(resolution.UnionForm)
	oldPrefix := strings.TrimSuffix(oldName, goName)
	newPrefix := strings.TrimSuffix(newName, goName)
	for _, ov := range oldUF.Variants {
		nv, ok := findVariant(newUF.Variants, ov.Name)
		if !ok || nv.Inline != ov.Inline {
			continue
		}
		vf := c.structFuncFromForms(
			"", "", "", "variant",
			variantForm(oldUF, ov, c.oldTable),
			variantForm(newUF, nv, c.newTable),
		)
		fn.UsesCtx = fn.UsesCtx || vf.UsesCtx
		variantName := casing.VariantTypeName(goName, ov.Name)
		fn.Cases = append(fn.Cases, unionCase{
			OldType:   oldPrefix + variantName,
			NewType:   newPrefix + variantName,
			ZeroValue: fn.ZeroValue,
			Preamble:  vf.Preamble,
			Fields:    vf.Fields,
		})
	}
	return fn
}

// variantForm returns the struct shape a union variant renders as in Go: the union's
// bases and the named payload as embeds, or the inherited bases and own fields of an
// inline payload.
func variantForm(
	uf resolution.UnionForm,
	v resolution.UnionVariant,
	table *resolution.Table,
) resolution.StructForm {
	if !v.Inline {
		return resolution.StructForm{Extends: append(slices.Clone(uf.Extends), v.Type)}
	}
	payload, ok := v.Type.Resolve(table)
	if !ok {
		return resolution.StructForm{}
	}
	pform, _ := payload.Form.(resolution.StructForm)
	inherited, declared := resolver.VariantBases(uf, v, "go", table)
	defaultOnly := resolver.DefaultOnlyOverrides(inherited, pform.Fields, table)
	fields := slices.Clone(declared)
	for _, f := range pform.Fields {
		if !defaultOnly.Contains(f.Name) {
			fields = append(fields, f)
		}
	}
	return resolution.StructForm{Extends: inherited, Fields: fields}
}

func findVariant(
	variants []resolution.UnionVariant,
	name string,
) (resolution.UnionVariant, bool) {
	for _, v := range variants {
		if v.Name == name {
			return v, true
		}
	}
	return resolution.UnionVariant{}, false
}

// unionMismatch reports whether a field's old and new types disagree on
// union-ness at any position, which auto-copy cannot bridge; such fields are
// left to the hand-written migration.
func (c *collector) unionMismatch(oldRef, newRef resolution.TypeRef) bool {
	if isUnionIn(oldRef, c.oldTable) != isUnionIn(newRef, c.newTable) {
		return true
	}
	n := min(len(oldRef.TypeArgs), len(newRef.TypeArgs))
	for i := range n {
		if c.unionMismatch(oldRef.TypeArgs[i], newRef.TypeArgs[i]) {
			return true
		}
	}
	return false
}

// declarationMismatch reports whether a field's old and new types name different
// declarations: a raw record becoming a named struct, a rename, or a type moving to
// another resource. Only the declaration a version resolves the field to can carry the
// value, so a mismatch leaves the field to the hand-written migration.
func (c *collector) declarationMismatch(oldRef, newRef resolution.TypeRef) bool {
	oldResolved, oldOk := oldRef.Resolve(c.oldTable)
	newResolved, newOk := newRef.Resolve(c.newTable)
	if !oldOk || !newOk {
		return false
	}
	if oldResolved.Name != newResolved.Name {
		return true
	}
	oldLive, _, oldVersioned := versions.ParseDepNS(oldResolved.Namespace)
	newLive, _, newVersioned := versions.ParseDepNS(newResolved.Namespace)
	return oldVersioned && newVersioned && oldLive != newLive
}

// castMismatch reports whether a field's old and new types resolve to leaf
// primitives a Go conversion cannot bridge, such as a numeric key becoming a
// UUID. Such fields are left to the hand-written migration.
func (c *collector) castMismatch(oldRef, newRef resolution.TypeRef) bool {
	oldLeaf, oldOk := leafPrimitive(oldRef, c.oldTable)
	newLeaf, newOk := leafPrimitive(newRef, c.newTable)
	if !oldOk || !newOk || oldLeaf == newLeaf {
		return false
	}
	return !resolution.IsNumberPrimitive(oldLeaf) ||
		!resolution.IsNumberPrimitive(newLeaf)
}

// leafPrimitive chases distinct bases and alias targets to the primitive a
// reference bottoms out on. Non-scalar forms report no leaf.
func leafPrimitive(ref resolution.TypeRef, table *resolution.Table) (string, bool) {
	if resolution.IsPrimitive(ref.Name) {
		return ref.Name, true
	}
	resolved, ok := ref.Resolve(table)
	if !ok {
		return "", false
	}
	switch form := resolved.Form.(type) {
	case resolution.PrimitiveForm:
		return form.Name, true
	case resolution.DistinctForm:
		return leafPrimitive(form.Base, table)
	case resolution.AliasForm:
		return leafPrimitive(form.Target, table)
	}
	return "", false
}

// isUnionIn reports whether a type reference resolves to a discriminated
// union within the given table.
func isUnionIn(ref resolution.TypeRef, table *resolution.Table) bool {
	resolved, _, ok := resolution.Dealias(ref, table)
	if !ok {
		return false
	}
	_, isUnion := resolved.Form.(resolution.UnionForm)
	return isUnion
}

func (c *collector) addField(
	fn *funcData,
	ref resolution.TypeRef,
	accessor, goName string,
	isOptional bool,
) {
	cls := c.classifyField(ref, accessor, goName, isOptional)
	if cls.omit {
		return
	}
	if cls.needsPreamble {
		if cls.step.VarName == fn.recv {
			cls.step.VarName += "Val"
		}
		fn.Preamble = append(fn.Preamble, cls.step)
		fn.Fields = append(fn.Fields, field{Name: goName, Value: cls.step.VarName})
		fn.UsesCtx = fn.UsesCtx || cls.usesCtx
	} else {
		fn.Fields = append(fn.Fields, field{Name: goName, Value: cls.inline})
	}
}

func (c *collector) sliceFunc(
	typ resolution.Type,
	goName, oldTypeName, newTypeName string,
	elemRef resolution.TypeRef,
) funcData {
	elemResolved, ok := c.resolveRef(elemRef)
	if !ok {
		elemResolved, ok = c.oldTable.Lookup(typ.Namespace, elemRef.Name)
	}
	if !ok {
		return funcData{
			GoName:      goName,
			OldTypeName: oldTypeName,
			NewTypeName: newTypeName,
			Kind:        "cast",
		}
	}
	if c.isSameType(elemResolved) {
		return funcData{
			GoName: goName, OldTypeName: oldTypeName, NewTypeName: newTypeName,
			Kind: "cast",
		}
	}
	td, hasDiff := c.diff[elemResolved.QualifiedName]
	needsMigration := hasDiff && td.Kind != schemadiff.TypeUnchanged
	if needsMigration || c.hasOracleDefinedFields(elemResolved) {
		c.usesLo = true
		return funcData{
			GoName: goName, OldTypeName: oldTypeName, NewTypeName: newTypeName,
			Kind: "slice", UsesCtx: true, SliceHasErr: true,
			SliceElemExpr: c.requireFunc(elemResolved) + "(ctx, v)",
			SliceOldElem:  c.resolveTypeName(elemResolved, c.oldTable),
			SliceNewElem:  c.resolveNewTypeName(elemResolved),
		}
	}
	if c.isOracleDefined(elemResolved) {
		c.usesLo = true
		return funcData{
			GoName: goName, OldTypeName: oldTypeName, NewTypeName: newTypeName,
			Kind: "slice", SliceElemExpr: c.resolveNewTypeName(elemResolved) + "(v)",
			SliceOldElem: c.resolveTypeName(elemResolved, c.oldTable),
			SliceNewElem: c.resolveNewTypeName(elemResolved),
		}
	}
	return funcData{
		GoName:      goName,
		OldTypeName: oldTypeName,
		NewTypeName: newTypeName,
		Kind:        "cast",
	}
}

// --- Field classification ---

func (c *collector) classifyField(
	ref resolution.TypeRef, accessor, goName string, isOptional bool,
) classification {
	resolved, ok := c.resolveRef(ref)
	if !ok {
		return classification{inline: accessor}
	}
	if bgf, ok := resolved.Form.(resolution.BuiltinGenericForm); ok {
		if bgf.Name == "Array" && len(ref.TypeArgs) > 0 {
			return c.classifySlice(ref.TypeArgs[0], accessor, goName)
		}
		if bgf.Name == "Map" && len(ref.TypeArgs) >= 2 {
			return c.classifyMap(
				ref.TypeArgs[0], ref.TypeArgs[1], accessor, goName,
			)
		}
		return classification{inline: accessor}
	}
	if elemRef, ok := arrayBaseRef(resolved, c.oldTable); ok {
		return c.classifySlice(elemRef, accessor, goName)
	}
	if !c.isOracleDefined(resolved) {
		return classification{inline: accessor}
	}
	// A type the incoming version aliases to its predecessor denotes the same Go
	// type, so the value assigns directly. Structural equality alone is not enough:
	// an add-only enum is wire-compatible yet still redeclared, giving the two
	// versions distinct Go types that only a conversion bridges.
	if c.isSameType(resolved) {
		return classification{inline: accessor}
	}
	varName := naming.LowerFirst(goName)
	helper := c.requireFunc(resolved)
	if isOptional {
		newType := c.resolveNewTypeName(resolved)
		return classification{needsPreamble: true, usesCtx: true, step: step{
			Kind: "optionalMigrate", VarName: varName, Accessor: accessor,
			Call: fmt.Sprintf("%s(ctx, *%s)", helper, accessor), Type: newType,
		}}
	}
	return classification{needsPreamble: true, usesCtx: true, step: step{
		Kind: "migrate", VarName: varName,
		Call: fmt.Sprintf("%s(ctx, %s)", helper, accessor),
	}}
}

func (c *collector) classifySlice(
	elemRef resolution.TypeRef, accessor, goName string,
) classification {
	elemResolved, ok := c.resolveRef(elemRef)
	if !ok {
		return classification{inline: accessor}
	}
	// Element types the incoming version aliases to their predecessors make the two
	// slice types identical, so the value assigns directly.
	if c.isSameType(elemResolved) {
		return classification{inline: accessor}
	}
	varName := naming.LowerFirst(goName)
	c.usesLo = true
	return classification{needsPreamble: true, usesCtx: true, step: step{
		Kind: "sliceMigrate", VarName: varName, Accessor: accessor,
		ElemExpr: c.requireFunc(elemResolved) + "(ctx, v)",
		OldElem:  c.resolveTypeName(elemResolved, c.oldTable),
		NewElem:  c.resolveNewTypeName(elemResolved),
	}}
}

// classifyMap emits a per-entry conversion when a map field's value type changed
// between versions. A map whose value type the incoming version cannot bridge
// mechanically (a non-primitive key alongside a changed value) is left to the
// hand-written migration.
func (c *collector) classifyMap(
	keyRef, valRef resolution.TypeRef, accessor, goName string,
) classification {
	valResolved, ok := c.resolveRef(valRef)
	if !ok || c.isSameType(valResolved) {
		return classification{inline: accessor}
	}
	keyGo, ok := typemap.PrimitiveGoType(keyRef.Name)
	if !ok {
		return classification{omit: true}
	}
	varName := naming.LowerFirst(goName)
	return classification{needsPreamble: true, usesCtx: true, step: step{
		Kind:     "mapMigrate",
		VarName:  varName,
		Accessor: accessor,
		Type:     "map[" + keyGo + "]" + c.resolveNewTypeName(valResolved),
		ElemExpr: c.requireFunc(valResolved) + "(ctx, v)",
	}}
}

// --- Type resolution helpers ---

// isSameType reports whether both versions denote typ with one Go type, making any
// migration a pure copy. A version that aliases the name back to the declaring version
// shares its Go type even though the two packages differ.
func (c *collector) isSameType(typ resolution.Type) bool {
	if same, known := c.mapping.sameGoType(typ.QualifiedName); known {
		return same
	}
	oldT, ok := c.oldTable.Get(typ.QualifiedName)
	if !ok {
		return false
	}
	newT, ok := c.newCounterpart(typ)
	if !ok {
		return false
	}
	return naming.GetGoName(oldT) == naming.GetGoName(newT) &&
		output.GetPath(oldT, "go") == output.GetPath(newT, "go")
}

func (c *collector) requireFunc(typ resolution.Type) string {
	goName := naming.GetGoName(typ)
	goPath := output.GetPath(typ, "go")
	isLocal := goPath == c.outputPath
	if !isLocal {
		if nt, ok := c.newCounterpart(typ); ok {
			isLocal = output.GetPath(nt, "go") == c.outputPath
		}
	}
	if !isLocal {
		td, ok := c.diff[typ.QualifiedName]
		isLocal = !ok || td.Kind == schemadiff.TypeUnchanged
	}
	if isLocal {
		if !c.generated.Contains(typ.QualifiedName) {
			c.pending = append(c.pending, typ)
		}
		// Every changed struct or union gets a MigrateX/migrateX wrapper in the
		// developer template; route local references through it so
		// hand-written fixups in the wrapper apply to nested occurrences too.
		td, hasDiff := c.diff[typ.QualifiedName]
		_, isStruct := typ.Form.(resolution.StructForm)
		_, isUnion := typ.Form.(resolution.UnionForm)
		if hasDiff && td.Kind == schemadiff.TypeChanged && (isStruct || isUnion) {
			if n, ok := c.wrappers[typ.QualifiedName]; ok {
				return n
			}
			return "Migrate" + goName
		}
		return "autoMigrate" + goName
	}
	// External packages expose only MigrateX (the developer template); the
	// auto-copy helpers are unexported. Both live in the dependency's incoming
	// version package, so the import targets the type's NEW path, not the
	// frozen one being read from.
	prefix := "Migrate"
	if nt, ok := c.newCounterpart(typ); ok {
		if p := output.GetPath(nt, "go"); p != "" {
			goPath = p
		}
	}
	imp := c.addImport(goPath)
	return imp.Alias + "." + prefix + goName
}

func (c *collector) resolveTypeName(
	typ resolution.Type,
	table *resolution.Table,
) string {
	t, ok := table.Get(typ.QualifiedName)
	if !ok {
		return naming.GetGoName(typ)
	}
	goName := naming.GetGoName(t)
	goPath := output.GetPath(t, "go")
	if goPath == "" || goPath == c.outputPath {
		return goName
	}
	return c.addImport(goPath).Alias + "." + goName
}

// newCounterpart returns typ's counterpart in the incoming version's table.
func (c *collector) newCounterpart(typ resolution.Type) (resolution.Type, bool) {
	return c.newTable.Get(c.mapping.counterpart(typ.QualifiedName))
}

func (c *collector) resolveNewTypeName(typ resolution.Type) string {
	nt, ok := c.newCounterpart(typ)
	if !ok {
		return naming.GetGoName(typ)
	}
	return c.resolveTypeName(nt, c.newTable)
}

func (c *collector) addImport(goPath string) importEntry {
	importPath := gomod.ResolveImportPath(goPath, c.repoRoot, gomod.DefaultModulePrefix)
	if imp, ok := c.imports[importPath]; ok {
		return imp
	}
	// A sibling version of the same resource imports under its bare directory
	// name, matching the hand-written migration convention (v0 "…/versions/v0").
	// Other resources import under their resource name, falling back to the
	// versioned form only when two versions of one resource meet in a file.
	alias, forced := c.aliases[importPath]
	if forced {
		imp := importEntry{Alias: alias, Path: importPath}
		c.imports[importPath] = imp
		return imp
	}
	alias = naming.DerivePackageAlias(goPath, c.pkg)
	if filepath.Dir(goPath) == filepath.Dir(c.outputPath) {
		alias = filepath.Base(goPath)
	} else {
		for _, existing := range c.imports {
			if existing.Alias == alias {
				alias = naming.DeriveVersionedAlias(goPath, c.pkg)
				break
			}
		}
	}
	imp := importEntry{Alias: alias, Path: importPath}
	c.imports[importPath] = imp
	return imp
}

func (c *collector) resolveRef(ref resolution.TypeRef) (resolution.Type, bool) {
	if resolved, ok := ref.Resolve(c.oldTable); ok {
		return resolved, true
	}
	if ref.Name == "" {
		return resolution.Type{}, false
	}
	for _, typ := range c.oldTable.Types {
		if typ.Name == ref.Name {
			return typ, true
		}
	}
	return resolution.Type{}, false
}

// --- Type introspection ---

func (c *collector) isOracleDefined(typ resolution.Type) bool {
	nt, ok := c.newCounterpart(typ)
	return ok && output.GetPath(nt, "go") != ""
}

func (c *collector) hasOracleDefinedFields(typ resolution.Type) bool {
	sf, ok := typ.Form.(resolution.StructForm)
	if !ok {
		return c.isStructLike(typ)
	}
	for _, f := range sf.Fields {
		if c.refHasOracleType(f.Type) {
			return true
		}
	}
	return slices.ContainsFunc(sf.Extends, c.refHasOracleType)
}

func (c *collector) refHasOracleType(ref resolution.TypeRef) bool {
	if slices.ContainsFunc(ref.TypeArgs, c.refHasOracleType) {
		return true
	}
	resolved, ok := ref.Resolve(c.oldTable)
	if !ok {
		return false
	}
	return c.isOracleDefined(resolved)
}

func (c *collector) isStructLike(typ resolution.Type) bool {
	switch form := typ.Form.(type) {
	case resolution.StructForm:
		return true
	case resolution.AliasForm:
		// An alias to an array of an oracle struct (e.g. Members = Member[]) is
		// struct-like: its elements need a per-element migrate, not a slice cast.
		if elemRef, isArr := arrayBaseRef(typ, c.oldTable); isArr {
			if elem, ok := c.resolveRef(elemRef); ok {
				return c.isStructLike(elem)
			}
		}
		if r, ok := form.Target.Resolve(c.oldTable); ok {
			if _, s := r.Form.(resolution.StructForm); s {
				return true
			}
		}
	case resolution.DistinctForm:
		if elemRef, isArr := arrayBaseRef(typ, c.oldTable); isArr {
			if elem, ok := c.resolveRef(elemRef); ok {
				return c.isStructLike(elem)
			}
		}
		if r, ok := form.Base.Resolve(c.oldTable); ok {
			if _, s := r.Form.(resolution.StructForm); s {
				return true
			}
		}
	}
	return false
}

// --- Pure helpers ---

func needsAutoMigrate(
	types []resolution.Type,
	diff map[string]schemadiff.TypeDiff,
) bool {
	for _, typ := range types {
		if td, ok := diff[typ.QualifiedName]; ok &&
			td.Kind != schemadiff.TypeUnchanged {
			return true
		}
	}
	return false
}

// isMigrateEntry reports whether t carries the @go migrate directive, marking
// it as a top-level migration entry whose Migrate/AutoMigrate functions are
// generated in its own package rather than a sub-package mirror.
func isMigrateEntry(t resolution.Type) bool {
	return domain.HasExprFromType(t, "go", "migrate")
}

// decorateWithTypeParams applies generic type-param decoration to a funcData:
// appends the type-param reference (e.g., "[Details]") to OldTypeName and
// NewTypeName, sets TypeParamsDecl (e.g., "[Details any]"), and (for struct
// kinds) rebuilds ZeroValue against the now-decorated NewTypeName. A no-op
// when tparams is empty, so non-generic callers can use the helper unchanged.
func decorateWithTypeParams(fn funcData, tparams []resolution.TypeParam) funcData {
	if len(resolution.NonDefaultedTypeParams(tparams)) == 0 {
		return fn
	}
	tref := formatTypeParamsRef(tparams)
	fn.OldTypeName += tref
	fn.NewTypeName += tref
	fn.TypeParamsDecl = formatTypeParamsDecl(tparams)
	if fn.Kind == "struct" {
		fn.ZeroValue = fn.NewTypeName + "{}"
	}
	return fn
}

// formatTypeParamsDecl renders the type-param list as it appears in a function
// signature: "" when the slice is empty, "[Details any, V comparable]" when
// non-empty. Constraints fall back to "any" when none is set on the param.
func formatTypeParamsDecl(tps []resolution.TypeParam) string {
	tps = resolution.NonDefaultedTypeParams(tps)
	if len(tps) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[")
	for i, tp := range tps {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(tp.Name)
		b.WriteString(" ")
		if tp.Constraint != nil {
			b.WriteString(tp.Constraint.Name)
		} else {
			b.WriteString("any")
		}
	}
	b.WriteString("]")
	return b.String()
}

// formatTypeParamsRef renders the type-param list as it appears at a use site
// (parameter type, return type, type-param pass-through): "" when empty,
// "[Details, V]" otherwise.
func formatTypeParamsRef(tps []resolution.TypeParam) string {
	tps = resolution.NonDefaultedTypeParams(tps)
	if len(tps) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("[")
	for i, tp := range tps {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(tp.Name)
	}
	b.WriteString("]")
	return b.String()
}

func findField(fields []resolution.Field, name string) *resolution.Field {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}

// arrayBaseRef returns the element of the array that the Go type of an alias or
// distinct type is, through any chain of aliases and distinct types.
func arrayBaseRef(
	typ resolution.Type,
	table *resolution.Table,
) (resolution.TypeRef, bool) {
	var base resolution.TypeRef
	switch form := typ.Form.(type) {
	case resolution.AliasForm:
		base = form.Target
	case resolution.DistinctForm:
		base = form.Base
	default:
		return resolution.TypeRef{}, false
	}
	target, targetRef, ok := resolution.Dealias(base, table)
	if !ok {
		return resolution.TypeRef{}, false
	}
	if targetRef.Name == "Array" && len(targetRef.TypeArgs) > 0 {
		return targetRef.TypeArgs[0], true
	}
	return arrayBaseRef(target, table)
}

func substituteTypeParams(
	sf resolution.StructForm,
	typeArgs []resolution.TypeRef,
) resolution.StructForm {
	if len(sf.TypeParams) == 0 {
		return sf
	}
	paramMap := make(map[string]resolution.TypeRef, len(sf.TypeParams))
	for i, tp := range sf.TypeParams {
		if i < len(typeArgs) {
			paramMap[tp.Name] = typeArgs[i]
		} else if tp.HasDefault() {
			paramMap[tp.Name] = *tp.Default
		}
	}
	newFields := make([]resolution.Field, len(sf.Fields))
	for i, f := range sf.Fields {
		newFields[i] = f
		newFields[i].Type = substituteRef(f.Type, paramMap)
	}
	result := sf
	result.Fields = newFields
	return result
}

func substituteRef(
	ref resolution.TypeRef,
	paramMap map[string]resolution.TypeRef,
) resolution.TypeRef {
	if ref.TypeParam != nil {
		if concrete, ok := paramMap[ref.TypeParam.Name]; ok {
			return concrete
		}
	}
	if len(ref.TypeArgs) > 0 {
		newArgs := make([]resolution.TypeRef, len(ref.TypeArgs))
		for i, arg := range ref.TypeArgs {
			newArgs[i] = substituteRef(arg, paramMap)
		}
		result := ref
		result.TypeArgs = newArgs
		return result
	}
	return ref
}
