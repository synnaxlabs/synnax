// Copyright 2025 Synnax Labs, Inc.
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
)

// Writer represents a generalized key-value transaction that executes atomically against
// an underlying database. DB implements the Writer interface, which will execute
// queries directly against the DB. To open an isolated transaction against the DB, use
// cesium.BeginWrite.
type Writer[K Key, E Entry[K]] struct {
	BaseWriter
	keyCodec keyCodec[K, E]
}

// WrapWriter wraps the given key-value writer to provide a strongly
// typed interface for writing entries to the DB.
func WrapWriter[K Key, E Entry[K]](base BaseWriter) *Writer[K, E] {
	return &Writer[K, E]{BaseWriter: base, keyCodec: newKeyCodec[K, E]()}
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
	data, err := w.Encode(ctx, entry)
	if err != nil {
		return err
	}
	return w.BaseWriter.Set(
		ctx, w.keyCodec.encode(entry.GorpKey()), data, entry.SetOptions()...)
}

func (w *Writer[K, E]) delete(ctx context.Context, key K) error {
	// NOTE: We need to be careful with this operation in the future.
	// Because we aren't copying prefix, we're modifying the underlying slice.
	return w.BaseWriter.Delete(ctx, w.keyCodec.encode(key))
}
