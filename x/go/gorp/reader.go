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
	"bytes"
	"context"
	"fmt"
	"io"
	"iter"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/types"
)

// Reader wraps a key-value reader to provide a strongly typed interface for
// reading entries from the DB. Readonly only accesses entries that match
// its type arguments.
type Reader[K Key, E Entry[K]] struct {
	keyCodec keyCodec[K, E]
	// BaseReader is the underlying key-value reader that the Reader is wrapping.
	BaseReader
}

// WrapReader wraps the given key-value reader to provide a strongly
// typed interface for reading entries from the DB. It's important to note
// that the Reader only access to the entries provided as the type arguments
// to this function. The returned reader is safe for concurrent use.
// The following example reads from a DB:
//
//	r := gor.WrapReader[MyKey, MyEntry](db)
//
// The next example reads from a Tx:
//
//	r := gor.WrapReader[MyKey, MyEntry](tx)
func WrapReader[K Key, E Entry[K]](base BaseReader) *Reader[K, E] {
	return &Reader[K, E]{BaseReader: base, keyCodec: newKeyCodec[K, E]()}
}

// Get retrieves a single entry from the database. If the entry does not exist,
// query.NotFound is returned.
func (r Reader[K, E]) Get(ctx context.Context, key K) (E, error) {
	var e E
	b, closer, err := r.BaseReader.Get(ctx, r.keyCodec.encode(key))
	if err != nil {
		return e, err
	}
	err = r.Decode(ctx, b, &e)
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
			// We keep iterating here to ensure that we return all entries that
			// can be found.
			if errors.Is(err, query.NotFound) {
				notFound = append(notFound, keys[i])
				continue
			} else {
				// All other errors are considered no-ops.
				return entries, err
			}
		} else {
			entries = append(entries, e)
		}
	}
	if len(notFound) > 0 {
		return entries, errors.Wrapf(
			query.NotFound,
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
	base, err := r.BaseReader.OpenIterator(kv.IterPrefix(prefixedKey))
	return WrapIterator[E](base, r), err
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
			if !yield(*i.Value(ctx)) {
				return
			}
		}
	}, i, nil
}

// Iterator provides a simple wrapper around a kv.Iterator that decodes a byte-value
// before returning it to the caller. It provides no abstracted utilities for the
// iteration itself, and is focused only on maintaining a nearly identical interface to
// the underlying iterator. To create a new Iterator, call OpenIterator.
type Iterator[E any] struct {
	kv.Iterator
	err     error
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
	k.value = new(E)
	if err := k.decoder.Decode(ctx, k.Iterator.Value(), k.value); err != nil {
		k.err = err
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

// WrapTxReader wraps the given key-value reader to provide a strongly
// typed interface for iterating over a transactions operations in order. The given
// tools are used to implement gorp-specific functionality, such as
// decoding. Typically, the Tools interface is satisfied by a gorp.Tx
// or a gorp.Db. The following example reads from a tx;
//
//	tx := db.OpenTx()
//	defer tx.Close()
//
//	r := gor.WrapTxReader[MyKey, MyEntry](tx.NewReader(), tx)
//
//	for op := range r {
//	    // process op
//	}
func WrapTxReader[K Key, E Entry[K]](reader kv.TxReader, tools Tools) TxReader[K, E] {
	return func(yield func(change.Change[K, E]) bool) {
		var (
			ctx    = context.TODO()
			kCodec = newKeyCodec[K, E]()
			op     change.Change[K, E]
			err    error
		)
		for kvChange := range reader {
			if !bytes.HasPrefix(kvChange.Key, kCodec.prefix) {
				continue
			}
			op.Key = kCodec.decode(kvChange.Key)
			op.Variant = kvChange.Variant
			if op.Variant == change.Set {
				// Panicking in development here right now. Don't want to extend the
				// footprint of TxReader to NexterCloser.
				if err = tools.Decode(ctx, kvChange.Value, &op.Value); err != nil {
					panic(err)
				}
				op.Key = op.Value.GorpKey()
			}
			if !yield(op) {
				return
			}
		}
	}
}
