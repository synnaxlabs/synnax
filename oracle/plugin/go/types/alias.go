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
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
)

// aliasFileGenerator emits a re-export surface for a version-laid-out
// package: a type alias for every type generated into the current types/vN
// sub-package, plus const re-declarations for enum members and union
// discriminator values. Methods travel with the aliased types, so the alias
// package presents the full generated API. The types/ selector imports the
// current version package and the root imports the selector, both under
// their natural package names, so a version bump touches only the selector
// and the root never changes.
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
}

type aliasConst struct{ Name, Target string }

type aliasData struct {
	Package string
	Import  string
	Types   []aliasDecl
	Consts  []aliasConst
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
	prefix := "types"
	if g.pkg != "" {
		emitPkg = g.pkg
		importPath = versionedPath
		prefix = naming.DerivePackageName(versionedPath)
	}
	ad := &aliasData{
		Package: emitPkg,
		Import:  resolveGoImportPath(importPath, ctx.RepoRoot),
	}

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
		ad.Types = append(ad.Types, aliasDecl{Name: name, LHS: lhs, RHS: rhs, Doc: docStr})
	}

	for _, td := range ctx.TypeDefs {
		if omit.IsType(td, "go") {
			continue
		}
		switch form := td.Form.(type) {
		case resolution.DistinctForm:
			addType(naming.GetGoName(td), doc.Get(td.Domains), form.TypeParams)
		case resolution.AliasForm:
			addType(naming.GetGoName(td), doc.Get(td.Domains), form.TypeParams)
		default:
			addType(naming.GetGoName(td), doc.Get(td.Domains), nil)
		}
	}

	for _, e := range ctx.Enums {
		if e.Namespace != namespace || omit.IsType(e, "go") {
			continue
		}
		name := naming.GetGoName(e)
		addType(name, doc.Get(e.Domains), nil)
		form := e.Form.(resolution.EnumForm)
		for _, v := range form.Values {
			member := name + naming.ToPascalCase(v.Name)
			ad.Consts = append(ad.Consts, aliasConst{Name: member, Target: prefix + "." + member})
		}
	}

	for _, s := range ctx.Structs {
		if omit.IsType(s, "go") {
			continue
		}
		form, ok := s.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		addType(naming.GetGoName(s), doc.Get(s.Domains), form.TypeParams)
	}

	for _, u := range ctx.Unions {
		if omit.IsType(u, "go") {
			continue
		}
		form, ok := u.Form.(resolution.UnionForm)
		if !ok {
			continue
		}
		name := naming.GetGoName(u)
		addType(name, doc.Get(u.Domains), nil)
		addType(name+"Variant", "", nil)
		discType := name + "Type"
		addType(discType, "", nil)
		for _, v := range form.Variants {
			addType(casing.VariantTypeName(name, v.Name), doc.Get(v.Domains), nil)
			constName := discType + casing.PascalAcronym(v.Name)
			ad.Consts = append(ad.Consts, aliasConst{Name: constName, Target: prefix + "." + constName})
		}
	}

	// A path whose types are all @go omit generates no aliases; emitting the
	// file would leave an unused import.
	if len(ad.Types) == 0 && len(ad.Consts) == 0 {
		return "", nil
	}
	var buf bytes.Buffer
	if err := aliasTemplate.Execute(&buf, ad); err != nil {
		return "", err
	}
	return buf.String(), nil
}

var aliasTemplate = template.Must(template.New("go-alias").
	Funcs(template.FuncMap{"formatDoc": doc.FormatGo}).
	Parse(`// Code generated by oracle. DO NOT EDIT.

package {{.Package}}

import "{{.Import}}"
{{range .Types}}
{{- if .Doc}}
{{formatDoc .Name .Doc}}
{{- end}}
type {{.LHS}} = {{.RHS}}
{{- end}}
{{- if .Consts}}

const (
{{- range .Consts}}
	{{.Name}} = {{.Target}}
{{- end}}
)
{{- end}}
`))
