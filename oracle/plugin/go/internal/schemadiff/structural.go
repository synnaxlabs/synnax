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
// declared defaults must agree. Wire-compatible enum additions need no migration, but
// they are still a shape change the history must record.
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
	if !marshalEqual(old, new) || !defaultsEqual(old, new) {
		return false
	}
	return SchemasEqual(old, new, oldTable, newTable)
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

// bareTypeName strips a resolved reference's namespace so declarations from different
// tables compare on the type they name.
func bareTypeName(ref resolution.TypeRef) string {
	if _, rest, found := strings.Cut(ref.Name, "."); found {
		return rest
	}
	return ref.Name
}
