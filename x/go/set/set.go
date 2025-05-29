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

// Add inserts the provided values into the set.
// If a value already exists in the set, it will not be duplicated.
func (s Mapped[T, V]) Add(values ...T) {
	var v V
	for _, k := range values {
		s[k] = v
	}
}

// Remove deletes the specified keys from the set.
// If a key does not exist in the set, the operation is a no-op for that key.
func (s Mapped[T, V]) Remove(keys ...T) {
	for _, v := range keys {
		delete(s, v)
	}
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
