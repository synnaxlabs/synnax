// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/errors"
	xpayload "github.com/synnaxlabs/x/payload"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// MigrateWriteTarget lifts a v2 write target to v3. A plain target has no ICD until
// MigrateWriteConfig stages it.
func MigrateWriteTarget(ctx context.Context, old v2.WriteTarget) (WriteTarget, error) {
	return autoMigrateWriteTarget(ctx, old)
}

// MigrateWriteConfig lifts a v2 write config to v3. It moves each plain target into
// Inline, for ResolveWriteConfig to add back with an ICD.
func MigrateWriteConfig(ctx context.Context, old v2.WriteConfig) (WriteConfig, error) {
	cfg, err := autoMigrateWriteConfig(ctx, old)
	if err != nil {
		return WriteConfig{}, err
	}
	var (
		targets = make([]WriteTarget, 0, len(cfg.Targets))
		staged  []InlineTarget
	)
	for i, t := range old.Targets {
		plain, ok := t.Variant.(v1.PlainWriteTarget)
		if !ok {
			targets = append(targets, cfg.Targets[i])
			continue
		}
		staged = append(staged, InlineTarget{
			BaseWriteTarget: plain.BaseWriteTarget,
			At:              uint32(i),
			Topic:           plain.Topic,
			Qos:             plain.Qos,
			Retained:        plain.Retained,
			Channel:         plain.Channel,
			Fields:          plain.Fields,
		})
	}
	if len(staged) > 0 {
		cfg.Targets, cfg.Inline = targets, &InlineWrite{Targets: staged}
	}
	return cfg, nil
}

// CreateICD creates ic through tx.
type CreateICD func(ctx context.Context, tx gorp.Tx, ic icd.ICD) error

// NoteLift returns the note for ResolveWriteConfig that logs each change to a field of
// the config with key through ins.
func NoteLift(ins alamos.Instrumentation, key uuid.UUID) func(path, change string) {
	return func(path, change string) {
		ins.L.Warn(
			"changed an MQTT write field to v3",
			zap.Stringer("config", key),
			zap.String("field", path),
			zap.String("change", change),
		)
	}
}

// ResolveWriteConfig moves the payloads of the inline targets of cfg into a new ICD
// named name, with one JSON message for each target, named after its topic, and adds
// each target back to the targets of cfg, sending its message. It returns the ICD,
// which the caller must create, and false when cfg holds no inline targets. The channel
// field of a target takes the kind of its channel, read through snap with channels.
// When that channel does not exist, its binding holds no channel and the data type of a
// new one. note receives the path and each change of a field.
func ResolveWriteConfig(
	ctx context.Context,
	snap gorp.Snapshot,
	channels icdv0.Channels,
	name string,
	cfg *WriteConfig,
	note func(path, change string),
) (icd.ICD, bool, error) {
	if cfg.Inline == nil {
		return icd.ICD{}, false, nil
	}
	staged := cfg.Inline.Targets
	found, err := channels.Of(
		ctx,
		snap,
		lo.Map(staged, func(t InlineTarget, _ int) icdv0.InlineField {
			return icdv0.InlineField{Channel: t.Channel.Channel}
		}),
	)
	if err != nil {
		return icd.ICD{}, false, err
	}
	ic := icd.ICD{
		Key:      uuid.New(),
		Name:     name,
		Messages: make([]icd.Message, len(staged)),
	}
	names := make(set.Set[string])
	for i, t := range staged {
		msg, write, err := liftTarget(ctx, t, found, note)
		if err != nil {
			return icd.ICD{}, false, err
		}
		msg.Name = icdv0.UniqueName(names, lo.CoalesceOrEmpty(t.Topic, "Untitled"))
		ic.Messages[i] = msg
		cfg.Targets = slices.Insert(
			cfg.Targets,
			min(int(t.At), len(cfg.Targets)),
			WriteTarget{Variant: PlainWriteTarget{
				BaseWriteTarget: t.BaseWriteTarget,
				Topic:           t.Topic,
				Qos:             t.Qos,
				Retained:        t.Retained,
				ICD:             ic.Key,
				Message:         write,
			}},
		)
	}
	cfg.Inline = nil
	return ic, true, nil
}

// NewWriteMigration returns the migration that lifts stored v2 write configs to v3. It
// resolves each config as ResolveWriteConfig does, with the name of its task, creates
// the ICD with create, and logs each change to a field. The config of a snapshot task
// gets no ICD and loses its plain targets, because the task service deletes snapshots
// after this migration runs.
func NewWriteMigration(channels icdv0.Channels, create CreateICD) gorp.Migration {
	return migrate.New(
		"v63_mqtt_write_icd",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var olds []v2.WriteConfig
			if err := gorp.NewRetrieve[uuid.UUID, v2.WriteConfig]().
				Entries(&olds).
				Exec(ctx, tx); err != nil || len(olds) == 0 {
				return err
			}
			tasks, err := retrieveTasks(ctx, tx, olds)
			if err != nil {
				return err
			}
			news := make([]WriteConfig, len(olds))
			for i, old := range olds {
				if news[i], err = migrateStored(
					ctx,
					tx,
					channels,
					create,
					tasks[old.Key],
					old,
					NoteLift(ins, old.Key),
				); err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
			}
			return gorp.WrapWriter[uuid.UUID, WriteConfig](tx).Set(ctx, news...)
		},
	)
}

