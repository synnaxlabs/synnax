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
	"context"
	"slices"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/x/errors"
)

// ErrIndexInvalid indicates that a registered secondary index failed to
// populate. Retrieve.Where(idx.Filter(...)) catches this sentinel and falls
// back to a sequential scan with a synthesized predicate, so queries continue
// to return correct results — just at scan cost. Direct callers of Get
// receive the sentinel and are responsible for their own fallback (see
// ontology.parentsByIndex for the canonical pattern).
//
// A populate failure is logged via the table's instrumentation and stays
// sticky for the lifetime of the process: there is no automatic rebuild.
// Operators should investigate the underlying error and restart the affected
// node to retry population.
var ErrIndexInvalid = errors.New("gorp: index failed to populate")

// Index is a registered secondary index on a Table. Implementations are
// provided by gorp (LookupIndex, SortedIndex) and constructed via
// NewLookupIndex and NewSortedIndex. The interface methods are unexported
// so external code cannot substitute custom implementations; callers
// should use the provided generic types.
type Index[K Key, E Entry[K]] interface {
	// Name returns the human-readable name of the index, used in diagnostics.
	Name() string
	// populate returns an insert closure and a finish closure for the
	// populate phase. The caller must invoke insert once for every existing
	// entry in the table and then invoke finish exactly once with the
	// terminal scan error (or nil on success). finish is mandatory:
	// implementations hold a write lock across the populate phase and
	// signal readiness to readers blocked in Filter; skipping finish leaks
	// the lock and deadlocks every blocked reader.
	populate() (func(entry E), func(error))
	// set records entry in committed index state, keyed by entry.GorpKey(),
	// replacing any prior mapping for the same key.
	set(entry E)
	// delete removes any committed mapping for key.
	delete(key K)
	// stageSet records a pending insert or update of entry against tx's
	// per-tx delta, keyed by entry.GorpKey(). Committed index state is
	// not modified until tx commits. When tx has no per-tx identity (a
	// DB used directly), the mutation applies to committed state
	// immediately.
	stageSet(tx Tx, entry E)
	// stageDelete records a pending deletion of key against tx's per-tx
	// delta. Committed index state is not modified until tx commits.
	// When tx has no per-tx identity, the deletion applies to committed
	// state immediately.
	stageDelete(tx Tx, key K)
}

// baseIndex holds the scaffolding shared by every concrete index
// implementation. Concrete types embed it as a value.
type baseIndex[K Key, E Entry[K]] struct {
	// name is the human-readable index name, used in diagnostics and
	// in the ErrIndexInvalid wrap.
	name string
	// mu guards the concrete index's committed state. Held for writing
	// during populate and flush; held for reading during Get.
	mu sync.RWMutex
	// populateDone is closed by populate's finish closure once the
	// index has been populated (or has settled with an error). Filter
	// waits on this so reads issued before the index is ready block
	// instead of returning empty results.
	populateDone chan struct{}
	// populateErr captures the populate scan error, if any. Checked
	// after populateDone closes to decide whether the index is usable
	// or queries must fall back to a sequential scan.
	populateErr atomic.Pointer[error]
}

func (b *baseIndex[K, E]) Name() string { return b.name }

// waitPopulated blocks until populate finishes or ctx is canceled,
// returning ErrIndexInvalid (wrapped with the index name) if populate
// failed.
func (b *baseIndex[K, E]) waitPopulated(ctx context.Context) error {
	select {
	case <-b.populateDone:
	case <-ctx.Done():
		return ctx.Err()
	}
	if b.populateErr.Load() != nil {
		return errors.Wrapf(ErrIndexInvalid, "%q", b.name)
	}
	return nil
}

// populateErrWrapped returns ErrIndexInvalid wrapped with the index
// name when populate failed. Callers must have already synchronized
// with populateDone.
func (b *baseIndex[K, E]) populateErrWrapped() error {
	if b.populateErr.Load() != nil {
		return errors.Wrapf(ErrIndexInvalid, "%q", b.name)
	}
	return nil
}

