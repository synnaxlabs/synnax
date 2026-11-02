// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kafka

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"uuid"
)

// preciseNumbers decodes a JSON integer into an int64 or uint64, so large integers
// and nanosecond timestamps survive the trip through an any.
var preciseNumbers = json.WithUnmarshalers(json.UnmarshalFromFunc(
	func(dec *jsontext.Decoder, v *any) error {
		if dec.PeekKind() != '0' {
			return errors.ErrUnsupported
		}
		raw, err := dec.ReadValue()
		if err != nil {
			return err
		}
		*v, err = parseNumber(string(raw))
		return err
	},
))

// parseNumber converts the text of a JSON number into an int64 or uint64 when it is an
// integer that fits, and a float64 otherwise.
func parseNumber(text string) (any, error) {
	if !strings.ContainsAny(text, ".eE") {
		if i, err := strconv.ParseInt(text, 10, 64); err == nil {
			return i, nil
		}
		if u, err := strconv.ParseUint(text, 10, 64); err == nil {
			return u, nil
		}
	}
	f, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return nil, errors.Wrapf(validate.ErrValidation, "invalid number %q", text)
	}
	return f, nil
}

// timeUnit returns the span one unit of a numeric time format counts.
func timeUnit(format http.TimeFormat) (telem.TimeSpan, error) {
	switch format {
	case http.TimeFormatUnixSec:
		return telem.Second, nil
	case http.TimeFormatUnixMs:
		return telem.Millisecond, nil
	case http.TimeFormatUnixUs:
		return telem.Microsecond, nil
	case http.TimeFormatUnixNs:
		return telem.Nanosecond, nil
	default:
		return 0, errors.Wrapf(validate.ErrValidation, "unknown time format %q", format)
	}
}

// parseTimestamp converts a decoded JSON value encoded with format into a timestamp.
func parseTimestamp(raw any, format http.TimeFormat) (telem.TimeStamp, error) {
	if format == http.TimeFormatISO8601 {
		s, ok := raw.(string)
		if !ok {
			return 0, errors.Wrapf(
				validate.ErrValidation, "expected an ISO 8601 string, got %T", raw,
			)
		}
		t, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			return 0, errors.Wrapf(
				validate.ErrValidation, "invalid ISO 8601 timestamp %q", s,
			)
		}
		return telem.NewTimeStamp(t), nil
	}
	unit, err := timeUnit(format)
	if err != nil {
		return 0, err
	}
	switch n := raw.(type) {
	case int64:
		if n > math.MaxInt64/int64(unit) || n < math.MinInt64/int64(unit) {
			return 0, timestampOutOfRange(n, format)
		}
		return telem.TimeStamp(n) * telem.TimeStamp(unit), nil
	case uint64:
		if n > math.MaxInt64/uint64(unit) {
			return 0, timestampOutOfRange(n, format)
		}
		return telem.TimeStamp(n) * telem.TimeStamp(unit), nil
	case float64:
		// math.MaxInt64 rounds up to 2^63 as a float64, so the bound is exclusive.
		ns := n * float64(unit)
		if math.IsNaN(ns) || ns < math.MinInt64 || ns >= math.MaxInt64 {
			return 0, timestampOutOfRange(n, format)
		}
		return telem.TimeStamp(ns), nil
	default:
		return 0, errors.Wrapf(validate.ErrValidation, "expected a number, got %T", raw)
	}
}

func timestampOutOfRange(n any, format http.TimeFormat) error {
	return errors.Wrapf(
		validate.ErrValidation, "%v is out of range for a %s timestamp", n, format,
	)
}

// formatTimestamp encodes ts as a JSON value in format. Nanoseconds stay integral so
// no precision is lost on the round trip.
func formatTimestamp(ts telem.TimeStamp, format http.TimeFormat) (any, error) {
	if format == http.TimeFormatISO8601 {
		return ts.Time().UTC().Format(time.RFC3339Nano), nil
	}
	unit, err := timeUnit(format)
	if err != nil {
		return nil, err
	}
	if unit == telem.Nanosecond {
		return int64(ts), nil
	}
	return float64(ts) / float64(unit), nil
}

