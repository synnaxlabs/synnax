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
	_ msgpack.CustomEncoder = Auth{}
	_ msgpack.CustomDecoder = (*Auth)(nil)
)

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its type under "type", the same shape as the JSON encoding.
func (u Auth) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := u.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case NoneAuth:
		return enc.Encode(struct {
			Type AuthType `msgpack:"type"`
		}{Type: NoneAuthType})
	case BearerAuth:
		return enc.Encode(struct {
			Type AuthType `msgpack:"type"`
			BearerAuth
		}{Type: BearerAuthType, BearerAuth: v})
	case BasicAuth:
		return enc.Encode(struct {
			Type AuthType `msgpack:"type"`
			BasicAuth
		}{Type: BasicAuthType, BasicAuth: v})
	case APIKeyAuth:
		return enc.Encode(struct {
			Type AuthType `msgpack:"type"`
			APIKeyAuth
		}{Type: APIKeyAuthType, APIKeyAuth: v})
	default:
		return errors.Newf("Auth: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. A nil value clears the variant, and
// an omitted field takes its default, as in the JSON encoding. It returns a validation
// error for an unknown type.
func (u *Auth) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if len(raw) == 1 && raw[0] == msgpcode.Nil {
		u.Variant = nil
		return nil
	}
	var disc struct {
		Type AuthType `msgpack:"type"`
	}
	if err = msgpack.Unmarshal(raw, &disc); err != nil {
		return err
	}
	switch disc.Type {
	case NoneAuthType:
		var v NoneAuth
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	case BearerAuthType:
		var v BearerAuth
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	case BasicAuthType:
		var v BasicAuth
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	case APIKeyAuthType:
		var v APIKeyAuth
		v.SetDefaults()
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"unknown auth type %q",
			disc.Type,
		)
	}
	return nil
}

var (
	_ msgpack.CustomEncoder = Response{}
	_ msgpack.CustomDecoder = (*Response)(nil)
)

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its type under "expected_value_type", the same shape as the JSON encoding.
func (u Response) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := u.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case StringResponse:
		return enc.Encode(struct {
			ExpectedValueType ResponseType `msgpack:"expected_value_type"`
			StringResponse
		}{ExpectedValueType: StringResponseType, StringResponse: v})
	case NumberResponse:
		return enc.Encode(struct {
			ExpectedValueType ResponseType `msgpack:"expected_value_type"`
			NumberResponse
		}{ExpectedValueType: NumberResponseType, NumberResponse: v})
	case BooleanResponse:
		return enc.Encode(struct {
			ExpectedValueType ResponseType `msgpack:"expected_value_type"`
			BooleanResponse
		}{ExpectedValueType: BooleanResponseType, BooleanResponse: v})
	case NullResponse:
		return enc.Encode(struct {
			ExpectedValueType ResponseType `msgpack:"expected_value_type"`
			NullResponse
		}{ExpectedValueType: NullResponseType, NullResponse: v})
	default:
		return errors.Newf("Response: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. A nil value clears the variant, and
// an omitted field takes its default, as in the JSON encoding. It returns a validation
// error for an unknown expected value type.
func (u *Response) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if len(raw) == 1 && raw[0] == msgpcode.Nil {
		u.Variant = nil
		return nil
	}
	var disc struct {
		ExpectedValueType ResponseType `msgpack:"expected_value_type"`
	}
	if err = msgpack.Unmarshal(raw, &disc); err != nil {
		return err
	}
	switch disc.ExpectedValueType {
	case StringResponseType:
		var v StringResponse
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	case NumberResponseType:
		var v NumberResponse
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	case BooleanResponseType:
		var v BooleanResponse
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	case NullResponseType:
		var v NullResponse
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"unknown response expected value type %q",
			disc.ExpectedValueType,
		)
	}
	return nil
}
