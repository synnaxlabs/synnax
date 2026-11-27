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
	_ msgpack.CustomEncoder = Transport{}
	_ msgpack.CustomDecoder = (*Transport)(nil)
)

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its type under "type", the same shape as the JSON encoding.
func (u Transport) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := u.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case PlaintextTransport:
		return enc.Encode(struct {
			Type TransportType `msgpack:"type"`
		}{Type: PlaintextTransportType})
	case TlsTransport:
		return enc.Encode(struct {
			Type TransportType `msgpack:"type"`
			TlsTransport
		}{Type: TlsTransportType, TlsTransport: v})
	default:
		return errors.Newf("Transport: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. A nil value clears the variant, and
// an omitted field takes its default, as in the JSON encoding. It returns a validation
// error for an unknown type.
func (u *Transport) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if len(raw) == 1 && raw[0] == msgpcode.Nil {
		u.Variant = nil
		return nil
	}
	var disc struct {
		Type TransportType `msgpack:"type"`
	}
	if err = msgpack.Unmarshal(raw, &disc); err != nil {
		return err
	}
	switch disc.Type {
	case PlaintextTransportType:
		u.Variant = PlaintextTransport{}
	case TlsTransportType:
		var v TlsTransport
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"unknown transport type %q",
			disc.Type,
		)
	}
	return nil
}

var (
	_ msgpack.CustomEncoder = Auth{}
	_ msgpack.CustomDecoder = (*Auth)(nil)
)

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its mechanism under "mechanism", the same shape as the JSON encoding.
func (u Auth) EncodeMsgpack(enc *msgpack.Encoder) error {
	type withCredentials struct {
		Mechanism AuthType `msgpack:"mechanism"`
		Credentials
	}
	switch v := u.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case NoneAuth:
		return enc.Encode(struct {
			Mechanism AuthType `msgpack:"mechanism"`
		}{Mechanism: NoneAuthType})
	case PlainAuth:
		return enc.Encode(withCredentials{PlainAuthType, v.Credentials})
	case ScramSha256Auth:
		return enc.Encode(withCredentials{ScramSha256AuthType, v.Credentials})
	case ScramSha512Auth:
		return enc.Encode(withCredentials{ScramSha512AuthType, v.Credentials})
	default:
		return errors.Newf("Auth: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. A nil value clears the variant. It
// returns a validation error for an unknown mechanism.
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
		Mechanism AuthType `msgpack:"mechanism"`
	}
	if err = msgpack.Unmarshal(raw, &disc); err != nil {
		return err
	}
	switch disc.Mechanism {
	case NoneAuthType:
		u.Variant = NoneAuth{}
	case PlainAuthType:
		var v PlainAuth
		if err = msgpack.Unmarshal(raw, &v.Credentials); err != nil {
			return err
		}
		u.Variant = v
	case ScramSha256AuthType:
		var v ScramSha256Auth
		if err = msgpack.Unmarshal(raw, &v.Credentials); err != nil {
			return err
		}
		u.Variant = v
	case ScramSha512AuthType:
		var v ScramSha512Auth
		if err = msgpack.Unmarshal(raw, &v.Credentials); err != nil {
			return err
		}
		u.Variant = v
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"unknown auth mechanism %q",
			disc.Mechanism,
		)
	}
	return nil
}
