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
	"fmt"
	"math"
	"strconv"
	"time"
	"unsafe"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// timeFormat is a payload.TimeFormat in the form that a conversion reads for each
// sample.
type timeFormat uint8

const (
	iso8601 timeFormat = iota
	unixSecond
	unixMillisecond
	unixMicrosecond
	unixNanosecond
)

var payloadTimeFormats = [...]payload.TimeFormat{
	iso8601:         payload.TimeFormatISO8601,
	unixSecond:      payload.TimeFormatUnixSec,
	unixMillisecond: payload.TimeFormatUnixMs,
	unixMicrosecond: payload.TimeFormatUnixUs,
	unixNanosecond:  payload.TimeFormatUnixNs,
}

// timeFormatOf returns the timeFormat of f. It returns an error wrapping
// validate.ErrValidation when f is not a payload.TimeFormat.
func timeFormatOf(f payload.TimeFormat) (timeFormat, error) {
	for t, p := range payloadTimeFormats {
		if f == p {
			return timeFormat(t), nil
		}
	}
	return iso8601, errors.Wrapf(
		validate.ErrValidation,
		`unknown time format %q: expected "iso8601", "unix_sec", "unix_ms", `+
			`"unix_us", or "unix_ns"`,
		f,
	)
}

// String returns the payload.TimeFormat of f.
func (f timeFormat) String() string { return string(payloadTimeFormats[f]) }

// timeFormatOfField returns the timeFormat of f when it holds timestamps. Any other f
// needs none.
func timeFormatOfField(f Field) (timeFormat, error) {
	if f.DataType != telem.TimestampT {
		return iso8601, nil
	}
	return timeFormatOf(f.TimeFormat)
}

// nanosExponent returns the base 10 logarithm of the nanoseconds in one unit of a
// numeric format.
func (f timeFormat) nanosExponent() int {
	switch f {
	case unixSecond:
		return 9
	case unixMillisecond:
		return 6
	case unixMicrosecond:
		return 3
	case unixNanosecond:
		return 0
	case iso8601:
	}
	panic(fmt.Sprintf("json: %s is not a numeric time format", f))
}

func (f timeFormat) nanosPerUnit() int64 {
	n := int64(1)
	for range f.nanosExponent() {
		n *= 10
	}
	return n
}

// appendTimeStamp appends ts as a JSON value in format: a string for iso8601, a whole
// number rounded toward negative infinity for unixMillisecond and unixMicrosecond, and
// a number with every digit of ts otherwise.
func appendTimeStamp(dst []byte, ts telem.TimeStamp, format timeFormat) []byte {
	if format == iso8601 {
		dst = append(dst, '"')
		dst = ts.Time().UTC().AppendFormat(dst, time.RFC3339Nano)
		return append(dst, '"')
	}
	unit := format.nanosPerUnit()
	whole, frac := int64(ts)/unit, int64(ts)%unit
	if format != unixSecond || frac == 0 {
		// Division truncates toward zero, so a negative remainder takes one unit off.
		if frac < 0 {
			whole--
		}
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

// parseUnix parses text, number text or the content of a JSON string, as a count of
// format's units. It takes what parseQuoted takes. Scaling decimal text instead of a
// float64 keeps every digit down to the nanosecond.
func parseUnix(text []byte, format timeFormat) (telem.TimeStamp, error) {
	text = bytes.Trim(text, quotedSpace)
	var (
		nanos int64
		fits  bool
	)
	if d, ok := parseDecimal(text, format.nanosExponent()); ok {
		limit := uint64(math.MaxInt64)
		if d.negative {
			limit++
		}
		fits = !d.overflow && d.magnitude <= limit
		nanos = int64(d.magnitude)
		if d.negative {
			nanos = int64(-d.magnitude)
		}
	} else {
		n, err := parseQuoted(text)
		if err != nil {
			return 0, err
		}
		nanos, fits = n.int64Times(format.nanosPerUnit())
	}
	if !fits {
		return 0, errors.Wrapf(
			validate.ErrValidation,
			"%s is out of range for a %s timestamp",
			text,
			format,
		)
	}
	return telem.TimeStamp(nanos), nil
}