// LookupIndex is an in-memory exact-match secondary index on a field of
// type V extracted from entries of type E. Construct via NewLookupIndex
// and register on a Table through TableConfig.Indexes.
//
// LookupIndex owns the populate state machine, the E → V extraction,
// Filter construction, and the per-key bookkeeping (forward / reverse
// maps, per-tx staging deltas, commit-time flush) inline. The mutex
// inherited from baseIndex guards both committed maps and serves as the
// commit-time lock for the staging overlay.
type LookupIndex[K Key, E Entry[K], V comparable] struct {
	// baseIndex carries the populate lifecycle, mutex, and name.
	baseIndex[K, E]
	// extract reads the indexed field from an entry.
	extract func(e *E) V
	// forward is the value-to-keys map.
	forward map[V][]K
	// reverse is the key-to-value map; used to locate the old bucket
	// when an entry's value changes.
	reverse map[K]V
	// overlay tracks per-tx staging deltas.
	overlay deltaOverlay[K, V]
}

// NewLookupIndex constructs a LookupIndex over a comparable primary key K. The
// returned index is empty; register it on a Table through TableConfig.Indexes
// to populate it from the existing table contents and keep it in sync with
// future writes.
func NewLookupIndex[K Key, E Entry[K], V comparable](
	name string,
	extract func(e *E) V,
) *LookupIndex[K, E, V] {
	l := &LookupIndex[K, E, V]{
		name:         name,
		populateDone: make(chan struct{}),
		extract:      extract,
		forward:      make(map[V][]K),
		reverse:      make(map[K]V),
	}
	l.overlay.commitSet = func(key K, value V) {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.putLocked(key, value)
	}
	l.overlay.commitDelete = func(key K) {
		l.mu.Lock()
		defer l.mu.Unlock()
		l.deleteLocked(key)
	}
	l.overlay.flush = l.flush
	return l
}

// putLocked records (key, value) in committed state. Caller must hold
// l.mu for writing.
func (l *LookupIndex[K, E, V]) putLocked(key K, value V) {
	if oldValue, existed := l.reverse[key]; existed {
		if oldValue == value {
			return
		}
		l.removeFromForward(key, oldValue)
	}
	l.forward[value] = append(l.forward[value], key)
	l.reverse[key] = value
}

// deleteLocked removes any committed mapping for key. Caller must hold
// l.mu for writing.
func (l *LookupIndex[K, E, V]) deleteLocked(key K) {
	oldValue, existed := l.reverse[key]
	if !existed {
		return
	}
	l.removeFromForward(key, oldValue)
	delete(l.reverse, key)
}

// removeFromForward removes key from the forward bucket for value.
// Caller must hold l.mu for writing.
func (l *LookupIndex[K, E, V]) removeFromForward(key K, value V) {
	keys := l.forward[value]
	for i, k := range keys {
		if k == key {
			keys = slices.Delete(keys, i, i+1)
			break
		}
	}
	if len(keys) == 0 {
		delete(l.forward, value)
		return
	}
	l.forward[value] = keys
}

// getLocked returns the committed keys mapped to value. Caller must hold
// l.mu (read or write).
//
//nolint:unused
func (l *LookupIndex[K, E, V]) getLocked(value V) []K {
	src := l.forward[value]
	if len(src) == 0 {
		return nil
	}
	out := make([]K, len(src))
	copy(out, src)
	return out
}

// flush applies a committed tx's staged delta into committed state. It
// is wired into overlay.flush at construction and runs without any
// gorp-side lock held; it acquires l.mu before mutating.
func (l *LookupIndex[K, E, V]) flush(d *delta[K, V]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, entry := range d.state {
		if entry.deleted {
			l.deleteLocked(k)
			continue
		}
		l.putLocked(k, entry.value)
	}
}

