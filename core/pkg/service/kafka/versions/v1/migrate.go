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
	"math"
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	http "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	"github.com/synnaxlabs/x/errors"
	xjson "github.com/synnaxlabs/x/json"
	xpayload "github.com/synnaxlabs/x/payload"
	payloadv0 "github.com/synnaxlabs/x/payload/versions/v0"
	payload "github.com/synnaxlabs/x/payload/versions/v1"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
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
	case old.TimeFormat != nil && old.JSONType == payloadv0.JSONTypeNumber &&
		len(old.EnumValues) == 0:
		v = TimestampWriteChannel{BaseWriteChannel: base, TimeFormat: *old.TimeFormat}
	case old.JSONType == payloadv0.JSONTypeBoolean:
		v = BooleanWriteChannel{BaseWriteChannel: base}
	case old.JSONType == payloadv0.JSONTypeString && len(old.EnumValues) == 0:
		v = StringWriteChannel{BaseWriteChannel: base}
	default:
		v = FloatWriteChannel{
			BaseWriteChannel: base,
			EnumValues: lo.Map(
				old.EnumValues,
				func(e payloadv0.EnumEntry, _ int) payload.FloatEntry {
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
// as payload.LiftStaticWriteField and payload.LiftGeneratedWriteField do. A null field
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

// MigrateReadConfig lifts a v0 Kafka read task config to v1.
func MigrateReadConfig(ctx context.Context, old v0.ReadConfig) (ReadConfig, error) {
	return autoMigrateReadConfig(ctx, old)
}

// MigrateReadRecord lifts a v0 read record to v1.
func MigrateReadRecord(ctx context.Context, old v0.ReadRecord) (ReadRecord, error) {
	rec, err := autoMigrateReadRecord(ctx, old)
	if err != nil {
		return ReadRecord{}, err
	}
	rec.Fields, err = lo.MapErr(
		old.Fields,
		func(f v0.ReadField, _ int) (ReadField, error) {
			return MigrateReadField(ctx, f)
		},
	)
	return rec, err
}

// MigrateReadField lifts a v0 read field to the v1 variant of the kind of its stored
// data type, or to float64 when the data type is unknown. A numeric kind keeps each
// entry that is whole where the kind needs it and fits the data type, a boolean entry
// becomes value != 0, a string field drops its entries, and a timestamp field with no
// time format reads iso8601.
func MigrateReadField(_ context.Context, old v0.ReadField) (ReadField, error) {
	fld, _ := migrateReadField(old)
	return fld, nil
}

// migrateReadField also returns each change that the lift makes beyond the shape.
func migrateReadField(old v0.ReadField) (ReadField, []string) {
	var changes []string
	dt := old.DataType
	kind, ok := xpayload.KindOf(dt)
	if !ok {
		changes = append(
			changes,
			fmt.Sprintf("unknown data type %q became float64", dt),
		)
		dt, kind = telem.Float64T, xpayload.KindFloat
	}
	base := BaseReadField{
		Key:      old.Key,
		Disabled: old.Disabled,
		Name:     old.Name,
		Channel:  old.Channel,
		Pointer:  old.Pointer,
	}
	keep := func(e payloadv0.EnumEntry, fits bool) bool {
		if !fits {
			changes = append(changes, "dropped entry "+strconv.Quote(e.Label))
		}
		return fits
	}
	switch kind {
	case xpayload.KindFloat:
		v := FloatReadField{
			BaseReadField: base, DataType: new(dt),
			EnumValues: make([]xpayload.FloatEntry, 0, len(old.EnumValues)),
		}
		for _, e := range old.EnumValues {
			_, err := xjson.Convert(e.Value, dt)
			if keep(e, err == nil) {
				v.EnumValues = append(
					v.EnumValues,
					xpayload.FloatEntry{Label: e.Label, Value: xpayload.Float(e.Value)},
				)
			}
		}
		return ReadField{Variant: v}, changes
	case xpayload.KindInteger:
		v := IntegerReadField{
			BaseReadField: base, DataType: new(dt),
			EnumValues: make([]xpayload.IntegerEntry, 0, len(old.EnumValues)),
		}
		for _, e := range old.EnumValues {
			fits := whole(e.Value) && e.Value >= math.MinInt64 &&
				e.Value < math.MaxInt64
			if fits {
				_, err := xjson.Convert(int64(e.Value), dt)
				fits = err == nil
			}
			if keep(e, fits) {
				v.EnumValues = append(v.EnumValues, xpayload.IntegerEntry{
					Label: e.Label, Value: xpayload.Integer(int64(e.Value)),
				})
			}
		}
		return ReadField{Variant: v}, changes
	case xpayload.KindUnsigned:
		v := UnsignedReadField{
			BaseReadField: base, DataType: new(dt),
			EnumValues: make([]xpayload.UnsignedEntry, 0, len(old.EnumValues)),
		}
		for _, e := range old.EnumValues {
			fits := whole(e.Value) && e.Value >= 0 && e.Value < math.MaxUint64
			if fits {
				_, err := xjson.Convert(uint64(e.Value), dt)
				fits = err == nil
			}
			if keep(e, fits) {
				v.EnumValues = append(v.EnumValues, xpayload.UnsignedEntry{
					Label: e.Label, Value: xpayload.Unsigned(uint64(e.Value)),
				})
			}
		}
		return ReadField{Variant: v}, changes
	case xpayload.KindBoolean:
		v := BooleanReadField{
			BaseReadField: base,
			EnumValues:    make([]xpayload.BooleanEntry, 0, len(old.EnumValues)),
		}
		for _, e := range old.EnumValues {
			v.EnumValues = append(
				v.EnumValues,
				xpayload.BooleanEntry{Label: e.Label, Value: e.Value != 0},
			)
		}
		return ReadField{Variant: v}, changes
	case xpayload.KindString:
		for _, e := range old.EnumValues {
			keep(e, false)
		}
		return ReadField{Variant: StringReadField{
			BaseReadField: base,
			EnumValues:    []xpayload.StringEntry{},
		}}, changes
	case xpayload.KindTimestamp:
		format := xpayload.TimeFormatISO8601
		if old.TimeFormat != nil {
			format = *old.TimeFormat
		}
		return ReadField{Variant: TimestampReadField{
			BaseReadField: base,
			TimeFormat:    format,
		}}, changes
	case xpayload.KindJSON:
		return ReadField{Variant: JsonReadField{BaseReadField: base}}, changes
	case xpayload.KindUUID:
		return ReadField{Variant: UuidReadField{BaseReadField: base}}, changes
	}
	return ReadField{Variant: BytesReadField{BaseReadField: base}}, changes
}

func whole(v float64) bool { return v == math.Trunc(v) }

// NewReadMigration returns the migration that lifts stored read configs to v1, logs
// each change the lift makes, and then runs reconcile on each config.
func NewReadMigration(
	reconcile func(context.Context, gorp.Tx, *ReadConfig) error,
) gorp.Migration {
	return migrate.New(
		"v62_kafka_typed_read_fields",
		func(ctx context.Context, tx gorp.Tx, ins alamos.Instrumentation) error {
			var olds []v0.ReadConfig
			if err := gorp.NewRetrieve[uuid.UUID, v0.ReadConfig]().
				Entries(&olds).
				Exec(ctx, tx); err != nil {
				return err
			}
			news := make([]ReadConfig, 0, len(olds))
			for _, old := range olds {
				cfg, err := MigrateReadConfig(ctx, old)
				if err != nil {
					return errors.Wrapf(err, "config %s", old.GorpKey())
				}
				logChanges(ins, old)
				if err = reconcile(ctx, tx, &cfg); err != nil {
					return errors.Wrapf(err, "config %s", old.GorpKey())
				}
				news = append(news, cfg)
			}
			return gorp.WrapWriter[uuid.UUID, ReadConfig](tx).Set(ctx, news...)
		},
	)
}

func logChanges(ins alamos.Instrumentation, old v0.ReadConfig) {
	for _, rec := range old.Records {
		for _, fld := range rec.Fields {
			if _, changes := migrateReadField(fld); len(changes) > 0 {
				ins.L.Warn(
					"changed a Kafka read field while lifting it to v1",
					zap.Stringer("config", old.Key),
					zap.String("field", fld.Key),
					zap.String("data_type", string(fld.DataType)),
					zap.Strings("changes", changes),
				)
			}
		}
	}
}
