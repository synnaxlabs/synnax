// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("KindOf", func() {
	DescribeTable(
		"should map a data type to its read field kind",
		func(dt telem.DataType, expected payload.Kind) {
			Expect(MustBeOk(payload.KindOf(dt))).To(Equal(expected))
		},
		Entry("float32", telem.Float32T, payload.KindFloat),
		Entry("float64", telem.Float64T, payload.KindFloat),
		Entry("int8", telem.Int8T, payload.KindInteger),
		Entry("int16", telem.Int16T, payload.KindInteger),
		Entry("int32", telem.Int32T, payload.KindInteger),
		Entry("int64", telem.Int64T, payload.KindInteger),
		Entry("uint8", telem.Uint8T, payload.KindUnsigned),
		Entry("uint16", telem.Uint16T, payload.KindUnsigned),
		Entry("uint32", telem.Uint32T, payload.KindUnsigned),
		Entry("uint64", telem.Uint64T, payload.KindUnsigned),
		Entry("boolean", telem.BooleanT, payload.KindBoolean),
		Entry("string", telem.StringT, payload.KindString),
		Entry("timestamp", telem.TimestampT, payload.KindTimestamp),
		Entry("json", telem.JSONT, payload.KindJSON),
		Entry("uuid", telem.UUIDT, payload.KindUUID),
		Entry("bytes", telem.BytesT, payload.KindBytes),
	)

	DescribeTable(
		"should not map an unknown data type",
		func(dt telem.DataType) {
			k, ok := payload.KindOf(dt)
			Expect(ok).To(BeFalse())
			Expect(k).To(BeZero())
		},
		Entry("empty", telem.UnknownT),
		Entry("unlisted", telem.DataType("complex128")),
	)
})
