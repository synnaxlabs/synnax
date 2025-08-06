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

// decodeFallbackCodec wraps a set of Codecs. When the first Codec in the chain fails to
// decode a value, it falls back to the next Codec in the chain.
type decodeFallbackCodec struct {
	// Codecs is the list of codecs to fallback on.
	Codecs []Codec
}

var _ Codec = (*decodeFallbackCodec)(nil)

func NewDecodeFallbackCodec(base Codec, codecs ...Codec) *decodeFallbackCodec {
	return &decodeFallbackCodec{Codecs: append([]Codec{base}, codecs...)}
}

// Encode implements the Encoder interface.
func (dfc *decodeFallbackCodec) Encode(ctx context.Context, value any) ([]byte, error) {
	return dfc.Codecs[0].Encode(ctx, value)
}

func (dfc *decodeFallbackCodec) EncodeStream(
	ctx context.Context,
	w io.Writer,
	value any,
) error {
	return dfc.Codecs[0].EncodeStream(ctx, w, value)
}

// Decode implements the Decoder interface.
func (dfc *decodeFallbackCodec) Decode(
	ctx context.Context,
	data []byte,
	value any,
) error {
	var err error
	for _, c := range dfc.Codecs {
		if err = c.Decode(ctx, data, value); err == nil {
			return nil
		}
	}
	return err
}

// DecodeStream implements the Decoder interface.
func (dfc *decodeFallbackCodec) DecodeStream(
	ctx context.Context,
	r io.Reader,
	value any,
) error {
	if len(dfc.Codecs) == 0 {
		panic("[binary] - no codecs provided to decodeFallbackCodec")
	}
	// We need to read out all the data here, otherwise an initial codec that fails will
	// leave the reader in a bad state. It's not ideal, but we need to do it.
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	for _, c := range dfc.Codecs {
		if err = c.DecodeStream(ctx, bytes.NewReader(data), value); err == nil {
			return nil
		}
	}
	return err
}
