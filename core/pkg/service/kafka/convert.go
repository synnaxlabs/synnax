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
	"strconv"
	"time"
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/service/http"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

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
	f, ok := raw.(float64)
	if !ok {
		return 0, errors.Wrapf(validate.ErrValidation, "expected a number, got %T", raw)
	}
	unit, err := timeUnit(format)
	if err != nil {
		return 0, err
	}
	return telem.TimeStamp(f * float64(unit)), nil
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
	case telem.UUIDT, telem.BytesT:
		return nil, errors.Wrapf(validate.ErrValidation, "unsupported data type %s", dt)
	}
	switch v := raw.(type) {
	case float64, bool:
		return v, nil
	case string:
		for _, e := range enums {
			if e.Label == v {
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

// numericAt returns sample i of a fixed-density series as a float64.
func numericAt(s telem.Series, i int) (float64, error) {
	switch s.DataType {
	case telem.Float64T:
		return s.ValueAt[float64](i), nil
	case telem.Float32T:
		return float64(s.ValueAt[float32](i)), nil
	case telem.Int64T:
		return float64(s.ValueAt[int64](i)), nil
	case telem.Int32T:
		return float64(s.ValueAt[int32](i)), nil
	case telem.Int16T:
		return float64(s.ValueAt[int16](i)), nil
	case telem.Int8T:
		return float64(s.ValueAt[int8](i)), nil
	case telem.Uint64T:
		return float64(s.ValueAt[uint64](i)), nil
	case telem.Uint32T:
		return float64(s.ValueAt[uint32](i)), nil
	case telem.Uint16T:
		return float64(s.ValueAt[uint16](i)), nil
	case telem.Uint8T:
		return float64(s.ValueAt[uint8](i)), nil
	case telem.BooleanT:
		if s.ValueAt[bool](i) {
			return 1, nil
		}
		return 0, nil
	case telem.TimestampT:
		return float64(s.ValueAt[telem.TimeStamp](i)), nil
	default:
		return 0, errors.Wrapf(
			validate.ErrValidation, "data type %s is not numeric", s.DataType,
		)
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
	if s.DataType.IsVariable() {
		if ch.JSONType != http.JSONTypeString {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"data type %s only serializes as a string",
				s.DataType,
			)
		}
		return string(s.At(i)), nil
	}
	f, err := numericAt(s, i)
	if err != nil {
		return nil, err
	}
	switch ch.JSONType {
	case http.JSONTypeNumber:
		return f, nil
	case http.JSONTypeBoolean:
		return f != 0, nil
	case http.JSONTypeString:
		if len(ch.EnumValues) == 0 {
			return strconv.FormatFloat(f, 'g', -1, 64), nil
		}
		for _, e := range ch.EnumValues {
			if e.Value == f {
				return e.Label, nil
			}
		}
		return nil, errors.Wrapf(
			validate.ErrValidation, "no enum label for value %v", f,
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
			return v.Pointer, value, err
		default:
			return "", nil, errors.Wrapf(
				validate.ErrValidation, "unknown generator %q", v.Generator,
			)
		}
	default:
		return "", nil, errors.Newf("unknown write field variant %T", v)
	}
}
