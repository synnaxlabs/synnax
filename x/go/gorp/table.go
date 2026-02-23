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

// TableConfig configures a Table opened via OpenTable.
type TableConfig[E any] struct {
	DB    *DB
	Codec Codec[E]
}

// Table provides a strongly typed interface for a specific entry type within a gorp DB.
// It holds an optional Codec for custom encoding/decoding and provides methods for
// creating query builders that are automatically configured with the codec.
type Table[K Key, E Entry[K]] struct {
	codec Codec[E]
	DB    *DB
}

func (t *Table[K, E]) Close() error {
	return nil
}

// OpenTable creates or opens a table for the given entry type. It runs key migrations
// to ensure entries are stored under the current prefix and key encoding format.
func OpenTable[K Key, E Entry[K]](
	ctx context.Context,
	cfg TableConfig[E],
) (*Table[K, E], error) {
	if err := migrateKeys[K, E](ctx, cfg.DB, cfg.Codec); err != nil {
		return nil, err
	}
	return &Table[K, E]{codec: cfg.Codec, DB: cfg.DB}, nil
}

// NewCreate returns a Create query builder configured with this table's codec.
func (t *Table[K, E]) NewCreate() Create[K, E] {
	return Create[K, E]{entries: new(Entries[K, E]), codec: t.codec}
}

// NewRetrieve returns a Retrieve query builder configured with this table's codec.
func (t *Table[K, E]) NewRetrieve() Retrieve[K, E] {
	return Retrieve[K, E]{entries: new(Entries[K, E]), codec: t.codec}
}

// NewUpdate returns an Update query builder configured with this table's codec.
func (t *Table[K, E]) NewUpdate() Update[K, E] {
	return Update[K, E]{retrieve: t.NewRetrieve(), codec: t.codec}
}

// NewDelete returns a Delete query builder configured with this table's codec.
func (t *Table[K, E]) NewDelete() Delete[K, E] {
	return Delete[K, E]{retrieve: t.NewRetrieve(), codec: t.codec}
}

func migrateKeys[K Key, E Entry[K]](ctx context.Context, db *DB, codec Codec[E]) error {
	return db.WithTx(ctx, func(tx Tx) error {
		if err := migrateOldPrefixKeys[K, E](ctx, tx, codec); err != nil {
			return err
		}
		return reEncodeKeys[K, E](ctx, tx, codec)
	})
}

// migrateOldPrefixKeys finds entries stored under the old codec-based prefix
// (e.g. msgpack-encoded type name) and re-writes them under the new prefix
// format (__gorp__//TypeName).
func migrateOldPrefixKeys[K Key, E Entry[K]](ctx context.Context, tx Tx, codec Codec[E]) (err error) {
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
	writer := wrapWriter[K, E](tx, codec)
	for iter.First(); iter.Valid(); iter.Next() {
		var entry E
		if codec != nil {
			entry, err = codec.Unmarshal(ctx, iter.Value())
		} else {
			err = tx.Decode(ctx, iter.Value(), &entry)
		}
		if err != nil {
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
func reEncodeKeys[K Key, E Entry[K]](ctx context.Context, tx Tx, codec Codec[E]) error {
	reader := wrapReader[K, E](tx, codec)
	iter, err := reader.OpenIterator(IterOptions{})
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Combine(err, iter.Close())
	}()
	writer := wrapWriter[K, E](tx, codec)
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
