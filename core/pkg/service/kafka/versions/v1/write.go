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
	"encoding/json/v2"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	http "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// CreateICD creates ic through tx.
type CreateICD func(ctx context.Context, tx gorp.Tx, ic icd.ICD) error

// MigrateWriteConfig lifts a v0 write config to v1. When old has channels, it moves
// them into Inline with the topic, record key, and record of old, for
// ResolveWriteConfig to add back as targets with an ICD. Those fields of the config
// then take their defaults.
func MigrateWriteConfig(ctx context.Context, old v0.WriteConfig) (WriteConfig, error) {
	cfg, err := autoMigrateWriteConfig(ctx, old)
	if err != nil {
		return cfg, err
	}
	stage(&cfg)
	return cfg, nil
}

// stage moves the channels of cfg into Inline with its topic, record key, and record,
// and gives those fields their defaults. It does nothing when cfg has no channels.
func stage(cfg *WriteConfig) {
	if len(cfg.Channels) == 0 {
		return
	}
	cfg.Inline = &InlineWrite{
		Topic:     cfg.Topic,
		RecordKey: cfg.RecordKey,
		Record:    cfg.Record,
		Channels:  cfg.Channels,
	}
	cfg.Topic, cfg.RecordKey, cfg.Channels = "", RecordKeyChannelName, []WriteChannel{}
	cfg.Record = Record{Fields: []http.WriteField{}}
	cfg.Record.SetDefaults()
}

// ResolveWriteConfig moves the inline channels of cfg into a new ICD named name, with
// one JSON message for each channel, named after it. It adds a target for each channel
// to the targets of cfg, which sends its message to the inline topic. It returns the
// ICD, which the caller must create, and false when cfg holds no inline channels. The
// value field of a message takes the kind of its channel, read through snap with
// channels. When that channel does not exist, its binding holds no channel and the
// float64 data type of a new one. note receives the path and each change of a field.
func ResolveWriteConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	channels Channels,
	name string,
	cfg *WriteConfig,
	note func(path, change string),
) (icd.ICD, bool, error) {
	if cfg.Inline == nil {
		return icd.ICD{}, false, nil
	}
	in := cfg.Inline
	found, err := inlineWriteChannels(ctx, snap, channels, in.Channels)
	if err != nil {
		return icd.ICD{}, false, err
	}
	ic := icd.ICD{
		Key:      uuid.New(),
		Name:     name,
		Messages: make([]icd.Message, len(in.Channels)),
	}
	if in.Record.TimestampPointer != nil {
		note(
			"record.timestamp_pointer",
			"the field sends the time of the write, not the time of the sample",
		)
	}
	names := make(set.Set[string])
	for i, wc := range in.Channels {
		// Each message repeats the fields of the record, so only the first one notes
		// their changes.
		recordNote := note
		if i > 0 {
			recordNote = func(string, string) {}
		}
		ch, exists := found[wc.Channel]
		msg, t, err := liftChannel(
			ctx, in, wc, ch, exists, "channels."+strconv.Itoa(i), note, recordNote,
		)
		if err != nil {
			return icd.ICD{}, false, err
		}
		msg.Name = uniqueName(names, lo.CoalesceOrEmpty(ch.Name, "Untitled"))
		ic.Messages[i] = msg
		t.ICD = ic.Key
		cfg.Targets = append(cfg.Targets, t)
	}
	cfg.Inline = nil
	return ic, true, nil
}

// NewWriteMigration returns the migration that moves the channels of each stored write
// config into targets, after WriteMigration. It resolves each config as
// ResolveWriteConfig does, with the name of its task, creates the ICD with create, and
// logs each change to a field. The config of a snapshot task gets no ICD and loses its
// channels, because the task service deletes snapshots after this migration runs.
func NewWriteMigration(channels Channels, create CreateICD) gorp.Migration {
	return migrate.New(
		"v63_kafka_write_icd",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var cfgs []WriteConfig
			if err := gorp.NewRetrieve[uuid.UUID, WriteConfig]().
				Where(gorp.Match[uuid.UUID](
					func(_ gorp.Context, c *WriteConfig) (bool, error) {
						return len(c.Channels) > 0, nil
					},
				)).
				Entries(&cfgs).
				Exec(ctx, tx); err != nil || len(cfgs) == 0 {
				return err
			}
			tasks, err := retrieveTasks(ctx, tx, cfgs)
			if err != nil {
				return err
			}
			news := make([]WriteConfig, len(cfgs))
			for i, cfg := range cfgs {
				if news[i], err = migrateStored(
					ctx,
					tx,
					channels,
					create,
					tasks[cfg.Key],
					cfg,
					NoteLift(ins, cfg.Key),
				); err != nil {
					return errors.Wrapf(err, "config %s", cfg.Key)
				}
			}
			return gorp.WrapWriter[uuid.UUID, WriteConfig](tx).Set(ctx, news...)
		},
	)
}

