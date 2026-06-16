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
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/vmihailenco/msgpack/v5"
)

var _ = Describe("Channel Decoding", func() {
	Describe("Operation DecodeMsgpack", func() {
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
		It("Should decode a channel with legacy operations embedded", func() {
			legacy := map[string]any{
				"name":        "fuel_tc_avg",
				"leaseholder": 1,
				"local_key":   5,
				"operations": []map[string]any{
					{
						"Type":         "avg",
						"ResetChannel": 0,
						"Duration":     0,
					},
				},
			}
			data := MustSucceed(msgpack.Marshal(legacy))
			var c channel.Channel
			Expect(msgpack.Unmarshal(data, &c)).To(Succeed())
			Expect(c.Operations).To(HaveLen(1))
			Expect(c.Operations[0].Type).To(Equal(channel.OperationTypeAvg))
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
	Describe("UnmarshalJSON / DecodeMsgpack node_id fallback", func() {
		It("Should fall back to node_id when leaseholder is absent (JSON)", func() {
			data := []byte(`{"name":"test","node_id":7,"local_key":2}`)
			var c channel.Channel
			Expect(c.UnmarshalJSON(data)).To(Succeed())
			Expect(c.Leaseholder).To(Equal(node.Key(7)))
		})
		It("Should fall back to node_id when leaseholder is absent (msgpack)", func() {
			data := MustSucceed(msgpack.Marshal(map[string]any{
				"name":    "test",
				"node_id": 7,
			}))
			var c channel.Channel
			Expect(msgpack.Unmarshal(data, &c)).To(Succeed())
			Expect(c.Leaseholder).To(Equal(node.Key(7)))
		})
	})
})
