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
	"context"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/types"
)

// RawFilter is a predicate that operates on the raw encoded bytes of an entry
// before it is decoded. Returning false skips the entry without allocating a
// decoded value. Returning true allows normal decode + filter processing.
type RawFilter func(data []byte) (bool, error)

// Validator is a batch check that runs on the final bound result set after
// Exec populates entries. Returning a non-nil error causes Exec to return that
// error. Validators are attached via Retrieve.Validate and cannot filter
// results — if you need to filter, use Where.
type Validator[K Key, E Entry[K]] func(ctx Context, entries []E) error

// Retrieve is a query that retrieves Entries from the DB.
type Retrieve[K Key, E Entry[K]] struct {
	entries    *Entries[K, E]
	limit      int
	offset     int
	keys       *[]K
	prefix     []byte
	filter     Filter[K, E]
	hasFilter  bool
	orderBy    OrderQuery[K, E]
	validators []Validator[K, E]
}

// NewRetrieve opens a new Retrieve query.
func NewRetrieve[K Key, E Entry[K]]() Retrieve[K, E] {
	return Retrieve[K, E]{entries: new(Entries[K, E])}
}

// GetEntries returns the entries bound to the query.
func (r Retrieve[K, E]) GetEntries() *Entries[K, E] { return r.entries }

// Where adds the provided filters to the query, ANDing them with any existing
// filter. If filtering by the key of the Entry, use the far more efficient
// WhereKeys method instead.
//
// Fast path: when there's exactly one filter and no existing one, the filter
// is stored directly without going through And. This avoids allocating the
// And combinator's Eval closure and intersectKeys output, which is the
// dominant query path for index-backed lookups (most callers do
// Where(MatchUsernames("foo")) — one filter, no prior).
func (r Retrieve[K, E]) Where(filters ...Filter[K, E]) Retrieve[K, E] {
	if !r.hasFilter && len(filters) == 1 {
		r.filter = filters[0]
		r.hasFilter = true
		return r
	}
	combined := And(filters...)
	if r.hasFilter {
		combined = And(r.filter, combined)
	}
	r.filter = combined
	r.hasFilter = true
	return r
}

// HasLimit returns true if a limit was set on the query.
func (r Retrieve[K, E]) HasLimit() bool { return r.limit > 0 }

// HasOffset returns true if an offset was set on the query.
func (r Retrieve[K, E]) HasOffset() bool { return r.offset > 0 }

// HasWhereKeys returns true if WhereKeys was called on the query.
func (r Retrieve[K, E]) HasWhereKeys() bool { return r.keys != nil }

// GetWhereKeys returns the keys set by WhereKeys, or nil if not set.
func (r Retrieve[K, E]) GetWhereKeys() []K {
	if r.keys != nil {
		return *r.keys
	}
	return nil
}

// HasFilters returns true if any Where filters were added to the query.
func (r Retrieve[K, E]) HasFilters() bool { return r.hasFilter }

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

// Offset sets the number of results that the query will skip before returning results.
func (r Retrieve[K, E]) Offset(offset int) Retrieve[K, E] {
	r.offset = offset
	return r
}

// WhereRaw adds a raw byte filter that is evaluated before decoding each entry.
// Entries whose raw bytes cause the filter to return false are skipped without
// being decoded.
func (r Retrieve[K, E]) WhereRaw(filter RawFilter) Retrieve[K, E] {
	return r.Where(MatchRaw[K, E](filter))
}

// OrderBy walks the results in the order defined by the given OrderQuery,
// typically obtained from Sorted.Ordered(dir) (optionally with .After(cursor)
// chained for cursor-based pagination). Combine with Limit for paged walks.
//
// The cursor lives inside the SortedQuery handle that satisfies OrderQuery,
// captured at construction time when V is statically known. Retrieve stores
// the V-erased OrderQuery interface; the typed cursor never crosses the
// boundary as a value, so there's no `any` boxing on the pagination path.
func (r Retrieve[K, E]) OrderBy(o OrderQuery[K, E]) Retrieve[K, E] {
	r.orderBy = o
	return r
}

// HasOrderBy returns true if OrderBy was set on the query.
func (r Retrieve[K, E]) HasOrderBy() bool { return r.orderBy != nil }

// Validate attaches a batch validator that runs once on the final bound
// result set after Exec populates entries. A non-nil error from any validator
// causes Exec to return that error. Multiple Validate calls accumulate and
// run in the order they were attached; the first error wins. Validators
// cannot filter results — use Where for filtering.
func (r Retrieve[K, E]) Validate(f Validator[K, E]) Retrieve[K, E] {
	r.validators = append(r.validators, f)
	return r
}

// runValidators runs every attached validator against the provided entry
// snapshot, returning the first non-nil error.
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

