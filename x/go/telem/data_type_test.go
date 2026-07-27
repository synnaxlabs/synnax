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
	Specify("timestamp", DataTypeInferTest[telem.TimeStamp](telem.TimeStampT))
	Specify("uuid", DataTypeInferTest[uuid.UUID](telem.UUIDT))
	Specify("string", DataTypeInferTest[string](telem.StringT))
	Specify("bytes", DataTypeInferTest[[]byte](telem.BytesT))
})
