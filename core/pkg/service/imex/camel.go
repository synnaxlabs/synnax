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
	"context"
	"encoding/json"
	"maps"
	"reflect"
	"strings"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
)

// DecodeCamel materializes the envelope body as T, tolerating camelCase keys on
// fields whose json tags are snake_case. Console-written files carry camelCase
// keys; the typed structs' tags are snake_case. A key is renamed only when it
// matches a field declared on the target struct — map keys and opaque payloads
// (types with their own UnmarshalJSON) pass through untouched, so user-chosen
// identifiers are never rewritten.
func DecodeCamel[T any](ctx context.Context, e Envelope) (T, error) {
	var t T
	if e.codec == nil {
		return t, errors.New(
			"decode envelope body: envelope has no codec bound; " +
				"Encode-side envelopes must be marshaled and unmarshaled through " +
				"one of the UnmarshalX methods before Decode",
		)
	}
	var body map[string]any
	if err := e.codec.Decode(ctx, e.raw, &body); err != nil {
		return t, errors.Wrap(err, "decode envelope body")
	}
	raw, err := json.Marshal(normalizeKeys(reflect.TypeOf(t), body))
	if err != nil {
		return t, errors.Wrap(err, "decode envelope body")
	}
	if err := json.Unmarshal(raw, &t); err != nil {
		var zero T
		return zero, errors.Wrap(err, "decode envelope body")
	}
	return t, nil
}

var jsonUnmarshaler = reflect.TypeFor[json.Unmarshaler]()

// normalizeKeys rewrites the object keys of v to the json tag names declared by
// target, recursing through struct fields, slice elements, and map values. Types
// with a custom UnmarshalJSON own their wire form and are left untouched, as are
// keys with no matching field.
func normalizeKeys(target reflect.Type, v any) any {
	for target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	if target.Implements(jsonUnmarshaler) ||
		reflect.PointerTo(target).Implements(jsonUnmarshaler) {
		return v
	}
	switch target.Kind() {
	case reflect.Struct:
		m, ok := v.(map[string]any)
		if !ok {
			return v
		}
		fields := structFields(target)
		out := make(map[string]any, len(m))
		for k, val := range m {
			f, ok := fields[k]
			if !ok {
				f, ok = fields[lo.SnakeCase(k)]
			}
			if !ok {
				out[k] = val
				continue
			}
			out[f.tag] = normalizeKeys(f.typ, val)
		}
		return out
	case reflect.Slice, reflect.Array:
		if target.Elem().Kind() == reflect.Uint8 {
			return v
		}
		s, ok := v.([]any)
		if !ok {
			return v
		}
		out := make([]any, len(s))
		for i, el := range s {
			out[i] = normalizeKeys(target.Elem(), el)
		}
		return out
	case reflect.Map:
		m, ok := v.(map[string]any)
		if !ok {
			return v
		}
		out := make(map[string]any, len(m))
		for k, val := range m {
			out[k] = normalizeKeys(target.Elem(), val)
		}
		return out
	default:
		return v
	}
}

type fieldInfo struct {
	tag string
	typ reflect.Type
}

// structFields maps each of target's visible fields to its json tag name, keyed
// by that tag. Embedded structs are flattened the same way encoding/json
// promotes their fields.
func structFields(target reflect.Type) map[string]fieldInfo {
	out := make(map[string]fieldInfo, target.NumField())
	for f := range target.Fields() {
		if !f.IsExported() {
			continue
		}
		tag, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if tag == "-" {
			continue
		}
		if f.Anonymous && tag == "" {
			maps.Copy(out, structFields(f.Type))
			continue
		}
		// Untagged fields keep their wire keys: encoding/json already matches
		// them by field name case-insensitively.
		if tag == "" {
			continue
		}
		out[tag] = fieldInfo{tag: tag, typ: f.Type}
	}
	return out
}
