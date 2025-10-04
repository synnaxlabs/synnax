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

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Update is a query that updates Entries in the DB.
type Update[K Key, E Entry[K]] struct{ retrieve Retrieve[K, E] }

// NewUpdate opens a new Update query.
func NewUpdate[K Key, E Entry[K]]() Update[K, E] {
	return Update[K, E]{retrieve: NewRetrieve[K, E]()}
}

func (u Update[K, E]) Where(filter FilterFunc[K, E]) Update[K, E] {
	u.retrieve = u.retrieve.Where(filter)
	return u
}

func (u Update[K, E]) WhereKeys(keys ...K) Update[K, E] {
	u.retrieve = u.retrieve.WhereKeys(keys...)
	return u
}

func (u Update[K, E]) Change(f func(Context, E) E) Update[K, E] {
	return u.ChangeErr(func(ctx Context, e E) (E, error) { return f(ctx, e), nil })
}

func (u Update[K, E]) ChangeErr(f func(Context, E) (E, error)) Update[K, E] {
	addChange[K, E](u.retrieve.Params, f)
	return u
}

func (u Update[K, E]) Exec(ctx context.Context, tx Tx) (err error) {
	checkForNilTx("update.Exec", tx)
	var entries []E
	if err := u.retrieve.Entries(&entries).Exec(ctx, tx); err != nil {
		return err
	}
	c := getChanges[K, E](u.retrieve.Params)
	if len(c) == 0 {
		return errors.Wrap(query.InvalidParameters, "[gorp] - update query must specify at least one change function")
	}
	for i, e := range entries {
		if entries[i], err = c.exec(Context{Context: ctx, Tx: tx}, e); err != nil {
			return err
		}
	}
	return WrapWriter[K, E](tx).Set(ctx, entries...)
}

const updateChangeKey = "updateChange"

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

func addChange[K Key, E Entry[K]](q query.Parameters, change ChangeFunc[K, E]) {
	var c changes[K, E]
	rc, ok := q.Get(updateChangeKey)
	if !ok {
		c = make(changes[K, E], 0, 1)
	} else {
		c = rc.(changes[K, E])
	}
	c = append(c, change)
	q.Set(updateChangeKey, c)
}

func getChanges[K Key, E Entry[K]](q query.Parameters) (c changes[K, E]) {
	rc, ok := q.Get(updateChangeKey)
	if !ok {
		return c
	}
	return rc.(changes[K, E])
}
