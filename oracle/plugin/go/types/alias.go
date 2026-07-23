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
	"text/template"

	"github.com/synnaxlabs/oracle/domain/doc"
	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/go/internal/imports"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/go/internal/versioning"
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
)

// aliasFileGenerator emits a re-export surface for a version-laid-out
// package: a type alias for every type generated into the current types/vN
// sub-package, plus const re-declarations for enum members and union
// discriminator values. Methods travel with the aliased types, so the alias
// package presents the full generated API. The types/ selector imports the
// current version package and the root imports the selector, so a version
// bump touches only the selector and the root never changes.
type aliasFileGenerator struct {
	// pathMap maps each version-laid-out root path to its current types/vN
	// sub-path.
	pathMap map[string]string
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

func (g *aliasFileGenerator) GenerateFile(ctx *framework.GenerateContext) (string, error) {
	versionedPath, ok := g.pathMap[ctx.OutputPath]
	if !ok {
		return "", nil
	}
	namespace := ctx.Namespace
	pkg := naming.DerivePackageName(ctx.OutputPath)
	imp := imports.NewManager()
	rctx := &resolver.Context{
		Table:                         ctx.Table,
		OutputPath:                    ctx.OutputPath,
		Namespace:                     namespace,
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
	data := &templateData{
		Package:    pkg,
		OutputPath: ctx.OutputPath,
		Namespace:  namespace,
		imports:    imp,
		table:      ctx.Table,
		repoRoot:   ctx.RepoRoot,
		resolver:   r,
		ctx:        rctx,
	}

	emitPkg := pkg
	importPath := ctx.OutputPath + "/types"
	if g.pkg != "" {
		emitPkg = g.pkg
		importPath = versionedPath
	}
	prefix := naming.DerivePackageName(importPath)
	ad := &aliasData{
		Package: emitPkg,
		Import:  resolveGoImportPath(importPath, ctx.RepoRoot),
	}

	for _, d := range orderDecls(ctx.Table, ctx.TypeDefs, ctx.Enums, ctx.Structs, ctx.Unions) {
		if omit.IsSkipped(d.typ, "go") {
			continue
		}
		// Transient types generate real declarations at the package root
		// (transient.gen.go), not in the version layout: no alias to emit.
		if _, versioned := versioning.Version(d.typ); !versioned {
			continue
		}
		if d.kind == declEnum && d.typ.Namespace != namespace {
			continue
		}
		ad.Types = append(ad.Types, buildAliasDecls(d, prefix, data)...)
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
// re-declarations for enum members and union discriminator values.
func buildAliasDecls(d orderedDecl, prefix string, data *templateData) []aliasDecl {
	var decls []aliasDecl
	addType := func(name, docStr string, tparams []resolution.TypeParam) {
		lhs, rhs := name, prefix+"."+name
		params := resolution.NonDefaultedTypeParams(tparams)
		if len(params) > 0 {
			lhs += "["
			rhs += "["
			for i, tp := range params {
				if i > 0 {
					lhs += ", "
					rhs += ", "
				}
				tpd := processTypeParam(tp, data)
				lhs += tpd.Name + " " + tpd.Constraint
				rhs += tpd.Name
			}
			lhs += "]"
			rhs += "]"
		}
		decls = append(decls, aliasDecl{Name: name, LHS: lhs, RHS: rhs, Doc: docStr})
	}
	switch d.kind {
	case declTypeDef:
		switch form := d.typ.Form.(type) {
		case resolution.DistinctForm:
			addType(naming.GetGoName(d.typ), doc.Get(d.typ.Domains), form.TypeParams)
		case resolution.AliasForm:
			addType(naming.GetGoName(d.typ), doc.Get(d.typ.Domains), form.TypeParams)
		default:
			addType(naming.GetGoName(d.typ), doc.Get(d.typ.Domains), nil)
		}
	case declEnum:
		name := naming.GetGoName(d.typ)
		addType(name, doc.Get(d.typ.Domains), nil)
		form := d.typ.Form.(resolution.EnumForm)
		decl := &decls[len(decls)-1]
		for _, v := range form.Values {
			member := name + naming.ToPascalCase(v.Name)
			decl.Consts = append(decl.Consts, aliasConst{
				Name:   member,
				Type:   name,
				Target: prefix + "." + member,
				Doc:    doc.Get(v.Domains),
			})
		}
	case declStruct:
		form, ok := d.typ.Form.(resolution.StructForm)
		if !ok {
			return nil
		}
		addType(naming.GetGoName(d.typ), doc.Get(d.typ.Domains), form.TypeParams)
	case declUnion:
		form, ok := d.typ.Form.(resolution.UnionForm)
		if !ok {
			return nil
		}
		name := naming.GetGoName(d.typ)
		addType(name, doc.Get(d.typ.Domains), nil)
		addType(name+"Variant", "", nil)
		discType := name + "Type"
		addType(discType, "", nil)
		discDecl := len(decls) - 1
		for _, v := range form.Variants {
			addType(casing.VariantTypeName(name, v.Name), doc.Get(v.Domains), nil)
			constName := discType + casing.PascalAcronym(v.Name)
			decls[discDecl].Consts = append(decls[discDecl].Consts, aliasConst{
				Name:   constName,
				Type:   discType,
				Target: prefix + "." + constName,
				Doc:    doc.Get(v.Domains),
			})
		}
	}
	return decls
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
