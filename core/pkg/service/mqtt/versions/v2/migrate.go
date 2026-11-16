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
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	task "github.com/synnaxlabs/synnax/pkg/service/task/versions/v3"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// liftField returns the kind and the options of old, and describes each change. The
// data type of ch, when old has a channel, picks the kind. Otherwise the stored data
// type picks it, and a numeric kind keeps it as the data type of a new channel. An
// index field lifts to the timestamp kind, and so does a field with a time format, no
// entries, and the default or an unknown data type. An entry that the kind cannot hold
// is dropped.
func liftField(old ReadField, index bool, ch *Channel) (payload.Read, []string) {
	var changes []string
	note := func(format string, args ...any) {
		changes = append(changes, fmt.Sprintf(format, args...))
	}
	dt := telem.DataType(old.DataType)
	if dt == "" {
		dt = telem.Float64T
	}
	if ch != nil && ch.DataType != dt {
		note("data type %s became %s, the data type of its channel", dt, ch.DataType)
		dt = ch.DataType
	}
	kind, known := payload.KindOf(dt)
	if !known {
		kind = payload.KindFloat
		note("unknown data type %q became the float kind", dt)
	}
	switch {
	case ch == nil && kind != payload.KindTimestamp &&
		(index || (old.TimeFormat != nil && len(old.EnumValues) == 0 &&
			(dt == telem.Float64T || !known))):
		kind = payload.KindTimestamp
		note("data type %s became the timestamp kind of an index or time format", dt)
	case old.TimeFormat != nil && kind != payload.KindTimestamp:
		note(
			"time format %s was dropped, because the field is %s",
			*old.TimeFormat,
			kind,
		)
	}
	r := payload.Read{Kind: kind}
	drop := func(e payload.EnumEntry, why string) {
		note("enum label %q was dropped: %s", e.Label, why)
	}
	seen := set.New[string]()
	// usable reports whether e is an entry the lifted field can keep: its label is not
	// empty, and no earlier entry has it.
	usable := func(e payload.EnumEntry) bool {
		if e.Label == "" {
			drop(e, "it is empty")
			return false
		}
		if seen.Contains(e.Label) {
			drop(e, "it repeats an earlier label")
			return false
		}
		seen.Add(e.Label)
		return true
	}
	switch kind {
	case payload.KindFloat, payload.KindInteger, payload.KindUnsigned:
		if known && ch == nil {
			r.Number.DataType = &dt
		}
		for _, e := range old.EnumValues {
			v, ok := payload.FloatNumber(payload.Float(e.Value)).To(kind)
			switch {
			case !ok:
				drop(
					e,
					strconv.FormatFloat(e.Value, 'g', -1, 64)+" is not "+string(kind),
				)
			case usable(e):
				r.Number.EnumValues = append(
					r.Number.EnumValues,
					payload.NumberEntry{Label: e.Label, Value: v},
				)
			}
		}
	case payload.KindBoolean:
		for _, e := range old.EnumValues {
			if !usable(e) {
				continue
			}
			r.Boolean.EnumValues = append(
				r.Boolean.EnumValues,
				payload.BooleanEntry{Label: e.Label, Value: e.Value != 0},
			)
		}
	case payload.KindTimestamp:
		r.Timestamp.TimeFormat = payload.TimeFormatISO8601
		if old.TimeFormat != nil {
			r.Timestamp.TimeFormat = *old.TimeFormat
		}
	}
	if kind != payload.KindFloat && kind != payload.KindInteger &&
		kind != payload.KindUnsigned && kind != payload.KindBoolean {
		for _, e := range old.EnumValues {
			drop(e, "a "+string(kind)+" field takes no numeric entries")
		}
	}
	return r, changes
}

// MigrateReadEntry lifts a v1 read entry to v2. A plain entry has no ICD until
// MigrateReadConfig stages it.
func MigrateReadEntry(ctx context.Context, old v1.ReadEntry) (ReadEntry, error) {
	return autoMigrateReadEntry(ctx, old)
}

// MigrateReadConfig lifts a v1 read config to v2. It moves each plain entry into
// Inline,
// for ResolveReadConfig to add back with an ICD.
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
// the ICD with create, and logs each change to a field.
func NewReadMigration(channels Channels, create CreateICD) gorp.Migration {
	return migrate.New(
		"v62_mqtt_read_icd",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var olds []v1.ReadConfig
			if err := gorp.NewRetrieve[uuid.UUID, v1.ReadConfig]().
				Entries(&olds).
				Exec(ctx, tx); err != nil || len(olds) == 0 {
				return err
			}
			names, err := taskNames(ctx, tx, olds)
			if err != nil {
				return err
			}
			news := make([]ReadConfig, len(olds))
			for i, old := range olds {
				if news[i], err = migrateStored(
					ctx,
					tx,
					channels,
					create,
					names[old.Key],
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
	channels Channels,
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
	if err != nil || !ok {
		return cfg, err
	}
	return cfg, create(ctx, tx, ic)
}

// taskNames returns the name of the task of each config of cfgs, read through tx.
func taskNames(
	ctx context.Context,
	tx gorp.Tx,
	cfgs []v1.ReadConfig,
) (map[uuid.UUID]string, error) {
	var tasks []task.Task
	if err := gorp.NewRetrieve[task.Key, task.Task]().
		Where(gorp.MatchKeys[task.Key, task.Task](
			lo.Map(cfgs, func(c v1.ReadConfig, _ int) task.Key { return c.Key })...,
		)).
		Entries(&tasks).
		Exec(ctx, tx); err != nil {
		return nil, errors.Wrap(err, "tasks of MQTT read configs")
	}
	return lo.SliceToMap(tasks, func(t task.Task) (uuid.UUID, string) {
		return t.Key, t.Name
	}), nil
}
