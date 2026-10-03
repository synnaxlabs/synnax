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
	"google.golang.org/protobuf/proto"
)

// decodeCommand returns the payload of a command and its only metric.
func decodeCommand(payload []byte) (*pb.Payload, *pb.Payload_Metric) {
	GinkgoHelper()
	var p pb.Payload
	Expect(proto.Unmarshal(payload, &p)).To(Succeed())
	Expect(p.Metrics).To(HaveLen(1))
	return &p, p.Metrics[0]
}

var _ = Describe("Metric", func() {
	const now = 1700000000123 * telem.MillisecondTS

	Describe("DataType", func() {
		DescribeTable("String should return the name in the specification",
			func(dt sparkplug.DataType, expected string) {
				Expect(dt.String()).To(Equal(expected))
			},
			Entry("Int8", sparkplug.Int8, "Int8"),
			Entry("Int16", sparkplug.Int16, "Int16"),
			Entry("Int32", sparkplug.Int32, "Int32"),
			Entry("Int64", sparkplug.Int64, "Int64"),
			Entry("UInt8", sparkplug.UInt8, "UInt8"),
			Entry("UInt16", sparkplug.UInt16, "UInt16"),
			Entry("UInt32", sparkplug.UInt32, "UInt32"),
			Entry("UInt64", sparkplug.UInt64, "UInt64"),
			Entry("Float", sparkplug.Float, "Float"),
			Entry("Double", sparkplug.Double, "Double"),
			Entry("Boolean", sparkplug.Boolean, "Boolean"),
			Entry("String", sparkplug.String, "String"),
			Entry("DateTime", sparkplug.DateTime, "DateTime"),
			Entry("Text", sparkplug.Text, "Text"),
			Entry("UUID", sparkplug.UUID, "UUID"),
			Entry("a type with no Go form", sparkplug.DataType(16), "DataSet"),
			Entry("a type outside the specification", sparkplug.DataType(99), "99"),
		)

		DescribeTable(
			"Supported should report whether a Metric can hold the type",
			func(dt sparkplug.DataType, expected bool) {
				Expect(dt.Supported()).To(Equal(expected))
			},
			Entry("Int8", sparkplug.Int8, true),
			Entry("Int16", sparkplug.Int16, true),
			Entry("Int32", sparkplug.Int32, true),
			Entry("Int64", sparkplug.Int64, true),
			Entry("UInt8", sparkplug.UInt8, true),
			Entry("UInt16", sparkplug.UInt16, true),
			Entry("UInt32", sparkplug.UInt32, true),
			Entry("UInt64", sparkplug.UInt64, true),
			Entry("Float", sparkplug.Float, true),
			Entry("Double", sparkplug.Double, true),
			Entry("Boolean", sparkplug.Boolean, true),
			Entry("String", sparkplug.String, true),
			Entry("DateTime", sparkplug.DateTime, true),
			Entry("Text", sparkplug.Text, true),
			Entry("UUID", sparkplug.UUID, true),
			Entry("Unknown", sparkplug.DataType(pb.DataType_Unknown), false),
			Entry("DataSet", sparkplug.DataType(pb.DataType_DataSet), false),
			Entry("Bytes", sparkplug.DataType(pb.DataType_Bytes), false),
			Entry("File", sparkplug.DataType(pb.DataType_File), false),
			Entry("Template", sparkplug.DataType(pb.DataType_Template), false),
			Entry("PropertySet", sparkplug.DataType(pb.DataType_PropertySet), false),
			Entry(
				"PropertySetList",
				sparkplug.DataType(pb.DataType_PropertySetList),
				false,
			),
			Entry("Int8Array", sparkplug.DataType(pb.DataType_Int8Array), false),
			Entry("Int16Array", sparkplug.DataType(pb.DataType_Int16Array), false),
			Entry("Int32Array", sparkplug.DataType(pb.DataType_Int32Array), false),
			Entry("Int64Array", sparkplug.DataType(pb.DataType_Int64Array), false),
			Entry("UInt8Array", sparkplug.DataType(pb.DataType_UInt8Array), false),
			Entry("UInt16Array", sparkplug.DataType(pb.DataType_UInt16Array), false),
			Entry("UInt32Array", sparkplug.DataType(pb.DataType_UInt32Array), false),
			Entry("UInt64Array", sparkplug.DataType(pb.DataType_UInt64Array), false),
			Entry("FloatArray", sparkplug.DataType(pb.DataType_FloatArray), false),
			Entry("DoubleArray", sparkplug.DataType(pb.DataType_DoubleArray), false),
			Entry("BooleanArray", sparkplug.DataType(pb.DataType_BooleanArray), false),
			Entry("StringArray", sparkplug.DataType(pb.DataType_StringArray), false),
			Entry(
				"DateTimeArray",
				sparkplug.DataType(pb.DataType_DateTimeArray),
				false,
			),
			Entry("a type outside the specification", sparkplug.DataType(99), false),
		)

		DescribeTable("Should hold the type code of the specification",
			func(dt sparkplug.DataType, code uint32) {
				Expect(uint32(dt)).To(Equal(code))
			},
			Entry("Int8", sparkplug.Int8, uint32(1)),
			Entry("Int16", sparkplug.Int16, uint32(2)),
			Entry("Int32", sparkplug.Int32, uint32(3)),
			Entry("Int64", sparkplug.Int64, uint32(4)),
			Entry("UInt8", sparkplug.UInt8, uint32(5)),
			Entry("UInt16", sparkplug.UInt16, uint32(6)),
			Entry("UInt32", sparkplug.UInt32, uint32(7)),
			Entry("UInt64", sparkplug.UInt64, uint32(8)),
			Entry("Float", sparkplug.Float, uint32(9)),
			Entry("Double", sparkplug.Double, uint32(10)),
			Entry("Boolean", sparkplug.Boolean, uint32(11)),
			Entry("String", sparkplug.String, uint32(12)),
			Entry("DateTime", sparkplug.DateTime, uint32(13)),
			Entry("Text", sparkplug.Text, uint32(14)),
			Entry("UUID", sparkplug.UUID, uint32(15)),
		)
	})

	Describe("BdSeqMetric", func() {
		It("Should be the tag name in the specification", func() {
			Expect(sparkplug.BdSeqMetric).To(Equal("bdSeq"))
		})
	})

	Describe("EncodeCommand", func() {
		It("Should name the tag and carry its data type", func() {
			_, m := decodeCommand(MustSucceed(sparkplug.EncodeCommand(
				sparkplug.Metric{
					Name:     "pump/speed",
					DataType: sparkplug.Double,
					Value:    1.5,
				},
				now,
			)))
			Expect(m.GetName()).To(Equal("pump/speed"))
			Expect(m.GetDatatype()).To(Equal(uint32(pb.DataType_Double)))
			Expect(m.Alias).To(BeNil())
		})

		It("Should carry no sequence number", func() {
			p, _ := decodeCommand(MustSucceed(sparkplug.EncodeCommand(
				sparkplug.Metric{Name: "on", DataType: sparkplug.Boolean, Value: true},
				now,
			)))
			Expect(p.Seq).To(BeNil())
		})

		It("Should stamp the payload and the metric in milliseconds", func() {
			p, m := decodeCommand(MustSucceed(sparkplug.EncodeCommand(
				sparkplug.Metric{Name: "on", DataType: sparkplug.Boolean, Value: true},
				now+999_999,
			)))
			Expect(p.Timestamp).To(HaveValue(Equal(uint64(1700000000123))))
			Expect(m.Timestamp).To(HaveValue(Equal(uint64(1700000000123))))
		})

		DescribeTable("Should put the value in the wire field of the data type",
			func(dt sparkplug.DataType, value, expected any) {
				_, m := decodeCommand(MustSucceed(sparkplug.EncodeCommand(
					sparkplug.Metric{Name: "tag", DataType: dt, Value: value},
					now,
				)))
				Expect(m.GetDatatype()).To(Equal(uint32(dt)))
				Expect(m.GetValue()).To(Equal(expected))
			},
			Entry("Int8",
				sparkplug.Int8, int64(127),
				&pb.Payload_Metric_IntValue{IntValue: 127},
			),
			Entry("a negative Int8 as two's complement",
				sparkplug.Int8, int64(-128),
				&pb.Payload_Metric_IntValue{IntValue: 0xFFFFFF80},
			),
			Entry("Int16",
				sparkplug.Int16, int64(32767),
				&pb.Payload_Metric_IntValue{IntValue: 32767},
			),
			Entry("a negative Int16 as two's complement",
				sparkplug.Int16, int64(-32768),
				&pb.Payload_Metric_IntValue{IntValue: 0xFFFF8000},
			),
			Entry("Int32",
				sparkplug.Int32, int64(math.MaxInt32),
				&pb.Payload_Metric_IntValue{IntValue: math.MaxInt32},
			),
			Entry("a negative Int32 as two's complement",
				sparkplug.Int32, int64(math.MinInt32),
				&pb.Payload_Metric_IntValue{IntValue: 0x80000000},
			),
			Entry("Int64 in long_value",
				sparkplug.Int64, int64(math.MaxInt64),
				&pb.Payload_Metric_LongValue{LongValue: math.MaxInt64},
			),
			Entry("a negative Int64 as two's complement",
				sparkplug.Int64, int64(-5),
				&pb.Payload_Metric_LongValue{LongValue: 0xFFFFFFFFFFFFFFFB},
			),
			Entry("UInt8",
				sparkplug.UInt8, uint64(255),
				&pb.Payload_Metric_IntValue{IntValue: 255},
			),
			Entry("UInt16",
				sparkplug.UInt16, uint64(65535),
				&pb.Payload_Metric_IntValue{IntValue: 65535},
			),
			Entry("UInt32",
				sparkplug.UInt32, uint64(math.MaxUint32),
				&pb.Payload_Metric_IntValue{IntValue: math.MaxUint32},
			),
			Entry("UInt64 in long_value",
				sparkplug.UInt64, uint64(math.MaxUint64),
				&pb.Payload_Metric_LongValue{LongValue: math.MaxUint64},
			),
			Entry("a uint64 as Int16",
				sparkplug.Int16, uint64(12),
				&pb.Payload_Metric_IntValue{IntValue: 12},
			),
			Entry("an int64 as UInt16",
				sparkplug.UInt16, int64(12),
				&pb.Payload_Metric_IntValue{IntValue: 12},
			),
			Entry("a whole float64 as Int32",
				sparkplug.Int32, float64(-3),
				&pb.Payload_Metric_IntValue{IntValue: 0xFFFFFFFD},
			),
			Entry("a whole float64 as UInt8",
				sparkplug.UInt8, float64(200),
				&pb.Payload_Metric_IntValue{IntValue: 200},
			),
			Entry("a bool as Int8",
				sparkplug.Int8, true,
				&pb.Payload_Metric_IntValue{IntValue: 1},
			),
			Entry("a bool as UInt64",
				sparkplug.UInt64, false,
				&pb.Payload_Metric_LongValue{LongValue: 0},
			),
			Entry("Float",
				sparkplug.Float, 1.5,
				&pb.Payload_Metric_FloatValue{FloatValue: 1.5},
			),
			Entry("an int64 as Float",
				sparkplug.Float, int64(-2),
				&pb.Payload_Metric_FloatValue{FloatValue: -2},
			),
			Entry("Double",
				sparkplug.Double, 2.25,
				&pb.Payload_Metric_DoubleValue{DoubleValue: 2.25},
			),
			Entry("a uint64 as Double",
				sparkplug.Double, uint64(7),
				&pb.Payload_Metric_DoubleValue{DoubleValue: 7},
			),
			Entry("a bool as Double",
				sparkplug.Double, true,
				&pb.Payload_Metric_DoubleValue{DoubleValue: 1},
			),
			Entry("Boolean true",
				sparkplug.Boolean, true,
				&pb.Payload_Metric_BooleanValue{BooleanValue: true},
			),
			Entry("Boolean false",
				sparkplug.Boolean, false,
				&pb.Payload_Metric_BooleanValue{BooleanValue: false},
			),
			Entry("a number that is not zero as Boolean",
				sparkplug.Boolean, int64(2),
				&pb.Payload_Metric_BooleanValue{BooleanValue: true},
			),
			Entry("a float64 zero as Boolean",
				sparkplug.Boolean, float64(0),
				&pb.Payload_Metric_BooleanValue{BooleanValue: false},
			),
			Entry("a uint64 zero as Boolean",
				sparkplug.Boolean, uint64(0),
				&pb.Payload_Metric_BooleanValue{BooleanValue: false},
			),
			Entry("String",
				sparkplug.String, "open",
				&pb.Payload_Metric_StringValue{StringValue: "open"},
			),
			Entry("Text",
				sparkplug.Text, "a note",
				&pb.Payload_Metric_StringValue{StringValue: "a note"},
			),
			Entry("UUID",
				sparkplug.UUID, "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
				&pb.Payload_Metric_StringValue{
					StringValue: "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
				},
			),
			Entry("DateTime in milliseconds",
				sparkplug.DateTime, 1700000000456*telem.MillisecondTS+999,
				&pb.Payload_Metric_LongValue{LongValue: 1700000000456},
			),
		)

		DescribeTable(
			"Should reject a value that does not fit the data type",
			func(dt sparkplug.DataType, value any, message string) {
				Expect(sparkplug.EncodeCommand(
					sparkplug.Metric{Name: "tag", DataType: dt, Value: value},
					now,
				)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("tag tag: ")),
					MatchError(ContainSubstring(message)),
					MatchError(ContainSubstring("value does not fit the data type")),
				))
			},
			Entry("300 as Int8", sparkplug.Int8, int64(300), "300 as Int8"),
			Entry("-129 as Int8", sparkplug.Int8, int64(-129), "-129 as Int8"),
			Entry("32768 as Int16", sparkplug.Int16, int64(32768), "32768 as Int16"),
			Entry("2147483648 as Int32",
				sparkplug.Int32, int64(math.MaxInt32+1), "2147483648 as Int32",
			),
			Entry(
				"a uint64 over the Int64 range",
				sparkplug.Int64,
				uint64(math.MaxInt64+1),
				"9223372036854775808 as Int64",
			),
			Entry("a float64 over the Int64 range",
				sparkplug.Int64, float64(1<<63), "9.223372036854776e+18 as Int64",
			),
			Entry("1.5 as Int32", sparkplug.Int32, 1.5, "1.5 as Int32"),
			Entry("NaN as Int32", sparkplug.Int32, math.NaN(), "NaN as Int32"),
			Entry("-1 as UInt8", sparkplug.UInt8, int64(-1), "-1 as UInt8"),
			Entry("256 as UInt8", sparkplug.UInt8, uint64(256), "256 as UInt8"),
			Entry(
				"65536 as UInt16",
				sparkplug.UInt16,
				uint64(65536),
				"65536 as UInt16",
			),
			Entry("4294967296 as UInt32",
				sparkplug.UInt32, uint64(math.MaxUint32+1), "4294967296 as UInt32",
			),
			Entry("-1 as UInt64", sparkplug.UInt64, int64(-1), "-1 as UInt64"),
			Entry("-0.5 as UInt64", sparkplug.UInt64, -0.5, "-0.5 as UInt64"),
			Entry("a float64 over the UInt64 range",
				sparkplug.UInt64, float64(1<<63)*2, "1.8446744073709552e+19 as UInt64",
			),
			Entry("a string as Int32", sparkplug.Int32, "12", "12 as Int32"),
			Entry("a string as UInt32", sparkplug.UInt32, "12", "12 as UInt32"),
			Entry("a string as Float", sparkplug.Float, "1.5", "1.5 as Float"),
			Entry("a string as Double", sparkplug.Double, "1.5", "1.5 as Double"),
			Entry("a string as Boolean", sparkplug.Boolean, "true", "true as Boolean"),
			Entry("a number as String", sparkplug.String, int64(5), "5 as String"),
			Entry("a bool as Text", sparkplug.Text, true, "true as Text"),
			Entry("a number as UUID", sparkplug.UUID, 1.5, "1.5 as UUID"),
			Entry("a nil value", sparkplug.Double, nil, "<nil> as Double"),
			Entry("an int64 as DateTime",
				sparkplug.DateTime, int64(1700000000456), "1700000000456 as DateTime",
			),
			Entry("a negative DateTime",
				sparkplug.DateTime, -1*telem.MillisecondTS, " as DateTime",
			),
			Entry("a data set",
				sparkplug.DataType(pb.DataType_DataSet), int64(1),
				"DataSet is not supported",
			),
			Entry("bytes",
				sparkplug.DataType(pb.DataType_Bytes), "abc", "Bytes is not supported",
			),
			Entry("the unknown type",
				sparkplug.DataType(pb.DataType_Unknown), int64(1),
				"Unknown is not supported",
			),
		)

		It("Should reject a number over the Float range", func() {
			Expect(sparkplug.EncodeCommand(
				sparkplug.Metric{Name: "tag", DataType: sparkplug.Float, Value: 1e300},
				now,
			)).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("1e+300 as Float")),
			))
		})
	})

	Describe("EncodeRebirth", func() {
		It("Should set Node Control/Rebirth to Boolean true", func() {
			p, m := decodeCommand(MustSucceed(sparkplug.EncodeRebirth(now)))
			Expect(sparkplug.RebirthMetric).To(Equal("Node Control/Rebirth"))
			Expect(m.GetName()).To(Equal(sparkplug.RebirthMetric))
			Expect(m.GetDatatype()).To(Equal(uint32(pb.DataType_Boolean)))
			Expect(m.GetValue()).To(Equal(
				&pb.Payload_Metric_BooleanValue{BooleanValue: true},
			))
			Expect(p.Seq).To(BeNil())
			Expect(p.Timestamp).To(HaveValue(Equal(uint64(1700000000123))))
			Expect(m.Timestamp).To(HaveValue(Equal(uint64(1700000000123))))
		})
	})
})
