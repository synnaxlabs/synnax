// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex

import (
	"reflect"
	"strings"
)

// structToMap reduces a struct value to a map keyed by JSON wire name. Only fields
// with an explicit, non-empty name in their `json:"..."` tag are included — anything
// after the first comma (e.g. ",omitempty") is stripped to produce the key. Fields
// without a json tag, fields whose tag has no name (e.g. `json:",omitempty"`), and
// fields tagged `json:"-"` are all skipped, as are unexported fields. Pointer inputs
// are dereferenced; nested structs are stored as-is, not recursively flattened.
//
// structToMap does not enforce the omitempty option — every (non-skipped) field is
// emitted with its current value. The codec used by Marshal handles omitempty on the
// wire; this helper only builds the codec-independent intermediate.
func structToMap(v any) map[string]any {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Pointer {
		val = val.Elem()
	}
	typ := val.Type()
	result := make(map[string]any)
	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		key, ok := fieldKey(field)
		if !ok {
			continue
		}
		result[key] = val.Field(i).Interface()
	}
	return result
}

// fieldKey resolves the wire key for a struct field. Returns ok=false when the field
// has no json tag, an empty name in its tag, or is explicitly skipped via `json:"-"`.
func fieldKey(field reflect.StructField) (string, bool) {
	tag, hasTag := field.Tag.Lookup("json")
	if !hasTag {
		return "", false
	}
	name, _, _ := strings.Cut(tag, ",")
	if name == "" || name == "-" {
		return "", false
	}
	return name, true
}
