// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

// TypedWriter represents a generalized key-value transaction that executes atomically against
// an underlying database. DB implements the TypedWriter interface, which will execute
// queries directly against the DB. To open an isolated transaction against the DB, use
// DB.NewWriter.
type TypedWriter[K Key, E Entry[K]] struct{ Writer }

func NewTypedWriter[K Key, E Entry[K]](writer Writer) *TypedWriter[K, E] {
	return &TypedWriter[K, E]{Writer: writer}
}

func (w *TypedWriter[K, E]) Write(entry E) error {
	prefix := typePrefix[K, E](w.options)
	data, err := w.encoder.Encode(entry)
	if err != nil {
		return err
	}
	key, err := w.encoder.Encode(entry.GorpKey())
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

func (w *TypedWriter[K, E]) WriteMany(entries []E) error {
	for _, entry := range entries {
		if err := w.Write(entry); err != nil {
			return err
		}
	}
	return nil
}

func (w *TypedWriter[K, E]) Delete(key K) error {
	prefix := typePrefix[K, E](w.options)
	data, err := w.encoder.Encode(key)
	if err != nil {
		return err
	}
	// NOTE: We need to be careful with this operation in the future.
	// Because we aren't copying prefix, we're modifying the underlying slice.
	if err = w.Writer.Delete(append(prefix, data...)); err != nil {
		return err
	}
	return nil
}
