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
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/clamp"
	"github.com/synnaxlabs/x/types"
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

func (ts *TimeStamp) UnmarshalJSON(b []byte) error {
	n, err := binary.UnmarshalStringInt64(b)
	*ts = TimeStamp(n)
	return err
}

func (ts TimeStamp) MarshalJSON() ([]byte, error) {
	return binary.MarshalStringInt64(int64(ts))
}

// Now returns the current time as a TimeStamp.
func Now() TimeStamp { return NewTimeStamp(time.Now()) }

func Since(ts TimeStamp) TimeSpan { return ts.Span(Now()) }

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

// NewSecondsRange creates a new TimeRange between start and end seconds.
func NewSecondsRange(start, end int) TimeRange {
	return TimeRange{Start: TimeStamp(start) * SecondTS, End: TimeStamp(end) * SecondTS}
}

// Span returns the TimeSpan that the TimeRange occupies.
func (tr TimeRange) Span() TimeSpan { return TimeSpan(tr.End - tr.Start) }

// IsZero returns true if both the start and end timestamps are zero.
func (tr TimeRange) IsZero() bool { return tr.Start.IsZero() && tr.End.IsZero() }

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

// ContainsRange returns true if the TimeRange contains the provided TimeRange.
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
	return tr.Start.RawString() + " - " + tr.End.RawString()
}

// String displays the time range with both timestamps in a human-readable format,
// omitting redundant time components between start and end times.
func (tr TimeRange) String() string {
	if tr.Start == TimeStampMax || tr.End == TimeStampMax {
		return "end of time"
	}

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
	if nanos < 1000 {
		return fmt.Sprintf("%09d", nanos)
	} else if nanos < 1000000 {
		microseconds := nanos / 1000
		return fmt.Sprintf("%06d", microseconds)
	}
	milliseconds := nanos / 1000000
	return fmt.Sprintf("%03d", milliseconds)
}

