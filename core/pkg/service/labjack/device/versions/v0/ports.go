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
)

var (
	_ msgpack.CustomEncoder = Ports{}
	_ msgpack.CustomDecoder = (*Ports)(nil)
)

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its model under "model", the same shape as the JSON encoding.
func (p Ports) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := p.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case T4Ports:
		return enc.Encode(struct {
			Model Model `msgpack:"model"`
			T4Ports
		}{Model: ModelT4, T4Ports: v})
	case T7Ports:
		return enc.Encode(struct {
			Model Model `msgpack:"model"`
			T7Ports
		}{Model: ModelT7, T7Ports: v})
	case T8Ports:
		return enc.Encode(struct {
			Model Model `msgpack:"model"`
			T8Ports
		}{Model: ModelT8, T8Ports: v})
	default:
		return errors.Newf("Ports: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. It returns a validation error for a
// model no variant has.
func (p *Ports) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	var disc struct {
		Model Model `msgpack:"model"`
	}
	if err = msgpack.Unmarshal(raw, &disc); err != nil {
		return err
	}
	switch disc.Model {
	case ModelT4:
		var v T4Ports
		err = msgpack.Unmarshal(raw, &v)
		p.Variant = v
	case ModelT7:
		var v T7Ports
		err = msgpack.Unmarshal(raw, &v)
		p.Variant = v
	case ModelT8:
		var v T8Ports
		err = msgpack.Unmarshal(raw, &v)
		p.Variant = v
	default:
		return errors.Wrapf(validate.ErrValidation, "unknown model %q", disc.Model)
	}
	return err
}
