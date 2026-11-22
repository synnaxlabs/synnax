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
	"github.com/synnaxlabs/arc/literal"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Placeholder comment check", func() {
	It(
		"Should split a quoted channel name before a comment that holds a quote",
		func() {
			segments := MustSucceed(literal.FmtStrParse("{'pt-01' // it's\n}"))
			Expect(segments).To(HaveLen(1))
			Expect(segments[0].IsPlaceholder).To(BeTrue())
			Expect(segments[0].Text).To(Equal("'pt-01' // it's\n"))
		},
	)
})
