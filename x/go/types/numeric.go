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

// Numeric represents a generic numeric value.
type Numeric interface {
	~int | ~float64 | ~float32 | ~int64 | ~int32 | ~int16 | ~int8 | ~uint64 | ~uint32 | ~uint16 | ~uint8
}

type Integer interface {
	~int | ~int64 | ~int32 | ~int16 | ~int8 | ~uint64 | ~uint32 | ~uint16 | ~uint8
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
