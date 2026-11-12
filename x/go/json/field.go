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
	"encoding/base64"
	"encoding/json/jsontext"
	"math"
	"strconv"
	"uuid"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Type is the JSON type that a sample converts to.
type Type uint8

const (
	// Number is a JSON number.
	Number Type = iota
	// String is a JSON string.
	String
	// Boolean is a JSON boolean.
	Boolean
)

// ParseType parses the config form of a Type: "number", "string", or "boolean". It
// returns an error wrapping validate.ErrValidation for any other text.
func ParseType(text string) (Type, error) {
	switch text {
	case "number":
		return Number, nil
	case "string":
		return String, nil
	case "boolean":
		return Boolean, nil
	}
	return Number, errors.Wrapf(
		validate.ErrValidation,
		`unknown JSON type %q: expected "number", "string", or "boolean"`,
		text,
	)
}

// String implements fmt.Stringer.
func (t Type) String() string {
	switch t {
	case String:
		return "string"
	case Boolean:
		return "boolean"
	}
	return "number"
}

// MarshalText implements encoding.TextMarshaler.
func (t Type) MarshalText() ([]byte, error) { return []byte(t.String()), nil }

// UnmarshalText implements encoding.TextUnmarshaler.
func (t *Type) UnmarshalText(text []byte) (err error) {
	*t, err = ParseType(string(text))
	return err
}

// Enum pairs a JSON string label with one sample. A numeric or boolean sample converts
// to an integer or boolean data type that holds it exactly, or to the nearest value of
// a float data type, failing past its range. A string sample converts only to a
// string.
type Enum struct {
	label    string
	dataType telem.DataType
	// sample has no length prefix.
	sample []byte
}

// NewEnum returns an Enum that labels v.
func NewEnum[T telem.Sample](label string, v T) Enum {
	s := telem.NewSeriesV(v)
	sample := s.Data
	if s.DataType.IsVariable() {
		sample, _ = telem.UnmarshalVariableSample(s.Data)
	}
	return Enum{label: label, dataType: s.DataType, sample: sample}
}

func encodeEnums(enums []Enum, dt telem.DataType) ([]Enum, error) {
	out := make([]Enum, len(enums))
	for i, en := range enums {
		sample, err := encodeEnum(en, dt)
		if err != nil {
			return nil, errors.Wrapf(err, "enum label %q", en.label)
		}
		out[i] = Enum{label: en.label, dataType: dt, sample: sample}
	}
	return out, nil
}

func encodeEnum(en Enum, dt telem.DataType) ([]byte, error) {
	from := en.dataType
	switch {
	case dt == telem.StringT && from == telem.StringT:
		return en.sample, nil
	case numeric(dt) && numeric(from):
		n := sampleNumber(from, en.sample)
		if err := checkFits(n, dt); err != nil {
			return nil, err
		}
		if f := n.float(); dt == telem.BooleanT && f != 0 && f != 1 {
			return nil, errors.Wrapf(validate.ErrValidation, "%s is not a boolean", n)
		}
		return appendNumber(nil, dt, n), nil
	default:
		return nil, errors.Wrapf(
			validate.ErrValidation, "a %s value does not convert to %s", from, dt,
		)
	}
}

// Field is how the values at one place in a JSON document convert to and from samples.
type Field struct {
	// DataType is the data type of the samples.
	DataType telem.DataType
	// TimeFormat encodes timestamp samples.
	TimeFormat TimeFormat
	// Type is the JSON type an Encoder writes a sample as. Timestamp samples take
	// TimeFormat instead.
	Type Type
	// Enums label the samples of a numeric, boolean, or string data type. A Decoder
	// reads a label as its sample. An Encoder writes a sample as the label of the first
	// entry that matches it, whatever the Type. Floats match by value, and a NaN entry
	// matches any NaN.
	Enums []Enum
}

// variablePrefix is the size of the length prefix of a variable-density sample.
const variablePrefix = 4

// strictBase64 rejects base64 with padding bits that are not zero. It still skips line
// breaks.
var strictBase64 = base64.StdEncoding.Strict()

// rejectDuplicateNames overrides the leniency of jsontext.Value.Compact.
var rejectDuplicateNames = []jsontext.Options{jsontext.AllowDuplicateNames(false)}

// Decoder converts the JSON values of a Field into samples without allocating. It is
// not safe for concurrent use.
type Decoder struct {
	field Field
	enums []Enum
	// text is scratch space for a string that is not stored as text.
	text []byte
}

// NewDecoder returns a Decoder for f. It returns an error wrapping
// validate.ErrValidation when no JSON value converts to f.DataType, or an enum value
// does not convert to it.
func NewDecoder(f Field) (*Decoder, error) {
	switch dt := f.DataType; {
	case dt == telem.TimestampT, dt == telem.StringT, dt == telem.JSONT,
		dt == telem.UUIDT, dt == telem.BytesT, numeric(dt):
	default:
		return nil, errors.Wrapf(
			validate.ErrValidation, "cannot convert a JSON value to %s", dt,
		)
	}
	enums, err := encodeEnums(f.Enums, f.DataType)
	if err != nil {
		return nil, err
	}
	return &Decoder{field: f, enums: enums}, nil
}

