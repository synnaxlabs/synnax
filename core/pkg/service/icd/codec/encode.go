// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package codec

import (
	"encoding/json/jsontext"
	"math"
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// JSONEncoder encodes samples into the documents of one JSON message. It is not safe
// for concurrent use.
type JSONEncoder struct {
	name     string
	template xjson.Template
	slots    encodeSlots
}

// encodeSlots fills the slots of a JSONEncoder's template with the samples of one
// Encode.
type encodeSlots struct {
	// fields holds the field of each binding.
	fields []encodeField
	// bindings holds the binding of each slot of the template, or -1 for a literal.
	bindings []int
	samples  [][]byte
}

type encodeField struct {
	name    string
	kind    payload.Kind
	encoder *xjson.Encoder
	// scaled is true when the field's scale or offset changes its value.
	scaled        bool
	scale, offset float64
}

// NewJSONEncoder returns an encoder of the documents of m. A document holds the value
// of m's match rule, the sample of each bound field, and the default of each other
// field. A scaled field writes its sample before the scale and offset, and an integer
// field rounds it to the nearest integer. NewJSONEncoder returns an error wrapping
// validate.ErrValidation when:
//   - a field has no binding and no default
//   - a binding names no field of m, or a field has two bindings
//   - a binding's data type does not convert to its field's kind
//   - two pointers are equal, or one holds the other
func NewJSONEncoder(m JSONMessage) (*JSONEncoder, error) {
	e, err := newJSONEncoder(m)
	if err != nil {
		return nil, errors.Wrapf(err, "message %s", m.Name)
	}
	return e, nil
}

func newJSONEncoder(m JSONMessage) (*JSONEncoder, error) {
	byKey := make(map[icd.FieldKey]icd.JSONField, len(m.Payload.Fields))
	for _, f := range m.Payload.Fields {
		byKey[f.Base().Key] = f
	}
	bound := make(map[icd.FieldKey]int, len(m.Bindings))
	for i, b := range m.Bindings {
		f, ok := byKey[b.Field]
		if !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation, "field %s does not exist", b.Field,
			)
		}
		if _, ok = bound[b.Field]; ok {
			return nil, errors.Wrapf(
				validate.ErrValidation,
				"field %s has more than one binding",
				f.Base().Name,
			)
		}
		bound[b.Field] = i
	}
	e := &JSONEncoder{
		name:  m.Name,
		slots: encodeSlots{fields: make([]encodeField, len(m.Bindings))},
	}
	var fields []xjson.TemplateField
	if m.Payload.Match != nil {
		f, err := matchLiteral(*m.Payload.Match)
		if err != nil {
			return nil, errors.Wrap(err, "match")
		}
		fields = append(fields, f)
		e.slots.bindings = append(e.slots.bindings, -1)
	}
	for _, f := range m.Payload.Fields {
		tf, binding, err := e.templateField(m, f, bound)
		if err != nil {
			return nil, errors.Wrapf(err, "field %s", f.Base().Name)
		}
		fields = append(fields, tf)
		e.slots.bindings = append(e.slots.bindings, binding)
	}
	var err error
	if e.template, err = xjson.NewTemplate(fields); err != nil {
		return nil, err
	}
	return e, nil
}

// templateField returns the template field of f and its binding in m, or -1 when f
// takes its default.
func (e *JSONEncoder) templateField(
	m JSONMessage,
	f icd.JSONField,
	bound map[icd.FieldKey]int,
) (xjson.TemplateField, int, error) {
	b := f.Base()
	pointer, err := xjson.ParsePointer(b.Pointer)
	if err != nil {
		return xjson.TemplateField{}, 0, err
	}
	tf := xjson.TemplateField{Label: "field " + b.Name, Pointer: pointer}
	binding, ok := bound[b.Key]
	if !ok {
		tf.Literal, err = defaultLiteral(f)
		return tf, -1, err
	}
	e.slots.fields[binding], err = newEncodeField(f, m.Bindings[binding].DataType)
	return tf, binding, err
}

