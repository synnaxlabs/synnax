// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/expression"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("RedeclareHint", func() {
	It("Should give the new declaration back", func() {
		hint := expression.RedeclareHint("t := 120s", "t i64 ns := 120s")
		Expect(hint).To(Equal("hint: change `t := 120s` to `t i64 ns := 120s`"))
		Expect(
			MustBeOk(expression.ParseRedeclareHint(hint)),
		).To(Equal("t i64 ns := 120s"))
	})

	DescribeTable("Should not parse another note",
		func(note string) {
			_, ok := expression.ParseRedeclareHint(note)
			Expect(ok).To(BeFalse())
		},
		Entry("a cast hint", "hint: use i64 ns(value) to convert"),
		Entry("a signature", "signature: time.wait(duration i64 ns) u8"),
		Entry("a hint with no new declaration", "hint: change `t := 120s`"),
	)
})
