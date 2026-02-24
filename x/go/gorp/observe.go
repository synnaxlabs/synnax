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
	"bytes"
	"context"
	"iter"

	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/observe"
	"go.uber.org/zap"
)

// Observe wraps an observable key-value store and returns an observable that notifies
// its caller whenever a change is made to the provided entry type. This standalone
// function does NOT use a custom codec and relies on the DB's default codec for
// decoding values. For codec-aware observation, use Table.Observe().
func Observe[K Key, E Entry[K]](kvo BaseObservable) observe.Observable[iter.Seq[change.Change[K, E]]] {
	kCodec := newKeyCodec[K, E]()
	return observe.Translator[kv.TxReader, TxReader[K, E]]{
		Observable: kvo,
		Translate: func(ctx context.Context, reader kv.TxReader) (TxReader[K, E], bool) {
			var matched []kv.Change
			for ch := range reader {
				if bytes.HasPrefix(ch.Key, kCodec.prefix) {
					matched = append(matched, ch)
				}
			}
			if len(matched) == 0 {
				return nil, false
			}
			return wrapMatchedChanges[K, E](ctx, matched, kCodec, kvo, nil), true
		},
	}
}

// Observe returns an observable that notifies its caller whenever a change is made
// to entries in this table. If the table has a custom codec, it will be used to
// decode values.
func (t *Table[K, E]) Observe() observe.Observable[iter.Seq[change.Change[K, E]]] {
	kCodec := newKeyCodec[K, E]()
	return observe.Translator[kv.TxReader, TxReader[K, E]]{
		Observable: t.DB,
		Translate: func(ctx context.Context, reader kv.TxReader) (TxReader[K, E], bool) {
			var matched []kv.Change
			for ch := range reader {
				if bytes.HasPrefix(ch.Key, kCodec.prefix) {
					matched = append(matched, ch)
				}
			}
			if len(matched) == 0 {
				return nil, false
			}
			return wrapMatchedChanges[K, E](ctx, matched, kCodec, t.DB, t.codec), true
		},
	}
}

func wrapMatchedChanges[K Key, E Entry[K]](
	ctx context.Context,
	changes []kv.Change,
	kCodec *keyCodec[K, E],
	tools Tools,
	codec Codec[E],
) TxReader[K, E] {
	return func(yield func(change.Change[K, E]) bool) {
		for _, kvChange := range changes {
			var op change.Change[K, E]
			op.Variant = kvChange.Variant
			if op.Variant == change.VariantSet {
				if codec != nil {
					v, err := codec.Unmarshal(ctx, kvChange.Value)
					if err != nil {
						zap.S().DPanic("failed to decode value", zap.Error(err))
						continue
					}
					op.Value = v
				} else {
					if err := tools.Decode(ctx, kvChange.Value, &op.Value); err != nil {
						zap.S().DPanic("failed to decode value", zap.Error(err))
						continue
					}
				}
				op.Key = op.Value.GorpKey()
			} else {
				op.Key = kCodec.decode(kvChange.Key)
			}
			if !yield(op) {
				return
			}
		}
	}
}
