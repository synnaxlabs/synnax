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
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/synnaxlabs/oracle/resolution"
)

// copyType deep-copies a resolved type into namespace ns, rewriting every reference
// name through rw. Domain maps are shared: they are read-only after analysis. Actions
// are not copied, because a version file declares none.
func copyType(
	t resolution.Type, ns string, rw func(string) string,
) resolution.Type {
	t.Namespace = ns
	t.QualifiedName = ns + "." + t.Name
	t.Form = copyForm(t.Form, rw)
	if t.Filter != nil {
		f := *t.Filter
		f.Target = copyRef(f.Target, rw)
		f.Lines = slices.Clone(f.Lines)
		for i := range f.Lines {
			f.Lines[i].Type = copyRef(f.Lines[i].Type, rw)
		}
		t.Filter = &f
	}
	return t
}

func copyForm(form resolution.TypeForm, rw func(string) string) resolution.TypeForm {
	switch f := form.(type) {
	case resolution.StructForm:
		f.Fields = copyFields(f.Fields, rw)
		f.TypeParams = copyParams(f.TypeParams, rw)
		f.Extends = copyRefs(f.Extends, rw)
		f.OmittedFields = slices.Clone(f.OmittedFields)
		return f
	case resolution.EnumForm:
		f.Values = slices.Clone(f.Values)
		f.Declared = slices.Clone(f.Declared)
		f.Extends = copyRefs(f.Extends, rw)
		return f
	case resolution.UnionForm:
		f.Variants = copyVariants(f.Variants, rw)
		f.Declared = copyVariants(f.Declared, rw)
		f.Extends = copyRefs(f.Extends, rw)
		f.Included = copyRefs(f.Included, rw)
		if f.BorrowsEnum() {
			f.Enum = copyRef(f.Enum, rw)
		}
		return f
	case resolution.DistinctForm:
		f.Base = copyRef(f.Base, rw)
		f.TypeParams = copyParams(f.TypeParams, rw)
		return f
	case resolution.AliasForm:
		f.Target = copyRef(f.Target, rw)
		f.TypeParams = copyParams(f.TypeParams, rw)
		return f
	default:
		return form
	}
}

func copyVariants(
	variants []resolution.UnionVariant, rw func(string) string,
) []resolution.UnionVariant {
	out := slices.Clone(variants)
	for i := range out {
		out[i].Type = copyRef(out[i].Type, rw)
	}
	return out
}

func copyFields(fields []resolution.Field, rw func(string) string) []resolution.Field {
	out := slices.Clone(fields)
	for i := range out {
		out[i].Type = copyRef(out[i].Type, rw)
		out[i].Domains = maps.Clone(out[i].Domains)
	}
	return out
}

func copyRefs(refs []resolution.TypeRef, rw func(string) string) []resolution.TypeRef {
	out := slices.Clone(refs)
	for i := range out {
		out[i] = copyRef(out[i], rw)
	}
	return out
}

func copyRef(ref resolution.TypeRef, rw func(string) string) resolution.TypeRef {
	if ref.TypeParam != nil {
		p := copyParam(*ref.TypeParam, rw)
		ref.TypeParam = &p
	}
	if ref.ArraySize != nil {
		size := *ref.ArraySize
		ref.ArraySize = &size
	}
	if ref.Name != "" {
		ref.Name = rw(ref.Name)
	}
	ref.TypeArgs = copyRefs(ref.TypeArgs, rw)
	return ref
}

func copyParams(
	params []resolution.TypeParam, rw func(string) string,
) []resolution.TypeParam {
	out := slices.Clone(params)
	for i := range out {
		out[i] = copyParam(out[i], rw)
	}
	return out
}

func copyParam(p resolution.TypeParam, rw func(string) string) resolution.TypeParam {
	if p.Constraint != nil {
		c := copyRef(*p.Constraint, rw)
		p.Constraint = &c
	}
	if p.Default != nil {
		d := copyRef(*p.Default, rw)
		p.Default = &d
	}
	return p
}

// versionNS matches a chain-sibling namespace qualifier ("v0", "v12").
var versionNS = regexp.MustCompile(`^v(0|[1-9][0-9]*)$`)

// rewriter maps a resolved reference name: chain-sibling qualifiers (vK) map to selfNS
// when it is non-empty, a qualified name that rewrites holds maps to its value, and any
// other dependency resource qualifier maps through rewrites. Unqualified names
// (primitives) pass through untouched.
func rewriter(selfNS string, rewrites map[string]string) func(string) string {
	return func(name string) string {
		i := strings.IndexByte(name, '.')
		if i < 0 {
			return name
		}
		prefix := name[:i]
		if selfNS != "" && versionNS.MatchString(prefix) {
			return selfNS + name[i:]
		}
		if to, ok := rewrites[name]; ok {
			return to
		}
		if to, ok := rewrites[prefix]; ok {
			return to + name[i:]
		}
		return name
	}
}

// pinRewrites maps each resource qualifier that f pins to the dependency's internal
// pinned-surface namespace. When a pin shares its qualifier with another import of f,
// it maps each qualified name that the pinned surface declares instead, so a name goes
// to the pin that declares it.
func (r *Resolver) pinRewrites(f *File) (map[string]string, error) {
	imports := make(map[string]int, len(f.Pins)+len(f.LivePaths))
	for live := range f.Pins {
		imports[resourceOf(live)]++
	}
	for _, p := range f.LivePaths {
		imports[resourceOf(p)]++
	}
	m := make(map[string]string, len(f.Pins))
	for live, v := range f.Pins {
		ns, to := resourceOf(live), DepNS(live, v)
		if imports[ns] == 1 {
			m[ns] = to
			continue
		}
		names, err := r.surfaceNames(live, v)
		if err != nil {
			return nil, err
		}
		for name := range names {
			m[ns+"."+name] = to + "." + name
		}
	}
	return m, nil
}
