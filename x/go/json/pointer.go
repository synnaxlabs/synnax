// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package json converts between JSON documents and telemetry samples without
// allocating: an Extractor finds values by JSON Pointer, a Decoder and an Encoder
// convert values to and from samples, and a Template renders documents.
package json

import (
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Pointer is a parsed JSON Pointer (RFC 6901): the unescaped reference tokens that lead
// from the root of a document to a value. The zero value points at the whole document.
type Pointer []string

// ParsePointer parses the text form of a JSON Pointer. It returns an error wrapping
// validate.ErrValidation when text is not empty and does not start with "/", or when a
// "~" is not followed by "0" or "1".
func ParsePointer(text string) (Pointer, error) {
	if text == "" {
		return nil, nil
	}
	if text[0] != '/' {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			`json pointer %q must be empty or start with "/"`,
			text,
		)
	}
	tokens := strings.Split(text[1:], "/")
	for i, token := range tokens {
		for j := 0; j < len(token); j++ {
			if token[j] != '~' {
				continue
			}
			if j+1 == len(token) || (token[j+1] != '0' && token[j+1] != '1') {
				return nil, errors.Wrapf(
					validate.ErrValidation,
					`json pointer %q is malformed: "~" must be followed by "0" or "1"`,
					text,
				)
			}
		}
		// RFC 6901 §4 orders the replacements so that "~01" decodes to "~1".
		tokens[i] = strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
	}
	return tokens, nil
}

// String returns the text form of the pointer.
func (p Pointer) String() string {
	var b strings.Builder
	for _, token := range p {
		b.WriteByte('/')
		b.WriteString(
			strings.ReplaceAll(strings.ReplaceAll(token, "~", "~0"), "/", "~1"),
		)
	}
	return b.String()
}

// MarshalText implements encoding.TextMarshaler.
func (p Pointer) MarshalText() ([]byte, error) { return []byte(p.String()), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (p *Pointer) UnmarshalText(text []byte) (err error) {
	*p, err = ParsePointer(string(text))
	return err
}

// arrayIndex parses an RFC 6901 array index: decimal digits with no leading zero.
func arrayIndex(token string) (int, bool) {
	if token == "" || (len(token) > 1 && token[0] == '0') {
		return 0, false
	}
	for i := 0; i < len(token); i++ {
		if token[i] < '0' || token[i] > '9' {
			return 0, false
		}
	}
	i, err := strconv.Atoi(token)
	return i, err == nil
}
