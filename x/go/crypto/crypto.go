// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// crypto is a package that handles cryptographic functions.

package crypto

import (
	"errors"

	"github.com/synnaxlabs/x/math"
)

// Cipher takes a number entry of digits numDigits and shifts it down by
// distance. This does not output negative numbers, allowing for a one-to-one
// cipher for every number between 0 and 10^numDigits - 1.
func Cipher(entry int, distance int, numDigits int) (int, error) {
	if entry < 0 {
		return entry, errors.New("entry is below 0")
	} else if entry >= math.IntPow(10, numDigits) {
		return entry, errors.New("entry is above 10^numDigits-1")
	}
	entry -= distance
	if entry < 0 {
		entry += math.IntPow(10, numDigits)
	}
	return entry, nil
}
