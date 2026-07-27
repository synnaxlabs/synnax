// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import v0 "github.com/synnaxlabs/x/telem/versions/v0"

const (
	// TimeSpanZero represents the zero value for a TimeSpan.
	TimeSpanZero TimeSpan = v0.TimeSpanZero
	// TimeSpanMax represents the maximum possible TimeSpan.
	TimeSpanMax TimeSpan = v0.TimeSpanMax
	// Nanosecond is a 1 nanosecond TimeSpan.
	Nanosecond TimeSpan = v0.Nanosecond
	// Microsecond is a single microsecond TimeSpan.
	Microsecond TimeSpan = v0.Microsecond
	// Millisecond is a 1-millisecond TimeSpan.
	Millisecond TimeSpan = v0.Millisecond
	// Second is a 1-second TimeSpan.
	Second TimeSpan = v0.Second
	// Minute is a 1-minute TimeSpan.
	Minute TimeSpan = v0.Minute
	// Hour is a 1-hour TimeSpan.
	Hour TimeSpan = v0.Hour
	// Day is a 1-day long TimeSpan.
	Day TimeSpan = v0.Day
)
