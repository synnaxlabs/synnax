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
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/math"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Channel Tests", func() {
	Describe("Key Tests", func() {
		Describe("Construction", func() {
			It("Should return the correct leaseholder for the key", func() {
				k := channel.NewKey(node.Key(1), 1)
				Expect(k.Leaseholder()).To(Equal(node.Key(1)))
			})
			It("Should return the correct localKey for the key", func() {
				k := channel.NewKey(node.Key(1), 2)
				Expect(k.LocalKey()).To(Equal(channel.LocalKey(2)))
			})
			It("Should correctly handle the maximum value of a 12 bit node key and 20 bit cesium key", func() {
				k := channel.NewKey(node.Key(math.MaxUint12), channel.LocalKey(math.MaxUint20))
				Expect(k.Leaseholder()).To(Equal(node.Key(math.MaxUint12)))
				Expect(k.LocalKey()).To(Equal(channel.LocalKey(math.MaxUint20)))
			})
		})
		Describe("Lease", func() {
			It("Should return the leaseholder node Name", func() {
				k := channel.NewKey(node.Key(1), 1)
				Expect(k.Lease()).To(Equal(k.Leaseholder()))
			})
		})
		Describe("Free", func() {
			It("Should return true if the channel is a free channel", func() {
				k := channel.NewKey(node.KeyFree, 1)
				Expect(k.Free()).To(BeTrue())
			})
			It("Should return false if the channel is not a free channel", func() {
				k := channel.NewKey(node.Key(1), 1)
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
					channel.NewKey(node.Key(1), 1),
					channel.NewKey(node.Key(1), 2),
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
	Describe("UnmarshalJSON", func() {
		It("Should unmarshal a channel with the leaseholder field", func() {
			data := []byte(`{"name":"test","leaseholder":5,"data_type":"float32","local_key":1}`)
			var c channel.Channel
			Expect(json.Unmarshal(data, &c)).To(Succeed())
			Expect(c.Name).To(Equal("test"))
			Expect(c.Leaseholder).To(Equal(node.Key(5)))
			Expect(c.LocalKey).To(Equal(channel.LocalKey(1)))
		})
		It("Should return an error for invalid JSON", func() {
			Expect(json.Unmarshal([]byte(`not json`), &channel.Channel{})).To(HaveOccurred())
		})
	})
	Describe("DecodeMsgpack", func() {
		It("Should decode a channel with the leaseholder field", func() {
			original := channel.Channel{
				Name:        "test",
				Leaseholder: 5,
				LocalKey:    1,
				DataType:    "float32",
			}
			data := MustSucceed(msgpack.Marshal(original))
			var c channel.Channel
			Expect(msgpack.Unmarshal(data, &c)).To(Succeed())
			Expect(c.Name).To(Equal("test"))
			Expect(c.Leaseholder).To(Equal(node.Key(5)))
			Expect(c.LocalKey).To(Equal(channel.LocalKey(1)))
		})
		It("Should return an error for invalid msgpack", func() {
			Expect(msgpack.Unmarshal([]byte{0xff}, &channel.Channel{})).To(HaveOccurred())
		})
	})
})
