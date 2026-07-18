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
	latest "github.com/synnaxlabs/x/telem/types/v0"
	"github.com/synnaxlabs/x/zyn"
)

// TimeRangeSchema is a zyn schema for parsing a time range.
var TimeRangeSchema = zyn.Object(map[string]zyn.Schema{
	"start": zyn.Int64().Coerce(),
	"end":   zyn.Int64().Coerce(),
})

var (
	// TimeRangeMax represents the maximum possible value for a TimeRange.
	TimeRangeMax = latest.TimeRangeMax
	// TimeRangeMin represents the minimum possible value for a TimeRange.
	TimeRangeMin = latest.TimeRangeMin
	// TimeRangeZero represents the zero value for a TimeRange.
	TimeRangeZero = latest.TimeRangeZero
)

// NewRangeSeconds creates a new TimeRange between start and end seconds.
func NewRangeSeconds(start, end int) TimeRange {
	return TimeRange{Start: TimeStamp(start) * SecondTS, End: TimeStamp(end) * SecondTS}
}
