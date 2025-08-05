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

var (
	ErrDecode = errors.New("failed to decode")
	ErrEncode = errors.New("failed to encode")
)

// sugarEncodingErr adds additional context to encoding errors.
func sugarEncodingErr(value any, base error) error {
	if base == nil {
		return nil
	}
	val := reflect.ValueOf(value)
	main := errors.Wrapf(ErrEncode, "failed to encode value: kind=%s, type=%s, value=%+v", val.Kind(), val.Type(), value)
	return errors.Combine(main, base)
}

// sugarDecodingErr adds additional context to decoding errors.
func sugarDecodingErr(data []byte, value any, base error) error {
	if base == nil {
		return nil
	}
	val := reflect.ValueOf(value)
	main := errors.Wrapf(ErrDecode, "kind=%s, type=%s, data=%x", val.Kind(), val.Type(), data)
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

// MarshalStringInt64 marshals the int64 value to a UTF-8 string.
func MarshalStringInt64(n int64) ([]byte, error) {
	return []byte(`"` + strconv.Itoa(int(n)) + `"`), nil
}

// MarshalStringUint64 marshals the uint64 value to a UTF-8 string.
func MarshalStringUint64(n uint64) ([]byte, error) {
	return []byte(`"` + strconv.FormatUint(n, 10) + `"`), nil
}

func wrapStreamEncoder(enc Encoder, ctx context.Context, value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := enc.EncodeStream(ctx, &buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func wrapStreamDecoder(dec Decoder, ctx context.Context, data []byte, value any) error {
	return dec.DecodeStream(ctx, bytes.NewReader(data), value)
}