//nolint:unused
func (l *LookupIndex[K, E, V]) populate() (func(E), func(error)) {
	l.mu.Lock()
	insert := func(entry E) {
		l.putLocked(entry.GorpKey(), l.extract(&entry))
	}
	finish := func(err error) {
		if err != nil {
			l.populateErr.Store(&err)
		}
		close(l.populateDone)
		l.mu.Unlock()
	}
	return insert, finish
}

//nolint:unused
func (l *LookupIndex[K, E, V]) set(entry E) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.putLocked(entry.GorpKey(), l.extract(&entry))
}

//nolint:unused
func (l *LookupIndex[K, E, V]) delete(key K) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.deleteLocked(key)
}

//nolint:unused
func (l *LookupIndex[K, E, V]) stageSet(tx Tx, entry E) {
	l.overlay.stage(tx, entry.GorpKey(), l.extract(&entry))
}

//nolint:unused
func (l *LookupIndex[K, E, V]) stageDelete(tx Tx, key K) {
	l.overlay.unstage(tx, key)
}

// Get returns the primary keys of entries whose indexed field matches
// any of the provided values, merging committed index state with any
// per-tx delta staged against tx. Pass a nil tx to read committed state
// only.
//
// Get blocks until the index has finished populating. After populate
// settles, Get returns ErrIndexInvalid if population failed; callers
// should fall back to a sequential scan or surface the error.
//
// Returned slices are owned by the caller and may be freely retained.
func (l *LookupIndex[K, E, V]) Get(tx Tx, values ...V) ([]K, error) {
	if len(values) == 0 {
		return nil, nil
	}
	<-l.populateDone
	l.mu.RLock()
	defer l.mu.RUnlock()
	if err := l.populateErrWrapped(); err != nil {
		return nil, err
	}
	var committed []K
	if len(values) == 1 {
		committed = l.getLocked(values[0])
	} else {
		for _, v := range values {
			committed = append(committed, l.getLocked(v)...)
		}
	}
	if tx == nil {
		return committed, nil
	}
	return l.overlay.resolve(tx, committed, values), nil
}

// Filter returns a Filter[K, E] matching entries whose indexed field is
// any of values. The filter sees read-your-own-writes: an indexed
// Retrieve inside a write tx that created, updated, or deleted an
// entry observes those pending changes alongside committed index
// state. A Retrieve against a DB used directly returns committed
// state only.
//
// Filter blocks the Retrieve until the index has finished populating.
// If population fails, the resolver returns ErrIndexInvalid; Retrieve
// catches the sentinel and falls back to a sequential scan using the
// eval predicate carried alongside the resolver, so queries continue
// to return correct results at scan cost.
func (l *LookupIndex[K, E, V]) Filter(values ...V) Filter[K, E] {
	captured := append([]V(nil), values...)
	return Filter[K, E]{
		resolve: func(ctx context.Context, tx Tx) (resolved[K, E], error) {
			if err := l.waitPopulated(ctx); err != nil {
				return resolved[K, E]{}, err
			}
			keys, err := l.Get(tx, captured...)
			if err != nil {
				return resolved[K, E]{}, err
			}
			return resolved[K, E]{keys: keys, build: indexedKeyMembership[K]}, nil
		},
		eval: func(_ Context, e *E, _, _ []byte) (bool, error) {
			return slices.Contains(captured, l.extract(e)), nil
		},
	}
}

