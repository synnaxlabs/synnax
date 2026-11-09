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
	"strings"

	"github.com/synnaxlabs/oracle/domain/omit"
	"github.com/synnaxlabs/oracle/plugin/go/internal/naming"
	"github.com/synnaxlabs/oracle/plugin/output"
	"github.com/synnaxlabs/oracle/plugin/resolver"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// CanEmbed reports whether a struct's parents can be Go-embedded rather than flattened
// into the child. Beyond the conditions resolver.CanUseInheritance covers, embedding
// fails when two parents promote the same name, because Go rejects an ambiguous
// promoted name both as a selector and as a composite literal key.
func CanEmbed(form resolution.StructForm, table *resolution.Table) bool {
	if !resolver.CanUseInheritance(form, "go", table) {
		return false
	}
	promoted := make(set.Set[string])
	for _, ext := range form.Extends {
		parent, _, ok := resolution.Dealias(ext, table)
		if !ok {
			continue
		}
		for _, name := range promotedGoNames(parent, table) {
			if promoted.Contains(name) {
				return false
			}
			promoted.Add(name)
		}
	}
	return true
}

// promotedGoNames returns the names a parent promotes into a struct that embeds it. An
// embedded parent promotes its own fields and the type names of the parents it embeds
// in turn; a flattened one promotes its whole unified field list.
func promotedGoNames(parent resolution.Type, table *resolution.Table) []string {
	form, ok := parent.Form.(resolution.StructForm)
	if !ok {
		return nil
	}
	if !CanEmbed(form, table) {
		fields := resolution.UnifiedFields(parent, table)
		names := make([]string, 0, len(fields))
		for _, f := range fields {
			names = append(names, naming.GetFieldName(f))
		}
		return names
	}
	defaultOnly := resolver.DefaultOnlyOverrides(form.Extends, form.Fields, table)
	names := make([]string, 0, len(form.Extends)+len(form.Fields))
	for _, ext := range form.Extends {
		if grandparent, ok := ext.Resolve(table); ok {
			names = append(names, naming.GetGoName(grandparent))
		}
	}
	for _, f := range form.Fields {
		if !defaultOnly.Contains(f.Name) {
			names = append(names, naming.GetFieldName(f))
		}
	}
	return names
}

// embedKey identifies a generated Go type across every version of its package, so a
// type embedded in one version counts as embedded in all of them.
func embedKey(outputPath, goName string) string {
	if root, dir, ok := strings.Cut(outputPath, "/versions/"); ok &&
		!strings.Contains(dir, "/") {
		outputPath = root
	}
	return outputPath + "." + goName
}

// embeddedTypes returns the embedKey of every struct that a generated Go struct embeds
// in any of tables: struct parents, union bases, and union variant payloads. JSON
// rejects an embedded type with decode methods, and msgpack stops flattening it, so
// these types take no decode hooks.
func embeddedTypes(tables ...*resolution.Table) set.Set[string] {
	out := make(set.Set[string])
	for _, table := range tables {
		add := func(refs ...resolution.TypeRef) {
			for _, ref := range refs {
				// An alias names the same Go type as its target.
				for _, t := range resolvedAndDealiased(ref, table) {
					out.Add(embedKey(output.GetPath(t, "go"), naming.GetGoName(t)))
				}
			}
		}
		for _, t := range table.Types {
			if omit.IsSkipped(t, "go") {
				continue
			}
			switch form := t.Form.(type) {
			case resolution.StructForm:
				if len(form.Extends) > 0 && CanEmbed(form, table) {
					add(form.Extends...)
				}
			case resolution.UnionForm:
				add(form.Extends...)
				for _, v := range form.Variants {
					inherited, _ := resolver.VariantBases(form, v, "go", table)
					add(inherited...)
					if !v.Inline {
						add(v.Type)
					}
				}
			}
		}
	}
	return out
}

func resolvedAndDealiased(
	ref resolution.TypeRef,
	table *resolution.Table,
) []resolution.Type {
	var out []resolution.Type
	if t, ok := ref.Resolve(table); ok {
		out = append(out, t)
	}
	if t, _, ok := resolution.Dealias(ref, table); ok {
		out = append(out, t)
	}
	return out
}

