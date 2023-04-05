// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

// Writer represents a generalized key-value transaction that executes atomically against
// an underlying database. DB implements the Writer interface, which will execute
// queries directly against the DB. To open an isolated transaction against the DB, use
// DB.BeginWrite.
type Writer[K Key, E Entry[K]] struct{ WriteContext }

func NewWriter[K Key, E Entry[K]](ctx WriteContext) *Writer[K, E] {
	return &Writer[K, E]{WriteContext: ctx}
}

func (w *Writer[K, E]) Write(entry E) error {
	var (
		opts   = w.options()
		prefix = typePrefix[K, E](w.options())
	)
	data, err := opts.encoder.Encode(entry)
	if err != nil {
		return err
	}
	key, err := opts.encoder.Encode(entry.GorpKey())
	if err != nil {
		return err
	}
	// NOTE: We need to be careful with this operation in the future.
	// Because we aren't copying prefix, we're modifying the underlying slice.
	prefixedKey := append(prefix, key...)
	if err = w.Set(prefixedKey, data, entry.SetOptions()...); err != nil {
		return err
	}
	return nil
}

func (w *Writer[K, E]) WriteMany(entries []E) error {
	for _, entry := range entries {
		if err := w.Write(entry); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer[K, E]) Delete(key K) error {
	var (
		opts   = w.options()
		prefix = typePrefix[K, E](opts)
	)
	data, err := opts.encoder.Encode(key)
	if err != nil {
		return err
	}
	// NOTE: We need to be careful with this operation in the future.
	// Because we aren't copying prefix, we're modifying the underlying slice.
	if err = w.WriteContext.Delete(append(prefix, data...)); err != nil {
		return err
	}
	return nil
}
