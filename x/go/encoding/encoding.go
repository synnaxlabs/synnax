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
	"io"
	"reflect"

	"github.com/synnaxlabs/x/errors"
)

// Decoder decodes values from binary.
type Decoder interface {
	// Decode decodes data into a pointer value.
	Decode(data []byte, value any) error
	// DecodeStream decodes data from the given reader into a pointer value.
	DecodeStream(r io.Reader, value any) error
}

// Encoder encodes values into binary.
type Encoder interface {
	// Encode encodes the value into binary. It returns the encoded value along with any
	// errors encountered.
	Encode(value any) ([]byte, error)
	// EncodeStream encodes the value into binary and writes it to the given writer. It
	// returns any errors encountered.
	EncodeStream(w io.Writer, value any) error
}

// FileEncoder is an Encoder that names the files its output belongs in, so a caller
// writing a directory can name each file without knowing which codec it holds.
type FileEncoder interface {
	Encoder
	// Extension returns the file extension the encoder's output is written under,
	// leading dot included.
	Extension() string
}

// Codec is an interface that encodes and decodes values.
type Codec interface {
	Decoder
	Encoder
}

// SugarDecodingError names the type that value points to in a decoding error. The
// result wraps base, so errors.Is matches base and the errors it wraps.
func SugarDecodingError(value any, base error) error {
	if base == nil {
		return nil
	}
	t := reflect.TypeOf(value)
	if t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return errors.Wrapf(base, "failed to decode %s", t)
}

// SugarEncodingError adds additional context to encoding errors.
func SugarEncodingError(value any, base error) error {
	if base == nil {
		return nil
	}
	val := reflect.ValueOf(value)
	main := errors.Wrapf(
		errors.New("failed to encode"),
		"failed to encode value: kind=%s, type=%s, value=%+v",
		val.Kind(),
		val.Type(),
		value,
	)
	return errors.WithStack(errors.Combine(main, base))
}
