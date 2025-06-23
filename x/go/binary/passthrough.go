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
)

var _ Codec = (*PassThroughCodec)(nil)

// PassThroughCodec wraps a Codec and checks for values that are already encoded
// ([]byte) and returns them as is.
type PassThroughCodec struct{ Codec }

// Encode implements the Encoder interface.
func (p *PassThroughCodec) Encode(ctx context.Context, value any) ([]byte, error) {
	if bv, ok := value.([]byte); ok {
		return bv, nil
	}
	return p.Codec.Encode(ctx, value)
}

// Decode implements the Decoder interface.
func (p *PassThroughCodec) Decode(ctx context.Context, data []byte, value any) error {
	return p.DecodeStream(ctx, bytes.NewReader(data), value)
}

// DecodeStream implements the Decoder interface.
func (p *PassThroughCodec) DecodeStream(ctx context.Context, r io.Reader, value any) error {
	if bv, ok := value.(*[]byte); ok {
		*bv, _ = io.ReadAll(r)
		return nil
	}
	return p.Codec.DecodeStream(ctx, r, value)
}

// EncodeStream implements the Encoder interface.
func (p *PassThroughCodec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	return p.Codec.EncodeStream(ctx, w, value)
}
