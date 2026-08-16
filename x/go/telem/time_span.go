// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import "github.com/synnaxlabs/x/telem/versions"

const (
	// TimeSpanZero represents the zero value for a TimeSpan.
	TimeSpanZero TimeSpan = versions.TimeSpanZero
	// TimeSpanMax represents the maximum possible TimeSpan.
	TimeSpanMax TimeSpan = versions.TimeSpanMax
	// Nanosecond is a 1 nanosecond TimeSpan.
	Nanosecond TimeSpan = versions.Nanosecond
	// Microsecond is a single microsecond TimeSpan.
	Microsecond TimeSpan = versions.Microsecond
	// Millisecond is a 1-millisecond TimeSpan.
	Millisecond TimeSpan = versions.Millisecond
	// Second is a 1-second TimeSpan.
	Second TimeSpan = versions.Second
	// Minute is a 1-minute TimeSpan.
	Minute TimeSpan = versions.Minute
	// Hour is a 1-hour TimeSpan.
	Hour TimeSpan = versions.Hour
	// Day is a 1-day long TimeSpan.
	Day TimeSpan = versions.Day
)
