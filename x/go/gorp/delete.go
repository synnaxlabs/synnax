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
	retrieve Retrieve[K, E]
	guards   guards[K, E]
}

// NewDelete opens a new Delete query.
func NewDelete[K Key, E Entry[K]]() Delete[K, E] {
	return Delete[K, E]{retrieve: NewRetrieve[K, E]()}
}

// Where adds the provided filter to the query. To delete by primary key,
// compose MatchKeys into the filter (e.g. d.Where(MatchKeys(1, 2, 3))) — the
// resolved filter's Keys field dispatches to the multi-get fast path.
func (d Delete[K, E]) Where(filters ...Filter[K, E]) Delete[K, E] {
	d.retrieve = d.retrieve.Where(filters...)
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

// Exec executes the query against the provided transaction. If the resolved
// filter is bounded by primary keys and any of those keys do not exist,
// Delete will assume the missing keys do not need to be deleted and continue
// with the keys that do exist.
func (d Delete[K, E]) Exec(ctx context.Context, tx Tx) error {
	checkForNilTx("DeleteChannel.Exec", tx)
	var (
		queryCtx = Context{Context: ctx, Tx: tx}
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
	return WrapWriter[K, E](tx).Delete(ctx, keys...)
}

type GuardFunc[K Key, E Entry[K]] = func(ctx Context, entry E) error
type guards[K Key, E Entry[K]] []GuardFunc[K, E]

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
