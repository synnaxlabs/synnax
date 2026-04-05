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
	"encoding/json"
	"io"
	"iter"
	"sort"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/types"
)

// TableConfig configures a Table opened via OpenTable.
type TableConfig[E any] struct {
	DB         *DB
	Migrations []migrate.Migration
	alamos.Instrumentation
}

// Table provides a strongly typed interface for a specific entry type within a gorp DB.
type Table[K Key, E Entry[K]] struct {
	DB *DB
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
) (_ *Table[K, E], err error) {
	wrapped := make([]migrate.Migration, len(cfg.Migrations)+1)
	copy(wrapped[1:], cfg.Migrations)
	wrapped[0] = normalizeKeysMigration[K, E]()
	withDeps := migrate.AllWithAddedDeps(wrapped[1:], normalizeKeysMigrationKey)
	copy(wrapped[1:], withDeps)
	if err = Migrate(ctx, MigrateConfig{
		DB:              cfg.DB,
		Namespace:       types.Name[E](),
		Migrations:      wrapped,
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	return &Table[K, E]{DB: cfg.DB}, nil
}

// NewCreate returns a Create query builder.
func (t *Table[K, E]) NewCreate() Create[K, E] {
	return NewCreate[K, E]()
}

// NewRetrieve returns a Retrieve query builder.
func (t *Table[K, E]) NewRetrieve() Retrieve[K, E] {
	return NewRetrieve[K, E]()
}

// NewUpdate returns an Update query builder.
func (t *Table[K, E]) NewUpdate() Update[K, E] {
	return NewUpdate[K, E]()
}

// NewDelete returns a Delete query builder.
func (t *Table[K, E]) NewDelete() Delete[K, E] {
	return NewDelete[K, E]()
}

// OpenNexter opens a new Nexter over entries in the table using the DB's codec for
// decoding.
func (t *Table[K, E]) OpenNexter(ctx context.Context) (iter.Seq[E], io.Closer, error) {
	return WrapReader[K, E](t.DB).OpenNexter(ctx)
}

var normalizeKeysMigrationKey = "normalize_keys"

func normalizeKeysMigration[K Key, E Entry[K]]() migrate.Migration {
	return NewMigration(normalizeKeysMigrationKey, func(ctx context.Context, tx Tx, _ alamos.Instrumentation) (err error) {
		kc := newKeyCodec[K, E]()
		oldPrefix, err := msgpack.Codec.Encode(ctx, types.Name[E]())
		if err != nil {
			return err
		}
		if string(oldPrefix) == string(kc.prefix) {
			return nil
		}
		itr, err := tx.OpenIterator(kv.IterPrefix(oldPrefix))
		if err != nil {
			return err
		}
		defer func() {
			err = errors.Combine(err, itr.Close())
		}()
		for itr.First(); itr.Valid(); itr.Next() {
			rawValue := itr.Value()
			var entry E
			if err = tx.Decode(ctx, rawValue, &entry); err != nil {
				return errors.Wrapf(err, "normalize_keys: failed to decode entry at old prefix key %x", itr.Key())
			}
			if err = tx.Delete(ctx, itr.Key()); err != nil {
				return err
			}
			if err = tx.Set(ctx, kc.encode(entry.GorpKey()), rawValue); err != nil {
				return err
			}
		}
		return nil
	})
}

func migrationKey(namespace string) []byte {
	return []byte(migrationVersionPrefix + namespace)
}

// readAppliedMigrations reads the set of applied migration names from the KV
// store. Names are stored as a newline-delimited string.
func readAppliedMigrations(
	ctx context.Context,
	tx Tx,
	namespace string,
) (applied set.Set[string], err error) {
	key := migrationKey(namespace)
	b, closer, getErr := tx.Get(ctx, key)
	if getErr != nil {
		if errors.Is(getErr, query.ErrNotFound) {
			return make(set.Set[string]), nil
		}
		return nil, getErr
	}
	defer func() {
		err = errors.Combine(err, closer.Close())
	}()
	var names []string
	if err := json.Unmarshal(b, &names); err != nil {
		return nil, err
	}
	applied = make(set.Set[string], len(names))
	for _, name := range names {
		applied.Add(name)
	}
	return applied, nil
}

// writeAppliedMigrations persists the set of applied migration names as a
// JSON string array.
func writeAppliedMigrations(
	ctx context.Context,
	tx Tx,
	namespace string,
	applied set.Set[string],
) error {
	key := migrationKey(namespace)
	names := applied.ToSlice()
	sort.Strings(names)
	b, err := json.Marshal(names)
	if err != nil {
		return err
	}
	return tx.Set(ctx, key, b)
}
