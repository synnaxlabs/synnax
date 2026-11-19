// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// MigrateWriteTarget lifts a v1 write target to v2. It returns an error wrapping
// validate.ErrValidation for a plain target, which needs an ICD.
func MigrateWriteTarget(ctx context.Context, old v1.WriteTarget) (WriteTarget, error) {
	if plain, ok := old.Variant.(v1.PlainWriteTarget); ok {
		return WriteTarget{}, errors.Wrapf(
			validate.ErrValidation,
			"plain target %s needs an ICD",
			plain.Topic,
		)
	}
	return autoMigrateWriteTarget(ctx, old)
}

// MigrateWriteConfig lifts a v1 write config to v2.
func MigrateWriteConfig(ctx context.Context, old v1.WriteConfig) (WriteConfig, error) {
	return autoMigrateWriteConfig(ctx, old)
}

// NewWriteMigration returns the migration that lifts stored v1 write configs to v2. It
// moves the plain targets of each config into a new ICD named after its task, with one
// JSON message for each target, named after its topic. It creates the ICD with create,
// and logs each change to a field. A field takes the kind of its channel, read with
// channels. The config of a snapshot task gets no ICD and loses its plain targets.
func NewWriteMigration(channels Channels, create CreateICD) gorp.Migration {
	return migrate.New(
		"v62_mqtt_write_icd",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var olds []v1.WriteConfig
			if err := gorp.NewRetrieve[uuid.UUID, v1.WriteConfig]().
				Entries(&olds).
				Exec(ctx, tx); err != nil || len(olds) == 0 {
				return err
			}
			tasks, err := retrieveTasks(ctx, tx, configKeys(olds), "write")
			if err != nil {
				return err
			}
			news := make([]WriteConfig, len(olds))
			for i, old := range olds {
				if news[i], err = migrateStoredWrite(
					ctx,
					tx,
					channels,
					create,
					tasks[old.Key],
					old,
					NoteLift(ins, "write", old.Key),
				); err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
			}
			return gorp.WrapWriter[uuid.UUID, WriteConfig](tx).Set(ctx, news...)
		},
	)
}

func migrateStoredWrite(
	ctx context.Context,
	tx gorp.Tx,
	channels Channels,
	create CreateICD,
	t task.Task,
	old v1.WriteConfig,
	note func(path, change string),
) (WriteConfig, error) {
	cfg := WriteConfig{
		StartConfig: old.StartConfig,
		Device:      old.Device,
		Targets:     make([]WriteTarget, 0, len(old.Targets)),
	}
	found, err := targetChannels(ctx, tx, channels, old.Targets)
	if err != nil {
		return WriteConfig{}, err
	}
	var (
		ic    = icd.ICD{Key: uuid.New(), Name: t.Name}
		names = make(set.Set[string])
	)
	for i, target := range old.Targets {
		plain, ok := target.Variant.(v1.PlainWriteTarget)
		if !ok {
			migrated, err := autoMigrateWriteTarget(ctx, target)
			if err != nil {
				return WriteConfig{}, err
			}
			cfg.Targets = append(cfg.Targets, migrated)
			continue
		}
		if t.Snapshot {
			continue
		}
		msg, write := liftTarget(plain, found, "targets."+strconv.Itoa(i), note)
		msg.Name = uniqueName(names, lo.CoalesceOrEmpty(plain.Topic, "Untitled"))
		ic.Messages = append(ic.Messages, msg)
		cfg.Targets = append(cfg.Targets, WriteTarget{Variant: PlainWriteTarget{
			BaseWriteTarget: plain.BaseWriteTarget,
			Topic:           plain.Topic,
			Qos:             plain.Qos,
			Retained:        plain.Retained,
			ICD:             ic.Key,
			Message:         write,
		}})
	}
	if len(ic.Messages) > 0 {
		if err = create(ctx, tx, ic); err != nil {
			return WriteConfig{}, err
		}
	}
	return cfg, nil
}

