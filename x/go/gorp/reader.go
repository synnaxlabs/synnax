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
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
)

type Reader[K Key, E Entry[K]] struct {
	lazyPrefix[K, E]
	BaseReader
}

func NewReader[K Key, E Entry[K]](base BaseReader) *Reader[K, E] {
	return &Reader[K, E]{
		BaseReader: base,
		lazyPrefix: lazyPrefix[K, E]{Options: base},
	}
}

// Get retrieves a single entry from the database. If the entry does not exist,
// query.NotFound is returned.
func (r *Reader[K, E]) Get(ctx context.Context, key K) (e E, err error) {
	bKey, err := encodeKey(ctx, r, r.prefix(ctx), key)
	if err != nil {
		return e, err
	}
	b, err := r.BaseReader.Get(ctx, bKey)
	if err != nil {
		return e, lo.Ternary(errors.Is(err, kv.NotFound), query.NotFound, err)
	}
	return e, r.Decode(ctx, b, &e)
}

// GetMany retrieves multiple entries from the database. Entries that are not
// found are simply omitted from the returned slice.
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
	return WrapIterator[E](r.BaseReader.OpenIterator(kv.IterPrefix(
		r.prefix(context.TODO()))),
		r.BaseReader,
	)
}

func (r *Reader[K, E]) OpenNext() iter.NextCloser[E] {
	return &Next[E]{Iterator: r.OpenIterator()}
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

// WrapIterator wraps the provided iterator. All valid calls to iter.Value are
// decoded into the entry type E.
func WrapIterator[E any](wrapped kv.Iterator, decoder binary.Decoder) *Iterator[E] {
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

type Next[E any] struct{ *Iterator[E] }

var _ iter.NextCloser[any] = (*Next[any])(nil)

func (n Next[E]) Next(ctx context.Context) (e E, ok bool, err error) {
	ok = n.Iterator.Next()
	if !ok {
		return e, ok, n.Iterator.Error()
	}
	return *n.Iterator.Value(ctx), ok, n.Iterator.Error()
}

type TxReader[K Key, E Entry[K]] interface{ iter.Next[E] }

type txReader[K Key, E Entry[K]] struct {
	kv.TxReader
	decoder       binary.Decoder
	prefixMatcher func(ctx context.Context, key []byte) bool
}

func WrapTxReader[K Key, E Entry[K]](reader kv.TxReader, opts Options) TxReader[K, E] {
	return txReader[K, E]{
		TxReader:      reader,
		decoder:       opts,
		prefixMatcher: prefixMatcher[K, E](opts),
	}
}

func (t txReader[K, E]) Next(ctx context.Context) (e E, ok bool, err error) {
	op, ok, err := t.TxReader.Next(ctx)
	if !ok || err != nil || !t.prefixMatcher(ctx, op.Key) {
		return e, ok, err
	}
	t.decoder.Decode(ctx, op.Value, &e)
	return e, ok, err
}
