// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color

import (
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// positionDigits are the base 62 digits of a fractional index, in code point order.
const positionDigits = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// validatePosition returns an error wrapping validate.ErrValidation when p is not a
// fractional index key, as the fractional-indexing npm package makes them.
func validatePosition(p string) error {
	if !positionValid(p) {
		return errors.Wrapf(
			validate.ErrValidation,
			"position: invalid fractional index %q",
			p,
		)
	}
	return nil
}

func positionValid(p string) bool {
	if p == "" || strings.IndexFunc(p, func(r rune) bool {
		return !strings.ContainsRune(positionDigits, r)
	}) >= 0 {
		return false
	}
	// The head digit sets the length of the integer part.
	var n int
	switch head := p[0]; {
	case head >= 'a' && head <= 'z':
		n = int(head-'a') + 2
	case head >= 'A' && head <= 'Z':
		n = int('Z'-head) + 2
	default:
		return false
	}
	if n > len(p) || p == "A"+strings.Repeat("0", 26) {
		return false
	}
	return len(p) == n || p[len(p)-1] != '0'
}
