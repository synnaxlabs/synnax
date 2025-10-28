// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package maps provides generic map utilities.
package maps

import (
	"iter"
	"slices"

	"github.com/samber/lo"
)

// Ordered is a map that maintains insertion order of key-value pairs.
// Keys must be comparable, and values can be of any type.
type Ordered[K comparable, V any] struct {
	Keys   []K `json:"keys"`
	Values []V `json:"values"`
}

// Count returns the number of key-value pairs in the map.
func (m *Ordered[K, V]) Count() int {
	return len(m.Keys)
}

// Copy creates a shallow copy of the map.
// Returns a new map with cloned key and value slices.
// Returns nil if the receiver is nil.
func (m *Ordered[K, V]) Copy() *Ordered[K, V] {
	if m == nil {
		return nil
	}
	return &Ordered[K, V]{Keys: slices.Clone(m.Keys), Values: slices.Clone(m.Values)}
}

// At returns the key-value pair at the given index. Panics if the index is out of
// bounds.
func (m *Ordered[K, V]) At(i int) (K, V) {
	return m.Keys[i], m.Values[i]
}

// Iter returns an iterator over the key-value pairs in insertion order.
// The iterator supports early termination via break or return in range loops.
func (m *Ordered[K, V]) Iter() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for i, k := range m.Keys {
			if !yield(k, m.Values[i]) {
				return
			}
		}
	}
}

// Get returns the value associated with the given key.
// Returns the value and true if the key exists, otherwise returns the zero value and false.
func (m *Ordered[K, V]) Get(key K) (V, bool) {
	for i, k := range m.Keys {
		if k == key {
			return m.Values[i], true
		}
	}
	var res V
	return res, false
}

// Put adds a new key-value pair to the map.
// Returns true if the key was added, false if the key already exists.
// If the key already exists, the map is not modified.
func (m *Ordered[K, V]) Put(key K, value V) bool {
	isNewKey := !lo.Contains(m.Keys, key)
	if isNewKey {
		m.Keys = append(m.Keys, key)
		m.Values = append(m.Values, value)
	}
	return isNewKey
}
