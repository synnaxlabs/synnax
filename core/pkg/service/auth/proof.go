// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package auth

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	_ msgpack.CustomEncoder = Proof{}
	_ msgpack.CustomDecoder = (*Proof)(nil)
)

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its method under "method", the same shape as the JSON encoding.
func (p Proof) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := p.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case PasswordProof:
		return enc.Encode(struct {
			Method   Method `msgpack:"method"`
			Username string `msgpack:"username"`
			Password string `msgpack:"password"`
		}{Method: MethodPassword, Username: v.Username, Password: v.Password})
	case APIKeyProof:
		return enc.Encode(struct {
			Method Method `msgpack:"method"`
			Key    string `msgpack:"key"`
		}{Method: MethodAPIKey, Key: v.Key})
	default:
		return errors.Newf("Proof: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. It returns a validation error for a
// method no variant has.
func (p *Proof) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	var disc struct {
		Method Method `msgpack:"method"`
	}
	if err := msgpack.Unmarshal(raw, &disc); err != nil {
		return err
	}
	switch disc.Method {
	case MethodPassword:
		var creds Credentials
		if err := msgpack.Unmarshal(raw, &creds); err != nil {
			return err
		}
		p.Variant = PasswordProof{Credentials: creds}
	case MethodAPIKey:
		var creds APIKeyCredentials
		if err := msgpack.Unmarshal(raw, &creds); err != nil {
			return err
		}
		p.Variant = APIKeyProof{APIKeyCredentials: creds}
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"unknown proof method %q",
			disc.Method,
		)
	}
	return nil
}
