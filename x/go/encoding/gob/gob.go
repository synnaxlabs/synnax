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
	"github.com/synnaxlabs/x/errors"
)

// Codec is a gob implementation of encoding.Codec.
var Codec encoding.Codec = &codec{}

type codec struct{}

func (c *codec) Decode(ctx context.Context, data []byte, value any) error {
	return c.DecodeStream(ctx, bytes.NewReader(data), value)
}

func (*codec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := gob.NewDecoder(r).Decode(value); err != nil {
		data, ioErr := io.ReadAll(r)
		return encoding.SugarDecodingErr(data, value, errors.Combine(err, ioErr))
	}
	return nil
}

func (c *codec) Encode(ctx context.Context, value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := c.EncodeStream(ctx, &buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (*codec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	err := gob.NewEncoder(w).Encode(value)
	if err != nil {
		return encoding.SugarEncodingErr(value, err)
	}
	return nil
}
