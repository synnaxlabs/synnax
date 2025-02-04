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
	"fmt"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/clamp"
	"math"
	"strconv"
	"strings"
	"time"
)

const (
	// TimeStampMin represents the minimum value for a TimeStamp
	TimeStampMin = TimeStamp(0)
	// TimeStampMax represents the maximum value for a TimeStamp
	TimeStampMax = TimeStamp(^uint64(0) >> 1)
)

// TimeStamp stores an epoch time in nanoseconds.
type TimeStamp int64

func (ts *TimeStamp) UnmarshalJSON(b []byte) error {
	n, err := binary.UnmarshalStringInt64(b)
	*ts = TimeStamp(n)
	return err
}

func (ts *TimeStamp) MarshalJSON() ([]byte, error) {
	return []byte("\"" + strconv.Itoa(int(*ts)) + "\""), nil
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

// RawString returns the timestamp in nanoseconds.
func (ts TimeStamp) RawString() string { return strconv.Itoa(int(ts)) + "ns" }

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

// Sub returns a new TimeStamp with the provided TimeSpan subtracted from it.
func (ts TimeStamp) Sub(tspan TimeSpan) TimeStamp { return ts.Add(-tspan) }

func (ts TimeStamp) Abs() TimeStamp {
	if ts < 0 {
		return -ts
	}
	return ts
}

// SpanRange constructs a new TimeRange with the TimeStamp and provided TimeSpan.
func (ts TimeStamp) SpanRange(span TimeSpan) TimeRange {
	rng := ts.Range(ts.Add(span))
	if !rng.Valid() {
		rng = rng.Swap()
	}
	return rng
}

// Range constructs a new TimeRange with the TimeStamp and provided TimeStamp.
func (ts TimeStamp) Range(ts2 TimeStamp) TimeRange { return TimeRange{ts, ts2} }

func (ts TimeStamp) Span(t TimeStamp) TimeSpan { return TimeSpan(t - ts) }

// TimeRange represents a range of time between two TimeStamp. It's important
// to note that the start of the range is inclusive, while the end of the range is
// exclusive.
type TimeRange struct {
	// Start is the start of the range.
	Start TimeStamp `json:"start" msgpack:"start"`
	// End is the end of the range.
	End TimeStamp `json:"end" msgpack:"end"`
}

// Span returns the TimeSpan that the TimeRange occupies.
func (tr TimeRange) Span() TimeSpan { return TimeSpan(tr.End - tr.Start) }

// IsZero returns true if the TimeSpan of TimeRange is empty.
func (tr TimeRange) IsZero() bool { return tr.Span().IsZero() }

// BoundBy limits the time range to the provided bounds.
func (tr TimeRange) BoundBy(otr TimeRange) TimeRange {
	if otr.Start.After(tr.Start) {
		tr.Start = otr.Start
	}
	if otr.Start.After(tr.End) {
		tr.End = otr.Start
	}
	if otr.End.Before(tr.End) {
		tr.End = otr.End
	}
	if otr.End.Before(tr.Start) {
		tr.Start = otr.End
	}
	return tr
}

// ContainsStamp returns true if the TimeRange contains the provided TimeStamp
func (tr TimeRange) ContainsStamp(stamp TimeStamp) bool {
	return stamp.AfterEq(tr.Start) && stamp.Before(tr.End)
}

// ContainsRange returns true if provided TimeRange contains the provided TimeRange.
// Returns true if the two ranges are equal.
func (tr TimeRange) ContainsRange(rng TimeRange) bool {
	return rng.Start.AfterEq(tr.Start) && rng.End.BeforeEq(tr.End)
}

// OverlapsWith returns true if the provided TimeRange overlaps with tr.
// Note that a range with span 0 is treated as a timestamp.
func (tr TimeRange) OverlapsWith(rng TimeRange) bool {
	if tr == rng {
		return true
	}

	vTr := tr.MakeValid()
	rng = rng.MakeValid()

	if rng.Start == vTr.Start {
		return true
	}

	if rng.End == vTr.Start || rng.Start == vTr.End {
		return false
	}

	return tr.ContainsStamp(rng.End) ||
		tr.ContainsStamp(rng.Start) ||
		rng.ContainsStamp(tr.Start) ||
		rng.ContainsStamp(tr.End)
}

func (tr TimeRange) MakeValid() TimeRange {
	if tr.Valid() {
		return tr
	}
	return tr.Swap()
}

func (tr TimeRange) Swap() TimeRange { return TimeRange{Start: tr.End, End: tr.Start} }

func (tr TimeRange) Valid() bool { return tr.Span() >= 0 }

func (tr TimeRange) Midpoint() TimeStamp { return tr.Start.Add(tr.Span() / 2) }

// RawString displays the time range with both timestamps in raw string format.
func (tr TimeRange) RawString() string {
	return tr.Start.String() + " - " + tr.End.String()
}

// String displays the time range with both timestamps as formatted time.
// String implements fmt.Stringer.
func (tr TimeRange) String() string {
	return tr.Start.String() + " - " + tr.End.String()
}

func (tr TimeRange) Union(rng TimeRange) (ret TimeRange) {
	if tr.Start.Before(rng.Start) {
		ret.Start = tr.Start
	} else {
		ret.Start = rng.Start
	}

	if tr.End.After(rng.End) {
		ret.End = tr.End
	} else {
		ret.End = rng.End
	}

	return
}

func (tr TimeRange) Intersection(rng TimeRange) (ret TimeRange) {
	if tr.Start.Before(rng.Start) {
		ret.Start = rng.Start
	} else {
		ret.Start = tr.Start
	}

	if tr.End.After(rng.End) {
		ret.End = rng.End
	} else {
		ret.End = tr.End
	}

	return
}

var (
	// TimeRangeMax represents the maximum possible value for a TimeRange.
	TimeRangeMax = TimeRange{Start: TimeStampMin, End: TimeStampMax}
	// TimeRangeMin represents the minimum possible value for a TimeRange.
	TimeRangeMin = TimeRange{Start: TimeStampMax, End: TimeStampMin}
	// TimeRangeZero represents the zero value for a TimeRange.
	TimeRangeZero = TimeRange{Start: TimeStampMin, End: TimeStampMin}
)

// TimeSpan represents a duration of time in nanoseconds.
type TimeSpan int64

func (ts *TimeSpan) MarshalJSON() ([]byte, error) {
	return []byte("\"" + strconv.Itoa(int(*ts)) + "\""), nil
}

const (
	// TimeSpanZero represents the zero value for a TimeSpan.
	TimeSpanZero = TimeSpan(0)
	// TimeSpanMax represents the maximum possible TimeSpan.
	TimeSpanMax = TimeSpan(^uint64(0) >> 1)
)

var (
	_ json.Unmarshaler = (*TimeSpan)(nil)
	_ json.Marshaler   = (*TimeSpan)(nil)
)

func (ts *TimeSpan) UnmarshalJSON(b []byte) error {
	n, err := binary.UnmarshalStringInt64(b)
	*ts = TimeSpan(n)
	return err
}

// Duration converts TimeSpan to a values.Duration.
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

// String returns the timespan in a formated duration.
// String implements fmt.Stringer.
// String returns a string representation of the TimeSpan
func (ts TimeSpan) String() string {
	if ts == TimeSpanMax {
		return "max time span"
	}
	positiveTS := ts
	if positiveTS == 0 {
		return "0s"
	}

	var parts []string

	if positiveTS < 0 {
		parts = append(parts, "-")
		positiveTS = -ts
	}

	if positiveTS >= Day {
		days := positiveTS / Day
		parts = append(parts, fmt.Sprintf("%dd", days))
		positiveTS -= days * Day
	}
	if positiveTS >= Hour {
		hours := positiveTS / Hour
		parts = append(parts, fmt.Sprintf("%dh", hours))
		positiveTS -= hours * Hour
	}
	if positiveTS >= Minute {
		minutes := positiveTS / Minute
		parts = append(parts, fmt.Sprintf("%dm", minutes))
		positiveTS -= minutes * Minute
	}
	if positiveTS >= Second {
		seconds := positiveTS / Second
		parts = append(parts, fmt.Sprintf("%ds", seconds))
		positiveTS -= seconds * Second
	}
	if positiveTS >= Millisecond {
		milliseconds := positiveTS / Millisecond
		parts = append(parts, fmt.Sprintf("%dms", milliseconds))
		positiveTS -= milliseconds * Millisecond
	}
	if positiveTS >= Microsecond {
		microseconds := positiveTS / Microsecond
		parts = append(parts, fmt.Sprintf("%dµs", microseconds))
		positiveTS -= microseconds * Microsecond
	}
	if positiveTS > 0 {
		parts = append(parts, fmt.Sprintf("%dns", positiveTS))
	}

	return strings.Join(parts, " ")
}

// RawString returns the timespan in nanoseconds.
func (ts TimeSpan) RawString() string { return strconv.Itoa(int(ts)) + "ns" }

const (
	Nanosecond    = TimeSpan(1)
	NanosecondTS  = TimeStamp(1)
	Microsecond   = 1000 * Nanosecond
	MicrosecondTS = 1000 * NanosecondTS
	Millisecond   = 1000 * Microsecond
	MillisecondTS = 1000 * MicrosecondTS
	Second        = 1000 * Millisecond
	SecondTS      = 1000 * MillisecondTS
	Minute        = 60 * Second
	MinuteTS      = 60 * SecondTS
	Hour          = 60 * Minute
	HourTS        = 60 * MinuteTS
	Day           = 24 * Hour
	DayTS         = 24 * HourTS
)

// Size represents the size of an element in bytes.
type Size int64

const (
	ByteSize = Size(1)
	Kilobyte = 1024 * ByteSize
	Megabyte = 1024 * Kilobyte
	Gigabyte = 1024 * Megabyte
)

type Offset = Size

// String implements fmt.Stringer.
func (s Size) String() string { return strconv.Itoa(int(s)) + "B" }

// Rate represents a rate in Hz.
type Rate float64

// Period returns a TimeSpan representing the period of the Rate.
func (dr Rate) Period() TimeSpan { return TimeSpan(1 / float64(dr) * float64(Second)) }

// SampleCount returns n integer representing the number of samples in the provided Span.
func (dr Rate) SampleCount(t TimeSpan) int { return int(t.Seconds() * float64(dr)) }

// Span returns a TimeSpan representing the number of samples that occupy the provided Span.
func (dr Rate) Span(sampleCount int) TimeSpan {
	return dr.Period() * TimeSpan(sampleCount)
}

// SizeSpan returns a TimeSpan representing the number of samples that occupy a provided number of bytes.
func (dr Rate) SizeSpan(size Size, Density Density) TimeSpan {
	return dr.Span(int(size) / int(Density))
}

// ClosestGE returns the closest larger timestamp that is a whole number multiple of the rate's period.
func (dr Rate) ClosestGE(ts TimeStamp) TimeStamp {
	if TimeSpan(ts)%dr.Period() == 0 {
		return ts
	}

	return ts.Add(dr.Period() - TimeSpan(ts)%dr.Period())
}

// ClosestLE returns the closest smaller timestamp that is a whole number multiple of the rate's period.
func (dr Rate) ClosestLE(ts TimeStamp) TimeStamp {
	return ts.Sub(TimeSpan(ts) % dr.Period())
}

const (
	// Hz represents a data rate of 1 Hz.
	Hz  Rate = 1
	KHz      = 1000 * Hz
	MHz      = 1000 * KHz
)

// Density represents a density in bytes per value.
type Density uint32

func (d Density) SampleCount(size Size) int64 {
	if d == 0 {
		panic("attempted to call sample count on undefined density")
	}
	return int64(size) / int64(d)
}

func (d Density) Size(sampleCount int64) Size { return Size(sampleCount) * Size(d) }

const (
	DensityUnknown   Density = 0
	Bit128           Density = 16
	Bit64            Density = 8
	Bit32            Density = 4
	Bit16            Density = 2
	Bit8             Density = 1
	TimeStampDensity         = Bit64
	TimeSpanDensity          = Bit64
)

// AlignmentPair is essentially two array index values that can be used to represent
// the location of a sample within a group of arrays. For example, if you have two arrays
// that have 50 elements each, and you want the 15th element of the second array, you would
// use NewAlignmentPair(1, 15). The first index is called the 'domain index' and the second
// index is called the 'sample index'. The domain index is the index of the array, and the
// sample index is the index of the sample within that array.
//
// You may think a better design is to just use a single number that overflows the arrays
// before it i.e. the value of our previous example would be 50 + 14 = 64. However, this
// requires us to know the size of all arrays, which is not always possible.
//
// While not as meaningful as a single number, AlignmentPair is a uint64 that guarantees
// that a larger value is, in fact, 'positionally' after a smaller value. This is useful
// for ordering samples correctly.
type AlignmentPair uint64

var (
	_ json.Unmarshaler = (*AlignmentPair)(nil)
	_ json.Marshaler   = (*AlignmentPair)(nil)
)

// NewAlignmentPair takes the given array index and sample index within that array and
// returns a new AlignmentPair (see AlignmentPair for more information).
func NewAlignmentPair(domainIdx, sampleIdx uint32) AlignmentPair {
	return AlignmentPair(domainIdx)<<32 | AlignmentPair(sampleIdx)
}

// ZeroLeadingAlignment represents the start of a region reserved for written data that
// has not yet been persisted. This is useful for correctly ordering new data while
// ensuring that it is significantly after any persisted data.
const ZeroLeadingAlignment = math.MaxUint32 - 1e6

// LeadingAlignment returns an AlignmentPair whose array index is the maximum possible value
// and whose sample index is the provided value.
func LeadingAlignment(domainIdx, sampleIdx uint32) AlignmentPair {
	return NewAlignmentPair(ZeroLeadingAlignment+domainIdx, sampleIdx)
}

// DomainIndex returns the domain index of the AlignmentPair. See AlignmentPair for more information.
func (a AlignmentPair) DomainIndex() uint32 { return uint32(a >> 32) }

// SampleIndex returns the sample index of the AlignmentPair. See AlignmentPair for more information.
func (a AlignmentPair) SampleIndex() uint32 { return uint32(a) }

// UnmarshalJSON implements json.Unmarshaler.
func (a *AlignmentPair) UnmarshalJSON(b []byte) error {
	n, err := binary.UnmarshalStringUint64(b)
	*a = AlignmentPair(n)
	return err
}

// MarshalJSON implements json.Marshaler.
func (a *AlignmentPair) MarshalJSON() ([]byte, error) {
	return []byte("\"" + strconv.FormatUint(uint64(*a), 10) + "\""), nil
}

func (a AlignmentPair) AddSamples(samples uint32) AlignmentPair {
	return NewAlignmentPair(a.DomainIndex(), a.SampleIndex()+samples)
}
