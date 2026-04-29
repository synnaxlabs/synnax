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
// before it is decoded. The filter receives both the pebble key and the
// encoded value, so callers can filter on key-shaped data without ever
// touching the value (or vice versa). Returning false skips the entry without
// allocating a decoded value. Returning true allows normal decode + filter
// processing.
type RawFilter func(key, value []byte) (bool, error)

// Validator is a batch check that runs on the final result set after Exec
// populates entries. A non-nil error from any validator causes Exec to return
// that error. Validators cannot filter results — use Where for that.
type Validator[K Key, E Entry[K]] func(ctx Context, entries []E) error

// Retrieve is a query that retrieves Entries from the DB.
type Retrieve[K Key, E Entry[K]] struct {
	entries    Entries[K, E]
	limit      int
	offset     int
	prefix     []byte
	filter     Filter[K, E]
	orderBy    OrderQuery[K, E]
	validators []Validator[K, E]
	keyPrefix  []byte
}

// NewRetrieve opens a new Retrieve query.
func NewRetrieve[K Key, E Entry[K]]() Retrieve[K, E] {
	return Retrieve[K, E]{}
}

// GetEntries returns the entries binding for the query.
func (r *Retrieve[K, E]) GetEntries() *Entries[K, E] { return &r.entries }

// Where adds the provided filter to the query, ANDing it with any existing
// filter. To restrict by primary key, compose MatchKeys into the filter
// (e.g. r.Where(MatchKeys(1, 2, 3))). To compose with an indexed filter,
// pass it through the same Where call: r.Where(idx.Filter(value)). And
// across the two paths intersects their candidate key sets via
// intersectKeys, so MatchKeys + indexed filter still routes through the
// execKeys fast path.
func (r Retrieve[K, E]) Where(filter Filter[K, E]) Retrieve[K, E] {
	if r.filter.present() {
		filter = And(r.filter, filter)
	}
	r.filter = filter
	return r
}

// HasLimit returns true if a limit was set on the query.
func (r Retrieve[K, E]) HasLimit() bool { return r.limit > 0 }

// HasOffset returns true if an offset was set on the query.
func (r Retrieve[K, E]) HasOffset() bool { return r.offset > 0 }

// HasFilters returns true if any Where filters were added to the query.
func (r Retrieve[K, E]) HasFilters() bool { return r.filter.present() }

// HasNonKeyFilters returns true if any Where filter that requires running
// the query (an eval or raw filter) was added. A query whose only filter
// is MatchKeys returns false — its key set already determines the result
// without consulting gorp, which lets routing layers (e.g. ontology
// traversal) skip the lookup entirely.
func (r Retrieve[K, E]) HasNonKeyFilters() bool {
	return r.filter.eval != nil || r.filter.raw != nil
}

// HasFilterKeys returns true if the resolved filter is bounded by a primary
// key set — either set directly via Where(MatchKeys(...)) or carried by an
// indexed filter (Lookup.Filter / Sorted.Filter / BytesLookup.Filter) whose
// keys have already been resolved. Note that for filters carrying a deferred
// resolver, this returns false until resolveFilter has populated keys.
func (r Retrieve[K, E]) HasFilterKeys() bool {
	return r.filter.keys != nil
}

// GetFilterKeys returns the resolved filter's primary key set, or nil if the
// query is not bounded by keys.
func (r Retrieve[K, E]) GetFilterKeys() []K {
	return r.filter.keys
}

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

// WhereRaw adds a raw byte filter that runs against each entry's pebble key
// and encoded value before decoding. Returning false skips the entry without
// allocating a decoded value, so a key-shaped predicate can drop most rows
// without paying decode cost. Use in tandem with WherePrefix when the
// keyspace itself can be narrowed.
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
// or Entries will override All previous calls to Entries or Entry. If isMultiple results
// are returned by the query, entry will be set to the last result.
func (r Retrieve[K, E]) Entry(entry *E) Retrieve[K, E] {
	r.entries.bindSingle(entry)
	return r
}

// isBareKeys reports whether the filter is exactly Where(MatchKeys(...)) with
// no eval or raw constraints layered on top. Used by Exec to decide whether
// missing keys should surface as query.ErrNotFound (bare keys) or just be
// silently omitted (any other shape, including indexed filters).
func (r Retrieve[K, E]) isBareKeys() bool {
	return r.filter.keys != nil &&
		r.filter.eval == nil &&
		r.filter.raw == nil &&
		r.filter.resolve == nil
}