// readSample converts raw, a decoded JSON value, into a sample of dt. enums map string
// labels onto the numeric or boolean value stored, and timeFormat decodes timestamp
// channels. It returns an error wrapping validate.ErrValidation when raw does not
// convert.
func readSample(
	raw any,
	dt telem.DataType,
	timeFormat *http.TimeFormat,
	enums []http.EnumEntry,
) (any, error) {
	if dt == telem.TimestampT {
		if timeFormat == nil {
			return nil, errors.Wrap(
				validate.ErrValidation, "time_format: required for timestamp channels",
			)
		}
		return parseTimestamp(raw, *timeFormat)
	}
	switch dt {
	case telem.StringT:
		s, ok := raw.(string)
		if !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation, "expected a string, got %T", raw,
			)
		}
		return s, nil
	case telem.JSONT:
		return raw, nil
	case telem.UUIDT:
		s, ok := raw.(string)
		if !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation, "expected a UUID string, got %T", raw,
			)
		}
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, errors.Wrapf(validate.ErrValidation, "invalid UUID %q", s)
		}
		return id, nil
	case telem.BytesT:
		return nil, errors.Wrapf(validate.ErrValidation, "unsupported data type %s", dt)
	}
	switch v := raw.(type) {
	case bool:
		return v, nil
	case float64, int64, uint64:
		if err := checkFits(v, dt); err != nil {
			return nil, err
		}
		return v, nil
	case string:
		for _, e := range enums {
			if e.Label == v {
				if err := checkFits(e.Value, dt); err != nil {
					return nil, err
				}
				return e.Value, nil
			}
		}
		return nil, errors.Wrapf(validate.ErrValidation, "unknown enum label %q", v)
	default:
		return nil, errors.Wrapf(
			validate.ErrValidation, "expected a number or boolean, got %T", raw,
		)
	}
}

// integerBounds returns the range of the integer data type dt. ok is false when dt
// is not an integer type.
func integerBounds(dt telem.DataType) (lo int64, hi uint64, ok bool) {
	switch dt {
	case telem.Int8T:
		return math.MinInt8, math.MaxInt8, true
	case telem.Int16T:
		return math.MinInt16, math.MaxInt16, true
	case telem.Int32T:
		return math.MinInt32, math.MaxInt32, true
	case telem.Int64T:
		return math.MinInt64, math.MaxInt64, true
	case telem.Uint8T:
		return 0, math.MaxUint8, true
	case telem.Uint16T:
		return 0, math.MaxUint16, true
	case telem.Uint32T:
		return 0, math.MaxUint32, true
	case telem.Uint64T:
		return 0, math.MaxUint64, true
	default:
		return 0, 0, false
	}
}

// checkFits returns an error wrapping validate.ErrValidation when n, an int64,
// uint64, or float64, does not convert into dt without wrapping, truncating, or
// overflowing to infinity.
func checkFits(n any, dt telem.DataType) error {
	if f, ok := n.(float64); ok && dt == telem.Float32T &&
		math.Abs(f) > math.MaxFloat32 {
		return errors.Wrapf(validate.ErrValidation, "%v overflows %s", f, dt)
	}
	lo, hi, ok := integerBounds(dt)
	if !ok {
		return nil
	}
	fits := true
	switch v := n.(type) {
	case int64:
		fits = v >= lo && (v < 0 || uint64(v) <= hi)
	case uint64:
		fits = v <= hi
	case float64:
		if v != math.Trunc(v) {
			return errors.Wrapf(validate.ErrValidation, "%v is not an integer", v)
		}
		// float64(hi)+1 rounds to the first value past hi for every integer type.
		fits = v >= float64(lo) && v < float64(hi)+1
	}
	if !fits {
		return errors.Wrapf(validate.ErrValidation, "%v is out of range for %s", n, dt)
	}
	return nil
}

