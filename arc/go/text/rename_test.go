// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package text_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/text"
)

var _ = Describe("RenameChannel", func() {
	DescribeTable(
		"Should rename each token of the channel",
		func(source string, block bool, to, expected string) {
			renamed, ok := text.RenameChannel(
				source,
				block,
				"sy_x",
				to,
				parser.Config{},
			)
			Expect(ok).To(BeTrue())
			Expect(renamed).To(Equal(expected))
		},
		Entry(
			"a bare token in an expression",
			"return sy_x * 2 + sy_x",
			true,
			"sy_x_2",
			"return sy_x_2 * 2 + sy_x_2",
		),
		Entry(
			"a quoted token, which stays quoted",
			"return 'sy_x' * 2",
			true,
			"sy_x_2",
			"return 'sy_x_2' * 2",
		),
		Entry(
			"a bare token as a quoted one when the new name is no identifier",
			"return sy_x * 2",
			true,
			"sy-x",
			"return 'sy-x' * 2",
		),
		Entry(
			"no other name",
			"return sy_xy + sy_x",
			true,
			"sy_x_2",
			"return sy_xy + sy_x_2",
		),
		Entry(
			"a token in a placeholder of a format string",
			`'trig' -> f"{sy_x}" -> 'log'`,
			false,
			"sy_x_2",
			`'trig' -> f"{sy_x_2}" -> 'log'`,
		),
		Entry(
			"a flow node of a program",
			"sy_x -> 'log'",
			false,
			"sy_x_2",
			"sy_x_2 -> 'log'",
		),
		Entry(
			"only a quoted token when the source declares the name",
			"sy_x := 1\nreturn sy_x + 'sy_x'",
			true,
			"sy_x_2",
			"sy_x := 1\nreturn sy_x + 'sy_x_2'",
		),
		Entry(
			"nothing in source that does not parse",
			"return sy_x +",
			true,
			"sy_x_2",
			"return sy_x +",
		),
	)

	It("Should refuse a token that a raw string cannot hold", func() {
		source := `'trig' -> rf"{sy_x}" -> 'log'`
		renamed, ok := text.RenameChannel(
			source,
			false,
			"sy_x",
			`sy"x`,
			parser.Config{},
		)
		Expect(ok).To(BeFalse())
		Expect(renamed).To(Equal(source))
	})
})
