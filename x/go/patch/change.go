// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package patch

import (
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/synnaxlabs/x/errors"
	"github.com/vmihailenco/msgpack/v5"
)

// Change is the change that an update makes to one field: none, a new value, or a
// clear. The zero Change makes no change. Its JSON form is absent for no change, null
// for a clear, and the value otherwise, so a struct field that holds a Change needs the
// omitzero JSON option and the omitempty msgpack option, and the struct needs the
// msgpack methods that EncodeMsgpack describes.
type Change[T any] struct {
	value T
	op    changeOp
}

type changeOp uint8

const (
	opNone changeOp = iota
	opSet
	opClear
	opMerge
)

// To returns the Change that sets a field to v.
func To[T any](v T) Change[T] { return Change[T]{value: v, op: opSet} }

// Clear returns the Change that clears an optional field.
func Clear[T any]() Change[T] { return Change[T]{op: opClear} }

// Get returns the value that c sets, and false when c sets no value.
func (c Change[T]) Get() (T, bool) { return c.value, c.op == opSet }

// Cleared reports whether c clears the field.
func (c Change[T]) Cleared() bool { return c.op == opClear }

// IsZero reports whether c makes no change.
func (c Change[T]) IsZero() bool { return c.op == opNone }

// MarshalJSONTo implements json.MarshalerTo. It fails for a Change that makes no
// change, which only an omitzero field leaves out.
func (c Change[T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	switch c.op {
	case opSet:
		return json.MarshalEncode(enc, c.value)
	case opClear:
		return enc.WriteToken(jsontext.Null)
	default:
		return errUnset
	}
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom.
func (c *Change[T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if dec.PeekKind() == 'n' {
		if _, err := dec.ReadToken(); err != nil {
			return err
		}
		*c = Clear[T]()
		return nil
	}
	var v T
	if err := json.UnmarshalDecode(dec, &v); err != nil {
		return err
	}
	*c = To(v)
	return nil
}

// EncodeMsgpack implements msgpack.CustomEncoder.
func (c Change[T]) EncodeMsgpack(enc *msgpack.Encoder) error {
	return EncodeMsgpack(enc, c)
}

// DecodeMsgpack implements msgpack.CustomDecoder.
func (c *Change[T]) DecodeMsgpack(dec *msgpack.Decoder) error {
	return DecodeMsgpack(dec, c)
}

var errUnset = errors.New("a change that changes nothing has no wire form")
