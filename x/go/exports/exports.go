// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package exports finds exported Go identifiers that no other package uses.
package exports

import (
	"context"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
	"golang.org/x/tools/go/packages"
)

// Config selects the code that Check reads.
type Config struct {
	// Modules are the directories of the Go modules that Check loads as one workspace.
	Modules []string
	// GOOS are the target operating systems that Check loads the modules for.
	GOOS []string
}

// Finding is an exported identifier that no other package uses.
type Finding struct {
	// Position is where the identifier is declared.
	Position token.Position
	// Kind is one of "func", "method", "type", "var", or "const".
	Kind string
	// Name is the identifier.
	Name string
	// TestsOnly is true when only test code uses the identifier: test files, or the
	// packages of test helpers.
	TestsOnly bool
}

// String formats the finding as "file:line: kind Name is exported but ...".
func (f Finding) String() string {
	users := "no other package uses it"
	if f.TestsOnly {
		users = "only test code uses it"
	}
	return fmt.Sprintf(
		"%s:%d: %s %s is exported but %s",
		f.Position.Filename, f.Position.Line, f.Kind, f.Name, users,
	)
}

// Check loads every package of cfg.Modules for each GOOS in cfg.GOOS and returns the
// exported identifiers that no other package uses, sorted by position. An identifier
// that only one GOOS builds is judged on that GOOS alone. Check keeps exported:
//
//   - an identifier that generated code or another package's non-test code uses, or
//     that another package's tests use when it lives in a test helper package;
//   - a method that satisfies an interface or that template data can reach;
//   - a type that the API of a kept identifier carries;
//   - every constant of a const block when one of them is kept.
//
// Check skips generated files, test files, main packages, versions/vN packages, and
// methods of unexported types. It returns validate.ErrValidation when cfg has no
// modules or no GOOS values, and an error when a package fails to load.
func Check(ctx context.Context, cfg Config) (findings []Finding, err error) {
	if len(cfg.Modules) == 0 {
		return nil, errors.Wrap(
			validate.ErrValidation,
			"exports: modules must be non-empty",
		)
	}
	if len(cfg.GOOS) == 0 {
		return nil, errors.Wrap(
			validate.ErrValidation,
			"exports: goos must be non-empty",
		)
	}
	modules, err := absolute(cfg.Modules)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "exports")
	if err != nil {
		return nil, errors.Wrap(err, "exports: create workspace directory")
	}
	defer func() {
		if rmErr := os.RemoveAll(dir); rmErr != nil && err == nil {
			findings = nil
			err = errors.Wrap(rmErr, "exports: remove workspace directory")
		}
	}()
	work, err := writeWorkspace(dir, modules)
	if err != nil {
		return nil, err
	}
	merged := map[string]*Finding{}
	kept, internal := set.Set[string]{}, set.Set[string]{}
	for _, goos := range cfg.GOOS {
		syms, err := load(ctx, work, goos, modules)
		if err != nil {
			return nil, err
		}
		for k, s := range syms {
			if s.kept {
				kept.Add(k)
				continue
			}
			f, ok := merged[k]
			if !ok {
				f = &Finding{Position: s.pos, Kind: s.kind, Name: s.obj.Name()}
				merged[k] = f
			}
			f.TestsOnly = f.TestsOnly || s.tested
			if s.internal {
				internal.Add(k)
			}
		}
	}
	findings = make([]Finding, 0, len(merged))
	for k, f := range merged {
		if !kept.Contains(k) {
			f.TestsOnly = f.TestsOnly && !internal.Contains(k)
			findings = append(findings, *f)
		}
	}
	slices.SortFunc(findings, func(a, b Finding) int {
		if c := strings.Compare(a.Position.Filename, b.Position.Filename); c != 0 {
			return c
		}
		return a.Position.Line - b.Position.Line
	})
	return findings, nil
}