// liftTarget returns the message that holds the channel field and the fields of t,
// with no name, and the write of that message. path is the path of t.
func liftTarget(
	t v1.PlainWriteTarget,
	found map[channel.Key]Channel,
	path string,
	note func(path, change string),
) (icd.Message, WriteMessage) {
	var (
		names  = make(set.Set[string])
		fields = make([]icd.JSONField, 0, 1+len(t.Fields))
		write  = WriteMessage{Message: uuid.New()}
		key    = uuid.New()
		ch     *Channel
	)
	if c, exists := found[t.Channel.Channel]; exists {
		ch = &c
	}
	field, dataType := liftChannel(t.Channel, ch, icd.JSONBaseField{
		Key:     key,
		Name:    uniqueName(names, pointerName(t.Channel.Pointer)),
		Scale:   1,
		Pointer: t.Channel.Pointer,
	}, func(format string, args ...any) {
		note(path+".channel", fmt.Sprintf(format, args...))
	})
	fields = append(fields, field)
	b := WriteBinding{Field: key, DataType: dataType}
	if ch != nil {
		b = WriteBinding{Field: key, Channel: t.Channel.Channel}
	}
	write.Bindings = []WriteBinding{b}
	for i, wf := range t.Fields {
		base := icd.JSONBaseField{
			Key:     uuid.New(),
			Name:    uniqueName(names, pointerName(wf.Base().Pointer)),
			Scale:   1,
			Pointer: wf.Base().Pointer,
		}
		f, ok := liftWriteField(wf, base, func(format string, args ...any) {
			note(path+".fields."+strconv.Itoa(i), fmt.Sprintf(format, args...))
		})
		if ok {
			fields = append(fields, f)
		}
	}
	return icd.Message{
		Key:     write.Message,
		Payload: icd.Payload{Variant: icd.JSONPayload{Fields: fields}},
	}, write
}

// liftChannel returns the JSON field of base that sends old the way v1 sent it, and the
// data type of a new channel for it. The data type of ch, when old has a channel, picks
// the kind. Otherwise the stored data type picks it, and is the data type of a new
// channel. note receives each change to what the field sends.
func liftChannel(
	old v1.ChannelField,
	ch *Channel,
	base icd.JSONBaseField,
	note func(format string, args ...any),
) (icd.JSONField, telem.DataType) {
	dt := lo.CoalesceOrEmpty(old.DataType, telem.Float64T)
	if ch != nil {
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
	field := liftKind(
		ReadField{EnumValues: old.EnumValues, TimeFormat: format},
		kind,
		base,
		enumFilter{seen: set.New[string](), note: note},
	)
	if !known {
		dt = ""
	}
	return field, dt
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
) (icd.JSONField, bool) {
	switch v := old.Variant.(type) {
	case v1.StaticWriteField:
		return liftStatic(v.Value, base), true
	case v1.GeneratedWriteField:
		if v.Generator != payload.GeneratorTypeTimestamp {
			return icd.JSONField{Variant: icd.UuidJSONField{
				JSONBaseField: base,
				Default:       &icd.UUIDDefault{Variant: icd.RandomUUIDDefault{}},
			}}, true
		}
		format := payload.TimeFormatISO8601
		if f := validFormat(v.TimeFormat, note); f != nil {
			format = *f
		}
		return icd.JSONField{Variant: icd.TimestampJSONField{
			JSONBaseField: base,
			TimeFormat:    format,
			Default:       &icd.TimestampDefault{Variant: icd.NowTimestampDefault{}},
		}}, true
	case nil:
		note("a null field was dropped")
		return icd.JSONField{}, false
	}
	panic(fmt.Sprintf("unknown write field variant %T", old.Variant))
}

// liftStatic returns the JSON field of base whose default is value: a string, boolean,
// or float field for a value of that type, and a JSON field for any other value.
func liftStatic(value any, base icd.JSONBaseField) icd.JSONField {
	switch v := value.(type) {
	case string:
		return icd.JSONField{
			Variant: icd.StringJSONField{JSONBaseField: base, Default: &v},
		}
	case bool:
		return icd.JSONField{
			Variant: icd.BooleanJSONField{JSONBaseField: base, Default: &v},
		}
	}
	if n, ok := number(value); ok {
		return icd.JSONField{Variant: icd.FloatJSONField{
			JSONBaseField: base,
			Default:       new(payload.Float(n)),
		}}
	}
	return icd.JSONField{
		Variant: icd.JSONJSONField{JSONBaseField: base, Default: &value},
	}
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

func targetChannels(
	ctx context.Context,
	snap gorp.Snapshot,
	channels Channels,
	targets []v1.WriteTarget,
) (map[channel.Key]Channel, error) {
	var keys []channel.Key
	for _, t := range targets {
		if plain, ok := t.Variant.(v1.PlainWriteTarget); ok &&
			plain.Channel.Channel != 0 {
			keys = append(keys, plain.Channel.Channel)
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	return channels(ctx, snap, lo.Uniq(keys))
}
