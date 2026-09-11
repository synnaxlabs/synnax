// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package msgpack

import (
	"reflect"
	"uuid"

	"github.com/synnaxlabs/x/errors"
	"github.com/vmihailenco/msgpack/v5"
)

// registerUUID represents a UUID as its 16 raw bytes. uuid.UUID marshals only to
// text, which would widen every stored UUID to a 36 character string and reject
// records written by earlier versions. Decoding accepts both widths.
func registerUUID() {
	msgpack.Register(
		uuid.UUID{},
		func(enc *msgpack.Encoder, v reflect.Value) error {
			u, ok := reflect.TypeAssert[uuid.UUID](v)
			if !ok {
				return errors.Newf("expected a uuid, got %s", v.Type())
			}
			return enc.EncodeBytes(u[:])
		},
		func(dec *msgpack.Decoder, v reflect.Value) error {
			b, err := dec.DecodeBytes()
			if err != nil {
				return err
			}
			u, err := decodeUUID(b)
			if err != nil {
				return err
			}
			v.Set(reflect.ValueOf(u))
			return nil
		},
	)
}

// decodeUUID reads a UUID from its 16 byte form or from any textual form that
// uuid.Parse accepts.
func decodeUUID(b []byte) (uuid.UUID, error) {
	if len(b) == len(uuid.UUID{}) {
		return uuid.UUID(b), nil
	}
	u, err := uuid.Parse(string(b))
	return u, errors.Wrapf(err, "failed to decode uuid from %d bytes", len(b))
}
