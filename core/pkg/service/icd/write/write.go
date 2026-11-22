// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package write checks the writes of ICD messages that task configs hold against their
// ICDs and channels.
package write

import (
	"encoding/json/v2"
	"slices"
	"strconv"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/codec"
	"github.com/synnaxlabs/x/content"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// defaultDataTypes holds the data type of the command channel that a client creates for
// a field of each kind, when the binding of the field names none.
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

// Encoder returns the encoder of the documents that wm sends of its message in ic. The
// encoder takes one sample for each binding of wm with no value, in binding order. A
// binding with a value sends that value as its field. dataTypes holds the data type of
// each channel that exists. A binding to another channel takes its own data type, or
// else the default of its field's kind. Encoder returns a validation error when ic
// lacks the message, the message is not JSON, wm binds no channels, a value does not
// suit its binding, or the encoder refuses the bindings.
func Encoder(
	wm icd.WriteMessage,
	ic icd.ICD,
	dataTypes map[channel.Key]telem.DataType,
) (*codec.JSONEncoder, error) {
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
		return b.Value == nil
	}) {
		return nil, validate.Errorf("message %s binds no channels", msg.Name)
	}
	jp.Fields = slices.Clone(jp.Fields)
	at := make(map[icd.FieldKey]int, len(jp.Fields))
	for k, f := range jp.Fields {
		at[f.Base().Key] = k
	}
	bound := lo.CountValuesBy(wm.Bindings, func(b icd.WriteBinding) icd.FieldKey {
		return b.Field
	})
	var bindings []codec.Binding
	for j, b := range wm.Bindings {
		f, exists := at[b.Field]
		if b.Value == nil {
			dt, ok := dataTypes[b.Channel]
			if !ok && exists {
				dt = lo.CoalesceOrEmpty(
					b.DataType,
					defaultDataTypes[jp.Fields[f].Type()],
				)
			}
			bindings = append(bindings, codec.Binding{Field: b.Field, DataType: dt})
			continue
		}
		if err := fill(&jp, at, b, bound[b.Field]); err != nil {
			return nil, validate.PathedError(err, "bindings", strconv.Itoa(j))
		}
	}
	return codec.NewJSONEncoder(codec.JSONMessage{
		Name:     msg.Name,
		Payload:  jp,
		Bindings: bindings,
	})
}

// fill makes the value of b the default of its field in jp, where at holds the index of
// each field. count is the number of bindings to the field. It returns a validation
// error at the path of the problem within b.
func fill(
	jp *icd.JSONPayload,
	at map[icd.FieldKey]int,
	b icd.WriteBinding,
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
	case b.Channel != 0:
		return validate.PathedError(
			validate.Errorf("a binding with a value cannot have a channel"), "channel",
		)
	case b.DataType != "":
		return validate.PathedError(
			validate.Errorf("a binding with a value cannot have a data type"),
			"data_type",
		)
	}
	f, ok := withValue(jp.Fields[i], *b.Value)
	if !ok {
		return validate.PathedError(validate.Errorf(
			"field %s of kind %s needs a value of its kind",
			f.Base().Name, f.Type(),
		), "value")
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

// Hash writes to h the payload of the message of ic that wm sends, so an edit to the
// format of the message changes the hash. When ic lacks the message, Hash writes only
// that it is absent.
func Hash(h *content.Hash, ic icd.ICD, wm icd.WriteMessage) error {
	m, ok := lo.Find(ic.Messages, func(m icd.Message) bool {
		return m.Key == wm.Message
	})
	h.Bool(ok)
	if !ok {
		return nil
	}
	b, err := json.Marshal(m.Payload, json.Deterministic(true))
	if err != nil {
		return err
	}
	h.Bytes(b)
	return nil
}
