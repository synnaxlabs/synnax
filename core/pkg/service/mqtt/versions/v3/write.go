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
	"fmt"
	"slices"
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	icdv0 "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v1"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
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
		msg, write, err := icd.LiftWrite(ctx, inlineWrite(t, note), found, note)
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

// inlineWrite returns the write that t held inline. Each field takes its path in the
// config. A null field is dropped, and note receives its path.
func inlineWrite(t InlineTarget, note func(path, change string)) icd.InlineWrite {
	var (
		path   = "targets." + strconv.Itoa(int(t.At))
		c      = t.Channel
		fields = make([]icd.InlineWriteField, 0, len(t.Fields))
	)
	for i, f := range t.Fields {
		fieldPath := path + ".fields." + strconv.Itoa(i)
		switch v := f.Variant.(type) {
		case v1.StaticWriteField:
			fields = append(fields, icd.StaticInlineField{
				Path:    fieldPath,
				Pointer: v.Pointer,
				Value:   v.Value,
			})
		case v1.GeneratedWriteField:
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
		Target: "topic " + t.Topic,
		Channel: icd.InlineChannelField{
			Path:       path + ".channel",
			Pointer:    c.Pointer,
			JSONType:   c.JSONType,
			Channel:    c.Channel,
			Name:       c.Name,
			DataType:   c.DataType,
			TimeFormat: c.TimeFormat,
			EnumValues: c.EnumValues,
		},
		Fields: fields,
	}
}
