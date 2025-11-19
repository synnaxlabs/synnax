// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// All included pebble code is copyrighted by the cockroachdb team, and is licensed under
// the BSD 3-Clause License. See the repository file license/BSD-3-Clause.txt for more
// information.

// Package pebblekv implements a wrapper around cockroachdb's pebble storage engine that
// implements the kv.db interface. To use it, open a new pebble.DB and call Wrap() to
// wrap it.
package pebblekv

import (
	"context"
	"io"

	"github.com/cockroachdb/pebble/v2"
	"github.com/cockroachdb/pebble/v2/batchrepr"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/observe"
	"go.uber.org/zap"
)

type db struct {
	observe.Observer[kv.TxReader]
	*pebble.DB
}

var _ kv.DB = (*db)(nil)

var defaultWriteOpts = pebble.Sync

func parseOpts(opts []any) *pebble.WriteOptions {
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

type logger struct{ alamos.Instrumentation }

var _ pebble.Logger = (*logger)(nil)

// NewLogger wraps the provided instrumentation to create a pebble compatible logger for
// communicating events through the alamos logging infrastructure as opposed to the
// internal pebble logger.
func NewLogger(ins alamos.Instrumentation) pebble.Logger {
	ins.L = ins.L.WithOptions(zap.AddCallerSkip(2))
	return logger{Instrumentation: ins}
}

func (l logger) Infof(format string, args ...any) { l.L.Infof(format, args...) }
func (l logger) Errorf(format string, args ...any) {
	l.L.Zap().Sugar().Errorf(format, args...)
}
func (l logger) Fatalf(format string, args ...any) {
	l.L.Zap().Sugar().Fatalf(format, args...)
}

// OpenTx implement kv.DB.
func (d db) OpenTx() kv.Tx { return &tx{Batch: d.NewIndexedBatch(), db: d} }

// Commit implements kv.DB.
func (d db) Commit(ctx context.Context, opts ...any) error { return nil }

// NewReader implement kv.DB.
func (d db) NewReader() kv.TxReader { return d.OpenTx().NewReader() }

// Set implement kv.DB.
func (d db) Set(ctx context.Context, key, value []byte, opts ...any) error {
	tx := d.OpenTx()
	if err := tx.Set(ctx, key, value, opts...); err != nil {
		return err
	}
	return tx.Commit(ctx, opts...)
}

// Get implement kv.DB.
func (d db) Get(_ context.Context, key []byte, _ ...any) ([]byte, io.Closer, error) {
	b, c, err := d.DB.Get(key)
	return b, c, translateError(err)
}

// Delete implement kv.DB.
func (d db) Delete(ctx context.Context, key []byte, opts ...any) error {
	return translateError(d.DB.Delete(key, parseOpts(opts)))
}

// OpenIterator implement kv.DB.
func (d db) OpenIterator(opts kv.IteratorOptions) (kv.Iterator, error) {
	return d.NewIter(parseIterOpts(opts))
}

func (d db) apply(ctx context.Context, txn *tx) error {
	err := d.Apply(txn.Batch, nil)
	if err != nil {
		return translateError(err)
	}
	// We need to notify with a generator so that each subscriber gets a fresh reader.
	d.NotifyGenerator(ctx, txn.NewReader)
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
func (txn *tx) Set(_ context.Context, key, value []byte, opts ...any) error {
	return translateError(txn.Batch.Set(key, value, parseOpts(opts)))
}

// Get implements kv.Writer.
func (txn *tx) Get(
	_ context.Context,
	key []byte,
	_ ...any,
) ([]byte, io.Closer, error) {
	b, closer, err := txn.Batch.Get(key)
	return b, closer, translateError(err)
}

// Delete implements kv.Writer.
func (txn *tx) Delete(
	_ context.Context,
	key []byte,
	opts ...any,
) error {
	return translateError(txn.Batch.Delete(key, parseOpts(opts)))
}

// OpenIterator implements kv.Writer.
func (txn *tx) OpenIterator(opts kv.IteratorOptions) (kv.Iterator, error) {
	return txn.NewIter(parseIterOpts(opts))
}

// Commit implements kv.Writer.
func (txn *tx) Commit(ctx context.Context, opts ...any) error {
	txn.committed = true
	return txn.db.apply(ctx, txn)
}

func (txn *tx) Close() error {
	// In our codebase, Close should be called regardless of whether the transaction was
	// committed or not. Pebble does not follow the same semantics, so we need to make
	// sure that we don't close the underlying batch if it was already committed.
	if !txn.committed {
		return txn.Batch.Close()
	}
	return nil
}

// NewReader implements kv.Writer.
func (txn *tx) NewReader() kv.TxReader {
	return &txReader{
		count:  int(txn.Count()),
		Reader: txn.Reader(),
	}
}

var kindsToVariant = map[pebble.InternalKeyKind]change.Variant{
	pebble.InternalKeyKindSet:    change.Set,
	pebble.InternalKeyKindDelete: change.Delete,
}

func parseIterOpts(opts kv.IteratorOptions) *pebble.IterOptions {
	return &pebble.IterOptions{
		LowerBound: opts.LowerBound,
		UpperBound: opts.UpperBound,
	}
}

type txReader struct {
	count int
	batchrepr.Reader
}

var _ kv.TxReader = (*txReader)(nil)

// Count implements kv.TxReader.
func (r *txReader) Count() int { return r.count }

// Next implements kv.TxReader.
func (r *txReader) Next(_ context.Context) (kv.Change, bool) {
	kind, k, v, ok, _ := r.Reader.Next()
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
	if errors.Is(err, pebble.ErrNotFound) {
		return kv.NotFound
	}
	return err
}
