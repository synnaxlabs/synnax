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

	"github.com/samber/lo"
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
	filter     *Filter[K, E]
	validators []Validator[K, E]
}

// NewRetrieve opens a new Retrieve query.
func NewRetrieve[K Key, E Entry[K]]() Retrieve[K, E] {
	return Retrieve[K, E]{entries: new(Entries[K, E])}
}

// GetEntries returns the entries bound to the query.
func (r Retrieve[K, E]) GetEntries() *Entries[K, E] { return r.entries }

// Where adds the provided filters to the query, ANDing them with any existing filter.
// If filtering by the key of the Entry, use the far more efficient WhereKeys method
// instead.
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
func (r Retrieve[K, E]) HasFilters() bool { return r.filter != nil }

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
func (r Retrieve[K, E]) Exec(ctx context.Context, tx Tx) error {
	checkForNilTx("Retriever.Exec", tx)
	f := lo.Ternary(r.HasWhereKeys(), r.execKeys, r.execFilter)
	return f(ctx, tx)
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
	if r.filter == nil || r.filter.Eval == nil {
		return true, nil
	}
	return r.filter.Eval(ctx, e, raw)
}

func (r Retrieve[K, E]) matchRaw(data []byte) (bool, error) {
	if r.filter == nil || r.filter.Raw == nil {
		return true, nil
	}
	return r.filter.Raw(data)
}

func (r Retrieve[K, E]) execKeys(ctx context.Context, tx Tx) error {
	var (
		reader                   = WrapReader[K, E](tx)
		keysResult, raws, getErr = reader.getMany(ctx, *r.keys)
		toReplace                = make([]E, 0, len(keysResult))
		validCount               int
		gorpCtx                  = Context{Context: ctx, Tx: tx}
	)
	// We don't return early even if getErr fails with a not found result in order
	// to do a best effort retrieval of available items.
	if getErr != nil && !errors.Is(getErr, query.ErrNotFound) {
		return getErr
	}
	for i, e := range keysResult {
		if !reader.keyCodec.matchPrefix(r.prefix, e.GorpKey()) {
			continue
		}
		match, err := r.match(gorpCtx, &e, raws[i])
		if err != nil {
			return err
		}
		if match {
			validCount += 1
			if (validCount > r.offset) && (r.limit == 0 || validCount <= r.limit+r.offset) {
				toReplace = append(toReplace, e)
			}
		}
	}
	r.entries.Replace(toReplace)
	if err := r.runValidators(gorpCtx, toReplace); err != nil {
		return err
	}
	return getErr
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
