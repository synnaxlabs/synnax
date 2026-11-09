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
	"reflect"
	"strings"

	"github.com/synnaxlabs/oracle/plugin/domain"
	"github.com/synnaxlabs/oracle/resolution"
)

// StructurallyEqual reports whether two declarations share a persisted shape, resolving
// each side's references through its own table. It is schemadiff's persisted-shape
// equality with three tightenings: enums must match member-for-member, the declared
// field list — omitted fields and their marshal values included — must agree, and the
// declared defaults must agree. A union's inline variant payloads get the same checks.
// Wire-compatible enum additions need no migration, but they are still a shape change
// the history must record.
func StructurallyEqual(
	old, new resolution.Type, oldTable, newTable *resolution.Table,
) bool {
	oldEnum, oldOK := old.Form.(resolution.EnumForm)
	newEnum, newOK := new.Form.(resolution.EnumForm)
	if oldOK != newOK {
		return false
	}
	if oldOK {
		if len(oldEnum.Values) != len(newEnum.Values) {
			return false
		}
		for i, v := range oldEnum.Values {
			if newEnum.Values[i].Name != v.Name ||
				newEnum.Values[i].Value != v.Value {
				return false
			}
		}
	}
	if !marshalEqual(old, new, oldTable, newTable) || !defaultsEqual(old, new) ||
		!inlinePayloadsEqual(old, new, oldTable, newTable) {
		return false
	}
	return SchemasEqual(old, new, oldTable, newTable)
}

// inlinePayloadsEqual compares the inline variant payloads of two unions with
// StructurallyEqual. An inline payload is part of its union's declaration, so its field
// list and defaults are the union's.
func inlinePayloadsEqual(
	old, new resolution.Type, oldTable, newTable *resolution.Table,
) bool {
	oldForm, oldOK := old.Form.(resolution.UnionForm)
	newForm, newOK := new.Form.(resolution.UnionForm)
	if !oldOK || !newOK {
		return true
	}
	for _, ov := range oldForm.Variants {
		nv, ok := newForm.Variant(ov.Name)
		if !ok || nv.Inline != ov.Inline {
			return false
		}
		if !ov.Inline {
			continue
		}
		op, oldFound := ov.Type.Resolve(oldTable)
		np, newFound := nv.Type.Resolve(newTable)
		if !oldFound || !newFound || !StructurallyEqual(op, np, oldTable, newTable) {
			return false
		}
	}
	return len(oldForm.Variants) == len(newForm.Variants)
}

// marshalEqual compares the codec surface schemadiff cannot see: the declared field
// list including omitted fields, and each field's @go marshal and @go wire values. Two
// versions with the same persisted shape still declare different Go structs when one
// omits a field the other stores. It also requires each field type to name the same
// declaration, so a type moving to another resource is a change even when its shape
// holds.
func marshalEqual(old, new resolution.Type, oldTable, newTable *resolution.Table) bool {
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
			!sameDeclaration(f.Type, n.Type, oldTable, newTable) ||
			domain.GetStringFromField(n, "go", "marshal") !=
				domain.GetStringFromField(f, "go", "marshal") ||
			domain.GetStringFromField(n, "go", "wire") !=
				domain.GetStringFromField(f, "go", "wire") {
			return false
		}
	}
	return true
}

// defaultsEqual compares the two declarations' static field defaults. A default is part
// of the declared shape: it fills on write, so a version that adds one persists bytes
// its predecessor never wrote. Callers run it after marshalEqual, which rejects a
// differing form or field count.
func defaultsEqual(old, new resolution.Type) bool {
	oldForm, oldOK := old.Form.(resolution.StructForm)
	newForm, newOK := new.Form.(resolution.StructForm)
	if !oldOK || !newOK || len(oldForm.Fields) != len(newForm.Fields) {
		return true
	}
	for i, f := range oldForm.Fields {
		if !reflect.DeepEqual(f.Default, newForm.Fields[i].Default) {
			return false
		}
	}
	return true
}

// sameDeclaration reports whether two references name the same declaration of the
// same resource at every position. Each side resolves through its own table, so
// versions of one resource compare equal.
func sameDeclaration(
	oldRef, newRef resolution.TypeRef, oldTable, newTable *resolution.Table,
) bool {
	if bareTypeName(oldRef) != bareTypeName(newRef) ||
		len(oldRef.TypeArgs) != len(newRef.TypeArgs) {
		return false
	}
	oldT, oldOK := oldRef.Resolve(oldTable)
	newT, newOK := newRef.Resolve(newTable)
	if oldOK && newOK && resourceOf(oldT.Namespace) != resourceOf(newT.Namespace) {
		return false
	}
	for i := range oldRef.TypeArgs {
		if !sameDeclaration(
			oldRef.TypeArgs[i],
			newRef.TypeArgs[i],
			oldTable,
			newTable,
		) {
			return false
		}
	}
	return true
}

// resourceOf strips the version from a namespace. A version file's own namespace is
// its bare version, such as v2, so it maps to the empty resource.
func resourceOf(namespace string) string {
	resource, _, _ := strings.Cut(namespace, "@v")
	if v, ok := strings.CutPrefix(resource, "v"); ok && isDigits(v) {
		return ""
	}
	return resource
}

func isDigits(s string) bool {
	return s != "" && strings.Trim(s, "0123456789") == ""
}

// bareTypeName strips a resolved reference's namespace so declarations from different
// tables compare on the type they name.
func bareTypeName(ref resolution.TypeRef) string {
	if _, rest, found := strings.Cut(ref.Name, "."); found {
		return rest
	}
	return ref.Name
}
