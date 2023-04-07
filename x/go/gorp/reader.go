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
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

type Reader[K Key, E Entry[K]] struct {
	prefix []byte
	Tx
}

func NewReader[K Key, E Entry[K]](tx Tx) *Reader[K, E] {
	return &Reader[K, E]{Tx: tx, prefix: prefix[K, E](tx)}
}

func (r *Reader[K, E]) Get(ctx context.Context, key K) (e E, err error) {
	bKey, err := encodeKey(r.Tx.encoder(), r.prefix, key)
	if err != nil {
		return e, err
	}
	b, err := r.Tx.Get(ctx, bKey)
	if err != nil {
		return e, lo.Ternary(errors.Is(err, kv.NotFound), query.NotFound, err)
	}
	return e, r.Tx.decoder().Decode(ctx, b, &e)
}

func (r *Reader[K, E]) GetMany(ctx context.Context, keys []K) ([]E, error) {
	var (
		err     error
		entries = make([]E, len(keys))
	)
	for i := range keys {
		entries[i], err = r.Get(ctx, keys[i])
		if err != nil {
			return nil, err
		}
	}
	return entries, nil
}

func (r *Reader[K, E]) OpenIterator() *Iterator[E] {
	return OpenIterator[E](r.Tx.OpenIterator(kv.PrefixIter(r.prefix)), r.Tx.decoder())
}

// Iterator provides a simple wrapper around a kv.Iterator that decodes a byte-value
// before returning it to the caller. It provides no abstracted utilities for the
// iteration itself, and is focused only on maintaining a nearly identical interface to
// the underlying iterator. To create a new Iterator, call OpenIterator.
type Iterator[E any] struct {
	kv.Iterator
	error   error
	value   *E
	decoder binary.Decoder
}

// OpenIterator wraps the provided iterator. All valid calls to iter.Value are
// decoded into the entry type E.
func OpenIterator[E any](wrapped kv.Iterator, decoder binary.Decoder) *Iterator[E] {
	return &Iterator[E]{Iterator: wrapped, decoder: decoder}
}

// Value returns the decoded value from the iterator. Iter.Alive must be true
// for calls to return a valid value.
func (k *Iterator[E]) Value() (entry *E) {
	if k.value == nil {
		k.value = new(E)
	}
	if err := k.decoder.Decode(ctx, k.Iterator.Value(), k.value); err != nil {
		k.error = err
	}
	return k.value
}

func (k *Iterator[E]) Error() error {
	return lo.Ternary(k.error != nil, k.error, k.Iterator.Error())
}

func (k *Iterator[E]) Valid() bool {
	return lo.Ternary(k.error != nil, false, k.Iterator.Valid())
}
