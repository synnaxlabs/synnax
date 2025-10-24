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
type Create[K Key, E Entry[K]] struct{ params query.Parameters }

// NewCreate opens a new Create query.
func NewCreate[K Key, E Entry[K]]() Create[K, E] {
	return Create[K, E]{params: make(query.Parameters)}
}

// MergeExisting adds a function to the query that can be used to prevent the accidental
// override of existing entries with the same GorpKey. The provided function receives
// the existing entry and should return an error if the entry should not be overwritten.
// If no entry with a matching GorpKey is found, the function is not called. MergeExisting
// adds overhead to the query, as a retrieval is required to check for existing entries.
func (c Create[K, E]) MergeExisting(filter func(ctx Context, creating E, existing E) (E, error)) Create[K, E] {
	addMergeExisting[K, E](c.params, filter)
	return c
}

// Entries sets the Entries to write to the DB.
func (c Create[K, E]) Entries(entries *[]E) Create[K, E] { SetEntries[K](c.params, entries); return c }

// Entry sets the entry to write to the DB.
func (c Create[K, E]) Entry(entry *E) Create[K, E] { SetEntry[K](c.params, entry); return c }

// Exec executes the Inputs against the provided DB. It returns any errors encountered during execution.
func (c Create[K, E]) Exec(ctx context.Context, tx Tx) error {
	checkForNilTx("Create.Exec", tx)
	entries, w := GetEntries[K, E](c.params), WrapWriter[K, E](tx)
	mergeExisting, hasMergeExisting := getMergeExisting[K, E](c.params)
	if hasMergeExisting {
		r := WrapReader[K, E](tx)
		for i, entry := range entries.All() {
			e, err := r.Get(ctx, entry.GorpKey())
			if errors.Is(err, query.NotFound) {
				continue
			}
			if err != nil {
				return err
			}
			if e, err = mergeExisting.exec(Context{
				Context: ctx,
				Tx:      tx,
			}, entry, e); err != nil {
				return err
			}
			entries.Set(i, e)
		}
	}
	return w.Set(ctx, entries.All()...)
}

const mergeExistingKey = "mergeExisting"

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

func addMergeExisting[K Key, E Entry[K]](q query.Parameters, f MergeExistingFunc[K, E]) {
	var o onUpdate[K, E]
	ro, ok := q.Get(mergeExistingKey)
	if !ok {
		o = make(onUpdate[K, E], 0, 1)
	} else {
		o = ro.(onUpdate[K, E])
	}
	o = append(o, f)
	q.Set(mergeExistingKey, o)
}

func getMergeExisting[K Key, E Entry[K]](q query.Parameters) (o onUpdate[K, E], ok bool) {
	ro, ok := q.Get(mergeExistingKey)
	if !ok {
		return nil, false
	}
	return ro.(onUpdate[K, E]), true
}
