// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package content hashes the content of typed records. Oracle generates the HashContent
// methods that call it.
package content

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"maps"
	"math"
	"slices"

	"github.com/cespare/xxhash/v2"
)

// keyField is the name of a record's key, which identifies the record and is not part
// of its content.
const keyField = "key"

// Hash accumulates a hash of the content of a record. A record writes each of its
// values except its keys in a fixed order, so records with the same content have the
// same hash. The zero Hash is not ready for use; call NewHash.
type Hash struct{ d xxhash.Digest }

// NewHash returns an empty Hash.
func NewHash() Hash {
	var h Hash
	h.d.Reset()
	return h
}

// Sum returns the hash as 16 lowercase hexadecimal characters.
func (h *Hash) Sum() string {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], h.d.Sum64())
	return hex.EncodeToString(b[:])
}

// Uint writes v.
func (h *Hash) Uint(v uint64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], v)
	check(h.d.Write(b[:]))
}

// Int writes v.
func (h *Hash) Int(v int64) { h.Uint(uint64(v)) }

// Float writes v.
func (h *Hash) Float(v float64) { h.Uint(math.Float64bits(v)) }

// Bool writes v.
func (h *Hash) Bool(v bool) {
	if v {
		h.Uint(1)
	} else {
		h.Uint(0)
	}
}

// Len writes the length of a list, so that lists hash apart from their neighbors.
func (h *Hash) Len(n int) { h.Uint(uint64(n)) }

// String writes s after its length.
func (h *Hash) String(s string) {
	h.Len(len(s))
	check(h.d.WriteString(s))
}

// Bytes writes b after its length.
func (h *Hash) Bytes(b []byte) {
	h.Len(len(b))
	check(h.d.Write(b))
}

// check panics on a write error, which an xxhash digest never returns.
func check(_ int, err error) {
	if err != nil {
		panic(err)
	}
}

const (
	tagNull uint64 = iota
	tagFalse
	tagTrue
	tagNumber
	tagString
	tagList
	tagObject
)

// Any writes the JSON value v. Object members named key are left out, at any depth.
func (h *Hash) Any(v any) {
	switch t := v.(type) {
	case nil:
		h.Uint(tagNull)
	case bool:
		if t {
			h.Uint(tagTrue)
		} else {
			h.Uint(tagFalse)
		}
	case float64:
		h.Uint(tagNumber)
		h.Float(t)
	case string:
		h.Uint(tagString)
		h.String(t)
	case []any:
		h.Uint(tagList)
		h.Len(len(t))
		for _, e := range t {
			h.Any(e)
		}
	case map[string]any:
		h.Uint(tagObject)
		names := slices.Sorted(maps.Keys(t))
		names = slices.DeleteFunc(names, func(n string) bool { return n == keyField })
		h.Len(len(names))
		for _, n := range names {
			h.String(n)
			h.Any(t[n])
		}
	default:
		panic(fmt.Sprintf("content: a JSON value cannot hold %T", v))
	}
}
