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
	"encoding/binary"
	"math"
	"strconv"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	xmath "github.com/synnaxlabs/x/math"
	"github.com/synnaxlabs/x/payload"
	xstrconv "github.com/synnaxlabs/x/strconv"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// TextEncoder encodes samples into the line of one text message. It is not safe for
// concurrent use.
type TextEncoder struct {
	name      string
	delimiter []byte
	fields    []textEncodeField
	items     []textItem
	// tagged holds the index in fields of each tagged field, in field order.
	tagged []int
	// bound holds the index in fields of the field of each binding.
	bound []int
	// checked is scratch space for the text of a sample that Check writes.
	checked []byte
}

// textItem is one item of a line before its tagged fields.
type textItem struct {
	// text is the part of the prefix that fills the item.
	text []byte
	// field is the index in TextEncoder.fields of the delimited field at the item, or
	// -1 for an item that holds only text.
	field int
}

type textEncodeField struct {
	base icd.BaseField
	// tag is the text before the value of a tagged field, or nil for a delimited field.
	tag []byte
	// scaled is true when the field's scale or offset changes its value.
	scaled bool
	// kind is the kind of the samples of the field's channel.
	kind payload.Kind
	// narrow is true for float32 samples.
	narrow bool
	// binding is the index of the field's sample, or -1 for a field that takes its
	// default.
	binding int
	// fallback is the default of the field when defaulted is true.
	fallback  float64
	defaulted bool
}

// maxPosition is the last position of a delimited field.
const maxPosition = 65535

// NewTextEncoder returns an encoder of the lines of m. Each binding names a field that
// takes a sample, in sample order, and every other field takes its default. It returns
// an error wrapping validate.ErrValidation when the delimiter is empty, a binding names
// no field, a field has two bindings, a tag is empty, contains the delimiter, or starts
// with whitespace, two fields have the same position, a position is past 65535, the
// prefix fills the position of a field, a default is not finite, a bound field has a
// zero scale, a field's channel does not hold numbers, a scaled field's channel does
// not hold floats, or a field has no binding and no default.
func NewTextEncoder(m TextMessage) (*TextEncoder, error) {
	e, err := newTextEncoder(m)
	if err != nil {
		return nil, errors.Wrapf(err, "message %s", m.Name)
	}
	return e, nil
}

func newTextEncoder(m TextMessage) (*TextEncoder, error) {
	delimiter := m.Payload.Delimiter
	if delimiter == "" {
		return nil, errors.Wrap(validate.ErrValidation, "the delimiter is empty")
	}
	bound, err := bindingIndexes(m.Payload.Fields, textBase, m.Bindings)
	if err != nil {
		return nil, err
	}
	e := &TextEncoder{
		name:      m.Name,
		delimiter: []byte(delimiter),
		fields:    make([]textEncodeField, len(m.Payload.Fields)),
		bound:     make([]int, len(m.Bindings)),
	}
	// fixed is the number of items that the prefix fills.
	fixed := 0
	if prefix := m.Payload.Prefix; prefix != "" {
		for item := range strings.SplitSeq(prefix, delimiter) {
			e.items = append(e.items, textItem{text: []byte(item), field: -1})
		}
		fixed = len(e.items)
		if len(e.items[fixed-1].text) == 0 {
			fixed--
		}
	}
	for i, f := range m.Payload.Fields {
		b := f.Base()
		binding, ok := bound[b.Key]
		var dt telem.DataType
		if ok {
			dt = m.Bindings[binding].DataType
			e.bound[binding] = i
		} else {
			binding = -1
		}
		if e.fields[i], err = newTextEncodeField(
			f,
			binding,
			dt,
			delimiter,
		); err != nil {
			return nil, errors.Wrapf(err, "field %s", b.Name)
		}
		switch v := f.Variant.(type) {
		case icd.TaggedTextField:
			e.tagged = append(e.tagged, i)
		case icd.DelimitedTextField:
			if err = e.place(i, v.Position, fixed); err != nil {
				return nil, err
			}
		default:
			panic(errors.Newf("codec: unknown text field %T", v))
		}
	}
	// The prefix ends in a delimiter, and the tagged fields fill the item after it.
	if n := len(e.items); len(e.tagged) > 0 && n > 0 &&
		e.items[n-1].field < 0 && len(e.items[n-1].text) == 0 {
		e.items = e.items[:n-1]
	}
	return e, nil
}

func textBase(f icd.TextField) icd.BaseField { return f.Base().BaseField }

