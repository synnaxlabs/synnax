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

func bdSeqMetric(bdSeq uint64) *pb.Payload_Metric {
	return declared(sparkplug.BdSeqMetric, 0, sparkplug.UInt64, longValue(bdSeq))
}

// birth returns an NBIRTH payload. Its first metric is bdSeq.
func birth(seq, bdSeq uint64, metrics ...*pb.Payload_Metric) *pb.Payload {
	return &pb.Payload{
		Timestamp: new(payloadMillis),
		Seq:       new(seq),
		Metrics:   append([]*pb.Payload_Metric{bdSeqMetric(bdSeq)}, metrics...),
	}
}

// data returns the payload of an NDATA, DBIRTH, DDATA, or DDEATH message.
func data(seq uint64, metrics ...*pb.Payload_Metric) *pb.Payload {
	return &pb.Payload{
		Timestamp: new(payloadMillis),
		Seq:       new(seq),
		Metrics:   metrics,
	}
}

// death returns an NDEATH payload, which has no sequence number.
func death(bdSeq uint64) *pb.Payload {
	return &pb.Payload{
		Timestamp: new(payloadMillis),
		Metrics:   []*pb.Payload_Metric{bdSeqMetric(bdSeq)},
	}
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
	BeforeEach(func() { h = sparkplug.NewHost(sparkplug.DefaultRebirthInterval) })
	// handle applies the message on topic with payload p to h.
	handle := func(topic sparkplug.Topic, p *pb.Payload) sparkplug.Event {
		GinkgoHelper()
		return h.Handle(MustSucceed(sparkplug.Decode(topic, encode(p))))
	}
	// born reports whether h holds a birth of n. It sends data with no sequence number,
	// which h takes only from a born node and which uses no sequence number up.
	born := func(n sparkplug.NodeID) bool {
		GinkgoHelper()
		topic := sparkplug.Topic{Type: sparkplug.NData, Node: n}
		ev := handle(topic, &pb.Payload{Timestamp: new(payloadMillis)})
		return ev.Type == sparkplug.NData
	}

	Describe("NewHost", func() {
		It("Should know no edge node", func() {
			Expect(born(node)).To(BeFalse())
		})
	})

	Describe("DefaultRebirthInterval", func() {
		It("Should be five seconds", func() {
			Expect(sparkplug.DefaultRebirthInterval).To(Equal(5 * time.Second))
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
					born := handle(
						nodeTopic(sparkplug.NBirth),
						birth(0, 0, declared("tag", 1, dt, value)),
					)
					Expect(born.Metrics).To(HaveLen(2))
					Expect(born.Metrics[1].Name).To(Equal("tag"))
					Expect(born.Metrics[1].DataType).To(Equal(dt))
					Expect(born.Metrics[1].Value).To(expected)
					ev := handle(
						nodeTopic(sparkplug.NData),
						data(1, byAlias(1, value)),
					)
					Expect(ev.Rebirth).To(BeFalse())
					Expect(ev.Metrics).To(HaveLen(1))
					Expect(ev.Metrics[0].Name).To(Equal("tag"))
					Expect(ev.Metrics[0].DataType).To(Equal(dt))
					Expect(ev.Metrics[0].Value).To(expected)
				},
				Entry(
					"Int8",
					sparkplug.Int8,
					intValue(127),
					Equal(sparkplug.IntValue(127)),
				),
				Entry("a negative Int8 in 32 bits",
					sparkplug.Int8, intValue(0xFFFFFFFB), Equal(sparkplug.IntValue(-5)),
				),
				Entry("a negative Int8 in 8 bits",
					sparkplug.Int8, intValue(0xFB), Equal(sparkplug.IntValue(-5)),
				),
				Entry(
					"Int16",
					sparkplug.Int16,
					intValue(32767),
					Equal(sparkplug.IntValue(32767)),
				),
				Entry(
					"a negative Int16 in 32 bits",
					sparkplug.Int16,
					intValue(0xFFFF8000),
					Equal(sparkplug.IntValue(-32768)),
				),
				Entry(
					"a negative Int16 in 16 bits",
					sparkplug.Int16,
					intValue(0x8000),
					Equal(sparkplug.IntValue(-32768)),
				),
				Entry(
					"Int32",
					sparkplug.Int32,
					intValue(math.MaxInt32),
					Equal(sparkplug.IntValue(math.MaxInt32)),
				),
				Entry(
					"a negative Int32",
					sparkplug.Int32,
					intValue(0x80000000),
					Equal(sparkplug.IntValue(math.MinInt32)),
				),
				Entry(
					"an Int32 in long_value",
					sparkplug.Int32,
					longValue(0xFFFFFFFF),
					Equal(sparkplug.IntValue(-1)),
				),
				Entry(
					"Int64",
					sparkplug.Int64,
					longValue(math.MaxInt64),
					Equal(sparkplug.IntValue(math.MaxInt64)),
				),
				Entry(
					"a negative Int64",
					sparkplug.Int64,
					longValue(0xFFFFFFFFFFFFFFFB),
					Equal(sparkplug.IntValue(-5)),
				),
				Entry("an Int64 in int_value",
					sparkplug.Int64, intValue(12), Equal(sparkplug.IntValue(12)),
				),
				Entry(
					"UInt8",
					sparkplug.UInt8,
					intValue(255),
					Equal(sparkplug.UintValue(255)),
				),
				Entry(
					"UInt16",
					sparkplug.UInt16,
					intValue(65535),
					Equal(sparkplug.UintValue(65535)),
				),
				Entry("UInt32",
					sparkplug.UInt32,
					intValue(math.MaxUint32),
					Equal(sparkplug.UintValue(math.MaxUint32)),
				),
				Entry("a UInt32 in long_value",
					sparkplug.UInt32, longValue(12), Equal(sparkplug.UintValue(12)),
				),
				Entry("UInt64",
					sparkplug.UInt64,
					longValue(math.MaxUint64),
					Equal(sparkplug.UintValue(math.MaxUint64)),
				),
				Entry("a UInt64 in int_value",
					sparkplug.UInt64, intValue(12), Equal(sparkplug.UintValue(12)),
				),
				Entry(
					"Float",
					sparkplug.Float,
					floatValue(1.5),
					Equal(sparkplug.FloatValue(1.5)),
				),
				Entry(
					"Double",
					sparkplug.Double,
					doubleValue(2.25),
					Equal(sparkplug.FloatValue(2.25)),
				),
				Entry(
					"Boolean true",
					sparkplug.Boolean,
					boolValue(true),
					Equal(sparkplug.BoolValue(true)),
				),
				Entry(
					"Boolean false",
					sparkplug.Boolean,
					boolValue(false),
					Equal(sparkplug.BoolValue(false)),
				),
				Entry(
					"String",
					sparkplug.String,
					stringValue("open"),
					Equal(sparkplug.StringValue("open")),
				),
				Entry(
					"Text",
					sparkplug.Text,
					stringValue("a note"),
					Equal(sparkplug.StringValue("a note")),
				),
				Entry(
					"UUID",
					sparkplug.UUID,
					stringValue("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
					Equal(
						sparkplug.StringValue("6ba7b810-9dad-11d1-80b4-00c04fd430c8"),
					),
				),
				Entry("DateTime",
					sparkplug.DateTime,
					longValue(1700000000456),
					Equal(sparkplug.DateTimeValue(1700000000456*telem.MillisecondTS)),
				),
				Entry("the latest DateTime that a timestamp holds",
					sparkplug.DateTime,
					longValue(uint64(telem.TimeStampMax/telem.MillisecondTS)),
					Equal(sparkplug.DateTimeValue(
						telem.TimeStampMax/telem.MillisecondTS*telem.MillisecondTS,
					)),
				),
				Entry("a DateTime later than a timestamp holds",
					sparkplug.DateTime, longValue(253402214400000), BeNil(),
				),
				Entry("a DateTime past the signed range",
					sparkplug.DateTime, longValue(math.MaxUint64), BeNil(),
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
				handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, declared("tag", 1, sparkplug.Int8, intValue(1))),
				)
				m := byAlias(1, intValue(0xFFFF))
				m.Datatype = new(uint32(sparkplug.UInt16))
				ev := handle(nodeTopic(sparkplug.NData), data(1, m))
				Expect(ev.Metrics).To(Equal([]sparkplug.Metric{{
					Name:      "tag",
					DataType:  sparkplug.UInt16,
					Value:     sparkplug.UintValue(0xFFFF),
					Timestamp: payloadTime,
				}}))
			})
		})

		Describe("Timestamps", func() {
			BeforeEach(func() {
				handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, declared("tag", 1, sparkplug.Double, doubleValue(0))),
				)
			})
			It("Should use the timestamp of the metric", func() {
				m := byAlias(1, doubleValue(1))
				m.Timestamp = new(payloadMillis + 250)
				ev := handle(nodeTopic(sparkplug.NData), data(1, m))
				Expect(ev.Metrics).To(HaveLen(1))
				Expect(ev.Metrics[0].Timestamp).To(
					Equal(payloadTime + 250*telem.MillisecondTS),
				)
			})
			It("Should fall back to the timestamp of the payload", func() {
				ev := handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				)
				Expect(ev.Metrics).To(HaveLen(1))
				Expect(ev.Metrics[0].Timestamp).To(Equal(payloadTime))
			})
			It("Should give zero for a timestamp later than a timestamp holds", func() {
				m := byAlias(1, doubleValue(1))
				m.Timestamp = new(uint64(253402214400000))
				ev := handle(nodeTopic(sparkplug.NData), data(1, m))
				Expect(ev.Metrics).To(HaveLen(1))
				Expect(ev.Metrics[0].Timestamp).To(BeZero())
			})
			It("Should give zero when the message has no timestamp", func() {
				ev := handle(
					nodeTopic(sparkplug.NData),
					&pb.Payload{
						Seq:     new(uint64(1)),
						Metrics: []*pb.Payload_Metric{byAlias(1, doubleValue(1))},
					},
				)
				Expect(ev.Metrics).To(HaveLen(1))
				Expect(ev.Metrics[0].Timestamp).To(BeZero())
			})
			It("Should use the timestamps of a birth the same way", func() {
				stamped := declared("stamped", 2, sparkplug.Double, doubleValue(1))
				stamped.Timestamp = new(payloadMillis - 1000)
				ev := handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, stamped),
				)
				Expect(ev.Metrics).To(HaveLen(2))
				Expect(ev.Metrics[0].Timestamp).To(Equal(payloadTime))
				Expect(ev.Metrics[1].Timestamp).To(
					Equal(payloadTime - 1000*telem.MillisecondTS),
				)
			})
		})

		Describe("Historical", func() {
			It("Should carry the historical flag of a data metric", func() {
				handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, declared("tag", 1, sparkplug.Double, doubleValue(0))),
				)
				old := byAlias(1, doubleValue(1))
				old.IsHistorical = new(true)
				ev := handle(
					nodeTopic(sparkplug.NData),
					data(1, old, byAlias(1, doubleValue(2))),
				)
				Expect(ev.Metrics).To(HaveLen(2))
				Expect(ev.Metrics[0].Historical).To(BeTrue())
				Expect(ev.Metrics[1].Historical).To(BeFalse())
			})
			It("Should carry the historical flag of a birth metric", func() {
				old := declared("tag", 1, sparkplug.Double, doubleValue(0))
				old.IsHistorical = new(true)
				ev := handle(nodeTopic(sparkplug.NBirth), birth(0, 0, old))
				Expect(ev.Metrics).To(HaveLen(2))
				Expect(ev.Metrics[0].Historical).To(BeFalse())
				Expect(ev.Metrics[1].Historical).To(BeTrue())
			})
		})

		Describe("NBIRTH", func() {
			It(
				"Should return every metric of the birth and mark the node born",
				func() {
					ev := handle(
						nodeTopic(sparkplug.NBirth),
						birth(0, 7,
							declared("temp", 1, sparkplug.Double, doubleValue(21.5)),
							declared("state", 2, sparkplug.String, stringValue("idle")),
						),
					)
					Expect(ev).To(Equal(sparkplug.Event{
						Type: sparkplug.NBirth,
						Node: node,
						Metrics: []sparkplug.Metric{
							{
								Name:      sparkplug.BdSeqMetric,
								DataType:  sparkplug.UInt64,
								Value:     sparkplug.UintValue(7),
								Timestamp: payloadTime,
							},
							{
								Name:      "temp",
								DataType:  sparkplug.Double,
								Value:     sparkplug.FloatValue(21.5),
								Timestamp: payloadTime,
							},
							{
								Name:      "state",
								DataType:  sparkplug.String,
								Value:     sparkplug.StringValue("idle"),
								Timestamp: payloadTime,
							},
						},
					}))
					Expect(born(node)).To(BeTrue())
					Expect(born(other)).To(BeFalse())
				},
			)
			It("Should replace the tags of the last birth", func() {
				handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, declared("temp", 1, sparkplug.Double, doubleValue(0))),
				)
				handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				)
				ev := handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, intValue(40)), byName("temp", doubleValue(1))),
				)
				Expect(ev.Rebirth).To(BeTrue())
				Expect(ev.Metrics).To(Equal([]sparkplug.Metric{{
					Name:      "speed",
					DataType:  sparkplug.Int32,
					Value:     sparkplug.IntValue(40),
					Timestamp: payloadTime,
				}}))
			})
			It("Should drop the devices of the last birth", func() {
				handle(nodeTopic(sparkplug.NBirth), birth(0, 0))
				handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				)
				handle(nodeTopic(sparkplug.NBirth), birth(0, 1))
				Expect(handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(1, byAlias(1, intValue(40))),
				)).To(Equal(sparkplug.Event{Node: node, Device: "pump", Rebirth: true}))
			})
		})

		Describe("NDATA", func() {
			BeforeEach(func() {
				handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0,
						declared("temp", 1, sparkplug.Double, doubleValue(0)),
						declared("count", 2, sparkplug.Int16, intValue(0)),
					),
				)
			})
			It(
				"Should resolve an alias to the name and data type of the birth",
				func() {
					Expect(handle(
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
								Value:     sparkplug.IntValue(-1),
								Timestamp: payloadTime,
							},
							{
								Name:      "temp",
								DataType:  sparkplug.Double,
								Value:     sparkplug.FloatValue(3.5),
								Timestamp: payloadTime,
							},
						},
					}))
				},
			)
			It("Should resolve a metric by its name", func() {
				ev := handle(
					nodeTopic(sparkplug.NData),
					data(1, byName("count", intValue(9))),
				)
				Expect(ev.Rebirth).To(BeFalse())
				Expect(ev.Metrics).To(Equal([]sparkplug.Metric{{
					Name:      "count",
					DataType:  sparkplug.Int16,
					Value:     sparkplug.IntValue(9),
					Timestamp: payloadTime,
				}}))
			})
			It("Should resolve a tag that the birth declared with no alias", func() {
				bare := byName("bare", doubleValue(0))
				bare.Datatype = new(uint32(sparkplug.Double))
				handle(nodeTopic(sparkplug.NBirth), birth(0, 0, bare))
				ev := handle(
					nodeTopic(sparkplug.NData),
					data(1, byName("bare", doubleValue(4))),
				)
				Expect(ev.Rebirth).To(BeFalse())
				Expect(ev.Metrics).To(HaveLen(1))
				Expect(ev.Metrics[0].Value).To(Equal(sparkplug.FloatValue(4)))
			})
			It(
				"Should ask for a rebirth on an unknown alias and keep the rest",
				func() {
					ev := handle(
						nodeTopic(sparkplug.NData),
						data(
							1,
							byAlias(99, doubleValue(1)),
							byAlias(1, doubleValue(2)),
						),
					)
					Expect(ev.Type).To(Equal(sparkplug.NData))
					Expect(ev.Rebirth).To(BeTrue())
					Expect(ev.Metrics).To(Equal([]sparkplug.Metric{{
						Name:      "temp",
						DataType:  sparkplug.Double,
						Value:     sparkplug.FloatValue(2.0),
						Timestamp: payloadTime,
					}}))
					Expect(born(node)).To(BeTrue())
				},
			)
			It("Should ask for a rebirth on an unknown name", func() {
				ev := handle(
					nodeTopic(sparkplug.NData),
					data(1, byName("missing", doubleValue(1))),
				)
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeTrue())
				Expect(ev.Metrics).To(BeEmpty())
			})
			It(
				"Should ask for a rebirth on a metric with no name and no alias",
				func() {
					ev := handle(
						nodeTopic(sparkplug.NData),
						data(1, doubleValue(1)),
					)
					Expect(ev.Rebirth).To(BeTrue())
					Expect(ev.Metrics).To(BeEmpty())
				},
			)
			It("Should drop data before a birth and ask for a rebirth", func() {
				Expect(handle(
					sparkplug.Topic{Type: sparkplug.NData, Node: other},
					data(1, byAlias(1, doubleValue(1))),
				)).To(Equal(sparkplug.Event{Node: other, Rebirth: true}))
				Expect(born(other)).To(BeFalse())
			})
		})

		Describe("Sequence", func() {
			BeforeEach(func() {
				handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, declared("temp", 1, sparkplug.Double, doubleValue(0))),
				)
			})
			It("Should accept messages in sequence", func() {
				for seq := uint64(1); seq <= 3; seq++ {
					ev := handle(
						nodeTopic(sparkplug.NData),
						data(seq, byAlias(1, doubleValue(1))),
					)
					Expect(ev.Type).To(Equal(sparkplug.NData))
					Expect(ev.Rebirth).To(BeFalse())
				}
			})
			It("Should drop a message after a gap and clear the birth", func() {
				Expect(handle(
					nodeTopic(sparkplug.NData),
					data(2, byAlias(1, doubleValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Rebirth: true, BirthLost: true}))
				Expect(born(node)).To(BeFalse())
			})
			It("Should drop a message that repeats a sequence number", func() {
				handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				)
				Expect(handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Rebirth: true, BirthLost: true}))
				Expect(born(node)).To(BeFalse())
			})
			It("Should drop every message after a gap until a new birth", func() {
				handle(nodeTopic(sparkplug.NData), data(5))
				Expect(handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Rebirth: true}))
				handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 1, declared("temp", 1, sparkplug.Double, doubleValue(0))),
				)
				Expect(born(node)).To(BeTrue())
				ev := handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				)
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeFalse())
			})
			It("Should wrap from 255 to 0", func() {
				handle(nodeTopic(sparkplug.NBirth), birth(254, 0))
				for _, seq := range []uint64{255, 0, 1} {
					ev := handle(nodeTopic(sparkplug.NData), data(seq))
					Expect(ev.Type).To(Equal(sparkplug.NData))
					Expect(ev.Rebirth).To(BeFalse())
				}
				Expect(born(node)).To(BeTrue())
			})
			It("Should wrap to 0 after a birth with sequence number 255", func() {
				handle(nodeTopic(sparkplug.NBirth), birth(255, 0))
				ev := handle(nodeTopic(sparkplug.NData), data(0))
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeFalse())
			})
			It("Should accept a message with no sequence number", func() {
				noSeq := &pb.Payload{
					Timestamp: new(payloadMillis),
					Metrics:   []*pb.Payload_Metric{byAlias(1, doubleValue(1))},
				}
				ev := handle(nodeTopic(sparkplug.NData), noSeq)
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeFalse())
				Expect(ev.Metrics).To(HaveLen(1))
				// The message with no sequence number does not use one up.
				ev = handle(nodeTopic(sparkplug.NData), data(1))
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(born(node)).To(BeTrue())
			})
			It("Should count device messages in the sequence of the node", func() {
				handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				)
				Expect(handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(3, byAlias(1, intValue(1))),
				)).To(Equal(sparkplug.Event{
					Node:      node,
					Device:    "pump",
					Rebirth:   true,
					BirthLost: true,
				}))
				Expect(born(node)).To(BeFalse())
			})
			It("Should keep one sequence for each edge node", func() {
				handle(
					sparkplug.Topic{Type: sparkplug.NBirth, Node: other},
					birth(0, 0),
				)
				handle(nodeTopic(sparkplug.NData), data(1))
				ev := handle(
					sparkplug.Topic{Type: sparkplug.NData, Node: other},
					data(1),
				)
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeFalse())
			})
		})

		Describe("Devices", func() {
			BeforeEach(func() {
				handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 0, declared("temp", 1, sparkplug.Double, doubleValue(0))),
				)
			})
			It("Should return the metrics of a DBIRTH", func() {
				Expect(handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(1200))),
				)).To(Equal(sparkplug.Event{
					Type:   sparkplug.DBirth,
					Node:   node,
					Device: "pump",
					Metrics: []sparkplug.Metric{{
						Name:      "speed",
						DataType:  sparkplug.Int32,
						Value:     sparkplug.IntValue(1200),
						Timestamp: payloadTime,
					}},
				}))
			})
			It("Should resolve the aliases of each device on their own", func() {
				handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				)
				handle(
					deviceTopic(sparkplug.DBirth, "valve"),
					data(2, declared("open", 1, sparkplug.Boolean, boolValue(false))),
				)
				Expect(handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(3, byAlias(1, intValue(900))),
				)).To(Equal(sparkplug.Event{
					Type:   sparkplug.DData,
					Node:   node,
					Device: "pump",
					Metrics: []sparkplug.Metric{{
						Name:      "speed",
						DataType:  sparkplug.Int32,
						Value:     sparkplug.IntValue(900),
						Timestamp: payloadTime,
					}},
				}))
				Expect(handle(
					deviceTopic(sparkplug.DData, "valve"),
					data(4, byAlias(1, boolValue(true))),
				)).To(Equal(sparkplug.Event{
					Type:   sparkplug.DData,
					Node:   node,
					Device: "valve",
					Metrics: []sparkplug.Metric{{
						Name:      "open",
						DataType:  sparkplug.Boolean,
						Value:     sparkplug.BoolValue(true),
						Timestamp: payloadTime,
					}},
				}))
				ev := handle(
					nodeTopic(sparkplug.NData),
					data(5, byAlias(1, doubleValue(20))),
				)
				Expect(ev.Metrics).To(HaveLen(1))
				Expect(ev.Metrics[0].Name).To(Equal("temp"))
			})
			It("Should ask for a rebirth on DDATA of a device with no DBIRTH", func() {
				Expect(handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(1, byAlias(1, doubleValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Device: "pump", Rebirth: true}))
				Expect(born(node)).To(BeTrue())
			})
			It("Should forget the tags of a device on DDEATH", func() {
				handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				)
				Expect(handle(deviceTopic(sparkplug.DDeath, "pump"), data(2))).To(
					Equal(sparkplug.Event{
						Type:   sparkplug.DDeath,
						Node:   node,
						Device: "pump",
					}),
				)
				Expect(born(node)).To(BeTrue())
				Expect(handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(3, byAlias(1, intValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Device: "pump", Rebirth: true}))
			})
			It("Should keep the other devices and the node on DDEATH", func() {
				handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				)
				handle(
					deviceTopic(sparkplug.DBirth, "valve"),
					data(2, declared("open", 1, sparkplug.Boolean, boolValue(false))),
				)
				handle(deviceTopic(sparkplug.DDeath, "pump"), data(3))
				ev := handle(
					deviceTopic(sparkplug.DData, "valve"),
					data(4, byAlias(1, boolValue(true))),
				)
				Expect(ev.Type).To(Equal(sparkplug.DData))
				Expect(ev.Rebirth).To(BeFalse())
				ev = handle(
					nodeTopic(sparkplug.NData),
					data(5, byAlias(1, doubleValue(1))),
				)
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeFalse())
			})
			It("Should accept a new DBIRTH after a DDEATH", func() {
				handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				)
				handle(deviceTopic(sparkplug.DDeath, "pump"), data(2))
				handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(3, declared("flow", 1, sparkplug.Double, doubleValue(0))),
				)
				ev := handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(4, byAlias(1, doubleValue(2.5))),
				)
				Expect(ev.Metrics).To(HaveLen(1))
				Expect(ev.Metrics[0].Name).To(Equal("flow"))
			})
			It("Should ask for a rebirth on a DBIRTH before an NBIRTH", func() {
				Expect(handle(
					sparkplug.Topic{
						Type:   sparkplug.DBirth,
						Node:   other,
						Device: "pump",
					},
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				)).To(Equal(sparkplug.Event{Node: other, Device: "pump", Rebirth: true}))
				Expect(born(other)).To(BeFalse())
			})
			It("Should ask for a rebirth on a DDEATH before an NBIRTH", func() {
				Expect(handle(
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
				handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 4, declared("temp", 1, sparkplug.Double, doubleValue(0))),
				)
				Expect(handle(nodeTopic(sparkplug.NDeath), death(4))).To(
					Equal(sparkplug.Event{Type: sparkplug.NDeath, Node: node}),
				)
				Expect(born(node)).To(BeFalse())
				Expect(handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Rebirth: true}))
			})
			It("Should drop a death with another bdSeq", func() {
				handle(
					nodeTopic(sparkplug.NBirth),
					birth(0, 4, declared("temp", 1, sparkplug.Double, doubleValue(0))),
				)
				Expect(handle(nodeTopic(sparkplug.NDeath), death(3))).To(
					Equal(sparkplug.Event{Node: node}),
				)
				Expect(born(node)).To(BeTrue())
				ev := handle(
					nodeTopic(sparkplug.NData),
					data(1, byAlias(1, doubleValue(1))),
				)
				Expect(ev.Type).To(Equal(sparkplug.NData))
				Expect(ev.Rebirth).To(BeFalse())
			})
			It("Should accept a death when the birth gave no bdSeq", func() {
				handle(nodeTopic(sparkplug.NBirth), data(0))
				Expect(born(node)).To(BeTrue())
				Expect(handle(nodeTopic(sparkplug.NDeath), death(9))).To(
					Equal(sparkplug.Event{Type: sparkplug.NDeath, Node: node}),
				)
				Expect(born(node)).To(BeFalse())
			})
			It("Should accept a death that carries no bdSeq", func() {
				handle(nodeTopic(sparkplug.NBirth), birth(0, 4))
				Expect(handle(
					nodeTopic(sparkplug.NDeath),
					&pb.Payload{Timestamp: new(payloadMillis)},
				)).To(Equal(sparkplug.Event{Type: sparkplug.NDeath, Node: node}))
				Expect(born(node)).To(BeFalse())
			})
			It("Should accept a death of a node that the host never saw", func() {
				Expect(handle(nodeTopic(sparkplug.NDeath), death(0))).To(
					Equal(sparkplug.Event{Type: sparkplug.NDeath, Node: node}),
				)
				Expect(born(node)).To(BeFalse())
			})
			It("Should read a bdSeq that the edge node sent in int_value", func() {
				bdSeq := declared(
					sparkplug.BdSeqMetric,
					0,
					sparkplug.UInt64,
					intValue(4),
				)
				handle(nodeTopic(sparkplug.NBirth), data(0, bdSeq))
				Expect(handle(nodeTopic(sparkplug.NDeath), death(3))).To(
					Equal(sparkplug.Event{Node: node}),
				)
				Expect(born(node)).To(BeTrue())
			})
			It("Should take the devices of the node offline", func() {
				handle(nodeTopic(sparkplug.NBirth), birth(0, 4))
				handle(
					deviceTopic(sparkplug.DBirth, "pump"),
					data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
				)
				handle(nodeTopic(sparkplug.NDeath), death(4))
				Expect(handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(2, byAlias(1, intValue(1))),
				)).To(Equal(sparkplug.Event{Node: node, Device: "pump", Rebirth: true}))
			})
		})

		Describe("Other topics", func() {
			DescribeTable(
				"Should give an empty event",
				func(topic sparkplug.Topic) {
					Expect(handle(topic, data(0))).To(Equal(sparkplug.Event{}))
					Expect(born(node)).To(BeFalse())
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
	})

	Describe("Born", func() {
		It("Should be false for a node that the host never saw", func() {
			Expect(born(node)).To(BeFalse())
		})
		It("Should be true after an NBIRTH, for that node only", func() {
			handle(nodeTopic(sparkplug.NBirth), birth(0, 0))
			Expect(born(node)).To(BeTrue())
			Expect(born(other)).To(BeFalse())
		})
		It("Should stay false after data with no birth", func() {
			handle(nodeTopic(sparkplug.NData), data(1))
			Expect(born(node)).To(BeFalse())
		})
	})

	Describe("Reset", func() {
		It("Should clear the birth of every edge node", func() {
			handle(
				nodeTopic(sparkplug.NBirth),
				birth(0, 0, declared("temp", 1, sparkplug.Double, doubleValue(0))),
			)
			handle(
				deviceTopic(sparkplug.DBirth, "pump"),
				data(1, declared("speed", 1, sparkplug.Int32, intValue(0))),
			)
			handle(
				sparkplug.Topic{Type: sparkplug.NBirth, Node: other},
				birth(0, 0),
			)
			h.Reset()
			Expect(born(node)).To(BeFalse())
			Expect(born(other)).To(BeFalse())
			Expect(handle(
				nodeTopic(sparkplug.NData),
				data(2, byAlias(1, doubleValue(1))),
			)).To(Equal(sparkplug.Event{Node: node, Rebirth: true}))
			Expect(handle(
				deviceTopic(sparkplug.DData, "pump"),
				data(3, byAlias(1, intValue(1))),
			)).To(Equal(sparkplug.Event{Node: node, Device: "pump", Rebirth: true}))
			Expect(handle(
				sparkplug.Topic{Type: sparkplug.NData, Node: other},
				data(1),
			)).To(Equal(sparkplug.Event{Node: other, Rebirth: true}))
		})
		It("Should leave a rebirth request for every edge node", func() {
			handle(nodeTopic(sparkplug.NBirth), birth(0, 0))
			handle(sparkplug.Topic{Type: sparkplug.NBirth, Node: other}, birth(0, 0))
			h.Reset()
			due, _ := h.TakeRebirths(time.Now())
			Expect(due).To(ConsistOf(node, other))
		})
		It("Should accept a new birth after the reset", func() {
			handle(nodeTopic(sparkplug.NBirth), birth(0, 0))
			h.Reset()
			handle(
				nodeTopic(sparkplug.NBirth),
				birth(0, 1, declared("temp", 1, sparkplug.Double, doubleValue(0))),
			)
			Expect(born(node)).To(BeTrue())
			ev := handle(
				nodeTopic(sparkplug.NData),
				data(1, byAlias(1, doubleValue(1))),
			)
			Expect(ev.Type).To(Equal(sparkplug.NData))
			Expect(ev.Metrics).To(HaveLen(1))
		})
		It("Should do nothing on a host that knows no edge node", func() {
			h.Reset()
			Expect(born(node)).To(BeFalse())
		})
	})

	Describe("Forget", func() {
		It("Should drop the state of the node and keep the others", func() {
			handle(
				nodeTopic(sparkplug.NBirth),
				birth(0, 0, declared("temp", 1, sparkplug.Double, doubleValue(0))),
			)
			handle(
				sparkplug.Topic{Type: sparkplug.NBirth, Node: other},
				birth(0, 0),
			)
			h.Forget(node)
			Expect(born(node)).To(BeFalse())
			Expect(born(other)).To(BeTrue())
			Expect(handle(
				nodeTopic(sparkplug.NData),
				data(1, byAlias(1, doubleValue(1))),
			)).To(Equal(sparkplug.Event{Node: node, Rebirth: true}))
		})
		It("Should drop the bdSeq of the node", func() {
			handle(nodeTopic(sparkplug.NBirth), birth(0, 4))
			h.Forget(node)
			Expect(handle(nodeTopic(sparkplug.NDeath), death(3))).To(
				Equal(sparkplug.Event{Type: sparkplug.NDeath, Node: node}),
			)
		})
		It("Should do nothing for a node that the host never saw", func() {
			h.Forget(node)
			Expect(born(node)).To(BeFalse())
		})
	})

	Describe("TakeRebirths", func() {
		now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)

		It("Should hold no request on a new host", func() {
			due, wait := h.TakeRebirths(now)
			Expect(due).To(BeEmpty())
			Expect(wait).To(BeZero())
		})
		It("Should return a wanted node once", func() {
			h.WantRebirth(node)
			due, wait := h.TakeRebirths(now)
			Expect(due).To(Equal([]sparkplug.NodeID{node}))
			Expect(wait).To(BeZero())
			due, _ = h.TakeRebirths(now)
			Expect(due).To(BeEmpty())
		})
		It("Should hold a request until the rebirth interval has passed", func() {
			h.WantRebirth(node)
			h.TakeRebirths(now)
			h.WantRebirth(node)
			due, wait := h.TakeRebirths(now.Add(2 * time.Second))
			Expect(due).To(BeEmpty())
			Expect(wait).To(Equal(3 * time.Second))
			due, _ = h.TakeRebirths(now.Add(5 * time.Second))
			Expect(due).To(Equal([]sparkplug.NodeID{node}))
		})
		It("Should wait for the held request that comes due first", func() {
			h.WantRebirth(node)
			h.TakeRebirths(now)
			h.WantRebirth(other)
			h.TakeRebirths(now.Add(time.Second))
			h.WantRebirth(node)
			h.WantRebirth(other)
			due, wait := h.TakeRebirths(now.Add(2 * time.Second))
			Expect(due).To(BeEmpty())
			Expect(wait).To(Equal(3 * time.Second))
		})
		It("Should use the rebirth interval of the host", func() {
			h = sparkplug.NewHost(time.Minute)
			h.WantRebirth(node)
			h.TakeRebirths(now)
			h.WantRebirth(node)
			_, wait := h.TakeRebirths(now.Add(10 * time.Second))
			Expect(wait).To(Equal(50 * time.Second))
		})
		It("Should keep the interval across a Reset", func() {
			h.WantRebirth(node)
			h.TakeRebirths(now)
			h.Reset()
			due, wait := h.TakeRebirths(now.Add(time.Second))
			Expect(due).To(BeEmpty())
			Expect(wait).To(Equal(4 * time.Second))
		})
		It("Should hold a request for data before a birth", func() {
			handle(nodeTopic(sparkplug.NData), data(1))
			due, _ := h.TakeRebirths(now)
			Expect(due).To(Equal([]sparkplug.NodeID{node}))
		})
		It("Should hold a request for DDATA of a device with no DBIRTH",
			func() {
				handle(nodeTopic(sparkplug.NBirth), birth(0, 0))
				handle(
					deviceTopic(sparkplug.DData, "pump"),
					data(1, byAlias(1, doubleValue(1))),
				)
				due, _ := h.TakeRebirths(now)
				Expect(due).To(Equal([]sparkplug.NodeID{node}))
			},
		)
		It("Should drop a request that a birth answers", func() {
			h.WantRebirth(node)
			handle(nodeTopic(sparkplug.NBirth), birth(0, 0))
			due, _ := h.TakeRebirths(now)
			Expect(due).To(BeEmpty())
		})
		It("Should drop the request of a forgotten node", func() {
			h.WantRebirth(node)
			h.Forget(node)
			due, _ := h.TakeRebirths(now)
			Expect(due).To(BeEmpty())
		})
	})
})
