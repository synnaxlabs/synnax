// Copyright 2025 Synnax Labs, Inc.
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
	"time"

	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/clamp"
)

const (
	// TimeStampMin represents the minimum value for a TimeStamp
	TimeStampMin = TimeStamp(0)
	// TimeStampMax represents the maximum value for a TimeStamp
	TimeStampMax = TimeStamp(^uint64(0) >> 1)
)

// TimeStamp stores an epoch time in nanoseconds.
type TimeStamp int64

var (
	_ json.Marshaler   = TimeStamp(0)
	_ json.Unmarshaler = (*TimeStamp)(nil)
)

// UnmarshalJSON implements json.Unmarshaler.
func (ts *TimeStamp) UnmarshalJSON(b []byte) error {
	n, err := binary.UnmarshalJSONStringInt64(b)
	*ts = TimeStamp(n)
	return err
}

// MarshalJSON implements json.Marshaler.
func (ts TimeStamp) MarshalJSON() ([]byte, error) {
	return binary.MarshalStringInt64(int64(ts))
}

// Now returns the current time as a TimeStamp.
func Now() TimeStamp { return NewTimeStamp(time.Now()) }

// NewTimeStamp creates a new TimeStamp from a time.Time.
func NewTimeStamp(t time.Time) TimeStamp { return TimeStamp(t.UnixNano()) }

// String returns the timestamp in the string format. All digits after are truncated.
// "2006-01-02T15:04:05.999Z"
// String implements fmt.Stringer
func (ts TimeStamp) String() string {
	if ts == TimeStampMax {
		return "end of time"
	}
	return ts.Time().UTC().Format("2006-01-02T15:04:05.999Z")
}

// Time returns the time.Time representation of the TimeStamp.
func (ts TimeStamp) Time() time.Time { return time.Unix(0, int64(ts)) }

// IsZero returns true if the TimeStamp is TimeStampMin.
func (ts TimeStamp) IsZero() bool { return ts == TimeStampMin }

// After returns true if the TimeStamp is greater than the provided one.
func (ts TimeStamp) After(t TimeStamp) bool { return ts > t }

// AfterEq returns true if ts is less than or equal t t.
func (ts TimeStamp) AfterEq(t TimeStamp) bool { return ts >= t }

// Before returns true if the TimeStamp is less than the provided one.
func (ts TimeStamp) Before(t TimeStamp) bool { return ts < t }

// BeforeEq returns true if ts ie less than or equal to t.
func (ts TimeStamp) BeforeEq(t TimeStamp) bool { return ts <= t }

// Add returns a new TimeStamp with the provided TimeSpan added to it.
func (ts TimeStamp) Add(tspan TimeSpan) TimeStamp {
	return TimeStamp(clamp.AddInt64(int64(ts), int64(tspan)))
}

// Since returns a TimeSpan representing the amount of time that has passed
// since the provided TimeStamp.
func Since(time TimeStamp) TimeSpan {
	return TimeSpan(Now() - time)
}

// Sub returns a new TimeStamp with the provided TimeSpan subtracted from it.
func (ts TimeStamp) Sub(tspan TimeSpan) TimeStamp { return ts.Add(-tspan) }

// SpanRange constructs a new TimeRange with the TimeStamp and provided TimeSpan.
func (ts TimeStamp) SpanRange(span TimeSpan) TimeRange {
	return ts.Range(ts.Add(span)).MakeValid()
}

// Range constructs a new TimeRange with the TimeStamp and provided TimeStamp.
func (ts TimeStamp) Range(ts2 TimeStamp) TimeRange { return TimeRange{ts, ts2} }

// Span returns a TimeSpan representing the amount of time between ts and the
// given time span. The returned span is positive if t is after ts, and negative
// if t is before is ts.
func (ts TimeStamp) Span(t TimeStamp) TimeSpan { return TimeSpan(t - ts) }

const (
	// NanosecondTS is a TimeStamp 1 nanosecond after the unix epoch.
	NanosecondTS = TimeStamp(1)
	// MicrosecondTS is a TimeStamp 1 microsecond after the unix epoch.
	MicrosecondTS = 1000 * NanosecondTS
	// MillisecondTS is a TimeStamp 1 millisecond after the unix epoch.
	MillisecondTS = 1000 * MicrosecondTS
	// SecondTS is a TimeStamp 1 second after the unix epoch.
	SecondTS = 1000 * MillisecondTS
	// MinuteTS is a TimeStamp 1 minute after the unix epoch.
	MinuteTS = 60 * SecondTS
	// HourTS is a TimeStamp 1 hour after the unix epoch.
	HourTS = 60 * MinuteTS
	// DayTS is a TimeStamp 1 day after the unix epoch.
	DayTS = 24 * HourTS
)
