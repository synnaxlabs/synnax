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
	"bytes"
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/types"
)

// Key is a unique key for an entry of a particular type.
type Key any

// Entry is a go type that can be queried against a DB. All go types must implement the Entry
// interface so that they can be stored. Entry must be serializable by the Encodings and decoder
// provided to the WithEncoderDecoder option when instantiating a DB.
type Entry[K Key] interface {
	// GorpKey returns a unique key for the entry. gorp.DB will not raise
	// an error if the key is a duplicate. Key must be serializable by encoder and decoder.
	GorpKey() K
	// SetOptions returns a slice of options passed to kv.db.set.
	SetOptions() []interface{}
}

const entriesOptKey query.Parameter = "entries"

// Entries is a query option used to bind entities from a retrieve query or
// write values to a create query.
type Entries[K Key, E Entry[K]] struct {
	// multiple is a boolean flag indicating whether the query expects one or multiple
	// entities.
	multiple bool
	// entry is used when the client expects/passes a single entry.
	entry *E
	// entries are used when the client expects/passes a slice of entries.
	entries *[]E
}

// Add adds the provided entry to the query. If the client expects a single result,
// sets the entry to the provided value. If the client expects a slice of entries,
// appends the provided value to the slice.
func (e *Entries[K, E]) Add(entry E) {
	if e.multiple {
		*e.entries = append(*e.entries, entry)
	} else if e.entry != nil {
		*e.entry = entry
	}
}

func (e *Entries[K, E]) Replace(entries []E) {
	if e.multiple {
		*e.entries = entries
		return
	}
	if len(entries) != 0 && e.entry != nil {
		*e.entry = entries[0]
	}
}

func (e *Entries[K, E]) Set(i int, entry E) {
	if e.multiple {
		(*e.entries)[i] = entry
	} else if i == 0 {
		*e.entry = entry
	}
}

// All returns a slice of all entries currently bound to the query.
func (e *Entries[K, E]) All() []E {
	if e.multiple {
		return *e.entries
	}
	return []E{*e.entry}
}

func (e *Entries[K, E]) Any() bool {
	if e.multiple {
		return len(*e.entries) > 0
	}
	return e.entry != nil
}

// SetEntry sets the entry that the query will fill results into or write results to.
//
//	Calls to SetEntry will override All previous calls to SetEntry or SetEntries.
func SetEntry[K Key, E Entry[K]](q query.Parameters, entry *E) {
	q.Set(entriesOptKey, &Entries[K, E]{entry: entry, multiple: false})
}

// SetEntries sets the entries that the query will fill results into or write results to.
// Calls to SetEntries will override All previous calls to SetEntry or SetEntries.
func SetEntries[K Key, E Entry[K]](q query.Parameters, e *[]E) {
	q.Set(entriesOptKey, &Entries[K, E]{entries: e, multiple: true})
}

// GetEntries returns the entries that the query will fill results into or write
// results from.
func GetEntries[K Key, E Entry[K]](q query.Parameters) *Entries[K, E] {
	re, ok := q.Get(entriesOptKey)
	if !ok {
		SetEntries[K](q, &[]E{})
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

func prefixMatcher[K Key, E Entry[K]](opts Tools) func(ctx context.Context, b []byte) bool {
	var (
		prefix_   []byte
		getPrefix = func(ctx context.Context) []byte {
			if prefix_ == nil {
				prefix_ = prefix[K, E](ctx, opts)
			}
			return prefix_
		}
	)
	return func(ctx context.Context, b []byte) bool {
		return bytes.HasPrefix(b, getPrefix(ctx))
	}
}

func encodeKey[K Key](
	ctx context.Context,
	encoder binary.Encoder,
	prefix []byte,
	key K,
) ([]byte, error) {
	byteKey, err := encoder.Encode(ctx, key)
	if err != nil {
		return nil, err
	}
	return append(prefix, byteKey...), nil
}
