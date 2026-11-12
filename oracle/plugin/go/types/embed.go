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
	w := heldWalk{table: data.table, visited: set.New[string]()}
	refs, err := w.refs(field.Type, field.Optional, 0)
	if err != nil {
		return errors.Wrapf(err, "%s.%s", owner, field.Name)
	}
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

// heldWalk collects the types that a field holds.
type heldWalk struct {
	table *resolution.Table
	// visited holds the instantiations of generic structs already walked.
	visited set.Set[string]
}

// maxGenericDepth bounds how deep generic structs nest in one field. Only a generic
// struct that instantiates itself with ever larger type arguments reaches it.
const maxGenericDepth = 32

// refs returns the types that ref holds in slices, map values, and the fields of a
// generic struct, and ref itself when the field is optional. A generic struct sets no
// defaults, so each of its fields counts as held. refs looks through aliases and
// distinct types to the collection or generic struct they name. It returns an error
// when generic structs nest deeper than maxGenericDepth.
func (w heldWalk) refs(
	ref resolution.TypeRef,
	optional bool,
	depth int,
) ([]resolution.TypeRef, error) {
	if under, ok := underlying(ref, w.table); ok {
		ref = under
	}
	switch {
	case ref.Name == "Array" && len(ref.TypeArgs) == 1:
		return w.refs(ref.TypeArgs[0], true, depth)
	case ref.Name == "Map" && len(ref.TypeArgs) == 2:
		return w.refs(ref.TypeArgs[1], true, depth)
	}
	var out []resolution.TypeRef
	if optional {
		out = append(out, ref)
	}
	t, form, ok := genericStruct(ref, w.table)
	key := instanceKey(ref)
	if !ok || w.visited.Contains(key) {
		return out, nil
	}
	if depth == maxGenericDepth {
		return nil, errors.Newf("%s instantiates itself without end", t.Name)
	}
	w.visited.Add(key)
	args := resolution.TypeArgMap(form.TypeParams, ref.TypeArgs)
	for _, f := range resolution.UnifiedFields(t, w.table) {
		held, err := w.refs(
			resolution.SubstituteTypeRef(f.Type, args),
			true,
			depth+1,
		)
		if err != nil {
			return nil, err
		}
		out = append(out, held...)
	}
	return out, nil
}

// instanceKey identifies ref by its name and its type arguments.
func instanceKey(ref resolution.TypeRef) string {
	if len(ref.TypeArgs) == 0 {
		return ref.Name
	}
	args := make([]string, len(ref.TypeArgs))
	for i, a := range ref.TypeArgs {
		args[i] = instanceKey(a)
	}
	return ref.Name + "<" + strings.Join(args, ",") + ">"
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
