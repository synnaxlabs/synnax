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
	"fmt"

	"github.com/synnaxlabs/x/zyn"
)

// TimeRangeSchema is a zyn schema for parsing a time range.
var TimeRangeSchema = zyn.Object(map[string]zyn.Schema{
	"start": zyn.Int64().Coerce(),
	"end":   zyn.Int64().Coerce(),
})

// NewRangeSeconds creates a new TimeRange between start and end seconds.
func NewRangeSeconds(start, end int) TimeRange {
	return TimeRange{Start: TimeStamp(start) * SecondTS, End: TimeStamp(end) * SecondTS}
}

// Span returns the TimeSpan that the TimeRange occupies.
func (tr TimeRange) Span() TimeSpan { return TimeSpan(tr.End - tr.Start) }

// IsZero returns true if both the start and end timestamps are zero.
func (tr TimeRange) IsZero() bool { return tr.Start.IsZero() && tr.End.IsZero() }

// BoundBy limits the time range to the provided bounds.
// Example One:
//
//	[5s, 10s).BoundBy([7s, 11s)) == [7s, 10s)
//
// Example Two:
//
// [5s, 10s).BoundBy([11s, 17s)) == [11s, 11s)
func (tr TimeRange) BoundBy(bound TimeRange) TimeRange {
	if bound.Start.After(tr.Start) {
		tr.Start = bound.Start
	}
	if bound.Start.After(tr.End) {
		tr.End = bound.Start
	}
	if bound.End.Before(tr.End) {
		tr.End = bound.End
	}
	if bound.End.Before(tr.Start) {
		tr.Start = bound.End
	}
	return tr
}

// ContainsStamp returns true if the TimeRange contains the provided TimeStamp
func (tr TimeRange) ContainsStamp(stamp TimeStamp) bool {
	return stamp.AfterEq(tr.Start) && stamp.Before(tr.End)
}

// ContainsRange returns true if the TimeRange contains the provided TimeRange.
// Returns true if the two ranges are equal.
func (tr TimeRange) ContainsRange(rng TimeRange) bool {
	return rng.Start.AfterEq(tr.Start) && rng.End.BeforeEq(tr.End)
}

// OverlapsWith returns true if the provided TimeRange overlaps with this time range.
// This function will ensure that both ranges are valid (i.e. start before end) as
// part of its checks.
func (tr TimeRange) OverlapsWith(rng TimeRange) bool {
	if tr == rng {
		return true
	}

	validTR := tr.MakeValid()
	rng = rng.MakeValid()

	if rng.Start == validTR.Start {
		return true
	}

	if rng.End == validTR.Start || rng.Start == validTR.End {
		return false
	}

	return tr.ContainsStamp(rng.End) ||
		tr.ContainsStamp(rng.Start) ||
		rng.ContainsStamp(tr.Start) ||
		rng.ContainsStamp(tr.End)
}

// MakeValid returns a copy of the time range that conditionally swaps TimeRange.Start
// and TimeRange.End to make sure that TimeRange.Start is before TimeRange.End.
func (tr TimeRange) MakeValid() TimeRange {
	if tr.Valid() {
		return tr
	}
	return tr.Swap()
}

// Swap swaps TimeRange.Start with TimeRange.End.
func (tr TimeRange) Swap() TimeRange { return TimeRange{Start: tr.End, End: tr.Start} }

// Valid returns true if TimeRange.Start is before or equal to TimeRange.End
func (tr TimeRange) Valid() bool { return tr.Span() >= 0 }

// Midpoint returns the TimeStamp half-way in between TimeRange.Start and TimeRange.End.
func (tr TimeRange) Midpoint() TimeStamp { return tr.Start.Add(tr.Span() / 2) }

// String displays the time range with both timestamps in a human-readable format,
// omitting redundant time components between start and end times.
func (tr TimeRange) String() string {
	start := tr.Start.Time().UTC()
	end := tr.End.Time().UTC()

	startYear, startMonth, startDay := start.Date()
	endYear, endMonth, endDay := end.Date()
	startHour, startMin, startSec := start.Clock()
	endHour, endMin, endSec := end.Clock()
	startNano := start.Nanosecond()
	endNano := end.Nanosecond()

	var endStr string
	if startYear != endYear {
		endStr = end.Format("2006-01-02T15:04:05") + formatNanos(endNano)
	} else if startMonth != endMonth || startDay != endDay {
		endStr = end.Format("01-02T15:04:05") + formatNanos(endNano)
	} else if startHour != endHour {
		endStr = end.Format("15:04:05") + formatNanos(endNano)
	} else if startMin != endMin {
		endStr = end.Format("04:05") + formatNanos(endNano)
	} else if startSec != endSec {
		endStr = end.Format(":05") + formatNanos(endNano)
	} else if startNano != endNano {
		endStr = "." + formatSubsecond(endNano)
	} else {
		endStr = end.Format("15:04:05")
	}

	startStr := start.Format("2006-01-02T15:04:05")
	if startNano > 0 {
		startStr += "." + formatSubsecond(startNano)
	}
	startStr += "Z"

	return startStr + " - " + endStr + " (" + tr.Span().String() + ")"
}

func formatNanos(nanos int) string {
	if nanos == 0 {
		return ""
	}
	return "." + formatSubsecond(nanos)
}

func formatSubsecond(nanos int) string {
	if nanos < 1_000 {
		return fmt.Sprintf("%09d", nanos)
	} else if nanos < 1_000_000 {
		microseconds := nanos / 1_000
		return fmt.Sprintf("%06d", microseconds)
	}
	milliseconds := nanos / 1_000_000
	return fmt.Sprintf("%03d", milliseconds)
}

// Union returns the maximum possible union of this time range and the provided one.
// This function returns the time range with the maximum possible span, i.e., the
// combination of the lowest start and the highest end.
func (tr TimeRange) Union(other TimeRange) TimeRange {
	return TimeRange{Start: min(tr.Start, other.Start), End: max(tr.End, other.End)}
}

// Intersection returns the intersection of the two time ranges if one exists. If one
// does not exist, returns TimeRangeZero.
func (tr TimeRange) Intersection(rng TimeRange) (ret TimeRange) {
	if !tr.OverlapsWith(rng) {
		return TimeRangeZero
	}
	return TimeRange{Start: max(tr.Start, rng.Start), End: min(tr.End, rng.End)}
}

// Split returns the time between the start of the time range and the given
// timestamp and the time between the end of the time range and the given timestamp.
func (tr TimeRange) Split(ts TimeStamp) (before TimeRange, after TimeRange) {
	return TimeRange{Start: tr.Start, End: ts}, TimeRange{Start: ts, End: tr.End}
}

var (
	// TimeRangeMax represents the maximum possible value for a TimeRange.
	TimeRangeMax = TimeRange{Start: TimeStampMin, End: TimeStampMax}
	// TimeRangeMin represents the minimum possible value for a TimeRange.
	TimeRangeMin = TimeRange{Start: TimeStampMax, End: TimeStampMin}
	// TimeRangeZero represents the zero value for a TimeRange.
	TimeRangeZero = TimeRange{Start: TimeStampMin, End: TimeStampMin}
)
