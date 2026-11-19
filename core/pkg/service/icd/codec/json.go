// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package codec converts between the payloads that ICD messages describe and channel
// samples.
package codec

import (
	"encoding/json/jsontext"
	"math"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Binding names a field of a message and the data type of the channel bound to it. A
// decode writes the field to the channel, and an encode the channel to the field.
type Binding struct {
	// Field is the key of the message field.
	Field icd.FieldKey
	// DataType is the data type of the channel.
	DataType telem.DataType
}

// JSONMessage is a JSON message and the fields bound to channels.
type JSONMessage struct {
	// Name identifies the message in errors.
	Name string
	// Payload is the layout of the message's documents.
	Payload icd.JSONPayload
	// Bindings name the bound fields, in sample order.
	Bindings []Binding
}

// JSON decodes documents that each belong to one of several JSON messages into samples.
// It is not safe for concurrent use.
type JSON struct {
	extractor *xjson.Extractor
	// values holds the match value, when a message has a match rule, then the value of
	// each field of each message, in message order.
	values   []jsontext.Value
	messages []jsonMessage
	router   *router
}

type jsonMessage struct {
	name   string
	fields []jsonField
	// values are the values of fields, a part of JSON.values.
	values []jsontext.Value
	// lengths holds the length of each sample slice before a decode, to undo it.
	lengths []int
}

type jsonField struct {
	name     string
	pointer  xjson.Pointer
	kind     payload.Kind
	dataType telem.DataType
	// scaled is true when the field's scale or offset changes its value.
	scaled        bool
	scale, offset float64
	// number reads the JSON number of a scaled field as a float64.
	number *xjson.Decoder
	// scratch holds the float64 sample that number reads.
	scratch []byte
	// sample converts a JSON value to a sample of the channel's data type.
	sample *xjson.Decoder
	// fallback is the sample used when a document has no value, or nil when the field
	// requires one.
	fallback []byte
}

// NewJSON returns a JSON decoder for messages that share one source of documents. A
// document belongs to the message whose match rule it meets, or else to the message
// with no rule. NewJSON returns an error wrapping validate.ErrValidation when:
//   - two match rules test different pointers
//   - one document value can meet two match rules
//   - two messages have no match rule
//   - a binding names no field of its message
//   - a field's kind does not convert to its binding's data type
func NewJSON(messages []JSONMessage) (*JSON, error) {
	r, err := newRouter(messages)
	if err != nil {
		return nil, err
	}
	j := &JSON{messages: make([]jsonMessage, len(messages)), router: r}
	var pointers []xjson.Pointer
	if r.ruled {
		pointers = append(pointers, r.pointer)
	}
	for i, m := range messages {
		if j.messages[i], err = newJSONMessage(m); err != nil {
			return nil, errors.Wrapf(err, "message %s", m.Name)
		}
		for _, f := range j.messages[i].fields {
			pointers = append(pointers, f.pointer)
		}
	}
	j.extractor = xjson.NewExtractor(pointers)
	j.values = make([]jsontext.Value, len(pointers))
	values := j.values
	if r.ruled {
		values = values[1:]
	}
	for i := range j.messages {
		m := &j.messages[i]
		m.values, values = values[:len(m.fields)], values[len(m.fields):]
	}
	return j, nil
}

func newJSONMessage(m JSONMessage) (jsonMessage, error) {
	byKey := make(map[icd.FieldKey]icd.JSONField, len(m.Payload.Fields))
	for _, f := range m.Payload.Fields {
		byKey[f.Base().Key] = f
	}
	msg := jsonMessage{
		name:    m.Name,
		fields:  make([]jsonField, len(m.Bindings)),
		lengths: make([]int, len(m.Bindings)),
	}
	for i, b := range m.Bindings {
		f, ok := byKey[b.Field]
		if !ok {
			return jsonMessage{}, errors.Wrapf(
				validate.ErrValidation, "field %s does not exist", b.Field,
			)
		}
		field, err := newJSONField(f, b.DataType)
		if err != nil {
			return jsonMessage{}, errors.Wrapf(err, "field %s", f.Base().Name)
		}
		msg.fields[i] = field
	}
	return msg, nil
}

// Converts reports whether a field of kind k converts to samples of dt. A number or
// boolean also converts to a float, and an unsigned number or boolean to any integer.
func Converts(k payload.Kind, dt telem.DataType) bool {
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
	case payload.KindString, payload.KindBoolean, payload.KindTimestamp,
		payload.KindJSON, payload.KindUUID, payload.KindBytes:
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
		name:     b.Name,
		pointer:  pointer,
		dataType: dt,
		scale:    b.Scale,
		offset:   b.Offset,
	}
	v, err := field.readVariant(f)
	if err != nil {
		return jsonField{}, err
	}
	if err = field.compile(v); err != nil {
		return jsonField{}, err
	}
	return field, nil
}