// SortedIndex is an ordered in-memory index on a field of type V extracted from
// entries of type E. V is constrained to cmp.Ordered so the index can
// compare values without a caller-supplied comparator. SortedIndex supports
// exact-match lookups via Filter (same semantics as LookupIndex) and ordered
// cursor-based pagination via Retrieve.OrderBy.
//
// Read-your-own-writes is v1-scoped to equality Filter. Ordered cursor
// iteration via Retrieve.OrderBy does NOT reflect uncommitted tx
// writes; an open write tx that staged inserts or deletes will not
// see those changes during ordered iteration.
type SortedIndex[K Key, E Entry[K], V cmp.Ordered] struct {
	// baseIndex carries the populate lifecycle, mutex, and name.
	baseIndex[K, E]
	// extract reads the indexed field from an entry.
	extract func(e *E) V
	// entries holds the index contents in ascending V order. Within
	// equal values, entries are kept in insertion order. Insertion is
	// O(log n) binary search plus O(n) slice shift; at the target scale
	// (<100k entries) this is acceptable.
	entries []sortedEntry[K, V]
	// reverse maps each primary key to its current indexed value; used
	// to locate the old slot when an entry's value changes.
	reverse map[K]V
	// overlay tracks per-tx staging deltas for read-your-own-writes
	// equality lookups.
	overlay deltaOverlay[K, V]
}

// NewSortedIndex constructs a SortedIndex index over the provided extract function.
// V must satisfy cmp.Ordered (any built-in ordered primitive: signed and
// unsigned integers, floats, or strings).
func NewSortedIndex[K Key, E Entry[K], V cmp.Ordered](
	name string,
	extract func(e *E) V,
) *SortedIndex[K, E, V] {
	s := &SortedIndex[K, E, V]{
		name:         name,
		populateDone: make(chan struct{}),
		extract:      extract,
		reverse:      make(map[K]V),
	}
	s.overlay.commitSet = func(key K, value V) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.setCommitted(key, value)
	}
	s.overlay.commitDelete = func(key K) {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.deleteCommitted(key)
	}
	s.overlay.flush = s.flushTx
	return s
}

//nolint:unused
func (s *SortedIndex[K, E, V]) populate() (func(E), func(error)) {
	s.mu.Lock()
	insert := func(entry E) {
		key := entry.GorpKey()
		value := s.extract(&entry)
		s.entries = append(
			s.entries,
			sortedEntry[K, V]{value: value, key: key},
		)
		s.reverse[key] = value
	}
	finish := func(err error) {
		if err != nil {
			s.populateErr.Store(&err)
		} else {
			s.sortBulk()
		}
		close(s.populateDone)
		s.mu.Unlock()
	}
	return insert, finish
}

// lowerBound returns the first index i such that entries[i].value >= value.
// Caller must hold s.mu.
func (s *SortedIndex[K, E, V]) lowerBound(value V) int {
	return sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].value >= value
	})
}

// upperBound returns the first index i such that entries[i].value > value.
// Caller must hold s.mu.
func (s *SortedIndex[K, E, V]) upperBound(value V) int {
	return sort.Search(len(s.entries), func(i int) bool {
		return s.entries[i].value > value
	})
}

// put inserts (key, value) into the sorted slice at the upper bound of
// value, so entries with equal values stay in insertion order. Caller
// must hold s.mu for writing.
func (s *SortedIndex[K, E, V]) put(key K, value V) {
	i := s.upperBound(value)
	s.entries = slices.Insert(s.entries, i, sortedEntry[K, V]{value: value, key: key})
}

// remove deletes the (key, value) pair from the sorted slice. Caller
// must hold s.mu for writing.
func (s *SortedIndex[K, E, V]) remove(key K, value V) {
	lo := s.lowerBound(value)
	hi := s.upperBound(value)
	for i := lo; i < hi; i++ {
		if s.entries[i].key == key {
			s.entries = slices.Delete(s.entries, i, i+1)
			return
		}
	}
}

// get returns the primary keys whose indexed value equals value. Caller
// must hold s.mu (read or write).
func (s *SortedIndex[K, E, V]) get(value V) []K {
	lo := s.lowerBound(value)
	hi := s.upperBound(value)
	if lo == hi {
		return nil
	}
	out := make([]K, hi-lo)
	for i := lo; i < hi; i++ {
		out[i-lo] = s.entries[i].key
	}
	return out
}

