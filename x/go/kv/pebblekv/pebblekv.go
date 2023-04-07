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

type db struct{ *pebble.DB }

var _ kvc.DB = (*db)(nil)

var defaultWriteOpts = pebble.Sync

func parseOpts(opts []interface{}) *pebble.WriteOptions {
	if len(opts) > 0 {
		for _, o := range opts {
			if o, ok := o.(*pebble.WriteOptions); ok {
				return o
			}
		}
	}
	return defaultWriteOpts
}

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
func Wrap(db_ *pebble.DB) kvc.DB { return &db{DB: db_} }

// OpenTx implement kv.DB.
func (d db) OpenTx() kvc.Tx {
	return tx{Batch: d.DB.NewIndexedBatch()}
}

func (d db) Set(ctx context.Context, key, value []byte, opts ...interface{}) error {
	return d.DB.Set(key, value, parseOpts(opts))
}

func (d db) Get(ctx context.Context, key []byte, opts ...interface{}) ([]byte, error) {
	return get(d.DB, key)
}

func (d db) Delete(ctx context.Context, key []byte, opts ...interface{}) error {
	return d.DB.Delete(key, parseOpts(opts))
}

func (d db) OpenIterator(opts kvc.IteratorOptions) kvc.Iterator {
	return d.DB.NewIter(parseIterOpts(opts))
}

// Report implement alamos.ReportProvider.
func (d db) Report() alamos.Report { return alamos.Report{"engine": "pebble"} }

// Close implement io.Closer.
func (d db) Close() error { return d.DB.Close() }

type tx struct {
	*pebble.Batch
}

var _ kvc.Tx = tx{}

// Set implements kv.Writer.
func (b tx) Set(_ context.Context, key, value []byte, opts ...interface{}) error {
	return b.Batch.Set(key, value, parseOpts(opts))
}

// Get implements kv.Writer.
func (b tx) Get(_ context.Context, key []byte, opts ...interface{}) ([]byte, error) {
	return get(b.Batch, key)
}

// Delete implements kv.Writer.
func (b tx) Delete(_ context.Context, key []byte, opts ...interface{}) error {
	return b.Batch.Delete(key, parseOpts(opts))
}

// OpenIterator implements kv.Writer.
func (b tx) OpenIterator(opts kvc.IteratorOptions) kvc.Iterator {
	return b.Batch.NewIter(parseIterOpts(opts))
}

// Commit implements kv.Writer.
func (b tx) Commit(_ context.Context, opts ...interface{}) error {
	return b.Batch.Commit(defaultWriteOpts)
}

func get(reader pebble.Reader, key []byte) ([]byte, error) {
	v, c, err := reader.Get(key)
	if err != nil {
		return v, err
	}
	return v, c.Close()
}

func parseIterOpts(opts kvc.IteratorOptions) *pebble.IterOptions {
	return &pebble.IterOptions{
		LowerBound: opts.LowerBound,
		UpperBound: opts.UpperBound,
	}
}
