// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import (
	"strings"

	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// Transient returns the names of members reachable from the file's @go marshal
// declarations only through omitted fields. These types never reach the stored
// wire format: they ride the version layout solely because the current record
// references them, so they track the live shape, may redeclare identically to
// a predecessor, and resolve their references at read time. A file with no
// marshal declarations (a resource persisted through another resource's bytes)
// has no transient members.
func (f *File) Transient() set.Set[string] {
	roots := make([]string, 0)
	for _, t := range f.Defined {
		if t.Synthetic {
			continue
		}
		if dom, ok := t.Domains["go"]; ok {
			if _, marshal := dom.Expressions.Find("marshal"); marshal {
				roots = append(roots, t.Name)
			}
		}
	}
	stored := f.closure(roots, true)
	all := f.closure(roots, false)
	transient := make(set.Set[string])
	for name := range all {
		if !stored.Contains(name) {
			transient.Add(name)
		}
	}
	return transient
}

// closure walks the file's local reference graph from the given roots. When
// persistedOnly is set, omitted struct fields do not contribute edges.
func (f *File) closure(roots []string, persistedOnly bool) set.Set[string] {
	local := make(map[string]resolution.Type, len(f.Defined))
	for _, t := range f.Defined {
		local[t.Name] = t
	}
	out := make(set.Set[string])
	var visit func(name string)
	edge := func(ref resolution.TypeRef) {
		for _, name := range f.localNames(ref) {
			visit(name)
		}
	}
	visit = func(name string) {
		if out.Contains(name) {
			return
		}
		t, ok := local[name]
		if !ok {
			return
		}
		out.Add(name)
		switch form := t.Form.(type) {
		case resolution.StructForm:
			for _, fd := range form.Fields {
				if persistedOnly && fieldOmitted(fd) {
					continue
				}
				edge(fd.Type)
			}
			for _, e := range form.Extends {
				edge(e)
			}
			for _, p := range form.TypeParams {
				if p.Constraint != nil {
					edge(*p.Constraint)
				}
				if p.Default != nil {
					edge(*p.Default)
				}
			}
		case resolution.EnumForm:
			for _, e := range form.Extends {
				edge(e)
			}
		case resolution.UnionForm:
			for _, v := range form.Variants {
				edge(v.Type)
			}
			for _, e := range form.Extends {
				edge(e)
			}
		case resolution.DistinctForm:
			edge(form.Base)
		case resolution.AliasForm:
			edge(form.Target)
		}
	}
	for _, name := range roots {
		visit(name)
	}
	return out
}

// localNames extracts the file-local member names a reference points at,
// walking type arguments. A chain-sibling qualifier (v0.Key) and the
// resource's own qualifier both resolve locally.
func (f *File) localNames(ref resolution.TypeRef) []string {
	var names []string
	var walk func(ref resolution.TypeRef)
	walk = func(ref resolution.TypeRef) {
		if ref.TypeParam == nil {
			if ns, name, found := strings.Cut(ref.Name, "."); found {
				if chainSiblingNS(ns) || ns == f.Chain.Resource {
					names = append(names, name)
				}
			} else {
				names = append(names, ref.Name)
			}
		}
		for _, a := range ref.TypeArgs {
			walk(a)
		}
	}
	walk(ref)
	return names
}

// fieldOmitted reports whether a struct field carries @go marshal omit.
func fieldOmitted(fd resolution.Field) bool {
	dom, ok := fd.Domains["go"]
	if !ok {
		return false
	}
	expr, ok := dom.Expressions.Find("marshal")
	if !ok {
		return false
	}
	for _, v := range expr.Values {
		if v.IdentValue == "omit" || v.StringValue == "omit" {
			return true
		}
	}
	return false
}

// chainSiblingNS reports whether ns is a chain-sibling qualifier ("v0").
func chainSiblingNS(ns string) bool {
	if len(ns) < 2 || ns[0] != 'v' {
		return false
	}
	for _, r := range ns[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
