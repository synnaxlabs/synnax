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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/errors"
	"go.uber.org/zap"
)

// MigrateReadEntry lifts a v1 read entry to v2. A plain entry has no ICD until
// MigrateReadConfig stages it.
func MigrateReadEntry(ctx context.Context, old v1.ReadEntry) (ReadEntry, error) {
	return autoMigrateReadEntry(ctx, old)
}

// MigrateReadConfig lifts a v1 read config to v2. It moves each plain entry into
// Inline, for ResolveReadConfig to add back with an ICD.
func MigrateReadConfig(ctx context.Context, old v1.ReadConfig) (ReadConfig, error) {
	cfg, err := autoMigrateReadConfig(ctx, old)
	if err != nil {
		return ReadConfig{}, err
	}
	var (
		entries = make([]ReadEntry, 0, len(cfg.Entries))
		staged  []InlineEntry
	)
	for i, e := range old.Entries {
		plain, ok := e.Variant.(v1.PlainReadEntry)
		if !ok {
			entries = append(entries, cfg.Entries[i])
			continue
		}
		staged = append(staged, InlineEntry{
			BaseReadEntry:   plain.BaseReadEntry,
			At:              uint32(i),
			Topic:           plain.Topic,
			Qos:             plain.Qos,
			RetainedIgnored: plain.RetainedIgnored,
			Fields:          plain.Fields,
			Index:           plain.Index,
		})
	}
	if len(staged) > 0 {
		cfg.Entries, cfg.Inline = entries, &InlineRead{Entries: staged}
	}
	return cfg, nil
}

// CreateICD creates ic through tx.
type CreateICD func(ctx context.Context, tx gorp.Tx, ic icd.ICD) error

// NoteLift returns the note for ResolveReadConfig that logs each change to a field of
// the config with key through ins.
func NoteLift(ins alamos.Instrumentation, key uuid.UUID) func(path, change string) {
	return func(path, change string) {
		ins.L.Warn(
			"changed an MQTT read field to v2",
			zap.Stringer("config", key),
			zap.String("field", path),
			zap.String("change", change),
		)
	}
}

// NewReadMigration returns the migration that lifts stored v1 read configs to v2. It
// resolves each config as ResolveReadConfig does, with the name of its task, creates
// the ICD with create, and logs each change to a field. The config of a snapshot task
// gets no ICD and loses its plain entries, because the task service deletes snapshots
// after this migration runs.
func NewReadMigration(channels icd.Channels, create CreateICD) gorp.Migration {
	return migrate.New(
		"v62_mqtt_read_icd",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var olds []v1.ReadConfig
			if err := gorp.NewRetrieve[uuid.UUID, v1.ReadConfig]().
				Entries(&olds).
				Exec(ctx, tx); err != nil || len(olds) == 0 {
				return err
			}
			tasks, err := retrieveTasks(ctx, tx, olds)
			if err != nil {
				return err
			}
			news := make([]ReadConfig, len(olds))
			for i, old := range olds {
				if tasks[old.Key].Snapshot {
					if news[i], err = MigrateReadConfig(ctx, old); err != nil {
						return errors.Wrapf(err, "config %s", old.Key)
					}
					continue
				}
				if news[i], err = migrateStored(
					ctx,
					tx,
					channels,
					create,
					tasks[old.Key].Name,
					old,
					NoteLift(ins, old.Key),
				); err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
			}
			return gorp.WrapWriter[uuid.UUID, ReadConfig](tx).Set(ctx, news...)
		},
	)
}

func migrateStored(
	ctx context.Context,
	tx gorp.Tx,
	channels icd.Channels,
	create CreateICD,
	name string,
	old v1.ReadConfig,
	note func(field, change string),
) (ReadConfig, error) {
	cfg, err := MigrateReadConfig(ctx, old)
	if err != nil {
		return ReadConfig{}, err
	}
	ic, ok, err := ResolveReadConfig(ctx, tx, channels, name, &cfg, note)
	if err != nil {
		return ReadConfig{}, err
	}
	if !ok {
		return cfg, nil
	}
	if err = create(ctx, tx, ic); err != nil {
		return ReadConfig{}, err
	}
	return cfg, nil
}

// retrieveTasks returns the task of each config of cfgs by key, read through tx.
func retrieveTasks(
	ctx context.Context,
	tx gorp.Tx,
	cfgs []v1.ReadConfig,
) (map[uuid.UUID]task.Task, error) {
	var tasks []task.Task
	if err := gorp.NewRetrieve[task.Key, task.Task]().
		Where(gorp.MatchKeys[task.Key, task.Task](
			lo.Map(cfgs, func(c v1.ReadConfig, _ int) task.Key { return c.Key })...,
		)).
		Entries(&tasks).
		Exec(ctx, tx); err != nil {
		return nil, errors.Wrap(err, "tasks of MQTT read configs")
	}
	return lo.KeyBy(tasks, func(t task.Task) uuid.UUID { return t.Key }), nil
}
