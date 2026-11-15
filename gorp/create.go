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
)

// Create is a query that creates Entries in the DB.
type Create[K Key, E Entry[K]] struct {
	// entries is the set of entries the query will write.
	entries Entries[K, E]
	// keyPrefix is the gorp key prefix for entry type E.
	keyPrefix []byte
	// table is the Table the query writes through. Nil for a query that no Table
	// built.
	table *Table[K, E]
	// exclusive is true when Exec refuses an entry whose key exists.
	exclusive bool
}

// NewCreate opens a new Create query.
func NewCreate[K Key, E Entry[K]]() Create[K, E] {
	return Create[K, E]{}
}

// Entries sets the Entries to write to the DB.
func (c Create[K, E]) Entries(entries *[]E) Create[K, E] {
	c.entries.bindMultiple(entries)
	return c
}

// Entry sets the entry to write to the DB.
func (c Create[K, E]) Entry(entry *E) Create[K, E] {
	c.entries.bindSingle(entry)
	return c
}

// Exclusive makes Exec refuse an entry whose key exists in the transaction, including
// one that an earlier write in the transaction created.
func (c Create[K, E]) Exclusive() Create[K, E] {
	c.exclusive = true
	return c
}

// Exec executes the query against the provided transaction. It returns any errors
// encountered during execution. An exclusive query returns an error wrapping
// validate.ErrValidation that names the table and the key of an entry that exists.
func (c Create[K, E]) Exec(ctx context.Context, tx Tx) error {
	checkForNilTx("Create.Exec", tx)
	var w Writer[K, E]
	if c.table != nil {
		w = c.table.writer(tx, true)
	} else {
		w = wrapWriter[K, E](tx, "", c.keyPrefix, nil)
	}
	w.exclusive = c.exclusive
	return w.Set(ctx, c.entries.All()...)
}