// jsonVariant is what the variant of a JSON field sets: the options of its samples and
// its default.
type jsonVariant struct {
	sample xjson.Field
	// fallback is the default of a field that is not a float, as a JSON value.
	fallback     jsontext.Value
	floatDefault *payload.Float
}

// readVariant sets the kind of f from the variant of jf, and the fallback of a
// timestamp field. It returns the rest of what the variant sets.
func (f *jsonField) readVariant(jf icd.JSONField) (jsonVariant, error) {
	out := jsonVariant{sample: xjson.Field{DataType: f.dataType, Enums: enumsOf(jf)}}
	var err error
	switch v := jf.Variant.(type) {
	case icd.FloatJSONField:
		f.kind = payload.KindFloat
		out.floatDefault = v.Default
	case icd.IntegerJSONField:
		f.kind, out.sample.Range = payload.KindInteger, telem.Int64T
		if v.Default != nil {
			out.fallback = strconv.AppendInt(nil, int64(*v.Default), 10)
		}
	case icd.UnsignedJSONField:
		f.kind, out.sample.Range = payload.KindUnsigned, telem.Uint64T
		if v.Default != nil {
			out.fallback = strconv.AppendUint(nil, uint64(*v.Default), 10)
		}
	case icd.StringJSONField:
		f.kind = payload.KindString
		if v.Default != nil {
			if out.fallback, err = jsontext.AppendQuote(nil, *v.Default); err != nil {
				return jsonVariant{}, errors.Wrap(validate.ErrValidation, err.Error())
			}
		}
	case icd.BooleanJSONField:
		f.kind = payload.KindBoolean
		if v.Default != nil {
			out.fallback = strconv.AppendBool(nil, *v.Default)
		}
	case icd.TimestampJSONField:
		f.kind = payload.KindTimestamp
		format := string(v.TimeFormat)
		if out.sample.TimeFormat, err = xjson.ParseTimeFormat(format); err != nil {
			return jsonVariant{}, err
		}
		if v.Default != nil {
			f.fallback = telem.ByteOrder.AppendUint64(nil, uint64(*v.Default))
		}
	case icd.JSONJSONField:
		f.kind = payload.KindJSON
	case icd.UuidJSONField:
		f.kind = payload.KindUUID
	case icd.BytesJSONField:
		f.kind = payload.KindBytes
	default:
		return jsonVariant{}, errors.Wrap(validate.ErrValidation, "type is required")
	}
	return out, nil
}

// enumsOf returns the enum labels of jf.
func enumsOf(jf icd.JSONField) []xjson.Enum {
	switch v := jf.Variant.(type) {
	case icd.FloatJSONField:
		return lo.Map(v.EnumValues, func(e payload.FloatEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, float64(e.Value))
		})
	case icd.IntegerJSONField:
		return lo.Map(v.EnumValues, func(e payload.IntegerEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, int64(e.Value))
		})
	case icd.UnsignedJSONField:
		return lo.Map(v.EnumValues, func(e payload.UnsignedEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, uint64(e.Value))
		})
	case icd.StringJSONField:
		return lo.Map(v.EnumValues, func(e payload.StringEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, e.Value)
		})
	case icd.BooleanJSONField:
		return lo.Map(v.EnumValues, func(e payload.BooleanEntry, _ int) xjson.Enum {
			return xjson.NewEnum(e.Label, boolSample(e.Value))
		})
	}
	return nil
}

