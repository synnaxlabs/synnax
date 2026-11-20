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
	"fmt"
	"math"
	"slices"
	"strconv"
	"uuid"

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
	// bindings holds the binding of each field of the template, or -1 for a default.
	bindings []int
	// generate holds, for each field of the template, the generator of a default that
	// takes a new value at each encode, or nil.
	generate []generator
	samples  [][]byte
}

// generator appends a new JSON value to dst.
type generator func(dst []byte) ([]byte, error)

type encodeField struct {
	name    string
	kind    payload.Kind
	encoder *xjson.Encoder
	// scaled is true when the field's scale or offset changes its value.
	scaled bool
	// converted is true when the number that encoder writes needs a scale, a rounding,
	// a range check, or a change to a boolean.
	converted     bool
	scale, offset float64
}

// NewJSONEncoder returns an encoder of the documents of m. A document holds the value
// of m's match rule, the sample of each bound field, and the default of each other
// field. A scaled field writes its sample before the scale and offset, and an integer
// field rounds it to the nearest integer. A number channel writes to any number or
// boolean field, and a boolean field takes only 0 and 1 from one. NewJSONEncoder
// returns an error wrapping validate.ErrValidation when:
//   - a field has no binding and no default
//   - a binding names no field of m, or a field has two bindings
//   - a binding's data type does not write to its field's kind
//   - a scaled field's channel is not a float
//   - two pointers are equal, or one holds the other
//   - a pointer holds an array index, which a document cannot place
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
		e.slots.generate = append(e.slots.generate, nil)
	}
	for _, f := range m.Payload.Fields {
		tf, binding, gen, err := e.templateField(m, f, bound)
		if err != nil {
			return nil, errors.Wrapf(err, "field %s", f.Base().Name)
		}
		fields = append(fields, tf)
		e.slots.bindings = append(e.slots.bindings, binding)
		e.slots.generate = append(e.slots.generate, gen)
	}
	var err error
	if e.template, err = xjson.NewTemplate(fields); err != nil {
		return nil, err
	}
	return e, nil
}

// templateField returns the template field of f and its binding in m, or -1 when f
// takes its default. It returns the generator of a default that takes a new value at
// each encode. It stores the encode field of a bound f in e.
func (e *JSONEncoder) templateField(
	m JSONMessage,
	f icd.JSONField,
	bound map[icd.FieldKey]int,
) (xjson.TemplateField, int, generator, error) {
	b := f.Base()
	pointer, err := memberPointer(b.Pointer)
	if err != nil {
		return xjson.TemplateField{}, 0, nil, err
	}
	tf := xjson.TemplateField{Label: "field " + b.Name, Pointer: pointer}
	binding, ok := bound[b.Key]
	if !ok {
		var gen generator
		if tf.Literal, gen, err = defaultOf(f); err != nil {
			return xjson.TemplateField{}, 0, nil, err
		}
		return tf, -1, gen, nil
	}
	ef, err := newEncodeField(f, m.Bindings[binding].DataType)
	if err != nil {
		return xjson.TemplateField{}, 0, nil, err
	}
	e.slots.fields[binding] = ef
	return tf, binding, nil, nil
}

// memberPointer parses text as a pointer that names object members. It returns an error
// wrapping validate.ErrValidation for an array index, since a decoder reads one as an
// element.
func memberPointer(text string) (xjson.Pointer, error) {
	pointer, err := xjson.ParsePointer(text)
	if err != nil {
		return nil, err
	}
	if slices.ContainsFunc(pointer, arrayIndex) {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"pointer %q holds an array index, which a document cannot place",
			text,
		)
	}
	return pointer, nil
}

// arrayIndex reports whether token is "0" or digits that do not start with "0".
func arrayIndex(token string) bool {
	if token == "" || (len(token) > 1 && token[0] == '0') {
		return false
	}
	for i := range len(token) {
		if token[i] < '0' || token[i] > '9' {
			return false
		}
	}
	return true
}

// writes reports whether samples of dt write to a field of kind k.
func writes(k payload.Kind, dt telem.DataType) bool {
	dk, ok := payload.KindOf(dt)
	return ok && (dk == k || (numeric(dk) && numeric(k)))
}

