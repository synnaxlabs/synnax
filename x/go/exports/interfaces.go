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
	"go/types"

	"github.com/synnaxlabs/x/set"
	"golang.org/x/tools/go/packages"
)

// interfaces indexes every interface that the loaded code and its dependencies
// declare or spell out, by method name.
type interfaces struct {
	plain map[string][]*types.Interface
	// generic holds the interfaces of generic types. A method satisfies one when its
	// type has every method of the interface with the same arity.
	generic map[string][]*types.Interface
}

func collectInterfaces(pkgs []*packages.Package) interfaces {
	ifaces := interfaces{
		plain:   map[string][]*types.Interface{},
		generic: map[string][]*types.Interface{},
	}
	index := func(byName map[string][]*types.Interface, it *types.Interface) {
		for m := range it.Methods() {
			byName[m.Name()] = append(byName[m.Name()], it)
		}
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
			if !ok {
				continue
			}
			it, ok := tn.Type().Underlying().(*types.Interface)
			if !ok {
				continue
			}
			if named, ok := tn.Type().(*types.Named); ok &&
				named.TypeParams().Len() > 0 {
				index(ifaces.generic, it)
			} else {
				index(ifaces.plain, it)
			}
		}
		for _, imp := range tp.Imports() {
			walk(imp)
		}
	}
	for _, p := range pkgs {
		walk(p.Types)
		for _, tv := range p.TypesInfo.Types {
			if tv.Type == nil {
				continue
			}
			if it, ok := tv.Type.Underlying().(*types.Interface); ok {
				index(ifaces.plain, it)
			}
		}
	}
	return ifaces
}

// satisfied reports whether the method m must stay exported for an interface.
func (ifaces interfaces) satisfied(m *types.Func) bool {
	recv := m.Signature().Recv().Type()
	if ptr, ok := recv.(*types.Pointer); ok {
		recv = ptr.Elem()
	}
	named, ok := types.Unalias(recv).(*types.Named)
	if !ok {
		return false
	}
	if named.TypeParams().Len() > 0 {
		return len(ifaces.plain[m.Name()])+len(ifaces.generic[m.Name()]) > 0
	}
	for _, it := range ifaces.generic[m.Name()] {
		if coversByArity(named, it) {
			return true
		}
	}
	ptr := types.NewPointer(named)
	for _, it := range ifaces.plain[m.Name()] {
		if types.Implements(named, it) || types.Implements(ptr, it) {
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
