// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/library/icd"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Parse", func() {
	DescribeTable("Should parse each format",
		func(format icd.Format, data func() []byte, messages int) {
			entries := MustSucceed(icd.Parse(format, data()))
			Expect(messagesOf(entries)).To(HaveLen(messages))
		},
		Entry("DBC", icd.FormatDBC, func() []byte {
			data, _ := readFixture("basic")
			return data
		}, 5),
		Entry("CSV", icd.FormatCSV, func() []byte { return toCSV(tableRows) }, 3),
		Entry("XLSX", icd.FormatXLSX, func() []byte { return toXLSX(tableRows) }, 3),
	)

	It("Should reject an unknown format", func() {
		Expect(icd.Parse("arxml", nil)).Error().To(MatchError(
			`format: unsupported format "arxml": validation error`,
		))
	})
})
