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
	"context"
	"io"

	"github.com/vmihailenco/msgpack/v5"
)

// MsgPackCodec is a MessagePack implementation of Codec.
var MsgPackCodec = &msgPackCodec{}

type msgPackCodec struct{}

var _ Codec = (*msgPackCodec)(nil)

// Encode implements the Encoder interface.
func (mpc *msgPackCodec) Encode(_ context.Context, value any) ([]byte, error) {
	b, err := msgpack.Marshal(value)
	return b, SugarEncodingErr(value, err)
}

// Decode implements the Decoder interface.
func (mpc *msgPackCodec) Decode(ctx context.Context, data []byte, value any) error {
	return WrapStreamDecoder(mpc, ctx, data, value)
}

// DecodeStream implements the Decoder interface.
func (mpc *msgPackCodec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := msgpack.NewDecoder(r).Decode(value); err != nil {
		data, _ := io.ReadAll(r)
		return SugarDecodingErr(data, value, err)
	}
	return nil
}

// EncodeStream implements the Encoder interface.
func (mpc *msgPackCodec) EncodeStream(
	ctx context.Context,
	w io.Writer,
	value any,
) error {
	b, err := mpc.Encode(ctx, value)
	if err != nil {
		return SugarEncodingErr(value, err)
	}
	_, err = w.Write(b)
	return SugarEncodingErr(value, err)
}
