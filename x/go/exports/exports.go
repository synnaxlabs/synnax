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
	"io/fs"
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
	// Workspace are more module directories that the workspace includes, so that the
	// local dependencies of Modules resolve. Check does not load their packages, so it
	// sees no use in them.
	Workspace []string
	// GOOS are the target operating systems that Check loads the modules for.
	GOOS []string
	// Base is the root of a checkout to compare with. When set, Check returns only
	// the findings that the same module directories under Base lack, and the findings
	// that Gated covers. Every module directory must then be relative.
	Base string
	// Gated are the modules and usages whose findings Check returns beside a base,
	// new or not.
	Gated []Gate
}

// Usage says which code uses the identifier of a finding: "unused" when no other
// package does, and "tests-only" when only test code does.
type Usage string

const (
	unused    Usage = "unused"
	testsOnly Usage = "tests-only"
)

// Gate selects the findings of one usage in one module.
type Gate struct {
	// Module is a module directory as Config.Modules gives it.
	Module string `json:"module"`
	// Usage is the usage of the findings.
	Usage Usage `json:"usage"`
}

// Finding is an exported identifier that no other package uses.
type Finding struct {
	// Position is where the identifier is declared.
	Position token.Position
	// Module is the directory, as Config.Modules gives it, of the declaring module.
	Module string
	// Package is the import path of the package that declares the identifier.
	Package string
	// Kind is one of "func", "method", "type", "var", or "const".
	Kind string
	// Receiver is the name of the type that declares a method, and empty otherwise.
	Receiver string
	// Name is the identifier.
	Name string
	// TestsOnly is true when only test code uses the identifier: test files, or the
	// packages of test helpers.
	TestsOnly bool
}

// String formats the finding as "file:line: kind Name is exported but ...". A method's
// name carries its receiver: "method Type.Name".
func (f Finding) String() string {
	users := "no other package uses it"
	if f.TestsOnly {
		users = "only test code uses it"
	}
	name := f.Name
	if f.Receiver != "" {
		name = f.Receiver + "." + name
	}
	return fmt.Sprintf(
		"%s:%d: %s %s is exported but %s",
		f.Position.Filename, f.Position.Line, f.Kind, name, users,
	)
}

func (f Finding) usage() Usage {
	if f.TestsOnly {
		return testsOnly
	}
	return unused
}

type identity struct{ pkg, kind, receiver, name string }

func (f Finding) identity() identity {
	return identity{pkg: f.Package, kind: f.Kind, receiver: f.Receiver, name: f.Name}
}

// failing returns the findings of head that gates cover or that base lacks. It
// matches findings by package, kind, receiver, and name, so a moved declaration is not
// new.
func failing(base, head []Finding, gates []Gate) []Finding {
	old := make(set.Set[identity], len(base))
	for _, f := range base {
		old.Add(f.identity())
	}
	gated := set.New(gates...)
	var failed []Finding
	for _, f := range head {
		if gated.Contains(Gate{Module: f.Module, Usage: f.usage()}) ||
			!old.Contains(f.identity()) {
			failed = append(failed, f)
		}
	}
	return failed
}

// module returns the directory of modules that holds file.
func module(modules []string, file string) (string, error) {
	found, length := "", 0
	for _, m := range modules {
		abs, err := filepath.Abs(m)
		if err != nil {
			return "", errors.Wrapf(err, "exports: resolve module %s", m)
		}
		inside := strings.HasPrefix(file, abs+string(filepath.Separator))
		if inside && len(abs) > length {
			found, length = m, len(abs)
		}
	}
	return found, nil
}

// Check loads every package of cfg.Modules for each GOOS in cfg.GOOS and returns the
// exported identifiers that no other package uses, sorted by position. An identifier
// that only one GOOS builds is judged on that GOOS alone. Check keeps exported:
//
//   - an identifier that generated code or another package's non-test code uses, or
//     that another package's tests use when it lives in a test helper package;
//   - a method that another package can call through an interface it satisfies, one
//     that another package declares, or that another package's non-test code names,
//     holds a value of, or reaches through the API it uses;
//   - a method that template data can reach;
//   - a type that the API of a kept identifier carries;
//   - every constant of a const block when one of them is kept.
//
// Check skips generated files, test files, main packages, versions/vN packages, and
// methods of unexported types. It returns validate.ErrValidation when cfg has no
// modules, no GOOS values, an absolute module directory beside a base, or a gate of an
// unknown usage or module, and an error when a package fails to load.
func Check(ctx context.Context, cfg Config) ([]Finding, error) {
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
	known := set.New(slices.Concat(cfg.Modules, cfg.Workspace)...)
	for _, g := range cfg.Gated {
		if g.Usage != unused && g.Usage != testsOnly {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"exports: gate usage %q must be %q or %q",
				g.Usage, unused, testsOnly,
			)
		}
		if !known.Contains(g.Module) {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"exports: gate module %s must be one of the modules or workspace",
				g.Module,
			)
		}
	}
	if cfg.Base == "" {
		return check(ctx, cfg)
	}
	for _, m := range slices.Concat(cfg.Modules, cfg.Workspace) {
		if filepath.IsAbs(m) {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"exports: module %s must be relative to compare with a base",
				m,
			)
		}
	}
	baseCfg := Config{GOOS: cfg.GOOS}
	var err error
	if baseCfg.Modules, err = inBase(cfg.Base, cfg.Modules); err != nil {
		return nil, err
	}
	if baseCfg.Workspace, err = inBase(cfg.Base, cfg.Workspace); err != nil {
		return nil, err
	}
	head, err := check(ctx, cfg)
	if err != nil {
		return nil, err
	}
	var base []Finding
	if len(baseCfg.Modules) > 0 {
		if base, err = check(ctx, baseCfg); err != nil {
			return nil, err
		}
	}
	return failing(base, head, cfg.Gated), nil
}

