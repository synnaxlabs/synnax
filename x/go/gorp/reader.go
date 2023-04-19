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
	_prefix []byte
	Tx
}

func NewReader[K Key, E Entry[K]](tx Tx) *Reader[K, E] {
	return &Reader[K, E]{Tx: tx}
}

func (r *Reader[K, E]) prefix(ctx context.Context) []byte {
	if r._prefix == nil {
		r._prefix = prefix[K, E](ctx, r.Tx.noPrefix(), r.Tx.encoder())
	}
	return r._prefix
}

func (r *Reader[K, E]) Get(ctx context.Context, key K) (e E, err error) {
	bKey, err := encodeKey(ctx, r.Tx.encoder(), r.prefix(ctx), key)
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
		err_    error
		entries = make([]E, 0, len(keys))
	)
	for i := range keys {
		e, err := r.Get(ctx, keys[i])
		if err != nil {
			err_ = err
		} else {
			entries = append(entries, e)
		}
	}
	return entries, err_
}

func (r *Reader[K, E]) OpenIterator() *Iterator[E] {
	// TODO (emilbon99): Figure out if we want to use a proper context here.
	return OpenIterator[E](r.Tx.OpenIterator(kv.IterPrefix(r.prefix(context.TODO()))), r.Tx.decoder())
}

func (r *Reader[K, E]) Exhaust(ctx context.Context, f func(e E) error) error {
	iter := r.OpenIterator()
	for iter.First(); iter.Valid(); iter.Next() {
		v := iter.Value(ctx)
		if err := iter.Error(); err != nil {
			return err
		}
		if err := f(*v); err != nil {
			return err
		}
	}
	return iter.Close()
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

// Value returns the decoded value from the iterator. Iterate.Alive must be true
// for calls to return a valid value.
func (k *Iterator[E]) Value(ctx context.Context) (entry *E) {
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
