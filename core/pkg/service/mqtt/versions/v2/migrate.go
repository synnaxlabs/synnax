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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
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
// kind with no data type, so that reconciling with the channel can set the kind. A
// field with a time format lifts to the timestamp kind and keeps it, because v1 read
// the data type of the channel and used the time format only on a timestamp channel.
func MigrateReadField(old v1.ReadField) (ReadField, []string) {
	var changes []string
	dt := telem.DataType(old.DataType)
	if dt == "" {
		dt = telem.Float64T
	}
	kind, known := payload.KindOf(dt)
	switch {
	case old.TimeFormat != nil && kind != payload.KindTimestamp:
		kind, known = payload.KindTimestamp, true
		changes = append(changes, fmt.Sprintf(
			"data type %s became the timestamp kind, which its time format reads",
			old.DataType,
		))
	case !known:
		kind = payload.KindFloat
		changes = append(changes, fmt.Sprintf(
			"unknown data type %q became the float kind", old.DataType,
		))
	}
	r := payload.Read{Kind: kind}
	drop := func(e payload.EnumEntry, why string) {
		changes = append(
			changes,
			fmt.Sprintf("enum label %q was dropped: %s", e.Label, why),
		)
	}
	switch kind {
	case payload.KindFloat, payload.KindInteger, payload.KindUnsigned:
		if known {
			r.Number.DataType = &dt
		}
		for _, e := range old.EnumValues {
			v, ok := payload.FloatNumber(payload.Float(e.Value)).To(kind)
			switch {
			case e.Label == "":
				drop(e, "it is empty")
			case !ok:
				drop(
					e,
					strconv.FormatFloat(e.Value, 'g', -1, 64)+" is not "+string(kind),
				)
			default:
				r.Number.EnumValues = append(
					r.Number.EnumValues,
					payload.NumberEntry{Label: e.Label, Value: v},
				)
			}
		}
	case payload.KindBoolean:
		for _, e := range old.EnumValues {
			if e.Label == "" {
				drop(e, "it is empty")
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
	return NewReadField(BaseReadField{
		Key:      old.Key,
		Name:     old.Name,
		Disabled: old.Disabled,
		Channel:  old.Channel,
		Pointer:  old.Pointer,
	}, r), changes
}

// migrateReadConfig lifts a v1 read config to v2 and calls note with the path of each
// field that drops an entry, and a description of the drop.
func migrateReadConfig(
	ctx context.Context,
	old v1.ReadConfig,
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
			fields[j], changes = MigrateReadField(f)
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
	return migrateReadConfig(ctx, old, func(string, string) {})
}

// NewReadMigration returns the migration that lifts stored read configs to v2, then
// reconciles each field with its channel through channels. It logs each entry it drops
// and each change that reconciling makes.
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
				cfg, err := migrateReadConfig(ctx, old, func(field, change string) {
					ins.L.Warn(
						"changed an MQTT read field to v2",
						zap.Stringer("config", old.Key),
						zap.String("field", field),
						zap.String("change", change),
					)
				})
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
