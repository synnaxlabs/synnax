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
	_ msgpack.CustomEncoder = Connection{}
	_ msgpack.CustomDecoder = (*Connection)(nil)
)

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its transport under "transport", the same shape as the JSON encoding.
func (c Connection) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := c.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case TcpConnection:
		return enc.Encode(struct {
			Transport ConnectionType `msgpack:"transport"`
			TcpConnection
		}{Transport: TcpConnectionType, TcpConnection: v})
	case RtuConnection:
		return enc.Encode(struct {
			Transport ConnectionType `msgpack:"transport"`
			RtuConnection
		}{Transport: RtuConnectionType, RtuConnection: v})
	default:
		return errors.Newf("Connection: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. A nil value clears the variant, and
// an omitted field takes its default, as in the JSON encoding. It returns a validation
// error for an unknown transport.
func (c *Connection) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	if len(raw) == 1 && raw[0] == msgpcode.Nil {
		c.Variant = nil
		return nil
	}
	var disc struct {
		Transport ConnectionType `msgpack:"transport"`
	}
	if err = msgpack.Unmarshal(raw, &disc); err != nil {
		return err
	}
	switch disc.Transport {
	case TcpConnectionType:
		var v TcpConnection
		v.SetDefaults()
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		c.Variant = v
	case RtuConnectionType:
		var v RtuConnection
		v.SetDefaults()
		if err = msgpack.Unmarshal(raw, &v); err != nil {
			return err
		}
		c.Variant = v
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"unknown connection transport %q",
			disc.Transport,
		)
	}
	return nil
}
