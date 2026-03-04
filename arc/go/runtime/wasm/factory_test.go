// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("convertConfigValue", func() {
	DescribeTable("supported numeric and timestamp types",
		func(v any, expected uint64) {
			result, err := convertConfigValue(v)
			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(expected))
		},
		Entry("int8", int8(1), uint64(1)),
		Entry("int16", int16(2), uint64(2)),
		Entry("int32", int32(3), uint64(3)),
		Entry("int64", int64(4), uint64(4)),
		Entry("uint8", uint8(5), uint64(5)),
		Entry("uint16", uint16(6), uint64(6)),
		Entry("uint32", uint32(7), uint64(7)),
		Entry("uint64", uint64(8), uint64(8)),
		Entry("float32", float32(1.5), uint64(math.Float32bits(1.5))),
		Entry("float64", float64(2.5), math.Float64bits(2.5)),
		Entry("telem.TimeStamp", telem.TimeStamp(9), uint64(9)),
	)

	DescribeTable("unsupported types return an error instead of panicking",
		func(v any) {
			_, err := convertConfigValue(v)
			Expect(err).To(HaveOccurred())
		},
		Entry("bool", true),
	)
})
