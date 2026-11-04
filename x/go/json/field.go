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
	"encoding/json/jsontext"
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

// Enum pairs a JSON string label with the value of a numeric sample.
type Enum struct {
	Label string
	Value float64
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
	// Enums label the values of numeric samples. A Decoder reads a label as its value,
	// and an Encoder writes a value as its label when Type is String.
	Enums []Enum
}

// variablePrefix is the size of the length prefix of a variable-density sample.
const variablePrefix = 4

// rejectDuplicateNames overrides the leniency of jsontext.Value.Compact.
var rejectDuplicateNames = []jsontext.Options{jsontext.AllowDuplicateNames(false)}

// Decoder converts the JSON values of a Field into samples without allocating. It is
// not safe for concurrent use.
type Decoder struct {
	field Field
	// text is scratch space for a string that is not stored as text.
	text []byte
}

// NewDecoder returns a Decoder for f. It returns an error wrapping
// validate.ErrValidation when no JSON value converts to f.DataType.
func NewDecoder(f Field) (*Decoder, error) {
	switch dt := f.DataType; {
	case dt == telem.TimestampT, dt == telem.StringT, dt == telem.JSONT,
		dt == telem.UUIDT, numeric(dt):
		return &Decoder{field: f}, nil
	default:
		return nil, errors.Wrapf(
			validate.ErrValidation, "cannot convert a JSON value to %s", dt,
		)
	}
}

// Append appends the sample of raw, one JSON value, to dst. A variable-density sample
// starts with its length.
//   - A timestamp takes a string in ISO8601 format, and a number or a string holding
//     one in a Unix format.
//   - A string takes a string. JSON takes any value, compacted but spelled as raw.
//   - A UUID takes its string form.
//   - A numeric or boolean sample takes a number that it can hold exactly, a boolean
//     as 0 or 1, or the label of an enum.
//
// When raw does not convert, Append returns dst unchanged and an error wrapping
// validate.ErrValidation.
func (d *Decoder) Append(dst []byte, raw jsontext.Value) ([]byte, error) {
	out, err := d.append(dst, raw)
	if err != nil {
		return dst, err
	}
	return out, nil
}

func (d *Decoder) append(dst []byte, raw jsontext.Value) ([]byte, error) {
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
	case '"':
		if err := d.unquote(raw); err != nil {
			return number{}, err
		}
		for _, e := range d.field.Enums {
			if e.Label == string(d.text) {
				return number{kind: floatNumber, f: e.Value}, nil
			}
		}
		return number{}, errors.Wrapf(
			validate.ErrValidation, "unknown enum label %q", d.text,
		)
	default:
		return number{}, errors.Wrapf(
			validate.ErrValidation,
			"expected a number or boolean, got %s",
			kindName(raw),
		)
	}
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
	// labels[i] is the label of field.Enums[i] as a JSON string.
	labels [][]byte
}

// NewEncoder returns an Encoder for f. It returns an error wrapping
// validate.ErrValidation when f.Type cannot hold a sample of f.DataType.
func NewEncoder(f Field) (*Encoder, error) {
	dt := f.DataType
	if dt != telem.TimestampT && !numeric(dt) && f.Type != String {
		return nil, errors.Wrapf(
			validate.ErrValidation, "data type %s only serializes as a string", dt,
		)
	}
	e := &Encoder{field: f, labels: make([][]byte, len(f.Enums))}
	for i, en := range f.Enums {
		var err error
		if e.labels[i], err = jsontext.AppendQuote(nil, en.Label); err != nil {
			return nil, errors.Wrapf(validate.ErrValidation, "enum label %q", en.Label)
		}
	}
	return e, nil
}

// Append appends the JSON value of sample to dst. A variable-density sample is its data
// alone, without a length. It returns an error wrapping validate.ErrValidation when
// sample has no JSON value: a NaN or infinite number, or a value no enum labels.
func (e *Encoder) Append(dst, sample []byte) ([]byte, error) {
	dt := e.field.DataType
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
	case !numeric(dt):
		return jsontext.AppendQuote(dst, sample)
	}
	n := sampleNumber(dt, sample)
	switch e.field.Type {
	case Boolean:
		return strconv.AppendBool(dst, n.float() != 0), nil
	case String:
		if len(e.field.Enums) == 0 {
			dst = append(dst, '"')
			return append(n.appendText(dst), '"'), nil
		}
		f := n.float()
		for i, en := range e.field.Enums {
			if en.Value == f {
				return append(dst, e.labels[i]...), nil
			}
		}
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"no enum label for value %s",
			n,
		)
	default:
		return appendJSONNumber(dst, n)
	}
}
