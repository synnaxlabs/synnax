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
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// frozenAliasSplit compares a candidate define-all rendering of a current
// version package against the frozen predecessor's types.gen.go and returns
// the qualified names of types that can alias the predecessor. A type aliases
// only when every Go declaration it contributes (comments stripped) is
// identical in both files, and every locally-referenced type also aliases —
// so an alias is emitted only when it denotes literally the same code.
func frozenAliasSplit(
	candidate, frozen []byte,
	ctx *framework.GenerateContext,
) (set.Set[string], error) {
	candOwners, candRefs, err := groupDecls(candidate)
	if err != nil {
		return nil, errors.Wrap(err, "candidate rendering does not parse")
	}
	frozOwners, _, err := groupDecls(frozen)
	if err != nil {
		return nil, errors.Wrap(err, "frozen predecessor does not parse")
	}

	type entry struct {
		qualified string
		owners    []string
	}
	var entries []entry
	ownerToQualified := make(map[string]string)
	for _, d := range orderDecls(ctx.Table, ctx.TypeDefs, ctx.Enums, ctx.Structs, ctx.Unions) {
		if omit.IsType(d.typ, "go") {
			continue
		}
		if d.kind == declEnum && d.typ.Namespace != ctx.Namespace {
			continue
		}
		owners := declOwnerNames(d)
		if len(owners) == 0 {
			continue
		}
		entries = append(entries, entry{qualified: d.typ.QualifiedName, owners: owners})
		for _, o := range owners {
			ownerToQualified[o] = d.typ.QualifiedName
		}
	}

	aliased := make(set.Set[string])
	for _, e := range entries {
		match := true
		for _, o := range e.owners {
			c, cok := candOwners[o]
			f, fok := frozOwners[o]
			if !cok || !fok || c != f {
				match = false
				break
			}
		}
		if match {
			aliased.Add(e.qualified)
		}
	}

	// A textually-identical declaration referencing a locally re-defined type
	// still denotes a different shape; drop such types until stable.
	for changed := true; changed; {
		changed = false
		for _, e := range entries {
			if !aliased.Contains(e.qualified) {
				continue
			}
			for _, o := range e.owners {
				for _, ref := range candRefs[o] {
					refQualified, local := ownerToQualified[ref]
					if local && refQualified != e.qualified && !aliased.Contains(refQualified) {
						aliased.Remove(e.qualified)
						changed = true
						break
					}
				}
				if !aliased.Contains(e.qualified) {
					break
				}
			}
		}
	}
	return aliased, nil
}

// declOwnerNames returns the Go declaration owner names an oracle type
// contributes to a generated types file. Unions expand to their variant,
// discriminator, and interface types.
func declOwnerNames(d orderedDecl) []string {
	name := naming.GetGoName(d.typ)
	switch d.kind {
	case declTypeDef, declEnum, declStruct:
		return []string{name}
	case declUnion:
		form, ok := d.typ.Form.(resolution.UnionForm)
		if !ok {
			return nil
		}
		names := []string{name, name + "Variant", name + "Type"}
		for _, v := range form.Variants {
			names = append(names, casing.VariantTypeName(name, v.Name))
		}
		return names
	}
	return nil
}

// groupDecls parses a generated Go file with comments stripped and groups its
// declarations by owner: the declared type for type specs, the first spec's
// type for const blocks, and the receiver's base type for methods. It returns
// each owner's printed declarations and the identifiers they reference.
func groupDecls(src []byte) (map[string]string, map[string][]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.SkipObjectResolution)
	if err != nil {
		return nil, nil, err
	}
	grouped := make(map[string][]ast.Decl)
	addDecl := func(owner string, decl ast.Decl) {
		if owner == "" {
			return
		}
		grouped[owner] = append(grouped[owner], decl)
	}
	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			switch d.Tok {
			case token.TYPE:
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					addDecl(ts.Name.Name, &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{spec}})
				}
			case token.CONST:
				addDecl(constOwner(d), d)
			}
		case *ast.FuncDecl:
			if d.Recv != nil && len(d.Recv.List) > 0 {
				addDecl(receiverBase(d.Recv.List[0].Type), d)
			}
		}
	}
	contents := make(map[string]string, len(grouped))
	refs := make(map[string][]string, len(grouped))
	for owner, decls := range grouped {
		var buf bytes.Buffer
		seen := make(set.Set[string])
		for _, decl := range decls {
			if err := printer.Fprint(&buf, fset, decl); err != nil {
				return nil, nil, err
			}
			buf.WriteByte('\n')
			ast.Inspect(decl, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok && !seen.Contains(id.Name) {
					seen.Add(id.Name)
					refs[owner] = append(refs[owner], id.Name)
				}
				return true
			})
		}
		contents[owner] = buf.String()
	}
	return contents, refs, nil
}

// constOwner returns the type name governing a const block: the first
// explicitly-typed spec's type. Iota blocks type only their first spec.
func constOwner(d *ast.GenDecl) string {
	for _, spec := range d.Specs {
		if vs, ok := spec.(*ast.ValueSpec); ok && vs.Type != nil {
			if id, ok := vs.Type.(*ast.Ident); ok {
				return id.Name
			}
		}
	}
	return ""
}

// receiverBase unwraps a method receiver's type expression to its base
// identifier ("*Entry[K]" → "Entry").
func receiverBase(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return receiverBase(t.X)
	case *ast.IndexExpr:
		return receiverBase(t.X)
	case *ast.IndexListExpr:
		return receiverBase(t.X)
	case *ast.Ident:
		return t.Name
	}
	return ""
}
