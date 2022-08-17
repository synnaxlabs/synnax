// Package kv defines a general interface for a key-value store that provides support for get/set/delete operations
// as well as basic read-iteration. This package should be used as a boundary for separating an application from a
// specific storage implementation.
//
// For a general implementation of DB, see the pebblekv package.
// For an in-memory implementation of DB, see the memkv package.
//
package kv

import (
	"fmt"
	"github.com/cockroachdb/pebble"
)

// NotFound is returned when a key is not found in the DB store.
var NotFound = pebble.ErrNotFound

type IterValidityState = pebble.IterValidityState

// Reader is a readable key-value store.
type Reader interface {
	// Get returns the value for the given key.
	Get(key []byte, opts ...interface{}) ([]byte, error)
	// NewIterator returns an Iterator using the given IteratorOptions.
	NewIterator(opts IteratorOptions) Iterator
}

// Writer is a writeable key-value store.
type Writer interface {
	// Set sets the value for the given key. It is safe to modify the contents of key
	// and value after Set returns.
	Set(key []byte, value []byte, opts ...interface{}) error
	// Delete removes the value for the given key. It is safe to modify the contents
	// of key after Delete returns.
	Delete(key []byte) error
}

type BatchWriter interface {
	// NewBatch returns a read-write batch. Any reads on the batch will read both from
	// the batch and the DB. If the batch is committed it will be applied to the DB.
	NewBatch() Batch
}

// Closer is a closeable key-value store, which blocks until all pending
// operations have persisted to disk.
type Closer interface {
	// Close closes the DB.
	Close() error
}

// DB represents a general key-value store.
type DB interface {
	Writer
	BatchWriter
	Reader
	Closer
	// Stringer returns a string description of the DB. Used for logging and configuration.
	fmt.Stringer
}
