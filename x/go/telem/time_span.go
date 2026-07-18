// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import latest "github.com/synnaxlabs/x/telem/types/v0"

const (
	// TimeSpanZero represents the zero value for a TimeSpan.
	TimeSpanZero = latest.TimeSpanZero
	// TimeSpanMax represents the maximum possible TimeSpan.
	TimeSpanMax = latest.TimeSpanMax
	// Nanosecond is a 1 nanosecond TimeSpan.
	Nanosecond = latest.Nanosecond
	// Microsecond is a single microsecond TimeSpan.
	Microsecond = latest.Microsecond
	// Millisecond is a 1-millisecond TimeSpan.
	Millisecond = latest.Millisecond
	// Second is a 1-second TimeSpan.
	Second = latest.Second
	// Minute is a 1-minute TimeSpan.
	Minute = latest.Minute
	// Hour is a 1-hour TimeSpan.
	Hour = latest.Hour
	// Day is a 1-day long TimeSpan.
	Day = latest.Day
)
