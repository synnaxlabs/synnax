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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
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

// Set writes the provided entries to the DB.
func (w *Writer[K, E]) Set(ctx context.Context, entries ...E) error {
	for _, entry := range entries {
		if err := w.set(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the provided keys from the DB.
func (w *Writer[K, E]) Delete(ctx context.Context, keys ...K) error {
	for _, key := range keys {
		if err := w.delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer[K, E]) set(ctx context.Context, entry E) error {
	data, err := w.tx.Encode(ctx, entry)
	if err != nil {
		return err
	}
	v := w.keyCodec.Encode(entry.GorpKey())
	if w.tx.txIdentity() == nil {
		// A DB commits inside Set, and observers run before Set returns. The indexes
		// take the entry first so that those observers find it.
		for _, idx := range w.indexes {
			idx.set(entry)
		}
		if err := w.tx.Set(ctx, v, data, entry.SetOptions()...); err != nil {
			return errors.Combine(err, w.restore(ctx, entry.GorpKey()))
		}
		return nil
	}
	if err := w.tx.Set(ctx, v, data, entry.SetOptions()...); err != nil {
		return err
	}
	for _, idx := range w.indexes {
		idx.stageSet(w.tx, entry)
	}
	return nil
}

// restore returns the indexes to the entry that storage holds for key.
func (w *Writer[K, E]) restore(ctx context.Context, key K) error {
	r := Reader[K, E]{tx: w.tx, keyCodec: w.keyCodec}
	entry, err := r.Get(ctx, key)
	if errors.Is(err, query.ErrNotFound) {
		for _, idx := range w.indexes {
			idx.delete(key)
		}
		return nil
	}
	if err != nil {
		return err
	}
	for _, idx := range w.indexes {
		idx.set(entry)
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
