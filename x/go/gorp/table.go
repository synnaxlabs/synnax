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
	"sync/atomic"

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
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/types"
	"go.uber.org/zap"
)

// TableConfig configures a Table opened via OpenTable.
type TableConfig[K Key, E Entry[K]] struct {
	// DB is the gorp DB the Table is backed by.
	DB *DB
	// Migrations is the ordered set of versioned migrations to run
	// before the Table is usable. Already-applied migrations are
	// skipped.
	Migrations []migrate.Migration
	// Indexes is the set of secondary indexes to register on this table. Each
	// index is populated at open time from the current table contents, then
	// kept in sync via the table's observer pipeline for the lifetime of the
	// table. See NewLookupIndex and NewSortedIndex for constructing index values.
	Indexes []Index[K, E]
	// Instrumentation is used for migration and populate logging.
	alamos.Instrumentation
}

// Table provides a strongly typed interface for a specific entry type within a gorp DB.
type Table[K Key, E Entry[K]] struct {
	// db is the gorp DB the Table reads and writes through.
	db *DB
	// keyPrefix is the gorp key prefix for entry type E.
	keyPrefix []byte
	// indexes is the set of secondary indexes registered on the Table.
	indexes []Index[K, E]
	// disconnectObserver releases the index observer subscription. Nil
	// when no observer was installed.
	disconnectObserver func()
	// populateDone is closed when the background populate routine exits.
	// Nil when the table has no indexes.
	populateDone <-chan struct{}
	// populateShutdown cancels the populate context and waits for the
	// populate routine to exit. Nil when the table has no indexes.
	populateShutdown io.Closer
	// populateErr is the terminal scan error, set by runPopulate before
	// it returns. Read by WaitForIndexes after populateDone is closed.
	populateErr atomic.Pointer[error]
}

// Close disconnects the table's index observer, cancels any running
// populate goroutine, waits for it to exit, and releases related
// resources.
func (t *Table[K, E]) Close() error {
	var err error
	if t.populateShutdown != nil {
		err = t.populateShutdown.Close()
		t.populateShutdown = nil
	}
	if t.disconnectObserver != nil {
		t.disconnectObserver()
		t.disconnectObserver = nil
	}
	return err
}

// WaitForIndexes blocks until the table's secondary indexes finish
// populating. Returns nil on success, the populate error on failure,
// or ctx.Err() if ctx is canceled before populate completes. Returns
// nil immediately for tables with no registered indexes.
//
// Most callers do not need WaitForIndexes: idx.Filter inside Retrieve
// blocks transparently until the index is ready and falls back to a
// sequential scan if populate fails. WaitForIndexes is useful for tests
// that observe indexed state directly via Get / GetTx, and for service
// startup paths that want to fail fast on populate failure.
func (t *Table[K, E]) WaitForIndexes(ctx context.Context) error {
	if t.populateDone == nil {
		return nil
	}
	select {
	case <-t.populateDone:
		if e := t.populateErr.Load(); e != nil {
			return *e
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// OpenTable creates or opens a table for the given entry type. It runs
// any provided versioned migrations followed by key migrations to
// ensure entries are stored under the current prefix and key encoding
// format. After migrations complete, OpenTable returns immediately;
// any secondary indexes in cfg.Indexes are populated asynchronously by
// a background goroutine that scans the table once and feeds every
// index from the same scan.
//
// idx.Filter inside Retrieve blocks the query until the index has
// finished populating. After populate completes the index serves
// queries normally. If populate fails, idx.Filter falls back to a
// sequential scan with a synthesized predicate so queries continue to
// return correct results, and the failure is logged via
// cfg.Instrumentation. WaitForIndexes lets callers synchronize
// explicitly when they need to.
func OpenTable[K Key, E Entry[K]](
	ctx context.Context,
	cfg TableConfig[K, E],
) (_ *Table[K, E], err error) {
	wrapped := make([]migrate.Migration, 0, len(cfg.Migrations)+1)
	wrapped = append(wrapped, normalizeKeysMigration[K, E]())
	wrapped = append(wrapped, cfg.Migrations...)
	if err = Migrate(ctx, MigrateConfig{
		DB:              cfg.DB,
		Namespace:       types.Name[E](),
		Migrations:      wrapped,
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		return nil, err
	}
	t := &Table[K, E]{db: cfg.DB, keyPrefix: newKeyPrefix[E](), indexes: cfg.Indexes}
	if len(cfg.Indexes) == 0 {
		return t, nil
	}
	// Acquire each index's populate lock synchronously before attaching
	// the observer. If the observer were attached first, a remote write
	// replicated through aspen could race the goroutine and apply via
	// idx.set() before populate locks mu, leaving the bulk-load path to
	// double-insert when the iterator later sees the same row.
	var (
		inserts  = make([]func(E), 0, len(cfg.Indexes))
		finishes = make([]func(error), 0, len(cfg.Indexes))
	)
	for _, idx := range cfg.Indexes {
		insert, finish := idx.populate()
		inserts = append(inserts, insert)
		finishes = append(finishes, finish)
	}
	t.disconnectObserver = attachIndexObserver[K, E](
		override.Nil[observe.Observable[kv.TxReader]](cfg.DB, cfg.DB.IndexObservable),
		cfg.DB,
		cfg.Indexes,
	)
	// Populate runs on an isolated signal context: the caller's ctx is the
	// open-operation ctx (may be short-lived, e.g. an open timeout), but
	// populate must run for the Table's full lifetime. Termination flows
	// through Table.Close().
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
	t.populateDone = sCtx.Stopped()
	t.populateShutdown = signal.NewHardShutdown(sCtx, cancel)
	sCtx.Go(
		func(ctx context.Context) error {
			t.runPopulate(ctx, cfg.Instrumentation, inserts, finishes)
			return nil
		},
		signal.WithKey("gorp_index_populate"),
	)
	return t, nil
}

// runPopulate scans the table once and feeds every registered index
// from the same scan. Populate failures are recoverable: queries fall
// back to sequential scan, and the terminal scan error is stored on
// t.populateErr and passed to each index's finish closure.
func (t *Table[K, E]) runPopulate(
	ctx context.Context,
	ins alamos.Instrumentation,
	inserts []func(E),
	finishes []func(error),
) {
	var scanErr error
	defer func() {
		if scanErr != nil {
			t.populateErr.Store(&scanErr)
			ins.L.Error(
				"gorp index populate failed; queries fall back to sequential scan",
				zap.Error(scanErr),
			)
		}
		for _, f := range finishes {
			f(scanErr)
		}
	}()
	reader := wrapReader[K, E](t.db, t.keyPrefix)
	nexter, closer, openErr := reader.OpenNexter(ctx)
	if openErr != nil {
		scanErr = openErr
		return
	}
	defer func() { scanErr = errors.Combine(scanErr, closer.Close()) }()
	for e := range nexter {
		if ctx.Err() != nil {
			scanErr = ctx.Err()
			return
		}
		for _, insert := range inserts {
			insert(e)
		}
	}
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
	return wrapReader[K, E](t.db, t.keyPrefix).OpenNexter(ctx)
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
