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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	http "github.com/synnaxlabs/synnax/pkg/service/http/versions/v4"
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	payloadv1 "github.com/synnaxlabs/x/payload/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Migrate", func() {
	base := v1.BaseWriteChannel{Key: "c1", Disabled: true, Channel: 7}
	stored := func(c v0.WriteChannel) v0.WriteChannel {
		c.Key, c.Disabled, c.Channel = "c1", true, 7
		return c
	}
	unixMs := payload.TimeFormatUnixMs
	onOff := []payload.EnumEntry{{Label: "OFF", Value: 0}, {Label: "ON", Value: 1}}

	DescribeTable("MigrateWriteChannel",
		func(ctx SpecContext, old v0.WriteChannel, expected v1.WriteChannelVariant) {
			Expect(v1.MigrateWriteChannel(ctx, stored(old))).
				To(Equal(v1.WriteChannel{Variant: expected}))
		},
		Entry("a number as a float",
			v0.WriteChannel{JSONType: payload.JSONTypeNumber},
			v1.FloatWriteChannel{
				BaseWriteChannel: base,
				EnumValues:       []payload.FloatEntry{},
			}),
		Entry("an absent JSON type as a float",
			v0.WriteChannel{},
			v1.FloatWriteChannel{
				BaseWriteChannel: base,
				EnumValues:       []payload.FloatEntry{},
			}),
		Entry("a string with entries as a string with text entries",
			v0.WriteChannel{JSONType: payload.JSONTypeString, EnumValues: onOff},
			v1.StringWriteChannel{
				BaseWriteChannel: base,
				EnumValues: []payload.StringEntry{
					{Label: "OFF", Value: "0"},
					{Label: "ON", Value: "1"},
				},
			}),
		Entry("each entry value as its decimal text",
			v0.WriteChannel{
				JSONType: payload.JSONTypeString,
				EnumValues: []payload.EnumEntry{
					{Label: "OFF", Value: math.Copysign(0, -1)},
					{Label: "HALF", Value: 0.5},
					{Label: "BIG", Value: 1e21},
					{Label: "HIGH", Value: math.Inf(1)},
				},
			},
			v1.StringWriteChannel{
				BaseWriteChannel: base,
				EnumValues: []payload.StringEntry{
					{Label: "OFF", Value: "0"},
					{Label: "HALF", Value: "0.5"},
					{Label: "BIG", Value: "1000000000000000000000"},
					{Label: "HIGH", Value: "+Inf"},
				},
			}),
		Entry("a repeated entry value as the first entry alone",
			v0.WriteChannel{
				JSONType: payload.JSONTypeString,
				EnumValues: []payload.EnumEntry{
					{Label: "A", Value: 1},
					{Label: "B", Value: 1},
					{Label: "C", Value: math.NaN()},
					{Label: "D", Value: math.NaN()},
				},
			},
			v1.StringWriteChannel{
				BaseWriteChannel: base,
				EnumValues: []payload.StringEntry{
					{Label: "A", Value: "1"},
					{Label: "C", Value: "NaN"},
				},
			}),
		Entry("a number with entries as a float that keeps them",
			v0.WriteChannel{JSONType: payload.JSONTypeNumber, EnumValues: onOff},
			v1.FloatWriteChannel{
				BaseWriteChannel: base,
				EnumValues: []payload.FloatEntry{
					{Label: "OFF", Value: 0},
					{Label: "ON", Value: 1},
				},
			}),
		Entry("an unknown JSON type as a float",
			v0.WriteChannel{JSONType: "object"},
			v1.FloatWriteChannel{
				BaseWriteChannel: base,
				EnumValues:       []payload.FloatEntry{},
			}),
		Entry("a string with no entries as a string",
			v0.WriteChannel{JSONType: payload.JSONTypeString},
			v1.StringWriteChannel{
				BaseWriteChannel: base,
				EnumValues:       []payload.StringEntry{},
			}),
		Entry("a boolean as a boolean",
			v0.WriteChannel{JSONType: payload.JSONTypeBoolean, EnumValues: onOff},
			v1.BooleanWriteChannel{BaseWriteChannel: base}),
		Entry("a number with a time format as a timestamp",
			v0.WriteChannel{JSONType: payload.JSONTypeNumber, TimeFormat: &unixMs},
			v1.TimestampWriteChannel{
				BaseWriteChannel: base,
				TimeFormat:       payload.TimeFormatUnixMs,
			}),
		Entry("a string with entries and a stale time format as a string",
			v0.WriteChannel{
				JSONType:   payload.JSONTypeString,
				TimeFormat: &unixMs,
				EnumValues: onOff,
			},
			v1.StringWriteChannel{
				BaseWriteChannel: base,
				EnumValues: []payload.StringEntry{
					{Label: "OFF", Value: "0"},
					{Label: "ON", Value: "1"},
				},
			}),
	)

	It("Should lift every channel and keep the rest of a write config", func(
		ctx SpecContext,
	) {
		old := v0.WriteConfig{
			Topic:     "samples",
			RecordKey: v0.RecordKeyNone,
			Channels: []v0.WriteChannel{
				stored(v0.WriteChannel{JSONType: payload.JSONTypeBoolean}),
			},
		}
		cfg := MustSucceed(v1.MigrateWriteConfig(ctx, old))
		Expect(cfg.Topic).To(Equal("samples"))
		Expect(cfg.RecordKey).To(Equal(v1.RecordKeyNone))
		Expect(cfg.Channels).To(HaveExactElements(
			v1.WriteChannel{Variant: v1.BooleanWriteChannel{BaseWriteChannel: base}},
		))
	})

	It("Should lift the fields of a record and keep its pointers", func(
		ctx SpecContext,
	) {
		rec := MustSucceed(v1.MigrateRecord(ctx, v0.Record{
			ValuePointer:   "/v",
			ChannelPointer: new("/c"),
			TimeFormat:     payload.TimeFormatUnixMs,
			Fields: []http.WriteField{
				{Variant: http.StaticWriteField{
					BaseWriteField: http.BaseWriteField{Key: "s", Pointer: "/s"},
					JSONType:       payload.JSONTypeString,
					Value:          "plant-1",
				}},
				{Variant: http.GeneratedWriteField{
					BaseWriteField: http.BaseWriteField{Key: "id", Pointer: "/id"},
					Generator:      payload.GeneratorTypeUUID,
				}},
				{Variant: http.GeneratedWriteField{Generator: "id"}},
			},
		}))
		Expect(rec.ValuePointer).To(Equal("/v"))
		Expect(rec.ChannelPointer).To(Equal(new("/c")))
		Expect(rec.TimeFormat).To(Equal(payload.TimeFormatUnixMs))
		Expect(rec.Fields).To(Equal([]payloadv1.WriteField{
			{Variant: payloadv1.StringWriteField{
				BaseWriteField: payloadv1.BaseWriteField{Key: "s", Pointer: "/s"},
				Value:          "plant-1",
			}},
			{Variant: payloadv1.UuidWriteField{
				BaseWriteField: payloadv1.BaseWriteField{Key: "id", Pointer: "/id"},
			}},
		}))
	})
})
