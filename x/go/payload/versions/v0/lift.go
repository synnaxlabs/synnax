// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"math"
	"reflect"

	xjson "github.com/synnaxlabs/x/json"
)

// LiftStaticWriteField returns the write field at base that places value, the value of
// a static write field of a released Core. The kind comes from the stored Go type, not
// the JSON type, because the Driver sent the value as stored. A whole float64 becomes
// integer within int64 range, or unsigned within uint64 range, so it keeps its digits.
// Another number becomes float. A string or boolean keeps its kind, and any other
// value, null included, becomes JSON text.
func LiftStaticWriteField(base BaseWriteField, value any) (WriteField, error) {
	var v WriteFieldVariant
	switch x := value.(type) {
	case string:
		v = StringWriteField{BaseWriteField: base, Value: x}
	case bool:
		v = BooleanWriteField{BaseWriteField: base, Value: x}
	case int, int8, int16, int32, int64:
		v = IntegerWriteField{
			BaseWriteField: base,
			Value:          Integer(reflect.ValueOf(x).Int()),
		}
	case uint, uint8, uint16, uint32, uint64:
		v = UnsignedWriteField{
			BaseWriteField: base,
			Value:          Unsigned(reflect.ValueOf(x).Uint()),
		}
	case float32:
		return LiftStaticWriteField(base, float64(x))
	case float64:
		switch {
		case x != math.Trunc(x) || math.IsInf(x, 0):
			v = FloatWriteField{BaseWriteField: base, Value: Float(x)}
		case x >= -(1<<63) && x < 1<<63:
			v = IntegerWriteField{BaseWriteField: base, Value: Integer(x)}
		case x >= 0 && x < 1<<64:
			v = UnsignedWriteField{BaseWriteField: base, Value: Unsigned(x)}
		default:
			v = FloatWriteField{BaseWriteField: base, Value: Float(x)}
		}
	default:
		text, err := xjson.Marshal(x)
		if err != nil {
			return WriteField{}, err
		}
		v = JsonWriteField{BaseWriteField: base, Value: string(text)}
	}
	return WriteField{Variant: v}, nil
}

// LiftGeneratedWriteField returns the write field at base that generates the value of a
// generated write field of a released Core. An absent time format is ISO 8601. ok is
// false for an unknown generator, which no Driver ran.
func LiftGeneratedWriteField(
	base BaseWriteField,
	generator GeneratorType,
	format *TimeFormat,
) (f WriteField, ok bool) {
	switch generator {
	case GeneratorTypeUUID:
		return WriteField{Variant: UuidWriteField{BaseWriteField: base}}, true
	case GeneratorTypeTimestamp:
		t := TimestampWriteField{BaseWriteField: base, TimeFormat: TimeFormatISO8601}
		if format != nil {
			t.TimeFormat = *format
		}
		return WriteField{Variant: t}, true
	default:
		return WriteField{}, false
	}
}
