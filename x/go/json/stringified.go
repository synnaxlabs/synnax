// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

// String is a type that can represent both raw JSON strings and JSON objects/arrays.
// When unmarshaling, it first attempts to parse the input as a raw JSON string.
// If that fails, it attempts to parse it as a JSON object or array and stores the
// marshaled representation.
type String string

var detailsCodec = &binary.JSONCodec{}

// NewStaticString creates a new String from static data. The data is encoded using
// the binary.JSONCodec. This function should only be used with static data as it
// will panic if encoding fails. For dynamic data, use json.Marshal directly.
//
// Parameters:
//   - ctx: The context for the operation
//   - data: The data to encode. Can be any JSON-serializable type
//
// Returns:
//   - String: The encoded JSON string
func NewStaticString(ctx context.Context, data any) String {
	b, err := detailsCodec.Encode(ctx, data)
	if err != nil {
		zap.S().DPanic("unexpected static encode error", zap.Error(err))
	}
	return String(b)
}

// UnmarshalJSON implements the json.Unmarshaler interface for String.
// It first attempts to unmarshal the data as a raw JSON string. If that fails,
// it attempts to unmarshal it as a JSON object or array. The resulting String
// will contain either the raw string value or the marshaled representation of
// the JSON object/array.
//
// Parameters:
//   - data: The JSON data to unmarshal
//
// Returns:
//   - error: An error if unmarshaling fails
func (d *String) UnmarshalJSON(data []byte) error {
	var plainString string
	if err := json.Unmarshal(data, &plainString); err == nil {
		*d = String(plainString)
		return nil
	}
	var obj any
	if err := json.Unmarshal(data, &obj); err != nil {
		return errors.Wrap(err, "failed to unmarshal JSON")
	}
	bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	*d = String(bytes)
	return nil
}
