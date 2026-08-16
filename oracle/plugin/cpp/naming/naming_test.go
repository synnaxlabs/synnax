// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package naming_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/oracle/plugin/cpp/naming"
	. "github.com/synnaxlabs/x/testutil"
)

func TestNaming(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CPP Naming Suite")
}

var _ = Describe("VariantTypeName", func() {
	DescribeTable(
		"should derive the C++ variant struct name",
		func(union, variant, expected string) {
			Expect(naming.VariantTypeName(union, variant)).To(Equal(expected))
		},
		Entry("plain union takes the variant first", "Scale", "linear", "LinearScale"),
		Entry(
			"acronym union factors the shared prefix",
			"AIChannel",
			"ai_voltage",
			"AIVoltageChannel",
		),
		Entry(
			"variant not repeating the acronym prefixes the whole union name",
			"AIChannel",
			"voltage",
			"VoltageAIChannel",
		),
		Entry("reserved-word variant value", "Scale", "map", "MapScale"),
	)
})

var _ = ShouldNotLeakGoroutinesPerSpec()
