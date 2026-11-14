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
	"slices"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/types"
	"go.uber.org/zap"
)

// TableConfig configures a Table opened via OpenTable.
type TableConfig[K Key, E Entry[K]] struct {
	// DB is the gorp DB the Table is backed by.
	DB *DB
	// Migrations is the ordered set of versioned migrations to run before the Table is
	// usable. Already-applied migrations are skipped.
	Migrations []Migration
	// Indexes is the set of secondary indexes to register on this table. Each index is
	// populated at open time from the current table contents, then kept in sync by
	// every write to the table's entries while the table is open. See NewLookupIndex,
	// NewSortedIndex, and NewSearchIndex for constructing index values.
	Indexes []Index[K, E]
	// Name names the table for references. A Reference names its target tables by
	// this name, and two open tables on one DB cannot share it. Defaults to the name
	// of E.
	Name string
	// References are the fields of E that hold the keys of other entries. See
	// NewKeyReference and NewTargetReference.
	References []Reference[K, E]
	// OnDetach runs in the transaction of each delete that detaches a target from an
	// entry of the table, after Gorp writes the entry. It gets the entry as written. An
	// error fails the delete.
	OnDetach func(ctx context.Context, tx Tx, e E) error
	// Instrumentation is used for migration and populate logging.
	alamos.Instrumentation
}

// Table provides a strongly typed interface for a specific entry type within a Gorp DB.
type Table[K Key, E Entry[K]] struct {
	// db is the gorp DB the Table reads and writes through.
	db *DB
	// keyPrefix is the gorp key prefix for entry type E.
	keyPrefix []byte
	// indexes is the set of secondary indexes registered on the Table, references
	// included.
	indexes []Index[K, E]
	// name is the name of the table in the DB's registry.
	name string
	// references are the references registered on the Table.
	references []Reference[K, E]
	// registered is true when the table is in the DB's registry.
	registered bool
	// populateShutdown cancels the populate context and waits for the populate routine
	// to exit. Nil when the table has no indexes.
	populateShutdown io.Closer
	// onDetach is TableConfig.OnDetach.
	onDetach func(ctx context.Context, tx Tx, e E) error
}

// Close cancels any running populate goroutine, waits for it to exit, and releases
// related resources.
func (t *Table[K, E]) Close() error {
	var err error
	t.unregister()
	if len(t.indexes) > 0 {
		t.db.registry.removeIndexed(t.keyPrefix, t)
	}
	if t.populateShutdown != nil {
		err = t.populateShutdown.Close()
		t.populateShutdown = nil
	}
	return err
}

// OpenTable creates or opens a table for the given entry type. It runs any provided
// versioned migrations followed by key migrations to ensure entries are stored under
// the current prefix and key encoding format. After migrations complete, OpenTable
// returns immediately; any secondary indexes in cfg.Indexes are populated
// asynchronously by a background goroutine that scans the table once and feeds every
// index from the same scan.
//
// idx.Filter and SortedIndex.FilterRange inside Retrieve block the query until the
// index has finished populating. After populate completes the index serves queries
// normally. If populate fails, the failure is logged via cfg.Instrumentation, and
// idx.Filter, SortedIndex.FilterRange, and Referenced fall back to a sequential scan so
// queries continue to return correct results. Every other read of a failed index
// returns ErrIndexInvalid: idx.Get, Dependents, and the reference checks of each write
// and delete.
func OpenTable[K Key, E Entry[K]](
	ctx context.Context,
	cfg TableConfig[K, E],
) (*Table[K, E], error) {
	t := &Table[K, E]{
		db:         cfg.DB,
		keyPrefix:  key.Prefix[E](),
		indexes:    slices.Clone(cfg.Indexes),
		name:       cfg.Name,
		references: cfg.References,
		onDetach:   cfg.OnDetach,
	}
	if t.name == "" {
		t.name = types.Name[E]()
	}
	// The migrations of other tables can write references to the table while its own
	// migrations run.
	if err := t.register(cfg.Name != ""); err != nil {
		return nil, err
	}
	if err := Migrate(ctx, MigrateConfig{
		DB:              cfg.DB,
		Namespace:       types.Name[E](),
		Migrations:      t.bindDangling(cfg.Migrations),
		Instrumentation: cfg.Instrumentation,
	}); err != nil {
		t.unregister()
		return nil, err
	}
	t.registerReferences()
	if len(t.indexes) == 0 {
		return t, nil
	}
	// Acquire each index's populate lock before OpenTable returns, so that no commit
	// reaches an index before the scan that would insert the same entry again.
	var (
		inserts  = make([]func(E), 0, len(t.indexes))
		finishes = make([]func(error), 0, len(t.indexes))
	)
	for _, idx := range t.indexes {
		insert, finish := idx.populate()
		inserts = append(inserts, insert)
		finishes = append(finishes, finish)
	}
	// From here on, a writer that no table binds stages its writes against the indexes.
	cfg.DB.registry.addIndexed(t.keyPrefix, t)
	// Populate runs on an isolated signal context: the caller's ctx is the
	// open-operation ctx (may be short-lived, e.g. an open timeout), but populate must
	// run for the Table's full lifetime. Termination flows through Table.Close().
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(cfg.Instrumentation))
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

// runPopulate scans the table once and feeds every registered index from the same scan.
// The terminal scan error is logged and passed to each index's finish closure.
func (t *Table[K, E]) runPopulate(
	ctx context.Context,
	ins alamos.Instrumentation,
	inserts []func(E),
	finishes []func(error),
) {
	var scanErr error
	defer func() {
		if scanErr != nil {
			ins.L.Error(
				"gorp index populate failed; filters fall back to a sequential scan, and "+
					"reference checks fail",
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

// NewCreate returns a Create query builder bound to this Table. Writes through the
// returned query are visible to a Retrieve via idx.Filter in the same tx
// (read-your-own-writes).
func (t *Table[K, E]) NewCreate() Create[K, E] {
	c := NewCreate[K, E]()
	c.keyPrefix = t.keyPrefix
	c.table = t
	return c
}

// NewUncheckedWriter returns a Writer that keeps the table's indexes current and skips
// its reference checks and delete plans. Use it only in a migration that writes before
// a target of the table's references opens.
func (t *Table[K, E]) NewUncheckedWriter(tx Tx) *Writer[K, E] {
	w := t.writer(tx, false)
	return &w
}

// NewRetrieve returns a Retrieve query builder bound to this Table.
func (t *Table[K, E]) NewRetrieve() Retrieve[K, E] {
	r := NewRetrieve[K, E]()
	r.keyPrefix = t.keyPrefix
	return r
}

// NewUpdate returns an Update query builder bound to this Table. Writes through the
// returned query are visible to a Retrieve via idx.Filter in the same tx
// (read-your-own-writes).
func (t *Table[K, E]) NewUpdate() Update[K, E] {
	u := NewUpdate[K, E]()
	u.retrieve.keyPrefix = t.keyPrefix
	u.table = t
	return u
}

// NewDelete returns a Delete query builder bound to this Table. Deletions through the
// returned query are visible to a Retrieve via idx.Filter in the same tx
// (read-your-own-writes).
func (t *Table[K, E]) NewDelete() Delete[K, E] {
	d := NewDelete[K, E]()
	d.retrieve.keyPrefix = t.keyPrefix
	d.table = t
	return d
}
