// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package gorp exposes a simple, type-safe ORM that wraps a key-value store.
package gorp

import (
	"bytes"
	"context"
	"maps"
	"slices"

	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/types"
)

// rawFilter is a predicate that operates on the raw encoded bytes of an entry
// before it is decoded. The filter receives both the pebble key and the
// encoded value, so callers can filter on key-shaped data without ever
// touching the value (or vice versa). Returning false skips the entry without
// allocating a decoded value. Returning true allows normal decode + filter
// processing.
type rawFilter func(key, value []byte) (bool, error)

// Validator is a batch check that runs on the final result set after Exec
// populates entries. A non-nil error from any validator causes Exec to return
// that error. Validators cannot filter results — use Where for that.
type Validator[K Key, E Entry[K]] func(ctx Context, entries []E) error

// Retrieve is a query that retrieves Entries from the DB.
type Retrieve[K Key, E Entry[K]] struct {
	// entries is the destination binding for matched entries.
	entries Entries[K, E]
	// limit caps the number of returned entries; 0 means unbounded.
	limit int
	// prefix narrows the underlying scan to keys starting with this
	// byte prefix; nil scans the whole table.
	prefix []byte
	// filter is the composed predicate applied to each candidate entry.
	filter Filter[K, E]
	// orderBy drives ordered iteration via a SortedIndex. Nil means
	// keyspace order.
	orderBy OrderQuery[K, E]
	// after starts a keyspace-order query strictly past this key. Nil starts at the
	// first key.
	after *K
	// validators run after entries are collected; a non-nil return
	// aborts Exec with that error.
	validators []Validator[K, E]
	// keyPrefix is the gorp key prefix for entry type E.
	keyPrefix []byte
	// missingKeysSkipped is true when Exec treats a key with no entry as a non-match.
	missingKeysSkipped bool
}

// NewRetrieve opens a new Retrieve query.
func NewRetrieve[K Key, E Entry[K]]() Retrieve[K, E] {
	return Retrieve[K, E]{}
}

// GetEntries returns the entries binding for the query.
func (r *Retrieve[K, E]) GetEntries() *Entries[K, E] { return &r.entries }

// Where adds the provided filter to the query, ANDing it with any
// existing filter. To restrict by primary key, compose MatchKeys into
// the filter (e.g. r.Where(MatchKeys(1, 2, 3))). To compose with an
// indexed filter, pass it through the same Where call:
// r.Where(idx.Filter(value)).
func (r Retrieve[K, E]) Where(filter Filter[K, E]) Retrieve[K, E] {
	if r.filter.present() {
		filter = And(r.filter, filter)
	}
	r.filter = filter
	return r
}

// HasLimit returns true if a limit was set on the query.
func (r Retrieve[K, E]) HasLimit() bool { return r.limit > 0 }

// IsBareKeys returns true if the filter is exactly Where(MatchKeys(...)), with no eval,
// raw, or resolver filter layered on top.
func (r Retrieve[K, E]) IsBareKeys() bool {
	return r.filter.keys.Bounded() &&
		r.filter.eval == nil &&
		r.filter.raw == nil &&
		r.filter.resolve == nil
}

// HasPrefix returns true if WherePrefix narrowed the query.
func (r Retrieve[K, E]) HasPrefix() bool { return r.prefix != nil }

// HasFilterKeys returns true if the resolved filter is bounded by a primary
// key set — either set directly via Where(MatchKeys(...)) or carried by an
// indexed filter (LookupIndex.Filter / SortedIndex.Filter) whose keys have
// already been resolved. Note that for filters carrying a deferred resolver,
// this returns false until resolveFilter has populated keys.
func (r Retrieve[K, E]) HasFilterKeys() bool { return r.filter.keys.Bounded() }

// GetFilterKeys returns the resolved filter's primary key set, or nil if the
// query is not bounded by keys.
func (r Retrieve[K, E]) GetFilterKeys() []K { return r.filter.keys.Keys() }

// WherePrefix filters entries whose key starts with the given prefix.
func (r Retrieve[K, E]) WherePrefix(prefix []byte) Retrieve[K, E] {
	r.prefix = prefix
	return r
}