// WhereKeys queries the DB for Entries with the provided keys. Although more targeted,
// this lookup is substantially faster than a general Where query. If called in
// conjunction with Where, the WhereKeys filter will be applied first. Subsequent calls
// to WhereKeys will append the keys to the existing set.
func (r Retrieve[K, E]) WhereKeys(keys ...K) Retrieve[K, E] {
	if r.keys == nil {
		r.keys = new([]K)
	}
	*r.keys = append(*r.keys, keys...)
	return r
}

// Entries binds a slice that the Params will fill results into. Repeated calls to Entry
// or Entries will override all previous calls to Entries or Entry.
func (r Retrieve[K, E]) Entries(entries *[]E) Retrieve[K, E] {
	r.entries = multipleEntries(entries)
	return r
}

// Entry binds the entry that the Params will fill results into. Repeated calls to Entry
// or Entries will override All previous calls to Entries or Entry. If  isMultiple results
// are returned by the query, entry will be set to the last result.
func (r Retrieve[K, E]) Entry(entry *E) Retrieve[K, E] {
	r.entries = singleEntry(entry)
	return r
}

// Exec executes the Params against the provided Writer. If the WhereKeys method is set on
// the query, Retrieve will return a query.ErrNotFound  error if ANY of the keys do not
// exist in the database. If Where is set on the query, Retrieve will return a query.ErrNotFound
// if NO keys pass the Where filter.
//
// Dispatch order:
//   - OrderBy   → execOrdered
//   - WhereKeys OR an indexed filter (filter.Keys != nil) → execKeys (fast path)
//   - otherwise → execFilter (full table scan)
func (r Retrieve[K, E]) Exec(ctx context.Context, tx Tx) error {
	checkForNilTx("Retriever.Exec", tx)
	if r.HasOrderBy() {
		return r.execOrdered(ctx, tx)
	}
	if r.HasWhereKeys() || r.hasIndexedFilter() {
		return r.execKeys(ctx, tx)
	}
	return r.execFilter(ctx, tx)
}

// hasIndexedFilter reports whether the active filter chain carries a
// precomputed key set, which is the signal that Retrieve can short-circuit
// into the execKeys fast path.
func (r Retrieve[K, E]) hasIndexedFilter() bool {
	return r.hasFilter && r.filter.Keys != nil
}

// Exists checks whether records matching the query exist in the DB. If the WhereKeys method is
// set on the query, Exists will return true if ANY of the keys exist in the database. If
// Where is set on the query, Exists will return true if ANY keys pass the Where filter.
func (r Retrieve[K, E]) Exists(ctx context.Context, tx Tx) (bool, error) {
	if r.HasWhereKeys() {
		e := make([]E, 0, len(*r.keys))
		r.entries = multipleEntries(&e)
		if err := r.execKeys(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
			return false, err
		}
		return len(e) == len(*r.keys), nil
	}
	e := make([]E, 0, 1)
	r.entries = multipleEntries(&e)
	if err := r.execFilter(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return false, err
	}
	return len(e) > 0, nil
}

// Count returns the number of records matching the query. If the WhereKeys method is
// set on the query, Count will return the number of existing keys. If Where is set
// on the query, Count will return the number of records that pass the Where filter.
func (r Retrieve[K, E]) Count(ctx context.Context, tx Tx) (count int, err error) {
	checkForNilTx("Retriever.Count", tx)
	if r.HasWhereKeys() {
		e := make([]E, 0, len(*r.keys))
		r.entries = multipleEntries(&e)
		if err := r.execKeys(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
			return 0, err
		}
		return len(r.entries.All()), nil
	}

	reader := WrapReader[K, E](tx)
	iter, err := reader.OpenIterator(IterOptions{prefix: r.prefix})
	if err != nil {
		return 0, err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		rawMatched, rErr := r.matchRaw(iter.Iterator.Value())
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
		match, fErr := r.match(Context{Context: ctx, Tx: tx}, v)
		if fErr != nil {
			return 0, fErr
		}
		if match {
			count++
		}
	}
	return count, err
}

func (r Retrieve[K, E]) match(ctx Context, e *E) (bool, error) {
	if !r.hasFilter || r.filter.Eval == nil {
		return true, nil
	}
	return r.filter.Eval(ctx, e)
}

func (r Retrieve[K, E]) matchRaw(data []byte) (bool, error) {
	if !r.hasFilter || r.filter.Raw == nil {
		return true, nil
	}
	return r.filter.Raw(data)
}

// effectiveKeys returns the candidate key set for execKeys: the intersection
// of any keys set via WhereKeys and any keys carried by an indexed filter via
// filter.Keys. Either source may be nil, but at least one must be present
// (callers gate via HasWhereKeys || hasIndexedFilter before invoking).
func (r Retrieve[K, E]) effectiveKeys() []K {
	hasIndexedKeys := r.hasFilter && r.filter.Keys != nil
	switch {
	case r.keys == nil && !hasIndexedKeys:
		return nil
	case r.keys != nil && !hasIndexedKeys:
		return *r.keys
	case r.keys == nil && hasIndexedKeys:
		return r.filter.Keys
	}
	return intersectKeySlices(*r.keys, r.filter.Keys)
}

