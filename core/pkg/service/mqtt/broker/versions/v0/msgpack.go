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
	case TcpTransport:
		return enc.Encode(struct {
			Type TransportType `msgpack:"type"`
		}{Type: TcpTransportType})
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
	case TcpTransportType:
		u.Variant = TcpTransport{}
	case TlsTransportType:
		var v TlsTransport
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	default:
		return errors.Wrapf(validate.ErrValidation, "unknown transport type %q", disc.Type)
	}
	return nil
}

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
	case UsernameAuth:
		return enc.Encode(struct {
			Type AuthType `msgpack:"type"`
			UsernameAuth
		}{Type: UsernameAuthType, UsernameAuth: v})
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
		u.Variant = NoneAuth{}
	case UsernameAuthType:
		var v UsernameAuth
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	default:
		return errors.Wrapf(validate.ErrValidation, "unknown auth type %q", disc.Type)
	}
	return nil
}

var (
	_ msgpack.CustomEncoder = SparkplugMode{}
	_ msgpack.CustomDecoder = (*SparkplugMode)(nil)
)

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its type under "type", the same shape as the JSON encoding.
func (u SparkplugMode) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := u.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case PassiveSparkplugMode:
		return enc.Encode(struct {
			Type SparkplugModeType `msgpack:"type"`
		}{Type: PassiveSparkplugModeType})
	case PrimarySparkplugMode:
		return enc.Encode(struct {
			Type SparkplugModeType `msgpack:"type"`
			PrimarySparkplugMode
		}{Type: PrimarySparkplugModeType, PrimarySparkplugMode: v})
	default:
		return errors.Newf("SparkplugMode: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. A nil value clears the variant, and
// an omitted field takes its default, as in the JSON encoding. It returns a validation
// error for an unknown type.
func (u *SparkplugMode) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if len(raw) == 1 && raw[0] == msgpcode.Nil {
		u.Variant = nil
		return nil
	}
	var disc struct {
		Type SparkplugModeType `msgpack:"type"`
	}
	if err = msgpack.Unmarshal(raw, &disc); err != nil {
		return err
	}
	switch disc.Type {
	case PassiveSparkplugModeType:
		u.Variant = PassiveSparkplugMode{}
	case PrimarySparkplugModeType:
		var v PrimarySparkplugMode
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	default:
		return errors.Wrapf(validate.ErrValidation, "unknown Sparkplug mode type %q", disc.Type)
	}
	return nil
}

var (
	_ msgpack.CustomEncoder = Read{}
	_ msgpack.CustomDecoder = (*Read)(nil)
)

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its type under "type", the same shape as the JSON encoding.
func (u Read) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := u.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case PlainRead:
		return enc.Encode(struct {
			Type ReadType `msgpack:"type"`
			PlainRead
		}{Type: PlainReadType, PlainRead: v})
	case SparkplugRead:
		return enc.Encode(struct {
			Type ReadType `msgpack:"type"`
			SparkplugRead
		}{Type: SparkplugReadType, SparkplugRead: v})
	default:
		return errors.Newf("Read: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. A nil value clears the variant, and
// an omitted field takes its default, as in the JSON encoding. It returns a validation
// error for an unknown type.
func (u *Read) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if len(raw) == 1 && raw[0] == msgpcode.Nil {
		u.Variant = nil
		return nil
	}
	var disc struct {
		Type ReadType `msgpack:"type"`
	}
	if err = msgpack.Unmarshal(raw, &disc); err != nil {
		return err
	}
	switch disc.Type {
	case PlainReadType:
		var v PlainRead
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	case SparkplugReadType:
		var v SparkplugRead
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	default:
		return errors.Wrapf(validate.ErrValidation, "unknown read type %q", disc.Type)
	}
	return nil
}

var (
	_ msgpack.CustomEncoder = Write{}
	_ msgpack.CustomDecoder = (*Write)(nil)
)

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its type under "type", the same shape as the JSON encoding.
func (u Write) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := u.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case PlainWrite:
		return enc.Encode(struct {
			Type WriteType `msgpack:"type"`
			PlainWrite
		}{Type: PlainWriteType, PlainWrite: v})
	case SparkplugWrite:
		return enc.Encode(struct {
			Type WriteType `msgpack:"type"`
			SparkplugWrite
		}{Type: SparkplugWriteType, SparkplugWrite: v})
	default:
		return errors.Newf("Write: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. A nil value clears the variant, and
// an omitted field takes its default, as in the JSON encoding. It returns a validation
// error for an unknown type.
func (u *Write) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if len(raw) == 1 && raw[0] == msgpcode.Nil {
		u.Variant = nil
		return nil
	}
	var disc struct {
		Type WriteType `msgpack:"type"`
	}
	if err = msgpack.Unmarshal(raw, &disc); err != nil {
		return err
	}
	switch disc.Type {
	case PlainWriteType:
		var v PlainWrite
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	case SparkplugWriteType:
		var v SparkplugWrite
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		u.Variant = v
	default:
		return errors.Wrapf(validate.ErrValidation, "unknown write type %q", disc.Type)
	}
	return nil
}
