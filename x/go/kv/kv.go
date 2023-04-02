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
	"io"
)

// NotFound is returned when a key is not found in the DB store.
var NotFound = pebble.ErrNotFound

type IterValidityState = pebble.IterValidityState

// Reader is a readable key-value store.
type Reader interface {
	// Get returns the value for the given key.
	Get(ctx context.Context, key []byte, opts ...interface{}) ([]byte, error)
	// NewIterator returns an Iterator using the given IteratorOptions.
	NewIterator(ctx context.Context, opts IteratorOptions) Iterator
}

// Writer is a writeable key-value store.
type Writer interface {
	// Set sets the value for the given key. It is safe to modify the contents of key
	// and value after Set returns.
	Set(ctx context.Context, key []byte, value []byte, opts ...interface{}) error
	// Delete removes the value for the given key. It is safe to modify the contents
	// of key after Delete returns.
	Delete(ctx context.Context, key []byte) error
}

type BatchWriter interface {
	// NewBatch returns a read-write batch. Any reads on the batch will read both from
	// the batch and the DB. If the batch is committed it will be applied to the DB.
	NewBatch() Batch
}

// DB represents a general key-value store.
type DB interface {
	Writer
	BatchWriter
	Reader
	io.Closer
	alamos.Reporter
}
