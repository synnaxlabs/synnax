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
	"time"

	v0 "github.com/synnaxlabs/x/telem/types/v0"
)

const (
	// TimeStampMin represents the minimum value for a TimeStamp
	TimeStampMin TimeStamp = v0.TimeStampMin
	// TimeStampMax represents the maximum value for a TimeStamp
	TimeStampMax TimeStamp = v0.TimeStampMax
	// NanosecondTS is a TimeStamp 1 nanosecond after the unix epoch.
	NanosecondTS TimeStamp = v0.NanosecondTS
	// MicrosecondTS is a TimeStamp 1 microsecond after the unix epoch.
	MicrosecondTS TimeStamp = v0.MicrosecondTS
	// MillisecondTS is a TimeStamp 1 millisecond after the unix epoch.
	MillisecondTS TimeStamp = v0.MillisecondTS
	// SecondTS is a TimeStamp 1 second after the unix epoch.
	SecondTS TimeStamp = v0.SecondTS
	// MinuteTS is a TimeStamp 1 minute after the unix epoch.
	MinuteTS TimeStamp = v0.MinuteTS
	// HourTS is a TimeStamp 1 hour after the unix epoch.
	HourTS TimeStamp = v0.HourTS
	// DayTS is a TimeStamp 1 day after the unix epoch.
	DayTS TimeStamp = v0.DayTS
)

// Now returns the current time as a TimeStamp.
func Now() TimeStamp { return NewTimeStamp(time.Now()) }

// NewTimeStamp creates a new TimeStamp from a time.Time.
func NewTimeStamp(t time.Time) TimeStamp { return TimeStamp(t.UnixNano()) }

// Since returns a TimeSpan representing the amount of time that has passed since the
// provided TimeStamp.
func Since(ts TimeStamp) TimeSpan { return TimeSpan(Now() - ts) }
