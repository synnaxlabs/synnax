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
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("ParseKey", func() {
	It("Should correctly parse a key from its string representation", func() {
		Expect(MustSucceed(channel.ParseKey("123456"))).To(Equal(channel.Key(123456)))
	})
	It("Should return an error when the key is not a valid integer", func() {
		Expect(channel.ParseKey("123456a")).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("123456a is not a valid channel key")),
		))
	})
})

var _ = Describe("NewKey", func() {
	It("Should compose a key from a leaseholder and a local key", func() {
		k := channel.NewKey(1, 2)
		Expect(k.Lease()).To(Equal(node.Key(1)))
		Expect(k.LocalKey()).To(Equal(channel.LocalKey(2)))
	})
})

var _ = Describe("KeysFromUint32", func() {
	It("Should reinterpret a slice of uint32 as Keys", func() {
		Expect(channel.KeysFromUint32([]uint32{1, 2, 3})).To(Equal(channel.Keys{1, 2, 3}))
	})
})

var _ = Describe("KeysFromChannels", func() {
	It("Should return the keys of the given channels", func() {
		channels := []channel.Channel{
			{Leaseholder: 1, LocalKey: 1},
			{Leaseholder: 2, LocalKey: 3},
		}
		Expect(channel.KeysFromChannels(channels)).To(Equal(channel.Keys{
			channel.NewKey(1, 1),
			channel.NewKey(2, 3),
		}))
	})
	It("Should return an empty slice when given no channels", func() {
		Expect(channel.KeysFromChannels(nil)).To(BeEmpty())
	})
})

var _ = Describe("Names", func() {
	It("Should return the names of the given channels", func() {
		channels := []channel.Channel{{Name: "alpha"}, {Name: "beta"}}
		Expect(channel.Names(channels)).To(Equal([]string{"alpha", "beta"}))
	})
})

var _ = Describe("NewRandomName", func() {
	It("Should generate a name with the test channel prefix", func() {
		Expect(channel.NewRandomName()).To(HavePrefix("test_ch_"))
	})
	It("Should generate distinct names across calls", func() {
		Expect(channel.NewRandomName()).ToNot(Equal(channel.NewRandomName()))
	})
})

var _ = Describe("IsCalculated", func() {
	It("Should return true when the channel has an expression", func() {
		Expect(channel.Channel{Expression: "a + b"}.IsCalculated()).To(BeTrue())
	})
	It("Should return false when the channel has no expression", func() {
		Expect(channel.Channel{}.IsCalculated()).To(BeFalse())
	})
})

var _ = Describe("Key", func() {
	It("Should compose the key from the leaseholder and local key", func() {
		ch := channel.Channel{Leaseholder: 1, LocalKey: 2}
		Expect(ch.Key()).To(Equal(channel.NewKey(1, 2)))
	})
})

var _ = Describe("Index", func() {
	It("Should return the key of the channel's index channel", func() {
		ch := channel.Channel{Leaseholder: 1, LocalIndex: 5}
		Expect(ch.Index()).To(Equal(channel.NewKey(1, 5)))
	})
	It("Should return a zero key when the channel has no local index", func() {
		Expect(channel.Channel{Leaseholder: 1}.Index()).To(Equal(channel.Key(0)))
	})
})

var _ = Describe("GorpKey", func() {
	It("Should return the same value as Key", func() {
		ch := channel.Channel{Leaseholder: 1, LocalKey: 2}
		Expect(ch.GorpKey()).To(Equal(ch.Key()))
	})
})

var _ = Describe("SetOptions", func() {
	It("Should bootstrap-lease a free channel", func() {
		ch := channel.Channel{Leaseholder: node.KeyFree}
		Expect(ch.SetOptions()).To(Equal([]any{node.KeyBootstrapper}))
	})
	It("Should lease a non-free channel to its leaseholder", func() {
		ch := channel.Channel{Leaseholder: 2}
		Expect(ch.SetOptions()).To(Equal([]any{node.Key(2)}))
	})
})

