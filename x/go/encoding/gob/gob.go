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

	"github.com/synnaxlabs/x/encoding"
)

// Codec is a gob implementation of the encoding.Codec interface.
var Codec = &codec{}

type codec struct{}

// Encode implements the encoding.Encoder interface.
func (e *codec) Encode(_ context.Context, value any) ([]byte, error) {
	var (
		buff bytes.Buffer
		err  = gob.NewEncoder(&buff).Encode(value)
		b    = buff.Bytes()
	)
	if err != nil {
		return nil, encoding.SugarEncodingErr(value, err)
	}
	return b, nil
}

// EncodeStream implements the encoding.Encoder interface.
func (e *codec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	err := gob.NewEncoder(w).Encode(value)
	if err != nil {
		return encoding.SugarEncodingErr(value, err)
	}
	return nil
}

// Decode implements the encoding.Decoder interface.
func (e *codec) Decode(ctx context.Context, data []byte, value any) error {
	err := e.DecodeStream(ctx, bytes.NewReader(data), value)
	if err != nil {
		return encoding.SugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream implements the encoding.Decoder interface.
func (e *codec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := gob.NewDecoder(r).Decode(value); err != nil {
		data, _ := io.ReadAll(r)
		return encoding.SugarDecodingErr(data, value, err)
	}
	return nil
}
