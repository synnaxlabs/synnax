// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package pebblekv implements a wrapper around cockroachdb's pebble storage engine that implements
// the kv.db interface. To use it, open a new pebble.DB and call Wrap() to wrap it.
package pebblekv

import (
	"context"
	"github.com/cockroachdb/pebble"
	"github.com/synnaxlabs/alamos"
	kvc "github.com/synnaxlabs/x/kv"
)

type pebbleKV struct{ *pebble.DB }

var _ kvc.DB = (*pebbleKV)(nil)

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
func (db pebbleKV) Get(_ context.Context, key []byte, opts ...interface{}) ([]byte, error) {
	return get(db.DB, key)
}

// Set implements the kv.db interface.
func (db pebbleKV) Set(_ context.Context, key []byte, value []byte, opts ...interface{}) error {
	return db.DB.Set(key, value, parseWriterOpt(opts))
}

// Delete implements the kv.db interface.
func (db pebbleKV) Delete(_ context.Context, key []byte) error {
	return db.DB.Delete(key, pebble.NoSync)
}

// Close implements the kv.db interface.
func (db pebbleKV) Close() error { return db.DB.Close() }

// NewIterator implements the kv.db interface.
func (db pebbleKV) NewIterator(_ context.Context, opts kvc.IteratorOptions) kvc.Iterator {
	return db.DB.NewIter(&pebble.IterOptions{LowerBound: opts.LowerBound, UpperBound: opts.UpperBound})
}

func (db pebbleKV) NewBatch() kvc.Batch { return batch{db.DB.NewIndexedBatch()} }

func (db pebbleKV) Report() alamos.Report {
	return alamos.Report{"engine": "pebble"}
}

type batch struct{ *pebble.Batch }

func (b batch) Set(_ context.Context, key []byte, value []byte, opts ...interface{}) error {
	return b.Batch.Set(key, value, defaultWriteOpts)
}

func (b batch) Get(_ context.Context, key []byte, opts ...interface{}) ([]byte, error) {
	return get(b.Batch, key)
}

func (b batch) Delete(_ context.Context, key []byte) error {
	return b.Batch.Delete(key, defaultWriteOpts)
}

func (b batch) NewIterator(_ context.Context, opts kvc.IteratorOptions) kvc.Iterator {
	return b.Batch.NewIter(&pebble.IterOptions{LowerBound: opts.LowerBound, UpperBound: opts.UpperBound})
}

func (b batch) Commit(ctx context.Context, opts ...interface{}) error {
	return b.Batch.Commit(defaultWriteOpts)
}

func get(reader pebble.Reader, key []byte) ([]byte, error) {
	v, c, err := reader.Get(key)
	if err != nil {
		return v, err
	}
	return v, c.Close()
}
