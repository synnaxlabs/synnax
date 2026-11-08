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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug"
	"github.com/synnaxlabs/synnax/pkg/service/mqtt/sparkplug/pb"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// nan is the NaN with no payload. A float32 cannot keep the payload of math.NaN.
var nan = math.Float64frombits(0x7FF8000000000000)

var _ = Describe("Value", func() {
	const now = 1700000000123 * telem.MillisecondTS
	node := sparkplug.NodeID{Group: "plant", EdgeNode: "line1"}

	Describe("Compatible", func() {
		tagTypes := []sparkplug.DataType{
			sparkplug.Int8, sparkplug.Int16, sparkplug.Int32, sparkplug.Int64,
			sparkplug.UInt8, sparkplug.UInt16, sparkplug.UInt32, sparkplug.UInt64,
			sparkplug.Float, sparkplug.Double, sparkplug.Boolean, sparkplug.String,
			sparkplug.DateTime, sparkplug.Text, sparkplug.UUID,
			sparkplug.DataType(pb.DataType_DataSet),
			sparkplug.DataType(pb.DataType_Bytes),
			sparkplug.DataType(pb.DataType_Unknown),
		}
		numeric := []sparkplug.DataType{
			sparkplug.Int8, sparkplug.Int16, sparkplug.Int32, sparkplug.Int64,
			sparkplug.UInt8, sparkplug.UInt16, sparkplug.UInt32, sparkplug.UInt64,
			sparkplug.Float, sparkplug.Double, sparkplug.Boolean,
		}
		DescribeTable("Should match a channel type to these tag types only",
			func(ch telem.DataType, expected []sparkplug.DataType) {
				matched := make([]sparkplug.DataType, 0, len(expected))
				for _, t := range tagTypes {
					if sparkplug.Compatible(ch, t) {
						matched = append(matched, t)
					}
				}
				Expect(matched).To(ConsistOf(expected))
			},
			Entry("int8", telem.Int8T, numeric),
			Entry("int16", telem.Int16T, numeric),
			Entry("int32", telem.Int32T, numeric),
			Entry("int64", telem.Int64T, numeric),
			Entry("uint8", telem.Uint8T, numeric),
			Entry("uint16", telem.Uint16T, numeric),
			Entry("uint32", telem.Uint32T, numeric),
			Entry("uint64", telem.Uint64T, numeric),
			Entry("float32", telem.Float32T, numeric),
			Entry("float64", telem.Float64T, numeric),
			Entry("string", telem.StringT, []sparkplug.DataType{
				sparkplug.String, sparkplug.Text, sparkplug.UUID,
			}),
			Entry("timestamp", telem.TimestampT, []sparkplug.DataType{
				sparkplug.DateTime,
			}),
			Entry("boolean", telem.BooleanT, []sparkplug.DataType{}),
			Entry("uuid", telem.UUIDT, []sparkplug.DataType{}),
			Entry("json", telem.JSONT, []sparkplug.DataType{}),
			Entry("bytes", telem.BytesT, []sparkplug.DataType{}),
		)
	})

	Describe("Round trips", func() {
		// publish sends sample 0 of in as tag through an edge node and a host, and
		// returns the value that the host reads.
		publish := func(tag sparkplug.DataType, in telem.Series) sparkplug.Value {
			GinkgoHelper()
			var (
				e = sparkplug.NewEdge(
					node,
					[]sparkplug.Tag{{Name: "tag", DataType: tag}},
				)
				s = sparkplug.NewSequence(e)
				h = sparkplug.NewHost(sparkplug.DefaultRebirthInterval)
			)
			receive := func(pub sparkplug.Publication) sparkplug.Event {
				GinkgoHelper()
				topic := MustSucceed(sparkplug.ParseTopic(pub.Topic))
				return h.Handle(MustSucceed(sparkplug.Decode(topic, pub.Payload)))
			}
			receive(MustSucceed(s.Birth(nil, now)))
			pub, _ := MustSucceed2(s.Data(
				[]sparkplug.Metric{{Name: "tag", Value: sparkplug.ValueAt(in, 0)}},
				now,
			))
			ev := receive(pub)
			Expect(ev.Metrics).To(HaveLen(1))
			Expect(ev.Metrics[0].DataType).To(Equal(tag))
			return ev.Metrics[0].Value
		}

		// command sends sample 0 of in as a command for tag, and returns the value that
		// the edge node reads.
		command := func(tag sparkplug.DataType, in telem.Series) sparkplug.Value {
			GinkgoHelper()
			e := sparkplug.NewEdge(node, []sparkplug.Tag{{Name: "tag", DataType: tag}})
			pub := MustSucceed(sparkplug.EncodeCommand(node, "", sparkplug.Metric{
				Name:     "tag",
				DataType: tag,
				Value:    sparkplug.ValueAt(in, 0),
			}, now))
			topic := MustSucceed(sparkplug.ParseTopic(pub.Topic))
			cmd := e.Command(MustSucceed(sparkplug.Decode(topic, pub.Payload)))
			Expect(cmd.Metrics).To(HaveLen(1))
			return cmd.Metrics[0].Value
		}

		DescribeTable(
			"Should give back the sample both ways through a tag",
			func(tag sparkplug.DataType, in telem.Series) {
				Expect(sparkplug.Compatible(in.DataType, tag)).To(BeTrue())
				for _, v := range []sparkplug.Value{
					publish(tag, in),
					command(tag, in),
				} {
					Expect(sparkplug.AppendSample(nil, in.DataType, v)).
						To(Equal(in.Data))
				}
			},
			Entry("Int8", sparkplug.Int8, telem.NewSeriesV[int8](math.MinInt8)),
			Entry("Int16", sparkplug.Int16, telem.NewSeriesV[int16](math.MinInt16)),
			Entry("Int32", sparkplug.Int32, telem.NewSeriesV[int32](math.MinInt32)),
			Entry("Int64", sparkplug.Int64, telem.NewSeriesV[int64](math.MinInt64)),
			Entry("UInt8", sparkplug.UInt8, telem.NewSeriesV[uint8](math.MaxUint8)),
			Entry("UInt16", sparkplug.UInt16, telem.NewSeriesV[uint16](math.MaxUint16)),
			Entry("UInt32", sparkplug.UInt32, telem.NewSeriesV[uint32](math.MaxUint32)),
			Entry("UInt64", sparkplug.UInt64, telem.NewSeriesV[uint64](math.MaxUint64)),
			Entry("Float", sparkplug.Float, telem.NewSeriesV[float32](1.5)),
			Entry("Double", sparkplug.Double, telem.NewSeriesV(2.25)),
			Entry("Boolean", sparkplug.Boolean, telem.NewSeriesV[uint8](1)),
			Entry("String", sparkplug.String, telem.NewSeriesV("open")),
			Entry("DateTime",
				sparkplug.DateTime,
				telem.NewSeriesV(1700000000456*telem.MillisecondTS),
			),
			Entry("NaN as Float from float32",
				sparkplug.Float, telem.NewSeriesV(float32(nan)),
			),
			Entry("+Inf as Float from float32",
				sparkplug.Float, telem.NewSeriesV(float32(math.Inf(1))),
			),
			Entry("-Inf as Float from float32",
				sparkplug.Float, telem.NewSeriesV(float32(math.Inf(-1))),
			),
			Entry("NaN as Float from float64", sparkplug.Float, telem.NewSeriesV(nan)),
			Entry("+Inf as Float from float64",
				sparkplug.Float, telem.NewSeriesV(math.Inf(1)),
			),
			Entry("-Inf as Float from float64",
				sparkplug.Float, telem.NewSeriesV(math.Inf(-1)),
			),
			Entry("NaN as Double from float32",
				sparkplug.Double, telem.NewSeriesV(float32(nan)),
			),
			Entry("+Inf as Double from float32",
				sparkplug.Double, telem.NewSeriesV(float32(math.Inf(1))),
			),
			Entry("-Inf as Double from float32",
				sparkplug.Double, telem.NewSeriesV(float32(math.Inf(-1))),
			),
			Entry(
				"NaN as Double from float64",
				sparkplug.Double,
				telem.NewSeriesV(nan),
			),
			Entry("+Inf as Double from float64",
				sparkplug.Double, telem.NewSeriesV(math.Inf(1)),
			),
			Entry("-Inf as Double from float64",
				sparkplug.Double, telem.NewSeriesV(math.Inf(-1)),
			),
		)
	})

	Describe("ValueAt", func() {
		DescribeTable(
			"Should return the sample as the Value of its family",
			func(in telem.Series, expected sparkplug.Value) {
				Expect(sparkplug.ValueAt(in, 1)).To(Equal(expected))
			},
			Entry("int8", telem.NewSeriesV[int8](0, -3), sparkplug.IntValue(-3)),
			Entry("int16", telem.NewSeriesV[int16](0, -3), sparkplug.IntValue(-3)),
			Entry("int32", telem.NewSeriesV[int32](0, -3), sparkplug.IntValue(-3)),
			Entry("int64", telem.NewSeriesV[int64](0, -3), sparkplug.IntValue(-3)),
			Entry("uint8", telem.NewSeriesV[uint8](0, 3), sparkplug.UintValue(3)),
			Entry("uint16", telem.NewSeriesV[uint16](0, 3), sparkplug.UintValue(3)),
			Entry("uint32", telem.NewSeriesV[uint32](0, 3), sparkplug.UintValue(3)),
			Entry("uint64", telem.NewSeriesV[uint64](0, 3), sparkplug.UintValue(3)),
			Entry(
				"float32",
				telem.NewSeriesV[float32](0, 1.5),
				sparkplug.FloatValue(1.5),
			),
			Entry("float64", telem.NewSeriesV(0, 2.5), sparkplug.FloatValue(2.5)),
			Entry("string", telem.NewSeriesV("a", "b"), sparkplug.StringValue("b")),
			Entry("timestamp",
				telem.NewSeriesV[telem.TimeStamp](0, telem.SecondTS),
				sparkplug.DateTimeValue(telem.SecondTS),
			),
		)
		DescribeTable("Should panic for a data type that matches no tag",
			func(dt telem.DataType) {
				Expect(func() { sparkplug.ValueAt(telem.Series{DataType: dt}, 0) }).
					To(PanicWith("sparkplug: no tag takes samples of " + dt))
			},
			Entry("boolean", telem.BooleanT),
			Entry("uuid", telem.UUIDT),
			Entry("json", telem.JSONT),
			Entry("bytes", telem.BytesT),
		)
	})

	Describe("AppendSample", func() {
		It("Should append to dst", func() {
			dst := []byte{9}
			Expect(sparkplug.AppendSample(dst, telem.Uint8T, sparkplug.UintValue(4))).
				To(Equal([]byte{9, 4}))
		})
		DescribeTable("Should convert a number of another family when it fits",
			func(dt telem.DataType, v sparkplug.Value, expected telem.Series) {
				Expect(sparkplug.AppendSample(nil, dt, v)).To(Equal(expected.Data))
			},
			Entry("an IntValue into float32",
				telem.Float32T, sparkplug.IntValue(-2), telem.NewSeriesV[float32](-2),
			),
			Entry("a UintValue into int64",
				telem.Int64T, sparkplug.UintValue(7), telem.NewSeriesV[int64](7),
			),
			Entry("a whole FloatValue into int16",
				telem.Int16T, sparkplug.FloatValue(-300), telem.NewSeriesV[int16](-300),
			),
			Entry("a BoolValue into float64",
				telem.Float64T, sparkplug.BoolValue(true), telem.NewSeriesV(1.0),
			),
			Entry("a false BoolValue into uint8",
				telem.Uint8T, sparkplug.BoolValue(false), telem.NewSeriesV[uint8](0),
			),
		)
		DescribeTable("Should reject a value that the channel cannot hold",
			func(dt telem.DataType, v sparkplug.Value, message string) {
				Expect(sparkplug.AppendSample(nil, dt, v)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
					MatchError(ContainSubstring("value does not fit the data type")),
				))
			},
			Entry("300 into int8", telem.Int8T, sparkplug.IntValue(300), "300 as int8"),
			Entry("-1 into uint8", telem.Uint8T, sparkplug.IntValue(-1), "-1 as uint8"),
			Entry("a fraction into int32",
				telem.Int32T, sparkplug.FloatValue(1.5), "1.5 as int32",
			),
			Entry("NaN into int64",
				telem.Int64T, sparkplug.FloatValue(math.NaN()), "NaN as int64",
			),
			Entry("+Inf into uint64",
				telem.Uint64T, sparkplug.FloatValue(math.Inf(1)), "+Inf as uint64",
			),
			Entry("a finite number past the float32 range",
				telem.Float32T, sparkplug.FloatValue(1e300), "1e+300 as float32",
			),
			Entry("a StringValue into float64",
				telem.Float64T, sparkplug.StringValue("2.5"), "2.5 as float64",
			),
			Entry("a DateTimeValue into int64",
				telem.Int64T, sparkplug.DateTimeValue(telem.SecondTS), " as int64",
			),
			Entry("an IntValue into string",
				telem.StringT, sparkplug.IntValue(5), "5 as string",
			),
			Entry("a BoolValue into string",
				telem.StringT, sparkplug.BoolValue(true), "true as string",
			),
			Entry("an IntValue into timestamp",
				telem.TimestampT, sparkplug.IntValue(5), "5 as timestamp",
			),
			Entry("a nil value", telem.Float64T, nil, "<nil> as float64"),
			Entry("a StringValue into json",
				telem.JSONT, sparkplug.StringValue("{}"), "{} as json",
			),
			Entry("a number into boolean",
				telem.BooleanT, sparkplug.BoolValue(true), "true as boolean",
			),
		)
	})

	Describe("DateTimeValue", func() {
		It("Should print as a telem.TimeStamp", func() {
			ts := 1700000000456 * telem.MillisecondTS
			Expect(sparkplug.DateTimeValue(ts).String()).To(Equal(ts.String()))
		})
	})
})
