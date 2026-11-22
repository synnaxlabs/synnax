// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package validate_test

import (
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Path segments", func() {
	DescribeTable("IsPathSegment",
		func(s string, expected bool) {
			Expect(validate.IsPathSegment(s)).To(Equal(expected))
		},
		Entry("dashes and capitals", "PT-FU-01", true),
		Entry("an inner space", "Stand A", true),
		Entry("spaces around a name", " pt ", true),
		Entry("digits only", "2024", true),
		Entry("letters of other scripts", "\u5727\u529B \u00E9", true),
		Entry("a symbol with no variation selector", "\U0001F4A7", true),
		Entry("the replacement character", "\uFFFD", true),
		Entry("empty", "", false),
		Entry("spaces only", "   ", false),
		Entry("the separator", "a.b", false),
		Entry("a lone separator", ".", false),
		Entry("a control character (Cc)", "a\x00b", false),
		Entry("a tab (Cc)", "a\tb", false),
		Entry("a zero-width space (Cf)", "a\u200Bb", false),
		Entry("a right-to-left override (Cf)", "\u202Ept", false),
		Entry("a soft hyphen (Cf)", "a\u00ADb", false),
		Entry("a no-break space (Zs)", "a\u00A0b", false),
		Entry("an ideographic space (Zs)", "a\u3000b", false),
		Entry("a line separator (Zl)", "a\u2028b", false),
		Entry("a paragraph separator (Zp)", "a\u2029b", false),
		Entry("a Hangul choseong filler (default ignorable)", "\u115F", false),
		Entry("a Hangul jungseong filler (default ignorable)", "\u1160", false),
		Entry("a Hangul filler (default ignorable)", "\u3164", false),
		Entry("a halfwidth Hangul filler (default ignorable)", "\uFFA0", false),
		Entry("a Braille pattern blank, which shows as nothing", "a\u2800b", false),
		Entry("a combining grapheme joiner (default ignorable)", "a\u034Fb", false),
		Entry("a variation selector (default ignorable)", "\u2764\uFE0F", false),
		Entry("an invalid byte", "a\xffb", false),
		Entry("a name not in NFC", "e\u0301", false),
	)

	DescribeTable("SanitizePathSegment",
		func(s, expected string) {
			out := validate.SanitizePathSegment(s)
			Expect(out).To(Equal(expected))
			if out != "" {
				Expect(validate.IsPathSegment(out)).To(BeTrue())
			}
		},
		Entry("a legal name", "Stand A-01", "Stand A-01"),
		Entry("a separator", "stand.pt", "stand_pt"),
		Entry("a control character", "a\x00b", "ab"),
		Entry("a zero-width space", "a\u200Bb", "ab"),
		Entry("a no-break space", "a\u00A0b", "ab"),
		Entry("a Braille pattern blank", "a\u2800b", "ab"),
		Entry("each invalid byte", "a\xff\xfeb", "a\uFFFD\uFFFDb"),
		Entry("a name not in NFC", "e\u0301", "\u00E9"),
		Entry("refused characters around a mark", "e\u200B\u0301", "\u00E9"),
		Entry("refused characters only", "\u200B\u00A0", ""),
		Entry("spaces only after the rest goes", " \u200B ", ""),
		Entry(
			"31 combining marks in a row",
			"a"+strings.Repeat("\u0316", 31),
			"a"+strings.Repeat("\u0316", 30),
		),
		Entry(
			"61 combining marks in a row",
			"a"+strings.Repeat("\u0316", 61)+"b",
			"a"+strings.Repeat("\u0316", 30)+"b",
		),
	)

	It("Should allocate nothing for an ASCII name or a name in NFC", func() {
		for _, name := range []string{"pt_unique_01", "Stand A-01", "\u5727\u529b \u00E9"} {
			Expect(testing.AllocsPerRun(10, func() {
				_ = validate.IsPathSegment(name)
				_ = validate.NormalizeName(name)
			})).To(BeZero())
		}
	})

	Describe("NormalizeName", func() {
		It("Should compose a decomposed name", func() {
			Expect(validate.NormalizeName("e\u0301")).To(Equal("\u00E9"))
		})
	})
})
