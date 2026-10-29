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
	"context"
	"slices"

	"github.com/synnaxlabs/gorp/internal/index"
	"github.com/synnaxlabs/gorp/internal/keyset"
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

// Index is a registered secondary index on a Table. Implementations are provided by
// gorp (LookupIndex, SearchIndex) and constructed via NewLookupIndex and
// NewSearchIndex. The interface methods are unexported so external
// code cannot substitute custom implementations; callers should use the provided
// generic types.
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

// indexStore is the read side of the engine behind an index.
type indexStore[K Key, V comparable] interface {
	Wait(ctx context.Context) error
	Get(scope *txState, values ...V) ([]K, error)
}

// populateErr returns the error that every read of the index named name returns after
// a populate that ended with err.
func populateErr(name string, err error) error {
	if err == nil {
		return nil
	}
	return errors.Wrapf(ErrIndexInvalid, "%q", name)
}

// scopeOf returns the per-tx state of tx, or nil when tx is nil or is not a real
// transaction.
func scopeOf(tx Tx) *txState {
	if tx == nil {
		return nil
	}
	return tx.txIdentity()
}

// indexFilter returns a Filter matching entries whose field, read by extract, is any of
// values. The filter resolves its keys through store at Exec time and carries an eval
// predicate for the sequential-scan fallback.
func indexFilter[K Key, E Entry[K], V comparable](
	store indexStore[K, V],
	extract func(e *E) V,
	values []V,
) Filter[K, E] {
	captured := append([]V(nil), values...)
	return Filter[K, E]{
		resolve: func(ctx context.Context, tx Tx) (resolved[K, E], error) {
			if err := store.Wait(ctx); err != nil {
				return resolved[K, E]{}, err
			}
			keys, err := store.Get(scopeOf(tx), captured...)
			if err != nil {
				return resolved[K, E]{}, err
			}
			return resolved[K, E]{keys: keyset.Hashed(keys)}, nil
		},
		eval: func(_ Context, e *E, _, _ []byte) (bool, error) {
			return slices.Contains(captured, extract(e)), nil
		},
	}
}

// LookupIndex is an in-memory exact-match secondary index on a field of type V
// extracted from entries of type E. Construct via NewLookupIndex and register on a
// Table through TableConfig.Indexes.
type LookupIndex[K Key, E Entry[K], V comparable] struct {
	// name is the human-readable index name, used in diagnostics.
	name string
	// extract reads the indexed field from an entry.
	extract func(e *E) V
	store   *index.Lookup[*txState, K, V]
}

// NewLookupIndex constructs a LookupIndex over a comparable primary key K. The returned
// index is empty; register it on a Table through TableConfig.Indexes to populate it
// from the existing table contents and keep it in sync with future writes.
func NewLookupIndex[K Key, E Entry[K], V comparable](
	name string,
	extract func(e *E) V,
) *LookupIndex[K, E, V] {
	return &LookupIndex[K, E, V]{
		name:    name,
		extract: extract,
		store:   index.NewLookup[*txState, K, V](),
	}
}

// Name implements Index.
func (l *LookupIndex[K, E, V]) Name() string { return l.name }

//nolint:unused
func (l *LookupIndex[K, E, V]) populate() (func(E), func(error)) {
	insert, finish := l.store.Populate()
	return func(entry E) { insert(entry.GorpKey(), l.extract(&entry)) },
		func(err error) { finish(populateErr(l.name, err)) }
}

//nolint:unused
func (l *LookupIndex[K, E, V]) set(entry E) {
	l.store.Set(entry.GorpKey(), l.extract(&entry))
}

//nolint:unused
func (l *LookupIndex[K, E, V]) delete(key K) { l.store.Delete(key) }

//nolint:unused
func (l *LookupIndex[K, E, V]) stageSet(tx Tx, entry E) {
	l.store.Stage(tx.txIdentity(), entry.GorpKey(), l.extract(&entry))
}

//nolint:unused
func (l *LookupIndex[K, E, V]) stageDelete(tx Tx, key K) {
	l.store.StageDelete(tx.txIdentity(), key)
}

// Get returns the primary keys of entries whose indexed field matches any of the
// provided values, merging committed index state with any per-tx delta staged against
// tx. Pass a nil tx to read committed state only.
//
// Get blocks until the index has finished populating. After populate settles, Get
// returns ErrIndexInvalid if population failed; callers should fall back to a
// sequential scan or surface the error.
//
// Returned slices are owned by the caller and may be freely retained.
func (l *LookupIndex[K, E, V]) Get(tx Tx, values ...V) ([]K, error) {
	return l.store.Get(scopeOf(tx), values...)
}

// Filter returns a Filter[K, E] matching entries whose indexed field is any of values.
// The filter sees read-your-own-writes: an indexed Retrieve inside a write tx that
// created, updated, or deleted an entry observes those pending changes alongside
// committed index state. A Retrieve against a DB used directly returns committed state
// only.
//
// Filter blocks the Retrieve until the index has finished populating. If population
// fails, the resolver returns ErrIndexInvalid; Retrieve catches the sentinel and falls
// back to a sequential scan using the eval predicate carried alongside the resolver, so
// queries continue to return correct results at scan cost.
func (l *LookupIndex[K, E, V]) Filter(values ...V) Filter[K, E] {
	return indexFilter[K, E](l.store, l.extract, values)
}
