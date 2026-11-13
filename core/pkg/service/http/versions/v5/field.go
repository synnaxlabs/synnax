// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5

import (
	"math"

	"github.com/samber/lo"
	xpayload "github.com/synnaxlabs/x/payload"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// entry is an enum entry whose value is a payload.Float, payload.Integer,
// payload.Unsigned, string, or bool.
type entry struct {
	label string
	value any
}

// options holds the variant fields of a read field, whatever its kind. fill is nil or
// a value as entry holds one, or a telem.TimeStamp.
type options struct {
	dataType   *telem.DataType
	entries    []entry
	fill       any
	timeFormat payload.TimeFormat
}

func numeric(k payload.Kind) bool {
	return k == payload.KindFloat || k == payload.KindInteger ||
		k == payload.KindUnsigned
}

func withField(warn Warn, key string) Warn {
	return func(msg string, fields ...zap.Field) {
		warn(msg, append(fields, zap.String("field", key))...)
	}
}

func optionsOf(f ReadField) options {
	switch v := f.Variant.(type) {
	case FloatReadField:
		return options{
			dataType: v.DataType,
			entries: entriesOf(
				v.EnumValues,
				func(e payload.FloatEntry) entry { return entry{e.Label, e.Value} },
			),
			fill: fillOf(v.Fill),
		}
	case IntegerReadField:
		return options{
			dataType: v.DataType,
			entries: entriesOf(
				v.EnumValues,
				func(e payload.IntegerEntry) entry { return entry{e.Label, e.Value} },
			),
			fill: fillOf(v.Fill),
		}
	case UnsignedReadField:
		return options{
			dataType: v.DataType,
			entries: entriesOf(
				v.EnumValues,
				func(e payload.UnsignedEntry) entry { return entry{e.Label, e.Value} },
			),
			fill: fillOf(v.Fill),
		}
	case StringReadField:
		return options{
			entries: entriesOf(
				v.EnumValues,
				func(e payload.StringEntry) entry { return entry{e.Label, e.Value} },
			),
			fill: fillOf(v.Fill),
		}
	case BooleanReadField:
		return options{
			entries: entriesOf(
				v.EnumValues,
				func(e payload.BooleanEntry) entry { return entry{e.Label, e.Value} },
			),
			fill: fillOf(v.Fill),
		}
	case TimestampReadField:
		return options{timeFormat: v.TimeFormat, fill: fillOf(v.Fill)}
	}
	return options{}
}

func entriesOf[E any](es []E, f func(E) entry) []entry {
	return lo.Map(es, func(e E, _ int) entry { return f(e) })
}

func fillOf[T any](fill *T) any {
	if fill == nil {
		return nil
	}
	return *fill
}

// convert returns v as a value of kind k. ok is false when k cannot hold v, or when dt
// is not nil and a channel of data type dt cannot hold it.
func convert(v any, k payload.Kind, dt *telem.DataType) (any, bool) {
	var (
		out any
		ok  bool
	)
	switch v := v.(type) {
	case payload.Float:
		out, ok = convertNumber(v, k)
	case payload.Integer:
		out, ok = convertNumber(v, k)
	case payload.Unsigned:
		out, ok = convertNumber(v, k)
	case string:
		out, ok = v, k == payload.KindString
	case bool:
		out, ok = v, k == payload.KindBoolean
	case telem.TimeStamp:
		out, ok = v, k == payload.KindTimestamp
	}
	if !ok || dt == nil {
		return out, ok
	}
	switch n := out.(type) {
	case payload.Float:
		return n, xpayload.Fits(n, *dt)
	case payload.Integer:
		return n, xpayload.Fits(n, *dt)
	case payload.Unsigned:
		return n, xpayload.Fits(n, *dt)
	}
	return out, true
}

func convertNumber[From xpayload.Number](v From, k payload.Kind) (any, bool) {
	switch k {
	case payload.KindFloat:
		return xpayload.Convert[payload.Float](v)
	case payload.KindInteger:
		return xpayload.Convert[payload.Integer](v)
	case payload.KindUnsigned:
		return xpayload.Convert[payload.Unsigned](v)
	case payload.KindBoolean:
		if f, ok := any(v).(payload.Float); ok && math.IsNaN(float64(f)) {
			return nil, false
		}
		return v != 0, true
	}
	return nil, false
}

// withoutEmptyLabels drops each entry with an empty label, with a warning, when k is
// not string. Only a string field reads an empty response.
func withoutEmptyLabels(entries []entry, k payload.Kind, warn Warn) []entry {
	if k == payload.KindString {
		return entries
	}
	return lo.Filter(entries, func(e entry, _ int) bool {
		if e.label != "" {
			return true
		}
		warn("dropped an enum entry with an empty label", zap.Any("value", e.value))
		return false
	})
}

// assemble builds a read field of kind k from o. An entry or fill that k, or a channel
// of data type dt when dt is not nil, cannot hold drops with a warning.
func assemble(
	k payload.Kind,
	base BaseReadField,
	o options,
	dt *telem.DataType,
	warn Warn,
) ReadField {
	var entries []entry
	for _, e := range o.entries {
		v, ok := convert(e.value, k, dt)
		if !ok {
			warn(
				"dropped an enum entry that the field cannot hold",
				zap.String("label", e.label),
				zap.Any("value", e.value),
			)
			continue
		}
		entries = append(entries, entry{label: e.label, value: v})
	}
	fill, hasFill := o.fill, o.fill != nil
	if hasFill {
		var ok bool
		if fill, ok = convert(fill, k, dt); !ok {
			warn("dropped a fill that the field cannot hold", zap.Any("fill", o.fill))
			hasFill = false
		}
	}
	switch k {
	case payload.KindFloat:
		return ReadField{
			Variant: FloatReadField{
				BaseReadField: base,
				FloatRead: payload.FloatRead{
					DataType: o.dataType,
					EnumValues: typedEntries(
						entries,
						func(l string, v payload.Float) payload.FloatEntry {
							return payload.FloatEntry{Label: l, Value: v}
						},
					),
					Fill: typedFill[payload.Float](fill, hasFill),
				},
			},
		}
	case payload.KindInteger:
		return ReadField{
			Variant: IntegerReadField{
				BaseReadField: base,
				IntegerRead: payload.IntegerRead{
					DataType: o.dataType,
					EnumValues: typedEntries(
						entries,
						func(l string, v payload.Integer) payload.IntegerEntry {
							return payload.IntegerEntry{Label: l, Value: v}
						},
					),
					Fill: typedFill[payload.Integer](fill, hasFill),
				},
			},
		}
	case payload.KindUnsigned:
		return ReadField{
			Variant: UnsignedReadField{
				BaseReadField: base,
				UnsignedRead: payload.UnsignedRead{
					DataType: o.dataType,
					EnumValues: typedEntries(
						entries,
						func(l string, v payload.Unsigned) payload.UnsignedEntry {
							return payload.UnsignedEntry{Label: l, Value: v}
						},
					),
					Fill: typedFill[payload.Unsigned](fill, hasFill),
				},
			},
		}
	case payload.KindString:
		return ReadField{
			Variant: StringReadField{
				BaseReadField: base,
				StringRead: payload.StringRead{
					EnumValues: typedEntries(
						entries,
						func(l, v string) payload.StringEntry {
							return payload.StringEntry{Label: l, Value: v}
						},
					),
					Fill: typedFill[string](fill, hasFill),
				},
			},
		}
	case payload.KindBoolean:
		return ReadField{
			Variant: BooleanReadField{
				BaseReadField: base,
				BooleanRead: payload.BooleanRead{
					EnumValues: typedEntries(
						entries,
						func(l string, v bool) payload.BooleanEntry {
							return payload.BooleanEntry{Label: l, Value: v}
						},
					),
					Fill: typedFill[bool](fill, hasFill),
				},
			},
		}
	case payload.KindTimestamp:
		return ReadField{
			Variant: TimestampReadField{
				BaseReadField: base,
				TimestampRead: payload.TimestampRead{
					TimeFormat: o.timeFormat,
					Fill:       typedFill[telem.TimeStamp](fill, hasFill),
				},
			},
		}
	case payload.KindJSON:
		return ReadField{Variant: JsonReadField{BaseReadField: base}}
	case payload.KindUUID:
		return ReadField{Variant: UuidReadField{BaseReadField: base}}
	default:
		return ReadField{Variant: BytesReadField{BaseReadField: base}}
	}
}

func typedEntries[V, E any](entries []entry, f func(string, V) E) []E {
	out := make([]E, 0, len(entries))
	for _, e := range entries {
		out = append(out, f(e.label, e.value.(V)))
	}
	return out
}

func typedFill[V any](fill any, ok bool) *V {
	if !ok {
		return nil
	}
	return new(fill.(V))
}
