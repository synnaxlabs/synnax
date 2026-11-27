// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
)

func DataTypeInferTest[T telem.Sample](expected telem.DataType) func() {
	return func() {
		dt := telem.InferDataType[T]()
		ExpectWithOffset(1, dt).To(Equal(expected))
	}
}

var _ = Describe("InferDataType", func() {
	Specify("uint8", DataTypeInferTest[uint8](telem.Uint8T))
	Specify("uint16", DataTypeInferTest[uint16](telem.Uint16T))
	Specify("uint32", DataTypeInferTest[uint32](telem.Uint32T))
	Specify("uint64", DataTypeInferTest[uint64](telem.Uint64T))
	Specify("int8", DataTypeInferTest[int8](telem.Int8T))
	Specify("int16", DataTypeInferTest[int16](telem.Int16T))
	Specify("int32", DataTypeInferTest[int32](telem.Int32T))
	Specify("int64", DataTypeInferTest[int64](telem.Int64T))
	Specify("float32", DataTypeInferTest[float32](telem.Float32T))
	Specify("float64", DataTypeInferTest[float64](telem.Float64T))
	Specify("timestamp", DataTypeInferTest[telem.TimeStamp](telem.TimestampT))
	Specify("uuid", DataTypeInferTest[uuid.UUID](telem.UUIDT))
	Specify("string", DataTypeInferTest[string](telem.StringT))
	Specify("bytes", DataTypeInferTest[[]byte](telem.BytesT))
	Specify("bool", DataTypeInferTest[bool](telem.BooleanT))
})

var _ = Describe("DataTypeName", func() {
	DescribeTable("Should name each data type constant",
		func(name telem.DataTypeName, expected telem.DataType) {
			Expect(name.IsValid()).To(BeTrue())
			Expect(telem.DataType(name)).To(Equal(expected))
		},
		Entry("float64", telem.DataTypeNameFloat64, telem.Float64T),
		Entry("float32", telem.DataTypeNameFloat32, telem.Float32T),
		Entry("int64", telem.DataTypeNameInt64, telem.Int64T),
		Entry("int32", telem.DataTypeNameInt32, telem.Int32T),
		Entry("int16", telem.DataTypeNameInt16, telem.Int16T),
		Entry("int8", telem.DataTypeNameInt8, telem.Int8T),
		Entry("uint64", telem.DataTypeNameUint64, telem.Uint64T),
		Entry("uint32", telem.DataTypeNameUint32, telem.Uint32T),
		Entry("uint16", telem.DataTypeNameUint16, telem.Uint16T),
		Entry("uint8", telem.DataTypeNameUint8, telem.Uint8T),
		Entry("timestamp", telem.DataTypeNameTimestamp, telem.TimestampT),
		Entry("uuid", telem.DataTypeNameUUID, telem.UUIDT),
		Entry("boolean", telem.DataTypeNameBoolean, telem.BooleanT),
		Entry("string", telem.DataTypeNameString, telem.StringT),
		Entry("json", telem.DataTypeNameJSON, telem.JSONT),
		Entry("bytes", telem.DataTypeNameBytes, telem.BytesT),
	)

	It("Should reject the unknown data type", func() {
		Expect(telem.DataTypeName(telem.UnknownT).IsValid()).To(BeFalse())
	})
})

var _ = Describe("FixedWidthDataType", func() {
	DescribeTable("Should name a fixed-density data type",
		func(dt telem.FixedWidthDataType, expected telem.DataType) {
			Expect(telem.DataType(dt)).To(Equal(expected))
			Expect(expected.IsVariable()).To(BeFalse())
		},
		Entry("float64", telem.FixedWidthDataTypeFloat64, telem.Float64T),
		Entry("float32", telem.FixedWidthDataTypeFloat32, telem.Float32T),
		Entry("int64", telem.FixedWidthDataTypeInt64, telem.Int64T),
		Entry("int32", telem.FixedWidthDataTypeInt32, telem.Int32T),
		Entry("int16", telem.FixedWidthDataTypeInt16, telem.Int16T),
		Entry("int8", telem.FixedWidthDataTypeInt8, telem.Int8T),
		Entry("uint64", telem.FixedWidthDataTypeUint64, telem.Uint64T),
		Entry("uint32", telem.FixedWidthDataTypeUint32, telem.Uint32T),
		Entry("uint16", telem.FixedWidthDataTypeUint16, telem.Uint16T),
		Entry("uint8", telem.FixedWidthDataTypeUint8, telem.Uint8T),
		Entry("timestamp", telem.FixedWidthDataTypeTimestamp, telem.TimestampT),
		Entry("uuid", telem.FixedWidthDataTypeUUID, telem.UUIDT),
		Entry("boolean", telem.FixedWidthDataTypeBoolean, telem.BooleanT),
	)
})

var _ = Describe("NumericDataType", func() {
	DescribeTable("Should name an integer or float data type",
		func(dt telem.NumericDataType, expected telem.DataType) {
			Expect(dt.IsValid()).To(BeTrue())
			Expect(telem.DataType(dt)).To(Equal(expected))
		},
		Entry("float64", telem.NumericDataTypeFloat64, telem.Float64T),
		Entry("float32", telem.NumericDataTypeFloat32, telem.Float32T),
		Entry("int64", telem.NumericDataTypeInt64, telem.Int64T),
		Entry("int32", telem.NumericDataTypeInt32, telem.Int32T),
		Entry("int16", telem.NumericDataTypeInt16, telem.Int16T),
		Entry("int8", telem.NumericDataTypeInt8, telem.Int8T),
		Entry("uint64", telem.NumericDataTypeUint64, telem.Uint64T),
		Entry("uint32", telem.NumericDataTypeUint32, telem.Uint32T),
		Entry("uint16", telem.NumericDataTypeUint16, telem.Uint16T),
		Entry("uint8", telem.NumericDataTypeUint8, telem.Uint8T),
	)

	DescribeTable("Should reject a data type that is not a number",
		func(dt telem.DataType) {
			Expect(telem.NumericDataType(dt).IsValid()).To(BeFalse())
		},
		Entry("timestamp", telem.TimestampT),
		Entry("uuid", telem.UUIDT),
		Entry("boolean", telem.BooleanT),
		Entry("string", telem.StringT),
	)
})
