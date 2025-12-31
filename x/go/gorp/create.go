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
	"context"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Create is a query that creates Entries in the DB.
type Create[K Key, E Entry[K]] struct {
	entries  *Entries[K, E]
	onUpdate onUpdate[K, E]
}

// NewCreate opens a new Create query.
func NewCreate[K Key, E Entry[K]]() Create[K, E] {
	return Create[K, E]{entries: new(Entries[K, E])}
}

// MergeExisting adds a function to the query that can be used to prevent the accidental
// override of existing entries with the same GorpKey. The provided function receives
// the existing entry and should return an error if the entry should not be overwritten.
// If no entry with a matching GorpKey is found, the function is not called. MergeExisting
// adds overhead to the query, as a retrieval is required to check for existing entries.
func (c Create[K, E]) MergeExisting(filter func(ctx Context, creating E, existing E) (E, error)) Create[K, E] {
	c.onUpdate = append(c.onUpdate, filter)
	return c
}

// Entries sets the Entries to write to the DB.
func (c Create[K, E]) Entries(entries *[]E) Create[K, E] {
	c.entries = multipleEntries[K, E](entries)
	return c
}

// Entry sets the entry to write to the DB.
func (c Create[K, E]) Entry(entry *E) Create[K, E] {
	c.entries = singleEntry[K, E](entry)
	return c
}

// Exec executes the query against the provided transaction. It returns any errors
// encountered during execution.
func (c Create[K, E]) Exec(ctx context.Context, tx Tx) error {
	checkForNilTx("Create.Exec", tx)
	w := WrapWriter[K, E](tx)
	if len(c.onUpdate) == 0 {
		return w.Set(ctx, c.entries.All()...)
	}
	r := WrapReader[K, E](tx)
	all := c.entries.All()
	toWrite := make([]E, 0, len(all))
	for _, entry := range all {
		e, err := r.Get(ctx, entry.GorpKey())
		if errors.Is(err, query.NotFound) {
			toWrite = append(toWrite, entry)
			continue
		}
		if err != nil {
			return err
		}
		if e, err = c.onUpdate.exec(Context{
			Context: ctx,
			Tx:      tx,
		}, entry, e); err != nil {
			return err
		}
		toWrite = append(toWrite, e)
	}
	return w.Set(ctx, toWrite...)
}

type MergeExistingFunc[K Key, E Entry[K]] = func(ctx Context, creating, existing E) (E, error)

type onUpdate[K Key, E Entry[K]] []MergeExistingFunc[K, E]

func (o onUpdate[K, E]) exec(ctx Context, creating, existing E) (E, error) {
	var err error
	for _, f := range o {
		if creating, err = f(ctx, creating, existing); err != nil {
			return creating, err
		}
	}
	return creating, nil
}
