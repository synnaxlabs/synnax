// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/gorp"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func storedField(dt telem.DataType, entries ...payload.EnumEntry) v1.ReadField {
	return v1.ReadField{
		Key:        "f1",
		Name:       "level",
		Channel:    7,
		Pointer:    "/level",
		DataType:   dt,
		EnumValues: entries,
	}
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
	DescribeTable(
		"Should pick the kind of the stored data type",
		func(dt telem.DataType, kind payload.Kind) {
			f, changes := v2.MigrateReadField(storedField(dt), false)
			Expect(f.Type()).To(Equal(kind))
			Expect(changes).To(BeEmpty())
		},
		Entry("float32", telem.Float32T, payload.KindFloat),
		Entry("int16", telem.Int16T, payload.KindInteger),
		Entry("uint8", telem.Uint8T, payload.KindUnsigned),
		Entry("boolean", telem.BooleanT, payload.KindBoolean),
		Entry("string", telem.StringT, payload.KindString),
		Entry("timestamp", telem.TimestampT, payload.KindTimestamp),
		Entry("json", telem.JSONT, payload.KindJSON),
		Entry("uuid", telem.UUIDT, payload.KindUUID),
		Entry("bytes", telem.BytesT, payload.KindBytes),
	)

	It("Should keep the base of the field", func() {
		old := storedField(telem.Float64T)
		old.Disabled = true
		f, _ := v2.MigrateReadField(old, false)
		Expect(f.Base()).To(Equal(v2.BaseReadField{
			Key:      "f1",
			Name:     "level",
			Disabled: true,
			Channel:  7,
			Pointer:  "/level",
		}))
	})

	It("Should keep the stored data type as the width of a new channel", func() {
		f, _ := v2.MigrateReadField(storedField(telem.Int16T), false)
		Expect(v2.ReadOf(f).Number.DataType).To(HaveValue(Equal(telem.Int16T)))
	})

	It("Should convert whole entries and drop the others on an integer kind", func() {
		f, changes := v2.MigrateReadField(storedField(
			telem.Int32T,
			payload.EnumEntry{Label: "OFF", Value: 0},
			payload.EnumEntry{Label: "ON", Value: 1},
			payload.EnumEntry{Label: "HALF", Value: 0.5},
			payload.EnumEntry{Label: "", Value: 2},
		), false)
		integer, ok := f.Variant.(v2.IntegerReadField)
		Expect(ok).To(BeTrue())
		Expect(integer.EnumValues).To(Equal([]payload.IntegerEntry{
			{Label: "OFF", Value: 0},
			{Label: "ON", Value: 1},
		}))
		Expect(changes).To(ConsistOf(
			`enum label "HALF" was dropped: 0.5 is not integer`,
			`enum label "" was dropped: it is empty`,
		))
	})

	It("Should keep the first entry of a repeated label", func() {
		f, changes := v2.MigrateReadField(storedField(
			telem.BooleanT,
			payload.EnumEntry{Label: "ON", Value: 1},
			payload.EnumEntry{Label: "ON", Value: 0},
		), false)
		boolean, ok := f.Variant.(v2.BooleanReadField)
		Expect(ok).To(BeTrue())
		Expect(boolean.EnumValues).To(Equal(
			[]payload.BooleanEntry{{Label: "ON", Value: true}},
		))
		Expect(changes).To(ConsistOf(
			`enum label "ON" was dropped: it repeats an earlier label`,
		))
	})

	It("Should drop a negative entry on an unsigned kind", func() {
		f, changes := v2.MigrateReadField(storedField(
			telem.Uint16T,
			payload.EnumEntry{Label: "LOW", Value: -1},
			payload.EnumEntry{Label: "HIGH", Value: 3},
		), false)
		unsigned, ok := f.Variant.(v2.UnsignedReadField)
		Expect(ok).To(BeTrue())
		Expect(unsigned.EnumValues).To(Equal(
			[]payload.UnsignedEntry{{Label: "HIGH", Value: 3}},
		))
		Expect(
			changes,
		).To(ConsistOf(`enum label "LOW" was dropped: -1 is not unsigned`))
	})

	It("Should keep every finite entry on a float kind", func() {
		f, changes := v2.MigrateReadField(storedField(
			telem.Float64T,
			payload.EnumEntry{Label: "HALF", Value: 0.5},
		), false)
		float, ok := f.Variant.(v2.FloatReadField)
		Expect(ok).To(BeTrue())
		Expect(float.EnumValues).To(Equal(
			[]payload.FloatEntry{{Label: "HALF", Value: 0.5}},
		))
		Expect(changes).To(BeEmpty())
	})

	It("Should map a boolean entry to whether its value is not 0", func() {
		f, _ := v2.MigrateReadField(storedField(
			telem.BooleanT,
			payload.EnumEntry{Label: "OFF", Value: 0},
			payload.EnumEntry{Label: "ON", Value: 2},
		), false)
		boolean, ok := f.Variant.(v2.BooleanReadField)
		Expect(ok).To(BeTrue())
		Expect(boolean.EnumValues).To(Equal([]payload.BooleanEntry{
			{Label: "OFF", Value: false},
			{Label: "ON", Value: true},
		}))
	})

	It("Should drop the entries of a string field", func() {
		f, changes := v2.MigrateReadField(storedField(
			telem.StringT,
			payload.EnumEntry{Label: "ON", Value: 1},
		), false)
		str, ok := f.Variant.(v2.StringReadField)
		Expect(ok).To(BeTrue())
		Expect(str.EnumValues).To(BeEmpty())
		Expect(changes).To(ConsistOf(
			`enum label "ON" was dropped: a string field takes no numeric entries`,
		))
	})

	It("Should give a timestamp field with no time format ISO 8601", func() {
		f, _ := v2.MigrateReadField(storedField(telem.TimestampT), false)
		stamp, ok := f.Variant.(v2.TimestampReadField)
		Expect(ok).To(BeTrue())
		Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatISO8601))
	})

	It("Should keep the time format of a timestamp field", func() {
		old := storedField(telem.TimestampT)
		old.TimeFormat = new(payload.TimeFormatUnixMs)
		f, _ := v2.MigrateReadField(old, false)
		stamp, ok := f.Variant.(v2.TimestampReadField)
		Expect(ok).To(BeTrue())
		Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatUnixMs))
	})

	It("Should keep the time format of a field stored with another data type", func() {
		old := storedField(telem.Float64T)
		old.TimeFormat = new(payload.TimeFormatUnixMs)
		f, changes := v2.MigrateReadField(old, false)
		stamp, ok := f.Variant.(v2.TimestampReadField)
		Expect(ok).To(BeTrue())
		Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatUnixMs))
		Expect(changes).To(ConsistOf(
			"data type float64 became the timestamp kind of an index or time format",
		))
	})

	It("Should lift a typed field with a time format to the timestamp kind", func() {
		old := storedField(telem.Uint8T, payload.EnumEntry{Label: "ON", Value: 1})
		old.TimeFormat = new(payload.TimeFormatUnixSec)
		f, changes := v2.MigrateReadField(old, false)
		stamp, ok := f.Variant.(v2.TimestampReadField)
		Expect(ok).To(BeTrue())
		Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatUnixSec))
		Expect(changes).To(ConsistOf(
			"data type uint8 became the timestamp kind of an index or time format",
			`enum label "ON" was dropped: a timestamp field takes no numeric entries`,
		))
	})

	It("Should lift an index field with no time format to ISO 8601", func() {
		f, changes := v2.MigrateReadField(storedField(""), true)
		stamp, ok := f.Variant.(v2.TimestampReadField)
		Expect(ok).To(BeTrue())
		Expect(stamp.TimeFormat).To(Equal(payload.TimeFormatISO8601))
		Expect(changes).To(ConsistOf(
			"data type float64 became the timestamp kind of an index or time format",
		))
	})

	It("Should lift an unknown data type to the float kind and describe it", func() {
		f, changes := v2.MigrateReadField(storedField("complex128"), false)
		Expect(f.Type()).To(Equal(payload.KindFloat))
		Expect(v2.ReadOf(f).Number.DataType).To(BeNil())
		Expect(changes).To(ConsistOf(
			`unknown data type "complex128" became the float kind`,
		))
	})

	It("Should lift an unknown data type with a time format to the timestamp kind",
		func() {
			old := storedField("complex128")
			old.TimeFormat = new(payload.TimeFormatUnixSec)
			f, changes := v2.MigrateReadField(old, false)
			Expect(f.Type()).To(Equal(payload.KindTimestamp))
			Expect(changes).To(ConsistOf(
				`unknown data type "complex128" became the float kind`,
				"data type complex128 became the timestamp kind of an index or time format",
			))
		},
	)

	It("Should take float64 for an empty data type", func() {
		f, changes := v2.MigrateReadField(storedField(""), false)
		Expect(v2.ReadOf(f).Number.DataType).To(HaveValue(Equal(telem.Float64T)))
		Expect(changes).To(BeEmpty())
	})
})

