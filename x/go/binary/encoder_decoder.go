// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/alamos"
	"github.com/vmihailenco/msgpack/v5"
	"io"
)

// EncoderDecoder is an interface that encodes and decodes values.
type EncoderDecoder interface {
	Encoder
	Decoder
}

// Encoder encodes values into binary.
type Encoder interface {
	// Encode encodes the value into binary. It returns the encoded value along
	// with any errors encountered.
	Encode(ctx context.Context, value interface{}) ([]byte, error)
}

// Decoder decodes values from binary.
type Decoder interface {
	// Decode decodes data into a pointer value.
	Decode(ctx context.Context, data []byte, value interface{}) error
	// DecodeStream decodes data from the given reader into a pointer value.;
	DecodeStream(ctx context.Context, r io.Reader, value interface{}) error
}

var (
	_ EncoderDecoder = (*GobEncoderDecoder)(nil)
	_ EncoderDecoder = (*JSONEncoderDecoder)(nil)
	_ EncoderDecoder = (*MsgPackEncoderDecoder)(nil)
)

// GobEncoderDecoder is a gob implementation of the EncoderDecoder interface.
type GobEncoderDecoder struct{}

// Encode implements the Encoder interface.
func (e *GobEncoderDecoder) Encode(_ context.Context, value interface{}) ([]byte, error) {
	var (
		buff bytes.Buffer
		err  = gob.NewEncoder(&buff).Encode(value)
		b    = buff.Bytes()
	)
	return b, err
}

// Decode implements the Decoder interface.
func (e *GobEncoderDecoder) Decode(_ context.Context, data []byte, value interface{}) error {
	return e.DecodeStream(nil, bytes.NewReader(data), value)
}

// DecodeStream implements the Decoder interface.
func (e *GobEncoderDecoder) DecodeStream(_ context.Context, r io.Reader, value interface{}) error {
	return gob.NewDecoder(r).Decode(value)
}

// JSONEncoderDecoder is a JSON implementation of EncoderDecoder.
type JSONEncoderDecoder struct {
	// Pretty indicates whether the JSON should be pretty printed.
	Pretty bool
}

// Encode implements the Encoder interface.
func (j *JSONEncoderDecoder) Encode(_ context.Context, value interface{}) ([]byte, error) {
	if j.Pretty {
		return json.MarshalIndent(value, "", "  ")
	}
	return json.Marshal(value)
}

// Decode implements the Decoder interface.
func (j *JSONEncoderDecoder) Decode(_ context.Context, data []byte, value interface{}) error {
	return json.Unmarshal(data, value)
}

// DecodeStream implements the Decoder interface.
func (j *JSONEncoderDecoder) DecodeStream(_ context.Context, r io.Reader, value interface{}) error {
	return json.NewDecoder(r).Decode(value)
}

// MsgPackEncoderDecoder is a msgpack implementation of EncoderDecoder.
type MsgPackEncoderDecoder struct{}

// Encode implements the Encoder interface.
func (m *MsgPackEncoderDecoder) Encode(_ context.Context, value interface{}) ([]byte, error) {
	return msgpack.Marshal(value)
}

// Decode implements the Decoder interface.
func (m *MsgPackEncoderDecoder) Decode(ctx context.Context, data []byte, value interface{}) error {
	return m.DecodeStream(ctx, bytes.NewReader(data), value)
}

// DecodeStream implements the Decoder interface.
func (m *MsgPackEncoderDecoder) DecodeStream(_ context.Context, r io.Reader, value interface{}) error {
	return msgpack.NewDecoder(r).Decode(value)
}

// PassThroughEncoderDecoder wraps an EncoderDecoder and checks for values
// that are already encoded ([]byte) and returns them as is.
type PassThroughEncoderDecoder struct{ EncoderDecoder }

// Encode implements the Encoder interface.
func (enc *PassThroughEncoderDecoder) Encode(_ context.Context, value interface{}) ([]byte, error) {
	if bv, ok := value.([]byte); ok {
		return bv, nil
	}
	return enc.EncoderDecoder.Encode(nil, value)
}

// Decode implements the Decoder interface.
func (enc *PassThroughEncoderDecoder) Decode(_ context.Context, data []byte, value interface{}) error {
	return enc.DecodeStream(nil, bytes.NewReader(data), value)
}

// DecodeStream implements the Decoder interface.
func (enc *PassThroughEncoderDecoder) DecodeStream(_ context.Context, r io.Reader, value interface{}) error {
	if bv, ok := value.(*[]byte); ok {
		*bv, _ = io.ReadAll(r)
		return nil
	}
	return enc.EncoderDecoder.DecodeStream(nil, r, value)
}

// TracingEncoderDecoder wraps an EncoderDecoder and traces the encoding and decoding
// operations.
type TracingEncoderDecoder struct {
	alamos.Instrumentation
	Level alamos.Environment
	EncoderDecoder
}

// Encode implements the Encoder interface.
func (enc *TracingEncoderDecoder) Encode(ctx context.Context, value interface{}) ([]byte, error) {
	ctx, span := enc.T.Trace(ctx, "encode", enc.Level)
	b, err := enc.EncoderDecoder.Encode(ctx, value)
	return b, span.EndWith(err)
}

// Decode implements the Decoder interface.
func (enc *TracingEncoderDecoder) Decode(ctx context.Context, data []byte, value interface{}) error {
	ctx, span := enc.T.Trace(ctx, "decode", enc.Level)
	return span.EndWith(enc.EncoderDecoder.Decode(ctx, data, value))
}

// DecodeStream implements the Decoder interface.
func (enc *TracingEncoderDecoder) DecodeStream(ctx context.Context, r io.Reader, value interface{}) error {
	ctx, span := enc.T.Trace(ctx, "decode_stream", enc.Level)
	return span.EndWith(enc.EncoderDecoder.DecodeStream(ctx, r, value))
}
