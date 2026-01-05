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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/synnaxlabs/x/telem"
)

func DataTypeInferTest[T any](expected telem.DataType) func() {
	return func() {
		dt := telem.InferDataType[T]()
		ExpectWithOffset(1, dt).To(Equal(expected))
	}
}

var _ = Describe("DataType", func() {
	Describe("Infer", func() {
		Specify("float64", DataTypeInferTest[float64](telem.Float64T))
		Specify("float32", DataTypeInferTest[float32](telem.Float32T))
		Specify("int64", DataTypeInferTest[int64](telem.Int64T))
		Specify("int32", DataTypeInferTest[int32](telem.Int32T))
		Specify("int16", DataTypeInferTest[int16](telem.Int16T))
		Specify("int8", DataTypeInferTest[int8](telem.Int8T))
		Specify("uint64", DataTypeInferTest[uint64](telem.Uint64T))
		Specify("uint32", DataTypeInferTest[uint32](telem.Uint32T))
		Specify("uint16", DataTypeInferTest[uint16](telem.Uint16T))
		Specify("uint8", DataTypeInferTest[uint8](telem.Uint8T))
		Specify("string", DataTypeInferTest[string](telem.StringT))

		It("Should panic if a a struct if provided", func() {
			Expect(func() {
				telem.InferDataType[struct{}]()
			}).To(Panic())
		})
	})

	DescribeTable("Density", func(dataType telem.DataType, expected telem.Density) {
		Expect(dataType.Density()).To(Equal(expected))
	},
		Entry("float64", telem.Float64T, telem.Bit64),
		Entry("float32", telem.Float32T, telem.Bit32),
		Entry("int64", telem.Int64T, telem.Bit64),
		Entry("int32", telem.Int32T, telem.Bit32),
		Entry("int16", telem.Int16T, telem.Bit16),
		Entry("int8", telem.Int8T, telem.Bit8),
		Entry("uint64", telem.Uint64T, telem.Bit64),
		Entry("uint32", telem.Uint32T, telem.Bit32),
		Entry("uint16", telem.Uint16T, telem.Bit16),
		Entry("uint8", telem.Uint8T, telem.Bit8),
		Entry("string", telem.StringT, telem.UnknownDensity),
		Entry("timestamp", telem.TimeStampT, telem.Bit64),
		Entry("uuid", telem.UUIDT, telem.Bit128),
		Entry("random", telem.DataType("random"), telem.UnknownDensity),
	)
})
