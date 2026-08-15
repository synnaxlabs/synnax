// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package resolver

import (
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// HasFieldConflicts returns true if multiple parents have overlapping field names.
// This is used to determine if language inheritance/embedding can be used safely.
// When field names conflict across parents, inheritance cannot be used and fields
// must be flattened into the child struct.
func HasFieldConflicts(extends []resolution.TypeRef, table *resolution.Table) bool {
	if len(extends) < 2 {
		return false
	}
	seen := make(set.Set[string])
	for _, ext := range extends {
		parent, ok := ext.Resolve(table)
		if !ok {
			continue
		}
		for _, f := range resolution.UnifiedFields(parent, table) {
			if seen.Contains(f.Name) {
				return true
			}
			seen.Add(f.Name)
		}
	}
	return false
}

// HasDomainOmissions reports whether the struct removes a domain inherited from a
// parent field with `-@domain`. A removal cannot be expressed through language
// inheritance/embedding — the embedded parent still carries the domain — so the
// struct must be flattened to drop it. Typeless overrides are resolved into
// complete fields by the analyzer before generation, so they do not require this.
func HasDomainOmissions(form resolution.StructForm) bool {
	for _, f := range form.Fields {
		if len(f.OmittedDomains) > 0 {
			return true
		}
	}
	return false
}

// InheritedFields returns the fields the given parents contribute, keyed by name, with
// each parent's type arguments substituted. The leftmost parent wins a name collision,
// matching resolution.UnifiedFields.
func InheritedFields(
	extends []resolution.TypeRef,
	table *resolution.Table,
) map[string]resolution.Field {
	out := make(map[string]resolution.Field)
	for _, ext := range extends {
		parent, ok := ext.Resolve(table)
		if !ok {
			continue
		}
		parentForm, ok := parent.Form.(resolution.StructForm)
		if !ok {
			continue
		}
		typeArgs := make(map[string]resolution.TypeRef, len(parentForm.TypeParams))
		for i, tp := range parentForm.TypeParams {
			if i < len(ext.TypeArgs) {
				typeArgs[tp.Name] = ext.TypeArgs[i]
			}
		}
		for _, f := range resolution.UnifiedFields(parent, table) {
			if _, seen := out[f.Name]; seen {
				continue
			}
			f.Type = resolution.SubstituteTypeRef(f.Type, typeArgs)
			out[f.Name] = f
		}
	}
	return out
}

// DefaultOnlyOverrides returns the names of fields that redeclare a field inherited
// from extends to change only its default. Every language can express such an override
// while still inheriting: the field keeps its declared type and optionality, so only
// the value filled in when it is absent differs. A union variant passes the union's
// bases and its own, since both contribute to the variant it generates.
func DefaultOnlyOverrides(
	extends []resolution.TypeRef,
	fields []resolution.Field,
	table *resolution.Table,
) set.Set[string] {
	out := make(set.Set[string])
	inherited := InheritedFields(extends, table)
	for _, child := range fields {
		parent, ok := inherited[child.Name]
		if !ok || isStructuralOverride(parent, child) {
			continue
		}
		out.Add(child.Name)
	}
	return out
}

// HasStructuralOverride reports whether form redeclares an inherited field with a
// different type or optionality. Neither Go embedding nor C++ inheritance can restate
// an inherited field, so such a struct must be flattened; leaving it inherited emits a
// second field beside the inherited one.
func HasStructuralOverride(
	form resolution.StructForm,
	table *resolution.Table,
) bool {
	inherited := InheritedFields(form.Extends, table)
	for _, child := range form.Fields {
		if parent, ok := inherited[child.Name]; ok &&
			isStructuralOverride(parent, child) {
			return true
		}
	}
	return false
}

// isStructuralOverride reports whether child changes more than parent's default. A
// typeless override never can: the analyzer resolves it against the parent before
// generation.
func isStructuralOverride(parent, child resolution.Field) bool {
	return child.Optional != parent.Optional || !sameTypeRef(child.Type, parent.Type)
}

// sameTypeRef reports whether a and b name the same type, comparing array sizes, type
// parameters, and type arguments.
func sameTypeRef(a, b resolution.TypeRef) bool {
	if a.Name != b.Name || len(a.TypeArgs) != len(b.TypeArgs) {
		return false
	}
	if (a.ArraySize == nil) != (b.ArraySize == nil) {
		return false
	}
	if a.ArraySize != nil && *a.ArraySize != *b.ArraySize {
		return false
	}
	if (a.TypeParam == nil) != (b.TypeParam == nil) {
		return false
	}
	if a.TypeParam != nil && a.TypeParam.Name != b.TypeParam.Name {
		return false
	}
	for i := range a.TypeArgs {
		if !sameTypeRef(a.TypeArgs[i], b.TypeArgs[i]) {
			return false
		}
	}
	return true
}

// CanUseInheritance checks if a struct can use language inheritance/embedding.
// Returns false if:
// - There are no parent types (Extends is empty)
// - There are omitted fields (can't omit fields with inheritance)
// - There are field name conflicts between parents
// - A field removes an inherited domain (must flatten to drop it)
// - A field restates an inherited field's type or optionality
func CanUseInheritance(form resolution.StructForm, table *resolution.Table) bool {
	if len(form.Extends) == 0 {
		return false
	}
	if len(form.OmittedFields) > 0 {
		return false // Can't omit fields with inheritance
	}
	if HasDomainOmissions(form) {
		return false
	}
	if HasStructuralOverride(form, table) {
		return false
	}
	return !HasFieldConflicts(form.Extends, table)
}
