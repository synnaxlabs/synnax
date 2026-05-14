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
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/x/errors"
)

// ErrIndexInvalid indicates that a registered secondary index failed to
// populate. Retrieve.Where(idx.Filter(...)) catches this sentinel and falls
// back to a sequential scan with a synthesized predicate, so queries continue
// to return correct results — just at scan cost. Direct callers of Get / GetTx
// receive the sentinel and are responsible for their own fallback (see
// ontology.parentsByIndex for the canonical pattern).
//
// A populate failure is logged via the table's instrumentation and stays
// sticky for the lifetime of the process: there is no automatic rebuild.
// Operators should investigate the underlying error and restart the affected
// node to retry population.
var ErrIndexInvalid = errors.New("gorp: index failed to populate")

// Index is a registered secondary index on a Table. Implementations are
// provided by gorp (LookupIndex, SortedIndex) and constructed via NewLookupIndex,
// NewBytesLookup, and NewSortedIndex. The interface methods are unexported so
// external code cannot substitute custom implementations; callers should
// use the provided generic types.
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
	populate() (func(entry E), func(error), error)
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

// LookupIndex is an in-memory exact-match secondary index on a field of type V
// extracted from entries of type E. Construct comparable-keyed indexes via
// NewLookupIndex and byte-keyed indexes via NewBytesLookup; both produce a
// *LookupIndex, parameterized by the appropriate K. Register on a Table through
// TableConfig.Indexes.
//
// LookupIndex itself is data-type-agnostic: it owns the populate state machine,
// the E → V extraction, and Filter construction. Per-key bookkeeping
// (forward / reverse maps, per-tx staging deltas, commit-time flush) lives
// on the storage backend selected by the constructor.
type LookupIndex[K Key, E Entry[K], V comparable] struct {
	name    string
	extract func(e *E) V
	mu      sync.RWMutex
	storage lookupStorage[K, V]
	// buildMembership constructs an O(1) membership predicate over a
	// resolved key set. Supplied by the constructor with the right shape
	// for K (typed map for comparable K, string-keyed map for []byte).
	buildMembership func([]K) keyMembership[K]
	// populateDone is closed by populate's finish closure once the index
	// has been populated from the table (or once the populate scan has
	// failed). Filter waits on this so reads issued before the index is
	// ready block instead of returning empty results.
	populateDone chan struct{}
	// populateErr captures the populate scan error, if any. Filter checks
	// it after populateDone closes to decide whether to use the index or
	// fall back to a sequential scan via eval.
	populateErr atomic.Pointer[error]
}

// NewLookupIndex constructs a LookupIndex over a comparable primary key K. The bool
// specialization is selected automatically when V is bool. The returned
// index is empty; register it on a Table through TableConfig.Indexes to
// populate it from the existing table contents and keep it in sync with
// future writes.
func NewLookupIndex[K ComparableKey, E Entry[K], V comparable](
	name string,
	extract func(e *E) V,
) *LookupIndex[K, E, V] {
	l := &LookupIndex[K, E, V]{
		name:            name,
		extract:         extract,
		buildMembership: indexedKeyMembership[K],
		populateDone:    make(chan struct{}),
	}
	var zeroV V
	if _, ok := any(zeroV).(bool); ok {
		l.storage = any(newBoolLookupStorage[K](&l.mu)).(lookupStorage[K, V])
	} else {
		l.storage = newMapLookupStorage[K, V](&l.mu)
	}
	return l
}

// NewBytesLookup constructs a LookupIndex over a []byte primary key. The shape
// and semantics mirror NewLookupIndex; the only difference is the keying
// strategy required because []byte does not satisfy comparable. Use
// NewLookupIndex whenever K is comparable; reach for NewBytesLookup only when
// the table key is genuinely []byte (e.g. composite keys encoded inline).
func NewBytesLookup[E Entry[[]byte], V comparable](
	name string,
	extract func(e *E) V,
) *LookupIndex[[]byte, E, V] {
	l := &LookupIndex[[]byte, E, V]{
		name:            name,
		extract:         extract,
		buildMembership: bytesIndexedKeyMembership,
		populateDone:    make(chan struct{}),
	}
	l.storage = newBytesLookupStorage[V](&l.mu)
	return l
}

// Name implements Index.
func (l *LookupIndex[K, E, V]) Name() string { return l.name }

//nolint:unused
func (l *LookupIndex[K, E, V]) populate() (func(E), func(error), error) {
	l.mu.Lock()
	insert := func(entry E) {
		l.storage.put(entry.GorpKey(), l.extract(&entry))
	}
	finish := func(err error) {
		if err != nil {
			l.populateErr.Store(&err)
		}
		close(l.populateDone)
		l.mu.Unlock()
	}
	return insert, finish, nil
}

