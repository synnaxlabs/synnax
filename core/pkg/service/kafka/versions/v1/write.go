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
	"fmt"
	"strconv"
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
)

// CreateWriteICD creates ic, an ICD in the current shape, through tx.
type CreateWriteICD func(ctx context.Context, tx gorp.Tx, ic icd.ICD) error

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
	channels icdv0.Channels,
	name string,
	cfg *WriteConfig,
	note func(path, change string),
) (icd.ICD, bool, error) {
	if cfg.Inline == nil {
		return icd.ICD{}, false, nil
	}
	in := cfg.Inline
	found, err := channels.Of(
		ctx,
		snap,
		lo.Map(in.Channels, func(wc WriteChannel, _ int) icdv0.InlineField {
			return icdv0.InlineField{Channel: wc.Channel}
		}),
	)
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
		path, chNote := "channels."+strconv.Itoa(i), note
		// Each message repeats the fields of the record, so only the first one notes
		// their changes.
		if i > 0 {
			chNote = func(p, change string) {
				if p == path {
					note(p, change)
				}
			}
		}
		msg, t, err := liftTarget(ctx, in, wc, found, path, chNote)
		if err != nil {
			return icd.ICD{}, false, err
		}
		msg.Name = icdv0.UniqueName(
			names,
			lo.CoalesceOrEmpty(found[wc.Channel].Name, "Untitled"),
		)
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
func NewWriteMigration(
	channels icdv0.Channels,
	create CreateWriteICD,
) gorp.Migration {
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
	channels icdv0.Channels,
	create CreateWriteICD,
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

// liftTarget returns the message that sends wc in the shape of the record of in, with
// no name, and the target that sends it, with no ICD. found holds the channel of wc
// when it exists, and path is the path of wc. note receives the path and each change of
// a field.
func liftTarget(
	ctx context.Context,
	in *InlineWrite,
	wc WriteChannel,
	found map[channel.Key]icdv0.Channel,
	path string,
	note func(path, change string),
) (icd.Message, WriteTarget, error) {
	name := found[wc.Channel].Name
	msg, write, err := icd.LiftWrite(
		ctx, inlineWrite(in, wc, name, path, note), found, note,
	)
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
		t.RecordKey = name
	}
	return msg, t, nil
}

// inlineWrite returns the write that sends wc, of the channel named name at path, in
// the shape of the record of in. A null field of the record is dropped, and note
// receives its path.
func inlineWrite(
	in *InlineWrite,
	wc WriteChannel,
	name string,
	path string,
	note func(path, change string),
) icd.InlineWrite {
	rec := in.Record
	fields := make([]icd.InlineWriteField, 0, 2+len(rec.Fields))
	if p := rec.ChannelPointer; p != nil {
		fields = append(fields, icd.StaticInlineField{
			Path:    "record.channel_pointer",
			Pointer: *p,
			Value:   name,
		})
	}
	if p := rec.TimestampPointer; p != nil {
		fields = append(fields, icd.GeneratedInlineField{
			Path:       "record.time_format",
			Pointer:    *p,
			Generator:  payload.GeneratorTypeTimestamp,
			TimeFormat: &rec.TimeFormat,
		})
	}
	for i, f := range rec.Fields {
		fieldPath := "record.fields." + strconv.Itoa(i)
		switch v := f.Variant.(type) {
		case http.StaticWriteField:
			fields = append(fields, icd.StaticInlineField{
				Path:    fieldPath,
				Pointer: v.Pointer,
				Value:   v.Value,
			})
		case http.GeneratedWriteField:
			fields = append(fields, icd.GeneratedInlineField{
				Path:       fieldPath,
				Pointer:    v.Pointer,
				Generator:  v.Generator,
				TimeFormat: v.TimeFormat,
			})
		case nil:
			note(fieldPath, "a null field was dropped")
		default:
			panic(fmt.Sprintf("unknown write field variant %T", f.Variant))
		}
	}
	return icd.InlineWrite{
		Target: "topic " + in.Topic,
		Channel: icd.InlineChannelField{
			Path:       path,
			Pointer:    rec.ValuePointer,
			JSONType:   wc.JSONType,
			Channel:    wc.Channel,
			TimeFormat: wc.TimeFormat,
			EnumValues: wc.EnumValues,
		},
		Fields: fields,
	}
}
