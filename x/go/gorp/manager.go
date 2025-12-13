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
)

type EntryManager[K Key, E Entry[K]] struct{}

func (e EntryManager[K, E]) Close() error {
	return nil
}

func OpenEntryManager[K Key, E Entry[K]](ctx context.Context, db *DB) (*EntryManager[K, E], error) {
	if err := migrateKeys[K, E](ctx, db); err != nil {
		return nil, err
	}
	return &EntryManager[K, E]{}, nil
}

func migrateKeys[K Key, E Entry[K]](ctx context.Context, db *DB) error {
	iter, err := WrapReader[K, E](db).OpenIterator(IterOptions{})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	err = db.WithTx(ctx, func(tx Tx) error {
		writer := WrapWriter[K, E](tx)
		for iter.First(); iter.Valid(); iter.Next() {
			//// Delete the iterator by its previous key.
			//if err = writer.BaseWriter.Delete(ctx, iter.Key()); err != nil {
			//	return err
			//}
			// Reset the entry using its new gorp key.
			if err = writer.Set(ctx, *iter.Value(ctx)); err != nil {
				return err
			}
		}
		return nil
	})
	return err
}
