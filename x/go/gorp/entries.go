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

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/types"
	"go.uber.org/zap"
)

// Key is a unique key for an entry of a particular type.
type Key any

// Entry is a go type that can be queried against a DB. All go types must implement the Entry
// interface so that they can be stored. Entry must be serializable by the Encodings and decoder
// provided to the WithCodec option when instantiating a DB.
type Entry[K Key] interface {
	// GorpKey returns a unique key for the entry. gorp.DB will not raise
	// an error if the key is a duplicate. Key must be serializable by encoder and decoder.
	GorpKey() K
	// SetOptions returns a slice of options passed to kv.db.set.
	SetOptions() []any
}

func entryKeys[K Key, E Entry[K]](entries []E) []K {
	keys := make([]K, len(entries))
	for i, entry := range entries {
		keys[i] = entry.GorpKey()
	}
	return keys
}

const entriesOptKey query.Parameter = "entries"

// Entries is a query option used to bind entities from a retrieve query or
// write values to a create query.
type Entries[K Key, E Entry[K]] struct {
	// isMultiple is a boolean flag indicating whether the query expects one or isMultiple
	// entities.
	isMultiple bool
	// entry is used when the client expects/passes a single entry.
	entry *E
	// entries are used when the client expects/passes a slice of entries.
	entries *[]E
	changes int
}

// Add adds the provided entry to the query. If the client expects a single result,
// sets the entry to the provided value. If the client expects a slice of entries,
// appends the provided value to the slice.
func (e *Entries[K, E]) Add(entry E) {
	if e.isMultiple {
		*e.entries = append(*e.entries, entry)
	} else if e.entry != nil {
		*e.entry = entry
	}
	e.changes++
}

// Replace replaces the entries in the query with the provided entries. If Entries
// holds a single entry, the first entry in the slice will be used.
func (e *Entries[K, E]) Replace(entries []E) {
	if e.isMultiple {
		*e.entries = entries
		e.changes += len(entries)
		return
	}
	if len(entries) != 0 && e.entry != nil {
		*e.entry = entries[0]
		e.changes++
	}
}

// Set sets the entry at the provided index to the provided value. If the query expects
// a single entry, the index must be 0.
func (e *Entries[K, E]) Set(i int, entry E) {
	if e.isMultiple {
		if len(*e.entries) <= i {
			zap.S().DPanic("[gorp.Entries.Set] - index out of range")
			return
		}
		(*e.entries)[i] = entry
		e.changes++
	} else if i == 0 {
		*e.entry = entry
		e.changes++
	}
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

// Keys returns the keys of all entries currently bound to the query.
func (e *Entries[K, E]) Keys() []K {
	return entryKeys(e.All())
}

func (e *Entries[K, E]) Any() bool {
	if e.isMultiple {
		return len(*e.entries) > 0
	}
	return e.entry != nil
}

func (e *Entries[K, E]) IsMultiple() bool { return e.isMultiple }

// SetEntry sets the entry that the query will fill results into or write results to.
//
//	Calls to SetEntry will override All previous calls to SetEntry or SetEntries.
func SetEntry[K Key, E Entry[K]](q query.Parameters, entry *E) {
	q.Set(entriesOptKey, &Entries[K, E]{entry: entry, isMultiple: false})
}

// SetEntries sets the entries that the query will fill results into or write results to.
// Calls to SetEntries will override All previous calls to SetEntry or SetEntries.
func SetEntries[K Key, E Entry[K]](q query.Parameters, e *[]E) {
	q.Set(entriesOptKey, &Entries[K, E]{entries: e, isMultiple: true})
}

// GetEntries returns the entries that the query will fill results into or write
// results from.
func GetEntries[K Key, E Entry[K]](q query.Parameters) *Entries[K, E] {
	re, ok := q.Get(entriesOptKey)
	if !ok {
		SetEntries(q, &[]E{})
		return GetEntries[K, E](q)
	}
	return re.(*Entries[K, E])
}

func prefix[K Key, E Entry[K]](ctx context.Context, encoder binary.Encoder) []byte {
	return lo.Must(encoder.Encode(ctx, types.Name[E]()))
}

type lazyPrefix[K Key, E Entry[K]] struct {
	_prefix []byte
	Tools
}

func (lp *lazyPrefix[K, E]) prefix(ctx context.Context) []byte {
	if lp._prefix == nil {
		lp._prefix = prefix[K, E](ctx, lp)
	}
	return lp._prefix
}

func encodeKey[K Key](
	ctx context.Context,
	encoder binary.Encoder,
	prefix []byte,
	key K,
) ([]byte, error) {
	// if the key is already a byte slice, we can just append it to the prefix
	if b, ok := any(key).([]byte); ok {
		return append(prefix, b...), nil
	}
	byteKey, err := encoder.Encode(ctx, key)
	if err != nil {
		return nil, err
	}
	return append(prefix, byteKey...), nil
}

func decodeKey[K Key](
	ctx context.Context,
	decoder binary.Decoder,
	prefix []byte,
	b []byte,
) (v K, err error) {
	// if the key is a byte slice, we can just return it
	if _, ok := any(v).([]byte); ok {
		return any(b[len(prefix):]).(K), nil
	}
	return v, decoder.Decode(ctx, b[len(prefix):], &v)
}
