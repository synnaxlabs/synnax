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

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Update is a query that updates Entries in the DB.
type Update[K Key, E Entry[K]] struct {
	retrieve Retrieve[K, E]
	changes  changes[K, E]
	// indexes mirrors Create.indexes — propagated by Table.NewUpdate so the
	// writer built in Exec stages mutations against the Table's secondary
	// indexes.
	indexes []Index[K, E]
}

// NewUpdate opens a new Update query.
func NewUpdate[K Key, E Entry[K]]() Update[K, E] {
	return Update[K, E]{retrieve: NewRetrieve[K, E]()}
}

// Where adds the provided filter to the query. To update by primary key,
// compose MatchKeys into the filter (e.g. u.Where(MatchKeys(1, 2, 3))).
func (u Update[K, E]) Where(filter Filter[K, E]) Update[K, E] {
	u.retrieve = u.retrieve.Where(filter)
	return u
}

func (u Update[K, E]) Change(f func(Context, E) E) Update[K, E] {
	return u.ChangeErr(func(ctx Context, e E) (E, error) { return f(ctx, e), nil })
}

func (u Update[K, E]) ChangeErr(f func(Context, E) (E, error)) Update[K, E] {
	u.changes = append(u.changes, f)
	return u
}

func (u Update[K, E]) Exec(ctx context.Context, tx Tx) (err error) {
	checkForNilTx("update.Exec", tx)
	var entries []E
	if err = u.retrieve.Entries(&entries).Exec(ctx, tx); err != nil {
		return err
	}
	if len(u.changes) == 0 {
		return errors.Wrap(query.ErrInvalidParameters, "[gorp] - update query must specify at least one change function")
	}
	for i, e := range entries {
		if entries[i], err = u.changes.exec(Context{Context: ctx, Tx: tx}, e); err != nil {
			return err
		}
	}
	return wrapWriter[K, E](tx, u.retrieve.keyPrefix, u.indexes).Set(ctx, entries...)
}

type ChangeFunc[K Key, E Entry[K]] = func(Context, E) (E, error)

type changes[K Key, E Entry[K]] []ChangeFunc[K, E]

func (c changes[K, E]) exec(ctx Context, entry E) (o E, err error) {
	for _, change := range c {
		if o, err = change(ctx, entry); err != nil {
			return
		}
	}
	return
}
