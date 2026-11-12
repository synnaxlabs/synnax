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
	xpayload "github.com/synnaxlabs/x/payload"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	payloadv1 "github.com/synnaxlabs/x/payload/versions/v1"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func storedField(dt telem.DataType, entries ...xpayload.EnumEntry) v1.ReadField {
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

var _ = Describe("Migrate", func() {
	stored := func(c v1.ChannelField) v1.ChannelField {
		c.Pointer, c.Name = "/v", "cmd"
		return c
	}
	base := func(dt telem.DataType) v2.BaseChannelField {
		return v2.BaseChannelField{
			Pointer:  "/v",
			Name:     "cmd",
			DataType: &dt,
		}
	}
	unixMs := payload.TimeFormatUnixMs
	onOff := []payload.EnumEntry{{Label: "OFF", Value: 0}, {Label: "ON", Value: 1}}

	DescribeTable("MigrateChannelField",
		func(ctx SpecContext, old v1.ChannelField, expected v2.ChannelFieldVariant) {
			Expect(v2.MigrateChannelField(ctx, stored(old))).
				To(Equal(v2.ChannelField{Variant: expected}))
		},
		Entry("a number on a float64 channel as a float",
			v1.ChannelField{JSONType: payload.JSONTypeNumber, DataType: telem.Float64T},
			v2.FloatChannelField{
				BaseChannelField: base(telem.Float64T),
				EnumValues:       []payload.FloatEntry{},
			}),
		Entry("an absent data type as a float64 float",
			v1.ChannelField{},
			v2.FloatChannelField{
				BaseChannelField: base(telem.Float64T),
				EnumValues:       []payload.FloatEntry{},
			}),
		Entry("a number for an int16 channel as a float, for the caller to reconcile",
			v1.ChannelField{JSONType: payload.JSONTypeNumber, DataType: telem.Int16T},
			v2.FloatChannelField{
				BaseChannelField: base(telem.Int16T),
				EnumValues:       []payload.FloatEntry{},
			}),
		Entry("a string with entries as a float that keeps them",
			v1.ChannelField{
				JSONType:   payload.JSONTypeString,
				DataType:   telem.Uint8T,
				EnumValues: onOff,
			},
			v2.FloatChannelField{
				BaseChannelField: base(telem.Uint8T),
				EnumValues: []payload.FloatEntry{
					{Label: "OFF", Value: 0},
					{Label: "ON", Value: 1},
				},
			}),
		Entry("a string with no entries as a string",
			v1.ChannelField{JSONType: payload.JSONTypeString, DataType: telem.Float32T},
			v2.StringChannelField{BaseChannelField: base(telem.Float32T)}),
		Entry("a boolean as a boolean",
			v1.ChannelField{JSONType: payload.JSONTypeBoolean, DataType: telem.Uint8T},
			v2.BooleanChannelField{BaseChannelField: base(telem.Uint8T)}),
		Entry("a timestamp in its time format",
			v1.ChannelField{
				JSONType:   payload.JSONTypeString,
				DataType:   telem.TimestampT,
				TimeFormat: &unixMs,
			},
			v2.TimestampChannelField{
				BaseChannelField: base(telem.TimestampT),
				TimeFormat:       payload.TimeFormatUnixMs,
			}),
		Entry(
			"a timestamp with no time format as ISO 8601",
			v1.ChannelField{
				JSONType: payload.JSONTypeString,
				DataType: telem.TimestampT,
			},
			v2.TimestampChannelField{
				BaseChannelField: base(telem.TimestampT),
				TimeFormat:       payload.TimeFormatISO8601,
			},
		),
		Entry("a bound number with a time format as a timestamp that falls back to it",
			v1.ChannelField{
				Channel:    7,
				JSONType:   payload.JSONTypeNumber,
				TimeFormat: &unixMs,
			},
			v2.TimestampChannelField{
				Pointer:    "/v",
				Channel:    7,
				Name:       "cmd",
				DataType:   new(telem.Float64T),
				TimeFormat: payload.TimeFormatUnixMs,
			}),
		Entry("a bound string with a time format as a timestamp that falls back to it",
			v1.ChannelField{
				Channel:    7,
				JSONType:   payload.JSONTypeString,
				TimeFormat: &unixMs,
			},
			v2.TimestampChannelField{
				Pointer:    "/v",
				Channel:    7,
				Name:       "cmd",
				DataType:   new(telem.StringT),
				TimeFormat: payload.TimeFormatUnixMs,
			}),
		Entry("a number on a bound field as a float, for its channel to reconcile",
			v1.ChannelField{
				Channel:  7,
				JSONType: payload.JSONTypeNumber,
				DataType: telem.Int16T,
			},
			v2.FloatChannelField{
				Pointer:    "/v",
				Channel:    7,
				Name:       "cmd",
				DataType:   new(telem.Int16T),
				EnumValues: []payload.FloatEntry{},
			}),
		Entry("a bound boolean with a time format as a timestamp that falls back to it",
			v1.ChannelField{
				Channel:    7,
				JSONType:   payload.JSONTypeBoolean,
				TimeFormat: &unixMs,
			},
			v2.TimestampChannelField{
				Pointer:    "/v",
				Channel:    7,
				Name:       "cmd",
				DataType:   new(telem.BooleanT),
				TimeFormat: payload.TimeFormatUnixMs,
			}),
		Entry("a time format with no channel, ignored as v1 did",
			v1.ChannelField{JSONType: payload.JSONTypeString, TimeFormat: &unixMs},
			v2.StringChannelField{BaseChannelField: base(telem.Float64T)}),
		Entry("a time format on a bound field with entries as a float",
			v1.ChannelField{
				Channel:    7,
				JSONType:   payload.JSONTypeString,
				TimeFormat: &unixMs,
				EnumValues: onOff,
			},
			v2.FloatChannelField{
				Pointer:  "/v",
				Channel:  7,
				Name:     "cmd",
				DataType: new(telem.Float64T),
				EnumValues: []payload.FloatEntry{
					{Label: "OFF", Value: 0},
					{Label: "ON", Value: 1},
				},
			}),
	)

	It("Should lift every target and keep the rest of a write config", func(
		ctx SpecContext,
	) {
		plain := v1.PlainWriteTarget{
			Key:      "p1",
			Disabled: true,
			Topic:    "plant/valve/set",
			Qos:      v1.QoSExactlyOnce,
			Retained: true,
			Channel: stored(v1.ChannelField{
				JSONType: payload.JSONTypeBoolean,
				DataType: telem.Uint8T,
			}),
		}
		tag := v1.SparkplugWriteTarget{
			Key:      "s1",
			Group:    "plant",
			EdgeNode: "line1",
			Tag:      "rpm",
			TagType:  v1.SparkplugDataTypeInt32,
			Channel:  9,
			Name:     "rpm_cmd",
		}
		old := v1.WriteConfig{
			Device:  "broker",
			Targets: []v1.WriteTarget{{Variant: plain}, {Variant: tag}},
		}
		cfg := MustSucceed(v2.MigrateWriteConfig(ctx, old))
		Expect(cfg.Device).To(BeEquivalentTo("broker"))
		Expect(cfg.Targets).To(HaveExactElements(
			v2.WriteTarget{Variant: v2.PlainWriteTarget{
				BaseWriteTarget: plain.BaseWriteTarget,
				Topic:           "plant/valve/set",
				Qos:             v2.QoSExactlyOnce,
				Retained:        true,
				Channel: v2.ChannelField{Variant: v2.BooleanChannelField{
					BaseChannelField: base(telem.Uint8T),
				}},
			}},
			v2.WriteTarget{Variant: v2.SparkplugWriteTarget{
				BaseWriteTarget: tag.BaseWriteTarget,
				Group:           "plant",
				EdgeNode:        "line1",
				Tag:             "rpm",
				TagType:         v2.SparkplugDataTypeInt32,
				Channel:         9,
				Name:            "rpm_cmd",
			}},
		))
	})

	It("Should lift the additional fields of a plain target", func(ctx SpecContext) {
		t := MustSucceed(v2.MigrateWriteTarget(ctx, v1.WriteTarget{
			Variant: v1.PlainWriteTarget{
				Topic: "plant/valve/set",
				Fields: []v1.WriteField{
					{Variant: v1.StaticWriteField{
						BaseWriteField: v1.BaseWriteField{Key: "n", Pointer: "/n"},
						JSONType:       payload.JSONTypeNumber,
						Value:          3.0,
					}},
					{Variant: v1.GeneratedWriteField{
						BaseWriteField: v1.BaseWriteField{Key: "t", Pointer: "/t"},
						Generator:      payload.GeneratorTypeTimestamp,
						TimeFormat:     new(payload.TimeFormatUnixNs),
					}},
					{Variant: v1.GeneratedWriteField{Generator: "id"}},
				},
			},
		}))
		plain, ok := t.Variant.(v2.PlainWriteTarget)
		Expect(ok).To(BeTrue())
		Expect(plain.Fields).To(Equal([]payloadv1.WriteField{
			{Variant: payloadv1.IntegerWriteField{
				BaseWriteField: payloadv1.BaseWriteField{Key: "n", Pointer: "/n"},
				Value:          3,
			}},
			{Variant: payloadv1.TimestampWriteField{
				BaseWriteField: payloadv1.BaseWriteField{Key: "t", Pointer: "/t"},
				TimeFormat:     payloadv1.TimeFormatUnixNs,
			}},
		}))
	})
})

var _ = Describe("MigrateReadField", func() {
	DescribeTable(
		"Should pick the kind of the stored data type",
		func(dt telem.DataType, kind xpayload.Kind) {
			f, changes := v2.MigrateReadField(storedField(dt))
			Expect(f.Type()).To(Equal(kind))
			Expect(changes).To(BeEmpty())
		},
		Entry("float32", telem.Float32T, xpayload.KindFloat),
		Entry("int16", telem.Int16T, xpayload.KindInteger),
		Entry("uint8", telem.Uint8T, xpayload.KindUnsigned),
		Entry("boolean", telem.BooleanT, xpayload.KindBoolean),
		Entry("string", telem.StringT, xpayload.KindString),
		Entry("timestamp", telem.TimestampT, xpayload.KindTimestamp),
		Entry("json", telem.JSONT, xpayload.KindJSON),
		Entry("uuid", telem.UUIDT, xpayload.KindUUID),
		Entry("bytes", telem.BytesT, xpayload.KindBytes),
	)

	It("Should keep the base of the field", func() {
		old := storedField(telem.Float64T)
		old.Disabled = true
		f, _ := v2.MigrateReadField(old)
		Expect(f.Base()).To(Equal(v2.BaseReadField{
			Key:      "f1",
			Name:     "level",
			Disabled: true,
			Channel:  7,
			Pointer:  "/level",
		}))
	})

	It("Should keep the stored data type as the width of a new channel", func() {
		f, _ := v2.MigrateReadField(storedField(telem.Int16T))
		Expect(v2.ReadOf(f).Number.DataType).To(HaveValue(Equal(telem.Int16T)))
	})

	It("Should convert whole entries and drop the others on an integer kind", func() {
		f, changes := v2.MigrateReadField(storedField(
			telem.Int32T,
			xpayload.EnumEntry{Label: "OFF", Value: 0},
			xpayload.EnumEntry{Label: "ON", Value: 1},
			xpayload.EnumEntry{Label: "HALF", Value: 0.5},
			xpayload.EnumEntry{Label: "", Value: 2},
		))
		integer, ok := f.Variant.(v2.IntegerReadField)
		Expect(ok).To(BeTrue())
		Expect(integer.EnumValues).To(Equal([]xpayload.IntegerEntry{
			{Label: "OFF", Value: 0},
			{Label: "ON", Value: 1},
		}))
		Expect(changes).To(ConsistOf(
			`enum label "HALF" was dropped: 0.5 is not integer`,
			`enum label "" was dropped: it is empty`,
		))
	})

	It("Should drop a negative entry on an unsigned kind", func() {
		f, changes := v2.MigrateReadField(storedField(
			telem.Uint16T,
			xpayload.EnumEntry{Label: "LOW", Value: -1},
			xpayload.EnumEntry{Label: "HIGH", Value: 3},
		))
		unsigned, ok := f.Variant.(v2.UnsignedReadField)
		Expect(ok).To(BeTrue())
		Expect(unsigned.EnumValues).To(Equal(
			[]xpayload.UnsignedEntry{{Label: "HIGH", Value: 3}},
		))
		Expect(
			changes,
		).To(ConsistOf(`enum label "LOW" was dropped: -1 is not unsigned`))
	})

	It("Should keep every finite entry on a float kind", func() {
		f, changes := v2.MigrateReadField(storedField(
			telem.Float64T,
			xpayload.EnumEntry{Label: "HALF", Value: 0.5},
		))
		float, ok := f.Variant.(v2.FloatReadField)
		Expect(ok).To(BeTrue())
		Expect(float.EnumValues).To(Equal(
			[]xpayload.FloatEntry{{Label: "HALF", Value: 0.5}},
		))
		Expect(changes).To(BeEmpty())
	})

	It("Should map a boolean entry to whether its value is not 0", func() {
		f, _ := v2.MigrateReadField(storedField(
			telem.BooleanT,
			xpayload.EnumEntry{Label: "OFF", Value: 0},
			xpayload.EnumEntry{Label: "ON", Value: 2},
		))
		boolean, ok := f.Variant.(v2.BooleanReadField)
		Expect(ok).To(BeTrue())
		Expect(boolean.EnumValues).To(Equal([]xpayload.BooleanEntry{
			{Label: "OFF", Value: false},
			{Label: "ON", Value: true},
		}))
	})

	It("Should drop the entries of a string field", func() {
		f, changes := v2.MigrateReadField(storedField(
			telem.StringT,
			xpayload.EnumEntry{Label: "ON", Value: 1},
		))
		str, ok := f.Variant.(v2.StringReadField)
		Expect(ok).To(BeTrue())
		Expect(str.EnumValues).To(BeEmpty())
		Expect(changes).To(ConsistOf(
			`enum label "ON" was dropped: a string field takes no numeric entries`,
		))
	})

	It("Should give a timestamp field with no time format ISO 8601", func() {
		f, _ := v2.MigrateReadField(storedField(telem.TimestampT))
		stamp, ok := f.Variant.(v2.TimestampReadField)
		Expect(ok).To(BeTrue())
		Expect(stamp.TimeFormat).To(Equal(xpayload.TimeFormatISO8601))
	})

	It("Should keep the time format of a timestamp field", func() {
		old := storedField(telem.TimestampT)
		old.TimeFormat = new(xpayload.TimeFormatUnixMs)
		f, _ := v2.MigrateReadField(old)
		stamp, ok := f.Variant.(v2.TimestampReadField)
		Expect(ok).To(BeTrue())
		Expect(stamp.TimeFormat).To(Equal(xpayload.TimeFormatUnixMs))
	})

	It("Should keep the time format of a field stored with another data type", func() {
		old := storedField(telem.Float64T)
		old.TimeFormat = new(xpayload.TimeFormatUnixMs)
		f, changes := v2.MigrateReadField(old)
		stamp, ok := f.Variant.(v2.TimestampReadField)
		Expect(ok).To(BeTrue())
		Expect(stamp.TimeFormat).To(Equal(xpayload.TimeFormatUnixMs))
		Expect(changes).To(ConsistOf(
			"data type float64 became the timestamp kind, which its time format reads",
		))
	})

	It("Should keep the numeric kind and entries of a typed field with a time format",
		func() {
			old := storedField(telem.Uint8T, xpayload.EnumEntry{Label: "ON", Value: 1})
			old.TimeFormat = new(xpayload.TimeFormatUnixSec)
			f, changes := v2.MigrateReadField(old)
			unsigned, ok := f.Variant.(v2.UnsignedReadField)
			Expect(ok).To(BeTrue())
			Expect(unsigned.EnumValues).To(Equal(
				[]xpayload.UnsignedEntry{{Label: "ON", Value: 1}},
			))
			Expect(changes).To(ConsistOf(
				"time format unix_sec was dropped, because the field is unsigned",
			))
		},
	)

	It("Should keep the entries of a default field with a time format", func() {
		old := storedField("", xpayload.EnumEntry{Label: "ON", Value: 1})
		old.TimeFormat = new(xpayload.TimeFormatUnixSec)
		f, changes := v2.MigrateReadField(old)
		float, ok := f.Variant.(v2.FloatReadField)
		Expect(ok).To(BeTrue())
		Expect(
			float.EnumValues,
		).To(Equal([]xpayload.FloatEntry{{Label: "ON", Value: 1}}))
		Expect(changes).To(ConsistOf(
			"time format unix_sec was dropped, because the field is float",
		))
	})

	It("Should lift an unknown data type to the float kind and describe it", func() {
		f, changes := v2.MigrateReadField(storedField("complex128"))
		Expect(f.Type()).To(Equal(xpayload.KindFloat))
		Expect(v2.ReadOf(f).Number.DataType).To(BeNil())
		Expect(changes).To(ConsistOf(
			`unknown data type "complex128" became the float kind`,
		))
	})

	It("Should lift an unknown data type with a time format to the float kind", func() {
		old := storedField("complex128")
		old.TimeFormat = new(xpayload.TimeFormatUnixSec)
		f, changes := v2.MigrateReadField(old)
		Expect(f.Type()).To(Equal(xpayload.KindFloat))
		Expect(changes).To(ConsistOf(
			`unknown data type "complex128" became the float kind`,
			"time format unix_sec was dropped, because the field is float",
		))
	})

	It("Should take float64 for an empty data type", func() {
		f, changes := v2.MigrateReadField(storedField(""))
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
		Expect(plain.Fields[0].Type()).To(Equal(xpayload.KindUnsigned))
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
					xpayload.EnumEntry{Label: "ON", Value: 1},
					xpayload.EnumEntry{Label: "HALF", Value: 0.5},
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
			[]xpayload.IntegerEntry{{Label: "ON", Value: 1}},
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
		kinds := map[uuid.UUID]xpayload.Kind{}
		for _, cfg := range got {
			plain, ok := cfg.Entries[0].Variant.(v2.PlainReadEntry)
			Expect(ok).To(BeTrue())
			kinds[cfg.Key] = plain.Fields[0].Type()
		}
		Expect(kinds).To(Equal(map[uuid.UUID]xpayload.Kind{
			known.Key:   xpayload.KindUnsigned,
			unknown.Key: xpayload.KindFloat,
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
				f := storedField("", xpayload.EnumEntry{Label: "ON", Value: 1})
				f.TimeFormat = new(xpayload.TimeFormatUnixSec)
				return f
			}(),
			v2.ReadField{Variant: v2.UnsignedReadField{
				BaseReadField: v2.BaseReadField{
					Key: "f1", Name: "level", Channel: 7, Pointer: "/level",
				},
				UnsignedRead: xpayload.UnsignedRead{
					EnumValues: []xpayload.UnsignedEntry{{Label: "ON", Value: 1}},
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
				f.TimeFormat = new(xpayload.TimeFormatUnixMs)
				return f
			}(),
			v2.ReadField{Variant: v2.TimestampReadField{
				BaseReadField: v2.BaseReadField{
					Key: "f1", Name: "level", Channel: 7, Pointer: "/level",
				},
				TimestampRead: xpayload.TimestampRead{
					TimeFormat: xpayload.TimeFormatUnixMs,
				},
			}},
			[]string{
				"data type float64 became timestamp, the data type of its channel",
			},
		),
	)
})
