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
	"slices"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
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
	// OnDetach runs in the transaction of each delete that detaches or inlines a target
	// in an entry of the table, after Gorp writes the entry. It gets the entry as
	// written. An error fails the delete.
	OnDetach func(ctx context.Context, tx Tx, e E) error
	// OnTargetSet maps the name of a table to a hook. The hook runs in the transaction
	// of each write that sets an entry of that table which an entry of this table holds
	// through a Use reference, after Gorp writes it, with the holding entry. An error
	// fails the write.
	OnTargetSet map[string]func(ctx context.Context, tx Tx, e E) error
	// Issued reports whether the table issued key k: a deleted entry's key counts, and
	// a key the table has not issued yet does not. A Weak reference may point at the
	// table only when Issued is set.
	Issued func(k K) bool
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
	// uniques are the unique indexes of the Table, which its checked writes check.
	uniques []checkedIndex[K, E]
	// registered is true when the table is in the DB's registry.
	registered bool
	// closing closes when Close starts, to stop the populate of the indexes. Nil when
	// the table has no indexes.
	closing chan struct{}
	// populated closes when the populate of the indexes returns.
	populated chan struct{}
	// onDetach is TableConfig.OnDetach.
	onDetach func(ctx context.Context, tx Tx, e E) error
	// onTargetSet is TableConfig.OnTargetSet.
	onTargetSet map[string]func(ctx context.Context, tx Tx, e E) error
	// issuedKey is TableConfig.Issued.
	issuedKey func(k K) bool
}

// Close stops the populate of the indexes of the table, waits for it to return, and
// releases related resources.
func (t *Table[K, E]) Close() error {
	t.unregister()
	if len(t.indexes) > 0 {
		t.db.registry.removeIndexed(t.keyPrefix, t)
	}
	if t.closing != nil {
		close(t.closing)
		<-t.populated
		t.closing = nil
	}
	return nil
}

// OpenTable creates or opens a table for the given entry type. It runs any provided
// versioned migrations followed by key migrations to ensure entries are stored under
// the current prefix and key encoding format. After migrations complete, OpenTable
// returns immediately; DB.Run populates any secondary indexes in cfg.Indexes from one
// scan of the table. For a table with indexes or references, OpenTable waits for DB.Run
// to start, and returns an error when ctx ends first or DB.Run has stopped.
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
		db:          cfg.DB,
		keyPrefix:   key.Prefix[E](),
		indexes:     slices.Clone(cfg.Indexes),
		name:        cfg.Name,
		references:  cfg.References,
		onDetach:    cfg.OnDetach,
		onTargetSet: cfg.OnTargetSet,
		issuedKey:   cfg.Issued,
	}
	if t.name == "" {
		t.name = types.Name[E]()
	}
	for _, idx := range cfg.Indexes {
		if u, ok := idx.(checkedIndex[K, E]); ok {
			t.uniques = append(t.uniques, u)
		}
	}
	// The migrations of other tables can write references to the table while its own
	// migrations run.
	if err := t.register(cfg.Name != "" || len(t.uniques) > 0); err != nil {
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
	if len(t.indexes)+len(t.references) == 0 {
		return t, nil
	}
	var (
		inserts   []func(E)
		finishes  []func(error)
		locked    = make(chan struct{})
		closing   = make(chan struct{})
		populated = make(chan struct{})
	)
	if err := cfg.DB.goPopulate(ctx, func(ctx context.Context) {
		defer close(populated)
		<-locked
		t.runPopulate(ctx, closing, cfg.Instrumentation, inserts, finishes)
	}); err != nil {
		t.unregister()
		return nil, err
	}
	if err := t.registerIndexes(); err != nil {
		close(closing)
		close(locked)
		<-populated
		t.unregister()
		return nil, err
	}
	// Acquire each index's populate lock before OpenTable returns, so that no commit
	// reaches an index before the scan that would insert the same entry again.
	for _, idx := range t.indexes {
		insert, finish := idx.populate()
		inserts = append(inserts, insert)
		finishes = append(finishes, finish)
	}
	// From here on, a writer that no table binds stages its writes against the indexes.
	cfg.DB.registry.addIndexed(t.keyPrefix, t)
	close(locked)
	t.closing, t.populated = closing, populated
	return t, nil
}

// runPopulate scans the table once and feeds every registered index from the same scan.
// It stops when ctx ends or closing closes. It logs the terminal scan error and passes
// it to each index's finish closure.
func (t *Table[K, E]) runPopulate(
	ctx context.Context,
	closing <-chan struct{},
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
		select {
		case <-ctx.Done():
			scanErr = ctx.Err()
			return
		case <-closing:
			scanErr = errors.New("[gorp] - table closed")
			return
		default:
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
// its reference checks, unique checks, and delete plans. Use it only in a migration
// that writes before a target of the table's references opens.
func (t *Table[K, E]) NewUncheckedWriter(tx Tx) *Writer[K, E] {
	w := t.writer(tx, 0)
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
