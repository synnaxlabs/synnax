// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/x/telem/versions/v0"
)

var _ = Describe("DataType", func() {
	DescribeTable("Density", func(dataType v0.DataType, expected v0.Density) {
		Expect(dataType.Density()).To(Equal(expected))
	},
		Entry("unknown", v0.UnknownT, v0.UnknownDensity),
		Entry("uint8", v0.Uint8T, v0.Bit8),
		Entry("uint16", v0.Uint16T, v0.Bit16),
		Entry("uint32", v0.Uint32T, v0.Bit32),
		Entry("uint64", v0.Uint64T, v0.Bit64),
		Entry("int8", v0.Int8T, v0.Bit8),
		Entry("int16", v0.Int16T, v0.Bit16),
		Entry("int32", v0.Int32T, v0.Bit32),
		Entry("int64", v0.Int64T, v0.Bit64),
		Entry("float32", v0.Float32T, v0.Bit32),
		Entry("float64", v0.Float64T, v0.Bit64),
		Entry("timestamp", v0.TimeStampT, v0.Bit64),
		Entry("uuid", v0.UUIDT, v0.Bit128),
		Entry("string", v0.StringT, v0.UnknownDensity),
		Entry("bytes", v0.BytesT, v0.UnknownDensity),
		Entry("json", v0.JSONT, v0.UnknownDensity),
		Entry("bool", v0.BoolT, v0.Bit8),
	)

	DescribeTable("IsVariable", func(dataType v0.DataType, expected bool) {
		Expect(dataType.IsVariable()).To(Equal(expected))
	},
		Entry("unknown", v0.UnknownT, false),
		Entry("uint8", v0.Uint8T, false),
		Entry("uint16", v0.Uint16T, false),
		Entry("uint32", v0.Uint32T, false),
		Entry("uint64", v0.Uint64T, false),
		Entry("int8", v0.Int8T, false),
		Entry("int16", v0.Int16T, false),
		Entry("int32", v0.Int32T, false),
		Entry("int64", v0.Int64T, false),
		Entry("float32", v0.Float32T, false),
		Entry("float64", v0.Float64T, false),
		Entry("timestamp", v0.TimeStampT, false),
		Entry("uuid", v0.UUIDT, false),
		Entry("string", v0.StringT, true),
		Entry("bytes", v0.BytesT, true),
		Entry("json", v0.JSONT, true),
		Entry("bool", v0.BoolT, false),
	)
})
