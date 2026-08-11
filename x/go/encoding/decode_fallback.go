// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package encoding

import (
	"bytes"
	"context"
	"io"

	"github.com/synnaxlabs/x/errors"
)

type decodeFallbackCodec struct{ codecs []Codec }

// NewDecodeFallbackCodec creates a new Codec that wraps the given Codec and falls back
// to the next Codec in the chain if the first Codec fails to decode a value.
func NewDecodeFallbackCodec(base Codec, codecs ...Codec) Codec {
	return &decodeFallbackCodec{codecs: append([]Codec{base}, codecs...)}
}

func (f *decodeFallbackCodec) Encode(ctx context.Context, value any) ([]byte, error) {
	return f.codecs[0].Encode(ctx, value)
}

func (f *decodeFallbackCodec) EncodeStream(
	ctx context.Context,
	w io.Writer,
	value any,
) error {
	return f.codecs[0].EncodeStream(ctx, w, value)
}

func (f *decodeFallbackCodec) Decode(
	ctx context.Context,
	data []byte,
	value any,
) error {
	var errs []error
	for _, c := range f.codecs {
		if err := c.Decode(ctx, data, value); err != nil {
			errs = append(errs, err)
		} else {
			return nil
		}
	}
	return errors.Wrap(errors.Join(errs...), "all codecs failed to decode")
}

func (f *decodeFallbackCodec) DecodeStream(
	ctx context.Context,
	r io.Reader,
	value any,
) error {
	// We need to read out all the data here, otherwise an initial codec that fails will
	// leave the reader in a bad state. It's not ideal, but we need to do it.
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	var errs []error
	for _, c := range f.codecs {
		if err = c.DecodeStream(ctx, bytes.NewReader(data), value); err != nil {
			errs = append(errs, err)
		} else {
			return nil
		}
	}
	return errors.Wrap(errors.Join(errs...), "all codecs failed to decode")
}
