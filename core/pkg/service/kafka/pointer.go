// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// pointerTokens splits an RFC 6901 JSON Pointer into its unescaped reference tokens.
// The empty pointer has no tokens and addresses the whole document.
func pointerTokens(pointer string) ([]string, error) {
	if pointer == "" {
		return nil, nil
	}
	if pointer[0] != '/' {
		return nil, errors.Wrapf(
			validate.ErrValidation, "pointer %q must start with '/'", pointer,
		)
	}
	tokens := strings.Split(pointer[1:], "/")
	for i, t := range tokens {
		tokens[i] = strings.ReplaceAll(strings.ReplaceAll(t, "~1", "/"), "~0", "~")
	}
	return tokens, nil
}

// pointerGet resolves pointer within doc, a decoded JSON value. The bool is false when
// the path does not exist. It returns an error only for a malformed pointer.
func pointerGet(doc any, pointer string) (any, bool, error) {
	tokens, err := pointerTokens(pointer)
	if err != nil {
		return nil, false, err
	}
	cur := doc
	for _, t := range tokens {
		switch c := cur.(type) {
		case map[string]any:
			v, ok := c[t]
			if !ok {
				return nil, false, nil
			}
			cur = v
		case []any:
			i, err := strconv.Atoi(t)
			if err != nil || i < 0 || i >= len(c) {
				return nil, false, nil
			}
			cur = c[i]
		default:
			return nil, false, nil
		}
	}
	return cur, true, nil
}

// pointerSet places v at pointer within doc, creating the objects on the way. A
// non-object on the way is replaced. The empty pointer is an error, since it would
// replace doc itself.
func pointerSet(doc map[string]any, pointer string, v any) error {
	tokens, err := pointerTokens(pointer)
	if err != nil {
		return err
	}
	if len(tokens) == 0 {
		return errors.Wrap(validate.ErrValidation, "pointer must not be empty")
	}
	cur := doc
	for _, t := range tokens[:len(tokens)-1] {
		next, ok := cur[t].(map[string]any)
		if !ok {
			next = make(map[string]any)
			cur[t] = next
		}
		cur = next
	}
	cur[tokens[len(tokens)-1]] = v
	return nil
}
