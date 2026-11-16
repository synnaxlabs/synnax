// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload

import (
	"encoding/json/jsontext"
	"fmt"

	"github.com/samber/lo"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// NumberEntry maps a payload label to a channel value of a numeric kind.
type NumberEntry struct {
	// Label is the string label in the payload.
	Label string
	// Value is the channel value the label maps to.
	Value Number
}

// NumberRead holds the options of a field that reads a float, an integer, or an
// unsigned integer.
type NumberRead struct {
	// DataType is the data type of a new channel. Nil for an existing one.
	DataType *telem.DataType
	// EnumValues maps payload labels to channel values.
	EnumValues []NumberEntry
	// Fill is written when the payload has no value. Nil requires one.
	Fill *Number
}

// Read is the kind and the options of a read field. Number holds the options of a
// numeric kind, and String, Boolean, and Timestamp those of their kinds. The options of
// other kinds are zero.
type Read struct {
	// Kind is the kind of the value that the field reads.
	Kind Kind
	// Number holds the options of a float, integer, or unsigned field.
	Number NumberRead
	// String holds the options of a string field.
	String StringRead
	// Boolean holds the options of a boolean field.
	Boolean BooleanRead
	// Timestamp holds the options of a timestamp field.
	Timestamp TimestampRead
}

func numberRead[E, V any](
	dataType *telem.DataType,
	entries []E,
	fill *V,
	entry func(E) NumberEntry,
	value func(V) Number,
) NumberRead {
	r := NumberRead{
		DataType:   dataType,
		EnumValues: lo.Map(entries, func(e E, _ int) NumberEntry { return entry(e) }),
	}
	if fill != nil {
		n := value(*fill)
		r.Fill = &n
	}
	return r
}

// FromFloat returns the Read of a float field.
func FromFloat(r FloatRead) Read {
	return Read{Kind: KindFloat, Number: numberRead(
		r.DataType, r.EnumValues, r.Fill,
		func(e FloatEntry) NumberEntry {
			return NumberEntry{Label: e.Label, Value: FloatNumber(e.Value)}
		},
		FloatNumber,
	)}
}

// FromInteger returns the Read of an integer field.
func FromInteger(r IntegerRead) Read {
	return Read{Kind: KindInteger, Number: numberRead(
		r.DataType, r.EnumValues, r.Fill,
		func(e IntegerEntry) NumberEntry {
			return NumberEntry{Label: e.Label, Value: IntegerNumber(e.Value)}
		},
		IntegerNumber,
	)}
}

// FromUnsigned returns the Read of an unsigned field.
func FromUnsigned(r UnsignedRead) Read {
	return Read{Kind: KindUnsigned, Number: numberRead(
		r.DataType, r.EnumValues, r.Fill,
		func(e UnsignedEntry) NumberEntry {
			return NumberEntry{Label: e.Label, Value: UnsignedNumber(e.Value)}
		},
		UnsignedNumber,
	)}
}

func fill[V any](n *Number, value func(Number) V) *V {
	if n == nil {
		return nil
	}
	v := value(*n)
	return &v
}

// Float returns the options of r, which must be of the float kind.
func (r Read) Float() FloatRead {
	return FloatRead{
		DataType: r.Number.DataType,
		EnumValues: lo.Map(r.Number.EnumValues, func(e NumberEntry, _ int) FloatEntry {
			return FloatEntry{Label: e.Label, Value: e.Value.Float()}
		}),
		Fill: fill(r.Number.Fill, Number.Float),
	}
}

// Integer returns the options of r, which must be of the integer kind.
func (r Read) Integer() IntegerRead {
	return IntegerRead{
		DataType: r.Number.DataType,
		EnumValues: lo.Map(
			r.Number.EnumValues,
			func(e NumberEntry, _ int) IntegerEntry {
				return IntegerEntry{Label: e.Label, Value: e.Value.Integer()}
			},
		),
		Fill: fill(r.Number.Fill, Number.Integer),
	}
}

// Unsigned returns the options of r, which must be of the unsigned kind.
func (r Read) Unsigned() UnsignedRead {
	return UnsignedRead{
		DataType: r.Number.DataType,
		EnumValues: lo.Map(
			r.Number.EnumValues,
			func(e NumberEntry, _ int) UnsignedEntry {
				return UnsignedEntry{Label: e.Label, Value: e.Value.Unsigned()}
			},
		),
		Fill: fill(r.Number.Fill, Number.Unsigned),
	}
}

func isNumeric(k Kind) bool {
	return k == KindFloat || k == KindInteger || k == KindUnsigned
}

func (r Read) labels() []string {
	switch {
	case isNumeric(r.Kind):
		return lo.Map(r.Number.EnumValues, func(e NumberEntry, _ int) string {
			return e.Label
		})
	case r.Kind == KindString:
		return lo.Map(r.String.EnumValues, func(e StringEntry, _ int) string {
			return e.Label
		})
	case r.Kind == KindBoolean:
		return lo.Map(r.Boolean.EnumValues, func(e BooleanEntry, _ int) string {
			return e.Label
		})
	}
	return nil
}

func (r Read) hasFill() bool {
	switch {
	case isNumeric(r.Kind):
		return r.Number.Fill != nil
	case r.Kind == KindString:
		return r.String.Fill != nil
	case r.Kind == KindBoolean:
		return r.Boolean.Fill != nil
	case r.Kind == KindTimestamp:
		return r.Timestamp.Fill != nil
	}
	return false
}

// Validate returns an error wrapping validate.ErrValidation when two enum entries of r
// share a label, or when a label is empty on a kind other than string. Such a label
// never matches, because an empty string is no value on those kinds.
func (r Read) Validate() error {
	seen := set.New[string]()
	for _, label := range r.labels() {
		if label == "" && r.Kind != KindString {
			return errors.Wrapf(
				validate.ErrValidation,
				"an enum label cannot be empty on a %s field",
				r.Kind,
			)
		}
		if seen.Contains(label) {
			return errors.Wrapf(
				validate.ErrValidation, "enum label %q is repeated", label,
			)
		}
		seen.Add(label)
	}
	return nil
}

// Check returns an error wrapping validate.ErrValidation when r does not agree with a
// channel of data type dt: its kind is not the one dt takes, an entry or the fill does
// not fit dt, or the channel is an index and r has a fill.
func (r Read) Check(dt telem.DataType, index bool) error {
	if k, ok := KindOf(dt); !ok || k != r.Kind {
		return errors.Wrapf(
			validate.ErrValidation,
			"a field of kind %s cannot read into a channel of data type %s",
			r.Kind,
			dt,
		)
	}
	if index && r.hasFill() {
		return errors.Wrap(
			validate.ErrValidation, "an index channel cannot take a fill",
		)
	}
	if !isNumeric(r.Kind) {
		return nil
	}
	for _, e := range r.Number.EnumValues {
		if !e.Value.Fits(dt) {
			return errors.Wrapf(
				validate.ErrValidation,
				"enum label %q: %s does not fit %s", e.Label, e.Value, dt,
			)
		}
	}
	if f := r.Number.Fill; f != nil && !f.Fits(dt) {
		return errors.Wrapf(
			validate.ErrValidation, "fill: %s does not fit %s", f, dt,
		)
	}
	return nil
}

// Reconcile returns r changed to agree with a channel of data type dt, and a
// description of each change. It sets the kind that dt takes and clears the data type.
// Between numeric kinds, it keeps each entry and the fill that the new kind holds
// exactly and that fits dt. From a numeric kind to boolean, each entry and the fill
// become true when they are not 0, and a NaN drops. It drops them on other changes of
// kind, the fill of an index channel, each entry after the first with a label, and an
// empty label on a kind other than string. Reconcile returns r unchanged when dt is not
// a known data type.
func (r Read) Reconcile(dt telem.DataType, index bool) (Read, []string) {
	k, ok := KindOf(dt)
	if !ok {
		return r, nil
	}
	var (
		out     = Read{Kind: k}
		changes []string
		note    = func(format string, args ...any) {
			changes = append(changes, fmt.Sprintf(format, args...))
		}
	)
	if r.Kind != k {
		note("kind %s became %s", r.Kind, k)
	}
	if r.Number.DataType != nil {
		note("data type %s was cleared", *r.Number.DataType)
	}
	seen := set.New[string]()
	unique := func(label string) bool {
		if label == "" && k != KindString {
			note("an empty enum label was dropped")
			return false
		}
		if seen.Contains(label) {
			note("enum label %q was dropped as a repeat", label)
			return false
		}
		seen.Add(label)
		return true
	}
	switch {
	case isNumeric(k) && isNumeric(r.Kind):
		for _, e := range r.Number.EnumValues {
			v, ok := e.Value.To(k)
			if !ok || !v.Fits(dt) {
				note(
					"enum label %q was dropped: %s does not fit %s",
					e.Label,
					e.Value,
					dt,
				)
				continue
			}
			if unique(e.Label) {
				out.Number.EnumValues = append(
					out.Number.EnumValues, NumberEntry{Label: e.Label, Value: v},
				)
			}
		}
		if f := r.Number.Fill; f != nil {
			if v, ok := f.To(k); ok && v.Fits(dt) {
				out.Number.Fill = &v
			} else {
				note("fill %s was dropped: it does not fit %s", f, dt)
			}
		}
	case k == KindBoolean && isNumeric(r.Kind):
		for _, e := range r.Number.EnumValues {
			v, ok := e.Value.Boolean()
			if !ok {
				note("enum label %q was dropped: %s is not a boolean", e.Label, e.Value)
				continue
			}
			if unique(e.Label) {
				out.Boolean.EnumValues = append(
					out.Boolean.EnumValues, BooleanEntry{Label: e.Label, Value: v},
				)
			}
		}
		if f := r.Number.Fill; f != nil {
			if v, ok := f.Boolean(); ok {
				out.Boolean.Fill = &v
			} else {
				note("fill %s was dropped: it is not a boolean", f)
			}
		}
	case k == r.Kind && k == KindString:
		out.String.Fill = r.String.Fill
		out.String.EnumValues = lo.Filter(
			r.String.EnumValues,
			func(e StringEntry, _ int) bool { return unique(e.Label) },
		)
	case k == r.Kind && k == KindBoolean:
		out.Boolean.Fill = r.Boolean.Fill
		out.Boolean.EnumValues = lo.Filter(
			r.Boolean.EnumValues,
			func(e BooleanEntry, _ int) bool { return unique(e.Label) },
		)
	case k == r.Kind && k == KindTimestamp:
		out.Timestamp = r.Timestamp
	default:
		if n := len(r.labels()); n > 0 {
			note("%d enum entries were dropped", n)
		}
		if r.hasFill() {
			note("the fill was dropped")
		}
		if k == KindTimestamp {
			out.Timestamp.TimeFormat = TimeFormatISO8601
		}
	}
	if index && out.hasFill() {
		note("the fill was dropped, because an index channel takes none")
		out.String.Fill, out.Boolean.Fill = nil, nil
		out.Number.Fill, out.Timestamp.Fill = nil, nil
	}
	return out, changes
}

// Field returns the xjson.Field that reads r into a channel of data type dt, which r
// must agree with.
func (r Read) Field(dt telem.DataType) (xjson.Field, error) {
	f := xjson.Field{DataType: dt}
	switch r.Kind {
	case KindFloat:
		f.Enums = lo.Map(r.Number.EnumValues, func(e NumberEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, float64(e.Value.Float()))
		})
	case KindInteger:
		f.Enums = lo.Map(r.Number.EnumValues, func(e NumberEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, int64(e.Value.Integer()))
		})
	case KindUnsigned:
		f.Enums = lo.Map(r.Number.EnumValues, func(e NumberEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, uint64(e.Value.Unsigned()))
		})
	case KindString:
		f.Enums = lo.Map(r.String.EnumValues, func(e StringEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, e.Value)
		})
	case KindBoolean:
		f.Enums = lo.Map(r.Boolean.EnumValues, func(e BooleanEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, e.Value)
		})
	case KindTimestamp:
		var err error
		format := string(r.Timestamp.TimeFormat)
		if f.TimeFormat, err = xjson.ParseTimeFormat(format); err != nil {
			return xjson.Field{}, err
		}
	}
	return f, nil
}

