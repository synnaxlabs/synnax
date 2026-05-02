// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import "context"

// Writer wraps a key-value writer to provide a strongly typed interface for writing
// entries to the DB. Writer is NOT safe for concurrent use.
type Writer[K Key, E Entry[K]] struct {
	tx       Tx
	keyCodec keyCodec[K, E]
	// indexes is the set of secondary indexes that receive staged
	// mutations for each Set / Delete call. Nil means no per-write index
	// staging.
	indexes []Index[K, E]
}

// WrapWriter wraps the given Tx to provide a strongly typed Writer that
// does not stage mutations against any secondary indexes.
func WrapWriter[K Key, E Entry[K]](tx Tx) *Writer[K, E] {
	return wrapWriter[K, E](tx, nil, nil)
}

// wrapWriter constructs a Writer with an optional precomputed key prefix
// (nil falls back to types.Name[E]()) and an optional index list to
// stage mutations against on each Set / Delete.
func wrapWriter[K Key, E Entry[K]](
	tx Tx,
	prefix []byte,
	indexes []Index[K, E],
) *Writer[K, E] {
	return &Writer[K, E]{
		tx:       tx,
		keyCodec: newKeyCodec[K, E](prefix),
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
	v := w.keyCodec.encode(entry.GorpKey())
	if err := w.tx.Set(ctx, v, data, entry.SetOptions()...); err != nil {
		return err
	}
	for _, idx := range w.indexes {
		idx.stageSet(w.tx, entry)
	}
	return nil
}

func (w *Writer[K, E]) delete(ctx context.Context, key K) error {
	if err := w.tx.Delete(ctx, w.keyCodec.encode(key)); err != nil {
		return err
	}
	for _, idx := range w.indexes {
		idx.stageDelete(w.tx, key)
	}
	return nil
}
