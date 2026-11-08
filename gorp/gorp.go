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
	"sync"
	"sync/atomic"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/observe"
	"go.uber.org/zap"
)

// Wrap wraps the provided key-value database in a DB.
func Wrap(
	kv kv.DB,
	opts ...Option,
) *DB {
	return &DB{
		DB:       kv,
		options:  newOptions(opts),
		registry: newRegistry(),
		delivery: delivery{observer: observe.New[Commit]()},
	}
}

// DB is a wrapper around a kv.DB that queries can be executed against. DB implements
// the transaction (Tx) interface. A query that writes through a DB used as a Tx runs in
// a transaction of its own, and a read through it sees committed state only.
//
// Transactions of one DB run one at a time: each holds the lock of the DB alone from
// OpenTx until it commits or closes. WithSnapshot holds the lock shared, so snapshots
// run at the same time as each other but never during a transaction.
type DB struct {
	// DB is the underlying key-value store.
	kv.DB
	// options carries the encoding/instrumentation configuration applied
	// to every Tx opened from this DB.
	options
	// registry holds the tables that take part in references.
	registry *registry
	// lock is held alone by each open transaction and shared by each snapshot.
	lock sync.RWMutex
	// delivery delivers each commit to the readers of Observe.
	delivery delivery
}

var _ Tx = (*DB)(nil)

// lockWaitWarning is the wait for the lock past which OpenTx and WithSnapshot log a
// warning.
const lockWaitWarning = 500 * time.Millisecond

// OpenTx begins a new Tx against the DB. It blocks until no other Tx and no snapshot of
// the DB is open. A goroutine that opens a second Tx, or writes through the DB, while
// its first Tx is open deadlocks.
func (db *DB) OpenTx() Tx {
	start := time.Now()
	db.lock.Lock()
	if wait := time.Since(start); wait > lockWaitWarning {
		db.Instrumentation.L.Warn(
			"gorp transaction waited long for the write lock",
			zap.Duration("wait", wait),
		)
	}
	return &tx{Tx: db.DB.OpenTx(), options: db.options, db: db, unlock: db.lock.Unlock}
}

// Set sets key to value in a transaction of its own.
func (db *DB) Set(ctx context.Context, key, value []byte) error {
	return db.WithTx(ctx, func(tx Tx) error { return tx.Set(ctx, key, value) })
}

// Delete deletes key in a transaction of its own.
func (db *DB) Delete(ctx context.Context, key []byte) error {
	return db.WithTx(ctx, func(tx Tx) error { return tx.Delete(ctx, key) })
}

// inTx runs f against tx. When tx is a DB, f runs in a transaction of its own, so its
// reads and writes hold the write lock together.
func inTx(ctx context.Context, tx Tx, f func(tx Tx) error) error {
	if db, ok := tx.(*DB); ok {
		return db.WithTx(ctx, f)
	}
	return f(tx)
}

// txIdentity returns nil. DB has no per-tx state because operations against
// it commit directly.
func (db *DB) txIdentity() *txState { return nil }

// snapshot is the Snapshot that WithSnapshot passes to its callback. It has only read
// methods, so the callback cannot write through it.
type snapshot struct {
	kv.Reader
	encoding.Decoder
}

// txIdentity returns nil, because a snapshot stages no writes.
func (snapshot) txIdentity() *txState { return nil }

// WithTx executes the callback within the provided transaction Tx. If the callback
// returns an error, the transaction is aborted, no writes are committed, and the
// error is returned. If the callback returns nil, the transaction is committed.
func (db *DB) WithTx(ctx context.Context, f func(tx Tx) error) (err error) {
	txn := db.OpenTx()
	defer func() {
		err = errors.Combine(err, txn.Close())
	}()
	if err = f(txn); err == nil {
		err = txn.Commit(ctx)
	}
	return err
}

