// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/math"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Key", func() {
	Describe("NewKey", func() {
		It("Should return the correct leaseholder for the key", func() {
			k := channel.NewKey(node.Key(1), 1)
			Expect(k.Lease()).To(Equal(node.Key(1)))
		})
		It("Should return the correct localKey for the key", func() {
			k := channel.NewKey(node.Key(1), 2)
			Expect(k.LocalKey()).To(Equal(channel.LocalKey(2)))
		})
		It(
			"Should correctly handle the maximum value of a 12 bit node key and 20 bit ts key",
			func() {
				k := channel.NewKey(
					node.Key(math.MaxUint12), channel.LocalKey(math.MaxUint20),
				)
				Expect(k.Lease()).To(Equal(node.Key(math.MaxUint12)))
				Expect(k.LocalKey()).To(Equal(channel.LocalKey(math.MaxUint20)))
			},
		)
	})
	Describe("Free", func() {
		It("Should return true if the channel is a free channel", func() {
			k := channel.NewKey(node.KeyFree, 1)
			Expect(k.Free()).To(BeTrue())
		})
		It("Should return false if the channel is leased to a node", func() {
			k := channel.NewKey(node.Key(1), 1)
			Expect(k.Free()).To(BeFalse())
		})
	})
	Describe("StorageKey", func() {
		It("Should return the storage representation of the key", func() {
			Expect(channel.Key(42).StorageKey()).To(Equal(ts.ChannelKey(42)))
		})
	})
	DescribeTable("String", func(k channel.Key, expected string) {
		Expect(k.String()).To(Equal(expected))
	},
		Entry("Should return the string representation of the key",
			channel.Key(42),
			"42",
		),
		Entry("Should work for 0", channel.Key(0), "0"),
		Entry(
			"Should work for max value",
			channel.NewKey(node.Key(math.MaxUint12), channel.LocalKey(math.MaxUint20)),
			"4294967295",
		),
	)
})

var _ = Describe("Keys", func() {
	Describe("KeysFromChannels", func() {
		It("Should return a list of keys from a list of channels", func() {
			channels := []channel.Channel{
				{Leaseholder: 1, LocalKey: 1},
				{Leaseholder: 1, LocalKey: 2},
			}
			keys := channel.KeysFromChannels(channels)
			Expect(keys).To(Equal(channel.Keys{
				channel.NewKey(node.Key(1), 1),
				channel.NewKey(node.Key(1), 2),
			}))
		})
	})
	Describe("KeysFromUint32", func() {
		It(
			"Should correctly --reinterpret-- a slice of uint32 into a slice of keys",
			func() {
				uint32s := []uint32{1, 2, 3}
				keys := channel.KeysFromUint32(uint32s)
				Expect(keys).To(Equal(channel.Keys{1, 2, 3}))
			},
		)
	})
	Describe("Uint32", func() {
		It("Should correctly reinterpret the keys as a slice of uint32", func() {
			keys := channel.Keys{1, 2, 3}
			Expect(keys.Uint32()).To(Equal([]uint32{1, 2, 3}))
		})
	})
	Describe("Storage", func() {
		It("Should correctly return the storage representation of the keys", func() {
			keys := channel.Keys{1, 2, 3}
			Expect(keys.Storage()).To(Equal([]ts.ChannelKey{1, 2, 3}))
		})
	})
	Describe("UniqueLeaseholders", func() {
		It("Should return a slice of the unique node ids for a set of keys", func() {
			ids := channel.Keys{
				channel.NewKey(node.Key(1), 2),
				channel.NewKey(node.Key(3), 4),
				channel.NewKey(node.Key(1), 2),
			}
			Expect(ids.UniqueLeaseholders()).To(ConsistOf([]node.Key{1, 3}))
		})
	})
	Describe("Contains", func() {
		It("Should return true if the slice contains the given key", func() {
			keys := channel.Keys{1, 2, 3}
			Expect(keys.Contains(2)).To(BeTrue())
		})
		It("Should return false if the slice does not contain the given key", func() {
			keys := channel.Keys{1, 2, 3}
			Expect(keys.Contains(4)).To(BeFalse())
		})
	})
	Describe("Unique", func() {
		It("Should remove duplicate keys from the slice and return the result", func() {
			keys := channel.Keys{1, 2, 3, 2, 1}
			Expect(keys.Unique()).To(Equal(channel.Keys{1, 2, 3}))
		})
	})
})

