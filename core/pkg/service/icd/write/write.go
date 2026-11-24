// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package write checks the writes of ICD messages that task configs hold against their
// ICDs and channels, and builds the documents that a write task sends.
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

// newEncoder returns the encoder of the documents that wm sends of its message in ic.
// The encoder takes one sample for each channel binding of wm, in binding order. A
// value binding sends its value as its field. dataTypes holds the data type of each
// channel that exists. A binding to another channel takes its own data type, or else
// float64 for a scaled field and the default of its field's kind for any other.
// newEncoder returns a validation error when wm is not valid, ic lacks the message, the
// message is not JSON, wm binds no channels, a value does not suit its field, or the
// encoder refuses the bindings.
func newEncoder(
	wm icd.WriteMessage,
	ic icd.ICD,
	dataTypes map[channel.Key]telem.DataType,
) (*codec.JSONEncoder, error) {
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
	jp, ok := msg.Payload.Variant.(icd.JSONPayload)
	if !ok {
		return nil, validate.Errorf("message %s is not a JSON message", msg.Name)
	}
	if !slices.ContainsFunc(wm.Bindings, func(b icd.WriteBinding) bool {
		return b.Type() == icd.ChannelWriteBindingType
	}) {
		return nil, validate.Errorf("message %s binds no channels", msg.Name)
	}
	jp.Fields = slices.Clone(jp.Fields)
	at := make(map[icd.FieldKey]int, len(jp.Fields))
	for k, f := range jp.Fields {
		at[f.Base().Key] = k
	}
	bound := lo.CountValuesBy(wm.Bindings, func(b icd.WriteBinding) icd.FieldKey {
		return b.Base().Field
	})
	var bindings []codec.Binding
	for j, b := range wm.Bindings {
		switch v := b.Variant.(type) {
		case icd.ChannelWriteBinding:
			dt, ok := dataTypes[v.Channel]
			if f, exists := at[v.Field]; !ok && exists {
				dt = lo.FromPtrOr(v.DataType, defaultDataType(jp.Fields[f]))
			}
			bindings = append(bindings, codec.Binding{Field: v.Field, DataType: dt})
		case icd.ValueWriteBinding:
			if err := fill(&jp, at, v, bound[v.Field]); err != nil {
				return nil, validate.PathedError(err, "bindings", strconv.Itoa(j))
			}
		default:
			panic(errors.Newf("write: unknown write binding variant %T", v))
		}
	}
	return codec.NewJSONEncoder(codec.JSONMessage{
		Name:     msg.Name,
		Payload:  jp,
		Bindings: bindings,
	})
}

// defaultDataType returns the data type of the command channel that a client creates
// for f when its binding names none. A scaled field takes float64, because the codec
// writes a scaled field only from a float channel.
func defaultDataType(f icd.JSONField) telem.DataType {
	if b := f.Base(); b.Scale != 1 || b.Offset != 0 {
		return telem.Float64T
	}
	return defaultDataTypes[f.Type()]
}

// fill makes the value of b the default of its field in jp, where at holds the index of
// each field. count is the number of bindings to the field. It returns a validation
// error at the path of the problem within b.
func fill(
	jp *icd.JSONPayload,
	at map[icd.FieldKey]int,
	b icd.ValueWriteBinding,
	count int,
) error {
	i, exists := at[b.Field]
	switch {
	case !exists:
		return validate.PathedError(
			validate.Errorf("field %s does not exist", b.Field), "field",
		)
	case count > 1:
		return validate.PathedError(validate.Errorf(
			"field %s has more than one binding", jp.Fields[i].Base().Name,
		), "field")
	}
	f, ok := withValue(jp.Fields[i], b.Value)
	if !ok {
		return validate.PathedError(validate.Errorf(
			"field %s of kind %s needs a value of its kind",
			f.Base().Name, f.Type(),
		), "value")
	}
	if err := codec.CheckDefault(f); err != nil {
		return validate.PathedError(
			errors.Wrapf(err, "field %s", f.Base().Name), "value",
		)
	}
	jp.Fields[i] = f
	return nil
}

// withValue returns f with v as its default, and false when v is not of the kind of f.
func withValue(f icd.JSONField, v icd.WriteValue) (icd.JSONField, bool) {
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
