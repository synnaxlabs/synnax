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

// txIdentity returns nil. DB has no per-tx state because operations against
// it commit directly.
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
// from providing their own implementations.
type Tx interface {
	kv.Tx
	encoding.Codec
	// txIdentity returns a state handle scoped to this transaction's
	// lifetime. The handle is stable from open through Commit or Close,
	// so callers may use the pointer as a map key. A nil return means
	// the receiver is not a real transaction (e.g. a DB used directly),
	// and callers should treat the operation as already committed.
	txIdentity() *txState
}

// Context is an extension of the built-in context.Context type that adds additional
// fields useful in gorp callbacks.
type Context struct {
	context.Context
	// Tx is the transaction the query is operating under.
	Tx Tx
}

// txState owns per-tx ephemeral state and runs cleanups when the owning
// transaction commits or closes. The pointer is stable for the lifetime
// of the transaction, so callers can use it as a map key. Cleanups
// receive committed=true if Commit succeeded and committed=false
// otherwise; they fire after the underlying commit attempt completes.
type txState struct {
	mu       sync.Mutex
	cleanups []func(committed bool)
}

// onCleanup registers a hook to run when the owning transaction commits
// or closes. The hook is invoked with committed=true if the owning tx
// reached a successful Commit, and committed=false otherwise (Close
// without prior Commit, or a failed Commit). Safe to call from any
// goroutine that holds a reference to the state handle.
//
//nolint:unused
func (s *txState) onCleanup(fn func(committed bool)) {
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

type tx struct {
	kv.Tx
	options
	state txState
}

var _ Tx = (*tx)(nil)

// txIdentity returns this tx's state handle.
func (t *tx) txIdentity() *txState { return &t.state }

// Commit commits the transaction. Hooks registered via state.onCleanup
// run after the commit attempt completes, with committed=true on
// success and committed=false otherwise.
func (t *tx) Commit(ctx context.Context, opts ...any) error {
	err := t.Tx.Commit(ctx, opts...)
	t.state.runCleanups(err == nil)
	return err
}

// Close closes the transaction. Hooks registered via state.onCleanup
// run with committed=false.
func (t *tx) Close() error {
	err := t.Tx.Close()
	t.state.runCleanups(false)
	return err
}

func checkForNilTx(method string, tx Tx) {
	if tx == nil {
		panic("[gorp] - nil transaction - please provide transaction to " + method)
	}
}
