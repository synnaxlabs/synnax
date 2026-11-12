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
	v0 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/kafka/versions/v1"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
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
		Entry("a string with entries as a float that keeps them",
			v0.WriteChannel{JSONType: payload.JSONTypeString, EnumValues: onOff},
			v1.FloatWriteChannel{
				BaseWriteChannel: base,
				EnumValues: []payload.FloatEntry{
					{Label: "OFF", Value: 0},
					{Label: "ON", Value: 1},
				},
			}),
		Entry("a non-finite entry as it is",
			v0.WriteChannel{
				JSONType:   payload.JSONTypeString,
				EnumValues: []payload.EnumEntry{{Label: "HIGH", Value: math.Inf(1)}},
			},
			v1.FloatWriteChannel{
				BaseWriteChannel: base,
				EnumValues: []payload.FloatEntry{
					{Label: "HIGH", Value: payload.Float(math.Inf(1))},
				},
			}),
		Entry("a string with no entries as a string",
			v0.WriteChannel{JSONType: payload.JSONTypeString},
			v1.StringWriteChannel{BaseWriteChannel: base}),
		Entry("a boolean as a boolean",
			v0.WriteChannel{JSONType: payload.JSONTypeBoolean, EnumValues: onOff},
			v1.BooleanWriteChannel{BaseWriteChannel: base}),
		Entry("a time format as a timestamp, whatever the JSON type",
			v0.WriteChannel{JSONType: payload.JSONTypeString, TimeFormat: &unixMs},
			v1.TimestampWriteChannel{
				BaseWriteChannel: base,
				TimeFormat:       payload.TimeFormatUnixMs,
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
})
