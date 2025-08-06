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
	"context"
	"fmt"
	"io"
	"reflect"

	"github.com/synnaxlabs/x/errors"
)

// StringCodec is a codec that encodes and decodes data to and from strings.
var StringCodec = &stringCodec{}

// StringUnmarshaller is an interface that can be implemented by types to provide a
// custom string representation for unmarshalling string data.
type StringUnmarshaller interface{ UnmarshalString(string) error }

type stringCodec struct{}

var _ Codec = (*stringCodec)(nil)

// Encode encodes a value to its string representation.
func (sc *stringCodec) Encode(ctx context.Context, v any) ([]byte, error) {
	return WrapStreamEncoder(sc, ctx, v)
}

// EncodeStream encodes a value to its string representation and writes it to a writer.
func (sc *stringCodec) EncodeStream(_ context.Context, w io.Writer, v any) error {
	_, err := fmt.Fprint(w, v)
	return sugarEncodingErr(v, err)
}

// Decode decodes the plaintext string into the provided value.
func (sc *stringCodec) Decode(ctx context.Context, b []byte, v any) error {
	return WrapStreamDecoder(sc, ctx, b, v)
}

// DecodeStream decodes the plaintext string from a reader into the provided value.
func (sc *stringCodec) DecodeStream(ctx context.Context, r io.Reader, v any) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return sugarDecodingErr(data, v, err)
	}
	str := string(data)
	err = unmarshalString(str, v)
	return sugarDecodingErr(data, v, err)
}

func unmarshalString(str string, v any) error {
	if reflect.ValueOf(v).Kind() != reflect.Ptr {
		return errors.Newf("stringCodec: value must be a pointer, got %T", v)
	}
	switch target := v.(type) {
	case StringUnmarshaller:
		return target.UnmarshalString(str)
	case *string:
		*target = str
		return nil
	default:
		return errors.Newf("stringCodec: %T does not implement StringUnmarshaller", v)
	}
}
