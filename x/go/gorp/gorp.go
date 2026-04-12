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
	"sync"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
)

// Wrap wraps the provided key-value database in a DB.
func Wrap(kv kv.DB, opts ...Option) *DB { return &DB{DB: kv, options: newOptions(opts)} }

// DB is a wrapper around a kv.DB that queries can be executed against. DB implements
// the transaction (Tx) interface. Using a DB as a Tx will execute the query
// directly against the underlying key-value store, outside the isolated context of
// a transaction.
type DB struct {
	kv.DB
	options
}

var _ Tx = (*DB)(nil)

// OpenTx begins a new Tx against the DB.
func (db *DB) OpenTx() Tx { return &tx{Tx: db.DB.OpenTx(), options: db.options} }

// txIdentity implements Tx for DB. Because DB represents the "no transaction"
// mode (queries commit directly), it owns no per-tx state and returns nil.
// Subsystems that scope state to a transaction (e.g. secondary indexes
// maintaining a delta overlay) treat a nil identity as the committed-only
// path and skip staging.
func (db *DB) txIdentity() *txState { return nil }

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
	return
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
func OverrideTx(base, override Tx) Tx { return lo.Ternary(override != nil, override, base) }

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
// Tx is sealed: the unexported txIdentity method prevents external packages
// from providing their own implementations. This is deliberate — subsystems
// like secondary indexes scope per-tx state off the identity handle, and
// allowing external implementations would either require them to implement
// the handle or risk silently breaking the overlay contract.
type Tx interface {
	kv.Tx
	encoding.Codec
	// txIdentity returns the per-tx state handle used by subsystems that
	// scope ephemeral state to a transaction's lifetime. Concrete tx
	// implementations return a stable *txState that remains valid until
	// Commit or Close runs cleanups. DB returns nil, signaling "no
	// transaction" semantics — callers treat nil as committed-only and
	// skip their per-tx logic.
	txIdentity() *txState
}

// Context is an extension of the built-in context.Context type that adds additional
// fields useful in gorp callbacks.
type Context struct {
	context.Context
	// Tx is the transaction the query is operating under.
	Tx Tx
}

// txState owns per-tx ephemeral state registered by subsystems (like
// secondary indexes) that scope work to a single transaction's lifetime.
// The pointer returned from Tx.txIdentity is used as a concrete map key by
// those subsystems, so its lifetime must span from OpenTx through Commit
// or Close. runCleanups is invoked from *tx.Commit and *tx.Close after
// delegating to the underlying kv.Tx, giving commit-time observers (like
// the index observer wired in attachIndexObserver) a chance to fire
// against the global state before the per-tx overlay is dropped.
type txState struct {
	mu       sync.Mutex
	cleanups []func()
}

// onCleanup registers a hook to run when the owning transaction commits
// or closes. Safe to call from any goroutine that holds a reference to
// the state handle.
func (s *txState) onCleanup(fn func()) {
	s.mu.Lock()
	s.cleanups = append(s.cleanups, fn)
	s.mu.Unlock()
}

// runCleanups invokes every registered hook exactly once in registration
// order and then clears the list. Hooks themselves must not re-enter
// runCleanups on the same state.
func (s *txState) runCleanups() {
	s.mu.Lock()
	hooks := s.cleanups
	s.cleanups = nil
	s.mu.Unlock()
	for _, h := range hooks {
		h()
	}
}

type tx struct {
	kv.Tx
	options
	state txState
}

var _ Tx = (*tx)(nil)

// txIdentity returns this tx's state handle. The pointer is stable for
// the lifetime of the tx and is used as a map key by per-tx state
// registries (e.g. index delta overlays).
func (t *tx) txIdentity() *txState { return &t.state }

// Commit delegates to the embedded kv.Tx and then runs any cleanup hooks
// registered via state.onCleanup. Cleanups fire after the underlying
// commit so that commit-time observers on the DB (like the secondary
// index observer) have already updated global state before per-tx
// overlays are dropped.
func (t *tx) Commit(ctx context.Context, opts ...any) error {
	err := t.Tx.Commit(ctx, opts...)
	t.state.runCleanups()
	return err
}

// Close delegates to the embedded kv.Tx and then runs cleanups. For an
// uncommitted tx this discards the per-tx overlay without ever touching
// the global state.
func (t *tx) Close() error {
	err := t.Tx.Close()
	t.state.runCleanups()
	return err
}

func checkForNilTx(method string, tx Tx) {
	if tx == nil {
		panic("[gorp] - nil transaction - please provide transaction to " + method)
	}
}
