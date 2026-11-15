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

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Delete is a query that deletes Entries from the DB.
type Delete[K Key, E Entry[K]] struct {
	// retrieve is the underlying scan used to resolve entries to delete.
	retrieve Retrieve[K, E]
	// guards is the chain of GuardFuncs checked against each matched entry; any non-nil
	// error aborts the delete.
	guards guards[K, E]
	// table is the Table the query deletes from. Nil for a query that no Table built.
	table *Table[K, E]
}

// NewDelete opens a new Delete query.
func NewDelete[K Key, E Entry[K]]() Delete[K, E] {
	return Delete[K, E]{retrieve: NewRetrieve[K, E]()}
}

// Where adds the provided filter to the query. To delete by primary key, compose
// MatchKeys into the filter (e.g. d.Where(MatchKeys(1, 2, 3))).
func (d Delete[K, E]) Where(filter Filter[K, E]) Delete[K, E] {
	d.retrieve = d.retrieve.Where(filter)
	return d
}

// WhereRaw adds a raw byte filter that runs against each entry's pebble key and encoded
// value before decoding. Returning false skips the entry without allocating a decoded
// value, so a key-shaped predicate can drop most rows without paying decode cost.
func (d Delete[K, E]) WhereRaw(filter rawFilter) Delete[K, E] {
	d.retrieve = d.retrieve.WhereRaw(filter)
	return d
}

// Guard executes the given function on each entry matching the query. If the function
// returns an error, the query will fail and no further entries will be processed. If
// the provided guard function is nil, no guard will be applied.
func (d Delete[K, E]) Guard(filter GuardFunc[K, E]) Delete[K, E] {
	if filter == nil {
		return d
	}
	d.guards = append(d.guards, filter)
	return d
}

// Exec executes the query against the provided transaction. If the resolved filter is
// bounded by primary keys and any of those keys do not exist, Delete will assume the
// missing keys do not need to be deleted and continue with the keys that do exist.
//
// A Delete built by a Table also applies the OnDelete of every Reference that points
// at a deleted entry: it cascades, refuses, or detaches before it changes anything.
// Guards run only on the entries the query matches, never on cascaded ones.
func (d Delete[K, E]) Exec(ctx context.Context, tx Tx) error {
	checkForNilTx("Delete.Exec", tx)
	return inTx(ctx, tx, func(tx Tx) error { return d.exec(ctx, tx) })
}

func (d Delete[K, E]) exec(ctx context.Context, tx Tx) error {
	var (
		queryCtx = newContext(ctx, tx)
		entries  []E
		q        = d.retrieve.Entries(&entries)
	)
	if err := q.Exec(ctx, tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return err
	}
	if err := d.guards.checkMany(queryCtx, entries); err != nil {
		return err
	}
	keys := lo.Map(entries, func(entry E, _ int) K { return entry.GorpKey() })
	if d.table != nil {
		return d.table.delete(ctx, tx, keys)
	}
	w := wrapWriter[K, E](tx, "", d.retrieve.keyPrefix, nil)
	return w.Delete(ctx, keys...)
}

type (
	GuardFunc[K Key, E Entry[K]] = func(Context, E) error
	guards[K Key, E Entry[K]]    []GuardFunc[K, E]
)

func (g guards[K, E]) checkOne(ctx Context, entry E) error {
	for _, f := range g {
		if err := f(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

func (g guards[K, E]) checkMany(ctx Context, entries []E) error {
	for _, entry := range entries {
		if err := g.checkOne(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}
