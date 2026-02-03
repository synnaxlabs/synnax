// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"github.com/synnaxlabs/x/binary"
)

type IteratorOptions struct {
	LowerBound []byte
	UpperBound []byte
}

// Iterator iterates over key-value pairs in order. It is not necessary to exhaust the
// Iterator, but it is necessary to close it after use.
type Iterator interface {
	// First moves the iterator to the first key-value pair. Returns true if the Iterator
	// contains at least one key-value pair.
	First() bool
	// Last moves the iterator to the last key-value pair. Returns true if the Iterator
	// contains at least one key-value pair.
	Last() bool
	// Next advances to the next key-value pair. Returns true if the Iterator is pointing
	// to a valid key-value pair (i.e. an exhausted Iterator will return false).
	Next() bool
	// Prev returns the previous key-value pair. Returns true if the Iterator is pointing
	// to a valid key-value pair (i.e. a reverse-exhausted Iterator will return false).
	Prev() bool
	// Valid returns true if the iterator is currently positioned at a valid
	// key-value pair.
	Valid() bool
	// Key returns the key of the current key-value pair. Returns true if the iterator
	// is currently positioned at a valid key-value pair.
	Key() []byte
	// Value returns the value of the current key-value pair, or nil if the Iterator
	// is not pointing at a valid key. The caller must not modify the contents of the
	// returned slice, as it may change on subsequent movements.
	Value() []byte
	// SeekLT moves the iterator to the last key-value pair whose key is less than
	// the given key. Returns true if the Iterator is pointing at a valid entry and
	// false otherwise.
	SeekLT(key []byte) bool
	// SeekGE moves the Iterator to the first key-value pair whose key is greater than
	// or equal to the given key. Returns true if such a pair is found and false if
	// otherwise.
	SeekGE(key []byte) bool
	// SetBounds sets the lower and upper bounds for the iterator. Once SetBounds returns,
	// the caller is free to mutate the provided slices. The iterator will always be
	// invalidated and must be repositioned with a call to SeekGE, SeekLT, First, or Last.
	SetBounds(lower, upper []byte)
	// Error returns any accumulated error.
	Error() error
	// Close closes the Iterator and returns any accumulated error.
	Close() error
}

// IterPrefix returns IteratorOptions, that when passed to writer.NewStreamIterator, will
// return an Iterator that only iterates over keys with the given prefix.
func IterPrefix(prefix []byte) IteratorOptions {
	return IteratorOptions{LowerBound: prefix, UpperBound: prefixUpperBound(prefix)}
}

func prefixUpperBound(lower []byte) []byte {
	upper := binary.MakeCopy(lower)
	for i := len(upper) - 1; i >= 0; i-- {
		upper[i] = upper[i] + 1
		if upper[i] != 0 {
			return upper[:i+1]
		}
	}
	return nil
}
