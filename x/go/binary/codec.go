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
	"encoding/json"
	"io"
	"reflect"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errors"
	"github.com/vmihailenco/msgpack/v5"
)

var (
	DecodeError = errors.New("failed to decode")
	EncodeError = errors.New("failed to encode")
)

// sugarEncodingErr adds additional context to encoding errors.
func sugarEncodingErr(value any, base error) error {
	if base == nil {
		return base
	}
	val := reflect.ValueOf(value)
	main := errors.Wrapf(EncodeError, "failed to encode value: kind=%s, type=%s, value=%+v", val.Kind(), val.Type(), value)
	return errors.Combine(main, base)
}

// sugarDecodingErr adds additional context to decoding errors.
func sugarDecodingErr(data []byte, value any, base error) error {
	if base == nil {
		return base
	}
	val := reflect.ValueOf(value)
	main := errors.Wrapf(DecodeError, "kind=%s, type=%s, data=%x", val.Kind(), val.Type(), data)
	return errors.Combine(main, base)
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

var (
	_ Codec = (*GobCodec)(nil)
	_ Codec = (*JSONCodec)(nil)
	_ Codec = (*MsgPackCodec)(nil)
)

// GobCodec is a gob implementation of the Codec interface.
type GobCodec struct{}

// Encode implements the Encoder interface.
func (e *GobCodec) Encode(_ context.Context, value any) ([]byte, error) {
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
func (e *GobCodec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	err := gob.NewEncoder(w).Encode(value)
	if err != nil {
		return sugarEncodingErr(value, err)
	}
	return nil
}

// Decode implements the Decoder interface.
func (e *GobCodec) Decode(ctx context.Context, data []byte, value any) error {
	err := e.DecodeStream(ctx, bytes.NewReader(data), value)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream implements the Decoder interface.
func (e *GobCodec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := gob.NewDecoder(r).Decode(value); err != nil {
		data, _ := io.ReadAll(r)
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// JSONCodec is a JSON implementation of Codec.
type JSONCodec struct {
	// Pretty indicates whether the JSON should be pretty printed.
	Pretty bool
}

// Encode implements the Encoder interface.
func (j *JSONCodec) Encode(_ context.Context, value any) ([]byte, error) {
	var (
		b   []byte
		err error
	)
	if j.Pretty {
		b, err = json.MarshalIndent(value, "", "  ")
	} else {
		b, err = json.Marshal(value)
	}
	return b, sugarEncodingErr(value, err)
}

// Decode implements the Decoder interface.
func (j *JSONCodec) Decode(_ context.Context, data []byte, value any) error {
	if err := json.Unmarshal(data, value); err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream implements the Decoder interface.
func (j *JSONCodec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := json.NewDecoder(r).Decode(value); err != nil {
		data, _ := io.ReadAll(r)
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// EncodeStream implements the Encoder interface.
func (j *JSONCodec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	var err error
	if j.Pretty {
		err = json.NewEncoder(w).Encode(value)
	} else {
		b, err := j.Encode(ctx, value)
		if err != nil {
			return err
		}
		_, err = w.Write(b)
	}
	return sugarEncodingErr(value, err)
}

// MsgPackCodec is a msgpack implementation of Codec.
type MsgPackCodec struct{}

// Encode implements the Encoder interface.
func (m *MsgPackCodec) Encode(_ context.Context, value any) ([]byte, error) {
	b, err := msgpack.Marshal(value)
	return b, sugarEncodingErr(value, err)
}

// Decode implements the Decoder interface.
func (m *MsgPackCodec) Decode(ctx context.Context, data []byte, value any) error {
	err := m.DecodeStream(ctx, bytes.NewReader(data), value)
	return sugarDecodingErr(data, value, err)
}

// DecodeStream implements the Decoder interface.
func (m *MsgPackCodec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := msgpack.NewDecoder(r).Decode(value); err != nil {
		data, _ := io.ReadAll(r)
		return sugarDecodingErr(data, value, err)
	}
	return nil
}

// EncodeStream implements the Encoder interface.
func (m *MsgPackCodec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	b, err := m.Encode(ctx, value)
	if err != nil {
		return sugarEncodingErr(value, err)
	}
	_, err = w.Write(b)
	return sugarEncodingErr(value, err)
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

// TracingCodec wraps a Codec and traces the encoding and decoding operations.
type TracingCodec struct {
	alamos.Instrumentation
	Level alamos.Environment
	Codec
}

// Encode implements the Encoder interface.
func (enc *TracingCodec) Encode(ctx context.Context, value any) ([]byte, error) {
	ctx, span := enc.T.Trace(ctx, "encode", enc.Level)
	b, err := enc.Codec.Encode(ctx, value)
	if err != nil {
		return nil, sugarEncodingErr(value, err)
	}
	return b, span.EndWith(err)
}

// Decode implements the Decoder interface.
func (enc *TracingCodec) Decode(ctx context.Context, data []byte, value any) error {
	ctx, span := enc.T.Trace(ctx, "decode", enc.Level)
	err := enc.Codec.Decode(ctx, data, value)
	if err != nil {
		return sugarDecodingErr(data, value, err)
	}
	return span.EndWith(err)
}

// DecodeStream implements the Decoder interface.
func (enc *TracingCodec) DecodeStream(ctx context.Context, r io.Reader, value any) error {
	ctx, span := enc.T.Trace(ctx, "decode_stream", enc.Level)
	err := enc.Codec.DecodeStream(ctx, r, value)
	if err != nil {
		data, _ := io.ReadAll(r)
		err = sugarDecodingErr(data, value, err)
	}
	return span.EndWith(err)
}

// EncodeStream implements the Encoder interface.
func (enc *TracingCodec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	ctx, span := enc.T.Trace(ctx, "encode_stream", enc.Level)
	err := enc.Codec.EncodeStream(ctx, w, value)
	if err != nil {
		err = sugarEncodingErr(value, err)
	}
	return span.EndWith(err)
}

// UnmarshalJSONStringInt64 attempts to unmarshal an int64 directly. If that fails,
// it attempts to convert a string to an int64.
func UnmarshalJSONStringInt64(b []byte) (int64, error) {
	var n int64
	if err := json.Unmarshal(b, &n); err == nil {
		return n, nil
	}
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return n, err
	}
	return strconv.ParseInt(str, 10, 64)
}

// UnmarshalJSONStringUint64 attempts to unmarshal the uint64 directly. If that fails,
// it attempts to convert a string to a uint64.
func UnmarshalJSONStringUint64(b []byte) (uint64, error) {
	var n uint64
	if err := json.Unmarshal(b, &n); err == nil {
		return n, nil
	}
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return n, err
	}
	return strconv.ParseUint(str, 10, 64)
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

// MustEncodeJSONToString encodes the value to a JSON string, and panics if an error
// occurs.
func MustEncodeJSONToString(v any) string {
	return string(lo.Must((&JSONCodec{}).Encode(context.Background(), v)))
}
