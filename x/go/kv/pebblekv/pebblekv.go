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

	"github.com/cockroachdb/errors"
	"github.com/cockroachdb/pebble"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/observe"
)

type db struct {
	observe.Observer[kv.TxReader]
	*pebble.DB
}

var _ kv.DB = (*db)(nil)

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

// Wrap wraps a pebble.DB to satisfy the kv.db interface.
func Wrap(db_ *pebble.DB) kv.DB {
	return &db{DB: db_, Observer: observe.New[kv.TxReader]()}
}

// OpenTx implement kv.DB.
func (d db) OpenTx() kv.Tx { return tx{Batch: d.DB.NewIndexedBatch(), db: d} }

// Commit implements kv.DB.
func (d db) Commit(ctx context.Context, opts ...interface{}) error { return nil }

// NewReader implement kv.DB.
func (d db) NewReader() kv.TxReader { return d.OpenTx().NewReader() }

// Set implement kv.DB.
func (d db) Set(ctx context.Context, key, value []byte, opts ...interface{}) error {
	return translateError(d.DB.Set(key, value, parseOpts(opts)))
}

// Get implement kv.DB.
func (d db) Get(ctx context.Context, key []byte, opts ...interface{}) ([]byte, error) {
	b, err := get(d.DB, key)
	return b, translateError(err)
}

// Delete implement kv.DB.
func (d db) Delete(ctx context.Context, key []byte, opts ...interface{}) error {
	return translateError(d.DB.Delete(key, parseOpts(opts)))
}

// OpenIterator implement kv.DB.
func (d db) OpenIterator(opts kv.IteratorOptions) kv.Iterator {
	return d.DB.NewIter(parseIterOpts(opts))
}

func (d db) apply(ctx context.Context, txn tx) error {
	err := d.DB.Apply(txn.Batch, nil)
	if err != nil {
		return translateError(err)
	}
	d.Notify(ctx, txn.NewReader())
	return nil
}

// Report implement alamos.ReportProvider.
func (d db) Report() alamos.Report { return alamos.Report{"engine": "pebble"} }

// Close implement io.Closer.
func (d db) Close() error { return d.DB.Close() }

type tx struct {
	db db
	*pebble.Batch
}

var _ kv.Tx = tx{}

// Set implements kv.Writer.
func (txn tx) Set(_ context.Context, key, value []byte, opts ...interface{}) error {
	return translateError(txn.Batch.Set(key, value, parseOpts(opts)))
}

// Get implements kv.Writer.
func (txn tx) Get(_ context.Context, key []byte, opts ...interface{}) ([]byte, error) {
	b, err := get(txn.Batch, key)
	return b, translateError(err)
}

// Delete implements kv.Writer.
func (txn tx) Delete(_ context.Context, key []byte, opts ...interface{}) error {
	return translateError(txn.Batch.Delete(key, parseOpts(opts)))
}

// OpenIterator implements kv.Writer.
func (txn tx) OpenIterator(opts kv.IteratorOptions) kv.Iterator {
	return txn.Batch.NewIter(parseIterOpts(opts))
}

// Commit implements kv.Writer.
func (txn tx) Commit(ctx context.Context, opts ...interface{}) error {
	return txn.db.apply(ctx, txn)
}

// NewReader implements kv.Writer.
func (txn tx) NewReader() kv.TxReader { return &txReader{BatchReader: txn.Batch.Reader()} }

var kindsToVariant = map[pebble.InternalKeyKind]change.Variant{
	pebble.InternalKeyKindSet:    change.Set,
	pebble.InternalKeyKindDelete: change.Delete,
}

func get(reader pebble.Reader, key []byte) ([]byte, error) {
	v, c, err := reader.Get(key)
	if err != nil {
		return v, err
	}
	return v, c.Close()
}

func parseIterOpts(opts kv.IteratorOptions) *pebble.IterOptions {
	return &pebble.IterOptions{
		LowerBound: opts.LowerBound,
		UpperBound: opts.UpperBound,
	}
}

type txReader struct{ pebble.BatchReader }

var _ kv.TxReader = (*txReader)(nil)

// Next implements kv.TxReader.
func (r *txReader) Next(_ context.Context) (kv.Change, bool) {
	kind, k, v, ok := r.BatchReader.Next()
	if !ok {
		return kv.Change{}, false
	}
	variant, ok := kindsToVariant[kind]
	if !ok {
		return kv.Change{}, false
	}
	return kv.Change{Variant: variant, Key: k, Value: v}, true
}

func translateError(err error) error {
	if err == nil {
		return err
	}
	if errors.Is(err, pebble.ErrNotFound) {
		return kv.NotFound
	}
	return err

}
