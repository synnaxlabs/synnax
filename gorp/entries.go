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
	"github.com/synnaxlabs/x/types"
)

// Key is a unique key for an entry of a particular type.
type Key interface {
	types.Primitive
	comparable
}

// Entry is a go type that can be queried against a DB. All go types must implement the
// Entry interface so that they can be stored. Entry must be serializable by the
// Encodings and decoder provided to the WithCodec option when instantiating a DB.
type Entry[K Key] interface {
	// GorpKey returns a unique key for the entry. gorp.DB will not raise an error if
	// the key is a duplicate. Key must be serializable by encoder and decoder.
	GorpKey() K
	// SetOptions returns a slice of options passed to kv.db.set.
	SetOptions() []any
}

// Entries is a query option used to bind entities from a retrieve query or
// write values to a create query.
type Entries[K Key, E Entry[K]] struct {
	// entry is set when the client binds a single entry. Mutually
	// exclusive with entries.
	entry *E
	// entries is set when the client binds a slice. Mutually exclusive
	// with entry.
	entries *[]E
	// changes counts the number of writes applied to the bound
	// entry/entries since the last bind.
	changes int
	// isMultiple is true when the query was bound via bindMultiple.
	isMultiple bool
}

func (e *Entries[K, E]) add(entry E) {
	if e.isMultiple {
		*e.entries = append(*e.entries, entry)
	} else if e.entry != nil {
		*e.entry = entry
	}
	e.changes++
}

// MapInPlace iterates over all entries in the provided query and applies the given
// function to each entry. If the function returns true, the entry will be replaced
// with the new entry. If the function returns false, the entry will be removed from
// the query. If the function returns an error, the iteration will stop and the error
// will be returned.
func (e *Entries[K, E]) MapInPlace(f func(E) (E, bool, error)) error {
	if e.isMultiple {
		nEntries := make([]E, 0, len(*e.entries))
		for _, entry := range *e.entries {
			n, ok, err := f(entry)
			if err != nil {
				return err
			}
			if ok {
				nEntries = append(nEntries, n)
			}
		}
		*e.entries = nEntries
		e.changes += len(nEntries)
		return nil
	}
	if e.entry == nil {
		return nil
	}
	n, ok, err := f(*e.entry)
	if err != nil {
		return err
	}
	if ok {
		*e.entry = n
	} else {
		e.entry = nil
	}
	e.changes++
	return nil
}

// All returns a slice of all entries currently bound to the query.
func (e *Entries[K, E]) All() []E {
	if e.isMultiple {
		if e.entries == nil {
			return nil
		}
		return *e.entries
	}
	if e.entry == nil {
		return nil
	}
	return []E{*e.entry}
}

// Bound returns true if entries binding was set on the query.
func (e *Entries[K, E]) Bound() bool {
	if e.isMultiple {
		return e.entries != nil
	}
	return e.entry != nil
}

// IsMultiple returns true if multiple entries were bound to the query.
func (e *Entries[K, E]) IsMultiple() bool { return e.isMultiple }

func (e *Entries[K, E]) bindMultiple(entries *[]E) {
	e.entries = entries
	e.entry = nil
	e.isMultiple = true
	e.changes = 0
}

func (e *Entries[K, E]) bindSingle(entry *E) {
	e.entry = entry
	e.entries = nil
	e.isMultiple = false
	e.changes = 0
}

func (e *Entries[K, E]) ensureCap(n int) {
	if !e.isMultiple || e.entries == nil {
		return
	}
	if cap(*e.entries) < n {
		*e.entries = make([]E, 0, n)
		return
	}
	*e.entries = (*e.entries)[:0]
}
