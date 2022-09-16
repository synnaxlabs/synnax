// Package pebblekv implements a wrapper around cockroachdb's pebble storage engine that implements
// the kv.db interface. To use it, open a new pebble.DB and call Wrap() to wrap it.
package pebblekv

import (
	"github.com/cockroachdb/pebble"
	"github.com/synnaxlabs/x/alamos"
	kvc "github.com/synnaxlabs/x/kv"
)

type pebbleKV struct{ *pebble.DB }

var defaultWriteOpts = pebble.Sync

func parseWriterOpt(opts []interface{}) *pebble.WriteOptions {
	if len(opts) == 0 {
		return defaultWriteOpts
	}
	if o, ok := opts[0].(*pebble.WriteOptions); ok {
		return o
	}
	return defaultWriteOpts
}

// Wrap wraps a pebble.DB to satisfy the kv.db interface.
func Wrap(db *pebble.DB) kvc.DB { return &pebbleKV{DB: db} }

// Get implements the kv.db interface.
func (db pebbleKV) Get(key []byte, opts ...interface{}) ([]byte, error) {
	return get(db.DB, key)
}

// Set implements the kv.db interface.
func (db pebbleKV) Set(key []byte, value []byte, opts ...interface{}) error {
	return db.DB.Set(key, value, parseWriterOpt(opts))
}

// Delete implements the kv.db interface.
func (db pebbleKV) Delete(key []byte) error { return db.DB.Delete(key, pebble.NoSync) }

// Close implements the kv.db interface.
func (db pebbleKV) Close() error { return db.DB.Close() }

// NewIterator implements the kv.db interface.
func (db pebbleKV) NewIterator(opts kvc.IteratorOptions) kvc.Iterator {
	return db.DB.NewIter(&pebble.IterOptions{LowerBound: opts.LowerBound, UpperBound: opts.UpperBound})
}

func (db pebbleKV) NewBatch() kvc.Batch {
	return batch{db.DB.NewIndexedBatch()}
}

func (db pebbleKV) Report() alamos.Report {
	return alamos.Report{"engine": "pebble"}
}

type batch struct{ *pebble.Batch }

func (b batch) Set(key []byte, value []byte, opts ...interface{}) error {
	return b.Batch.Set(key, value, defaultWriteOpts)
}

func (b batch) Get(key []byte, opts ...interface{}) ([]byte, error) {
	return get(b.Batch, key)
}

func (b batch) Delete(key []byte) error { return b.Batch.Delete(key, defaultWriteOpts) }

func (b batch) NewIterator(opts kvc.IteratorOptions) kvc.Iterator {
	return b.Batch.NewIter(&pebble.IterOptions{LowerBound: opts.LowerBound, UpperBound: opts.UpperBound})
}

func (b batch) Commit(opts ...interface{}) error { return b.Batch.Commit(defaultWriteOpts) }

func get(reader pebble.Reader, key []byte) ([]byte, error) {
	v, c, err := reader.Get(key)
	if err != nil {
		return v, err
	}
	return v, c.Close()
}
