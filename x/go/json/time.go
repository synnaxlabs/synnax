// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json

import (
	"bytes"
	"math"
	"strconv"
	"time"
	"unsafe"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// TimeFormat is the encoding of a timestamp within a JSON document.
type TimeFormat uint8

const (
	// ISO8601 is an RFC 3339 string such as "2026-01-02T03:04:05.678Z".
	ISO8601 TimeFormat = iota
	// UnixSecond is a count of seconds since the Unix epoch.
	UnixSecond
	// UnixMillisecond is a count of milliseconds since the Unix epoch.
	UnixMillisecond
	// UnixMicrosecond is a count of microseconds since the Unix epoch.
	UnixMicrosecond
	// UnixNanosecond is a count of nanoseconds since the Unix epoch.
	UnixNanosecond
)

var timeFormatNames = [...]string{
	ISO8601:         "iso8601",
	UnixSecond:      "unix_sec",
	UnixMillisecond: "unix_ms",
	UnixMicrosecond: "unix_us",
	UnixNanosecond:  "unix_ns",
}

// ParseTimeFormat parses the config form of a TimeFormat: "iso8601", "unix_sec",
// "unix_ms", "unix_us", or "unix_ns". It returns an error wrapping
// validate.ErrValidation for any other text.
func ParseTimeFormat(text string) (TimeFormat, error) {
	for f, name := range timeFormatNames {
		if text == name {
			return TimeFormat(f), nil
		}
	}
	return ISO8601, errors.Wrapf(
		validate.ErrValidation,
		`unknown time format %q: expected "iso8601", "unix_sec", "unix_ms", `+
			`"unix_us", or "unix_ns"`,
		text,
	)
}

// String returns the config form of f.
func (f TimeFormat) String() string { return timeFormatNames[f] }

// nanosExponent returns the base 10 logarithm of the nanoseconds in one unit of a
// numeric format.
func (f TimeFormat) nanosExponent() int {
	switch f {
	case UnixSecond:
		return 9
	case UnixMillisecond:
		return 6
	case UnixMicrosecond:
		return 3
	default:
		return 0
	}
}

func (f TimeFormat) nanosPerUnit() int64 {
	n := int64(1)
	for range f.nanosExponent() {
		n *= 10
	}
	return n
}

// AppendTimeStamp appends ts as a JSON value in format: a string for ISO8601, and a
// number with every digit of ts otherwise.
func AppendTimeStamp(dst []byte, ts telem.TimeStamp, format TimeFormat) []byte {
	if format == ISO8601 {
		dst = append(dst, '"')
		dst = ts.Time().UTC().AppendFormat(dst, time.RFC3339Nano)
		return append(dst, '"')
	}
	unit := format.nanosPerUnit()
	whole, frac := int64(ts)/unit, int64(ts)%unit
	if frac == 0 {
		return strconv.AppendInt(dst, whole, 10)
	}
	if frac < 0 {
		dst = append(dst, '-')
		whole, frac = -whole, -frac
	}
	dst = strconv.AppendInt(dst, whole, 10)
	dst = append(dst, '.')
	// unit+frac pads the fraction with leading zeros to the digits of the unit.
	start := len(dst)
	dst = strconv.AppendInt(dst, unit+frac, 10)
	copy(dst[start:], dst[start+1:])
	return bytes.TrimRight(dst[:len(dst)-1], "0")
}

// parseISO8601 parses an RFC 3339 timestamp in text, which it may change. It also
// accepts a lowercase "t" or a space as the date and time separator and a lowercase
// "z", which RFC 3339 §5.6 permits and time.Parse rejects.
func parseISO8601(text []byte) (telem.TimeStamp, error) {
	if len(text) > 10 && (text[10] == 't' || text[10] == ' ') {
		text[10] = 'T'
	}
	if last := len(text) - 1; last >= 0 && text[last] == 'z' {
		text[last] = 'Z'
	}
	// time.Parse keeps no reference to its input on success.
	t, err := time.Parse(
		time.RFC3339Nano,
		unsafe.String(unsafe.SliceData(text), len(text)),
	)
	if err != nil {
		return 0, errors.Wrapf(
			validate.ErrValidation, "invalid ISO 8601 timestamp %q", text,
		)
	}
	// UnixNano is undefined outside the years 1677 to 2262.
	if t.Before(time.Unix(0, math.MinInt64)) || t.After(time.Unix(0, math.MaxInt64)) {
		return 0, errors.Wrapf(
			validate.ErrValidation, "%q is out of range for a timestamp", text,
		)
	}
	return telem.NewTimeStamp(t), nil
}

// parseUnix parses text, decimal number text, as a count of format's units. Scaling the
// text instead of a float64 keeps every digit down to the nanosecond.
func parseUnix(text []byte, format TimeFormat) (telem.TimeStamp, error) {
	d, ok := parseDecimal(text, format.nanosExponent())
	if !ok {
		return 0, invalidNumber(text)
	}
	limit := uint64(math.MaxInt64)
	if d.negative {
		limit++
	}
	if d.overflow || d.magnitude > limit {
		return 0, errors.Wrapf(
			validate.ErrValidation,
			"%s is out of range for a %s timestamp",
			text,
			format,
		)
	}
	if d.negative {
		return telem.TimeStamp(-d.magnitude), nil
	}
	return telem.TimeStamp(d.magnitude), nil
}
