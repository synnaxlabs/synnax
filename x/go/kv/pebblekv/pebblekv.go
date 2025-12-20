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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/telem"
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

type openOptions struct {
	enableObserver bool
}

// OpenOption configures behavior when wrapping a pebble.DB.
type OpenOption func(*openOptions)

// DisableObservation disables the observer pattern, preventing change notifications.
// This improves performance when observation is not needed.
func DisableObservation() OpenOption {
	return func(o *openOptions) { o.enableObserver = false }
}

// Wrap wraps a pebble.DB to satisfy the kv.db interface.
func Wrap(base *pebble.DB, opts ...OpenOption) kv.DB {
	o := &openOptions{enableObserver: true}
	for _, opt := range opts {
		opt(o)
	}
	wrapped := &db{DB: base}
	if o.enableObserver {
		wrapped.Observer = observe.New[kv.TxReader]()
	}
	return wrapped
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

// NewReader implement kv.DB.
func (d db) NewReader() kv.TxReader { return d.OpenTx().NewReader() }

func (d db) withTx(ctx context.Context, f func(tx kv.Tx) error) error {
	var (
		err error
		t   = d.OpenTx()
	)
	defer func() {
		err = errors.Combine(err, t.Close())
	}()
	if err = f(t); err != nil {
		return err
	}
	err = t.Commit(ctx)
	return err
}

// Set implement kv.DB.
func (d db) Set(ctx context.Context, key, value []byte, opts ...any) error {
	// Hot path: if we don't need to notify observers of changes, then go straight
	// to the underlying DB.
	if d.Observer == nil {
		return translateError(d.DB.Set(key, value, parseOpts(opts)))
	}
	return d.withTx(ctx, func(tx kv.Tx) error {
		return tx.Set(ctx, key, value, opts...)
	})
}

// Get implement kv.DB.
func (d db) Get(_ context.Context, key []byte, _ ...any) ([]byte, io.Closer, error) {
	b, c, err := d.DB.Get(key)
	return b, c, translateError(err)
}

// Delete implement kv.DB.
func (d db) Delete(ctx context.Context, key []byte, opts ...any) error {
	// Hot path: if we don't need to notify observers of changes, then go straight
	// to the underlying DB.
	if d.Observer == nil {
		return translateError(d.DB.Delete(key, parseOpts(opts)))
	}
	return d.withTx(ctx, func(tx kv.Tx) error {
		return tx.Delete(ctx, key, opts...)
	})
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
	if d.Observer != nil {
		// We need to notify with a generator so that each subscriber gets a fresh reader.
		d.NotifyGenerator(ctx, txn.NewReader)
	}
	return nil
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
	return func(yield func(kv.Change) bool) {
		r := txn.Reader()
		for {
			kind, k, v, ok, err := r.Next()
			if err != nil {
				zap.S().DPanic("unexpected error reading batch", zap.Error(err))
				return
			}
			if !ok {
				return
			}
			variant, ok := kindToVariant(kind)
			if !ok {
				continue
			}
			if !yield(kv.Change{Variant: variant, Key: k, Value: v}) {
				return
			}
		}
	}
}

func kindToVariant(kind pebble.InternalKeyKind) (change.Variant, bool) {
	switch kind {
	case pebble.InternalKeyKindSet:
		return change.Set, true
	case pebble.InternalKeyKindDelete:
		return change.Delete, true
	default:
		return 0, false
	}
}

func parseIterOpts(opts kv.IteratorOptions) *pebble.IterOptions {
	return &pebble.IterOptions{
		LowerBound: opts.LowerBound,
		UpperBound: opts.UpperBound,
	}
}

func translateError(err error) error {
	if errors.Is(err, pebble.ErrNotFound) {
		return kv.NotFound
	}
	return err
}
