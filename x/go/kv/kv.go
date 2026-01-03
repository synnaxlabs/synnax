// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package kv defines a general interface for a key-value store that provides support
// for get/set/delete operations as well as basic read-iteration. This package should
// be used as a boundary for separating an application from a specific storage implementation.
//
// It also provides additional utilities that leverage these interfaces to extend a key-value
// store's functionality.
package kv

import (
	"context"
	"io"
	"iter"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
)

// NotFound is returned when a key is not found in the DB.
var NotFound = errors.New("[kv] - not found")

// Reader is a readable key-value store.
type Reader interface {
	// Get returns the value for the given key.
	Get(
		ctx context.Context,
		key []byte,
		opts ...any,
	) ([]byte, io.Closer, error)
	// OpenIterator returns an Iterator using the given IteratorOptions.
	OpenIterator(opts IteratorOptions) (Iterator, error)
}

// Writer as a writable key-value store.
type Writer interface {
	// Set sets the value for the given key. It is safe to modify the contents of key
	// and value after Set returns.
	Set(ctx context.Context, key, value []byte, opts ...any) error
	// Delete removes the value for the given key. It is safe to modify the contents
	// of key after Delete returns.
	Delete(ctx context.Context, key []byte, opts ...any) error
}

// ReadWriter is a read-writeable key-value store.
type ReadWriter interface {
	Reader
	Writer
}

// Atomic is a key-value store that supports executing atomic transactions.
type Atomic interface {
	// OpenTx opens a new transaction on the DB.
	OpenTx() Tx
}

// Tx is a transaction of ordered key-value operations on a DB that are committed
// atomically. Tx implements the Reader interface,and will read key-value pairs from
// both the Tx and underlying DB. A transaction must be committed for its changes to
// be persisted. Tx is NOT safe for concurrent use, so the caller must implement their
// own synchronization logic if they wish to use a Tx concurrently.
type Tx interface {
	ReadWriter
	// NewReader returns an TxReader that can be used to iterate over the operations
	// executed in the transaction.
	NewReader() TxReader
	// Commit persists the batch to the underlying DB. Commit will panic if called
	// more than once.
	Commit(ctx context.Context, opts ...any) error
	// Close closes the transaction. It is necessary to Close the transaction after use,
	// even if Commit has been called. Failure to do so may result in resource leaks.
	// If the transaction has not been committed, all changes will be discarded. After
	// Close has been called, it is NOT safe to call any other methods on the Tx.
	Close() error
}

// DB represents a general key-value store.
type DB interface {
	// Tx allows the DB to behave as a transaction, although all operations are directly
	// executed without atomic guarantees.
	Tx
	Atomic
	Observable
	alamos.ReportProvider
	io.Closer
	Size() telem.Size
}

// Change represents a change to a key-value pair. The contents of Name and Value
// should be considered read-only, and modifications to them may cause unexpected
// behavior.
type Change = change.Change[[]byte, []byte]

// TxReader is used to read the operations in a transaction.
type TxReader = iter.Seq[Change]

// Observable allows the caller to observe changes to key-value pairs in the DB.
type Observable = observe.Observable[TxReader]

// WithTx executes a function with a transaction on the given DB. If the function
// returns an error, the transaction will be rolled back. If the function returns
// nil, the transaction will be committed.
func WithTx(ctx context.Context, db DB, f func(tx Tx) error) (err error) {
	txn := db.OpenTx()
	defer func() {
		err = errors.Combine(err, txn.Close())
	}()
	if err = f(txn); err == nil {
		err = txn.Commit(ctx)
	}
	return
}