// place puts field i, a delimited field, at position. fixed is the number of items that
// the prefix fills.
func (e *TextEncoder) place(i int, position uint32, fixed int) error {
	name := e.fields[i].base.Name
	switch {
	case position > maxPosition:
		return errors.Wrapf(
			validate.ErrValidation,
			"field %s: position %d is past the last position %d",
			name, position, maxPosition,
		)
	case int(position) < fixed:
		return errors.Wrapf(
			validate.ErrValidation,
			"field %s: the prefix fills position %d", name, position,
		)
	}
	for len(e.items) <= int(position) {
		e.items = append(e.items, textItem{field: -1})
	}
	item := &e.items[position]
	if item.field >= 0 {
		return errors.Wrapf(
			validate.ErrValidation,
			"fields %s and %s both have position %d",
			e.fields[item.field].base.Name, name, position,
		)
	}
	item.field = i
	return nil
}

// newTextEncodeField returns the field that writes f. binding is the index of the
// sample of f, of data type dt, or -1 when f takes its default.
func newTextEncodeField(
	f icd.TextField,
	binding int,
	dt telem.DataType,
	delimiter string,
) (textEncodeField, error) {
	b := f.Base()
	field := textEncodeField{
		base:    b.BaseField,
		scaled:  b.Scale != 1 || b.Offset != 0,
		binding: binding,
	}
	if t, ok := f.Variant.(icd.TaggedTextField); ok {
		if err := checkTag(t.Tag, delimiter); err != nil {
			return textEncodeField{}, err
		}
		field.tag = []byte(t.Tag)
	}
	if b.Default != nil {
		d := float64(*b.Default)
		if !xmath.Finite(d) {
			return textEncodeField{}, errors.Wrapf(
				validate.ErrValidation,
				"default: %s is not finite",
				xstrconv.FormatFloat(d, 64),
			)
		}
		field.fallback, field.defaulted = d, true
	}
	if binding < 0 {
		if field.defaulted {
			return field, nil
		}
		return textEncodeField{}, errors.Wrap(
			validate.ErrValidation,
			"needs a channel or a default",
		)
	}
	if err := icd.CheckScale(b.Scale); err != nil {
		return textEncodeField{}, errors.Wrap(validate.ErrValidation, err.Error())
	}
	if err := checkWrite(payload.KindFloat, field.scaled, dt); err != nil {
		return textEncodeField{}, err
	}
	field.kind, _ = payload.KindOf(dt)
	field.narrow = dt == telem.Float32T
	return field, nil
}

// Encode appends the line of samples to dst. The line starts with the prefix and holds
// an item for each position up to the last delimited field, with the prefix in the
// first items and an empty item where no field is. A tagged field comes after them as
// its tag and value. A value is the shortest decimal text that reads back as the sample
// after its scale and offset come off. A float32 sample counts as its shortest decimal
// text, so it gains no float64 digits. samples holds one sample for each binding, in
// binding order, and a nil sample writes the default of its field. Encode does not
// check the min and max of a field. It returns an error wrapping validate.ErrValidation
// when a field has no sample and no default, or a value is not finite.
func (e *TextEncoder) Encode(dst []byte, samples [][]byte) ([]byte, error) {
	var err error
	for i, item := range e.items {
		if i > 0 {
			dst = append(dst, e.delimiter...)
		}
		if item.field < 0 {
			dst = append(dst, item.text...)
		} else if dst, err = e.append(dst, item.field, samples); err != nil {
			return nil, err
		}
	}
	for i, field := range e.tagged {
		if i > 0 || len(e.items) > 0 {
			dst = append(dst, e.delimiter...)
		}
		dst = append(dst, e.fields[field].tag...)
		if dst, err = e.append(dst, field, samples); err != nil {
			return nil, err
		}
	}
	return dst, nil
}

// append appends the value of field i in samples, or its default, to dst.
func (e *TextEncoder) append(dst []byte, i int, samples [][]byte) ([]byte, error) {
	f := &e.fields[i]
	if f.binding >= 0 && samples[f.binding] != nil {
		out, err := f.append(dst, samples[f.binding])
		if err != nil {
			return nil, errors.Wrapf(err, "message %s: field %s", e.name, f.base.Name)
		}
		return out, nil
	}
	if !f.defaulted {
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"message %s: field %s: no value and no default",
			e.name, f.base.Name,
		)
	}
	return appendDecimal(dst, f.fallback), nil
}

