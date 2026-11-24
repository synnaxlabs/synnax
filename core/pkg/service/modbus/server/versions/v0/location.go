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
	_ msgpack.CustomEncoder = Location{}
	_ msgpack.CustomDecoder = (*Location)(nil)
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
	raw, typ, err := decodeTagged[ValueType](dec, "type")
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
// with its table under "table", the same shape as the JSON encoding.
func (loc Location) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := loc.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case CoilLocation:
		return enc.Encode(struct {
			Table LocationType `msgpack:"table"`
			CoilLocation
		}{Table: CoilLocationType, CoilLocation: v})
	case DiscreteInputLocation:
		return enc.Encode(struct {
			Table LocationType `msgpack:"table"`
			DiscreteInputLocation
		}{Table: DiscreteInputLocationType, DiscreteInputLocation: v})
	case HoldingRegisterLocation:
		return enc.Encode(struct {
			Table LocationType `msgpack:"table"`
			HoldingRegisterLocation
		}{Table: HoldingRegisterLocationType, HoldingRegisterLocation: v})
	case InputRegisterLocation:
		return enc.Encode(struct {
			Table LocationType `msgpack:"table"`
			InputRegisterLocation
		}{Table: InputRegisterLocationType, InputRegisterLocation: v})
	default:
		return errors.Newf("Location: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. A nil value clears the variant. It
// returns a validation error for an unknown type.
func (loc *Location) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, typ, err := decodeTagged[LocationType](dec, "table")
	if err != nil || raw == nil {
		loc.Variant = nil
		return err
	}
	switch typ {
	case CoilLocationType:
		var v CoilLocation
		err = msgpack.Unmarshal(raw, &v)
		loc.Variant = v
	case DiscreteInputLocationType:
		var v DiscreteInputLocation
		err = msgpack.Unmarshal(raw, &v)
		loc.Variant = v
	case HoldingRegisterLocationType:
		var v HoldingRegisterLocation
		err = msgpack.Unmarshal(raw, &v)
		loc.Variant = v
	case InputRegisterLocationType:
		var v InputRegisterLocation
		err = msgpack.Unmarshal(raw, &v)
		loc.Variant = v
	default:
		return errors.Wrapf(validate.ErrValidation, "unknown location table %q", typ)
	}
	return err
}

// decodeTagged reads the next value and the tag under key. It returns a nil raw value
// for a msgpack nil.
func decodeTagged[T ~string](dec *msgpack.Decoder, key string) ([]byte, T, error) {
	raw, err := dec.DecodeRaw()
	if err != nil || (len(raw) == 1 && raw[0] == msgpcode.Nil) {
		return nil, "", err
	}
	var fields map[string]msgpack.RawMessage
	if err = msgpack.Unmarshal(raw, &fields); err != nil {
		return nil, "", err
	}
	var tag T
	if t, ok := fields[key]; ok {
		err = msgpack.Unmarshal(t, &tag)
	}
	return raw, tag, err
}
