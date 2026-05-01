// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex

import (
	"encoding/json"
	"io"

	"github.com/synnaxlabs/x/errors"
)

// Decode unmarshals raw JSON bytes into dest, translating encoding/json's
// internal error types into user-friendly messages with field path and byte
// offset. Handlers should prefer Decode over json.Unmarshal so malformed or
// hand-edited import files produce actionable errors.
func Decode(raw json.RawMessage, dest any) error {
	if err := json.Unmarshal(raw, dest); err != nil {
		return translateDecodeError(err)
	}
	return nil
}

func translateDecodeError(err error) error {
	var ute *json.UnmarshalTypeError
	if errors.As(err, &ute) {
		field := ute.Field
		if field == "" {
			field = "(root)"
		}
		return errors.Newf(
			"field %s: expected %s, got %s (offset %d)",
			field, ute.Type, ute.Value, ute.Offset,
		)
	}
	var se *json.SyntaxError
	if errors.As(err, &se) {
		return errors.Wrapf(err, "malformed JSON at offset %d", se.Offset)
	}
	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return errors.Wrap(err, "truncated JSON")
	}
	return err
}
