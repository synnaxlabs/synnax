package set

import "github.com/samber/lo"

// Set represents a collection of unique, unordered values whose membership can be tested.
type Set[V comparable] interface {
	// Contains returns true if the provided value is in the Set.
	Contains(v V) bool
}

// ModifiableSet represents a Set that can be modified.
type ModifiableSet[V comparable] interface {
	Set[V]
	// Add adds the provided value to the Set.
	Add(v V)
	// Remove removes the provided value from the Set.
	Remove(v V)
}

type set[V comparable] map[V]struct{}

// New returns a new map-backed ModifiableSet.
func New[V comparable](items ...V) ModifiableSet[V] {
	return set[V](lo.SliceToMap(items, func(v V) (V, struct{}) { return v, struct{}{} }))
}

// Contains implements Set.
func (s set[V]) Contains(v V) bool { _, ok := s[v]; return ok }

// Add implements ModifiableSet.
func (s set[V]) Add(v V) { s[v] = struct{}{} }

// Remove implements ModifiableSet.
func (s set[V]) Remove(v V) { delete(s, v) }