// sampleNumber returns sample i of a fixed-density series as a JSON number. Integer
// types stay int64 or uint64 so no precision is lost; the rest become float64.
func sampleNumber(s telem.Series, i int) (any, error) {
	switch s.DataType {
	case telem.Float64T:
		return s.ValueAt[float64](i), nil
	case telem.Float32T:
		return float64(s.ValueAt[float32](i)), nil
	case telem.Int64T:
		return s.ValueAt[int64](i), nil
	case telem.Int32T:
		return int64(s.ValueAt[int32](i)), nil
	case telem.Int16T:
		return int64(s.ValueAt[int16](i)), nil
	case telem.Int8T:
		return int64(s.ValueAt[int8](i)), nil
	case telem.Uint64T:
		return s.ValueAt[uint64](i), nil
	case telem.Uint32T:
		return uint64(s.ValueAt[uint32](i)), nil
	case telem.Uint16T:
		return uint64(s.ValueAt[uint16](i)), nil
	case telem.Uint8T:
		return uint64(s.ValueAt[uint8](i)), nil
	case telem.BooleanT:
		if s.ValueAt[bool](i) {
			return int64(1), nil
		}
		return int64(0), nil
	case telem.TimestampT:
		return int64(s.ValueAt[telem.TimeStamp](i)), nil
	default:
		return nil, errors.Wrapf(
			validate.ErrValidation, "data type %s is not numeric", s.DataType,
		)
	}
}

// numberFloat converts a value sampleNumber returned into a float64.
func numberFloat(n any) float64 {
	switch v := n.(type) {
	case int64:
		return float64(v)
	case uint64:
		return float64(v)
	default:
		return v.(float64)
	}
}

// formatNumber formats a value sampleNumber returned as a decimal string.
func formatNumber(n any) string {
	switch v := n.(type) {
	case int64:
		return strconv.FormatInt(v, 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	default:
		return strconv.FormatFloat(v.(float64), 'g', -1, 64)
	}
}

// writeSample converts sample i of s into the JSON value ch produces.
func writeSample(s telem.Series, i int, ch WriteChannel) (any, error) {
	if s.DataType == telem.TimestampT {
		if ch.TimeFormat == nil {
			return nil, errors.Wrap(
				validate.ErrValidation, "time_format: required for timestamp channels",
			)
		}
		return formatTimestamp(s.ValueAt[telem.TimeStamp](i), *ch.TimeFormat)
	}
	if s.DataType.IsVariable() || s.DataType == telem.UUIDT {
		if ch.JSONType != http.JSONTypeString {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"data type %s only serializes as a string",
				s.DataType,
			)
		}
		if s.DataType == telem.UUIDT {
			return uuid.UUID(s.At(i)).String(), nil
		}
		return string(s.At(i)), nil
	}
	n, err := sampleNumber(s, i)
	if err != nil {
		return nil, err
	}
	switch ch.JSONType {
	case http.JSONTypeNumber:
		return n, nil
	case http.JSONTypeBoolean:
		return numberFloat(n) != 0, nil
	case http.JSONTypeString:
		if len(ch.EnumValues) == 0 {
			return formatNumber(n), nil
		}
		f := numberFloat(n)
		for _, e := range ch.EnumValues {
			if e.Value == f {
				return e.Label, nil
			}
		}
		return nil, errors.Wrapf(
			validate.ErrValidation, "no enum label for value %v", n,
		)
	default:
		return nil, errors.Wrapf(
			validate.ErrValidation, "unknown json_type %q", ch.JSONType,
		)
	}
}

// extraFieldValue returns the pointer and value of a static or generated record
// field. ts stamps a timestamp generator.
func extraFieldValue(
	field http.WriteField,
	ts telem.TimeStamp,
) (pointer string, value any, err error) {
	switch v := field.Variant.(type) {
	case http.StaticWriteField:
		return v.Pointer, v.Value, nil
	case http.GeneratedWriteField:
		switch v.Generator {
		case http.GeneratorTypeUUID:
			return v.Pointer, uuid.New().String(), nil
		case http.GeneratorTypeTimestamp:
			format := http.TimeFormatUnixNs
			if v.TimeFormat != nil {
				format = *v.TimeFormat
			}
			value, err = formatTimestamp(ts, format)
			if err != nil {
				return "", nil, err
			}
			return v.Pointer, value, nil
		default:
			return "", nil, errors.Wrapf(
				validate.ErrValidation, "unknown generator %q", v.Generator,
			)
		}
	default:
		return "", nil, errors.Newf("unknown write field variant %T", v)
	}
}
