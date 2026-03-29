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
	"io"
	"iter"

	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/types"
)

// TableConfig configures a Table opened via OpenTable.
type TableConfig[E any] struct {
	DB    *DB
	Codec binary.Codec
}

// Table provides a strongly typed interface for a specific entry type within a gorp DB.
// It holds a resolved Codec (either custom or the DB's default) and provides methods
// for creating query builders that are automatically configured with the codec.
type Table[K Key, E Entry[K]] struct {
	codec binary.Codec
	DB    *DB
}

// Codec returns the table's codec.
func (t *Table[K, E]) Codec() binary.Codec { return t.codec }

func (t *Table[K, E]) Close() error {
	return nil
}

// OpenTable creates or opens a table for the given entry type. It runs key migrations
// to ensure entries are stored under the current prefix and key encoding format.
func OpenTable[K Key, E Entry[K]](
	ctx context.Context,
	cfg TableConfig[E],
) (*Table[K, E], error) {
	codec := resolveCodec(cfg.Codec, cfg.DB)
	if err := migrateKeys[K, E](ctx, cfg.DB, codec); err != nil {
		return nil, err
	}
	return &Table[K, E]{codec: codec, DB: cfg.DB}, nil
}

// resolveCodec returns the override codec if non-nil, otherwise falls back to the
// fallback codec.
func resolveCodec(override binary.Codec, fallback binary.Codec) binary.Codec {
	if override != nil {
		return override
	}
	return fallback
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

// OpenNexter opens a new Nexter over entries in the table using the table's codec for
// decoding.
func (t *Table[K, E]) OpenNexter(ctx context.Context) (iter.Seq[E], io.Closer, error) {
	return wrapReader[K, E](t.DB, t.codec).OpenNexter(ctx)
}

func migrateKeys[K Key, E Entry[K]](ctx context.Context, db *DB, codec binary.Codec) error {
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
func migrateOldPrefixKeys[K Key, E Entry[K]](ctx context.Context, tx Tx, codec binary.Codec) (err error) {
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
		if err = codec.Decode(ctx, iter.Value(), &entry); err != nil {
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
func reEncodeKeys[K Key, E Entry[K]](ctx context.Context, tx Tx, codec binary.Codec) error {
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
		if err = writer.writer.Delete(ctx, iter.Key()); err != nil {
			return err
		}
		if err = writer.Set(ctx, *iter.Value(ctx)); err != nil {
			return err
		}
	}
	return err
}
