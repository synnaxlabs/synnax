// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/binary"
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

var (
	_ Tx             = (*DB)(nil)
	_ BaseReader     = (*DB)(nil)
	_ BaseWriter     = (*DB)(nil)
	_ BaseObservable = (*DB)(nil)
)

// OpenTx begins a new Tx against the DB.
func (db *DB) OpenTx() Tx { return tx{Tx: db.DB.OpenTx(), options: db.options} }

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
type Tx interface {
	kv.Tx
	Tools
}

// Context is an extension of the built-in context.Context type that adds additional
// fields useful in gorp callbacks.
type Context struct {
	context.Context
	// Tx is the transaction the query is operating under.
	Tx Tx
}

type tx struct {
	kv.Tx
	options
}

func checkForNilTx(method string, tx Tx) {
	if tx == nil {
		panic("[gorp] - nil transaction - please provide transaction to " + method)
	}
}

var _ Tx = (*tx)(nil)

// Tools provides the tools that gorp needs to translate key-value operations
// to strongly-typed requests. It doesn't provide any functionality itself,
// and is instead designed to be passed to the various other types that gorp uses.
type Tools interface{ binary.Codec }

// BaseReader is a simple extension of the kv.Reader interface that adds
// gorp-required tooling. For semantic purposes, it can be considered as
// equivalent to a kv.Reader.
type BaseReader interface {
	kv.Reader
	Tools
}

// BaseWriter is a simple extension of the kv.Writer interface that
// adds gorp-required tooling. For semantic purposes, it can be considered
// as equivalent to a kv.Writer.
type BaseWriter interface {
	kv.Writer
	Tools
}

// BaseObservable is a simple extension of the kv.Writer interface that
// adds gorp-required tooling. For semantic purposes, it can be considered
// as equivalent to a kv.Observable.
type BaseObservable interface {
	kv.Observable
	Tools
}