func absolute(modules []string) ([]string, error) {
	abs := make([]string, len(modules))
	for i, m := range modules {
		var err error
		if abs[i], err = filepath.Abs(m); err != nil {
			return nil, errors.Wrapf(err, "exports: resolve module %s", m)
		}
	}
	return abs, nil
}

func writeWorkspace(dir string, modules []string) (string, error) {
	var b strings.Builder
	// The running toolchain satisfies the go directive of every module it can load.
	fmt.Fprintf(&b, "go %s\n\nuse (\n", strings.TrimPrefix(runtime.Version(), "go"))
	for _, m := range modules {
		fmt.Fprintf(&b, "\t%q\n", m)
	}
	b.WriteString(")\n")
	work := filepath.Join(dir, "go.work")
	if err := os.WriteFile(work, []byte(b.String()), 0o644); err != nil {
		return "", errors.Wrap(err, "exports: write go.work")
	}
	return work, nil
}

type symbol struct {
	obj      types.Object
	pkg      string
	kind     string
	group    string
	pos      token.Position
	kept     bool
	tested   bool
	internal bool
}

var versionsPattern = regexp.MustCompile(`/versions/(.+/)?v[0-9]+$`)

// load returns the exported identifiers of the modules as built for goos, keyed by
// declaration position, with kept set on each identifier that must stay exported.
func load(
	ctx context.Context,
	work, goos string,
	modules []string,
) (map[string]*symbol, error) {
	patterns := make([]string, len(modules))
	for i, m := range modules {
		patterns[i] = m + "/..."
	}
	pkgs, err := packages.Load(&packages.Config{
		Context: ctx,
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax |
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports,
		Env: append(
			os.Environ(),
			"GOOS="+goos, "GOWORK="+work, "GOFLAGS=-buildvcs=false",
		),
		Tests: true,
	}, patterns...)
	if err != nil {
		return nil, errors.Wrapf(err, "exports: load packages for %s", goos)
	}
	var loadErrs []string
	packages.Visit(pkgs, nil, func(p *packages.Package) {
		for _, e := range p.Errors {
			loadErrs = append(loadErrs, e.Error())
		}
	})
	if len(loadErrs) > 0 {
		return nil, errors.Newf(
			"exports: load packages for %s: %s", goos, strings.Join(loadErrs, "; "),
		)
	}
	syms := map[string]*symbol{}
	generated := set.Set[string]{}
	for _, p := range pkgs {
		for _, f := range p.Syntax {
			if ast.IsGenerated(f) {
				generated.Add(p.Fset.File(f.Pos()).Name())
			}
		}
	}
	for _, p := range pkgs {
		if isTestVariant(p) || p.Name == "main" ||
			versionsPattern.MatchString(p.PkgPath) {
			continue
		}
		declare(p, generated, syms)
	}
	for _, p := range pkgs {
		markUses(p, generated, syms)
	}
	if len(pkgs) > 0 {
		keepRest(pkgs[0].Fset, pkgs, syms)
	}
	return syms, nil
}

// isTestVariant reports whether p is a test binary, an external test package, or a
// package recompiled with its in-package tests.
func isTestVariant(p *packages.Package) bool {
	return strings.HasSuffix(p.PkgPath, "_test") ||
		strings.HasSuffix(p.ID, ".test") ||
		strings.Contains(p.ID, "[")
}

func declare(p *packages.Package, generated set.Set[string], syms map[string]*symbol) {
	add := func(id *ast.Ident, kind, group string) {
		obj := p.TypesInfo.Defs[id]
		if obj == nil || !id.IsExported() {
			return
		}
		pos := p.Fset.Position(obj.Pos())
		syms[key(pos, obj)] = &symbol{
			obj: obj, pkg: p.PkgPath, kind: kind, group: group, pos: pos,
		}
	}
	for _, f := range p.Syntax {
		file := p.Fset.File(f.Pos()).Name()
		if generated.Contains(file) || strings.HasSuffix(file, "_test.go") {
			continue
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil {
					add(d.Name, "func", "")
				} else if fn, ok := p.TypesInfo.Defs[d.Name].(*types.Func); ok &&
					receiverExported(fn) {
					add(d.Name, "method", "")
				}
			case *ast.GenDecl:
				group := ""
				if d.Tok == token.CONST && d.Lparen.IsValid() {
					group = fmt.Sprintf("%s:%d", file, p.Fset.Position(d.Pos()).Line)
				}
				for _, spec := range d.Specs {
					switch s := spec.(type) {
					case *ast.TypeSpec:
						add(s.Name, "type", "")
					case *ast.ValueSpec:
						for _, n := range s.Names {
							add(n, d.Tok.String(), group)
						}
					}
				}
			}
		}
	}
}

