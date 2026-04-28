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
	"io"
	"iter"

	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/slices"
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
	var zero E
	b, closer, err := r.tx.Get(ctx, r.keyCodec.encode(key))
	if err != nil {
		return zero, err
	}
	var e E
	if err := r.tx.Decode(ctx, b, &e); err != nil {
		return zero, errors.Combine(err, closer.Close())
	}
	return e, closer.Close()
}

// GetMany retrieves multiple entries from the database. Missing keys are
// omitted from the returned slice but reported via a wrapped query.ErrNotFound
// in the returned error; callers that don't care about missing keys should
// use errors.Skip(err, query.ErrNotFound).
func (r Reader[K, E]) GetMany(ctx context.Context, keys []K) ([]E, error) {
	seq, closer := r.iterKeys(ctx, keys)
	return slices.CollectKeys(seq, len(keys)), closer.Close()
}

// iterKeys returns a sequence of (decoded, raw) pairs for each key that
// exists in the DB. The raw bytes yielded by the seq are valid only until
// the returned closer is closed; the caller MUST close once iteration is
// complete. Decoding happens lazily inside the seq, so filters that reject
// on raw bytes don't pay the decode cost.
//
// The returned error wraps query.ErrNotFound with the missing key list when
// any keys are absent, while the seq still yields the keys that were found.
// On a non-NotFound kv error the partial closer is still returned so the
// caller can release any successful gets.
func (r Reader[K, E]) iterKeys(
	ctx context.Context,
	keys []K,
) (iter.Seq2[E, []byte], io.Closer) {
	var (
		resErr   error
		notFound []K
		seq      = func(yield func(E, []byte) bool) {
			var e E
			for _, k := range keys {
				b, closer, err := r.tx.Get(ctx, r.keyCodec.encode(k))
				if err != nil {
					if errors.Is(err, query.ErrNotFound) {
						notFound = append(notFound, k)
						continue
					}
					resErr = err
					return
				}
				if err := r.tx.Decode(ctx, b, &e); err != nil {
					resErr = errors.Combine(err, closer.Close())
					return
				}
				ok := yield(e, b)
				if err := closer.Close(); err != nil {
					resErr = err
					return
				}
				if !ok {
					return
				}
			}
		}
		closer = xio.CloserFunc(func() error {
			if resErr != nil {
				return resErr
			}
			if len(notFound) > 0 {
				return errors.Wrapf(
					query.ErrNotFound,
					"%s with keys %v not found",
					types.PluralName[E](),
					notFound,
				)
			}
			return nil
		})
	)
	return seq, closer
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
	return errors.Join(k.err, k.Iterator.Error())
}

// Valid returns true if the current iterator Value is pointing
// to a valid entry and the iterator has not accumulated an error.
func (k *Iterator[E]) Valid() bool {
	return k.err == nil && k.Iterator.Valid()
}

type TxReader[K Key, E Entry[K]] = iter.Seq[change.Change[K, E]]