func newEncodeField(jf icd.JSONField, dt telem.DataType) (encodeField, error) {
	b := jf.Base()
	f := encodeField{
		name:   b.Name,
		scaled: b.Scale != 1 || b.Offset != 0,
		scale:  b.Scale,
		offset: b.Offset,
	}
	sample := xjson.Field{DataType: dt, Enums: enumsOf(jf), Type: xjson.Number}
	switch v := jf.Variant.(type) {
	case icd.FloatJSONField:
		f.kind = payload.KindFloat
	case icd.IntegerJSONField:
		f.kind = payload.KindInteger
	case icd.UnsignedJSONField:
		f.kind = payload.KindUnsigned
	case icd.StringJSONField:
		f.kind, sample.Type = payload.KindString, xjson.String
	case icd.BooleanJSONField:
		f.kind, sample.Type = payload.KindBoolean, xjson.Boolean
	case icd.TimestampJSONField:
		f.kind = payload.KindTimestamp
		var err error
		format := string(v.TimeFormat)
		if sample.TimeFormat, err = xjson.ParseTimeFormat(format); err != nil {
			return encodeField{}, err
		}
	case icd.JSONJSONField:
		f.kind = payload.KindJSON
	case icd.UuidJSONField:
		f.kind, sample.Type = payload.KindUUID, xjson.String
	case icd.BytesJSONField:
		f.kind, sample.Type = payload.KindBytes, xjson.String
	default:
		return encodeField{}, errors.Wrap(validate.ErrValidation, "type is required")
	}
	if !Converts(f.kind, dt) {
		return encodeField{}, errors.Wrapf(
			validate.ErrValidation, "cannot write %s channels to %s fields", dt, f.kind,
		)
	}
	if f.scaled && dt != telem.Float32T && dt != telem.Float64T {
		return encodeField{}, errors.Wrapf(
			validate.ErrValidation,
			"a scaled field requires a float channel, not %s",
			dt,
		)
	}
	var err error
	if f.kind != payload.KindJSON {
		if f.encoder, err = xjson.NewEncoder(sample); err != nil {
			return encodeField{}, err
		}
	}
	return f, nil
}

// matchLiteral returns the template field that writes the value of m.
func matchLiteral(m icd.JSONMatch) (xjson.TemplateField, error) {
	pointer, err := xjson.ParsePointer(m.Base().Pointer)
	if err != nil {
		return xjson.TemplateField{}, err
	}
	f := xjson.TemplateField{Label: "match", Pointer: pointer}
	switch v := m.Variant.(type) {
	case icd.FloatJSONMatch:
		f.Literal, err = xjson.AppendFloat(nil, float64(v.Value))
	case icd.IntegerJSONMatch:
		f.Literal = strconv.AppendInt(nil, int64(v.Value), 10)
	case icd.UnsignedJSONMatch:
		f.Literal = strconv.AppendUint(nil, uint64(v.Value), 10)
	case icd.StringJSONMatch:
		f.Literal, err = quote(v.Value)
	case icd.BooleanJSONMatch:
		f.Literal = strconv.AppendBool(nil, v.Value)
	default:
		return xjson.TemplateField{}, errors.Wrap(
			validate.ErrValidation,
			"type is required",
		)
	}
	return f, err
}

// defaultLiteral returns the JSON text of the default of jf.
func defaultLiteral(jf icd.JSONField) ([]byte, error) {
	switch v := jf.Variant.(type) {
	case icd.FloatJSONField:
		if v.Default != nil {
			return xjson.AppendFloat(nil, float64(*v.Default))
		}
	case icd.IntegerJSONField:
		if v.Default != nil {
			return strconv.AppendInt(nil, int64(*v.Default), 10), nil
		}
	case icd.UnsignedJSONField:
		if v.Default != nil {
			return strconv.AppendUint(nil, uint64(*v.Default), 10), nil
		}
	case icd.StringJSONField:
		if v.Default != nil {
			return quote(*v.Default)
		}
	case icd.BooleanJSONField:
		if v.Default != nil {
			return strconv.AppendBool(nil, *v.Default), nil
		}
	case icd.TimestampJSONField:
		if v.Default != nil {
			format, err := xjson.ParseTimeFormat(string(v.TimeFormat))
			if err != nil {
				return nil, err
			}
			return xjson.AppendTimeStamp(nil, *v.Default, format), nil
		}
	case nil:
		return nil, errors.Wrap(validate.ErrValidation, "type is required")
	}
	return nil, errors.Wrap(validate.ErrValidation, "no channel and no default")
}

