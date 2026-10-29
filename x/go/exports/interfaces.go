// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package exports

import (
	"go/token"
	"go/types"
	"strings"

	"github.com/synnaxlabs/x/set"
	"golang.org/x/tools/go/packages"
)

// iface is an interface and the package that declares or spells it out.
type iface struct {
	it *types.Interface
	// pkg is the import path of the package, and empty for error.
	pkg string
	// key identifies a declared interface, and is empty for an interface literal.
	key string
}

// interfaces indexes every interface that the loaded code and its dependencies
// declare or spell out, by method name.
type interfaces struct {
	plain map[string][]iface
	// generic holds the interfaces of generic types. A method satisfies one when its
	// type has every method of the interface with the same arity.
	generic map[string][]iface
	// embedders holds, for each named type, the named struct types that embed it.
	embedders map[*types.TypeName][]*types.Named
	// crossed holds the keys of the interfaces that another package's non-test code
	// names.
	crossed set.Set[string]
}

func collectInterfaces(fset *token.FileSet, pkgs []*packages.Package) interfaces {
	ifaces := interfaces{
		plain:     map[string][]iface{},
		generic:   map[string][]iface{},
		embedders: map[*types.TypeName][]*types.Named{},
		crossed:   set.Set[string]{},
	}
	index := func(byName map[string][]iface, i iface) {
		for m := range i.it.Methods() {
			byName[m.Name()] = append(byName[m.Name()], i)
		}
	}
	declared := func(tn *types.TypeName) iface {
		i := iface{it: tn.Type().Underlying().(*types.Interface)}
		if tn.Pkg() != nil {
			i.pkg = tn.Pkg().Path()
			i.key = key(fset.Position(tn.Pos()), tn)
		}
		return i
	}
	roots := set.Set[*types.Package]{}
	for _, p := range pkgs {
		roots.Add(p.Types)
	}
	seen := set.Set[*types.Package]{}
	var walk func(*types.Package)
	walk = func(tp *types.Package) {
		if tp == nil || seen.Contains(tp) {
			return
		}
		seen.Add(tp)
		for _, name := range tp.Scope().Names() {
			tn, ok := tp.Scope().Lookup(name).(*types.TypeName)
			// An unexported interface of a dependency keeps no method exported.
			if !ok || (!roots.Contains(tp) && !tn.Exported()) {
				continue
			}
			if _, ok := tn.Type().Underlying().(*types.Interface); !ok {
				continue
			}
			if named, ok := tn.Type().(*types.Named); ok &&
				named.TypeParams().Len() > 0 {
				index(ifaces.generic, declared(tn))
			} else {
				index(ifaces.plain, declared(tn))
			}
		}
		for _, imp := range tp.Imports() {
			walk(imp)
		}
	}
	for _, p := range pkgs {
		walk(p.Types)
		ifaces.indexEmbedders(p.Types)
		user := strings.TrimSuffix(p.PkgPath, "_test")
		for _, tv := range p.TypesInfo.Types {
			if tv.Type == nil {
				continue
			}
			it, ok := tv.Type.Underlying().(*types.Interface)
			if !ok {
				continue
			}
			if named, ok := tv.Type.(*types.Named); ok {
				index(ifaces.plain, declared(named.Origin().Obj()))
			} else {
				index(ifaces.plain, iface{it: it, pkg: user})
			}
		}
		for id, obj := range p.TypesInfo.Uses {
			tn, ok := obj.(*types.TypeName)
			if !ok || tn.Pkg() == nil || tn.Pkg().Path() == user ||
				strings.HasSuffix(fset.Position(id.Pos()).Filename, "_test.go") {
				continue
			}
			if _, ok := tn.Type().Underlying().(*types.Interface); ok {
				ifaces.crossed.Add(key(fset.Position(tn.Pos()), origin(tn)))
			}
		}
	}
	return ifaces
}

// dispatches reports whether i can call a method of package pkg from another package:
// another package declares i, or another package's non-test code names it.
func (ifaces interfaces) dispatches(i iface, pkg string) bool {
	return i.pkg != pkg || ifaces.crossed.Contains(i.key)
}

