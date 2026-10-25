// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package key implements the on-disk key format of Gorp entries.
package key

import (
	"context"
	"encoding/binary"
	"reflect"
	"slices"
	"unsafe"

	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/types"
)

const (
	tablePrefix     = "gorp."
	migrationPrefix = "gorp.migration."
)

// Prefix returns the prefix of every key in the table of entry type E.
func Prefix[E any]() []byte { return []byte(tablePrefix + types.Name[E]()) }

// Migrations returns the key that records the migrations applied to namespace.
func Migrations(namespace string) []byte { return []byte(migrationPrefix + namespace) }

// LegacyPrefix returns the key prefix that releases before v0.54 used for the table of
// typeName: the type name encoded with msgpack.
func LegacyPrefix(ctx context.Context, typeName string) ([]byte, error) {
	return msgpack.Codec.Encode(ctx, typeName)
}

// primitive is a primary key type that a Codec can encode.
type primitive interface {
	types.Primitive
	comparable
}

type kind uint8

const (
	kindFixed kind = iota
	kindString
)

// Codec encodes primary keys of type K under one table prefix and decodes them back.
// Fixed-size keys encode big-endian, and string keys encode as their raw bytes. A Codec
// is not safe for concurrent use: Encode reuses one buffer, so its result is valid only
// until the next call.
type Codec[K primitive] struct {
	// prefix is the table prefix that starts every encoded key.
	prefix []byte
	// size is the byte width of a fixed-size key. Zero for string keys.
	size int
	// buf is the reused encode buffer.
	buf  []byte
	kind kind
}

// NewCodec returns a Codec that encodes keys under prefix.
func NewCodec[K primitive](prefix []byte) Codec[K] {
	c := Codec[K]{prefix: prefix}
	var zero K
	if reflect.TypeOf(zero).Kind() == reflect.String {
		c.kind = kindString
		return c
	}
	c.kind = kindFixed
	c.size = int(unsafe.Sizeof(zero))
	c.buf = make([]byte, len(prefix)+c.size)
	copy(c.buf, prefix)
	return c
}

// Prefix returns the table prefix that starts every key the Codec encodes.
func (c *Codec[K]) Prefix() []byte { return c.prefix }

// Encode returns the full key for k.
func (c *Codec[K]) Encode(k K) []byte {
	switch c.kind {
	case kindFixed:
		c.putBigEndian(c.buf[len(c.prefix):], k)
		return c.buf
	case kindString:
		s := *(*string)(unsafe.Pointer(&k))
		n := len(c.prefix) + len(s)
		c.buf = slices.Grow(c.buf[:0], n)[:n]
		copy(c.buf, c.prefix)
		copy(c.buf[len(c.prefix):], s)
		return c.buf
	default:
		panic("unreachable")
	}
}

// Decode returns the primary key of b, a full key that Encode produced.
func (c *Codec[K]) Decode(b []byte) K {
	b = b[len(c.prefix):]
	switch c.kind {
	case kindFixed:
		return c.getBigEndian(b)
	case kindString:
		s := string(b)
		return *(*K)(unsafe.Pointer(&s))
	default:
		panic("unreachable")
	}
}

// HasPrefix reports whether the encoding of k, without the table prefix, starts with
// prefix.
func (c *Codec[K]) HasPrefix(k K, prefix []byte) bool {
	if len(prefix) == 0 {
		return true
	}
	switch c.kind {
	case kindFixed:
		if len(prefix) > c.size {
			return false
		}
		src := unsafe.Slice((*byte)(unsafe.Pointer(&k)), c.size)
		for i := range len(prefix) {
			if src[c.size-1-i] != prefix[i] {
				return false
			}
		}
		return true
	case kindString:
		s := *(*string)(unsafe.Pointer(&k))
		return len(s) >= len(prefix) && string(prefix) == s[:len(prefix)]
	default:
		panic("unreachable")
	}
}

func (c *Codec[K]) putBigEndian(dst []byte, k K) {
	switch c.size {
	case 1:
		dst[0] = *(*byte)(unsafe.Pointer(&k))
	case 2:
		binary.BigEndian.PutUint16(dst, *(*uint16)(unsafe.Pointer(&k)))
	case 4:
		binary.BigEndian.PutUint32(dst, *(*uint32)(unsafe.Pointer(&k)))
	case 8:
		binary.BigEndian.PutUint64(dst, *(*uint64)(unsafe.Pointer(&k)))
	default:
		src := unsafe.Slice((*byte)(unsafe.Pointer(&k)), c.size)
		for i := range c.size {
			dst[i] = src[c.size-1-i]
		}
	}
}

func (c *Codec[K]) getBigEndian(b []byte) K {
	var out K
	switch c.size {
	case 1:
		*(*byte)(unsafe.Pointer(&out)) = b[0]
	case 2:
		*(*uint16)(unsafe.Pointer(&out)) = binary.BigEndian.Uint16(b)
	case 4:
		*(*uint32)(unsafe.Pointer(&out)) = binary.BigEndian.Uint32(b)
	case 8:
		*(*uint64)(unsafe.Pointer(&out)) = binary.BigEndian.Uint64(b)
	default:
		dst := unsafe.Slice((*byte)(unsafe.Pointer(&out)), c.size)
		for i := range c.size {
			dst[i] = b[c.size-1-i]
		}
	}
	return out
}
