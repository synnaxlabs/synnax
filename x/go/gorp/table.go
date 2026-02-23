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
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/types"
)

type Table[K Key, E Entry[K]] struct{}

func (e Table[K, E]) Close() error {
	return nil
}

func OpenTable[K Key, E Entry[K]](ctx context.Context, db *DB) (*Table[K, E], error) {
	if err := migrateKeys[K, E](ctx, db); err != nil {
		return nil, err
	}
	return &Table[K, E]{}, nil
}

func migrateKeys[K Key, E Entry[K]](ctx context.Context, db *DB) error {
	return db.WithTx(ctx, func(tx Tx) error {
		if err := migrateOldPrefixKeys[K, E](ctx, tx); err != nil {
			return err
		}
		return reEncodeKeys[K, E](ctx, tx)
	})
}

// migrateOldPrefixKeys finds entries stored under the old codec-based prefix
// (e.g. msgpack-encoded type name) and re-writes them under the new prefix
// format (__gorp__//TypeName).
func migrateOldPrefixKeys[K Key, E Entry[K]](ctx context.Context, tx Tx) (err error) {
	oldPrefix, err := tx.Encode(ctx, types.Name[E]())
	if err != nil {
		return err
	}
	newCodec := newKeyCodec[K, E]()
	if string(oldPrefix) == string(newCodec.prefix) {
		return nil
	}
	iter, err := tx.OpenIterator(kv.IterPrefix(oldPrefix))
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	writer := WrapWriter[K, E](tx)
	for iter.First(); iter.Valid(); iter.Next() {
		var entry E
		if err = tx.Decode(ctx, iter.Value(), &entry); err != nil {
			return err
		}
		if err = tx.Delete(ctx, iter.Key()); err != nil {
			return err
		}
		if err = writer.Set(ctx, entry); err != nil {
			return err
		}
	}
	return err
}

// reEncodeKeys iterates entries already stored under the current prefix and
// re-writes them, ensuring the key portion uses the current primitive encoding.
// This is a no-op when the key encoding hasn't changed.
func reEncodeKeys[K Key, E Entry[K]](ctx context.Context, tx Tx) error {
	iter, err := WrapReader[K, E](tx).OpenIterator(IterOptions{})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	writer := WrapWriter[K, E](tx)
	for iter.First(); iter.Valid(); iter.Next() {
		if err = writer.BaseWriter.Delete(ctx, iter.Key()); err != nil {
			return err
		}
		if err = writer.Set(ctx, *iter.Value(ctx)); err != nil {
			return err
		}
	}
	return err
}
