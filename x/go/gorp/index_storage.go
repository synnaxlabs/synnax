// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"reflect"
	"slices"
	"sort"
)

// IndexKey is the subset of Key that supports secondary indexes. It excludes
// unhashable Key types (notably ~[]byte) because lookup backings must key on
// strictly comparable values. Tables whose primary key is not strictly
// comparable cannot register indexes.
type IndexKey interface {
	Key
	comparable
}

// lookupStorage is the contract satisfied by every Lookup backing structure.
// Implementations decide how to store the forward mapping from value to a set
// of primary keys. The interface is internal; Lookup selects an implementation
// from the value type V when it is populated.
type lookupStorage[K IndexKey, V comparable] interface {
	put(key K, value V)
	remove(key K, value V)
	get(value V) []K
}

// newLookupStorage constructs a lookup backing keyed on V. Specialized
// implementations are chosen by reflecting on the zero value of V:
//   - bool           -> boolLookupStorage (two buckets, no hashing)
//   - anything else  -> mapLookupStorage (map[V][]K)
//
// Additional specializations (dense arrays for small integers, etc.) can be
// added without changing the interface.
func newLookupStorage[K IndexKey, V comparable]() lookupStorage[K, V] {
	var zero V
	switch reflect.TypeOf(zero).Kind() {
	case reflect.Bool:
		// boolLookupStorage always satisfies lookupStorage[K, bool]; the type
		// assertion is safe because we only enter this branch when V is bool.
		var s any = newBoolLookupStorage[K]()
		return s.(lookupStorage[K, V])
	default:
		return newMapLookupStorage[K, V]()
	}
}

// mapLookupStorage is the default backing: a map from value to a slice of keys.
// Suitable for any comparable V including strings, structs, and wide integer types.
type mapLookupStorage[K IndexKey, V comparable] struct {
	forward map[V][]K
}

func newMapLookupStorage[K IndexKey, V comparable]() *mapLookupStorage[K, V] {
	return &mapLookupStorage[K, V]{forward: make(map[V][]K)}
}

func (s *mapLookupStorage[K, V]) put(key K, value V) {
	s.forward[value] = append(s.forward[value], key)
}

func (s *mapLookupStorage[K, V]) remove(key K, value V) {
	keys := s.forward[value]
	for i, k := range keys {
		if k == key {
			keys = slices.Delete(keys, i, i+1)
			break
		}
	}
	if len(keys) == 0 {
		delete(s.forward, value)
		return
	}
	s.forward[value] = keys
}

func (s *mapLookupStorage[K, V]) get(value V) []K {
	return s.forward[value]
}

// boolLookupStorage is a two-bucket specialization for bool-valued indexes.
// There are only two possible values, so we avoid hashing and map overhead
// entirely by keeping one slice per bucket.
type boolLookupStorage[K IndexKey] struct {
	trueKeys  []K
	falseKeys []K
}

func newBoolLookupStorage[K IndexKey]() *boolLookupStorage[K] {
	return &boolLookupStorage[K]{}
}

func (s *boolLookupStorage[K]) put(key K, value bool) {
	if value {
		s.trueKeys = append(s.trueKeys, key)
		return
	}
	s.falseKeys = append(s.falseKeys, key)
}

func (s *boolLookupStorage[K]) remove(key K, value bool) {
	bucket := &s.falseKeys
	if value {
		bucket = &s.trueKeys
	}
	for i, k := range *bucket {
		if k == key {
			*bucket = slices.Delete(*bucket, i, i+1)
			return
		}
	}
}

func (s *boolLookupStorage[K]) get(value bool) []K {
	if value {
		return s.trueKeys
	}
	return s.falseKeys
}

// sortedEntry is a single (value, key) pair inside a sorted index slice.
type sortedEntry[K IndexKey, V comparable] struct {
	Value V
	Key   K
}

// sortedStorage backs a Sorted index. It keeps entries in ascending order of V
// using a caller-supplied less function. Insertion is O(log n) binary search
// plus O(n) slice shift. At the target scale (<100k entries) this is
// acceptable; if profiling shows the shift cost matters, swap the backing
// for a B-tree without changing the outer API. Within equal values, entries
// are kept in insertion order; removal scans that sub-range for an exact key
// match.
type sortedStorage[K IndexKey, V comparable] struct {
	entries []sortedEntry[K, V]
	less    func(a, b V) bool
}

func newSortedStorage[K IndexKey, V comparable](less func(a, b V) bool) *sortedStorage[K, V] {
	return &sortedStorage[K, V]{less: less}
}

// lowerBound returns the first index i such that !less(entries[i].Value, value).
func (s *sortedStorage[K, V]) lowerBound(value V) int {
	return sort.Search(len(s.entries), func(i int) bool {
		return !s.less(s.entries[i].Value, value)
	})
}

// upperBound returns the first index i such that less(value, entries[i].Value).
func (s *sortedStorage[K, V]) upperBound(value V) int {
	return sort.Search(len(s.entries), func(i int) bool {
		return s.less(value, s.entries[i].Value)
	})
}

func (s *sortedStorage[K, V]) put(key K, value V) {
	i := s.upperBound(value)
	s.entries = slices.Insert(s.entries, i, sortedEntry[K, V]{Value: value, Key: key})
}

func (s *sortedStorage[K, V]) remove(key K, value V) {
	lo := s.lowerBound(value)
	hi := s.upperBound(value)
	for i := lo; i < hi; i++ {
		if s.entries[i].Key == key {
			s.entries = slices.Delete(s.entries, i, i+1)
			return
		}
	}
}

func (s *sortedStorage[K, V]) get(value V) []K {
	lo := s.lowerBound(value)
	hi := s.upperBound(value)
	if lo == hi {
		return nil
	}
	out := make([]K, hi-lo)
	for i := lo; i < hi; i++ {
		out[i-lo] = s.entries[i].Key
	}
	return out
}

// defaultLess returns a native less function for V if one can be derived from
// the type's kind, or nil if the caller must supply one. Supports signed and
// unsigned integers, floats, and strings.
func defaultLess[V comparable]() func(a, b V) bool {
	var zero V
	kind := reflect.TypeOf(zero).Kind()
	switch kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(a, b V) bool {
			return reflect.ValueOf(a).Int() < reflect.ValueOf(b).Int()
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return func(a, b V) bool {
			return reflect.ValueOf(a).Uint() < reflect.ValueOf(b).Uint()
		}
	case reflect.Float32, reflect.Float64:
		return func(a, b V) bool {
			return reflect.ValueOf(a).Float() < reflect.ValueOf(b).Float()
		}
	case reflect.String:
		return func(a, b V) bool {
			return reflect.ValueOf(a).String() < reflect.ValueOf(b).String()
		}
	}
	return nil
}
