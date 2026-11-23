// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v4 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v4"
)

var _ = Describe("AnalogReadRangeMigration", func() {
	DescribeTable("Should order the range of a stored channel",
		func(ctx SpecContext, stored, want v4.MinMaxVal) {
			got, _ := runStored[v4.AnalogReadConfig, v4.AnalogReadConfig](
				ctx,
				"ni_analog_read_config",
				v4.AnalogReadRangeMigration,
				v4.AnalogReadConfig{Key: uuid.New(), Channels: []v4.AIChannel{
					{Variant: v4.AITempBuiltinChannel{Key: "temp"}},
					{Variant: v4.AIVoltageChannel{
						MinMaxVal:   stored,
						CustomScale: noScale,
					}},
				}},
			)
			Expect(got.Channels[0].Variant).To(
				Equal(v4.AITempBuiltinChannel{Key: "temp"}),
			)
			Expect(got.Channels[1].Variant).To(
				Equal(v4.AIVoltageChannel{MinMaxVal: want, CustomScale: noScale}),
			)
		},
		Entry("swapping a reversed range",
			v4.MinMaxVal{MinVal: 5, MaxVal: -5}, v4.MinMaxVal{MinVal: -5, MaxVal: 5}),
		Entry("giving an equal range the default of the channel type",
			v4.MinMaxVal{MinVal: 3, MaxVal: 3}, v4.MinMaxVal{MinVal: 0, MaxVal: 1}),
		Entry("keeping a range in order",
			v4.MinMaxVal{MinVal: -2, MaxVal: 2}, v4.MinMaxVal{MinVal: -2, MaxVal: 2}),
	)
})

var _ = Describe("AnalogWriteRangeMigration", func() {
	It("Should swap a reversed range", func(ctx SpecContext) {
		got, _ := runStored[v4.AnalogWriteConfig, v4.AnalogWriteConfig](
			ctx,
			"ni_analog_write_config",
			v4.AnalogWriteRangeMigration,
			v4.AnalogWriteConfig{Key: uuid.New(), Channels: []v4.AOChannel{
				{Variant: v4.AOVoltageChannel{
					MinVal:      10,
					MaxVal:      -10,
					CustomScale: noScale,
				}},
			}},
		)
		Expect(got.Channels[0].Variant).To(Equal(v4.AOVoltageChannel{
			MinVal:      -10,
			MaxVal:      10,
			CustomScale: noScale,
		}))
	})
})

var _ = Describe("CounterReadRangeMigration", func() {
	It("Should give an equal range the default of the channel type",
		func(ctx SpecContext) {
			got, _ := runStored[v4.CounterReadConfig, v4.CounterReadConfig](
				ctx,
				"ni_counter_read_config",
				v4.CounterReadRangeMigration,
				v4.CounterReadConfig{Key: uuid.New(), Channels: []v4.CIChannel{
					{Variant: v4.CIFrequencyChannel{
						MinVal:      50,
						MaxVal:      50,
						CustomScale: noScale,
						Divisor:     4,
					}},
				}},
			)
			Expect(got.Channels[0].Variant).To(Equal(v4.CIFrequencyChannel{
				MinVal:      2,
				MaxVal:      100,
				CustomScale: noScale,
				Divisor:     4,
			}))
		},
	)
})

var _ = Describe("OrderAIRanges", func() {
	It("Should return the indices of the channels it changes", func() {
		channels := []v4.AIChannel{
			{Variant: v4.AIVoltageChannel{MinVal: 0, MaxVal: 1}},
			{Variant: v4.AIVoltageChannel{MinVal: 1, MaxVal: 0}},
			{Variant: v4.AITempBuiltinChannel{}},
		}
		Expect(v4.OrderAIRanges(channels)).To(Equal([]int{1}))
		Expect(channels[1].Variant).To(Equal(v4.AIVoltageChannel{MinVal: 0, MaxVal: 1}))
	})
})

var _ = Describe("OrderAORanges", func() {
	It("Should give an equal range the default of the channel type", func() {
		channels := []v4.AOChannel{{Variant: v4.AOCurrentChannel{MinVal: 2, MaxVal: 2}}}
		Expect(v4.OrderAORanges(channels)).To(Equal([]int{0}))
		Expect(channels[0].Variant).To(Equal(v4.AOCurrentChannel{MinVal: 0, MaxVal: 1}))
	})
})

var _ = Describe("OrderCIRanges", func() {
	It("Should leave a channel without a range unchanged", func() {
		channels := []v4.CIChannel{{Variant: v4.CIEdgeCountChannel{InitialCount: 3}}}
		Expect(v4.OrderCIRanges(channels)).To(BeEmpty())
		Expect(channels[0].Variant).To(Equal(v4.CIEdgeCountChannel{InitialCount: 3}))
	})
})