// WithSnapshot runs f with a Snapshot of the DB and returns the error of f. No
// transaction of the DB opens while f runs, so the entries and indexes that f reads
// agree. Snapshots run at the same time as each other. f must not open a transaction,
// write through the DB, or call WithSnapshot, since each can deadlock.
func (db *DB) WithSnapshot(_ context.Context, f func(Snapshot) error) error {
	start := time.Now()
	db.lock.RLock()
	defer db.lock.RUnlock()
	if wait := time.Since(start); wait > lockWaitWarning {
		db.Instrumentation.L.Warn(
			"gorp snapshot waited long for the read lock",
			zap.Duration("wait", wait),
		)
	}
	return f(snapshot{Reader: db.DB, Decoder: db.Codec})
}

// OverrideTx replaces the given Tx with the DB if the given Tx is nil. This
// method is useful for allowing a caller to execute directly against the underlying
// DB if they choose to do so, or to execute against a transaction if they provide one.
func (db *DB) OverrideTx(override Tx) Tx { return OverrideTx(db, override) }

// Close closes the underlying DB. Although embedded into the DB struct, Jetbrains
// Goland complains of an ambiguous reference to the Close method if it is not
// explicitly defined.
func (db *DB) Close() error { return db.DB.Close() }

// KV returns the underlying key-value storage backing the DB.
func (db *DB) KV() kv.DB { return db.DB }

// OverrideTx returns the override transaction if it is not nil. Otherwise,
// it returns the base transaction.
func OverrideTx(
	base, override Tx,
) Tx {
	return lo.Ternary(override != nil, override, base)
}

// OverrideSnapshot returns the override snapshot if it is not nil. Otherwise, it
// returns the base snapshot.
func OverrideSnapshot(base, override Snapshot) Snapshot {
	return lo.Ternary(override != nil, override, base)
}

// Tx extends the kv.Tx interface to provide gorp-required utilities for
// executing a strongly-typed, atomic transaction against a DB. To open a
// new transaction, call gorp.OpenTx and pass it to the required queries
// or readers/writers.
//
// DB itself implements the Tx interface, meaning it can be used as a
// transaction that directly commits its operations in a non-atomic manner. This is
// ideal for allowing a caller to execute operations within an atomic transaction
// if they desire to do so, or simply use the DB directly otherwise.
//
// A Tx is a Snapshot that also sees its own writes. Tx is sealed through Snapshot.
type Tx interface {
	Snapshot
	kv.Tx
	encoding.Codec
}

// Snapshot is the read interface of a DB. DB.WithSnapshot passes a Snapshot of
// committed state.
//
// Snapshot is sealed: the unexported txIdentity method prevents external packages from
// providing their own implementations.
type Snapshot interface {
	kv.Reader
	encoding.Decoder
	// txIdentity returns a state handle scoped to this transaction's lifetime. The
	// handle is stable from open through Commit or Close, so callers may use the
	// pointer as a map key. A nil return means the receiver is not a real transaction
	// (e.g. a DB used directly or a snapshot), and callers should treat the operation
	// as already committed.
	txIdentity() *txState
}

// Context is an extension of the built-in context.Context type that adds additional
// fields useful in gorp callbacks.
type Context struct {
	// Context is the underlying request-scoped context.
	context.Context
	// Snapshot is the snapshot the query reads through. Under a Tx, it is the Tx, so a
	// read through it sees the writes of the transaction.
	Snapshot Snapshot
}

// newContext returns the Context of a query that reads through snap.
func newContext(ctx context.Context, snap Snapshot) Context {
	return Context{Context: ctx, Snapshot: snap}
}