func quote(s string) ([]byte, error) {
	text, err := jsontext.AppendQuote(nil, s)
	if err != nil {
		return nil, errors.Wrap(validate.ErrValidation, err.Error())
	}
	return text, nil
}

// Encode appends the document of samples to dst. samples holds one sample for each
// binding, in binding order. A variable-density sample is its data alone, without a
// length. Encode returns an error wrapping validate.ErrValidation when a sample has no
// value in its field.
func (e *JSONEncoder) Encode(dst []byte, samples [][]byte) ([]byte, error) {
	e.slots.samples = samples
	out, err := e.template.Append(dst, &e.slots)
	e.slots.samples = nil
	if err != nil {
		return nil, errors.Wrapf(err, "message %s", e.name)
	}
	return out, nil
}

// AppendSlot implements xjson.Slots.
func (s *encodeSlots) AppendSlot(dst []byte, field int) ([]byte, error) {
	binding := s.bindings[field]
	f := &s.fields[binding]
	out, err := f.append(dst, s.samples[binding])
	if err != nil {
		return nil, errors.Wrapf(err, "field %s", f.name)
	}
	return out, nil
}

func (f *encodeField) append(dst, sample []byte) ([]byte, error) {
	if f.kind == payload.KindJSON {
		if !jsontext.Value(sample).IsValid() {
			return nil, errors.Wrap(validate.ErrValidation, "invalid JSON")
		}
		return append(dst, sample...), nil
	}
	start := len(dst)
	dst, err := f.encoder.Append(dst, sample)
	if err != nil {
		return nil, err
	}
	numeric := f.kind == payload.KindFloat || f.kind == payload.KindInteger ||
		f.kind == payload.KindUnsigned
	// An enum label writes a channel value, which takes no scale.
	if !numeric || dst[start] == '"' ||
		(f.kind == payload.KindFloat && !f.scaled) {
		return dst, nil
	}
	n, err := xjson.ParseNumber(dst[start:])
	if err != nil {
		return nil, err
	}
	dst = dst[:start]
	if f.kind == payload.KindFloat {
		return xjson.AppendFloat(dst, (n.Float64()-f.offset)/f.scale)
	}
	return f.appendInteger(dst, n)
}

// appendInteger appends n, unscaled and rounded to the nearest integer, to dst.
func (f *encodeField) appendInteger(dst []byte, n xjson.ParsedNumber) ([]byte, error) {
	if !f.scaled {
		if v, ok := n.Int64(); ok && f.kind == payload.KindInteger {
			return strconv.AppendInt(dst, v, 10), nil
		}
		if v, ok := n.Uint64(); ok && f.kind == payload.KindUnsigned {
			return strconv.AppendUint(dst, v, 10), nil
		}
	}
	v := n.Float64()
	if f.scaled {
		v = (v - f.offset) / f.scale
	}
	r := math.Round(v)
	switch {
	case f.kind == payload.KindInteger && r >= math.MinInt64 && r < math.MaxInt64:
		return strconv.AppendInt(dst, int64(r), 10), nil
	case f.kind == payload.KindUnsigned && r >= 0 && r < math.MaxUint64:
		return strconv.AppendUint(dst, uint64(r), 10), nil
	}
	return nil, errors.Wrapf(
		validate.ErrValidation, "%v is out of range for %s values", v, f.kind,
	)
}
