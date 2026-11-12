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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/mqtt/versions/v2"
	payload "github.com/synnaxlabs/x/payload/versions/v0"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

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
		Entry("a time format on a bound string field as a string",
			v1.ChannelField{
				Channel:    7,
				JSONType:   payload.JSONTypeString,
				TimeFormat: &unixMs,
			},
			v2.StringChannelField{
				Pointer:  "/v",
				Channel:  7,
				Name:     "cmd",
				DataType: new(telem.Float64T),
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
		Entry("a time format on a bound boolean field as a boolean",
			v1.ChannelField{
				Channel:    7,
				JSONType:   payload.JSONTypeBoolean,
				TimeFormat: &unixMs,
			},
			v2.BooleanChannelField{
				Pointer:  "/v",
				Channel:  7,
				Name:     "cmd",
				DataType: new(telem.Float64T),
			}),
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
})
