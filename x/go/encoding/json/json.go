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
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strconv"

	"github.com/synnaxlabs/x/encoding"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/http"
)

// Codec is a JSON implementation of http.FileCodec.
var Codec http.FileCodec = &codec{}

// PrettyCodec is a JSON implementation of http.FileCodec that encodes with two-space
// indentation and a trailing newline, for files a user reads. Decoding matches Codec.
var PrettyCodec http.FileCodec = &prettyCodec{}

type codec struct{}

func (*codec) ContentType() string { return "application/json" }

func (*codec) Decode(_ context.Context, data []byte, value any) error {
	if err := json.Unmarshal(data, value); err != nil {
		return encoding.SugarDecodingError(data, value, err)
	}
	return nil
}

func (*codec) DecodeStream(_ context.Context, r io.Reader, value any) error {
	if err := json.NewDecoder(r).Decode(value); err != nil {
		data, ioErr := io.ReadAll(r)
		return encoding.SugarDecodingError(data, value, errors.Combine(err, ioErr))
	}
	return nil
}

func (*codec) Encode(_ context.Context, value any) ([]byte, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return nil, encoding.SugarEncodingError(value, err)
	}
	return b, nil
}

func (*codec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return encoding.SugarEncodingError(value, err)
	}
	return nil
}

func (*codec) Extension() string { return ".json" }

type prettyCodec struct{ codec }

func (p *prettyCodec) Encode(ctx context.Context, value any) ([]byte, error) {
	var buf bytes.Buffer
	if err := p.EncodeStream(ctx, &buf, value); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (*prettyCodec) EncodeStream(_ context.Context, w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(value); err != nil {
		return encoding.SugarEncodingError(value, err)
	}
	return nil
}

// MarshalStringInt64 marshals the int64 value to a UTF-8 string.
func MarshalStringInt64(n int64) []byte {
	return []byte(`"` + strconv.FormatInt(n, 10) + `"`)
}

// MarshalStringUint64 marshals the uint64 value to a UTF-8 string.
func MarshalStringUint64(n uint64) []byte {
	return []byte(`"` + strconv.FormatUint(n, 10) + `"`)
}

// UnmarshalStringInt64 attempts to unmarshal an int64 directly. If that fails, it
// attempts to convert a string to an int64.
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

// UnmarshalStringUint32 attempts to unmarshal the uint32 directly. If that fails, it
// attempts to convert a string to a uint32.
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

// UnmarshalStringUint64 attempts to unmarshal the uint64 directly. If that fails, it
// attempts to convert a string to a uint64.
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
