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

	"github.com/synnaxlabs/alamos"
)

// TracingCodec wraps a Codec and traces the encoding and decoding operations.
type TracingCodec struct {
	alamos.Instrumentation
	Level alamos.Environment
	Codec
}

var _ Codec = (*TracingCodec)(nil)

// Encode implements the Encoder interface.
func (tc *TracingCodec) Encode(ctx context.Context, value any) ([]byte, error) {
	ctx, span := tc.T.Trace(ctx, "encode", tc.Level)
	b, err := tc.Codec.Encode(ctx, value)
	if err != nil {
		return nil, sugarEncodingErr(value, err)
	}
	return b, span.EndWith(err)
}

// Decode implements the Decoder interface.
func (tc *TracingCodec) Decode(ctx context.Context, data []byte, value any) error {
	ctx, span := tc.T.Trace(ctx, "decode", tc.Level)
	err := tc.Codec.Decode(ctx, data, value)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return span.EndWith(err)
}

// DecodeStream implements the Decoder interface.
func (tc *TracingCodec) DecodeStream(ctx context.Context, r io.Reader, value any) error {
	ctx, span := tc.T.Trace(ctx, "decode_stream", tc.Level)
	err := tc.Codec.DecodeStream(ctx, r, value)
	if err != nil {
		data, _ := io.ReadAll(r)
		err = sugarDecodingErr(data, value, err)
	}
	return span.EndWith(err)
}

// EncodeStream implements the Encoder interface.
func (tc *TracingCodec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	ctx, span := tc.T.Trace(ctx, "encode_stream", tc.Level)
	err := tc.Codec.EncodeStream(ctx, w, value)
	if err != nil {
		err = sugarEncodingErr(value, err)
	}
	return span.EndWith(err)
}