var _ = Describe("Channel", func() {
	Describe("Key", func() {
		It("Should combine the leaseholder and local key into a channel key", func() {
			c := channel.Channel{Leaseholder: 1, LocalKey: 2}
			Expect(c.Key()).To(Equal(channel.NewKey(node.Key(1), 2)))
		})
	})
	Describe("Index", func() {
		It("Should return a zero key when the channel has no local index", func() {
			c := channel.Channel{Leaseholder: 1, LocalKey: 2}
			Expect(c.Index()).To(Equal(channel.Key(0)))
		})
		It(
			"Should return the index key built from the leaseholder and local index",
			func() {
				c := channel.Channel{Leaseholder: 1, LocalKey: 2, LocalIndex: 3}
				Expect(c.Index()).To(Equal(channel.NewKey(node.Key(1), 3)))
			},
		)
	})
	Describe("Lease", func() {
		It("Should return the channel's leaseholder", func() {
			c := channel.Channel{Leaseholder: 7}
			Expect(c.Lease()).To(Equal(node.Key(7)))
		})
	})
	Describe("Free", func() {
		It("Should return true when the channel is not leased to a node", func() {
			c := channel.Channel{Leaseholder: node.KeyFree}
			Expect(c.Free()).To(BeTrue())
		})
		It("Should return false when the channel is leased to a node", func() {
			c := channel.Channel{Leaseholder: 1}
			Expect(c.Free()).To(BeFalse())
		})
	})
	Describe("String", func() {
		It("Should include the name and key when the channel is named", func() {
			c := channel.Channel{Name: "temp", LocalKey: 5}
			Expect(c.String()).To(Equal("[temp]<5>"))
		})
		It("Should include only the key when the channel is unnamed", func() {
			c := channel.Channel{LocalKey: 5}
			Expect(c.String()).To(Equal("<5>"))
		})
	})
	Describe("Storage", func() {
		It("Should map the channel to its storage representation", func() {
			c := channel.Channel{
				Name:        "sensor",
				LocalKey:    2,
				LocalIndex:  3,
				DataType:    telem.Float32T,
				IsIndex:     true,
				Virtual:     true,
				Concurrency: control.ConcurrencyShared,
			}
			Expect(c.Storage()).To(Equal(ts.Channel{
				Key:         ts.ChannelKey(2),
				Name:        "sensor",
				IsIndex:     true,
				DataType:    telem.Float32T,
				Index:       ts.ChannelKey(3),
				Virtual:     true,
				Concurrency: control.ConcurrencyShared,
			}))
		})
		It("Should preserve virtuality in a channel's storage registration", func() {
			ch := channel.Channel{
				Name:        "free_ch",
				LocalKey:    1,
				Leaseholder: node.KeyFree,
				DataType:    telem.Int64T,
				Virtual:     true,
			}
			stored := ch.Storage()
			Expect(stored.Key).To(Equal(ch.Key().StorageKey()))
			Expect(stored.Virtual).To(BeTrue())
		})
		It(
			"Should not mark a stored channel's storage registration as virtual",
			func() {
				ch := channel.Channel{
					Name:        "leased_ch",
					LocalKey:    1,
					Leaseholder: 1,
					DataType:    telem.Int64T,
				}
				Expect(ch.Storage().Virtual).To(BeFalse())
			},
		)
	})
})