var _ = Describe("Free", func() {
	It("Should return true for a channel with no leaseholder", func() {
		Expect(channel.Channel{Leaseholder: node.KeyFree}.Free()).To(BeTrue())
	})
	It("Should return false for a leased channel", func() {
		Expect(channel.Channel{Leaseholder: 1}.Free()).To(BeFalse())
	})
})

var _ = Describe("Distribution", func() {
	It("Should return the distribution-layer representation of the channel", func() {
		ch := channel.Channel{
			Name:        "temp",
			Leaseholder: 1,
			DataType:    telem.Float32T,
			IsIndex:     true,
			LocalKey:    2,
			LocalIndex:  3,
			Virtual:     true,
			Concurrency: control.ConcurrencyShared,
			Internal:    true,
			Expression:  "a + b",
		}
		Expect(ch.Distribution()).To(Equal(dcore.Channel{
			Name:        "temp",
			Leaseholder: 1,
			DataType:    telem.Float32T,
			IsIndex:     true,
			LocalKey:    2,
			LocalIndex:  3,
			Virtual:     true,
			Concurrency: control.ConcurrencyShared,
		}))
	})
})

var _ = Describe("Equals", func() {
	It("Should return true if the two channels are equal", func() {
		c1 := channel.Channel{Leaseholder: 1, LocalKey: 1}
		c2 := channel.Channel{Leaseholder: 1, LocalKey: 1}
		Expect(c1.Equals(c2)).To(BeTrue())
	})
	DescribeTable("Should ignore excluded fields", func(c1, c2 channel.Channel, exclude ...string) {
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
	DescribeTable("Should report inequality", func(c1, c2 channel.Channel, exclude ...string) {
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

var _ = Describe("String", func() {
	DescribeTable("Should format the channel as a string",
		func(ch channel.Channel, expected string) {
			Expect(ch.String()).To(Equal(expected))
		},
		Entry("named channel",
			channel.Channel{Name: "temp", Leaseholder: 1, LocalKey: 2},
			"[temp]<1048578>",
		),
		Entry("unnamed channel",
			channel.Channel{Leaseholder: 1, LocalKey: 2},
			"<1048578>",
		),
		Entry("zero-value channel",
			channel.Channel{},
			"<0>",
		),
	)
})

var _ = Describe("UnmarshalJSON", func() {
	It("Should decode a channel from JSON", func() {
		var c channel.Channel
		Expect(c.UnmarshalJSON(
			[]byte(`{"name":"test","leaseholder":3,"local_key":2}`),
		)).To(Succeed())
		Expect(c.Name).To(Equal("test"))
		Expect(c.Leaseholder).To(Equal(node.Key(3)))
	})
	It("Should fall back to node_id when leaseholder is absent", func() {
		var c channel.Channel
		Expect(c.UnmarshalJSON(
			[]byte(`{"name":"test","node_id":7,"local_key":2}`),
		)).To(Succeed())
		Expect(c.Leaseholder).To(Equal(node.Key(7)))
	})
	It("Should return an error when the JSON is malformed", func() {
		var c channel.Channel
		Expect(c.UnmarshalJSON([]byte(`{not json`))).To(
			MatchError(ContainSubstring("failed to decode channel from JSON")),
		)
	})
	It("Should return an error when the legacy node_id field is malformed", func() {
		var c channel.Channel
		Expect(c.UnmarshalJSON([]byte(`{"node_id":"abc"}`))).To(
			MatchError(ContainSubstring("failed to decode legacy node_id from JSON")),
		)
	})
})

var _ = Describe("Channel DecodeMsgpack", func() {
	It("Should decode a channel from msgpack", func() {
		data := MustSucceed(msgpack.Marshal(map[string]any{
			"name":        "test",
			"leaseholder": 3,
		}))
		var c channel.Channel
		Expect(msgpack.Unmarshal(data, &c)).To(Succeed())
		Expect(c.Name).To(Equal("test"))
		Expect(c.Leaseholder).To(Equal(node.Key(3)))
	})
	It("Should fall back to node_id when leaseholder is absent", func() {
		data := MustSucceed(msgpack.Marshal(map[string]any{
			"name":    "test",
			"node_id": 7,
		}))
		var c channel.Channel
		Expect(msgpack.Unmarshal(data, &c)).To(Succeed())
		Expect(c.Leaseholder).To(Equal(node.Key(7)))
	})
	It("Should decode a channel with legacy operations embedded", func() {
		data := MustSucceed(msgpack.Marshal(map[string]any{
			"name":        "fuel_tc_avg",
			"leaseholder": 1,
			"local_key":   5,
			"operations": []map[string]any{
				{"Type": "avg", "ResetChannel": 0, "Duration": 0},
			},
		}))
		var c channel.Channel
		Expect(msgpack.Unmarshal(data, &c)).To(Succeed())
		Expect(c.Operations).To(HaveLen(1))
		Expect(c.Operations[0].Type).To(Equal(channel.OperationTypeAvg))
	})
	It("Should return an error when the msgpack bytes are invalid", func() {
		var c channel.Channel
		Expect(msgpack.Unmarshal([]byte{0xc1}, &c)).To(
			MatchError(ContainSubstring("failed to read raw channel msgpack")),
		)
	})
	It("Should return an error when the value is not a channel", func() {
		data := MustSucceed(msgpack.Marshal("not a channel"))
		var c channel.Channel
		Expect(msgpack.Unmarshal(data, &c)).To(
			MatchError(ContainSubstring("failed to decode channel from msgpack")),
		)
	})
	It("Should return an error when the legacy node_id field is malformed", func() {
		data := MustSucceed(msgpack.Marshal(map[string]any{"node_id": "abc"}))
		var c channel.Channel
		Expect(msgpack.Unmarshal(data, &c)).To(
			MatchError(ContainSubstring("failed to decode legacy node_id from msgpack")),
		)
	})
})

var _ = Describe("Operation DecodeMsgpack", func() {
	It("Should decode an operation with new lowercase msgpack fields", func() {
		original := channel.Operation{
			Type:         channel.OperationTypeAvg,
			ResetChannel: 42,
			Duration:     5000000000,
		}
		data := MustSucceed(msgpack.Marshal(original))
		var decoded channel.Operation
		Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded.Type).To(Equal(channel.OperationTypeAvg))
		Expect(decoded.ResetChannel).To(Equal(channel.Key(42)))
		Expect(decoded.Duration).To(Equal(telem.TimeSpan(5000000000)))
	})
	It("Should decode legacy uppercase Go field names", func() {
		legacy := struct {
			Type         string
			ResetChannel uint32
			Duration     int64
		}{
			Type:         "max",
			ResetChannel: 10,
			Duration:     1000000000,
		}
		data := MustSucceed(msgpack.Marshal(legacy))
		var decoded channel.Operation
		Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
		Expect(decoded.Type).To(Equal(channel.OperationTypeMax))
		Expect(decoded.ResetChannel).To(Equal(channel.Key(10)))
		Expect(decoded.Duration).To(Equal(telem.TimeSpan(1000000000)))
	})
	It("Should return an error when the msgpack bytes are invalid", func() {
		var o channel.Operation
		Expect(msgpack.Unmarshal([]byte{0xc1}, &o)).To(
			MatchError(ContainSubstring("failed to read raw operation msgpack")),
		)
	})
	It("Should return an error when the value is not an operation", func() {
		data := MustSucceed(msgpack.Marshal("not an operation"))
		var o channel.Operation
		Expect(msgpack.Unmarshal(data, &o)).To(
			MatchError(ContainSubstring("failed to decode operation from msgpack")),
		)
	})
	It("Should return an error when a legacy operation field is malformed", func() {
		data := MustSucceed(msgpack.Marshal(map[string]any{"ResetChannel": "abc"}))
		var o channel.Operation
		Expect(msgpack.Unmarshal(data, &o)).To(
			MatchError(ContainSubstring("failed to decode legacy operation from msgpack")),
		)
	})
})
