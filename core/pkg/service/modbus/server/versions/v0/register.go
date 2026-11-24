// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
	"github.com/vmihailenco/msgpack/v5/msgpcode"
)

var (
	_ msgpack.CustomEncoder = Value{}
	_ msgpack.CustomDecoder = (*Value)(nil)
	_ msgpack.CustomEncoder = Register{}
	_ msgpack.CustomDecoder = (*Register)(nil)
)

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its type under "type", the same shape as the JSON encoding.
func (val Value) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := val.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case WholeValue:
		return enc.Encode(struct {
			Type ValueType `msgpack:"type"`
			WholeValue
		}{Type: WholeValueType, WholeValue: v})
	case BitValue:
		return enc.Encode(struct {
			Type ValueType `msgpack:"type"`
			BitValue
		}{Type: BitValueType, BitValue: v})
	case StringValue:
		return enc.Encode(struct {
			Type ValueType `msgpack:"type"`
			StringValue
		}{Type: StringValueType, StringValue: v})
	default:
		return errors.Newf("Value: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. A nil value clears the variant, and
// an omitted field takes its default. It returns a validation error for an unknown
// type.
func (val *Value) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, typ, err := decodeTagged[ValueType](dec)
	if err != nil || raw == nil {
		val.Variant = nil
		return err
	}
	switch typ {
	case WholeValueType:
		var v WholeValue
		v.SetDefaults()
		err = msgpack.Unmarshal(raw, &v)
		val.Variant = v
	case BitValueType:
		var v BitValue
		err = msgpack.Unmarshal(raw, &v)
		val.Variant = v
	case StringValueType:
		var v StringValue
		err = msgpack.Unmarshal(raw, &v)
		val.Variant = v
	default:
		return errors.Wrapf(validate.ErrValidation, "unknown value type %q", typ)
	}
	return err
}

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its type under "type", the same shape as the JSON encoding.
func (reg Register) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := reg.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case CoilRegister:
		return enc.Encode(struct {
			Type RegisterType `msgpack:"type"`
			CoilRegister
		}{Type: CoilRegisterType, CoilRegister: v})
	case DiscreteInputRegister:
		return enc.Encode(struct {
			Type RegisterType `msgpack:"type"`
			DiscreteInputRegister
		}{Type: DiscreteInputRegisterType, DiscreteInputRegister: v})
	case HoldingRegisterRegister:
		return enc.Encode(struct {
			Type RegisterType `msgpack:"type"`
			HoldingRegisterRegister
		}{Type: HoldingRegisterRegisterType, HoldingRegisterRegister: v})
	case InputRegisterRegister:
		return enc.Encode(struct {
			Type RegisterType `msgpack:"type"`
			InputRegisterRegister
		}{Type: InputRegisterRegisterType, InputRegisterRegister: v})
	default:
		return errors.Newf("Register: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. A nil value clears the variant. It
// returns a validation error for an unknown type.
func (reg *Register) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, typ, err := decodeTagged[RegisterType](dec)
	if err != nil || raw == nil {
		reg.Variant = nil
		return err
	}
	switch typ {
	case CoilRegisterType:
		var v CoilRegister
		err = msgpack.Unmarshal(raw, &v)
		reg.Variant = v
	case DiscreteInputRegisterType:
		var v DiscreteInputRegister
		err = msgpack.Unmarshal(raw, &v)
		reg.Variant = v
	case HoldingRegisterRegisterType:
		var v HoldingRegisterRegister
		err = msgpack.Unmarshal(raw, &v)
		reg.Variant = v
	case InputRegisterRegisterType:
		var v InputRegisterRegister
		err = msgpack.Unmarshal(raw, &v)
		reg.Variant = v
	default:
		return errors.Wrapf(validate.ErrValidation, "unknown register type %q", typ)
	}
	return err
}

// decodeTagged reads the next value and its "type" tag. It returns a nil raw value for
// a msgpack nil.
func decodeTagged[T ~string](dec *msgpack.Decoder) ([]byte, T, error) {
	raw, err := dec.DecodeRaw()
	if err != nil || (len(raw) == 1 && raw[0] == msgpcode.Nil) {
		return nil, "", err
	}
	var tag struct {
		Type T `msgpack:"type"`
	}
	err = msgpack.Unmarshal(raw, &tag)
	return raw, tag.Type, err
}
