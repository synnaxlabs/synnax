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
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/query"
)

// Retrieve is a query that retrieves Entries from the DB.
type Retrieve[K Key, E Entry[K]] struct{ Params query.Parameters }

// NewRetrieve opens a new Retrieve query.
func NewRetrieve[K Key, E Entry[K]]() Retrieve[K, E] {
	return Retrieve[K, E]{Params: make(query.Parameters)}
}

// Where adds the provided filter to the query. If filtering by the key of the Entry,
// use the far more efficient WhereKeys method instead.
func (r Retrieve[K, E]) Where(filter func(*E) bool) Retrieve[K, E] {
	addFilter[K](r.Params, filter)
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
// or Entries will override All previous calls to Entries or Entry. If  multiple results
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

type filters[K Key, E Entry[K]] []func(*E) bool

func (f filters[K, E]) exec(entry *E) bool {
	if len(f) == 0 {
		return true
	}
	for _, filter := range f {
		if filter(entry) {
			return true
		}
	}
	return false
}

func addFilter[K Key, E Entry[K]](q query.Parameters, filter func(*E) bool) {
	var f filters[K, E]
	rf, ok := q.Get(filtersKey)
	if !ok {
		f = filters[K, E]{}
	} else {
		f = rf.(filters[K, E])
	}
	f = append(f, filter)
	q.Set(filtersKey, f)
}

func getFilters[K Key, E Entry[K]](q query.Parameters) filters[K, E] {
	rf, ok := q.Get(filtersKey)
	if !ok {
		return filters[K, E]{}
	}
	return rf.(filters[K, E])
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
		keys, _ = getWhereKeys[K](q)
		f       = getFilters[K, E](q)
		entries = GetEntries[K, E](q)
		e, err  = WrapReader[K, E](tx).GetMany(ctx, keys)
	)
	entries.Replace(lo.Filter(e, func(v E, _ int) bool { return f.exec(&v) }))
	return err
}

func filterRetrieve[K Key, E Entry[K]](
	ctx context.Context,
	q query.Parameters,
	tx Tx,
) error {
	var (
		f       = getFilters[K, E](q)
		entries = GetEntries[K, E](q)
		i, err  = WrapReader[K, E](tx).OpenIterator()
	)
	if err != nil {
		return err
	}
	for i.First(); i.Valid(); i.Next() {
		v := i.Value(ctx)
		if f.exec(v) {
			entries.Add(*v)
		}
	}
	return i.Close()
}
