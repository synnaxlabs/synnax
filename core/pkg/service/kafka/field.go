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
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// entry is one enum entry of a read field, of any kind.
type entry struct {
	label string
	// value is a float64, int64, uint64, string, or bool.
	value any
}

// options are the kind-specific options of a read field, of any kind.
type options struct {
	entries []entry
	// fill is a float64, int64, uint64, string, bool, or telem.TimeStamp. Nil when the
	// field has no fill.
	fill       any
	timeFormat payload.TimeFormat
	// fillable is true when the field's kind takes a fill. An absent value then writes
	// the fill, or skips the record when there is none.
	fillable bool
}

func entriesOf[E any](in []E, fn func(E) entry) []entry {
	out := make([]entry, len(in))
	for i, e := range in {
		out[i] = fn(e)
	}
	return out
}

func fillOf[T, U any](v *T, fn func(T) U) any {
	if v == nil {
		return nil
	}
	return fn(*v)
}

// optionsOf returns the options of fld.
func optionsOf(fld ReadField) options {
	switch v := fld.Variant.(type) {
	case FloatReadField:
		return options{
			entries: entriesOf(v.EnumValues, func(e payload.FloatEntry) entry {
				return entry{e.Label, float64(e.Value)}
			}),
			fill: fillOf(
				v.Fill,
				func(f payload.Float) float64 { return float64(f) },
			),
			fillable: true,
		}
	case IntegerReadField:
		return options{
			entries: entriesOf(v.EnumValues, func(e payload.IntegerEntry) entry {
				return entry{e.Label, int64(e.Value)}
			}),
			fill:     fillOf(v.Fill, func(i payload.Integer) int64 { return int64(i) }),
			fillable: true,
		}
	case UnsignedReadField:
		return options{
			entries: entriesOf(v.EnumValues, func(e payload.UnsignedEntry) entry {
				return entry{e.Label, uint64(e.Value)}
			}),
			fill: fillOf(
				v.Fill,
				func(u payload.Unsigned) uint64 { return uint64(u) },
			),
			fillable: true,
		}
	case StringReadField:
		return options{
			entries: entriesOf(v.EnumValues, func(e payload.StringEntry) entry {
				return entry{e.Label, e.Value}
			}),
			fill:     fillOf(v.Fill, func(s string) string { return s }),
			fillable: true,
		}
	case BooleanReadField:
		return options{
			entries: entriesOf(v.EnumValues, func(e payload.BooleanEntry) entry {
				return entry{e.Label, e.Value}
			}),
			fill:     fillOf(v.Fill, func(b bool) bool { return b }),
			fillable: true,
		}
	case TimestampReadField:
		return options{
			timeFormat: v.TimeFormat,
			fill: fillOf(v.Fill, func(ts telem.TimeStamp) telem.TimeStamp {
				return ts
			}),
			fillable: true,
		}
	}
	return options{}
}

// convert returns v, a value of an entry or fill, as a sample of dt.
func convert(v any, dt telem.DataType) ([]byte, error) {
	switch v := v.(type) {
	case float64:
		return xjson.Convert(v, dt)
	case int64:
		return xjson.Convert(v, dt)
	case uint64:
		return xjson.Convert(v, dt)
	case string:
		return xjson.Convert(v, dt)
	case bool:
		return xjson.Convert(v, dt)
	case telem.TimeStamp:
		return xjson.Convert(v, dt)
	}
	panic(errors.Newf("kafka: entry or fill of unknown type %T", v))
}

func newEnum(e entry) xjson.Enum {
	switch v := e.value.(type) {
	case float64:
		return xjson.NewEnum(e.label, v)
	case int64:
		return xjson.NewEnum(e.label, v)
	case uint64:
		return xjson.NewEnum(e.label, v)
	case string:
		return xjson.NewEnum(e.label, v)
	case bool:
		return xjson.NewEnum(e.label, v)
	}
	panic(errors.Newf("kafka: entry of unknown type %T", e.value))
}

// newColumn returns the column of fld, which writes to ch. It returns an error wrapping
// validate.ErrValidation when the kind of fld does not match ch, or an entry or the
// fill does not fit ch.
func newColumn(fld ReadField, ch channel.Channel) (column, error) {
	if kind, _ := payload.KindOf(ch.DataType); fld.Type() != kind {
		return column{}, errors.Wrapf(
			validate.ErrValidation,
			"field of kind %s does not match channel %s (%s), which takes kind %s",
			fld.Type(), ch.Name, ch.DataType, kind,
		)
	}
	opts := optionsOf(fld)
	f := xjson.Field{DataType: ch.DataType}
	emptyLabeled := false
	for _, e := range opts.entries {
		f.Enums = append(f.Enums, newEnum(e))
		emptyLabeled = emptyLabeled || e.label == ""
	}
	if opts.timeFormat != "" {
		var err error
		if f.TimeFormat, err = xjson.ParseTimeFormat(
			string(opts.timeFormat),
		); err != nil {
			return column{}, err
		}
	}
	dec, err := xjson.NewDecoder(f)
	if err != nil {
		return column{}, err
	}
	col := newDecoderColumn(dec, ch.DataType)
	col.fillable = opts.fillable
	col.emptyLabeled = emptyLabeled
	if opts.fill == nil {
		return col, nil
	}
	if ch.IsIndex {
		return column{}, errors.Wrapf(
			validate.ErrValidation, "index channel %s takes no fill", ch.Name,
		)
	}
	if col.fill, err = convert(opts.fill, ch.DataType); err != nil {
		return column{}, errors.Wrap(err, "fill")
	}
	col.filled = true
	return col, nil
}
