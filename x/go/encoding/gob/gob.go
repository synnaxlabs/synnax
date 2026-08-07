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
	"encoding/gob"
	"io"

	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
)

// Codec is a gob implementation of encoding.Codec.
var Codec encoding.Codec = &codec{}

type codec struct{}

func (c *codec) Decode(data []byte, value any) error {
	return c.DecodeStream(bytes.NewReader(data), value)
}

func (*codec) DecodeStream(r io.Reader, value any) error {
	if err := gob.NewDecoder(r).Decode(value); err != nil {
		data, ioErr := io.ReadAll(r)
		return encoding.SugarDecodingError(data, value, errors.Combine(err, ioErr))
	}
	return nil
}

func (c *codec) Encode(value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := c.EncodeStream(&buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (*codec) EncodeStream(w io.Writer, value any) error {
	err := gob.NewEncoder(w).Encode(value)
	if err != nil {
		return encoding.SugarEncodingError(value, err)
	}
	return nil
}
