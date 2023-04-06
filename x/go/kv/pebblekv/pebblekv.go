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

// NewWriter implement kv.DB.
func (d db) NewWriter(ctx context.Context) kvc.Writer {
	return writer{ctx: ctx, Batch: d.DB.NewIndexedBatch()}
}

// NewReader implement kv.DB.
func (d db) NewReader(ctx context.Context) kvc.Reader { return reader{ctx: ctx, Reader: d} }

// Report implement alamos.ReportProvider.
func (d db) Report() alamos.Report { return alamos.Report{"engine": "pebble"} }

// Close implement io.Closer.
func (d db) Close() error { return d.DB.Close() }

type writer struct {
	ctx context.Context
	*pebble.Batch
}

var _ kvc.Writer = writer{}

func (b writer) Context() context.Context { return b.ctx }

// Set implements kv.Writer.
func (b writer) Set(key []byte, value []byte, opts ...interface{}) error {
	return b.Batch.Set(key, value, defaultWriteOpts)
}

// Get implements kv.Writer.
func (b writer) Get(key []byte, opts ...interface{}) ([]byte, error) {
	return get(b.Batch, key)
}

// Delete implements kv.Writer.
func (b writer) Delete(key []byte) error {
	return b.Batch.Delete(key, defaultWriteOpts)
}

// NewIterator implements kv.Writer.
func (b writer) Iterate(opts kvc.IteratorOptions) kvc.Iterator {
	return b.Batch.NewIter(&pebble.IterOptions{LowerBound: opts.LowerBound, UpperBound: opts.UpperBound})
}

// Commit implements kv.Writer.
func (b writer) Commit(opts ...interface{}) error {
	return b.Batch.Commit(defaultWriteOpts)
}

type reader struct {
	ctx context.Context
	pebble.Reader
}

func (r reader) Context() context.Context { return r.ctx }

func (r reader) Get(key []byte, opts ...interface{}) ([]byte, error) {
	return get(r.Reader, key)
}

func (r reader) Iterate(opts kvc.IteratorOptions) kvc.Iterator {
	return r.Reader.NewIter(&pebble.IterOptions{LowerBound: opts.LowerBound, UpperBound: opts.UpperBound})
}

func get(reader pebble.Reader, key []byte) ([]byte, error) {
	v, c, err := reader.Get(key)
	if err != nil {
		return v, err
	}
	return v, c.Close()
}