// Append appends the sample of raw, one JSON value, to dst. A variable-density sample
// starts with its length.
//   - A timestamp takes a string in ISO8601 format, and a number or a string holding
//     one in a Unix format.
//   - A string takes a string, or the label of an enum. JSON takes any value,
//     compacted but spelled as raw.
//   - A UUID takes its string form.
//   - Bytes take a string of standard, padded base64.
//   - A numeric or boolean sample takes a number that it can hold exactly, a boolean
//     as 0 or 1, or the label of an enum.
//
// When raw does not convert, Append returns an error wrapping validate.ErrValidation.
func (d *Decoder) Append(dst []byte, raw jsontext.Value) ([]byte, error) {
	switch d.field.DataType {
	case telem.TimestampT:
		ts, err := d.timeStamp(raw)
		if err != nil {
			return nil, err
		}
		return telem.ByteOrder.AppendUint64(dst, uint64(ts)), nil
	case telem.StringT:
		if raw.Kind() != '"' {
			return nil, errors.Wrapf(
				validate.ErrValidation, "expected a string, got %s", kindName(raw),
			)
		}
		if len(d.enums) > 0 {
			if err := d.unquote(raw); err != nil {
				return nil, err
			}
			if sample, ok := d.enum(); ok {
				dst = telem.ByteOrder.AppendUint32(dst, uint32(len(sample)))
				return append(dst, sample...), nil
			}
		}
		start := len(dst)
		dst = append(dst, make([]byte, variablePrefix)...)
		var err error
		if dst, err = jsontext.AppendUnquote(dst, raw); err != nil {
			return nil, errors.Wrap(validate.ErrValidation, err.Error())
		}
		return endVariable(dst, start), nil
	case telem.JSONT:
		start := len(dst)
		dst = append(dst, make([]byte, variablePrefix)...)
		dst = append(dst, raw...)
		if kind := raw.Kind(); kind == '{' || kind == '[' {
			v := jsontext.Value(dst[start+variablePrefix:])
			if err := v.Compact(rejectDuplicateNames...); err != nil {
				return nil, errors.Wrap(validate.ErrValidation, err.Error())
			}
			dst = dst[:start+variablePrefix+len(v)]
		}
		return endVariable(dst, start), nil
	case telem.UUIDT:
		if raw.Kind() != '"' {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"expected a UUID string, got %s",
				kindName(raw),
			)
		}
		if err := d.unquote(raw); err != nil {
			return nil, err
		}
		var id uuid.UUID
		if err := id.UnmarshalText(d.text); err != nil {
			return nil, errors.Wrapf(validate.ErrValidation, "invalid UUID %q", d.text)
		}
		return append(dst, id[:]...), nil
	case telem.BytesT:
		if raw.Kind() != '"' {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"expected a base64 string, got %s",
				kindName(raw),
			)
		}
		if err := d.unquote(raw); err != nil {
			return nil, err
		}
		if bytes.ContainsAny(d.text, "\r\n") {
			return nil, errors.Wrap(
				validate.ErrValidation, "base64 must not hold a line break",
			)
		}
		start := len(dst)
		dst = append(dst, make([]byte, variablePrefix)...)
		var err error
		if dst, err = strictBase64.AppendDecode(dst, d.text); err != nil {
			return nil, errors.Wrap(validate.ErrValidation, err.Error())
		}
		return endVariable(dst, start), nil
	}
	if raw.Kind() == '"' {
		if err := d.unquote(raw); err != nil {
			return nil, err
		}
		if sample, ok := d.enum(); ok {
			return append(dst, sample...), nil
		}
		return nil, errors.Wrapf(
			validate.ErrValidation, "unknown enum label %q", d.text,
		)
	}
	n, err := d.number(raw)
	if err != nil {
		return nil, err
	}
	if err := checkFits(n, d.field.DataType); err != nil {
		return nil, err
	}
	return appendNumber(dst, d.field.DataType, n), nil
}

func endVariable(dst []byte, start int) []byte {
	size := len(dst) - start - variablePrefix
	telem.ByteOrder.PutUint32(dst[start:], uint32(size))
	return dst
}

func (d *Decoder) number(raw jsontext.Value) (number, error) {
	switch raw.Kind() {
	case 't':
		return number{kind: intNumber, i: 1}, nil
	case 'f':
		return number{kind: intNumber}, nil
	case '0':
		return parseNumber(raw)
	default:
		return number{}, errors.Wrapf(
			validate.ErrValidation,
			"expected a number or boolean, got %s",
			kindName(raw),
		)
	}
}