//nolint:unused
func (l *LookupIndex[K, E, V]) set(entry E) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.storage.put(entry.GorpKey(), l.extract(&entry))
}

//nolint:unused
func (l *LookupIndex[K, E, V]) delete(key K) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.storage.delete(key)
}

//nolint:unused
func (l *LookupIndex[K, E, V]) stageSet(tx Tx, entry E) {
	l.storage.stageSet(tx, entry.GorpKey(), l.extract(&entry))
}

//nolint:unused
func (l *LookupIndex[K, E, V]) stageDelete(tx Tx, key K) {
	l.storage.stageDelete(tx, key)
}

// Get returns the primary keys of entries whose indexed field matches any
// of the provided values. Get blocks until the index has finished
// populating. After populate settles, Get returns ErrIndexInvalid if
// population failed; callers should fall back to a sequential scan or
// surface the error to their caller.
//
// Returned slices are owned by the caller and may be freely retained.
// For byte-keyed indexes (LookupIndex[[]byte, ...]) each returned key is a
// fresh clone.
func (l *LookupIndex[K, E, V]) Get(values ...V) ([]K, error) {
	if len(values) == 0 {
		return nil, nil
	}
	<-l.populateDone
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.populateErr.Load() != nil {
		return nil, errors.Wrapf(ErrIndexInvalid, "lookup %q", l.name)
	}
	if len(values) == 1 {
		return l.storage.get(values[0]), nil
	}
	var out []K
	for _, v := range values {
		out = append(out, l.storage.get(v)...)
	}
	return out, nil
}

// GetTx is the tx-aware counterpart to Get. It returns the primary keys
// of entries whose indexed field matches any of the provided values,
// merging committed index state with any per-tx delta staged against
// the open transaction. When tx has no per-tx scoping or no staged
// mutations for this index, it returns the same result as Get. Use
// GetTx when consuming keys directly outside of Retrieve — e.g. graph
// traversal helpers that probe the index for candidate IDs. Returns
// ErrIndexInvalid when population failed; see Get for the failure-mode
// contract.
func (l *LookupIndex[K, E, V]) GetTx(tx Tx, values ...V) ([]K, error) {
	if len(values) == 0 {
		return nil, nil
	}
	committed, err := l.Get(values...)
	if err != nil {
		return nil, err
	}
	return l.storage.resolve(tx, committed, values), nil
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
		resolve: func(ctx context.Context, tx Tx) ([]K, func([]K) keyMembership[K], error) {
			select {
			case <-l.populateDone:
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
			if l.populateErr.Load() != nil {
				return nil, nil, errors.Wrapf(ErrIndexInvalid, "lookup %q", l.name)
			}
			committed, err := l.Get(captured...)
			if err != nil {
				return nil, nil, err
			}
			return l.storage.resolve(tx, committed, captured), l.buildMembership, nil
		},
		eval: func(_ Context, e *E, _, _ []byte) (bool, error) {
			return slices.Contains(captured, l.extract(e)), nil
		},
	}
}

// SortedIndex is an ordered in-memory index on a field of type V extracted from
// entries of type E. V is constrained to cmp.Ordered so the storage can
// compare values without a caller-supplied comparator. SortedIndex supports
// exact-match lookups via Filter (same semantics as LookupIndex) and ordered
// cursor-based pagination via Retrieve.OrderBy.
//
// Read-your-own-writes is v1-scoped to equality Filter. Ordered cursor
// iteration via Retrieve.OrderBy does NOT reflect uncommitted tx
// writes; an open write tx that staged inserts or deletes will not
// see those changes during ordered iteration.
type SortedIndex[K ComparableKey, E Entry[K], V cmp.Ordered] struct {
	name    string
	extract func(e *E) V
	mu      sync.RWMutex
	storage *sortedStorage[K, V]
	reverse map[K]V
	overlay deltaOverlay[K, V]
	// populateDone mirrors LookupIndex.populateDone: closed by populate's
	// finish closure once the index is ready (or has settled with an
	// error).
	populateDone chan struct{}
	// populateErr mirrors LookupIndex.populateErr.
	populateErr atomic.Pointer[error]
}

// NewSortedIndex constructs a SortedIndex index over the provided extract function.
// V must satisfy cmp.Ordered (any built-in ordered primitive: signed and
// unsigned integers, floats, or strings).
func NewSortedIndex[K ComparableKey, E Entry[K], V cmp.Ordered](
	name string,
	extract func(e *E) V,
) *SortedIndex[K, E, V] {
	s := &SortedIndex[K, E, V]{
		name:         name,
		extract:      extract,
		storage:      newSortedStorage[K, V](),
		reverse:      make(map[K]V),
		populateDone: make(chan struct{}),
	}
	s.overlay.flush = s.flushTx
	return s
}

