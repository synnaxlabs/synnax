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
	"slices"

	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	xio "github.com/synnaxlabs/x/io"
	"github.com/synnaxlabs/x/kv"
)

// Reader wraps a key-value reader to provide a strongly typed interface for reading
// entries from the DB. Reader only accesses entries that match its type arguments.
// Reader is NOT safe for concurrent use.
type Reader[K Key, E Entry[K]] struct {
	// keyCodec encodes K to prefixed pebble keys and decodes the reverse.
	keyCodec key.Codec[K]
	// tx is the underlying transaction this Reader reads through.
	tx Tx
}

// WrapReader wraps the given Tx to provide a strongly typed Reader.
func WrapReader[K Key, E Entry[K]](tx Tx) Reader[K, E] {
	return wrapReader[K, E](tx, nil)
}

// wrapReader constructs a Reader that decodes keys under prefix. A nil prefix falls
// back to the prefix of E's table.
func wrapReader[K Key, E Entry[K]](tx Tx, prefix []byte) Reader[K, E] {
	if prefix == nil {
		prefix = key.Prefix[E]()
	}
	return Reader[K, E]{tx: tx, keyCodec: key.NewCodec[K](prefix)}
}

// Get retrieves a single entry from the database. If the entry does not exist,
// query.ErrNotFound is returned.
func (r *Reader[K, E]) Get(ctx context.Context, key K) (E, error) {
	var e E
	_, closer, err := r.get(ctx, key, &e)
	if err != nil {
		var zero E
		return zero, err
	}
	return e, closer.Close()
}

func (r *Reader[K, E]) get(
	ctx context.Context,
	key K,
	entry *E,
) ([]byte, io.Closer, error) {
	b, closer, err := r.tx.Get(ctx, r.keyCodec.Encode(key))
	if err != nil {
		return nil, nil, err
	}
	if err := r.tx.Decode(ctx, b, entry); err != nil {
		return nil, nil, errors.Combine(err, closer.Close())
	}
	return b, closer, nil
}

// IterOptions configures a Reader iterator.
type IterOptions struct {
	// prefix narrows the scan to keys starting with this byte prefix
	// (appended to the entry-type prefix). Nil scans all entries of
	// type E.
	prefix []byte
}

// OpenIterator opens a new Iterator over the entries in the Reader.
func (r Reader[K, E]) OpenIterator(opts IterOptions) (iter *Iterator[E], err error) {
	prefix := r.keyCodec.Prefix()
	// Every query of a table shares the codec prefix, so a narrowed prefix needs its
	// own slice.
	if len(opts.prefix) > 0 {
		prefix = slices.Concat(prefix, opts.prefix)
	}
	base, err := r.tx.OpenIterator(kv.IterPrefix(prefix))
	return &Iterator[E]{Iterator: base, codec: r.tx}, err
}

// OpenNexter opens a new Nexter that can be used to iterate over the entries in the
// reader in sequential order. Iteration stops at the first entry that fails to decode,
// and Close returns the decode error.
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
	}, xio.CloserFunc(func() error { return errors.Combine(i.err, i.Close()) }), nil
}

// Iterator provides a simple wrapper around a kv.Iterator that decodes a byte-value
// before returning it to the caller. To create a new Iterator, call OpenIterator.
type Iterator[E any] struct {
	// Iterator is the underlying byte-level iterator.
	kv.Iterator
	// err is the last decode error encountered by Value. Surfaced via
	// Error().
	err error
	// value is the reusable decode buffer; Value returns a pointer to
	// this and overwrites it on each call.
	value *E
	// codec decodes the iterator's raw bytes into E.
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
