// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package codec turns the payloads that ICD messages describe into channel samples.
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

// Binding names a field of a message and the data type of the channel it feeds.
type Binding struct {
	Field    icd.FieldKey
	DataType telem.DataType
}

// JSON decodes the documents of one JSON message into samples. It is not safe for
// concurrent use.
type JSON struct {
	fields    []jsonField
	extractor *xjson.Extractor
	// values holds the match value, when there is a match rule, then one value for
	// each field.
	values []jsontext.Value
	match  *jsonMatch
	// lengths holds the length of each sample slice before a Decode, to undo it.
	lengths []int
}

type jsonMatch struct {
	text   string
	number float64
	isText bool
	// unquoted is scratch space for the text of a string value.
	unquoted []byte
}

type jsonField struct {
	name    string
	pointer xjson.Pointer
	kind    payload.Kind
	// scaled is true when the field's scale or offset changes its value.
	scaled        bool
	scale, offset float64
	// number reads the JSON value of a scaled field as a float64.
	number *xjson.Decoder
	// scratch holds a scaled value between its conversions.
	scratch []byte
	// sample converts a JSON value to a sample of the channel's data type.
	sample *xjson.Decoder
	// fallback is the sample used when a document has no value, or nil when the
	// field requires one.
	fallback []byte
}

// NewJSON returns a JSON decoder for the fields of p that bindings name, in binding
// order. It returns an error wrapping validate.ErrValidation when a binding names no
// field of p, or a field's kind does not convert to its binding's data type.
func NewJSON(p icd.JSONPayload, bindings []Binding) (*JSON, error) {
	byKey := make(map[icd.FieldKey]icd.JSONField, len(p.Fields))
	for _, f := range p.Fields {
		byKey[f.Base().Key] = f
	}
	j := &JSON{fields: make([]jsonField, len(bindings))}
	pointers := make([]xjson.Pointer, 0, len(bindings)+1)
	if p.Match != nil {
		m, pointer, err := newJSONMatch(*p.Match)
		if err != nil {
			return nil, err
		}
		j.match = m
		pointers = append(pointers, pointer)
	}
	for i, b := range bindings {
		f, ok := byKey[b.Field]
		if !ok {
			return nil, errors.Wrapf(
				validate.ErrValidation, "field %s does not exist", b.Field,
			)
		}
		field, err := newJSONField(f, b.DataType)
		if err != nil {
			return nil, errors.Wrapf(err, "field %s", f.Base().Name)
		}
		j.fields[i] = field
		pointers = append(pointers, field.pointer)
	}
	j.extractor = xjson.NewExtractor(pointers)
	j.values = make([]jsontext.Value, len(pointers))
	j.lengths = make([]int, len(bindings))
	return j, nil
}

func newJSONMatch(m icd.JSONMatch) (*jsonMatch, xjson.Pointer, error) {
	pointer, err := xjson.ParsePointer(m.Pointer)
	if err != nil {
		return nil, nil, err
	}
	switch v := m.Value.(type) {
	case string:
		return &jsonMatch{text: v, isText: true}, pointer, nil
	case nil:
		return nil, nil, errors.Wrap(validate.ErrValidation, "match value is required")
	}
	n, ok := toFloat(m.Value)
	if !ok {
		return nil, nil, errors.Wrapf(
			validate.ErrValidation,
			"match value must be a string or a number, got %T",
			m.Value,
		)
	}
	return &jsonMatch{number: n}, pointer, nil
}

func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	}
	return 0, false
}

// converts reports whether a field of kind k converts to samples of dt. A number or
// boolean also converts to a float, and a boolean to any integer.
func converts(k payload.Kind, dt telem.DataType) bool {
	dk, ok := payload.KindOf(dt)
	if !ok {
		return false
	}
	if dk == k {
		return true
	}
	switch dk {
	case payload.KindFloat:
		return k == payload.KindInteger || k == payload.KindUnsigned ||
			k == payload.KindBoolean
	case payload.KindInteger:
		return k == payload.KindUnsigned || k == payload.KindBoolean
	case payload.KindUnsigned:
		return k == payload.KindBoolean
	}
	return false
}

