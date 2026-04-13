// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package filter

import (
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

// StringOrderBy carries a sort direction and an optional typed cursor for
// string-valued sorted fields.
type StringOrderBy struct {
	Direction string  `json:"direction" msgpack:"direction"`
	After     *string `json:"after,omitempty" msgpack:"after,omitempty"`
}

// TimestampOrderBy carries a sort direction and an optional typed cursor for
// telem.TimeStamp-valued sorted fields.
type TimestampOrderBy struct {
	Direction string           `json:"direction" msgpack:"direction"`
	After     *telem.TimeStamp `json:"after,omitempty" msgpack:"after,omitempty"`
}

// NumericOrderBy carries a sort direction and an optional typed cursor for
// numeric sorted fields.
type NumericOrderBy[T ~int8 | ~int16 | ~int32 | ~int64 |
	~uint8 | ~uint16 | ~uint32 | ~uint64 |
	~float32 | ~float64] struct {
	Direction string `json:"direction" msgpack:"direction"`
	After     *T     `json:"after,omitempty" msgpack:"after,omitempty"`
}

// ParseDirection converts a string to a gorp.Direction. Returns gorp.Asc for
// any value other than "desc".
func ParseDirection(s string) gorp.Direction {
	if s == "desc" {
		return gorp.Desc
	}
	return gorp.Asc
}
