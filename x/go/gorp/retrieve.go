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
	prefix     []byte
	filter     *Filter[K, E]
	validators []Validator[K, E]
}

// NewRetrieve opens a new Retrieve query.
func NewRetrieve[K Key, E Entry[K]]() Retrieve[K, E] {
	return Retrieve[K, E]{entries: new(Entries[K, E])}
}

// GetEntries returns the entries bound to the query.
func (r Retrieve[K, E]) GetEntries() *Entries[K, E] { return r.entries }

// Where adds the provided filters to the query, ANDing them with any existing
// filter. To restrict by primary key, compose MatchKeys into the Where clause
// (e.g. r.Where(MatchKeys(1, 2, 3))) — the resolved filter's Keys field is
// what dispatches Exec to the multi-get fast path.
func (r Retrieve[K, E]) Where(filters ...Filter[K, E]) Retrieve[K, E] {
	combined := And(filters...)
	if r.filter != nil {
		combined = And(*r.filter, combined)
	}
	r.filter = &combined
	return r
}

// HasLimit returns true if a limit was set on the query.
func (r Retrieve[K, E]) HasLimit() bool { return r.limit > 0 }

// HasOffset returns true if an offset was set on the query.
func (r Retrieve[K, E]) HasOffset() bool { return r.offset > 0 }

// HasFilters returns true if any Where filters were added to the query.
func (r Retrieve[K, E]) HasFilters() bool { return r.filter != nil }

// HasFilterKeys returns true if the resolved filter is bounded by a primary
// key set (i.e. Where(MatchKeys(...)) was called, possibly composed with
// other filters under And). Routing layers that fan out by key set use this
// to decide whether to read GetFilterKeys for sharded dispatch.
func (r Retrieve[K, E]) HasFilterKeys() bool {
	return r.filter != nil && r.filter.keys != nil
}

// GetFilterKeys returns the resolved filter's primary key set, or nil if the
// query is not bounded by keys. Mutating the returned slice mutates the
// underlying filter.
func (r Retrieve[K, E]) GetFilterKeys() []K {
	if r.filter == nil {
		return nil
	}
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

// WhereRaw adds a raw byte filter that is evaluated before decoding each entry.
// Entries whose raw bytes cause the filter to return false are skipped without
// being decoded.
func (r Retrieve[K, E]) WhereRaw(filter RawFilter) Retrieve[K, E] {
	return r.Where(MatchRaw[K, E](filter))
}

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

// isBareKeys reports whether the resolved filter is keys-only (Keys set with
// no Eval / Raw post-filter). When true, "key X requested but missing" is
// unambiguous — there is no other filter that could have excluded a present
// entry — so Exec applies the strict ErrNotFound semantic in that case.
// Composing any post-filter (e.g. Where(MatchKeys(...), MatchNames(...))) is
// enough to drop the strict semantic, since a missing key result could mean
// either "not in storage" or "filtered out".
func (r Retrieve[K, E]) isBareKeys() bool {
	return r.filter != nil &&
		r.filter.keys != nil &&
		r.filter.eval == nil &&
		r.filter.raw == nil
}

// Exec executes the Params against the provided Writer. If Where(MatchKeys(...))
// is the only filter on the query, Retrieve will return a query.ErrNotFound
// error if ANY of the keys do not exist in the database. If additional filters
// compose with the keys (e.g. Where(MatchKeys(...), MatchNames(...))), missing
// keys are no longer unambiguous and ErrNotFound is suppressed; an empty
// result with a single-entry binding still surfaces ErrNotFound through
// execFilter's existing path.
func (r Retrieve[K, E]) Exec(ctx context.Context, tx Tx) error {
	checkForNilTx("Retriever.Exec", tx)
	if r.HasFilterKeys() {
		return r.execKeys(ctx, tx)
	}
	return r.execFilter(ctx, tx)
}

// Exists checks whether records matching the query exist in the DB. If the
// resolved filter is bounded by primary keys, Exists returns true if ANY of
// those keys exist in the database (subject to any composed post-filter).
// Otherwise Exists returns true if any entry passes the Where filter.
func (r Retrieve[K, E]) Exists(ctx context.Context, tx Tx) (bool, error) {
	if r.HasFilterKeys() {
		keys := r.filter.keys
		e := make([]E, 0, len(keys))
		r.entries = multipleEntries(&e)
		if err := r.execKeys(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
			return false, err
		}
		if r.isBareKeys() {
			return len(e) == len(keys), nil
		}
		return len(e) > 0, nil
	}
	e := make([]E, 0, 1)
	r.entries = multipleEntries(&e)
	if err := r.execFilter(ctx, tx); errors.Skip(err, query.ErrNotFound) != nil {
		return false, err
	}
	return len(e) > 0, nil
}

// Count returns the number of records matching the query. If the resolved
// filter is bounded by primary keys, Count returns the number of those keys
// that exist (and pass any composed post-filter). Otherwise Count returns
// the number of records that pass the Where filter.
func (r Retrieve[K, E]) Count(ctx context.Context, tx Tx) (count int, err error) {
	checkForNilTx("Retriever.Count", tx)
	if r.HasFilterKeys() {
		e := make([]E, 0, len(r.filter.keys))
		r.entries = multipleEntries(&e)
		if err := r.execKeys(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
			return 0, err
		}
		return len(r.entries.All()), nil
	}

	iter, err := WrapReader[K, E](tx).OpenIterator(IterOptions{
		prefix: r.prefix,
	})
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
		rawBytes := iter.Iterator.Value()
		rawMatched, rErr := r.matchRaw(rawBytes)
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
		match, fErr := r.match(gorpCtx, v, rawBytes)
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

func (r Retrieve[K, E]) match(ctx Context, e *E, raw []byte) (bool, error) {
	if r.filter == nil || r.filter.eval == nil {
		return true, nil
	}
	return r.filter.eval(ctx, e, raw)
}

func (r Retrieve[K, E]) matchRaw(data []byte) (bool, error) {
	if r.filter == nil || r.filter.raw == nil {
		return true, nil
	}
	return r.filter.raw(data)
}

func (r Retrieve[K, E]) execKeys(ctx context.Context, tx Tx) error {
	keys := r.filter.keys
	var (
		reader      = WrapReader[K, E](tx)
		seq, closer = reader.iterKeys(ctx, keys)
		out         = make([]E, 0, len(keys))
		validCount  int
		gorpCtx     = Context{Context: ctx, Tx: tx}
	)
	for e, raw := range seq {
		if !reader.keyCodec.matchPrefix(r.prefix, e.GorpKey()) {
			continue
		}
		match, err := r.match(gorpCtx, &e, raw)
		if err != nil {
			return errors.Combine(err, closer.Close())
		}
		if !match {
			continue
		}
		validCount++
		if validCount > r.offset && (r.limit == 0 || validCount <= r.limit+r.offset) {
			out = append(out, e)
		}
	}
	cErr := closer.Close()
	r.entries.Replace(out)
	vErr := r.runValidators(gorpCtx, out)
	return errors.Join(vErr, cErr)
}

func (r Retrieve[K, E]) execFilter(ctx context.Context, tx Tx) error {
	var (
		validCount int
		match      bool
		gorpCtx    = Context{Context: ctx, Tx: tx}
	)
	iter, err := WrapReader[K, E](tx).OpenIterator(IterOptions{prefix: r.prefix})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		rawBytes := iter.Iterator.Value()
		rawMatched, rErr := r.matchRaw(rawBytes)
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
		match, err = r.match(gorpCtx, v, rawBytes)
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
