// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy holds the era normalization and rewrite helpers shared by the
// per-integration versions/legacy packages. A legacy config is one a released Console
// or Driver wrote before configs were typed: camelCase keys and enabled-style
// polarity. Era normalization converts that form; each integration adds its own shape
// rewrites through a Rewrite.
package legacy

import (
	"maps"
	"slices"
	"strconv"

	"github.com/synnaxlabs/x/encoding/msgpack"
)

// Rewrite converts one config type's legacy blobs to the current stored shape.
// The zero value applies era normalization alone.
type Rewrite struct {
	// Pre runs before the snake_case key pass, for shapes whose data lives in key
	// position and would otherwise be corrupted by it.
	Pre func(config msgpack.EncodedJSON)
	// Post runs after era normalization and carries the integration's shape
	// migrations. Keys it sees are already snake_case.
	Post func(config msgpack.EncodedJSON)
}

// Apply returns config converted to the current stored shape. The top-level map is
// copied; nested structures may be rewritten in place.
func (r Rewrite) Apply(config msgpack.EncodedJSON) msgpack.EncodedJSON {
	out := make(msgpack.EncodedJSON, len(config))
	maps.Copy(out, config)
	if r.Pre != nil {
		r.Pre(out)
	}
	out = snakeKeys(out)
	FlipBool(out, "data_saving", "data_saving_disabled")
	EachChild(out, "channels", func(ch msgpack.EncodedJSON) {
		FlipBool(ch, "enabled", "disabled")
	})
	if r.Post != nil {
		r.Post(out)
	}
	fillKeys(map[string]any(out))
	return out
}

// fillKeys gives every keyed list element that carries an empty key one derived from
// its index. Released clients other than the Console left the key at its empty default,
// which collapses every element of a list onto the first. The index is unique within
// its own list, which is the scope a key is addressed in.
func fillKeys(v any) {
	switch t := v.(type) {
	case map[string]any:
		for _, child := range t {
			fillKeys(child)
		}
	case []any:
		for i, el := range t {
			if child, ok := el.(map[string]any); ok {
				if k, has := child["key"]; has && k == "" {
					child["key"] = strconv.Itoa(i)
				}
			}
			fillKeys(el)
		}
	}
}

// Scan converts the stored driver form of every scan config: a scan_rate field and
// per-channel enabled flags.
var Scan = Rewrite{Post: func(config msgpack.EncodedJSON) {
	RenameKey(config, "scan_rate", "rate")
	FlipBool(config, "enabled", "disabled")
}}

// camelToSnake is a Go port of the TS wire codec's camelToSnake string conversion, so
// legacy Console-written keys convert exactly as they would have on the wire. A capital
// breaks a word when it follows a lowercase or digit, or when it continues an uppercase
// run entered from one: fooXY (from foo_x_y) splits fully while a leading run like
// NS=1;ID=5 stays put.
func camelToSnake(str string) string {
	isUpper := func(c byte) bool { return c >= 'A' && c <= 'Z' }
	isLowerOrDigit := func(c byte) bool {
		return (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')
	}
	n := len(str)
	i := 1
	for ; i < n; i++ {
		if isUpper(str[i]) && isLowerOrDigit(str[i-1]) {
			break
		}
	}
	if i >= n {
		return str
	}
	res := make([]byte, 0, n+4)
	res = append(res, str[:i]...)
	enteredFromLower := true
	for ; i < n; i++ {
		c := str[i]
		if !isUpper(c) {
			enteredFromLower = false
			res = append(res, c)
			continue
		}
		prev := str[i-1]
		if isLowerOrDigit(prev) {
			enteredFromLower = true
		} else if !isUpper(prev) {
			enteredFromLower = false
		}
		if enteredFromLower {
			res = append(res, '_', c+('a'-'A'))
		} else {
			res = append(res, c)
		}
	}
	return string(res)
}

// snakeKeys returns a copy of m with every map key recursively converted through
// camelToSnake, including keys of maps nested inside lists. When both spellings of a
// key are present, the snake_case one wins. Values are never converted, even former
// record keys that listification moved into value position. The input and its nested
// structures are left unmodified.
func snakeKeys(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		if camelToSnake(k) == k {
			out[k] = snakeKeysValue(v)
		}
	}
	for k, v := range m {
		nk := camelToSnake(k)
		if nk == k {
			continue
		}
		if _, taken := out[nk]; !taken {
			out[nk] = snakeKeysValue(v)
		}
	}
	return out
}

// snakeKeysValue recurses snakeKeys through nested maps and list elements.
func snakeKeysValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return snakeKeys(t)
	case []any:
		out := make([]any, len(t))
		for i, el := range t {
			out[i] = snakeKeysValue(el)
		}
		return out
	}
	return v
}

// RenameKey moves the value under from to under to. An existing value under to wins;
// the legacy key is dropped either way.
func RenameKey(m msgpack.EncodedJSON, from, to string) {
	v, ok := m[from]
	if !ok {
		return
	}
	delete(m, from)
	if _, taken := m[to]; !taken {
		m[to] = v
	}
}

// FlipBool replaces the boolean under from with its negation under to. A non-bool
// value under from is dropped.
func FlipBool(m msgpack.EncodedJSON, from, to string) {
	v, ok := m[from]
	if !ok {
		return
	}
	delete(m, from)
	b, ok := v.(bool)
	if !ok {
		return
	}
	if _, taken := m[to]; !taken {
		m[to] = !b
	}
}

// RemapValue replaces the string stored under key through mapping. A missing key, a
// non-string value, or a string outside the mapping is left unchanged.
func RemapValue(m msgpack.EncodedJSON, key string, mapping map[string]string) {
	s, ok := m[key].(string)
	if !ok {
		return
	}
	if to, ok := mapping[s]; ok {
		m[key] = to
	}
}

// EachChild applies f to every object element of the list stored under key.
func EachChild(m msgpack.EncodedJSON, key string, f func(msgpack.EncodedJSON)) {
	list, ok := m[key].([]any)
	if !ok {
		return
	}
	for _, el := range list {
		if child, ok := el.(map[string]any); ok {
			f(child)
		}
	}
}

// RecordToList converts the record stored under key into a list of objects, one per
// record entry, with the record key under keyField and its value under valueField.
// Entries are ordered by record key so the conversion is deterministic.
func RecordToList(m msgpack.EncodedJSON, key, keyField, valueField string) {
	rec, ok := m[key].(map[string]any)
	if !ok {
		return
	}
	list := make([]any, 0, len(rec))
	for _, k := range slices.Sorted(maps.Keys(rec)) {
		list = append(list, map[string]any{keyField: k, valueField: rec[k]})
	}
	m[key] = list
}
