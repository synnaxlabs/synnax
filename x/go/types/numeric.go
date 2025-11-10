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

// Numeric represents any numeric value.
type Numeric interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

type FixedNumeric interface {
	~int8 | ~int16 | ~int32 | ~int64 |
		~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~float32 | ~float64
}

// Integer represents any integer value.
type Integer interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Signed represents any numeric value that can be negative.
type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64
}

// SignedInteger represents any integer value that can be negative.
type SignedInteger interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// UnsignedInteger represents any integer value that can only be positive.
type UnsignedInteger interface {
	~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64
}

// Floating represents any floating-point value.
type Floating interface {
	~float32 | ~float64
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
