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
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

// setField is the member of the JSON form of a Struct that replaces the field whole.
const setField = "set"

// Struct is the change that an update makes to a struct field: none, a merge of the
// partial U, or a replacement by the whole T. The zero Struct makes no change. Its JSON
// form is absent for no change, an object with only a "set" member for a replacement,
// and the partial otherwise. A struct field that holds a Struct needs the same options
// as one that holds a Change. An optional struct field holds a Change of a Struct,
// whose null clears the field.
type Struct[U, T any] struct {
	merge U
	value T
	op    changeOp
}

// Merge returns the Struct that merges u into a field of type T.
func Merge[T, U any](u U) Struct[U, T] { return Struct[U, T]{merge: u, op: opMerge} }

// Replace returns the Struct that replaces a field with v.
func Replace[U, T any](v T) Struct[U, T] { return Struct[U, T]{value: v, op: opSet} }

// Merged returns the partial that s merges, and false when s merges nothing.
func (s Struct[U, T]) Merged() (U, bool) { return s.merge, s.op == opMerge }

// Replaced returns the value that s replaces the field with, and false when s does
// not replace the field.
func (s Struct[U, T]) Replaced() (T, bool) { return s.value, s.op == opSet }

// IsZero reports whether s makes no change.
func (s Struct[U, T]) IsZero() bool { return s.op == opNone }

// MarshalJSONTo implements json.MarshalerTo. It fails for a Struct that makes no
// change, which only an omitzero field leaves out.
func (s Struct[U, T]) MarshalJSONTo(enc *jsontext.Encoder) error {
	switch s.op {
	case opMerge:
		return json.MarshalEncode(enc, s.merge)
	case opSet:
		return json.MarshalEncode(enc, struct {
			Set T `json:"set"`
		}{Set: s.value})
	default:
		return errUnset
	}
}

// UnmarshalJSONFrom implements json.UnmarshalerFrom. It returns an error wrapping
// validate.ErrValidation for a value that is not an object, or for an object that
// holds "set" and any other member.
func (s *Struct[U, T]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	raw, err := dec.ReadValue()
	if err != nil {
		return err
	}
	if raw.Kind() != '{' {
		return errors.Wrapf(validate.ErrValidation, "expected an object, got %s", raw)
	}
	replaces, err := holdsSet(raw)
	if err != nil {
		return err
	}
	if replaces {
		var v struct {
			Set T `json:"set"`
		}
		if err = json.Unmarshal(raw, &v, dec.Options()); err != nil {
			return err
		}
		*s = Replace[U](v.Set)
		return nil
	}
	var u U
	if err = json.Unmarshal(raw, &u, dec.Options()); err != nil {
		return err
	}
	*s = Merge[T](u)
	return nil
}

// holdsSet reports whether the JSON object raw holds a "set" member, and fails when it
// holds one beside any other member.
func holdsSet(raw jsontext.Value) (bool, error) {
	dec := jsontext.NewDecoder(bytes.NewReader(raw))
	if _, err := dec.ReadToken(); err != nil {
		return false, err
	}
	var members int
	var replaces bool
	for dec.PeekKind() != '}' {
		name, err := dec.ReadToken()
		if err != nil {
			return false, err
		}
		replaces = replaces || name.String() == setField
		if err = dec.SkipValue(); err != nil {
			return false, err
		}
		members++
	}
	if replaces && members > 1 {
		return false, errors.Wrapf(
			validate.ErrValidation,
			"an object that holds %q cannot hold other members",
			setField,
		)
	}
	return replaces, nil
}

// EncodeMsgpack implements msgpack.CustomEncoder.
func (s Struct[U, T]) EncodeMsgpack(enc *msgpack.Encoder) error {
	return EncodeMsgpack(enc, s)
}

// DecodeMsgpack implements msgpack.CustomDecoder.
func (s *Struct[U, T]) DecodeMsgpack(dec *msgpack.Decoder) error {
	return DecodeMsgpack(dec, s)
}
