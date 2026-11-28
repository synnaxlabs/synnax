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
	"bytes"
	"cmp"
	"slices"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// TextMessage is a text message and the fields bound to channels.
type TextMessage struct {
	// Name identifies the message in errors.
	Name string
	// Payload is the layout of the message's lines.
	Payload icd.TextPayload
	// Bindings name the bound fields, in sample order.
	Bindings []Binding
}

// Text decodes lines that each belong to one of several text messages into samples. It
// is not safe for concurrent use.
type Text struct {
	messages []textMessage
	// prefixed holds the messages with a prefix, longest prefix first.
	prefixed []prefixedMessage
	// fallback is the index of the message with no prefix, or -1.
	fallback int
	// items is scratch space for the items of a line.
	items [][]byte
	// failures holds the failures of the last decode.
	failures []Failure
}

type prefixedMessage struct {
	prefix  []byte
	message int
}

type textMessage struct {
	name      string
	delimiter []byte
	fields    []textField
}

type textField struct {
	name   string
	tagged bool
	// tag is the text before the value of a tagged field.
	tag []byte
	// longer holds the tags of the message that start with tag and are longer, whose
	// items belong to their own fields.
	longer [][]byte
	// position is the item index of a delimited field.
	position int
	// scaled is true when the field's scale or offset changes its value.
	scaled        bool
	scale, offset float64
	sample        *xjson.Decoder
	// fallback is the sample used when a line has no value, or nil when the field
	// requires one.
	fallback []byte
}

// trimmed are the characters that a value or tagged item loses from each end.
const trimmed = " \t\r"

// NewText returns a Text decoder for messages that share one source of lines. A line
// belongs to the message with the longest prefix that starts it, or else to the
// message with no prefix. NewText returns an error wrapping validate.ErrValidation
// when two messages have the same prefix or no prefix, a delimiter or tag is empty, a
// tag contains its delimiter or starts with whitespace, a binding names no field of
// its message, a bound field has a zero scale, a default does not fit its channel, or
// a field does not convert to the data type of its binding.
func NewText(messages []TextMessage) (*Text, error) {
	t := &Text{messages: make([]textMessage, len(messages)), fallback: -1}
	for i, m := range messages {
		var err error
		if t.messages[i], err = newTextMessage(m); err != nil {
			return nil, errors.Wrapf(err, "message %s", m.Name)
		}
		if err = t.route(m.Name, m.Payload.Prefix, i); err != nil {
			return nil, err
		}
	}
	slices.SortStableFunc(t.prefixed, func(a, b prefixedMessage) int {
		return cmp.Compare(len(b.prefix), len(a.prefix))
	})
	return t, nil
}

func textKey(f icd.TextField) icd.FieldKey { return f.Base().Key }

func newTextMessage(m TextMessage) (textMessage, error) {
	delimiter := m.Payload.Delimiter
	if delimiter == "" {
		return textMessage{}, errors.Wrap(
			validate.ErrValidation,
			"the delimiter is empty",
		)
	}
	fields, err := boundFields(m.Payload.Fields, textKey, m.Bindings)
	if err != nil {
		return textMessage{}, err
	}
	msg := textMessage{
		name:      m.Name,
		delimiter: []byte(delimiter),
		fields:    make([]textField, len(fields)),
	}
	for i, f := range fields {
		field, err := newTextField(f, m.Bindings[i].DataType, delimiter)
		if err != nil {
			return textMessage{}, errors.Wrapf(err, "field %s", f.Base().Name)
		}
		if field.tagged {
			for _, other := range m.Payload.Fields {
				o, ok := other.Variant.(icd.TaggedTextField)
				if ok && len(o.Tag) > len(field.tag) &&
					strings.HasPrefix(o.Tag, string(field.tag)) {
					field.longer = append(field.longer, []byte(o.Tag))
				}
			}
		}
		msg.fields[i] = field
	}
	return msg, nil
}

func newTextField(
	f icd.TextField,
	dt telem.DataType,
	delimiter string,
) (textField, error) {
	b := f.Base()
	if err := icd.CheckScale(b.Scale); err != nil {
		return textField{}, errors.Wrap(validate.ErrValidation, err.Error())
	}
	field := textField{
		name:   b.Name,
		scaled: b.Scale != 1 || b.Offset != 0,
		scale:  b.Scale,
		offset: b.Offset,
	}
	switch v := f.Variant.(type) {
	case icd.TaggedTextField:
		if err := checkTag(v.Tag, delimiter); err != nil {
			return textField{}, err
		}
		field.tagged, field.tag = true, []byte(v.Tag)
	case icd.DelimitedTextField:
		field.position = int(v.Position)
	default:
		panic(errors.Newf("codec: unknown text field %T", v))
	}
	if !Converts(payload.KindFloat, dt, false) {
		return textField{}, errors.Wrapf(
			validate.ErrValidation,
			"cannot write %s values to %s channels",
			payload.KindFloat,
			dt,
		)
	}
	var err error
	if field.sample, err = xjson.NewDecoder(xjson.Field{DataType: dt}); err != nil {
		return textField{}, err
	}
	if b.Default == nil {
		return field, nil
	}
	if field.fallback, err = field.appendFloat(nil, float64(*b.Default)); err != nil {
		return textField{}, errors.Wrap(err, "default")
	}
	return field, nil
}

