// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package os_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xos "github.com/synnaxlabs/x/os"
)

// sanitize applies the full length budget, which the cases below do not exercise.
func sanitize(name string) string {
	return xos.SanitizeFileName(name, xos.MaxFileNameLength)
}

var _ = Describe("SanitizeFileName", func() {
	DescribeTable("Should replace every character a file name cannot hold",
		func(name, expected string) { Expect(sanitize(name)).To(Equal(expected)) },
		Entry("path separators", `a/b\c`, "a_b_c"),
		Entry("Windows-reserved characters", `a<b>c:d"e|f?g*h`, "a_b_c_d_e_f_g_h"),
		Entry("consecutive separators", "a///b", "a___b"),
		Entry("control characters", "a\x00b\tc\x7fd", "a_b_c_d"),
	)
	DescribeTable("Should drop the trailing dots and spaces Windows drops",
		func(name, expected string) { Expect(sanitize(name)).To(Equal(expected)) },
		Entry("a trailing dot", "report.", "report"),
		Entry("a trailing space", "report ", "report"),
		Entry("both, repeated", "report. . ", "report"),
		Entry("a leading dot", ".hidden", ".hidden"),
	)
	DescribeTable("Should leave a name that sanitizes to nothing empty",
		func(name string) { Expect(sanitize(name)).To(BeEmpty()) },
		Entry("an empty name", ""),
		Entry("dots alone", "..."),
		Entry("spaces alone", "   "),
	)
	DescribeTable("Should push a Windows device name out of the way",
		func(name, expected string) { Expect(sanitize(name)).To(Equal(expected)) },
		Entry("a bare device name", "NUL", "_NUL"),
		Entry("a lowercase device name", "con", "_con"),
		Entry("a device name with an extension", "aux.json", "_aux.json"),
		Entry("a numbered device name", "COM1", "_COM1"),
		Entry("a device name left bare by the trim", "prn.", "_prn"),
	)
	DescribeTable("Should leave a name that is already safe untouched",
		func(name string) { Expect(sanitize(name)).To(Equal(name)) },
		Entry("spaces", "My Project"),
		Entry("underscores and digits", "metrics_2025"),
		Entry("dashes", "name-with-dashes"),
		Entry("an extension", "file.json"),
		Entry("a name a device name only prefixes", "console.json"),
		Entry("a device name outside the first element", "my nul"),
	)
	Describe("Length", func() {
		It("Should shorten a name to the budget", func() {
			Expect(xos.SanitizeFileName(strings.Repeat("a", 300), 10)).
				To(Equal(strings.Repeat("a", 10)))
		})
		It("Should hold the budget over a whole name and its extension", func() {
			name := xos.SanitizeFileName(
				strings.Repeat("a", 300), xos.MaxFileNameLength-len(".json"),
			)
			Expect(name + ".json").To(HaveLen(xos.MaxFileNameLength))
		})
		It("Should cut on a rune boundary", func() {
			// Each rune takes two bytes, so an odd budget cannot be filled exactly.
			Expect(xos.SanitizeFileName(strings.Repeat("é", 10), 5)).To(Equal("éé"))
		})
		It("Should drop a trailing space the cut exposes", func() {
			Expect(xos.SanitizeFileName("report x", 7)).To(Equal("report"))
		})
		It("Should hold a byte back for a device name's prefix", func() {
			Expect(xos.SanitizeFileName("nul.jsonnn", 8)).To(Equal("_nul.jso"))
		})
		DescribeTable("Should leave a name with no room empty",
			func(maxBytes int) {
				Expect(xos.SanitizeFileName("report", maxBytes)).To(BeEmpty())
			},
			Entry("a zero budget", 0),
			Entry("a negative budget", -1),
		)
	})
})

var _ = Describe("FoldFileName", func() {
	It("Should fold names that differ only by case together", func() {
		Expect(xos.FoldFileName("Inlet.json")).To(Equal(xos.FoldFileName("inlet.JSON")))
	})
	It("Should fold names that differ only by Unicode composition together", func() {
		// The same name precomposed (\u00e9) and decomposed (e + combining acute).
		precomposed, decomposed := "caf\u00e9.json", "cafe\u0301.json"
		Expect(precomposed).ToNot(Equal(decomposed))
		Expect(xos.FoldFileName(precomposed)).To(Equal(xos.FoldFileName(decomposed)))
	})
	It("Should keep distinct names apart", func() {
		Expect(xos.FoldFileName("inlet.json")).
			ToNot(Equal(xos.FoldFileName("outlet.json")))
	})
})