// inBase returns the directories under base of the modules that base holds. A module
// that the base lacks is new, so each of its findings is new.
func inBase(base string, modules []string) ([]string, error) {
	var found []string
	for _, m := range modules {
		dir := filepath.Join(base, m)
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, errors.Wrapf(err, "exports: find module %s in base", m)
		}
		found = append(found, dir)
	}
	return found, nil
}

func check(ctx context.Context, cfg Config) (findings []Finding, err error) {
	modules, err := absolute(cfg.Modules)
	if err != nil {
		return nil, err
	}
	workspace, err := absolute(cfg.Workspace)
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
	work, err := writeWorkspace(dir, slices.Concat(modules, workspace))
	if err != nil {
		return nil, err
	}
	m := merge{
		findings: map[string]*Finding{},
		kept:     set.Set[string]{},
		internal: set.Set[string]{},
	}
	for _, goos := range cfg.GOOS {
		syms, err := load(ctx, work, goos, modules)
		if err != nil {
			return nil, err
		}
		m.add(syms)
	}
	return m.list(cfg.Modules)
}

// merge combines the symbols that each GOOS build loads into one finding per symbol.
type merge struct {
	findings map[string]*Finding
	// kept holds the keys of the symbols that any build keeps exported.
	kept set.Set[string]
	// internal holds the keys of the symbols that any build uses outside tests.
	internal set.Set[string]
}

func (m merge) add(syms map[string]*symbol) {
	for k, s := range syms {
		if s.kept {
			m.kept.Add(k)
			continue
		}
		f, ok := m.findings[k]
		if !ok {
			f = &Finding{
				Position: s.pos,
				Package:  s.pkg,
				Kind:     s.kind,
				Name:     s.obj.Name(),
			}
			if s.kind == "method" {
				f.Receiver = receiver(s.method()).Name()
			}
			m.findings[k] = f
		}
		f.TestsOnly = f.TestsOnly || s.tested
		if s.internal {
			m.internal.Add(k)
		}
	}
}

// list returns the findings that no build keeps, sorted by position.
func (m merge) list(modules []string) ([]Finding, error) {
	findings := make([]Finding, 0, len(m.findings))
	for k, f := range m.findings {
		if m.kept.Contains(k) {
			continue
		}
		f.TestsOnly = f.TestsOnly && !m.internal.Contains(k)
		var err error
		if f.Module, err = module(modules, f.Position.Filename); err != nil {
			return nil, err
		}
		findings = append(findings, *f)
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
	used := set.Set[string]{}
	for _, m := range modules {
		if !used.Contains(m) {
			used.Add(m)
			fmt.Fprintf(&b, "\t%q\n", m)
		}
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

// method returns the method that s, a symbol of kind "method", declares.
func (s *symbol) method() *types.Func {
	fn, ok := s.obj.(*types.Func)
	if !ok {
		panic(fmt.Sprintf("method symbol %s is a %T", s.obj.Name(), s.obj))
	}
	return fn
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
			packages.NeedTypes | packages.NeedTypesInfo | packages.NeedImports |
			packages.NeedModule |
			// Type checking every dependency from source costs less than compiling
			// export data for it.
			packages.NeedDeps,
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
	d := declarer{p: p, syms: syms}
	for _, f := range p.Syntax {
		file := p.Fset.File(f.Pos()).Name()
		if generated.Contains(file) || strings.HasSuffix(file, "_test.go") {
			continue
		}
		for _, decl := range f.Decls {
			d.decl(file, decl)
		}
	}
}

// declarer adds the exported identifiers that the declarations of p declare to syms.
type declarer struct {
	p    *packages.Package
	syms map[string]*symbol
}

func (d declarer) decl(file string, decl ast.Decl) {
	switch decl := decl.(type) {
	case *ast.FuncDecl:
		if decl.Recv == nil {
			d.add(decl.Name, "func", "")
		} else if fn, ok := d.p.TypesInfo.Defs[decl.Name].(*types.Func); ok &&
			receiverExported(fn) {
			d.add(decl.Name, "method", "")
		}
	case *ast.GenDecl:
		group := ""
		if decl.Tok == token.CONST && decl.Lparen.IsValid() {
			group = fmt.Sprintf("%s:%d", file, d.p.Fset.Position(decl.Pos()).Line)
		}
		for _, spec := range decl.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				d.add(s.Name, "type", "")
			case *ast.ValueSpec:
				for _, n := range s.Names {
					d.add(n, decl.Tok.String(), group)
				}
			}
		}
	}
}

func (d declarer) add(id *ast.Ident, kind, group string) {
	obj := d.p.TypesInfo.Defs[id]
	if obj == nil || !id.IsExported() {
		return
	}
	pos := d.p.Fset.Position(obj.Pos())
	d.syms[key(pos, obj)] = &symbol{
		obj: obj, pkg: d.p.PkgPath, kind: kind, group: group, pos: pos,
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
// named testutil, test, fake, or mock, or one that ends in mock.
func isTestUtility(path string) bool {
	for elem := range strings.SplitSeq(path, "/") {
		switch {
		case elem == "testutil", elem == "test", elem == "fake",
			strings.HasSuffix(elem, "mock"):
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
	ifaces := collectInterfaces(fset, pkgs)
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
		case s.kind == "method" && (ifaces.satisfied(s.method()) ||
			templated.Contains(receiver(s.method()))):
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