func receiverExported(fn *types.Func) bool {
	tn := receiver(fn)
	return tn != nil && tn.Exported()
}

// receiver returns the named type that declares the method fn, or nil.
func receiver(fn *types.Func) *types.TypeName {
	recv := fn.Signature().Recv().Type()
	if ptr, ok := recv.(*types.Pointer); ok {
		recv = ptr.Elem()
	}
	if named, ok := types.Unalias(recv).(*types.Named); ok {
		return named.Origin().Obj()
	}
	return nil
}

func key(pos token.Position, obj types.Object) string {
	return fmt.Sprintf("%s:%d:%d:%s", pos.Filename, pos.Line, pos.Column, obj.Name())
}

// markUses keeps each identifier that p uses from another package or from a generated
// file, and marks each identifier that p's tests use.
func markUses(p *packages.Package, generated set.Set[string], syms map[string]*symbol) {
	user := strings.TrimSuffix(p.PkgPath, "_test")
	for id, obj := range p.TypesInfo.Uses {
		obj = origin(obj)
		if obj == nil || obj.Pkg() == nil {
			continue
		}
		s, ok := syms[key(p.Fset.Position(obj.Pos()), obj)]
		if !ok {
			continue
		}
		file := p.Fset.Position(id.Pos()).Filename
		test := strings.HasSuffix(file, "_test.go") || isTestUtility(user)
		switch {
		case generated.Contains(file):
			s.kept = true
		case user != s.pkg && (!test || isTestUtility(s.pkg)):
			s.kept = true
		case test:
			s.tested = true
		default:
			s.internal = true
		}
	}
}

// isTestUtility reports whether the package at path holds test helpers: a path element
// named testutil, test, or mock, or one that ends in mock.
func isTestUtility(path string) bool {
	for elem := range strings.SplitSeq(path, "/") {
		if elem == "testutil" || elem == "test" || strings.HasSuffix(elem, "mock") {
			return true
		}
	}
	return false
}

func origin(obj types.Object) types.Object {
	switch o := obj.(type) {
	case *types.Func:
		return o.Origin()
	case *types.Var:
		return o.Origin()
	}
	return obj
}

// keepRest keeps the identifiers that markUses did not see a use of but that must stay
// exported: interface methods, const block siblings, and types that kept API carries.
func keepRest(fset *token.FileSet, pkgs []*packages.Package, syms map[string]*symbol) {
	ifaces := collectInterfaces(pkgs)
	templated := templateTypes(pkgs)
	groups := set.Set[string]{}
	for _, s := range syms {
		if s.kept && s.group != "" {
			groups.Add(s.group)
		}
	}
	for _, s := range syms {
		switch {
		case groups.Contains(s.group):
			s.kept = true
		case s.kind == "method" && (ifaces.satisfied(s.obj.(*types.Func)) ||
			templated.Contains(receiver(s.obj.(*types.Func)))):
			s.kept = true
		}
	}
	var pending []*symbol
	for _, s := range syms {
		if s.kept {
			pending = append(pending, s)
		}
	}
	for len(pending) > 0 {
		s := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		reach(s.obj.Type(), set.Set[types.Type]{}, func(tn *types.TypeName) {
			t, ok := syms[key(fset.Position(tn.Pos()), tn)]
			if ok && !t.kept {
				t.kept = true
				pending = append(pending, t)
			}
		})
	}
}
