// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"math/rand/v2"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/synnax/pkg/service/resource/versions/v0"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Names", func() {
	// A later change to the live rule deletes this spec and owes its own migration.
	It("Should judge each name of a fixed corpus as the live rule does", func() {
		for _, name := range []string{
			"PT-FU-01",
			"Stand A",
			" pt ",
			"2024",
			"\u5727\u529B \u00E9",
			"\U0001F4A7",
			"\uFFFD",
			"",
			"   ",
			"a.b",
			".",
			"a\x00b",
			"a\tb",
			"a\u200Bb",
			"\u202Ept",
			"a\u00ADb",
			"a\u00A0b",
			"a\u3000b",
			"a\u2028b",
			"a\u2029b",
			"\u115F",
			"\u1160",
			"\u3164",
			"\uFFA0",
			"a\u2800b",
			"a\u034Fb",
			"\u2764\uFE0F",
			"a\xffb",
			"e\u0301",
		} {
			Expect(
				v0.IsLegalName(name),
			).To(Equal(validate.IsPathSegment(name)), "%+q", name)
		}
	})

	DescribeTable("SanitizeName",
		func(s, expected string) {
			out := v0.SanitizeName(s)
			Expect(out).To(Equal(expected))
			if out != "" {
				Expect(v0.IsLegalName(out)).To(BeTrue())
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

	DescribeTable("Should return a name that passes the rule",
		func(s string) {
			out := v0.SanitizeName(s)
			Expect(out).ToNot(BeEmpty())
			Expect(v0.IsLegalName(out)).To(BeTrue(), "sanitized %+q", out)
		},
		Entry(
			"30 combining marks and then a Hangul vowel",
			"a"+strings.Repeat("\u0316", 30)+"\u1161",
		),
		Entry(
			"31 combining marks and then a Hangul vowel",
			"a"+strings.Repeat("\u0316", 31)+"\u1161",
		),
		Entry(
			"30 combining marks and then a Hangul final consonant",
			"a"+strings.Repeat("\u0316", 30)+"\u11A8",
		),
		Entry(
			"30 combining marks and then a Bengali vowel sign",
			"\u0995"+strings.Repeat("\u0316", 30)+"\u09BE",
		),
		Entry(
			"30 combining marks and then a Tamil vowel sign",
			"\u0B95"+strings.Repeat("\u0316", 30)+"\u0BBE",
		),
		Entry(
			"30 combining marks and then a Hangul compatibility vowel",
			"a"+strings.Repeat("\u0316", 30)+"\u3160",
		),
		Entry("31 Hangul vowels and no combining mark", strings.Repeat("\u1161", 31)),
		Entry(
			"31 Hangul compatibility vowels and no combining mark",
			strings.Repeat("\u3160", 31),
		),
	)

	DescribeTable("Should return a legal name that a second sanitize keeps",
		func(r rune) {
			out := v0.SanitizeName("a" + strings.Repeat(string(r), 31))
			Expect(v0.IsLegalName(out)).To(BeTrue())
			Expect(v0.SanitizeName(out)).To(Equal(out))
		},
		Entry("a combining acute accent", '\u0301'),
		Entry("a Hangul vowel jamo", '\u1161'),
		Entry("a Hangul final jamo", '\u11A8'),
		Entry("an Oriya vowel sign", '\u0B3E'),
		Entry("a Kannada vowel sign", '\u0CC2'),
		Entry("a Kannada length mark", '\u0CD5'),
		Entry("a Myanmar vowel sign", '\u102E'),
	)

	It("Should return a name that passes the rule and that a second sanitize keeps",
		func() {
			pool := []rune{
				'a',
				'Z',
				'.',
				' ',
				'\u0301',
				'\u0316',
				'\u1161',
				'\u11A8',
				'\u0B3E',
				'\u0CD5',
				'\u102E',
				'\uFF9E',
				'\u3160',
				'\u200B',
				'\u00A0',
				'\u034F',
				'\u2800',
				'\uFFFD',
			}
			rng := rand.New(rand.NewPCG(1, 2))
			for range 20_000 {
				var b strings.Builder
				for range rng.IntN(80) {
					b.WriteRune(pool[rng.IntN(len(pool))])
				}
				if rng.IntN(10) == 0 {
					b.WriteByte(0xff)
				}
				out := v0.SanitizeName(b.String())
				if out == "" {
					continue
				}
				Expect(v0.IsLegalName(out)).To(BeTrue(), "%q", b.String())
				Expect(v0.SanitizeName(out)).To(Equal(out), "%q", b.String())
			}
		})
})
