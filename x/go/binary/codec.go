// Copyright 2026 Synnax Labs, Inc.
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
	"reflect"

	"github.com/synnaxlabs/x/errors"
)

var (
	ErrDecode = errors.New("failed to decode")
	ErrEncode = errors.New("failed to encode")
)

// SugarEncodingErr adds additional context to encoding errors.
func SugarEncodingErr(value any, base error) error {
	if base == nil {
		return base
	}
	val := reflect.ValueOf(value)
	main := errors.Wrapf(ErrEncode, "failed to encode value: kind=%s, type=%s, value=%+v", val.Kind(), val.Type(), value)
	return errors.WithStack(errors.Combine(main, base))
}

// SugarDecodingErr adds additional context to decoding errors.
func SugarDecodingErr(data []byte, value any, base error) error {
	if base == nil {
		return base
	}
	val := reflect.ValueOf(value)
	main := errors.Wrapf(ErrDecode, "kind=%s, type=%s, data=%x", val.Kind(), val.Type(), data)
	return errors.WithStack(errors.Combine(main, base))
}

// Codec is an interface that encodes and decodes values.
type Codec interface {
	Encoder
	Decoder
}

// Encoder encodes values into binary.
type Encoder interface {
	// Encode encodes the value into binary. It returns the encoded value along with any
	// errors encountered.
	Encode(ctx context.Context, value any) ([]byte, error)
	// EncodeStream encodes the value into binary and writes it to the given writer. It
	// returns any errors encountered.
	EncodeStream(ctx context.Context, w io.Writer, value any) error
}

// Decoder decodes values from binary.
type Decoder interface {
	// Decode decodes data into a pointer value.
	Decode(ctx context.Context, data []byte, value any) error
	// DecodeStream decodes data from the given reader into a pointer value.
	DecodeStream(ctx context.Context, r io.Reader, value any) error
}

// decodeFallbackCodec wraps a set of Codecs. When the first Codec in the chain fails to
// decode a value, it falls back to the next Codec in the chain.
type decodeFallbackCodec struct {
	Codecs []Codec
}

func NewDecodeFallbackCodec(base Codec, codecs ...Codec) Codec {
	return &decodeFallbackCodec{Codecs: append([]Codec{base}, codecs...)}
}

var _ Codec = (*decodeFallbackCodec)(nil)

// Encode implements the Encoder interface.
func (f *decodeFallbackCodec) Encode(ctx context.Context, value any) ([]byte, error) {
	return f.Codecs[0].Encode(ctx, value)
}

func (f *decodeFallbackCodec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	return f.Codecs[0].EncodeStream(ctx, w, value)
}

// Decode implements the Decoder interface.
func (f *decodeFallbackCodec) Decode(ctx context.Context, data []byte, value any) error {
	for _, c := range f.Codecs {
		if err := c.Decode(ctx, data, value); err == nil {
			return err
		}
	}
	return nil
}

// DecodeStream implements the Decoder interface.
func (f *decodeFallbackCodec) DecodeStream(
	ctx context.Context,
	r io.Reader,
	value any,
) error {
	if len(f.Codecs) == 0 {
		panic("[binary] - no codecs provided to decodeFallbackCodec")
	}
	// We need to read out all the data here, otherwise an initial codec that fails will
	// leave the reader in a bad state. It's not ideal, but we need to do it.
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	for _, c := range f.Codecs {
		if err = c.DecodeStream(ctx, bytes.NewReader(data), value); err == nil {
			return err
		}
	}
	return err
}
