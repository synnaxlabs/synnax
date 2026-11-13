// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v5_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v5 "github.com/synnaxlabs/synnax/pkg/service/http/versions/v5"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

func readConfig(fields ...v5.ReadField) v5.ReadConfig {
	return v5.ReadConfig{
		Device:    "dev",
		Rate:      1,
		Endpoints: []v5.ReadEndpoint{{Key: "ep", Method: v5.MethodGet, Fields: fields}},
	}
}

func base(ch channel.Key) v5.BaseReadField {
	return v5.BaseReadField{Key: "f", Channel: ch, Pointer: "/v"}
}

func ignoreWarnings(string, ...zap.Field) {}

var _ = Describe("Resolve", func() {
	channels := storedChannels(map[channel.Key]v5.Channel{
		1: {DataType: telem.Float32T},
		2: {DataType: telem.Int8T},
		3: {DataType: telem.TimestampT, IsIndex: true},
		4: {DataType: telem.StringT},
		5: {DataType: telem.TimestampT},
		6: {DataType: telem.Float64T},
	})

	Describe("ReconcileReadConfig", func() {
		reconcile := func(ctx SpecContext, f v5.ReadField) v5.ReadField {
			GinkgoHelper()
			cfg := readConfig(f)
			Expect(v5.ReconcileReadConfig(ctx, nil, channels, &cfg, ignoreWarnings)).
				To(Succeed())
			return cfg.Endpoints[0].Fields[0]
		}

		It("Should convert entries and the fill to another numeric kind", func(
			ctx SpecContext,
		) {
			f := reconcile(ctx, v5.ReadField{Variant: v5.FloatReadField{
				BaseReadField: base(2),
				FloatRead: payload.FloatRead{
					DataType: new(telem.Float64T),
					EnumValues: []payload.FloatEntry{
						{Label: "LOW", Value: -2},
						{Label: "HALF", Value: 0.5},
						{Label: "BIG", Value: 300},
					},
					Fill: new(payload.Float(4)),
				},
			}})
			integer := f.Variant.(v5.IntegerReadField)
			Expect(integer.BaseReadField).To(Equal(base(2)))
			Expect(integer.DataType).To(BeNil())
			Expect(integer.EnumValues).To(Equal(
				[]payload.IntegerEntry{{Label: "LOW", Value: -2}},
			))
			Expect(*integer.Fill).To(Equal(payload.Integer(4)))
		})

		It(
			"Should drop a non-finite fill on an integer channel",
			func(ctx SpecContext) {
				f := reconcile(ctx, v5.ReadField{Variant: v5.FloatReadField{
					BaseReadField: base(2),
					FloatRead: payload.FloatRead{
						Fill: new(payload.Float(math.NaN())),
					},
				}})
				Expect(f.Variant.(v5.IntegerReadField).Fill).To(BeNil())
			},
		)

		It("Should drop an integer that has no equal float on a float channel", func(
			ctx SpecContext,
		) {
			f := reconcile(ctx, v5.ReadField{Variant: v5.IntegerReadField{
				BaseReadField: base(6),
				IntegerRead: payload.IntegerRead{
					EnumValues: []payload.IntegerEntry{
						{Label: "EXACT", Value: 1 << 53},
						{Label: "ROUNDED", Value: 1<<53 + 1},
					},
					Fill: new(payload.Integer(1<<53 + 1)),
				},
			}})
			float := f.Variant.(v5.FloatReadField)
			Expect(float.EnumValues).To(Equal(
				[]payload.FloatEntry{{Label: "EXACT", Value: 1 << 53}},
			))
			Expect(float.Fill).To(BeNil())
		})

		It("Should drop an empty enum label on a kind that is not string", func(
			ctx SpecContext,
		) {
			f := reconcile(ctx, v5.ReadField{Variant: v5.IntegerReadField{
				BaseReadField: base(2),
				IntegerRead: payload.IntegerRead{EnumValues: []payload.IntegerEntry{
					{Label: "", Value: 1},
					{Label: "ON", Value: 2},
				}},
			}})
			Expect(f.Variant.(v5.IntegerReadField).EnumValues).To(Equal(
				[]payload.IntegerEntry{{Label: "ON", Value: 2}},
			))
		})

		It("Should drop the entries and fill on a kind that is not numeric", func(
			ctx SpecContext,
		) {
			f := reconcile(ctx, v5.ReadField{Variant: v5.FloatReadField{
				BaseReadField: base(4),
				FloatRead: payload.FloatRead{
					EnumValues: []payload.FloatEntry{{Label: "ON", Value: 1}},
					Fill:       new(payload.Float(1)),
				},
			}})
			str := f.Variant.(v5.StringReadField)
			Expect(str.EnumValues).To(BeEmpty())
			Expect(str.Fill).To(BeNil())
		})

		It("Should give a new timestamp field ISO 8601", func(ctx SpecContext) {
			f := reconcile(ctx, v5.ReadField{Variant: v5.FloatReadField{
				BaseReadField: base(5),
			}})
			Expect(f.Variant.(v5.TimestampReadField).TimeFormat).
				To(Equal(payload.TimeFormatISO8601))
		})

		It(
			"Should drop the fill of a field on an index channel",
			func(ctx SpecContext) {
				f := reconcile(ctx, v5.ReadField{Variant: v5.TimestampReadField{
					BaseReadField: base(3),
					TimestampRead: payload.TimestampRead{
						TimeFormat: payload.TimeFormatUnixSec,
						Fill:       new(telem.SecondTS),
					},
				}})
				ts := f.Variant.(v5.TimestampReadField)
				Expect(ts.TimeFormat).To(Equal(payload.TimeFormatUnixSec))
				Expect(ts.Fill).To(BeNil())
			},
		)

		It("Should drop a float fill past the range of the channel", func(
			ctx SpecContext,
		) {
			f := reconcile(ctx, v5.ReadField{Variant: v5.FloatReadField{
				BaseReadField: base(1),
				FloatRead:     payload.FloatRead{Fill: new(payload.Float(1e39))},
			}})
			Expect(f.Variant.(v5.FloatReadField).Fill).To(BeNil())
		})

		It("Should leave a field whose channel does not exist", func(ctx SpecContext) {
			original := v5.ReadField{Variant: v5.FloatReadField{
				BaseReadField: base(99),
				FloatRead:     payload.FloatRead{DataType: new(telem.Int8T)},
			}}
			Expect(reconcile(ctx, original)).To(Equal(original))
		})
	})

	Describe("ResolveReadConfig", func() {
		resolve := func(ctx SpecContext, f v5.ReadField) error {
			cfg := readConfig(f)
			return v5.ResolveReadConfig(ctx, nil, channels, &cfg)
		}

		resolved := func(ctx SpecContext, f v5.ReadField) v5.ReadField {
			GinkgoHelper()
			cfg := readConfig(f)
			Expect(v5.ResolveReadConfig(ctx, nil, channels, &cfg)).To(Succeed())
			return cfg.Endpoints[0].Fields[0]
		}

		It("Should drop the data type of a field", func(ctx SpecContext) {
			f := resolved(ctx, v5.ReadField{Variant: v5.IntegerReadField{
				BaseReadField: base(2),
				IntegerRead: payload.IntegerRead{
					DataType: new(telem.Int64T),
					Fill:     new(payload.Integer(-5)),
				},
			}})
			integer := f.Variant.(v5.IntegerReadField)
			Expect(integer.DataType).To(BeNil())
			Expect(*integer.Fill).To(Equal(payload.Integer(-5)))
		})

		It("Should keep an empty enum label for validation to reject", func(
			ctx SpecContext,
		) {
			cfg := readConfig(v5.ReadField{Variant: v5.IntegerReadField{
				BaseReadField: base(2),
				IntegerRead: payload.IntegerRead{
					EnumValues: []payload.IntegerEntry{{Label: "", Value: 1}},
				},
			}})
			Expect(v5.ResolveReadConfig(ctx, nil, channels, &cfg)).To(Succeed())
			Expect(v5.ValidateReadConfig(&cfg)).
				To(MatchError(ContainSubstring("enum label is empty")))
		})

		DescribeTable("Should reject a field that does not fit its channel",
			func(ctx SpecContext, f v5.ReadField, msg string) {
				Expect(resolve(ctx, f)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry("a kind that is not the channel's",
				v5.ReadField{Variant: v5.FloatReadField{BaseReadField: base(2)}},
				"kind float does not match kind integer of data type int8"),
			Entry("a fill past the channel's range",
				v5.ReadField{Variant: v5.IntegerReadField{
					BaseReadField: base(2),
					IntegerRead:   payload.IntegerRead{Fill: new(payload.Integer(200))},
				}},
				"fill: 200 does not fit data type int8"),
			Entry("an entry past the channel's range",
				v5.ReadField{Variant: v5.IntegerReadField{
					BaseReadField: base(2),
					IntegerRead: payload.IntegerRead{
						EnumValues: []payload.IntegerEntry{{Label: "BIG", Value: -200}},
					},
				}},
				"enum_values.0.value: -200 does not fit data type int8"),
			Entry("a fill on an index channel",
				v5.ReadField{Variant: v5.TimestampReadField{
					BaseReadField: base(3),
					TimestampRead: payload.TimestampRead{Fill: new(telem.TimeStamp(0))},
				}},
				"an index channel cannot have a fill"),
		)

		It("Should skip a disabled field", func(ctx SpecContext) {
			b := base(2)
			b.Disabled = true
			Expect(resolve(ctx, v5.ReadField{Variant: v5.FloatReadField{
				BaseReadField: b,
			}})).To(Succeed())
		})

		It("Should skip a field whose channel does not exist", func(ctx SpecContext) {
			Expect(resolve(ctx, v5.ReadField{Variant: v5.StringReadField{
				BaseReadField: base(99),
			}})).To(Succeed())
		})
	})

	Describe("ValidateReadConfig", func() {
		It("Should accept a valid config", func() {
			cfg := readConfig(v5.ReadField{Variant: v5.UnsignedReadField{
				BaseReadField: base(1),
				UnsignedRead: payload.UnsignedRead{EnumValues: []payload.UnsignedEntry{
					{Label: "ON", Value: 1},
					{Label: "HIGH", Value: 1},
				}},
			}})
			Expect(v5.ValidateReadConfig(&cfg)).To(Succeed())
		})

		It("Should reject a repeated enum label", func() {
			cfg := readConfig(v5.ReadField{Variant: v5.StringReadField{
				BaseReadField: base(1),
				StringRead: payload.StringRead{EnumValues: []payload.StringEntry{
					{Label: "ON", Value: "on"},
					{Label: "ON", Value: "high"},
				}},
			}})
			Expect(v5.ValidateReadConfig(&cfg)).To(MatchError(ContainSubstring(
				`endpoints.0.fields.0.enum_values.1.label: enum label "ON" repeats`,
			)))
		})

		It("Should reject an empty enum label on a field that is not a string field",
			func() {
				cfg := readConfig(v5.ReadField{Variant: v5.FloatReadField{
					BaseReadField: base(1),
					FloatRead: payload.FloatRead{EnumValues: []payload.FloatEntry{
						{Label: "", Value: 1},
					}},
				}})
				Expect(v5.ValidateReadConfig(&cfg)).To(MatchError(ContainSubstring(
					"endpoints.0.fields.0.enum_values.0.label: enum label is empty",
				)))
			},
		)

		It("Should accept an empty enum label on a string field", func() {
			cfg := readConfig(v5.ReadField{Variant: v5.StringReadField{
				BaseReadField: base(1),
				StringRead: payload.StringRead{EnumValues: []payload.StringEntry{
					{Label: "", Value: "idle"},
				}},
			}})
			Expect(v5.ValidateReadConfig(&cfg)).To(Succeed())
		})

		It("Should reject a bytes field with no pointer beside another field", func() {
			raw := base(1)
			raw.Pointer = ""
			cfg := readConfig(
				v5.ReadField{Variant: v5.BytesReadField{BaseReadField: raw}},
				v5.ReadField{Variant: v5.FloatReadField{BaseReadField: base(2)}},
			)
			Expect(v5.ValidateReadConfig(&cfg)).To(MatchError(ContainSubstring(
				"must be the only enabled field",
			)))
		})

		It(
			"Should accept a bytes field with no pointer beside a disabled field",
			func() {
				raw := base(1)
				raw.Pointer = ""
				other := base(2)
				other.Disabled = true
				cfg := readConfig(
					v5.ReadField{Variant: v5.BytesReadField{BaseReadField: raw}},
					v5.ReadField{Variant: v5.FloatReadField{BaseReadField: other}},
				)
				Expect(v5.ValidateReadConfig(&cfg)).To(Succeed())
			},
		)
	})
})
