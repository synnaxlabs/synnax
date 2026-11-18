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
	"strings"
	"text/template"

	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// aliasFileGenerator emits a re-export surface for a version-laid-out
// package: a type alias for every type generated into the current versions/vN
// sub-package, plus const re-declarations for enum members and union
// discriminator values. Methods travel with the aliased types, so the alias
// package presents the full generated API. The types/ selector imports the
// current version package and the root imports the selector, so a version
// bump touches only the selector and the root never changes.
type aliasFileGenerator struct {
	// pathMap maps each version-laid-out root path to its current versions/vN
	// sub-path.
	pathMap map[string]string
	// members holds the qualified names of every chain's current-surface members.
	members set.Set[string]
	// pkg overrides the emitted package name; empty derives it from the
	// output path (the package-root case).
	pkg string
}

type aliasDecl struct {
	// Name is the bare type name, used to prefix the doc comment.
	Name string
	// LHS is the alias declaration head ("Status[Details any]").
	LHS string
	// RHS is the aliased target ("latest.Status[Details]").
	RHS string
	// Doc is the rendered documentation comment, if any.
	Doc string
	// Consts are the type's member re-declarations (enum values or union
	// discriminator values), emitted directly below the alias.
	Consts []aliasConst
}

type aliasConst struct{ Name, Type, Target, Doc string }

type aliasData struct {
	Package string
	Import  string
	Types   []aliasDecl
}

// newAliasTemplateData returns the template data that resolves types for the alias file
// of package pkg.
func newAliasTemplateData(ctx *framework.GenerateContext, pkg string) *templateData {
	imp := imports.NewManager()
	rctx := &resolver.Context{
		Table:                         ctx.Table,
		OutputPath:                    ctx.OutputPath,
		Namespace:                     ctx.Namespace,
		RepoRoot:                      ctx.RepoRoot,
		DomainName:                    "go",
		SubstituteDefaultedTypeParams: true,
	}
	r := &resolver.Resolver{
		Formatter:       GoFormatter(),
		ImportResolver:  &GoImportResolver{RepoRoot: ctx.RepoRoot, CurrentPackage: pkg},
		ImportAdder:     imp,
		PrimitiveMapper: primitiveMapper,
	}
	return &templateData{
		Package:    pkg,
		OutputPath: ctx.OutputPath,
		Namespace:  ctx.Namespace,
		Manager:    imp,
		table:      ctx.Table,
		repoRoot:   ctx.RepoRoot,
		resolver:   r,
		ctx:        rctx,
	}
}

