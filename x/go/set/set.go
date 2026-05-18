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

// Set is a collection of unique elements of type T.
type Set[T comparable] map[T]struct{}

// New creates a new Set containing the provided elements.
func New[T comparable](elements ...T) Set[T] {
	s := make(Set[T], len(elements))
	for _, e := range elements {
		s[e] = struct{}{}
	}
	return s
}

// Difference returns a new Set containing elements that are in s but not in other (s -
// other).
func (s Set[T]) Difference(other Set[T]) Set[T] {
	result := make(Set[T], len(s))
	for e := range s {
		if !other.Contains(e) {
			result.Add(e)
		}
	}
	return result
}

// Copy creates and returns a shallow copy of the Set. The returned Set contains the
// same elements as the original, but modifications to one Set will not affect the
// other.
func (s Set[T]) Copy() Set[T] { return maps.Clone(s) }

// Add inserts the provided elements into the Set.
func (s Set[T]) Add(elements ...T) Set[T] {
	for _, e := range elements {
		s[e] = struct{}{}
	}
	return s
}

// Remove deletes the specified elements from the Set.
func (s Set[T]) Remove(elements ...T) Set[T] {
	for _, e := range elements {
		delete(s, e)
	}
	return s
}

// Contains checks if the specified element exists in the Set. Returns true if the
// element is present, false otherwise.
func (s Set[T]) Contains(element T) bool {
	_, ok := s[element]
	return ok
}

// Slice returns a slice containing all elements in the Set. The order of the elements
// in the returned slice is not guaranteed.
func (s Set[T]) Slice() []T { return lo.Keys(s) }

// Equal compares if two Sets contain exactly the same elements.
func (s Set[T]) Equal(other Set[T]) bool {
	if len(s) != len(other) {
		return false
	}
	for e := range s {
		if !other.Contains(e) {
			return false
		}
	}
	return true
}

// IsSubsetOf checks if s is a subset of other (all elements of s are in other).
func (s Set[T]) IsSubsetOf(other Set[T]) bool {
	for e := range s {
		if !other.Contains(e) {
			return false
		}
	}
	return true
}
