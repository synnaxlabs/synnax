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
	"bytes"
	"encoding/binary"
	"unsafe"

	"github.com/synnaxlabs/x/types"
	"go.uber.org/zap"
)

// Key is a unique key for an entry of a particular type.
type Key types.Primitive

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

// Entries is a query option used to bind entities from a retrieve query or
// write values to a create query.
type Entries[K Key, E Entry[K]] struct {
	// entry is used when the client expects/passes a single entry.
	entry *E
	// entries are used when the client expects/passes a slice of entries.
	entries *[]E
	changes int
	// isMultiple is a boolean flag indicating whether the query expects one or isMultiple
	// entities.
	isMultiple bool
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
	} else if i == 0 && e.entry != nil {
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

// Keys returns the keys of all entries currently bound to the query.
func (e *Entries[K, E]) Keys() []K {
	return entryKeys(e.All())
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

func singleEntry[K Key, E Entry[K]](entry *E) *Entries[K, E] {
	return &Entries[K, E]{entry: entry, isMultiple: false}
}

func multipleEntries[K Key, E Entry[K]](entries *[]E) *Entries[K, E] {
	return &Entries[K, E]{entries: entries, isMultiple: true}
}

const magicPrefix = "__gorp__//"

type keyCodec[K Key, E Entry[K]] struct {
	prefix  []byte
	keySize int
	buf     []byte
}

func newKeyCodec[K Key, E Entry[K]]() *keyCodec[K, E] {
	c := &keyCodec[K, E]{prefix: []byte(magicPrefix + types.Name[E]())}
	var zero K
	switch any(zero).(type) {
	case string, []byte:
	default:
		c.keySize = int(unsafe.Sizeof(zero))
		c.buf = make([]byte, len(c.prefix)+c.keySize)
		copy(c.buf, c.prefix)
	}
	return c
}

func (k *keyCodec[K, E]) encode(key K) []byte {
	if k.keySize > 0 {
		k.putBigEndian(k.buf[len(k.prefix):], key)
		return k.buf
	}
	switch v := any(key).(type) {
	case string:
		out := make([]byte, len(k.prefix)+len(v))
		copy(out, k.prefix)
		copy(out[len(k.prefix):], v)
		return out
	case []byte:
		out := make([]byte, len(k.prefix)+len(v))
		copy(out, k.prefix)
		copy(out[len(k.prefix):], v)
		return out
	default:
		panic("unreachable")
	}
}

func (k *keyCodec[K, E]) decode(b []byte) K {
	b = b[len(k.prefix):]
	if k.keySize > 0 {
		return k.getBigEndian(b)
	}
	var zero K
	switch any(zero).(type) {
	case string:
		return any(string(b)).(K)
	case []byte:
		return any(bytes.Clone(b)).(K)
	default:
		panic("unreachable")
	}
}

func (k *keyCodec[K, E]) matchPrefix(prefix []byte, key K) bool {
	if len(prefix) == 0 {
		return true
	}
	if k.keySize > 0 {
		if len(prefix) > k.keySize {
			return false
		}
		src := unsafe.Slice((*byte)(unsafe.Pointer(&key)), k.keySize)
		for i := range len(prefix) {
			if src[k.keySize-1-i] != prefix[i] {
				return false
			}
		}
		return true
	}
	switch v := any(key).(type) {
	case string:
		return len(v) >= len(prefix) && string(prefix) == v[:len(prefix)]
	case []byte:
		return bytes.HasPrefix(v, prefix)
	default:
		panic("unreachable")
	}
}

func (k *keyCodec[K, E]) putBigEndian(dst []byte, key K) {
	switch k.keySize {
	case 1:
		dst[0] = *(*byte)(unsafe.Pointer(&key))
	case 2:
		binary.BigEndian.PutUint16(dst, *(*uint16)(unsafe.Pointer(&key)))
	case 4:
		binary.BigEndian.PutUint32(dst, *(*uint32)(unsafe.Pointer(&key)))
	case 8:
		binary.BigEndian.PutUint64(dst, *(*uint64)(unsafe.Pointer(&key)))
	default:
		src := unsafe.Slice((*byte)(unsafe.Pointer(&key)), k.keySize)
		for i := range k.keySize {
			dst[i] = src[k.keySize-1-i]
		}
	}
}

func (k *keyCodec[K, E]) getBigEndian(b []byte) K {
	var out K
	switch k.keySize {
	case 1:
		*(*byte)(unsafe.Pointer(&out)) = b[0]
	case 2:
		*(*uint16)(unsafe.Pointer(&out)) = binary.BigEndian.Uint16(b)
	case 4:
		*(*uint32)(unsafe.Pointer(&out)) = binary.BigEndian.Uint32(b)
	case 8:
		*(*uint64)(unsafe.Pointer(&out)) = binary.BigEndian.Uint64(b)
	default:
		dst := unsafe.Slice((*byte)(unsafe.Pointer(&out)), k.keySize)
		for i := range k.keySize {
			dst[i] = b[k.keySize-1-i]
		}
	}
	return out
}