var _ = Describe("MigrateReadConfig", func() {
	It("Should migrate the fields of a plain entry and keep a Sparkplug entry", func(
		ctx SpecContext,
	) {
		old := v1.ReadConfig{
			Key:    uuid.New(),
			Device: "broker",
			Entries: []v1.ReadEntry{
				{Variant: v1.PlainReadEntry{
					BaseReadEntry: v1.BaseReadEntry{Key: "e1"},
					Topic:         "plant/level",
					Index:         "f1",
					Fields:        []v1.ReadField{storedField(telem.Uint8T)},
				}},
				{Variant: v1.SparkplugReadEntry{
					BaseReadEntry: v1.BaseReadEntry{Key: "e2"},
					Tag:           "speed",
					DataType:      telem.Float32T,
				}},
			},
		}
		cfg := MustSucceed(v2.MigrateReadConfig(ctx, old))
		Expect(cfg.Key).To(Equal(old.Key))
		Expect(cfg.Device).To(Equal(old.Device))
		plain, ok := cfg.Entries[0].Variant.(v2.PlainReadEntry)
		Expect(ok).To(BeTrue())
		Expect(plain.Topic).To(Equal("plant/level"))
		Expect(plain.Index).To(Equal("f1"))
		Expect(plain.Fields).To(HaveLen(1))
		Expect(plain.Fields[0].Type()).To(Equal(payload.KindTimestamp))
		tag, ok := cfg.Entries[1].Variant.(v2.SparkplugReadEntry)
		Expect(ok).To(BeTrue())
		Expect(tag.Tag).To(Equal("speed"))
	})
})