// satisfied reports whether the method m satisfies an interface that another package
// calls it through. Such a call is a use of m.
func (ifaces interfaces) satisfied(m *types.Func) bool {
	recv := m.Signature().Recv().Type()
	if ptr, ok := recv.(*types.Pointer); ok {
		recv = ptr.Elem()
	}
	named, ok := types.Unalias(recv).(*types.Named)
	if !ok {
		return false
	}
	pending := []*types.Named{named.Origin()}
	seen := set.Set[*types.TypeName]{}
	for len(pending) > 0 {
		t := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen.Contains(t.Obj()) {
			continue
		}
		seen.Add(t.Obj())
		if ifaces.satisfiedBy(t, m) {
			return true
		}
		pending = append(pending, ifaces.embedders[t.Obj()]...)
	}
	return false
}

// satisfiedBy reports whether t, through a method set that selects m, satisfies an
// interface that has a method named like m.
func (ifaces interfaces) satisfiedBy(t *types.Named, m *types.Func) bool {
	ptr := types.NewPointer(t)
	sel := types.NewMethodSet(ptr).Lookup(m.Pkg(), m.Name())
	if sel == nil || sel.Obj().(*types.Func).Origin() != m {
		return false
	}
	pkg := m.Pkg().Path()
	for _, i := range ifaces.generic[m.Name()] {
		if ifaces.dispatches(i, pkg) && coversByArity(t, i.it) {
			return true
		}
	}
	generic := t.TypeParams().Len() > 0
	for _, i := range ifaces.plain[m.Name()] {
		if !ifaces.dispatches(i, pkg) {
			continue
		}
		if generic && coversByArity(t, i.it) ||
			!generic && (types.Implements(t, i.it) || types.Implements(ptr, i.it)) {
			return true
		}
	}
	return false
}

func coversByArity(named *types.Named, it *types.Interface) bool {
	methods := types.NewMethodSet(types.NewPointer(named))
	for im := range it.Methods() {
		sel := methods.Lookup(im.Pkg(), im.Name())
		if sel == nil {
			return false
		}
		got := sel.Obj().Type().(*types.Signature)
		want := im.Signature()
		if got.Params().Len() != want.Params().Len() ||
			got.Results().Len() != want.Results().Len() {
			return false
		}
	}
	return true
}

// reach calls f with each named type that t carries in its signature, element types,
// exported struct fields, and interface methods.
func reach(t types.Type, seen set.Set[types.Type], f func(*types.TypeName)) {
	if t == nil || seen.Contains(t) {
		return
	}
	seen.Add(t)
	switch t := t.(type) {
	case *types.Named:
		f(t.Origin().Obj())
		for arg := range t.TypeArgs().Types() {
			reach(arg, seen, f)
		}
		reach(t.Underlying(), seen, f)
	case *types.Alias:
		reach(types.Unalias(t), seen, f)
	case *types.Pointer:
		reach(t.Elem(), seen, f)
	case *types.Slice:
		reach(t.Elem(), seen, f)
	case *types.Array:
		reach(t.Elem(), seen, f)
	case *types.Chan:
		reach(t.Elem(), seen, f)
	case *types.Map:
		reach(t.Key(), seen, f)
		reach(t.Elem(), seen, f)
	case *types.Signature:
		for v := range t.Params().Variables() {
			reach(v.Type(), seen, f)
		}
		for v := range t.Results().Variables() {
			reach(v.Type(), seen, f)
		}
	case *types.Struct:
		for field := range t.Fields() {
			if field.Exported() || field.Embedded() {
				reach(field.Type(), seen, f)
			}
		}
	case *types.Interface:
		for m := range t.Methods() {
			reach(m.Type(), seen, f)
		}
		for e := range t.EmbeddedTypes() {
			reach(e, seen, f)
		}
	case *types.Union:
		for term := range t.Terms() {
			reach(term.Type(), seen, f)
		}
	case *types.TypeParam:
		reach(t.Constraint(), seen, f)
	}
}

func (ifaces interfaces) indexEmbedders(tp *types.Package) {
	for _, name := range tp.Scope().Names() {
		outer, ok := tp.Scope().Lookup(name).Type().(*types.Named)
		if !ok {
			continue
		}
		st, ok := outer.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		for field := range st.Fields() {
			if !field.Embedded() {
				continue
			}
			t := field.Type()
			if ptr, ok := t.(*types.Pointer); ok {
				t = ptr.Elem()
			}
			if inner, ok := types.Unalias(t).(*types.Named); ok {
				key := inner.Origin().Obj()
				ifaces.embedders[key] = append(ifaces.embedders[key], outer)
			}
		}
	}
}
