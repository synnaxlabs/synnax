// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package literal_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/internal/literal"
)

var _ = Describe("Float", func() {
	DescribeTable("Should render a literal that reads back exactly",
		func(v float64, expected string) {
			Expect(literal.Float(v)).To(Equal(expected))
		},
		Entry("a short fraction", 0.5, "0.500000"),
		Entry("zero", 0.0, "0.000000"),
		Entry("a negative", -2.25, "-2.250000"),
		Entry("a small value", 1e-7, "1e-07"),
		Entry("a long fraction", 0.1234567, "0.1234567"),
	)
})
