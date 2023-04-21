// Copyright 2023 Synnax Labs, Inc.
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
// It also provides additional utilites that leverage these interfaces to extend a key-value
// store's functilonality.
package kv

import (
	"context"
	"io"

	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/observe"
)

// NotFound is returned when a key is not found in the DB.
var NotFound = errors.New("[kv] - not found")

// Reader is a readable key-value store.
type Reader interface {
	// Get returns the value for the given key.
	Get(ctx context.Context, key []byte, opts ...interface{}) ([]byte, error)
	// OpenIterator returns an Iterator using the given IteratorOptions.
	OpenIterator(opts IteratorOptions) Iterator
}

// Writer as a writable key-value store.
type Writer interface {
	// Set sets the value for the given key. It is safe to modify the contents of key
	// and value after Set returns.
	Set(ctx context.Context, key, value []byte, opts ...interface{}) error
	// Delete removes the value for the given key. It is safe to modify the contents
	// of key after Delete returns.
	Delete(ctx context.Context, key []byte, opts ...interface{}) error
}

// ReadWriter is a read-writeable key-value store.
type ReadWriter interface {
	Reader
	Writer
}

/** Atomic is a key-value store that supports executing atomic transations */
type Atomic interface {
	// OpenTx opens a new transaction on the DB.
	OpenTx() Tx
}

// Tx is a transaction of ordered key-value operations on a DB that are committed atomically.
// Tx implements the Reader interface,and will read key-value pairs from both the Tx and
// underlying DB. A transaction must be committed for its changes to be persisted.
type Tx interface {
	ReadWriter
	// NewReader returns an TxReader that can be used to iterate over the operations
	// executed in the transaction.
	NewReader() TxReader
	// Commit persists the batch to the underlying DB. Commit will panic if called
	// more than once.
	Commit(ctx context.Context, opts ...interface{}) error
	// Close closes the transaction. If the transaction has not been committed, all
	// changes will be discarded.
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
}

// OperationVariant is an enum that indicates the type of Operation executed.
type OperationVariant uint8

const (
	// SetOperation indicates that the operation is a set operation.
	SetOperation OperationVariant = iota + 1
	// DeleteOperation indicates that the operation is a delete operation.
	DeleteOperation
)

// Operation is a key-value pair. The contents of Key and Value should be considered
// read-only, and modifications to them may cause unexpected behavior.
type Operation struct {
	// Variant is the type of operation.
	Variant OperationVariant
	// Key is the key for the key-value pair.
	Key []byte
	// Value is the value for the key-value pair.
	Value []byte
}

// TxReader is used to read the operations in a transaction.
type TxReader = iter.Next[Operation]

// Observable allows the caller to observe changes to key-value pairs in the DB.
type Observable = observe.Observable[TxReader]
