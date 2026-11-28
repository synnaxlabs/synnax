// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package write checks the writes of ICD messages that task configs hold against their
// ICDs and channels, and builds the payloads that a write task sends.
package write

import (
	"slices"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// defaultDataTypes holds the data type of the command channel that a client creates for
// an unscaled field of each kind, when the binding of the field names none.
var defaultDataTypes = map[payload.Kind]telem.DataType{
	payload.KindFloat:     telem.Float64T,
	payload.KindInteger:   telem.Int64T,
	payload.KindUnsigned:  telem.Uint64T,
	payload.KindBoolean:   telem.Uint8T,
	payload.KindString:    telem.StringT,
	payload.KindTimestamp: telem.TimestampT,
	payload.KindJSON:      telem.JSONT,
	payload.KindUUID:      telem.UUIDT,
	payload.KindBytes:     telem.BytesT,
}

// encoder encodes the payloads of one message, in any format.
type encoder interface {
	Encode(dst []byte, samples [][]byte) ([]byte, error)
	Check(i int, sample []byte) error
	Ready(samples [][]byte) bool
}

// newEncoder returns the encoder of the payloads that wm sends of its message in ic.
// The encoder takes one sample for each channel binding of wm, in binding order. A
// value binding sends its value as its field. dataTypes holds the data type of each
// channel that exists. A binding to another channel takes its own data type, or else
// float64 for a scaled field and the default of its field's kind for any other.
// newEncoder returns a validation error when wm is not valid, ic lacks the message, wm
// binds no channels, a binding does not suit its field, or the encoder refuses the
// bindings.
func newEncoder(
	wm icd.WriteMessage,
	ic icd.ICD,
	dataTypes map[channel.Key]telem.DataType,
) (encoder, error) {
	if err := wm.Validate(); err != nil {
		return nil, err
	}
	i := slices.IndexFunc(ic.Messages, func(m icd.Message) bool {
		return m.Key == wm.Message
	})
	if i < 0 {
		return nil, validate.Errorf(
			"message %s does not exist in ICD %s", wm.Message, ic.Name,
		)
	}
	msg := ic.Messages[i]
	if !slices.ContainsFunc(wm.Bindings, func(b icd.WriteBinding) bool {
		return b.Type() == icd.ChannelWriteBindingType
	}) {
		return nil, validate.Errorf("message %s binds no channels", msg.Name)
	}
	p, bindings, err := bind(wm, msg.Payload, dataTypes)
	if err != nil {
		return nil, err
	}
	switch v := p.Variant.(type) {
	case icd.JSONPayload:
		return built(codec.NewJSONEncoder(
			codec.JSONMessage{Name: msg.Name, Payload: v, Bindings: bindings},
		))
	case icd.BinaryPayload:
		return built(codec.NewBinaryEncoder(
			codec.BinaryMessage{Name: msg.Name, Payload: v, Bindings: bindings},
		))
	case icd.TextPayload:
		return built(codec.NewTextEncoder(
			codec.TextMessage{Name: msg.Name, Payload: v, Bindings: bindings},
		))
	default:
		panic(errors.Newf("write: unknown payload variant %T", v))
	}
}

// built returns e, or a nil encoder when err is not nil.
func built[E encoder](e E, err error) (encoder, error) {
	if err != nil {
		return nil, err
	}
	return e, nil
}

// bind returns p with the value of each value binding of wm as the default of its
// field, and the codec binding of each channel binding, in binding order. dataTypes is
// as in newEncoder. bind returns a validation error at the path of a binding that names
// no field of p, a field that a binding before it names, or the identifier field, or
// that holds a value its field cannot send.
func bind(
	wm icd.WriteMessage,
	p icd.Payload,
	dataTypes map[channel.Key]telem.DataType,
) (icd.Payload, []codec.Binding, error) {
	fields := fieldsOf(p)
	at := make(map[icd.FieldKey]int, len(fields))
	for i, f := range fields {
		at[f.key] = i
	}
	bound := make(set.Set[icd.FieldKey], len(wm.Bindings))
	var bindings []codec.Binding
	for j, b := range wm.Bindings {
		pathed := func(err error, segments ...string) error {
			path := slices.Concat([]string{"bindings", strconv.Itoa(j)}, segments)
			return validate.PathedError(err, path...)
		}
		key := b.Base().Field
		i, ok := at[key]
		switch {
		case !ok:
			return icd.Payload{}, nil, pathed(
				validate.Errorf("field %s does not exist", key), "field",
			)
		case bound.Contains(key):
			return icd.Payload{}, nil, pathed(validate.Errorf(
				"field %s has more than one binding", fields[i].name,
			), "field")
		case fields[i].identifier:
			return icd.Payload{}, nil, pathed(validate.Errorf(
				"field %s holds the identifier, so it takes no binding", fields[i].name,
			), "field")
		}
		bound.Add(key)
		switch v := b.Variant.(type) {
		case icd.ChannelWriteBinding:
			dt, found := dataTypes[v.Channel]
			if !found {
				dt = lo.FromPtrOr(v.DataType, fields[i].dataType())
			}
			bindings = append(bindings, codec.Binding{Field: key, DataType: dt})
		case icd.ValueWriteBinding:
			var err error
			if p, err = withFieldValue(p, i, v.Value); err != nil {
				return icd.Payload{}, nil, pathed(err, "value")
			}
		default:
			panic(errors.Newf("write: unknown write binding variant %T", v))
		}
	}
	return p, bindings, nil
}

// field is the part of a message field that a write checks.
type field struct {
	key    icd.FieldKey
	name   string
	kind   payload.Kind
	scaled bool
	// identifier is true when the field holds the identifier of its message.
	identifier bool
}

func newField(b icd.BaseField, k payload.Kind) field {
	return field{
		key:    b.Key,
		name:   b.Name,
		kind:   k,
		scaled: b.Scale != 1 || b.Offset != 0,
	}
}

// dataType returns the data type of the command channel that a client creates for f
// when its binding names none. A scaled field takes float64, because the codec writes a
// scaled field only from a float channel.
func (f field) dataType() telem.DataType {
	if f.scaled {
		return telem.Float64T
	}
	return defaultDataTypes[f.kind]
}

// fieldsOf returns the fields of p, in order. A text field encodes a float.
func fieldsOf(p icd.Payload) []field {
	switch v := p.Variant.(type) {
	case icd.JSONPayload:
		return lo.Map(v.Fields, func(f icd.JSONField, _ int) field {
			return newField(f.Base().BaseField, f.Type())
		})
	case icd.BinaryPayload:
		var id *icd.FieldKey
		if v.Identifier != nil {
			if fi, ok := v.Identifier.Variant.(icd.FieldIdentifier); ok {
				id = &fi.Field
			}
		}
		return lo.Map(v.Fields, func(f icd.BinaryField, _ int) field {
			out := newField(f.Base().BaseField, payload.Kind(f.Kind()))
			out.identifier = id != nil && *id == out.key
			return out
		})
	case icd.TextPayload:
		return lo.Map(v.Fields, func(f icd.TextField, _ int) field {
			return newField(f.Base().BaseField, payload.KindFloat)
		})
	default:
		panic(errors.Newf("write: unknown payload variant %T", v))
	}
}

// withFieldValue returns p with v as the default of field i. It returns a validation
// error when v is not of the kind of the field, or when the field cannot send v.
func withFieldValue(p icd.Payload, i int, v icd.WriteValue) (icd.Payload, error) {
	var (
		name, kind string
		ok         bool
		err        error
	)
	switch pv := p.Variant.(type) {
	case icd.JSONPayload:
		pv.Fields = slices.Clone(pv.Fields)
		f := &pv.Fields[i]
		name, kind = f.Base().Name, string(f.Type())
		if *f, ok = jsonWithValue(*f, v); ok {
			err = codec.CheckDefault(*f)
		}
		p.Variant = pv
	case icd.BinaryPayload:
		pv.Fields = slices.Clone(pv.Fields)
		f := &pv.Fields[i]
		name, kind = f.Base().Name, string(f.Kind())
		if *f, ok = binaryWithValue(*f, v); ok {
			err = icd.CheckBinaryDefault(*f)
		}
		p.Variant = pv
	case icd.TextPayload:
		pv.Fields = slices.Clone(pv.Fields)
		f := &pv.Fields[i]
		name, kind = f.Base().Name, string(payload.KindFloat)
		if *f, ok = textWithValue(*f, v); ok {
			err = icd.CheckTextDefault(*f)
		}
		p.Variant = pv
	default:
		panic(errors.Newf("write: unknown payload variant %T", pv))
	}
	if !ok {
		return icd.Payload{}, validate.Errorf(
			"field %s of kind %s needs a value of its kind",
			name,
			kind,
		)
	}
	if err != nil {
		return icd.Payload{}, errors.Wrapf(err, "field %s", name)
	}
	return p, nil
}

// jsonWithValue returns f with v as its default, and false when v is not of the kind of
// f.
func jsonWithValue(f icd.JSONField, v icd.WriteValue) (icd.JSONField, bool) {
	switch x := v.Variant.(type) {
	case icd.FloatWriteValue:
		return withDefault(f, func(fv *icd.FloatJSONField) { fv.Default = &x.Value })
	case icd.IntegerWriteValue:
		return withDefault(f, func(fv *icd.IntegerJSONField) { fv.Default = &x.Value })
	case icd.UnsignedWriteValue:
		return withDefault(f, func(fv *icd.UnsignedJSONField) { fv.Default = &x.Value })
	case icd.StringWriteValue:
		return withDefault(f, func(fv *icd.StringJSONField) { fv.Default = &x.Value })
	case icd.BooleanWriteValue:
		return withDefault(f, func(fv *icd.BooleanJSONField) { fv.Default = &x.Value })
	case icd.TimestampWriteValue:
		return withDefault(f, func(fv *icd.TimestampJSONField) {
			fv.Default = &icd.TimestampDefault{Variant: icd.ConstantTimestampDefault(x)}
		})
	case icd.JSONWriteValue:
		return withDefault(f, func(fv *icd.JSONJSONField) { fv.Default = &x.Value })
	case icd.UuidWriteValue:
		return withDefault(f, func(fv *icd.UuidJSONField) {
			fv.Default = &icd.UUIDDefault{Variant: icd.ConstantUUIDDefault(x)}
		})
	case icd.BytesWriteValue:
		return withDefault(f, func(fv *icd.BytesJSONField) { fv.Default = &x.Value })
	default:
		panic(errors.Newf("write: unknown write value variant %T", x))
	}
}

// withDefault returns f with set applied to its variant, and false when f is not an F.
func withDefault[F icd.JSONFieldVariant](
	f icd.JSONField,
	set func(*F),
) (icd.JSONField, bool) {
	fv, ok := f.Variant.(F)
	if ok {
		set(&fv)
		f.Variant = fv
	}
	return f, ok
}

// binaryWithValue returns f with v as its default, and false when v is not of the kind
// of f.
func binaryWithValue(f icd.BinaryField, v icd.WriteValue) (icd.BinaryField, bool) {
	switch x := v.Variant.(type) {
	case icd.FloatWriteValue:
		fv, ok := f.Variant.(icd.FloatBinaryField)
		if ok {
			fv.Default = &x.Value
			f.Variant = fv
		}
		return f, ok
	case icd.IntegerWriteValue:
		fv, ok := f.Variant.(icd.IntegerBinaryField)
		if ok {
			fv.Default = &x.Value
			f.Variant = fv
		}
		return f, ok
	case icd.UnsignedWriteValue:
		fv, ok := f.Variant.(icd.UnsignedBinaryField)
		if ok {
			fv.Default = &x.Value
			f.Variant = fv
		}
		return f, ok
	}
	return f, false
}

// textWithValue returns f with v as its default, and false when v is not a float.
func textWithValue(f icd.TextField, v icd.WriteValue) (icd.TextField, bool) {
	x, ok := v.Variant.(icd.FloatWriteValue)
	if !ok {
		return f, false
	}
	switch fv := f.Variant.(type) {
	case icd.DelimitedTextField:
		fv.Default = &x.Value
		f.Variant = fv
	case icd.TaggedTextField:
		fv.Default = &x.Value
		f.Variant = fv
	default:
		panic(errors.Newf("write: unknown text field variant %T", fv))
	}
	return f, true
}
