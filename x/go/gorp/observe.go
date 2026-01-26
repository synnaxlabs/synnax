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

func Observe[K Key, E Entry[K]](kvo BaseObservable) observe.Observable[iter.Seq[change.Change[K, E]]] {
	lp := &lazyPrefix[K, E]{Tools: kvo}
	return observe.Translator[kv.TxReader, TxReader[K, E]]{
		Observable: kvo,
		Translate: func(reader kv.TxReader) (TxReader[K, E], bool) {
			pref := lp.prefix(context.Background())
			var matched []kv.Change
			for ch := range reader {
				if bytes.HasPrefix(ch.Key, pref) {
					matched = append(matched, ch)
				}
			}
			if len(matched) == 0 {
				return nil, false
			}
			return wrapMatchedChanges[K, E](matched, kvo), true
		},
	}
}

func wrapMatchedChanges[K Key, E Entry[K]](changes []kv.Change, tools Tools) TxReader[K, E] {
	return func(yield func(change.Change[K, E]) bool) {
		ctx := context.Background()
		for _, kvChange := range changes {
			var op change.Change[K, E]
			op.Variant = kvChange.Variant
			if op.Variant == change.VariantSet {
				if err := tools.Decode(ctx, kvChange.Value, &op.Value); err != nil {
					zap.S().DPanic("failed to decode value", zap.Error(err))
					continue
				}
				op.Key = op.Value.GorpKey()
			} else {
				pref := prefix[K, E](ctx, tools)
				var err error
				if op.Key, err = decodeKey[K](ctx, tools, pref, kvChange.Key); err != nil {
					zap.S().DPanic("failed to decode key", zap.Error(err))
					continue
				}
			}
			if !yield(op) {
				return
			}
		}
	}
}
