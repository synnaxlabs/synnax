// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package filename_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/filename"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// maxLength mirrors the limit the package keeps to itself.
const maxLength = 255

// sanitize names a file that carries no extension, which the cases below do not vary.
func sanitize(name string) string {
	GinkgoHelper()
	return MustSucceed(filename.Sanitize(name, ""))
}

var _ = Describe("Sanitize", func() {
	DescribeTable("Should replace every character a file name cannot hold",
		func(name, expected string) { Expect(sanitize(name)).To(Equal(expected)) },
		Entry("path separators", `a/b\c`, "a_b_c"),
		Entry("Windows-reserved characters", `a<b>c:d"e|f?g*h`, "a_b_c_d_e_f_g_h"),
		Entry("consecutive separators", "a///b", "a___b"),
		Entry("control characters", "a\x00b\tc\x1fd", "a_b_c_d"),
	)
	DescribeTable("Should drop the trailing dots and spaces Windows drops",
		func(name, expected string) { Expect(sanitize(name)).To(Equal(expected)) },
		Entry("a trailing dot", "report.", "report"),
		Entry("a trailing space", "report ", "report"),
		Entry("both, repeated", "report. . ", "report"),
		Entry("a leading dot", ".hidden", ".hidden"),
	)
	DescribeTable("Should name a file that sanitizes to nothing with an underscore",
		func(name string) { Expect(sanitize(name)).To(Equal("_")) },
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
		Entry("a delete character, which Windows allows", "a\x7fb"),
	)
	Describe("Extension", func() {
		It("Should carry the extension", func() {
			Expect(filename.Sanitize("in/let", ".json")).To(Equal("in_let.json"))
		})
		It("Should name the file with an underscore when only the extension survives",
			func() {
				Expect(filename.Sanitize("...", ".json")).To(Equal("_.json"))
			},
		)
	})
	Describe("Length", func() {
		It("Should shorten a name too long for a file name", func() {
			Expect(sanitize(strings.Repeat("a", 400))).
				To(Equal(strings.Repeat("a", maxLength)))
		})
		It("Should hold the extension's bytes back", func() {
			Expect(filename.Sanitize(strings.Repeat("a", 400), ".json")).
				To(Equal(strings.Repeat("a", maxLength-len(".json")) + ".json"))
		})
		It("Should cut on a rune boundary", func() {
			// Each rune takes two bytes, so an odd limit cannot be filled exactly.
			Expect(sanitize(strings.Repeat("é", 200))).
				To(Equal(strings.Repeat("é", maxLength/2)))
		})
		It("Should drop a trailing space the cut exposes", func() {
			name := strings.Repeat("a", maxLength-1) + " b"
			Expect(sanitize(name)).To(Equal(strings.Repeat("a", maxLength-1)))
		})
		It("Should hold a byte back for a device name's prefix", func() {
			Expect(sanitize("nul." + strings.Repeat("a", 400))).
				To(SatisfyAll(HaveLen(maxLength), HavePrefix("_nul.")))
		})
		DescribeTable("Should reject an extension that fills a file name by itself",
			func(extension string) {
				Expect(filename.Sanitize("report", extension)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("leaves no room for a file name")),
				))
			},
			Entry("an extension the length of the limit",
				strings.Repeat("a", maxLength)),
			Entry("an extension past the limit",
				strings.Repeat("a", maxLength+1)),
		)
		It("Should name a file when the extension leaves one byte", func() {
			extension := strings.Repeat("a", maxLength-1)
			Expect(filename.Sanitize("report", extension)).
				To(Equal("r" + extension))
		})
	})
})

var _ = Describe("Fold", func() {
	It("Should fold names that differ only by case together", func() {
		Expect(filename.Fold("Inlet.json")).To(Equal(filename.Fold("inlet.JSON")))
	})
	It("Should fold names that differ only by Unicode composition together", func() {
		// The same name precomposed (\u00e9) and decomposed (e + combining acute).
		precomposed, decomposed := "caf\u00e9.json", "cafe\u0301.json"
		Expect(precomposed).ToNot(Equal(decomposed))
		Expect(filename.Fold(precomposed)).To(Equal(filename.Fold(decomposed)))
	})
	It("Should keep distinct names apart", func() {
		Expect(filename.Fold("inlet.json")).
			ToNot(Equal(filename.Fold("outlet.json")))
	})
})
