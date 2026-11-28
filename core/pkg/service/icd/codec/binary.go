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
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// BinaryMessage is a binary message and the fields bound to channels.
type BinaryMessage struct {
	// Name identifies the message in errors.
	Name string
	// Payload is the layout of the message's payloads.
	Payload icd.BinaryPayload
	// Bindings name the bound fields, in sample order.
	Bindings []Binding
}

// Binary decodes payloads that each belong to one of several binary messages into
// samples. It is not safe for concurrent use.
type Binary struct {
	messages []binaryMessage
	headers  []header
	// fallback is the index of the message with no identifier, or -1.
	fallback int
	// failures holds the failures of the last decode.
	failures []Failure
}

// header matches payloads by the raw bits of an identifier field. Messages whose
// identifier fields read the same bits share a header.
type header struct {
	bits     bitRange
	messages map[uint64]int
}

type binaryMessage struct {
	name   string
	fields []binaryField
}

type binaryField struct {
	name     string
	bits     bitRange
	encoding encoding
	// scaled is true when the field's scale or offset changes its value.
	scaled        bool
	scale, offset float64
	// size is the size of the integer channel that takes the raw bits of an unscaled
	// integer field, or 0 when the field's values go through a float.
	size   int
	sample *xjson.Decoder
	// fallback is the sample used when a payload does not hold the field, or nil when
	// the field requires one.
	fallback []byte
}

// NewBinary returns a Binary decoder for messages that share one source of payloads. A
// payload belongs to the message whose identifier field holds its value, or else to the
// message with no identifier. NewBinary returns an error wrapping
// validate.ErrValidation when a message has a CAN identifier, two messages have no
// identifier, two identifiers match the same value of one field layout, an identifier
// is invalid, a binding names no field of its message, a bound field has a
// multiplexor, a field's layout or default is invalid, a bound field has a zero scale,
// or a field does not convert to the data type of its binding.
func NewBinary(messages []BinaryMessage) (*Binary, error) {
	b := &Binary{messages: make([]binaryMessage, len(messages)), fallback: -1}
	for i, m := range messages {
		var err error
		if b.messages[i], err = newBinaryMessage(m); err != nil {
			return nil, errors.Wrapf(err, "message %s", m.Name)
		}
		if err = b.route(m, i); err != nil {
			return nil, err
		}
	}
	return b, nil
}

func binaryKey(f icd.BinaryField) icd.FieldKey { return f.Base().Key }

func newBinaryMessage(m BinaryMessage) (binaryMessage, error) {
	fields, err := boundFields(m.Payload.Fields, binaryKey, m.Bindings)
	if err != nil {
		return binaryMessage{}, err
	}
	msg := binaryMessage{name: m.Name, fields: make([]binaryField, len(fields))}
	for i, f := range fields {
		field, err := newBinaryField(f, m.Bindings[i].DataType)
		if err == nil && f.Base().Multiplexor != nil {
			err = errors.Wrap(validate.ErrValidation, "multiplexors are not supported")
		}
		if err == nil {
			err = checkLength(field.bits, m.Payload)
		}
		if err != nil {
			return binaryMessage{}, errors.Wrapf(err, "field %s", f.Base().Name)
		}
		msg.fields[i] = field
	}
	return msg, nil
}

func newBinaryField(f icd.BinaryField, dt telem.DataType) (binaryField, error) {
	b := f.Base()
	if err := icd.CheckScale(b.Scale); err != nil {
		return binaryField{}, errors.Wrap(validate.ErrValidation, err.Error())
	}
	bits, err := newBitRange(b)
	if err != nil {
		return binaryField{}, err
	}
	e, err := encodingOf(f)
	if err != nil {
		return binaryField{}, err
	}
	field := binaryField{
		name:     b.Name,
		bits:     bits,
		encoding: e,
		scaled:   b.Scale != 1 || b.Offset != 0,
		scale:    b.Scale,
		offset:   b.Offset,
	}
	if err = field.bind(dt, b.BitLength); err != nil {
		return binaryField{}, err
	}
	raw, ok, err := defaultRaw(f, e)
	if err == nil && ok {
		field.fallback, err = field.append(nil, raw)
	}
	if err != nil {
		return binaryField{}, errors.Wrap(err, "default")
	}
	return field, nil
}