func migrateStored(
	ctx context.Context,
	tx gorp.Tx,
	channels icdv0.Channels,
	create CreateICD,
	t task.Task,
	old v2.WriteConfig,
	note func(path, change string),
) (WriteConfig, error) {
	cfg, err := MigrateWriteConfig(ctx, old)
	if err != nil {
		return WriteConfig{}, err
	}
	if t.Snapshot {
		return cfg, nil
	}
	ic, ok, err := ResolveWriteConfig(ctx, tx, channels, t.Name, &cfg, note)
	if err != nil {
		return WriteConfig{}, err
	}
	if !ok {
		return cfg, nil
	}
	if err = create(ctx, tx, ic); err != nil {
		return WriteConfig{}, err
	}
	return cfg, nil
}

// retrieveTasks returns the task of each config of cfgs by key, read through tx.
func retrieveTasks(
	ctx context.Context,
	tx gorp.Tx,
	cfgs []v2.WriteConfig,
) (map[uuid.UUID]task.Task, error) {
	var tasks []task.Task
	if err := gorp.NewRetrieve[task.Key, task.Task]().
		Where(gorp.MatchKeys[task.Key, task.Task](
			lo.Map(cfgs, func(c v2.WriteConfig, _ int) task.Key { return c.Key })...,
		)).
		Entries(&tasks).
		Exec(ctx, tx); err != nil {
		return nil, errors.Wrap(err, "tasks of MQTT write configs")
	}
	return lo.KeyBy(tasks, func(t task.Task) uuid.UUID { return t.Key }), nil
}

// liftTarget returns the message that holds the channel field and the fields of t, with
// no name, and the write of that message.
func liftTarget(
	ctx context.Context,
	t InlineTarget,
	found map[channel.Key]icdv0.Channel,
	note func(path, change string),
) (icd.Message, icd.WriteMessage, error) {
	var (
		path   = "targets." + strconv.Itoa(int(t.At))
		names  = make(set.Set[string])
		fields = make([]icd.JSONField, 0, 1+len(t.Fields))
		write  = icd.WriteMessage{Message: uuid.New()}
		key    = uuid.New()
	)
	notef := func(path string) func(format string, args ...any) {
		return func(format string, args ...any) {
			note(path, fmt.Sprintf(format, args...))
		}
	}
	ch, exists := found[t.Channel.Channel]
	field, dataType, err := liftChannel(ctx, t.Channel, ch, exists, icd.JSONBaseField{
		Key:     key,
		Name:    icdv0.UniqueName(names, pointerName(t.Channel.Pointer)),
		Scale:   1,
		Pointer: t.Channel.Pointer,
	}, notef(path+".channel"))
	if err != nil {
		return icd.Message{}, icd.WriteMessage{}, err
	}
	fields = append(fields, field)
	binding := icd.ChannelWriteBinding{Field: key, Channel: t.Channel.Channel}
	if !exists {
		binding.Channel, binding.DataType = 0, dataType
		if t.Channel.Name != "" {
			note(path+".channel", fmt.Sprintf(
				"channel name %q of field %s on topic %s was dropped, so a client "+
					"names the new channel",
				t.Channel.Name, field.Base().Name, t.Topic,
			))
		}
	}
	write.Bindings = []icd.WriteBinding{{Variant: binding}}
	for i, wf := range t.Fields {
		base := icd.JSONBaseField{
			Key:     uuid.New(),
			Name:    icdv0.UniqueName(names, pointerName(wf.Base().Pointer)),
			Scale:   1,
			Pointer: wf.Base().Pointer,
		}
		f, ok, err := liftWriteField(wf, base, notef(path+".fields."+strconv.Itoa(i)))
		if err != nil {
			return icd.Message{}, icd.WriteMessage{}, err
		}
		if ok {
			fields = append(fields, f)
		}
	}
	return icd.Message{
		Key:     write.Message,
		Payload: icd.Payload{Variant: icd.JSONPayload{Fields: fields}},
	}, write, nil
}

// liftChannel returns the JSON field of base that sends old the way v1 sent it, and the
// data type of a new channel for it, or nil when that data type is not known. The data
// type of ch, when old's channel exists, picks the kind. Otherwise the stored data type
// picks it, and is the data type of a new channel. note receives each change to what
// the field sends.
func liftChannel(
	ctx context.Context,
	old v1.ChannelField,
	ch icdv0.Channel,
	exists bool,
	base icd.JSONBaseField,
	note func(format string, args ...any),
) (icd.JSONField, *telem.DataType, error) {
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
	field, err := icd.MigrateJSONField(ctx, icdv0.LiftKind(
		icdv0.InlineField{EnumValues: old.EnumValues, TimeFormat: format},
		kind,
		base,
		note,
	))
	if err != nil {
		return icd.JSONField{}, nil, err
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

// liftWriteField returns the JSON field of base that sends what old sent, and false
// when old is null. A static field sends its value as a constant default, and a
// generated field sends a generator default.
func liftWriteField(
	old v1.WriteField,
	base icd.JSONBaseField,
	note func(format string, args ...any),
) (icd.JSONField, bool, error) {
	switch v := old.Variant.(type) {
	case v1.StaticWriteField:
		f, err := liftStatic(v.Value, base)
		return f, err == nil, err
	case v1.GeneratedWriteField:
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

// pointerName returns the name of the field at pointer.
func pointerName(pointer string) string {
	return lo.CoalesceOrEmpty(strings.TrimPrefix(pointer, "/"), "value")
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