// numeric reports whether k holds numbers or booleans.
func numeric(k payload.Kind) bool {
	switch k {
	case payload.KindFloat, payload.KindInteger, payload.KindUnsigned,
		payload.KindBoolean:
		return true
	case payload.KindString, payload.KindTimestamp, payload.KindJSON,
		payload.KindUUID, payload.KindBytes:
	}
	return false
}

func newEncodeField(jf icd.JSONField, dt telem.DataType) (encodeField, error) {
	b := jf.Base()
	f := encodeField{
		name:   b.Name,
		scaled: b.Scale != 1 || b.Offset != 0,
		scale:  b.Scale,
		offset: b.Offset,
	}
	kind, sample, err := sampleField(jf, dt)
	if err != nil {
		return encodeField{}, err
	}
	if !writes(kind, dt) {
		return encodeField{}, errors.Wrapf(
			validate.ErrValidation, "cannot write %s channels to %s fields", dt, kind,
		)
	}
	if f.scaled && dt != telem.Float32T && dt != telem.Float64T {
		return encodeField{}, errors.Wrapf(
			validate.ErrValidation,
			"a scaled field requires a float channel, not %s",
			dt,
		)
	}
	f.kind, f.converted = kind, converted(kind, dt, f.scaled)
	if kind == payload.KindBoolean && f.converted {
		sample.Type = xjson.Number
	}
	if kind != payload.KindJSON {
		if f.encoder, err = xjson.NewEncoder(sample); err != nil {
			return encodeField{}, err
		}
	}
	return f, nil
}

// sampleField returns the kind of jf and the xjson field that writes a sample of dt as
// a value of that kind.
func sampleField(
	jf icd.JSONField,
	dt telem.DataType,
) (payload.Kind, xjson.Field, error) {
	f := xjson.Field{DataType: dt, Enums: enumsOf(jf), Type: xjson.Number}
	switch v := jf.Variant.(type) {
	case icd.FloatJSONField:
		return payload.KindFloat, f, nil
	case icd.IntegerJSONField:
		return payload.KindInteger, f, nil
	case icd.UnsignedJSONField:
		return payload.KindUnsigned, f, nil
	case icd.StringJSONField:
		f.Type = xjson.String
		return payload.KindString, f, nil
	case icd.BooleanJSONField:
		f.Type = xjson.Boolean
		return payload.KindBoolean, f, nil
	case icd.TimestampJSONField:
		var err error
		if f.TimeFormat, err = xjson.ParseTimeFormat(string(v.TimeFormat)); err != nil {
			return "", xjson.Field{}, err
		}
		return payload.KindTimestamp, f, nil
	case icd.JSONJSONField:
		return payload.KindJSON, f, nil
	case icd.UuidJSONField:
		f.Type = xjson.String
		return payload.KindUUID, f, nil
	case icd.BytesJSONField:
		f.Type = xjson.String
		return payload.KindBytes, f, nil
	}
	return "", xjson.Field{}, errors.Wrap(validate.ErrValidation, "type is required")
}

// converted reports whether a field of kind k changes the number that an encoder writes
// for a sample of dt.
func converted(k payload.Kind, dt telem.DataType, scaled bool) bool {
	dk, _ := payload.KindOf(dt)
	switch k {
	case payload.KindFloat:
		return scaled
	case payload.KindInteger, payload.KindUnsigned:
		return scaled || dk != k
	case payload.KindBoolean:
		return dk != payload.KindBoolean
	case payload.KindString, payload.KindTimestamp, payload.KindJSON,
		payload.KindUUID, payload.KindBytes:
	}
	return false
}

// matchLiteral returns the template field that writes the value of m.
func matchLiteral(m icd.JSONMatch) (xjson.TemplateField, error) {
	pointer, err := memberPointer(m.Base().Pointer)
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
	if err != nil {
		return xjson.TemplateField{}, err
	}
	return f, nil
}

