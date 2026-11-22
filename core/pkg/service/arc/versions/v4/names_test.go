// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v4 "github.com/synnaxlabs/synnax/pkg/service/arc/versions/v4"
	"github.com/synnaxlabs/x/set"
)

// dashed returns a rename that quotes each dashed name that held reports true for.
func dashed(held func(string) bool) func(string) (string, bool) {
	return func(name string) (string, bool) {
		return name, strings.Contains(name, "-") && held(name)
	}
}

var _ = Describe("RenameChannels", func() {
	held := set.New("pt-01", "valve-cmd", "a-", "a--b", "pressure").Contains
	DescribeTable("Should quote each dashed name that a channel holds",
		func(text, expected string) {
			Expect(v4.RenameChannels(text, dashed(held))).To(Equal(expected))
		},
		Entry("a flow node", "pt-01 -> out{}", "'pt-01' -> out{}"),
		Entry("an expression", "return pt-01 * 2", "return 'pt-01' * 2"),
		Entry("an assignment target", "pt-01 = 5", "'pt-01' = 5"),
		Entry("an authority entry", "authority (valve-cmd 200)",
			"authority ('valve-cmd' 200)"),
		Entry("a flow arrow after the name", "pt-01->out{}", "'pt-01'->out{}"),
		Entry("a compound assignment after the name", "pt-01-=1", "'pt-01'-=1"),
		Entry("a trailing dash", "return a- + 1", "return 'a-' + 1"),
		Entry("a trailing dash at the end", "return a-", "return 'a-'"),
		Entry("two dashes", "return a--b", "return 'a--b'"),
		Entry("each of several names", "x := pt-01 + pt-01",
			"x := 'pt-01' + 'pt-01'"),
		Entry("a placeholder", `f"{pt-01}"`, `f"{'pt-01'}"`),
		Entry("a placeholder with a spec", `f"v={pt-01:.2f}"`, `f"v={'pt-01':.2f}"`),
		Entry("a raw format string", `rf"{pt-01}"`, `rf"{'pt-01'}"`),
		Entry("a raw format string with a backslash", `rf"\u{pt-01}"`,
			`rf"\u{'pt-01'}"`),
		Entry("a format string with the prefixes swapped", `fr"{pt-01}"`,
			`fr"{'pt-01'}"`),
		Entry("a multi-line format string", "f`{pt-01}\n`", "f`{'pt-01'}\n`"),
		Entry("a placeholder after an escape", `f"\"{pt-01}"`, `f"\"{'pt-01'}"`),
		Entry("a string in a placeholder", `f"{len(\"x\") + pt-01}"`,
			`f"{len(\"x\") + 'pt-01'}"`),
		Entry("a placeholder after an unknown escape", `f"\q{pt-01}"`,
			`f"\q{'pt-01'}"`),
		Entry("a name after digits", "return 10pt-01", "return 10'pt-01'"),
		Entry("a name after a not-equal", "x!=pt-01", "x!='pt-01'"),
		Entry("a name after a stateful declaration", "x $= pt-01", "x $= 'pt-01'"),
		Entry("a name after an empty quoted token", "'' pt-01'", "'' 'pt-01''"),
		Entry("an escape inside a placeholder name", `f"{pt\u002d01}"`,
			`f"{'pt-01'}"`),
		Entry("a name on the line after an unclosed quote", "\"x\npt-01",
			"\"x\n'pt-01'"),
	)

	DescribeTable("Should leave text as it is",
		func(text string) {
			Expect(v4.RenameChannels(text, dashed(held))).To(Equal(text))
		},
		Entry("a dashed name that no channel holds", "return pt-02 * 2"),
		Entry("a longer dashed name", "return pt-01-b"),
		Entry("a name with no dash", "return pressure"),
		Entry("a subtraction with spaces", "return pt - 01"),
		Entry("a line comment", "// pt-01\nreturn 1"),
		Entry("a block comment", "/* pt-01 */ return 1"),
		Entry("a string", `x := "pt-01"`),
		Entry("a raw string", `x := r"pt-01"`),
		Entry("a multi-line string", "x := `pt-01`"),
		Entry("a quoted token", "return 'pt-01'"),
		Entry("escaped braces", `f"{{pt-01}}"`),
		Entry("a placeholder in a string with no f prefix", `"{pt-01}"`),
		Entry("an unclosed placeholder", `f"{pt-01"`),
		Entry("a placeholder with an empty spec", `f"{pt-01:}"`),
		Entry("a name after an unclosed quote on its line", `"pt-01`),
		Entry("a name right after a lone '!'", "!pt-01"),
	)

	DescribeTable("Should write each renamed name as a quoted token",
		func(text, expected string) {
			names := map[string]string{
				"Metrics": "Metrics_2",
				"a":       "it's",
				"b":       `c\d`,
				"q":       `x"y`,
				"t":       "x`y",
			}
			rename := func(name string) (string, bool) {
				to, ok := names[name]
				return to, ok
			}
			Expect(v4.RenameChannels(text, rename)).To(Equal(expected))
		},
		Entry("a bare name", "return Metrics + 1", "return 'Metrics_2' + 1"),
		Entry("a name with a quote", "x := a", `x := 'it\'s'`),
		Entry("a name with a backslash", "x := b", `x := 'c\\d'`),
		Entry("a quote in a placeholder", `f"{a}"`, `f"{'it\\'s'}"`),
		Entry("a double quote in a placeholder", `f"{q}"`, `f"{'x\"y'}"`),
		Entry("a backtick in a multi-line placeholder", "f`{t}`", "f`{'x\\`y'}`"),
		Entry("a backslash in a raw placeholder", `rf"{b}"`, `rf"{'c\\d'}"`),
		Entry("a double quote that a raw placeholder cannot hold", `rf"{q}"`,
			`rf"{q}"`),
		Entry("a placeholder in a placeholder", `f"{f\"{a}\"}"`,
			`f"{f\"{'it\\\\'s'}\"}"`),
	)
})
