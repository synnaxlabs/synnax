// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.
package sparkplug_test

import (
	"math"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug/pb"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"google.golang.org/protobuf/proto"
)

// payloadMillis is the timestamp of every payload that the helpers build.
const payloadMillis uint64 = 1700000000000

const payloadTime = telem.TimeStamp(payloadMillis) * telem.MillisecondTS

func intValue(v uint32) *pb.Payload_Metric {
	return &pb.Payload_Metric{Value: &pb.Payload_Metric_IntValue{IntValue: v}}
}

func longValue(v uint64) *pb.Payload_Metric {
	return &pb.Payload_Metric{Value: &pb.Payload_Metric_LongValue{LongValue: v}}
}

func floatValue(v float32) *pb.Payload_Metric {
	return &pb.Payload_Metric{Value: &pb.Payload_Metric_FloatValue{FloatValue: v}}
}

func doubleValue(v float64) *pb.Payload_Metric {
	return &pb.Payload_Metric{Value: &pb.Payload_Metric_DoubleValue{DoubleValue: v}}
}

func boolValue(v bool) *pb.Payload_Metric {
	return &pb.Payload_Metric{Value: &pb.Payload_Metric_BooleanValue{BooleanValue: v}}
}

func stringValue(v string) *pb.Payload_Metric {
	return &pb.Payload_Metric{Value: &pb.Payload_Metric_StringValue{StringValue: v}}
}

func nullValue() *pb.Payload_Metric { return &pb.Payload_Metric{IsNull: new(true)} }

func dataSetValue() *pb.Payload_Metric {
	return &pb.Payload_Metric{
		Value: &pb.Payload_Metric_DatasetValue{DatasetValue: &pb.Payload_DataSet{}},
	}
}

// declared returns a copy of m with what a birth message declares about a tag.
func declared(
	name string,
	alias uint64,
	dt sparkplug.DataType,
	m *pb.Payload_Metric,
) *pb.Payload_Metric {
	m = proto.Clone(m).(*pb.Payload_Metric)
	m.Name, m.Alias, m.Datatype = new(name), new(alias), new(uint32(dt))
	return m
}

// byAlias returns a copy of m that names its tag with alias only.
func byAlias(alias uint64, m *pb.Payload_Metric) *pb.Payload_Metric {
	m = proto.Clone(m).(*pb.Payload_Metric)
	m.Alias = new(alias)
	return m
}

// byName returns a copy of m that names its tag with name only.
func byName(name string, m *pb.Payload_Metric) *pb.Payload_Metric {
	m = proto.Clone(m).(*pb.Payload_Metric)
	m.Name = new(name)
	return m
}

func encode(p *pb.Payload) []byte {
	GinkgoHelper()
	return MustSucceed(proto.Marshal(p))
}

func bdSeqMetric(bdSeq uint64) *pb.Payload_Metric {
	return declared(sparkplug.BdSeqMetric, 0, sparkplug.UInt64, longValue(bdSeq))
}

// birth returns an NBIRTH payload. Its first metric is bdSeq.
func birth(seq, bdSeq uint64, metrics ...*pb.Payload_Metric) []byte {
	GinkgoHelper()
	return encode(&pb.Payload{
		Timestamp: new(payloadMillis),
		Seq:       new(seq),
		Metrics:   append([]*pb.Payload_Metric{bdSeqMetric(bdSeq)}, metrics...),
	})
}

// data returns the payload of an NDATA, DBIRTH, DDATA, or DDEATH message.
func data(seq uint64, metrics ...*pb.Payload_Metric) []byte {
	GinkgoHelper()
	return encode(&pb.Payload{
		Timestamp: new(payloadMillis),
		Seq:       new(seq),
		Metrics:   metrics,
	})
}

// death returns an NDEATH payload, which has no sequence number.
func death(bdSeq uint64) []byte {
	GinkgoHelper()
	return encode(&pb.Payload{
		Timestamp: new(payloadMillis),
		Metrics:   []*pb.Payload_Metric{bdSeqMetric(bdSeq)},
	})
}

