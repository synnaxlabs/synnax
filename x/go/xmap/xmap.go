// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package xmap provides additional functionality for the built-in map type.
package xmap

// Map is an alias for the built-in map type that provides additional functionality.
type Map[K comparable, V any] map[K]V

// Get returns the value associated with the given key. If the key is not found, the
// zero value of V is returned and ok is false.
func (m Map[K, V]) Get(key K) (V, bool) {
	v, ok := m[key]
	return v, ok
}

// Set sets the value associated with the given key.
func (m Map[K, V]) Set(key K, value V) { m[key] = value }

// Delete deletes the value associated with the given key.
func (m Map[K, V]) Delete(key K) { delete(m, key) }

// Keys returns a slice of all keys in the map.
func (m Map[K, V]) Keys() []K {
	keys := make([]K, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Values returns a slice of all values in the map.
func (m Map[K, V]) Values() []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}
