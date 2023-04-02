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
	"github.com/synnaxlabs/x/kv"
)

type KVBatch[K Key, E Entry[K]] struct {
	kv.Batch
	opts options
}

func WrapKVBatch[K Key, E Entry[K]](batch kv.Batch, opts ...Option) *KVBatch[K, E] {
	return &KVBatch[K, E]{Batch: batch, opts: newOptions(opts...)}
}

func (w *KVBatch[K, E]) Write(ctx context.Context, entry E) error {
	prefix := typePrefix[K, E](w.opts)
	data, err := w.opts.encoder.Encode(entry)
	if err != nil {
		return err
	}
	key, err := w.opts.encoder.Encode(entry.GorpKey())
	if err != nil {
		return err
	}
	// NOTE: We need to be careful with this operation in the future.
	// Because we aren't copying prefix, we're modifying the underlying slice.
	prefixedKey := append(prefix, key...)
	if err = w.Set(ctx, prefixedKey, data, entry.SetOptions()...); err != nil {
		return err
	}
	return nil
}

func (w *KVBatch[K, E]) WriteMany(ctx context.Context, entries []E) error {
	for _, entry := range entries {
		if err := w.Write(ctx, entry); err != nil {
			return err
		}
	}
	return nil
}

func (w *KVBatch[K, E]) Delete(ctx context.Context, key K) error {
	prefix := typePrefix[K, E](w.opts)
	data, err := w.opts.encoder.Encode(key)
	if err != nil {
		return err
	}
	// NOTE: We need to be careful with this operation in the future.
	// Because we aren't copying prefix, we're modifying the underlying slice.
	if err = w.Batch.Delete(ctx, append(prefix, data...)); err != nil {
		return err
	}
	return nil
}

func (w *KVBatch[K, E]) options() options {
	return w.opts
}
