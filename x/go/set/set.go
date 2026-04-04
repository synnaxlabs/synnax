// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package set provides generic set data structures and operations. It includes a basic
// Set implementation using maps as well as specialized set types for various use cases.
package set

import (
	"maps"

	"github.com/samber/lo"
)

// Set is a generic collection of unique elements of type T. It is implemented as a map
// where the keys are the set elements and the values are empty structs to minimize
// memory usage.
type Set[T comparable] map[T]struct{}

// New creates a new Set containing the provided entries.
func New[T comparable](entries ...T) Set[T] { return FromSlice(entries) }

// FromSlice creates a new Set containing all elements from the provided slice.
// Duplicate elements in the input slice will only appear once in the resulting set.
func FromSlice[T comparable](entries []T) Set[T] {
	s := make(Set[T], len(entries))
	s.Add(entries...)
	return s
}

// Difference returns a new set containing elements that are in a but not in b (a - b).
func Difference[T comparable](a, b Set[T]) Set[T] {
	s := make(Set[T], len(a))
	for k, v := range a {
		if !b.Contains(k) {
			s[k] = v
		}
	}
	return s
}

// Copy creates and returns a shallow copy of the set. The returned set contains the
// same key-value pairs as the original, but modifications to one set will not affect
// the other.
func (s Set[T]) Copy() Set[T] { return maps.Clone(s) }

// Add inserts the provided values into the set. If a value already exists in the set,
// it will not be duplicated.
func (s Set[T]) Add(values ...T) Set[T] {
	var v struct{}
	for _, k := range values {
		s[k] = v
	}
	return s
}

// Remove deletes the specified keys from the set. If a key does not exist in the set,
// the operation is a no-op for that key.
func (s Set[T]) Remove(keys ...T) Set[T] {
	for _, v := range keys {
		delete(s, v)
	}
	return s
}

// Contains checks if the specified value exists in the set. Returns true if the value
// is present, false otherwise.
func (s Set[T]) Contains(val T) bool {
	_, ok := s[val]
	return ok
}

// ToSlice returns a slice containing all keys in the set. The order of the keys in the
// returned slice is not guaranteed.
func (s Set[T]) ToSlice() []T { return lo.Keys(s) }

// Equals checks if two sets contain exactly the same elements.
func (s Set[T]) Equals(other Set[T]) bool {
	if len(s) != len(other) {
		return false
	}
	for k := range s {
		if !other.Contains(k) {
			return false
		}
	}
	return true
}

// IsSubsetOf checks if s is a subset of other (all elements of s are in other).
func (s Set[T]) IsSubsetOf(other Set[T]) bool {
	for k := range s {
		if !other.Contains(k) {
			return false
		}
	}
	return true
}