// Fill returns the fill of r as one sample of data type dt, in the form that
// xjson.Decoder appends. ok is false when r has no fill.
func (r Read) Fill(dt telem.DataType) (sample []byte, ok bool) {
	order := telem.ByteOrder
	switch {
	case isNumeric(r.Kind) && r.Number.Fill != nil:
		return r.Number.Fill.Append(nil, dt), true
	case r.Kind == KindString && r.String.Fill != nil:
		s := *r.String.Fill
		return append(order.AppendUint32(nil, uint32(len(s))), s...), true
	case r.Kind == KindBoolean && r.Boolean.Fill != nil:
		if *r.Boolean.Fill {
			return []byte{1}, true
		}
		return []byte{0}, true
	case r.Kind == KindTimestamp && r.Timestamp.Fill != nil:
		return order.AppendUint64(nil, uint64(*r.Timestamp.Fill)), true
	}
	return nil, false
}

// Absent reports whether raw, the value at the pointer of a field, is no value for a
// channel of data type dt: missing, null except on a JSON channel, or an empty string
// on a channel that cannot hold one. A string, bytes, or JSON channel holds one.
func Absent(raw jsontext.Value, dt telem.DataType) bool {
	switch {
	case raw == nil:
		return true
	case dt == telem.JSONT:
		return false
	case raw.Kind() == 'n':
		return true
	case dt == telem.StringT, dt == telem.BytesT:
		return false
	}
	return string(raw) == `""`
}