// checkHeldEmbedded returns an error when field holds an embedded type that has
// defaults in a slice, a map value, an optional field, or a generic struct. An embedded
// type takes no decode hooks, so a decode leaves the defaults of such a value unset.
func checkHeldEmbedded(owner string, field resolution.Field, data *templateData) error {
	refs := heldRefs(field.Type, field.Optional, data.table, set.New[string]())
	for _, ref := range refs {
		for _, t := range resolvedAndDealiased(ref, data.table) {
			key := embedKey(output.GetPath(t, "go"), naming.GetGoName(t))
			if data.embedded.Contains(key) &&
				typeSetsDefaults(ref, data, set.New[string]()) {
				return errors.Newf(
					"%s.%s: %s has defaults and another struct embeds it, so it "+
						"cannot be held in a slice, a map, an optional field, or a "+
						"generic struct",
					owner, field.Name, t.Name,
				)
			}
		}
	}
	return nil
}

// heldRefs returns the types that ref holds in slices, map values, and the fields of a
// generic struct, and ref itself when the field is optional. A generic struct sets no
// defaults, so each of its type parameter fields counts as held. heldRefs looks
// through aliases and distinct types to the collection or generic struct they name.
func heldRefs(
	ref resolution.TypeRef,
	optional bool,
	table *resolution.Table,
	visited set.Set[string],
) []resolution.TypeRef {
	if under, ok := underlying(ref, table); ok {
		ref = under
	}
	switch {
	case ref.Name == "Array" && len(ref.TypeArgs) == 1:
		return heldRefs(ref.TypeArgs[0], true, table, visited)
	case ref.Name == "Map" && len(ref.TypeArgs) == 2:
		return heldRefs(ref.TypeArgs[1], true, table, visited)
	}
	var out []resolution.TypeRef
	if optional {
		out = append(out, ref)
	}
	t, form, ok := genericStruct(ref, table)
	if !ok || visited.Contains(ref.Name) {
		return out
	}
	visited.Add(ref.Name)
	args := resolution.TypeArgMap(form.TypeParams, ref.TypeArgs)
	for _, f := range resolution.UnifiedFields(t, table) {
		out = append(out, heldRefs(
			resolution.SubstituteTypeRef(f.Type, args),
			f.Optional || f.Type.IsTypeParam(),
			table,
			visited,
		)...)
	}
	return out
}

// underlying follows ref through aliases and distinct types, and returns the slice,
// map, or generic struct instantiation that it names.
func underlying(
	ref resolution.TypeRef,
	table *resolution.Table,
) (resolution.TypeRef, bool) {
	visited := set.New[string]()
	for !ref.IsTypeParam() && !visited.Contains(ref.Name) {
		if ref.Name == "Array" || ref.Name == "Map" {
			return ref, true
		}
		if _, _, ok := genericStruct(ref, table); ok {
			return ref, true
		}
		visited.Add(ref.Name)
		t, ok := ref.Resolve(table)
		if !ok {
			break
		}
		switch form := t.Form.(type) {
		case resolution.AliasForm:
			ref = resolution.SubstituteTypeRef(
				form.Target, resolution.TypeArgMap(form.TypeParams, ref.TypeArgs),
			)
		case resolution.DistinctForm:
			ref = resolution.SubstituteTypeRef(
				form.Base, resolution.TypeArgMap(form.TypeParams, ref.TypeArgs),
			)
		default:
			return resolution.TypeRef{}, false
		}
	}
	return resolution.TypeRef{}, false
}

// genericStruct returns the generic struct that ref instantiates.
func genericStruct(
	ref resolution.TypeRef,
	table *resolution.Table,
) (resolution.Type, resolution.StructForm, bool) {
	if len(ref.TypeArgs) == 0 {
		return resolution.Type{}, resolution.StructForm{}, false
	}
	t, ok := ref.Resolve(table)
	if !ok {
		return resolution.Type{}, resolution.StructForm{}, false
	}
	form, ok := t.Form.(resolution.StructForm)
	if !ok || !form.IsGeneric() {
		return resolution.Type{}, resolution.StructForm{}, false
	}
	return t, form, true
}