var _ = Describe("NewReadMigration", func() {
	It("Should lift stored configs, reconcile them with channels, and log", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		stored := v1.ReadConfig{
			Key: uuid.New(),
			Entries: []v1.ReadEntry{{Variant: v1.PlainReadEntry{
				BaseReadEntry: v1.BaseReadEntry{Key: "e1"},
				Topic:         "plant/level",
				Fields: []v1.ReadField{storedField(
					telem.Float64T,
					payload.EnumEntry{Label: "ON", Value: 1},
					payload.EnumEntry{Label: "HALF", Value: 0.5},
				)},
			}}},
		}
		Expect(gorp.NewCreate[uuid.UUID, v1.ReadConfig]().Entry(&stored).Exec(ctx, db)).
			To(Succeed())
		channels := func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]v2.Channel, error) {
			return map[channel.Key]v2.Channel{
				7: {Name: "level", DataType: telem.Int32T},
			}, nil
		}
		ins, logs := observe()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: ins,
			DB:              db,
			Namespace:       "mqtt_read_config",
			Migrations:      []gorp.Migration{v2.NewReadMigration(channels)},
		})).To(Succeed())
		var got v2.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v2.ReadConfig]().
			Where(gorp.MatchKeys[uuid.UUID, v2.ReadConfig](stored.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		plain, ok := got.Entries[0].Variant.(v2.PlainReadEntry)
		Expect(ok).To(BeTrue())
		integer, ok := plain.Fields[0].Variant.(v2.IntegerReadField)
		Expect(ok).To(BeTrue())
		Expect(integer.DataType).To(BeNil())
		Expect(integer.EnumValues).To(Equal(
			[]payload.IntegerEntry{{Label: "ON", Value: 1}},
		))
		changes := logs.FilterField(zap.String("field", "entries.0.fields.0")).All()
		Expect(changes).To(HaveLen(2))
		Expect(changes[0].ContextMap()).To(HaveKeyWithValue(
			"change", "data type float64 became int32, the data type of its channel",
		))
		Expect(changes[1].ContextMap()).To(HaveKeyWithValue(
			"change", `enum label "HALF" was dropped: 0.5 is not integer`,
		))
	})

	It("Should rewrite each stored record in place, past an unknown data type", func(
		ctx SpecContext,
	) {
		Expect(v2.ReadConfig{}.CustomTypeName()).
			To(Equal(v1.ReadConfig{}.CustomTypeName()))
		db := DeferClose(gorp.Wrap(memkv.New()))
		known := v1.ReadConfig{
			Key: uuid.New(),
			Entries: []v1.ReadEntry{{Variant: v1.PlainReadEntry{
				Fields: []v1.ReadField{storedField(telem.Uint8T)},
			}}},
		}
		unknown := v1.ReadConfig{
			Key: uuid.New(),
			Entries: []v1.ReadEntry{{Variant: v1.PlainReadEntry{
				Fields: []v1.ReadField{storedField("complex128")},
			}}},
		}
		Expect(gorp.NewCreate[uuid.UUID, v1.ReadConfig]().
			Entries(&[]v1.ReadConfig{known, unknown}).
			Exec(ctx, db)).To(Succeed())
		none := func(
			context.Context,
			gorp.Snapshot,
			[]channel.Key,
		) (map[channel.Key]v2.Channel, error) {
			return nil, nil
		}
		ins, _ := observe()
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: ins,
			DB:              db,
			Namespace:       "mqtt_read_config",
			Migrations:      []gorp.Migration{v2.NewReadMigration(none)},
		})).To(Succeed())
		var got []v2.ReadConfig
		Expect(gorp.NewRetrieve[uuid.UUID, v2.ReadConfig]().
			Entries(&got).Exec(ctx, db)).To(Succeed())
		Expect(got).To(HaveLen(2))
		kinds := map[uuid.UUID]payload.Kind{}
		for _, cfg := range got {
			plain, ok := cfg.Entries[0].Variant.(v2.PlainReadEntry)
			Expect(ok).To(BeTrue())
			kinds[cfg.Key] = plain.Fields[0].Type()
		}
		Expect(kinds).To(Equal(map[uuid.UUID]payload.Kind{
			known.Key:   payload.KindUnsigned,
			unknown.Key: payload.KindFloat,
		}))
	})

	DescribeTable(
		"Should lift a field to the kind of its channel",
		func(
			ctx SpecContext,
			ch v2.Channel,
			old v1.ReadField,
			expected v2.ReadField,
			changes []string,
		) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			stored := v1.ReadConfig{
				Key: uuid.New(),
				Entries: []v1.ReadEntry{{Variant: v1.PlainReadEntry{
					Fields: []v1.ReadField{old},
				}}},
			}
			Expect(gorp.NewCreate[uuid.UUID, v1.ReadConfig]().
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			channels := func(
				context.Context,
				gorp.Snapshot,
				[]channel.Key,
			) (map[channel.Key]v2.Channel, error) {
				return map[channel.Key]v2.Channel{7: ch}, nil
			}
			ins, logs := observe()
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				Instrumentation: ins,
				DB:              db,
				Namespace:       "mqtt_read_config",
				Migrations:      []gorp.Migration{v2.NewReadMigration(channels)},
			})).To(Succeed())
			logged := []string{}
			for _, e := range logs.All() {
				logged = append(logged, e.ContextMap()["change"].(string))
			}
			Expect(logged).To(ConsistOf(changes))
			var got v2.ReadConfig
			Expect(gorp.NewRetrieve[uuid.UUID, v2.ReadConfig]().
				Where(gorp.MatchKeys[uuid.UUID, v2.ReadConfig](stored.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			plain, ok := got.Entries[0].Variant.(v2.PlainReadEntry)
			Expect(ok).To(BeTrue())
			Expect(plain.Fields[0]).To(Equal(expected))
		},
		Entry(
			"a numeric field with a time format and entries",
			v2.Channel{Name: "state", DataType: telem.Uint8T},
			func() v1.ReadField {
				f := storedField("", payload.EnumEntry{Label: "ON", Value: 1})
				f.TimeFormat = new(payload.TimeFormatUnixSec)
				return f
			}(),
			v2.ReadField{Variant: v2.UnsignedReadField{
				BaseReadField: v2.BaseReadField{
					Key: "f1", Name: "level", Channel: 7, Pointer: "/level",
				},
				UnsignedRead: payload.UnsignedRead{
					EnumValues: []payload.UnsignedEntry{{Label: "ON", Value: 1}},
				},
			}},
			[]string{
				"data type float64 became uint8, the data type of its channel",
				"time format unix_sec was dropped, because the field is unsigned",
			},
		),
		Entry(
			"a default field with a time format on a timestamp channel",
			v2.Channel{Name: "time", DataType: telem.TimestampT, Index: true},
			func() v1.ReadField {
				f := storedField("")
				f.TimeFormat = new(payload.TimeFormatUnixMs)
				return f
			}(),
			v2.ReadField{Variant: v2.TimestampReadField{
				BaseReadField: v2.BaseReadField{
					Key: "f1", Name: "level", Channel: 7, Pointer: "/level",
				},
				TimestampRead: payload.TimestampRead{
					TimeFormat: payload.TimeFormatUnixMs,
				},
			}},
			[]string{
				"data type float64 became timestamp, the data type of its channel",
			},
		),
	)
})
