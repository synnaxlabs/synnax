// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package json holds utilities for working with decoded JSON documents: JSON Pointer
// lookups and conversion between JSON values and telemetry samples.
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

// Get resolves the pointer against doc, a document of map[string]any, []any, and
// scalar values such as one from Decode. It returns false when the pointer does not
// lead to a value.
func (p Pointer) Get(doc any) (any, bool) {
	for _, token := range p {
		switch node := doc.(type) {
		case map[string]any:
			v, ok := node[token]
			if !ok {
				return nil, false
			}
			doc = v
		case []any:
			i, ok := arrayIndex(token)
			if !ok || i >= len(node) {
				return nil, false
			}
			doc = node[i]
		default:
			return nil, false
		}
	}
	return doc, true
}

// Set places value at the pointer within doc and returns the document, which is value
// itself for the root pointer. It creates the objects that the path lacks, so a nil
// doc gives a new document. It sets an element of an existing array, and "-" appends
// one. Set returns an error wrapping validate.ErrValidation when a scalar or an
// invalid array index blocks the path.
func (p Pointer) Set(doc, value any) (any, error) { return p.set(0, doc, value) }

// set places value at the tokens of p from depth on, within the node doc.
func (p Pointer) set(depth int, doc, value any) (any, error) {
	if depth == len(p) {
		return value, nil
	}
	token := p[depth]
	switch node := doc.(type) {
	case nil:
		child, err := p.set(depth+1, nil, value)
		if err != nil {
			return nil, err
		}
		return map[string]any{token: child}, nil
	case map[string]any:
		child, err := p.set(depth+1, node[token], value)
		if err != nil {
			return nil, err
		}
		node[token] = child
		return node, nil
	case []any:
		if token == "-" {
			child, err := p.set(depth+1, nil, value)
			if err != nil {
				return nil, err
			}
			return append(node, child), nil
		}
		i, ok := arrayIndex(token)
		if !ok || i >= len(node) {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"cannot set %s: %q is not an index of an array of %d elements",
				p, token, len(node),
			)
		}
		child, err := p.set(depth+1, node[i], value)
		if err != nil {
			return nil, err
		}
		node[i] = child
		return node, nil
	}
	return nil, errors.Wrapf(
		validate.ErrValidation, "cannot set %s: a scalar value blocks the path", p,
	)
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
