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
	"cmp"
	"slices"
	"sort"
	"sync"
)

// IndexKey is the subset of Key that supports secondary indexes whose
// primary key type is strictly comparable. Tables whose primary key is
// ~[]byte use a byte-storage backing instead.
type IndexKey interface {
	Key
	comparable
}

// lookupStorage is the contract every Lookup backing implements. The
// interface keeps Lookup data-type-agnostic; each implementation owns its
// own forward map (V → []K), reverse map (K → V), and per-tx staging
// overlay, choosing the keying strategy that fits its data type.
//
// Methods divide into two groups:
//
//   - Committed-state ops (put / delete / get): the caller holds the
//     owning Lookup's mu (write for put / delete, read for get). The
//     storage performs no extra locking on these calls.
//   - Per-tx staging ops (stageSet / stageDelete / resolve): internally
//     synchronized via the embedded deltaOverlay. stage* with a
//     transaction whose txIdentity() returns nil applies the mutation
//     immediately to committed state, acquiring commitMu internally.
type lookupStorage[K Key, V comparable] interface {
	put(key K, value V)
	delete(key K)
	get(value V) []K
	stageSet(tx Tx, key K, value V)
	stageDelete(tx Tx, key K)
	resolve(tx Tx, committed []K, values []V) []K
}

// mapLookupStorage backs a Lookup whose primary key type is strictly
// comparable. The forward map is keyed by V, the reverse map is keyed
// by K directly, and the staging overlay's storable key is also K.
type mapLookupStorage[K IndexKey, V comparable] struct {
	commitMu *sync.RWMutex
	forward  map[V][]K
	reverse  map[K]V
	overlay  deltaOverlay[K, V]
}

func newMapLookupStorage[K IndexKey, V comparable](
	commitMu *sync.RWMutex,
) *mapLookupStorage[K, V] {
	s := &mapLookupStorage[K, V]{
		commitMu: commitMu,
		forward:  make(map[V][]K),
		reverse:  make(map[K]V),
	}
	s.overlay.flush = s.flush
	return s
}

//nolint:unused
func (s *mapLookupStorage[K, V]) put(key K, value V) {
	if oldValue, existed := s.reverse[key]; existed {
		if oldValue == value {
			return
		}
		s.removeFromForward(key, oldValue)
	}
	s.forward[value] = append(s.forward[value], key)
	s.reverse[key] = value
}

//nolint:unused
func (s *mapLookupStorage[K, V]) delete(key K) {
	oldValue, existed := s.reverse[key]
	if !existed {
		return
	}
	s.removeFromForward(key, oldValue)
	delete(s.reverse, key)
}

