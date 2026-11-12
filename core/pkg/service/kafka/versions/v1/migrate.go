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
	"math"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/gorp/migrate"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/payload"
	"go.uber.org/zap"
)

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
// data type. A numeric kind keeps each entry that is whole and in range of the kind, a
// boolean entry becomes value != 0, a string field drops its entries, and a timestamp
// field with no time format reads iso8601.
func MigrateReadField(_ context.Context, old v0.ReadField) (ReadField, error) {
	fld, _, err := migrateReadField(old)
	return fld, err
}

// migrateReadField also returns the labels of the entries that the field drops.
func migrateReadField(old v0.ReadField) (ReadField, []string, error) {
	kind, ok := payload.KindOf(old.DataType)
	if !ok {
		return ReadField{}, nil, errors.Newf(
			"read field %q has unknown data type %q", old.Key, old.DataType,
		)
	}
	base := BaseReadField{
		Key:      old.Key,
		Disabled: old.Disabled,
		Name:     old.Name,
		Channel:  old.Channel,
		Pointer:  old.Pointer,
	}
	dt := new(old.DataType)
	var dropped []string
	switch kind {
	case payload.KindFloat:
		v := FloatReadField{
			BaseReadField: base, DataType: dt,
			EnumValues: make([]payload.FloatEntry, 0, len(old.EnumValues)),
		}
		for _, e := range old.EnumValues {
			v.EnumValues = append(
				v.EnumValues,
				payload.FloatEntry{Label: e.Label, Value: payload.Float(e.Value)},
			)
		}
		return ReadField{Variant: v}, nil, nil
	case payload.KindInteger:
		v := IntegerReadField{
			BaseReadField: base, DataType: dt,
			EnumValues: make([]payload.IntegerEntry, 0, len(old.EnumValues)),
		}
		for _, e := range old.EnumValues {
			if !whole(e.Value) || e.Value < math.MinInt64 || e.Value >= math.MaxInt64 {
				dropped = append(dropped, e.Label)
				continue
			}
			v.EnumValues = append(v.EnumValues, payload.IntegerEntry{
				Label: e.Label, Value: payload.Integer(int64(e.Value)),
			})
		}
		return ReadField{Variant: v}, dropped, nil
	case payload.KindUnsigned:
		v := UnsignedReadField{
			BaseReadField: base, DataType: dt,
			EnumValues: make([]payload.UnsignedEntry, 0, len(old.EnumValues)),
		}
		for _, e := range old.EnumValues {
			if !whole(e.Value) || e.Value < 0 || e.Value >= math.MaxUint64 {
				dropped = append(dropped, e.Label)
				continue
			}
			v.EnumValues = append(v.EnumValues, payload.UnsignedEntry{
				Label: e.Label, Value: payload.Unsigned(uint64(e.Value)),
			})
		}
		return ReadField{Variant: v}, dropped, nil
	case payload.KindBoolean:
		v := BooleanReadField{
			BaseReadField: base,
			EnumValues:    make([]payload.BooleanEntry, 0, len(old.EnumValues)),
		}
		for _, e := range old.EnumValues {
			v.EnumValues = append(
				v.EnumValues,
				payload.BooleanEntry{Label: e.Label, Value: e.Value != 0},
			)
		}
		return ReadField{Variant: v}, nil, nil
	case payload.KindString:
		for _, e := range old.EnumValues {
			dropped = append(dropped, e.Label)
		}
		return ReadField{Variant: StringReadField{
			BaseReadField: base,
			EnumValues:    []payload.StringEntry{},
		}}, dropped, nil
	case payload.KindTimestamp:
		format := payload.TimeFormatISO8601
		if old.TimeFormat != nil {
			format = *old.TimeFormat
		}
		return ReadField{Variant: TimestampReadField{
			BaseReadField: base,
			TimeFormat:    format,
		}}, nil, nil
	case payload.KindJSON:
		return ReadField{Variant: JsonReadField{BaseReadField: base}}, nil, nil
	case payload.KindUUID:
		return ReadField{Variant: UuidReadField{BaseReadField: base}}, nil, nil
	}
	return ReadField{Variant: BytesReadField{BaseReadField: base}}, nil, nil
}

func whole(v float64) bool { return v == math.Trunc(v) }

// NewReadMigration returns the migration that lifts stored read configs to v1, logs
// each entry a field drops, and then runs reconcile on each config.
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
				if err = logDropped(ins, old); err != nil {
					return errors.Wrapf(err, "config %s", old.GorpKey())
				}
				if err = reconcile(ctx, tx, &cfg); err != nil {
					return errors.Wrapf(err, "config %s", old.GorpKey())
				}
				news = append(news, cfg)
			}
			return gorp.WrapWriter[uuid.UUID, ReadConfig](tx).Set(ctx, news...)
		},
	)
}

func logDropped(ins alamos.Instrumentation, old v0.ReadConfig) error {
	for _, rec := range old.Records {
		for _, fld := range rec.Fields {
			_, dropped, err := migrateReadField(fld)
			if err != nil {
				return err
			}
			if len(dropped) > 0 {
				ins.L.Warn(
					"dropped Kafka read field entries that its kind cannot hold",
					zap.Stringer("config", old.Key),
					zap.String("field", fld.Key),
					zap.String("data_type", string(fld.DataType)),
					zap.Strings("labels", dropped),
				)
			}
		}
	}
	return nil
}