// defaultOf returns the JSON text of the default of jf, or the generator of a default
// that takes a new value at each encode.
func defaultOf(jf icd.JSONField) ([]byte, generator, error) {
	switch v := jf.Variant.(type) {
	case icd.FloatJSONField:
		if v.Default != nil {
			text, err := xjson.AppendFloat(nil, float64(*v.Default))
			return text, nil, err
		}
	case icd.IntegerJSONField:
		if v.Default != nil {
			return strconv.AppendInt(nil, int64(*v.Default), 10), nil, nil
		}
	case icd.UnsignedJSONField:
		if v.Default != nil {
			return strconv.AppendUint(nil, uint64(*v.Default), 10), nil, nil
		}
	case icd.StringJSONField:
		if v.Default != nil {
			text, err := quote(*v.Default)
			return text, nil, err
		}
	case icd.BooleanJSONField:
		if v.Default != nil {
			return strconv.AppendBool(nil, *v.Default), nil, nil
		}
	case icd.TimestampJSONField:
		if v.Default != nil {
			return timestampDefault(v)
		}
	case icd.JSONJSONField:
		if v.Default != nil {
			text, err := jsonDefault(*v.Default)
			return text, nil, err
		}
	case icd.UuidJSONField:
		if v.Default != nil {
			return uuidDefault(*v.Default)
		}
	case icd.BytesJSONField:
		if v.Default != nil {
			text, err := bytesDefault(*v.Default)
			return text, nil, err
		}
	case nil:
		return nil, nil, errors.Wrap(validate.ErrValidation, "type is required")
	}
	return nil, nil, errors.Wrap(validate.ErrValidation, "no channel and no default")
}

func timestampDefault(f icd.TimestampJSONField) ([]byte, generator, error) {
	format, err := xjson.ParseTimeFormat(string(f.TimeFormat))
	if err != nil {
		return nil, nil, err
	}
	switch d := f.Default.Variant.(type) {
	case icd.ConstantTimestampDefault:
		return xjson.AppendTimeStamp(nil, d.Value, format), nil, nil
	case icd.NowTimestampDefault:
		return nil, func(dst []byte) ([]byte, error) {
			return xjson.AppendTimeStamp(dst, telem.Now(), format), nil
		}, nil
	case nil:
	}
	return nil, nil, errUntypedDefault
}

func uuidDefault(d icd.UUIDDefault) ([]byte, generator, error) {
	switch d := d.Variant.(type) {
	case icd.ConstantUUIDDefault:
		text, err := appendUUID(nil, d.Value)
		return text, nil, err
	case icd.RandomUUIDDefault:
		return nil, func(dst []byte) ([]byte, error) {
			return appendUUID(dst, uuid.NewV4())
		}, nil
	case nil:
	}
	return nil, nil, errUntypedDefault
}

// appendUUID appends id to dst as a JSON string.
func appendUUID(dst []byte, id uuid.UUID) ([]byte, error) {
	dst, err := id.AppendText(append(dst, '"'))
	if err != nil {
		return nil, err
	}
	return append(dst, '"'), nil
}

func bytesDefault(text string) ([]byte, error) {
	if _, err := xjson.DecodeBase64(text); err != nil {
		return nil, errors.Wrap(err, "default")
	}
	return quote(text)
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
// length. Encode returns an error wrapping validate.ErrValidation when a field cannot
// hold its sample.
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
	if gen := s.generate[field]; gen != nil {
		return gen(dst)
	}
	binding := s.bindings[field]
	f := &s.fields[binding]
	out, err := f.encode(dst, s.samples[binding])
	if err != nil {
		return nil, errors.Wrapf(err, "field %s", f.name)
	}
	return out, nil
}

func (f *encodeField) encode(dst, sample []byte) ([]byte, error) {
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
	// An enum label writes a channel value, which takes no conversion.
	if !f.converted || dst[start] == '"' {
		return dst, nil
	}
	n, err := xjson.ParseNumber(dst[start:])
	if err != nil {
		return nil, err
	}
	dst = dst[:start]
	switch f.kind {
	case payload.KindFloat:
		return xjson.AppendFloat(dst, (n.Float64()-f.offset)/f.scale)
	case payload.KindInteger, payload.KindUnsigned:
		return f.appendInteger(dst, n)
	case payload.KindBoolean:
		return appendBoolean(dst, n.Float64())
	case payload.KindString, payload.KindTimestamp, payload.KindJSON,
		payload.KindUUID, payload.KindBytes:
	}
	panic(fmt.Sprintf("codec: %s fields are never converted", f.kind))
}

// appendBoolean appends v to dst as a JSON boolean when v is 0 or 1.
func appendBoolean(dst []byte, v float64) ([]byte, error) {
	switch v {
	case 0:
		return strconv.AppendBool(dst, false), nil
	case 1:
		return strconv.AppendBool(dst, true), nil
	}
	return nil, errors.Wrapf(validate.ErrValidation, "%v is not 0 or 1", v)
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