func newJSONField(f icd.JSONField, dt telem.DataType) (jsonField, error) {
	b := f.Base()
	pointer, err := xjson.ParsePointer(b.Pointer)
	if err != nil {
		return jsonField{}, err
	}
	field := jsonField{
		name:    b.Name,
		pointer: pointer,
		scale:   b.Scale,
		offset:  b.Offset,
	}
	var (
		xf       = xjson.Field{DataType: dt}
		fallback jsontext.Value
	)
	switch v := f.Variant.(type) {
	case icd.FloatJSONField:
		field.kind = payload.KindFloat
		for _, e := range v.EnumValues {
			xf.Enums = append(xf.Enums, xjson.NewEnum(e.Label, float64(e.Value)))
		}
		if v.Default != nil {
			if fallback, err = appendFloat(nil, float64(*v.Default)); err != nil {
				return jsonField{}, err
			}
		}
	case icd.IntegerJSONField:
		field.kind = payload.KindInteger
		for _, e := range v.EnumValues {
			xf.Enums = append(xf.Enums, xjson.NewEnum(e.Label, int64(e.Value)))
		}
		if v.Default != nil {
			fallback = strconv.AppendInt(nil, int64(*v.Default), 10)
		}
	case icd.UnsignedJSONField:
		field.kind = payload.KindUnsigned
		for _, e := range v.EnumValues {
			xf.Enums = append(xf.Enums, xjson.NewEnum(e.Label, uint64(e.Value)))
		}
		if v.Default != nil {
			fallback = strconv.AppendUint(nil, uint64(*v.Default), 10)
		}
	case icd.StringJSONField:
		field.kind = payload.KindString
		for _, e := range v.EnumValues {
			xf.Enums = append(xf.Enums, xjson.NewEnum(e.Label, e.Value))
		}
		if v.Default != nil {
			if fallback, err = jsontext.AppendQuote(nil, *v.Default); err != nil {
				return jsonField{}, errors.Wrap(validate.ErrValidation, err.Error())
			}
		}
	case icd.BooleanJSONField:
		field.kind = payload.KindBoolean
		for _, e := range v.EnumValues {
			xf.Enums = append(xf.Enums, xjson.NewEnum(e.Label, boolSample(e.Value)))
		}
		if v.Default != nil {
			fallback = strconv.AppendBool(nil, *v.Default)
		}
	case icd.TimestampJSONField:
		field.kind = payload.KindTimestamp
		if xf.TimeFormat, err = xjson.ParseTimeFormat(
			string(v.TimeFormat),
		); err != nil {
			return jsonField{}, err
		}
		if v.Default != nil {
			field.fallback = telem.ByteOrder.AppendUint64(nil, uint64(*v.Default))
		}
	case icd.JSONJSONField:
		field.kind = payload.KindJSON
	case icd.UuidJSONField:
		field.kind = payload.KindUUID
	case icd.BytesJSONField:
		field.kind = payload.KindBytes
	default:
		return jsonField{}, errors.Wrap(validate.ErrValidation, "type is required")
	}
	if !converts(field.kind, dt) {
		return jsonField{}, errors.Wrapf(
			validate.ErrValidation,
			"a %s field cannot write to a %s channel",
			field.kind,
			dt,
		)
	}
	field.scaled = b.Scale != 1 || b.Offset != 0
	if field.scaled {
		if dt != telem.Float32T && dt != telem.Float64T {
			return jsonField{}, errors.Wrapf(
				validate.ErrValidation,
				"a scaled field requires a float channel, not %s",
				dt,
			)
		}
		numberField := xjson.Field{DataType: telem.Float64T, Enums: xf.Enums}
		if field.number, err = xjson.NewDecoder(numberField); err != nil {
			return jsonField{}, err
		}
		xf.Enums = nil
	}
	if field.sample, err = xjson.NewDecoder(xf); err != nil {
		return jsonField{}, err
	}
	if fallback != nil {
		if field.fallback, err = field.decode(nil, fallback); err != nil {
			return jsonField{}, errors.Wrap(err, "default")
		}
	}
	return field, nil
}

