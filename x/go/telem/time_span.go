// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/synnaxlabs/x/binary"
)

const (
	// TimeSpanZero represents the zero value for a TimeSpan.
	TimeSpanZero = TimeSpan(0)
	// TimeSpanMax represents the maximum possible TimeSpan.
	TimeSpanMax = TimeSpan(^uint64(0) >> 1)
)

var (
	_ json.Unmarshaler = (*TimeSpan)(nil)
	_ json.Marshaler   = TimeSpan(0)
)

// MarshalJSON implements json.Marshaler.
func (ts TimeSpan) MarshalJSON() ([]byte, error) {
	return binary.MarshalStringInt64(int64(ts))
}

// UnmarshalJSON implements json.Unmarshaler.
func (ts *TimeSpan) UnmarshalJSON(b []byte) error {
	n, err := binary.UnmarshalJSONStringInt64(b)
	*ts = TimeSpan(n)
	return err
}

// Duration converts TimeSpan to a time.Duration.
func (ts TimeSpan) Duration() time.Duration { return time.Duration(ts) }

// Seconds returns a float64 value representing the number of seconds in the TimeSpan.
func (ts TimeSpan) Seconds() float64 { return ts.Duration().Seconds() }

// IsZero returns true if the TimeSpan is TimeSpanZero.
func (ts TimeSpan) IsZero() bool { return ts == TimeSpanZero }

// IsMax returns true if the TimeSpan is the maximum possible value.
func (ts TimeSpan) IsMax() bool { return ts == TimeSpanMax }

func (ts TimeSpan) ByteSize(rate Rate, density Density) Size {
	return Size(ts / rate.Period() * TimeSpan(density))
}

// String returns a string representation of the TimeSpan
func (ts TimeSpan) String() string {
	adjusted := ts
	if adjusted == 0 {
		return "0s"
	}

	var parts []string
	if adjusted < 0 {
		parts = append(parts, "-")
		adjusted = -ts
	}

	totalDays := adjusted.Truncate(Day)
	totalHours := adjusted.Truncate(Hour)
	totalMinutes := adjusted.Truncate(Minute)
	totalSeconds := adjusted.Truncate(Second)
	totalMilliseconds := adjusted.Truncate(Millisecond)
	totalMicroseconds := adjusted.Truncate(Microsecond)
	totalNanoseconds := adjusted

	days := totalDays / Day
	hours := (totalHours - totalDays) / Hour
	minutes := (totalMinutes - totalHours) / Minute
	seconds := (totalSeconds - totalMinutes) / Second
	milliseconds := (totalMilliseconds - totalSeconds) / Millisecond
	microseconds := (totalMicroseconds - totalMilliseconds) / Microsecond
	nanoseconds := totalNanoseconds - totalMicroseconds

	if days != 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours != 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if minutes != 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}
	if seconds != 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}
	if milliseconds != 0 {
		parts = append(parts, fmt.Sprintf("%dms", milliseconds))
	}
	if microseconds != 0 {
		parts = append(parts, fmt.Sprintf("%dµs", microseconds))
	}
	if nanoseconds != 0 {
		parts = append(parts, fmt.Sprintf("%dns", nanoseconds))
	}

	return strings.Join(parts, " ")
}

// Truncate returns a new TimeSpan that is truncated to the nearest multiple of the given TimeSpan.
func (ts TimeSpan) Truncate(unit TimeSpan) TimeSpan {
	if unit == 0 {
		return ts
	}
	return ts / unit * unit
}

const (
	// Nanosecond is a 1 nanosecond TimeSpan.
	Nanosecond = TimeSpan(1)
	// Microsecond is a single microsecond TimeSpan.
	Microsecond = 1000 * Nanosecond
	// Millisecond is a 1-millisecond TimeSpan.
	Millisecond = 1000 * Microsecond
	// Second is a 1-second TimeSpan.
	Second = 1000 * Millisecond
	// Minute is a 1-minute TimeSpan.
	Minute = 60 * Second
	// Hour is a 1-hour TimeSpan.
	Hour = 60 * Minute
	// Day is a 1-day long TimeSpan.
	Day = 24 * Hour
)
