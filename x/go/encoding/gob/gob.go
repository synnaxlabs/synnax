// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gob

import (
	"bytes"
	"context"
	"encoding/gob"
	"io"

	"github.com/synnaxlabs/x/binary"
)

var _ binary.Codec = (*Codec)(nil)

// Codec is a gob implementation of the binary.Codec interface.
type Codec struct{}

// Encode implements the binary.Encoder interface.
func (e *Codec) Encode(_ context.Context, value any) ([]byte, error) {
	var (
		buff bytes.Buffer
		err  = gob.NewEncoder(&buff).Encode(value)
		b    = buff.Bytes()
	)
	if err != nil {
		return nil, binary.SugarEncodingErr(value, err)
	}
	return b, nil
}

// EncodeStream implements the binary.Encoder interface.
func (e *Codec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	err := gob.NewEncoder(w).Encode(value)
	if err != nil {
		return binary.SugarEncodingErr(value, err)
	}
	return nil
}

// Decode implements the binary.Decoder interface.
func (e *Codec) Decode(ctx context.Context, data []byte, value any) error {
	err := e.DecodeStream(ctx, bytes.NewReader(data), value)
	if err != nil {
		return binary.SugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream implements the binary.Decoder interface.
func (e *Codec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := gob.NewDecoder(r).Decode(value); err != nil {
		data, _ := io.ReadAll(r)
		return binary.SugarDecodingErr(data, value, err)
	}
	return nil
}
