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
	"uuid"

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

// Failure is a field of a document that did not decode.
type Failure struct {
	// Binding is the index of the field's binding in its message.
	Binding int
	// Missing is true when the document has no value for the field, and the field has
	// no default.
	Missing bool
	// Err names the field, and also the message when the decoder holds more than one.
	Err error
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
	// values holds the values that the match rules test, then the value of each field
	// of each message, in message order.
	values []jsontext.Value
	// ruleValues is the number of values that the match rules test.
	ruleValues int
	messages   []jsonMessage
	router     *router
	// failures holds the failures of the last decode.
	failures []Failure
}

type jsonMessage struct {
	name   string
	fields []jsonField
	// values are the values of fields, a part of JSON.values.
	values []jsontext.Value
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
	// fallback is the sample used when a document has no value.
	fallback []byte
	// generate appends a new sample when a document has no value. A field with no
	// generate and no fallback requires a value.
	generate func(dst []byte) []byte
}

// NewJSON returns a JSON decoder for messages that share one source of documents. A
// document belongs to the message whose match rule it meets, or else to the message
// with no rule. NewJSON returns an error wrapping validate.ErrValidation when:
//   - two value rules test different pointers
//   - one document value can meet two value rules
//   - a present rule tests the pointer of another rule, or a value that holds it
//   - two messages have no match rule
//   - a binding names no field of its message
//   - a field's kind does not convert to its binding's data type
//   - a bound field's scale is zero
func NewJSON(messages []JSONMessage) (*JSON, error) {
	r, err := newRouter(messages)
	if err != nil {
		return nil, err
	}
	pointers := r.pointers()
	j := &JSON{
		messages:   make([]jsonMessage, len(messages)),
		router:     r,
		ruleValues: len(pointers),
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
	values := j.values[j.ruleValues:]
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
	msg := jsonMessage{name: m.Name, fields: make([]jsonField, len(m.Bindings))}
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
// boolean also converts to a float, and an unsigned number or boolean to any integer. A
// scaled field converts only to a float.
func Converts(k payload.Kind, dt telem.DataType, scaled bool) bool {
	dk, ok := payload.KindOf(dt)
	if !ok || (scaled && dk != payload.KindFloat) {
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
	if err := icd.CheckScale(b.Scale); err != nil {
		return jsonField{}, errors.Wrap(validate.ErrValidation, err.Error())
	}
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

// quotedDefault returns d as a JSON string, or nil when d is nil.
func quotedDefault(d *string) (jsontext.Value, error) {
	if d == nil {
		return nil, nil
	}
	v, err := jsontext.AppendQuote(nil, *d)
	if err != nil {
		return nil, errors.Wrap(validate.ErrValidation, err.Error())
	}
	return v, nil
}

// readTimestampDefault sets the fallback or generator of a timestamp field from d, if
// any.
func (f *jsonField) readTimestampDefault(d *icd.TimestampDefault) error {
	if d == nil {
		return nil
	}
	switch v := d.Variant.(type) {
	case icd.ConstantTimestampDefault:
		f.fallback = telem.ByteOrder.AppendUint64(nil, uint64(v.Value))
	case icd.NowTimestampDefault:
		f.generate = appendNow
	case nil:
		return errUntypedDefault
	}
	return nil
}

// readUUIDDefault sets the fallback or generator of a UUID field from d, if any.
func (f *jsonField) readUUIDDefault(d *icd.UUIDDefault) error {
	if d == nil {
		return nil
	}
	switch v := d.Variant.(type) {
	case icd.ConstantUUIDDefault:
		f.fallback = v.Value[:]
	case icd.RandomUUIDDefault:
		f.generate = appendRandomUUID
	case nil:
		return errUntypedDefault
	}
	return nil
}

// readVariant sets the kind of f from the variant of jf, and the fallback or generator
// of a timestamp or UUID field. It returns the rest of what the variant sets.
func (f *jsonField) readVariant(jf icd.JSONField) (jsonVariant, error) {
	out := jsonVariant{sample: xjson.Field{DataType: f.dataType, Enums: enumsOf(jf)}}
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
	case icd.BooleanJSONField:
		f.kind = payload.KindBoolean
		if v.Default != nil {
			out.fallback = strconv.AppendBool(nil, *v.Default)
		}
	default:
		if err := f.readTextVariant(jf, &out); err != nil {
			return jsonVariant{}, err
		}
	}
	return out, nil
}

// readTextVariant sets the kind of f and the fields of out from the variant of jf, a
// field whose values are not numbers or Booleans.
func (f *jsonField) readTextVariant(jf icd.JSONField, out *jsonVariant) error {
	var err error
	switch v := jf.Variant.(type) {
	case icd.StringJSONField:
		f.kind = payload.KindString
		out.fallback, err = quotedDefault(v.Default)
		return err
	case icd.TimestampJSONField:
		f.kind = payload.KindTimestamp
		format := string(v.TimeFormat)
		if out.sample.TimeFormat, err = xjson.ParseTimeFormat(format); err != nil {
			return err
		}
		return f.readTimestampDefault(v.Default)
	case icd.JSONJSONField:
		f.kind = payload.KindJSON
		if v.Default == nil {
			return nil
		}
		if out.fallback, err = jsonText(*v.Default); err != nil {
			return errors.Wrap(err, "default")
		}
		return nil
	case icd.UuidJSONField:
		f.kind = payload.KindUUID
		return f.readUUIDDefault(v.Default)
	case icd.BytesJSONField:
		f.kind = payload.KindBytes
		out.fallback, err = quotedDefault(v.Default)
		return err
	default:
		return errors.Wrap(validate.ErrValidation, "type is required")
	}
}

var (
	errUntyped        = errors.Wrap(validate.ErrValidation, "type is required")
	errUntypedDefault = errors.Wrap(errUntyped, "default")
)

func appendNow(dst []byte) []byte {
	return telem.ByteOrder.AppendUint64(dst, uint64(telem.Now()))
}

func appendRandomUUID(dst []byte) []byte {
	id := uuid.NewV4()
	return append(dst, id[:]...)
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
	if !Converts(f.kind, f.dataType, false) {
		return errors.Wrapf(
			validate.ErrValidation,
			"cannot write %s values to %s channels",
			f.kind,
			f.dataType,
		)
	}
	f.scaled = f.scale != 1 || f.offset != 0
	if !Converts(f.kind, f.dataType, f.scaled) {
		return errors.Wrapf(
			validate.ErrValidation,
			"a scaled field requires a float channel, not %s",
			f.dataType,
		)
	}
	var err error
	if f.scaled {
		// The enums of number fail a string that is no label as an unknown label. A
		// label itself goes to sample.
		numberField := xjson.Field{
			DataType: telem.Float64T,
			Enums:    v.sample.Enums,
			Range:    v.sample.Range,
		}
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
	boolean := f.kind == payload.KindBoolean
	if boolean && raw.Kind() == '0' {
		return nil, errors.Wrapf(
			validate.ErrValidation, "expected a boolean, got number %s", raw,
		)
	}
	if raw.Kind() == '"' && (boolean || f.scaled) {
		// An enum label maps straight to a channel value, which takes no scale.
		if f.sample.Labeled(raw) {
			return f.sample.Append(dst, raw)
		}
		if boolean {
			return nil, errors.Wrapf(
				validate.ErrValidation, "expected a boolean, got string %s", raw,
			)
		}
	}
	if !f.scaled {
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
// error, appending nothing, when doc is not JSON or meets two match rules.
//
// failures holds each field whose value does not convert, or that has no value and no
// default. A failed field appends no sample, and the other fields still append.
// failures is nil when every field decodes, and valid until the next Decode.
func (j *JSON) Decode(
	doc []byte,
	samples [][][]byte,
) (message int, ok bool, failures []Failure, err error) {
	if err := j.extractor.Extract(doc, j.values); err != nil {
		return 0, false, nil, errors.Wrap(validate.ErrValidation, err.Error())
	}
	if message, ok, err = j.router.find(j.values[:j.ruleValues]); err != nil || !ok {
		return 0, false, nil, err
	}
	m := &j.messages[message]
	if j.failures = m.decode(samples[message], j.failures[:0]); len(j.failures) == 0 {
		return message, true, nil, nil
	}
	if len(j.messages) > 1 {
		for i := range j.failures {
			j.failures[i].Err = errors.Wrapf(j.failures[i].Err, "message %s", m.name)
		}
	}
	return message, true, j.failures, nil
}

// decode appends the sample of each field to samples, and each failed field to
// failures.
func (m *jsonMessage) decode(samples [][]byte, failures []Failure) []Failure {
	for i := range m.fields {
		f := &m.fields[i]
		var err error
		missing := false
		if payload.Absent(m.values[i], f.dataType) {
			switch {
			case f.generate != nil:
				samples[i] = f.generate(samples[i])
			case f.fallback != nil:
				samples[i] = append(samples[i], f.fallback...)
			default:
				missing = true
				err = errors.Wrapf(
					validate.ErrValidation, "no value at %s", f.pointer,
				)
			}
		} else {
			var out []byte
			if out, err = f.decode(samples[i], m.values[i]); err == nil {
				samples[i] = out
			}
		}
		if err != nil {
			failures = append(failures, Failure{
				Binding: i,
				Missing: missing,
				Err:     errors.Wrapf(err, "field %s", f.name),
			})
		}
	}
	return failures
}

// jsonText returns the exact bytes of text. It returns a validation error when text is
// not valid JSON.
func jsonText(text string) ([]byte, error) {
	if !jsontext.Value(text).IsValid() {
		return nil, errors.Wrap(validate.ErrValidation, "not valid JSON")
	}
	return []byte(text), nil
}
