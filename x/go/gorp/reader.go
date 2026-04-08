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
	"fmt"
	"io"
	"iter"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/types"
)

// Reader wraps a key-value reader to provide a strongly typed interface for reading
// entries from the DB. Reader only accesses entries that match its type arguments.
// Reader is NOT safe for concurrent use.
type Reader[K Key, E Entry[K]] struct {
	keyCodec *keyCodec[K, E]
	tx       Tx
}

// WrapReader wraps the given BaseReader to provide a strongly typed interface for reading
// entries from the DB.
func WrapReader[K Key, E Entry[K]](tx Tx) *Reader[K, E] {
	return &Reader[K, E]{tx: tx, keyCodec: newKeyCodec[K, E]()}
}

// Get retrieves a single entry from the database. If the entry does not exist,
// query.ErrNotFound is returned.
func (r Reader[K, E]) Get(ctx context.Context, key K) (E, error) {
	var e E
	b, closer, err := r.tx.Get(ctx, r.keyCodec.encode(key))
	if err != nil {
		return e, err
	}
	err = r.tx.Decode(ctx, b, &e)
	return e, errors.Combine(err, closer.Close())
}

// GetMany retrieves isMultiple entries from the database. Entries that are not
// found are simply omitted from the returned slice.
func (r Reader[K, E]) GetMany(ctx context.Context, keys []K) ([]E, error) {
	var (
		entries  = make([]E, 0, len(keys))
		notFound []K
	)
	for i := range keys {
		e, err := r.Get(ctx, keys[i])
		if err != nil {
			if errors.Is(err, query.ErrNotFound) {
				notFound = append(notFound, keys[i])
				continue
			} else {
				return entries, err
			}
		} else {
			entries = append(entries, e)
		}
	}
	if len(notFound) > 0 {
		return entries, errors.Wrapf(
			query.ErrNotFound,
			fmt.Sprintf("%s with keys %v not found", types.PluralName[E](), notFound),
		)
	}
	return entries, nil
}

type IterOptions struct {
	prefix []byte
}

// OpenIterator opens a new Iterator over the entries in the Reader.
func (r Reader[K, E]) OpenIterator(opts IterOptions) (iter *Iterator[E], err error) {
	prefixedKey := append(r.keyCodec.prefix, opts.prefix...)
	base, err := r.tx.OpenIterator(kv.IterPrefix(prefixedKey))
	return &Iterator[E]{Iterator: base, codec: r.tx}, err
}

// OpenNexter opens a new Nexter that can be used to iterate over
// the entries in the reader in sequential order.
func (r Reader[K, E]) OpenNexter(ctx context.Context) (iter.Seq[E], io.Closer, error) {
	i, err := r.OpenIterator(IterOptions{})
	if err != nil {
		return nil, nil, err
	}
	return func(yield func(E) bool) {
		for i.First(); i.Valid(); i.Next() {
			v := i.Value(ctx)
			if v == nil {
				continue
			}
			if !yield(*v) {
				return
			}
		}
	}, i, nil
}

// Iterator provides a simple wrapper around a kv.Iterator that decodes a byte-value
// before returning it to the caller. To create a new Iterator, call OpenIterator.
type Iterator[E any] struct {
	kv.Iterator
	err   error
	value *E
	codec encoding.Codec
}

// Value returns the decoded value from the iterator. Iterate.Alive must be true
// for calls to return a valid value. The returned pointer is reused across calls,
// so callers must copy the value if they need it to persist.
func (k *Iterator[E]) Value(ctx context.Context) (entry *E) {
	if k.value == nil {
		k.value = new(E)
	}
	var zero E
	*k.value = zero
	if err := k.codec.Decode(ctx, k.Iterator.Value(), k.value); err != nil {
		k.err = err
		return nil
	}
	return k.value
}

// Error returns the error accumulated by the Iterator.
func (k *Iterator[E]) Error() error {
	return lo.Ternary(k.err != nil, k.err, k.Iterator.Error())
}

// Valid returns true if the current iterator Value is pointing
// to a valid entry and the iterator has not accumulated an error.
func (k *Iterator[E]) Valid() bool {
	return lo.Ternary(k.err != nil, false, k.Iterator.Valid())
}

type TxReader[K Key, E Entry[K]] = iter.Seq[change.Change[K, E]]
