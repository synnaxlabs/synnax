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
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
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
func (d db) OpenTx() kv.Tx { return &tx{Batch: d.DB.NewIndexedBatch(), db: d} }

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
func (d db) OpenIterator(opts kv.IteratorOptions) (kv.Iterator, error) {
	return d.DB.NewIter(parseIterOpts(opts))
}

func (d db) apply(ctx context.Context, txn *tx) error {
	err := d.DB.Apply(txn.Batch, nil)
	if err != nil {
		return translateError(err)
	}
	d.Notify(ctx, txn.NewReader())
	return nil
}

// Report implement alamos.ReportProvider.
func (d db) Report() alamos.Report {
	return alamos.Report{"engine": "pebble"}
}

// Close implement io.Closer.
func (d db) Close() error { return d.DB.Close() }

type tx struct {
	db        db
	committed bool
	*pebble.Batch
}

var _ kv.Tx = (*tx)(nil)

// Set implements kv.Writer.
func (txn *tx) Set(_ context.Context, key, value []byte, opts ...interface{}) error {
	return translateError(txn.Batch.Set(key, value, parseOpts(opts)))
}

// Get implements kv.Writer.
func (txn *tx) Get(_ context.Context, key []byte, opts ...interface{}) ([]byte, error) {
	b, err := get(txn.Batch, key)
	return b, translateError(err)
}

// Delete implements kv.Writer.
func (txn *tx) Delete(_ context.Context, key []byte, opts ...interface{}) error {
	return translateError(txn.Batch.Delete(key, parseOpts(opts)))
}

// OpenIterator implements kv.Writer.
func (txn *tx) OpenIterator(opts kv.IteratorOptions) (kv.Iterator, error) {
	return txn.Batch.NewIter(parseIterOpts(opts))
}

// Commit implements kv.Writer.
func (txn *tx) Commit(ctx context.Context, opts ...interface{}) error {
	txn.committed = true
	return txn.db.apply(ctx, txn)
}

func (txn *tx) Close() error {
	// In our codebase, Close should be called regardless of whether the transaction
	// was committed or not. Pebble does not follow the same semantics, so we need to
	// make sure that we don't close the underlying batch if it was already committed.
	if !txn.committed {
		return txn.Batch.Close()
	}
	return nil
}

// NewReader implements kv.Writer.
func (txn *tx) NewReader() kv.TxReader {
	return &txReader{
		count:       int(txn.Batch.Count()),
		BatchReader: txn.Batch.Reader(),
	}
}

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

type txReader struct {
	count int
	pebble.BatchReader
}

var _ kv.TxReader = (*txReader)(nil)

// Count implements kv.TxReader.
func (r *txReader) Count() int { return r.count }

// Next implements kv.TxReader.
func (r *txReader) Next(_ context.Context) (kv.Change, bool) {
	kind, k, v, ok, _ := r.BatchReader.Next()
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