// bind sets how the field, of bitLength bits, writes to channels of dt. It returns an
// error wrapping validate.ErrValidation when the field's values do not convert to dt,
// or need more bits than dt holds.
func (f *binaryField) bind(dt telem.DataType, bitLength uint8) error {
	kind := f.encoding.kind()
	if !Converts(kind, dt, false) {
		return errors.Wrapf(
			validate.ErrValidation, "cannot write %s values to %s channels", kind, dt,
		)
	}
	if !Converts(kind, dt, f.scaled) {
		return errors.Wrapf(
			validate.ErrValidation,
			"a scaled field requires a float channel, not %s",
			dt,
		)
	}
	dk, _ := payload.KindOf(dt)
	if dk == payload.KindFloat {
		var err error
		f.sample, err = xjson.NewDecoder(xjson.Field{DataType: dt})
		return err
	}
	f.size = int(dt.Density())
	needed := int(bitLength)
	if dk == payload.KindInteger && f.encoding != encodingSigned {
		needed++
	}
	if needed > 8*f.size {
		return errors.Wrapf(
			validate.ErrValidation,
			"cannot write %d-bit %s values to %s channels",
			bitLength,
			kind,
			dt,
		)
	}
	return nil
}

// route sends the payloads that hold the identifier of m to the message at index i. It
// returns an error wrapping validate.ErrValidation when m has a CAN identifier or an
// invalid field identifier, or an earlier message matches the same payloads.
func (b *Binary) route(m BinaryMessage, i int) error {
	if m.Payload.Identifier == nil {
		if b.fallback >= 0 {
			return errors.Wrapf(
				validate.ErrValidation,
				"messages %s and %s both have no identifier",
				b.messages[b.fallback].name,
				m.Name,
			)
		}
		b.fallback = i
		return nil
	}
	var id icd.FieldIdentifier
	switch v := m.Payload.Identifier.Variant.(type) {
	case icd.FieldIdentifier:
		id = v
	case icd.CanIdentifier:
		return errors.Wrapf(
			validate.ErrValidation,
			"message %s: identifier: only the Driver reads CAN frames",
			m.Name,
		)
	default:
		panic(errors.Newf("codec: unknown identifier %T", v))
	}
	bits, err := identifierBits(id, m.Payload)
	if err != nil {
		return errors.Wrapf(err, "message %s: identifier", m.Name)
	}
	h := slices.IndexFunc(b.headers, func(h header) bool { return h.bits == bits })
	if h < 0 {
		h = len(b.headers)
		b.headers = append(b.headers, header{bits: bits, messages: map[uint64]int{}})
	}
	raw := uint64(int64(id.Value))
	if bits.width < 64 {
		raw &= 1<<bits.width - 1
	}
	if other, ok := b.headers[h].messages[raw]; ok {
		return errors.Wrapf(
			validate.ErrValidation,
			"messages %s and %s match the same identifier value %d",
			b.messages[other].name,
			m.Name,
			id.Value,
		)
	}
	b.headers[h].messages[raw] = i
	return nil
}

// identifierBits returns the layout of the field of id in p. It returns an error
// wrapping validate.ErrValidation when the field does not exist, is a float field, has
// an invalid layout, ends past the payload length, or cannot hold the identifier value.
func identifierBits(id icd.FieldIdentifier, p icd.BinaryPayload) (bitRange, error) {
	i := slices.IndexFunc(p.Fields, func(f icd.BinaryField) bool {
		return binaryKey(f) == id.Field
	})
	if i < 0 {
		return bitRange{}, errors.Wrapf(
			validate.ErrValidation, "field %s does not exist", id.Field,
		)
	}
	f := p.Fields[i]
	b := f.Base()
	_, signed := f.Variant.(icd.IntegerBinaryField)
	if _, float := f.Variant.(icd.FloatBinaryField); float {
		return bitRange{}, errors.Wrapf(
			validate.ErrValidation, "field %s is a float field", b.Name,
		)
	}
	bits, err := newBitRange(b)
	if err == nil {
		err = checkLength(bits, p)
	}
	if err != nil {
		return bitRange{}, errors.Wrapf(err, "field %s", b.Name)
	}
	v, n := int64(id.Value), bits.width
	fits := v >= 0 && (n >= 63 || v < 1<<n)
	if signed {
		fits = n >= 64 || (v >= -(1<<(n-1)) && v < 1<<(n-1))
	}
	if !fits {
		return bitRange{}, errors.Wrapf(
			validate.ErrValidation, "%d is out of range for field %s", v, b.Name,
		)
	}
	return bits, nil
}

