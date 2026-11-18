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
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// MigrateReadField lifts a v1 read field to v2, and describes each entry it drops. The
// stored data type picks the kind, and a numeric kind keeps it as the data type of a
// new channel. A float entry converts to an integer or unsigned kind when the kind
// holds it exactly. A boolean entry becomes true when its value is not 0. A string
// field drops its entries, and a timestamp field with no time format takes ISO 8601.
// Each kind but string drops an entry with an empty label, which it can never match.
// An empty data type is the v1 default, float64. An unknown one lifts to the float
// kind with no data type, so that reconciling with the channel can set the kind. The
// index field of its entry lifts to the timestamp kind. So does a field with a time
// format, no entries, and the default or an unknown data type, and it keeps the format,
// because v1 left the data type at its default and read the channel's data type. On
// any other field, the time format is left over from an earlier data type, so the
// field keeps its kind and entries.
func MigrateReadField(old v1.ReadField, index bool) (ReadField, []string) {
	return migrateReadField(old, index, nil)
}

// migrateReadField is MigrateReadField with the field's channel, when known. The
// channel's data type then picks the kind, as v1 read it, and a numeric kind takes no
// data type, because the channel exists.
func migrateReadField(
	old v1.ReadField,
	index bool,
	ch *Channel,
) (ReadField, []string) {
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
	return NewReadField(BaseReadField{
		Key:      old.Key,
		Name:     old.Name,
		Disabled: old.Disabled,
		Channel:  old.Channel,
		Pointer:  old.Pointer,
	}, migrateRead(old, kind, dt, known && ch == nil, note)), changes
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

// migrateRead returns the read options of kind for old and notes each enum entry that
// it drops. A typed number read keeps dt as its data type.
func migrateRead(
	old v1.ReadField,
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
	case payload.KindString, payload.KindJSON, payload.KindUUID, payload.KindBytes:
	}
	if kind != payload.KindFloat && kind != payload.KindInteger &&
		kind != payload.KindUnsigned && kind != payload.KindBoolean {
		for _, e := range old.EnumValues {
			filter.drop(e, "a "+string(kind)+" field takes no numeric entries")
		}
	}
	return r
}

// migrateReadConfig lifts a v1 read config to v2, with the channels of its fields in
// found, and calls note with the path and a description of each change to a field.
func migrateReadConfig(
	ctx context.Context,
	old v1.ReadConfig,
	found map[channel.Key]Channel,
	note func(field, change string),
) (ReadConfig, error) {
	cfg, err := autoMigrateReadConfig(ctx, old)
	if err != nil {
		return ReadConfig{}, err
	}
	for i, e := range old.Entries {
		plain, ok := e.Variant.(v1.PlainReadEntry)
		if !ok {
			continue
		}
		fields := make([]ReadField, len(plain.Fields))
		for j, f := range plain.Fields {
			path := "entries." + strconv.Itoa(i) + ".fields." + strconv.Itoa(j)
			var changes []string
			var ch *Channel
			if c, ok := found[f.Channel]; ok {
				ch = &c
			}
			fields[j], changes = migrateReadField(f, f.Key == plain.Index, ch)
			for _, c := range changes {
				note(path, c)
			}
		}
		entry, ok := cfg.Entries[i].Variant.(PlainReadEntry)
		if !ok {
			return ReadConfig{}, errors.Newf("entries.%d: expected a plain entry", i)
		}
		entry.Fields = fields
		cfg.Entries[i].Variant = entry
	}
	return cfg, nil
}

// MigrateReadConfig lifts a v1 read config to v2, as MigrateReadField lifts each of its
// fields.
func MigrateReadConfig(ctx context.Context, old v1.ReadConfig) (ReadConfig, error) {
	return migrateReadConfig(ctx, old, nil, func(string, string) {})
}

// noteLift returns the note that logs each change the v1 lift makes to a field of the
// config with key.
func noteLift(ins alamos.Instrumentation, key uuid.UUID) func(field, change string) {
	return func(field, change string) {
		ins.L.Warn(
			"changed an MQTT read field to v2",
			zap.Stringer("config", key),
			zap.String("field", field),
			zap.String("change", change),
		)
	}
}

// NewReadMigration returns the migration that lifts stored read configs to v2, with the
// kind of each field's channel read through channels, then reconciles each field with
// its channel. It logs each change to a field.
func NewReadMigration(channels Channels) gorp.Migration {
	return migrate.New(
		"v62_mqtt_read_typed_fields",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var olds []v1.ReadConfig
			if err := gorp.NewRetrieve[uuid.UUID, v1.ReadConfig]().
				Entries(&olds).
				Exec(ctx, tx); err != nil {
				return err
			}
			news := make([]ReadConfig, 0, len(olds))
			for _, old := range olds {
				found, err := v1FieldChannels(ctx, tx, channels, old)
				if err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				cfg, err := migrateReadConfig(ctx, old, found, noteLift(ins, old.Key))
				if err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				if err = ReconcileReadConfig(ctx, tx, channels, ins, &cfg); err != nil {
					return errors.Wrapf(err, "config %s", old.Key)
				}
				news = append(news, cfg)
			}
			return gorp.WrapWriter[uuid.UUID, ReadConfig](tx).Set(ctx, news...)
		},
	)
}

func v1FieldChannels(
	ctx context.Context,
	tx gorp.Tx,
	channels Channels,
	cfg v1.ReadConfig,
) (map[channel.Key]Channel, error) {
	var keys []channel.Key
	for _, e := range cfg.Entries {
		if plain, ok := e.Variant.(v1.PlainReadEntry); ok {
			for _, f := range plain.Fields {
				if f.Channel != 0 {
					keys = append(keys, f.Channel)
				}
			}
		}
	}
	if len(keys) == 0 {
		return nil, nil
	}
	return channels(ctx, tx, lo.Uniq(keys))
}
