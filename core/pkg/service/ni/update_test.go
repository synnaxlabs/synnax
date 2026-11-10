// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ni_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/ni"
	"github.com/synnaxlabs/x/patch"
)

var _ = Describe("Update", func() {
	voltage := func(key string, port uint32) ni.AIChannel {
		var v ni.AIVoltageChannel
		v.ApplyDefaults()
		v.Key, v.Name, v.Port, v.MinVal = key, "Pressure", port, -10
		return ni.AIChannel{Variant: v}
	}
	var config ni.AnalogReadConfig
	BeforeEach(func() {
		config = ni.AnalogReadConfig{
			Channels: []ni.AIChannel{voltage("c3", 1), voltage("c4", 2)},
		}
	})
	apply := func(
		ops patch.List[ni.AIChannel, string, ni.AIChannelUpdate],
	) ni.AnalogReadConfig {
		GinkgoHelper()
		next := config
		Expect(ni.AnalogReadConfigUpdate{Channels: &ops}.ApplyTo(&next)).To(Succeed())
		return next
	}
	updateChannel := func(u ni.AIChannelUpdateVariant) ni.AnalogReadConfig {
		GinkgoHelper()
		return apply(patch.List[ni.AIChannel, string, ni.AIChannelUpdate]{
			Update: []ni.AIChannelUpdate{{Variant: u}},
		})
	}
	keys := func(c ni.AnalogReadConfig) []string {
		return lo.Map(c.Channels, func(ch ni.AIChannel, _ int) string {
			return ch.Variant.(ni.AIVoltageChannel).Key
		})
	}

	It("Should merge an update into a channel of the same variant", func() {
		next := updateChannel(ni.AIVoltageChannelUpdate{Key: "c3", MaxVal: new(5.0)})
		c3 := next.Channels[0].Variant.(ni.AIVoltageChannel)
		Expect(c3.MaxVal).To(Equal(5.0))
		Expect(c3.MinVal).To(Equal(-10.0))
		Expect(config.Channels[0].Variant.(ni.AIVoltageChannel).MaxVal).
			ToNot(Equal(5.0))
	})

	It("Should insert a channel at the end", func() {
		next := apply(patch.List[ni.AIChannel, string, ni.AIChannelUpdate]{
			Insert: []ni.AIChannel{voltage("c5", 3)},
		})
		Expect(keys(next)).To(Equal([]string{"c3", "c4", "c5"}))
	})

	It("Should remove a channel", func() {
		next := apply(patch.List[ni.AIChannel, string, ni.AIChannelUpdate]{
			Remove: []string{"c3"},
		})
		Expect(keys(next)).To(Equal([]string{"c4"}))
	})

	It("Should move a channel first", func() {
		next := apply(patch.List[ni.AIChannel, string, ni.AIChannelUpdate]{
			Move: []patch.ListMove[string]{{Key: "c4"}},
		})
		Expect(keys(next)).To(Equal([]string{"c4", "c3"}))
	})

	It("Should change the variant and keep the base fields", func() {
		next := updateChannel(ni.AICurrentChannelUpdate{Key: "c3"})
		c3 := next.Channels[0].Variant.(ni.AICurrentChannel)
		Expect(c3.Key).To(Equal("c3"))
		Expect(c3.Name).To(Equal("Pressure"))
		Expect(c3.Port).To(Equal(uint32(1)))
		Expect(c3.ExtShuntResistorVal).To(Equal(1.0))
		Expect(c3.MinVal).To(Equal(0.0))
	})

	It("Should change the variant of a nested scale", func() {
		next := updateChannel(ni.AIVoltageChannelUpdate{
			Key: "c3",
			CustomScale: &ni.ScaleUpdate{
				Variant: ni.LinearScaleUpdate{YIntercept: new(2.0)},
			},
		})
		scale := next.Channels[0].Variant.(ni.AIVoltageChannel).CustomScale.CustomScale
		linear := scale.Variant.(ni.LinearScale)
		Expect(linear.YIntercept).To(Equal(2.0))
		Expect(linear.Slope).To(Equal(1.0))
	})

	It("Should rename a channel without a type", func() {
		next := updateChannel(ni.AIChannelBaseUpdate{Key: "c3", Name: new("Flow")})
		c3 := next.Channels[0].Variant.(ni.AIVoltageChannel)
		Expect(c3.Name).To(Equal("Flow"))
		Expect(c3.Port).To(Equal(uint32(1)))
	})

	It("Should return a stale error for a channel that does not exist", func() {
		ops := patch.List[ni.AIChannel, string, ni.AIChannelUpdate]{
			Update: []ni.AIChannelUpdate{{
				Variant: ni.AIVoltageChannelUpdate{Key: "c9", MaxVal: new(5.0)},
			}},
		}
		next := config
		Expect(ni.AnalogReadConfigUpdate{Channels: &ops}.ApplyTo(&next)).To(And(
			MatchError(patch.ErrStale),
			MatchError(ContainSubstring("c9 not found")),
		))
		Expect(keys(config)).To(Equal([]string{"c3", "c4"}))
	})
})