var _ = Describe("Host", func() {
	var (
		node  = sparkplug.NodeID{Group: "plant", EdgeNode: "line1"}
		other = sparkplug.NodeID{Group: "plant", EdgeNode: "line2"}
		h     *sparkplug.Host
	)
	nodeTopic := func(t sparkplug.MessageType) sparkplug.Topic {
		return sparkplug.Topic{Type: t, Node: node}
	}
	deviceTopic := func(t sparkplug.MessageType, device string) sparkplug.Topic {
		return sparkplug.Topic{Type: t, Node: node, Device: device}
	}
	BeforeEach(func() { h = sparkplug.NewHost() })

	Describe("NewHost", func() {
		It("Should know no edge node", func() {
			Expect(h.Born(node)).To(BeFalse())
		})
		It("Should use the default rebirth interval", func() {
			Expect(sparkplug.DefaultRebirthInterval).To(Equal(5 * time.Second))
			Expect(h.RebirthInterval).To(Equal(sparkplug.DefaultRebirthInterval))
		})
	})

	Describe("Handle", func() {
		Describe("Values", func() {
			DescribeTable(
				"Should decode a value of each data type",
				func(
					dt sparkplug.DataType,
					value *pb.Payload_Metric,
					expected types.GomegaMatcher,
				) {
					born := MustSucceed(h.Handle(
						nodeTopic(sparkplug.NBirth),
						birth(0, 0, declared("tag", 1, dt, value)),
					))
					Expect(born.Metrics).To(HaveLen(2))
					Expect(born.Metrics[1].Name).To(Equal("tag"))
					Expect(born.Metrics[1].DataType).To(Equal(dt))
					Expect(born.Metrics[1].Value).To(expected)
					ev := MustSucceed(h.Handle(
						nodeTopic(sparkplug.NData),
						data(1, byAlias(1, value)),
					))
					Expect(ev.Rebirth).To(BeFalse())
					Expect(ev.Metrics).To(HaveLen(1))
					Expect(ev.Metrics[0].Name).To(Equal("tag"))
					Expect(ev.Metrics[0].DataType).To(Equal(dt))
					Expect(ev.Metrics[0].Value).To(expected)
				},
				Entry("Int8", sparkplug.Int8, intValue(127), Equal(int64(127))),
				Entry("a negative Int8 in 32 bits",
					sparkplug.Int8, intValue(0xFFFFFFFB), Equal(int64(-5)),
				),
				Entry("a negative Int8 in 8 bits",
					sparkplug.Int8, intValue(0xFB), Equal(int64(-5)),
				),
				Entry("Int16", sparkplug.Int16, intValue(32767), Equal(int64(32767))),
				Entry("a negative Int16 in 32 bits",
					sparkplug.Int16, intValue(0xFFFF8000), Equal(int64(-32768)),
				),
				Entry("a negative Int16 in 16 bits",
					sparkplug.Int16, intValue(0x8000), Equal(int64(-32768)),
				),
				Entry(
					"Int32",
					sparkplug.Int32,
					intValue(math.MaxInt32),
					Equal(int64(math.MaxInt32)),
				),
				Entry("a negative Int32",
					sparkplug.Int32, intValue(0x80000000), Equal(int64(math.MinInt32)),
				),
				Entry("an Int32 in long_value",
					sparkplug.Int32, longValue(0xFFFFFFFF), Equal(int64(-1)),
				),
				Entry(
					"Int64",
					sparkplug.Int64,
					longValue(math.MaxInt64),
					Equal(int64(math.MaxInt64)),
				),
				Entry("a negative Int64",
					sparkplug.Int64, longValue(0xFFFFFFFFFFFFFFFB), Equal(int64(-5)),
				),
				Entry("an Int64 in int_value",
					sparkplug.Int64, intValue(12), Equal(int64(12)),
				),
				Entry("UInt8", sparkplug.UInt8, intValue(255), Equal(uint64(255))),
				Entry(
					"UInt16",
					sparkplug.UInt16,
					intValue(65535),
					Equal(uint64(65535)),
				),
				Entry("UInt32",
					sparkplug.UInt32,
					intValue(math.MaxUint32),
					Equal(uint64(math.MaxUint32)),
				),
				Entry("a UInt32 in long_value",
					sparkplug.UInt32, longValue(12), Equal(uint64(12)),
				),
				Entry("UInt64",
					sparkplug.UInt64,
					longValue(math.MaxUint64),
					Equal(uint64(math.MaxUint64)),
				),
				Entry("a UInt64 in int_value",
					sparkplug.UInt64, intValue(12), Equal(uint64(12)),
				),
				Entry("Float", sparkplug.Float, floatValue(1.5), Equal(float64(1.5))),
				Entry("Double", sparkplug.Double, doubleValue(2.25), Equal(2.25)),
				Entry("Boolean true", sparkplug.Boolean, boolValue(true), Equal(true)),
				Entry(
					"Boolean false",
					sparkplug.Boolean,
					boolValue(false),
					Equal(false),
				),
				Entry("String", sparkplug.String, stringValue("open"), Equal("open")),
				Entry("Text", sparkplug.Text, stringValue("a note"), Equal("a note")),
				Entry("UUID",
					sparkplug.UUID,
					stringValue("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
					Equal("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
				),
				Entry("DateTime",
					sparkplug.DateTime,
					longValue(1700000000456),
					Equal(1700000000456*telem.MillisecondTS),
				),
				Entry("a null Double", sparkplug.Double, nullValue(), BeNil()),
				Entry("a null String", sparkplug.String, nullValue(), BeNil()),
				Entry("a metric with no value", sparkplug.Int32,
					&pb.Payload_Metric{}, BeNil(),
				),
				Entry("a data set",
					sparkplug.DataType(pb.DataType_DataSet), dataSetValue(), BeNil(),
				),
				Entry("bytes",
					sparkplug.DataType(pb.DataType_Bytes),
					&pb.Payload_Metric{
						Value: &pb.Payload_Metric_BytesValue{BytesValue: []byte{1}},
					},
					BeNil(),
				),
				Entry("a string value for Int32",
					sparkplug.Int32, stringValue("12"), BeNil(),
				),
				Entry("an integer value for Double",
					sparkplug.Double, intValue(2), BeNil(),
				),
				Entry("a double value for Float",
					sparkplug.Float, doubleValue(1.5), BeNil(),
				),
				Entry("an integer value for Boolean",
					sparkplug.Boolean, intValue(1), BeNil(),
				),
				Entry("an integer value for String",
					sparkplug.String, intValue(1), BeNil(),
				),
			)

			It("Should use the data type of a data message that carries one", func() {
				MustSucceed(h.Handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, declared("tag", 1, sparkplug.Int8, intValue(1))),
				))
				m := byAlias(1, intValue(0xFFFF))
				m.Datatype = new(uint32(sparkplug.UInt16))
				ev := MustSucceed(h.Handle(nodeTopic(sparkplug.NData), data(1, m)))
				Expect(ev.Metrics).To(Equal([]sparkplug.Metric{{
					Name:      "tag",
					DataType:  sparkplug.UInt16,
					Value:     uint64(0xFFFF),
					Timestamp: payloadTime,
				}}))
			})
		})

		Describe("Timestamps", func() {
			BeforeEach(func() {
				MustSucceed(h.Handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, declared("tag", 1, sparkplug.Double, doubleValue(0))),
				))
			})
			It("Should use the timestamp of the metric", func() {
				m := byAlias(1, doubleValue(1))
				m.Timestamp = new(payloadMillis + 250)
				ev := MustSucceed(h.Handle(nodeTopic(sparkplug.NData), data(1, m)))
				Expect(ev.Metrics).To(HaveLen(1))
				Expect(ev.Metrics[0].Timestamp).To(
					Equal(payloadTime + 250*telem.MillisecondTS),
				)
			})
			It("Should fall back to the timestamp of the payload", func() {
				ev := MustSucceed(h.Handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				))
				Expect(ev.Metrics).To(HaveLen(1))
				Expect(ev.Metrics[0].Timestamp).To(Equal(payloadTime))
			})
			It("Should give zero when the message has no timestamp", func() {
				ev := MustSucceed(h.Handle(
					nodeTopic(sparkplug.NData),
					encode(&pb.Payload{
						Seq:     new(uint64(1)),
						Metrics: []*pb.Payload_Metric{byAlias(1, doubleValue(1))},
					}),
				))
				Expect(ev.Metrics).To(HaveLen(1))
				Expect(ev.Metrics[0].Timestamp).To(BeZero())
			})
			It("Should use the timestamps of a birth the same way", func() {
				stamped := declared("stamped", 2, sparkplug.Double, doubleValue(1))
				stamped.Timestamp = new(payloadMillis - 1000)
				ev := MustSucceed(h.Handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, stamped),
				))
				Expect(ev.Metrics).To(HaveLen(2))
				Expect(ev.Metrics[0].Timestamp).To(Equal(payloadTime))
				Expect(ev.Metrics[1].Timestamp).To(
					Equal(payloadTime - 1000*telem.MillisecondTS),
				)
			})
		})

		Describe("Historical", func() {
			It("Should carry the historical flag of a data metric", func() {
				MustSucceed(h.Handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, declared("tag", 1, sparkplug.Double, doubleValue(0))),
				))
				old := byAlias(1, doubleValue(1))
				old.IsHistorical = new(true)
				ev := MustSucceed(h.Handle(
					nodeTopic(sparkplug.NData),
					data(1, old, byAlias(1, doubleValue(2))),
				))
				Expect(ev.Metrics).To(HaveLen(2))
				Expect(ev.Metrics[0].Historical).To(BeTrue())
				Expect(ev.Metrics[1].Historical).To(BeFalse())
			})
			It("Should carry the historical flag of a birth metric", func() {
				old := declared("tag", 1, sparkplug.Double, doubleValue(0))
				old.IsHistorical = new(true)
				ev := MustSucceed(
					h.Handle(nodeTopic(sparkplug.NBirth), birth(0, 0, old)),
				)
				Expect(ev.Metrics).To(HaveLen(2))
				Expect(ev.Metrics[0].Historical).To(BeFalse())
				Expect(ev.Metrics[1].Historical).To(BeTrue())
			})
		})

		Describe("NBIRTH", func() {
			It(
				"Should return every metric of the birth and mark the node born",
				func() {
					ev := MustSucceed(h.Handle(
						nodeTopic(sparkplug.NBirth),
						birth(0, 7,
							declared("temp", 1, sparkplug.Double, doubleValue(21.5)),
							declared("state", 2, sparkplug.String, stringValue("idle")),
						),
					))
					Expect(ev).To(Equal(sparkplug.Event{
						Type: sparkplug.NBirth,
						Node: node,
						Metrics: []sparkplug.Metric{
							{
								Name:      sparkplug.BdSeqMetric,
								DataType:  sparkplug.UInt64,
								Value:     uint64(7),
								Timestamp: payloadTime,
							},
							{
								Name:      "temp",
								DataType:  sparkplug.Double,
								Value:     21.5,
								Timestamp: payloadTime,
							},
							{
								Name:      "state",
								DataType:  sparkplug.String,
								Value:     "idle",
								Timestamp: payloadTime,
							},
						},
					}))
					Expect(h.Born(node)).To(BeTrue())
					Expect(h.Born(other)).To(BeFalse())
				},
			)
			It("Should replace the tags of the last birth", func() {
				MustSucceed(h.Handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, declared("temp", 1, sparkplug.Double, doubleValue(0))),
				))
				MustSucceed(h.Handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				))
				ev := MustSucceed(h.Handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, intValue(40)), byName("temp", doubleValue(1))),
				))
				Expect(ev.Rebirth).To(BeTrue())
				Expect(ev.Metrics).To(Equal([]sparkplug.Metric{{
					Name:      "speed",
					DataType:  sparkplug.Int32,
					Value:     int64(40),
					Timestamp: payloadTime,
				}}))
			})
			It("Should drop the devices of the last birth", func() {
				MustSucceed(h.Handle(nodeTopic(sparkplug.NBirth), birth(0, 0)))
				MustSucceed(h.Handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				))
				MustSucceed(h.Handle(nodeTopic(sparkplug.NBirth), birth(0, 1)))
				Expect(h.Handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(1, byAlias(1, intValue(40))),
				)).To(Equal(sparkplug.Event{Node: node, Device: "pump", Rebirth: true}))
			})
		})

		Describe("NDATA", func() {
			BeforeEach(func() {
				MustSucceed(h.Handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0,
						declared("temp", 1, sparkplug.Double, doubleValue(0)),
						declared("count", 2, sparkplug.Int16, intValue(0)),
					),
				))
			})
			It(
				"Should resolve an alias to the name and data type of the birth",
				func() {
					Expect(h.Handle(
						nodeTopic(sparkplug.NData),
						data(
							1,
							byAlias(2, intValue(0xFFFFFFFF)),
							byAlias(1, doubleValue(3.5)),
						),
					)).To(Equal(sparkplug.Event{
						Type: sparkplug.NData,
						Node: node,
						Metrics: []sparkplug.Metric{
							{
								Name:      "count",
								DataType:  sparkplug.Int16,
								Value:     int64(-1),
								Timestamp: payloadTime,
							},
							{
								Name:      "temp",
								DataType:  sparkplug.Double,
								Value:     3.5,
								Timestamp: payloadTime,
							},
						},
					}))
				},
			)
			It("Should resolve a metric by its name", func() {
				ev := MustSucceed(h.Handle(
					nodeTopic(sparkplug.NData),
					data(1, byName("count", intValue(9))),
				))
				Expect(ev.Rebirth).To(BeFalse())
				Expect(ev.Metrics).To(Equal([]sparkplug.Metric{{
					Name:      "count",
					DataType:  sparkplug.Int16,
					Value:     int64(9),
					Timestamp: payloadTime,
				}}))
			})
			It("Should resolve a tag that the birth declared with no alias", func() {
				bare := byName("bare", doubleValue(0))
				bare.Datatype = new(uint32(sparkplug.Double))
				MustSucceed(h.Handle(nodeTopic(sparkplug.NBirth), birth(0, 0, bare)))
				ev := MustSucceed(h.Handle(
					nodeTopic(sparkplug.NData),
					data(1, byName("bare", doubleValue(4))),
				))
				Expect(ev.Rebirth).To(BeFalse())
				Expect(ev.Metrics).To(HaveLen(1))
				Expect(ev.Metrics[0].Value).To(Equal(4.0))
			})
			It(
				"Should ask for a rebirth on an unknown alias and keep the rest",
				func() {
					ev := MustSucceed(h.Handle(
						nodeTopic(sparkplug.NData),
						data(
							1,
							byAlias(99, doubleValue(1)),
							byAlias(1, doubleValue(2)),
						),
					))
					Expect(ev.Type).To(Equal(sparkplug.NData))
					Expect(ev.Rebirth).To(BeTrue())
					Expect(ev.Metrics).To(Equal([]sparkplug.Metric{{
						Name:      "temp",
						DataType:  sparkplug.Double,
						Value:     2.0,
						Timestamp: payloadTime,
					}}))
					Expect(h.Born(node)).To(BeTrue())
				},
			)
			It("Should ask for a rebirth on an unknown name", func() {
				ev := MustSucceed(h.Handle(
					nodeTopic(sparkplug.NData),
					data(1, byName("missing", doubleValue(1))),
				))
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeTrue())
				Expect(ev.Metrics).To(BeEmpty())
			})
			It(
				"Should ask for a rebirth on a metric with no name and no alias",
				func() {
					ev := MustSucceed(h.Handle(
						nodeTopic(sparkplug.NData),
						data(1, doubleValue(1)),
					))
					Expect(ev.Rebirth).To(BeTrue())
					Expect(ev.Metrics).To(BeEmpty())
				},
			)
			It("Should drop data before a birth and ask for a rebirth", func() {
				Expect(h.Handle(
					sparkplug.Topic{Type: sparkplug.NData, Node: other},
					data(1, byAlias(1, doubleValue(1))),
				)).To(Equal(sparkplug.Event{Node: other, Rebirth: true}))
				Expect(h.Born(other)).To(BeFalse())
			})
		})

		Describe("Sequence", func() {
			BeforeEach(func() {
				MustSucceed(h.Handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, declared("temp", 1, sparkplug.Double, doubleValue(0))),
				))
			})
			It("Should accept messages in sequence", func() {
				for seq := uint64(1); seq <= 3; seq++ {
					ev := MustSucceed(h.Handle(
						nodeTopic(sparkplug.NData),
						data(seq, byAlias(1, doubleValue(1))),
					))
					Expect(ev.Type).To(Equal(sparkplug.NData))
					Expect(ev.Rebirth).To(BeFalse())
				}
			})
			It("Should drop a message after a gap and clear the birth", func() {
				Expect(h.Handle(
					nodeTopic(sparkplug.NData),
					data(2, byAlias(1, doubleValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Rebirth: true}))
				Expect(h.Born(node)).To(BeFalse())
			})
			It("Should drop a message that repeats a sequence number", func() {
				MustSucceed(h.Handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				))
				Expect(h.Handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Rebirth: true}))
				Expect(h.Born(node)).To(BeFalse())
			})
			It("Should drop every message after a gap until a new birth", func() {
				MustSucceed(h.Handle(nodeTopic(sparkplug.NData), data(5)))
				Expect(h.Handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Rebirth: true}))
				MustSucceed(h.Handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 1, declared("temp", 1, sparkplug.Double, doubleValue(0))),
				))
				Expect(h.Born(node)).To(BeTrue())
				ev := MustSucceed(h.Handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				))
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeFalse())
			})
			It("Should wrap from 255 to 0", func() {
				MustSucceed(h.Handle(nodeTopic(sparkplug.NBirth), birth(254, 0)))
				for _, seq := range []uint64{255, 0, 1} {
					ev := MustSucceed(h.Handle(nodeTopic(sparkplug.NData), data(seq)))
					Expect(ev.Type).To(Equal(sparkplug.NData))
					Expect(ev.Rebirth).To(BeFalse())
				}
				Expect(h.Born(node)).To(BeTrue())
			})
			It("Should wrap to 0 after a birth with sequence number 255", func() {
				MustSucceed(h.Handle(nodeTopic(sparkplug.NBirth), birth(255, 0)))
				ev := MustSucceed(h.Handle(nodeTopic(sparkplug.NData), data(0)))
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeFalse())
			})
			It("Should accept a message with no sequence number", func() {
				noSeq := encode(&pb.Payload{
					Timestamp: new(payloadMillis),
					Metrics:   []*pb.Payload_Metric{byAlias(1, doubleValue(1))},
				})
				ev := MustSucceed(h.Handle(nodeTopic(sparkplug.NData), noSeq))
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeFalse())
				Expect(ev.Metrics).To(HaveLen(1))
				// The message with no sequence number does not use one up.
				ev = MustSucceed(h.Handle(nodeTopic(sparkplug.NData), data(1)))
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(h.Born(node)).To(BeTrue())
			})
			It("Should count device messages in the sequence of the node", func() {
				MustSucceed(h.Handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				))
				Expect(h.Handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(3, byAlias(1, intValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Device: "pump", Rebirth: true}))
				Expect(h.Born(node)).To(BeFalse())
			})
			It("Should keep one sequence for each edge node", func() {
				MustSucceed(h.Handle(
					sparkplug.Topic{Type: sparkplug.NBirth, Node: other},
					birth(0, 0),
				))
				MustSucceed(h.Handle(nodeTopic(sparkplug.NData), data(1)))
				ev := MustSucceed(h.Handle(
					sparkplug.Topic{Type: sparkplug.NData, Node: other},
					data(1),
				))
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeFalse())
			})
		})

		Describe("Devices", func() {
			BeforeEach(func() {
				MustSucceed(h.Handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, declared("temp", 1, sparkplug.Double, doubleValue(0))),
				))
			})
			It("Should return the metrics of a DBIRTH", func() {
				Expect(h.Handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(1200))),
				)).To(Equal(sparkplug.Event{
					Type:   sparkplug.DBirth,
					Node:   node,
					Device: "pump",
					Metrics: []sparkplug.Metric{{
						Name:      "speed",
						DataType:  sparkplug.Int32,
						Value:     int64(1200),
						Timestamp: payloadTime,
					}},
				}))
			})
			It("Should resolve the aliases of each device on their own", func() {
				MustSucceed(h.Handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				))
				MustSucceed(h.Handle(
					deviceTopic(sparkplug.DBirth, "valve"),
					data(2, declared("open", 1, sparkplug.Boolean, boolValue(false))),
				))
				Expect(h.Handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(3, byAlias(1, intValue(900))),
				)).To(Equal(sparkplug.Event{
					Type:   sparkplug.DData,
					Node:   node,
					Device: "pump",
					Metrics: []sparkplug.Metric{{
						Name:      "speed",
						DataType:  sparkplug.Int32,
						Value:     int64(900),
						Timestamp: payloadTime,
					}},
				}))
				Expect(h.Handle(
					deviceTopic(sparkplug.DData, "valve"),
					data(4, byAlias(1, boolValue(true))),
				)).To(Equal(sparkplug.Event{
					Type:   sparkplug.DData,
					Node:   node,
					Device: "valve",
					Metrics: []sparkplug.Metric{{
						Name:      "open",
						DataType:  sparkplug.Boolean,
						Value:     true,
						Timestamp: payloadTime,
					}},
				}))
				ev := MustSucceed(h.Handle(
					nodeTopic(sparkplug.NData),
					data(5, byAlias(1, doubleValue(20))),
				))
				Expect(ev.Metrics).To(HaveLen(1))
				Expect(ev.Metrics[0].Name).To(Equal("temp"))
			})
			It("Should ask for a rebirth on DDATA of a device with no DBIRTH", func() {
				Expect(h.Handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(1, byAlias(1, doubleValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Device: "pump", Rebirth: true}))
				Expect(h.Born(node)).To(BeTrue())
			})
			It("Should forget the tags of a device on DDEATH", func() {
				MustSucceed(h.Handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				))
				Expect(h.Handle(deviceTopic(sparkplug.DDeath, "pump"), data(2))).To(
					Equal(sparkplug.Event{
						Type:   sparkplug.DDeath,
						Node:   node,
						Device: "pump",
					}),
				)
				Expect(h.Born(node)).To(BeTrue())
				Expect(h.Handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(3, byAlias(1, intValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Device: "pump", Rebirth: true}))
			})
			It("Should keep the other devices and the node on DDEATH", func() {
				MustSucceed(h.Handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				))
				MustSucceed(h.Handle(
					deviceTopic(sparkplug.DBirth, "valve"),
					data(2, declared("open", 1, sparkplug.Boolean, boolValue(false))),
				))
				MustSucceed(h.Handle(deviceTopic(sparkplug.DDeath, "pump"), data(3)))
				ev := MustSucceed(h.Handle(
					deviceTopic(sparkplug.DData, "valve"),
					data(4, byAlias(1, boolValue(true))),
				))
				Expect(ev.Type).To(Equal(sparkplug.DData))
				Expect(ev.Rebirth).To(BeFalse())
				ev = MustSucceed(h.Handle(
					nodeTopic(sparkplug.NData),
					data(5, byAlias(1, doubleValue(1))),
				))
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeFalse())
			})
			It("Should accept a new DBIRTH after a DDEATH", func() {
				MustSucceed(h.Handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				))
				MustSucceed(h.Handle(deviceTopic(sparkplug.DDeath, "pump"), data(2)))
				MustSucceed(h.Handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(3, declared("flow", 1, sparkplug.Double, doubleValue(0))),
				))
				ev := MustSucceed(h.Handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(4, byAlias(1, doubleValue(2.5))),
				))
				Expect(ev.Metrics).To(HaveLen(1))
				Expect(ev.Metrics[0].Name).To(Equal("flow"))
			})
			It("Should ask for a rebirth on a DBIRTH before an NBIRTH", func() {
				Expect(h.Handle(
					sparkplug.Topic{
						Type:   sparkplug.DBirth,
						Node:   other,
						Device: "pump",
					},
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				)).To(Equal(sparkplug.Event{Node: other, Device: "pump", Rebirth: true}))
				Expect(h.Born(other)).To(BeFalse())
			})
			It("Should ask for a rebirth on a DDEATH before an NBIRTH", func() {
				Expect(h.Handle(
					sparkplug.Topic{
						Type:   sparkplug.DDeath,
						Node:   other,
						Device: "pump",
					},
					data(1),
				)).To(Equal(sparkplug.Event{Node: other, Device: "pump", Rebirth: true}))
			})
		})

		Describe("NDEATH", func() {
			It("Should clear the birth on a matching bdSeq", func() {
				MustSucceed(h.Handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 4, declared("temp", 1, sparkplug.Double, doubleValue(0))),
				))
				Expect(h.Handle(nodeTopic(sparkplug.NDeath), death(4))).To(
					Equal(sparkplug.Event{Type: sparkplug.NDeath, Node: node}),
				)
				Expect(h.Born(node)).To(BeFalse())
				Expect(h.Handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Rebirth: true}))
			})
			It("Should drop a death with another bdSeq", func() {
				MustSucceed(h.Handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 4, declared("temp", 1, sparkplug.Double, doubleValue(0))),
				))
				Expect(h.Handle(nodeTopic(sparkplug.NDeath), death(3))).To(
					Equal(sparkplug.Event{Node: node}),
				)
				Expect(h.Born(node)).To(BeTrue())
				ev := MustSucceed(h.Handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				))
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeFalse())
			})
			It("Should accept a death when the birth gave no bdSeq", func() {
				MustSucceed(h.Handle(nodeTopic(sparkplug.NBirth), data(0)))
				Expect(h.Born(node)).To(BeTrue())
				Expect(h.Handle(nodeTopic(sparkplug.NDeath), death(9))).To(
					Equal(sparkplug.Event{Type: sparkplug.NDeath, Node: node}),
				)
				Expect(h.Born(node)).To(BeFalse())
			})
			It("Should accept a death that carries no bdSeq", func() {
				MustSucceed(h.Handle(nodeTopic(sparkplug.NBirth), birth(0, 4)))
				Expect(h.Handle(
					nodeTopic(sparkplug.NDeath),
					encode(&pb.Payload{Timestamp: new(payloadMillis)}),
				)).To(Equal(sparkplug.Event{Type: sparkplug.NDeath, Node: node}))
				Expect(h.Born(node)).To(BeFalse())
			})
			It("Should accept a death of a node that the host never saw", func() {
				Expect(h.Handle(nodeTopic(sparkplug.NDeath), death(0))).To(
					Equal(sparkplug.Event{Type: sparkplug.NDeath, Node: node}),
				)
				Expect(h.Born(node)).To(BeFalse())
			})
			It("Should read a bdSeq that the edge node sent in int_value", func() {
				bdSeq := declared(
					sparkplug.BdSeqMetric,
					0,
					sparkplug.UInt64,
					intValue(4),
				)
				MustSucceed(h.Handle(nodeTopic(sparkplug.NBirth), data(0, bdSeq)))
				Expect(h.Handle(nodeTopic(sparkplug.NDeath), death(3))).To(
					Equal(sparkplug.Event{Node: node}),
				)
				Expect(h.Born(node)).To(BeTrue())
			})
			It("Should take the devices of the node offline", func() {
				MustSucceed(h.Handle(nodeTopic(sparkplug.NBirth), birth(0, 4)))
				MustSucceed(h.Handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				))
				MustSucceed(h.Handle(nodeTopic(sparkplug.NDeath), death(4)))
				Expect(h.Handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(2, byAlias(1, intValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Device: "pump", Rebirth: true}))
			})
		})

		Describe("Other topics", func() {
			DescribeTable(
				"Should give an empty event and no error",
				func(topic sparkplug.Topic) {
					Expect(h.Handle(topic, []byte("not a payload"))).To(
						Equal(sparkplug.Event{}),
					)
					Expect(h.Born(node)).To(BeFalse())
				},
				Entry("NCMD", sparkplug.CommandTopic(node, "")),
				Entry("DCMD", sparkplug.CommandTopic(node, "pump")),
				Entry(
					"STATE",
					sparkplug.Topic{Type: sparkplug.State, HostID: "synnax"},
				),
				Entry("a topic with no type", sparkplug.Topic{Node: node}),
			)
		})

		Describe("Invalid payloads", func() {
			DescribeTable("Should return an error that names the topic",
				func(topic sparkplug.Topic, text string) {
					Expect(h.Handle(topic, []byte{0xFF, 0xFF, 0xFF})).Error().To(
						MatchError(ContainSubstring("invalid payload on " + text)),
					)
				},
				Entry("NBIRTH",
					sparkplug.Topic{Type: sparkplug.NBirth, Node: node},
					"spBv1.0/plant/NBIRTH/line1",
				),
				Entry("NDATA",
					sparkplug.Topic{Type: sparkplug.NData, Node: node},
					"spBv1.0/plant/NDATA/line1",
				),
				Entry("NDEATH",
					sparkplug.Topic{Type: sparkplug.NDeath, Node: node},
					"spBv1.0/plant/NDEATH/line1",
				),
				Entry("DBIRTH",
					sparkplug.Topic{Type: sparkplug.DBirth, Node: node, Device: "pump"},
					"spBv1.0/plant/DBIRTH/line1/pump",
				),
				Entry("DDATA",
					sparkplug.Topic{Type: sparkplug.DData, Node: node, Device: "pump"},
					"spBv1.0/plant/DDATA/line1/pump",
				),
				Entry("DDEATH",
					sparkplug.Topic{Type: sparkplug.DDeath, Node: node, Device: "pump"},
					"spBv1.0/plant/DDEATH/line1/pump",
				),
			)
			It("Should keep the session state of the node", func() {
				MustSucceed(h.Handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, declared("temp", 1, sparkplug.Double, doubleValue(0))),
				))
				Expect(h.Handle(nodeTopic(sparkplug.NData), []byte{0xFF})).Error().To(
					MatchError(ContainSubstring("invalid payload on")),
				)
				Expect(h.Born(node)).To(BeTrue())
				ev := MustSucceed(h.Handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				))
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeFalse())
			})
		})
	})

	Describe("Born", func() {
		It("Should be false for a node that the host never saw", func() {
			Expect(h.Born(node)).To(BeFalse())
		})
		It("Should be true after an NBIRTH, for that node only", func() {
			MustSucceed(h.Handle(nodeTopic(sparkplug.NBirth), birth(0, 0)))
			Expect(h.Born(node)).To(BeTrue())
			Expect(h.Born(other)).To(BeFalse())
		})
		It("Should stay false after data with no birth", func() {
			MustSucceed(h.Handle(nodeTopic(sparkplug.NData), data(1)))
			Expect(h.Born(node)).To(BeFalse())
		})
	})

	Describe("Reset", func() {
		It("Should clear the birth of every edge node", func() {
			MustSucceed(h.Handle(
				nodeTopic(sparkplug.NBirth),
				birth(0, 0, declared("temp", 1, sparkplug.Double, doubleValue(0))),
			))
			MustSucceed(h.Handle(
				deviceTopic(sparkplug.DBirth, "pump"),
				data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
			))
			MustSucceed(h.Handle(
				sparkplug.Topic{Type: sparkplug.NBirth, Node: other},
				birth(0, 0),
			))
			h.Reset()
			Expect(h.Born(node)).To(BeFalse())
			Expect(h.Born(other)).To(BeFalse())
			Expect(h.Handle(
				nodeTopic(sparkplug.NData),
				data(2, byAlias(1, doubleValue(1))),
			)).To(Equal(sparkplug.Event{Node: node, Rebirth: true}))
			Expect(h.Handle(
				deviceTopic(sparkplug.DData, "pump"),
				data(3, byAlias(1, intValue(1))),
			)).To(Equal(sparkplug.Event{Node: node, Device: "pump", Rebirth: true}))
			Expect(h.Handle(
				sparkplug.Topic{Type: sparkplug.NData, Node: other},
				data(1),
			)).To(Equal(sparkplug.Event{Node: other, Rebirth: true}))
		})
		It("Should accept a new birth after the reset", func() {
			MustSucceed(h.Handle(nodeTopic(sparkplug.NBirth), birth(0, 0)))
			h.Reset()
			MustSucceed(h.Handle(
				nodeTopic(sparkplug.NBirth),
				birth(0, 1, declared("temp", 1, sparkplug.Double, doubleValue(0))),
			))
			Expect(h.Born(node)).To(BeTrue())
			ev := MustSucceed(h.Handle(
				nodeTopic(sparkplug.NData),
				data(1, byAlias(1, doubleValue(1))),
			))
			Expect(ev.Type).To(Equal(sparkplug.NData))
			Expect(ev.Metrics).To(HaveLen(1))
		})
		It("Should do nothing on a host that knows no edge node", func() {
			h.Reset()
			Expect(h.Born(node)).To(BeFalse())
		})
	})

	Describe("Forget", func() {
		It("Should drop the state of the node and keep the others", func() {
			MustSucceed(h.Handle(
				nodeTopic(sparkplug.NBirth),
				birth(0, 0, declared("temp", 1, sparkplug.Double, doubleValue(0))),
			))
			MustSucceed(h.Handle(
				sparkplug.Topic{Type: sparkplug.NBirth, Node: other},
				birth(0, 0),
			))
			h.Forget(node)
			Expect(h.Born(node)).To(BeFalse())
			Expect(h.Born(other)).To(BeTrue())
			Expect(h.Handle(
				nodeTopic(sparkplug.NData),
				data(1, byAlias(1, doubleValue(1))),
			)).To(Equal(sparkplug.Event{Node: node, Rebirth: true}))
		})
		It("Should drop the bdSeq of the node", func() {
			MustSucceed(h.Handle(nodeTopic(sparkplug.NBirth), birth(0, 4)))
			h.Forget(node)
			Expect(h.Handle(nodeTopic(sparkplug.NDeath), death(3))).To(
				Equal(sparkplug.Event{Type: sparkplug.NDeath, Node: node}),
			)
		})
		It("Should do nothing for a node that the host never saw", func() {
			h.Forget(node)
			Expect(h.Born(node)).To(BeFalse())
		})
	})

	Describe("TakeRebirth", func() {
		now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

		It("Should return zero the first time", func() {
			Expect(h.TakeRebirth(node, now)).To(BeZero())
		})
		It("Should return the time left within the interval", func() {
			Expect(h.TakeRebirth(node, now)).To(BeZero())
			Expect(h.TakeRebirth(node, now)).To(Equal(5 * time.Second))
			Expect(h.TakeRebirth(node, now.Add(2*time.Second))).To(
				Equal(3 * time.Second),
			)
			Expect(h.TakeRebirth(node, now.Add(5*time.Second-time.Nanosecond))).To(
				Equal(time.Nanosecond),
			)
		})
		It("Should return zero again after the interval", func() {
			Expect(h.TakeRebirth(node, now)).To(BeZero())
			Expect(h.TakeRebirth(node, now.Add(5*time.Second))).To(BeZero())
			Expect(h.TakeRebirth(node, now.Add(6*time.Second))).To(
				Equal(4 * time.Second),
			)
		})
		It("Should not record a request that it refused", func() {
			Expect(h.TakeRebirth(node, now)).To(BeZero())
			Expect(h.TakeRebirth(node, now.Add(4*time.Second))).To(Equal(time.Second))
			Expect(h.TakeRebirth(node, now.Add(5*time.Second))).To(BeZero())
		})
		It("Should keep one timer for each edge node", func() {
			Expect(h.TakeRebirth(node, now)).To(BeZero())
			Expect(h.TakeRebirth(other, now.Add(time.Second))).To(BeZero())
			Expect(h.TakeRebirth(node, now.Add(time.Second))).To(Equal(4 * time.Second))
			Expect(h.TakeRebirth(other, now.Add(2*time.Second))).To(
				Equal(4 * time.Second),
			)
		})
		It("Should use the RebirthInterval of the host", func() {
			h.RebirthInterval = time.Minute
			Expect(h.TakeRebirth(node, now)).To(BeZero())
			Expect(h.TakeRebirth(node, now.Add(10*time.Second))).To(
				Equal(50 * time.Second),
			)
			Expect(h.TakeRebirth(node, now.Add(time.Minute))).To(BeZero())
		})
		It("Should keep the timer across a Reset", func() {
			Expect(h.TakeRebirth(node, now)).To(BeZero())
			h.Reset()
			Expect(h.TakeRebirth(node, now.Add(time.Second))).To(Equal(4 * time.Second))
		})
	})
})