// txState owns per-tx ephemeral state and runs cleanups when the owning
// transaction commits or closes. The pointer is stable for the lifetime
// of the transaction, so callers can use it as a map key. Cleanups
// receive committed=true if Commit succeeded and committed=false
// otherwise; they fire after the underlying commit attempt completes and
// before the transaction releases the write lock.
type txState struct {
	// mu guards cleanups.
	mu sync.Mutex
	// cleanups is the list of hooks to invoke when the owning tx
	// commits or closes, in registration order.
	cleanups []func(committed bool)
	// committing is true during a commit attempt.
	committing atomic.Bool
	// changes is what the typed writes through the transaction changed. Only the
	// goroutine that writes through the transaction touches it.
	changes changeList
	// attached holds the values of each Attachment. Only the goroutine that writes
	// through the transaction touches it.
	attached []attached
}

// Committing reports whether a commit attempt is in flight.
func (s *txState) Committing() bool { return s.committing.Load() }

// OnCleanup registers a hook to run when the owning transaction commits
// or closes. The hook is invoked with committed=true if the owning tx
// reached a successful Commit, and committed=false otherwise (Close
// without prior Commit, or a failed Commit). Safe to call from any
// goroutine that holds a reference to the state handle.
func (s *txState) OnCleanup(fn func(committed bool)) {
	s.mu.Lock()
	s.cleanups = append(s.cleanups, fn)
	s.mu.Unlock()
}

// runCleanups invokes every registered hook exactly once in registration
// order, passing committed through to each, and then clears the list.
// Hooks themselves must not re-enter runCleanups on the same state.
func (s *txState) runCleanups(committed bool) {
	s.mu.Lock()
	hooks := s.cleanups
	s.cleanups = nil
	s.mu.Unlock()
	for _, h := range hooks {
		h(committed)
	}
}

// tx is the concrete Tx returned by DB.OpenTx.
type tx struct {
	// Tx is the underlying key-value transaction.
	kv.Tx
	// options carries the encoding/instrumentation configuration
	// inherited from the parent DB.
	options
	// state is the per-tx ephemeral state used to register and run
	// cleanups (e.g. index delta flushes).
	state txState
	// unlock releases the write lock of the DB. It is nil once the transaction ends.
	unlock func()
	// db is the DB that opened the transaction.
	db *DB
	// ticket is the place of the commit in the delivery order of db.
	ticket ticket
}

// txDB returns the DB of tx, or nil when tx is neither a DB nor one of its
// transactions.
func txDB(t Tx) *DB {
	switch t := t.(type) {
	case *DB:
		return t
	case *tx:
		return t.db
	}
	return nil
}

var _ Tx = (*tx)(nil)

// txIdentity returns this tx's state handle.
func (t *tx) txIdentity() *txState { return &t.state }

// Commit commits the transaction and releases the write lock. Hooks registered via
// state.OnCleanup run before the release, with committed=true on success and
// committed=false otherwise. The release comes before the store notifies observers,
// because an observer can open a transaction. Then the readers of DB.Observe run.
func (t *tx) Commit(ctx context.Context, opts ...any) error {
	t.state.committing.Store(true)
	applied := kv.OnApplied(func() { t.end(true) })
	err := t.Tx.Commit(ctx, append(slices.Clip(opts), applied)...)
	t.end(err == nil)
	t.state.committing.Store(false)
	t.db.delivery.run(ctx, t.ticket, &t.state)
	return err
}

// Close closes the transaction. When it has not committed, hooks registered via
// state.OnCleanup run with committed=false, and the transaction releases the write
// lock.
func (t *tx) Close() error {
	err := t.Tx.Close()
	t.end(false)
	return err
}

// end runs the cleanups of the transaction, numbers a commit that changed a record or
// carries a value, and releases the write lock. Calls after the first do nothing.
func (t *tx) end(committed bool) {
	if t.unlock == nil {
		return
	}
	t.state.runCleanups(committed)
	if committed && (len(t.state.attached) > 0 || t.state.changes.changed()) {
		t.ticket = t.db.delivery.take()
	}
	t.unlock()
	t.unlock = nil
}

func checkForNilTx(method string, snap Snapshot) {
	if snap == nil {
		panic("[gorp] - nil transaction - please provide transaction to " + method)
	}
}