// Decode finds the message that data, one payload, belongs to by its identifier fields
// and appends one sample for each of its bindings to samples[message]. samples is as in
// JSON.Decode. ok is false, and Decode appends nothing, when data belongs to no
// message. Decode returns an error wrapping validate.ErrValidation, appending nothing,
// with the first of the messages, when data meets the identifiers of two messages.
//
// failures holds each field with no default that data is too short for, or whose value
// does not fit its channel. A failed field appends no sample, and the other fields
// still append. A failure names the field, and also the message when the decoder holds
// more than one. failures is nil when every field decodes, and valid until the next
// Decode.
func (b *Binary) Decode(
	data []byte,
	samples [][][]byte,
) (message int, ok bool, failures []Failure, err error) {
	message = b.fallback
	matched := false
	for _, h := range b.headers {
		if len(data) < h.bits.end {
			continue
		}
		i, found := h.messages[h.bits.read(data)]
		if !found {
			continue
		}
		if matched {
			first, second := min(message, i), max(message, i)
			return first, true, nil, errors.Wrapf(
				validate.ErrValidation,
				"frame matches messages %s and %s",
				b.messages[first].name,
				b.messages[second].name,
			)
		}
		message, matched = i, true
	}
	if message < 0 {
		return 0, false, nil, nil
	}
	m := &b.messages[message]
	b.failures = m.decode(data, samples[message], b.failures[:0])
	if len(b.failures) == 0 {
		return message, true, nil, nil
	}
	if len(b.messages) > 1 {
		for i := range b.failures {
			b.failures[i].Err = errors.Wrapf(b.failures[i].Err, "message %s", m.name)
		}
	}
	return message, true, b.failures, nil
}

// decode appends the sample of each field in data to samples, and each failed field to
// failures.
func (m *binaryMessage) decode(
	data []byte,
	samples [][]byte,
	failures []Failure,
) []Failure {
	for i := range m.fields {
		f := &m.fields[i]
		missing := len(data) < f.bits.end
		var err error
		switch {
		case !missing:
			var out []byte
			if out, err = f.decode(samples[i], data); err == nil {
				samples[i] = out
			}
		case f.fallback != nil:
			samples[i] = append(samples[i], f.fallback...)
		default:
			err = tooShort(len(data), f.bits.end)
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

// decode appends the sample of the field in data, which holds the field, to dst.
func (f *binaryField) decode(dst, data []byte) ([]byte, error) {
	if f.encoding == encodingSigned {
		return f.append(dst, uint64(f.bits.readSigned(data)))
	}
	return f.append(dst, f.bits.read(data))
}

// append appends the sample of raw, the raw bits of the field, to dst.
func (f *binaryField) append(dst []byte, raw uint64) ([]byte, error) {
	if f.size > 0 {
		for i := range f.size {
			dst = append(dst, byte(raw>>(8*i)))
		}
		return dst, nil
	}
	v := f.encoding.value(raw)
	if f.scaled {
		v = v*f.scale + f.offset
	}
	return f.sample.AppendFloat(dst, v)
}

// checkLength returns an error wrapping validate.ErrValidation when bits end past the
// length of p.
func checkLength(bits bitRange, p icd.BinaryPayload) error {
	if p.Length == nil || bits.end <= int(*p.Length) {
		return nil
	}
	return errors.Wrapf(
		validate.ErrValidation, "the field ends past the %d-byte payload", *p.Length,
	)
}

// tooShort returns the error for a payload of size bytes that is shorter than the end
// of a field.
func tooShort(size, end int) error {
	return errors.Wrapf(
		validate.ErrValidation,
		"the payload has %d bytes and the field needs %d",
		size,
		end,
	)
}