func (g *aliasFileGenerator) GenerateFile(
	ctx *framework.GenerateContext,
) (string, error) {
	versionedPath, ok := g.pathMap[ctx.OutputPath]
	if !ok {
		return "", nil
	}
	namespace := ctx.Namespace
	pkg := naming.DerivePackageName(ctx.OutputPath)
	data := newAliasTemplateData(ctx, pkg)

	emitPkg := pkg
	importPath := ctx.OutputPath + "/versions"
	if g.pkg != "" {
		emitPkg = g.pkg
		importPath = versionedPath
	}
	prefix := naming.DerivePackageName(importPath)
	ad := &aliasData{
		Package: emitPkg,
		Import:  resolveGoImportPath(importPath, ctx.RepoRoot),
	}

	for _, d := range orderDecls(
		ctx.Table, nil, ctx.TypeDefs, ctx.Enums, ctx.Structs, ctx.Unions,
	) {
		if omit.IsType(d.typ, "go") {
			continue
		}
		// Transient types generate real declarations at the package root,
		// not in the version layout: no alias to emit. Hand types follow the
		// version layout, so their aliases are emitted even when unversioned.
		if !g.members.Contains(d.typ.QualifiedName) && !omit.IsHand(d.typ, "go") {
			continue
		}
		if d.kind == declEnum && d.typ.Namespace != namespace {
			continue
		}
		ad.Types = append(ad.Types, buildAliasDecls(d, prefix, "", data)...)
	}

	// A path whose types are all @go omit generates no aliases; emitting the
	// file would leave an unused import.
	if len(ad.Types) == 0 {
		return "", nil
	}
	var buf bytes.Buffer
	if err := aliasTemplate.Execute(&buf, ad); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// buildAliasDecls returns the alias declarations re-exporting d's type from
// the package referenced as prefix: a type alias per generated type (unions
// expand to their variant, discriminator, and interface types) plus const
// re-declarations for enum members and union discriminator values. target is the
// Go name the prefix package declares for d's type; empty means d's own name.
func buildAliasDecls(
	d orderedDecl, prefix, target string, data *templateData,
) []aliasDecl {
	b := aliasDeclBuilder{prefix: prefix, data: data}
	name := naming.GetGoName(d.typ)
	if target == "" {
		target = name
	}
	typeDoc := doc.Get(d.typ.Domains)
	switch d.kind {
	case declTypeDef:
		switch form := d.typ.Form.(type) {
		case resolution.DistinctForm:
			b.addType(name, target, typeDoc, form.TypeParams)
		case resolution.AliasForm:
			b.addType(name, target, typeDoc, form.TypeParams)
		default:
			b.addType(name, target, typeDoc, nil)
		}
	case declEnum:
		b.addType(name, target, typeDoc, nil)
		form := resolution.MustForm[resolution.EnumForm](d.typ)
		decl := &b.decls[len(b.decls)-1]
		for _, v := range form.Values {
			member := naming.ToPascalCase(v.Name)
			decl.Consts = append(decl.Consts, aliasConst{
				Name:   name + member,
				Type:   name,
				Target: prefix + "." + target + member,
				Doc:    doc.Get(v.Domains),
			})
		}
	case declStruct:
		form, ok := d.typ.Form.(resolution.StructForm)
		if !ok {
			return nil
		}
		b.addType(name, target, typeDoc, form.TypeParams)
	case declUnion:
		form, ok := d.typ.Form.(resolution.UnionForm)
		if !ok {
			return nil
		}
		b.addType(name, target, typeDoc, nil)
		b.addUnion(name, target, form)
	}
	return b.decls
}

// aliasDeclBuilder collects the alias declarations that re-export types from the
// package referenced as prefix.
type aliasDeclBuilder struct {
	decls  []aliasDecl
	prefix string
	data   *templateData
}

// addType adds a type alias named name for the prefix package's targetName.
func (b *aliasDeclBuilder) addType(
	name, targetName, docStr string,
	tparams []resolution.TypeParam,
) {
	lhs, rhs := name, b.prefix+"."+targetName
	params := resolution.NonDefaultedTypeParams(tparams)
	if len(params) > 0 {
		lhsParams := make([]string, len(params))
		rhsParams := make([]string, len(params))
		for i, tp := range params {
			tpd := processTypeParam(tp, b.data)
			lhsParams[i] = tpd.Name + " " + tpd.Constraint
			rhsParams[i] = tpd.Name
		}
		lhs += "[" + strings.Join(lhsParams, ", ") + "]"
		rhs += "[" + strings.Join(rhsParams, ", ") + "]"
	}
	b.decls = append(b.decls, aliasDecl{Name: name, LHS: lhs, RHS: rhs, Doc: docStr})
}

// addUnion adds the aliases of a union's interface and variant types, plus its
// discriminator type and values when it does not borrow an enum.
func (b *aliasDeclBuilder) addUnion(name, target string, form resolution.UnionForm) {
	b.addType(name+"Variant", target+"Variant", interfaceDoc(name), nil)
	if form.BorrowsEnum() {
		for _, v := range form.Variants {
			b.addType(
				casing.VariantTypeName(name, v.Name),
				casing.VariantTypeName(target, v.Name),
				doc.Get(v.Domains),
				nil,
			)
		}
		return
	}
	discType := name + "Type"
	b.addType(
		discType,
		target+"Type",
		discDoc(name, casing.FieldSnake(form.Discriminator)),
		nil,
	)
	discDecl := len(b.decls) - 1
	for _, v := range form.Variants {
		variantDoc := doc.Get(v.Domains)
		variantType := casing.VariantTypeName(name, v.Name)
		b.addType(
			variantType, casing.VariantTypeName(target, v.Name), variantDoc, nil,
		)
		b.decls[discDecl].Consts = append(b.decls[discDecl].Consts, aliasConst{
			Name:   casing.VariantConstName(name, v.Name),
			Type:   discType,
			Target: b.prefix + "." + casing.VariantConstName(target, v.Name),
			Doc:    constDoc(variantDoc, variantType),
		})
	}
}

var aliasTemplate = template.Must(template.New("go-alias").
	Funcs(template.FuncMap{
		"formatDoc": doc.FormatGo,
		"formatFieldDoc": func(name, docStr string) string {
			return doc.FormatGo(name, docStr, fieldDocIndent)
		},
	}).
	Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import "{{.Import}}"
{{range .Types}}
{{- if .Doc}}
{{formatDoc .Name .Doc}}
{{- end}}
type {{.LHS}} = {{.RHS}}
{{- if eq (len .Consts) 1}}
{{- with index .Consts 0}}

{{- if .Doc}}

{{formatDoc .Name .Doc}}
{{- end}}
const {{.Name}} {{.Type}} = {{.Target}}
{{- end}}
{{- else if .Consts}}

const (
{{- range .Consts}}
{{- if .Doc}}
	{{formatFieldDoc .Name .Doc | printf "%s"}}
{{- end}}
	{{.Name}} {{.Type}} = {{.Target}}
{{- end}}
)
{{- end}}
{{- end}}
`))
