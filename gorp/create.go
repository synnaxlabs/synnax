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
	// indexes is the set of secondary indexes the query stages writes
	// against. Nil means writes are not staged.
	indexes []Index[K, E]
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

// Exec executes the query against the provided transaction. It returns any errors
// encountered during execution.
func (c Create[K, E]) Exec(ctx context.Context, tx Tx) error {
	checkForNilTx("Create.Exec", tx)
	w := wrapWriter(tx, c.keyPrefix, c.indexes)
	return w.Set(ctx, c.entries.All()...)
}