// Limit sets the maximum number of results that the query will return, discarding
// any results beyond the limit.
func (r Retrieve[K, E]) Limit(limit int) Retrieve[K, E] {
	r.limit = limit
	return r
}

// SkipMissingKeys makes a key with no entry a non-match, as for every other filter,
// instead of a query.ErrNotFound from Exec. Exists then reports whether any key has an
// entry.
func (r Retrieve[K, E]) SkipMissingKeys() Retrieve[K, E] {
	r.missingKeysSkipped = true
	return r
}

// WhereRaw adds a raw byte filter that runs against each entry's pebble key
// and encoded value before decoding. Returning false skips the entry without
// allocating a decoded value, so a key-shaped predicate can drop most rows
// without paying decode cost. Use in tandem with WherePrefix when the
// keyspace itself can be narrowed.
func (r Retrieve[K, E]) WhereRaw(filter rawFilter) Retrieve[K, E] {
	return r.Where(MatchRaw[K, E](filter))
}

// After starts the query strictly past key in keyspace order, so a page can resume
// after the last key of the previous one. The key need not have an entry. After panics
// on a query with an OrderBy, since an OrderQuery carries its own cursor.
func (r Retrieve[K, E]) After(key K) Retrieve[K, E] {
	if r.hasOrderBy() {
		panic(errCursorWithOrder)
	}
	r.after = &key
	return r
}

const errCursorWithOrder = "gorp: After cannot combine with OrderBy"

// OrderBy walks the results in the order defined by the given OrderQuery,
// typically obtained from SortedIndex.Ordered(dir) (optionally with .After(value, key)
// chained for cursor-based pagination). Combine with Limit for paged walks. OrderBy
// panics on a query with an After.
func (r Retrieve[K, E]) OrderBy(o OrderQuery[K, E]) Retrieve[K, E] {
	if r.after != nil {
		panic(errCursorWithOrder)
	}
	r.orderBy = o
	return r
}

// hasOrderBy returns true if OrderBy was set on the query.
func (r Retrieve[K, E]) hasOrderBy() bool { return r.orderBy != nil }

// Validate attaches a batch validator that runs once on the final bound
// result set after Exec populates entries. A non-nil error from any validator
// causes Exec to return that error. Multiple Validate calls accumulate and
// run in the order they were attached; the first error wins. Validators
// cannot filter results — use Where for filtering.
func (r Retrieve[K, E]) Validate(f Validator[K, E]) Retrieve[K, E] {
	r.validators = append(r.validators, f)
	return r
}

func (r Retrieve[K, E]) runValidators(ctx Context, entries []E) error {
	for _, v := range r.validators {
		if v == nil {
			continue
		}
		if err := v(ctx, entries); err != nil {
			return err
		}
	}
	return nil
}

// Entries binds a slice that the Params will fill results into. Repeated calls to Entry
// or Entries will override all previous calls to Entries or Entry.
func (r Retrieve[K, E]) Entries(entries *[]E) Retrieve[K, E] {
	r.entries.bindMultiple(entries)
	return r
}

// Entry binds the entry that the Params will fill results into. Repeated calls to Entry
// or Entries will override all previous calls to Entries or Entry. If isMultiple
// results are returned by the query, entry will be set to the last result.
func (r Retrieve[K, E]) Entry(entry *E) Retrieve[K, E] {
	r.entries.bindSingle(entry)
	return r
}

// keysRequired reports whether every requested key must have an entry: the filter is
// exactly Where(MatchKeys(...)), and SkipMissingKeys is not set.
func (r Retrieve[K, E]) keysRequired() bool {
	return r.IsBareKeys() && !r.missingKeysSkipped
}

