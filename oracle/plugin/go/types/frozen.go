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
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin/framework"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/internal/casing"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// frozenAliasSplit compares a candidate define-all rendering of a current
// version package against the frozen predecessor package on disk and returns
// the qualified names of types that can alias the predecessor. A type aliases
// only when every declaration the candidate contributes (comments stripped)
// appears identically in the predecessor and every locally-referenced type
// also aliases. The predecessor may carry extra declarations — hand-written
// methods live with the definer and travel through the alias; a duplicate
// method surfaces as a compile error.
func frozenAliasSplit(
	candidate []byte,
	ctx *framework.GenerateContext,
	predDir string,
) (set.Set[string], error) {
	candOwners, candRefs, candSpecs, err := groupDecls(candidate)
	if err != nil {
		return nil, errors.Wrap(err, "candidate rendering does not parse")
	}
	frozOwners, frozSpecs, err := packageDecls(predDir)
	if err != nil {
		return nil, err
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
			if cok && fok && declsSubset(c, f) {
				continue
			}
			// A frozen decl that is itself a predecessor-chain alias denotes
			// the definer's type; follow the chain and compare type specs.
			if !cok || !fok || !chainAliasMatches(o, frozSpecs[o], candSpecs[o], predDir) {
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

// normalizeQualifiers rewrites every package qualifier in the file to its
// full import path, so printed declarations compare by what a selector
// denotes rather than how the import happens to be aliased. Two files
// importing different versions of a resource under the same alias ("ir")
// would otherwise print byte-identical declarations for different types.
func normalizeQualifiers(file *ast.File) {
	paths := make(map[string]string, len(file.Imports))
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, `"`)
		name := filepath.Base(path)
		if imp.Name != nil {
			name = imp.Name.Name
		}
		paths[name] = path
	}
	ast.Inspect(file, func(n ast.Node) bool {
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok {
			if p, ok := paths[id.Name]; ok {
				id.Name = "«" + p + "»"
			}
		}
		return true
	})
}

// groupDecls parses a Go file with comments stripped and groups its
// declarations by owner: the declared type for type specs, the first spec's
// type for const blocks, and the receiver's base type for methods. It returns
// each owner's printed declarations in order, the identifiers they reference,
// and each owner's printed type spec alone. Package qualifiers print as full
// import paths (see normalizeQualifiers).
func groupDecls(src []byte) (map[string][]string, map[string][]string, map[string]string, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", src, parser.SkipObjectResolution)
	if err != nil {
		return nil, nil, nil, err
	}
	normalizeQualifiers(file)
	grouped := make(map[string][]ast.Decl)
	typeSpecs := make(map[string]string)
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
					single := &ast.GenDecl{Tok: token.TYPE, Specs: []ast.Spec{spec}}
					addDecl(ts.Name.Name, single)
					var buf bytes.Buffer
					if err := printer.Fprint(&buf, fset, single); err != nil {
						return nil, nil, nil, err
					}
					typeSpecs[ts.Name.Name] = buf.String()
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
	contents := make(map[string][]string, len(grouped))
	refs := make(map[string][]string, len(grouped))
	for owner, decls := range grouped {
		seen := make(set.Set[string])
		// Field and parameter names are not type references; a field named
		// like a local type must not chain the two declarations together.
		var collect func(n ast.Node) bool
		collect = func(n ast.Node) bool {
			if f, ok := n.(*ast.Field); ok {
				ast.Inspect(f.Type, collect)
				return false
			}
			if id, ok := n.(*ast.Ident); ok && !seen.Contains(id.Name) {
				seen.Add(id.Name)
				refs[owner] = append(refs[owner], id.Name)
			}
			return true
		}
		for _, decl := range decls {
			var buf bytes.Buffer
			if err := printer.Fprint(&buf, fset, decl); err != nil {
				return nil, nil, nil, err
			}
			contents[owner] = append(contents[owner], buf.String())
			ast.Inspect(decl, collect)
		}
	}
	return contents, refs, typeSpecs, nil
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

// chainAliasRe matches a predecessor-chain alias type spec after qualifier
// normalization: "type X = «…/vN».X", optionally generic. Only aliases into
// sibling version directories hop; any other target compares as a spec.
var chainAliasRe = regexp.MustCompile(`^type (\w+)(?:\[[^\]]*\])? = «[^»]*/(v\d+)»\.(\w+)`)

// chainAliasMatches reports whether the frozen type spec is an alias into a
// sibling version package whose definition of owner has the same type spec as
// the candidate. It follows further alias hops until a definition is found.
func chainAliasMatches(owner, frozenSpec, candidateSpec, predDir string) bool {
	if candidateSpec == "" {
		return false
	}
	dir := predDir
	spec := frozenSpec
	// Version chains are short; the bound only guards against cycles.
	for hop := range 32 {
		m := chainAliasRe.FindStringSubmatch(spec)
		if m == nil || m[1] != owner || m[3] != owner {
			// The chain ended in a real definition: compare type specs. A
			// frozen decl that was never an alias is governed solely by the
			// strict group comparison.
			if hop == 0 {
				return false
			}
			return spec == candidateSpec
		}
		dir = filepath.Join(filepath.Dir(dir), m[2])
		next, ok := packageTypeSpec(dir, owner)
		if !ok {
			return false
		}
		spec = next
	}
	return false
}

// packageTypeSpec returns the printed type spec declaring name within the
// package directory, searching every non-test Go file.
func packageTypeSpec(dir, name string) (string, bool) {
	_, specs, err := packageDecls(dir)
	if err != nil {
		return "", false
	}
	spec, ok := specs[name]
	return spec, ok
}

// declsSubset reports whether every candidate declaration appears identically
// among the frozen declarations.
func declsSubset(candidate, frozen []string) bool {
	remaining := make(map[string]int, len(frozen))
	for _, d := range frozen {
		remaining[d]++
	}
	for _, d := range candidate {
		if remaining[d] == 0 {
			return false
		}
		remaining[d]--
	}
	return true
}

// packageDecls merges groupDecls over every non-test Go file in dir.
func packageDecls(dir string) (map[string][]string, map[string]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "frozen predecessor %s is unreadable", dir)
	}
	owners := make(map[string][]string)
	specs := make(map[string]string)
	for _, f := range files {
		if f.IsDir() || !strings.HasSuffix(f.Name(), ".go") ||
			strings.HasSuffix(f.Name(), "_test.go") {
			continue
		}
		src, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, nil, err
		}
		fo, _, fs, err := groupDecls(src)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "frozen predecessor %s does not parse", f.Name())
		}
		for o, decls := range fo {
			owners[o] = append(owners[o], decls...)
		}
		maps.Copy(specs, fs)
	}
	return owners, specs, nil
}
