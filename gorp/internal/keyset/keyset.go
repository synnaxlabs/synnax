// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package keyset implements the candidate primary key sets that Gorp filters carry.
package keyset

import (
	"slices"
	"sync"

	"github.com/synnaxlabs/x/set"
)

// Set is a candidate set of primary keys. The zero value is unbounded and contains
// every key. A bounded Set contains only its keys, so an empty bounded Set contains
// none. Copies of a Set share its hash, which is safe for concurrent use.
type Set[K comparable] struct {
	// keys is nil if and only if the Set is unbounded.
	keys []K
	// hash answers Contains in constant time when non-nil. Nil means Contains scans
	// keys.
	hash *hash[K]
}

// hash is a set over keys, built on the first probe.
type hash[K comparable] struct {
	once sync.Once
	set  set.Set[K]
}

func newSet[K comparable](keys []K, hashed bool) Set[K] {
	if keys == nil {
		keys = []K{}
	}
	s := Set[K]{keys: keys}
	if hashed {
		s.hash = &hash[K]{}
	}
	return s
}

// Of returns a bounded Set of keys whose Contains scans keys. Use it for sets that
// Contains probes rarely.
func Of[K comparable](keys []K) Set[K] { return newSet(keys, false) }

// Hashed returns a bounded Set of keys whose Contains builds a hash set on first use.
func Hashed[K comparable](keys []K) Set[K] { return newSet(keys, true) }

// Bounded reports whether s constrains keys.
func (s Set[K]) Bounded() bool { return s.keys != nil }

// Keys returns the keys of a bounded Set, or nil for an unbounded one. The caller must
// not modify the returned slice.
func (s Set[K]) Keys() []K { return s.keys }

// Contains reports whether s contains k. An unbounded Set contains every key.
func (s Set[K]) Contains(k K) bool {
	if s.keys == nil {
		return true
	}
	if s.hash == nil {
		return slices.Contains(s.keys, k)
	}
	s.hash.once.Do(func() { s.hash.set = set.New(s.keys...) })
	return s.hash.set.Contains(k)
}

// Intersect returns the keys that every bounded set contains. Unbounded sets do not
// constrain the result, so the result is unbounded only when no set is bounded. The
// result is hashed when any bounded set is.
//
// Intersect walks the largest set and probes the others, so it never builds a hash of
// the largest set.
func Intersect[K comparable](sets ...Set[K]) Set[K] {
	var (
		bounded = make([]Set[K], 0, len(sets))
		hashed  bool
	)
	for _, s := range sets {
		if s.Bounded() {
			bounded = append(bounded, s)
			hashed = hashed || s.hash != nil
		}
	}
	switch len(bounded) {
	case 0:
		return Set[K]{}
	case 1:
		return bounded[0]
	}
	slices.SortFunc(bounded, compareLen[K])
	var (
		largest = bounded[len(bounded)-1]
		rest    = bounded[:len(bounded)-1]
		out     = make([]K, 0, len(bounded[0].keys))
	)
	for _, k := range largest.keys {
		if containedByAll(rest, k) {
			out = append(out, k)
		}
	}
	return newSet(out, hashed)
}

// Union returns the keys that any set contains. The result is unbounded when no sets
// are given or any set is unbounded. The result is hashed when any set is.
//
// Union walks the sets from smallest to largest and probes only the smaller sets for
// duplicates, so it never builds a hash of the largest set.
func Union[K comparable](sets ...Set[K]) Set[K] {
	if len(sets) == 0 {
		return Set[K]{}
	}
	var (
		total  int
		hashed bool
	)
	for _, s := range sets {
		if !s.Bounded() {
			return Set[K]{}
		}
		total += len(s.keys)
		hashed = hashed || s.hash != nil
	}
	sorted := slices.Clone(sets)
	slices.SortFunc(sorted, compareLen[K])
	out := make([]K, 0, total)
	for i, s := range sorted {
		for _, k := range s.keys {
			if !containedByAny(sorted[:i], k) {
				out = append(out, k)
			}
		}
	}
	return newSet(out, hashed)
}

func compareLen[K comparable](a, b Set[K]) int { return len(a.keys) - len(b.keys) }

func containedByAll[K comparable](sets []Set[K], k K) bool {
	for _, s := range sets {
		if !s.Contains(k) {
			return false
		}
	}
	return true
}

func containedByAny[K comparable](sets []Set[K], k K) bool {
	for _, s := range sets {
		if s.Contains(k) {
			return true
		}
	}
	return false
}
