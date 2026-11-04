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
	"slices"
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

// labeledPointer is a pointer named after the config entry that set it.
type labeledPointer struct{ label, pointer string }

// validateDisjoint returns an error wrapping validate.ErrValidation when a pointer is
// malformed or empty, or when two pointers are equal or nested, since setting both
// would overwrite one.
func validateDisjoint(pointers []labeledPointer) error {
	tokens := make([][]string, len(pointers))
	for i, p := range pointers {
		t, err := pointerTokens(p.pointer)
		if err != nil {
			return errors.Wrapf(err, "%s", p.label)
		}
		if len(t) == 0 {
			return errors.Wrapf(
				validate.ErrValidation,
				"%s: must not be empty",
				p.label,
			)
		}
		tokens[i] = t
	}
	for i, p := range pointers {
		for j, q := range pointers[:i] {
			if hasPrefix(tokens[i], tokens[j]) || hasPrefix(tokens[j], tokens[i]) {
				return errors.Wrapf(
					validate.ErrValidation,
					"%s %q and %s %q overlap",
					q.label, q.pointer, p.label, p.pointer,
				)
			}
		}
	}
	return nil
}

func hasPrefix(tokens, prefix []string) bool {
	return len(tokens) >= len(prefix) && slices.Equal(tokens[:len(prefix)], prefix)
}