func migrateStored(
	ctx context.Context,
	tx gorp.Tx,
	channels Channels,
	create CreateICD,
	t task.Task,
	cfg WriteConfig,
	note func(path, change string),
) (WriteConfig, error) {
	stage(&cfg)
	if t.Snapshot {
		return cfg, nil
	}
	ic, ok, err := ResolveWriteConfig(ctx, tx, channels, t.Name, &cfg, note)
	if err != nil || !ok {
		return cfg, err
	}
	return cfg, create(ctx, tx, ic)
}

// retrieveTasks returns the task of each config of cfgs by key, read through tx.
func retrieveTasks(
	ctx context.Context,
	tx gorp.Tx,
	cfgs []WriteConfig,
) (map[uuid.UUID]task.Task, error) {
	var tasks []task.Task
	if err := gorp.NewRetrieve[task.Key, task.Task]().
		Where(gorp.MatchKeys[task.Key, task.Task](
			lo.Map(cfgs, func(c WriteConfig, _ int) task.Key { return c.Key })...,
		)).
		Entries(&tasks).
		Exec(ctx, tx); err != nil {
		return nil, errors.Wrap(err, "tasks of Kafka write configs")
	}
	return lo.KeyBy(tasks, func(t task.Task) uuid.UUID { return t.Key }), nil
}

func inlineWriteChannels(
	ctx context.Context,
	snap gorp.Snapshot,
	channels Channels,
	staged []WriteChannel,
) (map[channel.Key]Channel, error) {
	keys := lo.FilterMap(staged, func(wc WriteChannel, _ int) (channel.Key, bool) {
		return wc.Channel, wc.Channel != 0
	})
	if len(keys) == 0 {
		return nil, nil
	}
	return channels(ctx, snap, lo.Uniq(keys))
}

// liftChannel returns the message that sends the samples of wc in the shape of the
// record of in, with no name, and the target that sends it, with no ICD. ch is the
// channel of wc when exists is true, and path is the path of wc. note receives each
// change to the value field, and recordNote each change to a field of the record.
func liftChannel(
	ctx context.Context,
	in *InlineWrite,
	wc WriteChannel,
	ch Channel,
	exists bool,
	path string,
	note func(path, change string),
	recordNote func(path, change string),
) (icd.Message, WriteTarget, error) {
	var (
		names = make(set.Set[string])
		value = newBase(names, in.Record.ValuePointer)
		write = icd.WriteMessage{Message: uuid.New()}
	)
	field, err := liftValue(ctx, wc, ch, exists, value, notef(note, path))
	if err != nil {
		return icd.Message{}, WriteTarget{}, err
	}
	binding := icd.ChannelWriteBinding{Field: value.Key, Channel: wc.Channel}
	if !exists {
		binding.Channel, binding.DataType = 0, new(telem.Float64T)
	}
	write.Bindings = []icd.WriteBinding{{Variant: binding}}
	rest, err := liftRecordFields(in.Record, ch.Name, names, recordNote)
	if err != nil {
		return icd.Message{}, WriteTarget{}, err
	}
	t := WriteTarget{
		Key:      lo.CoalesceOrEmpty(wc.Key, uuid.New().String()),
		Disabled: wc.Disabled,
		Topic:    in.Topic,
		Message:  write,
	}
	if in.RecordKey == RecordKeyChannelName {
		t.RecordKey = ch.Name
	}
	return icd.Message{
		Key: write.Message,
		Payload: icd.Payload{Variant: icd.JSONPayload{
			Fields: append([]icd.JSONField{field}, rest...),
		}},
	}, t, nil
}

// liftValue returns the JSON field of base that sends the samples of wc the way v0 sent
// them. The data type of ch picks the kind when exists is true, and float64 does
// otherwise. note receives each change to what the field sends.
func liftValue(
	ctx context.Context,
	wc WriteChannel,
	ch Channel,
	exists bool,
	base icd.JSONBaseField,
	note func(format string, args ...any),
) (icd.JSONField, error) {
	dt := telem.Float64T
	if exists {
		dt = ch.DataType
	}
	kind, known := payload.KindOf(dt)
	if !known {
		kind = payload.KindFloat
		note("unknown data type %q became the float kind", dt)
	}
	switch {
	case kind == payload.KindJSON:
		note("a %s channel sent as a JSON string now sends its JSON value", dt)
	case !numeric(kind):
	case wc.JSONType == payload.JSONTypeBoolean:
		if kind != payload.KindBoolean {
			note("a %s channel sent as a JSON boolean now sends only 0 and 1", dt)
		}
		kind = payload.KindBoolean
	default:
		if wc.JSONType == payload.JSONTypeString && len(wc.EnumValues) == 0 {
			note("a %s channel sent as a JSON string now sends a JSON number", dt)
		}
		if kind == payload.KindBoolean {
			kind = payload.KindUnsigned
		}
	}
	format := wc.TimeFormat
	if kind == payload.KindTimestamp {
		format = validFormat(format, note)
	}
	return icd.MigrateJSONField(ctx, icdv0.LiftKind(
		icdv0.InlineField{EnumValues: wc.EnumValues, TimeFormat: format},
		kind,
		base,
		note,
	))
}