// checkTag returns an error wrapping validate.ErrValidation when tag is empty,
// contains delimiter, or starts with whitespace.
func checkTag(tag, delimiter string) error {
	switch {
	case tag == "":
		return errors.Wrap(validate.ErrValidation, "the tag is empty")
	case strings.Contains(tag, delimiter):
		return errors.Wrapf(
			validate.ErrValidation, "the tag contains the delimiter %q", delimiter,
		)
	case strings.ContainsRune(trimmed, rune(tag[0])):
		return errors.Wrap(validate.ErrValidation, "the tag starts with whitespace")
	}
	return nil
}

// route sends the lines that start with prefix to the message at index i. It returns
// an error wrapping validate.ErrValidation when an earlier message has the same
// prefix.
func (t *Text) route(name, prefix string, i int) error {
	if prefix == "" {
		if t.fallback >= 0 {
			return errors.Wrapf(
				validate.ErrValidation,
				"messages %s and %s both have no prefix",
				t.messages[t.fallback].name,
				name,
			)
		}
		t.fallback = i
		return nil
	}
	for _, p := range t.prefixed {
		if string(p.prefix) == prefix {
			return errors.Wrapf(
				validate.ErrValidation,
				"messages %s and %s have the same prefix %q",
				t.messages[p.message].name,
				name,
				prefix,
			)
		}
	}
	t.prefixed = append(t.prefixed, prefixedMessage{prefix: []byte(prefix), message: i})
	return nil
}

// Decode finds the message that line belongs to and appends one sample for each of its
// bindings to samples[message]. samples is as in JSON.Decode. The line splits into
// items on the delimiter of its message. A delimited field reads the item at its
// position, and a tagged field reads the rest of the first item whose longest tag of
// the message is its tag. A value is a decimal number, with any spaces, tabs, or
// carriage returns around it. An empty or missing item has no value. ok is false, and
// Decode appends nothing, when line belongs to no message. A line of only spaces, tabs,
// and carriage returns belongs to none.
//
// failures holds each field with no value and no default, whose value is not a number,
// or whose value does not fit its channel. A failed field appends no sample, and the
// other fields still append. A failure names the field, and also the message when the
// decoder holds more than one. failures is nil when every field decodes, and valid
// until the next Decode.
func (t *Text) Decode(
	line []byte,
	samples [][][]byte,
) (message int, ok bool, failures []Failure) {
	if len(bytes.Trim(line, trimmed)) == 0 {
		return 0, false, nil
	}
	message = t.fallback
	for _, p := range t.prefixed {
		if bytes.HasPrefix(line, p.prefix) {
			message = p.message
			break
		}
	}
	if message < 0 {
		return 0, false, nil
	}
	m := &t.messages[message]
	t.items = t.items[:0]
	for rest := line; ; {
		end := bytes.Index(rest, m.delimiter)
		if end < 0 {
			t.items = append(t.items, rest)
			break
		}
		t.items = append(t.items, rest[:end])
		rest = rest[end+len(m.delimiter):]
	}
	out := samples[message]
	t.failures = t.failures[:0]
	for i := range m.fields {
		f := &m.fields[i]
		value := f.value(t.items)
		sample, err := f.decode(out[i], value)
		if err == nil {
			out[i] = sample
			continue
		}
		err = errors.Wrapf(err, "field %s", f.name)
		if len(t.messages) > 1 {
			err = errors.Wrapf(err, "message %s", m.name)
		}
		t.failures = append(
			t.failures,
			Failure{Binding: i, Missing: len(value) == 0, Err: err},
		)
	}
	if len(t.failures) == 0 {
		return message, true, nil
	}
	return message, true, t.failures
}

// value returns the value of the field in items, the items of one line, or nil when
// the line has none.
func (f *textField) value(items [][]byte) []byte {
	if f.tagged {
		for _, item := range items {
			t := bytes.Trim(item, trimmed)
			if bytes.HasPrefix(t, f.tag) && !f.claimed(t) {
				return bytes.Trim(t[len(f.tag):], trimmed)
			}
		}
		return nil
	}
	if f.position < len(items) {
		return bytes.Trim(items[f.position], trimmed)
	}
	return nil
}

// decode appends the sample of value, or the field's default when value is empty, to
// dst.
func (f *textField) decode(dst, value []byte) ([]byte, error) {
	if len(value) == 0 {
		switch {
		case f.fallback != nil:
			return append(dst, f.fallback...), nil
		case f.tagged:
			return nil, errors.Wrapf(
				validate.ErrValidation, "no value after tag %q", f.tag,
			)
		default:
			return nil, errors.Wrapf(
				validate.ErrValidation, "no value at position %d", f.position,
			)
		}
	}
	v, err := xjson.ParseFloat64(value)
	if err != nil {
		return nil, err
	}
	return f.appendFloat(dst, v)
}

// claimed reports whether item starts with a longer tag than the field's, so it
// belongs to another field.
func (f *textField) claimed(item []byte) bool {
	for _, l := range f.longer {
		if bytes.HasPrefix(item, l) {
			return true
		}
	}
	return false
}

// appendFloat appends the sample of v to dst, scaled when the field scales.
func (f *textField) appendFloat(dst []byte, v float64) ([]byte, error) {
	if f.scaled {
		v = v*f.scale + f.offset
	}
	return f.sample.AppendFloat(dst, v)
}
