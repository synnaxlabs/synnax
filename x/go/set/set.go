// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package set provides generic set data structures and operations.
// It includes a basic Set implementation using maps as well as specialized
// set types for various use cases.
package set

import "maps"

// Mapped is a generic map-based collection that associates keys of type T
// with values of type V. It serves as the foundation for the Set type.
type Mapped[T comparable, V any] map[T]V

// Set is a generic collection of unique elements of type T.
// It is implemented as a map where the keys are the set elements
// and the values are empty structs to minimize memory usage.
type Set[T comparable] = Mapped[T, struct{}]

// FromSlice creates a new Set containing all elements from the provided slice.
// Duplicate elements in the input slice will only appear once in the resulting set.
func FromSlice[T comparable](values []T) Set[T] {
	s := make(Set[T], len(values))
	s.Add(values...)
	return s
}

func Union[T comparable, V any](sets ...Mapped[T, V]) Mapped[T, V] {
	count := 0
	for _, set := range sets {
		count += len(set)
	}
	s := make(Mapped[T, V], count)
	for _, set := range sets {
		for k, v := range set {
			s[k] = v
		}
	}
	return s
}

// Reset removes all elements from the set, leaving it empty.
// The underlying map is cleared but not deallocated.
func (s Mapped[T, V]) Reset() { clear(s) }

// Copy creates and returns a shallow copy of the set.
// The returned set contains the same key-value pairs as the original,
// but modifications to one set will not affect the other.
func (s Mapped[T, V]) Copy() Mapped[T, V] { return maps.Clone(s) }

// Add inserts the provided values into the set.
// If a value already exists in the set, it will not be duplicated.
func (s Mapped[T, V]) Add(values ...T) Mapped[T, V] {
	var v V
	for _, k := range values {
		s[k] = v
	}
	return s
}

// Remove deletes the specified keys from the set.
// If a key does not exist in the set, the operation is a no-op for that key.
func (s Mapped[T, V]) Remove(keys ...T) Mapped[T, V] {
	for _, v := range keys {
		delete(s, v)
	}
	return s
}

// Contains checks if the specified value exists in the set.
// Returns true if the value is present, false otherwise.
func (s Mapped[T, V]) Contains(v T) bool {
	_, ok := s[v]
	return ok
}

// Keys returns a slice containing all keys in the set.
// The order of the keys in the returned slice is not guaranteed.
func (s Mapped[T, V]) Keys() []T {
	values := make([]T, 0, len(s))
	for k := range s {
		values = append(values, k)
	}
	return values
}

// Values returns a slice containing all values in the set.
// The order of the values in the returned slice is not guaranteed.
func (s Mapped[T, V]) Values() []V {
	values := make([]V, 0, len(s))
	for _, v := range s {
		values = append(values, v)
	}
	return values
}