// Exec executes the query against the provided snapshot. Through a Tx, it sees the
// writes of the transaction.
//
// When the filter is exactly Where(MatchKeys(...)) and SkipMissingKeys is not set, Exec
// returns query.ErrNotFound, joined with any other error, if a requested key has no
// entry. Otherwise a missing key is a non-match. A query bound to a single entry with
// no result returns query.ErrNotFound for any filter shape.
func (r Retrieve[K, E]) Exec(ctx context.Context, snap Snapshot) error {
	checkForNilTx("Retriever.Exec", snap)
	if err := r.resolveFilter(ctx, snap); err != nil {
		return err
	}
	if r.hasOrderBy() {
		return r.execOrdered(ctx, snap)
	}
	if r.HasFilterKeys() {
		notFound, err := r.execKeys(ctx, snap)
		if r.keysRequired() && len(notFound) > 0 {
			return errors.Join(err, errors.Wrapf(
				query.ErrNotFound,
				"%s with keys %v not found",
				types.PluralName[E](),
				notFound,
			))
		}
		if err != nil {
			return err
		}
		if r.entries.Bound() && !r.entries.isMultiple && r.entries.changes == 0 {
			return errors.Wrapf(
				query.ErrNotFound,
				"no %s found matching query",
				types.PluralName[E](),
			)
		}
		return nil
	}
	return r.execFilter(ctx, snap)
}

// resolveFilter materializes the filter's deferred resolver against snap, merging
// committed index state with any per-tx staged mutations. No-op when the filter has no
// resolver.
func (r *Retrieve[K, E]) resolveFilter(ctx context.Context, snap Snapshot) error {
	if r.filter.resolve == nil {
		return nil
	}
	res, err := r.filter.resolve(ctx, snap)
	if err != nil {
		return err
	}
	r.filter.keys = res.keys
	// Or / Not return a per-Exec eval closed over freshly materialized
	// children; installing it on the per-call r.filter avoids sharing
	// mutable child state across concurrent Execs.
	if res.eval != nil {
		r.filter.eval = res.eval
	}
	return nil
}

// Exists returns true if any record passes the filter. When every requested key must
// have an entry (see SkipMissingKeys), it returns true only if every key exists.
func (r Retrieve[K, E]) Exists(ctx context.Context, snap Snapshot) (bool, error) {
	checkForNilTx("Retriever.Exists", snap)
	if err := r.resolveFilter(ctx, snap); err != nil {
		return false, err
	}
	if r.HasFilterKeys() {
		keys := r.filter.keys.Keys()
		if len(keys) == 0 {
			return false, nil
		}
		e := make([]E, 0, len(keys))
		r.entries.bindMultiple(&e)
		if _, err := r.execKeys(ctx, snap); errors.Skip(err, query.ErrNotFound) != nil {
			return false, err
		}
		if r.keysRequired() {
			return len(e) == len(keys), nil
		}
		return len(e) > 0, nil
	}
	e := make([]E, 0, 1)
	r.entries.bindMultiple(&e)
	r.limit = 1
	if err := r.execFilter(ctx, snap); errors.Skip(err, query.ErrNotFound) != nil {
		return false, err
	}
	return len(e) > 0, nil
}

