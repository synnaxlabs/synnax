// Copyright 2025 Synnax Labs, Inc.
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
type Delete[K Key, E Entry[K]] struct{ params query.Parameters }

// NewDelete opens a new Delete query.
func NewDelete[K Key, E Entry[K]]() Delete[K, E] {
	return Delete[K, E]{params: make(query.Parameters)}
}

// Where adds the provided filter to the query. If filtering by the key of the Entry,
// use the far more efficient WhereKeys method instead.
func (d Delete[K, E]) Where(filter func(*E) bool, opts ...FilterOption) Delete[K, E] {
	addFilter[K](d.params, filter, opts)
	return d
}

// Guard executes the given function on each entry matching the query. If the function
// returns an error, the query will fail and no further entries will be processed.
func (d Delete[K, E]) Guard(filter func(E) error) Delete[K, E] {
	addGuard[K, E](d.params, filter)
	return d
}

// WhereKeys queries the DB for Entries with the provided keys. Although more targeted,
// this lookup is substantially faster than a general Where query.
// If called in conjunction with Where, the WhereKeys filter will be applied first.
// Subsequent calls to WhereKeys will append the keys to the existing filter.
func (d Delete[K, E]) WhereKeys(keys ...K) Delete[K, E] {
	setWhereKeys(d.params, keys...)
	return d
}

// Exec executes the Params against the provided Writer. If any entries matching WhereKeys
// do not exist in the database, Delete will assume that the keys do not exist and
// do nothing.
func (d Delete[K, E]) Exec(ctx context.Context, tx Tx) error {
	checkForNilTx("DeleteChannel.Exec", tx)
	var (
		entries []E
		q       = (Retrieve[K, E]{Params: d.params}).Entries(&entries)
	)
	if err := q.Exec(ctx, tx); err != nil && !errors.Is(err, query.NotFound) {
		return err
	}
	if err := checkGuards[K](d.params, entries); err != nil {
		return err
	}
	keys := lo.Map(entries, func(entry E, _ int) K { return entry.GorpKey() })
	return WrapWriter[K, E](tx).Delete(ctx, keys...)
}

const deleteGuardKey = "deleteGuard"

type guards[K Key, E Entry[K]] []func(E) error

func (g guards[K, E]) exec(entry E) error {
	for _, f := range g {
		if err := f(entry); err != nil {
			return err
		}
	}
	return nil
}

func addGuard[K Key, E Entry[K]](q query.Parameters, guard func(E) error) {
	var g guards[K, E]
	rg, ok := q.Get(deleteGuardKey)
	if !ok {
		g = make(guards[K, E], 0, 1)
	} else {
		g = rg.(guards[K, E])
	}
	g = append(g, guard)
	q.Set(deleteGuardKey, g)
}

func checkGuards[K Key, E Entry[K]](q query.Parameters, entries []E) error {
	g, ok := q.Get(deleteGuardKey)
	if !ok {
		return nil
	}
	guards_ := g.(guards[K, E])
	for _, entry := range entries {
		if err := guards_.exec(entry); err != nil {
			return err
		}
	}
	return nil
}
