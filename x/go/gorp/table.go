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
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/types"
)

// TableConfig configures a Table opened via OpenTable.
type TableConfig[K Key, E Entry[K]] struct {
	DB         *DB
	Migrations []migrate.Migration
	// Indexes is the set of secondary indexes to register on this table. Each
	// index is populated at open time from the current table contents, then
	// kept in sync via the table's observer pipeline for the lifetime of the
	// table. See NewLookup and NewSorted for constructing index values.
	Indexes []Index[K, E]
	alamos.Instrumentation
}

// Table provides a strongly typed interface for a specific entry type within a gorp DB.
type Table[K Key, E Entry[K]] struct {
	DB *DB
	// keyPrefix is the gorp key prefix for entry type E.
	keyPrefix []byte
	// indexes is the set of secondary indexes registered on this Table.
	indexes []Index[K, E]
	// disconnectObserver releases the index observer subscription, or
	// nil if no observer was installed.
	disconnectObserver func()
}

// Close disconnects the table's index observer, if any, and releases related
// resources.
func (t *Table[K, E]) Close() error {
	if t.disconnectObserver != nil {
		t.disconnectObserver()
		t.disconnectObserver = nil
	}
	return nil
}

// OpenTable creates or opens a table for the given entry type. It runs any provided
// versioned migrations followed by key migrations to ensure entries are stored under
// the current prefix and key encoding format. After migrations complete, any
// secondary indexes in cfg.Indexes are populated by scanning the full table and
// kept in sync via the observer pipeline.
func OpenTable[K Key, E Entry[K]](
	ctx context.Context,
	cfg TableConfig[K, E],
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
	t := &Table[K, E]{
		DB:        cfg.DB,
		keyPrefix: newKeyPrefix[E](),
		indexes:   cfg.Indexes,
	}
	if len(cfg.Indexes) > 0 {
		if err = populateIndexes[K, E](ctx, cfg.DB, t.keyPrefix, cfg.Indexes); err != nil {
			return nil, err
		}
		t.disconnectObserver = attachIndexObserver[K, E](
			override.Nil[observe.Observable[kv.TxReader]](cfg.DB, cfg.DB.IndexObservable),
			cfg.DB,
			cfg.Indexes,
		)
	}
	return t, nil
}

// populateIndexes populates every registered index from the current
// table contents in a single sequential scan. If any index fails to
// start, finish runs on every index that did start so populate-phase
// locks are released regardless of which step failed.
func populateIndexes[K Key, E Entry[K]](
	ctx context.Context,
	db *DB,
	keyPrefix []byte,
	indexes []Index[K, E],
) (err error) {
	inserts := make([]func(E), 0, len(indexes))
	finishes := make([]func(), 0, len(indexes))
	defer func() {
		for _, f := range finishes {
			f()
		}
	}()
	for _, idx := range indexes {
		insert, finish, startErr := idx.populate()
		if startErr != nil {
			return startErr
		}
		inserts = append(inserts, insert)
		finishes = append(finishes, finish)
	}
	reader := wrapReader[K, E](db, keyPrefix)
	nexter, closer, nexterErr := reader.OpenNexter(ctx)
	if nexterErr != nil {
		return nexterErr
	}
	defer func() { _ = closer.Close() }()
	for e := range nexter {
		for _, insert := range inserts {
			insert(e)
		}
	}
	return nil
}

// attachIndexObserver subscribes to src and propagates every set and
// delete change into every registered index. The returned function
// unregisters the subscription.
func attachIndexObserver[K Key, E Entry[K]](
	src observe.Observable[kv.TxReader],
	db *DB,
	indexes []Index[K, E],
) func() {
	return newObservable[K, E](src, db).
		OnChange(func(_ context.Context, changes iter.Seq[change.Change[K, E]]) {
			for ch := range changes {
				switch ch.Variant {
				case change.VariantSet:
					for _, idx := range indexes {
						idx.set(ch.Value)
					}
				case change.VariantDelete:
					for _, idx := range indexes {
						idx.delete(ch.Key)
					}
				}
			}
		})
}

// NewCreate returns a Create query builder bound to this Table. Writes
// through the returned query are visible to a Retrieve via idx.Filter
// in the same tx (read-your-own-writes).
func (t *Table[K, E]) NewCreate() Create[K, E] {
	c := NewCreate[K, E]()
	c.keyPrefix = t.keyPrefix
	c.indexes = t.indexes
	return c
}

// NewRetrieve returns a Retrieve query builder bound to this Table.
func (t *Table[K, E]) NewRetrieve() Retrieve[K, E] {
	r := NewRetrieve[K, E]()
	r.keyPrefix = t.keyPrefix
	return r
}

// NewUpdate returns an Update query builder bound to this Table. Writes
// through the returned query are visible to a Retrieve via idx.Filter
// in the same tx (read-your-own-writes).
func (t *Table[K, E]) NewUpdate() Update[K, E] {
	u := NewUpdate[K, E]()
	u.retrieve.keyPrefix = t.keyPrefix
	u.indexes = t.indexes
	return u
}

// NewDelete returns a Delete query builder bound to this Table.
// Deletions through the returned query are visible to a Retrieve via
// idx.Filter in the same tx (read-your-own-writes).
func (t *Table[K, E]) NewDelete() Delete[K, E] {
	d := NewDelete[K, E]()
	d.retrieve.keyPrefix = t.keyPrefix
	d.indexes = t.indexes
	return d
}

// OpenNexter opens a new Nexter over entries in the table using the DB's codec for
// decoding.
func (t *Table[K, E]) OpenNexter(ctx context.Context) (iter.Seq[E], io.Closer, error) {
	reader := wrapReader[K, E](t.DB, t.keyPrefix)
	return reader.OpenNexter(ctx)
}

var normalizeKeysMigrationKey = "normalize_keys"

func normalizeKeysMigration[K Key, E Entry[K]]() migrate.Migration {
	return NewMigration(normalizeKeysMigrationKey, func(ctx context.Context, tx Tx, _ alamos.Instrumentation) (err error) {
		kc := newKeyCodec[K, E](nil)
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
	names := applied.Slice()
	sort.Strings(names)
	b, err := json.Marshal(names)
	if err != nil {
		return err
	}
	return tx.Set(ctx, key, b)
}
