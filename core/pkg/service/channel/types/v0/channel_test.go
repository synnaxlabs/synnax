// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	v0 "github.com/synnaxlabs/synnax/pkg/service/channel/types/v0"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Channel", func() {
	Describe("IsCalculated", func() {
		It("Should return true when the channel has an expression", func() {
			Expect(v0.Channel{Expression: "a + b"}.IsCalculated()).To(BeTrue())
		})
		It("Should return false when the channel has no expression", func() {
			Expect(v0.Channel{}.IsCalculated()).To(BeFalse())
		})
	})

	Describe("Key", func() {
		It("Should compose the key from the leaseholder and local key", func() {
			ch := v0.Channel{Leaseholder: 1, LocalKey: 2}
			Expect(ch.Key()).To(Equal(channel.NewKey(1, 2)))
		})
	})

	Describe("Index", func() {
		It("Should return the key of the channel's index channel", func() {
			ch := v0.Channel{Leaseholder: 1, LocalIndex: 5}
			Expect(ch.Index()).To(Equal(channel.NewKey(1, 5)))
		})
		It("Should return a zero key when the channel has no local index", func() {
			Expect(v0.Channel{Leaseholder: 1}.Index()).To(Equal(v0.Key(0)))
		})
	})

	Describe("GorpKey", func() {
		It("Should return the same value as Key", func() {
			ch := v0.Channel{Leaseholder: 1, LocalKey: 2}
			Expect(ch.GorpKey()).To(Equal(ch.Key()))
		})
	})

	Describe("SetOptions", func() {
		It("Should bootstrap-lease a free channel", func() {
			ch := v0.Channel{Leaseholder: node.KeyFree}
			Expect(ch.SetOptions()).To(Equal([]any{node.KeyBootstrapper}))
		})
		It("Should lease a non-free channel to its leaseholder", func() {
			ch := v0.Channel{Leaseholder: 2}
			Expect(ch.SetOptions()).To(Equal([]any{node.Key(2)}))
		})
	})

	Describe("Free", func() {
		It("Should return true for a channel with no leaseholder", func() {
			Expect(v0.Channel{Leaseholder: node.KeyFree}.Free()).To(BeTrue())
		})
		It("Should return false for a leased channel", func() {
			Expect(v0.Channel{Leaseholder: 1}.Free()).To(BeFalse())
		})
	})

	Describe("Distribution", func() {
		It("Should return the distribution-layer representation of the channel", func() {
			ch := v0.Channel{
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
			Expect(ch.Distribution()).To(Equal(channel.Channel{
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

	Describe("Equals", func() {
		It("Should return true if the two channels are equal", func() {
			c1 := v0.Channel{Leaseholder: 1, LocalKey: 1}
			c2 := v0.Channel{Leaseholder: 1, LocalKey: 1}
			Expect(c1.Equals(c2)).To(BeTrue())
		})
		DescribeTable("Should ignore excluded fields", func(c1, c2 v0.Channel, exclude ...string) {
			Expect(c1.Equals(c2, exclude...)).To(BeTrue())
		},
			Entry(
				"Names",
				v0.Channel{Name: "name1", LocalKey: 1},
				v0.Channel{Name: "name2", LocalKey: 1},
				"Name",
			),
			Entry(
				"Leaseholders",
				v0.Channel{Leaseholder: 1, LocalKey: 1},
				v0.Channel{Leaseholder: 2, LocalKey: 1},
				"Leaseholder",
			),
			Entry(
				"LocalKeys",
				v0.Channel{Leaseholder: 1, LocalKey: 1},
				v0.Channel{Leaseholder: 1, LocalKey: 2},
				"LocalKey",
			),
			Entry(
				"Virtual",
				v0.Channel{Leaseholder: 1, LocalKey: 1},
				v0.Channel{Leaseholder: 1, LocalKey: 1, Virtual: true},
				"Virtual",
			),
			Entry(
				"DataType",
				v0.Channel{Leaseholder: 1, LocalKey: 1},
				v0.Channel{Leaseholder: 1, LocalKey: 1, DataType: "int"},
				"DataType",
			),
			Entry(
				"LocalIndex",
				v0.Channel{Leaseholder: 1, LocalKey: 1},
				v0.Channel{Leaseholder: 1, LocalKey: 1, LocalIndex: 1},
				"LocalIndex",
			),
			Entry(
				"Operations",
				v0.Channel{
					Leaseholder: 1,
					LocalKey:    1,
					Operations:  []v0.Operation{{Type: "max"}},
				},
				v0.Channel{
					Leaseholder: 1,
					LocalKey:    1,
					Operations:  []v0.Operation{{Type: "min"}},
				},
				"Operations",
			),
		)
		DescribeTable("Should report inequality", func(c1, c2 v0.Channel, exclude ...string) {
			Expect(c1.Equals(c2, exclude...)).To(BeFalse())
		},
			Entry(
				"By LocalIndex",
				v0.Channel{Leaseholder: 1, LocalKey: 1, LocalIndex: 1},
				v0.Channel{Leaseholder: 1, LocalKey: 1, LocalIndex: 2},
			),
			Entry(
				"By Name",
				v0.Channel{Name: "name1", LocalKey: 1},
				v0.Channel{Name: "name2", LocalKey: 1},
			),
			Entry(
				"By Leaseholder",
				v0.Channel{Leaseholder: 1, LocalKey: 1},
				v0.Channel{Leaseholder: 2, LocalKey: 1},
			),
			Entry(
				"By LocalKey",
				v0.Channel{Leaseholder: 1, LocalKey: 1},
				v0.Channel{Leaseholder: 1, LocalKey: 2},
			),
			Entry(
				"By Data Type",
				v0.Channel{Leaseholder: 1, LocalKey: 1, DataType: "int"},
				v0.Channel{Leaseholder: 1, LocalKey: 1, DataType: "float"},
			),
			Entry(
				"By Virtual",
				v0.Channel{Leaseholder: 1, LocalKey: 1, Virtual: true},
				v0.Channel{Leaseholder: 1, LocalKey: 1},
			),
			Entry(
				"By Operations",
				v0.Channel{
					Leaseholder: 1,
					LocalKey:    1,
					Operations:  []v0.Operation{{Type: "max"}},
				},
				v0.Channel{
					Leaseholder: 1,
					LocalKey:    1,
					Operations:  []v0.Operation{{Type: "min"}},
				},
			),
		)
	})

	Describe("String", func() {
		DescribeTable("Should format the channel as a string",
			func(ch v0.Channel, expected string) {
				Expect(ch.String()).To(Equal(expected))
			},
			Entry("named channel",
				v0.Channel{Name: "temp", Leaseholder: 1, LocalKey: 2},
				"[temp]<1048578>",
			),
			Entry("unnamed channel",
				v0.Channel{Leaseholder: 1, LocalKey: 2},
				"<1048578>",
			),
			Entry("zero-value channel",
				v0.Channel{},
				"<0>",
			),
		)
	})

	Describe("UnmarshalJSON", func() {
		It("Should decode a channel from JSON", func() {
			var c v0.Channel
			Expect(c.UnmarshalJSON(
				[]byte(`{"name":"test","leaseholder":3,"local_key":2}`),
			)).To(Succeed())
			Expect(c.Name).To(Equal("test"))
			Expect(c.Leaseholder).To(Equal(node.Key(3)))
		})
		It("Should fall back to node_id when leaseholder is absent", func() {
			var c v0.Channel
			Expect(c.UnmarshalJSON(
				[]byte(`{"name":"test","node_id":7,"local_key":2}`),
			)).To(Succeed())
			Expect(c.Leaseholder).To(Equal(node.Key(7)))
		})
		It("Should return an error when the JSON is malformed", func() {
			var c v0.Channel
			Expect(c.UnmarshalJSON([]byte(`{not json`))).To(
				MatchError(ContainSubstring("failed to decode channel from JSON")),
			)
		})
		It("Should return an error when the legacy node_id field is malformed", func() {
			var c v0.Channel
			Expect(c.UnmarshalJSON([]byte(`{"node_id":"abc"}`))).To(
				MatchError(ContainSubstring("failed to decode legacy node_id from JSON")),
			)
		})
	})

	Describe("DecodeMsgpack", func() {
		It("Should decode a channel from msgpack", func() {
			data := MustSucceed(msgpack.Marshal(map[string]any{
				"name":        "test",
				"leaseholder": 3,
			}))
			var c v0.Channel
			Expect(msgpack.Unmarshal(data, &c)).To(Succeed())
			Expect(c.Name).To(Equal("test"))
			Expect(c.Leaseholder).To(Equal(node.Key(3)))
		})
		It("Should fall back to node_id when leaseholder is absent", func() {
			data := MustSucceed(msgpack.Marshal(map[string]any{
				"name":    "test",
				"node_id": 7,
			}))
			var c v0.Channel
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
			var c v0.Channel
			Expect(msgpack.Unmarshal(data, &c)).To(Succeed())
			Expect(c.Operations).To(HaveLen(1))
			Expect(c.Operations[0].Type).To(Equal(v0.OperationTypeAvg))
		})
		It("Should return an error when the msgpack bytes are invalid", func() {
			var c v0.Channel
			Expect(msgpack.Unmarshal([]byte{0xc1}, &c)).To(
				MatchError(ContainSubstring("failed to read raw channel msgpack")),
			)
		})
		It("Should return an error when the value is not a channel", func() {
			data := MustSucceed(msgpack.Marshal("not a channel"))
			var c v0.Channel
			Expect(msgpack.Unmarshal(data, &c)).To(
				MatchError(ContainSubstring("failed to decode channel from msgpack")),
			)
		})
		It("Should return an error when the legacy node_id field is malformed", func() {
			data := MustSucceed(msgpack.Marshal(map[string]any{"node_id": "abc"}))
			var c v0.Channel
			Expect(msgpack.Unmarshal(data, &c)).To(
				MatchError(ContainSubstring("failed to decode legacy node_id from msgpack")),
			)
		})
	})

	Describe("OntologyID", func() {
		It("Should return the channel ontology identifier", func() {
			ch := v0.Channel{Leaseholder: 1, LocalKey: 2}
			Expect(ch.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeChannel, Key: ch.Key().String(),
			}))
		})
	})
})

var _ = Describe("Operation", func() {
	Describe("DecodeMsgpack", func() {
		It("Should decode an operation with new lowercase msgpack fields", func() {
			original := v0.Operation{
				Type:         v0.OperationTypeAvg,
				ResetChannel: 42,
				Duration:     5000000000,
			}
			data := MustSucceed(msgpack.Marshal(original))
			var decoded v0.Operation
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Type).To(Equal(v0.OperationTypeAvg))
			Expect(decoded.ResetChannel).To(Equal(v0.Key(42)))
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
			var decoded v0.Operation
			Expect(msgpack.Unmarshal(data, &decoded)).To(Succeed())
			Expect(decoded.Type).To(Equal(v0.OperationTypeMax))
			Expect(decoded.ResetChannel).To(Equal(v0.Key(10)))
			Expect(decoded.Duration).To(Equal(telem.TimeSpan(1000000000)))
		})
		It("Should return an error when the msgpack bytes are invalid", func() {
			var o v0.Operation
			Expect(msgpack.Unmarshal([]byte{0xc1}, &o)).To(
				MatchError(ContainSubstring("failed to read raw operation msgpack")),
			)
		})
		It("Should return an error when the value is not an operation", func() {
			data := MustSucceed(msgpack.Marshal("not an operation"))
			var o v0.Operation
			Expect(msgpack.Unmarshal(data, &o)).To(
				MatchError(ContainSubstring("failed to decode operation from msgpack")),
			)
		})
		It("Should return an error when a legacy operation field is malformed", func() {
			data := MustSucceed(msgpack.Marshal(map[string]any{"ResetChannel": "abc"}))
			var o v0.Operation
			Expect(msgpack.Unmarshal(data, &o)).To(
				MatchError(ContainSubstring("failed to decode legacy operation from msgpack")),
			)
		})
	})
})
