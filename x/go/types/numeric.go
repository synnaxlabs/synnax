// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types

import (
	"encoding/json"
	"strconv"
)

// SizedSignedInteger represents signed integers with fixed bit widths.
type SizedSignedInteger interface {
	~int8 | ~int16 | ~int32 | ~int64
}

// SizedUnsignedInteger represents unsigned integers with fixed bit widths.
type SizedUnsignedInteger interface {
	~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Floating represents any floating-point value.
type Floating interface {
	~float32 | ~float64
}

// SignedInteger represents any integer value that can be negative.
type SignedInteger interface {
	~int | SizedSignedInteger
}

// UnsignedInteger represents any integer value that can only be positive.
type UnsignedInteger interface {
	~uint | SizedUnsignedInteger
}

// Integer represents any integer value.
type Integer interface {
	SignedInteger | UnsignedInteger
}

// SizedNumeric represents any numeric value with a fixed bit width.
type SizedNumeric interface {
	SizedSignedInteger | SizedUnsignedInteger | Floating
}

// Signed represents any numeric value that can be negative.
type Signed interface {
	SignedInteger | Floating
}

// Numeric represents any numeric value.
type Numeric interface {
	Integer | Floating
}

type StringParseableUint64 uint64

var _ json.Unmarshaler = (*StringParseableUint64)(nil)

func (s *StringParseableUint64) UnmarshalJSON(b []byte) error {
	// Try to unmarshal as a number first.
	var n uint64
	if err := json.Unmarshal(b, &n); err == nil {
		*s = StringParseableUint64(n)
		return nil
	}

	// Unmarshal as a string.
	var str string
	if err := json.Unmarshal(b, &str); err != nil {
		return err
	}

	// Parse the string.
	n, err := strconv.ParseUint(str, 10, 64)
	if err != nil {
		return err
	}
	*s = StringParseableUint64(n)
	return nil
}