// liftRecordFields returns the fields that rec adds to the message of the channel named
// name: that name at the channel pointer, the time of the write at the timestamp
// pointer, and each static and generated field. names holds the names that the message
// takes. note receives the path and each change of a field.
func liftRecordFields(
	rec Record,
	name string,
	names set.Set[string],
	note func(path, change string),
) ([]icd.JSONField, error) {
	var fields []icd.JSONField
	if p := rec.ChannelPointer; p != nil {
		fields = append(fields, icd.JSONField{Variant: icd.StringJSONField{
			JSONBaseField: newBase(names, *p),
			Default:       new(name),
		}})
	}
	if p := rec.TimestampPointer; p != nil {
		format := payload.TimeFormatISO8601
		if f := validFormat(
			&rec.TimeFormat,
			notef(note, "record.time_format"),
		); f != nil {
			format = *f
		}
		fields = append(fields, icd.JSONField{Variant: icd.TimestampJSONField{
			JSONBaseField: newBase(names, *p),
			TimeFormat:    format,
			Default:       &icd.TimestampDefault{Variant: icd.NowTimestampDefault{}},
		}})
	}
	for i, wf := range rec.Fields {
		path := "record.fields." + strconv.Itoa(i)
		f, ok, err := liftWriteField(
			wf, newBase(names, wf.Base().Pointer), notef(note, path),
		)
		if err != nil {
			return nil, errors.Wrap(err, path)
		}
		if ok {
			fields = append(fields, f)
		}
	}
	return fields, nil
}

// liftWriteField returns the JSON field of base that sends what old sent, and false
// when old is null. A static field sends its value as a constant default, and a
// generated field sends a generator default.
func liftWriteField(
	old http.WriteField,
	base icd.JSONBaseField,
	note func(format string, args ...any),
) (icd.JSONField, bool, error) {
	switch v := old.Variant.(type) {
	case http.StaticWriteField:
		f, err := liftStatic(v.Value, base)
		return f, err == nil, err
	case http.GeneratedWriteField:
		if v.Generator != payload.GeneratorTypeTimestamp {
			return icd.JSONField{Variant: icd.UuidJSONField{
				JSONBaseField: base,
				Default:       &icd.UUIDDefault{Variant: icd.RandomUUIDDefault{}},
			}}, true, nil
		}
		format := payload.TimeFormatISO8601
		if f := validFormat(v.TimeFormat, note); f != nil {
			format = *f
		}
		return icd.JSONField{Variant: icd.TimestampJSONField{
			JSONBaseField: base,
			TimeFormat:    format,
			Default:       &icd.TimestampDefault{Variant: icd.NowTimestampDefault{}},
		}}, true, nil
	case nil:
		note("a null field was dropped")
		return icd.JSONField{}, false, nil
	}
	panic(fmt.Sprintf("unknown write field variant %T", old.Variant))
}

// liftStatic returns the JSON field of base whose default is value: a string, boolean,
// or float field for a value of that type, and a JSON field for any other value.
func liftStatic(value any, base icd.JSONBaseField) (icd.JSONField, error) {
	switch v := value.(type) {
	case string:
		return icd.JSONField{
			Variant: icd.StringJSONField{JSONBaseField: base, Default: &v},
		}, nil
	case bool:
		return icd.JSONField{
			Variant: icd.BooleanJSONField{JSONBaseField: base, Default: &v},
		}, nil
	}
	if n, ok := number(value); ok {
		return icd.JSONField{Variant: icd.FloatJSONField{
			JSONBaseField: base,
			Default:       new(payload.Float(n)),
		}}, nil
	}
	text, err := json.Marshal(value)
	if err != nil {
		return icd.JSONField{}, errors.Wrap(validate.ErrValidation, err.Error())
	}
	return icd.JSONField{Variant: icd.JSONJSONField{
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

// numeric reports whether k holds numbers or booleans.
func numeric(k payload.Kind) bool {
	return k == payload.KindFloat || k == payload.KindInteger ||
		k == payload.KindUnsigned || k == payload.KindBoolean
}

// newBase returns the base of a new field at pointer, named after it and unique in
// names.
func newBase(names set.Set[string], pointer string) icd.JSONBaseField {
	return icd.JSONBaseField{
		Key:     uuid.New(),
		Name:    uniqueName(names, pointerName(pointer)),
		Scale:   1,
		Pointer: pointer,
	}
}

// pointerName returns the name of the field at pointer.
func pointerName(pointer string) string {
	return lo.CoalesceOrEmpty(strings.TrimPrefix(pointer, "/"), "value")
}

// notef returns the note of each change at path, for note.
func notef(
	note func(path, change string),
	path string,
) func(format string, args ...any) {
	return func(format string, args ...any) { note(path, fmt.Sprintf(format, args...)) }
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
