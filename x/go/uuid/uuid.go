// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package uuid

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

type UUID struct{ uuid.UUID }

var (
	_ json.Marshaler        = (*UUID)(nil)
	_ json.Unmarshaler      = (*UUID)(nil)
	_ msgpack.CustomEncoder = UUID{}
	_ msgpack.CustomDecoder = (*UUID)(nil)
)

func (u UUID) MarshalJSON() ([]byte, error) {
	return u.Bytes(), nil
}

func (u *UUID) UnmarshalJSON(data []byte) error {
	u_, err := uuid.FromBytes(data)
	u.UUID = u_
	return err
}

func (u UUID) EncodeMsgpack(enc *msgpack.Encoder) error {
	return enc.EncodeString(u.String())
}

func (u *UUID) DecodeMsgpack(dec *msgpack.Decoder) error {
	str, err := dec.DecodeString()
	if err != nil {
		return err
	}
	*u, err = Parse(str)
	return err
}

func New() UUID { return UUID{uuid.New()} }

func NewString() string { return uuid.NewString() }

func (u UUID) Bytes() []byte { return u.UUID[:] }

func Parse(s string) (UUID, error) {
	u, err := uuid.Parse(s)
	return UUID{u}, err
}

func MustParse(s string) UUID { return UUID{uuid.MustParse(s)} }

var Nil = UUID{uuid.Nil}
