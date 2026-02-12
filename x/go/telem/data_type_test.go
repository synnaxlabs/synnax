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
	"github.com/google/uuid"
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

var _ = Describe("DataType", func() {
	DescribeTable("Density", func(dataType telem.DataType, expected telem.Density) {
		Expect(dataType.Density()).To(Equal(expected))
	},
		Entry("unknown", telem.UnknownT, telem.UnknownDensity),
		Entry("uint8", telem.Uint8T, telem.Bit8),
		Entry("uint16", telem.Uint16T, telem.Bit16),
		Entry("uint32", telem.Uint32T, telem.Bit32),
		Entry("uint64", telem.Uint64T, telem.Bit64),
		Entry("int8", telem.Int8T, telem.Bit8),
		Entry("int16", telem.Int16T, telem.Bit16),
		Entry("int32", telem.Int32T, telem.Bit32),
		Entry("int64", telem.Int64T, telem.Bit64),
		Entry("float32", telem.Float32T, telem.Bit32),
		Entry("float64", telem.Float64T, telem.Bit64),
		Entry("timestamp", telem.TimeStampT, telem.Bit64),
		Entry("uuid", telem.UUIDT, telem.Bit128),
		Entry("string", telem.StringT, telem.UnknownDensity),
		Entry("bytes", telem.BytesT, telem.UnknownDensity),
		Entry("json", telem.JSONT, telem.UnknownDensity),
	)

	DescribeTable("IsVariable", func(dataType telem.DataType, expected bool) {
		Expect(dataType.IsVariable()).To(Equal(expected))
	},
		Entry("unknown", telem.UnknownT, false),
		Entry("uint8", telem.Uint8T, false),
		Entry("uint16", telem.Uint16T, false),
		Entry("uint32", telem.Uint32T, false),
		Entry("uint64", telem.Uint64T, false),
		Entry("int8", telem.Int8T, false),
		Entry("int16", telem.Int16T, false),
		Entry("int32", telem.Int32T, false),
		Entry("int64", telem.Int64T, false),
		Entry("float32", telem.Float32T, false),
		Entry("float64", telem.Float64T, false),
		Entry("timestamp", telem.TimeStampT, false),
		Entry("uuid", telem.UUIDT, false),
		Entry("string", telem.StringT, true),
		Entry("bytes", telem.BytesT, true),
		Entry("json", telem.JSONT, true),
	)

	Describe("Infer", func() {
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
		Specify("timestamp", DataTypeInferTest[telem.TimeStamp](telem.TimeStampT))
		Specify("uuid", DataTypeInferTest[uuid.UUID](telem.UUIDT))
		Specify("string", DataTypeInferTest[string](telem.StringT))
		Specify("bytes", DataTypeInferTest[[]byte](telem.BytesT))
	})
})
