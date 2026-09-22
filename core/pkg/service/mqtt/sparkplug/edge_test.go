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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug/pb"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/proto"
)

// decodePayload parses a payload that an edge node built.
func decodePayload(payload []byte) *pb.Payload {
	GinkgoHelper()
	var p pb.Payload
	Expect(proto.Unmarshal(payload, &p)).To(Succeed())
	return &p
}

var _ = Describe("Edge", func() {
	const now = 1700000000123 * telem.MillisecondTS
	const nowMillis uint64 = 1700000000123

	var (
		tags = []sparkplug.Tag{
			{Name: "temp", DataType: sparkplug.Double},
			{Name: "count", DataType: sparkplug.Int32},
			{Name: "mode", DataType: sparkplug.String},
		}
		e *sparkplug.Edge
	)
	BeforeEach(func() { e = sparkplug.NewEdge(tags) })

	bdSeq := func(p *pb.Payload) *pb.Payload_Metric {
		GinkgoHelper()
		Expect(p.Metrics).ToNot(BeEmpty())
		m := p.Metrics[0]
		Expect(m.GetName()).To(Equal(sparkplug.BdSeqMetric))
		Expect(m.GetDatatype()).To(Equal(uint32(sparkplug.Int64)))
		Expect(m.Alias).To(BeNil())
		return m
	}

	Describe("NewEdge", func() {
		It("Should give each tag the alias of its position, from 1", func() {
			p := decodePayload(MustSucceed(e.Birth(nil, now)))
			Expect(p.Metrics).To(HaveLen(5))
			for i, t := range tags {
				m := p.Metrics[2+i]
				Expect(m.GetName()).To(Equal(t.Name))
				Expect(m.Alias).To(HaveValue(Equal(uint64(i + 1))))
				Expect(m.GetDatatype()).To(Equal(uint32(t.DataType)))
			}
			p = decodePayload(MustSucceed(e.Data([]sparkplug.Metric{
				{Name: "mode", Value: "auto"},
				{Name: "temp", Value: 1.5},
			}, now)))
			Expect(p.Metrics).To(HaveLen(2))
			Expect(p.Metrics[0].Alias).To(HaveValue(Equal(uint64(3))))
			Expect(p.Metrics[1].Alias).To(HaveValue(Equal(uint64(1))))
		})

		It("Should declare no tag for an edge node with none", func() {
			e = sparkplug.NewEdge(nil)
			p := decodePayload(MustSucceed(e.Birth(nil, now)))
			Expect(p.Metrics).To(HaveLen(2))
			Expect(e.Data([]sparkplug.Metric{{Name: "temp", Value: 1.0}}, now)).
				Error().To(MatchError("tag temp is not a tag of the edge node"))
		})
	})

	Describe("Death", func() {
		It("Should carry only the bdSeq of the session, with no sequence number",
			func() {
				p := decodePayload(MustSucceed(e.Death()))
				Expect(p.Seq).To(BeNil())
				Expect(p.Metrics).To(HaveLen(1))
				Expect(bdSeq(p).GetValue()).To(Equal(
					&pb.Payload_Metric_LongValue{LongValue: 0},
				))
			},
		)

		It("Should hold the same bdSeq as the birth of the session", func() {
			e.EndSession()
			e.EndSession()
			death := decodePayload(MustSucceed(e.Death()))
			birth := decodePayload(MustSucceed(e.Birth(nil, now)))
			Expect(bdSeq(death).GetLongValue()).To(Equal(uint64(2)))
			Expect(bdSeq(birth).GetLongValue()).To(Equal(uint64(2)))
		})
	})

	Describe("Birth", func() {
		It("Should start the sequence at 0 and stamp the payload", func() {
			p := decodePayload(MustSucceed(e.Birth(nil, now)))
			Expect(p.Seq).To(HaveValue(Equal(uint64(0))))
			Expect(p.Timestamp).To(HaveValue(Equal(nowMillis)))
		})

		It("Should lead with bdSeq and a false Node Control/Rebirth", func() {
			p := decodePayload(MustSucceed(e.Birth(nil, now)))
			Expect(bdSeq(p).GetValue()).To(Equal(
				&pb.Payload_Metric_LongValue{LongValue: 0},
			))
			rebirth := p.Metrics[1]
			Expect(rebirth.GetName()).To(Equal(sparkplug.RebirthMetric))
			Expect(rebirth.Alias).To(BeNil())
			Expect(rebirth.GetDatatype()).To(Equal(uint32(sparkplug.Boolean)))
			Expect(rebirth.GetValue()).To(Equal(
				&pb.Payload_Metric_BooleanValue{BooleanValue: false},
			))
		})

		It("Should declare a tag with no value as null, stamped now", func() {
			p := decodePayload(MustSucceed(e.Birth(nil, now)))
			for i, t := range tags {
				m := p.Metrics[2+i]
				Expect(m.GetName()).To(Equal(t.Name))
				Expect(m.GetIsNull()).To(BeTrue())
				Expect(m.GetValue()).To(BeNil())
				Expect(m.Timestamp).To(HaveValue(Equal(nowMillis)))
			}
		})

		It("Should carry the value and timestamp of a tag that has one", func() {
			p := decodePayload(MustSucceed(e.Birth(map[string]sparkplug.Metric{
				"count": {
					Name:      "count",
					Value:     int64(-4),
					Timestamp: now - telem.SecondTS,
				},
				"mode": {Name: "mode", Value: "auto"},
			}, now)))
			temp, count, mode := p.Metrics[2], p.Metrics[3], p.Metrics[4]
			Expect(temp.GetIsNull()).To(BeTrue())
			Expect(count.IsNull).To(BeNil())
			Expect(count.GetValue()).To(Equal(
				&pb.Payload_Metric_IntValue{IntValue: 0xFFFFFFFC},
			))
			Expect(count.Timestamp).To(HaveValue(Equal(nowMillis - 1000)))
			Expect(mode.IsNull).To(BeNil())
			Expect(mode.GetValue()).To(Equal(
				&pb.Payload_Metric_StringValue{StringValue: "auto"},
			))
			// A value with no timestamp takes the time of the birth.
			Expect(mode.Timestamp).To(HaveValue(Equal(nowMillis)))
		})

		It("Should encode a value as the data type of its tag", func() {
			p := decodePayload(MustSucceed(e.Birth(map[string]sparkplug.Metric{
				"temp": {Name: "temp", Value: int64(3), DataType: sparkplug.Int32},
			}, now)))
			Expect(p.Metrics[2].GetDatatype()).To(Equal(uint32(sparkplug.Double)))
			Expect(p.Metrics[2].GetValue()).To(Equal(
				&pb.Payload_Metric_DoubleValue{DoubleValue: 3},
			))
		})

		It("Should ignore a value of a tag that the edge node does not declare",
			func() {
				p := decodePayload(MustSucceed(e.Birth(map[string]sparkplug.Metric{
					"other": {Name: "other", Value: 1.0},
				}, now)))
				Expect(p.Metrics).To(HaveLen(5))
			},
		)

		It("Should reject a value that does not fit the data type of its tag",
			func() {
				Expect(e.Birth(map[string]sparkplug.Metric{
					"count": {Name: "count", Value: "twelve"},
				}, now)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"tag count: twelve as Int32: value does not fit the data type",
					)),
				))
			},
		)

		It("Should reset the sequence of the session", func() {
			MustSucceed(e.Birth(nil, now))
			MustSucceed(e.Data(nil, now))
			p := decodePayload(MustSucceed(e.Data(nil, now)))
			Expect(p.Seq).To(HaveValue(Equal(uint64(2))))
			p = decodePayload(MustSucceed(e.Birth(nil, now)))
			Expect(p.Seq).To(HaveValue(Equal(uint64(0))))
			p = decodePayload(MustSucceed(e.Data(nil, now)))
			Expect(p.Seq).To(HaveValue(Equal(uint64(1))))
		})
	})

	Describe("Data", func() {
		BeforeEach(func() { MustSucceed(e.Birth(nil, now)) })

		It("Should identify each tag by its alias only", func() {
			p := decodePayload(MustSucceed(e.Data([]sparkplug.Metric{
				{Name: "count", Value: int64(7)},
				{Name: "temp", Value: 21.5},
			}, now)))
			Expect(p.Metrics).To(HaveLen(2))
			Expect(p.Metrics[0].Name).To(BeNil())
			Expect(p.Metrics[0].Alias).To(HaveValue(Equal(uint64(2))))
			Expect(p.Metrics[0].GetDatatype()).To(Equal(uint32(sparkplug.Int32)))
			Expect(p.Metrics[0].GetValue()).To(Equal(
				&pb.Payload_Metric_IntValue{IntValue: 7},
			))
			Expect(p.Metrics[1].Name).To(BeNil())
			Expect(p.Metrics[1].Alias).To(HaveValue(Equal(uint64(1))))
			Expect(p.Metrics[1].GetDatatype()).To(Equal(uint32(sparkplug.Double)))
			Expect(p.Metrics[1].GetValue()).To(Equal(
				&pb.Payload_Metric_DoubleValue{DoubleValue: 21.5},
			))
		})

		It("Should encode a value as the data type of its tag", func() {
			p := decodePayload(MustSucceed(e.Data([]sparkplug.Metric{
				{Name: "count", Value: 3.0, DataType: sparkplug.Double},
			}, now)))
			Expect(p.Metrics[0].GetDatatype()).To(Equal(uint32(sparkplug.Int32)))
			Expect(p.Metrics[0].GetValue()).To(Equal(
				&pb.Payload_Metric_IntValue{IntValue: 3},
			))
		})

		It("Should stamp a metric with its own timestamp or with now", func() {
			p := decodePayload(MustSucceed(e.Data([]sparkplug.Metric{
				{Name: "temp", Value: 1.0, Timestamp: now + 250*telem.MillisecondTS},
				{Name: "temp", Value: 2.0},
			}, now)))
			Expect(p.Timestamp).To(HaveValue(Equal(nowMillis)))
			Expect(p.Metrics[0].Timestamp).To(HaveValue(Equal(nowMillis + 250)))
			Expect(p.Metrics[1].Timestamp).To(HaveValue(Equal(nowMillis)))
		})

		It("Should count the sequence up from the birth", func() {
			for seq := uint64(1); seq <= 3; seq++ {
				p := decodePayload(MustSucceed(e.Data(nil, now)))
				Expect(p.Seq).To(HaveValue(Equal(seq)))
			}
		})

		It("Should wrap the sequence from 255 to 0", func() {
			for range 254 {
				MustSucceed(e.Data(nil, now))
			}
			for _, seq := range []uint64{255, 0, 1} {
				p := decodePayload(MustSucceed(e.Data(nil, now)))
				Expect(p.Seq).To(HaveValue(Equal(seq)))
			}
		})

		It("Should reject a tag that the edge node does not declare", func() {
			Expect(e.Data([]sparkplug.Metric{
				{Name: "temp", Value: 1.0},
				{Name: "flow", Value: 1.0},
			}, now)).Error().To(MatchError("tag flow is not a tag of the edge node"))
		})

		It("Should reject a value that does not fit the data type of its tag",
			func() {
				Expect(e.Data(
					[]sparkplug.Metric{{Name: "count", Value: int64(1) << 40}},
					now,
				)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"tag count: 1099511627776 as Int32: "+
							"value does not fit the data type",
					)),
				))
			},
		)

		It("Should not use up a sequence number on an error", func() {
			Expect(e.Data([]sparkplug.Metric{{Name: "flow", Value: 1.0}}, now)).
				Error().To(MatchError(ContainSubstring("is not a tag")))
			p := decodePayload(MustSucceed(e.Data(nil, now)))
			Expect(p.Seq).To(HaveValue(Equal(uint64(1))))
		})
	})

	Describe("EndSession", func() {
		It("Should move the next death and birth to the next bdSeq", func() {
			e.EndSession()
			Expect(bdSeq(decodePayload(MustSucceed(e.Death()))).GetLongValue()).
				To(Equal(uint64(1)))
			Expect(bdSeq(decodePayload(MustSucceed(e.Birth(nil, now)))).GetLongValue()).
				To(Equal(uint64(1)))
			e.EndSession()
			Expect(bdSeq(decodePayload(MustSucceed(e.Death()))).GetLongValue()).
				To(Equal(uint64(2)))
		})

		It("Should wrap the bdSeq from 255 to 0", func() {
			for range 255 {
				e.EndSession()
			}
			Expect(bdSeq(decodePayload(MustSucceed(e.Death()))).GetLongValue()).
				To(Equal(uint64(255)))
			e.EndSession()
			Expect(bdSeq(decodePayload(MustSucceed(e.Death()))).GetLongValue()).
				To(Equal(uint64(0)))
		})

		It("Should leave the message sequence alone", func() {
			MustSucceed(e.Birth(nil, now))
			MustSucceed(e.Data(nil, now))
			e.EndSession()
			p := decodePayload(MustSucceed(e.Data(nil, now)))
			Expect(p.Seq).To(HaveValue(Equal(uint64(2))))
		})
	})

	Describe("DecodeCommand", func() {
		It("Should report a rebirth request", func() {
			cmd := MustSucceed(
				e.DecodeCommand(MustSucceed(sparkplug.EncodeRebirth(now))),
			)
			Expect(cmd).To(Equal(sparkplug.Command{Rebirth: true}))
		})

		It("Should not report a rebirth request that is false", func() {
			cmd := MustSucceed(e.DecodeCommand(encode(&pb.Payload{
				Timestamp: new(nowMillis),
				Metrics: []*pb.Payload_Metric{
					declared(
						sparkplug.RebirthMetric,
						0,
						sparkplug.Boolean,
						boolValue(false),
					),
				},
			})))
			Expect(cmd).To(Equal(sparkplug.Command{}))
		})

		It("Should resolve a metric by its name", func() {
			cmd := MustSucceed(e.DecodeCommand(MustSucceed(sparkplug.EncodeCommand(
				sparkplug.Metric{Name: "temp", DataType: sparkplug.Double, Value: 2.5},
				now,
			))))
			Expect(cmd).To(Equal(sparkplug.Command{Metrics: []sparkplug.Metric{{
				Name:      "temp",
				DataType:  sparkplug.Double,
				Value:     2.5,
				Timestamp: now,
			}}}))
		})

		It("Should resolve a metric by its alias", func() {
			m := byAlias(2, intValue(0xFFFFFFF6))
			m.Datatype = new(uint32(sparkplug.Int32))
			cmd := MustSucceed(e.DecodeCommand(encode(&pb.Payload{
				Timestamp: new(nowMillis),
				Metrics:   []*pb.Payload_Metric{m},
			})))
			Expect(cmd).To(Equal(sparkplug.Command{Metrics: []sparkplug.Metric{{
				Name:      "count",
				DataType:  sparkplug.Int32,
				Value:     int64(-10),
				Timestamp: now,
			}}}))
		})

		It("Should resolve an empty name through the alias", func() {
			m := byAlias(3, stringValue("manual"))
			m.Name = new("")
			cmd := MustSucceed(e.DecodeCommand(encode(&pb.Payload{
				Timestamp: new(nowMillis),
				Metrics:   []*pb.Payload_Metric{m},
			})))
			Expect(cmd.Metrics).To(HaveLen(1))
			Expect(cmd.Metrics[0].Name).To(Equal("mode"))
			Expect(cmd.Metrics[0].Value).To(Equal("manual"))
		})

		It("Should take the data type of the tag when the command gives none", func() {
			cmd := MustSucceed(e.DecodeCommand(encode(&pb.Payload{
				Timestamp: new(nowMillis),
				Metrics: []*pb.Payload_Metric{
					byName("count", intValue(12)),
					byAlias(1, doubleValue(0.5)),
				},
			})))
			Expect(cmd.Metrics).To(Equal([]sparkplug.Metric{
				{
					Name:      "count",
					DataType:  sparkplug.Int32,
					Value:     int64(12),
					Timestamp: now,
				},
				{Name: "temp", DataType: sparkplug.Double, Value: 0.5, Timestamp: now},
			}))
		})

		It("Should take the data type of the command when it gives one", func() {
			m := byName("count", intValue(0xFFFF))
			m.Datatype = new(uint32(sparkplug.UInt16))
			cmd := MustSucceed(e.DecodeCommand(encode(&pb.Payload{
				Metrics: []*pb.Payload_Metric{m},
			})))
			Expect(cmd.Metrics).To(Equal([]sparkplug.Metric{{
				Name: "count", DataType: sparkplug.UInt16, Value: uint64(0xFFFF),
			}}))
		})

		It("Should give a nil value for a null metric", func() {
			cmd := MustSucceed(e.DecodeCommand(encode(&pb.Payload{
				Metrics: []*pb.Payload_Metric{byName("temp", nullValue())},
			})))
			Expect(cmd.Metrics).To(HaveLen(1))
			Expect(cmd.Metrics[0].Value).To(BeNil())
		})

		It("Should use the timestamp of the metric over that of the payload", func() {
			m := byName("temp", doubleValue(1))
			m.Timestamp = new(nowMillis + 500)
			cmd := MustSucceed(e.DecodeCommand(encode(&pb.Payload{
				Timestamp: new(nowMillis),
				Metrics:   []*pb.Payload_Metric{m, byName("temp", doubleValue(2))},
			})))
			Expect(cmd.Metrics).To(HaveLen(2))
			Expect(cmd.Metrics[0].Timestamp).To(Equal(now + 500*telem.MillisecondTS))
			Expect(cmd.Metrics[1].Timestamp).To(Equal(now))
		})

		It("Should give a zero timestamp when the command has none", func() {
			cmd := MustSucceed(e.DecodeCommand(encode(&pb.Payload{
				Metrics: []*pb.Payload_Metric{byName("temp", doubleValue(1))},
			})))
			Expect(cmd.Metrics).To(HaveLen(1))
			Expect(cmd.Metrics[0].Timestamp).To(BeZero())
		})

		It("Should drop a metric of a tag it does not declare and keep the rest",
			func() {
				cmd := MustSucceed(e.DecodeCommand(encode(&pb.Payload{
					Timestamp: new(nowMillis),
					Metrics: []*pb.Payload_Metric{
						byName("flow", doubleValue(1)),
						byAlias(99, doubleValue(1)),
						doubleValue(1),
						byName("temp", doubleValue(4)),
					},
				})))
				Expect(cmd.Rebirth).To(BeFalse())
				Expect(cmd.Metrics).To(Equal([]sparkplug.Metric{{
					Name:      "temp",
					DataType:  sparkplug.Double,
					Value:     4.0,
					Timestamp: now,
				}}))
			},
		)

		It("Should drop a metric whose name is unknown even with a known alias",
			func() {
				m := byAlias(1, doubleValue(1))
				m.Name = new("flow")
				cmd := MustSucceed(e.DecodeCommand(encode(&pb.Payload{
					Metrics: []*pb.Payload_Metric{m},
				})))
				Expect(cmd.Metrics).To(BeEmpty())
			},
		)

		It("Should report a rebirth request next to the metrics", func() {
			cmd := MustSucceed(e.DecodeCommand(encode(&pb.Payload{
				Timestamp: new(nowMillis),
				Metrics: []*pb.Payload_Metric{
					byName("mode", stringValue("auto")),
					declared(
						sparkplug.RebirthMetric,
						0,
						sparkplug.Boolean,
						boolValue(true),
					),
				},
			})))
			Expect(cmd.Rebirth).To(BeTrue())
			Expect(cmd.Metrics).To(HaveLen(1))
			Expect(cmd.Metrics[0].Name).To(Equal("mode"))
		})

		It("Should reject a payload that does not decode", func() {
			Expect(e.DecodeCommand([]byte{0xFF, 0xFF, 0xFF})).Error().
				To(MatchError(ContainSubstring("invalid command payload")))
		})

		It("Should give an empty command for an empty payload", func() {
			Expect(e.DecodeCommand(nil)).To(Equal(sparkplug.Command{}))
		})
	})
})