func boolSample(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}

// appendFloat appends v to dst as a JSON number.
func appendFloat(dst []byte, v float64) (jsontext.Value, error) {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return nil, errors.Wrapf(validate.ErrValidation, "default %v is not finite", v)
	}
	return strconv.AppendFloat(dst, v, 'g', -1, 64), nil
}

// decode appends the sample of raw to dst.
func (f *jsonField) decode(dst []byte, raw jsontext.Value) ([]byte, error) {
	if err := f.checkKind(raw); err != nil {
		return nil, err
	}
	if !f.scaled {
		return f.sample.Append(dst, raw)
	}
	var err error
	if f.scratch, err = f.number.Append(f.scratch[:0], raw); err != nil {
		return nil, err
	}
	v := math.Float64frombits(telem.ByteOrder.Uint64(f.scratch))*f.scale + f.offset
	if f.scratch, err = appendFloat(f.scratch[:0], v); err != nil {
		return nil, err
	}
	return f.sample.Append(dst, f.scratch)
}

// checkKind rejects a JSON number that the field's kind cannot hold, such as 1.5 for
// an integer field that writes to a float channel.
func (f *jsonField) checkKind(raw jsontext.Value) error {
	if raw.Kind() != '0' {
		return nil
	}
	switch f.kind {
	case payload.KindInteger, payload.KindUnsigned:
	case payload.KindBoolean:
		return errors.Wrapf(
			validate.ErrValidation, "expected a boolean, got number %s", raw,
		)
	default:
		return nil
	}
	n, err := strconv.ParseFloat(string(raw), 64)
	if err != nil {
		return errors.Wrap(validate.ErrValidation, err.Error())
	}
	if n != math.Trunc(n) {
		return errors.Wrapf(validate.ErrValidation, "%s is not an integer", raw)
	}
	if f.kind == payload.KindUnsigned && n < 0 {
		return errors.Wrapf(validate.ErrValidation, "%s is negative", raw)
	}
	return nil
}

// Decode appends one sample for each binding to samples, which holds one slice for
// each binding, in binding order. It returns
// false, appending nothing, when doc does not match the message. It returns an error,
// appending nothing, when doc is not JSON or a field's value does not convert.
func (j *JSON) Decode(doc []byte, samples [][]byte) (bool, error) {
	if err := j.extractor.Extract(doc, j.values); err != nil {
		return false, errors.Wrap(validate.ErrValidation, err.Error())
	}
	values := j.values
	if j.match != nil {
		if !j.match.matches(values[0]) {
			return false, nil
		}
		values = values[1:]
	}
	for i := range samples {
		j.lengths[i] = len(samples[i])
	}
	for i := range j.fields {
		f := &j.fields[i]
		var err error
		if values[i] == nil {
			if f.fallback == nil {
				err = errors.Wrapf(
					validate.ErrValidation, "no value at %s", f.pointer,
				)
			} else {
				samples[i] = append(samples[i], f.fallback...)
			}
		} else {
			var out []byte
			if out, err = f.decode(samples[i], values[i]); err == nil {
				samples[i] = out
			}
		}
		if err != nil {
			for k := range samples {
				samples[k] = samples[k][:j.lengths[k]]
			}
			return false, errors.Wrapf(err, "field %s", f.name)
		}
	}
	return true, nil
}

func (m *jsonMatch) matches(raw jsontext.Value) bool {
	switch raw.Kind() {
	case '"':
		if !m.isText {
			return false
		}
		var err error
		m.unquoted, err = jsontext.AppendUnquote(m.unquoted[:0], raw)
		return err == nil && string(m.unquoted) == m.text
	case '0':
		if m.isText {
			return false
		}
		n, err := strconv.ParseFloat(string(raw), 64)
		return err == nil && n == m.number
	}
	return false
}
