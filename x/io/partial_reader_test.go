// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io_test

import (
	"bytes"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xio "github.com/synnaxlabs/x/io"
	"io"
)

var _ = Describe("PartialReader", func() {
	buf := []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
	DescribeTable("ReadAt", func(
		offset int,
		span int,
		length int,
		at int,
		expectedB []byte,
		expectedN int,
		expectedErr error,
	) {
		r := xio.PartialReaderAt(bytes.NewReader(buf), int64(offset), int64(span))
		b := make([]byte, length)
		n, err := r.ReadAt(b, int64(at))
		if expectedErr != nil {
			Expect(err).To(Equal(expectedErr))
		} else {
			Expect(err).To(BeNil())
		}
		Expect(n).To(Equal(int(expectedN)))
		Expect(b).To(Equal(expectedB))
	},
		Entry("Entire Buffer",
			/* off */ 0 /* span */, 20 /* len */, 20 /* at */, 0, buf /* n */, 20, nil),
		Entry("End of Buffer, from center",
			10, 10, 10, 0, buf[10:], 10, nil),
		Entry("Past end of Buffer, from center",
			10, 10, 15, 0, append(buf[10:], 0, 0, 0, 0, 0), 10, io.EOF),
		Entry("To end of length, from center, in buffer",
			5, 10, 10, 0, buf[5:15], 10, nil),
		Entry("Past end of length, from center, in buffer",
			5, 10, 10, 1, buf[6:16], 9, io.EOF),
		Entry("Read at past end of length, from center, in buffer",
			5, 10, 1, 11, []byte{0}, 0, io.EOF),
		Entry("Offset past end of buffer",
			20, 10, 10, 0, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 0, io.EOF),
		Entry("Span past end of buffer, read past end of buffer",
			10, 20, 15, 0, append(buf[10:], 0, 0, 0, 0, 0), 10, io.EOF),
		Entry("Offset and span past end of buffer",
			20, 20, 10, 0, []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, 0, io.EOF),
	)
})
