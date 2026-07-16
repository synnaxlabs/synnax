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

	latest "github.com/synnaxlabs/x/telem/types/v0"
)

// Density represents a density in bytes per value.
type Density = latest.Density

// NumericSample represents any numeric value that can be stored in a Series and have
// mathematical operations performed on it.
type NumericSample = latest.NumericSample

// FixedSample represents any numeric value that can be stored in a Series and has a
// fixed density.
type FixedSample = latest.FixedSample

// VariableSample is a type that can be stored in a variable-density series.
type VariableSample = latest.VariableSample

// Sample represents any value that can be stored in a non-JSON Series.
type Sample = latest.Sample

// AlignmentBounds is a set of lower and upper bounds for the alignment of a
// multi-sample data structure (such as a Series or MultiSeries). The lower bound
// represents the alignment of the first sample, while the upper bound represents the
// alignment of the last sample + 1. The lower bound is inclusive, while the upper bound
// is exclusive.
type AlignmentBounds = latest.AlignmentBounds

// MultiSeries is a collection of ordered Series that share the same data type.
type MultiSeries = latest.MultiSeries

const (
	// MaxAlignment is the maximum possible value for an alignment.
	MaxAlignment = latest.MaxAlignment

	// UnknownDensity is for type structure occupying an unknown number of bytes.
	UnknownDensity = latest.UnknownDensity
	// Bit128 is for a type occupying 16 bytes.
	Bit128 = latest.Bit128
	// Bit64 is for a type occupying 8 bytes.
	Bit64 = latest.Bit64
	// Bit32 is for a type occupying 4 bytes.
	Bit32 = latest.Bit32
	// Bit16 is for a data type occupying 2 bytes.
	Bit16 = latest.Bit16
	// Bit8 is for a data type occupying 1 byte.
	Bit8 = latest.Bit8

	// UnknownT is an unknown data type.
	UnknownT = latest.UnknownT
	// Uint8T is a data type for an 8-bit unsigned integer, i.e., a single byte.
	Uint8T = latest.Uint8T
	// Uint16T is a data type for a 16-bit unsigned integer.
	Uint16T = latest.Uint16T
	// Uint32T is a data type for a 32-bit unsigned integer.
	Uint32T = latest.Uint32T
	// Uint64T is a data type for a 64-bit unsigned integer.
	Uint64T = latest.Uint64T
	// Int8T is a data type for an 8-bit integer.
	Int8T = latest.Int8T
	// Int16T is a data type for a 16-bit integer.
	Int16T = latest.Int16T
	// Int32T is a data type for a 32-bit integer.
	Int32T = latest.Int32T
	// Int64T is a data type for a 64-bit integer.
	Int64T = latest.Int64T
	// Float32T is a data type for a 32-bit IEEE-754 floating point number.
	Float32T = latest.Float32T
	// Float64T is a data type for a 64-bit IEEE-754 floating point number.
	Float64T = latest.Float64T
	// TimeStampT is a data type for a signed 64-bit nanosecond count since the Unix
	// epoch.
	TimeStampT = latest.TimeStampT
	// UUIDT is a data type for a 128-bit UUID value.
	UUIDT = latest.UUIDT
	// StringT is a variable density data type for a UTF-8 encoded string.
	StringT = latest.StringT
	// BytesT is a variable density data type for an arbitrary byte array.
	BytesT = latest.BytesT
	// JSONT is a variable density data type for UTF-8 encoded JSON text.
	JSONT = latest.JSONT

	// Hertz is a data rate of 1 Hz.
	Hertz = latest.Hertz
	// Kilohertz is a data rate of 1 kHz.
	Kilohertz = latest.Kilohertz
	// Megahertz is a data rate of 1 MHz.
	Megahertz = latest.Megahertz

	// Byte is a single byte.
	Byte = latest.Byte
	// Kilobyte is 1,000 bytes.
	Kilobyte = latest.Kilobyte
	// Megabyte is 1,000 kilobytes.
	Megabyte = latest.Megabyte
	// Gigabyte is 1,000 megabytes.
	Gigabyte = latest.Gigabyte
	// Terabyte is 1,000 gigabytes.
	Terabyte = latest.Terabyte
	// Petabyte is 1,000 terabytes.
	Petabyte = latest.Petabyte
	// Exabyte is 1,000 petabytes.
	Exabyte = latest.Exabyte

	// TimeStampMin represents the minimum value for a TimeStamp
	TimeStampMin = latest.TimeStampMin
	// TimeStampMax represents the maximum value for a TimeStamp
	TimeStampMax = latest.TimeStampMax
	// NanosecondTS is a TimeStamp 1 nanosecond after the unix epoch.
	NanosecondTS = latest.NanosecondTS
	// MicrosecondTS is a TimeStamp 1 microsecond after the unix epoch.
	MicrosecondTS = latest.MicrosecondTS
	// MillisecondTS is a TimeStamp 1 millisecond after the unix epoch.
	MillisecondTS = latest.MillisecondTS
	// SecondTS is a TimeStamp 1 second after the unix epoch.
	SecondTS = latest.SecondTS
	// MinuteTS is a TimeStamp 1 minute after the unix epoch.
	MinuteTS = latest.MinuteTS
	// HourTS is a TimeStamp 1 hour after the unix epoch.
	HourTS = latest.HourTS
	// DayTS is a TimeStamp 1 day after the unix epoch.
	DayTS = latest.DayTS

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

var (
	// AlignmentBoundsZero is a set of alignment bounds whose lower and upper bound are
	// both zero.
	AlignmentBoundsZero = latest.AlignmentBoundsZero
	// ByteOrder is the standard order for encoding/decoding numeric values across the
	// Synnax telemetry ecosystem.
	ByteOrder = latest.ByteOrder
	// TimeRangeSchema is a zyn schema for parsing a time range.
	TimeRangeSchema = latest.TimeRangeSchema
	// TimeRangeMax represents the maximum possible value for a TimeRange.
	TimeRangeMax = latest.TimeRangeMax
	// TimeRangeMin represents the minimum possible value for a TimeRange.
	TimeRangeMin = latest.TimeRangeMin
	// TimeRangeZero represents the zero value for a TimeRange.
	TimeRangeZero = latest.TimeRangeZero
)

// NewAlignment takes the given array index and sample index within that array and
// returns a new Alignment (see Alignment for more information).
func NewAlignment(domainIdx, sampleIdx uint32) Alignment {
	return latest.NewAlignment(domainIdx, sampleIdx)
}

// InferDataType infers the data type of the given Sample.
func InferDataType[T Sample]() DataType { return latest.InferDataType[T]() }

// Now returns the current time as a TimeStamp.
func Now() TimeStamp { return latest.Now() }

// NewTimeStamp creates a new TimeStamp from a time.Time.
func NewTimeStamp(t time.Time) TimeStamp { return latest.NewTimeStamp(t) }

// Since returns a TimeSpan representing the amount of time that has passed
// since the provided TimeStamp.
func Since(time TimeStamp) TimeSpan { return latest.Since(time) }

// NewRangeSeconds creates a new TimeRange between start and end seconds.
func NewRangeSeconds(start, end int) TimeRange {
	return latest.NewRangeSeconds(start, end)
}

// NewSeries creates a new Series from a slice of sample values. It automatically
// determines the data type from the type parameter.
func NewSeries[T Sample](data []T) Series { return latest.NewSeries(data) }

// NewSeriesV is a variadic version of NewSeries.
func NewSeriesV[T Sample](data ...T) Series { return latest.NewSeriesV(data...) }

// MakeSeries allocates a new Series with the specified DataType and length. Note that
// this function allocates a length and not a capacity.
func MakeSeries(dt DataType, len int) Series { return latest.MakeSeries(dt, len) }

// NewSeriesSecondsTSV creates a new Series containing TimeStamp values. All input
// timestamps are multiplied by SecondTS to convert them to the standard time unit used
// in the system.
func NewSeriesSecondsTSV(data ...TimeStamp) Series {
	return latest.NewSeriesSecondsTSV(data...)
}

// NewJSONSeries creates a new JSON Series from a slice of JSON values. It returns an
// error if the data cannot be marshalled into JSON.
func NewJSONSeries[T any](data []T) (Series, error) {
	return latest.NewJSONSeries(data)
}

// NewJSONSeriesV constructs a new JSON Series from an arbitrary set of JSON values,
// marshaling each one in the process. It returns an error if the data cannot be
// marshalled into JSON.
func NewJSONSeriesV[T any](data ...T) (Series, error) {
	return latest.NewJSONSeriesV(data...)
}

// MarshalVariableSample wraps a single variable-length sample with a uint32 LE length
// prefix. This is useful for code that accumulates samples into a Series.Data buffer
// incrementally rather than using NewSeriesV.
func MarshalVariableSample(sample []byte) []byte {
	return latest.MarshalVariableSample(sample)
}

// UnmarshalSeries converts a Series back into a slice of the specified data type. Note
// that this function does NOT check the Series' DataType, it simply unmarshals the data
// according to type T.
func UnmarshalSeries[T Sample](series Series) []T {
	return latest.UnmarshalSeries[T](series)
}

// UnmarshalJSONSeries unmarshals a JSON-encoded series into a slice of JSON values
// of the specified type T. This function does NOT check the Series' DataType, it simply
// unmarshals the data according to type T.
func UnmarshalJSONSeries[T any](s Series) ([]T, error) {
	return latest.UnmarshalJSONSeries[T](s)
}

// Arrange creates a new Series containing count values starting from start, with each
// subsequent value incremented by spacing. For example, Arrange(0, 5, 2) produces [0,
// 2, 4, 6, 8]. Panics if count is less than or equal to 0.
func Arrange[T NumericSample](start T, count int, spacing T) Series {
	return latest.Arrange(start, count, spacing)
}

// NewSeriesFromAny creates a single-value Series from a value of type any, casting it
// to the specified DataType. Supports numeric types, strings, TimeStamp, JSON, and
// bytes. Panics if the value cannot be converted to the target DataType.
func NewSeriesFromAny(value any, dt DataType) Series {
	return latest.NewSeriesFromAny(value, dt)
}

// ValueAt returns the numeric value at the given index in the series. ValueAt supports
// negative indices, which will be wrapped around the end of the series. This function
// cannot be used for variable density series.
func ValueAt[T FixedSample](s Series, i int) T { return latest.ValueAt[T](s, i) }

// SetValueAt sets the value at the given index in the series. SetValueAt supports
// negative indices, which will be wrapped around the end of the series. This function
// cannot be used for variable density series.
func SetValueAt[T FixedSample](s Series, i int, v T) { latest.SetValueAt(s, i, v) }

// CopyValue copies the sample from src at the index srcIdx to the index srcIdx in src.
// dst and src must have the same DataType, and that DataType cannot be of variable
// density.
func CopyValue(dst, src Series, dstIdx, srcIdx int) {
	latest.CopyValue(dst, src, dstIdx, srcIdx)
}

// NewMultiSeries constructs a new MultiSeries from the given set of Series. The series
// are sorted by their alignment, and the data type of the series must be the same. If
// the data types are different, a panic will occur.
func NewMultiSeries(series []Series) MultiSeries {
	return latest.NewMultiSeries(series)
}

// NewMultiSeriesV constructs a new MultiSeries from the given set of variadic Series.
// The series are sorted by their alignment, and the data type of the series must be the
// same. If the data types are different, a panic will occur.
func NewMultiSeriesV(series ...Series) MultiSeries {
	return latest.NewMultiSeriesV(series...)
}

// MultiSeriesAtAlignment returns the value at the given alignment in the MultiSeries.
func MultiSeriesAtAlignment[T FixedSample](ms MultiSeries, alignment Alignment) T {
	return latest.MultiSeriesAtAlignment[T](ms, alignment)
}
