// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// All included pebble code is copyrighted by the cockroachdb team, and is licensed
// under the BSD 3-Clause License. See the repository file license/BSD-3-Clause.txt for
// more information.

// Package pebblekv implements a wrapper around cockroachdb's pebble storage engine that
// implements the kv.db interface. To use it, open a new pebble.DB and call Wrap() to
// wrap it.
package pebblekv

import (
	"context"
	"io"

	"github.com/cockroachdb/pebble/v2"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type db struct {
	*pebble.DB
	logger pebble.Logger
}

var _ kv.DB = (*db)(nil)

// Wrap wraps a pebble.DB to satisfy the kv.DB interface. logger receives the fatal
// errors of the DB, and should be the Logger that opened base.
func Wrap(base *pebble.DB, logger pebble.Logger) kv.DB {
	return &db{DB: base, logger: logger}
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

func NewNoopLogger() pebble.Logger {
	return logger{Instrumentation: alamos.Instrumentation{}}
}

func (l logger) Infof(format string, args ...any) {
	l.L.Infof(format, args...)
}

func (l logger) Errorf(format string, args ...any) {
	l.L.Zap().Sugar().Errorf(format, args...)
}

func (l logger) Fatalf(format string, args ...any) {
	l.L.Zap().Sugar().Fatalf(format, args...)
}

// OpenTx implement kv.DB.
func (d db) OpenTx() kv.Tx { return &tx{Batch: d.NewIndexedBatch(), db: d} }

// Commit implements kv.DB.
func (d db) Commit(context.Context, ...any) error { return nil }

func (d db) withTx(ctx context.Context, f func(tx kv.Tx) error) (err error) {
	tx := d.OpenTx()
	defer func() {
		err = errors.Combine(err, tx.Close())
	}()
	if err = f(tx); err != nil {
		return err
	}
	err = tx.Commit(ctx)
	return err
}

// Set implement kv.DB.
func (d db) Set(ctx context.Context, key, value []byte) error {
	return d.withTx(ctx, func(tx kv.Tx) error { return tx.Set(ctx, key, value) })
}

// Get implement kv.DB.
func (d db) Get(_ context.Context, key []byte, _ ...any) ([]byte, io.Closer, error) {
	b, c, err := d.DB.Get(key)
	return b, c, translateError(err)
}

// Delete implement kv.DB.
func (d db) Delete(ctx context.Context, key []byte) error {
	return d.withTx(ctx, func(tx kv.Tx) error { return tx.Delete(ctx, key) })
}

// OpenIterator implement kv.DB.
func (d db) OpenIterator(opts kv.IteratorOptions) (kv.Iterator, error) {
	return d.NewIter(parseIterOpts(opts))
}

// apply runs the OnApplied hooks before it waits for the WAL sync. Commits that wait at
// the same time share one sync.
func (d db) apply(txn *tx, opts []any) error {
	if err := d.ApplyNoSyncWait(txn.Batch, pebble.Sync); err != nil {
		return translateError(err)
	}
	for _, o := range opts {
		if applied, ok := o.(kv.OnApplied); ok {
			applied()
		}
	}
	err := txn.SyncWait()
	if err != nil {
		// Readers already see the batch, and Pebble cannot recover a failed WAL sync.
		d.logger.Fatalf("failed to sync the WAL: %v", err)
	}
	return err
}

// Report implement alamos.ReportProvider.
func (d db) Report() alamos.Report {
	return alamos.Report{"engine": "pebble"}
}

// Size implements kv.DB.
func (d db) Size() telem.Size { return telem.Size(d.DB.Metrics().DiskSpaceUsage()) }

// Close implement io.Closer.
func (d db) Close() error { return d.DB.Close() }

type tx struct {
	db db
	*pebble.Batch
	committed bool
}

var _ kv.Tx = (*tx)(nil)

// Set implements kv.Writer.
func (txn *tx) Set(_ context.Context, key, value []byte) error {
	return translateError(txn.Batch.Set(key, value, nil))
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
func (txn *tx) Delete(_ context.Context, key []byte) error {
	return translateError(txn.Batch.Delete(key, nil))
}

// OpenIterator implements kv.Writer.
func (txn *tx) OpenIterator(opts kv.IteratorOptions) (kv.Iterator, error) {
	return txn.NewIter(parseIterOpts(opts))
}

// Commit implements kv.Writer.
func (txn *tx) Commit(_ context.Context, opts ...any) error {
	txn.committed = true
	return txn.db.apply(txn, opts)
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

func parseIterOpts(opts kv.IteratorOptions) *pebble.IterOptions {
	return &pebble.IterOptions{
		LowerBound: opts.LowerBound,
		UpperBound: opts.UpperBound,
	}
}

func translateError(err error) error {
	if errors.Is(err, pebble.ErrNotFound) {
		return query.ErrNotFound
	}
	return err
}
