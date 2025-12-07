// Copyright 2025 Synnax Labs, Inc.
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

// Retrieve is a query that retrieves Entries from the DB.
type Retrieve[K Key, E Entry[K]] struct {
	entries   *Entries[K, E]
	limit     int
	offset    int
	whereKeys *[]K
	prefix    []byte
	filters   filters[K, E]
}

// NewRetrieve opens a new Retrieve query.
func NewRetrieve[K Key, E Entry[K]]() Retrieve[K, E] {
	return Retrieve[K, E]{entries: new(Entries[K, E])}
}

type filterOptions struct {
	required bool
}

type FilterOption func(*filterOptions)

func Required() FilterOption {
	return func(o *filterOptions) { o.required = true }
}

type FilterFunc[K Key, E Entry[K]] = func(ctx Context, e *E) (bool, error)

// GetEntries returns the entries bound to the query.
func (r Retrieve[K, E]) GetEntries() *Entries[K, E] { return r.entries }

// Where adds the provided filter to the query. If filtering by the key of the Entry,
// use the far more efficient WhereKeys method instead.
func (r Retrieve[K, E]) Where(filter FilterFunc[K, E], opts ...FilterOption) Retrieve[K, E] {
	r.filters = append(r.filters, newFilter(filter, opts))
	return r
}

// HasLimit returns true if a limit was set on the query.
func (r Retrieve[K, E]) HasLimit() bool { return r.limit > 0 }

// HasOffset returns true if an offset was set on the query.
func (r Retrieve[K, E]) HasOffset() bool { return r.offset > 0 }

// HasWhereKeys returns true if WhereKeys was called on the query.
func (r Retrieve[K, E]) HasWhereKeys() bool { return r.whereKeys != nil }

// GetWhereKeys returns the keys set by WhereKeys, or nil if not set.
func (r Retrieve[K, E]) GetWhereKeys() []K {
	if r.whereKeys != nil {
		return *r.whereKeys
	}
	return nil
}

// HasFilters returns true if any Where filters were added to the query.
func (r Retrieve[K, E]) HasFilters() bool { return len(r.filters) > 0 }

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

// WhereKeys queries the DB for Entries with the provided keys. Although more targeted,
// this lookup is substantially faster than a general Where query. If called in
// conjunction with Where, the WhereKeys filter will be applied first. Subsequent calls
// to WhereKeys will append the keys to the existing set.
func (r Retrieve[K, E]) WhereKeys(keys ...K) Retrieve[K, E] {
	if r.whereKeys == nil {
		r.whereKeys = new([]K)
	}
	*r.whereKeys = append(*r.whereKeys, keys...)
	return r
}

// Entries binds a slice that the Params will fill results into. Repeated calls to Entry
// or Entries will override all previous calls to Entries or Entry.
func (r Retrieve[K, E]) Entries(entries *[]E) Retrieve[K, E] {
	r.entries = multipleEntries[K, E](entries)
	return r
}

// Entry binds the entry that the Params will fill results into. Repeated calls to Entry
// or Entries will override All previous calls to Entries or Entry. If  isMultiple results
// are returned by the query, entry will be set to the last result.
func (r Retrieve[K, E]) Entry(entry *E) Retrieve[K, E] {
	r.entries = singleEntry[K, E](entry)
	return r
}

// Exec executes the Params against the provided Writer. If the WhereKeys method is set on
// the query, Retrieve will return a query.NotFound  error if ANY of the keys do not
// exist in the database. If Where is set on the query, Retrieve will return a query.NotFound
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
		e := make([]E, 0, len(*r.whereKeys))
		r.entries = multipleEntries[K, E](&e)
		if err := r.execKeys(ctx, tx); errors.Skip(err, query.NotFound) != nil {
			return false, err
		}
		return len(e) == len(*r.whereKeys), nil
	}
	e := make([]E, 0, 1)
	r.entries = multipleEntries(&e)
	if err := r.execFilter(ctx, tx); errors.Skip(err, query.NotFound) != nil {
		return false, err
	}
	return len(e) > 0, nil
}

// Count returns the number of records matching the query. If the WhereKeys method is
// set on the query, Count will return the number of existing keys. If Where is set
// on the query, Count will return the number of records that pass the Where filter.
func (r Retrieve[K, E]) Count(ctx context.Context, tx Tx) (int, error) {
	checkForNilTx("Retriever.Count", tx)
	if r.HasWhereKeys() {
		// For key-based queries, we can optimize by only retrieving the keys
		e := make([]E, 0, len(*r.whereKeys))
		r.entries = multipleEntries[K, E](&e)
		if err := r.execKeys(ctx, tx); err != nil && !errors.Is(err, query.NotFound) {
			return 0, err
		}
		return len(r.entries.All()), nil
	}

	// For filter-based queries, we need to iterate through all records
	var count int
	f := r.filters
	iter, err := WrapReader[K, E](tx).OpenIterator(IterOptions{
		prefix: r.prefix,
	})
	if err != nil {
		return 0, err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		match, err := f.exec(Context{
			Context: ctx,
			Tx:      tx,
		}, iter.Value(ctx))
		if err != nil {
			return 0, err
		}
		if match {
			count++
		}
	}
	return count, err
}

type filter[K Key, E Entry[K]] struct {
	filterOptions
	f FilterFunc[K, E]
}

type filters[K Key, E Entry[K]] []filter[K, E]

func (f filters[K, E]) exec(ctx Context, entry *E) (bool, error) {
	if len(f) == 0 {
		return true, nil
	}
	match := false
	for _, fil := range f {
		iMatch, err := fil.f(ctx, entry)
		if err != nil {
			return false, err
		}
		if iMatch {
			match = true
		} else if fil.required {
			return false, err
		}
	}
	return match, nil
}

func newFilter[K Key, E Entry[K]](
	filterFunc FilterFunc[K, E],
	options []FilterOption,
) filter[K, E] {
	opts := &filterOptions{}
	for _, o := range options {
		o(opts)
	}
	return filter[K, E]{f: filterFunc, filterOptions: *opts}
}

func (r Retrieve[K, E]) execKeys(ctx context.Context, tx Tx) error {
	var (
		keysResult, err = WrapReader[K, E](tx).GetMany(ctx, *r.whereKeys)
		toReplace       = make([]E, 0, len(keysResult))
		validCount      int
	)
	for _, e := range keysResult {
		match, err := r.filters.exec(Context{Context: ctx, Tx: tx}, &e)
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
	return err
}

func (r Retrieve[K, E]) execFilter(ctx context.Context, tx Tx) error {
	var (
		validCount int
		match      bool
	)
	iter, err := WrapReader[K, E](tx).OpenIterator(IterOptions{prefix: r.prefix})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		v := iter.Value(ctx)
		if err = iter.Error(); err != nil {
			return err
		}
		match, err = r.filters.exec(Context{Context: ctx, Tx: tx}, v)
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
	if r.entries.isMultiple || !r.entries.Bound() {
		return err
	}
	if r.entries.changes == 0 {
		return errors.Wrapf(
			query.NotFound,
			"no %s found matching query",
			types.PluralName[E](),
		)
	}
	return err
}
