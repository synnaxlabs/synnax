// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v1"
)

// renameTo returns a rename that maps each old name in names to its new name.
func renameTo(names map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		to, ok := names[name]
		return to, ok
	}
}

var _ = Describe("RenameInText", func() {
	rename := renameTo(map[string]string{
		"pt-01":   "pt_01",
		"a-":      "a_2",
		"a--b":    "a_b",
		"Metrics": "Metrics_2",
	})
	DescribeTable("Should write the new name of each renamed channel",
		func(text, expected string) {
			Expect(v1.RenameInText(text, rename)).To(Equal(expected))
		},
		Entry("a bare name", "return Metrics + 1", "return Metrics_2 + 1"),
		Entry("a flow node", "pt-01 -> out{}", "pt_01 -> out{}"),
		Entry("an expression", "return pt-01 * 2", "return pt_01 * 2"),
		Entry("an assignment target", "pt-01 = 5", "pt_01 = 5"),
		Entry("an authority entry", "authority (pt-01 200)", "authority (pt_01 200)"),
		Entry("a flow arrow after the name", "pt-01->out{}", "pt_01->out{}"),
		Entry("a compound assignment after the name", "pt-01-=1", "pt_01-=1"),
		Entry("a trailing dash", "return a- + 1", "return a_2 + 1"),
		Entry("a trailing dash at the end", "return a-", "return a_2"),
		Entry("two dashes", "return a--b", "return a_b"),
		Entry("each of several names", "x := pt-01 + pt-01", "x := pt_01 + pt_01"),
		Entry("a placeholder", `f"{pt-01}"`, `f"{pt_01}"`),
		Entry("a placeholder with a spec", `f"v={pt-01:.2f}"`, `f"v={pt_01:.2f}"`),
		Entry("a raw format string", `rf"{pt-01}"`, `rf"{pt_01}"`),
		Entry("a raw format string with a backslash", `rf"\u{pt-01}"`,
			`rf"\u{pt_01}"`),
		Entry("a format string with the prefixes swapped", `fr"{pt-01}"`,
			`fr"{pt_01}"`),
		Entry("a multi-line format string", "f`{pt-01}\n`", "f`{pt_01}\n`"),
		Entry("a placeholder after an escape", `f"\"{pt-01}"`, `f"\"{pt_01}"`),
		Entry("a string in a placeholder", `f"{len(\"x\") + pt-01}"`,
			`f"{len(\"x\") + pt_01}"`),
		Entry("a placeholder after an unknown escape", `f"\q{pt-01}"`,
			`f"\q{pt_01}"`),
		Entry("a placeholder in a placeholder", `f"{f\"{Metrics}\"}"`,
			`f"{f\"{Metrics_2}\"}"`),
		Entry("an escape inside a placeholder name", `f"{pt-01}"`,
			`f"{pt_01}"`),
		Entry("a name after digits", "return 10pt-01", "return 10pt_01"),
		Entry("a name after a not-equal", "x!=pt-01", "x!=pt_01"),
		Entry("a name after a stateful declaration", "x $= pt-01", "x $= pt_01"),
		Entry("a name between quotes, which v0.62 dropped", "'pt-01'", "'pt_01'"),
		Entry("a name on the line after an unclosed quote", "\"x\npt-01",
			"\"x\npt_01"),
	)

	DescribeTable("Should leave text as it is",
		func(text string) {
			Expect(v1.RenameInText(text, rename)).To(Equal(text))
		},
		Entry("a name that no renamed channel held", "return pt-02 * 2"),
		Entry("a longer dashed name", "return pt-01-b"),
		Entry("a subtraction with spaces", "return pt - 01"),
		Entry("a line comment", "// pt-01\nreturn 1"),
		Entry("a block comment", "/* pt-01 */ return 1"),
		Entry("a string", `x := "pt-01"`),
		Entry("a raw string", `x := r"pt-01"`),
		Entry("a multi-line string", "x := `pt-01`"),
		Entry("escaped braces", `f"{{pt-01}}"`),
		Entry("a placeholder in a string with no f prefix", `"{pt-01}"`),
		Entry("an unclosed placeholder", `f"{pt-01"`),
		Entry("a placeholder with an empty spec", `f"{pt-01:}"`),
		Entry("a name after an unclosed quote on its line", `"pt-01`),
		Entry("a name right after a lone '!'", "!pt-01"),
	)
})
