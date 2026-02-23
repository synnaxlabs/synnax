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
	stdbinary "encoding/binary"
	"io"
	"iter"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/types"
)

// TableConfig configures a Table opened via OpenTable.
type TableConfig[E any] struct {
	DB         *DB
	Codec      Codec[E]
	Migrations []Migration
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

// OpenTable creates or opens a table for the given entry type. It runs any provided
// versioned migrations followed by key migrations to ensure entries are stored under
// the current prefix and key encoding format.
func OpenTable[K Key, E Entry[K]](
	ctx context.Context,
	cfg TableConfig[E],
) (*Table[K, E], error) {
	prefix := []byte(magicPrefix + types.Name[E]())
	versionKey := []byte(migrationVersionPrefix + types.Name[E]())
	kvTx := cfg.DB.KV().OpenTx()
	defer func() {
		_ = kvTx.Close()
	}()
	migCfg := MigrationConfig{Prefix: prefix, Codec: cfg.DB.Codec}
	if len(cfg.Migrations) > 0 {
		currentVersion, err := readMigrationVersion(ctx, kvTx, versionKey)
		if err != nil {
			return nil, err
		}
		for i := int(currentVersion); i < len(cfg.Migrations); i++ {
			if err := cfg.Migrations[i].Run(ctx, kvTx, migCfg); err != nil {
				return nil, errors.Wrapf(
					err,
					"migration %d (%s) failed",
					i+1,
					cfg.Migrations[i].Name(),
				)
			}
			if err := writeMigrationVersion(
				ctx, kvTx, versionKey, uint16(i+1),
			); err != nil {
				return nil, err
			}
		}
	}
	gorpTx := WrapTx(kvTx, cfg.DB.Codec)
	if err := migrateOldPrefixKeys[K, E](ctx, gorpTx, cfg.Codec); err != nil {
		return nil, err
	}
	if err := reEncodeKeys[K, E](ctx, gorpTx, cfg.Codec); err != nil {
		return nil, err
	}
	if err := kvTx.Commit(ctx); err != nil {
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

// OpenNexter opens a new Nexter over entries in the table using the table's codec
// for decoding.
func (t *Table[K, E]) OpenNexter(ctx context.Context) (iter.Seq[E], io.Closer, error) {
	return wrapReader[K, E](t.DB, t.codec).OpenNexter(ctx)
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

func readMigrationVersion(
	ctx context.Context,
	kvTx kv.Tx,
	key []byte,
) (uint16, error) {
	b, closer, err := kvTx.Get(ctx, key)
	if err != nil {
		if errors.Is(err, query.ErrNotFound) {
			return 0, nil
		}
		return 0, err
	}
	defer func() {
		err = errors.Combine(err, closer.Close())
	}()
	if len(b) < 2 {
		return 0, err
	}
	return stdbinary.BigEndian.Uint16(b), err
}

func writeMigrationVersion(
	ctx context.Context,
	kvTx kv.Tx,
	key []byte,
	version uint16,
) error {
	b := make([]byte, 2)
	stdbinary.BigEndian.PutUint16(b, version)
	return kvTx.Set(ctx, key, b)
}
