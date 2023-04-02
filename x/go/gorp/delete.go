// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

// Delete is a query that deletes Entries from the DB.
type Delete[K Key, E Entry[K]] struct{ query.Query }

// NewDelete opens a new Delete query.
func NewDelete[K Key, E Entry[K]]() Delete[K, E] { return Delete[K, E]{Query: query.New()} }

// Where adds the provided filter to the query. If filtering by the key of the Entry,
// use the far more efficient WhereKeys method instead.
func (d Delete[K, E]) Where(filter func(*E) bool) Delete[K, E] {
	addFilter[K, E](d.Query, filter)
	return d
}

// WhereKeys queries the DB for Entries with the provided keys. Although more targeted,
// this lookup is substantially faster than a general Where query.
// If called in conjunction with Where, the WhereKeys filter will be applied first.
// Subsequent calls to WhereKeys will append the keys to the existing filter.
func (d Delete[K, E]) WhereKeys(keys ...K) Delete[K, E] {
	setWhereKeys[K](d.Query, keys...)
	return d
}

// Exec executes the Query against the provided Txn. If any entries matching WhereKeys
// do not exist in the database, Delete will assume that the keys do not exist and
// do nothing.
func (d Delete[K, E]) Exec(ctx context.Context, txn Txn) error {
	return (&del[K, E]{Txn: txn}).exec(ctx, d)
}

type del[K Key, E Entry[K]] struct{ Txn }

func (d *del[K, E]) exec(ctx context.Context, q query.Query) error {
	var (
		opts    = d.Txn.options()
		entries []E
		err     = (Retrieve[K, E]{Query: q}).Entries(&entries).Exec(d)
		prefix  = typePrefix[K, E](opts)
		keys    whereKeys[K]
	)
	if err != nil && !errors.Is(err, query.NotFound) {
		return err
	}
	for _, entry := range entries {
		keys = append(keys, entry.GorpKey())
	}
	byteKeys, err := keys.bytes(opts.encoder)
	if err != nil {
		return err
	}
	for _, key := range byteKeys {
		if err = d.Delete(ctx, append(prefix, key...)); err != nil && !errors.Is(err, kv.NotFound) {
			return err
		}
	}
	return nil
}
