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

var _ = Describe("Split breaker", func() {
	It("Should keep a name with a brace in a nested backtick format string", func() {
		segments := MustSucceed(literal.FmtStrSplit(
			"{f`{'a}b'}`}",
			literal.StringFlags{Format: true},
		))
		Expect(segments).To(HaveLen(1))
		Expect(segments[0].IsPlaceholder).To(BeTrue())
		Expect(segments[0].Text).To(Equal("f`{'a}b'}`"))
	})

	It("Should keep a name with a brace in a nested escaped format string", func() {
		segments := MustSucceed(literal.FmtStrSplit(
			`{f\"{'a}b'}\"}`,
			literal.StringFlags{Format: true},
		))
		Expect(segments).To(HaveLen(1))
		Expect(segments[0].IsPlaceholder).To(BeTrue())
		Expect(segments[0].Text).To(Equal(`f"{'a}b'}"`))
	})

	It("Should close a placeholder whose nested format string names 'a{b'", func() {
		segments := MustSucceed(literal.FmtStrParse("{f`{'a{b'}`} and {x}"))
		Expect(segments).To(HaveLen(3))
		Expect(segments[0].Text).To(Equal("f`{'a{b'}`"))
		Expect(segments[2].Text).To(Equal("x"))
	})
})
