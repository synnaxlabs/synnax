// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"context"

	"github.com/synnaxlabs/gorp/internal/key"
)

// Writer wraps a key-value writer to provide a strongly typed interface for writing
// entries to the DB. Writer is NOT safe for concurrent use.
type Writer[K Key, E Entry[K]] struct {
	// tx is the underlying transaction this Writer writes through.
	tx Tx
	// keyCodec encodes K to prefixed pebble keys.
	keyCodec key.Codec[K]
	// indexes is the set of secondary indexes that receive staged
	// mutations for each Set / Delete call. Nil means no per-write index
	// staging.
	indexes []Index[K, E]
	// table checks the references of each entry Set writes. Nil means no checks.
	table *Table[K, E]
}

// WrapWriter wraps the given Tx to provide a strongly typed Writer that
// does not stage mutations against any secondary indexes.
func WrapWriter[K Key, E Entry[K]](tx Tx) *Writer[K, E] {
	w := wrapWriter[K, E](tx, nil, nil)
	return &w
}

// wrapWriter constructs a Writer that encodes keys under prefix and stages each Set /
// Delete against indexes. A nil prefix falls back to the prefix of E's table. It
// returns a value so that the Writer of a query stays off the heap.
func wrapWriter[K Key, E Entry[K]](
	tx Tx,
	prefix []byte,
	indexes []Index[K, E],
) Writer[K, E] {
	if prefix == nil {
		prefix = key.Prefix[E]()
	}
	return Writer[K, E]{
		tx:       tx,
		keyCodec: key.NewCodec[K](prefix),
		indexes:  indexes,
	}
}

// Set writes the provided entries to the DB. When the Writer wraps a DB, Set writes all
// of them in one transaction.
func (w *Writer[K, E]) Set(ctx context.Context, entries ...E) error {
	return inTx(ctx, w.tx, func(tx Tx) error {
		bound := w.bind(tx)
		for _, entry := range entries {
			if err := bound.set(ctx, entry); err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete deletes the provided keys from the DB. When the Writer wraps a DB, Delete
// deletes all of them in one transaction.
func (w *Writer[K, E]) Delete(ctx context.Context, keys ...K) error {
	return inTx(ctx, w.tx, func(tx Tx) error {
		bound := w.bind(tx)
		for _, key := range keys {
			if err := bound.delete(ctx, key); err != nil {
				return err
			}
		}
		return nil
	})
}

// bind returns a copy of w that writes through tx.
func (w *Writer[K, E]) bind(tx Tx) Writer[K, E] {
	bound := *w
	bound.tx = tx
	return bound
}

func (w *Writer[K, E]) set(ctx context.Context, entry E) error {
	if w.table != nil {
		if err := w.table.check(ctx, w.tx, &entry); err != nil {
			return err
		}
	}
	data, err := w.tx.Encode(ctx, entry)
	if err != nil {
		return err
	}
	v := w.keyCodec.Encode(entry.GorpKey())
	if err := w.tx.Set(ctx, v, data, entry.SetOptions()...); err != nil {
		return err
	}
	for _, idx := range w.indexes {
		idx.stageSet(w.tx, entry)
	}
	return nil
}

func (w *Writer[K, E]) delete(ctx context.Context, key K) error {
	if err := w.tx.Delete(ctx, w.keyCodec.Encode(key)); err != nil {
		return err
	}
	for _, idx := range w.indexes {
		idx.stageDelete(w.tx, key)
	}
	return nil
}
