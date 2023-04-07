// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/query"
)

// Create is a query that creates Entries in the DB.
type Create[K Key, E Entry[K]] struct{ params query.Parameters }

// NewCreate opens a new Create query.
func NewCreate[K Key, E Entry[K]]() Create[K, E] {
	return Create[K, E]{params: make(query.Parameters)}
}

// Entries sets the Entries to write to the DB.
func (c Create[K, E]) Entries(entries *[]E) Create[K, E] { SetEntries[K](c.params, entries); return c }

// Entry sets the entry to write to the DB.
func (c Create[K, E]) Entry(entry *E) Create[K, E] { SetEntry[K](c.params, entry); return c }

// Exec executes the Params against the provided DB. It returns any errors encountered during execution.
func (c Create[K, E]) Exec(ctx context.Context, tx Tx) error {
	entries, w := GetEntries[K, E](c.params), NewWriter[K, E](tx)
	return w.Set(ctx, entries.All()...)
}
