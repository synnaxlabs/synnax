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
)

var _ = Describe("Names", func() {
	DescribeTable("NeedsRename",
		func(s string, expected bool) {
			Expect(v0.NeedsRename(s)).To(Equal(expected))
		},
		Entry("an identifier", "pt_01", false),
		Entry("a dash and a space", "Stand A-01", false),
		Entry("spaces only", "   ", false),
		Entry("a character that shows as nothing", "a\u200Bb", false),
		Entry("a name not in NFC", "e\u0301", false),
		Entry("an empty name", "", true),
		Entry("a separator", "stand.pt", true),
		Entry("an invalid byte", "a\xffb", true),
	)

	DescribeTable("SanitizeName",
		func(s, expected string) {
			out := v0.SanitizeName(s)
			Expect(out).To(Equal(expected))
			if out != "" {
				Expect(v0.NeedsRename(out)).To(BeFalse())
			}
		},
		Entry("a name that needs no rename", "Stand A-01", "Stand A-01"),
		Entry("a separator", "stand.pt", "stand_pt"),
		Entry("each invalid byte", "a\xff\xfeb", "a\uFFFD\uFFFDb"),
		Entry("a separator in a name not in NFC", "e\u0301.x", "\u00E9_x"),
		Entry("a character that shows as nothing", "a\u200Bb", "a\u200Bb"),
		Entry("an empty name", "", ""),
	)

	It("Should return a name that needs no rename and that a second sanitize keeps",
		func() {
			pool := []rune{'a', 'Z', '.', ' ', '\u0301', '\u200B', '\u00A0', '\uFFFD'}
			rng := rand.New(rand.NewPCG(1, 2))
			for range 20_000 {
				var b strings.Builder
				for range rng.IntN(40) {
					b.WriteRune(pool[rng.IntN(len(pool))])
				}
				if rng.IntN(10) == 0 {
					b.WriteByte(0xff)
				}
				out := v0.SanitizeName(b.String())
				if out == "" {
					continue
				}
				Expect(v0.NeedsRename(out)).To(BeFalse(), "%q", b.String())
				Expect(v0.SanitizeName(out)).To(Equal(out), "%q", b.String())
			}
		})
})
