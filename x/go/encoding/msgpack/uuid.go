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

// A UUID is represented as its 16 raw bytes. uuid.UUID marshals only to text, which
// would widen every stored UUID to a 36 character string and reject records written by
// earlier versions. Decoding accepts both widths. The representation is installed at
// load time because it must hold for the whole process: msgpack.Marshal and
// msgpack.Unmarshal encode without going through Codec.
func init() {
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
