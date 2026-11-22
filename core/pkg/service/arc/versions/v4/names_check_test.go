// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	"github.com/synnaxlabs/x/set"
)

// Forms that only the fuzz spec reaches today, pinned so they outlive it.
var _ = Describe("QuoteDashedNames check", func() {
	held := set.New("pt-01", "flow-1", "rpm-2").Contains
	DescribeTable("Should quote a dashed name in a form the fuzz alone reaches",
		func(text, expected string) {
			Expect(v4.QuoteDashedNames(text, held)).To(Equal(expected))
		},
		Entry("a name that starts with f", "return flow-1", "return 'flow-1'"),
		Entry("a name that starts with r", "return rpm-2 * 2", "return 'rpm-2' * 2"),
		Entry("a slice colon before a name in a placeholder",
			`f"{x[0:pt-01]}"`, `f"{x[0:'pt-01']}"`),
		Entry("a format string nested in a placeholder",
			`f"{f\"{a}\" + pt-01}"`, `f"{f\"{a}\" + 'pt-01'}"`),
	)
})