func (tr TimeRange) MaxUnion(rng TimeRange) (ret TimeRange) {
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

// PointIntersection returns the time between the start of the time range and the given
// timestamp and the time between the end of the time range and the given timestamp.
func (tr TimeRange) PointIntersection(ts TimeStamp) (before TimeSpan, after TimeSpan) {
	return tr.Start.Span(ts), -tr.End.Span(ts)
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

var (
	_ json.Unmarshaler = (*TimeSpan)(nil)
	_ json.Marshaler   = TimeSpan(0)
)

func (ts TimeSpan) MarshalJSON() ([]byte, error) {
	return binary.MarshalStringInt64(int64(ts))
}

const (
	// TimeSpanZero represents the zero value for a TimeSpan.
	TimeSpanZero = TimeSpan(0)
	// TimeSpanMax represents the maximum possible TimeSpan.
	TimeSpanMax = TimeSpan(^uint64(0) >> 1)
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

	if len(parts) == 0 {
		return "0ns"
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
	DensityUnknown Density = 0
	Bit128         Density = 16
	Bit64          Density = 8
	Bit32          Density = 4
	Bit16          Density = 2
	Bit8           Density = 1
)

// Alignment is essentially two array index values that can be used to represent
// the location of a sample within a group of arrays. For example, if you have two arrays
// that have 50 elements each, and you want the 15th element of the second array, you would
// use NewAlignment(1, 15). The first index is called the 'domain index' and the second
// index is called the 'sample index'. The domain index is the index of the array, and the
// sample index is the index of the sample within that array.
//
// You may think a better design is to just use a single number that overflows the arrays
// before it i.e. the value of our previous example would be 50 + 14 = 64. However, this
// requires us to know the size of all arrays, which is not always possible.
//
// While not as meaningful as a single number, Alignment is a uint64 that guarantees
// that a larger value is, in fact, 'positionally' after a smaller value. This is useful
// for ordering samples correctly.
type Alignment uint64

var (
	_ json.Unmarshaler = (*Alignment)(nil)
	_ json.Marshaler   = (*Alignment)(nil)
)

// NewAlignment takes the given array index and sample index within that array and
// returns a new Alignment (see Alignment for more information).
func NewAlignment(domainIdx, sampleIdx uint32) Alignment {
	return Alignment(domainIdx)<<32 | Alignment(sampleIdx)
}

// ZeroLeadingAlignment represents the start of a region reserved for written data that
// has not yet been persisted. This is useful for correctly ordering new data while
// ensuring that it is significantly after any persisted data.
const ZeroLeadingAlignment uint32 = math.MaxUint32 - 1e6
const MaxAlignmentPair = Alignment(math.MaxUint64)

// LeadingAlignment returns an Alignment whose array index is the maximum possible value
// and whose sample index is the provided value.
func LeadingAlignment(domainIdx, sampleIdx uint32) Alignment {
	return NewAlignment(ZeroLeadingAlignment+domainIdx, sampleIdx)
}

// DomainIndex returns the domain index of the Alignment. See Alignment for more information.
func (a Alignment) DomainIndex() uint32 { return uint32(a >> 32) }

// SampleIndex returns the sample index of the Alignment. See Alignment for more information.
func (a Alignment) SampleIndex() uint32 { return uint32(a) }

// String implements fmt.Stringer to return a nicely formatted string representing the
// alignment.
func (a Alignment) String() string {
	return fmt.Sprintf("%v-%v", a.DomainIndex(), a.SampleIndex())
}

// UnmarshalJSON implements json.Unmarshaler.
func (a *Alignment) UnmarshalJSON(b []byte) error {
	n, err := binary.UnmarshalStringUint64(b)
	*a = Alignment(n)
	return err
}

// MarshalJSON implements json.Marshaler.
func (a Alignment) MarshalJSON() ([]byte, error) {
	return binary.MarshalStringUint64(uint64(a))
}

func (a Alignment) AddSamples(samples uint32) Alignment {
	return NewAlignment(a.DomainIndex(), a.SampleIndex()+samples)
}

// DataType is a string that represents a data type.
type DataType string

// Density returns the density of the given data type. If the data type has no known
// density, DensityUnknown is returned.
func (d DataType) Density() Density {
	switch d {
	case TimeStampT, Float64T, Uint64T, Int64T:
		return Bit64
	case Float32T, Int32T, Uint32T:
		return Bit32
	case Int16T, Uint16T:
		return Bit16
	case Int8T, Uint8T:
		return Bit8
	case UUIDT:
		return Bit128
	default:
		return DensityUnknown
	}
}

// IsVariable returns true if the data type has a variable density i.e. is a string,
// JSON, or bytes.
func (d DataType) IsVariable() bool {
	return d == BytesT || d == StringT || d == JSONT
}

var dataTypes = map[string]DataType{
	"timestamp": TimeStampT,
	"uuid":      UUIDT,
	"float64":   Float64T,
	"float32":   Float32T,
	"int64":     Int64T,
	"int32":     Int32T,
	"int16":     Int16T,
	"int8":      Int8T,
	"uint8":     Uint8T,
	"uint64":    Uint64T,
	"uint32":    Uint32T,
	"uint16":    Uint16T,
	"bytes":     BytesT,
	"string":    StringT,
	"json":      JSONT,
}

func InferDataType[T any]() DataType {
	name := strings.ToLower(types.Name[T]())
	if dt, ok := dataTypes[name]; ok {
		return dt
	}
	panic(fmt.Sprintf("unknown data type %s", name))
}

var (
	UnknownT   DataType = ""
	TimeStampT          = DataType("timestamp")
	UUIDT               = DataType("uuid")
	Float64T   DataType = "float64"
	Float32T   DataType = "float32"
	Int64T     DataType = "int64"
	Int32T     DataType = "int32"
	Int16T     DataType = "int16"
	Int8T      DataType = "int8"
	Uint64T    DataType = "uint64"
	Uint32T    DataType = "uint32"
	Uint16T    DataType = "uint16"
	Uint8T     DataType = "uint8"
	BytesT     DataType = "bytes"
	StringT    DataType = "string"
	JSONT      DataType = "json"
)