func (d *Decoder) enum() ([]byte, bool) {
	for _, e := range d.enums {
		if e.label == string(d.text) {
			return e.sample, true
		}
	}
	return nil, false
}

func (d *Decoder) timeStamp(raw jsontext.Value) (telem.TimeStamp, error) {
	format := d.field.TimeFormat
	if format == ISO8601 {
		if raw.Kind() != '"' {
			return 0, errors.Wrapf(
				validate.ErrValidation,
				"expected an ISO 8601 string, got %s",
				kindName(raw),
			)
		}
		if err := d.unquote(raw); err != nil {
			return 0, err
		}
		return parseISO8601(d.text)
	}
	switch raw.Kind() {
	case '0':
		return parseUnix(raw, format)
	case '"':
		if err := d.unquote(raw); err != nil {
			return 0, err
		}
		return parseUnix(d.text, format)
	default:
		return 0, errors.Wrapf(
			validate.ErrValidation, "expected a number, got %s", kindName(raw),
		)
	}
}

func (d *Decoder) unquote(raw jsontext.Value) error {
	var err error
	if d.text, err = jsontext.AppendUnquote(d.text[:0], raw); err != nil {
		return errors.Wrap(validate.ErrValidation, err.Error())
	}
	return nil
}

func kindName(raw jsontext.Value) string {
	switch raw.Kind() {
	case '"':
		return "a string"
	case '0':
		return "a number"
	case 't', 'f':
		return "a boolean"
	case 'n':
		return "null"
	case '{':
		return "an object"
	default:
		return "an array"
	}
}

// Encoder converts the samples of a Field into JSON values without allocating. It is
// safe for concurrent use.
type Encoder struct {
	field Field
	// enums hold each label as a JSON string.
	enums []Enum
}

// NewEncoder returns an Encoder for f. It returns an error wrapping
// validate.ErrValidation when f.Type cannot hold a sample of f.DataType, or an enum
// value does not convert to f.DataType.
func NewEncoder(f Field) (*Encoder, error) {
	dt := f.DataType
	if dt != telem.TimestampT && !numeric(dt) && f.Type != String {
		return nil, errors.Wrapf(
			validate.ErrValidation, "data type %s only serializes as a string", dt,
		)
	}
	enums, err := encodeEnums(f.Enums, dt)
	if err != nil {
		return nil, err
	}
	for i := range enums {
		label := enums[i].label
		quoted, err := jsontext.AppendQuote(nil, label)
		if err != nil {
			return nil, errors.Wrapf(validate.ErrValidation, "enum label %q", label)
		}
		enums[i].label = string(quoted)
	}
	return &Encoder{field: f, enums: enums}, nil
}

// Append appends the JSON value of sample to dst. A variable-density sample is its data
// alone, without a length. Bytes are standard, padded base64. A sample that matches no
// enum and is a NaN or infinite number writes as NaN, Infinity, or -Infinity when Type
// is String. It returns an error wrapping validate.ErrValidation when sample has no
// JSON value: a NaN or infinite number, or a NaN boolean.
func (e *Encoder) Append(dst, sample []byte) ([]byte, error) {
	dt := e.field.DataType
	for _, en := range e.enums {
		if matches(dt, en.sample, sample) {
			return append(dst, en.label...), nil
		}
	}
	switch {
	case dt == telem.TimestampT:
		ts := telem.TimeStamp(telem.ByteOrder.Uint64(sample))
		return AppendTimeStamp(dst, ts, e.field.TimeFormat), nil
	case dt == telem.UUIDT:
		dst = append(dst, '"')
		var err error
		if dst, err = uuid.UUID(sample).AppendText(dst); err != nil {
			return nil, err
		}
		return append(dst, '"'), nil
	case dt == telem.BytesT:
		dst = base64.StdEncoding.AppendEncode(append(dst, '"'), sample)
		return append(dst, '"'), nil
	case !numeric(dt):
		return jsontext.AppendQuote(dst, sample)
	}
	n := sampleNumber(dt, sample)
	switch e.field.Type {
	case Boolean:
		if n.kind == floatNumber && math.IsNaN(n.f) {
			return nil, errors.Wrap(validate.ErrValidation, "NaN is not a JSON boolean")
		}
		return strconv.AppendBool(dst, n.float() != 0), nil
	case String:
		dst = append(dst, '"')
		return append(n.appendText(dst), '"'), nil
	default:
		return appendJSONNumber(dst, n)
	}
}

// matches reports whether a and b, samples of dt, are equal. Floats are equal by value,
// and any two NaNs are equal.
func matches(dt telem.DataType, a, b []byte) bool {
	switch dt {
	case telem.Float64T, telem.Float32T:
		x, y := sampleNumber(dt, a).f, sampleNumber(dt, b).f
		return x == y || (math.IsNaN(x) && math.IsNaN(y))
	case telem.BooleanT:
		return (a[0] != 0) == (b[0] != 0)
	default:
		return bytes.Equal(a, b)
	}
}
