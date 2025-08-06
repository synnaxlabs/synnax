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
	"encoding/json"
	"io"
	"strconv"

	"github.com/samber/lo"
)

// JSONCodec is a JSON implementation of Codec.
var JSONCodec = &jsonCodec{}

type jsonCodec struct{}

var _ Codec = (*jsonCodec)(nil)

// Encode implements the Encoder interface.
func (jc *jsonCodec) Encode(_ context.Context, value any) ([]byte, error) {
	b, err := json.Marshal(value)
	return b, SugarEncodingErr(value, err)
}

// Decode implements the Decoder interface.
func (jc *jsonCodec) Decode(_ context.Context, data []byte, value any) error {
	err := json.Unmarshal(data, value)
	return SugarDecodingErr(data, value, err)
}

// DecodeStream implements the Decoder interface.
func (jc *jsonCodec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := json.NewDecoder(r).Decode(value); err != nil {
		data, _ := io.ReadAll(r)
		return SugarDecodingErr(data, value, err)
	}
	return nil
}

// EncodeStream implements the Encoder interface.
func (jc *jsonCodec) EncodeStream(ctx context.Context, w io.Writer, value any) error {
	b, err := jc.Encode(ctx, value)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return SugarEncodingErr(value, err)
}

// UnmarshalJSONStringInt64 attempts to unmarshal an int64 directly. If that fails, it
// attempts to convert a string to an int64.
func UnmarshalJSONStringInt64(b []byte) (int64, error) {
	var n int64
	if err := json.Unmarshal(b, &n); err == nil {
		return n, nil
	}
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return 0, err
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
		return 0, err
	}
	return strconv.ParseUint(str, 10, 64)
}

// MustEncodeJSONToString encodes the value to a JSON string, and panics if an error
// occurs.
func MustEncodeJSONToString(v any) string {
	return string(lo.Must(JSONCodec.Encode(context.Background(), v)))
}
