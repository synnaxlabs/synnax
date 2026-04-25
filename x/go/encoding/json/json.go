// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json

import (
	"context"
	"encoding/json"
	"io"
	"strconv"

	"github.com/synnaxlabs/x/encoding"
)

// Codec is a JSON implementation of encoding.Codec.
var Codec = &codec{}

type codec struct{}

// ContentType implements http.Codec to return the http content type for the codec.
func (c *codec) ContentType() string { return "application/json" }

// Encode implements the encoding.Encoder interface.
func (c *codec) Encode(_ context.Context, value any) ([]byte, error) {
	b, err := json.Marshal(value)
	return b, encoding.SugarEncodingErr(value, err)
}

// Decode implements the encoding.Decoder interface.
func (c *codec) Decode(_ context.Context, data []byte, value any) error {
	if err := json.Unmarshal(data, value); err != nil {
		return encoding.SugarDecodingErr(data, value, err)
	}
	return nil
}

// DecodeStream implements the encoding.Decoder interface.
func (c *codec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := json.NewDecoder(r).Decode(value); err != nil {
		data, _ := io.ReadAll(r)
		return encoding.SugarDecodingErr(data, value, err)
	}
	return nil
}

// EncodeStream implements the encoding.Encoder interface.
func (c *codec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	b, err := c.Encode(ctx, value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return encoding.SugarEncodingErr(value, err)
}

// UnmarshalStringInt64 attempts to unmarshal an int64 directly. If that fails,
// it attempts to convert a string to an int64.
func UnmarshalStringInt64(b []byte) (int64, error) {
	var n int64
	if err := json.Unmarshal(b, &n); err == nil {
		return n, nil
	}
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return 0, err
	}
	v, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// UnmarshalStringUint64 attempts to unmarshal the uint64 directly. If that fails,
// it attempts to convert a string to a uint64.
func UnmarshalStringUint64(b []byte) (uint64, error) {
	var n uint64
	if err := json.Unmarshal(b, &n); err == nil {
		return n, nil
	}
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return 0, err
	}
	v, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return 0, err
	}
	return v, nil
}

// UnmarshalStringUint32 attempts to unmarshal the uint32 directly. If that fails,
// it attempts to convert a string to a uint32.
func UnmarshalStringUint32(b []byte) (uint32, error) {
	var n uint32
	if err := json.Unmarshal(b, &n); err == nil {
		return n, nil
	}
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return 0, err
	}
	v, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(v), nil
}

// MarshalStringInt64 marshals the int64 value to a UTF-8 string.
func MarshalStringInt64(n int64) []byte {
	return []byte(`"` + strconv.Itoa(int(n)) + `"`)
}

// MarshalStringUint64 marshals the uint64 value to a UTF-8 string.
func MarshalStringUint64(n uint64) []byte {
	return []byte(`"` + strconv.FormatUint(n, 10) + `"`)
}
