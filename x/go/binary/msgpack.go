// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package binary

import (
	"bytes"
	"context"
	"io"

	"github.com/vmihailenco/msgpack/v5"
)

var _ Codec = (*MsgPackCodec)(nil)

// MsgPackCodec is a msgpack implementation of Codec.
type MsgPackCodec struct{}

// Encode implements the Encoder interface.
func (m *MsgPackCodec) Encode(_ context.Context, value any) ([]byte, error) {
	b, err := msgpack.Marshal(value)
	return b, sugarEncodingErr(value, err)
}

// Decode implements the Decoder interface.
func (m *MsgPackCodec) Decode(ctx context.Context, data []byte, value any) error {
	err := m.DecodeStream(ctx, bytes.NewReader(data), value)
	return sugarDecodingErr(data, value, err)
}

// DecodeStream implements the Decoder interface.
func (m *MsgPackCodec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := msgpack.NewDecoder(r).Decode(value); err != nil {
		data, _ := io.ReadAll(r)
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// EncodeStream implements the Encoder interface.
func (m *MsgPackCodec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	b, err := m.Encode(ctx, value)
	if err != nil {
		return sugarEncodingErr(value, err)
	}
	_, err = w.Write(b)
	return sugarEncodingErr(value, err)
}
