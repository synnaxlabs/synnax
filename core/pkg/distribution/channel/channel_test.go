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
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/math"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Channel Tests", func() {
	Describe("Key Tests", func() {
		Describe("Construction", func() {
			It("Should return the correct leaseholder for the key", func() {
				k := channel.NewKey(cluster.NodeKey(1), 1)
				Expect(k.Leaseholder()).To(Equal(cluster.NodeKey(1)))
			})
			It("Should return the correct localKey for the key", func() {
				k := channel.NewKey(cluster.NodeKey(1), 2)
				Expect(k.LocalKey()).To(Equal(channel.LocalKey(2)))
			})
			It("Should correctly handle the maximum value of a 12 bit node key and 20 bit cesium key", func() {
				k := channel.NewKey(cluster.NodeKey(math.MaxUint12), channel.LocalKey(math.MaxUint20))
				Expect(k.Leaseholder()).To(Equal(cluster.NodeKey(math.MaxUint12)))
				Expect(k.LocalKey()).To(Equal(channel.LocalKey(math.MaxUint20)))
			})
		})
		Describe("ParseKey", func() {
			It("Should correctly parse a key from its string representation", func() {
				k, err := channel.ParseKey("123456")
				Expect(err).ToNot(HaveOccurred())
				Expect(k).To(Equal(channel.Key(123456)))
			})
			It("Should return an error when the key is not a valid integer", func() {
				_, err := channel.ParseKey("123456a")
				Expect(err).To(HaveOccurredAs(validate.ErrValidation))
				Expect(err.Error()).To(ContainSubstring("123456a is not a valid channel key"))
			})
		})
		Describe("Lease", func() {
			It("Should return the leaseholder node Name", func() {
				k := channel.NewKey(cluster.NodeKey(1), 1)
				Expect(k.Lease()).To(Equal(k.Leaseholder()))
			})
		})
		Describe("OntologyID", func() {
			It("Should return the ontology Name for the channel", func() {
				ok := channel.OntologyID(channel.NewKey(cluster.NodeKey(1), 2))
				Expect(ok).To(Equal(ontology.ID{
					Type: "channel",
					Key:  channel.NewKey(cluster.NodeKey(1), 2).String(),
				}))
			})
		})
		Describe("Free", func() {
			It("Should return true if the channel is a free channel", func() {
				k := channel.NewKey(cluster.NodeKeyFree, 1)
				Expect(k.Free()).To(BeTrue())
			})
			It("Should return false if the channel is not a free channel", func() {
				k := channel.NewKey(cluster.NodeKey(1), 1)
				Expect(k.Free()).To(BeFalse())
			})
		})
	})
	Describe("Keys Tests", func() {
		Describe("KeysFromChannels", func() {
			It("Should return a list of keys from a list of channels", func() {
				channels := []channel.Channel{
					{Leaseholder: 1, LocalKey: 1},
					{Leaseholder: 1, LocalKey: 2},
				}
				keys := channel.KeysFromChannels(channels)
				Expect(keys).To(Equal(channel.Keys{
					channel.NewKey(cluster.NodeKey(1), 1),
					channel.NewKey(cluster.NodeKey(1), 2),
				}))
			})
		})
		Describe("KeysFromUint32", func() {
			It("Should correctly --reinterpret-- a slice of uint32 into a slice of keys", func() {
				uint32s := []uint32{1, 2, 3}
				keys := channel.KeysFromUint32(uint32s)
				Expect(keys).To(Equal(channel.Keys{1, 2, 3}))
			})
		})
		Describe("KeysFromOntologyIDs", func() {
			It("Should should correctly parse a list of ontology IDs into a list of keys", func() {
				ids := []ontology.ID{
					{Type: "channel", Key: "1"},
					{Type: "channel", Key: "2"},
				}
				keys, err := channel.KeysFromOntologyIDs(ids)
				Expect(err).ToNot(HaveOccurred())
				Expect(keys).To(Equal(channel.Keys{1, 2}))
			})
			It("Should skip any ontology IDs that are not of the correct type", func() {
				ids := []ontology.ID{
					{Type: "channel", Key: "1"},
					{Type: "not_channel", Key: "2"},
				}
				keys, err := channel.KeysFromOntologyIDs(ids)
				Expect(err).ToNot(HaveOccurred())
				Expect(keys).To(Equal(channel.Keys{1}))
			})
			It("Should return an error if the key cannot be parsed", func() {
				ids := []ontology.ID{
					{Type: "channel", Key: "1"},
					{Type: "channel", Key: "a"},
				}
				_, err := channel.KeysFromOntologyIDs(ids)
				Expect(err).To(HaveOccurredAs(validate.ErrValidation))
				Expect(err.Error()).To(ContainSubstring("a is not a valid channel key"))
			})
		})
		Describe("UniqueLeaseholders", func() {
			It("Should return a slice of the unique node ids for a set of keys", func() {
				ids := channel.Keys{
					channel.NewKey(cluster.NodeKey(1), 2),
					channel.NewKey(cluster.NodeKey(3), 4),
					channel.NewKey(cluster.NodeKey(1), 2),
				}
				Expect(ids.UniqueLeaseholders()).To(Equal([]cluster.NodeKey{1, 3}))
			})
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
		Describe("Strings", func() {
			It("Should return the keys as a slice of strings", func() {
				keys := channel.Keys{1, 2, 3}
				Expect(keys.Strings()).To(Equal([]string{"1", "2", "3"}))
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
		Describe("Difference", func() {
			It("Should return the keys that are absent in other followed by the keys that are absent in k", func() {
				k := channel.Keys{1, 2, 3}
				other := channel.Keys{2, 3, 4}
				added, removed := k.Difference(other)
				Expect(added).To(Equal(channel.Keys{1}))
				Expect(removed).To(Equal(channel.Keys{4}))
			})
		})
	})
	Describe("Equal", func() {
		It("Should return true if the two channels are equal", func() {
			c1 := channel.Channel{Leaseholder: 1, LocalKey: 1}
			c2 := channel.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(c1.Equals(c2)).To(BeTrue())
		})
		DescribeTable("Exclusion", func(c1, c2 channel.Channel, exclude ...string) {
			Expect(c1.Equals(c2, exclude...)).To(BeTrue())
		},
			Entry(
				"Names",
				channel.Channel{Name: "name1", LocalKey: 1},
				channel.Channel{Name: "name2", LocalKey: 1},
				"Name",
			),
			Entry(
				"Leaseholders",
				channel.Channel{Leaseholder: 1, LocalKey: 1},
				channel.Channel{Leaseholder: 2, LocalKey: 1},
				"Leaseholder",
			),
			Entry(
				"LocalKeys",
				channel.Channel{Leaseholder: 1, LocalKey: 1},
				channel.Channel{Leaseholder: 1, LocalKey: 2},
				"LocalKey",
			),
			Entry(
				"Virtual",
				channel.Channel{Leaseholder: 1, LocalKey: 1},
				channel.Channel{Leaseholder: 1, LocalKey: 1, Virtual: true},
				"Virtual",
			),
			Entry(
				"DataType",
				channel.Channel{Leaseholder: 1, LocalKey: 1},
				channel.Channel{Leaseholder: 1, LocalKey: 1, DataType: "int"},
				"DataType",
			),
			Entry(
				"LocalIndex",
				channel.Channel{Leaseholder: 1, LocalKey: 1},
				channel.Channel{Leaseholder: 1, LocalKey: 1, LocalIndex: 1},
				"LocalIndex",
			),
			Entry(
				"Operations",
				channel.Channel{
					Leaseholder: 1,
					LocalKey:    1,
					Operations:  []channel.Operation{{Type: "max"}},
				},
				channel.Channel{
					Leaseholder: 1,
					LocalKey:    1,
					Operations:  []channel.Operation{{Type: "min"}},
				},
				"Operations",
			),
		)
		DescribeTable("Not Equal", func(c1, c2 channel.Channel, exclude ...string) {
			Expect(c1.Equals(c2, exclude...)).To(BeFalse())
		},
			Entry(
				"By LocalIndex",
				channel.Channel{Leaseholder: 1, LocalKey: 1, LocalIndex: 1},
				channel.Channel{Leaseholder: 1, LocalKey: 1, LocalIndex: 2},
			),
			Entry(
				"By Name",
				channel.Channel{Name: "name1", LocalKey: 1},
				channel.Channel{Name: "name2", LocalKey: 1},
			),
			Entry(
				"By Leaseholder",
				channel.Channel{Leaseholder: 1, LocalKey: 1},
				channel.Channel{Leaseholder: 2, LocalKey: 1},
			),
			Entry(
				"By LocalKey",
				channel.Channel{Leaseholder: 1, LocalKey: 1},
				channel.Channel{Leaseholder: 1, LocalKey: 2},
			),
			Entry(
				"By Data Type",
				channel.Channel{Leaseholder: 1, LocalKey: 1, DataType: "int"},
				channel.Channel{Leaseholder: 1, LocalKey: 1, DataType: "float"},
			),
			Entry(
				"By Virtual",
				channel.Channel{Leaseholder: 1, LocalKey: 1, Virtual: true},
				channel.Channel{Leaseholder: 1, LocalKey: 1},
			),
			Entry(
				"By Operations",
				channel.Channel{
					Leaseholder: 1,
					LocalKey:    1,
					Operations:  []channel.Operation{{Type: "max"}},
				},
				channel.Channel{
					Leaseholder: 1,
					LocalKey:    1,
					Operations:  []channel.Operation{{Type: "min"}},
				},
			),
		)
	})
})
