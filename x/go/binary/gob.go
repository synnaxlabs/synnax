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

// GobCodec is a gob implementation of the Codec interface.
var GobCodec = &gobCodec{}

type gobCodec struct{}

var _ Codec = (*gobCodec)(nil)

// Encode implements the Encoder interface.
func (gc *gobCodec) Encode(ctx context.Context, value any) ([]byte, error) {
	var buff bytes.Buffer
	if err := gc.EncodeStream(ctx, &buff, value); err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

// EncodeStream implements the Encoder interface.
func (gc *gobCodec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	err := gob.NewEncoder(w).Encode(value)
	return sugarEncodingErr(value, err)
}

// Decode implements the Decoder interface.
func (gc *gobCodec) Decode(ctx context.Context, data []byte, value any) error {
	return gc.DecodeStream(ctx, bytes.NewReader(data), value)
}

// DecodeStream implements the Decoder interface.
func (gc *gobCodec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := gob.NewDecoder(r).Decode(value); err != nil {
		data, _ := io.ReadAll(r)
		return sugarDecodingErr(data, value, err)
	}
	return nil
}