// Exec executes the query against the provided transaction.
//
// Dispatch order:
//   - Resolve indexed filter (if deferred) → populate filter.keys
//   - OrderBy → execOrdered
//   - filter has keys (from MatchKeys or an indexed filter) → execKeys (fast path)
//   - otherwise → execFilter (full table scan)
//
// If the filter is bare keys (Where(MatchKeys(...)) only) and any requested
// key is missing, Exec returns query.ErrNotFound joined with any other error
// the execKeys path produced. Indexed filters or composed (keys + eval/raw)
// filters do not surface ErrNotFound for missing keys — a missing key is
// just an empty slot in the result set.
func (r Retrieve[K, E]) Exec(ctx context.Context, tx Tx) error {
	checkForNilTx("Retriever.Exec", tx)
	if err := r.resolveFilter(ctx, tx); err != nil {
		return err
	}
	if r.HasOrderBy() {
		return r.execOrdered(ctx, tx)
	}
	if r.HasFilterKeys() {
		notFound, err := r.execKeys(ctx, tx)
		// Any keys that were looked up but missing from the DB. For a
		// user-supplied MatchKeys filter this means "you asked for keys
		// that don't exist"; for an indexed filter (where keys come from
		// a resolver) it indicates index/DB drift. Either way, surface
		// as ErrNotFound joined with any validator/exec error so callers
		// can match on both. Mirrors rc's WhereKeys contract: any missing
		// key in the input set is an error regardless of single- or
		// multi-entry binding.
		if len(notFound) > 0 {
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
		// Indexed-filter path with an empty resolve: no candidate keys
		// were produced at all (e.g., name lookup on a deleted channel).
		// For single-entry bound queries this mirrors execFilter's
		// contract — surface as ErrNotFound so callers (e.g., the Arc
		// symbol resolver) see "not found" rather than a silent zero
		// entry. Multi-entry or unbound queries treat empty as a normal
		// empty result.
		if r.entries.Bound() && !r.entries.isMultiple && r.entries.changes == 0 {
			return errors.Wrapf(
				query.ErrNotFound,
				"no %s found matching query",
				types.PluralName[E](),
			)
		}
		return nil
	}
	return r.execFilter(ctx, tx)
}

// resolveFilter invokes the filter's deferred resolver, if any, and
// populates r.filter.keys + r.filter.membership with the merge of
// committed index state and per-tx staged mutations. No-op for bare
// Match/MatchRaw filters and for composed filters whose children are
// all eager. Called from Exec/Exists/Count before dispatch so the rest
// of the pipeline sees read-your-own-writes semantics through the
// normal keys/membership fields.
func (r *Retrieve[K, E]) resolveFilter(ctx context.Context, tx Tx) error {
	if r.filter.resolve == nil {
		return nil
	}
	keys, build, err := r.filter.resolve(ctx, tx)
	if err != nil {
		return err
	}
	r.filter.keys = keys
	if build != nil && keys != nil {
		r.filter.membership = newLazyMembership(keys, build)
	} else {
		r.filter.membership = nil
	}
	return nil
}

// Exists checks whether records matching the query exist in the DB. Exists
// returns true if any record passes the filter; for bare-keys queries this
// degenerates to "every requested key exists".
//
// Dispatch mirrors Exec: filters with resolved keys (from MatchKeys or an
// indexed filter) route through execKeys, otherwise execFilter.
func (r Retrieve[K, E]) Exists(ctx context.Context, tx Tx) (bool, error) {
	if err := r.resolveFilter(ctx, tx); err != nil {
		return false, err
	}
	if r.HasFilterKeys() {
		keys := r.filter.keys
		if len(keys) == 0 {
			return false, nil
		}
		e := make([]E, 0, len(keys))
		r.entries.bindMultiple(&e)
		if _, err := r.execKeys(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
			return false, err
		}
		if r.isBareKeys() {
			return len(e) == len(keys), nil
		}
		return len(e) > 0, nil
	}
	e := make([]E, 0, 1)
	r.entries.bindMultiple(&e)
	if err := r.execFilter(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return false, err
	}
	return len(e) > 0, nil
}

// Count returns the number of records matching the query. Indexed filters
// route through execKeys so the candidate set comes from the index rather
// than a full table scan.
func (r Retrieve[K, E]) Count(ctx context.Context, tx Tx) (count int, err error) {
	checkForNilTx("Retriever.Count", tx)
	if err := r.resolveFilter(ctx, tx); err != nil {
		return 0, err
	}
	if r.HasFilterKeys() {
		r.entries.bindMultiple(new(make([]E, 0, len(r.filter.keys))))
		if _, err := r.execKeys(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
			return 0, err
		}
		return len(r.entries.All()), nil
	}

	reader := wrapReader[K, E](tx, r.keyPrefix)
	iter, err := reader.OpenIterator(IterOptions{prefix: r.prefix})
	if err != nil {
		return 0, err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	gorpCtx := Context{Context: ctx, Tx: tx}
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

// match runs the filter's eval/keys-membership predicate against a decoded
// entry. Returns true if no filter is set. The keys-path passes nil for the
// rawKey since the keys are already known and key-shape pre-screening is
// moot; the iterator path passes the actual key bytes.
func (r Retrieve[K, E]) match(ctx Context, e *E, key, value []byte) (bool, error) {
	if !r.filter.present() {
		return true, nil
	}
	if r.filter.keys != nil && !r.filter.containsKey((*e).GorpKey()) {
		return false, nil
	}
	if r.filter.eval != nil {
		return r.filter.eval(ctx, e, key, value)
	}
	// No eval. raw was already enforced by matchRaw at the pre-decode
	// pre-screen on the iterator path; the keys path doesn't run matchRaw
	// (decode happens unconditionally inside reader.get). Either way an
	// entry that reached here passes; honor the keys constraint above
	// (already checked) and accept.
	return true, nil
}

// matchRaw runs the filter's raw-byte pre-screen, used by execFilter and
// Count to skip decode for entries the filter rejects on raw bytes alone.
// The keys path doesn't call this — reader.get always decodes — so raw-only
// filters layered onto MatchKeys still work, just without the decode-skip
// optimization.
func (r Retrieve[K, E]) matchRaw(key, value []byte) (bool, error) {
	if r.filter.raw == nil {
		return true, nil
	}
	return r.filter.raw(key, value)
}

// execKeys streams the candidate key set per-key, fetching one entry at a
// time and immediately running match against it. Returns the slice of keys
// that were not found in the underlying KV store; the caller (Exec) decides
// whether to surface that as query.ErrNotFound or silently skip.
//
// The per-key streaming approach avoids the [][]byte allocation a batched
// fetch would require: each iteration borrows the entry's raw bytes from
// the kv.Closer, runs match, then closes the closer before moving on.
// Memory pressure stays at O(1) regardless of how many keys are requested.
func (r *Retrieve[K, E]) execKeys(ctx context.Context, tx Tx) ([]K, error) {
	var (
		keys       = r.filter.keys
		reader     = wrapReader[K, E](tx, r.keyPrefix)
		notFound   []K
		validCount int
		gorpCtx    = Context{Context: ctx, Tx: tx}
		e          E
		zero       E
	)
	r.entries.ensureCap(len(keys))
	for _, k := range keys {
		e = zero
		b, closer, err := reader.get(ctx, k, &e)
		if err != nil {
			if errors.Is(err, query.ErrNotFound) {
				notFound = append(notFound, k)
				continue
			}
			return nil, err
		}
		if reader.keyCodec.matchPrefix(r.prefix, e.GorpKey()) {
			match, mErr := r.match(gorpCtx, &e, nil, b)
			if mErr != nil {
				return nil, errors.Combine(mErr, closer.Close())
			}
			if match {
				validCount++
				if validCount > r.offset && (r.limit == 0 || validCount <= r.limit+r.offset) {
					r.entries.Add(e)
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

func (r *Retrieve[K, E]) execFilter(ctx context.Context, tx Tx) error {
	var (
		validCount int
		match      bool
		gorpCtx    = Context{Context: ctx, Tx: tx}
	)
	reader := wrapReader[K, E](tx, r.keyPrefix)
	iter, err := reader.OpenIterator(IterOptions{prefix: r.prefix})
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
// fetching matching entries per-key in walk order. Any Where filters are
// applied as post-filters after each fetch.
//
// Like execKeys, this path streams per-key without allocating a batched
// raws slice. The walk is driven by the typed SortedQuery (which satisfies
// OrderQuery) captured at OrderBy time; the cursor lives inside that
// handle, so this path involves no `any` boxing.
func (r *Retrieve[K, E]) execOrdered(ctx context.Context, tx Tx) error {
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
	var (
		reader   = wrapReader[K, E](tx, r.keyPrefix)
		filtered = make([]E, 0, len(keys))
		gorpCtx  = Context{Context: ctx, Tx: tx}
		e        E
		zero     E
	)
	for _, k := range keys {
		e = zero
		b, closer, err := reader.get(ctx, k, &e)
		if err != nil {
			if errors.Is(err, query.ErrNotFound) {
				continue
			}
			return err
		}
		match, mErr := r.match(gorpCtx, &e, nil, b)
		if mErr != nil {
			return errors.Combine(mErr, closer.Close())
		}
		if match {
			filtered = append(filtered, e)
		}
		if cErr := closer.Close(); cErr != nil {
			return cErr
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
