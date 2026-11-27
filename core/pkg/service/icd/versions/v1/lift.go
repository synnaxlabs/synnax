// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"uuid"

	"github.com/samber/lo"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	"github.com/synnaxlabs/x/errors"
	xpayload "github.com/synnaxlabs/x/payload"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// InlineWrite is a write that a config held inline, before the config sent the messages
// of an ICD.
type InlineWrite struct {
	// Target names where the write sends its messages, such as "topic a/b", in notes.
	Target string
	// Channel is the field that sends the commands of a channel.
	Channel InlineChannelField
	// Fields are the fields that send a fixed or generated value.
	Fields []InlineWriteField
}

// InlineChannelField is the field of an inline write that sends the commands of a
// channel.
type InlineChannelField struct {
	// Path names the field in its config, such as targets.0.channel, in notes.
	Path string
	// Pointer is the JSON Pointer to the value of the field.
	Pointer string
	// JSONType is the JSON type that the write sent each command as.
	JSONType payload.JSONType
	// Channel is the channel whose commands the field sends.
	Channel channel.Key
	// Name is the name of a new channel for the field.
	Name string
	// DataType is the data type of a new channel for the field.
	DataType telem.DataType
	// TimeFormat is the encoding of a timestamp command.
	TimeFormat *payload.TimeFormat
	// EnumValues maps text labels to command values.
	EnumValues []payload.EnumEntry
}

// InlineWriteField is a field of an inline write: a StaticInlineField, a
// ValueInlineField, or a GeneratedInlineField.
type InlineWriteField interface{ isInlineWriteField() }

// StaticInlineField is a field of an inline write that sends a fixed value.
type StaticInlineField struct {
	// Path names the field in its config, such as targets.0.fields.1, in notes.
	Path string
	// Pointer is the JSON Pointer to the value of the field.
	Pointer string
	// Value is the value that the field sends.
	Value any
}

func (StaticInlineField) isInlineWriteField() {}

// ValueInlineField is a field of an inline write that sends a string that the task
// picks, such as the name of a channel. The task holds the string, not the ICD.
type ValueInlineField struct {
	// Pointer is the JSON Pointer to the value of the field.
	Pointer string
	// Value is the string that the field sends.
	Value string
}

func (ValueInlineField) isInlineWriteField() {}

// GeneratedInlineField is a field of an inline write that sends a new value in each
// message.
type GeneratedInlineField struct {
	// Path names the field in its config, such as targets.0.fields.1, in notes.
	Path string
	// Pointer is the JSON Pointer to the value of the field.
	Pointer string
	// Generator makes the value of the field.
	Generator payload.GeneratorType
	// TimeFormat is the encoding of a generated timestamp.
	TimeFormat *payload.TimeFormat
}

func (GeneratedInlineField) isInlineWriteField() {}

// LiftWrite returns a JSON message that holds the fields of w, with no name, and the
// write of that message. The channel field takes the kind of its channel in found. When
// found lacks that channel, the stored data type picks the kind, and the binding holds
// no channel but the data type of a new one. A static field sends its value as a
// constant default, a value field has no default and a binding to its value, and a
// generated field sends a generator default. note receives the path of a field and
// each change to what that field sends. LiftWrite returns a validation error that names
// the path of a static field whose value is not JSON.
func LiftWrite(
	ctx context.Context,
	w InlineWrite,
	found map[channel.Key]v0.Channel,
	note func(path, change string),
) (Message, WriteMessage, error) {
	var (
		names  = make(set.Set[string])
		fields = make([]JSONField, 0, 1+len(w.Fields))
		write  = WriteMessage{Message: uuid.New()}
	)
	notef := func(path string) func(format string, args ...any) {
		return func(format string, args ...any) {
			note(path, fmt.Sprintf(format, args...))
		}
	}
	ch, exists := found[w.Channel.Channel]
	base := writeBase(names, w.Channel.Pointer)
	field, dataType, err := liftChannel(
		ctx,
		w.Channel,
		ch,
		exists,
		base,
		notef(w.Channel.Path),
	)
	if err != nil {
		return Message{}, WriteMessage{}, err
	}
	fields = append(fields, field)
	binding := ChannelWriteBinding{Field: base.Key, Channel: w.Channel.Channel}
	if !exists {
		binding.Channel, binding.DataType = 0, dataType
		if w.Channel.Name != "" {
			note(w.Channel.Path, fmt.Sprintf(
				"channel name %q of field %s on %s was dropped, so a client names "+
					"the new channel",
				w.Channel.Name, base.Name, w.Target,
			))
		}
	}
	write.Bindings = []WriteBinding{{Variant: binding}}
	for _, wf := range w.Fields {
		f, bound, err := liftWriteField(wf, names, notef)
		if err != nil {
			return Message{}, WriteMessage{}, err
		}
		fields = append(fields, f)
		if bound != nil {
			write.Bindings = append(write.Bindings, WriteBinding{Variant: *bound})
		}
	}
	return Message{
		Key:     write.Message,
		Payload: Payload{Variant: JSONPayload{Fields: fields}},
	}, write, nil
}

// writeBase returns the base of a new field at pointer, which v0.FieldName names apart
// from the names in taken.
func writeBase(taken set.Set[string], pointer string) JSONBaseField {
	return JSONBaseField{
		Key:     uuid.New(),
		Name:    v0.FieldName(taken, "", pointer),
		Scale:   1,
		Pointer: pointer,
	}
}