// Ready reports whether each field has a sample or a default, so Encode has a value for
// it. samples is as in Encode.
func (e *TextEncoder) Ready(samples [][]byte) bool {
	for i, s := range samples {
		if s == nil && !e.fields[e.bound[i]].defaulted {
			return false
		}
	}
	return true
}

// Check returns the error that Encode returns when binding i holds sample, or an error
// wrapping validate.ErrValidation when sample is outside the min and max of the field
// of binding i. A float32 sample that equals a limit at float32 precision is at that
// limit.
func (e *TextEncoder) Check(i int, sample []byte) error {
	f := &e.fields[e.bound[i]]
	out, err := f.append(e.checked[:0], sample)
	if err == nil {
		e.checked = out
		err = icd.CheckRange(f.base, atLimit(f.base, f.narrow, f.float(sample)))
	}
	if err != nil {
		return errors.Wrapf(err, "message %s: field %s", e.name, f.base.Name)
	}
	return nil
}

// append appends the text of sample to dst.
func (f *textEncodeField) append(dst, sample []byte) ([]byte, error) {
	switch f.kind {
	case payload.KindInteger:
		return strconv.AppendInt(dst, signed(sample), 10), nil
	case payload.KindUnsigned, payload.KindBoolean:
		return strconv.AppendUint(dst, unsigned(sample), 10), nil
	case payload.KindFloat:
		return f.appendFloat(dst, f.float(sample))
	case payload.KindString, payload.KindTimestamp, payload.KindJSON,
		payload.KindUUID, payload.KindBytes:
	}
	panic(errors.Newf("codec: %s channels never write text fields", f.kind))
}

// appendFloat appends the text of v, a float sample, to dst.
func (f *textEncodeField) appendFloat(dst []byte, v float64) ([]byte, error) {
	if f.narrow {
		v = widen(float32(v))
	}
	if f.scaled {
		v = (v - f.base.Offset) / f.base.Scale
	}
	if !xmath.Finite(v) {
		return nil, errors.Wrapf(
			validate.ErrValidation, "%s is not finite", xstrconv.FormatFloat(v, 64),
		)
	}
	return appendDecimal(dst, v), nil
}

// float returns the value of sample as a float64.
func (f *textEncodeField) float(sample []byte) float64 {
	switch {
	case f.kind == payload.KindInteger:
		return float64(signed(sample))
	case f.kind != payload.KindFloat:
		return float64(unsigned(sample))
	case f.narrow:
		return float64(math.Float32frombits(binary.LittleEndian.Uint32(sample)))
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(sample))
}

// signed returns the value of sample, a little-endian signed integer of its length.
func signed(sample []byte) int64 {
	switch len(sample) {
	case 1:
		return int64(int8(sample[0]))
	case 2:
		return int64(int16(binary.LittleEndian.Uint16(sample)))
	case 4:
		return int64(int32(binary.LittleEndian.Uint32(sample)))
	}
	return int64(binary.LittleEndian.Uint64(sample))
}

// unsigned returns the value of sample, a little-endian unsigned integer of its length.
func unsigned(sample []byte) uint64 {
	switch len(sample) {
	case 1:
		return uint64(sample[0])
	case 2:
		return uint64(binary.LittleEndian.Uint16(sample))
	case 4:
		return uint64(binary.LittleEndian.Uint32(sample))
	}
	return binary.LittleEndian.Uint64(sample)
}

// widen returns the float64 of the shortest decimal text of v, so that a float32 sample
// gains no float64 digits. A value that is not finite stays as it is.
func widen(v float32) float64 {
	if !xmath.Finite(float64(v)) {
		return float64(v)
	}
	var buf [32]byte
	w, err := xjson.ParseFloat64(strconv.AppendFloat(buf[:0], float64(v), 'g', -1, 32))
	if err != nil {
		panic(err)
	}
	return w
}

// appendDecimal appends the shortest decimal text that reads back as v, a finite value,
// to dst. The text is in fixed or exponent notation, whichever is shorter, and fixed on
// a tie, as C++ std::to_chars writes it.
func appendDecimal(dst []byte, v float64) []byte {
	start := len(dst)
	dst = strconv.AppendFloat(dst, v, 'f', -1, 64)
	fixed := len(dst) - start
	dst = strconv.AppendFloat(dst, v, 'e', -1, 64)
	if len(dst)-start-fixed < fixed {
		return append(dst[:start], dst[start+fixed:]...)
	}
	return dst[:start+fixed]
}
