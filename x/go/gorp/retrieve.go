// Copyright 2023 Synnax Labs, Inc.
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
	"fmt"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/types"
)

// Retrieve is a query that retrieves Entries from the DB.
type Retrieve[K Key, E Entry[K]] struct{ Params query.Parameters }

// NewRetrieve opens a new Retrieve query.
func NewRetrieve[K Key, E Entry[K]]() Retrieve[K, E] {
	return Retrieve[K, E]{Params: make(query.Parameters)}
}

type filterOptions struct {
	required bool
}

type FilterOption func(*filterOptions)

func Required() FilterOption {
	return func(o *filterOptions) { o.required = true }
}

// Where adds the provided filter to the query. If filtering by the key of the Entry,
// use the far more efficient WhereKeys method instead.
func (r Retrieve[K, E]) Where(filter func(*E) bool, opts ...FilterOption) Retrieve[K, E] {
	addFilter[K](r.Params, filter, opts)
	return r
}

func (r Retrieve[K, E]) WherePrefix(prefix []byte) Retrieve[K, E] {
	setWherePrefix(r.Params, prefix)
	return r
}

// Limit sets the maximum number of results that the query will return, discarding
// any results beyond the limit.
func (r Retrieve[K, E]) Limit(limit int) Retrieve[K, E] {
	SetLimit(r.Params, limit)
	return r
}

// Offset sets the number of results that the query will skip before returning results.
func (r Retrieve[K, E]) Offset(offset int) Retrieve[K, E] {
	SetOffset(r.Params, offset)
	return r
}

// WhereKeys queries the DB for Entries with the provided keys. Although more targeted,
// this lookup is substantially faster than a general Where query. If called in
// conjunction with Where, the WhereKeys filter will be applied first. Subsequent calls
// to WhereKeys will append the keys to the existing set.
func (r Retrieve[K, E]) WhereKeys(keys ...K) Retrieve[K, E] {
	setWhereKeys(r.Params, keys...)
	return r
}

// Entries binds a slice that the Params will fill results into. Repeated calls to Entry
// or Entries will override all previous calls to Entries or Entry.
func (r Retrieve[K, E]) Entries(entries *[]E) Retrieve[K, E] {
	SetEntries[K](r.Params, entries)
	return r
}

// Entry binds the entry that the Params will fill results into. Repeated calls to Entry
// or Entries will override All previous calls to Entries or Entry. If  isMultiple results
// are returned by the query, entry will be set to the last result.
func (r Retrieve[K, E]) Entry(entry *E) Retrieve[K, E] {
	SetEntry[K](r.Params, entry)
	return r
}

// Exec executes the Params against the provided Writer. If the WhereKeys method is set on
// the query, Retrieve will return a query.NotFound  error if ANY of the keys do not
// exist in the database. If Where is set on the query, Retrieve will return a query.NotFound
// if NO keys pass the Where filter.
func (r Retrieve[K, E]) Exec(ctx context.Context, tx Tx) error {
	checkForNilTx("Retriever.Exec", tx)
	_, ok := getWhereKeys[K](r.Params)
	f := lo.Ternary(ok, keysRetrieve[K, E], filterRetrieve[K, E])
	return f(ctx, r.Params, tx)
}

// Exists checks whether records matching the query exist in the DB. If the WhereKeys method is
// set on the query, Exists will return true if ANY of the keys exist in the database. If
// Where is set on the query, Exists will return true if ANY keys pass the Where filter.
func (r Retrieve[K, E]) Exists(ctx context.Context, tx Tx) (bool, error) {
	return checkExists[K, E](ctx, r.Params, tx)
}

const filtersKey query.Parameter = "filters"

type filter[K Key, E Entry[K]] struct {
	filterOptions
	f func(*E) bool
}

type filters[K Key, E Entry[K]] []filter[K, E]

func (f filters[K, E]) exec(entry *E) bool {
	if len(f) == 0 {
		return true
	}
	match := false
	for _, fil := range f {
		if fil.f(entry) {
			match = true
		} else if fil.required {
			return false
		}
	}
	return match
}

func addFilter[K Key, E Entry[K]](
	q query.Parameters,
	filterFunc func(*E) bool,
	options []FilterOption,
) {
	var f filters[K, E]
	rf, ok := q.Get(filtersKey)
	if !ok {
		f = filters[K, E]{}
	} else {
		f = rf.(filters[K, E])
	}
	opts := &filterOptions{}
	for _, o := range options {
		o(opts)
	}
	f = append(f, filter[K, E]{f: filterFunc, filterOptions: *opts})
	q.Set(filtersKey, f)
}

