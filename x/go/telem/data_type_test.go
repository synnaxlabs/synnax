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

var _ = Describe("DataTypeCode", func() {
	DescribeTable("Should map each data type to its code and back",
		func(dt telem.DataType, code telem.DataTypeCode, value uint8) {
			Expect(uint8(code)).To(Equal(value))
			Expect(telem.CodeOf(dt)).To(Equal(code))
			Expect(code.DataType()).To(Equal(dt))
		},
		Entry("uint8", telem.Uint8T, telem.DataTypeCodeUint8, uint8(1)),
		Entry("uint16", telem.Uint16T, telem.DataTypeCodeUint16, uint8(2)),
		Entry("uint32", telem.Uint32T, telem.DataTypeCodeUint32, uint8(3)),
		Entry("uint64", telem.Uint64T, telem.DataTypeCodeUint64, uint8(4)),
		Entry("int8", telem.Int8T, telem.DataTypeCodeInt8, uint8(5)),
		Entry("int16", telem.Int16T, telem.DataTypeCodeInt16, uint8(6)),
		Entry("int32", telem.Int32T, telem.DataTypeCodeInt32, uint8(7)),
		Entry("int64", telem.Int64T, telem.DataTypeCodeInt64, uint8(8)),
		Entry("float32", telem.Float32T, telem.DataTypeCodeFloat32, uint8(9)),
		Entry("float64", telem.Float64T, telem.DataTypeCodeFloat64, uint8(10)),
		Entry("timestamp", telem.TimestampT, telem.DataTypeCodeTimestamp, uint8(11)),
		Entry("uuid", telem.UUIDT, telem.DataTypeCodeUUID, uint8(12)),
		Entry("string", telem.StringT, telem.DataTypeCodeString, uint8(13)),
		Entry("bytes", telem.BytesT, telem.DataTypeCodeBytes, uint8(14)),
		Entry("json", telem.JSONT, telem.DataTypeCodeJSON, uint8(15)),
		Entry("boolean", telem.BooleanT, telem.DataTypeCodeBoolean, uint8(16)),
	)

	It("Should give the unknown code to a data type with no code", func() {
		Expect(telem.CodeOf(telem.UnknownT)).To(Equal(telem.DataTypeCodeUnknown))
		Expect(telem.CodeOf("int128")).To(Equal(telem.DataTypeCodeUnknown))
	})

	DescribeTable("Should give the unknown data type to a code with no data type",
		func(code telem.DataTypeCode) {
			Expect(code.DataType()).To(Equal(telem.UnknownT))
		},
		Entry("unknown", telem.DataTypeCodeUnknown),
		Entry("the first free code", telem.DataTypeCode(17)),
		Entry("the reserved code", telem.DataTypeCode(255)),
	)
})