// sortBulk sorts entries by value. Used by populate to finalize a
// bulk-loaded index in O(N log N) instead of inserting one entry at a
// time at O(N²).
//
//nolint:unused
func (s *SortedIndex[K, E, V]) sortBulk() {
	slices.SortFunc(s.entries, func(a, b sortedEntry[K, V]) int {
		return cmp.Compare(a.value, b.value)
	})
}

// setCommitted applies a write to committed state. Caller must hold s.mu.
func (s *SortedIndex[K, E, V]) setCommitted(key K, value V) {
	if oldValue, existed := s.reverse[key]; existed {
		if cmp.Compare(oldValue, value) == 0 {
			return
		}
		s.remove(key, oldValue)
	}
	s.put(key, value)
	s.reverse[key] = value
}

// deleteCommitted removes a key from committed state. Caller must hold s.mu.
func (s *SortedIndex[K, E, V]) deleteCommitted(key K) {
	oldValue, existed := s.reverse[key]
	if !existed {
		return
	}
	s.remove(key, oldValue)
	delete(s.reverse, key)
}

//nolint:unused
func (s *SortedIndex[K, E, V]) set(entry E) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setCommitted(entry.GorpKey(), s.extract(&entry))
}

//nolint:unused
func (s *SortedIndex[K, E, V]) delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.deleteCommitted(key)
}

//nolint:unused
func (s *SortedIndex[K, E, V]) stageSet(tx Tx, entry E) {
	s.overlay.stage(tx, entry.GorpKey(), s.extract(&entry))
}

//nolint:unused
func (s *SortedIndex[K, E, V]) stageDelete(tx Tx, key K) {
	s.overlay.unstage(tx, key)
}

// flushTx promotes the staged tx delta into committed index state.
func (s *SortedIndex[K, E, V]) flushTx(d *delta[K, V]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, entry := range d.state {
		if entry.deleted {
			s.deleteCommitted(k)
			continue
		}
		s.setCommitted(k, entry.value)
	}
}

// Get returns the primary keys of entries whose indexed field matches
// any of the provided values, merging committed index state with any
// per-tx delta staged against tx. Pass a nil tx to read committed state
// only. See LookupIndex.Get for the populate-failure contract.
func (s *SortedIndex[K, E, V]) Get(tx Tx, values ...V) ([]K, error) {
	if len(values) == 0 {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if err := s.populateErrWrapped(); err != nil {
		return nil, err
	}
	var committed []K
	if len(values) == 1 {
		src := s.get(values[0])
		committed = make([]K, len(src))
		copy(committed, src)
	} else {
		for _, v := range values {
			committed = append(committed, s.get(v)...)
		}
	}
	if tx == nil {
		return committed, nil
	}
	return s.overlay.resolve(tx, committed, values), nil
}

// Filter returns an exact-match Filter[K, E] matching entries whose
// indexed field is any of values. Read-your-own-writes semantics
// match LookupIndex.Filter. Ordered cursor iteration (SortedIndex.Ordered /
// OrderBy) is not covered; only equality Filter.
//
// Filter blocks the Retrieve until the index has finished populating
// and falls back to a sequential scan when populate fails. See
// LookupIndex.Filter for the populate-failure contract.
func (s *SortedIndex[K, E, V]) Filter(values ...V) Filter[K, E] {
	captured := append([]V(nil), values...)
	return Filter[K, E]{
		resolve: func(ctx context.Context, tx Tx) (resolved[K, E], error) {
			if err := s.waitPopulated(ctx); err != nil {
				return resolved[K, E]{}, err
			}
			keys, _ := s.Get(tx, captured...)
			return resolved[K, E]{keys: keys, build: indexedKeyMembership[K]}, nil
		},
		eval: func(_ Context, e *E, _, _ []byte) (bool, error) {
			return slices.Contains(captured, s.extract(e)), nil
		},
	}
}
