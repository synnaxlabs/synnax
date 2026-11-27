// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("CheckFolderName", func() {
	DescribeTable("Should keep a name that a path segment can hold",
		func(name string) {
			Expect(channel.CheckFolderName(name)).To(Equal(name))
		},
		Entry("dashes and capitals", "PT-FU-01"),
		Entry("an inner space", "Stand A"),
		Entry("spaces around a name", " pt "),
		Entry("digits only", "2024"),
		Entry("letters of other scripts", "圧力 \u00E9"),
		Entry("a symbol with no variation selector", "💧"),
		Entry("the replacement character", "\uFFFD"),
	)

	DescribeTable("Should refuse a name that a path segment cannot hold",
		func(name string) {
			Expect(channel.CheckFolderName(name)).Error().
				To(MatchError(validate.ErrValidation))
		},
		Entry("empty", ""),
		Entry("spaces only", "   "),
		Entry("the separator", "a.b"),
		Entry("a lone separator", "."),
		Entry("a control character (Cc)", "a\x00b"),
		Entry("a tab (Cc)", "a\tb"),
		Entry("a zero-width space (Cf)", "a\u200Bb"),
		Entry("a right-to-left override (Cf)", "\u202Ept"),
		Entry("a soft hyphen (Cf)", "a\u00ADb"),
		Entry("a no-break space (Zs)", "a\u00A0b"),
		Entry("an ideographic space (Zs)", "a\u3000b"),
		Entry("a line separator (Zl)", "a\u2028b"),
		Entry("a paragraph separator (Zp)", "a\u2029b"),
		Entry("a Hangul choseong filler (default ignorable)", "\u115F"),
		Entry("a Hangul jungseong filler (default ignorable)", "\u1160"),
		Entry("a Hangul filler (default ignorable)", "\u3164"),
		Entry("a halfwidth Hangul filler (default ignorable)", "\uFFA0"),
		Entry("a Braille pattern blank, which shows as nothing", "a\u2800b"),
		Entry("a combining grapheme joiner (default ignorable)", "a\u034Fb"),
		Entry("a variation selector (default ignorable)", "❤\uFE0F"),
		Entry("an invalid byte", "a\xffb"),
	)

	It("Should store a decomposed name in NFC", func() {
		Expect(channel.CheckFolderName("e\u0301")).To(Equal("\u00E9"))
	})

	It("Should allocate nothing for an ASCII name or a name in NFC", func() {
		for _, name := range []string{"pt_unique_01", "Stand A-01", "\u5727\u529b \u00E9"} {
			var err error
			Expect(testing.AllocsPerRun(10, func() {
				_, err = channel.CheckFolderName(name)
			})).To(BeZero())
			Expect(err).ToNot(HaveOccurred())
		}
	})
})