//nolint:unused
func (s *mapLookupStorage[K, V]) removeFromForward(key K, value V) {
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

//nolint:unused
func (s *mapLookupStorage[K, V]) get(value V) []K {
	src := s.forward[value]
	if len(src) == 0 {
		return nil
	}
	out := make([]K, len(src))
	copy(out, src)
	return out
}

//nolint:unused
func (s *mapLookupStorage[K, V]) stageSet(tx Tx, key K, value V) {
	if tx.txIdentity() == nil {
		s.commitMu.Lock()
		defer s.commitMu.Unlock()
		s.put(key, value)
		return
	}
	s.overlay.stage(tx, key, value)
}

//nolint:unused
func (s *mapLookupStorage[K, V]) stageDelete(tx Tx, key K) {
	if tx.txIdentity() == nil {
		s.commitMu.Lock()
		defer s.commitMu.Unlock()
		s.delete(key)
		return
	}
	s.overlay.unstage(tx, key)
}

//nolint:unused
func (s *mapLookupStorage[K, V]) resolve(tx Tx, committed []K, values []V) []K {
	return s.overlay.resolve(tx, committed, values)
}

// flush applies a committed tx's staged delta into committed state. It
// is wired into overlay.flush at construction and runs without any
// gorp-side lock held; it acquires commitMu before mutating.
func (s *mapLookupStorage[K, V]) flush(d *delta[K, V]) {
	s.commitMu.Lock()
	defer s.commitMu.Unlock()
	for k, entry := range d.state {
		if entry.deleted {
			s.delete(k)
			continue
		}
		s.put(k, entry.value)
	}
}

// boolLookupStorage backs a Lookup whose indexed value type V is bool.
// The two-bucket forward representation avoids hashing the bool key
// entirely; reverse remains a K-keyed map so we can locate the old
// bucket when an entry's value flips.
type boolLookupStorage[K IndexKey] struct {
	commitMu   *sync.RWMutex
	trueKeys   []K
	falseKeys  []K
	reverse    map[K]bool
	overlay    deltaOverlay[K, bool]
}

func newBoolLookupStorage[K IndexKey](
	commitMu *sync.RWMutex,
) *boolLookupStorage[K] {
	s := &boolLookupStorage[K]{
		commitMu: commitMu,
		reverse:  make(map[K]bool),
	}
	s.overlay.flush = s.flush
	return s
}

//nolint:unused
func (s *boolLookupStorage[K]) put(key K, value bool) {
	if oldValue, existed := s.reverse[key]; existed {
		if oldValue == value {
			return
		}
		s.removeFromBucket(key, oldValue)
	}
	if value {
		s.trueKeys = append(s.trueKeys, key)
	} else {
		s.falseKeys = append(s.falseKeys, key)
	}
	s.reverse[key] = value
}

//nolint:unused
func (s *boolLookupStorage[K]) delete(key K) {
	oldValue, existed := s.reverse[key]
	if !existed {
		return
	}
	s.removeFromBucket(key, oldValue)
	delete(s.reverse, key)
}

//nolint:unused
func (s *boolLookupStorage[K]) removeFromBucket(key K, value bool) {
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

//nolint:unused
func (s *boolLookupStorage[K]) get(value bool) []K {
	src := s.falseKeys
	if value {
		src = s.trueKeys
	}
	if len(src) == 0 {
		return nil
	}
	out := make([]K, len(src))
	copy(out, src)
	return out
}

//nolint:unused
func (s *boolLookupStorage[K]) stageSet(tx Tx, key K, value bool) {
	if tx.txIdentity() == nil {
		s.commitMu.Lock()
		defer s.commitMu.Unlock()
		s.put(key, value)
		return
	}
	s.overlay.stage(tx, key, value)
}

//nolint:unused
func (s *boolLookupStorage[K]) stageDelete(tx Tx, key K) {
	if tx.txIdentity() == nil {
		s.commitMu.Lock()
		defer s.commitMu.Unlock()
		s.delete(key)
		return
	}
	s.overlay.unstage(tx, key)
}

//nolint:unused
func (s *boolLookupStorage[K]) resolve(tx Tx, committed []K, values []bool) []K {
	return s.overlay.resolve(tx, committed, values)
}

func (s *boolLookupStorage[K]) flush(d *delta[K, bool]) {
	s.commitMu.Lock()
	defer s.commitMu.Unlock()
	for k, entry := range d.state {
		if entry.deleted {
			s.delete(k)
			continue
		}
		s.put(k, entry.value)
	}
}

// bytesLookupStorage backs a Lookup whose primary key type is []byte.
// Because []byte is not comparable it cannot key Go maps directly; the
// reverse map and staging overlay use string-conversions at the boundary
// (string([]byte) is no-alloc when used as a map key in modern Go). The
// forward map stores cloned []byte entries so callers can retain the
// returned slices.
type bytesLookupStorage[V comparable] struct {
	commitMu *sync.RWMutex
	forward  map[V][][]byte
	reverse  map[string]V
	overlay  deltaOverlay[string, V]
}

func newBytesLookupStorage[V comparable](
	commitMu *sync.RWMutex,
) *bytesLookupStorage[V] {
	s := &bytesLookupStorage[V]{
		commitMu: commitMu,
		forward:  make(map[V][][]byte),
		reverse:  make(map[string]V),
	}
	s.overlay.flush = s.flush
	return s
}

//nolint:unused
func (s *bytesLookupStorage[V]) put(key []byte, value V) {
	skey := string(key)
	if oldValue, existed := s.reverse[skey]; existed {
		if oldValue == value {
			return
		}
		s.removeFromForward(key, oldValue)
	}
	s.forward[value] = append(s.forward[value], slices.Clone(key))
	s.reverse[skey] = value
}

//nolint:unused
func (s *bytesLookupStorage[V]) delete(key []byte) {
	skey := string(key)
	oldValue, existed := s.reverse[skey]
	if !existed {
		return
	}
	s.removeFromForward(key, oldValue)
	delete(s.reverse, skey)
}

//nolint:unused
func (s *bytesLookupStorage[V]) removeFromForward(key []byte, value V) {
	keys := s.forward[value]
	for i, k := range keys {
		if string(k) == string(key) {
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

//nolint:unused
func (s *bytesLookupStorage[V]) get(value V) [][]byte {
	src := s.forward[value]
	if len(src) == 0 {
		return nil
	}
	out := make([][]byte, len(src))
	for i, k := range src {
		out[i] = slices.Clone(k)
	}
	return out
}

//nolint:unused
func (s *bytesLookupStorage[V]) stageSet(tx Tx, key []byte, value V) {
	if tx.txIdentity() == nil {
		s.commitMu.Lock()
		defer s.commitMu.Unlock()
		s.put(key, value)
		return
	}
	s.overlay.stage(tx, string(key), value)
}

//nolint:unused
func (s *bytesLookupStorage[V]) stageDelete(tx Tx, key []byte) {
	if tx.txIdentity() == nil {
		s.commitMu.Lock()
		defer s.commitMu.Unlock()
		s.delete(key)
		return
	}
	s.overlay.unstage(tx, string(key))
}

//nolint:unused
func (s *bytesLookupStorage[V]) resolve(tx Tx, committed [][]byte, values []V) [][]byte {
	if tx.txIdentity() == nil {
		return committed
	}
	committedStrings := make([]string, len(committed))
	for i, k := range committed {
		committedStrings[i] = string(k)
	}
	resolved := s.overlay.resolve(tx, committedStrings, values)
	out := make([][]byte, len(resolved))
	for i, sk := range resolved {
		out[i] = []byte(sk)
	}
	return out
}

func (s *bytesLookupStorage[V]) flush(d *delta[string, V]) {
	s.commitMu.Lock()
	defer s.commitMu.Unlock()
	for skey, entry := range d.state {
		bkey := []byte(skey)
		if entry.deleted {
			s.delete(bkey)
			continue
		}
		s.put(bkey, entry.value)
	}
}

// sortedEntry is a single (value, key) pair inside a Sorted index slice.
type sortedEntry[K IndexKey, V cmp.Ordered] struct {
	Value V
	Key   K
}

// sortedStorage backs a Sorted index. It keeps entries in ascending order
// of V using the native `<` operator (V is constrained to cmp.Ordered).
// Insertion is O(log n) binary search plus O(n) slice shift. At the
// target scale (<100k entries) this is acceptable; if profiling shows
// the shift cost matters, swap the backing for a B-tree without changing
// the outer API. Within equal values, entries are kept in insertion
// order; removal scans that sub-range for an exact key match.
type sortedStorage[K IndexKey, V cmp.Ordered] struct {
	entries []sortedEntry[K, V]
}

func newSortedStorage[K IndexKey, V cmp.Ordered]() *sortedStorage[K, V] {
	return &sortedStorage[K, V]{}
}

// lowerBound returns the first index i such that entries[i].Value >= value.
func (s *sortedStorage[K, V]) lowerBound(value V) int {
	return sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].Value >= value
	})
}

// upperBound returns the first index i such that entries[i].Value > value.
func (s *sortedStorage[K, V]) upperBound(value V) int {
	return sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].Value > value
	})
}

//nolint:unused
func (s *sortedStorage[K, V]) put(key K, value V) {
	i := s.upperBound(value)
	s.entries = slices.Insert(s.entries, i, sortedEntry[K, V]{Value: value, Key: key})
}

//nolint:unused
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

// sortBulk sorts the entries slice by Value. Used by Sorted.populate to
// finalize a bulk-loaded index in O(N log N) instead of inserting one
// entry at a time at O(N²).
//
//nolint:unused
func (s *sortedStorage[K, V]) sortBulk() {
	slices.SortFunc(s.entries, func(a, b sortedEntry[K, V]) int {
		return cmp.Compare(a.Value, b.Value)
	})
}
