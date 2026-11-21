// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v3 "github.com/synnaxlabs/synnax/pkg/service/ni/versions/v3"
)

var _ = Describe("AnalogReadRangeMigration", func() {
	DescribeTable("Should order the range of a stored channel",
		func(ctx SpecContext, stored, want v3.MinMaxVal) {
			got, _ := runStored[v3.AnalogReadConfig, v3.AnalogReadConfig](
				ctx,
				"ni_analog_read_config",
				v3.AnalogReadRangeMigration,
				v3.AnalogReadConfig{Key: uuid.New(), Channels: []v3.AIChannel{
					{Variant: v3.AITempBuiltinChannel{Key: "temp"}},
					{Variant: v3.AIVoltageChannel{
						MinMaxVal:   stored,
						CustomScale: noScale,
					}},
				}},
			)
			Expect(got.Channels[0].Variant).To(
				Equal(v3.AITempBuiltinChannel{Key: "temp"}),
			)
			Expect(got.Channels[1].Variant).To(
				Equal(v3.AIVoltageChannel{MinMaxVal: want, CustomScale: noScale}),
			)
		},
		Entry("swapping a reversed range",
			v3.MinMaxVal{MinVal: 5, MaxVal: -5}, v3.MinMaxVal{MinVal: -5, MaxVal: 5}),
		Entry("giving an equal range the default of the channel type",
			v3.MinMaxVal{MinVal: 3, MaxVal: 3}, v3.MinMaxVal{MinVal: 0, MaxVal: 1}),
		Entry("keeping a range in order",
			v3.MinMaxVal{MinVal: -2, MaxVal: 2}, v3.MinMaxVal{MinVal: -2, MaxVal: 2}),
	)
})

var _ = Describe("AnalogWriteRangeMigration", func() {
	It("Should swap a reversed range", func(ctx SpecContext) {
		got, _ := runStored[v3.AnalogWriteConfig, v3.AnalogWriteConfig](
			ctx,
			"ni_analog_write_config",
			v3.AnalogWriteRangeMigration,
			v3.AnalogWriteConfig{Key: uuid.New(), Channels: []v3.AOChannel{
				{Variant: v3.AOVoltageChannel{
					MinVal:      10,
					MaxVal:      -10,
					CustomScale: noScale,
				}},
			}},
		)
		Expect(got.Channels[0].Variant).To(Equal(v3.AOVoltageChannel{
			MinVal:      -10,
			MaxVal:      10,
			CustomScale: noScale,
		}))
	})
})

var _ = Describe("CounterReadRangeMigration", func() {
	It("Should give an equal range the default of the channel type",
		func(ctx SpecContext) {
			got, _ := runStored[v3.CounterReadConfig, v3.CounterReadConfig](
				ctx,
				"ni_counter_read_config",
				v3.CounterReadRangeMigration,
				v3.CounterReadConfig{Key: uuid.New(), Channels: []v3.CIChannel{
					{Variant: v3.CIFrequencyChannel{
						MinVal:      50,
						MaxVal:      50,
						CustomScale: noScale,
						Divisor:     4,
					}},
				}},
			)
			Expect(got.Channels[0].Variant).To(Equal(v3.CIFrequencyChannel{
				MinVal:      2,
				MaxVal:      100,
				CustomScale: noScale,
				Divisor:     4,
			}))
		},
	)
})
