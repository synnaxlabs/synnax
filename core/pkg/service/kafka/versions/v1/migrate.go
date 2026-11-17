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
	icd "github.com/synnaxlabs/synnax/pkg/service/icd/versions/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
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
	dt := old.DataType
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
	return liftRead(old, kind, dt, known && ch == nil, note), changes
}

// enumFilter keeps the enum entries that a lifted field can use and notes each one it
// drops.
type enumFilter struct {
	seen set.Set[string]
	note func(format string, args ...any)
}

func (f enumFilter) drop(e payload.EnumEntry, why string) {
	f.note("enum label %q was dropped: %s", e.Label, why)
}

// usable reports whether the lifted field can keep e: its label is not empty, and no
// earlier entry has it.
func (f enumFilter) usable(e payload.EnumEntry) bool {
	if e.Label == "" {
		f.drop(e, "it is empty")
		return false
	}
	if f.seen.Contains(e.Label) {
		f.drop(e, "it repeats an earlier label")
		return false
	}
	f.seen.Add(e.Label)
	return true
}

// liftRead returns the read options of kind for old and notes each enum entry that it
// drops. A typed number read keeps dt as its data type.
func liftRead(
	old ReadField,
	kind payload.Kind,
	dt telem.DataType,
	typed bool,
	note func(format string, args ...any),
) payload.Read {
	r := payload.Read{Kind: kind}
	filter := enumFilter{seen: set.New[string](), note: note}
	switch kind {
	case payload.KindFloat, payload.KindInteger, payload.KindUnsigned:
		if typed {
			r.Number.DataType = &dt
		}
		for _, e := range old.EnumValues {
			v, ok := payload.FloatNumber(payload.Float(e.Value)).To(kind)
			switch {
			case !ok:
				filter.drop(
					e,
					strconv.FormatFloat(e.Value, 'g', -1, 64)+" is not "+string(kind),
				)
			case filter.usable(e):
				r.Number.EnumValues = append(
					r.Number.EnumValues,
					payload.NumberEntry{Label: e.Label, Value: v},
				)
			}
		}
	case payload.KindBoolean:
		for _, e := range old.EnumValues {
			if !filter.usable(e) {
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
			filter.drop(e, "a "+string(kind)+" field takes no numeric entries")
		}
	}
	return r
}

// MigrateReadRecord lifts a v0 read record to v1. The record has no ICD until
// MigrateReadConfig stages it.
func MigrateReadRecord(ctx context.Context, old v0.ReadRecord) (ReadRecord, error) {
	return autoMigrateReadRecord(ctx, old)
}

// MigrateReadConfig lifts a v0 read config to v1. It moves each record into Inline, for
// ResolveReadConfig to add back with an ICD.
func MigrateReadConfig(ctx context.Context, old v0.ReadConfig) (ReadConfig, error) {
	cfg, err := autoMigrateReadConfig(ctx, old)
	if err != nil || len(old.Records) == 0 {
		return cfg, err
	}
	staged := make([]InlineRecord, len(old.Records))
	for i, r := range old.Records {
		staged[i] = InlineRecord{
			Key:       r.Key,
			RecordKey: r.RecordKey,
			Index:     r.Index,
			Fields:    r.Fields,
		}
	}
	cfg.Records, cfg.Inline = []ReadRecord{}, &InlineRead{Records: staged}
	return cfg, nil
}

// CreateICD creates ic through tx.
type CreateICD func(ctx context.Context, tx gorp.Tx, ic icd.ICD) error

// NoteLift returns the note for ResolveReadConfig that logs each change to a field of
// the config with key through ins.
func NoteLift(ins alamos.Instrumentation, key uuid.UUID) func(path, change string) {
	return func(path, change string) {
		ins.L.Warn(
			"changed a Kafka read field to v1",
			zap.Stringer("config", key),
			zap.String("field", path),
			zap.String("change", change),
		)
	}
}

// NewReadMigration returns the migration that lifts stored v0 read configs to v1. It
// resolves each config as ResolveReadConfig does, with the name of its task, creates
// the ICD with create, and logs each change to a field.
func NewReadMigration(channels Channels, create CreateICD) gorp.Migration {
	return migrate.New(
		"v62_kafka_read_icd",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var olds []v0.ReadConfig
			if err := gorp.NewRetrieve[uuid.UUID, v0.ReadConfig]().
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
	old v0.ReadConfig,
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
	cfgs []v0.ReadConfig,
) (map[uuid.UUID]string, error) {
	var tasks []task.Task
	if err := gorp.NewRetrieve[task.Key, task.Task]().
		Where(gorp.MatchKeys[task.Key, task.Task](
			lo.Map(cfgs, func(c v0.ReadConfig, _ int) task.Key { return c.Key })...,
		)).
		Entries(&tasks).
		Exec(ctx, tx); err != nil {
		return nil, errors.Wrap(err, "tasks of Kafka read configs")
	}
	return lo.SliceToMap(tasks, func(t task.Task) (uuid.UUID, string) {
		return t.Key, t.Name
	}), nil
}
