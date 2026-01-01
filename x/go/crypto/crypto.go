// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package crypto provides simple cryptographic and encoding utilities.
package crypto

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/math"
)

// Cipher applies a reversible shift cipher to an integer.
//
// It interprets the input entry as a number with 'numDigits' digits, and shifts it
// downward by distance. If the result is negative, it wraps around at 10^numDigits,
// ensuring the output remains in the range [10, 10^numDigits).
//
// For example, if numDigits = 3 (range 0-999):
//
//	Cipher(5, 7, 3)  => 998
//
// Returns an error if 'entry' is out of bounds or if invalid parameters are given.
//
// See: https://en.wikipedia.org/wiki/Caesar_cipher
func Cipher(entry, distance, numDigits int) (int, error) {
	if entry < 0 {
		return 0, errors.Newf("crypto: entry (%d) must be nonnegative", entry)
	}
	if numDigits <= 0 {
		return 0, errors.Newf("crypto: numDigits (%d) must be positive", numDigits)
	}
	ceiling := math.IntPow(10, numDigits)
	if entry >= ceiling {
		return 0, errors.Newf(
			"crypto: entry (%d) must be less than 10^numDigits (%d)",
			entry,
			numDigits,
		)
	}
	shifted := (entry - distance) % ceiling
	if shifted < 0 {
		shifted += ceiling
	}
	return shifted, nil
}
