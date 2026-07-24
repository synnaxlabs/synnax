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