// Count returns the number of records matching the query.
func (r Retrieve[K, E]) Count(
	ctx context.Context,
	snap Snapshot,
) (count int, err error) {
	checkForNilTx("Retriever.Count", snap)
	if err := r.resolveFilter(ctx, snap); err != nil {
		return 0, err
	}
	if r.HasFilterKeys() {
		r.entries.bindMultiple(new(make([]E, 0, len(r.filter.keys.Keys()))))
		if _, err := r.execKeys(
			ctx,
			snap,
		); err != nil &&
			!errors.Is(err, query.ErrNotFound) {
			return 0, err
		}
		return len(r.entries.All()), nil
	}

	reader := wrapReader[K, E](snap, r.keyPrefix)
	iter, err := reader.OpenIterator(
		IterOptions{prefix: r.prefix, after: r.encodedAfter()},
	)
	if err != nil {
		return 0, err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	gorpCtx := newContext(ctx, snap)
	var matched []E
	if len(r.validators) > 0 {
		matched = make([]E, 0)
	}
	for iter.First(); iter.Valid(); iter.Next() {
		rawKey := iter.Key()
		rawValue := iter.Iterator.Value()
		rawMatched, rErr := r.matchRaw(rawKey, rawValue)
		if rErr != nil {
			return 0, rErr
		}
		if !rawMatched {
			continue
		}
		v := iter.Value(ctx)
		if err = iter.Error(); err != nil {
			return 0, err
		}
		match, fErr := r.match(gorpCtx, v, rawKey, rawValue)
		if fErr != nil {
			return 0, fErr
		}
		if match {
			count++
			if matched != nil {
				matched = append(matched, *v)
			}
		}
	}
	if matched != nil {
		if vErr := r.runValidators(gorpCtx, matched); vErr != nil {
			return 0, vErr
		}
	}
	return count, err
}

// match runs the filter's predicates against a decoded entry. Returns
// true if no filter is set. key may be nil when the caller does not
// have the raw key bytes available.
func (r Retrieve[K, E]) match(ctx Context, e *E, key, value []byte) (bool, error) {
	if !r.filter.keys.Contains((*e).GorpKey()) {
		return false, nil
	}
	return r.matchEval(ctx, e, key, value)
}

// matchEval is match for an entry that the filter's keys contain.
func (r Retrieve[K, E]) matchEval(ctx Context, e *E, key, value []byte) (bool, error) {
	if r.filter.eval == nil {
		return true, nil
	}
	return r.filter.eval(ctx, e, key, value)
}

// matchRaw runs the filter's raw-byte pre-screen against key/value.
// Returns true (don't skip) when no raw filter is set.
func (r Retrieve[K, E]) matchRaw(key, value []byte) (bool, error) {
	if r.filter.raw == nil {
		return true, nil
	}
	return r.filter.raw(key, value)
}

// execKeys fetches and matches each candidate key in turn. Returns the
// keys that were not found in the underlying KV store; the caller
// decides whether to surface that as query.ErrNotFound. Unless the filter
// is bare keys, execKeys stops at the limit, so it reports only the keys
// it fetched.
func (r *Retrieve[K, E]) execKeys(ctx context.Context, snap Snapshot) ([]K, error) {
	var (
		keys       = r.filter.keys.Keys()
		reader     = wrapReader[K, E](snap, r.keyPrefix)
		notFound   []K
		validCount int
		gorpCtx    = newContext(ctx, snap)
		e          E
		zero       E
		capacity   = len(keys)
		stops      = r.limit > 0 && !r.keysRequired()
	)
	if r.limit > 0 {
		capacity = min(capacity, r.limit)
	}
	// An index returns its keys in no fixed order, so a page follows storage order.
	if r.after != nil {
		keys = storageOrder(keys, r.after)
	} else if r.limit > 0 && !r.IsBareKeys() && !r.filter.keys.Ranked() {
		keys = storageOrder(keys, nil)
	}
	r.entries.ensureCap(capacity)
	for _, k := range keys {
		if stops && validCount == r.limit {
			break
		}
		e = zero
		b, closer, err := reader.get(ctx, k, &e)
		if err != nil {
			if errors.Is(err, query.ErrNotFound) {
				notFound = append(notFound, k)
				continue
			}
			return nil, err
		}
		if reader.keyCodec.HasPrefix(e.GorpKey(), r.prefix) {
			match, mErr := r.matchEval(gorpCtx, &e, nil, b)
			if mErr != nil {
				return nil, errors.Combine(mErr, closer.Close())
			}
			if match {
				validCount++
				if r.limit == 0 || validCount <= r.limit {
					r.entries.add(e)
				}
			}
		}
		if cErr := closer.Close(); cErr != nil {
			return nil, cErr
		}
	}
	if len(r.validators) == 0 {
		return notFound, nil
	}
	return notFound, r.runValidators(gorpCtx, r.entries.All())
}

// encodedAfter returns the encoded form of the keyspace cursor, or nil when the query
// has none.
func (r Retrieve[K, E]) encodedAfter() []byte {
	if r.after == nil {
		return nil
	}
	codec := key.NewCodec[K](nil)
	return bytes.Clone(codec.Encode(*r.after))
}

// storageOrder returns a copy of keys in the order a table scan visits them. A non-nil
// after drops the keys that a scan visits at or before it.
func storageOrder[K Key](keys []K, after *K) []K {
	type encoded struct {
		key        K
		start, end int
	}
	var (
		codec = key.NewCodec[K](nil)
		raw   []byte
		pairs = make([]encoded, len(keys))
	)
	for i, k := range keys {
		start := len(raw)
		raw = append(raw, codec.Encode(k)...)
		pairs[i] = encoded{key: k, start: start, end: len(raw)}
	}
	slices.SortFunc(pairs, func(a, b encoded) int {
		return bytes.Compare(raw[a.start:a.end], raw[b.start:b.end])
	})
	sorted := make([]K, 0, len(pairs))
	var bound []byte
	if after != nil {
		bound = bytes.Clone(codec.Encode(*after))
	}
	for _, p := range pairs {
		if bound == nil || bytes.Compare(raw[p.start:p.end], bound) > 0 {
			sorted = append(sorted, p.key)
		}
	}
	return sorted
}

func (r *Retrieve[K, E]) execFilter(ctx context.Context, snap Snapshot) (err error) {
	var (
		validCount int
		match      bool
		gorpCtx    = newContext(ctx, snap)
	)
	reader := wrapReader[K, E](snap, r.keyPrefix)
	iter, err := reader.OpenIterator(
		IterOptions{prefix: r.prefix, after: r.encodedAfter()},
	)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		rawKey := iter.Key()
		rawValue := iter.Iterator.Value()
		rawMatched, rErr := r.matchRaw(rawKey, rawValue)
		if rErr != nil {
			return rErr
		}
		if !rawMatched {
			continue
		}
		v := iter.Value(ctx)
		if err = iter.Error(); err != nil {
			return err
		}
		match, err = r.match(gorpCtx, v, rawKey, rawValue)
		if err != nil {
			return err
		}
		if match {
			validCount += 1
			r.entries.add(*v)
			if validCount == r.limit {
				break
			}
		}
	}
	if vErr := r.runValidators(gorpCtx, r.entries.All()); vErr != nil {
		return vErr
	}
	if r.entries.isMultiple || !r.entries.Bound() {
		return err
	}
	if r.entries.changes == 0 {
		return errors.Wrapf(
			query.ErrNotFound,
			"no %s found matching query",
			types.PluralName[E](),
		)
	}
	return err
}