// intersectKeySlices returns the intersection of two key slices via linear
// scan. Used at Exec dispatch to merge WhereKeys with an indexed filter's
// Keys. Inputs are typically small (single-digit element counts) so the
// O(n*m) cost is fine.
func intersectKeySlices[K Key](a, b []K) []K {
	if len(a) == 0 || len(b) == 0 {
		return []K{}
	}
	out := make([]K, 0, min(len(a), len(b)))
	for _, k := range a {
		if containsKey(b, k) {
			out = append(out, k)
		}
	}
	return out
}

func (r Retrieve[K, E]) execKeys(ctx context.Context, tx Tx) error {
	keys := r.effectiveKeys()
	var (
		reader             = WrapReader[K, E](tx)
		keysResult, getErr = reader.GetMany(ctx, keys)
		validCount         int
		gorpCtx            = Context{Context: ctx, Tx: tx}
	)
	// We don't return early even if getErr fails with a not found result in
	// order to do a best effort retrieval of available items. WhereKeys
	// callers expect query.ErrNotFound when ANY requested key was missing;
	// indexed-filter-only callers don't, but we still surface the error.
	if getErr != nil && !errors.Is(getErr, query.ErrNotFound) {
		return getErr
	}
	// Filter in place by reusing keysResult's backing array. The kept-write
	// index never overtakes the read index, so this is safe.
	filtered := keysResult[:0]
	for _, e := range keysResult {
		if !reader.keyCodec.matchPrefix(r.prefix, e.GorpKey()) {
			continue
		}
		match, err := r.match(gorpCtx, &e)
		if err != nil {
			return err
		}
		if match {
			validCount += 1
			if (validCount > r.offset) && (r.limit == 0 || validCount <= r.limit+r.offset) {
				filtered = append(filtered, e)
			}
		}
	}
	r.entries.Replace(filtered)
	if err := r.runValidators(gorpCtx, filtered); err != nil {
		return err
	}
	// Suppress not-found from indexed-filter-only queries: an indexed filter
	// matching no keys is an empty result, not an error. Only propagate
	// not-found when WhereKeys was explicitly set.
	if !r.HasWhereKeys() {
		return nil
	}
	return getErr
}

func (r Retrieve[K, E]) execFilter(ctx context.Context, tx Tx) error {
	var (
		validCount int
		match      bool
		reader     = WrapReader[K, E](tx)
		gorpCtx    = Context{Context: ctx, Tx: tx}
	)
	iter, err := reader.OpenIterator(IterOptions{prefix: r.prefix})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		rawMatched, rErr := r.matchRaw(iter.Iterator.Value())
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
		match, err = r.match(gorpCtx, v)
		if err != nil {
			return err
		}
		if match {
			validCount += 1
			if (validCount > r.offset) && (r.limit == 0 || validCount <= r.limit+r.offset) {
				r.entries.Add(*v)
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

// execOrdered walks a Sorted index via the configured OrderQuery handle,
// fetching entries from the KV store in page-sized batches. Any Where
// filters are applied as post-filters after each batch decode.
//
// The walk is driven by the typed SortedQuery (which satisfies OrderQuery)
// captured at OrderBy time. The cursor lives inside that handle, so this
// path involves no `any` boxing.
func (r Retrieve[K, E]) execOrdered(ctx context.Context, tx Tx) error {
	if r.orderBy == nil {
		return nil
	}
	keys := r.orderBy.walkOrder(r.limit)
	if len(keys) == 0 {
		if r.entries.isMultiple {
			r.entries.Replace(nil)
			return nil
		}
		if !r.entries.Bound() {
			return nil
		}
		return errors.Wrapf(
			query.ErrNotFound,
			"no %s found matching query",
			types.PluralName[E](),
		)
	}
	reader := WrapReader[K, E](tx)
	entries, getErr := reader.GetMany(ctx, keys)
	if getErr != nil && !errors.Is(getErr, query.ErrNotFound) {
		return getErr
	}
	// GetMany preserves input key order (omitting any not-found keys), so
	// entries is already in sorted-walk order.
	filtered := make([]E, 0, len(entries))
	for _, e := range entries {
		match, err := r.match(Context{Context: ctx, Tx: tx}, &e)
		if err != nil {
			return err
		}
		if match {
			filtered = append(filtered, e)
		}
	}
	r.entries.Replace(filtered)
	if r.entries.isMultiple || !r.entries.Bound() {
		return nil
	}
	if len(filtered) == 0 {
		return errors.Wrapf(
			query.ErrNotFound,
			"no %s found matching query",
			types.PluralName[E](),
		)
	}
	return nil
}