// liftChannel returns the JSON field of base that sends old the way the inline write
// sent it, and the data type of a new channel for it, or nil when that data type is not
// known. The data type of ch, when old's channel exists, picks the kind. Otherwise the
// stored data type picks it, and is the data type of a new channel. note receives each
// change to what the field sends.
func liftChannel(
	ctx context.Context,
	old InlineChannelField,
	ch v0.Channel,
	exists bool,
	base JSONBaseField,
	note func(format string, args ...any),
) (JSONField, *telem.DataType, error) {
	dt := lo.CoalesceOrEmpty(old.DataType, telem.Float64T)
	if exists {
		dt = ch.DataType
	}
	kind, known := xpayload.KindOf(dt)
	if !known {
		kind = payload.KindFloat
		note("unknown data type %q became the float kind", dt)
	}
	switch {
	case kind == payload.KindJSON:
		note("a %s channel sent as a JSON string now sends its JSON value", dt)
	case !numeric(kind):
	case old.JSONType == payload.JSONTypeBoolean:
		if kind != payload.KindBoolean {
			note("a %s channel sent as a JSON boolean now sends only 0 and 1", dt)
		}
		kind = payload.KindBoolean
	default:
		if old.JSONType == payload.JSONTypeString && len(old.EnumValues) == 0 {
			note("a %s channel sent as a JSON string now sends a JSON number", dt)
		}
		if kind == payload.KindBoolean {
			kind = payload.KindUnsigned
		}
	}
	format := old.TimeFormat
	if kind == payload.KindTimestamp {
		format = validFormat(format, note)
	}
	field, err := MigrateJSONField(ctx, v0.LiftKind(
		v0.InlineField{EnumValues: old.EnumValues, TimeFormat: format},
		kind,
		base,
		note,
	))
	if err != nil {
		return JSONField{}, nil, err
	}
	if !known {
		return field, nil, nil
	}
	return field, &dt, nil
}

// numeric reports whether k holds numbers or booleans.
func numeric(k payload.Kind) bool {
	return k == payload.KindFloat || k == payload.KindInteger ||
		k == payload.KindUnsigned || k == payload.KindBoolean
}

// liftWriteField returns the JSON field that sends what old sent, named apart from the
// names in taken, and the binding of a value field, or nil for another field. A static
// field sends its value as a constant default, a value field has no default, and a
// generated field sends a generator default. notef returns the note of a field path.
func liftWriteField(
	old InlineWriteField,
	taken set.Set[string],
	notef func(path string) func(format string, args ...any),
) (JSONField, *ValueWriteBinding, error) {
	switch v := old.(type) {
	case StaticInlineField:
		f, err := liftStatic(v.Value, writeBase(taken, v.Pointer))
		return f, nil, errors.Wrap(err, v.Path)
	case ValueInlineField:
		base := writeBase(taken, v.Pointer)
		return JSONField{Variant: StringJSONField{JSONBaseField: base}},
			&ValueWriteBinding{
				Field: base.Key,
				Value: WriteValue{Variant: StringWriteValue{Value: v.Value}},
			}, nil
	case GeneratedInlineField:
		base := writeBase(taken, v.Pointer)
		if v.Generator != payload.GeneratorTypeTimestamp {
			return JSONField{Variant: UuidJSONField{
				JSONBaseField: base,
				Default:       &UUIDDefault{Variant: RandomUUIDDefault{}},
			}}, nil, nil
		}
		format := payload.TimeFormatISO8601
		if f := validFormat(v.TimeFormat, notef(v.Path)); f != nil {
			format = *f
		}
		return JSONField{Variant: TimestampJSONField{
			JSONBaseField: base,
			TimeFormat:    format,
			Default:       &TimestampDefault{Variant: NowTimestampDefault{}},
		}}, nil, nil
	}
	panic(fmt.Sprintf("unknown inline write field %T", old))
}

// liftStatic returns the JSON field of base whose default is value: a string, boolean,
// or float field for a value of that type, and a JSON field for any other value.
func liftStatic(value any, base JSONBaseField) (JSONField, error) {
	switch v := value.(type) {
	case string:
		return JSONField{
			Variant: StringJSONField{JSONBaseField: base, Default: &v},
		}, nil
	case bool:
		return JSONField{
			Variant: BooleanJSONField{JSONBaseField: base, Default: &v},
		}, nil
	}
	if n, ok := number(value); ok {
		return JSONField{Variant: FloatJSONField{
			JSONBaseField: base,
			Default:       new(payload.Float(n)),
		}}, nil
	}
	text, err := json.Marshal(value)
	if err != nil {
		return JSONField{}, errors.Wrap(validate.ErrValidation, err.Error())
	}
	return JSONField{Variant: JSONJSONField{
		JSONBaseField: base,
		Default:       new(string(text)),
	}}, nil
}

// number returns value as a float64, and false when value is not a number.
func number(value any) (float64, bool) {
	v := reflect.ValueOf(value)
	switch {
	case v.CanFloat():
		return v.Float(), true
	case v.CanInt():
		return float64(v.Int()), true
	case v.CanUint():
		return float64(v.Uint()), true
	}
	return 0, false
}

// validFormat returns format, or nil when format is not valid.
func validFormat(
	format *payload.TimeFormat,
	note func(format string, args ...any),
) *payload.TimeFormat {
	if format == nil || format.IsValid() {
		return format
	}
	note(
		"time format %q is not valid and became %s",
		*format,
		payload.TimeFormatISO8601,
	)
	return nil
}
