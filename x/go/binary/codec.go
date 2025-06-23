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
	"reflect"
	"strconv"

	"github.com/synnaxlabs/x/errors"
)

// sugarEncodingErr adds additional context to encoding errors.
func sugarEncodingErr(value any, err error) error {
	if err == nil {
		return err
	}
	val := reflect.ValueOf(value)
	return errors.Wrapf(err, "failed to encode value: kind=%s, type=%s, value=%+v", val.Kind(), val.Type(), value)
}

// sugarDecodingErr adds additional context to decoding errors.
func sugarDecodingErr(data []byte, value any, err error) error {
	if err == nil {
		return err
	}
	val := reflect.ValueOf(value)
	return errors.Wrapf(err, "failed to decode into value: kind=%s, type=%s, data=%x", val.Kind(), val.Type(), data)
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

// PassThroughCodec wraps a Codec and checks for values that are already encoded
// ([]byte) and returns them as is.
type PassThroughCodec struct{ Codec }

// Encode implements the Encoder interface.
func (enc *PassThroughCodec) Encode(ctx context.Context, value any) ([]byte, error) {
	if bv, ok := value.([]byte); ok {
		return bv, nil
	}
	return enc.Codec.Encode(ctx, value)
}

// Decode implements the Decoder interface.
func (enc *PassThroughCodec) Decode(ctx context.Context, data []byte, value any) error {
	return enc.DecodeStream(ctx, bytes.NewReader(data), value)
}

// DecodeStream implements the Decoder interface.
func (enc *PassThroughCodec) DecodeStream(ctx context.Context, r io.Reader, value any) error {
	if bv, ok := value.(*[]byte); ok {
		*bv, _ = io.ReadAll(r)
		return nil
	}
	return enc.Codec.DecodeStream(ctx, r, value)
}

// EncodeStream implements the Encoder interface.
func (enc *PassThroughCodec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	return enc.Codec.EncodeStream(ctx, w, value)
}

// MarshalStringInt64 marshals the int64 value to a UTF-8 string.
func MarshalStringInt64(n int64) ([]byte, error) {
	return []byte(`"` + strconv.Itoa(int(n)) + `"`), nil
}

// MarshalStringUint64 marshals the uint64 value to a UTF-8 string.
func MarshalStringUint64(n uint64) ([]byte, error) {
	return []byte(`"` + strconv.FormatUint(n, 10) + `"`), nil
}

// decodeFallbackCodec wraps a set of Codecs. When the first Codec in the chain fails to
// decode a value, it falls back to the next Codec in the chain.
type decodeFallbackCodec struct {
	// Codecs is the list of codecs to fallback on.
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
