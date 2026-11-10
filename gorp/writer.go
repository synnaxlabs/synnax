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

	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/types"
	"github.com/synnaxlabs/x/validate"
)

// Writer wraps a key-value writer to provide a strongly typed interface for writing
// entries to the DB. Writer is NOT safe for concurrent use.
type Writer[K Key, E Entry[K]] struct {
	// tx is the underlying transaction this Writer writes through.
	tx Tx
	// keyCodec encodes K to prefixed pebble keys.
	keyCodec key.Codec[K]
	// indexes is the set of secondary indexes that receive staged
	// mutations for each Set / Delete call. Nil means no per-write index
	// staging.
	indexes []Index[K, E]
	// table checks the references of each entry Set writes. Nil means no checks.
	table *Table[K, E]
	// tableName names the table of each record in the change list of tx.
	tableName string
	// references are the references whose links the change list of tx records.
	references []Reference[K, E]
	// derived is true when the writes of the Writer are cascades or detaches.
	derived bool
	// exclusive is true when Set refuses an entry whose key exists.
	exclusive bool
}

// WrapWriter wraps the given Tx to provide a strongly typed Writer that checks no
// references. The Writer keeps the indexes of each open Table of E on the DB of tx
// current, and records its writes under the name of the table of E, with the links of
// the references of each open Table of E.
func WrapWriter[K Key, E Entry[K]](tx Tx) *Writer[K, E] {
	w := wrapWriter[K, E](tx, "", nil, nil)
	return &w
}

// wrapWriter constructs a Writer that records writes under the table named name,
// encodes keys under prefix, and stages each Set / Delete against indexes. An empty
// name falls back to the name of the table that stores under prefix. A nil prefix falls
// back to the prefix of E's table. Nil indexes fall back to the indexes and the
// references of the open tables of E. It returns a value so that the Writer of a query
// stays off the heap.
func wrapWriter[K Key, E Entry[K]](
	tx Tx,
	name string,
	prefix []byte,
	indexes []Index[K, E],
) Writer[K, E] {
	if prefix == nil {
		prefix = key.Prefix[E]()
	}
	if name == "" {
		name = tableName[E](tx, prefix)
	}
	w := Writer[K, E]{
		tx:        tx,
		keyCodec:  key.NewCodec[K](prefix),
		indexes:   indexes,
		tableName: name,
	}
	if indexes == nil {
		w.indexes, w.references = openTables[K, E](tx, prefix)
	}
	return w
}

// tableName returns the name of the table that stores under prefix: the name of the
// registered table that does, or else the type name of E, which an unregistered table
// takes.
func tableName[E any](tx Tx, prefix []byte) string {
	if db := txDB(tx); db != nil {
		if name, ok := db.registry.tableName(prefix); ok {
			return name
		}
	}
	return types.Name[E]()
}

// Set writes the provided entries to the DB. When the Writer wraps a DB, Set writes all
// of them in one transaction.
func (w *Writer[K, E]) Set(ctx context.Context, entries ...E) error {
	return inTx(ctx, w.tx, func(tx Tx) error {
		bound := w.bind(tx)
		for _, entry := range entries {
			if err := bound.set(ctx, entry); err != nil {
				return err
			}
		}
		return nil
	})
}

// Delete deletes the provided keys from the DB. When the Writer wraps a DB, Delete
// deletes all of them in one transaction.
func (w *Writer[K, E]) Delete(ctx context.Context, keys ...K) error {
	return inTx(ctx, w.tx, func(tx Tx) error {
		bound := w.bind(tx)
		for _, key := range keys {
			if err := bound.delete(ctx, key); err != nil {
				return err
			}
		}
		return nil
	})
}

// openTables returns the indexes and the references of the open tables of E on the DB
// of tx that store under prefix.
func openTables[K Key, E Entry[K]](
	tx Tx,
	prefix []byte,
) ([]Index[K, E], []Reference[K, E]) {
	db := txDB(tx)
	if db == nil {
		return nil, nil
	}
	var (
		indexes    []Index[K, E]
		references []Reference[K, E]
	)
	for _, t := range db.registry.indexedTables(prefix) {
		if t, ok := t.(*Table[K, E]); ok {
			indexes = append(indexes, t.indexes...)
			references = append(references, t.references...)
		}
	}
	return indexes, references
}

// bind returns a copy of w that writes through tx.
func (w *Writer[K, E]) bind(tx Tx) Writer[K, E] {
	bound := *w
	bound.tx = tx
	return bound
}

func (w *Writer[K, E]) set(ctx context.Context, entry E) error {
	k := entry.GorpKey()
	encoded := w.keyCodec.Encode(k)
	changes := &w.tx.txIdentity().changes
	s, err := w.find(ctx, changes, k, encoded)
	if err != nil {
		return err
	}
	if w.exclusive && changes.present(s) {
		return errors.Wrapf(
			validate.ErrValidation,
			"%s %s already exists",
			w.tableName,
			key.Format(k),
		)
	}
	if w.table != nil {
		if err := w.table.check(ctx, w.tx, &entry); err != nil {
			return err
		}
	}
	data, err := w.tx.Encode(entry)
	if err != nil {
		return err
	}
	if err = w.tx.Set(ctx, encoded, data); err != nil {
		return err
	}
	changes.set(s, data, w.links(&entry), w.derived)
	for _, idx := range w.indexes {
		idx.stageSet(w.tx, entry)
	}
	return nil
}

func (w *Writer[K, E]) delete(ctx context.Context, k K) error {
	encoded := w.keyCodec.Encode(k)
	changes := &w.tx.txIdentity().changes
	s, err := w.find(ctx, changes, k, encoded)
	if err != nil {
		return err
	}
	if err = w.tx.Delete(ctx, encoded); err != nil {
		return err
	}
	changes.delete(s, w.derived)
	for _, idx := range w.indexes {
		idx.stageDelete(w.tx, k)
	}
	return nil
}

// find returns the slot in changes of a write to the record with key k, which encoded
// stores.
func (w *Writer[K, E]) find(
	ctx context.Context,
	changes *changeList,
	k K,
	encoded []byte,
) (slot, error) {
	return changes.find(
		ctx,
		w.tx,
		NewTarget(w.tableName, k),
		encoded,
		func(stored []byte) ([]Link, error) { return w.storedLinks(k, stored) },
	)
}

// storedLinks returns the links that the record with key k holds at its first write in
// tx. It reads them from the index of each reference that has loaded, and else from
// stored, the encoded record. A record that cannot be decoded fails the write, so that
// no change misses a removed link.
func (w *Writer[K, E]) storedLinks(k K, stored []byte) ([]Link, error) {
	var (
		links []Link
		known bool
		e     *E
	)
	for _, r := range w.references {
		if links, known = r.indexedLinks(links, w.tx, k); known {
			continue
		}
		if e == nil {
			e = new(E)
			if err := w.tx.Decode(stored, e); err != nil {
				return nil, errors.Wrapf(
					err,
					"[gorp] - read the links of %s %s",
					w.tableName,
					FormatKey(k),
				)
			}
		}
		links = r.links(links, e)
	}
	return links, nil
}

// links returns the links that e holds.
func (w *Writer[K, E]) links(e *E) []Link {
	var links []Link
	for _, r := range w.references {
		links = r.links(links, e)
	}
	return links
}
