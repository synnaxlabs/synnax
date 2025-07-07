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
	"encoding/gob"
	"io"
)

var _ Codec = (*GobCodec)(nil)

// GobCodec is a gob implementation of the Codec interface.
type GobCodec struct{}

// Encode implements the Encoder interface.
func (g *GobCodec) Encode(_ context.Context, value any) ([]byte, error) {
	var (
		buff bytes.Buffer
		err  = gob.NewEncoder(&buff).Encode(value)
		b    = buff.Bytes()
	)
	if err != nil {
		return nil, sugarEncodingErr(value, err)
	}
	return b, nil
}

// EncodeStream implements the Encoder interface.
func (g *GobCodec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	err := gob.NewEncoder(w).Encode(value)
	if err != nil {
		return sugarEncodingErr(value, err)
	}
	return nil
}

// Decode implements the Decoder interface.
func (g *GobCodec) Decode(ctx context.Context, data []byte, value any) error {
	err := g.DecodeStream(ctx, bytes.NewReader(data), value)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream implements the Decoder interface.
func (g *GobCodec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := gob.NewDecoder(r).Decode(value); err != nil {
		data, _ := io.ReadAll(r)
		return sugarDecodingErr(data, value, err)
	}
	return nil
}
