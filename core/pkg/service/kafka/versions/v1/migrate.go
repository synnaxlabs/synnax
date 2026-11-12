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

	"github.com/samber/lo"
	http "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	payload "github.com/synnaxlabs/x/payload/versions/v1"
)

// MigrateWriteChannel lifts a v0 write channel to v1 by its stored shape, without its
// Synnax channel. A number with a time format and no entries, the only timestamp shape
// v0 ran, becomes a timestamp channel. Another number, or a string with entries,
// becomes a float channel that keeps the entries. A string with no entries stays a
// string, and a boolean becomes a boolean with no entries. The kind may not fit the
// Synnax channel, so the caller reconciles it against the channel.
func MigrateWriteChannel(_ context.Context, old v0.WriteChannel) (WriteChannel, error) {
	old.ApplyDefaults()
	base := BaseWriteChannel{Key: old.Key, Disabled: old.Disabled, Channel: old.Channel}
	var v WriteChannelVariant
	switch {
	case old.TimeFormat != nil && old.JSONType == payload.JSONTypeNumber &&
		len(old.EnumValues) == 0:
		v = TimestampWriteChannel{BaseWriteChannel: base, TimeFormat: *old.TimeFormat}
	case old.JSONType == payload.JSONTypeBoolean:
		v = BooleanWriteChannel{BaseWriteChannel: base}
	case old.JSONType == payload.JSONTypeString && len(old.EnumValues) == 0:
		v = StringWriteChannel{BaseWriteChannel: base}
	default:
		v = FloatWriteChannel{
			BaseWriteChannel: base,
			EnumValues: lo.Map(
				old.EnumValues,
				func(e payload.EnumEntry, _ int) payload.FloatEntry {
					return payload.FloatEntry{
						Label: e.Label,
						Value: payload.Float(e.Value),
					}
				},
			),
		}
	}
	return WriteChannel{Variant: v}, nil
}

// MigrateRecord lifts a v0 record shape to v1, with its fields lifted as
// migrateRecordFields does.
func MigrateRecord(ctx context.Context, old v0.Record) (Record, error) {
	return autoMigrateRecord(ctx, old)
}

// MigrateWriteConfig lifts a v0 write config and each of its channels to v1.
func MigrateWriteConfig(ctx context.Context, old v0.WriteConfig) (WriteConfig, error) {
	cfg, err := autoMigrateWriteConfig(ctx, old)
	if err != nil {
		return WriteConfig{}, err
	}
	if cfg.Channels, err = lo.MapErr(
		old.Channels,
		func(c v0.WriteChannel, _ int) (WriteChannel, error) {
			return MigrateWriteChannel(ctx, c)
		},
	); err != nil {
		return WriteConfig{}, err
	}
	return cfg, nil
}

// migrateRecordFields lifts the record fields of a released Core to typed write fields,
// as
// payload.LiftStaticWriteField and payload.LiftGeneratedWriteField do. A null field
// and a field with an unknown generator, which no Driver ran, drop.
func migrateRecordFields(
	_ context.Context,
	old []http.WriteField,
) ([]payload.WriteField, error) {
	var fields []payload.WriteField
	for _, o := range old {
		switch v := o.Variant.(type) {
		case http.StaticWriteField:
			f, err := payload.LiftStaticWriteField(
				payload.BaseWriteField(v.BaseWriteField),
				v.Value,
			)
			if err != nil {
				return nil, err
			}
			fields = append(fields, f)
		case http.GeneratedWriteField:
			if f, ok := payload.LiftGeneratedWriteField(
				payload.BaseWriteField(v.BaseWriteField),
				v.Generator,
				v.TimeFormat,
			); ok {
				fields = append(fields, f)
			}
		}
	}
	return fields, nil
}
