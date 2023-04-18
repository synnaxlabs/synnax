// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package kv defines a general interface for a key-value store that provides support for get/set/delete operations
// as well as basic read-iteration. This package should be used as a boundary for separating an application from a
// specific storage implementation.
//
// For a general implementation of DB, see the pebblekv package.
// For an in-memory implementation of DB, see the memkv package.
package kv

import (
	"context"
	"github.com/cockroachdb/pebble"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/observe"
	"io"
)

// NotFound is returned when a key is not found in the DB store.
var NotFound = pebble.ErrNotFound

// Reader is a readable key-value store.
type Reader interface {
	// Get returns the value for the given key.
	Get(ctx context.Context, key []byte, opts ...interface{}) ([]byte, error)
	// OpenIterator returns an Iterator using the given IteratorOptions.
	OpenIterator(opts IteratorOptions) Iterator
}

// Writer is an ordered collection of key-value operations on the DB. Writer implements
// the Reader interface, and will read key-value pairs from both the Writer and underlying DB.
// A batch must be committed for its changes to be persisted.
type Writer interface {
	// Set sets the value for the given key. It is safe to modify the contents of key
	// and value after Set returns.
	Set(ctx context.Context, key, value []byte, opts ...interface{}) error
	// Delete removes the value for the given key. It is safe to modify the contents
	// of key after Delete returns.
	Delete(ctx context.Context, key []byte, opts ...interface{}) error
}

type ReadWriter interface {
	Reader
	Writer
}

type TxnFactory interface {
	// OpenTx opens a new transaction on the DB.
	OpenTx() Tx
}

// Tx is a transaction on the DB.Tx implements the Reader interface, and will read
// key-value pairs from both the Tx and underlying DB. A transaction must be committed
// for its changes to be persisted.
type Tx interface {
	ReadWriter
	// Commit persists the batch to the underlying DB. Commit will panic if called
	// more than once.
	Commit(ctx context.Context, opts ...interface{}) error
	// Close closes the transaction. If the transaction has not been committed, all
	// changes will be discarded.
	Close() error
}

// DB represents a general key-value store.
type DB interface {
	Writer
	Reader
	TxnFactory
	alamos.ReportProvider
	io.Closer
}

// Pair is a key-value pair.
type Pair struct {
	// Key is the key for the key-value pair.
	Key []byte
	// Value is the value for the key-value pair.
	Value []byte
}

// Observable allows the caller to observe changes to key-value pairs in the DB.
type Observable = observe.Observable[[]Pair]
