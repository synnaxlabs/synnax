// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

var _ = Describe("ChannelGroup", Ordered, func() {
	var (
		cg   channel.ChannelGroup
		keys channel.Keys
	)
	BeforeAll(func() {
		var uints []uint32
		for i := 0; i < 100; i++ {
			uints = append(uints, uint32(i))
		}
		keys = channel.KeysFromUint32(uints)

	})
	BeforeEach(func() {
		cg = channel.ChannelGroup{}
	})

	Describe("Create", func() {
		It("Should create a ChannelGroup", func() {
			cg.InsertKey(keys[0])
			Expect(cg.GetChannelsBeforeKey(keys[1])).To(Equal(uint16(1)))
			cg.InsertKey(keys[1])
			Expect(cg.GetChannelsBeforeKey(keys[2])).To(Equal(uint16(2)))
			cg.InsertKeys(keys[0:2])
			Expect(cg.GetChannelsBeforeKey(keys[2])).To(Equal(uint16(2)))

			cg.InsertKeys(keys[0:10])
			Expect(cg.GetChannelsBeforeKey(keys[10])).To(Equal(uint16(10)))

			newSlice := make(channel.Keys, 40)
			copy(newSlice[0:10], keys[0:10])
			copy(newSlice[10:30], keys[20:40])
			copy(newSlice[30:40], keys[50:60])

			cg.InsertKeys(newSlice)
			Expect(cg.Len()).To(Equal(3))
			Expect(cg.GetChannelsBeforeKey(keys[45])).To(Equal(uint16(30)))
		})
		It("Should insert an empty key into an empty channel", func() {
			cg.InsertKeys(keys[88:88])
			Expect(cg.GetChannelsBeforeKey(keys[10])).To(Equal(uint16(0)))
			Expect(cg.Len()).To(Equal(0))

		})
	})
	Describe("Check Compression", func() {
		It("Should create a compressed ChannelGroup", func() {
			cg.InsertKeys(keys[0:10])
			cg.InsertKey(keys[11])
			cg.InsertKeys(keys[14:20])
			cg.InsertKey(keys[10])
			cg.InsertKeys(keys[12:14])
			Expect(cg.GetChannelsBeforeKey(keys[20])).To(Equal(uint16(20)))
			Expect(cg.Len()).To(Equal(1))
		})
		It("should check edge cases", func() {
			cg.InsertKeys(keys[2:12])
			cg.InsertKeys(keys[0:2])
			cg.InsertKeys(keys[15:20])
			cg.InsertKeys(keys[20:24])
			cg.InsertKeys(keys[12:15])
			Expect(cg.Len()).To(Equal(1))
			Expect(cg.GetChannelsBeforeKey(keys[24])).To(Equal(uint16(24)))
		})
	})
	Describe("Check retrieval of keys", func() {
		It("Should retrieve the correct number while in the middle of a channelRange", func() {
			cg.InsertKeys(keys[0:25])
			Expect(cg.GetChannelsBeforeKey(keys[15])).To(Equal(uint16(15)))
		})
	})
	Describe("Check removal of keys", func() {
		It("Should accurately delete and reinsert slices of keys", func() {
			cg.InsertKeys(keys[0:25])
			cg.RemoveKeys(keys[10:20])
			Expect(cg.Len()).To(Equal(2))
			Expect(cg.GetChannelsBeforeKey(keys[25])).To(Equal(uint16(15)))
			Expect(cg.GetChannelsBeforeKey(keys[10])).To(Equal(uint16(10)))
			Expect(cg.GetChannelsBeforeKey(keys[20])).To(Equal(uint16(10)))
			cg.RemoveKeys(keys[0:10])
			cg.RemoveKeys(keys[20:25])
			Expect(cg.Len()).To(Equal(0))
			Expect(cg.GetChannelsBeforeKey(keys[10])).To(Equal(uint16(0)))
		})
		It("Should be idempotent when removing keys", func() {
			cg.RemoveKeys(keys[0:100])
			Expect(cg.Len()).To(Equal(0))
			Expect(cg.GetChannelsBeforeKey(keys[50])).To(Equal(uint16(0)))
			cg.InsertKeys(keys[0:10])
			cg.InsertKeys(keys[25:50])
			cg.RemoveKeys(keys[5:30])
			Expect(cg.Len()).To(Equal(2))
			Expect(cg.GetChannelsBeforeKey(keys[99])).To(Equal(uint16(25)))
		})
	})
	Describe("Testing when the ChannelGroup is empty", func() {
		It("should work", func() {
			Expect(cg.GetChannelsBeforeKey(keys[10])).To(Equal(uint16(0)))
			Expect(cg.Len()).To(Equal(0))
			cg.InsertKeys(keys[0:10])
			cg.RemoveKeys(keys[0:10])
			cg.RemoveKeys(keys[0:10])
		})

	})

})