// Name implements Index.
func (s *SortedIndex[K, E, V]) Name() string { return s.name }

//nolint:unused
func (s *SortedIndex[K, E, V]) populate() (func(E), func(error), error) {
	s.mu.Lock()
	insert := func(entry E) {
		key := entry.GorpKey()
		value := s.extract(&entry)
		s.storage.entries = append(
			s.storage.entries,
			sortedEntry[K, V]{value: value, key: key},
		)
		s.reverse[key] = value
	}
	finish := func(err error) {
		if err != nil {
			s.populateErr.Store(&err)
		} else {
			s.storage.sortBulk()
		}
		close(s.populateDone)
		s.mu.Unlock()
	}
	return insert, finish, nil
}

//nolint:unused
func (s *SortedIndex[K, E, V]) set(entry E) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := entry.GorpKey()
	newValue := s.extract(&entry)
	if oldValue, existed := s.reverse[key]; existed {
		if cmp.Compare(oldValue, newValue) == 0 {
			return
		}
		s.storage.remove(key, oldValue)
	}
	s.storage.put(key, newValue)
	s.reverse[key] = newValue
}

//nolint:unused
func (s *SortedIndex[K, E, V]) delete(key K) {
	s.mu.Lock()
	defer s.mu.Unlock()
	oldValue, existed := s.reverse[key]
	if !existed {
		return
	}
	s.storage.remove(key, oldValue)
	delete(s.reverse, key)
}

//nolint:unused
func (s *SortedIndex[K, E, V]) stageSet(tx Tx, entry E) {
	if tx.txIdentity() == nil {
		s.set(entry)
		return
	}
	s.overlay.stage(tx, entry.GorpKey(), s.extract(&entry))
}

//nolint:unused
func (s *SortedIndex[K, E, V]) stageDelete(tx Tx, key K) {
	if tx.txIdentity() == nil {
		s.delete(key)
		return
	}
	s.overlay.unstage(tx, key)
}

// flushTx promotes the staged tx delta into committed index state.
func (s *SortedIndex[K, E, V]) flushTx(d *delta[K, V]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, entry := range d.state {
		oldValue, existed := s.reverse[k]
		if entry.deleted {
			if existed {
				s.storage.remove(k, oldValue)
				delete(s.reverse, k)
			}
			continue
		}
		if existed {
			if cmp.Compare(oldValue, entry.value) == 0 {
				continue
			}
			s.storage.remove(k, oldValue)
		}
		s.storage.put(k, entry.value)
		s.reverse[k] = entry.value
	}
}

// resolveTx mirrors Lookup.resolveTx — Filter has already gated on
// populateErr by the time this runs, so any error from Get is dropped.
func (s *SortedIndex[K, E, V]) resolveTx(tx Tx, values []V) []K {
	keys, _ := s.Get(values...)
	return s.overlay.resolve(tx, keys, values)
}

// Get returns the primary keys of entries whose indexed field matches any
// of the provided values. See LookupIndex.Get for the populate-failure contract.
func (s *SortedIndex[K, E, V]) Get(values ...V) ([]K, error) {
	if len(values) == 0 {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.populateErr.Load() != nil {
		return nil, errors.Wrapf(ErrIndexInvalid, "sorted %q", s.name)
	}
	if len(values) == 1 {
		src := s.storage.get(values[0])
		out := make([]K, len(src))
		copy(out, src)
		return out, nil
	}
	var out []K
	for _, v := range values {
		out = append(out, s.storage.get(v)...)
	}
	return out, nil
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
		resolve: func(ctx context.Context, tx Tx) ([]K, func([]K) keyMembership[K], error) {
			select {
			case <-s.populateDone:
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			}
			if s.populateErr.Load() != nil {
				return nil, nil, errors.Wrapf(ErrIndexInvalid, "sorted %q", s.name)
			}
			return s.resolveTx(tx, captured), indexedKeyMembership[K], nil
		},
		eval: func(_ Context, e *E, _, _ []byte) (bool, error) {
			return slices.Contains(captured, s.extract(e)), nil
		},
	}
}

// GetTx is the tx-aware counterpart to Get. See LookupIndex.GetTx for
// semantics.
func (s *SortedIndex[K, E, V]) GetTx(tx Tx, values ...V) ([]K, error) {
	if len(values) == 0 {
		return nil, nil
	}
	committed, err := s.Get(values...)
	if err != nil {
		return nil, err
	}
	return s.overlay.resolve(tx, committed, values), nil
}
