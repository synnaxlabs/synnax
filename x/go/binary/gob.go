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
	"encoding/gob"
	"io"
)

// GobCodec is a gob implementation of the Codec interface.
var GobCodec = &gobCodec{}

type gobCodec struct{}

var _ Codec = (*gobCodec)(nil)

// Encode implements the Encoder interface.
func (gc *gobCodec) Encode(ctx context.Context, value any) ([]byte, error) {
	return WrapStreamEncoder(gc, ctx, value)
}

// EncodeStream implements the Encoder interface.
func (gc *gobCodec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	err := gob.NewEncoder(w).Encode(value)
	return SugarEncodingErr(value, err)
}

// Decode implements the Decoder interface.
func (gc *gobCodec) Decode(ctx context.Context, data []byte, value any) error {
	return WrapStreamDecoder(gc, ctx, data, value)
}

// DecodeStream implements the Decoder interface.
func (gc *gobCodec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := gob.NewDecoder(r).Decode(value); err != nil {
		data, _ := io.ReadAll(r)
		return SugarDecodingErr(data, value, err)
	}
	return nil
}