// minWalkBatch is the fewest keys an ordered walk takes from its index at a time, so a
// filter that rejects most entries does not cost one index lock per entry.
const minWalkBatch = 128

// execOrdered walks the configured OrderQuery, fetching entries per-key in walk order
// and matching each against the filter until it collects the limit.
func (r *Retrieve[K, E]) execOrdered(ctx context.Context, snap Snapshot) error {
	var (
		reader  = wrapReader[K, E](snap, r.keyPrefix)
		matched = make([]E, 0, min(r.limit, minWalkBatch))
		counted = set.New[K]()
		moved   = make(map[K]E)
		gorpCtx = newContext(ctx, snap)
		batch   int
		bounded []K
	)
	if r.HasFilterKeys() {
		bounded = r.filter.keys.Keys()
	}
	if r.limit > 0 {
		batch = max(r.limit, minWalkBatch)
	}
	// A walk with no limit keeps every match and sorts them into place at the end.
	snapshot := batch == 0
	for k, holds := range r.orderBy.walk(bounded, batch) {
		// A commit between two batches can move a counted key ahead of the walk.
		if counted.Contains(k) {
			continue
		}
		var e E
		b, closer, err := reader.get(ctx, k, &e)
		if err != nil {
			if errors.Is(err, query.ErrNotFound) {
				continue
			}
			return err
		}
		match, mErr := r.matchEval(gorpCtx, &e, nil, b)
		if cErr := closer.Close(); mErr != nil || cErr != nil {
			return errors.Combine(mErr, cErr)
		}
		if !match {
			continue
		}
		if snapshot {
			matched = append(matched, e)
			continue
		}
		// A commit between the walk and the read can move the entry, so it is sorted
		// into place at the end by the value it holds now.
		if !holds(&e) {
			moved[k] = e
			continue
		}
		counted.Add(k)
		delete(moved, k)
		matched = append(matched, e)
		if r.limit > 0 && len(matched) == r.limit {
			break
		}
	}
	if snapshot || len(moved) > 0 {
		matched = r.orderBy.sort(append(matched, slices.Collect(maps.Values(moved))...))
	}
	if r.limit > 0 {
		matched = matched[:min(r.limit, len(matched))]
	}
	r.entries.replace(matched)
	if vErr := r.runValidators(gorpCtx, r.entries.All()); vErr != nil {
		return vErr
	}
	if r.entries.isMultiple || !r.entries.Bound() || len(matched) > 0 {
		return nil
	}
	return errors.Wrapf(
		query.ErrNotFound,
		"no %s found matching query",
		types.PluralName[E](),
	)
}
