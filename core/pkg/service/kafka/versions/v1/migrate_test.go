// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func oldField(dt telem.DataType, entries ...payload.EnumEntry) v0.ReadField {
	return v0.ReadField{
		Key:        "f",
		Disabled:   true,
		Name:       "temp",
		Channel:    7,
		Pointer:    "/v",
		DataType:   dt,
		EnumValues: entries,
	}
}

var base = v1.BaseReadField{
	Key:      "f",
	Disabled: true,
	Name:     "temp",
	Channel:  7,
	Pointer:  "/v",
}

func observe() (alamos.Instrumentation, *observer.ObservedLogs) {
	GinkgoHelper()
	core, logs := observer.New(zapcore.WarnLevel)
	logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
		ZapLogger: zap.New(core),
	}))
	return alamos.New("test", alamos.WithLogger(logger)), logs
}

var _ = Describe("MigrateReadField", func() {
	DescribeTable("Should lift a field to the kind of its stored data type",
		func(ctx SpecContext, old v0.ReadField, expected v1.ReadFieldVariant) {
			Expect(MustSucceed(v1.MigrateReadField(ctx, old)).Variant).
				To(Equal(expected))
		},
		Entry("float, keeping every entry",
			oldField(telem.Float32T, payload.EnumEntry{Label: "HALF", Value: 0.5}),
			v1.FloatReadField{
				BaseReadField: base,
				DataType:      new(telem.Float32T),
				EnumValues:    []payload.FloatEntry{{Label: "HALF", Value: 0.5}},
			},
		),
		Entry("integer, dropping a fraction and a value past the data type",
			oldField(
				telem.Int16T,
				payload.EnumEntry{Label: "LOW", Value: -1},
				payload.EnumEntry{Label: "HALF", Value: 0.5},
				payload.EnumEntry{Label: "BIG", Value: 40000},
			),
			v1.IntegerReadField{
				BaseReadField: base,
				DataType:      new(telem.Int16T),
				EnumValues:    []payload.IntegerEntry{{Label: "LOW", Value: -1}},
			},
		),
		Entry("unsigned, dropping a negative",
			oldField(
				telem.Uint8T,
				payload.EnumEntry{Label: "ON", Value: 1},
				payload.EnumEntry{Label: "LOW", Value: -1},
			),
			v1.UnsignedReadField{
				BaseReadField: base,
				DataType:      new(telem.Uint8T),
				EnumValues:    []payload.UnsignedEntry{{Label: "ON", Value: 1}},
			},
		),
		Entry("boolean, as value != 0",
			oldField(
				telem.BooleanT,
				payload.EnumEntry{Label: "ON", Value: 1},
				payload.EnumEntry{Label: "OFF", Value: 0},
			),
			v1.BooleanReadField{
				BaseReadField: base,
				EnumValues: []payload.BooleanEntry{
					{Label: "ON", Value: true},
					{Label: "OFF", Value: false},
				},
			},
		),
		Entry(
			"string, dropping its entries",
			oldField(telem.StringT, payload.EnumEntry{Label: "ON", Value: 1}),
			v1.StringReadField{
				BaseReadField: base,
				EnumValues:    []payload.StringEntry{},
			},
		),
		Entry("timestamp with no time format, as iso8601",
			oldField(telem.TimestampT),
			v1.TimestampReadField{
				BaseReadField: base,
				TimeFormat:    payload.TimeFormatISO8601,
			},
		),
		Entry("json", oldField(telem.JSONT), v1.JsonReadField{BaseReadField: base}),
		Entry("uuid", oldField(telem.UUIDT), v1.UuidReadField{BaseReadField: base}),
		Entry("bytes", oldField(telem.BytesT), v1.BytesReadField{BaseReadField: base}),
	)

	It(
		"Should keep the stored time format of a timestamp field",
		func(ctx SpecContext) {
			old := oldField(telem.TimestampT)
			old.TimeFormat = new(payload.TimeFormatUnixMs)
			Expect(MustSucceed(v1.MigrateReadField(ctx, old)).Variant).To(Equal(
				v1.TimestampReadField{
					BaseReadField: base,
					TimeFormat:    payload.TimeFormatUnixMs,
				},
			))
		},
	)

	It("Should lift a field of an unknown data type to float64", func(ctx SpecContext) {
		Expect(MustSucceed(v1.MigrateReadField(ctx, oldField(""))).Variant).To(Equal(
			v1.FloatReadField{
				BaseReadField: base,
				DataType:      new(telem.Float64T),
				EnumValues:    []payload.FloatEntry{},
			},
		))
	})
})

var _ = Describe("NewReadMigration", func() {
	It("Should lift, log, and reconcile each stored read config", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		stored := v0.ReadConfig{
			Key:   uuid.New(),
			Topic: "readings",
			Records: []v0.ReadRecord{{
				Key: "r",
				Fields: []v0.ReadField{oldField(
					telem.Int8T,
					payload.EnumEntry{Label: "ON", Value: 1},
					payload.EnumEntry{Label: "HALF", Value: 0.5},
				)},
			}},
		}
		Expect(gorp.NewCreate[uuid.UUID, v0.ReadConfig]().Entry(&stored).Exec(ctx, db)).
			To(Succeed())
		ins, logs := observe()
		var reconciled []uuid.UUID
		reconcile := func(_ context.Context, _ gorp.Tx, c *v1.ReadConfig) error {
			reconciled = append(reconciled, c.Key)
			return nil
		}
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: ins,
			DB:              db,
			Namespace:       "kafka_read_config",
			Migrations:      []gorp.Migration{v1.NewReadMigration(reconcile)},
		})).To(Succeed())
		var got v1.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v1.ReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v1.ReadConfig](stored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Topic).To(Equal("readings"))
		Expect(got.Records[0].Fields[0].Variant).To(Equal(v1.IntegerReadField{
			BaseReadField: base,
			DataType:      new(telem.Int8T),
			EnumValues:    []payload.IntegerEntry{{Label: "ON", Value: 1}},
		}))
		Expect(reconciled).To(Equal([]uuid.UUID{stored.Key}))
		Expect(logs.FilterMessage(
			"changed a Kafka read field while lifting it to v1",
		).Len()).To(Equal(1))
	})
})
