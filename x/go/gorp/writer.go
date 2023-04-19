// Copyright 2023 Synnax Labs, Inc.
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
// DB.BeginWrite.
type Writer[K Key, E Entry[K]] struct {
	Tx
	_prefix []byte
}

func NewWriter[K Key, E Entry[K]](tx Tx) *Writer[K, E] {
	return &Writer[K, E]{Tx: tx}
}

func (w *Writer[K, E]) prefix(ctx context.Context) []byte {
	if w._prefix == nil {
		w._prefix = prefix[K, E](ctx, w.Tx.noPrefix())
	}
	return w._prefix
}

func (w *Writer[K, E]) Set(ctx context.Context, entries ...E) error {
	for _, entry := range entries {
		if err := w.set(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer[K, E]) Delete(ctx context.Context, keys ...K) error {
	for _, key := range keys {
		if err := w.delete(ctx, key); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer[K, E]) set(ctx context.Context, entry E) error {
	data, err := w.encoder().Encode(ctx, entry)
	if err != nil {
		return err
	}
	key, err := w.encoder().Encode(ctx, entry.GorpKey())
	if err != nil {
		return err
	}
	// NOTE: We need to be careful with this operation in the future.
	// Because we aren't copying prefix, we're modifying the underlying slice.
	prefixedKey := append(w.prefix(ctx), key...)
	if err = w.Tx.Set(ctx, prefixedKey, data, entry.SetOptions()...); err != nil {
		return err
	}
	return nil
}

func (w *Writer[K, E]) delete(ctx context.Context, key K) error {
	data, err := w.encoder().Encode(nil, key)
	if err != nil {
		return err
	}
	// NOTE: We need to be careful with this operation in the future.
	// Because we aren't copying prefix, we're modifying the underlying slice.
	return w.Tx.Delete(ctx, append(w.prefix(ctx), data...))
}
