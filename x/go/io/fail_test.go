// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package io_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/io"
)

var _ = Describe("AlwaysFail", func() {
	DescribeTable("AlwaysFailWriter should always fail", func(p []byte) {
		Expect(io.FailWriter.Write(p)).Error().
			To(MatchError(io.ErrFailWriter))
	},
		Entry("with an empty byte slice", []byte{}),
		Entry("with a nil byte slice", nil),
		Entry("with a filled byte slice", []byte{1, 2, 3}),
	)
	DescribeTable("AlwaysFailReader should always fail", func(p []byte) {
		Expect(io.FailReader.Read(p)).Error().
			To(MatchError(io.ErrFailReader))
	},
		Entry("with an empty byte slice", []byte{}),
		Entry("with a nil byte slice", nil),
		Entry("with a filled byte slice", []byte{1, 2, 3}),
	)
})
