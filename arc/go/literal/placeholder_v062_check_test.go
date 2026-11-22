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

// Placeholders that v0.62 split, so stored v0.62 text still compiles.
var _ = Describe("Placeholder v0.62 check", func() {
	It("Should keep the spec of a placeholder with a bracket in a comment", func() {
		segments := MustSucceed(literal.FmtStrParse("{x /* range [0, 100) */:.1f}"))
		Expect(segments).To(HaveLen(1))
		Expect(segments[0].IsPlaceholder).To(BeTrue())
		Expect(segments[0].Text).To(Equal("x /* range [0, 100) */"))
		Expect(segments[0].Spec).To(Equal(".1f"))
	})

	It("Should close a placeholder whose brace in a string a comment balances", func() {
		segments := MustSucceed(literal.FmtStrParse(`{len("{") + x /* } */}`))
		Expect(segments).To(HaveLen(1))
		Expect(segments[0].IsPlaceholder).To(BeTrue())
		Expect(segments[0].Text).To(Equal(`len("{") + x /* } */`))
	})
})
