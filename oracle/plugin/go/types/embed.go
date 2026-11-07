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
// defaults in a slice, a map value, or an optional field. An embedded type takes no
// decode hooks, so a decode leaves the defaults of such a value unset.
func checkHeldEmbedded(owner string, field resolution.Field, data *templateData) error {
	for _, ref := range heldRefs(field.Type, field.Optional) {
		for _, t := range resolvedAndDealiased(ref, data.table) {
			key := embedKey(output.GetPath(t, "go"), naming.GetGoName(t))
			if data.embedded.Contains(key) &&
				typeSetsDefaults(ref, data, set.New[string]()) {
				return errors.Newf(
					"%s.%s: %s has defaults and another struct embeds it, so it "+
						"cannot be held in a slice, a map, or an optional field",
					owner, field.Name, t.Name,
				)
			}
		}
	}
	return nil
}

// heldRefs returns the types that ref holds in slices and map values, and ref itself
// when the field is optional.
func heldRefs(ref resolution.TypeRef, optional bool) []resolution.TypeRef {
	switch {
	case ref.Name == "Array" && len(ref.TypeArgs) == 1:
		return heldRefs(ref.TypeArgs[0], true)
	case ref.Name == "Map" && len(ref.TypeArgs) == 2:
		return heldRefs(ref.TypeArgs[1], true)
	case optional:
		return []resolution.TypeRef{ref}
	}
	return nil
}