// compile checks that the kind of f converts to its data type, then builds the decoders
// of f from v and sets its fallback from the default of v.
func (f *jsonField) compile(v jsonVariant) error {
	if !Converts(f.kind, f.dataType) {
		return errors.Wrapf(
			validate.ErrValidation,
			"cannot write %s values to %s channels",
			f.kind,
			f.dataType,
		)
	}
	var err error
	f.scaled = f.scale != 1 || f.offset != 0
	if f.scaled {
		if f.dataType != telem.Float32T && f.dataType != telem.Float64T {
			return errors.Wrapf(
				validate.ErrValidation,
				"a scaled field requires a float channel, not %s",
				f.dataType,
			)
		}
		numberField := xjson.Field{DataType: telem.Float64T, Range: v.sample.Range}
		if f.number, err = xjson.NewDecoder(numberField); err != nil {
			return err
		}
		// A scaled value need not be an integer.
		v.sample.Range = ""
	}
	if f.sample, err = xjson.NewDecoder(v.sample); err != nil {
		return err
	}
	switch {
	case v.floatDefault != nil:
		f.fallback, err = f.appendFloat(nil, float64(*v.floatDefault))
	case v.fallback != nil:
		f.fallback, err = f.decode(nil, v.fallback)
	}
	if err != nil {
		return errors.Wrap(err, "default")
	}
	return nil
}

func boolSample(v bool) uint8 {
	if v {
		return 1
	}
	return 0
}

// decode appends the sample of raw to dst.
func (f *jsonField) decode(dst []byte, raw jsontext.Value) ([]byte, error) {
	if f.kind == payload.KindBoolean && raw.Kind() == '0' {
		return nil, errors.Wrapf(
			validate.ErrValidation, "expected a boolean, got number %s", raw,
		)
	}
	// An enum label maps straight to a channel value, which takes no scale.
	if !f.scaled || raw.Kind() == '"' {
		return f.sample.Append(dst, raw)
	}
	var err error
	if f.scratch, err = f.number.Append(f.scratch[:0], raw); err != nil {
		return nil, err
	}
	return f.appendFloat(dst, math.Float64frombits(telem.ByteOrder.Uint64(f.scratch)))
}

// appendFloat appends the sample of v to dst, scaled when the field scales.
func (f *jsonField) appendFloat(dst []byte, v float64) ([]byte, error) {
	if f.scaled {
		v = v*f.scale + f.offset
	}
	return f.sample.AppendFloat(dst, v)
}

// Decode finds the message that doc belongs to and appends one sample for each of its
// bindings to samples[message]. samples holds an entry for each message in NewJSON
// order, and each entry holds one slice for each binding, in binding order. ok is
// false, and Decode appends nothing, when doc belongs to no message. Decode returns an
// error, appending nothing, when doc is not JSON or a value does not convert. The error
// names the field, and also the message when the decoder holds more than one.
func (j *JSON) Decode(
	doc []byte,
	samples [][][]byte,
) (message int, ok bool, err error) {
	if err := j.extractor.Extract(doc, j.values); err != nil {
		return 0, false, errors.Wrap(validate.ErrValidation, err.Error())
	}
	var match jsontext.Value
	if j.router.ruled {
		match = j.values[0]
	}
	if message, ok, err = j.router.find(match); err != nil || !ok {
		return 0, false, err
	}
	m := &j.messages[message]
	if err := m.decode(samples[message]); err != nil {
		if len(j.messages) > 1 {
			err = errors.Wrapf(err, "message %s", m.name)
		}
		return 0, false, err
	}
	return message, true, nil
}

func (m *jsonMessage) decode(samples [][]byte) error {
	for i := range samples {
		m.lengths[i] = len(samples[i])
	}
	for i := range m.fields {
		f := &m.fields[i]
		var err error
		if payload.Absent(m.values[i], f.dataType) {
			if f.fallback == nil {
				err = errors.Wrapf(
					validate.ErrValidation, "no value at %s", f.pointer,
				)
			} else {
				samples[i] = append(samples[i], f.fallback...)
			}
		} else {
			var out []byte
			if out, err = f.decode(samples[i], m.values[i]); err == nil {
				samples[i] = out
			}
		}
		if err != nil {
			for k := range samples {
				samples[k] = samples[k][:m.lengths[k]]
			}
			return errors.Wrapf(err, "field %s", f.name)
		}
	}
	return nil
}
