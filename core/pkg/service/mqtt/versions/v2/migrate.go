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
	"math"
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

// liftField returns the JSON field of base for old, the data type of a new channel for
// it, and a description of each change. The data type of ch, when old has a channel,
// picks the kind. Otherwise the stored data type picks it, and a numeric kind keeps it
// as the data type of a new channel. The returned data type is empty for any other
// field. An index field lifts to the timestamp kind, and so does a field with a time
// format, no entries, and the default or an unknown data type. An entry that the kind
// cannot hold is dropped.
func liftField(
	old ReadField,
	index bool,
	ch *Channel,
	base icd.JSONBaseField,
) (icd.JSONField, telem.DataType, []string) {
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
	field := liftRead(old, kind, base, enumFilter{seen: set.New[string](), note: note})
	numeric := kind == payload.KindFloat || kind == payload.KindInteger ||
		kind == payload.KindUnsigned
	if !known || ch != nil || !numeric {
		dt = ""
	}
	return field, dt, changes
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

// liftRead returns the JSON field of base with kind and the options of old.
func liftRead(
	old ReadField,
	kind payload.Kind,
	base icd.JSONBaseField,
	f enumFilter,
) icd.JSONField {
	if field, ok := liftEnumRead(old, kind, base, f); ok {
		return field
	}
	for _, e := range old.EnumValues {
		f.drop(e, "a "+string(kind)+" field takes no numeric entries")
	}
	switch kind {
	case payload.KindString:
		return icd.JSONField{Variant: icd.StringJSONField{JSONBaseField: base}}
	case payload.KindTimestamp:
		format := payload.TimeFormatISO8601
		if old.TimeFormat != nil {
			format = *old.TimeFormat
		}
		return icd.JSONField{Variant: icd.TimestampJSONField{
			JSONBaseField: base,
			TimeFormat:    format,
		}}
	case payload.KindJSON:
		return icd.JSONField{Variant: icd.JSONJSONField{JSONBaseField: base}}
	case payload.KindUUID:
		return icd.JSONField{Variant: icd.UuidJSONField{JSONBaseField: base}}
	case payload.KindBytes:
		return icd.JSONField{Variant: icd.BytesJSONField{JSONBaseField: base}}
	case payload.KindFloat, payload.KindInteger, payload.KindUnsigned,
		payload.KindBoolean:
	}
	panic("unknown read field kind " + string(kind))
}

// liftEnumRead returns the JSON field of base for a kind that takes enum entries, with
// the entries of old that f keeps, and false for any other kind.
func liftEnumRead(
	old ReadField,
	kind payload.Kind,
	base icd.JSONBaseField,
	f enumFilter,
) (icd.JSONField, bool) {
	switch kind {
	case payload.KindFloat:
		return icd.JSONField{Variant: icd.FloatJSONField{
			JSONBaseField: base,
			EnumValues:    liftEntries(old.EnumValues, f, kind, floatEntry),
		}}, true
	case payload.KindInteger:
		return icd.JSONField{Variant: icd.IntegerJSONField{
			JSONBaseField: base,
			EnumValues:    liftEntries(old.EnumValues, f, kind, integerEntry),
		}}, true
	case payload.KindUnsigned:
		return icd.JSONField{Variant: icd.UnsignedJSONField{
			JSONBaseField: base,
			EnumValues:    liftEntries(old.EnumValues, f, kind, unsignedEntry),
		}}, true
	case payload.KindBoolean:
		return icd.JSONField{Variant: icd.BooleanJSONField{
			JSONBaseField: base,
			EnumValues:    liftEntries(old.EnumValues, f, kind, booleanEntry),
		}}, true
	case payload.KindString, payload.KindTimestamp, payload.KindJSON,
		payload.KindUUID, payload.KindBytes:
	}
	return icd.JSONField{}, false
}

// liftEntries returns the entries of old that f keeps, through entry. It drops each one
// whose value entry reports that kind does not hold.
func liftEntries[E any](
	old []payload.EnumEntry,
	f enumFilter,
	kind payload.Kind,
	entry func(payload.EnumEntry) (E, bool),
) []E {
	entries := make([]E, 0, len(old))
	for _, e := range old {
		lifted, ok := entry(e)
		switch {
		case !ok:
			f.drop(
				e,
				strconv.FormatFloat(e.Value, 'g', -1, 64)+" is not "+string(kind),
			)
		case f.usable(e):
			entries = append(entries, lifted)
		}
	}
	return entries
}

func floatEntry(e payload.EnumEntry) (payload.FloatEntry, bool) {
	return payload.FloatEntry{Label: e.Label, Value: payload.Float(e.Value)}, true
}

// integerEntry returns false when the value of e is not a whole number that int64
// holds.
func integerEntry(e payload.EnumEntry) (payload.IntegerEntry, bool) {
	if v := e.Value; v != math.Trunc(v) || v < math.MinInt64 || v >= 1<<63 {
		return payload.IntegerEntry{}, false
	}
	return payload.IntegerEntry{Label: e.Label, Value: payload.Integer(e.Value)}, true
}

// unsignedEntry returns false when the value of e is not a whole number that uint64
// holds.
func unsignedEntry(e payload.EnumEntry) (payload.UnsignedEntry, bool) {
	if v := e.Value; v != math.Trunc(v) || v < 0 || v >= 1<<64 {
		return payload.UnsignedEntry{}, false
	}
	return payload.UnsignedEntry{Label: e.Label, Value: payload.Unsigned(e.Value)}, true
}

func booleanEntry(e payload.EnumEntry) (payload.BooleanEntry, bool) {
	return payload.BooleanEntry{Label: e.Label, Value: e.Value != 0}, true
}

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
