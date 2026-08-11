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
	main := errors.Wrapf(
		ErrEncode,
		"failed to encode value: kind=%s, type=%s, value=%+v",
		val.Kind(),
		val.Type(),
		value,
	)
	return errors.WithStack(errors.Combine(main, base))
}

// SugarDecodingErr adds additional context to decoding errors.
func SugarDecodingErr(data []byte, value any, base error) error {
	if base == nil {
		return base
	}
	val := reflect.ValueOf(value)
	main := errors.Wrapf(
		ErrDecode,
		"kind=%s, type=%s, data=%x",
		val.Kind(),
		val.Type(),
		data,
	)
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

// FileEncoder is an Encoder that names the files its output belongs in, so a caller
// writing a directory can name each file without knowing which codec it holds.
type FileEncoder interface {
	Encoder
	// Extension returns the file extension the encoder's output is written under,
	// leading dot included.
	Extension() string
}

// Decoder decodes values from binary.
type Decoder interface {
	// Decode decodes data into a pointer value.
	Decode(ctx context.Context, data []byte, value any) error
	// DecodeStream decodes data from the given reader into a pointer value.
	DecodeStream(ctx context.Context, r io.Reader, value any) error
}