func getFilters[K Key, E Entry[K]](q query.Parameters) filters[K, E] {
	rf, ok := q.Get(filtersKey)
	if !ok {
		return filters[K, E]{}
	}
	return rf.(filters[K, E])
}

const limitKey query.Parameter = "limit"

func SetLimit(q query.Parameters, limit int) { q.Set(limitKey, limit) }

func GetLimit(q query.Parameters) (int, bool) {
	limit, ok := q.Get(limitKey)
	if !ok {
		return 0, false
	}
	return limit.(int), true
}

const offsetKey query.Parameter = "offset"

func SetOffset(q query.Parameters, offset int) { q.Set(offsetKey, offset) }

func GetOffset(q query.Parameters) int {
	offset, ok := q.Get(offsetKey)
	if !ok {
		return 0
	}
	return offset.(int)
}

const whereKeysKey query.Parameter = "retrieveByKeys"

type whereKeys[K Key] []K

func setWhereKeys[K Key](q query.Parameters, keys ...K) {
	var (
		keysToSet whereKeys[K]
		ok        bool
	)
	if keysToSet, ok = getWhereKeys[K](q); ok {
		keysToSet = append(keysToSet, keys...)
	} else {
		keysToSet = keys
	}
	q.Set(whereKeysKey, keysToSet)
}

func getWhereKeys[K Key](q query.Parameters) (whereKeys[K], bool) {
	keys, ok := q.Get(whereKeysKey)
	if !ok {
		return nil, false
	}
	return keys.(whereKeys[K]), true
}

const wherePrefixKey query.Parameter = "retrieveByPrefix"

type wherePrefix struct {
	prefix []byte
}

func setWherePrefix(q query.Parameters, prefix []byte) {
	q.Set(wherePrefixKey, wherePrefix{prefix})
}

func getWherePrefix(q query.Parameters) (r []byte) {
	prefix, ok := q.Get(wherePrefixKey)
	if !ok {
		return
	}
	return prefix.(wherePrefix).prefix
}

func checkExists[K Key, E Entry[K]](ctx context.Context, q query.Parameters, reader Tx) (bool, error) {
	if keys, ok := getWhereKeys[K](q); ok {
		entries := make([]E, 0, len(keys))
		SetEntries[K](q, &entries)
		if err := keysRetrieve[K, E](ctx, q, reader); err != nil && !errors.Is(err, query.NotFound) {
			return false, err
		}
		return len(entries) == len(keys), nil
	}
	entries := make([]E, 0, 1)
	SetEntries[K](q, &entries)
	if err := filterRetrieve[K, E](ctx, q, reader); err != nil && !errors.Is(err, query.NotFound) {
		return false, err
	}
	return len(entries) > 0, nil
}

func keysRetrieve[K Key, E Entry[K]](
	ctx context.Context,
	q query.Parameters,
	tx Tx,
) error {
	var (
		limit, limitOk  = GetLimit(q)
		offset          = GetOffset(q)
		keys, _         = getWhereKeys[K](q)
		f               = getFilters[K, E](q)
		entries         = GetEntries[K, E](q)
		keysResult, err = WrapReader[K, E](tx).GetMany(ctx, keys)
		toReplace       = make([]E, 0, len(keysResult))
		validCount      int
	)
	for _, e := range keysResult {
		if f.exec(&e) {
			validCount += 1
			if (validCount > offset) && (!limitOk || validCount <= limit+offset) {
				toReplace = append(toReplace, e)
			}
		}
	}
	entries.Replace(toReplace)
	return err
}

func filterRetrieve[K Key, E Entry[K]](
	ctx context.Context,
	q query.Parameters,
	tx Tx,
) (err error) {
	var (
		limit, limitOk = GetLimit(q)
		offset         = GetOffset(q)
		f              = getFilters[K, E](q)
		entries        = GetEntries[K, E](q)
		validCount     int
	)
	iter, err := WrapReader[K, E](tx).OpenIterator(IterOptions{
		prefix: getWherePrefix(q),
	})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.CombineErrors(err, iter.Close())
	}()
	for iter.First(); iter.Valid(); iter.Next() {
		v := iter.Value(ctx)
		if f.exec(v) {
			validCount += 1
			if (validCount > offset) && (!limitOk || validCount <= limit+offset) {
				entries.Add(*v)
			}
		}
	}
	if entries.isMultiple {
		return nil
	}
	if entries.changes == 0 {
		return errors.Wrapf(
			query.NotFound,
			fmt.Sprintf("no %s found matching query", types.PluralName[E]()),
		)
	}
	return nil
}
