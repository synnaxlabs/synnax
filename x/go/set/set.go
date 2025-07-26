// Copyright 2025 Synnax Labs, Inc.
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

// Set is a collection of unique elements of type T.
type Set[T comparable] map[T]struct{}

// New creates a new Set from the provided elements.
func New[T comparable](elements ...T) Set[T] {
	s := make(Set[T], len(elements))
	s.Add(elements...)
	return s
}

// Add inserts the provided elements into the set.
func (s Set[T]) Add(elements ...T) {
	for _, e := range elements {
		s[e] = struct{}{}
	}
}

// Remove deletes the specified elements from the set.
func (s Set[T]) Remove(elements ...T) {
	for _, e := range elements {
		delete(s, e)
	}
}

// Contains checks if the specified element exists in the set.
func (s Set[T]) Contains(element T) bool {
	_, ok := s[element]
	return ok
}

// Elements returns a slice containing all elements in the set.
func (s Set[T]) Elements() []T {
	elements := make([]T, 0, len(s))
	for e := range s {
		elements = append(elements, e)
	}
	return elements
}
