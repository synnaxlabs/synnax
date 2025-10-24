// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package maps

import "iter"

type Ordered[K comparable, V any] struct {
	Keys   []K
	Values []V
}

func (m *Ordered[K, V]) Copy() *Ordered[K, V] {
	if m == nil {
		return nil
	}
	cpy := &Ordered[K, V]{Keys: make([]K, len(m.Keys)), Values: make([]V, len(m.Values))}
	copy(cpy.Keys, m.Keys)
	copy(cpy.Values, m.Values)
	return cpy
}

func (m *Ordered[K, V]) Count() int {
	return len(m.Keys)
}

func (m *Ordered[K, V]) At(i int) (K, V) {
	return m.Keys[i], m.Values[i]
}

func (m *Ordered[K, V]) Iter() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for i, k := range m.Keys {
			if !yield(k, m.Values[i]) {
				return
			}
		}
	}
}

func (m *Ordered[K, V]) Get(key K) (V, bool) {
	for i, k := range m.Keys {
		if k == key {
			return m.Values[i], true
		}
	}
	var res V
	return res, false
}

func (m *Ordered[K, V]) Put(key K, value V) bool {
	for _, k := range m.Keys {
		if k == key {
			return false
		}
	}
	m.Keys = append(m.Keys, key)
	m.Values = append(m.Values, value)
	return true
}
