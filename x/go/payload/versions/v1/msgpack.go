// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

var (
	_ msgpack.CustomEncoder = Scalar{}
	_ msgpack.CustomDecoder = (*Scalar)(nil)
)

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its type under "type", the same shape as the JSON encoding.
func (u Scalar) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := u.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case NumberScalar:
		return enc.Encode(struct {
			Type ScalarType `msgpack:"type"`
			NumberScalar
		}{Type: NumberScalarType, NumberScalar: v})
	case StringScalar:
		return enc.Encode(struct {
			Type ScalarType `msgpack:"type"`
			StringScalar
		}{Type: StringScalarType, StringScalar: v})
	case BooleanScalar:
		return enc.Encode(struct {
			Type ScalarType `msgpack:"type"`
			BooleanScalar
		}{Type: BooleanScalarType, BooleanScalar: v})
	case NullScalar:
		return enc.Encode(struct {
			Type ScalarType `msgpack:"type"`
		}{Type: NullScalarType})
	default:
		return errors.Newf("Scalar: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. A nil value clears the variant. It
// returns a validation error for an unknown type.
func (u *Scalar) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if len(raw) == 1 && raw[0] == msgpcode.Nil {
		u.Variant = nil
		return nil
	}
	var disc struct {
		Type ScalarType `msgpack:"type"`
	}
	if err = msgpack.Unmarshal(raw, &disc); err != nil {
		return err
	}
	switch disc.Type {
	case NumberScalarType:
		var v NumberScalar
		err = msgpack.Unmarshal(raw, &v)
		u.Variant = v
	case StringScalarType:
		var v StringScalar
		err = msgpack.Unmarshal(raw, &v)
		u.Variant = v
	case BooleanScalarType:
		var v BooleanScalar
		err = msgpack.Unmarshal(raw, &v)
		u.Variant = v
	case NullScalarType:
		u.Variant = NullScalar{}
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"unknown scalar type %q",
			disc.Type,
		)
	}
	return err
}
