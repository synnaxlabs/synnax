// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schemadiff

import (
	"strings"

	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/resolution"
	"github.com/synnaxlabs/x/set"
)

// StructurallyEqual reports whether two declarations share a persisted shape, resolving
// each side's references through its own table. It is schemadiff's persisted-shape
// equality with two tightenings: enums must match member-for-member, and the declared
// field list — omitted fields and their marshal values included — must agree. Both
// tightenings apply through every persisted reference: a struct holding an enum that
// gained a member is a new declaration, because its Go type is new. Wire-compatible
// enum additions need no migration, but they are still a shape change the history must
// record.
func StructurallyEqual(
	old, new resolution.Type, oldTable, newTable *resolution.Table,
) bool {
	return structurallyEqual(old, new, oldTable, newTable, make(set.Set[string]))
}

func structurallyEqual(
	old, new resolution.Type,
	oldTable, newTable *resolution.Table,
	visiting set.Set[string],
) bool {
	if visiting.Contains(old.QualifiedName) {
		return true
	}
	visiting.Add(old.QualifiedName)
	defer visiting.Remove(old.QualifiedName)
	if !enumsEqual(old, new) || !marshalEqual(old, new) {
		return false
	}
	if !SchemasEqual(old, new, oldTable, newTable) {
		return false
	}
	oldRefs, newRefs := persistedRefs(old), persistedRefs(new)
	if len(oldRefs) != len(newRefs) {
		return false
	}
	for i := range oldRefs {
		oldResolved, oldOK := oldRefs[i].Resolve(oldTable)
		newResolved, newOK := newRefs[i].Resolve(newTable)
		if oldOK != newOK {
			return false
		}
		if oldOK && !structurallyEqual(
			oldResolved, newResolved, oldTable, newTable, visiting,
		) {
			return false
		}
	}
	return true
}

// enumsEqual compares two enum declarations member-for-member. Non-enums are equal.
func enumsEqual(old, new resolution.Type) bool {
	oldEnum, oldOK := old.Form.(resolution.EnumForm)
	newEnum, newOK := new.Form.(resolution.EnumForm)
	if oldOK != newOK {
		return false
	}
	if !oldOK {
		return true
	}
	if len(oldEnum.Values) != len(newEnum.Values) {
		return false
	}
	for i, v := range oldEnum.Values {
		if newEnum.Values[i].Name != v.Name || newEnum.Values[i].Value != v.Value {
			return false
		}
	}
	return true
}

// persistedRefs returns the references that reach a declaration's stored shape, in
// declaration order, with type arguments flattened in. SchemasEqual has already matched
// the two sides' forms and counts, so the lists of two equal shapes zip.
func persistedRefs(t resolution.Type) []resolution.TypeRef {
	var refs []resolution.TypeRef
	var add func(ref resolution.TypeRef)
	add = func(ref resolution.TypeRef) {
		refs = append(refs, ref)
		for _, arg := range ref.TypeArgs {
			add(arg)
		}
	}
	switch form := t.Form.(type) {
	case resolution.StructForm:
		for _, f := range PersistedFields(form.Fields) {
			add(f.Type)
		}
		for _, ext := range form.Extends {
			add(ext)
		}
	case resolution.UnionForm:
		for _, v := range form.Variants {
			add(v.Type)
		}
		for _, ext := range form.Extends {
			add(ext)
		}
	case resolution.AliasForm:
		add(form.Target)
	case resolution.DistinctForm:
		add(form.Base)
	}
	return refs
}

// marshalEqual compares the codec surface schemadiff cannot see: the declared field
// list including omitted fields, and each field's @go marshal value. Two versions with
// the same persisted shape still declare different Go structs when one omits a field
// the other stores. Field types are schemadiff's concern — the two sides resolve
// against different tables, so their qualified names never compare directly.
func marshalEqual(old, new resolution.Type) bool {
	// The type-level persistence set is part of the declared shape: a version that
	// flips a type between hand and generated, or changes its codec marker, is a
	// real delta even when the field list is unchanged (auth v0 legacy → v1 Orc).
	for _, expr := range []string{"hand", "marshal"} {
		if domain.HasExprFromType(old, "go", expr) !=
			domain.HasExprFromType(new, "go", expr) ||
			domain.GetStringFromType(old, "go", expr) !=
				domain.GetStringFromType(new, "go", expr) {
			return false
		}
	}
	oldForm, oldOK := old.Form.(resolution.StructForm)
	newForm, newOK := new.Form.(resolution.StructForm)
	if oldOK != newOK || !oldOK {
		return true
	}
	if len(oldForm.Fields) != len(newForm.Fields) {
		return false
	}
	for i, f := range oldForm.Fields {
		n := newForm.Fields[i]
		if n.Name != f.Name ||
			n.Optional != f.Optional ||
			bareTypeName(n.Type) != bareTypeName(f.Type) ||
			domain.GetStringFromField(n, "go", "marshal") !=
				domain.GetStringFromField(f, "go", "marshal") {
			return false
		}
	}
	return true
}

// bareTypeName strips a resolved reference's namespace so declarations from different
// tables compare on the type they name.
func bareTypeName(ref resolution.TypeRef) string {
	if _, rest, found := strings.Cut(ref.Name, "."); found {
		return rest
	}
	return ref.Name
}
