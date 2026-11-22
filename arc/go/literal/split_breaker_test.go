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

// A format string nested in a placeholder is a string to the outer split, so a quote in
// it opens no name and each brace in it counts. A name that holds a brace is not
// supported there.
var _ = Describe("Nested format string names", func() {
	It(
		"Should close the outer placeholder at a brace in a nested backtick name",
		func() {
			segments := MustSucceed(literal.FmtStrSplit(
				"{f`{'a}b'}`}",
				literal.StringFlags{Format: true},
			))
			Expect(segments).To(HaveLen(2))
			Expect(segments[0].IsPlaceholder).To(BeTrue())
			Expect(segments[0].Text).To(Equal("f`{'a}b'"))
			Expect(segments[1].IsPlaceholder).To(BeFalse())
			Expect(segments[1].Text).To(Equal("`}"))
		},
	)

	It(
		"Should close the outer placeholder at a brace in a nested escaped name",
		func() {
			segments := MustSucceed(literal.FmtStrSplit(
				`{f\"{'a}b'}\"}`,
				literal.StringFlags{Format: true},
			))
			Expect(segments).To(HaveLen(2))
			Expect(segments[0].IsPlaceholder).To(BeTrue())
			Expect(segments[0].Text).To(Equal(`f"{'a}b'`))
			Expect(segments[1].IsPlaceholder).To(BeFalse())
			Expect(segments[1].Text).To(Equal(`"}`))
		},
	)

	It("Should reject a nested name that opens a brace", func() {
		Expect(literal.FmtStrParse("{f`{'a{b'}`} and {x}")).Error().
			To(MatchError(ContainSubstring("unmatched '{'")))
	})
})
