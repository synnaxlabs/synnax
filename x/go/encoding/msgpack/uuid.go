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

// RegisterUUID makes MessagePack encode a UUID as its 16 raw bytes, for the whole
// process. Without it, uuid.UUID marshals to its 36 character text, and records that
// earlier versions wrote fail to decode. Decoding accepts both widths. Call it once at
// the start of a program, before any encoding: msgpack.Marshal and msgpack.Unmarshal
// encode without going through Codec.
func RegisterUUID() {
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
			var u uuid.UUID
			if len(b) == 16 {
				u = uuid.UUID(b)
			} else if u, err = uuid.Parse(string(b)); err != nil {
				return errors.Wrapf(err, "failed to decode uuid from %d bytes", len(b))
			}
			v.Set(reflect.ValueOf(u))
			return nil
		},
	)
}
